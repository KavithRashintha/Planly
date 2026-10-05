package model

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	Persona   string `json:"persona,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	WorkStart string `json:"work_start,omitempty"`
	WorkEnd   string `json:"work_end,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateProfileRequest struct {
	FullName  string `json:"full_name"`
	Persona   string `json:"persona"`
	Timezone  string `json:"timezone"`
	WorkStart string `json:"work_start"`
	WorkEnd   string `json:"work_end"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Persona   string    `json:"persona"`
	Timezone  string    `json:"timezone"`
	WorkStart string    `json:"work_start"`
	WorkEnd   string    `json:"work_end"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // in seconds, e.g. 900
	User         UserResponse `json:"user"`
}
