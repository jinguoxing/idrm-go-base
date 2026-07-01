package telemetry

// Config Telemetry配置
type Config struct {
	// 服务基本信息
	ServiceName    string `json:",default=idrm-api"`
	ServiceVersion string `json:",default=1.0.0"`
	Environment    string `json:",default=dev"` // dev/test/prod

	// 日志配置
	Log LogConfig

	// 链路追踪配置
	Trace TraceConfig

	// 指标配置
	Metrics MetricsConfig

	// 审计日志配置
	Audit AuditConfig
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `json:",default=info"` // trace/debug/info/warn/error/fatal
	Mode     string `json:",default=file"` // console/file/volume
	Path     string `json:",default=logs"`
	KeepDays int    `json:",default=7"`

	// 远程日志上报
	RemoteEnabled bool   `json:",default=false"`
	RemoteUrl     string `json:",optional"`    // 远程日志接收地址
	RemoteBatch   int    `json:",default=100"` // 批量发送数量
	RemoteTimeout int    `json:",default=5"`   // 超时时间(秒)
}

// TraceConfig 链路追踪配置
type TraceConfig struct {
	Enabled    bool              `json:",default=true"`
	Endpoint   string            `json:",default=localhost:4317"` // OTLP endpoint，gRPC 默认 4317
	Protocol   string            `json:",default=grpc"`           // otlp 传输协议：grpc 或 http
	Sampler    float64           `json:",default=1.0"`            // 采样率 0.0-1.0
	TLSEnabled bool              `json:",default=false"`          // 启用 TLS 连接 Collector
	Headers    map[string]string `json:",optional"`               // OTLP 导出请求头（如认证 token）
	Batcher    string            `json:",default=otlp"`           // 保留字段，当前固定使用 OTLP
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled        bool              `json:",default=false"`
	Endpoint       string            `json:",default=localhost:4317"` // OTLP endpoint，gRPC 默认 4317
	Protocol       string            `json:",default=grpc"`           // otlp 传输协议：grpc 或 http
	Interval       int               `json:",default=15"`             // 导出间隔(秒)
	Timeout        int               `json:",default=10"`             // 导出超时(秒)
	TLSEnabled     bool              `json:",default=false"`          // 启用 TLS 连接 Collector
	Headers        map[string]string `json:",optional"`               // OTLP 导出请求头（如认证 token）
	RuntimeEnabled bool              `json:",default=false"`          // 启用 Go runtime 指标
}

// AuditConfig 审计日志配置
type AuditConfig struct {
	Enabled bool   `json:",default=false"`
	Url     string `json:",optional"`    // 审计日志上报地址
	Buffer  int    `json:",default=100"` // 缓冲区大小
}
