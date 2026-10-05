package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrorDetail represents the error body payload.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents the standard API error response: {"error":{"code":"...","message":"..."}}
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// WriteJSON serializes the given data to JSON and sets the Content-Type header.
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return nil
	}
	return json.NewEncoder(w).Encode(data)
}

// ReadJSON decodes the request body into the target interface.
func ReadJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
	}

	// Ensure there is only one JSON object in the stream
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must only contain a single JSON object")
	}

	return nil
}

// WriteError writes a standard error JSON response.
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	_ = WriteJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
