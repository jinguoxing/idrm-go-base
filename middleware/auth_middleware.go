package middleware

import (
	"net/http"
	"strings"

	"github.com/jinguoxing/idrm-go-base/errorx"
	"github.com/jinguoxing/idrm-go-base/response"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 获取Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, errorx.NewWithCode(errorx.ErrCodeUnauthorized))
				return
			}

			// 检查Bearer token格式
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, errorx.NewWithCode(errorx.ErrCodeTokenInvalid))
				return
			}

			token := parts[1]

			// TODO: 验证JWT token
			// 这里需要使用jwt库验证token
			// 验证成功后，可以将用户信息放入context
			_ = token

			// 调用下一个处理器
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuthMiddleware 可选认证中间件
func OptionalAuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 如果有token则验证，没有则跳过
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// TODO: 验证token并设置用户信息到context
			}

			next.ServeHTTP(w, r)
		})
	}
}
