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
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/logx"
)

type Config struct {
	Port int `envconfig:"GATEWAY_PORT" default:"8080"`
}

func main() {
	logx.InitLogger()

	var cfg Config
	if err := config.Load("", &cfg); err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Logging)
	r.Use(httpx.Recover)

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
