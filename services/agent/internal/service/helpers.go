package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/repository"
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

func TimePtrToPgtype(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func PgtypeToTimePtr(p pgtype.Timestamptz) *time.Time {
	if !p.Valid {
		return nil
	}
	t := p.Time.UTC()
	return &t
}

func TextToPgtype(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func PgtypeToTextPtr(p pgtype.Text) *string {
	if !p.Valid {
		return nil
	}
	s := p.String
	return &s
}

func MapConversationToResponse(c repository.Conversation) model.ConversationResponse {
	return model.ConversationResponse{
		ID:        PgtypeToUUID(c.ID),
		UserID:    PgtypeToUUID(c.UserID),
		Title:     c.Title,
		CreatedAt: c.CreatedAt.Time.UTC(),
	}
}

func MapMessageToResponse(m repository.Message) model.MessageResponse {
	return model.MessageResponse{
		ID:             PgtypeToUUID(m.ID),
		ConversationID: PgtypeToUUID(m.ConversationID),
		Role:           m.Role,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.Time.UTC(),
	}
}
