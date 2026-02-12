package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// IssueToken 使用 local 配置签发 JWT，供登录/注册使用
// rememberMe: true 表示 7 天，false 表示 24 小时
func IssueToken(cfg Config, userID, email string, rememberMe bool) (token string, expiresIn int64, err error) {
	cfg.Defaults()
	if cfg.Mode != "local" || cfg.AccessSecret == "" {
		return "", 0, ErrConfig
	}
	var expSec int64 = 86400 // 24h
	if rememberMe {
		expSec = 604800 // 7d
	}
	now := time.Now()
	claims := jwt.MapClaims{
		cfg.UserIDClaim: userID,
		"email":         email,
		"exp":           now.Add(time.Duration(expSec) * time.Second).Unix(),
		"iat":           now.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return "", 0, err
	}
	return token, expSec, nil
}
