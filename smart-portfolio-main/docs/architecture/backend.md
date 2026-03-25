# Backend Architecture

The backend is a Go REST API that follows a layered, modular monolith architecture. All application code lives under `backend/internal/`.

---

## Directory Structure

```
backend/
├── cmd/server/
│   └── main.go                Entry point: dependency wiring, graceful shutdown
├── docs/
│   ├── openapi.yaml           Full OpenAPI 3.0 specification
│   └── swagger.go             Embedded Swagger UI served at /docs
├── internal/
│   ├── config/                Configuration loading from environment
│   ├── database/              PostgreSQL connection pool and migrations
│   ├── httputil/              Response envelope helpers and typed errors
│   ├── middleware/            HTTP middleware stack
│   ├── server/                chi router setup and route registration
│   ├── modules/               Domain modules (ai, content, payment, admin, notification)
│   └── platform/              Shared infrastructure (cache, event bus)
└── migrations/
    └── 001_init.sql           Idempotent database schema
```

---

## Middleware Stack

Every HTTP request passes through 6 middleware layers in order:

| Order | Middleware | What It Does |
|-------|-----------|-------------|
| 1 | `Recoverer` | Catches panics and returns 500 without crashing the server |
| 2 | `RequestID` | Assigns a UUID to every request; added to response headers and log context |
| 3 | `RequestLogger` | Logs method, path, status, and duration using zerolog; DEBUG for 2xx/3xx, WARN for 4xx, ERROR for 5xx |
| 4 | `SecurityHeaders` | Sets OWASP-recommended headers: `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`, `Content-Security-Policy` |
| 5 | `RateLimiter` | Per-IP sliding window; returns 429 when exceeded (`RATE_LIMIT_RPS`, default: 10 req/s) |
| 6 | `CORS` | Allows the configured `FRONTEND_URL` origin (and localhost dev origins) |

The `AdminAuth` middleware is applied per-route only to admin-protected endpoints, not globally.

---

## Entry Point and Dependency Wiring

`cmd/server/main.go` performs 14 sequential initialization steps:

1. Configure zerolog (pretty console in development, JSON in production)
2. Load configuration from environment / `.env` file
3. Create root context with SIGINT/SIGTERM cancellation
4. Connect PostgreSQL pool and run migrations
5. Initialize in-memory cache (`go-cache`) and event bus
6. Initialize Discord notification service
7. Wire Content module (projects + contact)
8. Wire AI module (embeddings + RAG + ingestion)
9. Wire Payment module (Razorpay webhook)
10. Wire Admin module (stats + health)
11. Subscribe `SPONSOR_CREATED` event handler to Discord notifications
12. Start outbox poller background goroutine
13. Start HTTP server
14. Wait for shutdown signal

---

## Graceful Shutdown

On SIGINT or SIGTERM, the server coordinates a 6-step ordered drain with a 30-second timeout:

1. HTTP server stops accepting new connections and drains in-flight requests
2. Outbox poller background goroutine is stopped
3. Event bus is shut down (waits for all in-flight handler goroutines)
4. Discord notification service is shut down (waits for all in-flight webhook goroutines)
5. RAG service waits for background semantic cache save goroutines
6. PostgreSQL connection pool is closed

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `GROQ_API_KEY` | Yes | — | Groq LLM API key |
| `JINA_API_KEY` | Yes | — | Jina Embeddings API key |
| `SERVER_PORT` | No | `8080` | HTTP listen port |
| `ENV` | No | `development` | `production` enables JSON log format |
| `LOG_LEVEL` | No | `debug` | Zerolog log level |
| `DB_MAX_OPEN_CONNS` | No | `10` | PostgreSQL connection pool size |
| `AI_MODEL` | No | `llama-3.3-70b-versatile` | Groq model name |
| `AI_TEMPERATURE` | No | `0.3` | LLM temperature (0.0–1.0) |
| `EMBEDDING_MODEL` | No | `jina-embeddings-v2-base-en` | Jina embedding model |
| `EMBEDDING_DIMENSIONS` | No | `768` | Vector dimensions |
| `DISCORD_WEBHOOK_URL` | No | empty | Discord webhook for notifications |
| `RAZORPAY_KEY_ID` | No | — | Razorpay pubkey |
| `RAZORPAY_KEY_SECRET` | No | — | Razorpay secret |
| `RAZORPAY_WEBHOOK_SECRET` | No | — | Razorpay webhook HMAC secret |
| `FRONTEND_URL` | No | `http://localhost:5173` | Allowed CORS origin in production |
| `RATE_LIMIT_RPS` | No | `10` | Requests per second per IP |
| `OUTBOX_POLL_INTERVAL` | No | `10` | Seconds between outbox polls |
| `CACHE_TTL_HOURS` | No | `24` | In-memory cache TTL |
| `ADMIN_API_KEY` | No | empty | Admin route authentication key |

---

## Response Envelope

Every endpoint returns the same JSON shape:

```json
// Success
{"success": true, "data": { ... }}

// Error
{"success": false, "error": {"code": 404, "message": "project not found"}}
```

Typed errors (`ErrNotFound`, `ErrValidation`) are used internally. `HandleServiceError()` in `httputil` maps them to the correct HTTP status codes. This keeps HTTP concerns out of service and repository layers.

---

## Logging

Zerolog provides structured, zero-allocation logging. Every log entry is correlated with a request ID. In development (`ENV=development`), logs are formatted for human readability. In production, they output as JSON for ingestion by log aggregators.

Log levels by HTTP status:
- **2xx / 3xx** → `DEBUG`
- **4xx** → `WARN`
- **5xx** → `ERROR`
