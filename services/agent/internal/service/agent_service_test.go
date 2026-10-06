package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/repository"
	"github.com/planly/services/agent/internal/service"
)

// Mock Querier
type mockQuerier struct {
	countMessages   func(ctx context.Context, arg repository.CountMessagesForUserSinceParams) (int64, error)
	createConv      func(ctx context.Context, arg repository.CreateConversationParams) (repository.Conversation, error)
	getConv         func(ctx context.Context, arg repository.GetConversationByIDParams) (repository.Conversation, error)
	listConvs       func(ctx context.Context, arg repository.ListConversationsByUserIDParams) ([]repository.Conversation, error)
	deleteConv      func(ctx context.Context, arg repository.DeleteConversationParams) error
	createMsg       func(ctx context.Context, arg repository.CreateMessageParams) (repository.Message, error)
	listMsgs        func(ctx context.Context, conversationID pgtype.UUID) ([]repository.Message, error)
	createRun       func(ctx context.Context, arg repository.CreateAgentRunParams) (repository.AgentRun, error)
	updateRun       func(ctx context.Context, arg repository.UpdateAgentRunStatusParams) (repository.AgentRun, error)
	getRun          func(ctx context.Context, arg repository.GetAgentRunByIDParams) (repository.AgentRun, error)
	createToolCall  func(ctx context.Context, arg repository.CreateToolCallParams) (repository.ToolCall, error)
	listToolCalls   func(ctx context.Context, runID pgtype.UUID) ([]repository.ToolCall, error)
	createProposal  func(ctx context.Context, arg repository.CreateProposalParams) (repository.Proposal, error)
	getProposal     func(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error)
	listProposals   func(ctx context.Context, arg repository.ListProposalsByUserIDParams) ([]repository.Proposal, error)
	updateProposal  func(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error)
	hasBriefing     func(ctx context.Context, arg repository.HasBriefingForDateParams) (bool, error)
	recordBriefing  func(ctx context.Context, arg repository.RecordBriefingParams) error
}

func (m *mockQuerier) CountMessagesForUserSince(ctx context.Context, arg repository.CountMessagesForUserSinceParams) (int64, error) {
	if m.countMessages != nil {
		return m.countMessages(ctx, arg)
	}
	return 0, nil
}
func (m *mockQuerier) CreateAgentRun(ctx context.Context, arg repository.CreateAgentRunParams) (repository.AgentRun, error) {
	if m.createRun != nil {
		return m.createRun(ctx, arg)
	}
	return repository.AgentRun{ID: service.UUIDToPgtype(uuid.New()), Status: "running"}, nil
}
func (m *mockQuerier) CreateConversation(ctx context.Context, arg repository.CreateConversationParams) (repository.Conversation, error) {
	if m.createConv != nil {
		return m.createConv(ctx, arg)
	}
	return repository.Conversation{ID: service.UUIDToPgtype(uuid.New()), Title: arg.Title}, nil
}
func (m *mockQuerier) CreateMessage(ctx context.Context, arg repository.CreateMessageParams) (repository.Message, error) {
	if m.createMsg != nil {
		return m.createMsg(ctx, arg)
	}
	return repository.Message{ID: service.UUIDToPgtype(uuid.New()), Role: arg.Role}, nil
}
func (m *mockQuerier) CreateProposal(ctx context.Context, arg repository.CreateProposalParams) (repository.Proposal, error) {
	if m.createProposal != nil {
		return m.createProposal(ctx, arg)
	}
	return repository.Proposal{ID: service.UUIDToPgtype(uuid.New()), Summary: arg.Summary, Status: arg.Status}, nil
}
func (m *mockQuerier) CreateToolCall(ctx context.Context, arg repository.CreateToolCallParams) (repository.ToolCall, error) {
	if m.createToolCall != nil {
		return m.createToolCall(ctx, arg)
	}
	return repository.ToolCall{ID: service.UUIDToPgtype(uuid.New())}, nil
}
func (m *mockQuerier) DeleteConversation(ctx context.Context, arg repository.DeleteConversationParams) error {
	if m.deleteConv != nil {
		return m.deleteConv(ctx, arg)
	}
	return nil
}
func (m *mockQuerier) GetAgentRunByID(ctx context.Context, arg repository.GetAgentRunByIDParams) (repository.AgentRun, error) {
	if m.getRun != nil {
		return m.getRun(ctx, arg)
	}
	return repository.AgentRun{ID: arg.ID, Status: "completed"}, nil
}
func (m *mockQuerier) GetConversationByID(ctx context.Context, arg repository.GetConversationByIDParams) (repository.Conversation, error) {
	if m.getConv != nil {
		return m.getConv(ctx, arg)
	}
	return repository.Conversation{ID: arg.ID, Title: "Existing Chat"}, nil
}
func (m *mockQuerier) GetProposalByID(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
	if m.getProposal != nil {
		return m.getProposal(ctx, arg)
	}
	return repository.Proposal{ID: arg.ID, Status: "pending"}, nil
}
func (m *mockQuerier) HasBriefingForDate(ctx context.Context, arg repository.HasBriefingForDateParams) (bool, error) {
	if m.hasBriefing != nil {
		return m.hasBriefing(ctx, arg)
	}
	return false, nil
}
func (m *mockQuerier) ListConversationsByUserID(ctx context.Context, arg repository.ListConversationsByUserIDParams) ([]repository.Conversation, error) {
	if m.listConvs != nil {
		return m.listConvs(ctx, arg)
	}
	return nil, nil
}
func (m *mockQuerier) ListMessagesByConversationID(ctx context.Context, conversationID pgtype.UUID) ([]repository.Message, error) {
	if m.listMsgs != nil {
		return m.listMsgs(ctx, conversationID)
	}
	return nil, nil
}
func (m *mockQuerier) ListProposalsByUserID(ctx context.Context, arg repository.ListProposalsByUserIDParams) ([]repository.Proposal, error) {
	if m.listProposals != nil {
		return m.listProposals(ctx, arg)
	}
	return nil, nil
}
func (m *mockQuerier) ListToolCallsByRunID(ctx context.Context, runID pgtype.UUID) ([]repository.ToolCall, error) {
	if m.listToolCalls != nil {
		return m.listToolCalls(ctx, runID)
	}
	return nil, nil
}
func (m *mockQuerier) RecordBriefing(ctx context.Context, arg repository.RecordBriefingParams) error {
	if m.recordBriefing != nil {
		return m.recordBriefing(ctx, arg)
	}
	return nil
}
func (m *mockQuerier) UpdateAgentRunStatus(ctx context.Context, arg repository.UpdateAgentRunStatusParams) (repository.AgentRun, error) {
	if m.updateRun != nil {
		return m.updateRun(ctx, arg)
	}
	return repository.AgentRun{ID: arg.ID, Status: arg.Status}, nil
}
func (m *mockQuerier) UpdateProposalStatus(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error) {
	if m.updateProposal != nil {
		return m.updateProposal(ctx, arg)
	}
	return repository.Proposal{ID: arg.ID, Status: arg.Status}, nil
}

// Mock Planner
type mockPlannerClient struct {
	tasksCreated     []service.CreateTaskDTO
	timeBlockCreated []service.CreateTimeBlockDTO
	tasksUpdated     map[uuid.UUID]service.UpdateTaskDTO
	tasksDeleted     []uuid.UUID
	listTasksFunc    func(ctx context.Context, authHeader string, queryParams map[string]string) ([]service.TaskDTO, error)
}

func newMockPlanner() *mockPlannerClient {
	return &mockPlannerClient{
		tasksUpdated: make(map[uuid.UUID]service.UpdateTaskDTO),
	}
}

func (p *mockPlannerClient) ListTasks(ctx context.Context, authHeader string, queryParams map[string]string) ([]service.TaskDTO, error) {
	if p.listTasksFunc != nil {
		return p.listTasksFunc(ctx, authHeader, queryParams)
	}
	return []service.TaskDTO{
		{ID: uuid.New(), Title: "Finish Assignment", Status: "todo"},
	}, nil
}
func (p *mockPlannerClient) GetTask(ctx context.Context, authHeader string, taskID uuid.UUID) (*service.TaskDTO, error) {
	return &service.TaskDTO{ID: taskID, Title: "Existing Task", Status: "todo"}, nil
}
func (p *mockPlannerClient) ListProjects(ctx context.Context, authHeader string) ([]service.ProjectDTO, error) {
	return []service.ProjectDTO{{ID: uuid.New(), Name: "Uni Work"}}, nil
}
func (p *mockPlannerClient) ListTimeBlocks(ctx context.Context, authHeader string, from, to string) ([]service.TimeBlockDTO, error) {
	return nil, nil
}
func (p *mockPlannerClient) GetWorkload(ctx context.Context, authHeader string, from, to string) (*service.WorkloadDTO, error) {
	return &service.WorkloadDTO{From: from, To: to, Days: []service.WorkloadDayDTO{}}, nil
}
func (p *mockPlannerClient) CreateTask(ctx context.Context, authHeader string, req service.CreateTaskDTO) (*service.TaskDTO, error) {
	p.tasksCreated = append(p.tasksCreated, req)
	return &service.TaskDTO{ID: uuid.New(), Title: req.Title, Status: req.Status}, nil
}
func (p *mockPlannerClient) UpdateTask(ctx context.Context, authHeader string, taskID uuid.UUID, req service.UpdateTaskDTO) (*service.TaskDTO, error) {
	p.tasksUpdated[taskID] = req
	return &service.TaskDTO{ID: taskID, Title: req.Title, Status: req.Status}, nil
}
func (p *mockPlannerClient) DeleteTask(ctx context.Context, authHeader string, taskID uuid.UUID) error {
	p.tasksDeleted = append(p.tasksDeleted, taskID)
	return nil
}
func (p *mockPlannerClient) CreateTimeBlock(ctx context.Context, authHeader string, req service.CreateTimeBlockDTO) (*service.TimeBlockDTO, error) {
	p.timeBlockCreated = append(p.timeBlockCreated, req)
	return &service.TimeBlockDTO{ID: uuid.New(), Title: req.Title}, nil
}

// ----------------- TESTS -----------------

func TestAgentChat_DirectAnswer(t *testing.T) {
	mockRepo := &mockQuerier{}
	fakeLLM := service.NewFakeLLM()
	fakeLLM.DefaultReply = "I can certainly help plan your schedule!"
	planner := newMockPlanner()

	svc := service.NewAgentService(mockRepo, fakeLLM, planner, 50)
	userID := uuid.New()

	resp, err := svc.Chat(context.Background(), userID, "Bearer test", model.ChatRequest{
		Message: "Hello assistant!",
	}, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Reply != "I can certainly help plan your schedule!" {
		t.Fatalf("unexpected reply: %s", resp.Reply)
	}
	if len(resp.Proposals) != 0 {
		t.Fatalf("expected 0 proposals, got %d", len(resp.Proposals))
	}
}

func TestAgentChat_ReadToolExecution(t *testing.T) {
	mockRepo := &mockQuerier{}
	fakeLLM := service.NewFakeLLM()
	planner := newMockPlanner()

	// First turn: model requests list_tasks
	fakeLLM.QueueResponse(&service.LLMResponse{
		ToolCalls: []service.LLMToolCall{
			{
				ID:    "call_1",
				Name:  "list_tasks",
				Input: json.RawMessage(`{"status": "todo"}`),
			},
		},
		StopReason:   "tool_use",
		InputTokens:  100,
		OutputTokens: 25,
	})
	// Second turn: model answers with summary
	fakeLLM.QueueResponse(&service.LLMResponse{
		Text:         "You have 1 pending task: Finish Assignment.",
		StopReason:   "end_turn",
		InputTokens:  150,
		OutputTokens: 20,
	})

	svc := service.NewAgentService(mockRepo, fakeLLM, planner, 50)
	userID := uuid.New()

	resp, err := svc.Chat(context.Background(), userID, "Bearer test", model.ChatRequest{
		Message: "What tasks do I have?",
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Reply != "You have 1 pending task: Finish Assignment." {
		t.Fatalf("unexpected reply: %s", resp.Reply)
	}
}

func TestAgentChat_WriteToolStagesProposal_DoesNotCallPlanner(t *testing.T) {
	var proposalCreated bool
	mockRepo := &mockQuerier{
		createProposal: func(ctx context.Context, arg repository.CreateProposalParams) (repository.Proposal, error) {
			proposalCreated = true
			return repository.Proposal{
				ID:        service.UUIDToPgtype(uuid.New()),
				Summary:   arg.Summary,
				Status:    "pending",
				CreatedAt: service.TimeToPgtype(time.Now()),
			}, nil
		},
	}
	fakeLLM := service.NewFakeLLM()
	planner := newMockPlanner()

	// Model calls write tool: create_task
	fakeLLM.QueueResponse(&service.LLMResponse{
		ToolCalls: []service.LLMToolCall{
			{
				ID:    "call_write_1",
				Name:  "create_task",
				Input: json.RawMessage(`{"title": "Study Operating Systems", "priority": 1}`),
			},
		},
		StopReason:   "tool_use",
		InputTokens:  100,
		OutputTokens: 30,
	})
	// Model returns final explanation
	fakeLLM.QueueResponse(&service.LLMResponse{
		Text:         "I have prepared a proposal to create 'Study Operating Systems' task.",
		StopReason:   "end_turn",
		InputTokens:  150,
		OutputTokens: 30,
	})

	svc := service.NewAgentService(mockRepo, fakeLLM, planner, 50)
	userID := uuid.New()

	resp, err := svc.Chat(context.Background(), userID, "Bearer test", model.ChatRequest{
		Message: "Please schedule Study Operating Systems",
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CRITICAL REQUIREMENT: Write tools NEVER call planner during the agent loop!
	if len(planner.tasksCreated) != 0 {
		t.Fatalf("VIOLATION: planner.CreateTask was called directly by agent loop! Expected 0, got %d", len(planner.tasksCreated))
	}

	if !proposalCreated {
		t.Fatalf("expected proposal to be staged and stored in DB")
	}
	if len(resp.Proposals) != 1 {
		t.Fatalf("expected 1 proposal in response, got %d", len(resp.Proposals))
	}
	if resp.Proposals[0].Status != "pending" {
		t.Fatalf("expected proposal status 'pending', got '%s'", resp.Proposals[0].Status)
	}
}

func TestProposal_Approve_ExecutesAgainstPlanner(t *testing.T) {
	proposalID := uuid.New()
	userID := uuid.New()
	planner := newMockPlanner()

	actions := []model.ProposalAction{
		{
			ID:     "act_1",
			Tool:   "create_task",
			Input:  json.RawMessage(`{"title": "Prepare Exam", "priority": 1}`),
			Status: "pending",
		},
	}
	actionsBytes, _ := json.Marshal(actions)

	mockRepo := &mockQuerier{
		getProposal: func(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
			return repository.Proposal{
				ID:        service.UUIDToPgtype(proposalID),
				UserID:    service.UUIDToPgtype(userID),
				Actions:   actionsBytes,
				Status:    "pending",
				CreatedAt: service.TimeToPgtype(time.Now()),
			}, nil
		},
		updateProposal: func(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error) {
			return repository.Proposal{
				ID:        arg.ID,
				UserID:    service.UUIDToPgtype(userID),
				Actions:   arg.Actions,
				Status:    arg.Status,
				CreatedAt: service.TimeToPgtype(time.Now()),
				DecidedAt: arg.DecidedAt,
			}, nil
		},
	}

	svc := service.NewAgentService(mockRepo, service.NewFakeLLM(), planner, 50)

	resp, err := svc.ApproveProposal(context.Background(), userID, proposalID, "Bearer user-token", model.ApproveProposalRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "approved" {
		t.Fatalf("expected status 'approved', got '%s'", resp.Status)
	}
	// Verify that planner.CreateTask was executed upon approval!
	if len(planner.tasksCreated) != 1 {
		t.Fatalf("expected 1 task created in planner, got %d", len(planner.tasksCreated))
	}
	if planner.tasksCreated[0].Title != "Prepare Exam" {
		t.Fatalf("unexpected task title: %s", planner.tasksCreated[0].Title)
	}
}

func TestProposal_AlreadyDecided_ReturnsConflict(t *testing.T) {
	proposalID := uuid.New()
	userID := uuid.New()
	planner := newMockPlanner()

	mockRepo := &mockQuerier{
		getProposal: func(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
			return repository.Proposal{
				ID:      service.UUIDToPgtype(proposalID),
				UserID:  service.UUIDToPgtype(userID),
				Actions: []byte("[]"),
				Status:  "approved", // already decided!
			}, nil
		},
	}

	svc := service.NewAgentService(mockRepo, service.NewFakeLLM(), planner, 50)

	_, err := svc.ApproveProposal(context.Background(), userID, proposalID, "Bearer token", model.ApproveProposalRequest{})
	if !errors.Is(err, service.ErrProposalAlreadyDecided) {
		t.Fatalf("expected ErrProposalAlreadyDecided, got %v", err)
	}

	_, err = svc.RejectProposal(context.Background(), userID, proposalID)
	if !errors.Is(err, service.ErrProposalAlreadyDecided) {
		t.Fatalf("expected ErrProposalAlreadyDecided on reject, got %v", err)
	}
}

func TestProposal_Reject_DoesNotCallPlanner(t *testing.T) {
	proposalID := uuid.New()
	userID := uuid.New()
	planner := newMockPlanner()

	mockRepo := &mockQuerier{
		getProposal: func(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
			return repository.Proposal{
				ID:      service.UUIDToPgtype(proposalID),
				UserID:  service.UUIDToPgtype(userID),
				Actions: []byte(`[{"id":"a1","tool":"create_task","input":{}}]`),
				Status:  "pending",
			}, nil
		},
		updateProposal: func(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error) {
			return repository.Proposal{
				ID:        arg.ID,
				UserID:    service.UUIDToPgtype(userID),
				Status:    "rejected",
				CreatedAt: service.TimeToPgtype(time.Now()),
				DecidedAt: arg.DecidedAt,
			}, nil
		},
	}

	svc := service.NewAgentService(mockRepo, service.NewFakeLLM(), planner, 50)

	resp, err := svc.RejectProposal(context.Background(), userID, proposalID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "rejected" {
		t.Fatalf("expected 'rejected', got '%s'", resp.Status)
	}
	if len(planner.tasksCreated) != 0 {
		t.Fatalf("expected 0 planner calls on reject, got %d", len(planner.tasksCreated))
	}
}

func TestAgentChat_DailyMessageCapExceeded(t *testing.T) {
	mockRepo := &mockQuerier{
		countMessages: func(ctx context.Context, arg repository.CountMessagesForUserSinceParams) (int64, error) {
			return 50, nil // hit cap
		},
	}

	svc := service.NewAgentService(mockRepo, service.NewFakeLLM(), newMockPlanner(), 50)

	_, err := svc.Chat(context.Background(), uuid.New(), "Bearer token", model.ChatRequest{
		Message: "Another message",
	}, nil)

	if !errors.Is(err, service.ErrDailyMessageCapExceeded) {
		t.Fatalf("expected ErrDailyMessageCapExceeded, got %v", err)
	}
}
