package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/model"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/repository"
	"github.com/rs/zerolog/log"
)

// PaymentService defines the interface for payment-related business logic.
// It handles Razorpay webhook verification and orchestrates the transactional
// outbox pattern for reliable event processing.
type PaymentService interface {
	// VerifyWebhookSignature validates the cryptographic signature sent by
	// Razorpay in the X-Razorpay-Signature header. Returns true if the
	// signature is valid, false otherwise.
	VerifyWebhookSignature(payload []byte, signature string) bool

	// HandlePaymentCaptured processes a "payment.captured" webhook event.
	// It extracts the relevant fields from the JSON payload, persists the
	// sponsor + outbox event atomically, and returns nil on success.
	//
	// If the same Razorpay event ID has already been processed (duplicate
	// webhook delivery), this method returns a DuplicateEventError so the
	// caller can respond with 200 OK without re-processing.
	HandlePaymentCaptured(ctx context.Context, payload []byte) error

	// CreateRazorpayOrder creates a new order in Razorpay
	CreateRazorpayOrder(amount float64, currency string) (map[string]interface{}, error)

	// GetRecentSponsors fetches the top recent sponsors.
	GetRecentSponsors(ctx context.Context) ([]model.Sponsor, error)
}

// DuplicateEventError is returned when a webhook event has already been
// processed. The caller should treat this as a safe no-op and respond
// with HTTP 200 to acknowledge receipt.
type DuplicateEventError struct {
	EventID string
}

func (e *DuplicateEventError) Error() string {
	return fmt.Sprintf("duplicate webhook event: %s", e.EventID)
}

// IsDuplicateEventError checks whether the given error is a DuplicateEventError.
// This is useful in the handler layer to distinguish duplicate events from real
// failures without type-asserting directly.
func IsDuplicateEventError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*DuplicateEventError)
	return ok
}

// paymentService is the concrete implementation of PaymentService.
type paymentService struct {
	repo          *repository.PaymentRepository
	keyID         string
	keySecret     string
	webhookSecret string
}

// NewPaymentService creates a new PaymentService configured with the Razorpay
// credentials and backed by the given payment repository.
func NewPaymentService(repo *repository.PaymentRepository, cfg config.RazorpayConfig) PaymentService {
	if cfg.WebhookSecret == "" {
		log.Warn().Msg("payment_service: RAZORPAY_WEBHOOK_SECRET is empty — webhook signature verification will always fail")
	}
	if cfg.KeyID == "" || cfg.KeySecret == "" {
		log.Warn().Msg("payment_service: RAZORPAY_KEY_ID or RAZORPAY_KEY_SECRET is empty — order creation will fail")
	}

	log.Info().Msg("payment_service: initialized")

	return &paymentService{
		repo:          repo,
		keyID:         cfg.KeyID,
		keySecret:     cfg.KeySecret,
		webhookSecret: cfg.WebhookSecret,
	}
}

// VerifyWebhookSignature computes an HMAC-SHA256 of the raw request body using
// the configured webhook secret and compares it to the signature provided by
// Razorpay in the X-Razorpay-Signature header.
//
// This is the standard Razorpay webhook verification algorithm:
//
//	expected = HMAC-SHA256(webhook_secret, request_body)
//	valid    = constant_time_compare(expected, X-Razorpay-Signature)
//
// Using hmac.Equal ensures the comparison is constant-time, preventing timing
// side-channel attacks.
func (s *paymentService) VerifyWebhookSignature(payload []byte, signature string) bool {
	if s.webhookSecret == "" {
		log.Error().Msg("payment_service: cannot verify signature — webhook secret is not configured")
		return false
	}

	if len(payload) == 0 || signature == "" {
		log.Warn().Msg("payment_service: empty payload or signature — rejecting")
		return false
	}

	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	valid := hmac.Equal([]byte(expectedSignature), []byte(signature))

	if !valid {
		log.Warn().
			Str("expected_prefix", truncate(expectedSignature, 12)).
			Str("received_prefix", truncate(signature, 12)).
			Msg("payment_service: webhook signature mismatch")
	}

	return valid
}

// HandlePaymentCaptured processes a Razorpay "payment.captured" webhook event.
//
// It performs the following steps:
//  1. Parses the JSON payload to extract the Razorpay event ID, payment entity
//     fields (payment ID, email, currency, amount), and the sponsor name from
//     the custom "notes" object.
//  2. Converts the amount from paise (Razorpay's smallest currency unit) to the
//     major currency unit by dividing by 100.
//  3. Delegates to the repository's ProcessSponsorshipTx method which atomically
//     inserts both the sponsor row and the outbox event inside a single database
//     transaction.
//
// If the Razorpay event ID has already been seen (UNIQUE constraint violation on
// the outbox_events.event_id column), a DuplicateEventError is returned so the
// caller can handle it gracefully.
func (s *paymentService) HandlePaymentCaptured(ctx context.Context, payload []byte) error {
	// ── Parse the webhook payload ───────────────────────────────────────
	var root struct {
		Event   string `json:"event"`
		ID      string `json:"id"` // Razorpay's unique webhook event ID
		Payload struct {
			Payment struct {
				Entity struct {
					ID       string            `json:"id"` // e.g. "pay_xxxxx"
					Email    string            `json:"email"`
					Currency string            `json:"currency"`
					Amount   float64           `json:"amount"` // in paise
					Notes    map[string]string `json:"notes"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(payload, &root); err != nil {
		return fmt.Errorf("payment_service.HandlePaymentCaptured: failed to parse webhook payload: %w", err)
	}

	razorpayEventID := root.ID
	entity := root.Payload.Payment.Entity
	paymentID := entity.ID
	email := entity.Email
	currency := entity.Currency

	// Razorpay sends amounts in the smallest currency unit (paise for INR,
	// cents for USD). Convert to the major unit.
	amount := entity.Amount / 100.0

	// Extract the sponsor name from the custom "notes" object that the
	// frontend attaches when creating the Razorpay order. Fall back to a
	// sensible default if the note is missing.
	name := "Anonymous Sponsor"
	if n, ok := entity.Notes["sponsor_name"]; ok && strings.TrimSpace(n) != "" {
		name = strings.TrimSpace(n)
	}

	// ── Validate extracted fields ───────────────────────────────────────
	if razorpayEventID == "" {
		return fmt.Errorf("payment_service.HandlePaymentCaptured: missing Razorpay event ID in payload")
	}
	if paymentID == "" {
		return fmt.Errorf("payment_service.HandlePaymentCaptured: missing payment ID in payload")
	}
	if email == "" {
		return fmt.Errorf("payment_service.HandlePaymentCaptured: missing email in payload")
	}
	if currency == "" {
		currency = "INR"
	}
	if amount <= 0 {
		return fmt.Errorf("payment_service.HandlePaymentCaptured: invalid amount %.2f", amount)
	}

	log.Info().
		Str("razorpay_event_id", razorpayEventID).
		Str("payment_id", paymentID).
		Str("email", email).
		Str("name", name).
		Float64("amount", amount).
		Str("currency", currency).
		Msg("payment_service: processing captured payment")

	// ── Execute the transactional outbox write ──────────────────────────
	sponsorID, err := s.repo.ProcessSponsorshipTx(
		ctx,
		razorpayEventID,
		paymentID,
		name,
		email,
		amount,
		currency,
	)
	if err != nil {
		// Check for duplicate event (unique constraint on event_id).
		if isDuplicateKeyError(err) {
			log.Info().
				Str("razorpay_event_id", razorpayEventID).
				Msg("payment_service: duplicate webhook event — safely ignored")
			return &DuplicateEventError{EventID: razorpayEventID}
		}
		return fmt.Errorf("payment_service.HandlePaymentCaptured: %w", err)
	}

	log.Info().
		Str("sponsor_id", sponsorID.String()).
		Str("payment_id", paymentID).
		Msg("payment_service: sponsorship processed and committed to outbox")

	return nil
}

// isDuplicateKeyError checks whether a database error is a unique constraint
// violation. PostgreSQL reports this as SQLSTATE 23505 and pgx surfaces it in
// the error message.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint")
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// CreateRazorpayOrder creates a new order via the Razorpay API.
func (s *paymentService) CreateRazorpayOrder(amount float64, currency string) (map[string]interface{}, error) {
	if s.keyID == "" || s.keySecret == "" {
		return nil, fmt.Errorf("razorpay credentials not configured")
	}

	importRazorpay := "github.com/razorpay/razorpay-go"
	_ = importRazorpay // tricking go imports if we don't use it directly at top, wait we should just do HTTP request to avoid messing with top imports in this replace block, or I can update the top imports later. I'll use standard net/http for a simple API call.

	url := "https://api.razorpay.com/v1/orders"

	// Razorpay requires amount in paise
	amountInPaise := int(amount * 100)
	receipt := "txn_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	payload := map[string]interface{}{
		"amount":   amountInPaise,
		"currency": currency,
		"receipt":  receipt,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(s.keyID, s.keySecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("razorpay api error: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Return the public key with the order payload so the frontend does not need
	// a separate Razorpay env var. The secret remains backend-only.
	return map[string]interface{}{
		"id":       result["id"],
		"amount":   result["amount"],
		"currency": result["currency"],
		"key_id":   s.keyID,
	}, nil
}

// GetRecentSponsors fetches the top 10 recent sponsors.
func (s *paymentService) GetRecentSponsors(ctx context.Context) ([]model.Sponsor, error) {
	return s.repo.FindRecentSponsors(ctx, 10)
}
