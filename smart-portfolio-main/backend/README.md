# Smart Portfolio Backend

A fast, modular, and production-ready portfolio backend written in Go. Powers a personal developer portfolio with AI-powered resume chat (RAG), contact form with Discord notifications, project showcase management, and Razorpay sponsorship payments via a transactional outbox pattern.

## Architecture

```text
cmd/server/main.go              ← Entry point & dependency wiring

internal/
  config/                        ← Environment-based configuration loader
  database/                      ← PostgreSQL pool + migration runner
  httputil/                      ← JSON envelope, error types, ParseUUID
  middleware/                    ← Logging, recovery, rate-limit, security, admin auth
  server/                        ← HTTP server, chi router, route registration

  modules/
    admin/handler/               ← Dashboard stats, sponsors listing, deep health check
    ai/
      dto/                       ← ChatRequest, ChatResponse, IngestResponse
      handler/                   ← /api/chat and /api/ingest HTTP handlers
      repository/                ← SemanticCacheRepo, VectorStoreRepo (pgvector)
      service/                   ← EmbeddingService (Jina), RAGService (Groq), IngestionService
    content/
      dto/                       ← ProjectRequest/Response, ContactMessageRequest/Response
      handler/                   ← /api/projects and /api/contact HTTP handlers
      model/                     ← Project, ContactMessage DB models
      repository/                ← ProjectRepo, ContactRepo (pgx)
      service/                   ← ProjectService (cached), ContactMessageService (Discord)
    notification/service/        ← NotificationService interface, Discord implementation
    payment/
      dto/                       ← SponsorResponse, DashboardStatsResponse
      handler/                   ← Razorpay webhook with HMAC-SHA256 verification
      model/                     ← Sponsor, OutboxEvent DB models
      repository/                ← Transactional sponsor + outbox insert
      service/                   ← Signature verification, event parsing
      worker/                    ← OutboxPoller background goroutine

  platform/
    cache/                       ← In-memory TTL cache (go-cache wrapper)
    eventbus/                    ← In-process async event bus with goroutine dispatch

docs/
  openapi.yaml                   ← OpenAPI 3.0 specification
  swagger.go                     ← Embedded Swagger UI handler

migrations/
  001_init.sql                   ← Idempotent schema for all tables
```

## Quick Start

### Prerequisites

- Go 1.26+
- PostgreSQL 15+ with the [pgvector](https://github.com/pgvector/pgvector) extension
- A [Groq](https://console.groq.com/) API key (free tier)
- A [Jina](https://jina.ai/) embeddings API key (free tier)

### Option 1: Docker Compose (Recommended)

```bash
cp .env.example .env
# Edit .env — at minimum set GROQ_API_KEY and JINA_API_KEY

docker compose up -d --build
# Server: http://localhost:8080
# API docs: http://localhost:8080/docs

docker compose logs -f app     # Tail logs
docker compose down            # Stop
```

### Option 2: Local Development

```bash
cp .env.example .env
# Fill in DATABASE_URL, GROQ_API_KEY, JINA_API_KEY
# Also set FRONTEND_URL to your frontend origin and ADMIN_API_KEY for admin protection

go mod tidy
make run          # Build and run
# OR
make dev          # Live reload with Air
```

### Verify

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/api/projects
```

## API Documentation

Interactive Swagger UI is available at **[/docs](http://localhost:8080/docs)** when the server is running.

The raw OpenAPI 3.0 spec lives at [`docs/openapi.yaml`](docs/openapi.yaml).

### Endpoint Summary

#### Public

| Method   | Path                     | Description                    |
|----------|--------------------------|--------------------------------|
| `GET`    | `/healthz`               | Liveness probe                 |
| `GET`    | `/api/projects`          | List all projects              |
| `GET`    | `/api/projects/{id}`     | Get project by ID              |
| `POST`   | `/api/projects`          | Create a project               |
| `PUT`    | `/api/projects/{id}`     | Update a project               |
| `DELETE` | `/api/projects/{id}`     | Delete a project               |
| `POST`   | `/api/contact`           | Submit a contact message       |
| `POST`   | `/api/chat`              | Ask AI (JSON response)         |
| `POST`   | `/api/chat/stream`       | Ask AI (SSE streaming)         |

#### Admin-Protected

Requires `X-Admin-Key` header or `Authorization: Bearer <key>`.

| Method   | Path                        | Description                    |
|----------|-----------------------------|--------------------------------|
| `GET`    | `/api/contact`              | List all contact messages      |
| `GET`    | `/api/contact/unread`       | List unread messages           |
| `PATCH`  | `/api/contact/{id}/read`    | Mark message as read           |
| `DELETE` | `/api/contact/{id}`         | Delete a contact message       |
| `POST`   | `/api/ingest`               | Upload PDF for RAG             |
| `POST`   | `/api/ingest/text`          | Ingest raw text for RAG        |
| `DELETE` | `/api/ingest`               | Clear vector store             |
| `GET`    | `/api/admin/health`         | Deep health check (DB ping)    |
| `GET`    | `/api/admin/stats`          | Dashboard statistics           |
| `GET`    | `/api/admin/sponsors`       | List all sponsors              |

#### Webhook

| Method   | Path                        | Auth            |
|----------|-----------------------------|-----------------|
| `POST`   | `/api/webhooks/razorpay`    | HMAC-SHA256     |

### Response Envelope

Every response follows a consistent JSON envelope:

```json
{ "success": true, "data": { ... } }
```

```json
{ "success": false, "error": { "code": 400, "message": "title is required" } }
```

## Environment Variables

See [`.env.example`](.env.example) for the full annotated list.

| Variable                  | Required | Default                                 |
|---------------------------|----------|-----------------------------------------|
| `DATABASE_URL`            | Yes      | —                                       |
| `GROQ_API_KEY`            | Yes      | —                                       |
| `JINA_API_KEY`            | Yes      | —                                       |
| `SERVER_PORT`             | No       | `8080`                                  |
| `ADMIN_API_KEY`           | No       | *(empty = no auth)*                     |
| `DISCORD_WEBHOOK_URL`     | No       | *(empty = disabled)*                    |
| `RAZORPAY_KEY_ID`         | No       | —                                       |
| `RAZORPAY_KEY_SECRET`     | No       | —                                       |
| `RAZORPAY_WEBHOOK_SECRET` | No       | —                                       |
| `FRONTEND_URL`            | No       | `http://localhost:5173`                 |
| `RATE_LIMIT_RPS`          | No       | `10`                                    |

### Production Minimum

For production, set at least:

```env
ENV=production
LOG_LEVEL=info
SERVER_PORT=8080
DATABASE_URL=postgres://user:password@host:5432/dbname?sslmode=require
GROQ_API_KEY=...
JINA_API_KEY=...
FRONTEND_URL=https://your-frontend-domain.example
ADMIN_API_KEY=generate-a-long-random-secret
```

## Development

### Makefile Targets

```bash
make              # Build binary
make run          # Build and run
make dev          # Live reload (requires Air)
make test         # Run all tests
make test-v       # Verbose tests
make cover        # Tests with coverage report
make bench        # Benchmarks
make lint         # go vet + staticcheck
make fmt          # Format source files
make tidy         # Tidy go modules
make check        # Full pre-commit suite (fmt + vet + lint + test)
make docker-up    # Start docker compose stack
make docker-down  # Stop stack
make clean        # Remove build artifacts
make help         # Show all targets
```

### Running Tests

```bash
make test                                              # All tests
make test-v                                            # Verbose
make cover                                             # Coverage HTML report
go test ./internal/platform/eventbus/... -v -run TestPublish  # Specific test
```

### Project Conventions

- **Module structure** — Each domain module has `dto/`, `handler/`, `model/`, `repository/`, `service/` sub-packages.
- **Interfaces first** — Services expose interfaces at the top of each file. Swap implementations without touching callers.
- **Typed errors** — `ErrNotFound` and `ErrValidation` in `httputil` are matched with `errors.As`. Handlers use `HandleServiceError()` for consistent routing.
- **No import cycles** — The server package accepts `chi.Router` interfaces; modules never import each other.

## Key Design Decisions

### Concurrency

- **Embedding batches** — PDF chunks are embedded in parallel (up to 4 concurrent Jina API calls) via goroutines with a semaphore channel.
- **Discord notifications** — Every webhook call runs in its own goroutine, tracked by `sync.WaitGroup`.
- **Semantic cache writes** — Saved to DB in a background goroutine after the LLM stream completes.
- **Event bus dispatch** — Each handler invoked in its own goroutine with panic recovery.
- **Outbox poller** — Single background goroutine with `time.Ticker`.

### Transactional Outbox

Sponsor payments use the transactional outbox pattern: the sponsor row and its outbox event are committed in a single PostgreSQL transaction. A background poller relays events to the in-process event bus, which dispatches a Discord notification asynchronously. This guarantees at-least-once delivery even if the app crashes.

### Graceful Shutdown

`SIGINT`/`SIGTERM` triggers an orderly drain:

1. HTTP server — stop accepting, drain in-flight requests
2. Outbox poller — finish current cycle
3. Event bus — wait for in-flight handlers
4. Discord notifications — wait for webhook calls
5. RAG cache workers — flush background saves
6. Database pool — close connections

## Docker

### Multi-stage Build

```bash
docker build -t smart-portfolio .
```

Produces a minimal Alpine image (~11 MB) with non-root user, built-in health check, CA certificates, and bundled migrations.

### Compose Stack

Includes the Go app container:

```bash
docker compose up -d --build
```

## License

[MIT](../LICENSE)
