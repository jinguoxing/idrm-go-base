package ory

// Config Ory 组件配置
type Config struct {
	// KratosPublicURL Kratos Public API 地址 (用于会话验证)
	KratosPublicURL string `json:",optional"`
	// KratosAdminURL Kratos Admin API 地址 (用于管理用户)
	KratosAdminURL string `json:",optional"`

	// KetoReadURL Keto Read API 地址 (用于权限检查)
	KetoReadURL string `json:",optional"`
	// KetoWriteURL Keto Write API 地址 (用于权限写入)
	KetoWriteURL string `json:",optional"`

	// HydraPublicURL Hydra Public API 地址 (Consumer)
	HydraPublicURL string `json:",optional"`
	// HydraAdminURL Hydra Admin API 地址 (Provider)
	HydraAdminURL string `json:",optional"`
}
