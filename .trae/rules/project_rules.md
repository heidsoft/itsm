# 📘 Trae 项目开发规则：ITSM 系统（Go + Gin + Ent）

---

## 1. 框架版本与主要依赖

* 编程语言：Go 1.22+
* Web 框架：Gin v1.9.1
* ORM 工具：Ent v0.13.0
* 数据库：PostgreSQL v15+
* 配置管理：spf13/viper
* JSON 处理：Go 原生 encoding/json
* 日志组件：uber-go/zap v1.26.0
* 接口文档：swaggo/gin-swagger
* 鉴权机制：JWT + 基于角色的访问控制（RBAC）

---

## 2. 禁止使用的 APIs 与代码风格限制

为了保持结构一致、便于协作与自动生成代码，**禁止使用以下做法：**

* ❌ 不使用 `database/sql` 或 `gorm`，统一采用 `Ent` ORM
* ❌ 不允许在 controller 中直接访问数据库
* ❌ 不使用 `fmt.Println()` 或 `log.Println()`，应统一使用 `zap.Sugar()` 结构化日志
* ❌ 不允许未封装的错误直接 panic，应通过 `errors.Wrap()` 或统一错误返回结构处理
* ❌ 禁止返回非统一格式的响应，应使用：

```go
// 成功响应
Success(c, data)

// 错误响应
Fail(c, code, "错误信息")
```

* ❌ 不允许写死用户 ID 或硬编码权限逻辑，应通过中间件注入用户身份并进行验证

---

## 3. 测试框架与测试要求

* 单元测试框架：`stretchr/testify`
* 所有 `service` 层必须配有对应的 `_test.go` 文件
* 所有接口建议使用 `httptest.NewRecorder` 进行 handler 级测试
* 测试方式采用 Table Driven Test 风格
* 所有错误路径需要覆盖
* 推荐使用 `enttest.NewClient()` 创建测试用数据库环境（内存或临时文件）

---

## ✅ 统一接口响应规范

所有 API 接口返回结构如下：

```json
{
  "code": 0,
  "message": "操作成功",
  "data": {}
}
```

* 成功时：`code = 0`
* 参数错误：`code = 1001+`
* 鉴权失败：`code = 2001`
* 服务内部错误：`code = 5001`

---

## 📁 项目结构建议

| 目录            | 说明                   |
| ------------- | -------------------- |
| `controller/` | 控制器，仅接收参数并调用 service |
| `service/`    | 核心业务逻辑处理             |
| `ent/schema/` | 数据建模（自动生成 CRUD）      |
| `middleware/` | JWT、权限、日志、异常处理       |
| `router/`     | 路由注册                 |
| `config/`     | 配置文件、初始化数据库等         |
| `tests/`      | 单元测试与 mock 测试        |

---

## 🚀 推荐 Trae 命令示例

你可以在 Trae 中使用如下 prompt 快速生成模块：

```text
请为工单模块生成 controller、service、ent schema，并注册 /api/tickets 路由，所有返回结构使用统一格式，支持状态流转与审批嵌套逻辑。
```

```text
创建一个中间件 auth.go，支持解析 Bearer Token 并将 userID 注入 gin.Context，如无效则返回 401
```

---

> 本文件将被 Trae 用于辅助理解项目结构、生成模块代码、自动补全 API 设计与测试逻辑。
