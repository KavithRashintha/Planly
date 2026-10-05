package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/auth/internal/model"
	"github.com/planly/services/auth/internal/repository"
)

// UUIDToPgtype converts uuid.UUID to pgtype.UUID
func UUIDToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// PgtypeToUUID converts pgtype.UUID to uuid.UUID
func PgtypeToUUID(p pgtype.UUID) uuid.UUID {
	return uuid.UUID(p.Bytes)
}

// TimeToString converts pgtype.Time to "HH:MM" string
func TimeToString(t pgtype.Time) string {
	if !t.Valid {
		return "09:00"
	}
	hours := t.Microseconds / (3600 * 1000000)
	minutes := (t.Microseconds % (3600 * 1000000)) / (60 * 1000000)
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// StringToTime converts "HH:MM" string to pgtype.Time
func StringToTime(s string) (pgtype.Time, error) {
	if s == "" {
		s = "09:00"
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}, fmt.Errorf("invalid time format, expected HH:MM (e.g. 09:00): %w", err)
	}
	micros := int64(parsed.Hour())*3600*1000000 + int64(parsed.Minute())*60*1000000
	return pgtype.Time{Microseconds: micros, Valid: true}, nil
}

// GenerateRefreshToken generates a secure random 32-byte string and its SHA-256 hash
func GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}
	rawToken = hex.EncodeToString(bytes)
	tokenHash = HashToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashToken computes the SHA-256 hash of a raw token string
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// MapUserToResponse maps repository.User to model.UserResponse
func MapUserToResponse(u repository.User) model.UserResponse {
	return model.UserResponse{
		ID:        PgtypeToUUID(u.ID),
		Email:     u.Email,
		FullName:  u.FullName,
		Persona:   u.Persona,
		Timezone:  u.Timezone,
		WorkStart: TimeToString(u.WorkStart),
		WorkEnd:   TimeToString(u.WorkEnd),
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}
