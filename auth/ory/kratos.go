package ory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// IdentityManager 身份管理器接口
type IdentityManager interface {
	// ValidateSession 验证会话 (支持 Cookie 或 X-Session-Token)
	ValidateSession(ctx context.Context, sessionToken string) (*Session, error)
}

// Session Kratos 会话信息的简化定义
type Session struct {
	ID       string    `json:"id"`
	Active   bool      `json:"active"`
	Identity *Identity `json:"identity"`
}

// Identity Kratos 身份信息的简化定义
type Identity struct {
	ID        string                 `json:"id"`
	SchemaID  string                 `json:"schema_id"`
	Traits    map[string]interface{} `json:"traits"`
	UpdatedAt string                 `json:"updated_at"`
	CreatedAt string                 `json:"created_at"`
}

type kratosManager struct {
	publicURL string
}

// NewKratosManager 创建 Kratos 管理器
func NewKratosManager(cfg Config) IdentityManager {
	return &kratosManager{
		publicURL: cfg.KratosPublicURL,
	}
}

func (m *kratosManager) ValidateSession(ctx context.Context, sessionToken string) (*Session, error) {
	if m.publicURL == "" {
		return nil, fmt.Errorf("ory: KratosPublicURL is not configured")
	}

	url := fmt.Sprintf("%s/sessions/whoami", m.publicURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 自动识别 Token 类型
	// 如果包含 "ory_kratos_session", 认为是 Cookie
	// 否则认为是 X-Session-Token
	// 实际生产中建议明确区分，这里做简单兼容
	req.Header.Set("Accept", "application/json")
	if len(sessionToken) > 0 {
		req.Header.Set("X-Session-Token", sessionToken)
		req.Header.Set("Cookie", sessionToken) // 尝试作为 Cookie 发送 (Kratos 会自己解析有效的那个)
	}

	resp, err := httpc.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid session, status: %d", resp.StatusCode)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	if !session.Active {
		return nil, fmt.Errorf("session is inactive")
	}

	return &session, nil
}
