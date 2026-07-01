package telemetry

import (
	"context"

	"github.com/jinguoxing/idrm-go-base/telemetry/audit"
	"github.com/jinguoxing/idrm-go-base/telemetry/log"
	"github.com/jinguoxing/idrm-go-base/telemetry/metrics"
	"github.com/jinguoxing/idrm-go-base/telemetry/trace"

	"github.com/zeromicro/go-zero/core/logx"
)

// Init 初始化 Telemetry 系统（一站式初始化）
func Init(config Config) error {
	// 1. 初始化日志系统
	logConfig := log.LogConfig{
		Level:         config.Log.Level,
		Mode:          config.Log.Mode,
		Path:          config.Log.Path,
		KeepDays:      config.Log.KeepDays,
		RemoteEnabled: config.Log.RemoteEnabled,
		RemoteUrl:     config.Log.RemoteUrl,
		RemoteBatch:   config.Log.RemoteBatch,
		RemoteTimeout: config.Log.RemoteTimeout,
	}
	log.Init(logConfig, config.ServiceName)
	logx.Infof("Telemetry 初始化: %s v%s (%s)",
		config.ServiceName, config.ServiceVersion, config.Environment)

	// 2. 初始化链路追踪
	traceConfig := trace.TraceConfig{
		Enabled:    config.Trace.Enabled,
		Endpoint:   config.Trace.Endpoint,
		Protocol:   config.Trace.Protocol,
		Sampler:    config.Trace.Sampler,
		TLSEnabled: config.Trace.TLSEnabled,
		Headers:    config.Trace.Headers,
		Batcher:    config.Trace.Batcher,
	}
	if err := trace.Init(traceConfig, config.ServiceName, config.ServiceVersion, config.Environment); err != nil {
		logx.Errorf("链路追踪初始化失败: %v", err)
		return err
	}

	// 3. 初始化指标
	metricsConfig := metrics.MetricsConfig{
		Enabled:    config.Metrics.Enabled,
		Endpoint:   config.Metrics.Endpoint,
		Protocol:   config.Metrics.Protocol,
		Interval:   config.Metrics.Interval,
		Timeout:    config.Metrics.Timeout,
		TLSEnabled: config.Metrics.TLSEnabled,
		Headers:    config.Metrics.Headers,
	}
	if err := metrics.Init(metricsConfig, config.ServiceName, config.ServiceVersion, config.Environment); err != nil {
		logx.Errorf("Metrics 初始化失败: %v", err)
		return err
	}

	// 4. 初始化审计日志
	auditConfig := audit.AuditConfig{
		Enabled: config.Audit.Enabled,
		Url:     config.Audit.Url,
		Buffer:  config.Audit.Buffer,
	}
	audit.Init(auditConfig, config.ServiceName)

	logx.Info("Telemetry 系统初始化完成")
	return nil
}

// Close 关闭 Telemetry 系统
func Close(ctx context.Context) {
	logx.Info("正在关闭 Telemetry 系统...")

	// 关闭审计日志
	audit.Close()

	// 关闭指标
	if err := metrics.Close(ctx); err != nil {
		logx.Errorf("关闭 Metrics 失败: %v", err)
	}

	// 关闭链路追踪
	trace.Close(ctx)

	// 关闭日志系统（最后关闭）
	log.Close()
}
