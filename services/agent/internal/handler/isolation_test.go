package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/agent/internal/handler"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentCrossUserIsolation verifies that User B cannot access or modify User A's
// conversations, proposals, or agent runs.
func TestAgentCrossUserIsolation(t *testing.T) {
	jwtSecret := []byte("test-agent-isolation-secret-32-bytes!!")
	userA := uuid.New()
	userB := uuid.New()

	convAID := uuid.New()
	proposalAID := uuid.New()
	runAID := uuid.New()

	// Mock service with strict user_id checking
	mockSvc := &mockAgentSvc{
		getConvFunc: func(ctx context.Context, userID, convID uuid.UUID) (*model.ConversationDetailResponse, error) {
			if userID != userA || convID != convAID {
				return nil, service.ErrConversationNotFound
			}
			return &model.ConversationDetailResponse{
				Conversation: model.ConversationResponse{ID: convAID, Title: "User A Chat"},
			}, nil
		},
		deleteConvFunc: func(ctx context.Context, userID, convID uuid.UUID) error {
			if userID != userA || convID != convAID {
				return service.ErrConversationNotFound
			}
			return nil
		},
		getPropFunc: func(ctx context.Context, userID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
			if userID != userA || proposalID != proposalAID {
				return nil, service.ErrProposalNotFound
			}
			return &model.ProposalResponse{ID: proposalAID, Status: "pending"}, nil
		},
		approvePropFunc: func(ctx context.Context, userID, proposalID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error) {
			if userID != userA || proposalID != proposalAID {
				return nil, service.ErrProposalNotFound
			}
			return &model.ProposalResponse{ID: proposalAID, Status: "approved"}, nil
		},
		rejectPropFunc: func(ctx context.Context, userID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
			if userID != userA || proposalID != proposalAID {
				return nil, service.ErrProposalNotFound
			}
			return &model.ProposalResponse{ID: proposalAID, Status: "rejected"}, nil
		},
		getRunFunc: func(ctx context.Context, userID, runID uuid.UUID) (*model.AgentRunResponse, error) {
			if userID != userA || runID != runAID {
				return nil, service.ErrAgentRunNotFound
			}
			return &model.AgentRunResponse{ID: runAID, Status: "completed"}, nil
		},
	}

	h := handler.NewAgentHandler(mockSvc, "http://auth-svc:8081")
	r := chi.NewRouter()
	h.RegisterRoutes(r, jwtSecret)

	tokenB, err := jwtx.SignToken(userB, jwtSecret, time.Hour)
	require.NoError(t, err)
	authHeaderB := "Bearer " + tokenB

	t.Run("User B cannot view User A's conversation", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/agent/conversations/%s", convAID), nil)
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot delete User A's conversation", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/agent/conversations/%s", convAID), nil)
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot view User A's proposal", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/agent/proposals/%s", proposalAID), nil)
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot approve User A's proposal", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/agent/proposals/%s/approve", proposalAID), strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot reject User A's proposal", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/agent/proposals/%s/reject", proposalAID), nil)
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot view User A's agent run", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/agent/runs/%s", runAID), nil)
		req.Header.Set("Authorization", authHeaderB)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
