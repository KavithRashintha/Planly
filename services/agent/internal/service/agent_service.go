package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/planly/pkg/logx"
	"github.com/planly/services/agent/internal/model"
	"github.com/planly/services/agent/internal/repository"
)

var (
	ErrDailyMessageCapExceeded = errors.New("daily message limit exceeded")
	ErrConversationNotFound    = errors.New("conversation not found")
	ErrProposalNotFound        = errors.New("proposal not found")
	ErrProposalAlreadyDecided  = errors.New("proposal has already been decided")
	ErrAgentRunNotFound        = errors.New("agent run not found")
	ErrMaxIterationsExceeded   = errors.New("agent reached maximum iterations without completing")
)

type AgentService interface {
	Chat(ctx context.Context, userID uuid.UUID, authHeader string, req model.ChatRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	PlanGoal(ctx context.Context, userID uuid.UUID, authHeader string, req model.PlanGoalRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	Reschedule(ctx context.Context, userID uuid.UUID, authHeader string, req model.RescheduleRequest, userCtx *model.UserContext) (*model.ChatResponse, error)
	ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ConversationResponse, error)
	GetConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) (*model.ConversationDetailResponse, error)
	DeleteConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) error
	ListProposals(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ProposalResponse, error)
	GetProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error)
	ApproveProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error)
	RejectProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error)
	GetAgentRun(ctx context.Context, userID uuid.UUID, runID uuid.UUID) (*model.AgentRunResponse, error)
}

type agentServiceImpl struct {
	repo            repository.Querier
	llm             LLMClient
	planner         PlannerClient
	tools           *ToolRegistry
	dailyMessageCap int64
}

func NewAgentService(repo repository.Querier, llm LLMClient, planner PlannerClient, dailyMessageCap int64) AgentService {
	if dailyMessageCap <= 0 {
		dailyMessageCap = 50
	}
	return &agentServiceImpl{
		repo:            repo,
		llm:             llm,
		planner:         planner,
		tools:           NewToolRegistry(planner),
		dailyMessageCap: dailyMessageCap,
	}
}

// ----------------- Chat Endpoint -----------------

func (s *agentServiceImpl) Chat(ctx context.Context, userID uuid.UUID, authHeader string, req model.ChatRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	if req.Message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	// 1. Check daily message cap
	since := time.Now().UTC().Add(-24 * time.Hour)
	count, err := s.repo.CountMessagesForUserSince(ctx, repository.CountMessagesForUserSinceParams{
		UserID:    UUIDToPgtype(userID),
		CreatedAt: TimeToPgtype(since),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check daily message cap: %w", err)
	}
	if count >= s.dailyMessageCap {
		return nil, ErrDailyMessageCapExceeded
	}

	// 2. Resolve or create conversation
	var convID uuid.UUID
	if req.ConversationID != nil && *req.ConversationID != uuid.Nil {
		conv, err := s.repo.GetConversationByID(ctx, repository.GetConversationByIDParams{
			ID:     UUIDToPgtype(*req.ConversationID),
			UserID: UUIDToPgtype(userID),
		})
		if err != nil {
			return nil, ErrConversationNotFound
		}
		convID = PgtypeToUUID(conv.ID)
	} else {
		title := req.Message
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		conv, err := s.repo.CreateConversation(ctx, repository.CreateConversationParams{
			UserID: UUIDToPgtype(userID),
			Title:  title,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
		convID = PgtypeToUUID(conv.ID)
	}

	// 3. Store user message in DB
	userMsgBytes, _ := json.Marshal([]LLMContentBlock{{
		Type: "text",
		Text: req.Message,
	}})
	_, err = s.repo.CreateMessage(ctx, repository.CreateMessageParams{
		ConversationID: UUIDToPgtype(convID),
		Role:           string(RoleUser),
		Content:        userMsgBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 4. Run the agent loop
	return s.runAgentLoop(ctx, userID, authHeader, convID, "chat", req.Message, userCtx)
}

// ----------------- Plan Goal Endpoint -----------------

func (s *agentServiceImpl) PlanGoal(ctx context.Context, userID uuid.UUID, authHeader string, req model.PlanGoalRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	if req.Goal == "" {
		return nil, fmt.Errorf("goal cannot be empty")
	}
	if req.Deadline.IsZero() || !req.Deadline.After(time.Now().UTC()) {
		return nil, fmt.Errorf("deadline must be a future date/time")
	}

	prompt := fmt.Sprintf("Goal breakdown request: Break down the goal '%s' into actionable tasks. All tasks must have estimates and due dates on or before %s. Propose the new tasks using write tools.", req.Goal, req.Deadline.Format(time.RFC3339))

	var convID uuid.UUID
	if req.ConversationID != nil && *req.ConversationID != uuid.Nil {
		conv, err := s.repo.GetConversationByID(ctx, repository.GetConversationByIDParams{
			ID:     UUIDToPgtype(*req.ConversationID),
			UserID: UUIDToPgtype(userID),
		})
		if err != nil {
			return nil, ErrConversationNotFound
		}
		convID = PgtypeToUUID(conv.ID)
	} else {
		title := fmt.Sprintf("Goal: %s", req.Goal)
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		conv, err := s.repo.CreateConversation(ctx, repository.CreateConversationParams{
			UserID: UUIDToPgtype(userID),
			Title:  title,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create goal conversation: %w", err)
		}
		convID = PgtypeToUUID(conv.ID)
	}

	userMsgBytes, _ := json.Marshal([]LLMContentBlock{{
		Type: "text",
		Text: prompt,
	}})
	_, _ = s.repo.CreateMessage(ctx, repository.CreateMessageParams{
		ConversationID: UUIDToPgtype(convID),
		Role:           string(RoleUser),
		Content:        userMsgBytes,
	})

	return s.runAgentLoop(ctx, userID, authHeader, convID, "plan_goal", prompt, userCtx)
}

// ----------------- Reschedule Endpoint -----------------

func (s *agentServiceImpl) Reschedule(ctx context.Context, userID uuid.UUID, authHeader string, req model.RescheduleRequest, userCtx *model.UserContext) (*model.ChatResponse, error) {
	prompt := "Reschedule request: Check overdue and upcoming tasks and current workload. Propose realistic new due dates respecting working hours and daily capacity using reschedule_tasks or update_task."
	if len(req.TaskIDs) > 0 {
		prompt = fmt.Sprintf("Reschedule request for specific tasks %v: Check their current state and workload, then propose optimal new due dates.", req.TaskIDs)
	}

	var convID uuid.UUID
	if req.ConversationID != nil && *req.ConversationID != uuid.Nil {
		conv, err := s.repo.GetConversationByID(ctx, repository.GetConversationByIDParams{
			ID:     UUIDToPgtype(*req.ConversationID),
			UserID: UUIDToPgtype(userID),
		})
		if err != nil {
			return nil, ErrConversationNotFound
		}
		convID = PgtypeToUUID(conv.ID)
	} else {
		conv, err := s.repo.CreateConversation(ctx, repository.CreateConversationParams{
			UserID: UUIDToPgtype(userID),
			Title:  "Reschedule Assistant",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create reschedule conversation: %w", err)
		}
		convID = PgtypeToUUID(conv.ID)
	}

	userMsgBytes, _ := json.Marshal([]LLMContentBlock{{
		Type: "text",
		Text: prompt,
	}})
	_, _ = s.repo.CreateMessage(ctx, repository.CreateMessageParams{
		ConversationID: UUIDToPgtype(convID),
		Role:           string(RoleUser),
		Content:        userMsgBytes,
	})

	return s.runAgentLoop(ctx, userID, authHeader, convID, "reschedule", prompt, userCtx)
}

// ----------------- Core Agent Loop -----------------

func (s *agentServiceImpl) runAgentLoop(
	ctx context.Context,
	userID uuid.UUID,
	authHeader string,
	convID uuid.UUID,
	trigger string,
	latestPrompt string,
	userCtx *model.UserContext,
) (*model.ChatResponse, error) {
	// 1. Initialize agent run in DB
	run, err := s.repo.CreateAgentRun(ctx, repository.CreateAgentRunParams{
		ConversationID: UUIDToPgtype(convID),
		UserID:         UUIDToPgtype(userID),
		Trigger:        trigger,
		Status:         "running",
		Iterations:     0,
		InputTokens:    0,
		OutputTokens:   0,
		Error:          pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent run: %w", err)
	}
	runID := PgtypeToUUID(run.ID)
	startTime := time.Now()

	// 2. Build system prompt
	sysPrompt := s.buildSystemPrompt(userCtx)

	// 3. Load conversation history
	dbMsgs, err := s.repo.ListMessagesByConversationID(ctx, UUIDToPgtype(convID))
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation history: %w", err)
	}

	var history []LLMMessage
	for _, m := range dbMsgs {
		var blocks []LLMContentBlock
		if err := json.Unmarshal(m.Content, &blocks); err == nil {
			history = append(history, LLMMessage{
				Role:    LLMMessageRole(m.Role),
				Content: blocks,
			})
		}
	}

	// 4. Staged actions accumulator for human-approval proposals
	var stagedActions []model.ProposalAction
	stageActionFunc := func(toolName string, input json.RawMessage, summary string) (string, error) {
		actionID := uuid.New().String()
		stagedActions = append(stagedActions, model.ProposalAction{
			ID:     actionID,
			Tool:   toolName,
			Input:  input,
			Status: "pending",
		})
		return actionID, nil
	}

	execCtx := ToolExecutionContext{
		Context:     ctx,
		UserID:      userID,
		AuthHeader:  authHeader,
		StageAction: stageActionFunc,
	}

	// 5. Execution Loop (max 8 iterations)
	const maxIterations = 8
	var (
		iterations   int32
		totalInput   int32
		totalOutput  int32
		finalReply   string
		loopErr      error
	)

	toolDefs := s.tools.Definitions()

	for iter := 1; iter <= maxIterations; iter++ {
		iterations = int32(iter)

		llmResp, err := s.llm.Generate(ctx, LLMRequest{
			System:   sysPrompt,
			Messages: history,
			Tools:    toolDefs,
		})
		if err != nil {
			loopErr = err
			break
		}

		totalInput += int32(llmResp.InputTokens)
		totalOutput += int32(llmResp.OutputTokens)

		// If no tool calls requested, we have the final assistant answer
		if len(llmResp.ToolCalls) == 0 {
			finalReply = llmResp.Text
			if finalReply == "" {
				finalReply = "I've reviewed your request and made the necessary updates."
			}
			break
		}

		// Assistant asked to call tools: build assistant message block
		var assistantBlocks []LLMContentBlock
		if llmResp.Text != "" {
			assistantBlocks = append(assistantBlocks, LLMContentBlock{
				Type: "text",
				Text: llmResp.Text,
			})
		}
		for _, tc := range llmResp.ToolCalls {
			assistantBlocks = append(assistantBlocks, LLMContentBlock{
				Type:             "tool_use",
				ID:               tc.ID,
				Name:             tc.Name,
				Input:            tc.Input,
				ThoughtSignature: tc.ThoughtSignature,
			})
		}
		history = append(history, LLMMessage{
			Role:    RoleAssistant,
			Content: assistantBlocks,
		})

		// Execute tools and append tool_result blocks
		var userResultBlocks []LLMContentBlock
		for _, tc := range llmResp.ToolCalls {
			tool, found := s.tools.Get(tc.Name)
			var (
				outBytes []byte
				isErr    bool
			)

			if !found {
				isErr = true
				outBytes = []byte(fmt.Sprintf(`{"error":"unknown tool %s"}`, tc.Name))
			} else {
				out, execErr := tool.Handler(execCtx, tc.Input)
				if execErr != nil {
					isErr = true
					outBytes = []byte(fmt.Sprintf(`{"error":"%s"}`, execErr.Error()))
				} else {
					outBytes, _ = json.Marshal(out)
				}
			}

			// Record tool call in DB
			_, _ = s.repo.CreateToolCall(ctx, repository.CreateToolCallParams{
				RunID:    UUIDToPgtype(runID),
				ToolName: tc.Name,
				Input:    tc.Input,
				Output:   outBytes,
				IsError:  isErr,
			})

			userResultBlocks = append(userResultBlocks, LLMContentBlock{
				Type:       "tool_result",
				ToolUseID:  tc.ID,
				Name:       tc.Name,
				Content:    string(outBytes),
				IsError:    isErr,
			})
		}

		history = append(history, LLMMessage{
			Role:    RoleUser,
			Content: userResultBlocks,
		})
	}

	if loopErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		logx.FromContext(ctx).Error("agent run failed",
			"run_id", runID.String(),
			"user_id", userID.String(),
			"trigger", trigger,
			"iterations", iterations,
			"duration_ms", durationMs,
			"error", loopErr,
		)
		_, _ = s.repo.UpdateAgentRunStatus(ctx, repository.UpdateAgentRunStatusParams{
			ID:           UUIDToPgtype(runID),
			Status:       "failed",
			Iterations:   iterations,
			InputTokens:  totalInput,
			OutputTokens: totalOutput,
			Error:        TextToPgtype(loopErr.Error()),
		})
		return nil, fmt.Errorf("agent run failed: %w", loopErr)
	}

	if finalReply == "" && len(stagedActions) > 0 {
		finalReply = fmt.Sprintf("I have staged %d planned action(s) for your approval.", len(stagedActions))
	} else if finalReply == "" {
		finalReply = "I have completed processing your request."
	}

	// 6. Save assistant final message in DB
	assistantMsgBytes, _ := json.Marshal([]LLMContentBlock{{
		Type: "text",
		Text: finalReply,
	}})
	savedAssistantMsg, err := s.repo.CreateMessage(ctx, repository.CreateMessageParams{
		ConversationID: UUIDToPgtype(convID),
		Role:           string(RoleAssistant),
		Content:        assistantMsgBytes,
	})
	if err != nil {
		slog.Error("failed to save assistant final message", "error", err)
	}

	// 7. Persist proposals if actions were staged
	var proposalResponses []model.ProposalResponse
	if len(stagedActions) > 0 {
		actionsBytes, _ := json.Marshal(stagedActions)
		proposalSummary := fmt.Sprintf("Proposal with %d staged action(s)", len(stagedActions))
		if len(stagedActions) == 1 {
			proposalSummary = fmt.Sprintf("Proposal: %s", stagedActions[0].Tool)
		}

		dbProp, err := s.repo.CreateProposal(ctx, repository.CreateProposalParams{
			UserID:  UUIDToPgtype(userID),
			RunID:   UUIDToPgtype(runID),
			Summary: proposalSummary,
			Actions: actionsBytes,
			Status:  "pending",
		})
		if err == nil {
			proposalResponses = append(proposalResponses, model.ProposalResponse{
				ID:        PgtypeToUUID(dbProp.ID),
				UserID:    userID,
				RunID:     &runID,
				Summary:   dbProp.Summary,
				Actions:   stagedActions,
				Status:    dbProp.Status,
				CreatedAt: dbProp.CreatedAt.Time.UTC(),
			})
		} else {
			slog.Error("failed to persist proposal", "error", err)
		}
	}

	// 8. Update run status to completed
	durationMs := time.Since(startTime).Milliseconds()
	logx.FromContext(ctx).Info("agent run completed",
		"run_id", runID.String(),
		"user_id", userID.String(),
		"trigger", trigger,
		"iterations", iterations,
		"duration_ms", durationMs,
		"input_tokens", totalInput,
		"output_tokens", totalOutput,
		"staged_proposals", len(proposalResponses),
	)
	_, _ = s.repo.UpdateAgentRunStatus(ctx, repository.UpdateAgentRunStatusParams{
		ID:           UUIDToPgtype(runID),
		Status:       "completed",
		Iterations:   iterations,
		InputTokens:  totalInput,
		OutputTokens: totalOutput,
		Error:        pgtype.Text{Valid: false},
	})

	return &model.ChatResponse{
		ConversationID: convID,
		MessageID:      PgtypeToUUID(savedAssistantMsg.ID),
		Reply:          finalReply,
		Proposals:      proposalResponses,
		RunID:          runID,
	}, nil
}

func (s *agentServiceImpl) buildSystemPrompt(userCtx *model.UserContext) string {
	nowUTC := time.Now().UTC().Format(time.RFC3339)
	persona := "employee"
	timezone := "UTC"
	workStart := "09:00"
	workEnd := "17:00"

	if userCtx != nil {
		if userCtx.Persona != "" {
			persona = userCtx.Persona
		}
		if userCtx.Timezone != "" {
			timezone = userCtx.Timezone
		}
		if userCtx.WorkStart != "" {
			workStart = userCtx.WorkStart
		}
		if userCtx.WorkEnd != "" {
			workEnd = userCtx.WorkEnd
		}
	}

	return fmt.Sprintf(`You are Planly, an AI work planning assistant. You help users manage tasks, projects, and schedules.

CONTEXT:
- User type: %s
- User timezone: %s
- Working hours: %s to %s
- Current UTC time: %s

CRITICAL RULES — FOLLOW THESE EXACTLY:
1. NEVER respond with greetings, introductions, or filler like "Hi, I am Planly..." when the user is asking you to DO something.
2. When the user asks you to create, add, list, update, delete, or reschedule anything — IMMEDIATELY call the appropriate tool. Do not ask for confirmation first.
3. For CREATE actions: use create_task or create_time_block tools immediately.
4. For LIST actions: call list_tasks or list_projects immediately, then summarize what you found.
5. For UPDATE/RESCHEDULE actions: first call get_task or list_tasks to find the item, then call the update tool.
6. Write tools (create_task, update_task, delete_task, create_time_block, reschedule_tasks) stage a proposal for user approval — they do NOT apply changes immediately. After calling a write tool, briefly explain what you proposed.
7. NEVER invent task IDs or project IDs — only use real UUIDs from tool responses.
8. Always prefer using tools over explaining what you would do.

TOOL USAGE DECISION TREE:
- User says "add/create a task" → call create_task immediately
- User says "show/list my tasks" → call list_tasks immediately
- User says "reschedule/move a task" → call list_tasks first to find it, then reschedule_tasks
- User asks about workload → call get_workload
- User asks about projects → call list_projects
- Any other action request → pick the most relevant tool and call it

Keep responses concise and action-focused.`, persona, timezone, workStart, workEnd, nowUTC)
}


// ----------------- Proposal Approvals & Rejections -----------------

func (s *agentServiceImpl) ApproveProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID, authHeader string, req model.ApproveProposalRequest) (*model.ProposalResponse, error) {
	prop, err := s.repo.GetProposalByID(ctx, repository.GetProposalByIDParams{
		ID:     UUIDToPgtype(proposalID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, ErrProposalNotFound
	}

	if prop.Status != "pending" {
		return nil, ErrProposalAlreadyDecided
	}

	var actions []model.ProposalAction
	if err := json.Unmarshal(prop.Actions, &actions); err != nil {
		return nil, fmt.Errorf("failed to parse proposal actions: %w", err)
	}

	selectedSet := make(map[string]bool)
	for _, id := range req.ActionIDs {
		selectedSet[id] = true
	}

	var (
		appliedCount int
		failedCount  int
		skippedCount int
	)

	// Execute approved actions against planner-svc
	for i := range actions {
		act := &actions[i]

		// If specific action IDs provided and this one is not selected, skip
		if len(selectedSet) > 0 && !selectedSet[act.ID] {
			act.Status = "skipped"
			skippedCount++
			continue
		}

		execErr := s.executeAction(ctx, authHeader, act)
		if execErr != nil {
			act.Status = "failed"
			act.Error = execErr.Error()
			failedCount++
		} else {
			act.Status = "applied"
			appliedCount++
		}
	}

	var finalStatus string
	if failedCount == 0 && skippedCount == 0 {
		finalStatus = "approved"
	} else if appliedCount > 0 {
		finalStatus = "partial"
	} else if failedCount > 0 {
		finalStatus = "failed"
	} else {
		finalStatus = "approved"
	}

	updatedActionsBytes, _ := json.Marshal(actions)
	now := time.Now().UTC()

	updatedProp, err := s.repo.UpdateProposalStatus(ctx, repository.UpdateProposalStatusParams{
		ID:        UUIDToPgtype(proposalID),
		Status:    finalStatus,
		Actions:   updatedActionsBytes,
		DecidedAt: TimeToPgtype(now),
		UserID:    UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update proposal status: %w", err)
	}

	runID := PgtypeToUUIDPtr(updatedProp.RunID)
	decidedAt := PgtypeToTimePtr(updatedProp.DecidedAt)

	return &model.ProposalResponse{
		ID:        PgtypeToUUID(updatedProp.ID),
		UserID:    userID,
		RunID:     runID,
		Summary:   updatedProp.Summary,
		Actions:   actions,
		Status:    updatedProp.Status,
		CreatedAt: updatedProp.CreatedAt.Time.UTC(),
		DecidedAt: decidedAt,
	}, nil
}

func (s *agentServiceImpl) RejectProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
	prop, err := s.repo.GetProposalByID(ctx, repository.GetProposalByIDParams{
		ID:     UUIDToPgtype(proposalID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, ErrProposalNotFound
	}

	if prop.Status != "pending" {
		return nil, ErrProposalAlreadyDecided
	}

	var actions []model.ProposalAction
	_ = json.Unmarshal(prop.Actions, &actions)
	for i := range actions {
		actions[i].Status = "rejected"
	}
	actionsBytes, _ := json.Marshal(actions)

	now := time.Now().UTC()
	updatedProp, err := s.repo.UpdateProposalStatus(ctx, repository.UpdateProposalStatusParams{
		ID:        UUIDToPgtype(proposalID),
		Status:    "rejected",
		Actions:   actionsBytes,
		DecidedAt: TimeToPgtype(now),
		UserID:    UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reject proposal: %w", err)
	}

	runID := PgtypeToUUIDPtr(updatedProp.RunID)
	decidedAt := PgtypeToTimePtr(updatedProp.DecidedAt)

	return &model.ProposalResponse{
		ID:        PgtypeToUUID(updatedProp.ID),
		UserID:    userID,
		RunID:     runID,
		Summary:   updatedProp.Summary,
		Actions:   actions,
		Status:    updatedProp.Status,
		CreatedAt: updatedProp.CreatedAt.Time.UTC(),
		DecidedAt: decidedAt,
	}, nil
}

func (s *agentServiceImpl) executeAction(ctx context.Context, authHeader string, act *model.ProposalAction) error {
	switch act.Tool {
	case "create_task":
		var dto CreateTaskDTO
		if err := json.Unmarshal(act.Input, &dto); err != nil {
			return err
		}
		_, err := s.planner.CreateTask(ctx, authHeader, dto)
		return err

	case "update_task":
		var params struct {
			TaskID          string      `json:"task_id"`
			ProjectID       *uuid.UUID  `json:"project_id,omitempty"`
			Title           string      `json:"title"`
			Description     string      `json:"description"`
			Priority        int16       `json:"priority"`
			Status          string      `json:"status"`
			DueAt           *time.Time  `json:"due_at,omitempty"`
			EstimateMinutes *int32      `json:"estimate_minutes,omitempty"`
			TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
		}
		if err := json.Unmarshal(act.Input, &params); err != nil {
			return err
		}
		taskID, err := uuid.Parse(params.TaskID)
		if err != nil {
			return err
		}
		dto := UpdateTaskDTO{
			ProjectID:       params.ProjectID,
			Title:           params.Title,
			Description:     params.Description,
			Priority:        params.Priority,
			Status:          params.Status,
			DueAt:           params.DueAt,
			EstimateMinutes: params.EstimateMinutes,
			TagIDs:          params.TagIDs,
		}
		_, err = s.planner.UpdateTask(ctx, authHeader, taskID, dto)
		return err

	case "delete_task":
		var params struct {
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(act.Input, &params); err != nil {
			return err
		}
		taskID, err := uuid.Parse(params.TaskID)
		if err != nil {
			return err
		}
		return s.planner.DeleteTask(ctx, authHeader, taskID)

	case "create_time_block":
		var dto CreateTimeBlockDTO
		if err := json.Unmarshal(act.Input, &dto); err != nil {
			return err
		}
		_, err := s.planner.CreateTimeBlock(ctx, authHeader, dto)
		return err

	case "reschedule_tasks":
		var params struct {
			Reschedules []struct {
				TaskID   string `json:"task_id"`
				NewDueAt string `json:"new_due_at"`
			} `json:"reschedules"`
		}
		if err := json.Unmarshal(act.Input, &params); err != nil {
			return err
		}
		for _, item := range params.Reschedules {
			taskID, err := uuid.Parse(item.TaskID)
			if err != nil {
				return err
			}
			newTime, err := time.Parse(time.RFC3339, item.NewDueAt)
			if err != nil {
				return err
			}
			existing, err := s.planner.GetTask(ctx, authHeader, taskID)
			if err != nil {
				return err
			}
			updateDTO := UpdateTaskDTO{
				ProjectID:       existing.ProjectID,
				Title:           existing.Title,
				Description:     existing.Description,
				Priority:        existing.Priority,
				Status:          existing.Status,
				DueAt:           &newTime,
				EstimateMinutes: existing.EstimateMinutes,
			}
			if _, err := s.planner.UpdateTask(ctx, authHeader, taskID, updateDTO); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported action tool: %s", act.Tool)
	}
}

// ----------------- Queries & Run History -----------------

func (s *agentServiceImpl) ListConversations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ConversationResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	dbConvs, err := s.repo.ListConversationsByUserID(ctx, repository.ListConversationsByUserIDParams{
		UserID: UUIDToPgtype(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	resp := make([]model.ConversationResponse, len(dbConvs))
	for i, c := range dbConvs {
		resp[i] = MapConversationToResponse(c)
	}
	return resp, nil
}

func (s *agentServiceImpl) GetConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) (*model.ConversationDetailResponse, error) {
	conv, err := s.repo.GetConversationByID(ctx, repository.GetConversationByIDParams{
		ID:     UUIDToPgtype(convID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, ErrConversationNotFound
	}

	dbMsgs, err := s.repo.ListMessagesByConversationID(ctx, UUIDToPgtype(convID))
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]model.MessageResponse, len(dbMsgs))
	for i, m := range dbMsgs {
		messages[i] = MapMessageToResponse(m)
	}

	return &model.ConversationDetailResponse{
		Conversation: MapConversationToResponse(conv),
		Messages:     messages,
	}, nil
}

func (s *agentServiceImpl) DeleteConversation(ctx context.Context, userID uuid.UUID, convID uuid.UUID) error {
	return s.repo.DeleteConversation(ctx, repository.DeleteConversationParams{
		ID:     UUIDToPgtype(convID),
		UserID: UUIDToPgtype(userID),
	})
}

func (s *agentServiceImpl) ListProposals(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.ProposalResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	dbProps, err := s.repo.ListProposalsByUserID(ctx, repository.ListProposalsByUserIDParams{
		UserID: UUIDToPgtype(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list proposals: %w", err)
	}

	resp := make([]model.ProposalResponse, len(dbProps))
	for i, p := range dbProps {
		var actions []model.ProposalAction
		_ = json.Unmarshal(p.Actions, &actions)

		resp[i] = model.ProposalResponse{
			ID:        PgtypeToUUID(p.ID),
			UserID:    userID,
			RunID:     PgtypeToUUIDPtr(p.RunID),
			Summary:   p.Summary,
			Actions:   actions,
			Status:    p.Status,
			CreatedAt: p.CreatedAt.Time.UTC(),
			DecidedAt: PgtypeToTimePtr(p.DecidedAt),
		}
	}
	return resp, nil
}

func (s *agentServiceImpl) GetProposal(ctx context.Context, userID uuid.UUID, proposalID uuid.UUID) (*model.ProposalResponse, error) {
	p, err := s.repo.GetProposalByID(ctx, repository.GetProposalByIDParams{
		ID:     UUIDToPgtype(proposalID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, ErrProposalNotFound
	}

	var actions []model.ProposalAction
	_ = json.Unmarshal(p.Actions, &actions)

	return &model.ProposalResponse{
		ID:        PgtypeToUUID(p.ID),
		UserID:    userID,
		RunID:     PgtypeToUUIDPtr(p.RunID),
		Summary:   p.Summary,
		Actions:   actions,
		Status:    p.Status,
		CreatedAt: p.CreatedAt.Time.UTC(),
		DecidedAt: PgtypeToTimePtr(p.DecidedAt),
	}, nil
}

func (s *agentServiceImpl) GetAgentRun(ctx context.Context, userID uuid.UUID, runID uuid.UUID) (*model.AgentRunResponse, error) {
	run, err := s.repo.GetAgentRunByID(ctx, repository.GetAgentRunByIDParams{
		ID:     UUIDToPgtype(runID),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, ErrAgentRunNotFound
	}

	calls, err := s.repo.ListToolCallsByRunID(ctx, UUIDToPgtype(runID))
	if err != nil {
		return nil, fmt.Errorf("failed to list tool calls: %w", err)
	}

	toolCallResponses := make([]model.ToolCallResponse, len(calls))
	for i, c := range calls {
		toolCallResponses[i] = model.ToolCallResponse{
			ID:        PgtypeToUUID(c.ID),
			RunID:     runID,
			ToolName:  c.ToolName,
			Input:     c.Input,
			Output:    c.Output,
			IsError:   c.IsError,
			CreatedAt: c.CreatedAt.Time.UTC(),
		}
	}

	return &model.AgentRunResponse{
		ID:             runID,
		ConversationID: PgtypeToUUIDPtr(run.ConversationID),
		UserID:         userID,
		Trigger:        run.Trigger,
		Status:         run.Status,
		Iterations:     run.Iterations,
		InputTokens:    run.InputTokens,
		OutputTokens:   run.OutputTokens,
		Error:          PgtypeToTextPtr(run.Error),
		CreatedAt:      run.CreatedAt.Time.UTC(),
		ToolCalls:      toolCallResponses,
	}, nil
}
