# Response 响应助手包

增强版的HTTP响应助手，提供统一的响应格式和详细的错误信息。

## ✨ 特性

- ✅ 统一的响应格式
- ✅ 详细的错误信息结构
- ✅ 多种便捷响应方法
- ✅ 验证错误专用响应
- ✅ 分页响应支持
- ✅ HTTP状态码自动映射

## 📦 响应结构

### HttpResponse - 标准响应

```go
type HttpResponse struct {
    Code int         `json:"code"`               // 业务状态码，0表示成功
    Msg  string      `json:"msg"`                // 响应消息
    Data interface{} `json:"data,omitempty"`     // 响应数据
}
```

**示例**：
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "name": "测试"
  }
}
```

### HttpError - 增强版错误响应

```go
type HttpError struct {
    Code        string      `json:"code"`        // 错误码
    Description string      `json:"description"` // 错误描述
    Solution    string      `json:"solution"`    // 解决方案
    Cause       string      `json:"cause"`       // 错误原因
    Detail      interface{} `json:"detail"`      // 错误详情
}
```

**示例**：
```json
{
  "code": "idrm.common.validation_error",
  "description": "参数验证失败",
  "solution": "请检查请求参数是否符合要求",
  "cause": "请求参数不符合验证规则",
  "detail": {
    "name": "name长度必须至少为2个字符",
    "email": "email必须是一个有效的邮箱"
  }
}
```

## 🚀 使用方法

### 成功响应

#### Success - 基本成功响应

```go
response.Success(w, data)
```

#### SuccessWithMsg - 带自定义消息

```go
response.SuccessWithMsg(w, "创建成功", data)
```

#### SuccessPage - 分页响应

```go
response.SuccessPage(w, list, total, page, pageSize)
```

### 错误响应

#### Error - 基本错误响应

```go
if err != nil {
    response.Error(w, err)
    return
}
```

#### ErrorWithMsg - 自定义错误消息

```go
response.ErrorWithMsg(w, 400, "参数错误")
```

#### ErrorWithData - 带数据的错误响应

```go
response.ErrorWithData(w, 400, "验证失败", validationErrors)
```

### 详细错误响应

#### ErrorDetailed - 完整错误信息

```go
response.ErrorDetailed(w,
    "idrm.category.create_failed",
    "创建类别失败",
    "请检查类别名称是否重复",
    "数据库约束冲突",
    errorDetails,
)
```

#### ErrorValidation - 验证错误

```go
if err := validator.Validate(req); err != nil {
    errMsgs := validator.GetErrorMsg(err)
    response.ErrorValidation(w, errMsgs)
    return
}
```

**响应示例**：
```json
{
  "code": "idrm.common.validation_error",
  "description": "参数验证失败",
  "solution": "请检查请求参数是否符合要求",
  "cause": "请求参数不符合验证规则",
  "detail": {
    "name": "name长度必须至少为2个字符"
  }
}
```

### 常用错误响应

#### NotFound - 404错误

```go
response.NotFound(w, "类别")
```

**响应**：
```json
{
  "code": "idrm.common.not_found",
  "description": "类别不存在",
  "solution": "请确认资源ID是否正确",
  "cause": "未找到指定的资源"
}
```

#### Unauthorized - 401未授权

```go
response.Unauthorized(w, "请先登录")
```

#### Forbidden - 403禁止访问

```go
response.Forbidden(w, "没有权限访问此资源")
```

#### InternalError - 500内部错误

```go
response.InternalError(w, err)
```

## 📝 完整示例

### 在 Handler 中使用

```go
func CreateCategoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.CreateCategoryReq
        
        // 解析请求
        if err := httpx.Parse(r, &req); err != nil {
            response.ErrorWithMsg(w, 400, "请求参数解析失败")
            return
        }
        
        // 验证参数
        if err := validator.Validate(req); err != nil {
            errMsgs := validator.GetErrorMsg(err)
            response.ErrorValidation(w, errMsgs)
            return
        }
        
        // 调用Logic
        l := logic.NewCreateCategoryLogic(r.Context(), svcCtx)
        resp, err := l.CreateCategory(&req)
        if err != nil {
            response.Error(w, err)
            return
        }
        
        // 返回成功
        response.SuccessWithMsg(w, "创建成功", resp)
    }
}
```

### 在 Logic 中返回错误

```go
func (l *CreateCategoryLogic) CreateCategory(req *types.CreateCategoryReq) (*types.CategoryResp, error) {
    // 检查是否存在
    exists, err := l.svcCtx.CategoryModel.FindByCode(l.ctx, req.Code)
    if exists != nil {
        // 返回业务错误
        return nil, errorx.NewWithMsg(400, "类别代码已存在")
    }
    
    // 插入数据
    category, err := l.svcCtx.CategoryModel.Insert(l.ctx, data)
    if err != nil {
        // 返回系统错误
        return nil, err
    }
    
    return &types.CategoryResp{...}, nil
}
```

## 🎨 错误码规范

### 格式

```
服务名.模块名.错误类型
```

### 示例

| 错误码 | 说明 |
|--------|------|
| `idrm.common.validation_error` | 通用验证错误 |
| `idrm.common.not_found` | 资源不存在 |
| `idrm.common.unauthorized` | 未授权 |
| `idrm.common.forbidden` | 禁止访问 |
| `idrm.common.internal_error` | 内部错误 |
| `idrm.category.create_failed` | 类别创建失败 |
| `idrm.category.duplicate_code` | 类别代码重复 |

### HTTP状态码映射

错误码后三位自动映射到HTTP状态码：

| 后缀 | HTTP状态码 | 说明 |
|------|-----------|------|
| `401` | 401 | Unauthorized |
| `403` | 403 | Forbidden |
| `404` | 404 | Not Found |
| `500` | 500 | Internal Server Error |
| 其他 | 400 | Bad Request |

## 🔧 与 Validator 集成

```go
import (
    "idrm/pkg/response"
    "idrm/pkg/validator"
)

func HandleRequest(w http.ResponseWriter, req *types.Request) {
    // 验证
    if err := validator.Validate(req); err != nil {
        // 自动格式化验证错误
        errMsgs := validator.GetErrorMsg(err)
        response.ErrorValidation(w, errMsgs)
        return
    }
    
    // 业务逻辑...
}
```

## 🔧 与 Errorx 集成

```go
import (
    "idrm/pkg/response"
    "idrm/pkg/errorx"
)

// Logic中
if err != nil {
    return nil, errorx.NewWithMsg(404, "类别不存在")
}

// Handler中
resp, err := l.CreateCategory(&req)
if err != nil {
    // 自动处理errorx错误
    response.Error(w, err)
    return
}
```

## 📊 响应格式对比

### 简单格式（HttpResponse）

适用于：
- ✅ 快速开发
- ✅ 简单业务场景
- ✅ 移动端应用

```json
{
  "code": 400,
  "msg": "参数错误"
}
```

### 详细格式（HttpError）

适用于：
- ✅ 复杂业务场景
- ✅ 需要详细错误信息
- ✅ 客户端需要错误处理指导
- ✅ API文档需要详细说明

```json
{
  "code": "idrm.category.duplicate_code",
  "description": "类别代码已存在",
  "solution": "请使用不同的类别代码",
  "cause": "数据库唯一索引冲突",
  "detail": {
    "duplicate_code": "TEST001"
  }
}
```

## 💡 最佳实践

1. **统一使用响应助手**：不要直接写 JSON
2. **验证错误使用 ErrorValidation**：自动格式化
3. **业务错误使用 ErrorWithMsg**：清晰的错误消息
4. **系统错误使用 InternalError**：隐藏内部实现
5. **分页列表使用 SuccessPage**：标准格式

## 🧪 测试

```go
// 测试成功响应
func TestSuccess(t *testing.T) {
    w := httptest.NewRecorder()
    data := map[string]string{"key": "value"}
    
    response.Success(w, data)
    
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "success")
}
```
