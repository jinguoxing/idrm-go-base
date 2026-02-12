package auth

import "context"

type contextKey string

const (
	UserIDKey contextKey = "user_id" // 当前用户 ID
	ClaimsKey contextKey = "claims"  // 完整 claims
)

// UserIDFromContext 从 context 获取当前用户 ID（JWT 校验后由中间件注入）
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(UserIDKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// ClaimsFromContext 从 context 获取完整 claims
func ClaimsFromContext(ctx context.Context) (map[string]interface{}, bool) {
	v := ctx.Value(ClaimsKey)
	if v == nil {
		return nil, false
	}
	m, ok := v.(map[string]interface{})
	return m, ok
}
