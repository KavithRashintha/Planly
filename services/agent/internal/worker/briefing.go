package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/agent/internal/repository"
	"github.com/planly/services/agent/internal/service"
)

type BriefingUserCandidate struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Persona   string    `json:"persona"`
	Timezone  string    `json:"timezone"`
	WorkStart string    `json:"work_start"`
	WorkEnd   string    `json:"work_end"`
}

type InternalNotificationRequest struct {
	UserID *uuid.UUID `json:"user_id,omitempty"`
	Kind   string     `json:"kind"`
	Title  string     `json:"title"`
	Body   string     `json:"body"`
}

type BriefingWorker struct {
	repo           repository.Querier
	llm            service.LLMClient
	authBaseURL    string
	notifyBaseURL  string
	internalKey    string
	httpClient     *http.Client
	pollInterval   time.Duration
}

func NewBriefingWorker(
	repo repository.Querier,
	llm service.LLMClient,
	authBaseURL string,
	notifyBaseURL string,
	internalKey string,
	pollInterval time.Duration,
) *BriefingWorker {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Minute
	}
	return &BriefingWorker{
		repo:          repo,
		llm:           llm,
		authBaseURL:   authBaseURL,
		notifyBaseURL: notifyBaseURL,
		internalKey:   internalKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		pollInterval: pollInterval,
	}
}

// Start launches the background worker loop.
func (w *BriefingWorker) Start(ctx context.Context) {
	slog.Info("starting daily briefing worker", "interval", w.pollInterval)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Run once immediately on startup
	w.processBriefings(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping daily briefing worker")
			return
		case <-ticker.C:
			w.processBriefings(ctx)
		}
	}
}

func (w *BriefingWorker) processBriefings(ctx context.Context) {
	candidates, err := w.fetchCandidates(ctx)
	if err != nil {
		slog.Error("failed to fetch briefing candidates from auth-svc", "error", err)
		return
	}

	for _, user := range candidates {
		if err := w.processCandidate(ctx, user); err != nil {
			slog.Error("failed to process briefing for candidate", "user_id", user.ID, "error", err)
		}
	}
}

func (w *BriefingWorker) fetchCandidates(ctx context.Context) ([]BriefingUserCandidate, error) {
	url := fmt.Sprintf("%s/internal/users/briefing-candidates", w.authBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Key", w.internalKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}

	var candidates []BriefingUserCandidate
	if err := json.NewDecoder(resp.Body).Decode(&candidates); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (w *BriefingWorker) processCandidate(ctx context.Context, user BriefingUserCandidate) error {
	loc, err := time.LoadLocation(user.Timezone)
	if err != nil {
		loc = time.UTC
	}

	localNow := time.Now().In(loc)

	// Parse work_start (format: "15:04" or "15:04:05")
	workStart := user.WorkStart
	if workStart == "" {
		workStart = "09:00"
	}
	parts := strings.Split(workStart, ":")
	startHour := 9
	startMin := 0
	if len(parts) >= 2 {
		_, _ = fmt.Sscanf(parts[0], "%d", &startHour)
		_, _ = fmt.Sscanf(parts[1], "%d", &startMin)
	}

	// Check if local time is at or after work_start
	currentMinutes := localNow.Hour()*60 + localNow.Minute()
	startMinutes := startHour*60 + startMin
	if currentMinutes < startMinutes {
		return nil // Not yet work start for this user
	}

	// Date format for briefing
	localDate := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
	pgDate := pgtype.Date{Time: localDate, Valid: true}
	userUUID := service.UUIDToPgtype(user.ID)

	// Check if already briefed today
	hasBriefing, err := w.repo.HasBriefingForDate(ctx, repository.HasBriefingForDateParams{
		UserID:       userUUID,
		BriefingDate: pgDate,
	})
	if err != nil {
		return fmt.Errorf("failed to check briefing log: %w", err)
	}
	if hasBriefing {
		return nil
	}

	// Record in briefing log FIRST to prevent duplicate delivery
	if err := w.repo.RecordBriefing(ctx, repository.RecordBriefingParams{
		UserID:       userUUID,
		BriefingDate: pgDate,
	}); err != nil {
		return fmt.Errorf("failed to record briefing: %w", err)
	}

	// Generate briefing text
	briefingBody := fmt.Sprintf("Good morning, %s! Ready to tackle today? Check your dashboard to view your scheduled tasks and time blocks.", user.FullName)
	if w.llm != nil {
		llmPrompt := fmt.Sprintf("Write a warm, concise 2-sentence morning briefing for %s (role: %s) on %s. Motivate them for their day.", user.FullName, user.Persona, localNow.Format("Monday, 02 Jan 2006"))
		llmResp, err := w.llm.Generate(ctx, service.LLMRequest{
			System: "You are Planly, an encouraging and structured AI work planner.",
			Messages: []service.LLMMessage{
				{
					Role: service.RoleUser,
					Content: []service.LLMContentBlock{
						{Type: "text", Text: llmPrompt},
					},
				},
			},
			MaxTokens: 200,
		})
		if err == nil && llmResp.Text != "" {
			briefingBody = strings.TrimSpace(llmResp.Text)
		}
	}

	// Send notification to notify-svc
	return w.sendNotification(ctx, user.ID, localNow.Format("Monday, Jan 2"), briefingBody)
}

func (w *BriefingWorker) sendNotification(ctx context.Context, userID uuid.UUID, dateStr, body string) error {
	notifReq := InternalNotificationRequest{
		UserID: &userID,
		Kind:   "briefing",
		Title:  fmt.Sprintf("Daily Briefing • %s", dateStr),
		Body:   body,
	}

	b, err := json.Marshal(notifReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/internal/notifications", w.notifyBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", w.internalKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post notification to notify-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notify-svc returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	slog.Info("sent daily briefing notification", "user_id", userID)
	return nil
}
