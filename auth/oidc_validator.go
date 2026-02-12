package auth

import (
	"context"
	"fmt"
)

// oidcValidator 使用 JWKS 验签（预留，可接入 Hydra）
type oidcValidator struct {
	cfg Config
}

func newOIDCValidator(cfg Config) (Validator, error) {
	// TODO: 实现 JWKS 拉取与缓存，RS256/ES256 验签
	// 可参考: golang.org/x/oauth2/jws, github.com/lestrrat-go/jwx/jwk
	return &oidcValidator{cfg: cfg}, nil
}

func (v *oidcValidator) ValidateToken(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	// 占位：后续接入 JWKS URL 与 Issuer 校验
	return nil, fmt.Errorf("auth: oidc mode not implemented yet, use mode=local or implement JWKS validation for %s", v.cfg.JWKSURL)
}
