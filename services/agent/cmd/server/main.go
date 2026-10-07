package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/planly/pkg/config"
	"github.com/planly/pkg/dbx"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/logx"
	"github.com/planly/services/agent/internal/handler"
	"github.com/planly/services/agent/internal/repository"
	"github.com/planly/services/agent/internal/service"
	"github.com/planly/services/agent/internal/worker"
	"github.com/planly/services/agent/migrations"
)

type Config struct {
	Port                   int    `envconfig:"SERVICE_PORT" default:"8084"`
	JWTSecret              string `envconfig:"JWT_SECRET" required:"true"`
	InternalKey            string `envconfig:"INTERNAL_API_KEY" required:"true"`
	PlannerServiceURL      string `envconfig:"PLANNER_SERVICE_URL" default:"http://planner-svc:8082"`
	NotifyServiceURL       string `envconfig:"NOTIFY_SERVICE_URL" default:"http://notify-svc:8083"`
	AuthServiceURL         string `envconfig:"AUTH_SERVICE_URL" default:"http://auth-svc:8081"`
	// LLM providers — Anthropic > Gemini > Ollama (local, no key required)
	AnthropicAPIKey        string `envconfig:"ANTHROPIC_API_KEY"`
	AnthropicModel         string `envconfig:"ANTHROPIC_MODEL" default:"claude-3-5-sonnet-20241022"`
	GeminiAPIKey           string `envconfig:"GEMINI_API_KEY"`
	GeminiModel            string `envconfig:"GEMINI_MODEL" default:"gemini-3.5-flash-lite"`
	OllamaURL              string `envconfig:"OLLAMA_URL" default:"http://ollama:11434"`
	OllamaModel            string `envconfig:"OLLAMA_MODEL" default:"qwen2.5:7b"`
	DailyMessageCap        int64  `envconfig:"DAILY_MESSAGE_CAP" default:"50"`
	BriefingIntervalMinute int    `envconfig:"BRIEFING_INTERVAL_MINUTES" default:"5"`
	DB                     dbx.Config
}

func main() {
	logx.InitLogger()

	var cfg Config
	if err := config.Load("", &cfg); err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	dbPool, err := dbx.Connect(ctx, cfg.DB)
	if err != nil {
		slog.Error("failed to connect to agent_db", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Run migrations
	if err := migrations.Run(ctx, dbPool); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize LLM Client — priority: Anthropic > Gemini > Ollama (local, no key)
	var llmClient service.LLMClient
	switch {
	case cfg.AnthropicAPIKey != "":
		slog.Info("using Anthropic LLM provider", "model", cfg.AnthropicModel)
		llmClient = service.NewAnthropicClient(cfg.AnthropicAPIKey, cfg.AnthropicModel)
	case cfg.GeminiAPIKey != "":
		slog.Info("using Gemini LLM provider", "model", cfg.GeminiModel)
		llmClient = service.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	default:
		slog.Info("using Ollama LLM provider (local, no API key required)", "url", cfg.OllamaURL, "model", cfg.OllamaModel)
		llmClient = service.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel)
	}

	// Initialize Planner Client
	plannerClient := service.NewPlannerClient(cfg.PlannerServiceURL)

	// Initialize Repository & Service
	repo := repository.New(dbPool)
	agentSvc := service.NewAgentService(repo, llmClient, plannerClient, cfg.DailyMessageCap)
	agentHandler := handler.NewAgentHandler(agentSvc, cfg.AuthServiceURL)

	// Start Briefing Worker
	briefingWorker := worker.NewBriefingWorker(
		repo,
		llmClient,
		cfg.AuthServiceURL,
		cfg.NotifyServiceURL,
		cfg.InternalKey,
		time.Duration(cfg.BriefingIntervalMinute)*time.Minute,
	)
	go briefingWorker.Start(ctx)

	// Setup HTTP router
	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "agent-svc",
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, pingCancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer pingCancel()

		if err := dbPool.Ping(pingCtx); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "database ping failed")
			return
		}

		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"service":  "agent-svc",
			"database": "connected",
		})
	})

	// Register Agent routes
	agentHandler.RegisterRoutes(r, []byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("starting agent-svc", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("agent-svc server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down agent-svc gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}
	slog.Info("agent-svc stopped")
}
