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
	"github.com/planly/services/auth/internal/handler"
	"github.com/planly/services/auth/internal/repository"
	"github.com/planly/services/auth/internal/service"
	"github.com/planly/services/auth/migrations"
)

type Config struct {
	Port        int        `envconfig:"SERVICE_PORT" default:"8081"`
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

	// Connect to database with retry
	dbPool, err := dbx.Connect(ctx, cfg.DB)
	if err != nil {
		slog.Error("failed to connect to auth_db", "error", err)
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
	authSvc := service.NewAuthService(repo, []byte(cfg.JWTSecret))
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "auth-svc",
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
			"service":  "auth-svc",
			"database": "connected",
		})
	})

	// Register domain routes
	authHandler.RegisterRoutes(r, []byte(cfg.JWTSecret), cfg.InternalKey)

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
		slog.Info("starting auth-svc", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("auth-svc server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down auth-svc gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}
	slog.Info("auth-svc stopped")
}
