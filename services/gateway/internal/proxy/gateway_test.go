package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/planly/pkg/jwtx"
	"github.com/planly/services/gateway/internal/middleware"
	"github.com/planly/services/gateway/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestGatewayRoutingAndAuth(t *testing.T) {
	jwtSecret := []byte("gateway-test-super-secret-key-32b")
	userID := uuid.New()
	validToken, err := jwtx.SignToken(userID, jwtSecret, 15*time.Minute)
	require.NoError(t, err)

	// Mock Upstream: Auth Service
	var capturedAuthPath, capturedAuthHeader, capturedAuthUserID string
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthPath = r.URL.Path
		capturedAuthHeader = r.Header.Get("Authorization")
		capturedAuthUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"auth"}`))
	}))
	defer authServer.Close()

	// Mock Upstream: Planner Service
	var capturedPlannerPath, capturedPlannerQuery, capturedPlannerUserID string
	plannerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPlannerPath = r.URL.Path
		capturedPlannerQuery = r.URL.RawQuery
		capturedPlannerUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"planner"}`))
	}))
	defer plannerServer.Close()

	authProxy, err := proxy.NewReverseProxy(authServer.URL)
	require.NoError(t, err)

	plannerProxy, err := proxy.NewReverseProxy(plannerServer.URL)
	require.NoError(t, err)

	// Setup Gateway router
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// Public routes
	r.Post("/api/v1/auth/register", authProxy.ServeHTTP)
	r.Post("/api/v1/auth/login", authProxy.ServeHTTP)

	// Protected routes
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.JWTAuthMiddleware(jwtSecret))
		protected.Handle("/api/v1/auth/*", authProxy)
		protected.Handle("/api/v1/projects*", plannerProxy)
		protected.Handle("/api/v1/tasks*", plannerProxy)
	})

	// 1. Public route test (No JWT required, /api/v1 stripped)
	reqPublic := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
	recPublic := httptest.NewRecorder()
	r.ServeHTTP(recPublic, reqPublic)
	assert.Equal(t, http.StatusOK, recPublic.Code)
	assert.Equal(t, "/auth/register", capturedAuthPath) // Stripped /api/v1
	assert.Empty(t, capturedAuthUserID)

	// 2. Protected route without JWT -> 401 Unauthorized
	reqNoAuth := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	recNoAuth := httptest.NewRecorder()
	r.ServeHTTP(recNoAuth, reqNoAuth)
	assert.Equal(t, http.StatusUnauthorized, recNoAuth.Code)

	// 3. Protected route with invalid JWT -> 401 Unauthorized
	reqBadAuth := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	reqBadAuth.Header.Set("Authorization", "Bearer invalid.token.value")
	recBadAuth := httptest.NewRecorder()
	r.ServeHTTP(recBadAuth, reqBadAuth)
	assert.Equal(t, http.StatusUnauthorized, recBadAuth.Code)

	// 4. Protected route with valid JWT -> 200, stripped path, query preserved, X-User-ID forwarded
	reqValid := httptest.NewRequest("GET", "/api/v1/tasks?status=todo&limit=10", nil)
	reqValid.Header.Set("Authorization", "Bearer "+validToken)
	recValid := httptest.NewRecorder()
	r.ServeHTTP(recValid, reqValid)
	assert.Equal(t, http.StatusOK, recValid.Code)
	assert.Equal(t, "/tasks", capturedPlannerPath)
	assert.Equal(t, "status=todo&limit=10", capturedPlannerQuery)
	assert.Equal(t, userID.String(), capturedPlannerUserID)

	// 5. Protected auth route (e.g. /auth/me) forwards to auth upstream
	reqMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+validToken)
	recMe := httptest.NewRecorder()
	r.ServeHTTP(recMe, reqMe)
	assert.Equal(t, http.StatusOK, recMe.Code)
	assert.Equal(t, "/auth/me", capturedAuthPath)
	assert.Equal(t, userID.String(), capturedAuthUserID)
	assert.Equal(t, "Bearer "+validToken, capturedAuthHeader)

	// 6. CORS Preflight OPTIONS test
	reqCORS := httptest.NewRequest("OPTIONS", "/api/v1/tasks", nil)
	reqCORS.Header.Set("Origin", "http://localhost:5173")
	reqCORS.Header.Set("Access-Control-Request-Method", "GET")
	recCORS := httptest.NewRecorder()
	r.ServeHTTP(recCORS, reqCORS)
	assert.Equal(t, http.StatusOK, recCORS.Code)
	assert.Equal(t, "http://localhost:5173", recCORS.Header().Get("Access-Control-Allow-Origin"))
}

func TestRateLimiter(t *testing.T) {
	// Limiter: 1 req/sec, burst 2
	limiter := middleware.NewIPRateLimiter(rate.Limit(1), 2)
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.100:12345"

	// Req 1: OK (burst 1 used)
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = ip
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Req 2: OK (burst 2 used)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = ip
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Req 3: Exceeded burst -> 429 Too Many Requests
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = ip
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusTooManyRequests, rec3.Code)

	// Different IP should still be allowed
	reqOtherIP := httptest.NewRequest("GET", "/test", nil)
	reqOtherIP.RemoteAddr = "10.0.0.1:12345"
	recOtherIP := httptest.NewRecorder()
	handler.ServeHTTP(recOtherIP, reqOtherIP)
	assert.Equal(t, http.StatusOK, recOtherIP.Code)
}
