# AI Chat API

The Chat endpoints expose the full RAG pipeline, enabling visitors to ask natural language questions about the portfolio owner's resume and experience.

---

## How It Works

1. The question is embedded via the Jina API to produce a 768-dimension vector.
2. The vector is compared against previously answered questions in `ai_semantic_cache` using cosine distance. If a match is found (distance < 0.05), the cached answer is returned immediately without an LLM call.
3. Otherwise, the embedding is used to perform a similarity search over `resume_embeddings` (top-3 closest chunks).
4. The retrieved chunks are assembled into a system prompt and sent to the Groq LLM (`llama-3.3-70b-versatile` by default).
5. The answer is returned and saved to the semantic cache asynchronously.

Both endpoints use the same pipeline — the only difference is whether the answer is delivered as a complete JSON response or token-by-token via SSE.

---

## Ask a Question (JSON)

`POST /api/chat`

Returns the full answer as a single JSON response.

**Request Body**

```json
{
  "question": "What programming languages does the portfolio owner know?"
}
```

| Field | Type | Required |
|-------|------|----------|
| `question` | string | Yes |

**Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "answer": "Based on the resume, the owner has expertise in Go, Python, TypeScript, and SQL...",
    "cached": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `answer` | string | The LLM-generated answer |
| `cached` | boolean | `true` if the response came from the semantic cache |

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Empty question |
| `500 Internal Server Error` | Embedding or LLM API failure |

---

## Ask a Question (SSE Streaming)

`POST /api/chat/stream`

Streams the answer token-by-token using Server-Sent Events. The response begins as soon as the LLM starts generating.

**Request Body**

Same as the JSON endpoint:

```json
{
  "question": "Tell me about their work experience."
}
```

**Response — `text/event-stream`**

Each token arrives as an SSE data frame:

```
data: Based\n\n
data:  on\n\n
data:  the\n\n
...
event: done
data: \n\n
```

The `event: done` frame signals that the stream is complete.

!!! warning "Use fetch, not EventSource"
    `EventSource` only supports GET requests. Use `fetch` with `ReadableStream` to consume this endpoint.

**TypeScript implementation:**

```typescript
export async function streamChat(
  question: string,
  onToken: (token: string) => void,
  onDone: () => void,
  onError: (error: string) => void
) {
  const response = await fetch("/api/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question }),
  });

  if (!response.ok || !response.body) {
    onError("Failed to start stream");
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    for (const line of lines) {
      if (line.startsWith("event: done")) { onDone(); return; }
      if (line.startsWith("data: ")) onToken(line.slice(6));
    }
  }
  onDone();
}
```

**Error Responses**

| Status | When |
|--------|------|
| `400 Bad Request` | Empty question |
| `500 Internal Server Error` | Embedding or LLM API failure |

---

## Semantic Cache

Both endpoints share a semantic cache backed by pgvector. After an LLM answer is generated:

- The question embedding and answer are stored in `ai_semantic_cache`.
- Future questions with cosine distance < **0.05** to a cached question reuse the cached answer instantly.
- Cache saves happen asynchronously so they never delay the HTTP response.

The `cached: true` field in JSON responses indicates a cache hit.
