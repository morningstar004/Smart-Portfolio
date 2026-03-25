package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// RequestID
// ---------------------------------------------------------------------------

func TestRequestID_GeneratesUUID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID to be set in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	xReqID := resp.Header.Get("X-Request-ID")
	if xReqID == "" {
		t.Fatal("expected X-Request-ID response header to be set")
	}
	// UUID v4 format: 8-4-4-4-12 hex chars = 36 total
	if len(xReqID) != 36 {
		t.Errorf("expected UUID-length request ID (36 chars), got %d: %q", len(xReqID), xReqID)
	}
}

func TestRequestID_ReusesIncomingHeader(t *testing.T) {
	const customID = "my-custom-trace-id-12345"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != customID {
			t.Errorf("expected context request ID %q, got %q", customID, id)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Request-ID", customID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	xReqID := resp.Header.Get("X-Request-ID")
	if xReqID != customID {
		t.Errorf("expected X-Request-ID header %q, got %q", customID, xReqID)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	var ids []string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, GetRequestID(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate request ID generated: %q", id)
		}
		seen[id] = true
	}
}

// ---------------------------------------------------------------------------
// GetRequestID
// ---------------------------------------------------------------------------

func TestGetRequestID_EmptyContext(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	id := GetRequestID(r.Context())
	if id != "" {
		t.Errorf("expected empty string for context without request ID, got %q", id)
	}
}

// ---------------------------------------------------------------------------
// SecurityHeaders
// ---------------------------------------------------------------------------

func TestSecurityHeaders_AllPresent(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}

	for header, expected := range expectedHeaders {
		got := resp.Header.Get(header)
		if got != expected {
			t.Errorf("header %q: expected %q, got %q", header, expected, got)
		}
	}
}

func TestSecurityHeaders_DoesNotOverrideBody(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("expected body 'hello', got %q", string(body))
	}
}

// ---------------------------------------------------------------------------
// ContentTypeJSON
// ---------------------------------------------------------------------------

func TestContentTypeJSON(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type to start with 'application/json', got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// Healthcheck
// ---------------------------------------------------------------------------

func TestHealthcheck(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	Healthcheck(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected JSON content type, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("expected body to contain status ok, got %q", string(body))
	}
}

// ---------------------------------------------------------------------------
// AdminAuth
// ---------------------------------------------------------------------------

func TestAdminAuth_EmptyKey_AllowsAll(t *testing.T) {
	middleware := AdminAuth("")
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when admin key is empty, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "allowed" {
		t.Errorf("expected body 'allowed', got %q", string(body))
	}
}

func TestAdminAuth_ValidXAdminKey(t *testing.T) {
	const secret = "super-secret-key-123"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("admin-ok"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	r.Header.Set("X-Admin-Key", secret)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with valid X-Admin-Key, got %d", resp.StatusCode)
	}
}

func TestAdminAuth_ValidBearerToken(t *testing.T) {
	const secret = "my-bearer-token"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/sponsors", nil)
	r.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with valid Bearer token, got %d", resp.StatusCode)
	}
}

func TestAdminAuth_MissingKey_Unauthorized(t *testing.T) {
	const secret = "required-key"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when auth fails")
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	// No X-Admin-Key or Authorization header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "authentication required") {
		t.Errorf("expected authentication error message, got %q", string(body))
	}
}

func TestAdminAuth_WrongKey_Forbidden(t *testing.T) {
	const secret = "correct-key"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when auth fails")
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	r.Header.Set("X-Admin-Key", "wrong-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid API key") {
		t.Errorf("expected 'invalid API key' message, got %q", string(body))
	}
}

func TestAdminAuth_WrongBearerToken_Forbidden(t *testing.T) {
	const secret = "the-real-key"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when auth fails")
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	r.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestAdminAuth_BearerPrefix_TooShort(t *testing.T) {
	const secret = "key123"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	r.Header.Set("Authorization", "Bear") // Too short to be "Bearer "
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for malformed Authorization header, got %d", resp.StatusCode)
	}
}

func TestAdminAuth_XAdminKey_TakesPrecedence(t *testing.T) {
	const secret = "correct-key"
	middleware := AdminAuth(secret)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	r.Header.Set("X-Admin-Key", secret)
	r.Header.Set("Authorization", "Bearer wrong-key") // Should be ignored
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (X-Admin-Key takes precedence), got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// MaxBodySize
// ---------------------------------------------------------------------------

func TestMaxBodySize_WithinLimit(t *testing.T) {
	const limit int64 = 1024 // 1 KB

	middleware := MaxBodySize(limit)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error reading body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	payload := strings.Repeat("a", 512) // 512 bytes, well within limit
	r := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(payload))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for body within limit, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != payload {
		t.Errorf("expected echo of payload, got %d bytes", len(body))
	}
}

func TestMaxBodySize_ExceedsLimit(t *testing.T) {
	const limit int64 = 100 // 100 bytes

	middleware := MaxBodySize(limit)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// http.MaxBytesReader returns an error when the limit is exceeded
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	payload := strings.Repeat("x", 200) // 200 bytes, exceeds 100 byte limit
	r := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(payload))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for body exceeding limit, got %d", resp.StatusCode)
	}
}

func TestMaxBodySize_EmptyBody(t *testing.T) {
	const limit int64 = 1024

	middleware := MaxBodySize(limit)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(body))
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Recoverer
// ---------------------------------------------------------------------------

func TestRecoverer_NoPanic(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("no panic"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when no panic, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "no panic" {
		t.Errorf("expected 'no panic', got %q", string(body))
	}
}

func TestRecoverer_CatchesPanic(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Should NOT propagate the panic
	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Response should be JSON
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected JSON content type, got %q", ct)
	}

	// Should NOT leak the panic message to the client
	if strings.Contains(bodyStr, "something went wrong") {
		t.Errorf("panic message leaked to client: %q", bodyStr)
	}

	// Should contain a generic error message
	if !strings.Contains(bodyStr, "internal error") {
		t.Errorf("expected generic error message in body, got %q", bodyStr)
	}
}

func TestRecoverer_CatchesNilPanic(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// A nil panic should still be recoverable without crashing
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("panic escaped the Recoverer: %v", rec)
		}
	}()

	handler.ServeHTTP(w, r)

	// With panic(nil), recover() returns nil, so the Recoverer will
	// not trigger its error path. The response depends on whether
	// the handler wrote anything before panicking. This test just
	// ensures no crash.
}

// ---------------------------------------------------------------------------
// RequestLogger (smoke test — verifies it does not break the handler)
// ---------------------------------------------------------------------------

func TestRequestLogger_PassesThrough(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))

	r := httptest.NewRequest(http.MethodPost, "/api/projects?foo=bar", nil)
	r.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "created" {
		t.Errorf("expected 'created', got %q", string(body))
	}
}

// ---------------------------------------------------------------------------
// responseWriter wrapper
// ---------------------------------------------------------------------------

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rw := newResponseWriter(httptest.NewRecorder())
	if rw.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rw.statusCode)
	}
}

func TestResponseWriter_CapturesStatus(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := newResponseWriter(inner)
	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected captured status 404, got %d", rw.statusCode)
	}

	// Double write should be ignored
	rw.WriteHeader(http.StatusOK)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected status to remain 404 after double WriteHeader, got %d", rw.statusCode)
	}
}

func TestResponseWriter_CapturesBytesWritten(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := newResponseWriter(inner)

	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if rw.bytesWritten != 5 {
		t.Errorf("expected bytesWritten=5, got %d", rw.bytesWritten)
	}

	n2, _ := rw.Write([]byte(" world"))
	if rw.bytesWritten != 5+n2 {
		t.Errorf("expected bytesWritten=%d, got %d", 5+n2, rw.bytesWritten)
	}
}

func TestResponseWriter_Flush(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := newResponseWriter(inner)

	// Flush should not panic even if underlying writer supports it
	rw.Flush()

	// Verify the inner recorder was flushed
	if !inner.Flushed {
		t.Error("expected inner recorder to be flushed")
	}
}

// ---------------------------------------------------------------------------
// RateLimiter (smoke test — just verify it returns a valid middleware)
// ---------------------------------------------------------------------------

func TestRateLimiter_ReturnsMiddleware(t *testing.T) {
	mw := RateLimiter(100)
	if mw == nil {
		t.Fatal("expected non-nil middleware from RateLimiter")
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on first request, got %d", resp.StatusCode)
	}
}

func TestRateLimiter_ZeroRPS_UsesDefault(t *testing.T) {
	// Should not panic with zero or negative RPS
	mw := RateLimiter(0)
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:9999"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRateLimiter_NegativeRPS_UsesDefault(t *testing.T) {
	mw := RateLimiter(-5)
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}
