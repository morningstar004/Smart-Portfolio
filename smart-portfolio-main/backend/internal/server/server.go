package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ZRishu/smart-portfolio/docs"
	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/ZRishu/smart-portfolio/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
)

// Server wraps the http.Server and chi router, providing a clean interface
// for configuring routes, middleware, and graceful lifecycle management.
type Server struct {
	httpServer *http.Server
	router     chi.Router
	cfg        *config.Config
}

// New creates a new Server with the full middleware stack and CORS configured.
// Routes are NOT mounted here — call RegisterRoutes after construction so that
// all handler dependencies have been wired up by the caller.
func New(cfg *config.Config) *Server {
	r := chi.NewRouter()

	// ── Global middleware stack ──────────────────────────────────────────
	// Order matters: outermost middleware runs first.

	// 1. Panic recovery — must be outermost so it catches panics from
	//    every other middleware and handler.
	r.Use(middleware.Recoverer)

	// 2. Request ID — generates a UUID for correlated logging and injects
	//    it into the context + X-Request-ID response header.
	r.Use(middleware.RequestID)

	// 3. Structured request logging — logs method, path, status, latency,
	//    bytes written, and request ID for every request.
	r.Use(middleware.RequestLogger)

	// 4. Security headers — X-Content-Type-Options, X-Frame-Options, etc.
	r.Use(middleware.SecurityHeaders)

	// 5. Rate limiting — per-IP sliding window using httprate.
	r.Use(middleware.RateLimiter(cfg.RateLimit.RequestsPerSecond))

	// 6. CORS — configured to allow the frontend origin. In development
	//    mode (FRONTEND_URL empty or wildcard), all origins are allowed.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   buildAllowedOrigins(cfg.Frontend.URL),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Admin-Key"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300, // 5 minutes preflight cache
	}))

	srv := &Server{
		router: r,
		cfg:    cfg,
		httpServer: &http.Server{
			Addr:              ":" + cfg.Server.Port,
			Handler:           r,
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      120 * time.Second, // High because of SSE streaming.
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1 MB
		},
	}

	return srv
}

// ModuleRoutes groups all the sub-routers that each module exposes.
// By accepting chi.Router interfaces we break the import cycle: the server
// package never imports any handler package. The caller (main.go) wires
// concrete handlers and passes their .Routes() / .ChatRoutes() etc. here.
type ModuleRoutes struct {
	Projects        chi.Router // mounted at /api/projects
	Contact         chi.Router // mounted at /api/contact
	Chat            chi.Router // mounted at /api/chat
	Ingest          chi.Router // mounted at /api/ingest
	RazorpayWebhook chi.Router // mounted at /api/webhooks/razorpay
	Payments        chi.Router // mounted at /api/payments
	Sponsors        chi.Router // mounted at /api/sponsors
	Admin           chi.Router // mounted at /api/admin (auth-protected)
}

// RegisterRoutes mounts all module-level sub-routers onto the main chi router.
// This method should be called exactly once after all handlers are constructed.
//
// Route tree:
//
//	GET  /healthz                        → health check
//	GET  /docs                           → Swagger UI (API documentation)
//	GET  /docs/*                         → OpenAPI spec + Swagger assets
//	/api
//	  /projects                          → ProjectHandler.Routes()
//	    GET  /                           → list all projects
//	    POST /                           → create a project
//	    PUT  /{id}                       → update a project
//	    DELETE /{id}                     → delete a project
//	  /contact                           → ContactHandler.Routes()
//	    POST /                           → submit a contact message
//	    GET  /                           → list all messages (admin)
//	    GET  /unread                     → list unread messages (admin)
//	    PATCH /{id}/read                 → mark as read (admin)
//	    DELETE /{id}                     → delete a message (admin)
//	  /chat                              → AIHandler.ChatRoutes()
//	    POST /                           → ask question (JSON response)
//	    POST /stream                     → ask question (SSE streaming)
//	  /ingest                            → AIHandler.IngestRoutes() [admin-auth]
//	    POST /                           → upload PDF resume
//	    POST /text                       → ingest raw text
//	    DELETE /                         → clear vector store
//	  /webhooks/razorpay                 → WebhookHandler.Routes()
//	    POST /                           → Razorpay webhook
//	  /payments                          → PaymentHandler.PaymentRoutes()
//	    POST /create-order               → create razorpay order
//	  /sponsors                          → PaymentHandler.SponsorRoutes()
//	    GET /                            → list recent sponsors
//	  /admin                             → AdminHandler.Routes() [admin-auth]
//	    GET /health                      → deep health check (DB ping)
//	    GET /stats                       → dashboard aggregate stats
//	    GET /sponsors                    → list all sponsors
func (s *Server) RegisterRoutes(m ModuleRoutes) {
	r := s.router

	// Health check — outside /api so load balancers can hit it without
	// going through API middleware.
	r.Get("/healthz", middleware.Healthcheck)

	// Swagger UI — serves the interactive API documentation at /docs.
	r.Get("/docs", docs.SwaggerRedirect)
	r.Get("/docs/*", docs.SwaggerHandler("/docs"))

	// Admin auth middleware (no-op when ADMIN_API_KEY is empty)
	adminAuth := middleware.AdminAuth(s.cfg.Admin.APIKey)

	// Default body size limit for JSON endpoints (1 MB)
	jsonBodyLimit := middleware.MaxBodySize(1 << 20)

	// API route group.
	r.Route("/api", func(api chi.Router) {
		// Apply a default body size limit to all API routes.
		// Individual handlers (e.g. PDF upload) override this with their
		// own MaxBytesReader where a larger limit is needed.
		api.Use(jsonBodyLimit)

		// ── Public endpoints ─────────────────────────────────────────
		// Content module
		if m.Projects != nil {
			api.Mount("/projects", m.Projects)
		}
		if m.Contact != nil {
			// POST /api/contact is public; GET/PATCH/DELETE are admin but
			// the handler already mixes them. We apply admin auth at the
			// contact handler level for list/read/delete if needed, or
			// protect the whole group here for simplicity.
			api.Mount("/contact", m.Contact)
		}

		// AI chat is public (visitors can ask questions)
		if m.Chat != nil {
			api.Mount("/chat", m.Chat)
		}

		// Payment webhook is secured by HMAC, not admin key
		if m.RazorpayWebhook != nil {
			api.Mount("/webhooks/razorpay", m.RazorpayWebhook)
		}
		if m.Payments != nil {
			api.Mount("/payments", m.Payments)
		}
		if m.Sponsors != nil {
			api.Mount("/sponsors", m.Sponsors)
		}

		// ── Admin-protected endpoints ────────────────────────────────
		// Ingestion should be admin-only (only the portfolio owner uploads resumes)
		if m.Ingest != nil {
			api.Route("/ingest", func(ingest chi.Router) {
				ingest.Use(adminAuth)
				ingest.Mount("/", m.Ingest)
			})
		}

		// Admin dashboard, sponsors, stats, deep health check
		if m.Admin != nil {
			api.Route("/admin", func(admin chi.Router) {
				admin.Use(adminAuth)
				admin.Mount("/", m.Admin)
			})
		}
	})

	// Log all registered routes at startup for debugging.
	logRegisteredRoutes(r)
}

// Start begins listening for HTTP connections. It blocks until the server is
// shut down or encounters a fatal error. ErrServerClosed is NOT treated as
// an error because it is the expected result of a graceful Shutdown() call.
func (s *Server) Start() error {
	log.Info().
		Str("addr", s.httpServer.Addr).
		Msg("server: starting HTTP server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: listen failed: %w", err)
	}

	return nil
}

// Shutdown gracefully drains in-flight connections with the given timeout.
// After the timeout expires, any remaining connections are forcibly closed.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("server: initiating graceful shutdown")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server: shutdown failed: %w", err)
	}

	log.Info().Msg("server: HTTP server stopped")
	return nil
}

// Router returns the underlying chi.Router for testing or introspection.
func (s *Server) Router() chi.Router {
	return s.router
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildAllowedOrigins constructs the CORS allowed origins list. If the
// frontend URL is empty or "*", all origins are allowed (development mode).
// Otherwise, only the specified frontend URL is allowed.
func buildAllowedOrigins(frontendURL string) []string {
	if frontendURL == "" || frontendURL == "*" {
		return []string{"*"}
	}

	origins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:5174",
	}

	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		seen[origin] = struct{}{}
	}

	for _, raw := range strings.FieldsFunc(frontendURL, func(r rune) bool {
		return r == ',' || r == '\n'
	}) {
		origin := strings.TrimSpace(strings.TrimRight(raw, "/"))
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		origins = append(origins, origin)
		seen[origin] = struct{}{}
	}

	return origins
}

// logRegisteredRoutes walks the chi router tree and logs every registered
// route at Info level. This is invaluable for debugging routing issues at
// startup.
func logRegisteredRoutes(r chi.Router) {
	log.Info().Msg("server: registered routes:")

	walkFunc := func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		log.Info().
			Str("method", method).
			Str("route", route).
			Msg("  route")
		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		log.Warn().Err(err).Msg("server: failed to walk route tree")
	}
}
