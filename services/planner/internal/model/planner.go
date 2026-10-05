package model

import (
	"time"

	"github.com/google/uuid"
)

// Project models
type CreateProjectRequest struct {
	Name   string `json:"name"`
	Colour string `json:"colour,omitempty"`
}

type UpdateProjectRequest struct {
	Name     string `json:"name"`
	Colour   string `json:"colour"`
	Archived bool   `json:"archived"`
}

type ProjectResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Colour    string    `json:"colour"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Subtask models
type CreateSubtaskRequest struct {
	Title    string `json:"title"`
	Position int32  `json:"position,omitempty"`
}

type UpdateSubtaskRequest struct {
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Position int32  `json:"position"`
}

type SubtaskResponse struct {
	ID       uuid.UUID `json:"id"`
	TaskID   uuid.UUID `json:"task_id"`
	Title    string    `json:"title"`
	Done     bool      `json:"done"`
	Position int32     `json:"position"`
}

// Tag models
type CreateTagRequest struct {
	Name string `json:"name"`
}

type TagResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
}

// Task models
type CreateTaskRequest struct {
	ProjectID       *uuid.UUID  `json:"project_id,omitempty"`
	Title           string      `json:"title"`
	Description     string      `json:"description,omitempty"`
	Priority        int16       `json:"priority,omitempty"` // 1 - 4
	Status          string      `json:"status,omitempty"`   // todo | in_progress | done
	DueAt           *time.Time  `json:"due_at,omitempty"`
	EstimateMinutes *int32      `json:"estimate_minutes,omitempty"`
	TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
}

type UpdateTaskRequest struct {
	ProjectID       *uuid.UUID  `json:"project_id,omitempty"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	Priority        int16       `json:"priority"`
	Status          string      `json:"status"`
	DueAt           *time.Time  `json:"due_at,omitempty"`
	EstimateMinutes *int32      `json:"estimate_minutes,omitempty"`
	TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
}

type TaskFilter struct {
	Status    string
	ProjectID *uuid.UUID
	Priority  *int16
	DueFrom   *time.Time
	DueTo     *time.Time
	Query     string
	SortBy    string
	SortOrder string
	Limit     int32
	Offset    int32
}

type TaskResponse struct {
	ID              uuid.UUID         `json:"id"`
	UserID          uuid.UUID         `json:"user_id"`
	ProjectID       *uuid.UUID        `json:"project_id,omitempty"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Priority        int16             `json:"priority"`
	Status          string            `json:"status"`
	DueAt           *time.Time        `json:"due_at,omitempty"`
	EstimateMinutes *int32            `json:"estimate_minutes,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Subtasks        []SubtaskResponse `json:"subtasks,omitempty"`
	Tags            []TagResponse     `json:"tags,omitempty"`
}

// Time Block models
type CreateTimeBlockRequest struct {
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
}

type UpdateTimeBlockRequest struct {
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
}

type TimeBlockResponse struct {
	ID       uuid.UUID  `json:"id"`
	UserID   uuid.UUID  `json:"user_id"`
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
}

// Workload models
type WorkloadDay struct {
	Date            string `json:"date"` // YYYY-MM-DD
	PlannedMinutes  int32  `json:"planned_minutes"`
	CapacityMinutes int32  `json:"capacity_minutes"`
	OverCapacity    bool   `json:"over_capacity"`
	TaskCount       int    `json:"task_count"`
}

type WorkloadResponse struct {
	From string        `json:"from"`
	To   string        `json:"to"`
	Days []WorkloadDay `json:"days"`
}

// Dashboard models
type DashboardResponse struct {
	Overdue   []TaskResponse `json:"overdue"`
	Today     []TaskResponse `json:"today"`
	Upcoming  []TaskResponse `json:"upcoming"`
	Workload  []WorkloadDay  `json:"workload"`
}
