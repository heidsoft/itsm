# ITSM API 文档

## 📖 概述

ITSM 平台后端 API 采用 RESTful 风格设计，使用 Swagger/OpenAPI 3.0 规范进行文档化。

- **API 版本**: v1
- **基础路径**: `/api/v1`
- **认证方式**: Bearer Token (JWT)
- **数据格式**: JSON

## 🔗 快速链接

- **Swagger UI**: `http://localhost:8090/swagger/index.html`
- **Swagger JSON**: `http://localhost:8090/swagger/doc.json`
- **Swagger YAML**: `http://localhost:8090/swagger/doc.yaml`

## 🔐 认证

所有 API 请求（除登录接口外）需要在 Header 中携带 Bearer Token：

```bash
Authorization: Bearer <your_jwt_token>
```

### 获取 Token

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

## 📚 API 模块

### 1. 认证模块 (`/api/v1/auth`)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /login | 用户登录 |
| POST | /logout | 用户登出 |
| POST | /refresh | 刷新 Token |
| GET | /me | 获取当前用户信息 |

### 2. 工单模块 (`/api/v1/tickets`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /tickets | 获取工单列表 |
| GET | /tickets/:id | 获取工单详情 |
| POST | /tickets | 创建工单 |
| PUT | /tickets/:id | 更新工单 |
| DELETE | /tickets/:id | 删除工单 |
| POST | /tickets/:id/assign | 分配工单 |
| POST | /tickets/:id/transition | 工单状态流转 |
| GET | /tickets/:id/comments | 获取工单评论 |
| POST | /tickets/:id/comments | 添加工单评论 |

### 3. 自动化规则模块 (`/api/v1/automation-rules`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /automation-rules | 获取自动化规则列表 |
| GET | /automation-rules/:id | 获取规则详情 |
| POST | /automation-rules | 创建规则 |
| PUT | /automation-rules/:id | 更新规则 |
| DELETE | /automation-rules/:id | 删除规则 |
| POST | /automation-rules/:id/test | 测试规则 |

#### 自动化规则条件字段

| 字段 | 操作符 | 说明 |
|------|--------|------|
| status | equals, in, not_equals | 工单状态 |
| priority | equals, in, not_equals | 优先级 (low/medium/high/urgent) |
| category_id | equals, in | 分类 ID |
| department_id | equals, in | 部门 ID |
| assignee_id | equals, not_equals | 处理人 ID |
| title | contains | 工单标题 |

#### 自动化规则动作类型

| 类型 | 参数 | 说明 |
|------|------|------|
| set_category | category_id | 设置分类 |
| set_priority | priority | 设置优先级 |
| set_status | status | 设置状态 |
| assign | user_id | 分配给指定用户 |
| auto_assign | - | 自动分配 |
| escalate | - | 升级优先级 |
| send_notification | content | 发送通知 |

### 4. 事件管理模块 (`/api/v1/incidents`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /incidents | 获取事件列表 |
| GET | /incidents/:id | 获取事件详情 |
| POST | /incidents | 创建事件 |
| PUT | /incidents/:id | 更新事件 |
| DELETE | /incidents/:id | 删除事件 |

### 5. 问题管理模块 (`/api/v1/problems`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /problems | 获取问题列表 |
| GET | /problems/:id | 获取问题详情 |
| POST | /problems | 创建问题 |
| PUT | /problems/:id | 更新问题 |
| DELETE | /problems/:id | 删除问题 |

### 6. 变更管理模块 (`/api/v1/changes`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /changes | 获取变更列表 |
| GET | /changes/:id | 获取变更详情 |
| POST | /changes | 创建变更 |
| PUT | /changes/:id | 更新变更 |
| DELETE | /changes/:id | 删除变更 |

### 7. SLA 管理模块 (`/api/v1/sla`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /sla/policies | 获取 SLA 策略列表 |
| GET | /sla/metrics | 获取 SLA 指标 |
| GET | /sla/violations | 获取 SLA 违规记录 |

### 8. 统计分析模块 (`/api/v1/analytics`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /analytics/dashboard | 获取仪表盘数据 |
| GET | /analytics/tickets/trend | 工单趋势分析 |
| GET | /analytics/sla/report | SLA 报表 |
| GET | /analytics/team/performance | 团队绩效分析 |

### 9. 知识库模块 (`/api/v1/knowledge`)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /knowledge/articles | 获取知识文章列表 |
| GET | /knowledge/articles/:id | 获取文章详情 |
| POST | /knowledge/articles | 创建文章 |
| PUT | /knowledge/articles/:id | 更新文章 |
| DELETE | /knowledge/articles/:id | 删除文章 |
| GET | /knowledge/search | 搜索知识文章 |

### 10. AI 问答模块 (`/api/v1/ai`)

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /ai/chat | AI 问答（RAG） |
| GET | /ai/tools | 可用工具清单 |
| POST | /ai/tools/execute | 执行工具 |

## 📝 请求示例

### 创建工单

```bash
curl -X POST http://localhost:8090/api/v1/tickets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "系统登录失败",
    "description": "用户无法登录系统，提示密码错误",
    "priority": "high",
    "category_id": 1,
    "department_id": 2
  }'
```

### 查询工单列表

```bash
curl -X GET "http://localhost:8090/api/v1/tickets?status=open&priority=high&page=1&page_size=20" \
  -H "Authorization: Bearer <token>"
```

### 创建自动化规则

```bash
curl -X POST http://localhost:8090/api/v1/automation-rules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "紧急工单自动分配",
    "description": "紧急级别工单自动分配给值班工程师",
    "priority": 10,
    "conditions": [
      {"field": "priority", "operator": "equals", "value": "urgent"}
    ],
    "actions": [
      {"type": "auto_assign"},
      {"type": "send_notification", "content": "您有新的紧急工单"}
    ],
    "is_active": true
  }'
```

## 🔧 Swagger 注解规范

后端代码使用 swaggo/swag 生成 API 文档。在 Controller 中添加注解：

```go
// CreateTicket 创建工单
// @Summary      创建工单
// @Description  创建新的工单记录
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  dto.CreateTicketRequest  true  "工单信息"
// @Success      200   {object}  dto.TicketResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Router       /tickets [post]
func (c *TicketController) CreateTicket(ctx *gin.Context) {
    // ...
}
```

### 常用注解

| 注解 | 说明 |
|------|------|
| @Summary | API 简短描述 |
| @Description | API 详细描述 |
| @Tags | API 分组标签 |
| @Accept | 请求格式 |
| @Produce | 响应格式 |
| @Security | 认证方式 |
| @Param | 请求参数 |
| @Success | 成功响应 |
| @Failure | 失败响应 |
| @Router | 路由路径和方法 |

## 📦 响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "参数错误",
  "data": null
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [ ... ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

## 🔍 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证/Token 无效 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 🛠️ 开发工具

### 生成 Swagger 文档

```bash
cd itsm-backend
swag init --parseDependency --parseInternal
```

### 查看文档

启动服务后访问：`http://localhost:8090/swagger/index.html`

## 📋 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-02-28 | 初始版本，包含核心模块 API 文档 |

## 📞 支持

如有 API 相关问题，请联系开发团队或查看 GitHub Issues。

---

_最后更新：2026-02-28_
