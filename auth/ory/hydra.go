package ory

import (
	"context"
	"fmt"
)

// OAuthProvider OAuth2 提供商接口
type OAuthProvider interface {
	// GetLoginRequest 获取登录请求详情 (用于自定义登录页)
	GetLoginRequest(ctx context.Context, challenge string) (*LoginRequest, error)
	// AcceptLoginRequest 接受登录请求并返回重定向 URL
	AcceptLoginRequest(ctx context.Context, challenge string, subject string) (string, error)
}

type LoginRequest struct {
	Challenge         string   `json:"challenge"`
	Subject           string   `json:"subject"`
	RequestURL        string   `json:"request_url"`
	Skip              bool     `json:"skip"`
	RequestedScope    []string `json:"requested_scope"`
	RequestedAudience []string `json:"requested_audience"`
}

type hydraClient struct {
	adminURL string
}

// NewHydraProvider 创建 Hydra 客户端
func NewHydraProvider(cfg Config) OAuthProvider {
	return &hydraClient{
		adminURL: cfg.HydraAdminURL,
	}
}

func (c *hydraClient) GetLoginRequest(ctx context.Context, challenge string) (*LoginRequest, error) {
	// TODO: 使用 ory-hydra-client-go 实现
	// client.AdminApi.GetLoginRequest(ctx).LoginChallenge(challenge).Execute()
	return nil, fmt.Errorf("ory: hydra integration not implemented yet")
}

func (c *hydraClient) AcceptLoginRequest(ctx context.Context, challenge string, subject string) (string, error) {
	// TODO: 使用 ory-hydra-client-go 实现
	// client.AdminApi.AcceptLoginRequest(ctx).LoginChallenge(challenge).AcceptLoginRequest(...).Execute()
	return "", fmt.Errorf("ory: hydra integration not implemented yet")
}
