package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type samplePayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := samplePayload{Name: "Alice", Age: 30}

	err := WriteJSON(rec, http.StatusCreated, data)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var decoded samplePayload
	err = json.Unmarshal(rec.Body.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestReadJSON(t *testing.T) {
	body := `{"name":"Bob","age":25}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))

	var payload samplePayload
	err := ReadJSON(req, &payload)
	require.NoError(t, err)
	assert.Equal(t, "Bob", payload.Name)
	assert.Equal(t, 25, payload.Age)

	// Extra fields should fail
	badBody := `{"name":"Bob","age":25,"extra":"field"}`
	badReq := httptest.NewRequest("POST", "/test", bytes.NewBufferString(badBody))
	var badPayload samplePayload
	err = ReadJSON(badReq, &badPayload)
	assert.Error(t, err)
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "validation_error", "Title is required")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "validation_error", resp.Error.Code)
	assert.Equal(t, "Title is required", resp.Error.Message)
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		assert.NotEmpty(t, reqID)
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Auto-generates request ID
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))

	// Case 2: Preserves incoming request ID
	customID := "custom-req-id-1234"
	reqWithID := httptest.NewRequest("GET", "/test", nil)
	reqWithID.Header.Set("X-Request-ID", customID)
	recWithID := httptest.NewRecorder()
	handler.ServeHTTP(recWithID, reqWithID)
	assert.Equal(t, customID, recWithID.Header().Get("X-Request-ID"))
}

func TestRecoverMiddleware(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "internal_error", resp.Error.Code)
}
