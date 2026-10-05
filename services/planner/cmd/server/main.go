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
	"github.com/planly/services/planner/internal/handler"
	"github.com/planly/services/planner/internal/repository"
	"github.com/planly/services/planner/internal/service"
	"github.com/planly/services/planner/migrations"
)

type Config struct {
	Port        int        `envconfig:"SERVICE_PORT" default:"8082"`
	JWTSecret   string     `envconfig:"JWT_SECRET" required:"true"`
	InternalKey string     `envconfig:"INTERNAL_API_KEY" required:"true"`
	DB          dbx.Config
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
		slog.Error("failed to connect to planner_db", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Run migrations
	if err := migrations.Run(ctx, dbPool); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Initialize layers
	repo := repository.New(dbPool)
	plannerSvc := service.NewPlannerService(dbPool, repo)
	plannerHandler := handler.NewPlannerHandler(plannerSvc)

	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "planner-svc",
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
			"service":  "planner-svc",
			"database": "connected",
		})
	})

	// Register domain routes
	plannerHandler.RegisterRoutes(r, []byte(cfg.JWTSecret))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("starting planner-svc", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("planner-svc server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down planner-svc gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}
	slog.Info("planner-svc stopped")
}
