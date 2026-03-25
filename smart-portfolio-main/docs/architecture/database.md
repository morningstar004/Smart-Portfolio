# Database

Smart Portfolio uses PostgreSQL 16 with the `pgvector` extension for storing and querying high-dimensional embedding vectors alongside relational data.

---

## Schema Overview

All tables are created by `migrations/001_init.sql`, which runs automatically on server startup. The migration is fully idempotent — every statement uses `IF NOT EXISTS`, so re-running it on an existing database is safe.

| Table | Purpose |
|-------|---------|
| `projects` | Portfolio project entries |
| `contact_messages` | Contact form submissions |
| `resume_embeddings` | Chunked resume text with vector embeddings for RAG |
| `ai_semantic_cache` | Cached Q&A pairs for semantic deduplication |
| `sponsors` | Successful Razorpay payment records |
| `outbox_events` | Pending events for the transactional outbox |

---

## Table Definitions

### `projects`

```sql
CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    tech_stack  TEXT,
    github_url  TEXT,
    live_url    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### `contact_messages`

```sql
CREATE TABLE IF NOT EXISTS contact_messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_name  TEXT NOT NULL,
    sender_email TEXT NOT NULL,
    message_body TEXT NOT NULL,
    is_read      BOOLEAN NOT NULL DEFAULT FALSE,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial index: only unread rows — speeds up GET /api/contact/unread
CREATE INDEX IF NOT EXISTS idx_contact_messages_unread
    ON contact_messages(submitted_at DESC)
    WHERE is_read = FALSE;
```

### `resume_embeddings`

```sql
CREATE TABLE IF NOT EXISTS resume_embeddings (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content   TEXT NOT NULL,
    embedding VECTOR(768) NOT NULL,
    metadata  JSONB
);

-- HNSW index for fast approximate nearest-neighbor search
CREATE INDEX IF NOT EXISTS idx_resume_embeddings_hnsw
    ON resume_embeddings USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

### `ai_semantic_cache`

```sql
CREATE TABLE IF NOT EXISTS ai_semantic_cache (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_text      TEXT NOT NULL,
    prompt_embedding VECTOR(768) NOT NULL,
    cached_response  TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW index for semantic cache lookups
CREATE INDEX IF NOT EXISTS idx_ai_semantic_cache_hnsw
    ON ai_semantic_cache USING hnsw (prompt_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

### `sponsors`

```sql
CREATE TABLE IF NOT EXISTS sponsors (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sponsor_name         TEXT NOT NULL,
    email                TEXT NOT NULL,
    amount               DECIMAL(10, 2) NOT NULL,
    currency             TEXT NOT NULL DEFAULT 'INR',
    status               TEXT NOT NULL DEFAULT 'SUCCESS',
    razorpay_payment_id  TEXT NOT NULL UNIQUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sponsors_email ON sponsors(email);
```

The `razorpay_payment_id UNIQUE` constraint is the primary idempotency guard. Duplicate webhook deliveries for the same payment are rejected at the database level.

### `outbox_events`

```sql
CREATE TABLE IF NOT EXISTS outbox_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        JSONB NOT NULL,
    is_processed   BOOLEAN NOT NULL DEFAULT FALSE,
    event_id       TEXT NOT NULL UNIQUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial index: only unprocessed events — keeps the index small
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
    ON outbox_events(created_at ASC)
    WHERE is_processed = FALSE;
```

---

## pgvector

The `vector` extension adds the `VECTOR(n)` column type and vector-specific operators.

**Operator used:**

| Operator | Meaning |
|----------|---------|
| `<=>` | Cosine distance (0 = identical, 2 = opposite) |

**Example queries:**

```sql
-- RAG retrieval: top 3 most similar chunks to a query vector
SELECT content, metadata, (embedding <=> $1) AS distance
FROM resume_embeddings
ORDER BY embedding <=> $1
LIMIT 3;

-- Semantic cache lookup: return cached answer if close enough
SELECT cached_response FROM ai_semantic_cache
WHERE prompt_embedding <=> $1 < 0.05
ORDER BY prompt_embedding <=> $1
LIMIT 1;
```

**HNSW index parameters:**

| Parameter | Value | Effect |
|-----------|-------|--------|
| `m` | 16 | Maximum connections per node in the graph. Higher = better recall, more memory. |
| `ef_construction` | 64 | Search range during index build. Higher = better index quality, slower build. |

These are the pgvector defaults and are well-suited for a personal portfolio with hundreds of embeddings.

---

## Transactional Outbox Pattern

The payment module uses the transactional outbox to guarantee that every successful payment triggers a Discord notification, even if the server crashes after the insert.

**The problem without it:**

```
1. INSERT sponsor → OK
2. Server crash
3. Discord notification → never sent
```

**With the outbox:**

```
BEGIN TRANSACTION
  INSERT INTO sponsors (...)
  INSERT INTO outbox_events (...)  ← atomic with sponsor insert
COMMIT

Background poller:
  SELECT * FROM outbox_events WHERE is_processed = FALSE
  → publish event to event bus
  → Discord handler fires in goroutine
  → UPDATE outbox_events SET is_processed = TRUE WHERE id = ...
```

Both rows commit atomically. If the server crashes before the poller runs, the outbox event persists and is processed on the next startup. This gives at-least-once delivery semantics.

---

## Migrations

Migrations are run by the Go server at startup via `database.RunMigrations()`:

1. Reads all `.sql` files from `migrations/` in lexicographic order.
2. Executes each file in its own database transaction.
3. Idempotent SQL ensures re-runs are safe.

To run manually with psql:

```bash
psql "$DATABASE_URL" -f backend/migrations/001_init.sql
```

---

## Connection Pool

The application uses `pgxpool` (pgx v5):

| Config | Variable | Default |
|--------|----------|---------|
| Max open connections | `DB_MAX_OPEN_CONNS` | 10 |
| Connection timeout | hardcoded | 5 seconds |

The pool is closed as the last step of graceful shutdown, after all application goroutines have finished their database operations.
