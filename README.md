# Smart Portfolio

A full-stack personal developer portfolio with AI-powered resume chat, project showcase, contact form, and sponsorship payments.

## Project Structure

```text
smart-portfolio/
├── backend/          ← Go REST API (chi, pgx, pgvector, zerolog)
├── frontend/         ← Frontend application (see frontend/README.md)
└── README.md         ← You are here
```

## Features

| Feature | Description |
|---------|-------------|
| **AI Resume Chat (RAG)** | Upload a PDF resume → chunked, embedded via Jina, stored in pgvector. Visitors ask questions and get LLM-powered answers (Groq) with SSE streaming. Semantic cache avoids redundant LLM calls. |
| **Project Showcase** | Full CRUD for portfolio projects with in-memory caching (24h TTL, auto-invalidation on writes). |
| **Contact Form** | Visitors submit messages; persisted to PostgreSQL with async Discord notifications. Admin can list, mark as read, and delete. |
| **Sponsorship Payments** | Razorpay webhooks with HMAC-SHA256 verification. Transactional outbox pattern guarantees reliable event delivery + Discord alerts. |
| **Admin Dashboard API** | Aggregate stats, sponsor listing, deep health check — all behind API key authentication. |
| **Interactive API Docs** | Swagger UI served at `/docs` with a full OpenAPI 3.0 spec. |

## Tech Stack

### Backend

- **Language:** Go 1.26
- **Router:** chi/v5
- **Database:** PostgreSQL 16 + pgvector (pgx/v5 driver)
- **AI:** Groq (LLM, OpenAI-compatible) + Jina (embeddings)
- **Logging:** zerolog (structured, request-ID correlated)
- **Payments:** Razorpay webhooks
- **Notifications:** Discord webhooks
- **Caching:** go-cache (in-memory, TTL-based)
- **Docs:** OpenAPI 3.0 + embedded Swagger UI

### Frontend

The frontend directory is ready for any modern framework. See [`frontend/README.md`](frontend/README.md) for a detailed integration guide with TypeScript types, SSE streaming helpers, and deployment architecture.

## Quick Start

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- A [Groq API key](https://console.groq.com/) (free tier)
- A [Jina API key](https://jina.ai/) (free tier)

### One-Command Setup

```bash
cd backend
cp .env.example .env
# Edit .env — set GROQ_API_KEY and JINA_API_KEY at minimum

docker compose up -d --build
```

The backend starts at **http://localhost:8080**.

| URL | Description |
|-----|-------------|
| `http://localhost:8080/healthz` | Health check |
| `http://localhost:8080/docs` | Swagger UI (API docs) |
| `http://localhost:8080/api/projects` | Projects API |
| `http://localhost:8080/api/chat` | AI Chat API |

### Local Development (without Docker)

```bash
cd backend
cp .env.example .env
# Fill in DATABASE_URL, GROQ_API_KEY, JINA_API_KEY

go mod tidy
make run          # Build and run
# OR
make dev          # Live reload with Air
```

## API Overview

Full interactive documentation is available at **[/docs](http://localhost:8080/docs)** when the backend is running.

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/api/projects` | List all projects |
| `GET` | `/api/projects/{id}` | Get project by ID |
| `POST` | `/api/projects` | Create a project |
| `PUT` | `/api/projects/{id}` | Update a project |
| `DELETE` | `/api/projects/{id}` | Delete a project |
| `POST` | `/api/contact` | Submit a contact message |
| `POST` | `/api/chat` | Ask AI (JSON response) |
| `POST` | `/api/chat/stream` | Ask AI (SSE streaming) |

### Admin Endpoints

Protected by `X-Admin-Key` header or `Authorization: Bearer <key>`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/health` | Deep health check (DB ping) |
| `GET` | `/api/admin/stats` | Dashboard statistics |
| `GET` | `/api/admin/sponsors` | List all sponsors |
| `POST` | `/api/ingest` | Upload PDF for RAG |
| `POST` | `/api/ingest/text` | Ingest raw text for RAG |
| `DELETE` | `/api/ingest` | Clear vector store |
| `GET` | `/api/contact` | List all messages |
| `PATCH` | `/api/contact/{id}/read` | Mark message as read |
| `DELETE` | `/api/contact/{id}` | Delete a message |

### Webhook

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/api/webhooks/razorpay` | HMAC-SHA256 |

## Environment Variables

See [`backend/.env.example`](backend/.env.example) for the full annotated list. Key variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string (must have pgvector) |
| `GROQ_API_KEY` | Yes | Groq API key for LLM chat completions |
| `JINA_API_KEY` | Yes | Jina API key for text embeddings |
| `ADMIN_API_KEY` | No | Secret for admin endpoints (empty = no auth) |
| `DISCORD_WEBHOOK_URL` | No | Discord webhook for notifications |
| `FRONTEND_URL` | No | Frontend origin for CORS (default: `http://localhost:5173`) |

## Deployment

```text
┌─────────────────────────────────────────────┐
│           Frontend Host (Vercel)            │
│  Static + SSR pages, API route proxying     │
└──────────────────┬──────────────────────────┘
                   │ HTTPS
                   ▼
┌─────────────────────────────────────────────┐
│     Backend Host (Railway / Render / VPS)   │
│  Go Docker image (~11 MB), /api/*, /docs    │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│       PostgreSQL + pgvector                 │
│       (Neon / Supabase / Railway)           │
└─────────────────────────────────────────────┘
```

## License

[MIT](./LICENSE)