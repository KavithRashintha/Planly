package jwtx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerifyToken(t *testing.T) {
	secret := []byte("test-super-secret-key-at-least-32-bytes")
	userID := uuid.New()

	// Valid token test
	token, err := SignToken(userID, secret, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := VerifyToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, userID.String(), claims.Subject)

	// Expired token test
	expiredToken, err := SignToken(userID, secret, -1*time.Minute)
	require.NoError(t, err)
	_, err = VerifyToken(expiredToken, secret)
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Invalid signature test
	differentSecret := []byte("different-secret-key-at-least-32-bytes")
	_, err = VerifyToken(token, differentSecret)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthMiddleware(t *testing.T) {
	secret := []byte("test-super-secret-key-at-least-32-bytes")
	userID := uuid.New()
	token, err := SignToken(userID, secret, 15*time.Minute)
	require.NoError(t, err)

	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := GetUserID(r.Context())
		if !ok || uid != userID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Success with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Case 2: Missing authorization header
	reqMissing := httptest.NewRequest("GET", "/test", nil)
	recMissing := httptest.NewRecorder()
	handler.ServeHTTP(recMissing, reqMissing)
	assert.Equal(t, http.StatusUnauthorized, recMissing.Code)

	// Case 3: Invalid token format
	reqInvalid := httptest.NewRequest("GET", "/test", nil)
	reqInvalid.Header.Set("Authorization", "Basic 12345")
	recInvalid := httptest.NewRecorder()
	handler.ServeHTTP(recInvalid, reqInvalid)
	assert.Equal(t, http.StatusUnauthorized, recInvalid.Code)

	// Case 4: Invalid/corrupt token
	reqCorrupt := httptest.NewRequest("GET", "/test", nil)
	reqCorrupt.Header.Set("Authorization", "Bearer invalid.token.value")
	recCorrupt := httptest.NewRecorder()
	handler.ServeHTTP(recCorrupt, reqCorrupt)
	assert.Equal(t, http.StatusUnauthorized, recCorrupt.Code)
}

func TestRequireInternalKey(t *testing.T) {
	expectedKey := "internal-secret-key"
	handler := RequireInternalKey(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Success
	req := httptest.NewRequest("GET", "/internal", nil)
	req.Header.Set("X-Internal-Key", expectedKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Wrong key
	reqWrong := httptest.NewRequest("GET", "/internal", nil)
	reqWrong.Header.Set("X-Internal-Key", "wrong-key")
	recWrong := httptest.NewRecorder()
	handler.ServeHTTP(recWrong, reqWrong)
	assert.Equal(t, http.StatusForbidden, recWrong.Code)
}
