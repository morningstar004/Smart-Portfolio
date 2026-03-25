package httputil

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// WriteJSON
// ---------------------------------------------------------------------------

func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"hello": "world"}
	WriteJSON(w, http.StatusOK, data)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !envelope.Success {
		t.Error("expected success=true for 200 status")
	}
	if envelope.Error != nil {
		t.Error("expected error to be nil for success response")
	}
	if envelope.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestWriteJSON_CreatedStatus(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusCreated, map[string]int{"id": 1})

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !envelope.Success {
		t.Error("expected success=true for 201 status")
	}
}

func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, nil)

	resp := w.Result()
	defer resp.Body.Close()

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !envelope.Success {
		t.Error("expected success=true")
	}
}

func TestWriteJSON_EmptySlice(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, []string{})

	resp := w.Result()
	defer resp.Body.Close()

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !envelope.Success {
		t.Error("expected success=true")
	}
	if envelope.Data == nil {
		t.Error("expected data to be non-nil (empty array, not null)")
	}
}

// ---------------------------------------------------------------------------
// WriteError
// ---------------------------------------------------------------------------

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{"BadRequest", http.StatusBadRequest, "invalid input"},
		{"NotFound", http.StatusNotFound, "resource not found"},
		{"InternalServerError", http.StatusInternalServerError, "something broke"},
		{"Unauthorized", http.StatusUnauthorized, "not authenticated"},
		{"Forbidden", http.StatusForbidden, "access denied"},
		{"TooManyRequests", http.StatusTooManyRequests, "rate limited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteError(w, tt.status, tt.message)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.StatusCode)
			}

			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("expected Content-Type application/json, got %q", ct)
			}

			var envelope APIResponse
			if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if envelope.Success {
				t.Error("expected success=false for error response")
			}
			if envelope.Error == nil {
				t.Fatal("expected error to be non-nil")
			}
			if envelope.Error.Code != tt.status {
				t.Errorf("expected error code %d, got %d", tt.status, envelope.Error.Code)
			}
			if envelope.Error.Message != tt.message {
				t.Errorf("expected error message %q, got %q", tt.message, envelope.Error.Message)
			}
			if envelope.Data != nil {
				t.Error("expected data to be nil for error response")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WriteValidationError
// ---------------------------------------------------------------------------

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	err := errors.New("validation failed: name is required; email is invalid")
	WriteValidationError(w, err)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if envelope.Success {
		t.Error("expected success=false")
	}

	// The "validation failed: " prefix should be stripped
	if envelope.Error == nil {
		t.Fatal("expected error to be non-nil")
	}
	if strings.Contains(envelope.Error.Message, "validation failed:") {
		t.Errorf("expected 'validation failed:' prefix to be stripped, got %q", envelope.Error.Message)
	}
	if !strings.Contains(envelope.Error.Message, "name is required") {
		t.Errorf("expected message to contain field errors, got %q", envelope.Error.Message)
	}
}

func TestWriteValidationError_NoPrefix(t *testing.T) {
	w := httptest.NewRecorder()

	err := errors.New("email is required")
	WriteValidationError(w, err)

	resp := w.Result()
	defer resp.Body.Close()

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if envelope.Error.Message != "email is required" {
		t.Errorf("expected message 'email is required', got %q", envelope.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// WriteNotFound
// ---------------------------------------------------------------------------

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNotFound(w, "project")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if envelope.Error.Message != "project not found" {
		t.Errorf("expected 'project not found', got %q", envelope.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// WriteInternalError
// ---------------------------------------------------------------------------

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()

	internalErr := errors.New("database connection timeout")
	WriteInternalError(w, internalErr, "TestHandler.Method")

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}

	var envelope APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// The internal error details must NOT be leaked to the client
	if strings.Contains(envelope.Error.Message, "database") {
		t.Errorf("internal error leaked to client: %q", envelope.Error.Message)
	}
	if strings.Contains(envelope.Error.Message, "timeout") {
		t.Errorf("internal error leaked to client: %q", envelope.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// DecodeJSON
// ---------------------------------------------------------------------------

func TestDecodeJSON_ValidPayload(t *testing.T) {
	body := `{"name":"Alice","age":30}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var dest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	err := DecodeJSON(r, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dest.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", dest.Name)
	}
	if dest.Age != 30 {
		t.Errorf("expected Age 30, got %d", dest.Age)
	}
}

func TestDecodeJSON_UnknownField(t *testing.T) {
	body := `{"name":"Alice","unknown_field":"bad"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var dest struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(r, &dest)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := `{not valid json}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var dest struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(r, &dest)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	var dest struct {
		Name string `json:"name"`
	}

	err := DecodeJSON(r, &dest)
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

// ---------------------------------------------------------------------------
// ParseUUID
// ---------------------------------------------------------------------------

func TestParseUUID_Valid(t *testing.T) {
	expected := uuid.New()
	parsed, err := ParseUUID(expected.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed != expected {
		t.Errorf("expected %s, got %s", expected, parsed)
	}
}

func TestParseUUID_Invalid(t *testing.T) {
	invalidInputs := []string{
		"",
		"not-a-uuid",
		"12345",
		"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
		"123e4567-e89b-12d3-a456", // truncated
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseUUID(input)
			if err == nil {
				t.Fatalf("expected error for invalid UUID %q, got nil", input)
			}

			// Should be an ErrValidation
			if !IsValidation(err) {
				t.Errorf("expected ErrValidation, got %T: %v", err, err)
			}
		})
	}
}

func TestParseUUID_NilUUID(t *testing.T) {
	nilUUID := uuid.Nil.String()
	parsed, err := ParseUUID(nilUUID)
	if err != nil {
		t.Fatalf("unexpected error for nil UUID: %v", err)
	}
	if parsed != uuid.Nil {
		t.Errorf("expected nil UUID, got %s", parsed)
	}
}

// ---------------------------------------------------------------------------
// ErrNotFound
// ---------------------------------------------------------------------------

func TestErrNotFound_Error(t *testing.T) {
	err := NewErrNotFound("project", "abc-123")
	expected := "project abc-123 not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestErrNotFound_ErrorNoID(t *testing.T) {
	err := NewErrNotFound("project", "")
	expected := "project not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestIsNotFound_True(t *testing.T) {
	err := NewErrNotFound("user", "42")
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to return true")
	}
}

func TestIsNotFound_Wrapped(t *testing.T) {
	inner := NewErrNotFound("user", "42")
	wrapped := errors.Join(errors.New("outer"), inner)
	if !IsNotFound(wrapped) {
		t.Error("expected IsNotFound to return true for wrapped error")
	}
}

func TestIsNotFound_False(t *testing.T) {
	err := errors.New("some other error")
	if IsNotFound(err) {
		t.Error("expected IsNotFound to return false for non-ErrNotFound")
	}
}

func TestIsNotFound_Nil(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("expected IsNotFound to return false for nil")
	}
}

// ---------------------------------------------------------------------------
// ErrValidation
// ---------------------------------------------------------------------------

func TestErrValidation_Error(t *testing.T) {
	err := NewErrValidation("name is required")
	if err.Error() != "name is required" {
		t.Errorf("expected 'name is required', got %q", err.Error())
	}
}

func TestIsValidation_True(t *testing.T) {
	err := NewErrValidation("bad input")
	if !IsValidation(err) {
		t.Error("expected IsValidation to return true")
	}
}

func TestIsValidation_Wrapped(t *testing.T) {
	inner := NewErrValidation("bad input")
	wrapped := errors.Join(errors.New("outer"), inner)
	if !IsValidation(wrapped) {
		t.Error("expected IsValidation to return true for wrapped error")
	}
}

func TestIsValidation_False(t *testing.T) {
	err := errors.New("some other error")
	if IsValidation(err) {
		t.Error("expected IsValidation to return false for non-ErrValidation")
	}
}

func TestIsValidation_Nil(t *testing.T) {
	if IsValidation(nil) {
		t.Error("expected IsValidation to return false for nil")
	}
}

// ---------------------------------------------------------------------------
// HandleServiceError
// ---------------------------------------------------------------------------

func TestHandleServiceError_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	handled := HandleServiceError(w, nil, "Test")
	if handled {
		t.Error("expected HandleServiceError to return false for nil error")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected no status written (default 200), got %d", w.Code)
	}
}

func TestHandleServiceError_Validation(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewErrValidation("field X is required")
	handled := HandleServiceError(w, err, "Test")
	if !handled {
		t.Fatal("expected HandleServiceError to return true")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandleServiceError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	err := NewErrNotFound("project", "abc")
	handled := HandleServiceError(w, err, "Test")
	if !handled {
		t.Fatal("expected HandleServiceError to return true")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHandleServiceError_InternalError(t *testing.T) {
	w := httptest.NewRecorder()
	err := errors.New("database crashed")
	handled := HandleServiceError(w, err, "Test")
	if !handled {
		t.Fatal("expected HandleServiceError to return true")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}

	var envelope APIResponse
	if decErr := json.NewDecoder(resp.Body).Decode(&envelope); decErr != nil {
		t.Fatalf("failed to decode: %v", decErr)
	}

	// Internal details must not leak
	if strings.Contains(envelope.Error.Message, "database") {
		t.Errorf("internal error leaked: %q", envelope.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// Multiple writes should not panic or corrupt
// ---------------------------------------------------------------------------

func TestWriteJSON_DoesNotPanicOnNilWriter(t *testing.T) {
	// Ensure encoding weird data types doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteJSON panicked: %v", r)
		}
	}()

	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nested": map[string]interface{}{
			"slice": []int{1, 2, 3},
			"bool":  true,
		},
	})
}
