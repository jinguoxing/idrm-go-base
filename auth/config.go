package auth

// Config 认证配置，支持本地 HS256 与 OIDC/JWKS 两种模式
type Config struct {
	// Mode 认证模式: "local" = HS256 + AccessSecret; "oidc" = JWKS + Issuer
	Mode string `json:"mode" yaml:"mode"`

	// ----- Mode == "local" 时使用 -----
	AccessSecret string `json:"access_secret" yaml:"access_secret"`
	AccessExpire int64  `json:"access_expire" yaml:"access_expire"` // 秒，可选

	// ----- Mode == "oidc" 时使用 -----
	JWKSURL  string   `json:"jwks_url" yaml:"jwks_url"`
	Issuer   string   `json:"issuer" yaml:"issuer"`
	Audience []string `json:"audience" yaml:"audience"` // 可选

	// ----- 通用 -----
	// UserIDClaim 从 JWT 中取用户 ID 的 claim 名，默认 "user_id"（OIDC 常用 "sub"）
	UserIDClaim string `json:"user_id_claim" yaml:"user_id_claim"`
}

// Defaults 填充默认值
func (c *Config) Defaults() {
	if c.Mode == "" {
		c.Mode = "local"
	}
	if c.UserIDClaim == "" {
		c.UserIDClaim = "user_id"
	}
}
