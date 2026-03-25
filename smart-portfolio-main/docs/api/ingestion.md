# Ingestion API

The Ingestion endpoints populate the vector store with resume content used by the AI Chat RAG pipeline. All endpoints require admin authentication.

---

## Upload a PDF Resume

`POST /api/ingest` *(Admin)*

Accepts a multipart form upload containing a PDF file. The server extracts text, chunks it into overlapping segments, embeds each chunk via the Jina API, and stores the vectors in `resume_embeddings`.

**Max file size:** 20 MB

**Headers**

```
X-Admin-Key: your-admin-key
```

**Form Fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | binary | Yes | PDF file to ingest |

**Example with curl**

```bash
curl -s -X POST http://localhost:8080/api/ingest \
  -H "X-Admin-Key: your-admin-key" \
  -F "file=@/path/to/resume.pdf" | jq .
```

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "message": "Successfully ingested resume.pdf",
    "pages": 3,
    "chunks": 12
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message` | string | Human-readable summary |
| `pages` | integer | Number of pages extracted from the PDF |
| `chunks` | integer | Number of text chunks embedded and stored |

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | No file field, wrong content type, or file exceeds 20 MB |
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `422 Unprocessable Entity` | PDF has no extractable text, or resulted in zero chunks |
| `500 Internal Server Error` | Embedding API error or database error |

---

## Ingest Raw Text

`POST /api/ingest/text` *(Admin)*

Accepts plain text (e.g., a resume copy-pasted from a Word document). Chunks, embeds, and stores the same way as the PDF endpoint.

**Max body size:** 5 MB

**Headers**

```
X-Admin-Key: your-admin-key
Content-Type: application/json
```

**Request Body**

```json
{
  "text": "John Doe — Senior Go Developer\n\nExperience:\n...",
  "source_name": "resume-v2"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | Yes | The raw resume text to ingest |
| `source_name` | string | No | Label stored in embedding metadata (defaults to `"manual-text-input"`) |

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "message": "Successfully ingested resume-v2",
    "pages": 0,
    "chunks": 9
  }
}
```

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Empty text field or body exceeds 5 MB |
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `422 Unprocessable Entity` | Text resulted in zero chunks after processing |
| `500 Internal Server Error` | Embedding API error or database error |

---

## Clear Vector Store

`DELETE /api/ingest` *(Admin)*

Removes all document embeddings from `resume_embeddings`. Use this before re-ingesting a new version of the resume so the chat pipeline does not retrieve stale content.

**Headers**

```
X-Admin-Key: your-admin-key
```

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "message": "Cleared 12 documents from vector store",
    "deleted": 12
  }
}
```

**Error Responses**

| Status | When |
|--------|------|
| `401 Unauthorized` | No API key provided |
| `403 Forbidden` | Invalid API key |
| `500 Internal Server Error` | Database error |

---

## Chunking Details

Text is split into overlapping chunks before embedding:

| Parameter | Value |
|-----------|-------|
| Chunk size | 800 characters |
| Overlap | 200 characters |
| Split boundary | Word boundaries (no mid-word splits) |
| Batch size | 32 chunks per Jina API call |
| Concurrency | Up to 4 concurrent embedding batches |

Overlapping chunks ensure that context around chunk boundaries is not lost during retrieval.
