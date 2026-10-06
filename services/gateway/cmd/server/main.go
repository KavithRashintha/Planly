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
	"github.com/go-chi/cors"
	"github.com/planly/pkg/config"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/logx"
	"github.com/planly/services/gateway/internal/middleware"
	"github.com/planly/services/gateway/internal/proxy"
	"golang.org/x/time/rate"
)

type Config struct {
	Port              int    `envconfig:"GATEWAY_PORT" default:"8080"`
	JWTSecret         string `envconfig:"JWT_SECRET" required:"true"`
	AuthServiceURL    string `envconfig:"AUTH_SERVICE_URL" default:"http://localhost:8081"`
	PlannerServiceURL string `envconfig:"PLANNER_SERVICE_URL" default:"http://localhost:8082"`
	NotifyServiceURL  string `envconfig:"NOTIFY_SERVICE_URL" default:"http://localhost:8083"`
	AgentServiceURL   string `envconfig:"AGENT_SERVICE_URL" default:"http://localhost:8084"`
}

func main() {
	logx.InitLogger()

	var cfg Config
	if err := config.Load("", &cfg); err != nil {
		slog.Error("failed to load gateway configuration", "error", err)
		os.Exit(1)
	}

	// Create reverse proxies for each upstream service
	authProxy, err := proxy.NewReverseProxy(cfg.AuthServiceURL)
	if err != nil {
		slog.Error("invalid auth service url", "error", err)
		os.Exit(1)
	}

	plannerProxy, err := proxy.NewReverseProxy(cfg.PlannerServiceURL)
	if err != nil {
		slog.Error("invalid planner service url", "error", err)
		os.Exit(1)
	}

	notifyProxy, err := proxy.NewReverseProxy(cfg.NotifyServiceURL)
	if err != nil {
		slog.Error("invalid notify service url", "error", err)
		os.Exit(1)
	}

	agentProxy, err := proxy.NewReverseProxy(cfg.AgentServiceURL)
	if err != nil {
		slog.Error("invalid agent service url", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	// 1. CORS Middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Internal-Key"},
		ExposedHeaders:   []string{"Link", "X-Total-Count", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 2. Base Middlewares
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

	// 3. Per-IP Token Bucket Rate Limiter (e.g. 50 req/sec, burst 100)
	rateLimiter := middleware.NewIPRateLimiter(rate.Limit(50), 100)
	r.Use(rateLimiter.Middleware())

	// Health and readiness endpoints
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "api-gateway",
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ready",
			"service": "api-gateway",
		})
	})

	// Public Authentication Routes
	r.Post("/api/v1/auth/register", authProxy.ServeHTTP)
	r.Post("/api/v1/auth/login", authProxy.ServeHTTP)
	r.Post("/api/v1/auth/refresh", authProxy.ServeHTTP)

	// Protected Routes (Protected by Gateway JWT Check + Forwarding X-User-ID)
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.JWTAuthMiddleware([]byte(cfg.JWTSecret)))

		// Auth profile routes
		protected.Handle("/api/v1/auth/*", authProxy)

		// Planner service routes
		protected.Handle("/api/v1/projects*", plannerProxy)
		protected.Handle("/api/v1/tasks*", plannerProxy)
		protected.Handle("/api/v1/subtasks*", plannerProxy)
		protected.Handle("/api/v1/tags*", plannerProxy)
		protected.Handle("/api/v1/time-blocks*", plannerProxy)
		protected.Handle("/api/v1/workload*", plannerProxy)
		protected.Handle("/api/v1/dashboard*", plannerProxy)

		// Notify service routes
		protected.Handle("/api/v1/reminders*", notifyProxy)
		protected.Handle("/api/v1/notifications*", notifyProxy)

		// Agent service routes
		protected.Handle("/api/v1/agent*", agentProxy)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("starting api-gateway", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api-gateway server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down api-gateway gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}
	slog.Info("api-gateway stopped")
}
