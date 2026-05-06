package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/ZRishu/smart-portfolio/internal/database"
	adminhandler "github.com/ZRishu/smart-portfolio/internal/modules/admin/handler"
	aihandler "github.com/ZRishu/smart-portfolio/internal/modules/ai/handler"
	airepository "github.com/ZRishu/smart-portfolio/internal/modules/ai/repository"
	aiservice "github.com/ZRishu/smart-portfolio/internal/modules/ai/service"
	contenthandler "github.com/ZRishu/smart-portfolio/internal/modules/content/handler"
	contentrepository "github.com/ZRishu/smart-portfolio/internal/modules/content/repository"
	contentservice "github.com/ZRishu/smart-portfolio/internal/modules/content/service"
	contentworker "github.com/ZRishu/smart-portfolio/internal/modules/content/worker"
	notifservice "github.com/ZRishu/smart-portfolio/internal/modules/notification/service"
	paymenthandler "github.com/ZRishu/smart-portfolio/internal/modules/payment/handler"
	paymentrepository "github.com/ZRishu/smart-portfolio/internal/modules/payment/repository"
	paymentservice "github.com/ZRishu/smart-portfolio/internal/modules/payment/service"
	"github.com/ZRishu/smart-portfolio/internal/modules/payment/worker"
	"github.com/ZRishu/smart-portfolio/internal/platform/cache"
	"github.com/ZRishu/smart-portfolio/internal/platform/eventbus"
	"github.com/ZRishu/smart-portfolio/internal/server"
)

var version = "dev"

func main() {
	// ─────────────────────────────────────────────────────────────────────
	// 1. Logger
	// ─────────────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if os.Getenv("LOG_LEVEL") == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// Use pretty console output in development, JSON in production.
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	log.Info().Str("version", version).Msg("smart-portfolio: starting application")

	// ─────────────────────────────────────────────────────────────────────
	// 2. Configuration
	// ─────────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	log.Info().Msg("configuration loaded successfully")

	// ─────────────────────────────────────────────────────────────────────
	// 3. Root context — cancelled on SIGINT / SIGTERM
	// ─────────────────────────────────────────────────────────────────────
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// ─────────────────────────────────────────────────────────────────────
	// 4. Database
	// ─────────────────────────────────────────────────────────────────────
	pg, err := database.New(rootCtx, cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to PostgreSQL")
	}
	defer pg.Close()

	// Run idempotent migrations.
	if err := pg.RunMigrations(rootCtx, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("database migrations failed")
	}

	// ─────────────────────────────────────────────────────────────────────
	// 5. Platform services (cache, event bus)
	// ─────────────────────────────────────────────────────────────────────
	appCache := cache.New(cfg.Cache)
	bus := eventbus.New(rootCtx)

	// ─────────────────────────────────────────────────────────────────────
	// 6. Notification service
	// ─────────────────────────────────────────────────────────────────────
	discordService := notifservice.NewDiscordNotificationService(cfg.Discord)

	// ─────────────────────────────────────────────────────────────────────
	// 7. Content module (projects + contact messages + profile)
	// ─────────────────────────────────────────────────────────────────────
	projectRepo := contentrepository.NewProjectRepository(pg.Pool)
	githubProfileRepo := contentrepository.NewGitHubProfileRepository(pg.Pool)
	githubRepoRepo := contentrepository.NewGitHubRepositoryRepository(pg.Pool)
	contactRepo := contentrepository.NewContactRepository(pg.Pool)
	profileRepo := contentrepository.NewProfileRepository(pg)

	projectSvc := contentservice.NewProjectService(projectRepo, githubRepoRepo, githubProfileRepo, appCache)
	contactSvc := contentservice.NewContactMessageService(contactRepo, discordService)
	profileSvc := contentservice.NewProfileService(profileRepo)

	contactHandler := contenthandler.NewContactHandler(contactSvc, cfg.Admin.APIKey)
	profileHandler := contenthandler.NewProfileHandler(profileSvc)

	log.Info().Msg("content module: initialized (projects + contact messages + profile)")

	// ─────────────────────────────────────────────────────────────────────
	// 8. AI module (embeddings, RAG, ingestion)
	// ─────────────────────────────────────────────────────────────────────
	embeddingSvc, err := aiservice.NewEmbeddingService(cfg.Embedding)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize embedding service")
	}

	semanticCacheRepo := airepository.NewSemanticCacheRepository(pg.Pool)
	vectorStoreRepo := airepository.NewVectorStoreRepository(pg.Pool, cfg.Embedding.Dimensions)
	githubEmbeddingRepo := airepository.NewGitHubEmbeddingRepository(pg.Pool, cfg.Embedding.Dimensions)

	ragSvc := aiservice.NewRAGService(embeddingSvc, semanticCacheRepo, vectorStoreRepo, cfg.AI)
	ingestionSvc := aiservice.NewIngestionService(embeddingSvc, vectorStoreRepo)
	githubSyncSvc := contentservice.NewGitHubSyncService(
		cfg.GitHub,
		githubProfileRepo,
		githubRepoRepo,
		githubEmbeddingRepo,
		embeddingSvc,
		contentservice.InvalidateProjectCaches(appCache),
	)

	aiHandler := aihandler.NewAIHandler(ragSvc, ingestionSvc)
	projectHandler := contenthandler.NewProjectHandler(
		projectSvc,
		githubSyncSvc,
		githubSyncSvc.Username(),
		cfg.GitHub.ProjectsLimit,
	)

	log.Info().Msg("ai module: initialized (embeddings + RAG + ingestion)")

	// ─────────────────────────────────────────────────────────────────────
	// 9. Payment module (Razorpay webhooks + outbox)
	// ─────────────────────────────────────────────────────────────────────
	paymentRepo := paymentrepository.NewPaymentRepository(pg.Pool)
	paymentSvc := paymentservice.NewPaymentService(paymentRepo, cfg.Razorpay)
	webhookHandler := paymenthandler.NewWebhookHandler(paymentSvc)
	paymentPublicHandler := paymenthandler.NewPaymentHandler(paymentSvc)

	log.Info().Msg("payment module: initialized (Razorpay webhooks + public routes)")

	// ─────────────────────────────────────────────────────────────────────
	// 10. Admin module (dashboard stats, sponsors listing, deep health)
	// ─────────────────────────────────────────────────────────────────────
	adminH := adminhandler.NewAdminHandler(
		pg,
		projectRepo,
		githubRepoRepo,
		contactRepo,
		paymentRepo,
		vectorStoreRepo,
		semanticCacheRepo,
		githubSyncSvc,
	)

	log.Info().Msg("admin module: initialized (stats + sponsors + deep health)")

	// ─────────────────────────────────────────────────────────────────────
	// 11. Event bus subscribers
	// ─────────────────────────────────────────────────────────────────────
	bus.Subscribe("SPONSOR_CREATED", func(ctx context.Context, event eventbus.Event) error {
		log.Info().Msg("event_handler: received SPONSOR_CREATED event — sending Discord notification")

		var payload struct {
			SponsorName string  `json:"sponsorName"`
			Amount      float64 `json:"amount"`
			Currency    string  `json:"currency"`
			Email       string  `json:"email"`
			Status      string  `json:"status"`
			PaymentID   string  `json:"paymentId"`
		}

		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			log.Error().Err(err).Str("payload", event.Payload).Msg("event_handler: failed to parse SPONSOR_CREATED payload")
			return err
		}

		discordService.SendSponsorNotification(
			ctx,
			payload.SponsorName,
			payload.Email,
			payload.Currency,
			payload.Amount,
			payload.PaymentID,
			payload.Status,
		)
		return nil
	})

	log.Info().
		Int("total_handlers", bus.HandlerCount()).
		Msg("event bus: all subscribers registered")

	// ─────────────────────────────────────────────────────────────────────
	// 12. Outbox poller (background goroutine)
	// ─────────────────────────────────────────────────────────────────────
	outboxPoller := worker.NewOutboxPoller(paymentRepo, bus, cfg.Outbox.PollInterval, 50)
	outboxPoller.Start(rootCtx)
	githubSyncWorker := contentworker.NewGitHubSyncWorker(githubSyncSvc, cfg.GitHub.SyncInterval)
	githubSyncWorker.Start(rootCtx)

	log.Info().
		Dur("interval", cfg.Outbox.PollInterval).
		Msg("outbox poller: background worker started")

	// ─────────────────────────────────────────────────────────────────────
	// 13. HTTP server
	// ─────────────────────────────────────────────────────────────────────
	srv := server.New(cfg)
	srv.RegisterRoutes(server.ModuleRoutes{
		Projects:        projectHandler.Routes(),
		Profile:         profileHandler.Routes(),
		Contact:         contactHandler.Routes(),
		Chat:            aiHandler.ChatRoutes(),
		Ingest:          aiHandler.IngestRoutes(),
		RazorpayWebhook: webhookHandler.Routes(),
		Payments:        paymentPublicHandler.PaymentRoutes(),
		Sponsors:        paymentPublicHandler.SponsorRoutes(),
		Admin:           adminH.Routes(),
	})

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.Start()
	}()

	log.Info().
		Str("port", cfg.Server.Port).
		Msg("smart-portfolio: server is ready and accepting connections")

	// ─────────────────────────────────────────────────────────────────────
	// 14. Graceful shutdown
	// ─────────────────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil {
			log.Error().Err(err).Msg("server error — initiating shutdown")
		}
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	}

	rootCancel()

	log.Info().Msg("shutting down gracefully — draining in-flight work...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)

		outboxPoller.Stop()
		log.Info().Msg("shutdown: outbox poller stopped")

		bus.Shutdown()
		log.Info().Msg("shutdown: event bus drained")

		discordService.Shutdown()
		log.Info().Msg("shutdown: discord notifications drained")

		type cacheShutdowner interface {
			ShutdownCacheWorkers()
		}
		if cs, ok := ragSvc.(cacheShutdowner); ok {
			cs.ShutdownCacheWorkers()
			log.Info().Msg("shutdown: RAG cache workers drained")
		}
	}()

	select {
	case <-shutdownDone:
		log.Info().Msg("shutdown: all background services stopped cleanly")
	case <-shutdownCtx.Done():
		log.Warn().Msg("shutdown: timed out waiting for background services — forcing exit")
	}

	log.Info().Msg("smart-portfolio: shutdown complete — goodbye!")
}
