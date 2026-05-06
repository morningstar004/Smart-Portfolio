package dto

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateOrderRequest is the payload for initializing a sponsorship payment.
type CreateOrderRequest struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	SponsorName  string  `json:"sponsor_name,omitempty"`
	SponsorEmail string  `json:"sponsor_email,omitempty"`
}

func (r CreateOrderRequest) Validate() error {
	if r.Amount <= 0 {
		return fmt.Errorf("validation failed: amount must be greater than zero")
	}
	if !hasAtMostTwoDecimals(r.Amount) {
		return fmt.Errorf("validation failed: amount must have at most two decimal places")
	}
	if strings.TrimSpace(r.Currency) == "" {
		return fmt.Errorf("validation failed: currency is required")
	}
	return nil
}

// CreateOrderResponse is returned after a Razorpay order is created.
type CreateOrderResponse struct {
	ID       string            `json:"id"`
	Amount   int               `json:"amount"`
	Currency string            `json:"currency"`
	KeyID    string            `json:"key_id"`
	Receipt  string            `json:"receipt"`
	Notes    map[string]string `json:"notes,omitempty"`
}

// VerifyPaymentRequest validates a completed Razorpay checkout response.
type VerifyPaymentRequest struct {
	RazorpayOrderID   string  `json:"razorpay_order_id"`
	RazorpayPaymentID string  `json:"razorpay_payment_id"`
	RazorpaySignature string  `json:"razorpay_signature"`
	SponsorName       string  `json:"sponsor_name,omitempty"`
	SponsorEmail      string  `json:"sponsor_email,omitempty"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
}

func (r VerifyPaymentRequest) Validate() error {
	if strings.TrimSpace(r.RazorpayOrderID) == "" {
		return fmt.Errorf("validation failed: razorpay_order_id is required")
	}
	if strings.TrimSpace(r.RazorpayPaymentID) == "" {
		return fmt.Errorf("validation failed: razorpay_payment_id is required")
	}
	if strings.TrimSpace(r.RazorpaySignature) == "" {
		return fmt.Errorf("validation failed: razorpay_signature is required")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("validation failed: amount must be greater than zero")
	}
	if !hasAtMostTwoDecimals(r.Amount) {
		return fmt.Errorf("validation failed: amount must have at most two decimal places")
	}
	if strings.TrimSpace(r.Currency) == "" {
		return fmt.Errorf("validation failed: currency is required")
	}
	return nil
}

func hasAtMostTwoDecimals(value float64) bool {
	if value < 0 {
		return false
	}

	scaled := value * 100
	return math.Abs(scaled-math.Round(scaled)) < 1e-9
}

// PaymentReceiptResponse is returned after checkout verification succeeds.
type PaymentReceiptResponse struct {
	ReceiptNumber     string    `json:"receipt_number"`
	SponsorName       string    `json:"sponsor_name"`
	SponsorEmail      string    `json:"sponsor_email,omitempty"`
	RecipientName     string    `json:"recipient_name"`
	RecipientRole     string    `json:"recipient_role"`
	RecipientLocation string    `json:"recipient_location"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	RazorpayOrderID   string    `json:"razorpay_order_id"`
	RazorpayPaymentID string    `json:"razorpay_payment_id"`
	IssuedAt          time.Time `json:"issued_at"`
	Message           string    `json:"message"`
}

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
