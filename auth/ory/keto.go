package ory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// PermissionEngine 权限引擎接口
type PermissionEngine interface {
	// Check 检查权限 (Subject 是否对 Object 在 Namespace 下拥有 Relation 关系)
	Check(ctx context.Context, namespace, object, relation, subjectID string) (bool, error)
}

type ketoManager struct {
	readURL string
}

// NewKetoEngine 创建 Keto 权限引擎
func NewKetoEngine(cfg Config) PermissionEngine {
	return &ketoManager{
		readURL: cfg.KetoReadURL,
	}
}

// checkPayload Keto Check API 请求体
type checkPayload struct {
	Namespace string `json:"namespace"`
	Object    string `json:"object"`
	Relation  string `json:"relation"`
	Subject   struct {
		ID string `json:"id"`
	} `json:"subject"`
}

type checkResponse struct {
	Allowed bool `json:"allowed"`
}

func (m *ketoManager) Check(ctx context.Context, namespace, object, relation, subjectID string) (bool, error) {
	if m.readURL == "" {
		return false, fmt.Errorf("ory: KetoReadURL is not configured")
	}

	url := fmt.Sprintf("%s/relation-tuples/check", m.readURL)

	payload := checkPayload{
		Namespace: namespace,
		Object:    object,
		Relation:  relation,
	}
	payload.Subject.ID = subjectID

	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpc.DoRequest(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("check failed, status: %d", resp.StatusCode)
	}

	var res checkResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}

	return res.Allowed, nil
}
