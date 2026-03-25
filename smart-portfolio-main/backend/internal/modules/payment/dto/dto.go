package dto

import (
	"time"

	"github.com/google/uuid"
)

// SponsorResponse is the response payload returned when reading a sponsor.
type SponsorResponse struct {
	ID                uuid.UUID `json:"id"`
	SponsorName       string    `json:"sponsor_name"`
	Email             string    `json:"email"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	RazorpayPaymentID string    `json:"razorpay_payment_id"`
	CreatedAt         time.Time `json:"created_at"`
}

// SponsorStatsResponse is returned by the stats endpoint with aggregate
// sponsorship data for the admin dashboard.
type SponsorStatsResponse struct {
	TotalSponsors int64   `json:"total_sponsors"`
	TotalAmount   float64 `json:"total_amount"`
	Currency      string  `json:"currency"`
}

// DashboardStatsResponse aggregates stats from all modules into a single
// response for the admin dashboard overview endpoint.
type DashboardStatsResponse struct {
	Projects        int64                `json:"projects"`
	ContactMessages ContactMessageStats  `json:"contact_messages"`
	Sponsors        SponsorStatsResponse `json:"sponsors"`
	VectorStore     VectorStoreStats     `json:"vector_store"`
	SemanticCache   SemanticCacheStats   `json:"semantic_cache"`
	OutboxPending   int64                `json:"outbox_pending"`
}

// ContactMessageStats holds aggregate counts for contact messages.
type ContactMessageStats struct {
	Total  int64 `json:"total"`
	Unread int64 `json:"unread"`
}

// VectorStoreStats holds aggregate counts for the RAG vector store.
type VectorStoreStats struct {
	Documents int64 `json:"documents"`
}

// SemanticCacheStats holds aggregate counts for the AI semantic cache.
type SemanticCacheStats struct {
	Entries int64 `json:"entries"`
}
