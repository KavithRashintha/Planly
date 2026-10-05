package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/planner/internal/model"
	"github.com/planly/services/planner/internal/repository"
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

func MapProjectToResponse(p repository.Project) model.ProjectResponse {
	return model.ProjectResponse{
		ID:        PgtypeToUUID(p.ID),
		UserID:    PgtypeToUUID(p.UserID),
		Name:      p.Name,
		Colour:    p.Colour,
		Archived:  p.Archived,
		CreatedAt: p.CreatedAt.Time.UTC(),
		UpdatedAt: p.UpdatedAt.Time.UTC(),
	}
}

func MapSubtaskToResponse(s repository.Subtask) model.SubtaskResponse {
	return model.SubtaskResponse{
		ID:       PgtypeToUUID(s.ID),
		TaskID:   PgtypeToUUID(s.TaskID),
		Title:    s.Title,
		Done:     s.Done,
		Position: s.Position,
	}
}

func MapTagToResponse(t repository.Tag) model.TagResponse {
	return model.TagResponse{
		ID:     PgtypeToUUID(t.ID),
		UserID: PgtypeToUUID(t.UserID),
		Name:   t.Name,
	}
}

func MapTimeBlockToResponse(b repository.TimeBlock) model.TimeBlockResponse {
	return model.TimeBlockResponse{
		ID:       PgtypeToUUID(b.ID),
		UserID:   PgtypeToUUID(b.UserID),
		TaskID:   PgtypeToUUIDPtr(b.TaskID),
		Title:    b.Title,
		StartsAt: b.StartsAt.Time.UTC(),
		EndsAt:   b.EndsAt.Time.UTC(),
	}
}
