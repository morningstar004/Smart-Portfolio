package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/database"
	"github.com/ZRishu/smart-portfolio/internal/httputil"
	airepository "github.com/ZRishu/smart-portfolio/internal/modules/ai/repository"
	"github.com/ZRishu/smart-portfolio/internal/modules/content/repository"
	paymentdto "github.com/ZRishu/smart-portfolio/internal/modules/payment/dto"
	paymentmodel "github.com/ZRishu/smart-portfolio/internal/modules/payment/model"
	paymentrepository "github.com/ZRishu/smart-portfolio/internal/modules/payment/repository"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// AdminHandler handles HTTP requests for admin-only endpoints such as
// sponsor listings, dashboard statistics, and a deep health check that
// verifies database connectivity.
type AdminHandler struct {
	pg                *database.Postgres
	projectRepo       *repository.ProjectRepository
	contactRepo       *repository.ContactRepository
	paymentRepo       *paymentrepository.PaymentRepository
	vectorStoreRepo   *airepository.VectorStoreRepository
	semanticCacheRepo *airepository.SemanticCacheRepository
}

// NewAdminHandler creates a new AdminHandler wired to all the repositories
// needed for aggregate stats and listings.
func NewAdminHandler(
	pg *database.Postgres,
	projectRepo *repository.ProjectRepository,
	contactRepo *repository.ContactRepository,
	paymentRepo *paymentrepository.PaymentRepository,
	vectorStoreRepo *airepository.VectorStoreRepository,
	semanticCacheRepo *airepository.SemanticCacheRepository,
) *AdminHandler {
	return &AdminHandler{
		pg:                pg,
		projectRepo:       projectRepo,
		contactRepo:       contactRepo,
		paymentRepo:       paymentRepo,
		vectorStoreRepo:   vectorStoreRepo,
		semanticCacheRepo: semanticCacheRepo,
	}
}

// Routes returns a chi.Router with all admin routes mounted.
//
// Mounted at /api/admin (protected by AdminAuth middleware):
//
//	GET /health         → DeepHealthCheck (DB connectivity check)
//	GET /stats          → DashboardStats (aggregate stats from all modules)
//	GET /sponsors       → ListSponsors (all sponsors, newest first)
func (h *AdminHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/health", h.DeepHealthCheck)
	r.Get("/stats", h.DashboardStats)
	r.Get("/sponsors", h.ListSponsors)

	return r
}

// DeepHealthCheck handles GET /api/admin/health. Unlike the public /healthz
// endpoint which only returns a static response, this endpoint actually pings
// the database to verify connectivity. It returns the round-trip latency so
// operators can monitor database performance.
func (h *AdminHandler) DeepHealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := h.pg.HealthCheck(ctx)
	latency := time.Since(start)

	if err != nil {
		log.Error().Err(err).Dur("latency", latency).Msg("admin: deep health check failed")
		httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":     "unhealthy",
			"database":   "unreachable",
			"latency_ms": latency.Milliseconds(),
			"error":      err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "healthy",
		"database":   "connected",
		"latency_ms": latency.Milliseconds(),
	})
}

// DashboardStats handles GET /api/admin/stats. It aggregates counts and
// totals from every module into a single response that the admin dashboard
// can display as an overview.
func (h *AdminHandler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Fetch all stats concurrently for maximum speed.
	type result struct {
		stats paymentdto.DashboardStatsResponse
		err   error
	}

	ch := make(chan result, 1)
	go func() {
		var s paymentdto.DashboardStatsResponse

		// Project count — reuse FindAll and count in Go to avoid adding
		// a dedicated Count method right now. For large datasets, add a
		// COUNT(*) query.
		projects, err := h.projectRepo.FindAll(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.Projects = int64(len(projects))

		// Contact message counts
		total, unread, err := h.contactRepo.Count(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.ContactMessages = paymentdto.ContactMessageStats{
			Total:  total,
			Unread: unread,
		}

		// Sponsor stats
		sponsorCount, totalAmount, err := h.paymentRepo.CountSponsors(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.Sponsors = paymentdto.SponsorStatsResponse{
			TotalSponsors: sponsorCount,
			TotalAmount:   totalAmount,
			Currency:      "INR",
		}

		// Vector store document count
		docCount, err := h.vectorStoreRepo.Count(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.VectorStore = paymentdto.VectorStoreStats{Documents: docCount}

		// Semantic cache entry count
		cacheCount, err := h.semanticCacheRepo.Count(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.SemanticCache = paymentdto.SemanticCacheStats{Entries: cacheCount}

		// Outbox pending count
		pendingCount, err := h.paymentRepo.PendingOutboxCount(ctx)
		if err != nil {
			ch <- result{err: err}
			return
		}
		s.OutboxPending = pendingCount

		ch <- result{stats: s}
	}()

	res := <-ch
	if res.err != nil {
		httputil.WriteInternalError(w, res.err, "AdminHandler.DashboardStats")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, res.stats)
}

// ListSponsors handles GET /api/admin/sponsors. It returns every sponsor
// ordered by creation date descending, with payment details.
func (h *AdminHandler) ListSponsors(w http.ResponseWriter, r *http.Request) {
	sponsors, err := h.paymentRepo.FindAllSponsors(r.Context())
	if err != nil {
		httputil.WriteInternalError(w, err, "AdminHandler.ListSponsors")
		return
	}

	// Convert model to DTO
	responses := make([]paymentdto.SponsorResponse, 0, len(sponsors))
	for _, s := range sponsors {
		responses = append(responses, sponsorModelToResponse(s))
	}

	httputil.WriteJSON(w, http.StatusOK, responses)
}

// sponsorModelToResponse converts a payment model.Sponsor into a
// paymentdto.SponsorResponse.
func sponsorModelToResponse(s paymentmodel.Sponsor) paymentdto.SponsorResponse {
	return paymentdto.SponsorResponse{
		ID:                s.ID,
		SponsorName:       s.SponsorName,
		Email:             s.Email,
		Amount:            s.Amount,
		Currency:          s.Currency,
		Status:            s.Status,
		RazorpayPaymentID: s.RazorpayPaymentID,
		CreatedAt:         s.CreatedAt,
	}
}
