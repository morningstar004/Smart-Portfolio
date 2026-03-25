package model

import (
	"time"

	"github.com/google/uuid"
)

// Sponsor represents a successful sponsorship payment recorded after a
// Razorpay webhook confirms capture. It maps to the sponsors table.
type Sponsor struct {
	ID                uuid.UUID `json:"id"`
	SponsorName       string    `json:"sponsor_name"`
	Email             string    `json:"email"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	RazorpayPaymentID string    `json:"razorpay_payment_id"`
	CreatedAt         time.Time `json:"created_at"`
}

// OutboxEvent represents a row in the outbox_events table. The transactional
// outbox pattern guarantees that domain events (e.g. SPONSOR_CREATED) are
// persisted atomically alongside the business data, then picked up by a
// background poller and dispatched to the in-process event bus.
type OutboxEvent struct {
	ID            uuid.UUID `json:"id"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	EventType     string    `json:"event_type"`
	Payload       string    `json:"payload"`
	IsProcessed   bool      `json:"is_processed"`
	EventID       string    `json:"event_id"`
	CreatedAt     time.Time `json:"created_at"`
}
