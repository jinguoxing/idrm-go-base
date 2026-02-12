package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// Validator JWT 校验器
type Validator interface {
	ValidateToken(ctx context.Context, tokenString string) (claims map[string]interface{}, err error)
}

// localValidator HS256 本地密钥校验
type localValidator struct {
	secret      []byte
	userIDClaim string
}

// NewValidator 根据配置创建校验器
func NewValidator(cfg Config) (Validator, error) {
	cfg.Defaults()
	switch cfg.Mode {
	case "local":
		if cfg.AccessSecret == "" {
			return nil, fmt.Errorf("auth: AccessSecret required when mode=local")
		}
		return &localValidator{
			secret:      []byte(cfg.AccessSecret),
			userIDClaim: cfg.UserIDClaim,
		}, nil
	case "oidc":
		if cfg.JWKSURL == "" {
			return nil, fmt.Errorf("auth: JWKSURL required when mode=oidc")
		}
		return newOIDCValidator(cfg)
	default:
		return nil, fmt.Errorf("auth: unknown mode %q", cfg.Mode)
	}
}

func (v *localValidator) ValidateToken(_ context.Context, tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	// 转为 map[string]interface{} 便于注入 context
	out := make(map[string]interface{})
	for k, val := range claims {
		out[k] = val
	}
	return out, nil
}

// ExtractBearer 从 Authorization 头解析 Bearer token
func ExtractBearer(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if authHeader == "" || !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	return strings.TrimSpace(authHeader[len(prefix):]), true
}
