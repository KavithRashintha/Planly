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
	"github.com/planly/services/notify/internal/handler"
	"github.com/planly/services/notify/internal/repository"
	"github.com/planly/services/notify/internal/service"
	"github.com/planly/services/notify/internal/worker"
	"github.com/planly/services/notify/migrations"
)

type Config struct {
	Port        int        `envconfig:"SERVICE_PORT" default:"8083"`
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
		slog.Error("failed to connect to notify_db", "error", err)
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
	notifySvc := service.NewNotifyService(dbPool, repo)
	notifyHandler := handler.NewNotifyHandler(notifySvc)

	// Start background due-reminder worker
	notifyWorker := worker.NewWorker(notifySvc, 30*time.Second)
	notifyWorker.Start(ctx)

	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "notify-svc",
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
			"service":  "notify-svc",
			"database": "connected",
		})
	})

	// Register domain routes
	notifyHandler.RegisterRoutes(r, []byte(cfg.JWTSecret), cfg.InternalKey)

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
		slog.Info("starting notify-svc", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("notify-svc server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down notify-svc gracefully...")
	notifyWorker.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}
	slog.Info("notify-svc stopped")
}
