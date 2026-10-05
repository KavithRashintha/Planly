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
	"github.com/planly/services/auth/internal/handler"
	"github.com/planly/services/auth/internal/model"
	"github.com/planly/services/auth/internal/repository"
	"github.com/planly/services/auth/internal/service"
	"github.com/planly/services/auth/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPool    *pgxpool.Pool
	testRouter  chi.Router
	testSecret  = []byte("test-secret-at-least-32-bytes-long!")
	internalKey = "test-internal-key-12345"
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
		Database: "auth_db",
		SSLMode:  "disable",
	}

	pool, err := dbx.Connect(ctx, cfg)
	if err != nil {
		fmt.Printf("Skipping integration tests: DB not available: %v\n", err)
		os.Exit(0)
	}
	testPool = pool

	if err := migrations.Run(ctx, testPool); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	repo := repository.New(testPool)
	authSvc := service.NewAuthService(repo, testSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	r := chi.NewRouter()
	authHandler.RegisterRoutes(r, testSecret, internalKey)
	testRouter = r

	code := m.Run()
	testPool.Close()
	os.Exit(code)
}

func TestRegister(t *testing.T) {
	uniqueEmail := fmt.Sprintf("test-%s@example.com", uuid.New().String())

	// 1. Success
	reqBody := model.RegisterRequest{
		Email:     uniqueEmail,
		Password:  "password123",
		FullName:  "Test User",
		Persona:   "employee",
		Timezone:  "America/New_York",
		WorkStart: "08:30",
		WorkEnd:   "16:30",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var user model.UserResponse
	err := json.Unmarshal(rec.Body.Bytes(), &user)
	require.NoError(t, err)
	assert.Equal(t, uniqueEmail, user.Email)
	assert.Equal(t, "employee", user.Persona)
	assert.Equal(t, "America/New_York", user.Timezone)
	assert.Equal(t, "08:30", user.WorkStart)
	assert.Equal(t, "16:30", user.WorkEnd)

	// 2. Duplicate email -> 409
	recDup := httptest.NewRecorder()
	reqDup := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
	testRouter.ServeHTTP(recDup, reqDup)
	assert.Equal(t, http.StatusConflict, recDup.Code)

	// 3. Password < 8 characters -> 400
	weakBody, _ := json.Marshal(model.RegisterRequest{
		Email:    fmt.Sprintf("weak-%s@example.com", uuid.New().String()),
		Password: "short",
		FullName: "Short Pass",
	})
	recWeak := httptest.NewRecorder()
	reqWeak := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(weakBody))
	testRouter.ServeHTTP(recWeak, reqWeak)
	assert.Equal(t, http.StatusBadRequest, recWeak.Code)

	// 4. Invalid email format -> 400
	invalidEmailBody, _ := json.Marshal(model.RegisterRequest{
		Email:    "not-an-email",
		Password: "password123",
		FullName: "Invalid Email",
	})
	recInvalidEmail := httptest.NewRecorder()
	reqInvalidEmail := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(invalidEmailBody))
	testRouter.ServeHTTP(recInvalidEmail, reqInvalidEmail)
	assert.Equal(t, http.StatusBadRequest, recInvalidEmail.Code)
}

func TestLoginAndRefresh(t *testing.T) {
	email := fmt.Sprintf("login-%s@example.com", uuid.New().String())
	password := "correct-password"

	// Register user first
	regBody, _ := json.Marshal(model.RegisterRequest{
		Email:    email,
		Password: password,
		FullName: "Login User",
	})
	recReg := httptest.NewRecorder()
	testRouter.ServeHTTP(recReg, httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody)))
	require.Equal(t, http.StatusCreated, recReg.Code)

	// 1. Valid Login
	loginBody, _ := json.Marshal(model.LoginRequest{
		Email:    email,
		Password: password,
	})
	recLogin := httptest.NewRecorder()
	testRouter.ServeHTTP(recLogin, httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody)))
	assert.Equal(t, http.StatusOK, recLogin.Code)

	var authResp model.AuthResponse
	err := json.Unmarshal(recLogin.Body.Bytes(), &authResp)
	require.NoError(t, err)
	assert.NotEmpty(t, authResp.AccessToken)
	assert.NotEmpty(t, authResp.RefreshToken)
	assert.Equal(t, int64(900), authResp.ExpiresIn)
	assert.Equal(t, email, authResp.User.Email)

	// 2. Wrong Password -> 401
	wrongLoginBody, _ := json.Marshal(model.LoginRequest{
		Email:    email,
		Password: "wrong-password",
	})
	recWrong := httptest.NewRecorder()
	testRouter.ServeHTTP(recWrong, httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(wrongLoginBody)))
	assert.Equal(t, http.StatusUnauthorized, recWrong.Code)

	// 3. Non-existent email -> 401
	nonExistentBody, _ := json.Marshal(model.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})
	recNonExistent := httptest.NewRecorder()
	testRouter.ServeHTTP(recNonExistent, httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(nonExistentBody)))
	assert.Equal(t, http.StatusUnauthorized, recNonExistent.Code)

	// 4. Successful Refresh
	refreshBody, _ := json.Marshal(model.RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	})
	recRefresh := httptest.NewRecorder()
	testRouter.ServeHTTP(recRefresh, httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(refreshBody)))
	assert.Equal(t, http.StatusOK, recRefresh.Code)

	var refreshedResp model.AuthResponse
	err = json.Unmarshal(recRefresh.Body.Bytes(), &refreshedResp)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshedResp.AccessToken)
	assert.NotEmpty(t, refreshedResp.RefreshToken)
	assert.NotEqual(t, authResp.RefreshToken, refreshedResp.RefreshToken) // Rotated

	// 5. Reusing old refresh token (revoked) -> 401
	recReused := httptest.NewRecorder()
	testRouter.ServeHTTP(recReused, httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(refreshBody)))
	assert.Equal(t, http.StatusUnauthorized, recReused.Code)

	// 6. Invalid refresh token -> 401
	badRefreshBody, _ := json.Marshal(model.RefreshRequest{
		RefreshToken: "completely-bogus-token",
	})
	recBadRefresh := httptest.NewRecorder()
	testRouter.ServeHTTP(recBadRefresh, httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(badRefreshBody)))
	assert.Equal(t, http.StatusUnauthorized, recBadRefresh.Code)
}

func TestProfileFlow(t *testing.T) {
	email := fmt.Sprintf("profile-%s@example.com", uuid.New().String())
	password := "password123"

	// Register & Login
	regBody, _ := json.Marshal(model.RegisterRequest{
		Email:    email,
		Password: password,
		FullName: "Original Name",
	})
	recReg := httptest.NewRecorder()
	testRouter.ServeHTTP(recReg, httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(regBody)))
	require.Equal(t, http.StatusCreated, recReg.Code)

	loginBody, _ := json.Marshal(model.LoginRequest{Email: email, Password: password})
	recLogin := httptest.NewRecorder()
	testRouter.ServeHTTP(recLogin, httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody)))
	require.Equal(t, http.StatusOK, recLogin.Code)

	var authResp model.AuthResponse
	_ = json.Unmarshal(recLogin.Body.Bytes(), &authResp)

	// 1. GET /auth/me without token -> 401
	recMeNoToken := httptest.NewRecorder()
	testRouter.ServeHTTP(recMeNoToken, httptest.NewRequest("GET", "/auth/me", nil))
	assert.Equal(t, http.StatusUnauthorized, recMeNoToken.Code)

	// 2. GET /auth/me with token -> 200
	reqMe := httptest.NewRequest("GET", "/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authResp.AccessToken)
	recMe := httptest.NewRecorder()
	testRouter.ServeHTTP(recMe, reqMe)
	assert.Equal(t, http.StatusOK, recMe.Code)

	var meResp model.UserResponse
	err := json.Unmarshal(recMe.Body.Bytes(), &meResp)
	require.NoError(t, err)
	assert.Equal(t, "Original Name", meResp.FullName)

	// 3. PUT /auth/me -> 200
	updateBody, _ := json.Marshal(model.UpdateProfileRequest{
		FullName:  "Updated Name",
		Persona:   "undergraduate",
		Timezone:  "Asia/Colombo",
		WorkStart: "10:00",
		WorkEnd:   "18:00",
	})
	reqUpdate := httptest.NewRequest("PUT", "/auth/me", bytes.NewBuffer(updateBody))
	reqUpdate.Header.Set("Authorization", "Bearer "+authResp.AccessToken)
	recUpdate := httptest.NewRecorder()
	testRouter.ServeHTTP(recUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, recUpdate.Code)

	var updatedUser model.UserResponse
	err = json.Unmarshal(recUpdate.Body.Bytes(), &updatedUser)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedUser.FullName)
	assert.Equal(t, "undergraduate", updatedUser.Persona)
	assert.Equal(t, "Asia/Colombo", updatedUser.Timezone)
	assert.Equal(t, "10:00", updatedUser.WorkStart)
	assert.Equal(t, "18:00", updatedUser.WorkEnd)

	// 4. PUT /auth/me with invalid timezone -> 400
	badTzBody, _ := json.Marshal(model.UpdateProfileRequest{
		FullName:  "Bad Tz",
		Persona:   "student",
		Timezone:  "Invalid/Timezone",
		WorkStart: "09:00",
		WorkEnd:   "17:00",
	})
	reqBadTz := httptest.NewRequest("PUT", "/auth/me", bytes.NewBuffer(badTzBody))
	reqBadTz.Header.Set("Authorization", "Bearer "+authResp.AccessToken)
	recBadTz := httptest.NewRecorder()
	testRouter.ServeHTTP(recBadTz, reqBadTz)
	assert.Equal(t, http.StatusBadRequest, recBadTz.Code)

	// 5. GET /internal/users/briefing-candidates -> 200 with internal key
	reqCandidates := httptest.NewRequest("GET", "/internal/users/briefing-candidates", nil)
	reqCandidates.Header.Set("X-Internal-Key", internalKey)
	recCandidates := httptest.NewRecorder()
	testRouter.ServeHTTP(recCandidates, reqCandidates)
	assert.Equal(t, http.StatusOK, recCandidates.Code)
}
