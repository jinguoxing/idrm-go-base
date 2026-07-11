package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToLogConfigMapsRotationAndContentSettings(t *testing.T) {
	config := Config{
		Log: LogConfig{
			Level:            "info",
			Mode:             "file",
			Path:             "logs/test",
			KeepDays:         7,
			Rotation:         "size",
			MaxSize:          128,
			MaxBackups:       12,
			MaxContentLength: 4096,
			RemoteEnabled:    true,
			RemoteUrl:        "http://collector.test/logs",
			RemoteBatch:      100,
			RemoteTimeout:    5,
		},
	}

	got := toLogConfig(config)
	require.Equal(t, "size", got.Rotation)
	require.Equal(t, 128, got.MaxSize)
	require.Equal(t, 12, got.MaxBackups)
	require.EqualValues(t, 4096, got.MaxContentLength)
	require.Equal(t, config.Log.Path, got.Path)
	require.Equal(t, config.Log.KeepDays, got.KeepDays)
	require.Equal(t, config.Log.RemoteUrl, got.RemoteUrl)
}
