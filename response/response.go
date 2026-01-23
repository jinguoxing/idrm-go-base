package response

import (
	"encoding/json"
	"net/http"

	"errors"

	"github.com/jinguoxing/idrm-go-base/errorx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func InitErrorHandler() {
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		return http.StatusOK, buildHttpError(err)
	})
}

// HttpResponse 统一HTTP响应结构
type HttpResponse struct {
	Code int         `json:"code" example:"0"`                    // 业务状态码，0表示成功
	Msg  string      `json:"msg" example:"success"`               // 响应消息
	Data interface{} `json:"data,omitempty" swaggertype:"object"` // 响应数据
}

// HttpError 增强版错误响应结构
type HttpError struct {
	Code        int         `json:"code" example:"idrm.common.internal_error"`    // 错误码，格式: 服务名.模块.错误
	Description string      `json:"description" example:"内部错误"`                   // 错误描述
	Solution    string      `json:"solution,omitempty" example:"请联系管理员"`          // 解决方案
	Cause       string      `json:"cause,omitempty" example:"数据库连接失败"`            // 错误原因
	Detail      interface{} `json:"detail,omitempty" swaggertype:"object,string"` // 错误详情
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	resp := &HttpResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
	WriteJSON(w, http.StatusOK, resp)
}

// SuccessWithMsg 带自定义消息的成功响应
func SuccessWithMsg(w http.ResponseWriter, msg string, data interface{}) {
	resp := &HttpResponse{
		Code: 0,
		Msg:  msg,
		Data: data,
	}
	WriteJSON(w, http.StatusOK, resp)
}

// Error 错误响应（简单格式）
func Error(w http.ResponseWriter, err error) {
	var code int
	var msg string

	if e, ok := err.(*errorx.CodeError); ok {
		code = e.GetCode()
		msg = e.GetMsg()
	} else {
		code = errorx.ErrCodeSystem
		msg = err.Error()
	}

	resp := &HttpResponse{
		Code: code,
		Msg:  msg,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ErrorWithMsg 自定义错误消息响应
func ErrorWithMsg(w http.ResponseWriter, code int, msg string) {
	resp := &HttpResponse{
		Code: code,
		Msg:  msg,
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ErrorWithData 带数据的错误响应
func ErrorWithData(w http.ResponseWriter, code int, msg string, data interface{}) {
	resp := &HttpResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
	WriteJSON(w, http.StatusOK, resp)
}

// ErrorDetailed 详细错误响应（增强版）
func ErrorDetailed(w http.ResponseWriter, code string, description string, solution string, cause string, detail interface{}) {
	resp := &HttpError{
		Code:        code,
		Description: description,
		Solution:    solution,
		Cause:       cause,
		Detail:      detail,
	}

	statusCode := http.StatusBadRequest
	// 根据错误码前缀判断HTTP状态码
	if len(code) > 0 {
		switch code[len(code)-3:] {
		case "404":
			statusCode = http.StatusNotFound
		case "403":
			statusCode = http.StatusForbidden
		case "401":
			statusCode = http.StatusUnauthorized
		case "500":
			statusCode = http.StatusInternalServerError
		}
	}

	WriteJSON(w, statusCode, resp)
}

// ErrorValidation 验证错误响应
func ErrorValidation(w http.ResponseWriter, validationErrors map[string]string) {
	resp := &HttpError{
		Code:        errorx.ErrCodeParamInvalid,
		Description: "参数验证失败",
		Solution:    "请检查请求参数是否符合要求",
		Cause:       "请求参数不符合验证规则",
		Detail:      validationErrors,
	}
	WriteJSON(w, http.StatusBadRequest, resp)
}

// NotFound 404错误响应
func NotFound(w http.ResponseWriter, resource string) {
	resp := &HttpError{
		Code:        errorx.ErrCodeNotFound,
		Description: resource + "不存在",
		Solution:    "请确认资源ID是否正确",
		Cause:       "未找到指定的资源",
	}
	WriteJSON(w, http.StatusNotFound, resp)
}

// Unauthorized 401未授权响应
func Unauthorized(w http.ResponseWriter, msg string) {
	resp := &HttpError{
		Code:       errorx.ErrCodeAuth
		Description: msg,
		Solution:    "请先登录或检查认证信息",
		Cause:       "未提供有效的认证信息",
	}
	WriteJSON(w, http.StatusUnauthorized, resp)
}

// Forbidden 403禁止访问响应
func Forbidden(w http.ResponseWriter, msg string) {
	resp := &HttpError{
		Code:        errorx.ErrCodeForbidden,
		Description: msg,
		Solution:    "请联系管理员获取权限",
		Cause:       "当前用户没有执行此操作的权限",
	}
	WriteJSON(w, http.StatusForbidden, resp)
}

// InternalError 500内部错误响应
func InternalError(w http.ResponseWriter, err error) {
	resp := &HttpError{
		Code:        errorx.ErrCodeSystem,
		Description: "内部服务错误",
		Solution:    "请稍后重试或联系管理员",
		Cause:       err.Error(),
	}
	WriteJSON(w, http.StatusInternalServerError, resp)
}

// WriteJSON 写入JSON响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// 如果编码失败，记录错误但不再尝试写入响应
		// 因为header已经发送了
		// 这里可以添加日志记录
	}
}

// SuccessPage 分页成功响应
func SuccessPage(w http.ResponseWriter, list interface{}, total int64, page int, pageSize int) {
	data := map[string]interface{}{
		"entries":   list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	Success(w, data)
}

// ============================================
// go-frame 兼容函数
// ============================================

// ResOKJson 成功响应（直接返回data，不包装）
// 兼容 go-frame: ginx.ResOKJson
func ResOKJson(w http.ResponseWriter, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}
	WriteJSON(w, http.StatusOK, data)
}

// ResList 列表响应（直接返回，不包装）
// 兼容 go-frame: ginx.ResList
func ResList(w http.ResponseWriter, list interface{}, totalCount int64) {
	if list == nil {
		list = []interface{}{}
	}
	data := map[string]interface{}{
		"entries":     list,
		"total_count": totalCount,
	}
	WriteJSON(w, http.StatusOK, data)
}

// ResBadRequestJson 400错误响应
// 兼容 go-frame: ginx.ResBadRequestJson
func ResBadRequestJson(w http.ResponseWriter, err error) {
	ResErrJsonWithCode(w, http.StatusBadRequest, err)
}

// ResErrJsonWithCode 指定HTTP状态码的错误响应
// 兼容 go-frame: ginx.ResErrJsonWithCode
func ResErrJsonWithCode(w http.ResponseWriter, statusCode int, err error) {
	resp := buildHttpError(err)
	WriteJSON(w, statusCode, resp)
}

// ResErrJson 错误响应
// 兼容 go-frame: ginx.ResErrJson
func ResErrJson(w http.ResponseWriter, err error) {
	resp := buildHttpError(err)
	WriteJSON(w, http.StatusBadRequest, resp)
}

// buildHttpError 构建 HttpError 结构
func buildHttpError(err error) *HttpError {
	if err == nil {
		return &HttpError{
			Code:        0,
			Description: "成功",
		}
	}

	// 使用 errors.As 支持 wrapped error
	var e *errorx.CodeError
	if errors.As(err, &e) {
		return &HttpError{
			Code:        e.GetCode(),
			Description: e.GetMsg(),
		}
	}

	// 默认处理为系统错误，将原始错误信息放入 Cause 防止泄露敏感信息给 Description
	return &HttpError{
		Code:        errorx.ErrCodeSystem,
		Description: "系统内部错误",
		Cause:       err.Error(),
		Solution:    "请查看服务日志获取详细信息",
	}
}
