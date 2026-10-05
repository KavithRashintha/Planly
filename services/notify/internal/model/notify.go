package model

import (
	"time"

	"github.com/google/uuid"
)

type CreateReminderRequest struct {
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Message  string     `json:"message"`
	RemindAt time.Time  `json:"remind_at"`
}

type ReminderResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TaskID    *uuid.UUID `json:"task_id,omitempty"`
	Message   string     `json:"message"`
	RemindAt  time.Time  `json:"remind_at"`
	Fired     bool       `json:"fired"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateNotificationRequest struct {
	UserID *uuid.UUID `json:"user_id,omitempty"` // For internal calls
	Kind   string     `json:"kind"`              // reminder | briefing | agent | system
	Title  string     `json:"title"`
	Body   string     `json:"body"`
}

type NotificationResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}
