package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// PaymentHandler handles public payment and sponsor routes.
type PaymentHandler struct {
	paymentService service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler backed by the given PaymentService.
func NewPaymentHandler(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: svc}
}

// PaymentRoutes returns a chi.Router with all payment routes mounted.
// Mounted at /api/payments
func (h *PaymentHandler) PaymentRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/create-order", h.CreateOrder)
	return r
}

// SponsorRoutes returns a chi.Router with all sponsor routes mounted.
// Mounted at /api/sponsors
func (h *PaymentHandler) SponsorRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetSponsors)
	return r
}

// CreateOrder handles POST /api/payments/create-order.
func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Currency == "" {
		req.Currency = "INR"
	}

	if req.Amount <= 0 {
		httputil.WriteError(w, http.StatusBadRequest, "invalid amount")
		return
	}

	orderDetails, err := h.paymentService.CreateRazorpayOrder(req.Amount, req.Currency)
	if err != nil {
		log.Error().Err(err).Msg("payment_handler: failed to create razorpay order")
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create order: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orderDetails)
}

// GetSponsors handles GET /api/sponsors.
func (h *PaymentHandler) GetSponsors(w http.ResponseWriter, r *http.Request) {
	sponsors, err := h.paymentService.GetRecentSponsors(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "PaymentHandler.GetSponsors")
		return
	}

	// We only expose limited fields publicly, matching the Java backend.
	type SponsorResponse struct {
		SponsorName string  `json:"sponsorName"`
		Amount      float64 `json:"amount"`
		Currency    string  `json:"currency"`
	}

	responses := make([]SponsorResponse, 0, len(sponsors))
	for _, s := range sponsors {
		responses = append(responses, SponsorResponse{
			SponsorName: s.SponsorName,
			Amount:      s.Amount,
			Currency:    s.Currency,
		})
	}

	httputil.WriteJSON(w, http.StatusOK, responses)
}
