-- Smart Portfolio: Initial Migration
-- This script is idempotent — safe to run multiple times.

CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- Projects
-- =============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      VARCHAR(255) NOT NULL,
    description TEXT        NOT NULL,
    tech_stack VARCHAR(255),
    github_url VARCHAR(255),
    live_url   VARCHAR(255),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- =============================================================================
-- Contact Messages
-- =============================================================================
CREATE TABLE IF NOT EXISTS contact_messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_name  VARCHAR(100) NOT NULL,
    sender_email VARCHAR(255) NOT NULL,
    message_body TEXT         NOT NULL,
    is_read      BOOLEAN      NOT NULL DEFAULT FALSE,
    submitted_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contact_messages_unread
    ON contact_messages (submitted_at DESC)
    WHERE is_read = FALSE;

-- =============================================================================
-- Resume Embeddings (pgvector RAG store)
-- =============================================================================
CREATE TABLE IF NOT EXISTS resume_embeddings (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content   TEXT         NOT NULL,
    embedding VECTOR(768)  NOT NULL,
    metadata  JSONB
);

CREATE INDEX IF NOT EXISTS idx_resume_embeddings_hnsw
    ON resume_embeddings
    USING hnsw (embedding vector_cosine_ops);

-- =============================================================================
-- AI Semantic Cache
-- =============================================================================
CREATE TABLE IF NOT EXISTS ai_semantic_cache (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_text      TEXT         NOT NULL,
    prompt_embedding VECTOR(768)  NOT NULL,
    cached_response  TEXT         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_semantic_cache_hnsw
    ON ai_semantic_cache
    USING hnsw (prompt_embedding vector_cosine_ops);

-- =============================================================================
-- Sponsors (Razorpay payments)
-- =============================================================================
CREATE TABLE IF NOT EXISTS sponsors (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sponsor_name        VARCHAR(255)   NOT NULL,
    email               VARCHAR(255)   NOT NULL,
    amount              DECIMAL(10, 2) NOT NULL,
    currency            VARCHAR(3)     NOT NULL DEFAULT 'INR',
    status              VARCHAR(50)    NOT NULL DEFAULT 'PENDING',
    razorpay_payment_id VARCHAR(255)   UNIQUE,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sponsors_email
    ON sponsors (email);

-- =============================================================================
-- Outbox Events (transactional outbox pattern)
-- =============================================================================
CREATE TABLE IF NOT EXISTS outbox_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id   VARCHAR(255) NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    payload        JSONB        NOT NULL,
    is_processed   BOOLEAN      NOT NULL DEFAULT FALSE,
    event_id       VARCHAR(255) UNIQUE NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_pending
    ON outbox_events (created_at ASC)
    WHERE is_processed = FALSE;
