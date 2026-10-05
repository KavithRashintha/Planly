package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/planly/services/planner/internal/model"
	"github.com/planly/services/planner/internal/repository"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrValidation       = errors.New("validation error")
	ErrTimeBlockOverlap = errors.New("time block overlaps with an existing scheduled block")
)

type PlannerService interface {
	// Projects
	CreateProject(ctx context.Context, userID uuid.UUID, req model.CreateProjectRequest) (*model.ProjectResponse, error)
	GetProject(ctx context.Context, userID, id uuid.UUID) (*model.ProjectResponse, error)
	ListProjects(ctx context.Context, userID uuid.UUID) ([]model.ProjectResponse, error)
	UpdateProject(ctx context.Context, userID, id uuid.UUID, req model.UpdateProjectRequest) (*model.ProjectResponse, error)
	DeleteProject(ctx context.Context, userID, id uuid.UUID) error

	// Tasks
	CreateTask(ctx context.Context, userID uuid.UUID, req model.CreateTaskRequest) (*model.TaskResponse, error)
	GetTask(ctx context.Context, userID, id uuid.UUID) (*model.TaskResponse, error)
	UpdateTask(ctx context.Context, userID, id uuid.UUID, req model.UpdateTaskRequest) (*model.TaskResponse, error)
	DeleteTask(ctx context.Context, userID, id uuid.UUID) error
	ListTasks(ctx context.Context, userID uuid.UUID, filter model.TaskFilter) ([]model.TaskResponse, int64, error)

	// Subtasks
	CreateSubtask(ctx context.Context, userID, taskID uuid.UUID, req model.CreateSubtaskRequest) (*model.SubtaskResponse, error)
	UpdateSubtask(ctx context.Context, userID, id uuid.UUID, req model.UpdateSubtaskRequest) (*model.SubtaskResponse, error)
	DeleteSubtask(ctx context.Context, userID, id uuid.UUID) error

	// Tags
	CreateTag(ctx context.Context, userID uuid.UUID, req model.CreateTagRequest) (*model.TagResponse, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]model.TagResponse, error)

	// Time Blocks
	CreateTimeBlock(ctx context.Context, userID uuid.UUID, req model.CreateTimeBlockRequest) (*model.TimeBlockResponse, error)
	GetTimeBlock(ctx context.Context, userID, id uuid.UUID) (*model.TimeBlockResponse, error)
	ListTimeBlocks(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.TimeBlockResponse, error)
	UpdateTimeBlock(ctx context.Context, userID, id uuid.UUID, req model.UpdateTimeBlockRequest) (*model.TimeBlockResponse, error)
	DeleteTimeBlock(ctx context.Context, userID, id uuid.UUID) error

	// Workload & Dashboard
	GetWorkload(ctx context.Context, userID uuid.UUID, from, to time.Time, capacityMinutes int32) (*model.WorkloadResponse, error)
	GetDashboard(ctx context.Context, userID uuid.UUID, capacityMinutes int32) (*model.DashboardResponse, error)
}

type plannerService struct {
	pool *pgxpool.Pool
	repo repository.Querier
}

func NewPlannerService(pool *pgxpool.Pool, repo repository.Querier) PlannerService {
	return &plannerService{
		pool: pool,
		repo: repo,
	}
}

// ----------------- Projects -----------------

func (s *plannerService) CreateProject(ctx context.Context, userID uuid.UUID, req model.CreateProjectRequest) (*model.ProjectResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", ErrValidation)
	}
	colour := req.Colour
	if colour == "" {
		colour = "#3b82f6"
	}

	p, err := s.repo.CreateProject(ctx, repository.CreateProjectParams{
		UserID: UUIDToPgtype(userID),
		Name:   name,
		Colour: colour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	resp := MapProjectToResponse(p)
	return &resp, nil
}

func (s *plannerService) GetProject(ctx context.Context, userID, id uuid.UUID) (*model.ProjectResponse, error) {
	p, err := s.repo.GetProjectByID(ctx, repository.GetProjectByIDParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	resp := MapProjectToResponse(p)
	return &resp, nil
}

func (s *plannerService) ListProjects(ctx context.Context, userID uuid.UUID) ([]model.ProjectResponse, error) {
	projects, err := s.repo.ListProjects(ctx, UUIDToPgtype(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	resp := make([]model.ProjectResponse, len(projects))
	for i, p := range projects {
		resp[i] = MapProjectToResponse(p)
	}
	return resp, nil
}

func (s *plannerService) UpdateProject(ctx context.Context, userID, id uuid.UUID, req model.UpdateProjectRequest) (*model.ProjectResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", ErrValidation)
	}
	colour := req.Colour
	if colour == "" {
		colour = "#3b82f6"
	}

	p, err := s.repo.UpdateProject(ctx, repository.UpdateProjectParams{
		ID:       UUIDToPgtype(id),
		UserID:   UUIDToPgtype(userID),
		Name:     name,
		Colour:   colour,
		Archived: req.Archived,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	resp := MapProjectToResponse(p)
	return &resp, nil
}

func (s *plannerService) DeleteProject(ctx context.Context, userID, id uuid.UUID) error {
	// First check ownership
	if _, err := s.GetProject(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteProject(ctx, repository.DeleteProjectParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
}

// ----------------- Tasks -----------------

func (s *plannerService) CreateTask(ctx context.Context, userID uuid.UUID, req model.CreateTaskRequest) (*model.TaskResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: task title is required", ErrValidation)
	}

	priority := req.Priority
	if priority == 0 {
		priority = 2
	}
	if priority < 1 || priority > 4 {
		return nil, fmt.Errorf("%w: priority must be between 1 and 4", ErrValidation)
	}

	status := req.Status
	if status == "" {
		status = "todo"
	}
	if status != "todo" && status != "in_progress" && status != "done" {
		return nil, fmt.Errorf("%w: status must be todo, in_progress, or done", ErrValidation)
	}

	var completedAt pgtype.Timestamptz
	if status == "done" {
		completedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	if req.EstimateMinutes != nil && *req.EstimateMinutes <= 0 {
		return nil, fmt.Errorf("%w: estimate_minutes must be greater than 0", ErrValidation)
	}

	var estimate pgtype.Int4
	if req.EstimateMinutes != nil {
		estimate = pgtype.Int4{Int32: *req.EstimateMinutes, Valid: true}
	}

	// Verify project belongs to user if specified
	if req.ProjectID != nil {
		if _, err := s.GetProject(ctx, userID, *req.ProjectID); err != nil {
			return nil, fmt.Errorf("%w: invalid project_id", ErrValidation)
		}
	}

	task, err := s.repo.CreateTask(ctx, repository.CreateTaskParams{
		UserID:          UUIDToPgtype(userID),
		ProjectID:       UUIDPtrToPgtype(req.ProjectID),
		Title:           title,
		Description:     req.Description,
		Priority:        priority,
		Status:          status,
		DueAt:           TimePtrToPgtype(req.DueAt),
		EstimateMinutes: estimate,
		CompletedAt:     completedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Attach tags if provided
	for _, tagID := range req.TagIDs {
		_ = s.repo.AttachTagToTask(ctx, repository.AttachTagToTaskParams{
			TaskID: task.ID,
			TagID:  UUIDToPgtype(tagID),
		})
	}

	return s.GetTask(ctx, userID, PgtypeToUUID(task.ID))
}

func (s *plannerService) GetTask(ctx context.Context, userID, id uuid.UUID) (*model.TaskResponse, error) {
	task, err := s.repo.GetTaskByID(ctx, repository.GetTaskByIDParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	subtasks, err := s.repo.ListSubtasksByTaskID(ctx, repository.ListSubtasksByTaskIDParams{
		ID:     task.ID,
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subtasks: %w", err)
	}

	tags, err := s.repo.ListTagsByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	subtaskResponses := make([]model.SubtaskResponse, len(subtasks))
	for i, sub := range subtasks {
		subtaskResponses[i] = MapSubtaskToResponse(sub)
	}

	tagResponses := make([]model.TagResponse, len(tags))
	for i, tag := range tags {
		tagResponses[i] = MapTagToResponse(tag)
	}

	var est *int32
	if task.EstimateMinutes.Valid {
		val := task.EstimateMinutes.Int32
		est = &val
	}

	return &model.TaskResponse{
		ID:              PgtypeToUUID(task.ID),
		UserID:          PgtypeToUUID(task.UserID),
		ProjectID:       PgtypeToUUIDPtr(task.ProjectID),
		Title:           task.Title,
		Description:     task.Description,
		Priority:        task.Priority,
		Status:          task.Status,
		DueAt:           PgtypeToTimePtr(task.DueAt),
		EstimateMinutes: est,
		CompletedAt:     PgtypeToTimePtr(task.CompletedAt),
		CreatedAt:       task.CreatedAt.Time.UTC(),
		UpdatedAt:       task.UpdatedAt.Time.UTC(),
		Subtasks:        subtaskResponses,
		Tags:            tagResponses,
	}, nil
}

func (s *plannerService) UpdateTask(ctx context.Context, userID, id uuid.UUID, req model.UpdateTaskRequest) (*model.TaskResponse, error) {
	existing, err := s.GetTask(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: task title is required", ErrValidation)
	}

	priority := req.Priority
	if priority < 1 || priority > 4 {
		return nil, fmt.Errorf("%w: priority must be between 1 and 4", ErrValidation)
	}

	status := req.Status
	if status != "todo" && status != "in_progress" && status != "done" {
		return nil, fmt.Errorf("%w: status must be todo, in_progress, or done", ErrValidation)
	}

	// Set or clear completed_at
	var completedAt pgtype.Timestamptz
	if status == "done" {
		if existing.Status != "done" || existing.CompletedAt == nil {
			completedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
		} else {
			completedAt = pgtype.Timestamptz{Time: *existing.CompletedAt, Valid: true}
		}
	} else {
		completedAt = pgtype.Timestamptz{Valid: false}
	}

	var estimate pgtype.Int4
	if req.EstimateMinutes != nil {
		if *req.EstimateMinutes <= 0 {
			return nil, fmt.Errorf("%w: estimate_minutes must be greater than 0", ErrValidation)
		}
		estimate = pgtype.Int4{Int32: *req.EstimateMinutes, Valid: true}
	}

	if req.ProjectID != nil {
		if _, err := s.GetProject(ctx, userID, *req.ProjectID); err != nil {
			return nil, fmt.Errorf("%w: invalid project_id", ErrValidation)
		}
	}

	_, err = s.repo.UpdateTask(ctx, repository.UpdateTaskParams{
		ID:              UUIDToPgtype(id),
		UserID:          UUIDToPgtype(userID),
		ProjectID:       UUIDPtrToPgtype(req.ProjectID),
		Title:           title,
		Description:     req.Description,
		Priority:        priority,
		Status:          status,
		DueAt:           TimePtrToPgtype(req.DueAt),
		EstimateMinutes: estimate,
		CompletedAt:     completedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Update tags if provided
	if req.TagIDs != nil {
		_ = s.repo.DeleteTaskTagsByTaskID(ctx, UUIDToPgtype(id))
		for _, tagID := range req.TagIDs {
			_ = s.repo.AttachTagToTask(ctx, repository.AttachTagToTaskParams{
				TaskID: UUIDToPgtype(id),
				TagID:  UUIDToPgtype(tagID),
			})
		}
	}

	return s.GetTask(ctx, userID, id)
}

func (s *plannerService) DeleteTask(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.GetTask(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteTask(ctx, repository.DeleteTaskParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
}

func (s *plannerService) ListTasks(ctx context.Context, userID uuid.UUID, filter model.TaskFilter) ([]model.TaskResponse, int64, error) {
	// Construct dynamic query
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, *filter.ProjectID)
		argIdx++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *filter.Priority)
		argIdx++
	}

	if filter.DueFrom != nil {
		conditions = append(conditions, fmt.Sprintf("due_at >= $%d", argIdx))
		args = append(args, filter.DueFrom.UTC())
		argIdx++
	}

	if filter.DueTo != nil {
		conditions = append(conditions, fmt.Sprintf("due_at <= $%d", argIdx))
		args = append(args, filter.DueTo.UTC())
		argIdx++
	}

	if filter.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Query+"%")
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", whereClause)
	var totalCount int64
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Ordering
	orderBy := "created_at DESC"
	sortCol := strings.ToLower(filter.SortBy)
	orderDir := "ASC"
	if strings.ToUpper(filter.SortOrder) == "DESC" {
		orderDir = "DESC"
	}
	if sortCol == "due_at" || sortCol == "priority" || sortCol == "created_at" || sortCol == "title" {
		orderBy = fmt.Sprintf("%s %s", sortCol, orderDir)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		"SELECT id, user_id, project_id, title, description, priority, status, due_at, estimate_minutes, completed_at, created_at, updated_at FROM tasks WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
		whereClause, orderBy, limit, offset,
	)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.TaskResponse
	for rows.Next() {
		var t repository.Task
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.ProjectID, &t.Title, &t.Description,
			&t.Priority, &t.Status, &t.DueAt, &t.EstimateMinutes,
			&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}

		var est *int32
		if t.EstimateMinutes.Valid {
			v := t.EstimateMinutes.Int32
			est = &v
		}

		tasks = append(tasks, model.TaskResponse{
			ID:              PgtypeToUUID(t.ID),
			UserID:          PgtypeToUUID(t.UserID),
			ProjectID:       PgtypeToUUIDPtr(t.ProjectID),
			Title:           t.Title,
			Description:     t.Description,
			Priority:        t.Priority,
			Status:          t.Status,
			DueAt:           PgtypeToTimePtr(t.DueAt),
			EstimateMinutes: est,
			CompletedAt:     PgtypeToTimePtr(t.CompletedAt),
			CreatedAt:       t.CreatedAt.Time.UTC(),
			UpdatedAt:       t.UpdatedAt.Time.UTC(),
		})
	}

	return tasks, totalCount, nil
}

// ----------------- Subtasks -----------------

func (s *plannerService) CreateSubtask(ctx context.Context, userID, taskID uuid.UUID, req model.CreateSubtaskRequest) (*model.SubtaskResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: subtask title is required", ErrValidation)
	}

	// Verify task ownership
	if _, err := s.GetTask(ctx, userID, taskID); err != nil {
		return nil, err
	}

	sub, err := s.repo.CreateSubtask(ctx, repository.CreateSubtaskParams{
		TaskID:   UUIDToPgtype(taskID),
		Title:    title,
		Position: req.Position,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subtask: %w", err)
	}

	resp := MapSubtaskToResponse(sub)
	return &resp, nil
}

func (s *plannerService) UpdateSubtask(ctx context.Context, userID, id uuid.UUID, req model.UpdateSubtaskRequest) (*model.SubtaskResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: subtask title is required", ErrValidation)
	}

	sub, err := s.repo.UpdateSubtask(ctx, repository.UpdateSubtaskParams{
		ID:       UUIDToPgtype(id),
		UserID:   UUIDToPgtype(userID),
		Title:    title,
		Done:     req.Done,
		Position: req.Position,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update subtask: %w", err)
	}

	resp := MapSubtaskToResponse(sub)
	return &resp, nil
}

func (s *plannerService) DeleteSubtask(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteSubtask(ctx, repository.DeleteSubtaskParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
}

// ----------------- Tags -----------------

func (s *plannerService) CreateTag(ctx context.Context, userID uuid.UUID, req model.CreateTagRequest) (*model.TagResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: tag name is required", ErrValidation)
	}

	tag, err := s.repo.CreateTag(ctx, repository.CreateTagParams{
		UserID: UUIDToPgtype(userID),
		Name:   name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	resp := MapTagToResponse(tag)
	return &resp, nil
}

func (s *plannerService) ListTags(ctx context.Context, userID uuid.UUID) ([]model.TagResponse, error) {
	tags, err := s.repo.ListTags(ctx, UUIDToPgtype(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	resp := make([]model.TagResponse, len(tags))
	for i, t := range tags {
		resp[i] = MapTagToResponse(t)
	}
	return resp, nil
}

// ----------------- Time Blocks -----------------

func (s *plannerService) CreateTimeBlock(ctx context.Context, userID uuid.UUID, req model.CreateTimeBlockRequest) (*model.TimeBlockResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: time block title is required", ErrValidation)
	}

	startsAt := req.StartsAt.UTC()
	endsAt := req.EndsAt.UTC()
	if !endsAt.After(startsAt) {
		return nil, fmt.Errorf("%w: ends_at must be strictly after starts_at", ErrValidation)
	}

	// Verify task ownership if task_id is passed
	if req.TaskID != nil {
		if _, err := s.GetTask(ctx, userID, *req.TaskID); err != nil {
			return nil, fmt.Errorf("%w: invalid task_id", ErrValidation)
		}
	}

	// Overlap check in transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

	overlapCount, err := qtx.CheckTimeBlockOverlap(ctx, repository.CheckTimeBlockOverlapParams{
		UserID:           UUIDToPgtype(userID),
		ID:               UUIDToPgtype(uuid.Nil),
		ProposedStartsAt: TimePtrToPgtype(&startsAt),
		ProposedEndsAt:   TimePtrToPgtype(&endsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check time block overlap: %w", err)
	}
	if overlapCount > 0 {
		return nil, ErrTimeBlockOverlap
	}

	block, err := qtx.CreateTimeBlock(ctx, repository.CreateTimeBlockParams{
		UserID:   UUIDToPgtype(userID),
		TaskID:   UUIDPtrToPgtype(req.TaskID),
		Title:    title,
		StartsAt: TimePtrToPgtype(&startsAt),
		EndsAt:   TimePtrToPgtype(&endsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create time block: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit time block: %w", err)
	}

	resp := MapTimeBlockToResponse(block)
	return &resp, nil
}

func (s *plannerService) GetTimeBlock(ctx context.Context, userID, id uuid.UUID) (*model.TimeBlockResponse, error) {
	block, err := s.repo.GetTimeBlockByID(ctx, repository.GetTimeBlockByIDParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get time block: %w", err)
	}

	resp := MapTimeBlockToResponse(block)
	return &resp, nil
}

func (s *plannerService) ListTimeBlocks(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]model.TimeBlockResponse, error) {
	if to.IsZero() {
		to = from.Add(30 * 24 * time.Hour)
	}
	blocks, err := s.repo.ListTimeBlocks(ctx, repository.ListTimeBlocksParams{
		UserID:   UUIDToPgtype(userID),
		StartsAt: TimePtrToPgtype(&from),
		EndsAt:   TimePtrToPgtype(&to),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list time blocks: %w", err)
	}

	resp := make([]model.TimeBlockResponse, len(blocks))
	for i, b := range blocks {
		resp[i] = MapTimeBlockToResponse(b)
	}
	return resp, nil
}

func (s *plannerService) UpdateTimeBlock(ctx context.Context, userID, id uuid.UUID, req model.UpdateTimeBlockRequest) (*model.TimeBlockResponse, error) {
	if _, err := s.GetTimeBlock(ctx, userID, id); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: time block title is required", ErrValidation)
	}

	startsAt := req.StartsAt.UTC()
	endsAt := req.EndsAt.UTC()
	if !endsAt.After(startsAt) {
		return nil, fmt.Errorf("%w: ends_at must be strictly after starts_at", ErrValidation)
	}

	if req.TaskID != nil {
		if _, err := s.GetTask(ctx, userID, *req.TaskID); err != nil {
			return nil, fmt.Errorf("%w: invalid task_id", ErrValidation)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repository.New(tx)

	overlapCount, err := qtx.CheckTimeBlockOverlap(ctx, repository.CheckTimeBlockOverlapParams{
		UserID:           UUIDToPgtype(userID),
		ID:               UUIDToPgtype(id),
		ProposedStartsAt: TimePtrToPgtype(&startsAt),
		ProposedEndsAt:   TimePtrToPgtype(&endsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check time block overlap: %w", err)
	}
	if overlapCount > 0 {
		return nil, ErrTimeBlockOverlap
	}

	block, err := qtx.UpdateTimeBlock(ctx, repository.UpdateTimeBlockParams{
		ID:       UUIDToPgtype(id),
		UserID:   UUIDToPgtype(userID),
		TaskID:   UUIDPtrToPgtype(req.TaskID),
		Title:    title,
		StartsAt: TimePtrToPgtype(&startsAt),
		EndsAt:   TimePtrToPgtype(&endsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update time block: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit time block: %w", err)
	}

	resp := MapTimeBlockToResponse(block)
	return &resp, nil
}

func (s *plannerService) DeleteTimeBlock(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := s.GetTimeBlock(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteTimeBlock(ctx, repository.DeleteTimeBlockParams{
		ID:     UUIDToPgtype(id),
		UserID: UUIDToPgtype(userID),
	})
}

// ----------------- Workload & Dashboard -----------------

func (s *plannerService) GetWorkload(ctx context.Context, userID uuid.UUID, from, to time.Time, capacityMinutes int32) (*model.WorkloadResponse, error) {
	if capacityMinutes <= 0 {
		capacityMinutes = 480 // fallback 8h = 480 mins
	}

	fromUTC := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, time.UTC)

	tasks, err := s.repo.GetTasksForRange(ctx, repository.GetTasksForRangeParams{
		UserID: UUIDToPgtype(userID),
		DueAt:  TimePtrToPgtype(&fromUTC),
		DueAt_2: TimePtrToPgtype(&toUTC),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks for workload: %w", err)
	}

	blocks, err := s.repo.ListTimeBlocks(ctx, repository.ListTimeBlocksParams{
		UserID:   UUIDToPgtype(userID),
		StartsAt: TimePtrToPgtype(&fromUTC),
		EndsAt:   TimePtrToPgtype(&toUTC),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch time blocks for workload: %w", err)
	}

	type dayAggregate struct {
		plannedMinutes int32
		taskCount      int
	}
	aggregates := make(map[string]*dayAggregate)

	// Pre-fill every day in range
	current := fromUTC
	for !current.After(toUTC) {
		dateStr := current.Format("2006-01-02")
		aggregates[dateStr] = &dayAggregate{}
		current = current.AddDate(0, 0, 1)
	}

	// Accumulate tasks
	for _, t := range tasks {
		if t.DueAt.Valid {
			dStr := t.DueAt.Time.UTC().Format("2006-01-02")
			if agg, ok := aggregates[dStr]; ok {
				agg.taskCount++
				if t.EstimateMinutes.Valid {
					agg.plannedMinutes += t.EstimateMinutes.Int32
				}
			}
		}
	}

	// Accumulate time blocks
	for _, b := range blocks {
		if b.StartsAt.Valid && b.EndsAt.Valid {
			dStr := b.StartsAt.Time.UTC().Format("2006-01-02")
			durationMins := int32(b.EndsAt.Time.Sub(b.StartsAt.Time).Minutes())
			if agg, ok := aggregates[dStr]; ok {
				agg.plannedMinutes += durationMins
			}
		}
	}

	days := make([]model.WorkloadDay, 0, len(aggregates))
	current = fromUTC
	for !current.After(toUTC) {
		dateStr := current.Format("2006-01-02")
		agg := aggregates[dateStr]
		days = append(days, model.WorkloadDay{
			Date:            dateStr,
			PlannedMinutes:  agg.plannedMinutes,
			CapacityMinutes: capacityMinutes,
			OverCapacity:    agg.plannedMinutes > capacityMinutes,
			TaskCount:       agg.taskCount,
		})
		current = current.AddDate(0, 0, 1)
	}

	return &model.WorkloadResponse{
		From: fromUTC.Format("2006-01-02"),
		To:   toUTC.Format("2006-01-02"),
		Days: days,
	}, nil
}

func (s *plannerService) GetDashboard(ctx context.Context, userID uuid.UUID, capacityMinutes int32) (*model.DashboardResponse, error) {
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	endOf7Days := startOfToday.AddDate(0, 0, 7)

	overdueRaw, err := s.repo.GetOverdueTasks(ctx, repository.GetOverdueTasksParams{
		UserID: UUIDToPgtype(userID),
		DueAt:  TimePtrToPgtype(&startOfToday),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch overdue tasks: %w", err)
	}

	todayRaw, err := s.repo.GetTodayTasks(ctx, repository.GetTodayTasksParams{
		UserID:  UUIDToPgtype(userID),
		DueAt:   TimePtrToPgtype(&startOfToday),
		DueAt_2: TimePtrToPgtype(&endOfToday),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch today tasks: %w", err)
	}

	upcomingRaw, err := s.repo.GetUpcomingTasks(ctx, repository.GetUpcomingTasksParams{
		UserID:  UUIDToPgtype(userID),
		DueAt:   TimePtrToPgtype(&endOfToday),
		DueAt_2: TimePtrToPgtype(&endOf7Days),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming tasks: %w", err)
	}

	mapTasks := func(raw []repository.Task) []model.TaskResponse {
		out := make([]model.TaskResponse, len(raw))
		for i, t := range raw {
			var est *int32
			if t.EstimateMinutes.Valid {
				v := t.EstimateMinutes.Int32
				est = &v
			}
			out[i] = model.TaskResponse{
				ID:              PgtypeToUUID(t.ID),
				UserID:          PgtypeToUUID(t.UserID),
				ProjectID:       PgtypeToUUIDPtr(t.ProjectID),
				Title:           t.Title,
				Description:     t.Description,
				Priority:        t.Priority,
				Status:          t.Status,
				DueAt:           PgtypeToTimePtr(t.DueAt),
				EstimateMinutes: est,
				CompletedAt:     PgtypeToTimePtr(t.CompletedAt),
				CreatedAt:       t.CreatedAt.Time.UTC(),
				UpdatedAt:       t.UpdatedAt.Time.UTC(),
			}
		}
		return out
	}

	workloadResp, err := s.GetWorkload(ctx, userID, startOfToday, endOf7Days, capacityMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workload for dashboard: %w", err)
	}

	return &model.DashboardResponse{
		Overdue:  mapTasks(overdueRaw),
		Today:    mapTasks(todayRaw),
		Upcoming: mapTasks(upcomingRaw),
		Workload: workloadResp.Days,
	}, nil
}
