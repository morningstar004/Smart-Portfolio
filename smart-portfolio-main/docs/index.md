---
hide:
  - navigation
  - toc
---

<div class="hero" markdown>

# Smart Portfolio

**Ship your developer portfolio with an AI brain.** Resume chat, project showcase, contact form, and sponsor payments — all in one Go backend.

Built with Go, PostgreSQL + pgvector, Groq LLM, and Jina Embeddings.

[Get Started :material-arrow-right:](guide/index.md){ .md-button .md-button--primary }
[API Reference :material-book-open-variant:](api/index.md){ .md-button }
[GitHub :fontawesome-brands-github:](https://github.com/ZRishu/smart-portfolio){ .md-button }

</div>

---

## :material-star-four-points: Features

<div class="feature-grid" markdown>

<div class="feature-card" markdown>

### :material-robot: AI Resume Chat (RAG)

Upload a PDF resume — it's chunked, embedded via Jina, and stored in pgvector. Visitors ask questions and get LLM-powered answers from Groq with real-time SSE streaming. A semantic cache avoids redundant LLM calls.

</div>

<div class="feature-card" markdown>

### :material-folder-multiple: Project Showcase

Full CRUD for portfolio projects with in-memory caching (24h TTL). Cache auto-invalidates on writes so reads are always fast and consistent.

</div>

<div class="feature-card" markdown>

### :material-email-outline: Contact Form

Visitors submit messages through a contact form. Messages are persisted to PostgreSQL and an async Discord notification fires in a background goroutine.

</div>

<div class="feature-card" markdown>

### :material-cash-multiple: Sponsorship Payments

Razorpay webhooks with HMAC-SHA256 verification. The transactional outbox pattern guarantees reliable event delivery with Discord alerts for every new sponsor.

</div>

<div class="feature-card" markdown>

### :material-shield-lock: Admin Dashboard API

Aggregate stats from all modules, sponsor listing, deep health check with DB latency — all behind API key authentication with constant-time comparison.

</div>

<div class="feature-card" markdown>

### :material-file-document: Interactive API Docs

Swagger UI served at `/docs` with a comprehensive OpenAPI 3.0 spec. Every endpoint, schema, and error response is documented and testable from the browser.

</div>

</div>

---

## :material-layers-triple: Tech Stack

| Layer             | Technology                                             |
| ----------------- | ------------------------------------------------------ |
| **Language**      | Go 1.26                                                |
| **Router**        | chi/v5                                                 |
| **Database**      | PostgreSQL 16 + pgvector (pgx/v5)                      |
| **AI / LLM**      | Groq (OpenAI-compatible)                               |
| **Embeddings**    | Jina Embeddings v2                                     |
| **Logging**       | zerolog (structured, request-ID correlated)            |
| **Payments**      | Razorpay webhooks                                      |
| **Notifications** | Discord webhooks                                       |
| **Caching**       | go-cache (in-memory, TTL-based)                        |
| **API Docs**      | OpenAPI 3.0 + embedded Swagger UI                      |
| **CI/CD**         | GitHub Actions → GHCR + GitHub Releases                |
| **Deployment**    | Docker (~11 MB image), Railway / Render / Fly.io / VPS |

---

## :material-sitemap: Architecture

```mermaid
flowchart TD
    classDef client fill:#ffedd5,stroke:#f97316,stroke-width:2px,color:#0f172a
    classDef backend fill:#e0e7ff,stroke:#6366f1,stroke-width:2px,color:#0f172a
    classDef db fill:#dbeafe,stroke:#3b82f6,stroke-width:2px,color:#0f172a
    classDef ext fill:#d1fae5,stroke:#10b981,stroke-width:2px,color:#0f172a
    classDef mod fill:#f8fafc,stroke:#94a3b8,stroke-width:2px,color:#0f172a
    classDef plat fill:#ffffff,stroke:#cbd5e1,stroke-width:2px,stroke-dasharray: 5 5,color:#0f172a

    %% 1. Ingress Layer
    Client[Frontend<br/>Vercel / Netlify]:::client
    Razorpay[Razorpay Webhooks]:::ext
    
    %% 2. API Layer
    API[Go Backend API<br/>chi router]:::backend

    Client -->|HTTPS| API
    Razorpay -->|Webhook POST| API

    %% 3. Core Logic Layer
    subgraph AppModules [Application Modules]
        direction TB
        Content[Content Module]:::mod
        AI[AI Module<br/>RAG]:::mod
        Payment[Payment Module]:::mod
        Admin[Admin Module]:::mod
        Notify[Notification Module]:::mod
    end

    subgraph AppPlatform [Platform Services]
        direction TB
        Cache[In-Memory Cache]:::plat
        Bus[Event Bus]:::plat
    end

    API --> AppModules
    Content -.-> Cache
    Payment -.->|Publish Event| Bus
    Bus -.->|Trigger| Notify

    %% 4. Data Layer
    DB[(PostgreSQL + pgvector)]:::db
    AppModules --> DB

    %% 5. Egress / External APIs Layer
    Groq[Groq LLM API]:::ext
    Jina[Jina Embeddings API]:::ext
    Discord[Discord Webhooks]:::ext

    AI -.-> Groq
    AI -.-> Jina
    Notify -.-> Discord
```

---

## :material-lightning-bolt: Quick Start

!!! tip "Fastest path: Docker Compose"

    ```bash
    cd backend
    cp .env.example .env
    # Edit .env — set GROQ_API_KEY and JINA_API_KEY

    docker compose up -d --build
    ```

    Server: [http://localhost:8080](http://localhost:8080){ target="_blank" }
    · API Docs: [http://localhost:8080/docs](http://localhost:8080/docs){ target="_blank" }
    · Health: [http://localhost:8080/healthz](http://localhost:8080/healthz){ target="_blank" }

For detailed setup instructions, see the [Complete Guide](guide/index.md).

---

## :material-folder-open: Project Structure

```text
smart-portfolio/
├── .github/workflows/       CI/CD pipelines
│   ├── ci.yml               Lint → Test → Build (every push/PR)
│   └── cd.yml               Docker push + GitHub Release (on tags)
├── backend/                  Go REST API
│   ├── cmd/server/           Entry point & DI wiring
│   ├── docs/                 OpenAPI spec + Swagger UI handler
│   ├── internal/             Application code (config, modules, platform)
│   ├── migrations/           SQL migration files
│   ├── Dockerfile            Multi-stage production build
│   ├── docker-compose.yml    Local dev stack (app)
│   └── Makefile              Build, test, lint, Docker targets
├── frontend/                 Frontend application
├── docs/                     This documentation site (MkDocs)
├── mkdocs.yml                MkDocs configuration
├── GUIDE.md                  Standalone complete guide
└── README.md                 Project overview
```

---

<div style="text-align: center; padding: 3rem 0;" markdown>

**Ready to build your portfolio?**

[Read the Full Guide :material-book-open-page-variant:](guide/index.md){ .md-button .md-button--primary }
[Explore the API :material-api:](api/index.md){ .md-button }

</div>
