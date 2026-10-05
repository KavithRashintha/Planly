package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/planly/services/notify/internal/model"
	"github.com/planly/services/notify/internal/repository"
)

var (
	ErrNotFound   = errors.New("resource not found")
	ErrValidation = errors.New("validation error")
)

type NotifyService interface {
	CreateReminder(ctx context.Context, userID uuid.UUID, req model.CreateReminderRequest) (*model.ReminderResponse, error)
	CreateNotification(ctx context.Context, userID uuid.UUID, req model.CreateNotificationRequest) (*model.NotificationResponse, error)
	ListNotifications(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.NotificationResponse, int64, error)
	MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) (*model.NotificationResponse, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (*model.UnreadCountResponse, error)
	ProcessDueReminders(ctx context.Context) (int, error)
}

type notifyService struct {
	pool *pgxpool.Pool
	repo repository.Querier
}

func NewNotifyService(pool *pgxpool.Pool, repo repository.Querier) NotifyService {
	return &notifyService{
		pool: pool,
		repo: repo,
	}
}

func (s *notifyService) CreateReminder(ctx context.Context, userID uuid.UUID, req model.CreateReminderRequest) (*model.ReminderResponse, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, fmt.Errorf("%w: message is required", ErrValidation)
	}

	remindAt := req.RemindAt.UTC()
	if !remindAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("%w: remind_at must be in the future", ErrValidation)
	}

	reminder, err := s.repo.CreateReminder(ctx, repository.CreateReminderParams{
		UserID:   UUIDToPgtype(userID),
		TaskID:   UUIDPtrToPgtype(req.TaskID),
		Message:  message,
		RemindAt: TimeToPgtype(remindAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	resp := MapReminderToResponse(reminder)
	return &resp, nil
}

func (s *notifyService) CreateNotification(ctx context.Context, userID uuid.UUID, req model.CreateNotificationRequest) (*model.NotificationResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}

	kind := req.Kind
	if kind != "reminder" && kind != "briefing" && kind != "agent" && kind != "system" {
		return nil, fmt.Errorf("%w: kind must be reminder, briefing, agent, or system", ErrValidation)
	}

	notif, err := s.repo.CreateNotification(ctx, repository.CreateNotificationParams{
		UserID: UUIDToPgtype(userID),
		Kind:   kind,
		Title:  title,
		Body:   req.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	resp := MapNotificationToResponse(notif)
	return &resp, nil
}

func (s *notifyService) ListNotifications(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.NotificationResponse, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	total, err := s.repo.CountNotifications(ctx, UUIDToPgtype(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	items, err := s.repo.ListNotifications(ctx, repository.ListNotificationsParams{
		UserID: UUIDToPgtype(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	resp := make([]model.NotificationResponse, len(items))
	for i, item := range items {
		resp[i] = MapNotificationToResponse(item)
	}

	return resp, total, nil
}

func (s *notifyService) MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) (*model.NotificationResponse, error) {
	notif, err := s.repo.MarkNotificationAsRead(ctx, repository.MarkNotificationAsReadParams{
		ID:     UUIDToPgtype(notificationID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to mark notification as read: %w", err)
	}

	resp := MapNotificationToResponse(notif)
	return &resp, nil
}

func (s *notifyService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (*model.UnreadCountResponse, error) {
	count, err := s.repo.GetUnreadNotificationCount(ctx, UUIDToPgtype(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to get unread count: %w", err)
	}

	return &model.UnreadCountResponse{UnreadCount: count}, nil
}

func (s *notifyService) ProcessDueReminders(ctx context.Context) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

	now := time.Now().UTC()
	reminders, err := qtx.GetDueRemindersForUpdate(ctx, repository.GetDueRemindersForUpdateParams{
		RemindAt: TimeToPgtype(now),
		Limit:    50,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get due reminders: %w", err)
	}

	processedCount := 0
	for _, r := range reminders {
		// Insert notification
		_, err := qtx.CreateNotification(ctx, repository.CreateNotificationParams{
			UserID: r.UserID,
			Kind:   "reminder",
			Title:  "Reminder",
			Body:   r.Message,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to insert reminder notification: %w", err)
		}

		// Mark fired
		if err := qtx.MarkReminderFired(ctx, r.ID); err != nil {
			return 0, fmt.Errorf("failed to mark reminder fired: %w", err)
		}

		processedCount++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit reminder processing: %w", err)
	}

	return processedCount, nil
}
