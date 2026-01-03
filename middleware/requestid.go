package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type requestIDKey struct{}

// RequestID generates and injects request ID into context
func RequestID() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get from header or generate new ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Set to context
			ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)

			// Set response header
			w.Header().Set("X-Request-ID", requestID)

			// Execute next handler
			next(w, r.WithContext(ctx))
		}
	}
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
