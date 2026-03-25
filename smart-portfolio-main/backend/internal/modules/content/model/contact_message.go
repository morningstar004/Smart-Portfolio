package model

import (
	"time"

	"github.com/google/uuid"
)

// ContactMessage represents a message submitted through the portfolio contact form.
// It maps to the contact_messages table in PostgreSQL.
type ContactMessage struct {
	ID          uuid.UUID `json:"id"`
	SenderName  string    `json:"sender_name"`
	SenderEmail string    `json:"sender_email"`
	MessageBody string    `json:"message_body"`
	IsRead      bool      `json:"is_read"`
	SubmittedAt time.Time `json:"submitted_at"`
}
