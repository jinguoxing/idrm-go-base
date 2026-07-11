package log

import (
	"fmt"
	"io"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	remoteWriter *RemoteWriter
)

// LogConfig 日志配置
type LogConfig struct {
	Level    string
	Mode     string
	Path     string
	KeepDays int

	Rotation         string
	MaxSize          int
	MaxBackups       int
	MaxContentLength uint32

	RemoteEnabled bool
	RemoteUrl     string
	RemoteBatch   int
	RemoteTimeout int
}

// SetUp initializes the logging system and returns configuration errors to the caller.
func SetUp(config LogConfig, serviceName string) error {
	logConf, err := buildLogConf(config, serviceName)
	if err != nil {
		return err
	}

	if err := logx.SetUp(logConf); err != nil {
		return err
	}

	// If remote logging is enabled, retain the writer for callers that use it explicitly.
	if config.RemoteEnabled && config.RemoteUrl != "" {
		timeout := time.Duration(config.RemoteTimeout) * time.Second
		remoteWriter = NewRemoteWriter(
			serviceName,
			config.RemoteUrl,
			config.RemoteBatch,
			timeout,
		)

		setupRemoteWriter(remoteWriter)
	}

	logx.Infof("日志系统初始化完成 [mode=%s, level=%s, rotation=%s, remote=%v]",
		config.Mode, config.Level, logConf.Rotation, config.RemoteEnabled)
	return nil
}

// Init initializes the logging system and preserves the v0.2.x panic-on-error API.
// New callers should prefer SetUp so that configuration errors can be handled explicitly.
func Init(config LogConfig, serviceName string) {
	if err := SetUp(config, serviceName); err != nil {
		panic(err)
	}
}

func buildLogConf(config LogConfig, serviceName string) (logx.LogConf, error) {
	rotation := config.Rotation
	if rotation == "" {
		rotation = "daily"
	}

	if rotation != "daily" && rotation != "size" {
		return logx.LogConf{}, fmt.Errorf("invalid log rotation %q: must be daily or size", rotation)
	}
	if config.MaxSize < 0 {
		return logx.LogConf{}, fmt.Errorf("invalid log max size %d: must not be negative", config.MaxSize)
	}
	if config.MaxBackups < 0 {
		return logx.LogConf{}, fmt.Errorf("invalid log max backups %d: must not be negative", config.MaxBackups)
	}
	if rotation == "size" && config.MaxSize == 0 {
		return logx.LogConf{}, fmt.Errorf("log max size must be greater than zero when rotation is size")
	}
	if (config.Mode == "file" || config.Mode == "volume") && config.Path == "" {
		return logx.LogConf{}, fmt.Errorf("log path must not be empty when mode is %s", config.Mode)
	}

	return logx.LogConf{
		ServiceName:      serviceName,
		Mode:             config.Mode,
		Level:            config.Level,
		Path:             config.Path,
		KeepDays:         config.KeepDays,
		Compress:         true,
		Rotation:         rotation,
		MaxSize:          config.MaxSize,
		MaxBackups:       config.MaxBackups,
		MaxContentLength: config.MaxContentLength,
	}, nil
}

// setupRemoteWriter 设置远程日志写入器
// 由于 go-zero logx 的限制，我们需要通过其他方式集成
func setupRemoteWriter(writer io.Writer) {
	// go-zero 1.9.x 支持通过环境变量或直接修改内部writer
	// 这里我们简化处理，在业务代码中可以手动调用 remoteWriter
	// 或者通过 logx 的 stat/slow log 功能扩展

	// 临时方案：保存 writer 供外部使用
	// 实际使用时可以包装 logx 的方法
}

// Close 关闭日志系统
func Close() {
	if remoteWriter != nil {
		remoteWriter.Close()
	}
	logx.Close()
}

// GetRemoteWriter 获取远程日志写入器（供测试使用）
func GetRemoteWriter() *RemoteWriter {
	return remoteWriter
}
