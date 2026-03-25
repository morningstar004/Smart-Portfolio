package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	AI        AIConfig
	Embedding EmbeddingConfig
	Discord   DiscordConfig
	Razorpay  RazorpayConfig
	Frontend  FrontendConfig
	RateLimit RateLimitConfig
	Outbox    OutboxConfig
	Cache     CacheConfig
	Admin     AdminConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AIConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float32
}

type EmbeddingConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
}

type DiscordConfig struct {
	WebhookURL string
}

type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	WebhookSecret string
}

type FrontendConfig struct {
	URL string
}

type RateLimitConfig struct {
	RequestsPerSecond int
	Burst             int
}

type OutboxConfig struct {
	PollInterval time.Duration
}

type CacheConfig struct {
	TTL      time.Duration
	MaxItems int
}

type AdminConfig struct {
	// APIKey is the secret key required to access admin endpoints (contact
	// message management, sponsor listing, stats, ingestion, etc.). If empty,
	// admin endpoints are accessible without authentication — suitable for
	// local development but NOT for production.
	APIKey string
}

// Load reads the .env file (if present) and populates the Config struct.
// It returns an error if any required variable is missing.
func Load() (*Config, error) {
	// Attempt to load .env; it's fine if the file doesn't exist (e.g. in prod).
	_ = godotenv.Load()

	cfg := &Config{}
	var errs []string

	// ── Server ───────────────────────────────────────────────────────────
	cfg.Server.Port = envOrDefault("PORT", envOrDefault("SERVER_PORT", "8080"))

	// ── Database ─────────────────────────────────────────────────────────
	cfg.Database.URL = requireEnv("DATABASE_URL", &errs)
	cfg.Database.MaxOpenConns = envIntOrDefault("DB_MAX_OPEN_CONNS", 10)
	cfg.Database.MaxIdleConns = envIntOrDefault("DB_MAX_IDLE_CONNS", 5)
	cfg.Database.ConnMaxLifetime = time.Duration(envIntOrDefault("DB_CONN_MAX_LIFETIME_MIN", 30)) * time.Minute

	// ── AI (Groq / OpenAI-compatible) ────────────────────────────────────
	cfg.AI.APIKey = requireEnv("GROQ_API_KEY", &errs)
	cfg.AI.BaseURL = envOrDefault("GROQ_BASE_URL", "https://api.groq.com/openai/v1")
	cfg.AI.Model = envOrDefault("AI_MODEL", "llama-3.3-70b-versatile")
	cfg.AI.Temperature = float32(envFloatOrDefault("AI_TEMPERATURE", 0.3))

	// ── Embedding (Jina) ─────────────────────────────────────────────────
	cfg.Embedding.APIKey = requireEnv("JINA_API_KEY", &errs)
	cfg.Embedding.BaseURL = envOrDefault("JINA_BASE_URL", "https://api.jina.ai/v1")
	cfg.Embedding.Model = envOrDefault("EMBEDDING_MODEL", "jina-embeddings-v2-base-en")
	cfg.Embedding.Dimensions = envIntOrDefault("EMBEDDING_DIMENSIONS", 768)

	// ── Discord ──────────────────────────────────────────────────────────
	cfg.Discord.WebhookURL = envOrDefault("DISCORD_WEBHOOK_URL", "")

	// ── Razorpay ─────────────────────────────────────────────────────────
	cfg.Razorpay.KeyID = envOrDefault("RAZORPAY_KEY_ID", "")
	cfg.Razorpay.KeySecret = envOrDefault("RAZORPAY_KEY_SECRET", "")
	cfg.Razorpay.WebhookSecret = envOrDefault("RAZORPAY_WEBHOOK_SECRET", "")

	// ── Frontend ─────────────────────────────────────────────────────────
	cfg.Frontend.URL = envOrDefault("FRONTEND_URL", "http://localhost:5173")

	// ── Rate Limiting ────────────────────────────────────────────────────
	cfg.RateLimit.RequestsPerSecond = envIntOrDefault("RATE_LIMIT_RPS", 10)
	cfg.RateLimit.Burst = envIntOrDefault("RATE_LIMIT_BURST", 20)

	// ── Outbox ───────────────────────────────────────────────────────────
	cfg.Outbox.PollInterval = time.Duration(envIntOrDefault("OUTBOX_POLL_INTERVAL", 10)) * time.Second

	// ── Cache ────────────────────────────────────────────────────────────
	cfg.Cache.TTL = time.Duration(envIntOrDefault("CACHE_TTL_HOURS", 24)) * time.Hour
	cfg.Cache.MaxItems = envIntOrDefault("CACHE_MAX_ITEMS", 100)

	// ── Admin ────────────────────────────────────────────────────────────
	cfg.Admin.APIKey = envOrDefault("ADMIN_API_KEY", "")

	if len(errs) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", errs)
	}

	return cfg, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func requireEnv(key string, errs *[]string) string {
	v := os.Getenv(key)
	if v == "" {
		*errs = append(*errs, key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloatOrDefault(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
