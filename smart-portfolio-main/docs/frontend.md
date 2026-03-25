# Frontend Integration

The frontend for **Smart Portfolio** can be built using any modern JavaScript/TypeScript framework. The Go backend is fully operational and ready to be consumed as a headless REST API.

## Recommended Stack

| Technology       | Why                                                                   |
|------------------|-----------------------------------------------------------------------|
| **Next.js**      | SSR/SSG for SEO, React ecosystem, Vercel deploy, API route proxying   |
| **Astro**        | Maximum performance with minimal JS; great for static portfolio pages |
| **SvelteKit**    | Excellent SSE handling and smaller bundle sizes                       |
| **Nuxt (Vue)**   | Equivalent capabilities to Next.js for Vue.js users                   |
| **Vite + React** | Pure SPA — simpler setup but no SSR/SEO benefits                      |

## Integration Guide

### Response Envelope

Every backend endpoint returns a consistent JSON envelope. Ensure your API client handles this wrapper.

```json
{
  "success": true,
  "data": { ... }
}
```

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "title is required; description is required"
  }
}
```

### TypeScript Types

Types matching the Go backend DTOs:

```typescript
// lib/types.ts

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: number; message: string };
}

export interface Project {
  id: string;
  title: string;
  description: string;
  tech_stack?: string;
  github_url?: string;
  live_url?: string;
  created_at: string;
}

export interface ContactMessageResponse {
  id: string;
  sender_name: string;
  submitted_at: string;
}

export interface ChatResponse {
  answer: string;
  cached: boolean;
}
```

### API Client

A simple wrapper around `fetch`:

```typescript
// lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  const envelope: APIResponse<T> = await res.json();
  if (!envelope.success) throw new Error(envelope.error?.message || "Unknown error");
  return envelope.data as T;
}

export const getProjects = () => fetchAPI<Project[]>("/api/projects");
export const getProject = (id: string) => fetchAPI<Project>(`/api/projects/${id}`);

export const submitContact = (data: {
  sender_name: string;
  sender_email: string;
  message_body: string;
}) =>
  fetchAPI<ContactMessageResponse>("/api/contact", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const askQuestion = (question: string) =>
  fetchAPI<ChatResponse>("/api/chat", {
    method: "POST",
    body: JSON.stringify({ question }),
  });
```

### SSE Streaming Chat

The `POST /api/chat/stream` endpoint returns `text/event-stream`. Use `fetch` with `ReadableStream` — **not** `EventSource` (which only supports GET).

```typescript
// lib/sse.ts
export async function streamChat(
  question: string,
  onToken: (token: string) => void,
  onDone: () => void,
  onError: (error: string) => void
) {
  const response = await fetch(`${API_BASE}/api/chat/stream`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ question }),
  });

  if (!response.ok || !response.body) {
    onError("Failed to start chat stream");
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

### Admin Route Proxying

Keep `ADMIN_API_KEY` server-side by proxying admin calls through your framework's API routes. Example for Next.js App Router:

```typescript
// app/api/admin/stats/route.ts
export async function GET() {
  const res = await fetch(`${process.env.GO_BACKEND_URL}/api/admin/stats`, {
    headers: { "X-Admin-Key": process.env.ADMIN_API_KEY! },
  });
  return Response.json(await res.json());
}
```

### CORS

The Go backend allows these origins in development:

- `http://localhost:3000`
- `http://localhost:5173`
- `http://localhost:5174`

In production, set the `FRONTEND_URL` environment variable on the backend to your deployed frontend domain to restrict CORS access.

### Razorpay Payments

Load the Razorpay `checkout.js` script on the sponsor page. After a successful client-side payment, Razorpay sends a webhook to `POST /api/webhooks/razorpay` on the Go backend — everything is handled server-side from there.

## Deployment Architecture

```text
┌─────────────────────────────────────────────┐
│              Vercel / Netlify               │
│                                             │
│  Frontend (Next.js / Astro / SvelteKit)     │
│  ├── Static pages: /, /projects             │
│  ├── Client pages: /chat, /contact          │
│  └── API routes: proxy to Go backend        │
└──────────────────┬──────────────────────────┘
                   │ HTTPS
                   ▼
┌─────────────────────────────────────────────┐
│      Railway / Render / Fly.io / VPS        │
│                                             │
│  Go Backend (Docker, ~11 MB image)          │
│  └── /api/*, /healthz, /docs                │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│     PostgreSQL + pgvector                   │
│     (Neon / Supabase / Railway)             │
└─────────────────────────────────────────────┘
```