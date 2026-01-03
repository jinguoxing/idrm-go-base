package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	once     sync.Once
	validate *validator.Validate
	trans    ut.Translator
)

// Init 初始化验证器（单例模式）
func Init() {
	once.Do(func() {
		validate = validator.New()

		// 注册自定义字段名获取函数（使用 json tag）
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		// 初始化中文翻译器
		zhLocale := zh.New()
		uni := ut.New(zhLocale, zhLocale)
		trans, _ = uni.GetTranslator("zh")

		// 注册中文翻译
		_ = zh_translations.RegisterDefaultTranslations(validate, trans)

		// 注册自定义验证器
		registerCustomValidators()

		// 自定义翻译（可选）
		registerCustomTranslations()
	})
}

// Validate 验证结构体
func Validate(data interface{}) error {
	if validate == nil {
		Init()
	}
	return validate.Struct(data)
}

// ValidateVar 验证单个变量
func ValidateVar(field interface{}, tag string) error {
	if validate == nil {
		Init()
	}
	return validate.Var(field, tag)
}

// GetErrorMsg 获取友好的错误消息
// 返回格式: map[字段名]错误消息
func GetErrorMsg(err error) map[string]string {
	if err == nil {
		return nil
	}

	errs := make(map[string]string)

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			// 使用翻译后的错误消息
			errs[e.Field()] = e.Translate(trans)
		}
	} else {
		// 非验证错误
		errs["error"] = err.Error()
	}

	return errs
}

// GetFirstError 获取第一个错误消息
func GetFirstError(err error) string {
	if err == nil {
		return ""
	}

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		if len(validationErrs) > 0 {
			return validationErrs[0].Translate(trans)
		}
	}

	return err.Error()
}

// GetErrorList 获取错误列表
func GetErrorList(err error) []string {
	if err == nil {
		return nil
	}

	var errList []string

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			errList = append(errList, e.Translate(trans))
		}
	} else {
		errList = append(errList, err.Error())
	}

	return errList
}

// 注册自定义验证器
func registerCustomValidators() {
	// 示例：验证手机号
	validate.RegisterValidation("mobile", func(fl validator.FieldLevel) bool {
		mobile := fl.Field().String()
		if len(mobile) != 11 {
			return false
		}
		// 简单验证：1开头的11位数字
		return mobile[0] == '1'
	})

	// 示例：验证身份证号（简化版）
	validate.RegisterValidation("idcard", func(fl validator.FieldLevel) bool {
		idcard := fl.Field().String()
		// 15位或18位
		return len(idcard) == 15 || len(idcard) == 18
	})

	// 示例：验证中文
	validate.RegisterValidation("chinese", func(fl validator.FieldLevel) bool {
		str := fl.Field().String()
		for _, r := range str {
			if r < 0x4e00 || r > 0x9fa5 {
				return false
			}
		}
		return true
	})
}

// 注册自定义翻译
func registerCustomTranslations() {
	// 自定义 mobile 的翻译
	validate.RegisterTranslation("mobile", trans, func(ut ut.Translator) error {
		return ut.Add("mobile", "{0}必须是有效的手机号码", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile", fe.Field())
		return t
	})

	// 自定义 idcard 的翻译
	validate.RegisterTranslation("idcard", trans, func(ut ut.Translator) error {
		return ut.Add("idcard", "{0}必须是有效的身份证号码", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("idcard", fe.Field())
		return t
	})

	// 自定义 chinese 的翻译
	validate.RegisterTranslation("chinese", trans, func(ut ut.Translator) error {
		return ut.Add("chinese", "{0}必须是中文", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("chinese", fe.Field())
		return t
	})
}

// FormatError 格式化错误为字符串
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	errMsgs := GetErrorMsg(err)
	if len(errMsgs) == 0 {
		return err.Error()
	}

	var msgs []string
	for field, msg := range errMsgs {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}

	return strings.Join(msgs, "; ")
}
