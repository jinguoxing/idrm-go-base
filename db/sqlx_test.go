package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TestFormatDSN 测试 DSN 格式化
func TestFormatDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "complete_config",
			config: Config{
				Host:     "localhost",
				Port:     3306,
				Database: "test",
				Username: "root",
				Password: "password",
				Charset:  "utf8mb4",
			},
			expected: "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=true&loc=Local",
		},
		{
			name: "default_charset",
			config: Config{
				Host:     "192.168.1.100",
				Port:     3307,
				Database: "mydb",
				Username: "admin",
				Password: "admin123",
				// Charset 为空，应使用默认值 utf8mb4
			},
			expected: "admin:admin123@tcp(192.168.1.100:3307)/mydb?charset=utf8mb4&parseTime=true&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDSN(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInitSqlx 测试 sqlx 初始化
func TestInitSqlx(t *testing.T) {
	t.Run("returns_sqlconn", func(t *testing.T) {
		cfg := Config{
			Host:     "localhost",
			Port:     3306,
			Database: "test",
			Username: "root",
			Password: "password",
		}

		conn := InitSqlx(cfg)
		assert.NotNil(t, conn)

		// 验证返回类型
		_, ok := interface{}(conn).(sqlx.SqlConn)
		assert.True(t, ok, "should return sqlx.SqlConn interface")
	})
}

// TestInitSqlx_Integration 集成测试 (需要真实 MySQL)
func TestInitSqlx_Integration(t *testing.T) {
	t.Skip("需要 MySQL 连接，跳过集成测试")

	cfg := Config{
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "root",
		Password: "password",
	}

	conn := InitSqlx(cfg)

	// 测试简单查询
	var result int
	err := conn.QueryRow(&result, "SELECT 1")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}
