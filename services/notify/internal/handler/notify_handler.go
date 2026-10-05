package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/notify/internal/model"
	"github.com/planly/services/notify/internal/service"
)

type NotifyHandler struct {
	svc service.NotifyService
}

func NewNotifyHandler(svc service.NotifyService) *NotifyHandler {
	return &NotifyHandler{svc: svc}
}

func (h *NotifyHandler) RegisterRoutes(r chi.Router, jwtSecret []byte, internalKey string) {
	// User protected routes
	r.Group(func(protected chi.Router) {
		protected.Use(jwtx.AuthMiddleware(jwtSecret))

		protected.Post("/reminders", h.CreateReminder)
		protected.Get("/notifications", h.ListNotifications)
		protected.Post("/notifications/{id}/read", h.MarkNotificationRead)
		protected.Get("/notifications/unread-count", h.GetUnreadCount)
	})

	// Internal service-to-service routes
	r.Group(func(internal chi.Router) {
		internal.Use(jwtx.RequireInternalKey(internalKey))

		internal.Post("/internal/notifications", h.CreateInternalNotification)
	})
}

func (h *NotifyHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.CreateReminderRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateReminder(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create reminder")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *NotifyHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	q := r.URL.Query()
	limit := int32(20)
	offset := int32(0)

	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil {
			limit = int32(l)
		}
	}
	if oStr := q.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil {
			offset = int32(o)
		}
	}

	notifications, total, err := h.svc.ListNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list notifications")
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
	_ = httpx.WriteJSON(w, http.StatusOK, notifications)
}

func (h *NotifyHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid notification ID format")
		return
	}

	resp, err := h.svc.MarkAsRead(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Notification not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to mark notification as read")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *NotifyHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	resp, err := h.svc.GetUnreadCount(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get unread count")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *NotifyHandler) CreateInternalNotification(w http.ResponseWriter, r *http.Request) {
	var req model.CreateNotificationRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.UserID == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "user_id is required for internal notification")
		return
	}

	resp, err := h.svc.CreateNotification(r.Context(), *req.UserID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create internal notification")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}
