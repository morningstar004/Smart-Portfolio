# Admin API

The Admin endpoints provide aggregate statistics, deep health checking, and sponsor listing for the portfolio dashboard. All admin endpoints require the `X-Admin-Key` header.

---

## Authentication

Admin endpoints accept the API key in either of these formats:

```
X-Admin-Key: your-admin-key
```

or

```
Authorization: Bearer your-admin-key
```

When `ADMIN_API_KEY` is not set in the environment, admin endpoints are accessible without authentication (intended for local development only).

---

## Dashboard Statistics

`GET /api/admin/stats` *(Admin)*

Returns aggregate counts from all modules in a single response, suitable for populating an admin dashboard.

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "projects": 5,
    "contact_messages": {
      "total": 42,
      "unread": 3
    },
    "sponsors": {
      "total_sponsors": 7,
      "total_amount": 3500.00,
      "currency": "INR"
    },
    "vector_store": {
      "documents": 12
    },
    "semantic_cache": {
      "entries": 28
    },
    "outbox_pending": 0
  }
}
```

| Field | Description |
|-------|-------------|
| `projects` | Total portfolio project count |
| `contact_messages.total` | Total contact message count |
| `contact_messages.unread` | Unread contact message count |
| `sponsors.total_sponsors` | Number of distinct sponsors |
| `sponsors.total_amount` | Sum of all sponsor payments |
| `sponsors.currency` | Currency of the total amount |
| `vector_store.documents` | Chunks stored in `resume_embeddings` |
| `semantic_cache.entries` | Cached Q&A pairs in `ai_semantic_cache` |
| `outbox_pending` | Outbox events not yet processed (should be 0 normally) |

**Error Responses**

| Status | When |
|--------|------|
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `500 Internal Server Error` | Database error |

---

## Deep Health Check

`GET /api/admin/health` *(Admin)*

Pings the PostgreSQL database and returns the round-trip latency. Useful for monitoring dashboards and alerting.

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK — healthy)**

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "database": "connected",
    "latency_ms": 2
  }
}
```

**Response (503 Service Unavailable — database unreachable)**

```json
{
  "success": true,
  "data": {
    "status": "unhealthy",
    "database": "unreachable",
    "latency_ms": 0
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `status` | `healthy` \| `unhealthy` | Overall system status |
| `database` | `connected` \| `unreachable` | PostgreSQL connectivity |
| `latency_ms` | integer | Database ping round-trip in milliseconds |

!!! note "Liveness vs Deep Health"
    `GET /healthz` is the lightweight liveness probe (returns `{"status":"ok"}` instantly). Use `GET /api/admin/health` when you need to verify database connectivity specifically.

---

## List Sponsors

`GET /api/admin/sponsors` *(Admin)*

Returns all sponsors ordered by creation date descending, with full payment details.

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK)**

```json
{
  "success": true,
  "data": [
    {
      "id": "b5c6d7e8-f9a0-1234-bcde-f01234567890",
      "sponsor_name": "Alice Smith",
      "email": "alice@example.com",
      "amount": 500.00,
      "currency": "INR",
      "status": "SUCCESS",
      "razorpay_payment_id": "pay_ABC123",
      "created_at": "2024-03-10T15:00:00Z"
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `id` | Sponsor UUID (internal) |
| `sponsor_name` | Name from Razorpay payment notes |
| `email` | Email from Razorpay payment notes |
| `amount` | Payment amount in major currency units |
| `currency` | 3-letter currency code (e.g., `INR`, `USD`) |
| `status` | Payment status (`SUCCESS`) |
| `razorpay_payment_id` | Razorpay payment ID (unique per payment) |
| `created_at` | Timestamp when the sponsor was recorded |

**Error Responses**

| Status | When |
|--------|------|
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `500 Internal Server Error` | Database error |
