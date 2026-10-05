package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/notify/internal/model"
	"github.com/planly/services/notify/internal/repository"
)

func UUIDToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func UUIDPtrToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func PgtypeToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

func PgtypeToUUIDPtr(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	u := uuid.UUID(p.Bytes)
	return &u
}

func TimeToPgtype(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func MapReminderToResponse(r repository.Reminder) model.ReminderResponse {
	return model.ReminderResponse{
		ID:        PgtypeToUUID(r.ID),
		UserID:    PgtypeToUUID(r.UserID),
		TaskID:    PgtypeToUUIDPtr(r.TaskID),
		Message:   r.Message,
		RemindAt:  r.RemindAt.Time.UTC(),
		Fired:     r.Fired,
		CreatedAt: r.CreatedAt.Time.UTC(),
	}
}

func MapNotificationToResponse(n repository.Notification) model.NotificationResponse {
	return model.NotificationResponse{
		ID:        PgtypeToUUID(n.ID),
		UserID:    PgtypeToUUID(n.UserID),
		Kind:      n.Kind,
		Title:     n.Title,
		Body:      n.Body,
		Read:      n.Read,
		CreatedAt: n.CreatedAt.Time.UTC(),
	}
}
