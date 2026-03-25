# Prerequisites

Install these tools and gather the API keys before running Smart Portfolio.

---

## Required Tools

| Tool | Minimum Version | Purpose | Install |
|------|----------------|---------|---------|
| **Go** | 1.26+ | Compile and run the backend | [go.dev/dl](https://go.dev/dl/) |
| **Docker** | 24+ | Container runtime for the app | [docs.docker.com](https://docs.docker.com/get-docker/) |
| **Docker Compose** | v2+ | Multi-container orchestration | Bundled with Docker Desktop |
| **Git** | 2.40+ | Version control | [git-scm.com](https://git-scm.com/) |
| **make** | any | Run Makefile targets | Pre-installed on Linux and macOS |

## Optional Tools

| Tool | Purpose | Install |
|------|---------|---------|
| **psql** | Run migrations manually | Comes with any PostgreSQL package |
| **Air** | Live reload for Go during development | `go install github.com/air-verse/air@latest` |
| **staticcheck** | Static analysis for the lint pipeline | `go install honnef.co/go/tools/cmd/staticcheck@latest` |

---

## Required API Keys

Both keys are **required** — without them the AI chat and resume ingestion features will not work. Both services offer a free tier that is sufficient for a personal portfolio.

| Key | Where to Get | What it Powers |
|-----|--------------|---------------|
| `GROQ_API_KEY` | [console.groq.com](https://console.groq.com/) | LLM inference for AI chat answers |
| `JINA_API_KEY` | [jina.ai](https://jina.ai/) | Text embeddings for RAG retrieval |

## Database Requirement

You need an external PostgreSQL database with the `pgvector` extension installed. You will provide the connection string via the `DATABASE_URL` environment variable.

Providers like **Neon** ([neon.tech](https://neon.tech)) or **Supabase** ([supabase.com](https://supabase.com)) offer generous free tiers that include `pgvector` out of the box.

---

## Optional Credentials

These can be left empty during local development. The system will skip the relevant feature when the value is absent.

| Credential | Where to Get | What it Enables |
|-----------|--------------|----------------|
| `ADMIN_API_KEY` | Any strong secret you choose | Protects admin endpoints (`/api/admin/*`, `/api/ingest`, admin contact routes) |
| `DISCORD_WEBHOOK_URL` | Discord server → Integrations → Webhooks | Discord notifications on contact messages and new sponsors |
| `RAZORPAY_KEY_ID` | [dashboard.razorpay.com](https://dashboard.razorpay.com/) | Razorpay payment checkout |
| `RAZORPAY_KEY_SECRET` | Razorpay dashboard | Razorpay order creation |
| `RAZORPAY_WEBHOOK_SECRET` | Razorpay dashboard → Webhooks | HMAC-SHA256 webhook signature verification |

!!! note "Local development without an admin key"
    When `ADMIN_API_KEY` is empty, admin endpoints are accessible without any authentication header. This is intentional for local development convenience. **Always set a strong admin key in production.**

---

## Environment File

The backend reads configuration from environment variables. Copy the example file before the first run:

```bash
cd backend
cp .env.example .env
```

Then open `.env` in your editor. At minimum, set:

```dotenv
DATABASE_URL=your-external-postgres-url
GROQ_API_KEY=your-groq-key-here
JINA_API_KEY=your-jina-key-here
```

All other variables have sensible defaults for local development. See the [full environment variable reference](../architecture/backend.md#environment-variables) for every available option.
