package audit

import (
	"context"
	"net/http"
	"time"
)

// Helper 审计日志辅助结构
type Helper struct {
	ctx       context.Context
	log       AuditLog
	startTime time.Time
}

// NewHelper 创建审计日志辅助器
func NewHelper(ctx context.Context) *Helper {
	return &Helper{
		ctx:       ctx,
		log:       AuditLog{},
		startTime: time.Now(),
	}
}

// WithAction 设置操作类型
func (h *Helper) WithAction(action string) *Helper {
	h.log.Action = action
	return h
}

// WithResource 设置资源类型
func (h *Helper) WithResource(resource string) *Helper {
	h.log.Resource = resource
	return h
}

// WithUser 设置用户信息
func (h *Helper) WithUser(userID, username string) *Helper {
	h.log.UserID = userID
	h.log.Username = username
	return h
}

// WithIP 设置IP地址
func (h *Helper) WithIP(ip string) *Helper {
	h.log.IP = ip
	return h
}

// WithRequest 设置请求信息
func (h *Helper) WithRequest(req *http.Request) *Helper {
	if req != nil {
		h.log.Method = req.Method
		h.log.Path = req.URL.Path
		h.log.IP = req.RemoteAddr
	}
	return h
}

// WithBefore 设置操作前数据
func (h *Helper) WithBefore(before interface{}) *Helper {
	h.log.Before = before
	return h
}

// WithAfter 设置操作后数据
func (h *Helper) WithAfter(after interface{}) *Helper {
	h.log.After = after
	return h
}

// WithExtra 设置扩展字段
func (h *Helper) WithExtra(key string, value interface{}) *Helper {
	if h.log.Extra == nil {
		h.log.Extra = make(map[string]interface{})
	}
	h.log.Extra[key] = value
	return h
}

// Success 记录成功的审计日志
func (h *Helper) Success() {
	h.log.Success = true
	LogWithDuration(h.ctx, h.log, h.startTime)
}

// Fail 记录失败的审计日志
func (h *Helper) Fail(err error) {
	h.log.Success = false
	if err != nil {
		h.log.Error = err.Error()
	}
	LogWithDuration(h.ctx, h.log, h.startTime)
}

// SuccessOrFail 根据错误自动判断成功或失败
func (h *Helper) SuccessOrFail(err error) {
	if err != nil {
		h.Fail(err)
	} else {
		h.Success()
	}
}
