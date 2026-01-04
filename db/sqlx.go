package db

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// InitSqlx 初始化 go-zero sqlx 连接
func InitSqlx(cfg Config) sqlx.SqlConn {
	dsn := FormatDSN(cfg)
	return sqlx.NewMysql(dsn)
}

// FormatDSN 格式化 MySQL DSN 字符串
func FormatDSN(cfg Config) string {
	// 设置默认字符集
	if cfg.Charset == "" {
		cfg.Charset = "utf8mb4"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)
}
