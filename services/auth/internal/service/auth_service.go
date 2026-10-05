package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/auth/internal/model"
	"github.com/planly/services/auth/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail      = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid, expired, or revoked refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrValidation          = errors.New("validation error")
)

type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (*model.UserResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
	Refresh(ctx context.Context, req model.RefreshRequest) (*model.AuthResponse, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req model.UpdateProfileRequest) (*model.UserResponse, error)
	ListBriefingCandidates(ctx context.Context) ([]model.UserResponse, error)
}

type authService struct {
	repo      repository.Querier
	jwtSecret []byte
}

func NewAuthService(repo repository.Querier, jwtSecret []byte) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.UserResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return nil, fmt.Errorf("%w: invalid email address format", ErrValidation)
	}

	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters long", ErrValidation)
	}

	if strings.TrimSpace(req.FullName) == "" {
		return nil, fmt.Errorf("%w: full name is required", ErrValidation)
	}

	persona := req.Persona
	if persona == "" {
		persona = "employee"
	}
	if persona != "student" && persona != "undergraduate" && persona != "employee" {
		return nil, fmt.Errorf("%w: invalid persona, must be student, undergraduate, or employee", ErrValidation)
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("%w: invalid timezone: %s", ErrValidation, timezone)
	}

	workStart, err := StringToTime(req.WorkStart)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid work_start format", ErrValidation)
	}

	workEnd, err := StringToTime(req.WorkEnd)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid work_end format", ErrValidation)
	}

	// Bcrypt hash password (cost 12 as per NFR-2)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     strings.TrimSpace(req.FullName),
		Persona:      persona,
		Timezone:     timezone,
		WorkStart:    workStart,
		WorkEnd:      workEnd,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	resp := MapUserToResponse(user)
	return &resp, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	userID := PgtypeToUUID(user.ID)

	// Access token (15 minutes)
	accessToken, err := jwtx.SignToken(userID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token (7 days)
	rawRefreshToken, tokenHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.repo.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    900, // 15 minutes
		User:         MapUserToResponse(user),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, req model.RefreshRequest) (*model.AuthResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := HashToken(req.RefreshToken)
	storedToken, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// If token was revoked, revoke all tokens for this user as a security safeguard
	if storedToken.Revoked {
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, storedToken.UserID)
		return nil, ErrInvalidRefreshToken
	}

	// Check expiration
	if time.Now().UTC().After(storedToken.ExpiresAt.Time) {
		return nil, ErrInvalidRefreshToken
	}

	// Revoke current token (rotation)
	if err := s.repo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Fetch user
	user, err := s.repo.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user for refresh token: %w", err)
	}

	userID := PgtypeToUUID(user.ID)

	// Issue new token pair
	newAccessToken, err := jwtx.SignToken(userID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to sign new access token: %w", err)
	}

	newRawRefreshToken, newTokenHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	newExpiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	_, err = s.repo.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: newExpiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	return &model.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		ExpiresIn:    900,
		User:         MapUserToResponse(user),
	}, nil
}

func (s *authService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, UUIDToPgtype(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	resp := MapUserToResponse(user)
	return &resp, nil
}

func (s *authService) UpdateProfile(ctx context.Context, userID uuid.UUID, req model.UpdateProfileRequest) (*model.UserResponse, error) {
	if strings.TrimSpace(req.FullName) == "" {
		return nil, fmt.Errorf("%w: full name is required", ErrValidation)
	}

	persona := req.Persona
	if persona != "student" && persona != "undergraduate" && persona != "employee" {
		return nil, fmt.Errorf("%w: invalid persona, must be student, undergraduate, or employee", ErrValidation)
	}

	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return nil, fmt.Errorf("%w: invalid timezone: %s", ErrValidation, req.Timezone)
	}

	workStart, err := StringToTime(req.WorkStart)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid work_start format", ErrValidation)
	}

	workEnd, err := StringToTime(req.WorkEnd)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid work_end format", ErrValidation)
	}

	user, err := s.repo.UpdateUserProfile(ctx, repository.UpdateUserProfileParams{
		ID:        UUIDToPgtype(userID),
		FullName:  strings.TrimSpace(req.FullName),
		Persona:   persona,
		Timezone:  req.Timezone,
		WorkStart: workStart,
		WorkEnd:   workEnd,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	resp := MapUserToResponse(user)
	return &resp, nil
}

func (s *authService) ListBriefingCandidates(ctx context.Context) ([]model.UserResponse, error) {
	candidates, err := s.repo.ListBriefingCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list briefing candidates: %w", err)
	}

	resp := make([]model.UserResponse, len(candidates))
	for i, c := range candidates {
		resp[i] = model.UserResponse{
			ID:        PgtypeToUUID(c.ID),
			Email:     c.Email,
			FullName:  c.FullName,
			Persona:   c.Persona,
			Timezone:  c.Timezone,
			WorkStart: TimeToString(c.WorkStart),
			WorkEnd:   TimeToString(c.WorkEnd),
		}
	}
	return resp, nil
}
