package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Recovery recovers from panics and returns 500 error
func Recovery() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log panic with stack trace
					logx.WithContext(r.Context()).Errorw("Panic recovered",
						logx.Field("error", err),
						logx.Field("stack", string(debug.Stack())),
						logx.Field("method", r.Method),
						logx.Field("path", r.URL.Path),
						logx.Field("request_id", GetRequestID(r.Context())),
					)

					// Return 500 error
					httpx.Error(w, fmt.Errorf("internal server error"))
				}
			}()

			next(w, r)
		}
	}
}
