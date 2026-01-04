package db

// Config 数据库配置
type Config struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	Charset         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
	ConnMaxIdleTime int
	LogLevel        string
	SlowThreshold   int
	SkipDefaultTxn  bool
	PrepareStmt     bool
	SingularTable   bool
	DisableForeignKey bool
}


