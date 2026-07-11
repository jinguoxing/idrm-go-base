package telemetry

import (
	"context"
	"fmt"

	"github.com/jinguoxing/idrm-go-base/telemetry/audit"
	"github.com/jinguoxing/idrm-go-base/telemetry/log"
	"github.com/jinguoxing/idrm-go-base/telemetry/trace"

	"github.com/zeromicro/go-zero/core/logx"
)

// Init 初始化 Telemetry 系统（一站式初始化）
func Init(config Config) error {
	// 1. 初始化日志系统
	if err := log.SetUp(toLogConfig(config), config.ServiceName); err != nil {
		return fmt.Errorf("init log: %w", err)
	}
	logx.Infof("Telemetry 初始化: %s v%s (%s)",
		config.ServiceName, config.ServiceVersion, config.Environment)

	// 2. 初始化链路追踪
	traceConfig := trace.TraceConfig{
		Enabled:  config.Trace.Enabled,
		Endpoint: config.Trace.Endpoint,
		Sampler:  config.Trace.Sampler,
		Batcher:  config.Trace.Batcher,
	}
	if err := trace.Init(traceConfig, config.ServiceName, config.ServiceVersion, config.Environment); err != nil {
		logx.Errorf("链路追踪初始化失败: %v", err)
		return err
	}

	// 3. 初始化审计日志
	auditConfig := audit.AuditConfig{
		Enabled: config.Audit.Enabled,
		Url:     config.Audit.Url,
		Buffer:  config.Audit.Buffer,
	}
	audit.Init(auditConfig, config.ServiceName)

	logx.Info("Telemetry 系统初始化完成")
	return nil
}

func toLogConfig(config Config) log.LogConfig {
	return log.LogConfig{
		Level:            config.Log.Level,
		Mode:             config.Log.Mode,
		Path:             config.Log.Path,
		KeepDays:         config.Log.KeepDays,
		Rotation:         config.Log.Rotation,
		MaxSize:          config.Log.MaxSize,
		MaxBackups:       config.Log.MaxBackups,
		MaxContentLength: config.Log.MaxContentLength,
		RemoteEnabled:    config.Log.RemoteEnabled,
		RemoteUrl:        config.Log.RemoteUrl,
		RemoteBatch:      config.Log.RemoteBatch,
		RemoteTimeout:    config.Log.RemoteTimeout,
	}
}

// Close 关闭 Telemetry 系统
func Close(ctx context.Context) {
	logx.Info("正在关闭 Telemetry 系统...")

	// 关闭审计日志
	audit.Close()

	// 关闭链路追踪
	trace.Close(ctx)

	// 关闭日志系统（最后关闭）
	log.Close()
}
