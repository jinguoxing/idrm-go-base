package log

import (
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

	RemoteEnabled bool
	RemoteUrl     string
	RemoteBatch   int
	RemoteTimeout int
}

// Init 初始化日志系统
func Init(config LogConfig, serviceName string) {
	// 1. 配置 go-zero logx
	logConf := logx.LogConf{
		ServiceName: serviceName,
		Mode:        config.Mode,
		Level:       config.Level,
		Path:        config.Path,
		KeepDays:    config.KeepDays,
		Compress:    true,
	}

	if err := logx.SetUp(logConf); err != nil {
		panic(err)
	}

	// 2. 如果启用远程日志，添加远程 Writer
	if config.RemoteEnabled && config.RemoteUrl != "" {
		timeout := time.Duration(config.RemoteTimeout) * time.Second
		remoteWriter = NewRemoteWriter(
			serviceName,
			config.RemoteUrl,
			config.RemoteBatch,
			timeout,
		)

		// 添加远程 Writer 到 logx
		// 注意：go-zero logx 需要通过 writer 方式添加
		// 这里我们使用 logx.AddWriter 的替代方案
		setupRemoteWriter(remoteWriter)
	}

	logx.Infof("日志系统初始化完成 [mode=%s, level=%s, remote=%v]",
		config.Mode, config.Level, config.RemoteEnabled)
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
