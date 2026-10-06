package worker_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/agent/internal/repository"
	"github.com/planly/services/agent/internal/service"
	"github.com/planly/services/agent/internal/worker"
)

type mockQuerierForWorker struct {
	mu             sync.Mutex
	briefedRecords map[string]bool
}

func newMockQuerierForWorker() *mockQuerierForWorker {
	return &mockQuerierForWorker{
		briefedRecords: make(map[string]bool),
	}
}

func (m *mockQuerierForWorker) CountMessagesForUserSince(ctx context.Context, arg repository.CountMessagesForUserSinceParams) (int64, error) {
	return 0, nil
}
func (m *mockQuerierForWorker) CreateAgentRun(ctx context.Context, arg repository.CreateAgentRunParams) (repository.AgentRun, error) {
	return repository.AgentRun{}, nil
}
func (m *mockQuerierForWorker) CreateConversation(ctx context.Context, arg repository.CreateConversationParams) (repository.Conversation, error) {
	return repository.Conversation{}, nil
}
func (m *mockQuerierForWorker) CreateMessage(ctx context.Context, arg repository.CreateMessageParams) (repository.Message, error) {
	return repository.Message{}, nil
}
func (m *mockQuerierForWorker) CreateProposal(ctx context.Context, arg repository.CreateProposalParams) (repository.Proposal, error) {
	return repository.Proposal{}, nil
}
func (m *mockQuerierForWorker) CreateToolCall(ctx context.Context, arg repository.CreateToolCallParams) (repository.ToolCall, error) {
	return repository.ToolCall{}, nil
}
func (m *mockQuerierForWorker) DeleteConversation(ctx context.Context, arg repository.DeleteConversationParams) error {
	return nil
}
func (m *mockQuerierForWorker) GetAgentRunByID(ctx context.Context, arg repository.GetAgentRunByIDParams) (repository.AgentRun, error) {
	return repository.AgentRun{}, nil
}
func (m *mockQuerierForWorker) GetConversationByID(ctx context.Context, arg repository.GetConversationByIDParams) (repository.Conversation, error) {
	return repository.Conversation{}, nil
}
func (m *mockQuerierForWorker) GetProposalByID(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
	return repository.Proposal{}, nil
}
func (m *mockQuerierForWorker) HasBriefingForDate(ctx context.Context, arg repository.HasBriefingForDateParams) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := arg.UserID.String() + "_" + arg.BriefingDate.Time.Format("2006-01-02")
	return m.briefedRecords[key], nil
}
func (m *mockQuerierForWorker) ListConversationsByUserID(ctx context.Context, arg repository.ListConversationsByUserIDParams) ([]repository.Conversation, error) {
	return nil, nil
}
func (m *mockQuerierForWorker) ListMessagesByConversationID(ctx context.Context, conversationID pgtype.UUID) ([]repository.Message, error) {
	return nil, nil
}
func (m *mockQuerierForWorker) ListProposalsByUserID(ctx context.Context, arg repository.ListProposalsByUserIDParams) ([]repository.Proposal, error) {
	return nil, nil
}
func (m *mockQuerierForWorker) ListToolCallsByRunID(ctx context.Context, runID pgtype.UUID) ([]repository.ToolCall, error) {
	return nil, nil
}
func (m *mockQuerierForWorker) RecordBriefing(ctx context.Context, arg repository.RecordBriefingParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := arg.UserID.String() + "_" + arg.BriefingDate.Time.Format("2006-01-02")
	m.briefedRecords[key] = true
	return nil
}
func (m *mockQuerierForWorker) UpdateAgentRunStatus(ctx context.Context, arg repository.UpdateAgentRunStatusParams) (repository.AgentRun, error) {
	return repository.AgentRun{}, nil
}
func (m *mockQuerierForWorker) UpdateProposalStatus(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error) {
	return repository.Proposal{}, nil
}

func TestBriefingWorker_ProcessCandidates(t *testing.T) {
	internalKey := "secret-internal-key"
	userID := uuid.New()

	// 1. Mock Auth Server
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Key") != internalKey {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		candidates := []worker.BriefingUserCandidate{
			{
				ID:        userID,
				Email:     "student@example.com",
				FullName:  "Alex Student",
				Persona:   "student",
				Timezone:  "UTC",
				WorkStart: "00:00", // Start immediately
				WorkEnd:   "23:59",
			},
		}
		_ = json.NewEncoder(w).Encode(candidates)
	}))
	defer authServer.Close()

	// 2. Mock Notify Server
	var (
		mu           sync.Mutex
		notifsPosted []worker.InternalNotificationRequest
	)
	notifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Key") != internalKey {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var req worker.InternalNotificationRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		notifsPosted = append(notifsPosted, req)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer notifyServer.Close()

	mockRepo := newMockQuerierForWorker()
	fakeLLM := service.NewFakeLLM()
	fakeLLM.DefaultReply = "Rise and shine! You have 2 assignments to finish today."

	w := worker.NewBriefingWorker(
		mockRepo,
		fakeLLM,
		authServer.URL,
		notifyServer.URL,
		internalKey,
		10*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go w.Start(ctx)

	// Wait briefly for worker to complete the initial pass
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	count := len(notifsPosted)
	mu.Unlock()

	if count != 1 {
		t.Fatalf("expected exactly 1 briefing notification sent, got %d", count)
	}

	mu.Lock()
	firstNotif := notifsPosted[0]
	mu.Unlock()

	if *firstNotif.UserID != userID {
		t.Fatalf("expected notification for user %s, got %s", userID, firstNotif.UserID)
	}
	if firstNotif.Kind != "briefing" {
		t.Fatalf("expected kind 'briefing', got %s", firstNotif.Kind)
	}
}
