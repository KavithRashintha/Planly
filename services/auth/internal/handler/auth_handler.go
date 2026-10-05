package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/auth/internal/model"
	"github.com/planly/services/auth/internal/service"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router, jwtSecret []byte, internalKey string) {
	// Public auth routes
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)

	// Protected user profile routes
	r.Group(func(protected chi.Router) {
		protected.Use(jwtx.AuthMiddleware(jwtSecret))
		protected.Get("/auth/me", h.GetMe)
		protected.Put("/auth/me", h.UpdateMe)
	})

	// Internal service-to-service routes
	r.Group(func(internal chi.Router) {
		internal.Use(jwtx.RequireInternalKey(internalKey))
		internal.Get("/internal/users/briefing-candidates", h.ListBriefingCandidates)
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateEmail) {
			httpx.WriteError(w, http.StatusConflict, "duplicate_email", "Email is already registered")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to register user")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authResp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to login")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, authResp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authResp, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "Invalid, expired, or revoked refresh token")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to refresh token")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, authResp)
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found in context")
		return
	}

	user, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve profile")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found in context")
		return
	}

	var req model.UpdateProfileRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.svc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update profile")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) ListBriefingCandidates(w http.ResponseWriter, r *http.Request) {
	candidates, err := h.svc.ListBriefingCandidates(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve briefing candidates")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, candidates)
}
