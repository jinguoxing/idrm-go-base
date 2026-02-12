package ory

import (
	"context"
	"net/http"

	"github.com/jinguoxing/idrm-go-base/auth"
	"github.com/zeromicro/go-zero/rest"
)

// OathkeeperMiddleware 创建一个信任网关 Header 的中间件
func OathkeeperMiddleware() rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 从 Oathkeeper 转发的 Header 中获取用户 ID
			// 默认 Header 为 X-User-Id，可在 Oathkeeper 中配置
			userID := r.Header.Get("X-User-Id")
			if userID != "" {
				ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 如果没有 X-User-Id，可能是匿名请求或未经过认证
			// 由后续逻辑决定是否阻断
			next.ServeHTTP(w, r)
		}
	}
}
