package auth

import "errors"

var (
	errConfig = errors.New("auth: config must be mode=local with AccessSecret set")
	// ErrConfig 表示认证配置错误（如 IssueToken 时未设置 local + AccessSecret）
	ErrConfig = errConfig
)
