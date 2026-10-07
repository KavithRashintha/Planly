package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/planly/services/planner/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlannerCrossUserIsolation(t *testing.T) {
	userA := uuid.New()
	authA := authHeader(t, userA)
	userB := uuid.New()
	authB := authHeader(t, userB)

	// 1. User A creates a task
	taskReq := model.CreateTaskRequest{
		Title:    "User A Private Task",
		Priority: 1,
		Status:   "todo",
	}
	body, _ := json.Marshal(taskReq)
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Authorization", authA)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var taskA model.TaskResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &taskA))

	// 2. User A creates a subtask
	subReq := model.CreateSubtaskRequest{
		Title:    "User A Secret Subtask",
		Position: 1,
	}
	subBody, _ := json.Marshal(subReq)
	reqSub := httptest.NewRequest("POST", fmt.Sprintf("/tasks/%s/subtasks", taskA.ID), bytes.NewBuffer(subBody))
	reqSub.Header.Set("Authorization", authA)
	recSub := httptest.NewRecorder()
	testRouter.ServeHTTP(recSub, reqSub)
	require.Equal(t, http.StatusCreated, recSub.Code)

	var subA model.SubtaskResponse
	require.NoError(t, json.Unmarshal(recSub.Body.Bytes(), &subA))

	// 3. User A creates a time block
	now := time.Now().UTC().Truncate(time.Hour)
	tbReq := model.CreateTimeBlockRequest{
		Title:    "User A Time Block",
		StartsAt: now.Add(2 * time.Hour),
		EndsAt:   now.Add(3 * time.Hour),
	}
	tbBody, _ := json.Marshal(tbReq)
	reqTB := httptest.NewRequest("POST", "/time-blocks", bytes.NewBuffer(tbBody))
	reqTB.Header.Set("Authorization", authA)
	recTB := httptest.NewRecorder()
	testRouter.ServeHTTP(recTB, reqTB)
	require.Equal(t, http.StatusCreated, recTB.Code)

	var tbA model.TimeBlockResponse
	require.NoError(t, json.Unmarshal(recTB.Body.Bytes(), &tbA))

	t.Run("User B cannot get User A's task", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", taskA.ID), nil)
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot update User A's task", func(t *testing.T) {
		upReq := model.UpdateTaskRequest{
			Title:    "Malicious Update",
			Priority: 4,
			Status:   "done",
		}
		upBody, _ := json.Marshal(upReq)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/tasks/%s", taskA.ID), bytes.NewBuffer(upBody))
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot add subtask to User A's task", func(t *testing.T) {
		subReq := model.CreateSubtaskRequest{
			Title: "Malicious Subtask",
		}
		subBody, _ := json.Marshal(subReq)
		req := httptest.NewRequest("POST", fmt.Sprintf("/tasks/%s/subtasks", taskA.ID), bytes.NewBuffer(subBody))
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot update User A's subtask", func(t *testing.T) {
		subUp := model.UpdateSubtaskRequest{
			Title: "Hacked Subtask",
			Done:  true,
		}
		upBody, _ := json.Marshal(subUp)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/subtasks/%s", subA.ID), bytes.NewBuffer(upBody))
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot delete User A's subtask (subtask remains)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/subtasks/%s", subA.ID), nil)
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)

		// Verify User A's task still has the subtask
		reqCheck := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", taskA.ID), nil)
		reqCheck.Header.Set("Authorization", authA)
		recCheck := httptest.NewRecorder()
		testRouter.ServeHTTP(recCheck, reqCheck)
		require.Equal(t, http.StatusOK, recCheck.Code)

		var fetchedTask model.TaskResponse
		require.NoError(t, json.Unmarshal(recCheck.Body.Bytes(), &fetchedTask))
		assert.Len(t, fetchedTask.Subtasks, 1)
	})

	t.Run("User B cannot get User A's time-block", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/time-blocks/%s", tbA.ID), nil)
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot update User A's time-block", func(t *testing.T) {
		upTB := model.UpdateTimeBlockRequest{
			Title:    "Hacked Block",
			StartsAt: now.Add(4 * time.Hour),
			EndsAt:   now.Add(5 * time.Hour),
		}
		tbUpBody, _ := json.Marshal(upTB)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/time-blocks/%s", tbA.ID), bytes.NewBuffer(tbUpBody))
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("User B cannot delete User A's time-block (block remains)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/time-blocks/%s", tbA.ID), nil)
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)

		// Verify User A still has the time block
		reqCheck := httptest.NewRequest("GET", fmt.Sprintf("/time-blocks/%s", tbA.ID), nil)
		reqCheck.Header.Set("Authorization", authA)
		recCheck := httptest.NewRecorder()
		testRouter.ServeHTTP(recCheck, reqCheck)
		assert.Equal(t, http.StatusOK, recCheck.Code)
	})

	t.Run("User B cannot delete User A's task (task remains)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/tasks/%s", taskA.ID), nil)
		req.Header.Set("Authorization", authB)
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)

		// Verify User A can still retrieve the task
		reqCheck := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", taskA.ID), nil)
		reqCheck.Header.Set("Authorization", authA)
		recCheck := httptest.NewRecorder()
		testRouter.ServeHTTP(recCheck, reqCheck)
		assert.Equal(t, http.StatusOK, recCheck.Code)
	})
}
