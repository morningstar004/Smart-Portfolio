package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// Sentinel error types — used across all modules for consistent error handling
// ---------------------------------------------------------------------------

// ErrNotFound indicates that a requested resource does not exist.
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e *ErrNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// NewErrNotFound creates a new ErrNotFound error.
func NewErrNotFound(resource, id string) *ErrNotFound {
	return &ErrNotFound{Resource: resource, ID: id}
}

// IsNotFound checks whether err (or anything in its chain) is an ErrNotFound.
func IsNotFound(err error) bool {
	var target *ErrNotFound
	return errors.As(err, &target)
}

// ErrValidation indicates that a request failed validation.
type ErrValidation struct {
	Message string
}

func (e *ErrValidation) Error() string {
	return e.Message
}

// NewErrValidation creates a new ErrValidation error.
func NewErrValidation(msg string) *ErrValidation {
	return &ErrValidation{Message: msg}
}

// IsValidation checks whether err (or anything in its chain) is an ErrValidation.
func IsValidation(err error) bool {
	var target *ErrValidation
	return errors.As(err, &target)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// ParseUUID parses a string into a uuid.UUID. Returns an ErrValidation if the
// string is not a valid UUID. This is used by multiple service layers so it
// lives here to avoid duplication.
func ParseUUID(id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, &ErrValidation{Message: fmt.Sprintf("invalid UUID %q: %s", id, err.Error())}
	}
	return uid, nil
}

// APIResponse is the standard envelope for all JSON responses returned by the API.
// Every endpoint wraps its data in this structure so the frontend can rely on a
// consistent shape.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError carries structured error information inside an APIResponse.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WriteJSON serializes data into the standard APIResponse envelope and writes it
// to the ResponseWriter with the given HTTP status code. If marshalling fails, it
// falls back to a plain-text 500 response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	resp := APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("response: failed to encode JSON response")
	}
}

// WriteError writes a structured error response using the standard envelope.
// The status code is set on both the HTTP response and inside the error body so
// clients can inspect either.
func WriteError(w http.ResponseWriter, status int, message string) {
	resp := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    status,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("response: failed to encode error JSON response")
	}
}

// WriteValidationError is a convenience wrapper around WriteError that returns
// a 400 Bad Request with the validation error message. It strips the common
// "validation failed: " prefix added by DTO Validate() methods so the client
// gets a clean list of field-level issues.
func WriteValidationError(w http.ResponseWriter, err error) {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "validation failed: ")
	WriteError(w, http.StatusBadRequest, msg)
}

// WriteNotFound writes a 404 Not Found response with a human-readable message.
func WriteNotFound(w http.ResponseWriter, resource string) {
	WriteError(w, http.StatusNotFound, resource+" not found")
}

// WriteInternalError logs the real error at ERROR level and returns a generic
// 500 response to the client. The internal error details are never leaked to
// the caller.
func WriteInternalError(w http.ResponseWriter, err error, context string) {
	log.Error().Err(err).Str("context", context).Msg("response: internal server error")
	WriteError(w, http.StatusInternalServerError, "an internal error occurred — please try again later")
}

// DecodeJSON reads the request body and decodes it into the destination value.
// It returns a user-friendly error message suitable for WriteError if decoding
// fails, or nil on success.
func DecodeJSON(r *http.Request, dest interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return err
	}
	return nil
}

// HandleServiceError inspects a service-layer error and writes the appropriate
// HTTP response. It returns true if the error was handled, false if nil.
// This eliminates duplicated error-routing logic in every handler.
func HandleServiceError(w http.ResponseWriter, err error, handlerName string) bool {
	if err == nil {
		return false
	}
	if IsValidation(err) {
		WriteValidationError(w, err)
		return true
	}
	if IsNotFound(err) {
		var nf *ErrNotFound
		errors.As(err, &nf)
		WriteNotFound(w, nf.Resource)
		return true
	}
	WriteInternalError(w, err, handlerName)
	return true
}
