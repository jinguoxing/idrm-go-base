package validator

import (
	"testing"
)

type TestStruct struct {
	Name   string `json:"name" validate:"required,min=2,max=50"`
	Email  string `json:"email" validate:"required,email"`
	Age    int    `json:"age" validate:"required,gte=0,lte=150"`
	Mobile string `json:"mobile" validate:"required,mobile"`
}

func TestValidate(t *testing.T) {
	Init()

	tests := []struct {
		name    string
		data    TestStruct
		wantErr bool
	}{
		{
			name: "有效数据",
			data: TestStruct{
				Name:   "张三",
				Email:  "test@example.com",
				Age:    25,
				Mobile: "13800138000",
			},
			wantErr: false,
		},
		{
			name: "姓名太短",
			data: TestStruct{
				Name:   "a",
				Email:  "test@example.com",
				Age:    25,
				Mobile: "13800138000",
			},
			wantErr: true,
		},
		{
			name: "邮箱格式错误",
			data: TestStruct{
				Name:   "张三",
				Email:  "invalid-email",
				Age:    25,
				Mobile: "13800138000",
			},
			wantErr: true,
		},
		{
			name: "年龄超出范围",
			data: TestStruct{
				Name:   "张三",
				Email:  "test@example.com",
				Age:    200,
				Mobile: "13800138000",
			},
			wantErr: true,
		},
		{
			name: "手机号格式错误",
			data: TestStruct{
				Name:   "张三",
				Email:  "test@example.com",
				Age:    25,
				Mobile: "12345",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				if err != nil {
					t.Logf("错误详情: %v", GetErrorMsg(err))
				}
			}
		})
	}
}

func TestGetErrorMsg(t *testing.T) {
	Init()

	data := TestStruct{
		Name:   "a",
		Email:  "invalid",
		Age:    200,
		Mobile: "123",
	}

	err := Validate(data)
	if err == nil {
		t.Fatal("期望有验证错误")
	}

	errMsgs := GetErrorMsg(err)
	t.Logf("错误消息: %v", errMsgs)

	// 检查是否有错误消息
	if len(errMsgs) == 0 {
		t.Error("未获取到错误消息")
	}

	// 检查字段名是否使用 json tag
	if _, ok := errMsgs["name"]; !ok {
		t.Error("未找到 name 字段的错误")
	}
}

func TestGetFirstError(t *testing.T) {
	Init()

	data := TestStruct{
		Name:  "a",
		Email: "invalid",
	}

	err := Validate(data)
	if err == nil {
		t.Fatal("期望有验证错误")
	}

	firstErr := GetFirstError(err)
	t.Logf("第一个错误: %s", firstErr)

	if firstErr == "" {
		t.Error("未获取到第一个错误消息")
	}
}

func TestFormatError(t *testing.T) {
	Init()

	data := TestStruct{
		Name:  "a",
		Email: "invalid",
	}

	err := Validate(data)
	if err == nil {
		t.Fatal("期望有验证错误")
	}

	formatted := FormatError(err)
	t.Logf("格式化错误: %s", formatted)

	if formatted == "" {
		t.Error("未格式化错误消息")
	}
}
