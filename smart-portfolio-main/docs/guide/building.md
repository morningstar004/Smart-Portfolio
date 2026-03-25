# Building Binaries

## Local Build

```bash
cd backend

# Build for your current OS/architecture
make build

# Output: bin/smart-portfolio
# Run it:
./bin/smart-portfolio
```

The Makefile uses these flags:
- `-trimpath` — removes local file paths from the binary (reproducible builds)
- `-ldflags="-s -w"` — strips debug symbols and DWARF info (smaller binary)

---

## Cross-Compilation

Go makes cross-compilation trivial. No extra toolchains needed.

```bash
cd backend

# Linux amd64 (most servers, CI runners, Docker)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath -ldflags="-s -w" \
  -o bin/smart-portfolio-linux-amd64 \
  ./cmd/server

# Linux arm64 (AWS Graviton, Oracle Ampere, Raspberry Pi 4+)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -trimpath -ldflags="-s -w" \
  -o bin/smart-portfolio-linux-arm64 \
  ./cmd/server

# macOS Apple Silicon (M1/M2/M3/M4)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
  -trimpath -ldflags="-s -w" \
  -o bin/smart-portfolio-darwin-arm64 \
  ./cmd/server

# macOS Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
  -trimpath -ldflags="-s -w" \
  -o bin/smart-portfolio-darwin-amd64 \
  ./cmd/server

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -trimpath -ldflags="-s -w" \
  -o bin/smart-portfolio-windows-amd64.exe \
  ./cmd/server
```

Or use the Makefile shortcut for Linux:

```bash
make build-linux
# Output: bin/smart-portfolio-linux-amd64
```

**Key flags explained:**

| Flag | Purpose |
|------|---------|
| `CGO_ENABLED=0` | Disable CGo for a fully static binary (no libc dependency) |
| `GOOS=linux` | Target operating system |
| `GOARCH=amd64` | Target CPU architecture |
| `-trimpath` | Remove local filesystem paths from binary |
| `-ldflags="-s -w"` | Strip symbol table (`-s`) and DWARF debug info (`-w`) for smaller size |

Typical binary sizes: **~12-15 MB** depending on platform.

---

## Docker Image

```bash
cd backend

# Build the production Docker image
make docker-build

# Or manually:
docker build -t smart-portfolio:latest .

# Check the image size (~11 MB)
docker images smart-portfolio:latest

# Run it (you need a PostgreSQL instance accessible to the container)
docker run --rm -p 8080:8080 \
  --env-file .env \
  smart-portfolio:latest

# Or with individual env vars:
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=require" \
  -e GROQ_API_KEY="your-key" \
  -e JINA_API_KEY="your-key" \
  -e ADMIN_API_KEY="your-secret" \
  -e ENV="production" \
  smart-portfolio:latest
```

**The Dockerfile is a two-stage build:**

1. **Builder stage** (`golang:1.26-alpine`): Downloads deps, compiles a static binary
2. **Production stage** (`alpine:3.20`): Copies only the binary + migrations, runs as non-root user, includes a health check

The final image contains:
- The compiled Go binary (~12 MB)
- `migrations/` directory
- CA certificates (for outbound HTTPS to Groq, Jina, Discord, Razorpay)
- tzdata (for timezone support)
- Non-root `appuser` for security
