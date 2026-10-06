package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/agent/internal/handler"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/service"
)

type mockAgentSvc struct {
	chatFunc        func(ctx context.Context, userID uuid.UUID, authHeader string, req model.ChatRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	planGoalFunc    func(ctx context.Context, userID uuid.UUID, authHeader string, req model.PlanGoalRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	rescheduleFunc  func(ctx context.Context, userID uuid.UUID, authHeader string, req model.RescheduleRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	listConvsFunc   func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ConversationResponse, error)
	getConvFunc     func(ctx context.Context, userID uuid.UUID, convID uuid.UUID) (*model.ConversationDetailResponse, error)
	deleteConvFunc  func(ctx context.Context, userID uuid.UUID, convID uuid.UUID) error
	listPropsFunc   func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ProposalResponse, error)
	getPropFunc     func(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error)
	approvePropFunc func(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error)
	rejectPropFunc  func(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error)
	getRunFunc      func(ctx context.Context, userID uuid.UUID, runID uuid.UUID) (*model.AgentRunResponse, error)
}

func (m *mockAgentSvc) Chat(ctx context.Context, userID uuid.UUID, authHeader string, req model.ChatRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, userID, authHeader, req, userCtx)
	}
	return &model.ChatResponse{ConversationID: uuid.New(), Reply: "mock reply"}, nil
}
func (m *mockAgentSvc) PlanGoal(ctx context.Context, userID uuid.UUID, authHeader string, req model.PlanGoalRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	if m.planGoalFunc != nil {
		return m.planGoalFunc(ctx, userID, authHeader, req, userCtx)
	}
	return &model.ChatResponse{ConversationID: uuid.New(), Reply: "goal planned"}, nil
}
func (m *mockAgentSvc) Reschedule(ctx context.Context, userID uuid.UUID, authHeader string, req model.RescheduleRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	if m.rescheduleFunc != nil {
		return m.rescheduleFunc(ctx, userID, authHeader, req, userCtx)
	}
	return &model.ChatResponse{ConversationID: uuid.New(), Reply: "rescheduled"}, nil
}
func (m *mockAgentSvc) ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ConversationResponse, error) {
	if m.listConvsFunc != nil {
		return m.listConvsFunc(ctx, userID, limit, offset)
	}
	return []model.ConversationResponse{}, nil
}
func (m *mockAgentSvc) GetConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) (*model.ConversationDetailResponse, error) {
	if m.getConvFunc != nil {
		return m.getConvFunc(ctx, userID, convID)
	}
	return &model.ConversationDetailResponse{
		Conversation: model.ConversationResponse{ID: convID, Title: "Chat"},
	}, nil
}
func (m *mockAgentSvc) DeleteConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) error {
	if m.deleteConvFunc != nil {
		return m.deleteConvFunc(ctx, userID, convID)
	}
	return nil
}
func (m *mockAgentSvc) ListProposals(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ProposalResponse, error) {
	if m.listPropsFunc != nil {
		return m.listPropsFunc(ctx, userID, limit, offset)
	}
	return []model.ProposalResponse{}, nil
}
func (m *mockAgentSvc) GetProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
	if m.getPropFunc != nil {
		return m.getPropFunc(ctx, userID, proposalID)
	}
	return &model.ProposalResponse{ID: proposalID, Status: "pending"}, nil
}
func (m *mockAgentSvc) ApproveProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error) {
	if m.approvePropFunc != nil {
		return m.approvePropFunc(ctx, userID, proposalID, authHeader, req)
	}
	return &model.ProposalResponse{ID: proposalID, Status: "approved"}, nil
}
func (m *mockAgentSvc) RejectProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
	if m.rejectPropFunc != nil {
		return m.rejectPropFunc(ctx, userID, proposalID)
	}
	return &model.ProposalResponse{ID: proposalID, Status: "rejected"}, nil
}
func (m *mockAgentSvc) GetAgentRun(ctx context.Context, userID uuid.UUID, runID uuid.UUID) (*model.AgentRunResponse, error) {
	if m.getRunFunc != nil {
		return m.getRunFunc(ctx, userID, runID)
	}
	return &model.AgentRunResponse{ID: runID, Status: "completed"}, nil
}

func setupTestRouter(svc service.AgentService, secret []byte) chi.Router {
	r := chi.NewRouter()
	h := handler.NewAgentHandler(svc, "")
	h.RegisterRoutes(r, secret)
	return r
}

func TestHandler_Chat_SuccessAndUnauthorized(t *testing.T) {
	jwtSecret := []byte("super-secret-key-at-least-32-bytes-long!")
	mockSvc := &mockAgentSvc{}
	router := setupTestRouter(mockSvc, jwtSecret)

	userID := uuid.New()
	token, _ := jwtx.SignToken(userID, jwtSecret, time.Hour)

	// 1. Without Token -> 401
	reqBody, _ := json.Marshal(model.ChatRequest{Message: "Hello"})
	req := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}

	// 2. With Token -> 200
	req = httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Chat_RateLimit429(t *testing.T) {
	jwtSecret := []byte("super-secret-key-at-least-32-bytes-long!")
	mockSvc := &mockAgentSvc{
		chatFunc: func(ctx context.Context, userID uuid.UUID, authHeader string, req model.ChatRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
			return nil, service.ErrDailyMessageCapExceeded
		},
	}
	router := setupTestRouter(mockSvc, jwtSecret)

	userID := uuid.New()
	token, _ := jwtx.SignToken(userID, jwtSecret, time.Hour)

	reqBody, _ := json.Marshal(model.ChatRequest{Message: "Spamming"})
	req := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", w.Code)
	}
}

func TestHandler_Proposals_ApproveAndReject_Conflict(t *testing.T) {
	jwtSecret := []byte("super-secret-key-at-least-32-bytes-long!")
	propID := uuid.New()

	mockSvc := &mockAgentSvc{
		approvePropFunc: func(ctx context.Context, userID, pID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error) {
			return nil, service.ErrProposalAlreadyDecided
		},
		rejectPropFunc: func(ctx context.Context, userID, pID uuid.UUID) (*model.ProposalResponse, error) {
			return nil, service.ErrProposalAlreadyDecided
		},
	}
	router := setupTestRouter(mockSvc, jwtSecret)

	userID := uuid.New()
	token, _ := jwtx.SignToken(userID, jwtSecret, time.Hour)

	// Approve already decided proposal -> 409 Conflict
	req := httptest.NewRequest(http.MethodPost, "/agent/proposals/"+propID.String()+"/approve", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on approve, got %d", w.Code)
	}

	// Reject already decided proposal -> 409 Conflict
	req = httptest.NewRequest(http.MethodPost, "/agent/proposals/"+propID.String()+"/reject", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on reject, got %d", w.Code)
	}
}

func TestHandler_Conversations_GetNotFound(t *testing.T) {
	jwtSecret := []byte("super-secret-key-at-least-32-bytes-long!")
	mockSvc := &mockAgentSvc{
		getConvFunc: func(ctx context.Context, userID, convID uuid.UUID) (*model.ConversationDetailResponse, error) {
			return nil, service.ErrConversationNotFound
		},
	}
	router := setupTestRouter(mockSvc, jwtSecret)

	userID := uuid.New()
	token, _ := jwtx.SignToken(userID, jwtSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/agent/conversations/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestHandler_PlanGoal_And_Reschedule(t *testing.T) {
	jwtSecret := []byte("super-secret-key-at-least-32-bytes-long!")
	mockSvc := &mockAgentSvc{}
	router := setupTestRouter(mockSvc, jwtSecret)

	userID := uuid.New()
	token, _ := jwtx.SignToken(userID, jwtSecret, time.Hour)

	// Plan Goal
	goalReq, _ := json.Marshal(model.PlanGoalRequest{
		Goal:     "Complete Compiler Course Project",
		Deadline: time.Now().Add(72 * time.Hour),
	})
	req := httptest.NewRequest(http.MethodPost, "/agent/plan-goal", bytes.NewReader(goalReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for plan-goal, got %d", w.Code)
	}

	// Reschedule
	reschedReq, _ := json.Marshal(model.RescheduleRequest{})
	req = httptest.NewRequest(http.MethodPost, "/agent/reschedule", bytes.NewReader(reschedReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for reschedule, got %d", w.Code)
	}
}
