package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/service"
)

type AgentHandler struct {
	svc         service.AgentService
	authBaseURL string
	httpClient  *http.Client
}

func NewAgentHandler(svc service.AgentService, authBaseURL string) *AgentHandler {
	return &AgentHandler{
		svc:         svc,
		authBaseURL: authBaseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (h *AgentHandler) RegisterRoutes(r chi.Router, jwtSecret []byte) {
	r.Group(func(protected chi.Router) {
		protected.Use(jwtx.AuthMiddleware(jwtSecret))

		// Chat & Specialized agent actions
		protected.Post("/agent/chat", h.Chat)
		protected.Post("/agent/plan-goal", h.PlanGoal)
		protected.Post("/agent/reschedule", h.Reschedule)

		// Conversations
		protected.Get("/agent/conversations", h.ListConversations)
		protected.Get("/agent/conversations/{id}", h.GetConversation)
		protected.Delete("/agent/conversations/{id}", h.DeleteConversation)

		// Proposals
		protected.Get("/agent/proposals", h.ListProposals)
		protected.Get("/agent/proposals/{id}", h.GetProposal)
		protected.Post("/agent/proposals/{id}/approve", h.ApproveProposal)
		protected.Post("/agent/proposals/{id}/reject", h.RejectProposal)

		// Runs & Tool calls history
		protected.Get("/agent/runs/{id}", h.GetAgentRun)
	})
}

// ----------------- Chat Handler -----------------

func (h *AgentHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.ChatRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authHeader := r.Header.Get("Authorization")
	userCtx := h.fetchUserContext(r.Context(), authHeader)

	resp, err := h.svc.Chat(r.Context(), userID, authHeader, req, userCtx)
	if err != nil {
		if errors.Is(err, service.ErrDailyMessageCapExceeded) {
			httpx.WriteError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Daily message limit reached")
			return
		}
		if errors.Is(err, service.ErrConversationNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

// ----------------- Plan Goal Handler -----------------

func (h *AgentHandler) PlanGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.PlanGoalRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authHeader := r.Header.Get("Authorization")
	userCtx := h.fetchUserContext(r.Context(), authHeader)

	resp, err := h.svc.PlanGoal(r.Context(), userID, authHeader, req, userCtx)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

// ----------------- Reschedule Handler -----------------

func (h *AgentHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.RescheduleRequest
	if err := httpx.ReadJSON(r, &req); err != nil && err.Error() != "EOF" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authHeader := r.Header.Get("Authorization")
	userCtx := h.fetchUserContext(r.Context(), authHeader)

	resp, err := h.svc.Reschedule(r.Context(), userID, authHeader, req, userCtx)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

// ----------------- Conversation Handlers -----------------

func (h *AgentHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = int32(val)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = int32(val)
		}
	}

	convs, err := h.svc.ListConversations(r.Context(), userID, limit, offset)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list conversations")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, convs)
}

func (h *AgentHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	convID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid conversation ID format")
		return
	}

	detail, err := h.svc.GetConversation(r.Context(), userID, convID)
	if err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get conversation")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, detail)
}

func (h *AgentHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	convID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid conversation ID format")
		return
	}

	if err := h.svc.DeleteConversation(r.Context(), userID, convID); err != nil {
		if errors.Is(err, service.ErrConversationNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Conversation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete conversation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ----------------- Proposal Handlers -----------------

func (h *AgentHandler) ListProposals(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = int32(val)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = int32(val)
		}
	}

	proposals, err := h.svc.ListProposals(r.Context(), userID, limit, offset)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list proposals")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, proposals)
}

func (h *AgentHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	proposalID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid proposal ID format")
		return
	}

	p, err := h.svc.GetProposal(r.Context(), userID, proposalID)
	if err != nil {
		if errors.Is(err, service.ErrProposalNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Proposal not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get proposal")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *AgentHandler) ApproveProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	proposalID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid proposal ID format")
		return
	}

	var req model.ApproveProposalRequest
	if err := httpx.ReadJSON(r, &req); err != nil && !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "EOF") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authHeader := r.Header.Get("Authorization")
	resp, err := h.svc.ApproveProposal(r.Context(), userID, proposalID, authHeader, req)
	if err != nil {
		if errors.Is(err, service.ErrProposalNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Proposal not found")
			return
		}
		if errors.Is(err, service.ErrProposalAlreadyDecided) {
			httpx.WriteError(w, http.StatusConflict, "already_decided", "Proposal has already been approved or rejected")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *AgentHandler) RejectProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	proposalID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid proposal ID format")
		return
	}

	resp, err := h.svc.RejectProposal(r.Context(), userID, proposalID)
	if err != nil {
		if errors.Is(err, service.ErrProposalNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Proposal not found")
			return
		}
		if errors.Is(err, service.ErrProposalAlreadyDecided) {
			httpx.WriteError(w, http.StatusConflict, "already_decided", "Proposal has already been approved or rejected")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

// ----------------- Agent Run Handler -----------------

func (h *AgentHandler) GetAgentRun(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	idStr := chi.URLParam(r, "id")
	runID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid agent run ID format")
		return
	}

	run, err := h.svc.GetAgentRun(r.Context(), userID, runID)
	if err != nil {
		if errors.Is(err, service.ErrAgentRunNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Agent run not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve agent run")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, run)
}

// ----------------- Helper: Fetch User Context -----------------

func (h *AgentHandler) fetchUserContext(ctx context.Context, authHeader string) *model.UserContext {
	if h.authBaseURL == "" || authHeader == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.authBaseURL+"/auth/me", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var userCtx model.UserContext
	if err := json.NewDecoder(resp.Body).Decode(&userCtx); err != nil {
		return nil
	}

	return &userCtx
}
