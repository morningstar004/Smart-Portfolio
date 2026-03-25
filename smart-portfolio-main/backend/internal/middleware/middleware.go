package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/httputil"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// contextKey is an unexported type used for context value keys in this package.
// Using a dedicated type prevents collisions with keys defined in other packages.
type contextKey string

const (
	// RequestIDKey is the context key under which the unique request ID is stored.
	RequestIDKey contextKey = "request_id"
)

// RequestID generates a unique UUID for every incoming request and injects it
// into the request context and the X-Request-ID response header. Downstream
// handlers and services can retrieve it via GetRequestID(ctx) for correlated
// logging.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		w.Header().Set("X-Request-ID", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from the context. Returns an empty
// string if no request ID is present (e.g. in background goroutines that
// were not spawned from an HTTP request).
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// responseWriter wraps http.ResponseWriter to capture the status code and
// bytes written for logging purposes. It implements http.ResponseWriter,
// http.Flusher, and http.Hijacker where the underlying writer supports them.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Flush implements http.Flusher so SSE streaming works through this wrapper.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RequestLogger logs structured information about every HTTP request including
// method, path, status code, latency, and bytes written. It uses zerolog for
// high-performance structured logging.
//
// Log levels are selected based on the response status code:
//   - 5xx → Error
//   - 4xx → Warn
//   - everything else → Info
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriter(w)

		// Proceed with the request.
		next.ServeHTTP(wrapped, r)

		elapsed := time.Since(start)
		requestID := GetRequestID(r.Context())

		var event *zerolog.Event
		switch {
		case wrapped.statusCode >= 500:
			event = log.Error()
		case wrapped.statusCode >= 400:
			event = log.Warn()
		default:
			event = log.Info()
		}

		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Int("status", wrapped.statusCode).
			Int("bytes", wrapped.bytesWritten).
			Dur("latency", elapsed).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Str("request_id", requestID).
			Msg("http request")
	})
}

// Recoverer catches panics from downstream handlers and middleware, logs the
// stack trace at error level, and returns a 500 Internal Server Error JSON
// response to the client. Without this middleware, a panic would crash the
// entire server process.
//
// The recovered panic value and full stack trace are logged but never exposed
// to the client to prevent information leakage.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := GetRequestID(r.Context())
				stack := string(debug.Stack())

				log.Error().
					Str("request_id", requestID).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Interface("panic", rec).
					Str("stack", stack).
					Msg("recovered from panic")

				httputil.WriteError(w, http.StatusInternalServerError,
					"an internal error occurred — please try again later")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RateLimiter returns a middleware that limits requests per IP address using a
// sliding window algorithm. It returns HTTP 429 Too Many Requests when the
// limit is exceeded, with a JSON error body and standard Retry-After header.
//
// Parameters:
//   - requestsPerSecond: the maximum number of requests allowed per second per IP.
//   - burst: not used directly by httprate, but we pass requestsPerSecond * window
//     for a smooth experience.
func RateLimiter(requestsPerSecond int) func(http.Handler) http.Handler {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 10
	}

	// Allow requestsPerSecond requests per 1-second window per IP.
	limiter := httprate.Limit(
		requestsPerSecond,
		1*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Warn().
				Str("remote_addr", r.RemoteAddr).
				Str("path", r.URL.Path).
				Str("request_id", GetRequestID(r.Context())).
				Msg("rate limit exceeded")

			httputil.WriteError(w, http.StatusTooManyRequests,
				"rate limit exceeded — please slow down and try again shortly")
		})),
	)

	return limiter
}

// SecurityHeaders adds common security-related HTTP headers to every response.
// These headers help protect against common web vulnerabilities and are
// recommended by OWASP.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
	})
}

// ContentTypeJSON sets the Content-Type header to application/json for all
// responses passing through this middleware. Useful when mounted on API-only
// route groups where every response is JSON.
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

// Healthcheck is a standalone handler (not middleware) that returns a simple
// 200 OK response. It is used for liveness probes by load balancers and
// orchestration platforms (e.g. Kubernetes, Railway, Render).
func Healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// AdminAuth returns a middleware that protects admin-only endpoints with an
// API key. The key must be provided in the X-Admin-Key request header or as
// the Bearer token in the Authorization header. If the configured adminKey is
// empty, the middleware is a no-op (all requests are allowed) so the app
// remains usable during development without setting a key.
//
// The comparison uses constant-time equality to prevent timing side-channel
// attacks against the API key.
func AdminAuth(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no admin key is configured, skip authentication entirely.
			// This keeps local development frictionless while ensuring
			// production deployments can enforce the key.
			if adminKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Accept the key from either X-Admin-Key or Authorization: Bearer <key>
			provided := r.Header.Get("X-Admin-Key")
			if provided == "" {
				auth := r.Header.Get("Authorization")
				if len(auth) > 7 && auth[:7] == "Bearer " {
					provided = auth[7:]
				}
			}

			if provided == "" {
				log.Warn().
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Str("request_id", GetRequestID(r.Context())).
					Msg("admin_auth: missing authentication — access denied")
				httputil.WriteError(w, http.StatusUnauthorized, "authentication required — provide X-Admin-Key header or Authorization: Bearer <key>")
				return
			}

			if subtle.ConstantTimeCompare([]byte(provided), []byte(adminKey)) != 1 {
				log.Warn().
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Str("request_id", GetRequestID(r.Context())).
					Msg("admin_auth: invalid API key — access denied")
				httputil.WriteError(w, http.StatusForbidden, "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize returns a middleware that limits the request body size to the
// given number of bytes. If the body exceeds the limit, the request is
// rejected with a 413 Request Entity Too Large response. This prevents
// abuse on endpoints that accept JSON payloads.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
