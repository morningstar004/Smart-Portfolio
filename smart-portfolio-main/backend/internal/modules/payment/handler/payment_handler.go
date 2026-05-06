package handler

import (
	"net/http"
	"strings"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/dto"
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
	r.Post("/verify-payment", h.VerifyPayment)
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
	var req dto.CreateOrderRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Currency == "" {
		req.Currency = "INR"
	}
	if err := req.Validate(); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	orderDetails, err := h.paymentService.CreateRazorpayOrder(req)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			httputil.WriteValidationError(w, err)
			return
		}
		log.Error().Err(err).Msg("payment_handler: failed to create razorpay order")
		httputil.WriteError(w, http.StatusInternalServerError, "Failed to create order: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, orderDetails)
}

// VerifyPayment handles POST /api/payments/verify-payment.
func (h *PaymentHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyPaymentRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	receipt, err := h.paymentService.VerifyCheckoutPayment(req)
	if err != nil {
		if strings.Contains(err.Error(), "validation failed") {
			httputil.WriteValidationError(w, err)
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, receipt)
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
