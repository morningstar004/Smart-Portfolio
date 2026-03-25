package model

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a portfolio project stored in the projects table.
type Project struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TechStack   *string   `json:"tech_stack,omitempty"`
	GithubURL   *string   `json:"github_url,omitempty"`
	LiveURL     *string   `json:"live_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
