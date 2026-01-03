package audit

import "time"

// AuditLog 审计日志结构
type AuditLog struct {
	// 基础信息
	Timestamp   time.Time `json:"timestamp"`
	ServiceName string    `json:"service_name"`

	// 操作信息
	Action   string `json:"action"`   // 操作类型：create/update/delete/query/login/logout
	Resource string `json:"resource"` // 资源类型：category/user/order/config

	// 用户信息
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	IP       string `json:"ip,omitempty"`

	// 请求信息
	Method  string `json:"method,omitempty"`   // HTTP Method
	Path    string `json:"path,omitempty"`     // 请求路径
	TraceID string `json:"trace_id,omitempty"` // 链路ID

	// 操作详情
	Before interface{} `json:"before,omitempty"` // 操作前数据
	After  interface{} `json:"after,omitempty"`  // 操作后数据

	// 结果
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Duration int64  `json:"duration,omitempty"` // 执行时长(ms)

	// 扩展字段
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// AuditConfig 审计日志配置
type AuditConfig struct {
	Enabled bool
	Url     string
	Buffer  int
}

// 常用操作类型
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionQuery  = "query"
	ActionLogin  = "login"
	ActionLogout = "logout"
	ActionExport = "export"
	ActionImport = "import"
)

// 常用资源类型
const (
	ResourceCategory = "category"
	ResourceUser     = "user"
	ResourceRole     = "role"
	ResourceConfig   = "config"
)
