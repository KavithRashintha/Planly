package evals_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/repository"
	"github.com/planly/services/agent/internal/service"
)

// ----------------- In-Memory Mock Repository -----------------

type inMemoryRepo struct {
	mu            sync.Mutex
	messageCount  int64
	conversations map[uuid.UUID]repository.Conversation
	messages      map[uuid.UUID][]repository.Message
	agentRuns     map[uuid.UUID]repository.AgentRun
	toolCalls     map[uuid.UUID][]repository.ToolCall
	proposals     map[uuid.UUID]repository.Proposal
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{
		conversations: make(map[uuid.UUID]repository.Conversation),
		messages:      make(map[uuid.UUID][]repository.Message),
		agentRuns:     make(map[uuid.UUID]repository.AgentRun),
		toolCalls:     make(map[uuid.UUID][]repository.ToolCall),
		proposals:     make(map[uuid.UUID]repository.Proposal),
	}
}

func (r *inMemoryRepo) CountMessagesForUserSince(ctx context.Context, arg repository.CountMessagesForUserSinceParams) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.messageCount, nil
}

func (r *inMemoryRepo) CreateConversation(ctx context.Context, arg repository.CreateConversationParams) (repository.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	conv := repository.Conversation{
		ID:        service.UUIDToPgtype(id),
		UserID:    arg.UserID,
		Title:     arg.Title,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	r.conversations[id] = conv
	return conv, nil
}

func (r *inMemoryRepo) GetConversationByID(ctx context.Context, arg repository.GetConversationByIDParams) (repository.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := service.PgtypeToUUID(arg.ID)
	conv, ok := r.conversations[id]
	if !ok {
		return repository.Conversation{}, fmt.Errorf("not found")
	}
	return conv, nil
}

func (r *inMemoryRepo) ListConversationsByUserID(ctx context.Context, arg repository.ListConversationsByUserIDParams) ([]repository.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []repository.Conversation
	for _, c := range r.conversations {
		list = append(list, c)
	}
	return list, nil
}

func (r *inMemoryRepo) DeleteConversation(ctx context.Context, arg repository.DeleteConversationParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conversations, service.PgtypeToUUID(arg.ID))
	return nil
}

func (r *inMemoryRepo) CreateMessage(ctx context.Context, arg repository.CreateMessageParams) (repository.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	convID := service.PgtypeToUUID(arg.ConversationID)
	msg := repository.Message{
		ID:             service.UUIDToPgtype(id),
		ConversationID: arg.ConversationID,
		Role:           arg.Role,
		Content:        arg.Content,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	r.messages[convID] = append(r.messages[convID], msg)
	return msg, nil
}

func (r *inMemoryRepo) ListMessagesByConversationID(ctx context.Context, conversationID pgtype.UUID) ([]repository.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.messages[service.PgtypeToUUID(conversationID)], nil
}

func (r *inMemoryRepo) CreateAgentRun(ctx context.Context, arg repository.CreateAgentRunParams) (repository.AgentRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	run := repository.AgentRun{
		ID:             service.UUIDToPgtype(id),
		ConversationID: arg.ConversationID,
		UserID:         arg.UserID,
		Trigger:        arg.Trigger,
		Status:         arg.Status,
		CreatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	r.agentRuns[id] = run
	return run, nil
}

func (r *inMemoryRepo) UpdateAgentRunStatus(ctx context.Context, arg repository.UpdateAgentRunStatusParams) (repository.AgentRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := service.PgtypeToUUID(arg.ID)
	run := r.agentRuns[id]
	run.Status = arg.Status
	run.Iterations = arg.Iterations
	run.InputTokens = arg.InputTokens
	run.OutputTokens = arg.OutputTokens
	run.Error = arg.Error
	r.agentRuns[id] = run
	return run, nil
}

func (r *inMemoryRepo) GetAgentRunByID(ctx context.Context, arg repository.GetAgentRunByIDParams) (repository.AgentRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := service.PgtypeToUUID(arg.ID)
	run, ok := r.agentRuns[id]
	if !ok {
		return repository.AgentRun{}, fmt.Errorf("not found")
	}
	return run, nil
}

func (r *inMemoryRepo) CreateToolCall(ctx context.Context, arg repository.CreateToolCallParams) (repository.ToolCall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	runID := service.PgtypeToUUID(arg.RunID)
	tc := repository.ToolCall{
		ID:        service.UUIDToPgtype(id),
		RunID:     arg.RunID,
		ToolName:  arg.ToolName,
		Input:     arg.Input,
		Output:    arg.Output,
		IsError:   arg.IsError,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	r.toolCalls[runID] = append(r.toolCalls[runID], tc)
	return tc, nil
}

func (r *inMemoryRepo) ListToolCallsByRunID(ctx context.Context, runID pgtype.UUID) ([]repository.ToolCall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.toolCalls[service.PgtypeToUUID(runID)], nil
}

func (r *inMemoryRepo) CreateProposal(ctx context.Context, arg repository.CreateProposalParams) (repository.Proposal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	p := repository.Proposal{
		ID:        service.UUIDToPgtype(id),
		UserID:    arg.UserID,
		RunID:     arg.RunID,
		Summary:   arg.Summary,
		Actions:   arg.Actions,
		Status:    arg.Status,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	r.proposals[id] = p
	return p, nil
}

func (r *inMemoryRepo) GetProposalByID(ctx context.Context, arg repository.GetProposalByIDParams) (repository.Proposal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := service.PgtypeToUUID(arg.ID)
	p, ok := r.proposals[id]
	if !ok {
		return repository.Proposal{}, fmt.Errorf("not found")
	}
	return p, nil
}

func (r *inMemoryRepo) ListProposalsByUserID(ctx context.Context, arg repository.ListProposalsByUserIDParams) ([]repository.Proposal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []repository.Proposal
	for _, p := range r.proposals {
		list = append(list, p)
	}
	return list, nil
}

func (r *inMemoryRepo) UpdateProposalStatus(ctx context.Context, arg repository.UpdateProposalStatusParams) (repository.Proposal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := service.PgtypeToUUID(arg.ID)
	p := r.proposals[id]
	p.Status = arg.Status
	p.Actions = arg.Actions
	p.DecidedAt = arg.DecidedAt
	r.proposals[id] = p
	return p, nil
}

func (r *inMemoryRepo) HasBriefingForDate(ctx context.Context, arg repository.HasBriefingForDateParams) (bool, error) {
	return false, nil
}

func (r *inMemoryRepo) RecordBriefing(ctx context.Context, arg repository.RecordBriefingParams) error {
	return nil
}

// ----------------- Mock Planner Client -----------------

type mockPlanner struct {
	tasks      []service.TaskDTO
	projects   []service.ProjectDTO
	timeBlocks []service.TimeBlockDTO
	workload   *service.WorkloadDTO

	listTasksCalls      int
	getTaskCalls        int
	listProjectsCalls   int
	listTimeBlocksCalls int
	getWorkloadCalls    int
}

func (m *mockPlanner) ListTasks(ctx context.Context, authHeader string, q map[string]string) ([]service.TaskDTO, error) {
	m.listTasksCalls++
	return m.tasks, nil
}

func (m *mockPlanner) GetTask(ctx context.Context, authHeader string, taskID uuid.UUID) (*service.TaskDTO, error) {
	m.getTaskCalls++
	for _, t := range m.tasks {
		if t.ID == taskID {
			return &t, nil
		}
	}
	return nil, service.ErrPlannerNotFound
}

func (m *mockPlanner) ListProjects(ctx context.Context, authHeader string) ([]service.ProjectDTO, error) {
	m.listProjectsCalls++
	return m.projects, nil
}

func (m *mockPlanner) ListTimeBlocks(ctx context.Context, authHeader string, from, to string) ([]service.TimeBlockDTO, error) {
	m.listTimeBlocksCalls++
	return m.timeBlocks, nil
}

func (m *mockPlanner) GetWorkload(ctx context.Context, authHeader string, from, to string) (*service.WorkloadDTO, error) {
	m.getWorkloadCalls++
	if m.workload != nil {
		return m.workload, nil
	}
	return &service.WorkloadDTO{From: from, To: to, Days: []service.WorkloadDayDTO{}}, nil
}

func (m *mockPlanner) CreateTask(ctx context.Context, authHeader string, req service.CreateTaskDTO) (*service.TaskDTO, error) {
	return nil, nil
}

func (m *mockPlanner) UpdateTask(ctx context.Context, authHeader string, taskID uuid.UUID, req service.UpdateTaskDTO) (*service.TaskDTO, error) {
	return nil, nil
}

func (m *mockPlanner) DeleteTask(ctx context.Context, authHeader string, taskID uuid.UUID) error {
	return nil
}

func (m *mockPlanner) CreateTimeBlock(ctx context.Context, authHeader string, req service.CreateTimeBlockDTO) (*service.TimeBlockDTO, error) {
	return nil, nil
}

// ----------------- Eval Scenarios -----------------

type EvalScenario struct {
	ID          string
	Name        string
	Category    string
	UserPrompt  string
	SetupLLM    func(fake *service.FakeLLM)
	SetupData   func(planner *mockPlanner)
	Assert      func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error)
}

func TestAgentEvaluationSuite(t *testing.T) {
	userID := uuid.New()
	taskUUID1 := uuid.New()
	taskUUID2 := uuid.New()
	projUUID1 := uuid.New()

	scenarios := []EvalScenario{
		{
			ID:       "EVAL-01",
			Name:     "Simple task creation proposal",
			Category: "Task Staging",
			UserPrompt: "Add a task to buy groceries",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_1",
							Name: "create_task",
							Input: json.RawMessage(`{"title":"Buy groceries","priority":2,"status":"todo"}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "I have staged a proposal to create the task 'Buy groceries'.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if len(resp.Proposals[0].Actions) != 1 || resp.Proposals[0].Actions[0].Tool != "create_task" {
					t.Errorf("expected 1 action with tool 'create_task', got %+v", resp.Proposals[0].Actions)
				}
			},
		},
		{
			ID:       "EVAL-02",
			Name:     "High priority task creation with due date and estimate",
			Category: "Task Staging",
			UserPrompt: "Add high priority task 'Finish report' due 2026-10-15T18:00:00Z taking 90 minutes",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_2",
							Name: "create_task",
							Input: json.RawMessage(`{
								"title": "Finish report",
								"priority": 1,
								"status": "todo",
								"due_at": "2026-10-15T18:00:00Z",
								"estimate_minutes": 90
							}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "I staged a proposal for 'Finish report' (P1, 90 mins).",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				actions := resp.Proposals[0].Actions
				if len(actions) != 1 {
					t.Fatalf("expected 1 action, got %d", len(actions))
				}
				if actions[0].Tool != "create_task" {
					t.Errorf("expected tool create_task, got %s", actions[0].Tool)
				}
			},
		},
		{
			ID:       "EVAL-03",
			Name:     "Task creation linked to existing project",
			Category: "Task Staging",
			UserPrompt: "Add task 'Draft UI spec' in project " + projUUID1.String(),
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_3",
							Name: "create_task",
							Input: json.RawMessage(fmt.Sprintf(`{"title":"Draft UI spec","project_id":"%s"}`, projUUID1.String())),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Task staged under the project.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-04",
			Name:     "Goal breakdown into multiple staged tasks",
			Category: "Goal Breakdown",
			UserPrompt: "Break down 'Launch beta release' into actionable steps",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_4a",
							Name:  "create_task",
							Input: json.RawMessage(`{"title":"QA regression testing","priority":1}`),
						},
						{
							ID:    "call_4b",
							Name:  "create_task",
							Input: json.RawMessage(`{"title":"Deploy production build","priority":1}`),
						},
						{
							ID:    "call_4c",
							Name:  "create_task",
							Input: json.RawMessage(`{"title":"Send release announcement email","priority":2}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "I broke down the goal into 3 staged tasks for your approval.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if len(resp.Proposals[0].Actions) != 3 {
					t.Errorf("expected 3 actions in proposal, got %d", len(resp.Proposals[0].Actions))
				}
			},
		},
		{
			ID:       "EVAL-05",
			Name:     "Calendar time block proposal with valid interval",
			Category: "Calendar Staging",
			UserPrompt: "Schedule focus block tomorrow from 9am to 11am",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_5",
							Name: "create_time_block",
							Input: json.RawMessage(`{
								"title": "Focus Block",
								"starts_at": "2026-10-08T09:00:00Z",
								"ends_at": "2026-10-08T11:00:00Z"
							}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Proposed focus block on your calendar.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if resp.Proposals[0].Actions[0].Tool != "create_time_block" {
					t.Errorf("expected tool create_time_block, got %s", resp.Proposals[0].Actions[0].Tool)
				}
			},
		},
		{
			ID:       "EVAL-06",
			Name:     "Batch reschedule proposal for overdue tasks",
			Category: "Rescheduling",
			UserPrompt: "Reschedule overdue tasks to tomorrow",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_6",
							Name: "reschedule_tasks",
							Input: json.RawMessage(fmt.Sprintf(`{
								"reschedules": [
									{"task_id": "%s", "new_due_at": "2026-10-09T17:00:00Z", "reason": "overdue push"},
									{"task_id": "%s", "new_due_at": "2026-10-10T17:00:00Z", "reason": "overdue push"}
								]
							}`, taskUUID1.String(), taskUUID2.String())),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Proposed rescheduling 2 tasks.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if resp.Proposals[0].Actions[0].Tool != "reschedule_tasks" {
					t.Errorf("expected reschedule_tasks tool, got %s", resp.Proposals[0].Actions[0].Tool)
				}
			},
		},
		{
			ID:       "EVAL-07",
			Name:     "Task update proposal for status and priority",
			Category: "Task Staging",
			UserPrompt: "Set task " + taskUUID1.String() + " status to in_progress and priority 1",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_7",
							Name: "update_task",
							Input: json.RawMessage(fmt.Sprintf(`{
								"task_id": "%s",
								"status": "in_progress",
								"priority": 1
							}`, taskUUID1.String())),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Staged update for task.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if resp.Proposals[0].Actions[0].Tool != "update_task" {
					t.Errorf("expected update_task, got %s", resp.Proposals[0].Actions[0].Tool)
				}
			},
		},
		{
			ID:       "EVAL-08",
			Name:     "Task deletion proposal staging (planner not called directly)",
			Category: "Task Staging",
			UserPrompt: "Delete task " + taskUUID2.String(),
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_8",
							Name: "delete_task",
							Input: json.RawMessage(fmt.Sprintf(`{"task_id":"%s"}`, taskUUID2.String())),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Staged deletion proposal.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal, got %d", len(resp.Proposals))
				}
				if resp.Proposals[0].Actions[0].Tool != "delete_task" {
					t.Errorf("expected delete_task, got %s", resp.Proposals[0].Actions[0].Tool)
				}
			},
		},
		{
			ID:       "EVAL-09",
			Name:     "Read tool: list_tasks with filter execution",
			Category: "Read Operations",
			UserPrompt: "What tasks are in progress?",
			SetupData: func(planner *mockPlanner) {
				planner.tasks = []service.TaskDTO{
					{ID: taskUUID1, Title: "Refactor auth middleware", Status: "in_progress", Priority: 1},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_9",
							Name:  "list_tasks",
							Input: json.RawMessage(`{"status":"in_progress"}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "You have 1 task in progress: 'Refactor auth middleware'.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.listTasksCalls != 1 {
					t.Errorf("expected 1 listTasks call, got %d", planner.listTasksCalls)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals for read tool, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-10",
			Name:     "Read tool: get_task detail query",
			Category: "Read Operations",
			UserPrompt: "Show details for task " + taskUUID1.String(),
			SetupData: func(planner *mockPlanner) {
				planner.tasks = []service.TaskDTO{
					{
						ID:          taskUUID1,
						Title:       "Refactor auth middleware",
						Description: "Add jwt claims inspection",
						Subtasks: []service.SubtaskDTO{
							{Title: "Write tests", Done: true},
						},
					},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_10",
							Name:  "get_task",
							Input: json.RawMessage(fmt.Sprintf(`{"task_id":"%s"}`, taskUUID1.String())),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Task 'Refactor auth middleware' has 1 subtask completed.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.getTaskCalls != 1 {
					t.Errorf("expected 1 getTask call, got %d", planner.getTaskCalls)
				}
			},
		},
		{
			ID:       "EVAL-11",
			Name:     "Read tool: get_workload query",
			Category: "Read Operations",
			UserPrompt: "What is my workload from 2026-10-07 to 2026-10-10?",
			SetupData: func(planner *mockPlanner) {
				planner.workload = &service.WorkloadDTO{
					From: "2026-10-07",
					To:   "2026-10-10",
					Days: []service.WorkloadDayDTO{
						{Date: "2026-10-07", PlannedMinutes: 180, CapacityMinutes: 480},
					},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_11",
							Name:  "get_workload",
							Input: json.RawMessage(`{"from":"2026-10-07","to":"2026-10-10"}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "On Oct 7 you have 3 hours planned out of 8 hours capacity.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.getWorkloadCalls != 1 {
					t.Errorf("expected 1 getWorkload call, got %d", planner.getWorkloadCalls)
				}
			},
		},
		{
			ID:       "EVAL-12",
			Name:     "Read tool: list_projects inquiry",
			Category: "Read Operations",
			UserPrompt: "What projects am I working on?",
			SetupData: func(planner *mockPlanner) {
				planner.projects = []service.ProjectDTO{
					{ID: projUUID1, Name: "Planly Web App", Colour: "#3b82f6"},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_12",
							Name:  "list_projects",
							Input: json.RawMessage(`{}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "You currently have 1 active project: 'Planly Web App'.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.listProjectsCalls != 1 {
					t.Errorf("expected 1 listProjects call, got %d", planner.listProjectsCalls)
				}
			},
		},
		{
			ID:       "EVAL-13",
			Name:     "Read tool: list_time_blocks inquiry",
			Category: "Read Operations",
			UserPrompt: "Check what time blocks I have scheduled",
			SetupData: func(planner *mockPlanner) {
				planner.timeBlocks = []service.TimeBlockDTO{
					{
						ID:       uuid.New(),
						Title:    "Team Standup",
						StartsAt: time.Now().UTC(),
						EndsAt:   time.Now().UTC().Add(30 * time.Minute),
					},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_13",
							Name:  "list_time_blocks",
							Input: json.RawMessage(`{}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "You have Team Standup scheduled.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.listTimeBlocksCalls != 1 {
					t.Errorf("expected 1 listTimeBlocks call, got %d", planner.listTimeBlocksCalls)
				}
			},
		},
		{
			ID:       "EVAL-14",
			Name:     "Safety Guardrail: Malformed UUID in update_task rejected",
			Category: "Guardrails",
			UserPrompt: "Update task bad-id-123 title to New Title",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_14",
							Name:  "update_task",
							Input: json.RawMessage(`{"task_id":"not-a-valid-uuid","title":"New Title"}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "The task ID provided is not a valid UUID format.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals created on invalid UUID input, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-15",
			Name:     "Safety Guardrail: Inverted time block interval rejected",
			Category: "Guardrails",
			UserPrompt: "Block time from 3pm to 2pm tomorrow",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:   "call_15",
							Name: "create_time_block",
							Input: json.RawMessage(`{
								"title": "Invalid Block",
								"starts_at": "2026-10-08T15:00:00Z",
								"ends_at": "2026-10-08T14:00:00Z"
							}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Cannot schedule block because end time is before start time.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals created on inverted time block, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-16",
			Name:     "Safety Guardrail: Missing required task title rejected",
			Category: "Guardrails",
			UserPrompt: "Create a task without title",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_16",
							Name:  "create_task",
							Input: json.RawMessage(`{"title":"","priority":2}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Task creation requires a non-empty title.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals on empty title, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-17",
			Name:     "Safety Guardrail: Empty reschedule array rejected",
			Category: "Guardrails",
			UserPrompt: "Reschedule tasks with empty list",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_17",
							Name:  "reschedule_tasks",
							Input: json.RawMessage(`{"reschedules":[]}`),
						},
					},
				})
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "No tasks provided to reschedule.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals on empty reschedule list, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-18",
			Name:     "Direct answer without tool calls",
			Category: "General Interaction",
			UserPrompt: "What is the recommended pomodoro interval?",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "The traditional Pomodoro Technique recommends 25 minutes of work followed by a 5-minute break.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(resp.Proposals) != 0 {
					t.Errorf("expected 0 proposals, got %d", len(resp.Proposals))
				}
				if resp.Reply == "" {
					t.Error("expected non-empty reply")
				}
			},
		},
		{
			ID:       "EVAL-19",
			Name:     "Multi-turn read-then-propose sequence",
			Category: "Multi-Step Workflow",
			UserPrompt: "Check if I have tasks and add a follow-up task",
			SetupData: func(planner *mockPlanner) {
				planner.tasks = []service.TaskDTO{
					{ID: taskUUID1, Title: "Existing design", Status: "todo"},
				}
			},
			SetupLLM: func(fake *service.FakeLLM) {
				// Step 1: LLM calls list_tasks
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_19a",
							Name:  "list_tasks",
							Input: json.RawMessage(`{}`),
						},
					},
				})
				// Step 2: LLM calls create_task
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "tool_use",
					ToolCalls: []service.LLMToolCall{
						{
							ID:    "call_19b",
							Name:  "create_task",
							Input: json.RawMessage(`{"title":"Follow up on existing design","priority":2}`),
						},
					},
				})
				// Step 3: LLM finishes turn
				fake.QueueResponse(&service.LLMResponse{
					StopReason: "end_turn",
					Text:       "Checked your tasks and staged the follow-up task.",
				})
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if planner.listTasksCalls != 1 {
					t.Errorf("expected 1 listTasks call, got %d", planner.listTasksCalls)
				}
				if len(resp.Proposals) != 1 {
					t.Fatalf("expected 1 proposal staged, got %d", len(resp.Proposals))
				}
			},
		},
		{
			ID:       "EVAL-20",
			Name:     "Daily message cap rate limit enforcement",
			Category: "Rate Limiting",
			UserPrompt: "Hello assistant",
			SetupLLM: func(fake *service.FakeLLM) {
				fake.DefaultReply = "This should never be reached"
			},
			Assert: func(t *testing.T, resp *model.ChatResponse, repo *inMemoryRepo, planner *mockPlanner, err error) {
				if err == nil {
					t.Fatal("expected ErrDailyMessageCapExceeded error, got nil")
				}
				if !errorsIs(err, service.ErrDailyMessageCapExceeded) {
					t.Errorf("expected ErrDailyMessageCapExceeded, got: %v", err)
				}
			},
		},
	}

	passedCount := 0
	totalCount := len(scenarios)

	for _, sc := range scenarios {
		t.Run(fmt.Sprintf("%s_%s", sc.ID, sc.Name), func(t *testing.T) {
			repo := newInMemoryRepo()
			planner := &mockPlanner{}
			fakeLLM := service.NewFakeLLM()

			if sc.SetupLLM != nil {
				sc.SetupLLM(fakeLLM)
			}
			if sc.SetupData != nil {
				sc.SetupData(planner)
			}

			// For EVAL-20, simulate exceeding daily cap
			if sc.ID == "EVAL-20" {
				repo.messageCount = 50
			}

			agentSvc := service.NewAgentService(repo, fakeLLM, planner, 50)

			resp, err := agentSvc.Chat(context.Background(), userID, "Bearer test-jwt", model.ChatRequest{
				Message: sc.UserPrompt,
			}, nil)

			sc.Assert(t, resp, repo, planner, err)
			passedCount++
		})
	}

	passRate := float64(passedCount) / float64(totalCount) * 100
	t.Logf("\n==========================================")
	t.Logf("AGENT EVALUATION RESULTS: %d/%d passed (%.1f%%)", passedCount, totalCount, passRate)
	t.Logf("==========================================\n")
}

func errorsIs(err, target error) bool {
	if err == nil {
		return target == nil
	}
	if target == nil {
		return false
	}
	for {
		if err == target {
			return true
		}
		type unwrap interface{ Unwrap() error }
		u, ok := err.(unwrap)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
}
