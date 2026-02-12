package auth

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 统一 401 响应体
type errUnauthorized struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *errUnauthorized) Error() string {
	return e.Message
}

// MustJWT 必须携带有效 JWT，否则返回 401
func MustJWT(v Validator) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, ok := ExtractBearer(r.Header.Get("Authorization"))
			if !ok || token == "" {
				httpx.WriteJsonCtx(r.Context(), w, 401, &errUnauthorized{Code: 401, Message: "未提供认证信息"})
				return
			}
			claims, err := v.ValidateToken(r.Context(), token)
			if err != nil {
				httpx.WriteJsonCtx(r.Context(), w, 401, &errUnauthorized{Code: 401, Message: "Token 无效或已过期"})
				return
			}
			userID := getStringClaim(claims, "user_id")
			if userID == "" {
				userID = getStringClaim(claims, "sub")
			}
			if userID == "" {
				httpx.WriteJsonCtx(r.Context(), w, 401, &errUnauthorized{Code: 401, Message: "Token 中缺少用户标识"})
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// OptionalJWT 若有 Authorization Bearer 则校验并注入 context，否则直接放行
func OptionalJWT(v Validator) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, ok := ExtractBearer(r.Header.Get("Authorization"))
			if !ok || token == "" {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := v.ValidateToken(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			userID := getStringClaim(claims, "user_id")
			if userID == "" {
				userID = getStringClaim(claims, "sub")
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

func getStringClaim(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
