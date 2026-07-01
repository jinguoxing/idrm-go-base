package metrics

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled    bool              `json:",default=false"`
	Endpoint   string            `json:",default=localhost:4317"` // OTLP endpoint
	Protocol   string            `json:",default=grpc"`           // grpc 或 http
	Interval   int               `json:",default=15"`             // 导出间隔（秒）
	Timeout    int               `json:",default=10"`             // 导出超时（秒）
	TLSEnabled bool              `json:",default=false"`          // 启用 TLS 连接 Collector
	Headers    map[string]string `json:",optional"`               // OTLP 导出请求头（如认证 token）
}
