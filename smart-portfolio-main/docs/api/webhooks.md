# Webhooks API

## Razorpay Payment Webhook

`POST /api/webhooks/razorpay`

Receives payment event notifications from Razorpay. This endpoint is called by Razorpay's servers — your application does not call it directly.

---

## How It Works

1. Razorpay sends a POST request with a JSON body and an `X-Razorpay-Signature` header.
2. The backend reads the raw request body and verifies the HMAC-SHA256 signature using `RAZORPAY_WEBHOOK_SECRET`.
3. If the signature is valid, the event type is checked. Only `payment.captured` events are processed.
4. For `payment.captured`: a sponsor record and an outbox event are inserted atomically in a single PostgreSQL transaction.
5. The outbox poller background worker detects the new event, publishes it to the in-process event bus, and the Discord notification handler fires.
6. Duplicate deliveries (identified by Razorpay's event ID) are safely ignored via a UNIQUE constraint on `outbox_events.event_id`.

---

## Endpoint Details

**Headers**

| Header | Description |
|--------|-------------|
| `X-Razorpay-Signature` | HMAC-SHA256 of the raw request body, signed with `RAZORPAY_WEBHOOK_SECRET` |
| `Content-Type` | `application/json` |

**Request Body**

Razorpay's standard webhook payload. Example for `payment.captured`:

```json
{
  "event": "payment.captured",
  "payload": {
    "payment": {
      "entity": {
        "id": "pay_ABC123",
        "amount": 50000,
        "currency": "INR",
        "notes": {
          "sponsor_name": "Alice Smith",
          "email": "alice@example.com"
        }
      }
    }
  }
}
```

Note: `amount` is in the smallest currency unit (paise for INR). The backend converts it to the major unit (rupees) when storing the sponsor record.

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "message": "payment processed"
  }
}
```

Unrecognized event types (not `payment.captured`) are acknowledged with:

```json
{
  "success": true,
  "data": {
    "message": "event acknowledged"
  }
}
```

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Invalid or missing `X-Razorpay-Signature` header, or signature mismatch |
| `500 Internal Server Error` | Database error during sponsor insertion |

---

## Setting Up the Webhook in Razorpay

1. Log in to the [Razorpay Dashboard](https://dashboard.razorpay.com/).
2. Go to **Settings** → **Webhooks** → **Add New Webhook**.
3. **Webhook URL**: `https://your-api-domain.com/api/webhooks/razorpay`
4. **Secret**: Enter a strong random string. This becomes your `RAZORPAY_WEBHOOK_SECRET` environment variable.
5. **Active Events**: Check `payment.captured`.
6. Click **Save**.

Set the same secret in your backend environment:

```dotenv
RAZORPAY_WEBHOOK_SECRET=the-secret-you-entered-in-razorpay
```

---

## Security

- Signature verification uses **HMAC-SHA256** with constant-time comparison to prevent timing attacks.
- The raw request body is used for signature verification (before any JSON parsing) to ensure byte-for-byte accuracy.
- Duplicate event IDs are rejected by a `UNIQUE` constraint on `outbox_events.event_id`, making the handler idempotent.

---

## Transactional Outbox Pattern

The webhook handler uses the transactional outbox pattern to guarantee reliable Discord notification delivery:

```
Razorpay POST
     │
     ▼
Verify HMAC signature
     │
     ▼
INSERT sponsor + INSERT outbox_event  ← single DB transaction
     │
     ▼
HTTP 200 OK
     │
     (background)
     ▼
Outbox poller detects new event
     │
     ▼
Publish to in-process event bus
     │
     ▼
Discord notification handler fires
```

If the Discord notification fails, the outbox event remains unprocessed and the poller retries on its next tick.
