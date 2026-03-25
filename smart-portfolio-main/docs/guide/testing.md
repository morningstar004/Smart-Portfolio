# Testing

## Unit Tests

```bash
cd backend

# Run all tests
make test
```

This runs `go test ./... -count=1`. The `-count=1` flag bypasses the test cache so tests always execute fresh.

**Test coverage by package:**

| Package | What's Covered |
|---------|---------------|
| `internal/httputil` | JSON response envelope, error types, `ParseUUID`, `HandleServiceError` |
| `internal/middleware` | Request ID, security headers, rate limiter, admin auth, body size limit, panic recovery, request logger |
| `internal/platform/eventbus` | Subscribe, publish, concurrent publish, panic recovery in handlers, shutdown drain, context cancellation |
| `internal/modules/ai/service` | Text chunking algorithm, whitespace normalization, overlap behavior, edge cases (empty input, single word) |
| `internal/modules/content/dto` | Project and contact validation rules, error messages, boundary values |
| `internal/modules/payment/service` | HMAC-SHA256 signature verification, duplicate event detection, tampered payload rejection |

---

## Verbose and Targeted Tests

```bash
# All tests with full output (see each test name)
make test-v

# Run tests in one specific package
go test ./internal/platform/eventbus/... -v -count=1

# Run a single test by name
go test ./internal/platform/eventbus/... -v -run TestPublish_DeliversToHandler -count=1

# Run all tests matching a pattern
go test ./internal/modules/payment/service/... -v -run TestVerifyWebhookSignature -count=1

# Run all tests in one module's subtree
go test ./internal/modules/content/... -v -count=1
```

---

## Coverage Report

```bash
# Generate coverage HTML report
make cover

# Output files:
#   coverage/coverage.out   raw coverage data
#   coverage/coverage.html  interactive HTML report

# Open the report
open coverage/coverage.html        # macOS
xdg-open coverage/coverage.html    # Linux
```

To see a text summary in the terminal:

```bash
go test ./... -count=1 -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

Example output:

```
github.com/ZRishu/smart-portfolio/internal/httputil/response.go:42:    WriteJSON            100.0%
github.com/ZRishu/smart-portfolio/internal/httputil/response.go:58:    WriteError           100.0%
github.com/ZRishu/smart-portfolio/internal/httputil/response.go:75:    WriteValidationError 100.0%
...
total:                                                                   (statements)         87.3%
```

---

## Benchmarks

```bash
# Run all benchmarks
make bench

# Run benchmarks in a specific package
go test ./internal/platform/eventbus/... -bench=. -benchmem -run=^$ -count=1

# Run a specific benchmark
go test ./internal/modules/ai/service/... -bench=BenchmarkChunkText -benchmem -run=^$ -count=1
```

Benchmarks exist for:

- Event bus publish (single handler, 10 handlers, no handlers)
- Text chunking (small, medium, large inputs)
- Whitespace normalization (clean vs dirty input)
- HMAC-SHA256 signature verification (valid and invalid payloads)
- DTO validation (valid and invalid inputs)

---

## Race Detector

Go's built-in race detector finds concurrent memory access bugs. Always run this before opening a pull request:

```bash
go test ./... -count=1 -race
```

The CI pipeline runs with `-race` enabled on every push and pull request.

---

## Manual API Testing with curl

Once the server is running at `http://localhost:8080`:

=== "Health"

    ```bash
    curl http://localhost:8080/healthz
    # → {"status":"ok"}
    ```

=== "Projects"

    ```bash
    # Create a project
    curl -s -X POST http://localhost:8080/api/projects \
      -H "Content-Type: application/json" \
      -d '{
        "title": "Smart Portfolio",
        "description": "AI-powered developer portfolio",
        "tech_stack": "Go, PostgreSQL, pgvector"
      }' | jq .

    # List all
    curl -s http://localhost:8080/api/projects | jq .

    # Get by ID
    curl -s http://localhost:8080/api/projects/<uuid> | jq .

    # Update
    curl -s -X PUT http://localhost:8080/api/projects/<uuid> \
      -H "Content-Type: application/json" \
      -d '{"title": "Updated Title", "description": "Updated description"}' | jq .

    # Delete
    curl -s -X DELETE http://localhost:8080/api/projects/<uuid>
    ```

=== "Contact"

    ```bash
    curl -s -X POST http://localhost:8080/api/contact \
      -H "Content-Type: application/json" \
      -d '{
        "sender_name": "Jane Doe",
        "sender_email": "jane@example.com",
        "message_body": "Hello! Great portfolio."
      }' | jq .
    ```

=== "AI Chat"

    ```bash
    # Non-streaming JSON response
    curl -s -X POST http://localhost:8080/api/chat \
      -H "Content-Type: application/json" \
      -d '{"question": "What skills does the portfolio owner have?"}' | jq .

    # SSE streaming (prints tokens as they arrive)
    curl -N -X POST http://localhost:8080/api/chat/stream \
      -H "Content-Type: application/json" \
      -d '{"question": "Tell me about their experience"}'
    ```

=== "Ingestion (admin)"

    ```bash
    # Ingest a PDF resume
    curl -s -X POST http://localhost:8080/api/ingest \
      -H "X-Admin-Key: your-admin-key" \
      -F "file=@/path/to/resume.pdf" | jq .

    # Ingest raw text
    curl -s -X POST http://localhost:8080/api/ingest/text \
      -H "X-Admin-Key: your-admin-key" \
      -H "Content-Type: application/json" \
      -d '{"text": "John Doe, Go developer...", "source_name": "resume-v1"}' | jq .

    # Clear the vector store
    curl -s -X DELETE http://localhost:8080/api/ingest \
      -H "X-Admin-Key: your-admin-key" | jq .
    ```

=== "Admin"

    ```bash
    # Dashboard statistics
    curl -s http://localhost:8080/api/admin/stats \
      -H "X-Admin-Key: your-admin-key" | jq .

    # Deep health check (includes DB latency)
    curl -s http://localhost:8080/api/admin/health \
      -H "X-Admin-Key: your-admin-key" | jq .

    # List sponsors
    curl -s http://localhost:8080/api/admin/sponsors \
      -H "X-Admin-Key: your-admin-key" | jq .
    ```

!!! tip "Swagger UI"
    Open `http://localhost:8080/docs` in your browser for an interactive Swagger UI where you can test every endpoint without writing curl commands.
