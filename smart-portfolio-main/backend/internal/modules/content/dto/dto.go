package dto

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Project DTOs
// =============================================================================

// ProjectRequest is the payload for creating a new project.
type ProjectRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	TechStack   *string `json:"tech_stack,omitempty"`
	GithubURL   *string `json:"github_url,omitempty"`
	LiveURL     *string `json:"live_url,omitempty"`
}

// Validate checks that all required fields are present and valid.
func (r ProjectRequest) Validate() error {
	var errs []string

	if strings.TrimSpace(r.Title) == "" {
		errs = append(errs, "title is required")
	}
	if strings.TrimSpace(r.Description) == "" {
		errs = append(errs, "description is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ProjectResponse is the response payload returned when reading a project.
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TechStack   *string   `json:"tech_stack,omitempty"`
	GithubURL   *string   `json:"github_url,omitempty"`
	LiveURL     *string   `json:"live_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type WorkItemResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TechStack   *string    `json:"tech_stack,omitempty"`
	GithubURL   *string    `json:"github_url,omitempty"`
	LiveURL     *string    `json:"live_url,omitempty"`
	Source      string     `json:"source"`
	Stars       int        `json:"stars,omitempty"`
	IsPinned    bool       `json:"is_pinned,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type GitHubProfileResponse struct {
	Username        string  `json:"username"`
	DisplayName     *string `json:"display_name,omitempty"`
	ProfileURL      string  `json:"profile_url"`
	RepositoriesURL string  `json:"repositories_url"`
	AvatarURL       *string `json:"avatar_url,omitempty"`
}

type WorkHighlightsResponse struct {
	Items  []WorkItemResponse     `json:"items"`
	GitHub *GitHubProfileResponse `json:"github,omitempty"`
}

// =============================================================================
// Contact Message DTOs
// =============================================================================

// ContactMessageRequest is the payload for submitting a contact message.
type ContactMessageRequest struct {
	SenderName  string `json:"sender_name"`
	SenderEmail string `json:"sender_email"`
	MessageBody string `json:"message_body"`
}

// Validate checks that all required fields are present and the email is valid.
func (r ContactMessageRequest) Validate() error {
	var errs []string

	if strings.TrimSpace(r.SenderName) == "" {
		errs = append(errs, "sender_name is required")
	}

	email := strings.TrimSpace(r.SenderEmail)
	if email == "" {
		errs = append(errs, "sender_email is required")
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs = append(errs, "sender_email is not a valid email address")
	}

	if strings.TrimSpace(r.MessageBody) == "" {
		errs = append(errs, "message_body is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ContactMessageResponse is the response payload returned after a contact
// message has been submitted successfully.
type ContactMessageResponse struct {
	ID          uuid.UUID `json:"id"`
	SenderName  string    `json:"sender_name"`
	SubmittedAt time.Time `json:"submitted_at"`
}
