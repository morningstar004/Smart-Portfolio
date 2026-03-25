# Modules

Each domain module lives under `internal/modules/` and follows the same layered structure. Modules do not import each other — they communicate only through the central event bus or by the server passing `chi.Router` sub-routers.

---

## Module Layout

Every module uses this directory layout:

```
modules/<name>/
├── dto/          Data Transfer Objects: request/response structs and validation
├── handler/      HTTP handler functions (parse request → call service → write response)
├── model/        Database model structs (map to tables)
├── repository/   Database access (pgx queries)
└── service/      Business logic and orchestration (interface + implementation)
```

---

## Content Module

**Location:** `internal/modules/content/`

Manages portfolio projects and contact form messages.

### Projects

`ProjectService` wraps `ProjectRepository` with an in-memory cache:

- **Cache key:** `projects:all` — populated on the first `GetAllProjects` call, valid for `CACHE_TTL_HOURS`.
- **Cache invalidation:** Any write operation (Create, Update, Delete) calls `cache.DeleteByPrefix("projects:")`.
- **Concurrency:** Read path never hits the database on cache hits; write path always goes to the database.

### Contact Messages

`ContactMessageService` persists messages directly to PostgreSQL. After a successful insert, it calls `notifier.SendContactNotification()` in a background goroutine so the HTTP response is never delayed by the Discord API.

**Validation rules:**

| Field | Rule |
|-------|------|
| `sender_name` | Non-empty string |
| `sender_email` | Valid RFC 5322 email |
| `message_body` | Non-empty string |
| `title` (Project) | Non-empty string |
| `description` (Project) | Non-empty string |

---

## AI Module

**Location:** `internal/modules/ai/`

Implements the full RAG pipeline for AI-powered resume chat.

### Embedding Service

Wraps the Jina AI HTTP API (OpenAI-compatible format):

- `Embed(text string)` — embeds a single string, returns `[]float32` (768 dimensions)
- `EmbedBatch(texts []string)` — embeds multiple strings in one API call
- Uses `sync.Pool` for HTTP request buffers to reduce allocations

### Ingestion Service

Full pipeline: PDF/text → chunks → embeddings → pgvector store

- PDF text extraction via `ledongthuc/pdf` (no CGO required)
- Whitespace normalization (collapses multiple spaces/newlines)
- Chunking: 800-character chunks with 200-character overlap, splits at word boundaries
- Parallel embedding: up to 4 concurrent goroutines, each sending batches of 32 to Jina
- Atomic write: all vectors inserted in a single transaction

### RAG Service

The question-answering pipeline:

```
Question
   │
   ▼
Jina Embed (768d vector)
   │
   ▼
pgvector cosine search on ai_semantic_cache
   │
   ├─ cache hit (distance < 0.05) ──→ return cached answer
   │
   ▼
pgvector cosine search on resume_embeddings (top 3)
   │
   ▼
Build system prompt with retrieved chunks
   │
   ▼
Groq LLM inference (llama-3.3-70b-versatile)
   │
   ├──→ JSON response / SSE stream to client
   │
   ▼
Save Q+A to ai_semantic_cache (async goroutine)
```

`sync.Pool` is used for LLM request buffers. `sync.WaitGroup` tracks all async cache-save goroutines for graceful shutdown.

---

## Payment Module

**Location:** `internal/modules/payment/`

Handles Razorpay payment webhooks with the transactional outbox pattern.

### Webhook Handler

1. Reads raw body (before JSON parsing, for HMAC accuracy)
2. Verifies `X-Razorpay-Signature` with HMAC-SHA256 (constant-time comparison via `hmac.Equal`)
3. Routes `payment.captured` events to the payment service
4. Returns 200 for any other event type

### Payment Service

- Parses the Razorpay payload, extracts sponsor name/email/amount/currency
- Converts paise → rupees (or smallest unit → major unit)
- Calls `ProcessSponsorshipTx()` which performs an atomic DB transaction

### Payment Repository: Transactional Insert

```sql
BEGIN;
  INSERT INTO sponsors (...) VALUES (...);
  INSERT INTO outbox_events (...) VALUES (...);
COMMIT;
```

If the `razorpay_payment_id` already exists (UNIQUE constraint), the service returns a `DuplicateEventError` and the handler responds 200 (duplicate webhooks are safe to ignore).

### Outbox Poller

Background goroutine using `time.Ticker`:
1. Polls `outbox_events` for unprocessed rows
2. For each event: publishes to the in-process event bus
3. Marks the event as processed
4. Runs one poll immediately on startup (no wait for first tick)

Shutdown is coordinated via context cancellation and `sync.Once`.

---

## Admin Module

**Location:** `internal/modules/admin/`

Provides read-only aggregation for the admin dashboard.

| Handler | What It Queries |
|---------|----------------|
| `DeepHealthCheck` | Pings the database, measures round-trip latency |
| `DashboardStats` | Counts from all 6 tables (projects, contact_messages, sponsors, resume_embeddings, ai_semantic_cache, outbox_events) |
| `ListSponsors` | All rows from `sponsors` ordered by `created_at DESC` |

---

## Notification Module

**Location:** `internal/modules/notification/`

`NotificationService` interface with two implementations (only one is wired: Discord).

All send methods are fire-and-forget goroutines tracked by a `sync.WaitGroup`. `Shutdown()` blocks until all in-flight Discord webhook requests complete, ensuring no notifications are dropped during graceful shutdown.

```go
type NotificationService interface {
    SendContactNotification(msg ContactMessage) error
    SendSponsorNotification(sponsor Sponsor) error
    SendRaw(content string) error
    Shutdown()
}
```

---

## Platform Layer

**Location:** `internal/platform/`

### In-Memory Cache (`platform/cache`)

Wraps `patrickmn/go-cache` with a typed API:

- `Get(key) (any, bool)`
- `GetString(key) (string, bool)`
- `Set(key, value)` — uses default TTL from config
- `SetWithTTL(key, value, duration)`
- `Delete(key)`
- `DeleteByPrefix(prefix)` — used for cache invalidation patterns
- `Flush()`, `ItemCount()`, `Keys()`

### Event Bus (`platform/eventbus`)

In-process async event bus. This decouples the payment module from the notification module.

```go
bus.Subscribe("SPONSOR_CREATED", func(event Event) {
    // runs in its own goroutine
    notifier.SendSponsorNotification(...)
})

bus.Publish(Event{Type: "SPONSOR_CREATED", Payload: sponsor})
```

- Each `Publish` call dispatches all subscribers concurrently in separate goroutines
- Every handler is wrapped in a panic recovery block — one bad handler cannot crash others
- `Shutdown()` cancels the bus context and waits for all in-flight handlers via `sync.WaitGroup`
