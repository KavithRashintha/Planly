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
	"github.com/planly/services/notify/internal/handler"
	"github.com/planly/services/notify/internal/model"
	"github.com/planly/services/notify/internal/repository"
	"github.com/planly/services/notify/internal/service"
	"github.com/planly/services/notify/internal/worker"
	"github.com/planly/services/notify/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPool    *pgxpool.Pool
	testRouter  chi.Router
	testSvc     service.NotifyService
	testSecret  = []byte("notify-test-secret-at-least-32-bytes!")
	internalKey = "test-internal-notify-key"
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
		Database: "notify_db",
		SSLMode:  "disable",
		MaxConns: 5,
	}

	pool, err := dbx.Connect(ctx, cfg)
	if err != nil {
		fmt.Printf("Skipping notify tests: DB not available: %v\n", err)
		os.Exit(0)
	}
	testPool = pool

	if err := migrations.Run(ctx, testPool); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	repo := repository.New(testPool)
	testSvc = service.NewNotifyService(testPool, repo)
	notifyHandler := handler.NewNotifyHandler(testSvc)

	r := chi.NewRouter()
	notifyHandler.RegisterRoutes(r, testSecret, internalKey)
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

func TestRemindersAndValidation(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)

	// 1. Success: remind_at in future
	futureTime := time.Now().UTC().Add(1 * time.Hour)
	reqBody := model.CreateReminderRequest{
		Message:  "Prepare meeting presentation",
		RemindAt: futureTime,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/reminders", bytes.NewBuffer(body))
	req.Header.Set("Authorization", auth)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var reminder model.ReminderResponse
	err := json.Unmarshal(rec.Body.Bytes(), &reminder)
	require.NoError(t, err)
	assert.Equal(t, "Prepare meeting presentation", reminder.Message)
	assert.Equal(t, user, reminder.UserID)
	assert.False(t, reminder.Fired)

	// 2. Failure: remind_at in the past -> 400
	pastTime := time.Now().UTC().Add(-10 * time.Minute)
	badReqBody := model.CreateReminderRequest{
		Message:  "Past reminder",
		RemindAt: pastTime,
	}
	badBody, _ := json.Marshal(badReqBody)
	reqBad := httptest.NewRequest("POST", "/reminders", bytes.NewBuffer(badBody))
	reqBad.Header.Set("Authorization", auth)
	recBad := httptest.NewRecorder()
	testRouter.ServeHTTP(recBad, reqBad)
	assert.Equal(t, http.StatusBadRequest, recBad.Code)
}

func TestNotificationsInboxAndIsolation(t *testing.T) {
	userA := uuid.New()
	authA := authHeader(t, userA)
	userB := uuid.New()
	authB := authHeader(t, userB)

	// 1. Create a notification for User A via internal endpoint
	internalReq := model.CreateNotificationRequest{
		UserID: &userA,
		Kind:   "briefing",
		Title:  "Morning Briefing",
		Body:   "You have 3 tasks scheduled today",
	}
	intBody, _ := json.Marshal(internalReq)
	reqInt := httptest.NewRequest("POST", "/internal/notifications", bytes.NewBuffer(intBody))
	reqInt.Header.Set("X-Internal-Key", internalKey)
	recInt := httptest.NewRecorder()
	testRouter.ServeHTTP(recInt, reqInt)
	assert.Equal(t, http.StatusCreated, recInt.Code)

	var notifA model.NotificationResponse
	err := json.Unmarshal(recInt.Body.Bytes(), &notifA)
	require.NoError(t, err)
	assert.Equal(t, userA, notifA.UserID)
	assert.False(t, notifA.Read)

	// 2. User A gets notifications and unread count = 1
	reqList := httptest.NewRequest("GET", "/notifications", nil)
	reqList.Header.Set("Authorization", authA)
	recList := httptest.NewRecorder()
	testRouter.ServeHTTP(recList, reqList)
	assert.Equal(t, http.StatusOK, recList.Code)

	var listA []model.NotificationResponse
	_ = json.Unmarshal(recList.Body.Bytes(), &listA)
	assert.NotEmpty(t, listA)
	assert.Equal(t, "Morning Briefing", listA[0].Title)

	reqCount := httptest.NewRequest("GET", "/notifications/unread-count", nil)
	reqCount.Header.Set("Authorization", authA)
	recCount := httptest.NewRecorder()
	testRouter.ServeHTTP(recCount, reqCount)
	assert.Equal(t, http.StatusOK, recCount.Code)

	var countA model.UnreadCountResponse
	_ = json.Unmarshal(recCount.Body.Bytes(), &countA)
	assert.Equal(t, int64(1), countA.UnreadCount)

	// 3. User B cannot read User A's notification (Isolation test)
	reqMarkB := httptest.NewRequest("POST", fmt.Sprintf("/notifications/%s/read", notifA.ID), nil)
	reqMarkB.Header.Set("Authorization", authB)
	recMarkB := httptest.NewRecorder()
	testRouter.ServeHTTP(recMarkB, reqMarkB)
	assert.Equal(t, http.StatusNotFound, recMarkB.Code)

	// 4. User A marks notification as read
	reqMarkA := httptest.NewRequest("POST", fmt.Sprintf("/notifications/%s/read", notifA.ID), nil)
	reqMarkA.Header.Set("Authorization", authA)
	recMarkA := httptest.NewRecorder()
	testRouter.ServeHTTP(recMarkA, reqMarkA)
	assert.Equal(t, http.StatusOK, recMarkA.Code)

	var readNotif model.NotificationResponse
	_ = json.Unmarshal(recMarkA.Body.Bytes(), &readNotif)
	assert.True(t, readNotif.Read)

	// 5. User A unread count is now 0
	recCountAfter := httptest.NewRecorder()
	testRouter.ServeHTTP(recCountAfter, reqCount)
	var countAfter model.UnreadCountResponse
	_ = json.Unmarshal(recCountAfter.Body.Bytes(), &countAfter)
	assert.Equal(t, int64(0), countAfter.UnreadCount)
}

func TestWorkerDueReminderProcessing(t *testing.T) {
	user := uuid.New()
	auth := authHeader(t, user)

	// Create reminder 1s ahead
	remindAt := time.Now().UTC().Add(1 * time.Second)
	reqBody := model.CreateReminderRequest{
		Message:  "Reminder from 1s ahead",
		RemindAt: remindAt,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/reminders", bytes.NewBuffer(body))
	req.Header.Set("Authorization", auth)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Start worker ticking quickly (100ms) for the test
	w := worker.NewWorker(testSvc, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait up to 3 seconds for reminder to be processed
	var found bool
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		reqList := httptest.NewRequest("GET", "/notifications", nil)
		reqList.Header.Set("Authorization", auth)
		recList := httptest.NewRecorder()
		testRouter.ServeHTTP(recList, reqList)

		var notifs []model.NotificationResponse
		_ = json.Unmarshal(recList.Body.Bytes(), &notifs)
		for _, n := range notifs {
			if n.Body == "Reminder from 1s ahead" {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	w.Stop()
	assert.True(t, found, "Notification from due reminder should appear in notifications table")

	// Ensure no duplicate notifications are created
	reqListFinal := httptest.NewRequest("GET", "/notifications", nil)
	reqListFinal.Header.Set("Authorization", auth)
	recListFinal := httptest.NewRecorder()
	testRouter.ServeHTTP(recListFinal, reqListFinal)

	var finalNotifs []model.NotificationResponse
	_ = json.Unmarshal(recListFinal.Body.Bytes(), &finalNotifs)
	matchCount := 0
	for _, n := range finalNotifs {
		if n.Body == "Reminder from 1s ahead" {
			matchCount++
		}
	}
	assert.Equal(t, 1, matchCount, "Notification should appear exactly once")
}
