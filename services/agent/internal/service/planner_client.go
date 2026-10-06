package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPlannerNotFound = errors.New("planner resource not found")
	ErrPlannerConflict = errors.New("planner resource conflict")
	ErrPlannerBadReq   = errors.New("planner invalid request")
	ErrPlannerInternal = errors.New("planner internal error")
)

// DTOs for Planner interactions
type SubtaskDTO struct {
	ID       uuid.UUID `json:"id"`
	TaskID   uuid.UUID `json:"task_id"`
	Title    string    `json:"title"`
	Done     bool      `json:"done"`
	Position int32     `json:"position"`
}

type TagDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type TaskDTO struct {
	ID              uuid.UUID    `json:"id"`
	UserID          uuid.UUID    `json:"user_id"`
	ProjectID       *uuid.UUID   `json:"project_id,omitempty"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Priority        int16        `json:"priority"`
	Status          string       `json:"status"`
	DueAt           *time.Time   `json:"due_at,omitempty"`
	EstimateMinutes *int32       `json:"estimate_minutes,omitempty"`
	CompletedAt     *time.Time   `json:"completed_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	Subtasks        []SubtaskDTO `json:"subtasks,omitempty"`
	Tags            []TagDTO     `json:"tags,omitempty"`
}

type ProjectDTO struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Colour    string    `json:"colour"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TimeBlockDTO struct {
	ID       uuid.UUID  `json:"id"`
	UserID   uuid.UUID  `json:"user_id"`
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
}

type WorkloadDayDTO struct {
	Date            string `json:"date"`
	PlannedMinutes  int32  `json:"planned_minutes"`
	CapacityMinutes int32  `json:"capacity_minutes"`
	OverCapacity    bool   `json:"over_capacity"`
	TaskCount       int    `json:"task_count"`
}

type WorkloadDTO struct {
	From string           `json:"from"`
	To   string           `json:"to"`
	Days []WorkloadDayDTO `json:"days"`
}

type CreateTaskDTO struct {
	ProjectID       *uuid.UUID  `json:"project_id,omitempty"`
	Title           string      `json:"title"`
	Description     string      `json:"description,omitempty"`
	Priority        int16       `json:"priority,omitempty"`
	Status          string      `json:"status,omitempty"`
	DueAt           *time.Time  `json:"due_at,omitempty"`
	EstimateMinutes *int32      `json:"estimate_minutes,omitempty"`
	TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
}

type UpdateTaskDTO struct {
	ProjectID       *uuid.UUID  `json:"project_id,omitempty"`
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	Priority        int16       `json:"priority"`
	Status          string      `json:"status"`
	DueAt           *time.Time  `json:"due_at,omitempty"`
	EstimateMinutes *int32      `json:"estimate_minutes,omitempty"`
	TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
}

type CreateTimeBlockDTO struct {
	TaskID   *uuid.UUID `json:"task_id,omitempty"`
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"starts_at"`
	EndsAt   time.Time  `json:"ends_at"`
}

type PlannerClient interface {
	ListTasks(ctx context.Context, authHeader string, queryParams map[string]string) ([]TaskDTO, error)
	GetTask(ctx context.Context, authHeader string, taskID uuid.UUID) (*TaskDTO, error)
	ListProjects(ctx context.Context, authHeader string) ([]ProjectDTO, error)
	ListTimeBlocks(ctx context.Context, authHeader string, from, to string) ([]TimeBlockDTO, error)
	GetWorkload(ctx context.Context, authHeader string, from, to string) (*WorkloadDTO, error)
	CreateTask(ctx context.Context, authHeader string, req CreateTaskDTO) (*TaskDTO, error)
	UpdateTask(ctx context.Context, authHeader string, taskID uuid.UUID, req UpdateTaskDTO) (*TaskDTO, error)
	DeleteTask(ctx context.Context, authHeader string, taskID uuid.UUID) error
	CreateTimeBlock(ctx context.Context, authHeader string, req CreateTimeBlockDTO) (*TimeBlockDTO, error)
}

type HTTPPlannerClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPlannerClient(baseURL string) *HTTPPlannerClient {
	return &HTTPPlannerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *HTTPPlannerClient) doRequest(ctx context.Context, method, path, authHeader string, body any, out any) error {
	fullURL := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create planner request: %w", err)
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("planner client request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out != nil && resp.StatusCode != http.StatusNoContent {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("failed to decode planner response: %w", err)
			}
		}
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrPlannerNotFound, string(respBody))
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", ErrPlannerConflict, string(respBody))
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrPlannerBadReq, string(respBody))
	default:
		return fmt.Errorf("%w (status %d): %s", ErrPlannerInternal, resp.StatusCode, string(respBody))
	}
}

func (c *HTTPPlannerClient) ListTasks(ctx context.Context, authHeader string, queryParams map[string]string) ([]TaskDTO, error) {
	u, err := url.Parse("/tasks")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range queryParams {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	var tasks []TaskDTO
	if err := c.doRequest(ctx, http.MethodGet, u.String(), authHeader, nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *HTTPPlannerClient) GetTask(ctx context.Context, authHeader string, taskID uuid.UUID) (*TaskDTO, error) {
	var task TaskDTO
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/tasks/%s", taskID), authHeader, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *HTTPPlannerClient) ListProjects(ctx context.Context, authHeader string) ([]ProjectDTO, error) {
	var projects []ProjectDTO
	if err := c.doRequest(ctx, http.MethodGet, "/projects", authHeader, nil, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *HTTPPlannerClient) ListTimeBlocks(ctx context.Context, authHeader string, from, to string) ([]TimeBlockDTO, error) {
	path := "/time-blocks"
	if from != "" || to != "" {
		v := url.Values{}
		if from != "" {
			v.Set("from", from)
		}
		if to != "" {
			v.Set("to", to)
		}
		path += "?" + v.Encode()
	}

	var blocks []TimeBlockDTO
	if err := c.doRequest(ctx, http.MethodGet, path, authHeader, nil, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func (c *HTTPPlannerClient) GetWorkload(ctx context.Context, authHeader string, from, to string) (*WorkloadDTO, error) {
	path := "/workload"
	if from != "" || to != "" {
		v := url.Values{}
		if from != "" {
			v.Set("from", from)
		}
		if to != "" {
			v.Set("to", to)
		}
		path += "?" + v.Encode()
	}

	var workload WorkloadDTO
	if err := c.doRequest(ctx, http.MethodGet, path, authHeader, nil, &workload); err != nil {
		return nil, err
	}
	return &workload, nil
}

func (c *HTTPPlannerClient) CreateTask(ctx context.Context, authHeader string, req CreateTaskDTO) (*TaskDTO, error) {
	var task TaskDTO
	if err := c.doRequest(ctx, http.MethodPost, "/tasks", authHeader, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *HTTPPlannerClient) UpdateTask(ctx context.Context, authHeader string, taskID uuid.UUID, req UpdateTaskDTO) (*TaskDTO, error) {
	var task TaskDTO
	if err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/tasks/%s", taskID), authHeader, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *HTTPPlannerClient) DeleteTask(ctx context.Context, authHeader string, taskID uuid.UUID) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/tasks/%s", taskID), authHeader, nil, nil)
}

func (c *HTTPPlannerClient) CreateTimeBlock(ctx context.Context, authHeader string, req CreateTimeBlockDTO) (*TimeBlockDTO, error) {
	var block TimeBlockDTO
	if err := c.doRequest(ctx, http.MethodPost, "/time-blocks", authHeader, req, &block); err != nil {
		return nil, err
	}
	return &block, nil
}
