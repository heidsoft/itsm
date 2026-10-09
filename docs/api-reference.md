# ITSM API 参考文档

> Status: current

## 概述

本文档描述了 ITSM 系统的所有 API 接口。所有接口遵循 RESTful 设计原则，使用 JSON 格式进行数据交换。

## 基础信息

- **Base URL**: `http://localhost:8090/api/v1`
- **认证方式**: JWT Bearer Token
- **数据格式**: JSON
- **字符编码**: UTF-8

## AI 工作流生成

AI 工作流接口必须携带 JWT，并且只能访问认证上下文中的租户。请求体中的 `tenantId`（如存在）不会覆盖认证租户。

```http
POST /api/v1/bpmn/ai/generate?autoDeploy=false
POST /api/v1/bpmn/ai/preview
GET  /api/v1/bpmn/ai/templates/suggestions?keyword=事件&processType=incident
```

`generate` 需要 `workflow:create` 权限，`preview` 和模板推荐需要 `workflow:read` 权限。生成响应包含 `bpmnXml`、`lintResult` 和待人工确认的 `candidateDefinition`；只有 Lint 没有错误时才允许自动部署。自动部署成功时同时返回 `deploymentId`、`processDefinitionId` 和实际版本 `version`。AI provider、模型和失败原因记录在现有 AI 可观测链路中，服务不可用返回 `5003`，不会返回空成功结果。

### AI 工作流模板治理

租户管理员可通过以下接口管理模板草稿和发布版本。所有接口使用认证租户范围，模板发布会再次执行 BPMN Lint，已发布版本不可直接覆盖编辑。

```http
GET  /api/v1/bpmn/ai/templates?keyword=费用&domain=expense&status=draft&page=1&pageSize=20
GET  /api/v1/bpmn/ai/templates/{key}
GET  /api/v1/bpmn/ai/templates/{key}/versions
POST /api/v1/bpmn/ai/templates
PUT  /api/v1/bpmn/ai/templates/{key}
POST /api/v1/bpmn/ai/templates/{key}/publish
POST /api/v1/bpmn/ai/templates/{key}/archive
```

创建模板会生成新的草稿版本（例如 `1.0.0`、`1.1.0`），编辑只允许作用于草稿；发布成功后状态为 `published`，停用只将最新已发布版本标记为 `archived`，不会删除历史版本。

## AI 审计记录

需要 `ai:read` 权限，租户范围只取认证上下文，查询参数中的同名字段不生效。

```http
GET /api/v1/ai/audit-logs?page=1&pageSize=20&days=90&kind=triage
```

分页规则由 `common.GetPaginationFromQuery` 单点决定：缺省 `page=1`、`pageSize=20`，只采纳 `1–100`，越界或非数字回落 20，`page<=0` 归一为 1。响应 `data` 是标准五键列表信封 `{items,total,page,pageSize,totalPages}`，`page`/`pageSize` 回显的是**实际采纳值**而不是客户端原值。`days` 是回溯窗口天数（缺省及非正值均为 90），`kind` 为空时不加该过滤。

`items` 为 `AIAuditEntry`：`id`、`createdAt`、`tenantId`、`userId`、`requestId`、`scenario`、`inputRef`、`promptVersion`、`model`、`confidence`、`accepted`、`suggestion`、`notes`。排序固定为 `created_at DESC, id DESC`；只统计 `item_type='ai_audit'` 的记录，空结果集序列化为 `[]`。缺少租户上下文返回 401 / `2001` 且响应体只有这一份 JSON。

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

业务码与 HTTP 状态由 `itsm-backend/common/response.go` 的 `codeStatusPairs` 单点登记，
完整口径见本文「常见错误码」一节。

| 状态码 | 说明 |
|--------|------|
| 0 | 请求成功 |
| 1001 | 参数错误 |
| 2001 | 认证失败 |
| 2003 | 权限不足（禁止访问） |
| 4004 | 资源不存在（含跨租户探测） |
| 4090 | 状态/版本冲突 |
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

登录和刷新接口不再在 JSON 响应中返回令牌，改为通过 HttpOnly cookie 下发（`access_token` 15 分钟，`refresh_token` 7 天）。`expiresIn` 是服务端时钟给出的 access token 剩余秒数，前端据此安排续签，不得用浏览器时间推算。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "expiresIn": 900,
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "name": "管理员",
      "role": "admin",
      "tenantId": 1
    }
  }
}
```

### 刷新令牌

```http
POST /auth/refresh
```

浏览器端以空 body 调用，后端回退读取 `refresh_token` httpOnly cookie（也接受 `{"refreshToken": "..."}`，供非浏览器调用方）。

**refresh token 单次使用**：每次续签原子认领旧凭证并下发新凭证，重放旧值返回 `401`（业务码 2001）。并发续签中只有第一个请求能完成 rotation。

### 登出

```http
POST /auth/logout
```

**不需要认证凭证。** access token 过期（15 分钟）后仍必须能登出，否则 7 天的 `refresh_token` cookie 会留在浏览器里把会话自动续签回来。登出先无条件清除两类 cookie，再吊销请求中携带的 access/refresh token；吊销存储故障时返回 `503`（业务码 5003）并说明凭证已清除、服务端吊销未完成。重复登出幂等返回成功。

### 当前会话

```http
GET /auth/session
```

前端唯一的「是否已登录」真相：身份、可切换租户与服务端剩余有效期。前端不得再从 `document.cookie`、JWT 形状或持久化 store 推断登录态。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": { "id": 1, "username": "admin", "role": "admin", "tenantId": 1 },
    "tenants": [
      { "id": 1, "name": "默认租户", "code": "default", "type": "standard", "status": "active" }
    ],
    "expiresIn": 847
  }
}
```

未携带有效凭证时返回 `401`（业务码 2001）。响应中不含任何令牌值。

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
  "company": "公司名称"
}
```

角色与租户由服务端决定：注册是唯一未认证的写入入口，请求体中的 `role`/`tenantCode`
字段已移除（历史字段，携带时不报错但不再产生任何效果），落库角色固定为 `end_user`。
目标租户取系统中唯一的活跃租户；存在多个活跃租户时注册被拒绝（fail-closed），
需由管理员开通账号。

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
    "items": [
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
    "pageSize": 20,
    "totalPages": 5
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

响应是**不分页**的两键信封：`data = {items: TicketAttachmentResponse[], total}`，
`total` 等于 `items` 长度（后端按 ticket+tenant 全量返回，`created_at DESC` 排序，
无 `id` 次序键）。条目字段：`id`、`ticketId`、`fileName`、`filePath`、`fileUrl`、
`fileSize`、`fileType`、`mimeType`、`uploadedBy`、`uploader?`、`createdAt`。
`fileUrl` 由 `(ticketId, id)` 推导，等于下面的下载路由。集合键在 2026-10-04 之前叫
`attachments`，见 UPGRADE §1.17。

### 上传工单附件

```http
POST /tickets/{id}/attachments
Authorization: Bearer <accessToken>
Content-Type: multipart/form-data

file: [binary data]
```

### 下载工单附件

响应是文件流（`Content-Disposition: attachment`），不是 `{code, message, data}` 信封。列表与上传响应里的 `fileUrl` 就是这个地址，由 `(ticketId, id)` 推导，可直接点击。

```http
GET /tickets/{id}/attachments/{attachmentId}
Authorization: Bearer <accessToken>
```

### 预览工单附件

同一次下游读取，`Content-Disposition: inline`，供图片/PDF 在浏览器内打开。

```http
GET /tickets/{id}/attachments/{attachmentId}/preview
Authorization: Bearer <accessToken>
```

### 工单聚合统计

工单域的租户级聚合端点，与仪表板章节的 `GET /dashboard/ticket-stats` 不是同一个接口。

```http
GET /api/v1/tickets/stats
```

`data` 是租户全量的聚合计数，不是列表，因此不带任何分页字段：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 201,
    "open": 41,
    "inProgress": 40,
    "resolved": 50,
    "closed": 60,
    "pending": 5,
    "highPriority": 12,
    "overdue": 7,
    "byStatus": [{ "status": "in_progress", "count": 40 }],
    "byPriority": [{ "priority": "high", "count": 12 }]
  }
}
```

- `open` 含 `new`；`overdue` 由 `sla_states` 判定，工单状态词表里并没有 `overdue` 这个取值。
- `byStatus`/`byPriority` 只含真实存在的取值，顺序按后端词表基准，词表外的历史值按字母序追加；两者各自求和恒等于 `total`，且严格按租户过滤（`handlers/ticket/stats_contract_test.go` 锁死这几条）。
- 报表要分布就读这两个字段。禁止改成前端拉一页列表自己数：页长之外的工作量会被静默截断，界面却仍然把它当成全量读数。

## 工单标签接口

标签的唯一可写所有者是工单标签（`ticket_tags`）。`GET /tags`、`GET /system/tags` 是历史只读别名，不承载写操作。

### 获取标签列表

```http
GET /ticket-tags?page=1&pageSize=20&isActive=true
Authorization: Bearer <accessToken>
```

响应 `data` 为 `{ items, total }`，只包含当前租户的标签。

### 创建标签

```http
POST /ticket-tags
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "网络",
  "color": "#1890ff",
  "description": "网络设备相关",
  "isActive": true
}
```

`tenantId` 只取自认证上下文，请求体不需要也不接受该字段。`color` 缺省时使用 `#1890ff`；`isActive` 缺省时为启用。同名（同租户）返回 `409`/`code:4090`。

### 更新与删除标签

```http
PUT /ticket-tags/{id}
DELETE /ticket-tags/{id}
Authorization: Bearer <accessToken>
```

标签仍被本租户工单引用时删除返回 `409`/`code:4090`；跨租户访问统一返回 `404`/`code:4004`。

### 工单绑定标签

```http
POST /tickets/{id}/tags
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "tagIds": [1, 2]
}
```

也可以用 `{"tags": ["网络"]}` 按名称绑定，缺失的标签会在当前租户内自动创建。解绑使用 `DELETE /tickets/{id}/tags`，请求体相同。

## 工单依赖影响分析接口

分析对象是当前租户内某工单的子工单集合（`parent_ticket_id`），权限 `ticket:read`。被分析的工单只能来自路径段，租户只来自认证上下文；请求里自报的 `tenantId` 不参与判定。

```http
GET /tickets/{id}/dependencies?action=close
Authorization: Bearer <accessToken>
```

| 查询参数 | 必填 | 说明 |
|---|---|---|
| `action` | 是 | `close`、`delete`、`change_status` 三选一；缺失或非法取值返回 400/1001 |
| `newStatus` | 否 | 仅 `action=change_status` 使用，例如 `closed`、`resolved` |

响应 `data` 为分析结果，风险等级按告警数量分级（0 条 `low`，不超过 2 条 `medium`，更多 `high`）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ticketId": 1,
    "ticketNumber": "T-2024-001",
    "ticketTitle": "系统响应缓慢",
    "action": "close",
    "affectedCount": 2,
    "affectedTickets": [
      {
        "id": 2,
        "number": "T-2024-002",
        "title": "数据库优化",
        "status": "in_progress",
        "impactType": "blocked",
        "description": "父工单关闭可能导致此工单无法继续"
      }
    ],
    "warnings": ["子工单 T-2024-002 (数据库优化) 尚未完成"],
    "recommendations": ["建议先完成或取消所有子工单后再关闭父工单"],
    "riskLevel": "medium"
  }
}
```

`impactType` 随 `action` 变化：`close` 为 `blocked`、`delete` 为 `orphaned`、`change_status` 为 `status_change`。工单不存在与跨租户访问统一返回 404/4004（不确认对方资源是否存在），底层错误只进日志。

## 工单关联查询接口

三条只读端点，权限均为 `ticket:read`，实现在 `service/ticketassociationservice.go`。工单 ID 只来自路径段，租户只来自认证上下文；请求体或查询参数里自报的 `tenantId` 不参与判定。

```http
GET  /tickets/{id}/relations
GET  /tickets/{id}/relations/stats
GET  /tickets/{id}/configuration-items
Authorization: Bearer <accessToken>
```

`/relations` 的 `data` 是三条遍历路径的扁平集合（`parentChain` 自根向下、`childrenTree` 为直接子工单、`relatedTickets` 来自 `related_tickets` M2M 边），三项都是同一份 `service.TicketResponse` 投影：必有 `id`、`title`、`description`、`priority`、`status`、`tenantId`、`createdAt`、`updatedAt`，可选 `categoryId`、`templateId`、`parentId`、`relatedIds`、`tagIds`、`assignedTo`、`customFields`（零值/空值按 `omitempty` 省略）。三个集合恒为数组，没有关联时是 `[]` 而不是 `null`。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "childrenTree": [
      {"id": 2, "title": "数据库优化", "description": "…", "priority": "medium", "status": "open", "parentId": 1, "tenantId": 1, "assignedTo": 7, "createdAt": "2026-10-04T12:16:11+08:00", "updatedAt": "2026-10-04T12:16:11+08:00"}
    ],
    "parentChain": [
      {"id": 1, "title": "数据库性能劣化", "description": "…", "priority": "high", "status": "open", "tenantId": 1, "createdAt": "2026-10-04T12:16:11+08:00", "updatedAt": "2026-10-04T12:16:11+08:00"}
    ],
    "relatedTickets": [
      {"id": 3, "title": "网络抖动", "description": "…", "priority": "medium", "status": "open", "relatedIds": [4], "tenantId": 1, "createdAt": "2026-10-04T12:16:11+08:00", "updatedAt": "2026-10-04T12:16:11+08:00"}
    ]
  }
}
```

（外层 `data` 由 handler 用 `gin.H` 组装，所以键是按字典序输出的，上例即实测顺序。）

`/relations/stats` 与 `/configuration-items` 都不分页：前者返回十个计数键（`totalRelations`、`relationsByType`、`inboundCount`、`outboundCount`、`parentCount`、`childrenCount`、`blockedByCount`、`blockingCount`、`relatedCount`、`duplicateCount`），后者 `data` 直接是配置项数组 `{id, name, ciType, status, serialNumber?}`，无关联时同样是 `[]` 而非 `null`。

失败语义（由 `common.classifyError` 单点决定，回归见 `itsm-backend/router/ticket_relations_tenant_scope_route_test.go`）：

| 情况 | HTTP | 业务 code | 说明 |
|---|---|---|---|
| `id` 非正整数 | 400 | `1001` | `无效的工单ID`，不与「不存在」混淆 |
| 缺租户上下文 | 401 | `2001` | fail-closed，不回退默认租户 |
| 工单不存在 **与** 工单属于对方租户 | 404 | `4004` | 两种情况**逐字节相同**，调用者无法据此枚举其他租户的工单 ID |
| 数据库真实故障 | 500 | `5001` | 原始错误只进日志，响应文案是固定安全串 |

`parentChain` 遍历有 64 层深度上限与环检测，脏数据里的父子环不会让请求挂死。

> **如实申报**：`/relations/stats` 的 `blockedByCount`、`blockingCount`、`duplicateCount` 恒为 `0`、`relationsByType` 恒为 `{}`——该端点只统计 parent/related 三条路径，没有依赖关系（blocks/duplicates）的数据来源，这些键是为前端既有形状保留的占位读数，不代表「已计算且结果为零」。关联的**写入**端点（`UpdateTicketAssociations`、`Add/Remove/SetConfigurationItems`）目前只有 service 实现、没有任何注册路由，见台账 E4-35b。

## 事件管理接口

### 获取事件列表

```http
GET /incidents
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省 1；非正值回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- status: 状态过滤
- priority: 优先级过滤
- keyword: 搜索关键词（匹配标题/描述/事件编号）
- source: 来源过滤（manual / monitoring / email …）
- type: 事件类型过滤
- category: 事件分类过滤
- assigneeId: 处理人过滤（必须为正整数，否则返回 400/1001）
- isMajorIncident: 重大事件过滤（true/false；未传表示不过滤，false 是有效过滤值）
- dateFrom / dateTo: 创建时间区间（RFC3339 或 YYYY-MM-DD，格式错误返回 400/1001）
- scope: 取值 me 时仅返回分配给当前登录用户的事件；缺少认证上下文返回 401
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`GET /api/v1/incidents/alerts/active` 用同一套分页规则与信封。

> 破坏性变更（2026-10-04 E4-9b）：分页解析收敛到 `common.GetPaginationFromQuery` 单一所有者，
> 缺省页长由 handler 私有的 10 改为平台缺省 20，上界由文档自称的 200 收敛为 100。
> 此前 handler 把未经校验的解析结果直接下传给 Ent 的 `Offset/Limit`，而 `Limit(0)` 等于不加
> LIMIT 子句：`pageSize=abc`、`pageSize=0` 或 `page=0` 会让一次写错分页参数的请求把该租户的
> 事件整表读穿，同时信封又回显另一套夹紧后的页长，页数统计与实际读取分家。
> 回归见 `itsm-backend/handlers/incident/list_pagination_contract_test.go`、
> `alert_pagination_route_test.go`，新增同类写法由 `tests/contract/page_size_owner_ratchet_test.go` 拦下。

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

### 事件趋势报表

事件域唯一的时间窗口聚合端点，`/reports/incident-trends` 的曲线、分布与均值只能以它为准；
`GET /incidents` 的列表页长不构成统计来源。需要 `incident:read` 权限。

```http
GET /api/v1/incidents/stats/report
Authorization: Bearer <accessToken>

Query Parameters:
- dateFrom / dateTo: 窗口两端，接受 RFC3339 或 YYYY-MM-DD；必须成对提供，都不传取最近 30 天（含当天）
```

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "window": { "dateFrom": "2026-09-05", "dateTo": "2026-10-04", "days": 30 },
    "createdInWindow": 6,
    "resolvedInWindow": 3,
    "avgResolutionMinutes": 90,
    "byStatus": [
      { "value": "new", "count": 2 },
      { "value": "in_progress", "count": 1 },
      { "value": "resolved", "count": 2 },
      { "value": "awaiting_vendor", "count": 1 }
    ],
    "byPriority": [{ "value": "critical", "count": 2 }],
    "dailyTrend": [
      { "date": "2026-10-02", "created": 2, "resolved": 0 },
      { "date": "2026-10-03", "created": 1, "resolved": 2 },
      { "date": "2026-10-04", "created": 0, "resolved": 0 }
    ]
  }
}
```

口径（`itsm-backend/handlers/incident/report_contract_test.go` + `router/incident_report_route_test.go` 锁死）：

- 窗口是服务器本地时区的整日半开区间 `[Start, End)`，`dateTo` 为闭区间日；`window` 回显实际生效窗口，前端显示回显值而不是自己请求的值。
- 两个集合不是同一批记录：`createdInWindow` 按 `created_at` 归 cohort，`resolvedInWindow` 按 `resolved_at` 归窗口（状态取词表内的已解决/已关闭且 `resolved_at` 非空）。因此曲线两条线各自独立，堆叠会加成一条无意义的合计线。
- `byStatus` 与 `byPriority` 之和各自恒等于 `createdInWindow`；只含窗口内真实存在的取值，按后端词表顺序排列，词表外的历史脏值按字典序追加并原样返回（前端原样显示、不并进已知桶）。
- `avgResolutionMinutes` 单位是**分钟**（`resolved_at - created_at` 的均值，四舍五入到整数分钟）；`resolvedInWindow` 为 0 时后端返回 0，语义是「无样本」而不是「0 分钟」，消费方不得把它渲染成读数 0。
- `dailyTrend` 按窗口天数补零，没有事件的日子是 `created: 0, resolved: 0` 而不是缺项。
- 参数错误统一 HTTP 400 + 业务码 `1001`：只传一端（`dateFrom 与 dateTo 必须同时提供`）、区间倒置、日期无法解析、跨度超过 366 天；这些一律拒绝，不回落到默认窗口。
- 缺少租户上下文 HTTP 401 + `2001`，fail closed；查询始终带 `tenant_id` 与 `deleted_at IS NULL`，跨租户记录不进任何计数。
- `GET /incidents/stats` 是另一个端点：PostgreSQL 专有 SQL（`COUNT(*) FILTER`、`EXTRACT(EPOCH)`）的租户全量标量快照，不接受窗口参数，也不含分布；两者不要混用。

## 问题管理接口

### 获取问题列表

```http
GET /problems
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- status: 状态过滤（精确匹配单个值：`open`/`investigating`/`resolved`/`closed`/`identified`；`in_progress` 仅存在于历史数据）
- priority: 优先级过滤（精确匹配单个值：`low`/`medium`/`high`/`critical`）
- category: 分类过滤（精确匹配）
- keyword: 标题或描述关键词（`Contains`，两条 OR）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `ProblemResponse`（camelCase，含 `problemNumber`/`assigneeId`/`assigneeName`/
`createdBy`/`createdByName`）。空结果序列化为 `[]`。排序固定为 `created_at DESC, id ASC`，
`id` 兜底是为了让同一秒创建的问题在页边界归属确定。

口径说明（2026-10-03 E4-6c 实测收口）：

- **集合键只有一个 `items`。** 历史响应键是 `problems`，前端有三份列表响应类型声明（`problem-api.ts`、`problem-service.ts`、`types/biz/problem.ts`）、
  消费点写成 `resp.problems || resp.items || []`，属 AGENTS「请求/响应多字段兼容零新增」禁止项；
  现已按标准信封收敛，契约测试锁死五键集合。
- **分页夹紧只有一个所有者。** 此前 handler 传原值、repository 自己按 `size>200→200` 截断、
  响应又回显未夹紧的原值：205 条数据配 `pageSize=250` 时响应写着 `pageSize:250、totalPages:1`
  却只给 200 条，客户端按声明翻完第 1 页就停，第 201~205 条在任何一页都拿不到。现在 SQL 侧与
  声明侧读同一份 `common.GetPaginationFromQuery` 结果。
- **`sortBy`/`sortOrder`/`dateFrom`/`dateTo` 已从请求 DTO 删除。** 实测这四个字段既不在 handler
  的 filters 里也不在 repository 的查询条件里，传与不传结果完全相同，属假契约。排序需求请走固定
  排序或等后端实现后再宣传；未识别的查询参数一律忽略，不报错也不生效。
- **行级数据权限**：非管理角色（`super_admin`/`admin`/`manager`/`sysadmin` 之外）只返回
  `createdBy` 或 `assigneeId` 等于本人的问题；上下文缺失 `user_id` 时 fail-closed 返回空集。
  租户收敛与软删除排除在上游按 `tenant_id`、`deleted_at IS NULL` 生效。

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

### 问题聚合统计

问题域的租户级聚合端点。`/reports/problem-efficiency` 的状态与优先级分布只能以它为准，
`GET /problems` 的列表页长不构成统计来源。

```http
GET /api/v1/problems/stats
```

`data` 是租户全量的聚合计数，不带分页字段，也不接受查询参数（租户取自认证上下文）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 150,
    "open": 60,
    "inProgress": 50,
    "resolved": 30,
    "closed": 7,
    "highPriority": 22,
    "byStatus": [
      { "status": "open", "count": 60 },
      { "status": "investigating", "count": 35 },
      { "status": "identified", "count": 2 },
      { "status": "in_progress", "count": 15 },
      { "status": "resolved", "count": 30 },
      { "status": "closed", "count": 7 },
      { "status": "awaiting_vendor", "count": 1 }
    ],
    "byPriority": [{ "priority": "low", "count": 40 }]
  }
}
```

- 单值字段是折叠口径：`inProgress` = `investigating` + `in_progress`，`highPriority` = `high` + `critical`，`open` 只含 `open`。`identified` 历史上不属于任何单值桶，因此单值之和小于 `total` 是预期行为，不是丢数；需要与 `total` 对账时读分布。
- `byStatus`/`byPriority` 只含真实存在的取值，顺序按后端词表基准（状态 `open,investigating,identified,in_progress,resolved,closed`，优先级 `low,medium,high,critical`），词表外的历史值按字母序追加；两者各自求和恒等于 `total`，排除软删除记录并严格按租户过滤（`itsm-backend/handlers/problem/stats_contract_test.go` 锁死这几条）。
- 缺少租户上下文返回 HTTP 401 + 业务码 `2001`，不回落到默认租户、也不返回空成功。
- 报表要分布就读这两个字段。禁止改成前端拉一页列表自己数：2026-10-03 之前 `/reports/problem-efficiency` 正是取 `pageSize=100` 的第一页在浏览器里计数，第 101 条之后的问题被静默截断，界面却仍把它当全量读数展示。

### 问题 SLA 与评论（能力未就绪）

```http
GET /problems/{id}/sla
GET /problems/{id}/comments
Authorization: Bearer <accessToken>
```

两个端点恒定返回 `503` + 业务码 `5003`（unready）：问题域没有 SLA 截止时间字段，也没有任何写入 `sla_states`（`aggregate_type=problem`）的代码路径，问题评论同样没有独立存储。此前 `/sla` 返回 `200` + `{slaStatus:"none", responseTimeUsed:0, ...}` 的空成功，调用方无法区分「能力没接」和「该问题 SLA 正常且未超时」，现改为显式不可用；前端 `ProblemSLACard` 随之删除（它本就不可从任何页面到达），`Problem` 类型里后端从不返回的 `slaStatus`/`responseDeadline`/`resolutionDeadline` 字段一并移除。

问题不存在或跨租户访问仍是 `404`/`4004`（先按租户校验资源存在，再判能力），非数字 ID 是 `400`/`1001`。回归见 `itsm-backend/router/problem_sla_route_test.go`；接入真实问题 SLA 后需重开 `200` 契约并同步本节。

## 变更管理接口

### 获取变更列表

```http
GET /changes
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- status: 状态过滤（`draft`/`pending`/`approved`/`rejected`/`scheduled`/`in_progress`/`completed`/`failed`/`rolled_back`/`cancelled`/`closed`；`全部` 与空值不过滤）
- search: 标题/描述关键词（`Contains`，两条 OR）
- riskLevel: 风险等级过滤（`low`/`medium`/`high`）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `ChangeResponse`（camelCase，含 `assigneeId`/`createdBy`/`riskLevel`）。空结果
序列化为 `[]`。

口径说明（2026-10-03 E4-10 实测收口）：

- **`risk_level` 不再是第二套参数名。** 历史实现先读 `risk_level`、为空再读 `riskLevel`，属
  AGENTS「请求/响应多字段兼容零新增」禁止项；实测前端从未发送 `risk_level`，因此按只保留
  camelCase 收口。未识别的查询参数一律忽略，不会报错也不会生效。
- **`type`、`priority` 目前不被后端识别**（`handlers/change/repository_impl.go` 的 `List` 只
  构造 status/riskLevel/search 三个谓词）。前端 `ChangeApi.getChanges` 与本地 mock 仍接受这两个
  参数，因此「按类型/优先级筛选」在真实接口上是静默无效的——已登记为债务，见
  `plans/edge-feature-stability-audit-2026-10-02.md` 的 E4-11，不得当成已实现能力宣传。
- **行级数据权限**：非管理角色（`super_admin`/`admin`/`manager`/`sysadmin` 之外）只返回
  `createdBy` 或 `assigneeId` 等于本人的变更；缺失 `user_id` 时 fail-closed 返回空集。
  租户收敛在上游按 `tenant_id` 生效。

回归：`itsm-backend/router/change_list_route_test.go`（真实 `SetupRoutes` + `RequirePermission("change","read")`：
五键集合、三页拼接不重不漏、`pageSize=0`/`page=-1`/`pageSize=5000` 夹紧、`riskLevel` 收敛、
`risk_level` 不再生效、`status` 过滤与空 `[]`、agent 行级范围、租户 B 收敛、未认证 401）与
`itsm-backend/handlers/change/handler_test.go`。

### 创建变更

```http
POST /changes
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "升级数据库版本",
  "description": "升级 PostgreSQL 到 14 版本",
  "type": "standard",
  "priority": "medium",
  "impactScope": "high",
  "riskLevel": "medium",
  "plannedStartDate": "2024-01-15T00:00:00Z",
  "plannedEndDate": "2024-01-15T02:00:00Z",
  "implementationPlan": "分两批滚动升级",
  "rollbackPlan": "恢复原版本快照",
  "affectedCis": ["pg-prod-01"],
  "relatedTickets": ["TKT-1024"]
}
```

字段名以 `dto.CreateChangeRequest` 为准（`riskLevel`、`plannedStartDate`、`plannedEndDate`、
`affectedCis`）；旧文档里的 `risk`/`plannedStartAt` 从未被后端绑定。

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
  "riskLevel": "low"
}
```

`dto.UpdateChangeRequest` 的可选标量字段全部使用指针、切片按 nil 判断，未传字段不会被覆盖（`affectedCis` 传
`[]` 表示清空关联）；`status` 不在更新请求里，状态推进只能走下面的状态流转端点（由领域 service 校验允许的迁移）。

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

### 获取变更 PIR 列表

```http
GET /changes/pirs
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- result: 整体结果过滤（`successful` / `failed`；`全部` 与空值不过滤）
```

真分页端点（`service/pir_service.go` 做 `Count` + `Offset/Limit`），`data` 为
`{items, total, page, pageSize, totalPages}`，`total` 是过滤后的全量条数而非当前页长度。
排序为 `review_date` 降序 + `ID` 升序：`review_date` 非唯一列，同一天录入的 PIR 需要 ID
兜底才有确定的页边界归属。列表按认证上下文租户收敛，元素为 `ChangePIRResponse`。
契约回归见 `itsm-backend/router/change_pir_route_test.go`。

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

### 发布审批

- `POST /api/v1/releases/{id}/approve`：可选请求体 `{"comment":"同意"}`。
- `POST /api/v1/releases/{id}/reject`：请求体 `{"reason":"拒绝原因"}`，`reason` 必填。

两者均要求 `release:approve` 权限，审批人和租户来自认证上下文。审批人不属于当前租户、已停用或不存在时返回 HTTP 403，响应为 `{"code":2003,"message":"审批人不存在或已停用"}`，不写入发布、流程任务或审批审计。请求其他租户的发布资源仍返回 HTTP 404（业务码 4004），不泄露资源存在性。

### 删除发布

```http
DELETE /releases/{id}
Authorization: Bearer <accessToken>
```

## 服务目录接口

路由以 `router/service_catalog_routes.go` 注册的 4 个列表 URL 为准，全部要求
`service_catalog:read` 权限，租户只取认证上下文。

### 获取服务目录列表

```http
GET /service-catalog
GET /service-catalogs
GET /service-catalog-services
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- category: 分类过滤（精确匹配）
- status: 状态过滤，只接受 enabled | disabled
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `dto.ServiceCatalogResponse`（camelCase）。空结果序列化为 `[]`。
排序固定为 `created_at DESC, id ASC`。三个 URL 是同一个 handler，契约完全一致。

> 破坏性变更（2026-10-03 E4-6f）：集合键 `catalogs`→`items`、页长键 `size`→`pageSize`、
> 新增 `totalPages`，请求侧不再接受 `size`。见 UPGRADE.md §1.15。

### 搜索服务目录

```http
GET /service-catalogs/search
Authorization: Bearer <accessToken>

Query Parameters:
- q: 关键词，对 name / description 做包含匹配
- category: 可选分类过滤
- page / pageSize: 与列表端点同一套分页规则（2026-10-03 起真正生效，此前固定第一页 20 条）
```

`search` 只返回当前租户 `enabled`/`active` 的服务目录，响应信封与列表端点一致。

### 获取服务项详情

```http
GET /service-catalogs/{id}
GET /service-catalog-services/{id}
Authorization: Bearer <accessToken>
```

### 服务目录统计

```http
GET /service-catalogs/stats
Authorization: Bearer <accessToken>
```

返回 `dto` 口径的服务数量与分类计数；请求量、趋势、热门服务没有服务端实现，接口不会补零
伪装成「0 个请求」。

## 服务请求接口

### 获取服务请求列表

```http
GET /service-requests
GET /service-requests/me
Authorization: Bearer <accessToken>
需要权限: service_request:read

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- status: 状态过滤（精确匹配单个值）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `ServiceRequestResponse`（camelCase，含 `requestNumber`/`status`/`processorId`/
`createdAt`/`updatedAt`）。空结果序列化为 `[]`。排序固定为 `created_at DESC, id ASC`。

口径说明（2026-10-03 E4-4 实测收口）：

- **两个 URL 走同一个 handler。** `/me` 与显式 `scope=me` 把范围收窄到「本人创建或本人处理」，
  身份只取认证上下文；`userId` 查询参数不参与绑定，客户端无法自报身份放大或收窄可见范围。
- **`status` 有一层遗留别名归一。** `pending_approval`/`pending`→`submitted`、
  `approved`→`security_approved`、`in_progress`→`provisioning`、`completed`→`delivered`，
  其余值原样精确匹配。规范值集合是 `submitted`/`manager_approved`/`it_approved`/
  `security_approved`/`provisioning`/`delivered`/`failed`/`rejected`/`cancelled`。
- **行级数据权限**：`super_admin`/`admin`/`manager`/`sysadmin` 可见全租户，其余角色按
  DataScope（本人创建或本人处理）收敛。
- **分页夹紧只有一个所有者。** 此前请求 DTO 的 `binding:"min=1,max=100"`、handler 私有默认 10、
  repository 的 `>100→100` 三处同时生效，越界值先在绑定阶段返回 **HTTP 400 / code 1001**，
  与 handler 的「补默认值」是两种结局。现在 HTTP 入口单点夹紧，响应元数据由
  `common.SuccessWithPagination` 用同一组值算出；repository 只保留一道
  `common.ValidatePagination` 服务非 HTTP 调用方。破坏面见 [UPGRADE.md](../UPGRADE.md) §1.13。

### 获取待审批服务请求

```http
GET /service-requests/approvals/pending
Authorization: Bearer <accessToken>
需要权限: service_request:read

Query Parameters:
- page / pageSize: 同上（缺省 1 / 20）
```

收件箱**不接受任何查询过滤条件**：待办范围由审批记录的 pending 状态与认证上下文里的审批人、
角色决定，响应同样是五键分页信封，元素为请求单本身（不含审批步骤）。待办明细请另取
`GET /service-requests/{id}/approvals`。

### 创建/审批服务请求

`POST /service-requests`（`service_request:write`）创建请求单；审批动作为
`POST /service-requests/{id}/approvals`（`service_request:approve`），body 为
`{action, comment}`。⚠️ 实测 `/:id/approval` 与 `/:id/approvals` 两条路径注册到同一个
handler，而前端两个客户端各用其中一条（台账 E4-19）；新代码统一使用复数形式 `/:id/approvals`。

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
GET /api/v1/sla/stats
Authorization: Bearer <accessToken>
```

### 获取 SLA 违规记录

```http
GET /sla/violations
Authorization: Bearer <accessToken>
```

### 获取 SLA 合规报表

```http
GET /api/v1/sla/compliance-report?startDate=2026-01-01T00:00:00Z&endDate=2026-01-31T23:59:59Z
Authorization: Bearer <accessToken>
```

权限 `sla:read`。`startDate`/`endDate` 必填且必须是 RFC3339 完整时间戳：只传 `2026-01-31` 这类 date-only 值会被判为非法并返回 400 / code 1001，不会退化成「统计全部时间」。响应 `data` 为 `dto.SLAComplianceReport`：`totalTickets`、`metSla`、`violatedSla`、`complianceRate`、`avgResponseTime`、`avgResolutionTime`、`reportPeriod`。该端点与 `GET /api/v1/sla/stats` 口径不同（报表按窗口内的工单计率，stats 为全量口径），调用方不得在前者失败时静默改用后者数字冒充同一指标。

## 工作流接口

### BPMN 流程实例寻址键（`/api/v1/bpmn/process-instances/{id}` 家族）

```http
POST /api/v1/bpmn/process-instances                       # 启动，返回 BPMNProcessInstanceResponse（camelCase DTO）
GET  /api/v1/bpmn/process-instances?page=1&pageSize=20     # 列表，同 DTO
GET  /api/v1/bpmn/process-instances/{id}                  # 详情
GET  /api/v1/bpmn/process-instances/{id}/approval-history
PUT  /api/v1/bpmn/process-instances/{id}/variables
PUT  /api/v1/bpmn/process-instances/{id}/suspend
PUT  /api/v1/bpmn/process-instances/{id}/resume
PUT  /api/v1/bpmn/process-instances/{id}/terminate
```

`{id}` 全家族统一使用 BPMN 业务键 `processInstanceId`（`PI-*` 字符串，schema 全局唯一），即
响应中的 `instanceId` 字段；数字 Ent 主键只出现在响应 `id` 字段，**不再是任何端点的寻址键**，
传数字返回 404。查询与变更均收敛在认证租户范围内（跨租户 PI 同样 404）。详见
[UPGRADE.md](../UPGRADE.md) §1.28。

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

### BPMN 运行时监控

```http
GET /api/v1/bpmn/monitoring/metrics?timeRange=24h
GET /api/v1/bpmn/monitoring/metrics/{processKey}?timeRange=7d
GET /api/v1/bpmn/monitoring/instances/status?page=1&pageSize=20
GET /api/v1/bpmn/dashboard/audit-logs?page=1&pageSize=20
Authorization: Bearer <accessToken>
```

`monitoring/metrics`、`monitoring/metrics/{processKey}`、`monitoring/instances/status` 三个端点接受可选的 `startTime`/`endTime`，取值必须是 RFC3339；非法值返回 400 / code 1001（此前行为是静默忽略该过滤条件，响应看不出差异）。未传这两个参数时按 `timeRange`（默认 `24h`）。这些路由由 `handlers/bpmn/monitoring.go` 的 `RegisterRoutes` 挂在认证租户组下，实测**没有** `RequirePermission` 声明，只做认证与租户范围收敛。

三个列表端点（`instances/status`、`dashboard/audit-logs` 家族）的页长只有 `common.GetPaginationFromQuery` 一个所有者：缺省 `page=1`、`pageSize=20`，只采纳落在 `(0,100]` 的查询值，越界（`150`、`5000`）、非数字（`abc`）与 `0`/负数一律回落缺省，页码下界为 1；响应回显的就是真正进入 SQL `LIMIT` 的值，`data` 为平台五键 `{items,total,page,pageSize,totalPages}`。此前 `pageSize` 写错会让服务层的条件式分页分支整体不成立、`LIMIT` 子句消失从而把该租户的审计整表读出，越界值则按原值执行而信封另报 100，见 [UPGRADE.md](../UPGRADE.md) §1.26。

流程审计读取的唯一 HTTP 表面是 `GET /api/v1/bpmn/dashboard/audit-logs`（台账 E4-48 已收敛，见 [UPGRADE.md](../UPGRADE.md) §1.29）：

- `GET /api/v1/bpmn/monitoring/audit-logs` 已删除（重复表面），调用返回 404；
- 该家族（`audit-logs` 与 `audit-logs/user/{userId}`）挂 `RequirePermission("bpmn","read")`，无此权限返回 403 / code 2003，缺认证上下文 fail-closed 401 / code 2001；
- 响应项为 camelCase DTO `ProcessAuditLogResponse`（`processInstanceKey`、`activityId`、`userName`、`assigneeId` 等），**不再包含 `tenant_id`/`tenantId`**，租户只来自认证上下文；
- `dashboard/audit-logs` 的 `startTime`/`endTime` 沿用既有口径：按 `2006-01-02` 日期解析、非法值静默忽略（`/workflow/audit`、`/workflow/sla` 等页面按日期发送）；
- 时间线唯一表面为 `GET /api/v1/bpmn/monitoring/instances/{instanceId}/timeline`。

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

> `role` 取值是封闭词表，单一源在后端 `domain/role`：`super_admin` / `admin` / `sysadmin` /
> `security_admin` / `audit_admin` / `it_admin` / `manager` / `agent` / `technician` / `end_user`
> （`user` 是前端别名，服务端归一为 `end_user`）。传其他值返回参数错误。
> ITIL 实践角色（`change_manager` 等）与 MSP 协作角色（`msp_*`）不属于 `role` 字段，
> 通过 `roleIds` 对应的 `user_roles` 边叠加。2026-10-08 前的 legacy 值（`security`、
> `l1_support`、`it_director` 等）已退役，见 [UPGRADE §1.33](../UPGRADE.md)。

## 角色接口

### 获取角色列表

```http
GET /api/v1/roles
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码，缺省 1
- pageSize: 每页数量，缺省 20，只采纳 (0,100]，越界回落 20
- status: active | inactive，其他取值 400/1001（不静默回退）
- search: 按名称/编码模糊匹配
```

读取需 `role:read`，结果按当前租户收敛。响应 `data` 是平台五键信封 `{items,total,page,pageSize,totalPages}`
（`totalPages` 为 0 时省略），集合键是 `items` 而不是 `roles`：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "Agent",
        "code": "agent",
        "permissions": ["ticket:read", "ticket:write"],
        "status": "active",
        "createdAt": "2026-10-04T00:00:00Z",
        "updatedAt": "2026-10-04T00:00:00Z",
        "tenantId": 2,
        "dataScope": "owner"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20,
    "totalPages": 1
  }
}
```

`permissions` 是权限码字符串数组（`dto.RoleDTO` 由 `role.Permissions` 折叠出 `code`），`status` 只有
`active`/`inactive` 两个取值；不存在 `roles`、`page_size`、`isActive` 这些键，前端读取时不得做多字段兼容。
同一资源的创建/更新/删除/授权在 `/api/v1/roles`、`/api/v1/roles/{id}`、`/api/v1/roles/{id}/permissions`，
分别需要 `role:write` 与 `role:delete`。

## 资产管理接口

### 获取资产列表

```http
GET /api/v1/assets
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- status: 状态过滤（精确匹配）
- type: 类型过滤（精确匹配）
- category: 分类过滤（精确匹配）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `dto.AssetResponse`（camelCase）。空结果序列化为 `[]`。
排序固定为 `created_at DESC, id ASC`。`search` 参数后端不读取，文档此前把它列为可用过滤项属漂移。

> 破坏性变更（2026-10-04 E4-6g）：集合键 `assets`→`items`，新增 `page/pageSize/totalPages`；
> 非法或越界的 `pageSize` 原先由 handler 返回 400（且上限自称 200，与服务层的 100 相互矛盾），
> 现在统一由 `common.GetPaginationFromQuery` 回落默认页长。见 UPGRADE.md §1.16。

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
GET /api/v1/licenses
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- type: 类型过滤（精确匹配）
- status: 状态过滤（精确匹配）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `dto.LicenseResponse`（camelCase）。空结果序列化为 `[]`。
排序固定为 `created_at DESC, id ASC`。`search` 参数后端不读取。

> 破坏性变更（2026-10-04 E4-6g）：集合键 `licenses`→`items`，新增 `page/pageSize/totalPages`。
> 此前服务层只在 `page>0 && pageSize>0` 时才分页，因此不带页码（或把页码写成非数字，handler
> 会吞掉转换错误传 0）会把该租户的许可证全表读出来；现在缺省与非法值一律回落 1/20。

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

> 路径前缀为 `/api/v1`。历史兼容别名 `GET /api/v1/configuration-items` 仍然可用。

### 获取 CMDB 本体（AI-Native 自描述接口）

供 AI Agent / 集成方在运行时发现 CMDB 契约，无需硬编码字段与枚举。

```http
GET /api/v1/cmdb/ontology
Authorization: Bearer <accessToken>
```

返回 `data` 结构：

| 字段 | 说明 |
|:---|:---|
| `ciTypes[]` | 租户下的 CI 类型：`id` / `name` / `description` / `icon` / `color` / `parentTypeId` / `attributeSchema`（合法 JSON 时解析为对象）/ `attributeDefinitions[]` |
| `relationshipTypes[]` | 受控关系词表：`type` / `name` / `description` / `direction` / `reverse` / `icon` |
| `statuses[]` / `environments[]` | 生命周期与环境取值 |
| `tools[]` | CMDB 相关的 AI 工具定义（含 JSON-Schema 参数契约）；`list_cis` 的 `ci_type` 枚举由租户自有 CI 类型动态生成 |

单个 CI 类型的属性定义查询失败只降级该类型（记 warn 日志并置空数组），不影响整体返回。

### 获取 CI 关系类型词表

```http
GET /api/v1/cmdb/relationship-types
Authorization: Bearer <accessToken>
```

返回 13 种受控关系类型：`depends_on`、`hosts`、`hosted_on`、`connects_to`（双向）、`runs_on`、`contains`、`part_of`、`impacts`、`impacted_by`、`owns`、`owned_by`、`uses`、`used_by`。
创建/更新 CI 关系时传入该词表之外的值会返回 `400`。

### 获取配置项列表

```http
GET /api/v1/cmdb/cis
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- ciTypeId: CI 类型 ID（精确匹配）
- ciType: CI 类型字符串（精确匹配）
- status: 状态过滤（模糊匹配）
- environment: 环境过滤（精确匹配）
- ciNumber: 按 CI 业务编号精确过滤（如 CI-202609-000001）
- search: 按名称/资产标签/序列号模糊搜索
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数，
`items` 元素是 `dto.CIResponse`（camelCase，含 `ciTypeId`/`type`/`status`/`environment`/
`cloudResourceId`）。空结果序列化为 `[]`。排序为 `created_at DESC, id ASC`（未显式传 `sortBy`
时同样是这个固定顺序，`id` 兜底保证翻页不重不漏）。

CMDB 列表统一口径（2026-10-03 E4-6e 实测收口）：

- **七个列表端点共用一套五键信封**：`GET /cmdb/cis`、`/cmdb/ci-types`、`/cmdb/tags`、
  `/cmdb/cis/{id}/history`、`/cmdb/views`、`/cmdb/import`、`/cmdb/export`。它们实测都做
  `Count` + `Offset/Limit`，因此不再只回 `{items,total,page,size}`；`totalPages` 由
  `common.NewPaginationResponse` 单点算出。集合键一直是 `items`，本批没有改名。
- **分页只有一个所有者**：HTTP 入口统一 `common.GetPaginationFromQuery`。旧形态是三套并行——
  `dto.ListCIRequest` 的 `binding:"omitempty,min=1,max=200"`（越界返回 400 / code 1001，而
  `size=0` 因 `omitempty` 绕过校验，`Limit(0)` 在 Ent 里等于不加 LIMIT 而返回整表），其余六个
  handler 用裸 `strconv.Atoi(ctx.DefaultQuery("size"))` 完全不夹紧（`size=abc` 变 0 → 整表），
  `SearchCI` 里还有第三套 `>1000→20`。现在越界一律 200 / code 0 / 页长 20。
- **`size` 不再是 `pageSize` 的别名**，`offset`/`limit` 同样不参与分页决策。要一次取全请翻页
  （前端参考实现：`CMDBApi.getAllCIs(params, maxRecords)`、`CMDBApi.getCITypes()`）。
- **`GET /cmdb/views` 的 `include_public` 改为 `includePublic`**（camelCase 契约；实测旧名只存在于
  swagger，前端与测试零调用）。`includePublic=false` 只看本人视图。
- **兼容别名 `/api/v1/configuration-items*` 返回同一份信封**（同一 handler）：
  `GET /configuration-items`、`/configuration-items/types`、`/configuration-items/{id}/history`
  与 `/{id}/change-history`。别名已弃用，勿新增端点。
- `GET /cmdb/ontology` 的 `ciTypes[]` 现逐页读全该租户活跃类型；此前索要 `pageSize=500`
  把「要整表」藏在越界页长里，超过 500 个类型会静默截断。

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

### 云账号 / 云服务 / 云资源接口

`/api/v1/cmdb/cloud-*` 是这三个对象的**唯一**HTTP 表面（读取需 `cmdb:read`，写入需 `cmdb:write`，
删除需 `cmdb:delete`）。2026-10-04 实测发现 `/api/v1/cloud/accounts|services|resources` 是同一用例
的第二套已注册实现（15 条路由、独立 handler/service、零前端调用方），已按裁决**整体删除**，
详见 [UPGRADE.md](../UPGRADE.md) §1.22。

```http
GET /api/v1/cmdb/cloud-accounts
GET /api/v1/cmdb/cloud-services?provider=aliyun
GET /api/v1/cmdb/cloud-resources?provider=aliyun&serviceId=3&page=1&pageSize=20
Authorization: Bearer <accessToken>
```

`GET /api/v1/cmdb/cloud-resources` 查询参数只有 `provider`、`cloudAccountId`、`serviceId`、
`region`、`status`、`search`（按云资源唯一 ID 前缀匹配）加 `page`/`pageSize`。`provider` 走
云账号边（`HasAccountWith`），不是资源自身的冗余列；`search=no-such` 之类的空结果序列化为 `[]`。
页长与其他 CMDB 列表同源（`common.GetPaginationFromQuery`：缺省 1/20，越界回落 20 而不是夹到 100），
`serviceId=abc` 现在是 400 / code 1001，不再被 `strconv.Atoi` 静默当成 0。响应是
`{items, total, page, pageSize, totalPages}`，`total` 来自 `Count`，排序固定
`updated_at DESC, id ASC`（同一次发现任务批量写入的资源更新时间相同，缺 `id` 并列键会翻页重复或漏行）。

`GET /api/v1/cmdb/cloud-accounts` 与 `GET /api/v1/cmdb/cloud-services` 实测**不分页**：二者是 CI
表单与云资源页的选择器数据源，整份返回，因此响应只有 `{items, total}` 两键，不补假的
`page`/`pageSize`/`totalPages`。要按页读取请在 `cloud-resources` 上翻页。这两个端点当前是
无上限整表读取，还没有「选择器端点」的资源上限约定（台账 E4-20b 待裁）。

`provider` 的合法取值读写两侧是同一组六个规范值 `aliyun tencent huawei aws azure onprem`：
查询侧 `omitempty,oneof=…`（省略=不过滤），写入侧 `required,oneof=…`。别名
（alibaba/alicloud、qcloud/tencentcloud、huaweicloud、amazon、private）只在
`service/cloud.NormalizeProvider` 的适配器边界归一化，**不是**可过滤或可写入的取值；
拼错的厂商名现在是 400，不再静默落库、到发现阶段才表现为「没有适配器」。

云账号响应只外露 `hasCredential` 布尔值，`credentialRef`（含 access key 等凭据引用）从不进入
接口边界；创建/更新请求仍可写入该字段。云表面的租户一律只取认证上下文
（`handlerctx.ResolveTenantID`）：缺租户上下文即 fail closed，不再用 `c.GetInt("tenant_id")` 的 0
去查「租户 0」的数据。

**单个资源的租户与存在性语义**

```http
GET|PUT|DELETE /api/v1/cmdb/cloud-services/{id}
GET|PUT|DELETE /api/v1/cmdb/cloud-accounts/{id}
GET|PUT|DELETE /api/v1/cmdb/cloud-resources/{id}
Authorization: Bearer <accessToken>
```

- 「该行属于别的租户」与「该行真实不存在」返回**同一个** 404 / code `4004`，文案按资源固定为
  `cloud service not found`、`cloud account not found`、`cloud resource not found`；响应不回露对方
  租户的任何字段，因此不能靠状态码差异枚举对方的 ID。
- `DELETE` 以 `(tenant_id, id)` 为条件并检查受影响行数：命中 0 行返回 404。此前跨租户删除返回
  成功而对方的数据原封不动，属于假成功。
- `PUT` 的更新语句同样带租户谓词，跨租户写入 404 且对方的行保持原值。
- 云资源写入会校验 `cloudAccountId`/`serviceId` 指向的行属于本租户，引用他租户分别返回各自的
  404 文案；账号与服务厂商不一致返回 400 / code `4000`
  （`cloud account and cloud service belong to different providers`）。这与请求体绑定失败的
  400 / code `1001` 分属两层：前者是领域校验，后者是参数校验。

错误映射在 handler 边界只有一个所有者：云与发现的读取/写入端点统一走 `common.RespondError`，
状态码与业务码由 `common.classifyError` 决定。CMDB 域内私有的 `failCMDBError`（用
`switch appErr.Code` 复刻同一张映射表）已删除。

`GET /api/v1/cmdb/capabilities` 缺租户上下文时此前返回该域自定义的 401 / `2002`
「缺少租户认证上下文」，现与其他 CMDB 端点一致为 401 / `2001`「租户上下文缺失」。
`/api/v1/cmdb/cloud-*` 的 15 条路由已写入 Swagger；此前注解只存在于被删除的 `handlers/cloud`，
活表面在 Swagger 里零覆盖。

已知未收口（本片只登记，不改）：`GET /api/v1/cmdb/discovery/sources` 与
`GET /api/v1/cmdb/discovery/results` 返回裸数组、没有信封键，且后者的过滤参数绑定的是
snake_case `job_id`，与平台 camelCase 规则不一致。

契约锁：`router/cmdb_cloud_list_route_test.go`（信封键集、页长回落表、别名失效、跨租户收敛、
凭据不外泄、旧 `/api/v1/cloud/*` 三路径 404）与 `router/cmdb_cloud_authz_route_test.go`（均打在
真实 `SetupRoutes` + `RequirePermission` 上：缺租户上下文 401/`2001` 且响应不含任何列表载荷、
跨租户 GET/PUT/DELETE 404/`4004` 逐资源文案、DELETE 命中 0 行不再报成功、provider 枚举覆盖三个
列表与两个写入体、厂商矛盾 400/`4000`、合法写入后派生 identity 回填）。

## 通知接口

### 获取通知列表

```http
GET /notifications
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码（缺省或非法回落 1）
- pageSize: 每页数量（缺省 20；只有落在 1-100 的值被采纳，越界值回落 20 而不是夹到 100）
- type: 类型过滤
- read: 是否已读 (true/false)
```

`data` 为 `{items, total, page, pageSize, totalPages}`：本端点实测真的做 `Count` + `Offset/Limit`，
因此五键齐备，集合键只有 `items`（`notifications` 与 `size` 已删除）。`userId`/`tenantId` 不再从
查询参数绑定，只取认证上下文，自报身份无法读取他人通知。旧版在 handler 与 service 各写一份
夹紧规则，现在 HTTP 入口统一走 `common.GetPaginationFromQuery`。

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

## 统一工作台

跨域（事件 / 变更 / 问题 / 工单）待办列表，按统一阶段（phase）聚合。

```http
GET /api/v1/workbench?phase=active,submitted&priority=critical&recordType=incident&assigneeId=7&page=1&pageSize=20
Authorization: Bearer <accessToken>
```

- `phase` 取值 `draft|submitted|active|resolved|closed`，逗号分隔。阶段到各域 `status` 的映射由 `service/lifecycle` 单点提供并展开成 SQL 谓词，因此 `total`、分页与过滤同口径（不是在分页之后于内存里过滤）。
- `recordType` 取值 `incident|change|problem|ticket`；`priority`、`assigneeId` 为可选过滤，`assigneeId` 非法值当前被忽略。
- `page` 缺省 1，`pageSize` 缺省 20 且只采纳 `1..100`，越界回落默认值而不是报错。
- 租户上下文取自认证会话，缺失返回 2001 / HTTP 401；该路由实测没有 `RequirePermission` 声明。
- 响应 `data` 为平台列表信封 `{items,total,page,pageSize,totalPages}`，空结果序列化为 `[]`。

## 租户管理接口

### 创建租户

创建租户与产品基线安装命令在同一事务中提交：命令入队失败会连同租户创建一起回滚，
因此不会出现"已创建但没有任何角色/权限/菜单"的孤儿租户。基线由 `tenant.bootstrap.install`
命令异步安装，安装结果用下面的状态接口查询。

```http
POST /tenants
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "name": "客户名称",
  "code": "customer-code",
  "type": "saas_customer"
}
```

### 获取租户初始化状态

只读接口，返回真实安装状态：逐组件只读验证决定 `ready`，`commandStatus` 单独如实
呈现 outbox 命令状态（`none`/`pending`/`processing`/`succeeded`/`dead_letter`），
`recordedVersion` 是历史版本标记，仅作兼容读取。需要 `tenant:read` 权限。

```http
GET /tenants/{id}/initialization
Authorization: Bearer <accessToken>
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tenantId": 2,
    "templateVersion": "1.0.0",
    "recordedVersion": "1.0.0",
    "recordedAt": "2026-09-23T09:37:00Z",
    "commandStatus": "succeeded",
    "commandAttempts": 0,
    "ready": true,
    "components": [
      { "component": "identity-rbac", "verified": true },
      { "component": "workflow-core", "verified": true }
    ]
  }
}
```

安装失败进入 `dead_letter` 后，可由运维命令的重放入口（保留原幂等键）重放，
不新建第二次安装身份。

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

下表就是 `itsm-backend/common/response.go` 的 `codeStatusPairs` 登记表，两个方向都由这一张表读出。
此前本节列出的 `4001/4002/4003/5002` 实测在仓库里没有任何定义点与发射点（`grep` 全仓非测试代码零命中），
属于文档漂移，2026-10-04 按代码实测更正。

| 错误码 | 描述 | HTTP 状态码 |
|--------|------|------------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 数据验证失败 | 400 |
| 2001 | 认证失败（token 缺失/无效/过期，由认证中间件发出） | 401 |
| 2002 | 未授权访问（handler 侧缺租户或用户上下文） | 401 |
| 2003 | 禁止访问（权限不足） | 403 |
| 2004 | 工具权限不足（AI 工具 RBAC 拒绝） | 403 |
| 2005 | 未知工具 | 404 |
| 4000 | 请求错误 | 400 |
| 4004 | 资源不存在；跨租户探测与「不存在」返回同一个 4004，不确认对方资源是否存在 | 404 |
| 4090 | 冲突（非法状态迁移、唯一键冲突、乐观锁版本冲突） | 409 |
| 4220 | 请求格式合法但业务上无法处理 | 422 |
| 5001 | 服务器内部错误（仅用于 SQL/驱动/provider 等真实服务端故障） | 500 |
| 5003 | 服务暂不可用（connector/LLM/BPMN 等依赖未配置或未接线，不是用户参数错误） | 503 |

错误分类只有一个入口 `common.classifyError`：经 `errors.As` 识别 `AppError`、`BusinessError`
与 `VersionConflictError`，因此领域层用 `%w` 包装后的业务拒绝仍会映射成正确的 status/code；
无法识别的错误一律 500/`5001`，且原始错误串只进服务端日志，不出现在响应里。
`Fail` 对**未登记**的业务码仍回落 HTTP 200（既有兜底语义），新增业务码必须显式登记到这张表，
不能靠 default 分支蒙混。

版本冲突（`VersionConflictError`）的响应 `data` 携带冲突详情，供调用方做「重新拉取后重试」：

```json
{
  "code": 4090,
  "message": "工单已被他人修改，请刷新后重试",
  "data": { "resourceId": "123", "currentVersion": 3, "serverVersion": 5 }
}
```

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

分页事实只出现在这一层：列表集合固定用 `items`（领域名只出现在 item 类型里），不再同时返回
`tickets`/`changes`/`incidents`/`articles` 这类领域名别名，也不再嵌套重复的 `pagination` 对象。
消费方按 `data.items` 读取即可。

### 不分页的列表

后端确实不支持分页的列表（去重后的分类、按日期区间的全量聚合）**只返回 `{items, total}`**，
不得伪造 `page`/`pageSize`/`totalPages` 让调用方以为存在分页协议。静态门禁 5.10
（`scripts/static-gates/check-list-envelope.sh`，已在 backend-ci 执行）锁死这条边界：
带分页字段的响应体必须有 `items` 键。当前实例：

| 接口 | `data` 形状 | 说明 |
|------|------------|------|
| `GET /api/v1/known-errors/categories` | `{items: string[], total}` | 去重后的全量分类，无分页 |
| `GET /api/v1/known-errors/stats` | 聚合计数对象 | 不是列表，因此不带任何分页字段 |
| `GET /api/v1/tickets/templates` | `{items, total}` | 模板全量返回；历史实现伪造 `page=1`、`pageSize=len(items)` |
| `GET /api/v1/tickets/views` | `{items, total}` | 视图按租户全量返回（`service/ticket_view_service.go` 无 `Limit`），`total=len(items)` 诚实 |
| `GET /api/v1/tickets/{id}/comments` | `{items, total}` | 单工单评论全量返回，同上；分页若将来引入必须实装而非补键 |
| `GET /api/v1/tickets/assignment-rules` | `{items: AssignmentRuleResponse[], total}` | 按 `priority DESC, created_at DESC` 全量返回本租户规则；无任何查询参数，`total=len(items)` 诚实 |
| `GET /api/v1/tickets/automation-rules` | `{items: AutomationRuleResponse[], total}` | 同上排序；item 带 `createdBy`/`creator`/`tenantId`，读取需 `system_config:read` |
| `GET /api/v1/tickets/{id}/attachments` | `{items: TicketAttachmentResponse[], total}` | 单工单附件按 `created_at DESC` 全量返回；读取需 `ticket:read` 且调用者是工单相关方或管理角色，跨租户工单是 404/4004 |
| `GET /api/v1/tickets/cc/my` | `{items: TicketCCRecordResponse[], total}` | 抄送给当前用户的记录，按 `tenant_id + user_id + is_active` 全量返回，排序 `added_at DESC, id ASC`（`id` 是并列键）；`user`/`addedBy` 是 `WorkflowUserInfo`，姓名键是 `fullName`，没有 `name` |
| `GET /api/v1/tickets/{id}/cc` | `{items: TicketCCRecordResponse[], total}` | 单工单抄送列表，同上排序；读取需 `workflow:read` 且调用者是工单相关方/审批人/管理角色。跨租户探测与工单不存在返回同一个 404/`4004`（不确认对方资源存在），同租户但非相关方的用户是 403/`2003`；三种失败都不携带 `items` 载荷（2026-10-04 由 500/`5001` 纠正，见 UPGRADE §1.20） |
| `GET /api/v1/tickets/assign-recommendations/{id}` | `{items: AssignmentRecommendation[], total}` | 按评分降序全量返回本租户的可用候选（`service/ticket_assignment_smart_service.go:116`：`user.TenantIDEQ` + `ActiveEQ(true)` 整表取回后排序），`total=len(items)` 诚实；item 是 `{userId,username,name,email,score,reason,workload,skills?,categories?}`，**没有** `userName`/`userEmail`/`userAvatar`/`factors`。读取需 `system_config:read`；工单不存在与跨租户返回同一个 404/`4004`（同样是 E4-5 本片由 500/`5001` 纠正），不泄漏对方租户候选 |
| `GET /api/v1/tickets/{id}/notifications` | `{items, total}` | 单工单通知按 ticket+tenant 全量返回（`service/ticket_notification_service.go:831` 无 `Limit`），handler 里 `total=len(items)` 诚实 |
| `GET /api/v1/msp/reports/customers` | `{items, total}` | 区间聚合，字段为 camelCase DTO |
| `GET /api/v1/msp/reports/performance` | `{items, total}` | 同上 |
| `GET /api/v1/msp/allocations/history` | `{items, total, page, pageSize, totalPages}` | 标准分页信封，元素是 `MSPAllocationDTO` |

### MSP 报表的口径与限制

两个报表接口的响应行是 `MSPCustomerReportResponse` / `MSPPerformanceReportResponse`：
`{mspTenantId, dateFrom?, dateTo?, totalTickets, statusSummary}` 与
`{mspTenantId, dateFrom?, dateTo?, totalTickets, resolvedTickets, avgResolutionHours}`。

- `startDate`/`endDate`（camelCase，`YYYY-MM-DD`）为必填；snake_case 变体不再被读取，缺参是 400/1001。
- 聚合范围**只来自认证上下文的租户**（`mspTenantId` 即调用者租户），不接受 `customerTenantId` 过滤；
  `mspUserId`（按员工看绩效）从未实现，现在发送该参数会返回 400 并说明原因，而不是被静默忽略后
  返回一份租户级汇总。
- 报表**未按被服务的客户租户分组**，一次只产出一行汇总。这是已记录的功能缺口，
  见 `plans/edge-feature-stability-audit-2026-10-02.md` 行 2d；前端已删除后端从不返回的
  `customerName`/`slaComplianceRate` 等虚构列。
- `GET /api/v1/msp/customers/:customerTenantId/tickets` 的 `total` 是该客户租户（叠加 `status`
  过滤）的全量计数，不再用当前页长度冒充；元素统一过 `TicketResponse` DTO。

### MSP 分配历史

`GET /api/v1/msp/allocations/history`（`middleware.RequireMSPPermission("msp_allocation","read")`）
返回当前 MSP 租户的全部分配记录，**包含已解除的行**，按 `assignedAt` 倒序分页。

- 查询参数：`mspUserId`、`customerTenantId`、`startDate`、`endDate`（`YYYY-MM-DD`）、`page`、`pageSize`，
  全部可选；格式错误返回 400/1001。`endDate` 是日期，后端按「包含当天」处理（上界取次日 00:00 开区间），
  否则当天 00:00 之后的分配会被静默丢掉。
- 数据边界只能来自认证上下文：`msp_allocations` 没有 `tenant_id` 列，历史行通过 `msp_user` 所属的
  MSP 租户收敛，因此**不接受任何租户形式的查询参数**；缺少 `tenant_id` 或 `user_id` 一律 401/2002，
  不回退默认租户。
- 响应行就是 `MSPAllocationDTO`：`{id, mspUserId, mspUsername?, customerTenantId, customerTenantName?,
  role, assignedAt, deassignedAt?}`。表里没有解除原因和操作人列，所以契约里不存在
  `deallocationReason`/`createdBy`/`createdByName`；此前 `dto.MSPAllocationHistory` 声明过这些字段
  但没有任何生产代码能产出它们，已随本次接线删除。活跃行的 `deassignedAt` 直接缺省，不返回零值。
- `POST /api/v1/msp/allocations/deallocate` 只有在确实命中一条活跃分配时才返回成功；没有活跃行
  （不存在、已解除或不属于本 MSP 租户）返回 404/4004。历史上它忽略影响行数，把「什么都没解除」
  报成成功。
- 重新分配不再改写已归档行的 `deassigned_at`。此前 `Create` 会把同一「员工×客户」下所有已解除记录
  的结束时间刷成当前时间，导致这份历史的「何时解除」不可信。

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
