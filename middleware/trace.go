package middleware

import (
	"net/http"

	"github.com/jinguoxing/idrm-go-base/telemetry/trace"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Trace creates OpenTelemetry spans for HTTP requests
func Trace() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Create Server Span
			ctx, span := trace.StartServer(r.Context(), r.URL.Path,
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.host", r.Host),
				attribute.String("http.scheme", getScheme(r)),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.client_ip", getClientIP(r)),
				attribute.String("http.request_id", GetRequestID(r.Context())),
			)
			defer span.End()

			// Update request context
			r = r.WithContext(ctx)

			// Wrap response writer to capture status code
			sw := &traceStatusWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Execute next handler
			next(sw, r)

			// Record response information
			span.SetAttributes(
				attribute.Int("http.status_code", sw.statusCode),
			)

			// Mark error if status >= 400
			if sw.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(sw.statusCode))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
		}
	}
}

// traceStatusWriter wraps ResponseWriter to capture status code
type traceStatusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *traceStatusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// getScheme determines HTTP or HTTPS
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// getClientIP extracts real client IP
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	// Check X-Real-IP header
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
