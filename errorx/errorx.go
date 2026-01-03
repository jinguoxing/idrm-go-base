package errorx

// 错误码定义
const (
	// 系统错误 (10000-19999)
	ErrCodeSystem   = 10000
	ErrCodeDatabase = 10001
	ErrCodeRedis    = 10002
	ErrCodeKafka    = 10003
	ErrCodeExternal = 10004

	// 参数错误 (20000-29999)
	ErrCodeParam        = 20000
	ErrCodeParamMissing = 20001
	ErrCodeParamInvalid = 20002
	ErrCodeParamFormat  = 20003

	// 业务错误 (30000-39999)
	ErrCodeBusiness        = 30000
	ErrCodeNotFound        = 30001
	ErrCodeAlreadyExists   = 30002
	ErrCodePermissionDeny  = 30003
	ErrCodeOperationFailed = 30004

	// 认证授权错误 (40000-49999)
	ErrCodeAuth         = 40000
	ErrCodeTokenInvalid = 40001
	ErrCodeTokenExpired = 40002
	ErrCodeUnauthorized = 40003
	ErrCodeForbidden    = 40004
)

// 错误消息映射
var errMsgMap = map[int]string{
	ErrCodeSystem:   "系统错误",
	ErrCodeDatabase: "数据库错误",
	ErrCodeRedis:    "缓存错误",
	ErrCodeKafka:    "消息队列错误",
	ErrCodeExternal: "外部服务调用失败",

	ErrCodeParam:        "参数错误",
	ErrCodeParamMissing: "缺少必要参数",
	ErrCodeParamInvalid: "参数不合法",
	ErrCodeParamFormat:  "参数格式错误",

	ErrCodeBusiness:        "业务处理失败",
	ErrCodeNotFound:        "数据不存在",
	ErrCodeAlreadyExists:   "数据已存在",
	ErrCodePermissionDeny:  "权限不足",
	ErrCodeOperationFailed: "操作失败",

	ErrCodeAuth:         "认证失败",
	ErrCodeTokenInvalid: "Token无效",
	ErrCodeTokenExpired: "Token已过期",
	ErrCodeUnauthorized: "未登录",
	ErrCodeForbidden:    "禁止访问",
}

// CodeError 业务错误
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error 实现error接口
func (e *CodeError) Error() string {
	return e.Msg
}

// GetCode 获取错误码
func (e *CodeError) GetCode() int {
	return e.Code
}

// GetMsg 获取错误消息
func (e *CodeError) GetMsg() string {
	return e.Msg
}

// New 创建错误
func New(code int, msg string) error {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// NewWithCode 使用错误码创建错误
func NewWithCode(code int) error {
	msg, ok := errMsgMap[code]
	if !ok {
		msg = "未知错误"
	}
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// NewWithMsg 使用自定义消息创建错误
func NewWithMsg(code int, msg string) error {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}
