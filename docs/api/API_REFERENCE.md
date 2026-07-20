# ITSM API 参考文档

## 概述

本文档描述了 ITSM 系统的所有 API 接口。所有接口遵循 RESTful 设计原则，使用 JSON 格式进行数据交换。

## 基础信息

- **Base URL**: `http://localhost:8090/api/v1`
- **认证方式**: JWT Bearer Token
- **数据格式**: JSON
- **字符编码**: UTF-8

## 通用响应格式

所有 API 响应遵循以下格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 状态码说明

| 状态码 | 说明 |
|--------|------|
| 0 | 请求成功 |
| 1001 | 参数错误 |
| 2001 | 认证失败 |
| 4001 | 权限不足 |
| 5001 | 服务器内部错误 |

## 认证接口

### 登录

```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123",
  "tenantCode": "default"
}
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "name": "管理员",
      "role": "admin",
      "tenantId": 1
    },
    "tenant": {
      "id": 1,
      "name": "默认租户",
      "code": "default",
      "type": "standard",
      "status": "active"
    }
  }
}
```

### 刷新令牌

```http
POST /auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 登出

```http
POST /auth/logout
Authorization: Bearer <accessToken>
```

### 注册

```http
POST /auth/register
Content-Type: application/json

{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "fullName": "新用户",
  "phone": "13800138000",
  "company": "公司名称",
  "role": "user"
}
```

### 忘记密码

```http
POST /auth/forgot-password
Content-Type: application/json

{
  "email": "user@example.com",
  "tenantCode": "default"
}
```

### 重置密码

```http
POST /auth/reset-password
Content-Type: application/json

{
  "token": "reset-token",
  "email": "user@example.com",
  "password": "newpassword123",
  "passwordConfirm": "newpassword123"
}
```

## 工单接口

### 获取工单列表

```http
GET /tickets
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码 (默认: 1)
- pageSize: 每页数量 (默认: 20)
- status: 状态过滤
- priority: 优先级过滤
- categoryId: 分类过滤
- assigneeId: 负责人过滤
- search: 搜索关键词
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tickets": [
      {
        "id": "1",
        "ticketNumber": "TK-2024-00001",
        "title": "无法登录系统",
        "description": "用户报告无法登录系统",
        "status": "open",
        "priority": "high",
        "categoryId": 1,
        "assigneeId": 2,
        "reporterId": 3,
        "tenantId": 1,
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-02T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

### 创建工单

```http
POST /tickets
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "无法登录系统",
  "description": "用户报告无法登录系统",
  "priority": "high",
  "categoryId": 1,
  "tags": ["登录", "认证"]
}
```

### 获取工单详情

```http
GET /tickets/{id}
Authorization: Bearer <accessToken>
```

### 更新工单

```http
PUT /tickets/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "in_progress",
  "priority": "medium",
  "assigneeId": 3
}
```

### 更新工单状态

```http
PUT /tickets/{id}/status
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "status": "resolved",
  "resolution": "已重置密码"
}
```

### 删除工单

```http
DELETE /tickets/{id}
Authorization: Bearer <accessToken>
```

### 分配工单

```http
POST /tickets/{id}/assign
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "assigneeId": 2
}
```

### 工单流转操作

```http
POST /tickets/{id}/workflow/accept
POST /tickets/{id}/workflow/reject
POST /tickets/{id}/workflow/withdraw
POST /tickets/{id}/workflow/forward
POST /tickets/{id}/workflow/cc
POST /tickets/{id}/workflow/approve
POST /tickets/{id}/workflow/resolve
POST /tickets/{id}/workflow/close
POST /tickets/{id}/workflow/reopen
```

### 获取工单评论

```http
GET /tickets/{id}/comments
Authorization: Bearer <accessToken>
```

### 添加工单评论

```http
POST /tickets/{id}/comments
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "content": "这是一条评论",
  "isInternal": false
}
```

### 获取工单附件

```http
GET /tickets/{id}/attachments
Authorization: Bearer <accessToken>
```

### 上传工单附件

```http
POST /tickets/{id}/attachments
Authorization: Bearer <accessToken>
Content-Type: multipart/form-data

file: [binary data]
```

## 事件管理接口

### 获取事件列表

```http
GET /incidents
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
- severity: 严重程度过滤
- search: 搜索关键词
```

### 创建事件

```http
POST /incidents
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "服务器宕机",
  "description": "生产服务器宕机，需要紧急处理",
  "severity": "critical",
  "impact": "high",
  "categoryId": 1
}
```

### 获取事件详情

```http
GET /incidents/{id}
Authorization: Bearer <accessToken>
```

### 更新事件

```http
PUT /incidents/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "resolved",
  "severity": "medium"
}
```

### 删除事件

```http
DELETE /incidents/{id}
Authorization: Bearer <accessToken>
```

## 问题管理接口

### 获取问题列表

```http
GET /problems
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
- priority: 优先级过滤
```

### 创建问题

```http
POST /problems
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "系统响应缓慢",
  "description": "用户反馈系统响应时间过长",
  "priority": "high",
  "category": "performance"
}
```

### 获取问题详情

```http
GET /problems/{id}
Authorization: Bearer <accessToken>
```

### 更新问题

```http
PUT /problems/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "in_progress"
}
```

### 删除问题

```http
DELETE /problems/{id}
Authorization: Bearer <accessToken>
```

## 变更管理接口

### 获取变更列表

```http
GET /changes
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
- risk: 风险等级过滤
```

### 创建变更

```http
POST /changes
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "升级数据库版本",
  "description": "升级 PostgreSQL 到 14 版本",
  "type": "standard",
  "risk": "medium",
  "plannedStartAt": "2024-01-15T00:00:00Z",
  "plannedEndAt": "2024-01-15T02:00:00Z"
}
```

### 获取变更详情

```http
GET /changes/{id}
Authorization: Bearer <accessToken>
```

### 更新变更

```http
PUT /changes/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "approved",
  "risk": "low"
}
```

### 删除变更

```http
DELETE /changes/{id}
Authorization: Bearer <accessToken>
```

### 变更状态流转

```http
POST /changes/{id}/submit
POST /changes/{id}/assign
POST /changes/{id}/approve
POST /changes/{id}/reject
POST /changes/{id}/start
POST /changes/{id}/complete
POST /changes/{id}/rollback
POST /changes/{id}/cancel
```

## 发布管理接口

### 获取发布列表

```http
GET /releases
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
```

### 创建发布

```http
POST /releases
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "v1.1.0 发布",
  "description": "新功能发布",
  "version": "1.1.0",
  "plannedAt": "2024-01-20T00:00:00Z"
}
```

### 获取发布详情

```http
GET /releases/{id}
Authorization: Bearer <accessToken>
```

### 更新发布

```http
PUT /releases/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "in_progress"
}
```

### 更新发布状态

```http
PUT /releases/{id}/status
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "status": "completed"
}
```

### 删除发布

```http
DELETE /releases/{id}
Authorization: Bearer <accessToken>
```

## 服务目录接口

### 获取服务目录

```http
GET /service-catalog
Authorization: Bearer <accessToken>

Query Parameters:
- categoryId: 分类过滤
- search: 搜索关键词
```

### 获取服务项详情

```http
GET /service-catalog/{id}
Authorization: Bearer <accessToken>
```

## 知识库接口

### 获取知识文章列表

```http
GET /knowledge/articles
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- categoryId: 分类过滤
- search: 搜索关键词
```

### 获取知识文章详情

```http
GET /knowledge/articles/{id}
Authorization: Bearer <accessToken>
```

### 创建知识文章

```http
POST /knowledge/articles
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "如何重置密码",
  "content": "详细的密码重置步骤...",
  "categoryId": 1,
  "tags": ["密码", "账户"],
  "isPublished": true
}
```

### 更新知识文章

```http
PUT /knowledge/articles/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "content": "更新后的内容",
  "categoryId": 1
}
```

### 删除知识文章

```http
DELETE /knowledge/articles/{id}
Authorization: Bearer <accessToken>
```

### 知识库搜索

```http
POST /knowledge/search
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "query": "如何重置密码",
  "limit": 10
}
```

### RAG 增强搜索

```http
POST /knowledge/ask
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "query": "如何重置密码",
  "maxResults": 5
}
```

## SLA 管理接口

### 获取 SLA 策略列表

```http
GET /sla/policies
Authorization: Bearer <accessToken>
```

### 获取 SLA 统计

```http
GET /sla/statistics
Authorization: Bearer <accessToken>
```

### 获取 SLA 违规记录

```http
GET /sla/violations
Authorization: Bearer <accessToken>
```

## 工作流接口

### 获取工作流列表

```http
GET /workflows
Authorization: Bearer <accessToken>
```

### 获取工作流实例

```http
GET /workflows/instances/{id}
Authorization: Bearer <accessToken>
```

### 启动工作流

```http
POST /workflows/instances
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "workflowId": "workflow-1",
  "variables": {
    "ticketId": 123
  }
}
```

## 用户管理接口

### 获取用户列表

```http
GET /users
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- search: 搜索关键词
- role: 角色过滤
```

### 获取用户详情

```http
GET /users/{id}
Authorization: Bearer <accessToken>
```

### 创建用户

```http
POST /users
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "username": "newuser",
  "email": "newuser@example.com",
  "name": "新用户",
  "password": "password123",
  "role": "user",
  "departmentId": 1
}
```

### 更新用户

```http
PUT /users/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "email": "updated@example.com",
  "name": "更新后的名称",
  "role": "agent",
  "departmentId": 2
}
```

### 删除用户

```http
DELETE /users/{id}
Authorization: Bearer <accessToken>
```

## 资产管理接口

### 获取资产列表

```http
GET /assets
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
- type: 类型过滤
- search: 搜索关键词
```

### 获取资产详情

```http
GET /assets/{id}
Authorization: Bearer <accessToken>
```

### 创建资产

```http
POST /assets
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "服务器",
  "type": "server",
  "status": "active",
  "assetNumber": "AST-001",
  "serialNumber": "SN123456",
  "vendorId": 1,
  "purchaseDate": "2024-01-01",
  "warrantyExpireAt": "2026-01-01",
  "assignedUserId": 2,
  "location": "机房 A",
  "specifications": {
    "cpu": "Intel Xeon",
    "ram": "64GB",
    "storage": "2TB SSD"
  }
}
```

### 更新资产

```http
PUT /assets/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "更新后的名称",
  "status": "in_maintenance",
  "assignedUserId": 3
}
```

### 分配资产

```http
PUT /assets/{id}/assign
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "userId": 2
}
```

### 报废资产

```http
PUT /assets/{id}/retire
Authorization: Bearer <accessToken>
```

### 删除资产

```http
DELETE /assets/{id}
Authorization: Bearer <accessToken>
```

### 获取资产统计

```http
GET /assets/statistics
Authorization: Bearer <accessToken>
```

## 许可证管理接口

### 获取许可证列表

```http
GET /licenses
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- status: 状态过滤
- search: 搜索关键词
```

### 获取许可证详情

```http
GET /licenses/{id}
Authorization: Bearer <accessToken>
```

### 创建许可证

```http
POST /licenses
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "Windows Server 2022",
  "softwareName": "Windows Server",
  "licenseKey": "XXXXX-XXXXX-XXXXX-XXXXX-XXXXX",
  "vendorId": 1,
  "purchaseDate": "2024-01-01",
  "expireAt": "2025-01-01",
  "totalSeats": 50,
  "usedSeats": 30,
  "status": "active"
}
```

### 更新许可证

```http
PUT /licenses/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "更新后的名称",
  "expireAt": "2026-01-01",
  "totalSeats": 100
}
```

### 分配许可证给用户

```http
PUT /licenses/{id}/assign
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "userIds": [1, 2, 3]
}
```

### 删除许可证

```http
DELETE /licenses/{id}
Authorization: Bearer <accessToken>
```

### 获取许可证统计

```http
GET /licenses/statistics
Authorization: Bearer <accessToken>
```

## CMDB 接口

### 获取配置项列表

```http
GET /cmdb/items
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- typeId: 类型过滤
- search: 搜索关键词
```

### 获取配置项详情

```http
GET /cmdb/items/{id}
Authorization: Bearer <accessToken>
```

### 创建配置项

```http
POST /cmdb/items
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "typeId": 1,
  "name": "应用服务器",
  "status": "active",
  "attributes": {
    "ip": "192.168.1.100",
    "os": "Ubuntu 20.04",
    "cpu": "8 cores",
    "memory": "32GB"
  },
  "tags": ["server", "production"]
}
```

### 更新配置项

```http
PUT /cmdb/items/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "更新后的名称",
  "status": "maintenance",
  "attributes": {
    "ip": "192.168.1.101"
  }
}
```

### 删除配置项

```http
DELETE /cmdb/items/{id}
Authorization: Bearer <accessToken>
```

### 获取配置项关系

```http
GET /cmdb/items/{id}/relationships
Authorization: Bearer <accessToken>
```

### 获取配置项拓扑图

```http
GET /cmdb/items/{id}/topology
Authorization: Bearer <accessToken>
```

## 通知接口

### 获取通知列表

```http
GET /notifications
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码
- pageSize: 每页数量
- type: 类型过滤
- read: 是否已读 (true/false)
```

### 标记通知已读

```http
PUT /notifications/{id}/read
Authorization: Bearer <accessToken>
```

### 标记所有通知已读

```http
PUT /notifications/read-all
Authorization: Bearer <accessToken>
```

### 删除通知

```http
DELETE /notifications/{id}
Authorization: Bearer <accessToken>
```

### 获取未读数量

```http
GET /notifications/unread-count
Authorization: Bearer <accessToken>
```

## 仪表板接口

### 获取仪表板数据

```http
GET /dashboard
Authorization: Bearer <accessToken>
```

### 获取工单统计

```http
GET /dashboard/ticket-stats
Authorization: Bearer <accessToken>
```

### 获取 SLA 统计

```http
GET /dashboard/sla-stats
Authorization: Bearer <accessToken>
```

## 系统配置接口

### 获取系统配置列表

```http
GET /system-configs
Authorization: Bearer <accessToken>
```

### 获取系统配置

```http
GET /system-configs/{id}
Authorization: Bearer <accessToken>
```

### 获取系统配置（按键）

```http
GET /system-configs/key/{key}
Authorization: Bearer <accessToken>
```

### 更新系统配置

```http
PUT /system-configs/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "value": "new value"
}
```

### 批量更新系统配置

```http
PUT /system-configs/batch
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "configs": [
    {
      "key": "config1",
      "value": "value1"
    },
    {
      "key": "config2",
      "value": "value2"
    }
  ]
}
```

### 初始化默认配置

```http
GET /system-configs/init
Authorization: Bearer <accessToken>
```

## 搜索接口

### 全局搜索

```http
GET /search
Authorization: Bearer <accessToken>

Query Parameters:
- q: 搜索关键词
- type: 搜索类型 (ticket/incident/problem/knowledge/user)
- limit: 结果数量限制
```

## 错误处理

所有错误响应遵循以下格式：

```json
{
  "code": 1001,
  "message": "参数错误",
  "data": null
}
```

### 常见错误码

| 错误码 | 描述 | HTTP 状态码 |
|--------|------|------------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 数据验证失败 | 400 |
| 2001 | 认证失败 | 401 |
| 2002 | Token 已过期 | 401 |
| 2003 | Token 无效 | 401 |
| 4001 | 权限不足 | 403 |
| 4002 | 资源不存在 | 404 |
| 4003 | 资源已存在 | 409 |
| 5001 | 服务器内部错误 | 500 |
| 5002 | 服务暂不可用 | 503 |

## 分页

所有列表接口支持分页：

**请求参数:**
- `page`: 页码 (从 1 开始)
- `pageSize`: 每页数量 (默认 20, 最大 100)

**响应数据:**
```json
{
  "items": [],
  "total": 100,
  "page": 1,
  "pageSize": 20,
  "totalPages": 5
}
```

## 排序

部分列表接口支持排序：

**请求参数:**
- `sortBy`: 排序字段
- `sortOrder`: 排序方向 (asc/desc)

## 速率限制

- 普通用户: 100 次/分钟
- 管理员: 500 次/分钟
- API 调用超出限制将返回 429 状态码

## WebSocket

### 通知推送

连接地址: `ws://localhost:8090/api/v1/ws/notifications`

需要使用票据认证：
1. 先调用 `POST /api/v1/ws/ticket` 获取临时票据
2. 使用票据连接 WebSocket: `ws://localhost:8090/api/v1/ws/notifications?ticket=<ticket>`

### 消息格式

```json
{
  "type": "notification",
  "data": {
    "id": 1,
    "title": "新工单创建",
    "message": "您有一个新工单待处理",
    "type": "ticket",
    "createdAt": "2024-01-01T00:00:00Z"
  }
}
```

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2024-01-01 | 初始版本 |

## 联系支持

如有问题，请联系技术支持团队。
