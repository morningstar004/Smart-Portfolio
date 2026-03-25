# Contact API

The Contact endpoints handle public contact form submissions and admin-side message management.

---

## Submit a Contact Message

`POST /api/contact`

Public endpoint — no authentication required. Persists the message to PostgreSQL and fires an asynchronous Discord notification (if `DISCORD_WEBHOOK_URL` is configured).

**Request Body**

```json
{
  "sender_name": "Jane Doe",
  "sender_email": "jane@example.com",
  "message_body": "Hello! I'd love to collaborate."
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `sender_name` | string | Yes | Non-empty |
| `sender_email` | string | Yes | Valid email format |
| `message_body` | string | Yes | Non-empty |

**Response (201 Created)**

```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "sender_name": "Jane Doe",
    "submitted_at": "2024-03-10T14:30:00Z"
  }
}
```

Only `id`, `sender_name`, and `submitted_at` are returned — the full message body is not echoed back to the client.

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Missing required fields or invalid email format |
| `500 Internal Server Error` | Database error |

---

## List All Messages

`GET /api/contact` *(Admin)*

Returns every contact message ordered by `submitted_at` descending.

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
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "sender_name": "Jane Doe",
      "sender_email": "jane@example.com",
      "message_body": "Hello! I'd love to collaborate.",
      "is_read": false,
      "submitted_at": "2024-03-10T14:30:00Z"
    }
  ]
}
```

**Error Responses**

| Status | When |
|--------|------|
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `500 Internal Server Error` | Database error |

---

## List Unread Messages

`GET /api/contact/unread` *(Admin)*

Returns only messages where `is_read = false`, ordered by `submitted_at` descending.

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK)**

Same shape as [List All Messages](#list-all-messages), filtered to unread messages only.

---

## Mark a Message as Read

`PATCH /api/contact/{id}/read` *(Admin)*

Marks the specified message as read (`is_read = true`).

**Path Parameter**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Contact message ID |

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "message": "contact message marked as read"
  }
}
```

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Invalid UUID format |
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `404 Not Found` | Message not found |

---

## Delete a Message

`DELETE /api/contact/{id}` *(Admin)*

Permanently removes a contact message.

**Path Parameter**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Contact message ID |

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (204 No Content)**

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Invalid UUID format |
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `404 Not Found` | Message not found |
