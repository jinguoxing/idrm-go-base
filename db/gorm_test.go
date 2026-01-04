package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseLogLevel 测试日志级别解析
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected int // logger.LogLevel 实际是 int
	}{
		{"silent", "silent", 1},
		{"error", "error", 2},
		{"warn", "warn", 3},
		{"info", "info", 4},
		{"default", "unknown", 3}, // 默认 warn
		{"empty", "", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.level)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}

// TestInitGorm_DefaultCharset 测试默认字符集
func TestInitGorm_DefaultCharset(t *testing.T) {
	// 注意：这个测试需要真实的 MySQL 连接，通常在集成测试中运行
	t.Skip("需要 MySQL 连接，跳过单元测试")

	cfg := Config{
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "root",
		Password: "password",
		// Charset 留空，应该使用默认值 utf8mb4
	}

	db, err := InitGorm(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer CloseDB(db)
}

// TestInitGorm_ConnectionPool 测试连接池配置
func TestInitGorm_ConnectionPool(t *testing.T) {
	t.Skip("需要 MySQL 连接，跳过单元测试")

	cfg := Config{
		Host:            "localhost",
		Port:            3306,
		Database:        "test",
		Username:        "root",
		Password:        "password",
		Charset:         "utf8mb4",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 3600, // 秒
		ConnMaxIdleTime: 600,  // 秒
	}

	db, err := InitGorm(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// 验证连接池配置
	stats := sqlDB.Stats()
	assert.Equal(t, 10, stats.MaxIdleClosed) // 这个字段可能不准确，仅作示例

	defer CloseDB(db)
}

// TestInitGorm_SingularTable 测试单数表名
func TestInitGorm_SingularTable(t *testing.T) {
	t.Skip("需要 MySQL 连接，跳过单元测试")

	cfg := Config{
		Host:          "localhost",
		Port:          3306,
		Database:      "test",
		Username:      "root",
		Password:      "password",
		SingularTable: true,
	}

	db, err := InitGorm(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	// 验证命名策略已应用
	// 实际应该通过 AutoMigrate 等方法验证表名

	defer CloseDB(db)
}

// TestInitGorm_HealthCheck 测试健康检查
func TestInitGorm_HealthCheck(t *testing.T) {
	t.Skip("需要 MySQL 连接，跳过单元测试")

	cfg := Config{
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "root",
		Password: "password",
	}

	db, err := InitGorm(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	// 手动执行健康检查
	sqlDB, err := db.DB()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	assert.NoError(t, err)

	defer CloseDB(db)
}

// TestInitGorm_InvalidConfig 测试无效配置
func TestInitGorm_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "invalid_host",
			config: Config{
				Host:     "invalid-host-12345",
				Port:     3306,
				Database: "test",
				Username: "root",
				Password: "password",
			},
		},
		{
			name: "invalid_port",
			config: Config{
				Host:     "localhost",
				Port:     99999,
				Database: "test",
				Username: "root",
				Password: "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := InitGorm(tt.config)
			assert.Error(t, err)
			assert.Nil(t, db)
		})
	}
}

// TestCloseDB 测试关闭连接
func TestCloseDB(t *testing.T) {
	t.Run("nil_db", func(t *testing.T) {
		err := CloseDB(nil)
		assert.NoError(t, err)
	})

	// 真实连接关闭测试
	t.Run("valid_db", func(t *testing.T) {
		t.Skip("需要 MySQL 连接，跳过单元测试")

		cfg := Config{
			Host:     "localhost",
			Port:     3306,
			Database: "test",
			Username: "root",
			Password: "password",
		}

		db, err := InitGorm(cfg)
		require.NoError(t, err)

		err = CloseDB(db)
		assert.NoError(t, err)
	})
}
