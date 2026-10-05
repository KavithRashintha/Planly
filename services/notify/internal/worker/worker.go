package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/planly/services/notify/internal/service"
)

type Worker struct {
	svc      service.NotifyService
	interval time.Duration
	stop     chan struct{}
}

func NewWorker(svc service.NotifyService, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Worker{
		svc:      svc,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	slog.Info("notification due-reminder worker started", "interval", w.interval.String())

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("notification worker stopping (context cancelled)...")
				return
			case <-w.stop:
				slog.Info("notification worker stopping (stopped explicitly)...")
				return
			case <-ticker.C:
				processed, err := w.svc.ProcessDueReminders(ctx)
				if err != nil {
					slog.Error("error processing due reminders", "error", err)
				} else if processed > 0 {
					slog.Info("processed due reminders", "count", processed)
				}
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stop)
}
