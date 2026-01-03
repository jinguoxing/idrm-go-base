package middleware

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// Logger logs HTTP requests with detailed information
func Logger() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Execute request
			next(sw, r)

			// Log request details
			duration := time.Since(start)
			logx.WithContext(r.Context()).Infow("HTTP Request",
				logx.Field("method", r.Method),
				logx.Field("path", r.URL.Path),
				logx.Field("query", r.URL.RawQuery),
				logx.Field("status", sw.statusCode),
				logx.Field("duration_ms", duration.Milliseconds()),
				logx.Field("remote_addr", r.RemoteAddr),
				logx.Field("user_agent", r.UserAgent()),
				logx.Field("request_id", GetRequestID(r.Context())),
			)
		}
	}
}

// statusWriter wraps ResponseWriter to capture status code
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
