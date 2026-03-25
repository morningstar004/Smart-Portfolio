package dto

import (
	"fmt"
	"strings"
)

// ChatRequest is the payload for asking a question to the AI assistant.
type ChatRequest struct {
	Question string `json:"question"`
}

// Validate checks that the question field is present and non-empty.
func (r ChatRequest) Validate() error {
	if strings.TrimSpace(r.Question) == "" {
		return fmt.Errorf("validation failed: question cannot be empty")
	}
	return nil
}

// ChatResponse is the full (non-streaming) response from the AI assistant.
type ChatResponse struct {
	Answer string `json:"answer"`
	Cached bool   `json:"cached"`
}

// IngestResponse is returned after a successful PDF ingestion.
type IngestResponse struct {
	Message string `json:"message"`
	Pages   int    `json:"pages"`
	Chunks  int    `json:"chunks"`
}
