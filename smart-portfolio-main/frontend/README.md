# Smart Portfolio Frontend

Astro + TypeScript frontend for the Smart Portfolio backend. Requires **Bun** as the package manager and runtime.

## Prerequisites

- [Bun](https://bun.sh/) >= 1.0
- Go backend running on `http://localhost:8080` (see [`../backend/README.md`](../backend/README.md))

## Setup

```bash
cp .env.example .env
bun install
```

## Development

```bash
bun run dev
```

Opens at `http://localhost:5173`. API calls are proxied to `http://localhost:8080`.

## Type Checking

```bash
bun run check
```

## Production Build

```bash
bun run build
bun run preview
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PUBLIC_API_URL` | `""` (proxied) | Backend API base URL for production |
| `PUBLIC_DEV_API_PROXY` | `http://localhost:8080` | Dev-only proxy target for `/api` requests |
| `FRONTEND_PORT` | `5173` | Astro dev server port |

Create `.env` for overrides:

```env
PUBLIC_API_URL=https://api.example.com
PUBLIC_DEV_API_PROXY=http://localhost:8080
```

## Deployment

- Set `PUBLIC_API_URL` to your deployed backend origin, for example `https://api.example.com`
- The frontend is static, so you can deploy `dist/` to any static host after:

```bash
bun run build
```

## Architecture

- **Astro** — static-first framework with island architecture
- **TypeScript (strict)** — all code is strictly typed
- **Bun** — fast runtime and package manager
- Content is dynamically loaded from the backend AI chat endpoint, reflecting the admin-ingested resume
