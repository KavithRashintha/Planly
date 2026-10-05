package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/planner/internal/model"
	"github.com/planly/services/planner/internal/service"
)

type PlannerHandler struct {
	svc service.PlannerService
}

func NewPlannerHandler(svc service.PlannerService) *PlannerHandler {
	return &PlannerHandler{svc: svc}
}

func (h *PlannerHandler) RegisterRoutes(r chi.Router, jwtSecret []byte) {
	r.Group(func(protected chi.Router) {
		protected.Use(jwtx.AuthMiddleware(jwtSecret))

		// Projects
		protected.Get("/projects", h.ListProjects)
		protected.Post("/projects", h.CreateProject)
		protected.Get("/projects/{id}", h.GetProject)
		protected.Put("/projects/{id}", h.UpdateProject)
		protected.Delete("/projects/{id}", h.DeleteProject)

		// Tasks
		protected.Get("/tasks", h.ListTasks)
		protected.Post("/tasks", h.CreateTask)
		protected.Get("/tasks/{id}", h.GetTask)
		protected.Put("/tasks/{id}", h.UpdateTask)
		protected.Delete("/tasks/{id}", h.DeleteTask)

		// Subtasks
		protected.Post("/tasks/{id}/subtasks", h.CreateSubtask)
		protected.Put("/subtasks/{id}", h.UpdateSubtask)
		protected.Delete("/subtasks/{id}", h.DeleteSubtask)

		// Tags
		protected.Get("/tags", h.ListTags)
		protected.Post("/tags", h.CreateTag)

		// Time Blocks
		protected.Get("/time-blocks", h.ListTimeBlocks)
		protected.Post("/time-blocks", h.CreateTimeBlock)
		protected.Get("/time-blocks/{id}", h.GetTimeBlock)
		protected.Put("/time-blocks/{id}", h.UpdateTimeBlock)
		protected.Delete("/time-blocks/{id}", h.DeleteTimeBlock)

		// Workload & Dashboard
		protected.Get("/workload", h.GetWorkload)
		protected.Get("/dashboard", h.GetDashboard)
	})
}

// ----------------- Projects -----------------

func (h *PlannerHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.CreateProjectRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateProject(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create project")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *PlannerHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid project ID format")
		return
	}

	resp, err := h.svc.GetProject(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Project not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get project")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	resp, err := h.svc.ListProjects(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list projects")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid project ID format")
		return
	}

	var req model.UpdateProjectRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.UpdateProject(r.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Project not found")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update project")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid project ID format")
		return
	}

	if err := h.svc.DeleteProject(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Project not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ----------------- Tasks -----------------

func (h *PlannerHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.CreateTaskRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateTask(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create task")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *PlannerHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid task ID format")
		return
	}

	resp, err := h.svc.GetTask(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get task")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid task ID format")
		return
	}

	var req model.UpdateTaskRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.UpdateTask(r.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update task")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid task ID format")
		return
	}

	if err := h.svc.DeleteTask(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PlannerHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	q := r.URL.Query()
	filter := model.TaskFilter{
		Status:    q.Get("status"),
		Query:     q.Get("q"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}

	if pIDStr := q.Get("project_id"); pIDStr != "" {
		if pID, err := uuid.Parse(pIDStr); err == nil {
			filter.ProjectID = &pID
		}
	}

	if prioStr := q.Get("priority"); prioStr != "" {
		if prio, err := strconv.Atoi(prioStr); err == nil {
			p := int16(prio)
			filter.Priority = &p
		}
	}

	if dueFromStr := q.Get("due_from"); dueFromStr != "" {
		if t, err := time.Parse(time.RFC3339, dueFromStr); err == nil {
			filter.DueFrom = &t
		}
	}

	if dueToStr := q.Get("due_to"); dueToStr != "" {
		if t, err := time.Parse(time.RFC3339, dueToStr); err == nil {
			filter.DueTo = &t
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = int32(l)
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = int32(o)
		}
	}

	tasks, totalCount, err := h.svc.ListTasks(r.Context(), userID, filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list tasks")
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", totalCount))
	_ = httpx.WriteJSON(w, http.StatusOK, tasks)
}

// ----------------- Subtasks -----------------

func (h *PlannerHandler) CreateSubtask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid task ID format")
		return
	}

	var req model.CreateSubtaskRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateSubtask(r.Context(), userID, taskID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Task not found")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create subtask")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *PlannerHandler) UpdateSubtask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid subtask ID format")
		return
	}

	var req model.UpdateSubtaskRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.UpdateSubtask(r.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Subtask not found")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update subtask")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) DeleteSubtask(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid subtask ID format")
		return
	}

	if err := h.svc.DeleteSubtask(r.Context(), userID, id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete subtask")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ----------------- Tags -----------------

func (h *PlannerHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.CreateTagRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateTag(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create tag")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *PlannerHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	tags, err := h.svc.ListTags(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list tags")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, tags)
}

// ----------------- Time Blocks -----------------

func (h *PlannerHandler) CreateTimeBlock(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req model.CreateTimeBlockRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.CreateTimeBlock(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrTimeBlockOverlap) {
			httpx.WriteError(w, http.StatusConflict, "time_block_conflict", "Scheduled slot overlaps with an existing time block")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create time block")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *PlannerHandler) GetTimeBlock(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid time block ID format")
		return
	}

	resp, err := h.svc.GetTimeBlock(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Time block not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get time block")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) ListTimeBlocks(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	q := r.URL.Query()
	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC().AddDate(0, 0, 30)

	if fStr := q.Get("from"); fStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fStr); err == nil {
			from = parsed.UTC()
		}
	}
	if tStr := q.Get("to"); tStr != "" {
		if parsed, err := time.Parse(time.RFC3339, tStr); err == nil {
			to = parsed.UTC()
		}
	}

	blocks, err := h.svc.ListTimeBlocks(r.Context(), userID, from, to)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list time blocks")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, blocks)
}

func (h *PlannerHandler) UpdateTimeBlock(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid time block ID format")
		return
	}

	var req model.UpdateTimeBlockRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	resp, err := h.svc.UpdateTimeBlock(r.Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrTimeBlockOverlap) {
			httpx.WriteError(w, http.StatusConflict, "time_block_conflict", "Scheduled slot overlaps with an existing time block")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Time block not found")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update time block")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *PlannerHandler) DeleteTimeBlock(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid time block ID format")
		return
	}

	if err := h.svc.DeleteTimeBlock(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Time block not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete time block")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ----------------- Workload & Dashboard -----------------

func (h *PlannerHandler) GetWorkload(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	q := r.URL.Query()
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)

	if fStr := q.Get("from"); fStr != "" {
		if parsed, err := time.Parse("2006-01-02", fStr); err == nil {
			from = parsed
		} else if parsedRFC, err := time.Parse(time.RFC3339, fStr); err == nil {
			from = parsedRFC
		}
	}
	if tStr := q.Get("to"); tStr != "" {
		if parsed, err := time.Parse("2006-01-02", tStr); err == nil {
			to = parsed
		} else if parsedRFC, err := time.Parse(time.RFC3339, tStr); err == nil {
			to = parsedRFC
		}
	}

	capacityMinutes := int32(480) // 8h default
	if capStr := q.Get("capacity"); capStr != "" {
		if c, err := strconv.Atoi(capStr); err == nil && c > 0 {
			capacityMinutes = int32(c)
		}
	}

	workload, err := h.svc.GetWorkload(r.Context(), userID, from, to, capacityMinutes)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to calculate workload")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, workload)
}

func (h *PlannerHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtx.GetUserID(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	capacityMinutes := int32(480)
	if capStr := r.URL.Query().Get("capacity"); capStr != "" {
		if c, err := strconv.Atoi(capStr); err == nil && c > 0 {
			capacityMinutes = int32(c)
		}
	}

	dashboard, err := h.svc.GetDashboard(r.Context(), userID, capacityMinutes)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve dashboard")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, dashboard)
}
