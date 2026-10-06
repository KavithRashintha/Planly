package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Chat models
type ChatRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Message        string     `json:"message"`
}

type ChatResponse struct {
	ConversationID uuid.UUID          `json:"conversation_id"`
	MessageID      uuid.UUID          `json:"message_id"`
	Reply          string             `json:"reply"`
	Proposals      []ProposalResponse `json:"proposals,omitempty"`
	RunID          uuid.UUID          `json:"run_id"`
}

// Conversation models
type ConversationResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageResponse struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	Role           string          `json:"role"`
	Content        json.RawMessage `json:"content"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ConversationDetailResponse struct {
	Conversation ConversationResponse `json:"conversation"`
	Messages     []MessageResponse    `json:"messages"`
}

// Proposal models
type ProposalAction struct {
	ID     string          `json:"id"`
	Tool   string          `json:"tool"`
	Input  json.RawMessage `json:"input"`
	Status string          `json:"status"` // pending | applied | failed | skipped
	Error  string          `json:"error,omitempty"`
}

type ProposalResponse struct {
	ID        uuid.UUID        `json:"id"`
	UserID    uuid.UUID        `json:"user_id"`
	RunID     *uuid.UUID       `json:"run_id,omitempty"`
	Summary   string           `json:"summary"`
	Actions   []ProposalAction `json:"actions"`
	Status    string           `json:"status"` // pending | approved | rejected | partial | failed
	CreatedAt time.Time        `json:"created_at"`
	DecidedAt *time.Time       `json:"decided_at,omitempty"`
}

type ApproveProposalRequest struct {
	ActionIDs []string `json:"action_ids,omitempty"` // empty means approve all
}

// Plan Goal request
type PlanGoalRequest struct {
	Goal           string     `json:"goal"`
	Deadline       time.Time  `json:"deadline"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
}

// Reschedule request
type RescheduleRequest struct {
	TaskIDs        []uuid.UUID `json:"task_ids,omitempty"`
	ConversationID *uuid.UUID  `json:"conversation_id,omitempty"`
}

// Agent Run & Tool Call models
type ToolCallResponse struct {
	ID        uuid.UUID       `json:"id"`
	RunID     uuid.UUID       `json:"run_id"`
	ToolName  string          `json:"tool_name"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output,omitempty"`
	IsError   bool            `json:"is_error"`
	CreatedAt time.Time       `json:"created_at"`
}

type AgentRunResponse struct {
	ID           uuid.UUID          `json:"id"`
	ConversationID *uuid.UUID       `json:"conversation_id,omitempty"`
	UserID       uuid.UUID          `json:"user_id"`
	Trigger      string             `json:"trigger"`
	Status       string             `json:"status"`
	Iterations   int32              `json:"iterations"`
	InputTokens  int32              `json:"input_tokens"`
	OutputTokens int32              `json:"output_tokens"`
	Error        *string            `json:"error,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	ToolCalls    []ToolCallResponse `json:"tool_calls,omitempty"`
}

// User context passed into prompt
type UserContext struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Persona   string    `json:"persona"`
	Timezone  string    `json:"timezone"`
	WorkStart string    `json:"work_start"`
	WorkEnd   string    `json:"work_end"`
}
