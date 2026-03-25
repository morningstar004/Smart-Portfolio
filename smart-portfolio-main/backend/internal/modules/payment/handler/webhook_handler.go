package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// WebhookHandler handles incoming Razorpay webhook HTTP requests.
type WebhookHandler struct {
	paymentService service.PaymentService
}

// NewWebhookHandler creates a new WebhookHandler backed by the given PaymentService.
func NewWebhookHandler(svc service.PaymentService) *WebhookHandler {
	return &WebhookHandler{paymentService: svc}
}

// Routes returns a chi.Router with all webhook routes mounted.
//
// Mounted at /api/webhooks/razorpay:
//
//	POST /  → HandleRazorpayWebhook
func (h *WebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.HandleRazorpayWebhook)
	return r
}

// HandleRazorpayWebhook handles POST /api/webhooks/razorpay.
//
// Processing steps:
//  1. Read the raw request body (needed for HMAC signature verification).
//  2. Extract and verify the X-Razorpay-Signature header via HMAC-SHA256.
//  3. Parse the "event" field from the JSON payload.
//  4. For "payment.captured" events, delegate to PaymentService which
//     atomically persists the sponsor and outbox event.
//  5. Handle duplicate webhook deliveries gracefully (UNIQUE constraint
//     on outbox_events.event_id).
//
// All other event types are acknowledged with 200 OK but not processed.
func (h *WebhookHandler) HandleRazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	// Step 1: Read the raw body — HMAC is computed over raw bytes.
	const maxBodySize = 1 << 20 // 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("webhook_handler: failed to read request body")
		httputil.WriteError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	if len(payload) == 0 {
		log.Warn().Msg("webhook_handler: received empty webhook payload")
		httputil.WriteError(w, http.StatusBadRequest, "empty request body")
		return
	}

	// Step 2: Extract the signature header.
	signature := r.Header.Get("X-Razorpay-Signature")
	if signature == "" {
		log.Warn().Msg("webhook_handler: missing X-Razorpay-Signature header")
		httputil.WriteError(w, http.StatusBadRequest, "missing X-Razorpay-Signature header")
		return
	}

	// Step 3: Verify the cryptographic signature.
	if !h.paymentService.VerifyWebhookSignature(payload, signature) {
		log.Warn().Msg("webhook_handler: invalid Razorpay webhook signature detected")
		httputil.WriteError(w, http.StatusBadRequest, "invalid webhook signature")
		return
	}

	// Step 4: Determine the event type.
	eventType := extractEventType(payload)

	log.Info().
		Str("event_type", eventType).
		Int("payload_size", len(payload)).
		Msg("webhook_handler: verified Razorpay webhook received")

	// Step 5: Route by event type.
	switch eventType {
	case "payment.captured":
		if err := h.paymentService.HandlePaymentCaptured(r.Context(), payload); err != nil {
			// Handle duplicate events gracefully.
			if service.IsDuplicateEventError(err) {
				log.Info().Msg("webhook_handler: duplicate webhook event — safely ignored")
				httputil.WriteJSON(w, http.StatusOK, map[string]string{
					"message": "duplicate event ignored",
				})
				return
			}

			log.Error().Err(err).Msg("webhook_handler: failed to process payment.captured event")
			httputil.WriteError(w, http.StatusInternalServerError, "webhook processing failed")
			return
		}

		log.Info().Msg("webhook_handler: sponsorship processed successfully")
		httputil.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "sponsorship processed successfully",
		})

	default:
		// Acknowledge receipt of unhandled event types so Razorpay does not
		// retry them.
		log.Debug().
			Str("event_type", eventType).
			Msg("webhook_handler: event type received but not handled — acknowledging")

		httputil.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "event received but not processed",
		})
	}
}

// extractEventType performs a minimal JSON extraction to read the top-level
// "event" field from the Razorpay webhook payload without fully unmarshalling
// the entire nested structure.
func extractEventType(payload []byte) string {
	var envelope struct {
		Event string `json:"event"`
	}

	if err := json.Unmarshal(payload, &envelope); err != nil {
		log.Warn().
			Err(err).
			Msg("webhook_handler: failed to extract event type from payload")
		return ""
	}

	return envelope.Event
}
