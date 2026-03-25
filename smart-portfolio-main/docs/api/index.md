# API Overview

The Smart Portfolio backend exposes a fast, RESTful API built with Go and chi. It powers all frontend features including AI chat, contact submissions, project listing, and administrative dashboards.

## Base URL

All endpoints are relative to the root URL of your deployed backend. For local development, this is typically:

```
http://localhost:8080
```

## Interactive Documentation

When the backend is running locally, you can access the interactive Swagger UI at:

[http://localhost:8080/docs](http://localhost:8080/docs)

## Authentication

Public endpoints (like listing projects or asking questions via chat) do not require authentication.

Admin-protected endpoints (such as `/api/admin/*`, `/api/ingest`, and managing contact messages) require the admin API key. You can provide this key in one of two ways:

1. **Header:** `X-Admin-Key: <your_admin_key>`
2. **Bearer Token:** `Authorization: Bearer <your_admin_key>`

*Note: If the `ADMIN_API_KEY` environment variable is left empty during local development, authentication is bypassed for convenience.*

## Response Envelope

Every endpoint returns a consistent JSON envelope. This ensures clients can uniformly handle successes and errors.

### Success Response

```json
{
  "success": true,
  "data": {
    "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "title": "Smart Portfolio",
    "description": "A full-stack portfolio with AI chat"
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "title is required; description is required"
  }
}
```

Common HTTP status codes are reflected in the `code` field:
- `400` - Bad Request (e.g., validation failed)
- `401` - Unauthorized (missing or invalid admin key)
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests (rate limit exceeded)
- `500` - Internal Server Error

## Rate Limiting

The API employs a token bucket rate limiter to prevent abuse. If you exceed the configured `RATE_LIMIT_RPS` (default: 10 requests per second per IP), the API will return a `429 Too Many Requests` error.

## SSE Streaming

The AI chat feature supports Server-Sent Events (SSE) for streaming responses token-by-token. See the [AI Chat documentation](chat.md) for implementation details.
