package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/planly/pkg/dbx"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/planner/internal/handler"
	"github.com/planly/services/planner/internal/model"
	"github.com/planly/services/planner/internal/repository"
	"github.com/planly/services/planner/internal/service"
	"github.com/planly/services/planner/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPool   *pgxpool.Pool
	testRouter chi.Router
	testSecret = []byte("planner-test-secret-at-least-32-bytes!")
)

func TestMain(m *testing.M) {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 5432

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := dbx.Config{
		Host:     host,
		Port:     port,
		User:     "postgres",
		Password: "postgres",
		Database: "planner_db",
		SSLMode:  "disable",
		MaxConns: 5,
	}

	pool, err := dbx.Connect(ctx, cfg)
	if err != nil {
		fmt.Printf("Skipping planner tests: DB not available: %v\n", err)
		os.Exit(0)
	}
	testPool = pool

	if err := migrations.Run(ctx, testPool); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	repo := repository.New(testPool)
	plannerSvc := service.NewPlannerService(testPool, repo)
	plannerHandler := handler.NewPlannerHandler(plannerSvc)

	r := chi.NewRouter()
	plannerHandler.RegisterRoutes(r, testSecret)
	testRouter = r

	code := m.Run()
	testPool.Close()
	os.Exit(code)
}

func authHeader(t *testing.T, userID uuid.UUID) string {
	token, err := jwtx.SignToken(userID, testSecret, 1*time.Hour)
	require.NoError(t, err)
	return "Bearer " + token
}

func TestProjectsCRUDAndIsolation(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()

	authA := authHeader(t, userA)
	authB := authHeader(t, userB)

	// 1. User A creates a project
	createBody, _ := json.Marshal(model.CreateProjectRequest{
		Name:   "User A Project",
		Colour: "#10b981",
	})
	req := httptest.NewRequest("POST", "/projects", bytes.NewBuffer(createBody))
	req.Header.Set("Authorization", authA)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var projA model.ProjectResponse
	err := json.Unmarshal(rec.Body.Bytes(), &projA)
	require.NoError(t, err)
	assert.Equal(t, "User A Project", projA.Name)
	assert.Equal(t, userA, projA.UserID)

	// 2. User A can get their project
	reqGet := httptest.NewRequest("GET", fmt.Sprintf("/projects/%s", projA.ID), nil)
	reqGet.Header.Set("Authorization", authA)
	recGet := httptest.NewRecorder()
	testRouter.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)

	// 3. User B CANNOT get User A's project -> 404 (Isolation test)
	reqGetB := httptest.NewRequest("GET", fmt.Sprintf("/projects/%s", projA.ID), nil)
	reqGetB.Header.Set("Authorization", authB)
	recGetB := httptest.NewRecorder()
	testRouter.ServeHTTP(recGetB, reqGetB)
	assert.Equal(t, http.StatusNotFound, recGetB.Code)

	// 4. User A updates project
	updateBody, _ := json.Marshal(model.UpdateProjectRequest{
		Name:     "User A Updated",
		Colour:   "#ef4444",
		Archived: true,
	})
	reqUpdate := httptest.NewRequest("PUT", fmt.Sprintf("/projects/%s", projA.ID), bytes.NewBuffer(updateBody))
	reqUpdate.Header.Set("Authorization", authA)
	recUpdate := httptest.NewRecorder()
	testRouter.ServeHTTP(recUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, recUpdate.Code)

	var updatedProj model.ProjectResponse
	_ = json.Unmarshal(recUpdate.Body.Bytes(), &updatedProj)
	assert.Equal(t, "User A Updated", updatedProj.Name)
	assert.True(t, updatedProj.Archived)

	// 5. User B CANNOT delete User A's project -> 404
	reqDelB := httptest.NewRequest("DELETE", fmt.Sprintf("/projects/%s", projA.ID), nil)
	reqDelB.Header.Set("Authorization", authB)
	recDelB := httptest.NewRecorder()
	testRouter.ServeHTTP(recDelB, reqDelB)
	assert.Equal(t, http.StatusNotFound, recDelB.Code)

	// 6. User A deletes their project -> 204
	reqDelA := httptest.NewRequest("DELETE", fmt.Sprintf("/projects/%s", projA.ID), nil)
	reqDelA.Header.Set("Authorization", authA)
	recDelA := httptest.NewRecorder()
	testRouter.ServeHTTP(recDelA, reqDelA)
	assert.Equal(t, http.StatusNoContent, recDelA.Code)
}

func TestTasksCRUDAndCompletedAt(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)
	otherUser := uuid.New()
	authOther := authHeader(t, otherUser)

	due := time.Now().UTC().Add(24 * time.Hour)
	estimate := int32(60)

	// 1. Create Task (status = todo)
	createReq := model.CreateTaskRequest{
		Title:           "Write Go Microservice",
		Description:     "Clean architecture implementation",
		Priority:        1,
		Status:          "todo",
		DueAt:           &due,
		EstimateMinutes: &estimate,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Authorization", auth)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var task model.TaskResponse
	err := json.Unmarshal(rec.Body.Bytes(), &task)
	require.NoError(t, err)
	assert.Equal(t, "Write Go Microservice", task.Title)
	assert.Equal(t, "todo", task.Status)
	assert.Nil(t, task.CompletedAt) // completed_at must be nil

	// 2. User B cannot read User A's task -> 404
	reqOther := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", task.ID), nil)
	reqOther.Header.Set("Authorization", authOther)
	recOther := httptest.NewRecorder()
	testRouter.ServeHTTP(recOther, reqOther)
	assert.Equal(t, http.StatusNotFound, recOther.Code)

	// 3. Update task status to "done" -> completed_at set
	updateReq := model.UpdateTaskRequest{
		Title:           task.Title,
		Description:     task.Description,
		Priority:        task.Priority,
		Status:          "done",
		DueAt:           task.DueAt,
		EstimateMinutes: task.EstimateMinutes,
	}
	updateBody, _ := json.Marshal(updateReq)
	reqUp := httptest.NewRequest("PUT", fmt.Sprintf("/tasks/%s", task.ID), bytes.NewBuffer(updateBody))
	reqUp.Header.Set("Authorization", auth)
	recUp := httptest.NewRecorder()
	testRouter.ServeHTTP(recUp, reqUp)
	assert.Equal(t, http.StatusOK, recUp.Code)

	var doneTask model.TaskResponse
	_ = json.Unmarshal(recUp.Body.Bytes(), &doneTask)
	assert.Equal(t, "done", doneTask.Status)
	assert.NotNil(t, doneTask.CompletedAt) // completed_at must be populated

	// 4. Update task status back to "in_progress" -> completed_at cleared
	updateReq.Status = "in_progress"
	updateBody, _ = json.Marshal(updateReq)
	reqUpBack := httptest.NewRequest("PUT", fmt.Sprintf("/tasks/%s", task.ID), bytes.NewBuffer(updateBody))
	reqUpBack.Header.Set("Authorization", auth)
	recUpBack := httptest.NewRecorder()
	testRouter.ServeHTTP(recUpBack, reqUpBack)
	assert.Equal(t, http.StatusOK, recUpBack.Code)

	var inProgTask model.TaskResponse
	_ = json.Unmarshal(recUpBack.Body.Bytes(), &inProgTask)
	assert.Equal(t, "in_progress", inProgTask.Status)
	assert.Nil(t, inProgTask.CompletedAt) // completed_at must be cleared
}

func TestSubtasksAndTags(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)

	// Create tag
	tagBody, _ := json.Marshal(model.CreateTagRequest{Name: "Backend"})
	reqTag := httptest.NewRequest("POST", "/tags", bytes.NewBuffer(tagBody))
	reqTag.Header.Set("Authorization", auth)
	recTag := httptest.NewRecorder()
	testRouter.ServeHTTP(recTag, reqTag)
	assert.Equal(t, http.StatusCreated, recTag.Code)

	var tag model.TagResponse
	_ = json.Unmarshal(recTag.Body.Bytes(), &tag)

	// Create task with tag attached
	taskReq := model.CreateTaskRequest{
		Title:    "Task with Subtasks",
		Priority: 2,
		Status:   "todo",
		TagIDs:   []uuid.UUID{tag.ID},
	}
	taskBody, _ := json.Marshal(taskReq)
	reqTask := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskBody))
	reqTask.Header.Set("Authorization", auth)
	recTask := httptest.NewRecorder()
	testRouter.ServeHTTP(recTask, reqTask)
	assert.Equal(t, http.StatusCreated, recTask.Code)

	var task model.TaskResponse
	_ = json.Unmarshal(recTask.Body.Bytes(), &task)
	assert.Len(t, task.Tags, 1)
	assert.Equal(t, "Backend", task.Tags[0].Name)

	// Add subtask
	subReq := model.CreateSubtaskRequest{Title: "Subtask 1", Position: 1}
	subBody, _ := json.Marshal(subReq)
	reqSub := httptest.NewRequest("POST", fmt.Sprintf("/tasks/%s/subtasks", task.ID), bytes.NewBuffer(subBody))
	reqSub.Header.Set("Authorization", auth)
	recSub := httptest.NewRecorder()
	testRouter.ServeHTTP(recSub, reqSub)
	assert.Equal(t, http.StatusCreated, recSub.Code)

	var sub model.SubtaskResponse
	_ = json.Unmarshal(recSub.Body.Bytes(), &sub)
	assert.Equal(t, "Subtask 1", sub.Title)
	assert.False(t, sub.Done)

	// Toggle subtask
	toggleReq := model.UpdateSubtaskRequest{Title: "Subtask 1", Done: true, Position: 1}
	toggleBody, _ := json.Marshal(toggleReq)
	reqToggle := httptest.NewRequest("PUT", fmt.Sprintf("/subtasks/%s", sub.ID), bytes.NewBuffer(toggleBody))
	reqToggle.Header.Set("Authorization", auth)
	recToggle := httptest.NewRecorder()
	testRouter.ServeHTTP(recToggle, reqToggle)
	assert.Equal(t, http.StatusOK, recToggle.Code)

	var updatedSub model.SubtaskResponse
	_ = json.Unmarshal(recToggle.Body.Bytes(), &updatedSub)
	assert.True(t, updatedSub.Done)

	// Verify task detail includes subtask and tag
	reqGet := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", task.ID), nil)
	reqGet.Header.Set("Authorization", auth)
	recGet := httptest.NewRecorder()
	testRouter.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)

	var detailTask model.TaskResponse
	_ = json.Unmarshal(recGet.Body.Bytes(), &detailTask)
	assert.Len(t, detailTask.Subtasks, 1)
	assert.True(t, detailTask.Subtasks[0].Done)
}

func TestTimeBlocksOverlapAndRejection(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)

	baseTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)

	// 1. Create initial block: 10:00 to 12:00
	b1 := model.CreateTimeBlockRequest{
		Title:    "Block 1 (10:00 - 12:00)",
		StartsAt: baseTime,
		EndsAt:   baseTime.Add(2 * time.Hour),
	}
	body1, _ := json.Marshal(b1)
	req1 := httptest.NewRequest("POST", "/time-blocks", bytes.NewBuffer(body1))
	req1.Header.Set("Authorization", auth)
	rec1 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusCreated, rec1.Code)

	// 2. Adjacent block (12:00 to 13:00) -> Allowed (not overlapping)
	bAdjacent := model.CreateTimeBlockRequest{
		Title:    "Block 2 Adjacent (12:00 - 13:00)",
		StartsAt: baseTime.Add(2 * time.Hour),
		EndsAt:   baseTime.Add(3 * time.Hour),
	}
	bodyAdj, _ := json.Marshal(bAdjacent)
	reqAdj := httptest.NewRequest("POST", "/time-blocks", bytes.NewBuffer(bodyAdj))
	reqAdj.Header.Set("Authorization", auth)
	recAdj := httptest.NewRecorder()
	testRouter.ServeHTTP(recAdj, reqAdj)
	assert.Equal(t, http.StatusCreated, recAdj.Code)

	// 3. Overlapping block (11:00 to 12:30) -> Rejected with 409 Conflict
	bOverlap := model.CreateTimeBlockRequest{
		Title:    "Overlapping Block (11:00 - 12:30)",
		StartsAt: baseTime.Add(1 * time.Hour),
		EndsAt:   baseTime.Add(2*time.Hour + 30*time.Minute),
	}
	bodyOverlap, _ := json.Marshal(bOverlap)
	reqOverlap := httptest.NewRequest("POST", "/time-blocks", bytes.NewBuffer(bodyOverlap))
	reqOverlap.Header.Set("Authorization", auth)
	recOverlap := httptest.NewRecorder()
	testRouter.ServeHTTP(recOverlap, reqOverlap)
	assert.Equal(t, http.StatusConflict, recOverlap.Code)
}

func TestWorkloadAndDashboard(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)

	today := time.Now().UTC()
	startOfToday := time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, time.UTC)
	est := int32(120) // 2 hours

	// Create task due today
	taskReq := model.CreateTaskRequest{
		Title:           "Today Task",
		Priority:        1,
		Status:          "todo",
		DueAt:           &startOfToday,
		EstimateMinutes: &est,
	}
	taskBody, _ := json.Marshal(taskReq)
	reqTask := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(taskBody))
	reqTask.Header.Set("Authorization", auth)
	recTask := httptest.NewRecorder()
	testRouter.ServeHTTP(recTask, reqTask)
	require.Equal(t, http.StatusCreated, recTask.Code)

	// Create time block today: 14:00 to 16:00 (120 mins)
	tbReq := model.CreateTimeBlockRequest{
		Title:    "Today Block",
		StartsAt: time.Date(today.Year(), today.Month(), today.Day(), 14, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(today.Year(), today.Month(), today.Day(), 16, 0, 0, 0, time.UTC),
	}
	tbBody, _ := json.Marshal(tbReq)
	reqTb := httptest.NewRequest("POST", "/time-blocks", bytes.NewBuffer(tbBody))
	reqTb.Header.Set("Authorization", auth)
	recTb := httptest.NewRecorder()
	testRouter.ServeHTTP(recTb, reqTb)
	require.Equal(t, http.StatusCreated, recTb.Code)

	// GET /workload for today
	todayStr := startOfToday.Format("2006-01-02")
	reqWorkload := httptest.NewRequest("GET", fmt.Sprintf("/workload?from=%s&to=%s&capacity=200", todayStr, todayStr), nil)
	reqWorkload.Header.Set("Authorization", auth)
	recWorkload := httptest.NewRecorder()
	testRouter.ServeHTTP(recWorkload, reqWorkload)
	assert.Equal(t, http.StatusOK, recWorkload.Code)

	var workload model.WorkloadResponse
	err := json.Unmarshal(recWorkload.Body.Bytes(), &workload)
	require.NoError(t, err)
	require.Len(t, workload.Days, 1)
	assert.Equal(t, int32(240), workload.Days[0].PlannedMinutes) // 120 (task) + 120 (block)
	assert.Equal(t, int32(200), workload.Days[0].CapacityMinutes)
	assert.True(t, workload.Days[0].OverCapacity) // 240 > 200

	// GET /dashboard
	reqDash := httptest.NewRequest("GET", "/dashboard", nil)
	reqDash.Header.Set("Authorization", auth)
	recDash := httptest.NewRecorder()
	testRouter.ServeHTTP(recDash, reqDash)
	assert.Equal(t, http.StatusOK, recDash.Code)

	var dash model.DashboardResponse
	err = json.Unmarshal(recDash.Body.Bytes(), &dash)
	require.NoError(t, err)
	assert.NotEmpty(t, dash.Today)
	assert.NotEmpty(t, dash.Workload)
}
