# API 参考文档

**最后更新**: 2026-03-04
**版本**: v1.0
**基于**: OpenAPI 3.0

---

## 📋 目录

- [基础信息](#基础信息)
- [认证方式](#认证方式)
- [通用响应格式](#通用响应格式)
- [错误码表](#错误码表)
- [核心 API](#核心-api)
  - [认证 API](#认证-api)
  - [工单 API](#工单-api)
  - [事件 API](#事件-api)
  - [问题 API](#问题-api)
  - [变更 API](#变更-api)
  - [知识库 API](#知识库-api)
  - [工作流 API](#工作流-api)
  - [CMDB API](#cmdb-api)
  - [SLA API](#sla-api)
  - [用户管理 API](#用户管理-api)
- [Webhook](#webhook)
- [API 变更日志](#api-变更日志)

---

## 基础信息

- **Base URL**: `http://localhost:8090` (开发环境)
- **API 前缀**: `/api/v1`
- **数据格式**: JSON
- **字符编码**: UTF-8
- **超时时间**: 30 秒

### 请求示例

```bash
# curl 示例
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}'

# 带 Token
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/tickets
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "示例"
  }
}
```

---

## 认证方式

系统使用 **Bearer Token** (JWT) 认证。

### 获取 Token

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "admin123"
}
```

### 使用 Token

```http
GET /api/v1/tickets
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### Token 刷新

```http
POST /api/v1/auth/refresh
Authorization: Bearer <current_token>
```

---

## 通用响应格式

### 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    // API 具体数据
  }
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      { "id": 1, "name": "Item 1" },
      { "id": 2, "name": "Item 2" }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### 错误响应

```json
{
  "code": 1001,
  "message": "参数验证失败",
  "data": null,
  "errors": [
    {
      "field": "email",
      "message": "邮箱格式不正确"
    }
  ]
}
```

---

## 错误码表

| Code | 模块 | 说明 | HTTP 状态码 |
|------|------|------|-------------|
| 0 | 通用 | 成功 | 200 |
| 1001-1099 | 通用 | 参数错误 | 400 |
| 1101-1199 | 通用 | 资源未找到 | 404 |
| 2001-2099 | 认证 | 认证失败 | 401 |
| 2101-2199 | 权限 | 权限不足 | 403 |
| 3001-3099 | 业务 | 业务逻辑错误 | 400/409 |
| 5001-5099 | 系统 | 服务器内部错误 | 500 |

### 详细错误码

**通用错误**:

| Code | Message | 说明 |
|------|---------|------|
| 0 | success | 成功 |
| 1001 | invalid_params | 参数无效 |
| 1002 | missing_required_field | 缺少必填字段 |
| 1003 | validation_failed | 验证失败 |
| 1101 | resource_not_found | 资源不存在 |
| 1102 | resource_already_exists | 资源已存在 |

**认证错误**:

| Code | Message | 说明 |
|------|---------|------|
| 2001 | invalid_credentials | 认证凭据错误 |
| 2002 | token_expired | Token 已过期 |
| 2003 | invalid_token | Token 无效 |
| 2004 | token_missing | 缺少 Token |
| 2005 | account_disabled | 账户已禁用 |

**权限错误**:

| Code | Message | 说明 |
|------|---------|------|
| 2101 | permission_denied | 权限不足 |
| 2102 | role_not_found | 角色不存在 |

**业务错误**:

| Code | Message | 说明 |
|------|---------|------|
| 3001 | ticket_not_assignable | 工单不可分配 |
| 3002 | sla_violation | SLA 违规 |
| 3003 | workflow_not_found | 工作流不存在 |
| 3004 | invalid_transition | 状态流转无效 |
| 3005 | change_approval_required | 变更需要审批 |

**系统错误**:

| Code | Message | 说明 |
|------|---------|------|
| 5001 | internal_error | 系统内部错误 |
| 5002 | database_error | 数据库错误 |
| 5003 | external_service_error | 外部服务错误 |
| 5004 | timeout | 请求超时 |

---

## 核心 API

### 认证 API

#### 1. 登录

```http
POST /api/v1/auth/login
```

**请求体**:

```json
{
  "email": "admin@example.com",
  "password": "admin123"
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "email": "admin@example.com",
      "name": "Administrator",
      "role": "admin",
      "tenant_id": 1
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400
  }
}
```

---

#### 2. 刷新 Token

```http
POST /api/v1/auth/refresh
Authorization: Bearer <refresh_token>
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "new.jwt.token",
    "expires_in": 86400
  }
}
```

---

#### 3. 登出

```http
POST /api/v1/auth/logout
Authorization: Bearer <token>
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 工单 API

#### 1. 列出工单（支持过滤分页）

```http
GET /api/v1/tickets
```

**查询参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | integer | 页码（默认 1） |
| page_size | integer | 每页数量（默认 20，最大 100） |
| status | string | 状态过滤 |
| priority | string | 优先级过滤 |
| assignee_id | integer | 指派人过滤 |
| created_after | datetime | 创建时间之后 |
| created_before | datetime | 创建时间之前 |
| sort_by | string | 排序字段（默认 created_at） |
| sort_order | string | 排序方向 asc/desc |

**示例**:

```
GET /api/v1/tickets?status=open&priority=high&page=1&page_size=20&sort_by=created_at&sort_order=desc
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "ticket_number": "TKT-20260304-000001",
        "title": "无法登录系统",
        "description": "用户反馈无法登录...",
        "status": "open",
        "priority": "high",
        "category": "技术支持",
        "submitter": {
          "id": 1,
          "name": "张三",
          "email": "zhangsan@example.com"
        },
        "assignee": {
          "id": 2,
          "name": "李四",
          "email": "lisi@example.com"
        },
        "sla": {
          "response_time": 60,
          "resolution_time": 480,
          "status": "met"
        },
        "created_at": "2026-03-04T10:30:00Z",
        "updated_at": "2026-03-04T10:35:00Z",
        "due_date": "2026-03-05T00:00:00Z"
      }
    ],
    "total": 150,
    "page": 1,
    "page_size": 20,
    "total_pages": 8
  }
}
```

---

#### 2. 获取工单详情

```http
GET /api/v1/tickets/{id}
```

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| id | integer | 工单 ID |

---

#### 3. 创建工单

```http
POST /api/v1/tickets
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "title": "无法访问邮件系统",
  "description": "用户反映无法接收邮件...",
  "priority": "high",
  "category": "邮件服务",
  "assignee_id": 2,
  "due_date": "2026-03-05T18:00:00Z",
  "custom_fields": {
    "affected_users": 5,
    "location": "北京办公室"
  }
}
```

**响应**: 返回创建的工单对象（同 GET）

---

#### 4. 更新工单

```http
PUT /api/v1/tickets/{id}
Authorization: Bearer <token>
```

**可更新字段**:

```json
{
  "title": "更新后的标题",
  "description": "更新后的描述...",
  "status": "in_progress",
  "priority": "medium",
  "assignee_id": 3,
  "due_date": "2026-03-06T00:00:00Z",
  "custom_fields": {}
}
```

---

#### 5. 删除工单

```http
DELETE /api/v1/tickets/{id}
Authorization: Bearer <token>
```

**响应**:

```json
{
  "code": 0,
  "message": "Ticket deleted successfully",
  "data": null
}
```

---

#### 6. 添加评论

```http
POST /api/v1/tickets/{id}/comments
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "content": "已联系用户确认问题...",
  "is_internal": true,
  "attachments": [
    {
      "filename": "screenshot.png",
      "url": "https://..."
    }
  ]
}
```

---

#### 7. 工单统计

```http
GET /api/v1/tickets/stats/summary
Authorization: Bearer <token>
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 150,
    "by_status": {
      "open": 30,
      "in_progress": 45,
      "pending": 15,
      "resolved": 50,
      "closed": 10
    },
    "by_priority": {
      "low": 20,
      "medium": 80,
      "high": 40,
      "critical": 10
    },
    "sla_metrics": {
      "response_met": 85,
      "response_missed": 5,
      "resolution_met": 75,
      "resolution_missed": 15
    }
  }
}
```

---

### 事件 API

#### 1. 列出事件

```http
GET /api/v1/incidents
```

**查询参数**: 类似工单，支持按影响级别、状态过滤

**响应**: 事件对象列表

---

#### 2. 创建事件

```http
POST /api/v1/incidents
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "title": "数据库连接异常",
  "description": "检测到数据库连接数异常...",
  "impact": "major",     // low, medium, high, critical
  "urgency": "high",    // low, medium, high, critical
  "source": "monitoring", // manual, monitoring, auto
  "related_ticket_id": 123
}
```

---

#### 3. 升级为问题

```http
POST /api/v1/incidents/{id}/escalate
Authorization: Bearer <token>
```

**响应**:

```json
{
  "code": 0,
  "message": "Incident escalated to problem",
  "data": {
    "problem_id": 456
  }
}
```

---

### 问题 API

#### 1. 列出问题

```http
GET /api/v1/problems
```

---

#### 2. 创建已知错误

```http
POST /api/v1/problems/{id}/known-error
Authorization: Bearer <token>
```

---

### 变更 API

#### 1. 创建变更请求

```http
POST /api/v1/changes
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "title": "升级数据库版本",
  "description": "从 PostgreSQL 14 升级到 15",
  "risk_level": "medium",  // low, medium, high, critical
  "impact": "影响部分服务",
  "planned_start": "2026-03-10T02:00:00Z",
  "planned_end": "2026-03-10T04:00:00Z",
  "implementation_plan": "1. 停止服务 2. 备份 3. 升级...",
  "backout_plan": "回滚到备份",
  "approvers": [1, 2, 3],
  "tasks": [
    {
      "title": "停止应用",
      "order": 1,
      "responsible_user_id": 5
    }
  ]
}
```

---

#### 2. 审批变更

```http
POST /api/v1/changes/{id}/approve
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "decision": "approved",  // approved, rejected
  "comment": "同意该变更"
}
```

---

#### 3. 执行变更

```http
POST /api/v1/changes/{id}/execute
Authorization: Bearer <token>
```

---

### 知识库 API

#### 1. 列出知识文章

```http
GET /api/v1/knowledge/articles
```

**查询参数**:

| 参数 | 说明 |
|------|------|
| category | 分类过滤 |
| status | 状态（published, draft） |
| keyword | 关键词搜索 |

---

#### 2. 创建知识文章

```http
POST /api/v1/knowledge/articles
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "title": "如何重置用户密码",
  "content": "详细步骤...",
  "category": "常见问题",
  "tags": ["密码", "账户"],
  "status": "published",
  "attachments": []
}
```

---

#### 3. AI 智能搜索

```http
POST /api/v1/knowledge/search/semantic
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "query": "怎么重置密码",
  "limit": 5,
  "similarity_threshold": 0.7
}
```

**响应**:

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "article_id": 1,
      "title": "如何重置用户密码",
      "content": "重置密码步骤：1. 登录管理后台...",
      "similarity": 0.89,
      "highlights": ["重置密码", "管理后台"]
    }
  ]
}
```

---

### 工作流 API

#### 1. 列出工作流定义

```http
GET /api/v1/workflows/definitions
```

---

#### 2. 部署工作流

```http
POST /api/v1/workflows/deploy
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "name": "工单审批流程",
  "module": "ticket",
  "bpmn_xml": "<?xml version='1.0'?><bpmn:definitions>...</bpmn:definitions>",
  "description": "工单多级审批流程"
}
```

---

#### 3. 启动工作流实例

```http
POST /api/v1/workflows/instances
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "definition_id": "ticket_approval_001",
  "business_key": "TKT-20260304-000001",
  "variables": {
    "ticket_id": 123,
    "priority": "high",
    "total_amount": 5000
  }
}
```

---

### CMDB API

#### 1. 列出配置项

```http
GET /api/v1/cmdb/assets
```

---

#### 2. 创建配置项

```http
POST /api/v1/cmdb/assets
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "name": "MySQL 主库",
  "type": "database",
  "status": "running",
  "ip_address": "192.168.1.100",
  "os": "CentOS 7",
  "cpu": "16 cores",
  "memory": "64GB",
  "disk": "1TB SSD",
  "owner_id": 5,
  "relations": [
    {
      "type": "depends_on",
      "target_id": 2,
      "attributes": {
        "port": 3306
      }
    }
  ]
}
```

---

### SLA API

#### 1. 列出 SLA 策略

```http
GET /api/v1/sla/policies
```

---

#### 2. 创建 SLA 策略

```http
POST /api/v1/sla/policies
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "name": "P1 工单响应 SLA",
  "priority": "critical",
  "response_time": 30,     // 分钟
  "resolution_time": 240, // 分钟 (4 小时)
  "is_active": true,
  "conditions": {
    "category": ["故障", "紧急"]
  }
}
```

---

### 用户管理 API

#### 1. 列出用户

```http
GET /api/v1/users
```

**查询参数**:

| 参数 | 说明 |
|------|------|
| role | 角色过滤 |
| department_id | 部门过滤 |
| is_active | 是否活跃 |

---

#### 2. 创建用户

```http
POST /api/v1/users
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "email": "newuser@example.com",
  "password": "StrongPass123!",
  "name": "新用户",
  "role": "agent",
  "department_id": 1,
  "tenant_id": 1
}
```

---

## Webhook

### 配置 Webhook

```http
POST /api/v1/webhooks
Authorization: Bearer <token>
```

**请求体**:

```json
{
  "name": "钉钉通知",
  "url": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
  "events": ["ticket.created", "ticket.updated", "incident.created"],
  "secret": "optional-secret",
  "is_active": true
}
```

### 事件列表

| 事件名 | 触发时机 | 负载示例 |
|--------|----------|----------|
| `ticket.created` | 工单创建 | 见下方 |
| `ticket.updated` | 工单更新 | 见下方 |
| `ticket.assigned` | 工单分配 | 见下方 |
| `incident.created` | 事件创建 | 见下方 |
| `change.approved` | 变更审批通过 | 见下方 |
| `sla.breached` | SLA 违规 | 见下方 |

---

### Webhook Payload 示例

**ticket.created**:

```json
{
  "event": "ticket.created",
  "timestamp": "2026-03-04T10:30:00Z",
  "payload": {
    "ticket": {
      "id": 123,
      "ticket_number": "TKT-20260304-000001",
      "title": "无法登录系统",
      "status": "open",
      "priority": "high",
      "submitter": {
        "id": 1,
        "name": "张三"
      }
    }
  }
}
```

---

## API 变更日志

### v1.0 (2026-03-04) - Current

- ✅ 初始版本发布
- ✅ 认证 API (登录、刷新、登出)
- ✅ 工单管理 API
- ✅ 事件管理 API
- ✅ 问题管理 API
- ✅ 变更管理 API
- ✅ 知识库 API (含语义搜索)
- ✅ 工作流 API (BPMN)
- ✅ CMDB API
- ✅ SLA 策略 API
- ✅ 用户管理 API
- ✅ Webhook 支持

### 计划中的变更

- ⏳ 批量操作 API
- ⏳ 报表导出 (CSV/Excel)
- ⏳ 审计日志 API
- ⏳ 多语言支持
- ⏳ GraphQL 端点（可选）

---

## API 最佳实践

### 1. 分页请求

```bash
# ✅ 正确
GET /api/v1/tickets?page=1&page_size=20

# ❌ 避免
GET /api/v1/tickets?limit=1000&offset=0
```

### 2. 错误处理

```json
// 前端应检查 code 字段，而非 HTTP 状态码
if (response.code !== 0) {
  showError(response.message);
}
```

### 3. 并发控制

使用乐观锁:

```sql
UPDATE tickets
SET status = 'resolved', version = version + 1
WHERE id = ? AND version = ?  -- 提交时带上当前 version
```

---

## 测试 API

使用 `curl` 快速测试:

```bash
# 1. 登录
TOKEN=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | jq -r '.data.token')

# 2. 获取工单列表
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8090/api/v1/tickets?page=1&page_size=10"

# 3. 查看指标
curl http://localhost:8090/metrics
```

---

## 完整 OpenAPI 规范

详细 OpenAPI 3.0 定义请见 [OPENAPI.yaml](./OPENAPI.yaml) 文件。

如需生成交互式文档:

```bash
# 使用 Redoc
npx redoc-cli bundle OPENAPI.yaml -o docs.html

# 使用 Swagger UI
docker run -p 8081:8090 -e SWAGGER_JSON=/openapi.yaml \
  -v $(pwd)/OPENAPI.yaml:/openapi.yaml swaggerapi/swagger-ui
```

---

**文档维护**: ITSM API 团队
**最后更新**: 2026-03-04
