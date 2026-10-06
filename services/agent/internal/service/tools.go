package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidToolInput = errors.New("invalid tool input")
	ErrToolNotFound     = errors.New("tool not found")
)

type ToolExecutionContext struct {
	Context     context.Context
	UserID      uuid.UUID
	AuthHeader  string
	StageAction func(tool string, input json.RawMessage, summary string) (string, error)
}

type ToolHandler func(ctx ToolExecutionContext, input json.RawMessage) (any, error)

type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	IsWrite     bool
	Handler     ToolHandler
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry(planner PlannerClient) *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// ----------------- Read Tools -----------------

	// 1. list_tasks
	r.Register(Tool{
		Name:        "list_tasks",
		Description: "List tasks with optional filters (status: todo|in_progress|done, priority: 1-4, project_id, query, limit).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"status": {"type": "string", "enum": ["todo", "in_progress", "done"]},
				"priority": {"type": "integer", "minimum": 1, "maximum": 4},
				"project_id": {"type": "string", "format": "uuid"},
				"query": {"type": "string"},
				"limit": {"type": "integer", "maximum": 100}
			}
		}`),
		IsWrite: false,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				Status    string `json:"status"`
				Priority  *int   `json:"priority"`
				ProjectID string `json:"project_id"`
				Query     string `json:"query"`
				Limit     *int   `json:"limit"`
			}
			if len(input) > 0 && string(input) != "{}" {
				if err := json.Unmarshal(input, &params); err != nil {
					return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
				}
			}

			q := make(map[string]string)
			if params.Status != "" {
				q["status"] = params.Status
			}
			if params.Priority != nil {
				q["priority"] = fmt.Sprintf("%d", *params.Priority)
			}
			if params.ProjectID != "" {
				q["project_id"] = params.ProjectID
			}
			if params.Query != "" {
				q["q"] = params.Query
			}
			if params.Limit != nil {
				q["limit"] = fmt.Sprintf("%d", *params.Limit)
			}

			return planner.ListTasks(execCtx.Context, execCtx.AuthHeader, q)
		},
	})

	// 2. get_task
	r.Register(Tool{
		Name:        "get_task",
		Description: "Get complete details of a specific task by its UUID including subtasks and tags.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["task_id"],
			"properties": {
				"task_id": {"type": "string", "format": "uuid"}
			}
		}`),
		IsWrite: false,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				TaskID string `json:"task_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			taskID, err := uuid.Parse(params.TaskID)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid task_id uuid: %v", ErrInvalidToolInput, err)
			}

			return planner.GetTask(execCtx.Context, execCtx.AuthHeader, taskID)
		},
	})

	// 3. list_projects
	r.Register(Tool{
		Name:        "list_projects",
		Description: "List all existing projects belonging to the user.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
		IsWrite: false,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			return planner.ListProjects(execCtx.Context, execCtx.AuthHeader)
		},
	})

	// 4. list_time_blocks
	r.Register(Tool{
		Name:        "list_time_blocks",
		Description: "List calendar time blocks within a date/time range (ISO 8601 strings).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"from": {"type": "string", "format": "date-time"},
				"to": {"type": "string", "format": "date-time"}
			}
		}`),
		IsWrite: false,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				From string `json:"from"`
				To   string `json:"to"`
			}
			if len(input) > 0 && string(input) != "{}" {
				_ = json.Unmarshal(input, &params)
			}
			return planner.ListTimeBlocks(execCtx.Context, execCtx.AuthHeader, params.From, params.To)
		},
	})

	// 5. get_workload
	r.Register(Tool{
		Name:        "get_workload",
		Description: "Get workload breakdown (planned minutes vs capacity) for a date range (from and to as YYYY-MM-DD).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["from", "to"],
			"properties": {
				"from": {"type": "string", "pattern": "^\\d{4}-\\d{2}-\\d{2}$"},
				"to": {"type": "string", "pattern": "^\\d{4}-\\d{2}-\\d{2}$"}
			}
		}`),
		IsWrite: false,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				From string `json:"from"`
				To   string `json:"to"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if params.From == "" || params.To == "" {
				return nil, fmt.Errorf("%w: from and to dates are required", ErrInvalidToolInput)
			}
			return planner.GetWorkload(execCtx.Context, execCtx.AuthHeader, params.From, params.To)
		},
	})

	// ----------------- Write Tools (Proposals) -----------------
	// Never call planner directly! They stage actions into the proposal accumulator.

	// 6. create_task
	r.Register(Tool{
		Name:        "create_task",
		Description: "Propose creating a new task. Staged for user approval (not executed immediately).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["title"],
			"properties": {
				"title": {"type": "string"},
				"description": {"type": "string"},
				"priority": {"type": "integer", "minimum": 1, "maximum": 4},
				"status": {"type": "string", "enum": ["todo", "in_progress", "done"]},
				"due_at": {"type": "string", "format": "date-time"},
				"estimate_minutes": {"type": "integer", "minimum": 1},
				"project_id": {"type": "string", "format": "uuid"}
			}
		}`),
		IsWrite: true,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				Title           string  `json:"title"`
				Description     string  `json:"description"`
				Priority        int16   `json:"priority"`
				Status          string  `json:"status"`
				DueAt           *string `json:"due_at"`
				EstimateMinutes *int32  `json:"estimate_minutes"`
				ProjectID       *string `json:"project_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if params.Title == "" {
				return nil, fmt.Errorf("%w: task title is required", ErrInvalidToolInput)
			}
			if params.Priority == 0 {
				params.Priority = 2
			}
			if params.Status == "" {
				params.Status = "todo"
			}

			summary := fmt.Sprintf("Create task: '%s'", params.Title)
			if params.DueAt != nil {
				summary += fmt.Sprintf(" due at %s", *params.DueAt)
			}

			actionID, err := execCtx.StageAction("create_task", input, summary)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"status":    "proposal_created",
				"action_id": actionID,
				"summary":   summary,
				"note":      "Action staged in proposal for user approval.",
			}, nil
		},
	})

	// 7. update_task
	r.Register(Tool{
		Name:        "update_task",
		Description: "Propose updating an existing task. Staged for user approval (not executed immediately).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["task_id"],
			"properties": {
				"task_id": {"type": "string", "format": "uuid"},
				"title": {"type": "string"},
				"description": {"type": "string"},
				"priority": {"type": "integer", "minimum": 1, "maximum": 4},
				"status": {"type": "string", "enum": ["todo", "in_progress", "done"]},
				"due_at": {"type": "string", "format": "date-time"},
				"estimate_minutes": {"type": "integer", "minimum": 1},
				"project_id": {"type": "string", "format": "uuid"}
			}
		}`),
		IsWrite: true,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				TaskID string `json:"task_id"`
				Title  string `json:"title"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if _, err := uuid.Parse(params.TaskID); err != nil {
				return nil, fmt.Errorf("%w: invalid task_id uuid", ErrInvalidToolInput)
			}

			summary := fmt.Sprintf("Update task %s", params.TaskID)
			if params.Title != "" {
				summary += fmt.Sprintf(" (title: '%s')", params.Title)
			}

			actionID, err := execCtx.StageAction("update_task", input, summary)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"status":    "proposal_created",
				"action_id": actionID,
				"summary":   summary,
				"note":      "Action staged in proposal for user approval.",
			}, nil
		},
	})

	// 8. delete_task
	r.Register(Tool{
		Name:        "delete_task",
		Description: "Propose deleting an existing task. Staged for user approval (not executed immediately).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["task_id"],
			"properties": {
				"task_id": {"type": "string", "format": "uuid"}
			}
		}`),
		IsWrite: true,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				TaskID string `json:"task_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if _, err := uuid.Parse(params.TaskID); err != nil {
				return nil, fmt.Errorf("%w: invalid task_id uuid", ErrInvalidToolInput)
			}

			summary := fmt.Sprintf("Delete task %s", params.TaskID)
			actionID, err := execCtx.StageAction("delete_task", input, summary)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"status":    "proposal_created",
				"action_id": actionID,
				"summary":   summary,
				"note":      "Action staged in proposal for user approval.",
			}, nil
		},
	})

	// 9. create_time_block
	r.Register(Tool{
		Name:        "create_time_block",
		Description: "Propose scheduling a calendar time block. Staged for user approval (not executed immediately).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["title", "starts_at", "ends_at"],
			"properties": {
				"title": {"type": "string"},
				"task_id": {"type": "string", "format": "uuid"},
				"starts_at": {"type": "string", "format": "date-time"},
				"ends_at": {"type": "string", "format": "date-time"}
			}
		}`),
		IsWrite: true,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				Title    string `json:"title"`
				TaskID   string `json:"task_id,omitempty"`
				StartsAt string `json:"starts_at"`
				EndsAt   string `json:"ends_at"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if params.Title == "" || params.StartsAt == "" || params.EndsAt == "" {
				return nil, fmt.Errorf("%w: title, starts_at, and ends_at are required", ErrInvalidToolInput)
			}

			startTime, err := time.Parse(time.RFC3339, params.StartsAt)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid starts_at format (RFC3339 required)", ErrInvalidToolInput)
			}
			endTime, err := time.Parse(time.RFC3339, params.EndsAt)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid ends_at format (RFC3339 required)", ErrInvalidToolInput)
			}
			if !endTime.After(startTime) {
				return nil, fmt.Errorf("%w: ends_at must be after starts_at", ErrInvalidToolInput)
			}

			summary := fmt.Sprintf("Schedule block: '%s' from %s to %s", params.Title, params.StartsAt, params.EndsAt)
			actionID, err := execCtx.StageAction("create_time_block", input, summary)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"status":    "proposal_created",
				"action_id": actionID,
				"summary":   summary,
				"note":      "Action staged in proposal for user approval.",
			}, nil
		},
	})

	// 10. reschedule_tasks
	r.Register(Tool{
		Name:        "reschedule_tasks",
		Description: "Propose rescheduling multiple tasks to new due dates. Staged for user approval (not executed immediately).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["reschedules"],
			"properties": {
				"reschedules": {
					"type": "array",
					"items": {
						"type": "object",
						"required": ["task_id", "new_due_at"],
						"properties": {
							"task_id": {"type": "string", "format": "uuid"},
							"new_due_at": {"type": "string", "format": "date-time"},
							"reason": {"type": "string"}
						}
					}
				}
			}
		}`),
		IsWrite: true,
		Handler: func(execCtx ToolExecutionContext, input json.RawMessage) (any, error) {
			var params struct {
				Reschedules []struct {
					TaskID   string `json:"task_id"`
					NewDueAt string `json:"new_due_at"`
					Reason   string `json:"reason"`
				} `json:"reschedules"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidToolInput, err)
			}
			if len(params.Reschedules) == 0 {
				return nil, fmt.Errorf("%w: reschedules list cannot be empty", ErrInvalidToolInput)
			}

			for _, item := range params.Reschedules {
				if _, err := uuid.Parse(item.TaskID); err != nil {
					return nil, fmt.Errorf("%w: invalid task_id uuid in reschedules", ErrInvalidToolInput)
				}
				if _, err := time.Parse(time.RFC3339, item.NewDueAt); err != nil {
					return nil, fmt.Errorf("%w: invalid new_due_at format (RFC3339 required)", ErrInvalidToolInput)
				}
			}

			summary := fmt.Sprintf("Reschedule %d task(s)", len(params.Reschedules))
			actionID, err := execCtx.StageAction("reschedule_tasks", input, summary)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"status":    "proposal_created",
				"action_id": actionID,
				"summary":   summary,
				"count":     len(params.Reschedules),
				"note":      "Action staged in proposal for user approval.",
			}, nil
		},
	})

	return r
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) Definitions() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return defs
}
