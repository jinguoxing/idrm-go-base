# Validator 验证器包

统一的数据验证工具，基于 `github.com/go-playground/validator/v10`。

## ✨ 特性

- ✅ **中文错误消息**：自动翻译为中文
- ✅ **单例模式**：性能优化，避免重复初始化
- ✅ **自定义验证规则**：支持扩展验证器
- ✅ **友好的错误格式**：多种错误消息格式
- ✅ **使用 JSON tag**：错误消息使用 json tag 作为字段名

## 📦 安装

已包含在项目依赖中。

## 🚀 快速开始

### 1. 在 API 定义中添加 validate tag

```api
type CreateCategoryReq {
    Name string `json:"name" validate:"required,min=2,max=50"`
    Code string `json:"code" validate:"required,alphanum"`
    Age  int    `json:"age" validate:"gte=0,lte=150"`
}
```

### 2. 在 Logic 中使用

```go
package category

import (
    "idrm/pkg/validator"
    "idrm/pkg/errorx"
)

func (l *CreateCategoryLogic) CreateCategory(req *types.CreateCategoryReq) (*types.CategoryResp, error) {
    // 验证请求参数
    if err := validator.Validate(req); err != nil {
        // 方式1: 获取详细错误字典
        errMsgs := validator.GetErrorMsg(err)
        return nil, errorx.NewWithMsg(400, fmt.Sprintf("参数错误: %v", errMsgs))
        
        // 方式2: 获取第一个错误
        // return nil, errorx.NewWithMsg(400, validator.GetFirstError(err))
        
        // 方式3: 格式化所有错误
        // return nil, errorx.NewWithMsg(400, validator.FormatError(err))
    }
    
    // 业务逻辑...
}
```

## 📚 API 说明

### 验证函数

#### Validate(data interface{}) error
验证结构体

```go
err := validator.Validate(req)
```

#### ValidateVar(field interface{}, tag string) error
验证单个变量

```go
err := validator.ValidateVar(email, "required,email")
```

### 错误处理

#### GetErrorMsg(err error) map[string]string
获取错误字典（字段名 → 错误消息）

```go
errMsgs := validator.GetErrorMsg(err)
// 输出: map[string]string{"name": "name长度必须至少为2个字符", "email": "email必须是一个有效的邮箱"}
```

#### GetFirstError(err error) string
获取第一个错误消息

```go
msg := validator.GetFirstError(err)
// 输出: "name长度必须至少为2个字符"
```

#### GetErrorList(err error) []string
获取错误列表

```go
errList := validator.GetErrorList(err)
// 输出: []string{"name长度必须至少为2个字符", "email必须是一个有效的邮箱"}
```

#### FormatError(err error) string
格式化所有错误为字符串

```go
formatted := validator.FormatError(err)
// 输出: "name: name长度必须至少为2个字符; email: email必须是一个有效的邮箱"
```

## 🏷️ 内置验证标签

### 字符串验证
| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填 | `validate:"required"` |
| `min=N` | 最小长度 | `validate:"min=2"` |
| `max=N` | 最大长度 | `validate:"max=50"` |
| `len=N` | 固定长度 | `validate:"len=11"` |
| `email` | 邮箱 | `validate:"email"` |
| `url` | URL | `validate:"url"` |
| `alpha` | 只能字母 | `validate:"alpha"` |
| `alphanum` | 字母数字 | `validate:"alphanum"` |
| `numeric` | 数字 | `validate:"numeric"` |

### 数字验证
| 标签 | 说明 | 示例 |
|------|------|------|
| `gt=N` | 大于 | `validate:"gt=0"` |
| `gte=N` | 大于等于 | `validate:"gte=0"` |
| `lt=N` | 小于 | `validate:"lt=100"` |
| `lte=N` | 小于等于 | `validate:"lte=100"` |
| `eq=N` | 等于 | `validate:"eq=10"` |
| `ne=N` | 不等于 | `validate:"ne=0"` |
| `oneof=A B C` | 枚举值 | `validate:"oneof=1 2 3"` |

### 其他
| 标签 | 说明 | 示例 |
|------|------|------|
| `omitempty` | 可选 | `validate:"omitempty,min=1"` |
| `dive` | 验证数组元素 | `validate:"dive,required"` |
| `eqfield=Field` | 等于另一字段 | `validate:"eqfield=Password"` |
| `nefield=Field` | 不等于另一字段 | `validate:"nefield=OldPassword"` |

## 🎨 自定义验证器

### 已包含的自定义验证器

#### mobile
验证手机号（11位，1开头）

```go
Mobile string `json:"mobile" validate:"required,mobile"`
```

#### idcard
验证身份证号（15或18位）

```go
IDCard string `json:"id_card" validate:"required,idcard"`
```

#### chinese
验证是否为中文

```go
Name string `json:"name" validate:"required,chinese"`
```

### 添加自定义验证器

在 `pkg/validator/validator.go` 的 `registerCustomValidators()` 函数中添加：

```go
func registerCustomValidators() {
    // 示例：验证QQ号
    validate.RegisterValidation("qq", func(fl validator.FieldLevel) bool {
        qq := fl.Field().String()
        // 实现验证逻辑
        return len(qq) >= 5 && len(qq) <= 11
    })
}
```

在 `registerCustomTranslations()` 中添加翻译：

```go
validate.RegisterTranslation("qq", trans, func(ut ut.Translator) error {
    return ut.Add("qq", "{0}必须是有效的QQ号码", true)
}, func(ut ut.Translator, fe validator.FieldError) string {
    t, _ := ut.T("qq", fe.Field())
    return t
})
```

## 📝 完整示例

### API 定义

```api
type CreateUserReq {
    Username  string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Password  string `json:"password" validate:"required,min=6"`
    Email     string `json:"email" validate:"required,email"`
    Age       int    `json:"age" validate:"required,gte=18,lte=100"`
    Mobile    string `json:"mobile" validate:"required,mobile"`
    RealName  string `json:"real_name" validate:"omitempty,chinese"`
}
```

### Logic 实现

```go
func (l *CreateUserLogic) CreateUser(req *types.CreateUserReq) (*types.UserResp, error) {
    // 验证
    if err := validator.Validate(req); err != nil {
        // 使用详细错误
        errMsgs := validator.GetErrorMsg(err)
        logx.Errorf("参数验证失败: %v", errMsgs)
        
        // 返回友好的错误消息
        return nil, errorx.NewWithData(400, "参数验证失败", errMsgs)
    }
    
    // 业务逻辑...
    return &types.UserResp{...}, nil
}
```

### 错误响应示例

验证失败时的错误消息（中文）：

```json
{
  "code": 400,
  "msg": "参数验证失败",
  "data": {
    "username": "username长度必须至少为3个字符",
    "email": "email必须是一个有效的邮箱",
    "age": "age必须大于或等于18"
  }
}
```

## ⚙️ 配置

### 初始化

验证器会在首次使用时自动初始化（单例模式），也可以手动初始化：

```go
import "idrm/pkg/validator"

func main() {
    // 可选：手动初始化
    validator.Init()
}
```

### 性能

- ✅ 单例模式，避免重复初始化
- ✅ 使用反射缓存，性能优秀
- ✅ 支持并发安全

## 🧪 测试

运行测试：

```bash
go test -v ./pkg/validator
```

## 📖 更多资源

- [validator 官方文档](https://github.com/go-playground/validator)
- [所有内置验证标签](https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags)
