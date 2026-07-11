package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestLogConfigDefaults(t *testing.T) {
	var config Config
	require.NoError(t, conf.LoadFromYamlBytes([]byte("Log: {}\n"), &config))

	require.Equal(t, "daily", config.Log.Rotation)
	require.Zero(t, config.Log.MaxSize)
	require.Zero(t, config.Log.MaxBackups)
	require.Zero(t, config.Log.MaxContentLength)
}

func TestLogConfigRejectsInvalidRotation(t *testing.T) {
	var config Config
	err := conf.LoadFromYamlBytes([]byte("Log:\n  Rotation: hourly\n"), &config)
	require.Error(t, err)
}
