# Running the Application

Three options are available depending on your workflow. Docker Compose is the recommended starting point.

---

## Option A: Docker Compose (Recommended)

One command starts the Go server container connected to your external database. Migrations run automatically on startup.

```bash
# 1. Navigate to the backend directory
cd backend

# 2. Create your environment file from the example
cp .env.example .env
# Edit .env — fill in at minimum:
#   DATABASE_URL=your-external-postgres-url
#   GROQ_API_KEY=your-groq-key
#   JINA_API_KEY=your-jina-key

# 3. Start the backend stack
docker compose up -d --build
```

**What happens on startup:**

1. Docker builds the Go app from the Dockerfile (multi-stage, ~11 MB final image)
2. The Go server connects to your external database and runs `migrations/001_init.sql` automatically
3. The server starts on port 8080

**Verify it's working:**

```bash
# Liveness probe
curl http://localhost:8080/healthz
# → {"status":"ok"}

# List projects (empty initially)
curl http://localhost:8080/api/projects
# → {"success":true,"data":[]}

# Open Swagger UI in browser
open http://localhost:8080/docs
```

**Common Docker commands:**

```bash
# Tail all logs
docker compose logs -f

# Tail only the app logs
docker compose logs -f app

# See running containers
docker compose ps

# Stop everything
docker compose down

# Rebuild after code changes
docker compose up -d --build
```

---

## Option B: Local Go + External PostgreSQL

Use this when you want fast iteration running the Go binary locally, connecting to an external PostgreSQL database.

```bash
# 1. Navigate to the backend directory
cd backend

# 2. Create your .env file
cp .env.example .env
# Set at minimum:
#   DATABASE_URL=your-external-postgres-url
#   GROQ_API_KEY=your-key
#   JINA_API_KEY=your-key

# 3. Download Go dependencies
go mod tidy

# 4. Build and run
make run
```

The server starts at `http://localhost:8080`. Migrations run automatically on startup.

You can also run without Make:

```bash
go run ./cmd/server
```

---

## Option C: Live Reload with Air

[Air](https://github.com/air-verse/air) watches Go files and automatically rebuilds and restarts the server on every save. This is the fastest inner loop for active development.

```bash
# 1. Install Air (one-time setup)
go install github.com/air-verse/air@latest

# 2. Make sure your .env has DATABASE_URL set
cd backend

# 3. Start the dev server with live reload
make dev
```

Air uses the configuration in `.air.toml`. It watches `.go`, `.sql`, and `.toml` files, excludes test files, and rebuilds into `tmp/server`. The terminal clears on each rebuild.

**The dev loop:**

1. Edit a `.go` file and save
2. Air detects the change (~500ms delay)
3. Air sends SIGINT to the running process (graceful shutdown)
4. Air compiles the new binary to `tmp/server`
5. Air starts the new binary
6. Changes are live

---

## Stopping the Server

| Situation | Command |
|-----------|---------|
| Docker Compose running | `docker compose down` |
| Air or `go run` terminal | `Ctrl+C` |

The server performs a graceful shutdown on SIGINT/SIGTERM: it drains in-flight HTTP requests, stops the outbox poller, shuts down the event bus, waits for Discord notifications in flight, and closes the database pool.
