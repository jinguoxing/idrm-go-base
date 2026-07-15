package log

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

var _ func(LogConfig, string) = Init

func TestBuildLogConfMapsFields(t *testing.T) {
	config := LogConfig{
		Level:            "info",
		Mode:             "file",
		Path:             "logs/test",
		KeepDays:         7,
		Rotation:         "size",
		MaxSize:          128,
		MaxBackups:       12,
		MaxContentLength: 4096,
	}

	got, err := buildLogConf(config, "telemetry-test")
	require.NoError(t, err)
	require.Equal(t, "telemetry-test", got.ServiceName)
	require.Equal(t, config.Mode, got.Mode)
	require.Equal(t, config.Path, got.Path)
	require.Equal(t, config.KeepDays, got.KeepDays)
	require.True(t, got.Compress)
	require.Equal(t, config.Rotation, got.Rotation)
	require.Equal(t, config.MaxSize, got.MaxSize)
	require.Equal(t, config.MaxBackups, got.MaxBackups)
	require.Equal(t, config.MaxContentLength, got.MaxContentLength)
}

func TestBuildLogConfDefaultsRotation(t *testing.T) {
	got, err := buildLogConf(LogConfig{Mode: "console"}, "telemetry-test")
	require.NoError(t, err)
	require.Equal(t, "daily", got.Rotation)
}

func TestBuildLogConfRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config LogConfig
	}{
		{
			name:   "unknown rotation",
			config: LogConfig{Mode: "console", Rotation: "hourly"},
		},
		{
			name:   "negative max size",
			config: LogConfig{Mode: "console", MaxSize: -1},
		},
		{
			name:   "negative max backups",
			config: LogConfig{Mode: "console", MaxBackups: -1},
		},
		{
			name:   "size rotation without max size",
			config: LogConfig{Mode: "console", Rotation: "size"},
		},
		{
			name:   "file mode without path",
			config: LogConfig{Mode: "file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildLogConf(tt.config, "telemetry-test")
			require.Error(t, err)
		})
	}
}

func TestInitPreservesPanicOnInvalidConfiguration(t *testing.T) {
	require.Panics(t, func() {
		Init(LogConfig{Mode: "console", Rotation: "invalid"}, "telemetry-test")
	})
}

func TestLogIntegrationHelper(t *testing.T) {
	if os.Getenv("IDRM_LOG_TEST_HELPER") != "1" {
		return
	}

	path := os.Getenv("IDRM_LOG_TEST_PATH")
	if path == "" {
		t.Fatal("IDRM_LOG_TEST_PATH is required")
	}

	switch os.Getenv("IDRM_LOG_TEST_SCENARIO") {
	case "truncation":
		require.NoError(t, SetUp(LogConfig{
			Level:            "info",
			Mode:             "file",
			Path:             path,
			Rotation:         "daily",
			MaxContentLength: 10,
		}, "telemetry-test"))
		logx.Info(strings.Repeat("x", 11))
	case "field-boundary":
		require.NoError(t, SetUp(LogConfig{
			Level:            "info",
			Mode:             "file",
			Path:             path,
			Rotation:         "daily",
			MaxContentLength: 10,
		}, "telemetry-test"))
		logx.Infow("ok", logx.Field("payload", strings.Repeat("f", 11)))
	case "console-truncation":
		require.NoError(t, SetUp(LogConfig{
			Level:            "info",
			Mode:             "console",
			Rotation:         "daily",
			MaxContentLength: 10,
		}, "telemetry-test"))
		logx.Info(strings.Repeat("c", 11))
	case "rotation":
		require.NoError(t, SetUp(LogConfig{
			Level:      "info",
			Mode:       "file",
			Path:       path,
			Rotation:   "size",
			MaxSize:    1,
			MaxBackups: 2,
		}, "telemetry-test"))

		payload := strings.Repeat("r", 700*1024)
		for range 5 {
			logx.Info(payload)
		}
	default:
		t.Fatalf("unknown helper scenario %q", os.Getenv("IDRM_LOG_TEST_SCENARIO"))
	}

	logx.Close()
	// Rotated-file compression and cleanup run asynchronously in go-zero.
	time.Sleep(300 * time.Millisecond)
}

func TestSetUpTruncatesStringContent(t *testing.T) {
	path := t.TempDir()
	runLogHelper(t, "truncation", path)

	content, err := os.ReadFile(filepath.Join(path, "access.log"))
	require.NoError(t, err)

	var truncatedEntry struct {
		Content   string `json:"content"`
		Truncated bool   `json:"truncated"`
	}
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		var entry struct {
			Content   string `json:"content"`
			Truncated bool   `json:"truncated"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry.Truncated && entry.Content == strings.Repeat("x", 10) {
			truncatedEntry = entry
			break
		}
	}

	require.True(t, truncatedEntry.Truncated)
	require.Equal(t, strings.Repeat("x", 10), truncatedEntry.Content)
}

func TestSetUpDoesNotTruncateStructuredFields(t *testing.T) {
	path := t.TempDir()
	runLogHelper(t, "field-boundary", path)

	content, err := os.ReadFile(filepath.Join(path, "access.log"))
	require.NoError(t, err)

	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["payload"] == strings.Repeat("f", 11) {
			require.Equal(t, "ok", entry["content"])
			require.NotContains(t, entry, "truncated")
			return
		}
	}

	t.Fatal("structured log entry was not written")
}

func TestSetUpTruncatesStringContentInConsoleMode(t *testing.T) {
	output := runLogHelper(t, "console-truncation", t.TempDir())
	require.Contains(t, output, `"content":"cccccccccc"`)
	require.Contains(t, output, `"truncated":true`)
}

func TestSetUpRotatesFilesBySize(t *testing.T) {
	path := t.TempDir()
	runLogHelper(t, "rotation", path)

	backups, err := filepath.Glob(filepath.Join(path, "access-*.log*"))
	require.NoError(t, err)
	require.NotEmpty(t, backups)
	require.LessOrEqual(t, len(backups), 2)
}

func runLogHelper(t *testing.T, scenario, path string) string {
	t.Helper()

	command := exec.Command(os.Args[0], "-test.run=^TestLogIntegrationHelper$")
	command.Env = append(os.Environ(),
		"IDRM_LOG_TEST_HELPER=1",
		"IDRM_LOG_TEST_SCENARIO="+scenario,
		"IDRM_LOG_TEST_PATH="+path,
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "log helper failed: %s", fmt.Sprintf("%s", output))
	return string(output)
}
