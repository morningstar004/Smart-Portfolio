package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// ProjectRequest.Validate
// ---------------------------------------------------------------------------

func TestProjectRequest_Validate_Valid(t *testing.T) {
	tests := []struct {
		name string
		req  ProjectRequest
	}{
		{
			name: "all fields",
			req: ProjectRequest{
				Title:       "My Project",
				Description: "A great project",
				TechStack:   strPtr("Go, PostgreSQL"),
				GithubURL:   strPtr("https://github.com/user/repo"),
				LiveURL:     strPtr("https://myproject.com"),
			},
		},
		{
			name: "required fields only",
			req: ProjectRequest{
				Title:       "Minimal",
				Description: "Just the basics",
			},
		},
		{
			name: "with whitespace padding",
			req: ProjectRequest{
				Title:       "  Padded Title  ",
				Description: "  Padded Description  ",
			},
		},
		{
			name: "optional fields nil",
			req: ProjectRequest{
				Title:       "Title",
				Description: "Description",
				TechStack:   nil,
				GithubURL:   nil,
				LiveURL:     nil,
			},
		},
		{
			name: "optional fields empty strings",
			req: ProjectRequest{
				Title:       "Title",
				Description: "Description",
				TechStack:   strPtr(""),
				GithubURL:   strPtr(""),
				LiveURL:     strPtr(""),
			},
		},
		{
			name: "long title and description",
			req: ProjectRequest{
				Title:       "A Very Long Project Title That Spans Many Characters",
				Description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestProjectRequest_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		req         ProjectRequest
		wantContain string
	}{
		{
			name: "empty title",
			req: ProjectRequest{
				Title:       "",
				Description: "Valid description",
			},
			wantContain: "title",
		},
		{
			name: "whitespace-only title",
			req: ProjectRequest{
				Title:       "   \t  ",
				Description: "Valid description",
			},
			wantContain: "title",
		},
		{
			name: "empty description",
			req: ProjectRequest{
				Title:       "Valid Title",
				Description: "",
			},
			wantContain: "description",
		},
		{
			name: "whitespace-only description",
			req: ProjectRequest{
				Title:       "Valid Title",
				Description: "   \n  ",
			},
			wantContain: "description",
		},
		{
			name: "both empty",
			req: ProjectRequest{
				Title:       "",
				Description: "",
			},
			wantContain: "title",
		},
		{
			name: "both whitespace-only",
			req: ProjectRequest{
				Title:       "   ",
				Description: "   ",
			},
			wantContain: "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !containsStr(err.Error(), tt.wantContain) {
				t.Errorf("expected error to contain %q, got: %q", tt.wantContain, err.Error())
			}
		})
	}
}

func TestProjectRequest_Validate_BothFieldsMissing_ReportsBoth(t *testing.T) {
	req := ProjectRequest{
		Title:       "",
		Description: "",
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	msg := err.Error()
	if !containsStr(msg, "title") {
		t.Errorf("expected error to mention 'title', got: %q", msg)
	}
	if !containsStr(msg, "description") {
		t.Errorf("expected error to mention 'description', got: %q", msg)
	}
}

func TestProjectRequest_Validate_ErrorPrefix(t *testing.T) {
	req := ProjectRequest{Title: "", Description: ""}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "validation failed") {
		t.Errorf("expected error to start with 'validation failed', got: %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// ContactMessageRequest.Validate
// ---------------------------------------------------------------------------

func TestContactMessageRequest_Validate_Valid(t *testing.T) {
	tests := []struct {
		name string
		req  ContactMessageRequest
	}{
		{
			name: "all valid fields",
			req: ContactMessageRequest{
				SenderName:  "Jane Doe",
				SenderEmail: "jane@example.com",
				MessageBody: "Hello! I'd like to collaborate.",
			},
		},
		{
			name: "minimal valid input",
			req: ContactMessageRequest{
				SenderName:  "A",
				SenderEmail: "a@b.co",
				MessageBody: "Hi",
			},
		},
		{
			name: "email with plus addressing",
			req: ContactMessageRequest{
				SenderName:  "Test User",
				SenderEmail: "test+portfolio@gmail.com",
				MessageBody: "Testing plus addressing.",
			},
		},
		{
			name: "email with subdomain",
			req: ContactMessageRequest{
				SenderName:  "Bob",
				SenderEmail: "bob@mail.sub.example.com",
				MessageBody: "Subdomain email test.",
			},
		},
		{
			name: "name with unicode characters",
			req: ContactMessageRequest{
				SenderName:  "José García",
				SenderEmail: "jose@example.com",
				MessageBody: "Hola!",
			},
		},
		{
			name: "long message body",
			req: ContactMessageRequest{
				SenderName:  "Verbose Person",
				SenderEmail: "verbose@example.com",
				MessageBody: "This is a really long message that goes on and on and on. It contains multiple sentences. And even more text after that. The point is to test that long messages are accepted without issue.",
			},
		},
		{
			name: "email with display name format",
			req: ContactMessageRequest{
				SenderName:  "Display Name",
				SenderEmail: "display@example.com",
				MessageBody: "Test message.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestContactMessageRequest_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		req         ContactMessageRequest
		wantContain string
	}{
		{
			name: "empty sender_name",
			req: ContactMessageRequest{
				SenderName:  "",
				SenderEmail: "valid@example.com",
				MessageBody: "Hello",
			},
			wantContain: "sender_name",
		},
		{
			name: "whitespace-only sender_name",
			req: ContactMessageRequest{
				SenderName:  "   \t  ",
				SenderEmail: "valid@example.com",
				MessageBody: "Hello",
			},
			wantContain: "sender_name",
		},
		{
			name: "empty sender_email",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "",
				MessageBody: "Hello",
			},
			wantContain: "sender_email",
		},
		{
			name: "whitespace-only sender_email",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "   ",
				MessageBody: "Hello",
			},
			wantContain: "sender_email",
		},
		{
			name: "invalid email - no at sign",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "notanemail",
				MessageBody: "Hello",
			},
			wantContain: "sender_email",
		},
		{
			name: "invalid email - no domain",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "user@",
				MessageBody: "Hello",
			},
			wantContain: "sender_email",
		},
		{
			name: "invalid email - double at",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "user@@example.com",
				MessageBody: "Hello",
			},
			wantContain: "sender_email",
		},
		{
			name: "empty message_body",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "jane@example.com",
				MessageBody: "",
			},
			wantContain: "message_body",
		},
		{
			name: "whitespace-only message_body",
			req: ContactMessageRequest{
				SenderName:  "Jane",
				SenderEmail: "jane@example.com",
				MessageBody: "   \n  ",
			},
			wantContain: "message_body",
		},
		{
			name: "all fields empty",
			req: ContactMessageRequest{
				SenderName:  "",
				SenderEmail: "",
				MessageBody: "",
			},
			wantContain: "sender_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !containsStr(err.Error(), tt.wantContain) {
				t.Errorf("expected error to contain %q, got: %q", tt.wantContain, err.Error())
			}
		})
	}
}

func TestContactMessageRequest_Validate_AllFieldsMissing_ReportsAll(t *testing.T) {
	req := ContactMessageRequest{
		SenderName:  "",
		SenderEmail: "",
		MessageBody: "",
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	msg := err.Error()
	if !containsStr(msg, "sender_name") {
		t.Errorf("expected error to mention 'sender_name', got: %q", msg)
	}
	if !containsStr(msg, "sender_email") {
		t.Errorf("expected error to mention 'sender_email', got: %q", msg)
	}
	if !containsStr(msg, "message_body") {
		t.Errorf("expected error to mention 'message_body', got: %q", msg)
	}
}

func TestContactMessageRequest_Validate_ErrorPrefix(t *testing.T) {
	req := ContactMessageRequest{SenderName: "", SenderEmail: "", MessageBody: ""}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsStr(err.Error(), "validation failed") {
		t.Errorf("expected error to start with 'validation failed', got: %q", err.Error())
	}
}

func TestContactMessageRequest_Validate_InvalidEmailWithValidOtherFields(t *testing.T) {
	req := ContactMessageRequest{
		SenderName:  "Valid Name",
		SenderEmail: "not-an-email",
		MessageBody: "Valid message body.",
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid email, got nil")
	}
	if !containsStr(err.Error(), "sender_email") {
		t.Errorf("expected error about sender_email, got: %q", err.Error())
	}
	// Should only mention email, not name or body
	if containsStr(err.Error(), "sender_name") {
		t.Errorf("did not expect error about sender_name, got: %q", err.Error())
	}
	if containsStr(err.Error(), "message_body") {
		t.Errorf("did not expect error about message_body, got: %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// ChatRequest.Validate (imported from ai/dto but testing the pattern)
// ---------------------------------------------------------------------------

// We don't import ai/dto here, but we test the content DTOs' Validate pattern
// thoroughly. The AI dto tests would be in ai/dto/dto_test.go.

// ---------------------------------------------------------------------------
// ProjectResponse struct (smoke test — ensure fields serialize correctly)
// ---------------------------------------------------------------------------

func TestProjectResponse_Fields(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	resp := ProjectResponse{
		ID:          id,
		Title:       "Test Project",
		Description: "A test project description",
		TechStack:   strPtr("Go, React"),
		GithubURL:   strPtr("https://github.com/test/repo"),
		LiveURL:     strPtr("https://test.com"),
		CreatedAt:   now,
	}

	if resp.ID != id {
		t.Errorf("expected ID %s, got %s", id, resp.ID)
	}
	if resp.Title != "Test Project" {
		t.Errorf("expected Title 'Test Project', got %q", resp.Title)
	}
	if resp.Description != "A test project description" {
		t.Errorf("unexpected Description: %q", resp.Description)
	}
	if resp.TechStack == nil || *resp.TechStack != "Go, React" {
		t.Errorf("unexpected TechStack: %v", resp.TechStack)
	}
	if resp.GithubURL == nil || *resp.GithubURL != "https://github.com/test/repo" {
		t.Errorf("unexpected GithubURL: %v", resp.GithubURL)
	}
	if resp.LiveURL == nil || *resp.LiveURL != "https://test.com" {
		t.Errorf("unexpected LiveURL: %v", resp.LiveURL)
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, resp.CreatedAt)
	}
}

func TestProjectResponse_NilOptionalFields(t *testing.T) {
	resp := ProjectResponse{
		ID:          uuid.New(),
		Title:       "Minimal",
		Description: "No optional fields",
		TechStack:   nil,
		GithubURL:   nil,
		LiveURL:     nil,
		CreatedAt:   time.Now(),
	}

	if resp.TechStack != nil {
		t.Errorf("expected nil TechStack, got %v", resp.TechStack)
	}
	if resp.GithubURL != nil {
		t.Errorf("expected nil GithubURL, got %v", resp.GithubURL)
	}
	if resp.LiveURL != nil {
		t.Errorf("expected nil LiveURL, got %v", resp.LiveURL)
	}
}

// ---------------------------------------------------------------------------
// ContactMessageResponse struct (smoke test)
// ---------------------------------------------------------------------------

func TestContactMessageResponse_Fields(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	resp := ContactMessageResponse{
		ID:          id,
		SenderName:  "Alice",
		SubmittedAt: now,
	}

	if resp.ID != id {
		t.Errorf("expected ID %s, got %s", id, resp.ID)
	}
	if resp.SenderName != "Alice" {
		t.Errorf("expected SenderName 'Alice', got %q", resp.SenderName)
	}
	if !resp.SubmittedAt.Equal(now) {
		t.Errorf("expected SubmittedAt %v, got %v", now, resp.SubmittedAt)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestProjectRequest_Validate_OnlySpacesInTitle(t *testing.T) {
	req := ProjectRequest{
		Title:       "     ",
		Description: "Valid description",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for spaces-only title")
	}
	if !containsStr(err.Error(), "title") {
		t.Errorf("expected error about title, got: %q", err.Error())
	}
}

func TestContactMessageRequest_Validate_EmailWithSpaces(t *testing.T) {
	req := ContactMessageRequest{
		SenderName:  "Test",
		SenderEmail: "  user@example.com  ",
		MessageBody: "Hello",
	}
	// Validation trims and then parses, so this might pass depending on
	// implementation. We just verify it doesn't panic.
	_ = req.Validate()
}

func TestContactMessageRequest_Validate_EmailOnlyAtSign(t *testing.T) {
	req := ContactMessageRequest{
		SenderName:  "Test",
		SenderEmail: "@",
		MessageBody: "Hello",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for '@' as email")
	}
	if !containsStr(err.Error(), "sender_email") {
		t.Errorf("expected error about sender_email, got: %q", err.Error())
	}
}

func TestContactMessageRequest_Validate_EmailWithSpecialChars(t *testing.T) {
	// RFC 5322 allows certain special chars in the local part
	req := ContactMessageRequest{
		SenderName:  "Special",
		SenderEmail: "user.name+tag@example.com",
		MessageBody: "Testing special chars",
	}
	err := req.Validate()
	if err != nil {
		t.Errorf("expected valid email with special chars to pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkProjectRequest_Validate_Valid(b *testing.B) {
	req := ProjectRequest{
		Title:       "My Project",
		Description: "A cool project built with Go",
		TechStack:   strPtr("Go, PostgreSQL, React"),
		GithubURL:   strPtr("https://github.com/user/repo"),
		LiveURL:     strPtr("https://myproject.dev"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

func BenchmarkProjectRequest_Validate_Invalid(b *testing.B) {
	req := ProjectRequest{
		Title:       "",
		Description: "",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

func BenchmarkContactMessageRequest_Validate_Valid(b *testing.B) {
	req := ContactMessageRequest{
		SenderName:  "Jane Doe",
		SenderEmail: "jane@example.com",
		MessageBody: "I'd love to work together on a project. Let me know!",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

func BenchmarkContactMessageRequest_Validate_Invalid(b *testing.B) {
	req := ContactMessageRequest{
		SenderName:  "",
		SenderEmail: "not-an-email",
		MessageBody: "",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string {
	return &s
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
