package middleware

import (
	"net/http"
	"strings"
)

// CorsMiddleware CORS中间件
func CorsMiddleware(allowOrigins, allowMethods, allowHeaders []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 设置CORS头
			origin := r.Header.Get("Origin")
			if origin != "" && isAllowedOrigin(origin, allowOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// 处理预检请求
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAllowedOrigin 检查origin是否被允许
func isAllowedOrigin(origin string, allowOrigins []string) bool {
	for _, allowed := range allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
