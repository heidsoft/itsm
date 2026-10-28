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

## AI 智能分诊

需要 `ai:read` 权限。

```http
POST /api/v1/ai/triage
{"title":"数据库连接超时","description":"应用无法连接 PostgreSQL"}
```

响应携带置信度与溯源字段：`suggestions.confidence`（0-1）之外，顶层还有 `promptVersion`（口径版本，当前 `triage-v1`，分词语义变更必须递增）、`model`（决策链路：`guidance`/`llm`/`keyword`）与 `auditId`（服务端决策底账行 ID）。服务端在产出建议时**自动**落一条 `item_type='ai_audit'` 底账（`accepted=NULL` 未决策，溯源字段由服务端写入）；前端采纳/拒绝直接 `PATCH /api/v1/ai/audit-logs/{auditId}` 翻转，不再需要客户端伪造 `model` 二次 POST 自报。底账写失败不阻塞建议返回，仅响应不携带 `auditId`。

## AI 审计记录

需要 `ai:read` 权限，租户范围只取认证上下文，查询参数中的同名字段不生效。

```http
GET /api/v1/ai/audit-logs?page=1&pageSize=20&days=90&kind=triage
```

分页规则由 `common.GetPaginationFromQuery` 单点决定：缺省 `page=1`、`pageSize=20`，只采纳 `1–100`，越界或非数字回落 20，`page<=0` 归一为 1。响应 `data` 是标准五键列表信封 `{items,total,page,pageSize,totalPages}`，`page`/`pageSize` 回显的是**实际采纳值**而不是客户端原值。`days` 是回溯窗口天数（缺省及非正值均为 90），`kind` 为空时不加该过滤。

`items` 为 `AIAuditEntry`：`id`、`createdAt`、`tenantId`、`userId`、`requestId`、`scenario`、`inputRef`、`promptVersion`、`model`、`confidence`、`accepted`、`suggestion`、`notes`。排序固定为 `created_at DESC, id DESC`；只统计 `item_type='ai_audit'` 的记录，空结果集序列化为 `[]`。缺少租户上下文返回 401 / `2001` 且响应体只有这一份 JSON。

`POST /ai/audit`（客户端自报审计）的 `confidence` 契约统一 0-1：负值、大于 1 返回 400 / `1002`（`0` 与 `1` 边界合法）。此前服务端静默 clamp，非法值进入 evaluator 置信度校准分桶会把桶污染成"低置信"。

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
| 4290 | 请求限流（HTTP 429；客户端应退避重试，勿与 2003 权限不足混淆） |
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

### CSRF Token

```http
GET /api/v1/csrf-token
```

写操作（POST/PUT/DELETE/PATCH）使用 HttpOnly 会话 cookie 时需要双提交 CSRF token：登录前后调用本端点，token 即以 `csrf_token` cookie（HttpOnly）下发并在响应体回显，客户端随后在每个写请求携带 `X-CSRF-Token` 头；每次成功写请求后 token 轮换，必须重新获取。Bearer token 认证（`Authorization: Bearer ...`）不受 CSRF 校验约束。

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "csrfToken": "hKu4P9c_R8uF4_x1LbAaXg"
  }
}
```

`data.csrfToken` 为 camelCase 契约字段（v1.6 起由 `csrf_token` 改名，旧字段已移除）；`csrf_token` 这一 snake_case 名称仅保留给 cookie 名与 form 字段名，属传输层标识。

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

字段约束与失败语义（服务端为准，前端同规则先行拦截）：

| 情况 | HTTP | 业务码 | message |
|:---|:---|:---|:---|
| 用户名不是 3–20 位字母数字、邮箱格式非法、密码少于 8 位 | `400` | `1001` | 参数校验信息 |
| 用户名已存在 | `409` | `4090` | `用户名已被注册` |
| 邮箱已存在 | `409` | `4090` | `邮箱已被注册` |
| 多个活跃租户 / 尚无活跃租户 | `503` | `5003` | `自助注册暂未开放…` / `系统尚未初始化活跃租户…` |
| 存储层故障 | `500` | `5001` | `注册失败` |

冲突与部署不可用此前统一返回 `401/2001`「认证失败」，会被前端显示成通用「注册失败」，
新用户无法区分「用户名被占用」和「系统没开通自助注册」；现按上表如实分级。

### 忘记密码

```http
POST /auth/forgot-password
Content-Type: application/json

{
  "email": "user@example.com",
  "tenantCode": "default"
}
```

邮件服务未配置时返回 `503`（业务码 `5003`，message `密码重置功能未启用：邮件服务未配置`），
不再返回 `401/2001`——后者会让用户以为是自己的邮箱有问题并反复重试。邮箱未注册时
仍返回成功文案（不泄漏账号存在性）。

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

### SSO 登录发起（OIDC）

```http
POST /api/v1/auth/sso/initiate
Content-Type: application/json

{ "provider": "default", "tenantCode": "acme" }
```

`provider` 缺省 `default`；`tenantCode` 可选，用于多租户下消歧同编码提供方。成功返回
`{redirectUrl, state, provider}`，`state` 由服务端单次消费（回调时校验）。

发起阶段的失败按语义分类，不再一律兜底成 `401`：

| 情形 | HTTP / 业务码 |
|---|---|
| 提供方未配置 `clientId`/回调地址，或 OIDC 端点解析（discovery）失败、状态存储不可用 | `503` / `5003`（运维不可用，非凭证错误） |
| 提供方不存在或未启用、租户编码不存在 | `404` / `4004` |
| 同编码提供方在多个租户下启用且未携带 `tenantCode` | `400` / `4000`（fail-closed，要求消歧） |

`503` 与 `401` 的区分让运维能从状态码判断「IdP 没配好/不可达」还是「凭证不对」。
回调（`/auth/sso/callback`）的认证失败仍稳定返回 `401`。

## 工单接口

### 获取工单列表

```http
GET /tickets
Authorization: Bearer <accessToken>

Query Parameters:
- page: 页码 (默认: 1)
- pageSize: 每页数量 (默认: 20)
- status: 状态过滤，`new|open|assigned|in_progress|pending|resolved|closed|cancelled|approved|rejected`
- priority: 优先级过滤，`low|medium|high|critical|urgent`
- type: 工单类型过滤，`incident|service_request|change|ticket|problem|improvement`
- categoryId: 分类过滤（按 ID）
- assigneeId: 处理人过滤
- requesterId: 报障人过滤
- parentTicketId: 父工单过滤
- templateId: 模板过滤
- keyword: 标题或描述关键词
- isOverdue: 只看超期（`true`）
- sortBy / sortOrder: 排序字段与方向（`asc` 需显式传，其余按倒序）；
  `sortBy` ∈ `created_at|updated_at|title|status|priority|ticket_number`，`sortOrder` ∈ `asc|desc`
```

状态、优先级、类型与排序参数均按上表校验，非法值返回 HTTP 400 / `code: 1001`；
2026-10-24 前这些参数完全不校验，`status=bogus` 会返回 200 + 空列表，调用方无法区分
「没有此类工单」与「参数写错」。

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "ticketNumber": "TK-2024-00001",
        "title": "无法登录系统",
        "description": "用户报告无法登录系统",
        "status": "open",
        "priority": "high",
        "type": "service_request",
        "categoryId": 1,
        "assigneeId": 2,
        "assigneeName": "张三",
        "requesterId": 3,
        "requesterName": "李四",
        "tenantId": 1,
        "version": 1,
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

`items` 元素是 `dto.TicketResponse`（camelCase），工单详情 `GET /tickets/{id}` 返回同一 DTO。

人名口径（2026-10-21 收口）：

- **姓名由服务端按当前租户投影**，字段是 `requesterName`/`assigneeName`，与事件/问题/变更同一风格。
  单条与列表各触发一次批量查询（一页工单只查一次用户表），只取 `id` 与 `name` 两列。
- **不再返回 `requester`/`assignee` 用户对象。** 该结构在 `dto.TicketResponse` 上声明过却从未被赋值，
  前端因此把已派工单渲染成「未分配」；除姓名外的用户字段（邮箱、电话、角色）本接口不返回，
  需要用户目录请走 `GET /api/v1/users`。
- **跨租户与已删除的用户 ID 不进映射**：`assigneeId` 照常返回，`assigneeName` 缺省为空，
  由调用方如实显示用户 ID，而不是折叠成「未分配」。
- 前端读取点统一走 `displayUserName(name, id, emptyLabel)`（`src/lib/formatters.ts`）：
  有姓名用姓名，只有 ID 显示 `用户 #<id>`，两者都没有才是「未分配」。

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

创建成功后系统会自动启动一个 BPMN 流程。请求不传 `workflowDefinitionKey` 时，流程按「工单类型 → 工单分类 → 流程绑定表 → 内置兜底」逐级解析，详见「工单分类接口 · 关联工作流：建单选路优先级」。

### 获取工单详情

```http
GET /tickets/{id}
Authorization: Bearer <accessToken>
```

返回 `data` 直接承载 `dto.TicketResponse`（与列表 item 同一结构，含 `requesterName`/`assigneeName` 投影）。
`{id}` 同时接受数字 ID 与业务工单号（如 `TKT-202609-000010`）；跨租户与不存在都返回 HTTP 404 / `code: 4004`，
不区分「无权限」与「不存在」。其余故障按统一错误分类返回（例如数据库故障为 HTTP 500 / `code: 5001`），
2026-10-24 前所有读取错误都被折叠成 404，运维无法区分「工单真的没了」与「服务出故障了」。

### 更新工单

```http
PUT|PATCH /tickets/{id}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "title": "更新后的标题",
  "description": "更新后的描述",
  "status": "in_progress",
  "priority": "medium",
  "type": "incident",
  "assigneeId": 3,
  "version": 7
}
```

`PUT` 与 `PATCH` 是同一个处理入口，均为部分更新语义：

- 可写字段：`title`（2–200）、`description`（≤5000；空串视为未提交，不支持清空）、
  `status`、`priority`（`low|medium|high|critical|urgent`）、`type`、`assigneeId`、
  `categoryId`、`formFields`、`resolution`。创建与更新共用同一套优先级/类型词表；
  2026-10-24 前更新额外要求描述 ≥10 字且不接受 `urgent`，导致前端回传整个编辑表单时
  保存失败。
- `tags` 不在本端点受理：按名称绑定标签请走 `POST /tickets/{id}/tags`，
  关联关系请走 `PATCH /tickets/{id}/relations`；提交非空 `tags` 返回 HTTP 400 / `code: 1001`，
  不再静默忽略。
- **只写入真正变化的字段**：与当前值相同的字段不会落库，因此回传当前 `status` 不会把
  已解决/已关闭工单的 `resolvedAt`/`closedAt` 重置为当前时间。
- 全部字段都与当前值相同且 `version` 与当前版本一致时，直接返回当前工单，不推进
  `version`、不刷新 `updatedAt`。`version` 与当前版本不一致时仍返回 HTTP 409 / `code: 4090`，
  空更新不能用来掩盖过期读取。

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

工单详情、列表与状态流转后的读回响应包含 `resolution`、`resolutionCategory`、`resolvedAt`、
`closedAt`、`firstResponseAt`（未发生时为 `null`/缺省）。解决说明与生命周期时间戳是 SLO 与
PIR 的证据，不能只写库不返回——`2026-10-22` 前 `buildTicketResponse` 漏映射这四个字段，
导致 `resolve` 成功后读不到解决说明。

### 更新子任务

```http
PUT|PATCH /tickets/{id}/subtasks/{subtaskId}
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "priority": "high",
  "assigneeId": 9,
  "version": 3
}
```

与工单更新同一套部分更新语义：可写 `title`、`description`、`status`、`priority`、`type`、
`assigneeId`、`categoryId`、`formFields`、`resolution`，只落库真正变化的字段，`version`
为乐观锁。`subtaskId` 必须属于路径中的父工单且同租户，否则 400/`1001`；子任务不存在或跨租户
为 404/`4004`。版本冲突返回 HTTP 409 / `code: 4090`，`data` 含 `ticketId`、`currentVersion`、
`serverVersion`；非法状态迁移同样是 409；`tags` 返回 400 并指向 `/tickets/{id}/relations`。
2026-10-24 前该端点只写少数字段，`priority`/`type`/`assigneeId`/`categoryId`/`resolution`/
`formFields` 传了也不落库，版本冲突还会返回 HTTP 500。

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
POST /tickets/workflow/accept
POST /tickets/workflow/reject
POST /tickets/workflow/withdraw
POST /tickets/workflow/forward
POST /tickets/workflow/cc
POST /tickets/workflow/approve
POST /tickets/workflow/resolve
POST /tickets/workflow/close
POST /tickets/workflow/reopen
```

工单 id 在请求体 `ticketId` 中，不在路径里。权限码按工单动作而非流程定义管理授予：
accept/forward/cc 需 `ticket:assign`，approve/reject 需 `ticket:approve`，resolve 需
`ticket:resolve`，close/reopen 需 `ticket:close`，withdraw 需 `ticket:update`。
`workflow:read`/`workflow:write` 的目录语义是流程定义管理（含发布流程模板），不再守这组路由。

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
无 `id` 次序键）。条目字段：`id`、`ticketId`、`fileName`、`fileUrl`、
`fileSize`、`fileType`、`mimeType`、`uploadedBy`、`uploader?`、`createdAt`
（服务端磁盘路径 `filePath` 已于 B7.2 从契约移除，见 UPGRADE §1.43）。
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

### 工单搜索

```http
GET /api/v1/tickets/search?q={keyword}&page=1&pageSize=20
```

按标题/描述关键字检索本租户工单，返回标准分页信封（`{items,total,page,pageSize,totalPages}`）。

- `q` 必填，空白返回 400/`1001`。
- 分页参数与 `GET /api/v1/tickets` 相同：`page` 从 1 开始，`pageSize` 缺省 20、上限 100。
- 本接口的 `item` 与工单列表一致（含 `requesterName`/`assigneeName` 投影）。
- 权限：`ticket:read`。

### 逾期工单

```http
GET /api/v1/tickets/overdue?page=1&pageSize=20
```

返回 SLA 已过期且工单仍未终结（非 `closed`/`cancelled`/`resolved`）的工单，同样使用标准分页信封。

- `total` 是全部逾期工单数，不是当前页条数；排序固定为 `created_at ASC, id ASC`，翻页稳定。
- 逾期判定以 `sla_states.resolution_deadline < now` 且 `status=active` 为准，与 `GET /api/v1/tickets/stats` 的 `overdue` 读数同源。
- 权限：`ticket:read`。

## 工单分类接口

分类是工单的主数据，也是建单选路的一个真实输入：`workflowDefinitionKey` 决定该分类下工单默认启动哪个 BPMN 流程。

| 方法 | 路径 | 权限 |
| --- | --- | --- |
| `GET` | `/api/v1/ticket-categories` | `ticket_category:read` |
| `POST` | `/api/v1/ticket-categories` | `ticket_category:create` |
| `GET` | `/api/v1/ticket-categories/tree` | `ticket_category:read` |
| `GET` | `/api/v1/ticket-categories/{id}` | `ticket_category:read` |
| `PUT` | `/api/v1/ticket-categories/{id}` | `ticket_category:update` |
| `PUT` | `/api/v1/ticket-categories/{id}/move` | `ticket_category:update` |
| `DELETE` | `/api/v1/ticket-categories/{id}` | `ticket_category:delete` |
| `POST` | `/api/v1/ticket-categories/import/preview`、`/import` | `ticket_category:create` |

### 列表查询

```http
GET /api/v1/ticket-categories?page=1&pageSize=20&parentId=3&isActive=true
```

- 过滤参数只有 camelCase 的 `parentId`、`level`、`isActive`，分页只有 `page`、`pageSize`；`tenantId` 只取认证上下文，查询参数不认。
- `parentId=0` 表示只要顶级分类。
- 响应为标准信封 `data: {items,total,page,pageSize,totalPages}`，`total` 是过滤后的全量条数。
- item 字段：`id`、`name`、`code`、`description`、`parentId`、`sortOrder`、`isActive`、`workflowDefinitionKey`、`departmentId`、`tenantId`、`createdAt`、`updatedAt`；`parentId`/`departmentId`/`workflowDefinitionKey` 为空时省略。

### 关联工作流：写入契约

`PUT` 用指针区分「未传」与「清空」：

| 请求 | 结果 |
| --- | --- |
| 不带 `workflowDefinitionKey` | 保持原绑定 |
| `"workflowDefinitionKey": ""` | 解除绑定（写 NULL） |
| `"workflowDefinitionKey": "net_flow"` | 绑定本租户已激活、最新版本的 `net_flow`；解析不到时 HTTP 400 + code 4000，文案「工作流不存在、未激活或不属于当前租户」，且原值不变 |

`POST` 同理：`workflowDefinitionKey` 缺省或空串都是不绑定。

### 关联工作流：建单选路优先级

工单创建时流程 Key 按以下顺序取值，第一个非空者生效，并被写进同事务入箱的 `workflow.start` 命令 payload：

1. 请求显式 `workflowDefinitionKey`；
2. 工单类型（`ticket_types.workflow_definition_key`）；
3. **工单分类**（`ticket_categories.workflow_definition_key`）——按 ID 或按 `category` 名称解析出的分类都算；
4. 流程绑定表 `process_bindings`（按工单类型 + `priority` 变量匹配）；走到内置 `ticket_general_flow` 且工单是 `high`/`urgent` 时，改判 `ticket_urgent_flow`；
5. 命令侧最终兜底 `ticket_general_flow`（解析器不可用或返回空时）。

分类绑定只影响**新建**工单：分类改绑不会重启或迁移已经在跑的流程实例。

消费侧对失效绑定是 fail-safe：payload 里的 Key 在本租户解析不到激活的流程定义时（分类被改坏、流程被停用或删除），命令不会反复重试，而是记 warn 后回落到第 4/5 级。

回归见 `itsm-backend/service/ticket_category_workflow_test.go`（分类绑定进入选路、三级优先级、按名解析、未绑定留空、写入校验与跨租户拒绝）、
`itsm-backend/service/workflow_start_command_handler_ticket_test.go`（绑定失效回落、路由表接管、加急兜底、跨租户流程名不泄漏）与
`itsm-backend/handlers/ticket_category/handler_test.go`（未知流程 Key 的 400/4000 与「未传不清空、空串才清空」）。

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

标签仍被本租户工单引用时删除返回 `409`/`code:4090`；跨租户、不存在或重复删除（重放）统一返回 `404`/`code:4004`，不再静默假装成功（见 UPGRADE §1.47）。

### 工单绑定标签

```http
POST /tickets/{id}/tags
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "tagIds": [1, 2]
}
```

也可以用 `{"tags": ["网络"]}` 按名称绑定，缺失的标签会在当前租户内自动创建。解绑使用 `DELETE /tickets/{id}/tags`，请求体相同。绑定是幂等的：重复提交相同 `tagIds` 仍返回成功，不会产生重复关联；引用本租户不存在的标签（或他租标签/工单）返回 `404`/`code:4004`。

## 工单模板接口

模板是租户级配置数据，读写用独立权限码 `ticket_template:{read,create,update,delete}`，不复用 `ticket:*`：
`agent`/`manager`/`it_admin` 持 `read`，`create`/`update` 给流程运营职责，`delete` 只给租户管理员与目录管理员。

```http
GET    /api/v1/tickets/templates                # ticket_template:read，全量列表（无分页参数）
GET    /api/v1/tickets/templates/categories     # ticket_template:read，本租户去重分类
GET    /api/v1/tickets/templates/{id}           # ticket_template:read
POST   /api/v1/tickets/templates                # ticket_template:create
PUT    /api/v1/tickets/templates/{id}           # ticket_template:update
PATCH  /api/v1/tickets/templates/{id}/status    # ticket_template:update，仅启停
POST   /api/v1/tickets/templates/{id}/copy      # ticket_template:create
DELETE /api/v1/tickets/templates/{id}           # ticket_template:delete
```

列表 `data` 是 `{items, total}`，**不接受** `page`/`pageSize`/`category` 参数（服务端不分页，
`total = len(items)`，非历史实现里被伪造的 `page:1`）。单对象响应的 `data` 直接是模板 DTO：

```json
{
  "id": 7,
  "name": "网络故障模板",
  "description": "用于网络类工单",
  "category": "IT Support",
  "priority": "high",
  "fields": [],
  "formFields": { "type": "incident", "sla": "2 hours", "slaType": "hours" },
  "workflowSteps": [{ "name": "分派", "order": 1 }],
  "isActive": true,
  "createdAt": "2026-10-01T00:00:00Z",
  "updatedAt": "2026-10-01T00:00:00Z"
}
```

- `formFields` 是模板表单的原样载荷（创建时 `fields[]` 会并入 `formFields.fields`）。消费方读这一个键，
  不存在 `content` 之类的别名。
- `isActive` 在请求里是**指针语义**：POST 不传按启用落库，传 `false` 就是停用；PUT 不传保持原状态，
  传值则与 `PATCH /{id}/status` 等价。历史实现把创建的 `isActive` 写死 `true` 并丢弃该字段，
  管理页的「模板状态」开关因此是假控件（见 UPGRADE §1.52）。
- `workflowSteps` 会被持久化并回显；不传即不改动存量步骤。
- `POST /{id}/copy` 由服务端整份复制源模板的 `description`/`category`/`priority`/`fields`/`formFields`/
  `workflowSteps`/`isActive`，请求体 `{"name": "..."}` 可省略，省略时命名为「`源名称 - 副本`」。
  复制是服务端动作，前端不应自行拼一份 formFields 再 POST 创建——卡片上没映射的字段会静默丢失。
- 模板 ID 与租户分别只取自路径段和认证上下文；跨租户读取返回 404/`4004`，同租户无权限是 403/`2003`。

回归见 `itsm-backend/middleware/rbac_ticket_template_guard_test.go`（路由权限码口径 + 各角色可达性）、
`itsm-backend/handlers/ticket/handler_template_test.go`（真实落库、租户隔离、不存在 404、停用与 workflowSteps
按请求提交、复制继承状态），前端 `src/lib/api/__tests__/ticket-api.test.ts` 与
`src/app/(main)/tickets/templates/__tests__/page.test.tsx`（403 渲染权限拒绝态而非空数据、写按钮按权限码收口）。

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

### 问题 SLA

```http
GET /problems/{id}/sla
Authorization: Bearer <accessToken>
```

`200` 返回 `dto.ProblemSLAResponse`：`{problemId, problemNumber, priority, bound, slaDefinitionId, status, responseDeadline, resolutionDeadline, firstResponseAt, resolvedAt, pausedDurationSeconds}`。

- **写入路径**（与工单/事件同一 SLA 引擎、同一 `sla_states` 多态表，`aggregate_type="problem"`）：问题创建时按 `service_type="problem"` + 优先级匹配 SLA 定义并计算响应/解决截止时间（`TicketSLAService.CalculateSLADeadlineFromRequest`）；问题进入 `resolved`/`closed` 时打解决时间戳（`Store.RecordResolution`）。SLA monitor 的周期扫描与违规创建覆盖 problem（`slaviolation.ticket_type="problem"`）。
- **`bound=false` 是「没配 SLA」的如实回答**：租户没有 problem 类型的 SLA 定义、或该优先级无匹配定义时，问题不绑定 SLA（与工单不同——工单无匹配时回落隐藏默认 60/480 分钟，问题域有意不套用，避免存量问题瞬间全部违规）。`bound=false` 时 `status="unbound"`；调用方必须把 `bound=false` 与 `status="active"`（已绑定且未超时）区分开。其余 status 取值：`active / warning / breached / paused / met`。
- 历史注记：本端点曾返回 `200` + `{slaStatus:"none", ...}` 空成功，后改显式 `503/5003` unready（调用方无法区分「能力没接」和「SLA 正常」），2026-10-21 随 SLA 引擎接入重开 `200` 真实契约。
- 问题不存在或跨租户访问 `404`/`4004`，非数字 ID `400`/`1001`。回归见 `itsm-backend/router/problem_sla_route_test.go`（503 时代契约）与 `itsm-backend/handlers/problem/` 契约测试（200 契约）。

### 问题评论

```http
GET /problems/{id}/comments?page=1&size=20
POST /problems/{id}/comments
Authorization: Bearer <accessToken>
```

- `GET` `200` 返回 `{items, total, page, size}`，`items` 为 `dto.ProblemCommentResponse`（`{id, problemId, userId, userName, content, isInternal, mentions, attachments, createdAt, updatedAt}`），时间正序；`userName` 为服务端投影的显示名（与工单姓名单一投影口径一致，缺失时前端回退 `用户 #id`）。
- `POST` 请求体 `{content(必填,1-5000), isInternal?, mentions?, attachments?}`，`200` 返回新建评论（含投影后的 `userName`）。`isInternal` 区分「处理纪要」（团队内部）与「可对报障人可见的进展说明」，与工单评论同一语义。
- **跨租户隔离口径**：评论表自带 `tenant_id`，但 `problem_id` 是跨租户可猜的外键——写入/读取前一律校验问题归属，别家租户的 `problem_id` 统一 `404`/`4004`，不暴露 ID 可枚举信号。
- 评论存储为独立表 `problem_comments`（迁移 `20261021_create_problem_comments.sql`），`problem_id` 外键 + 级联删除。历史注记：本端点曾恒定返回 `503/5003` unready（评论无独立存储），2026-10-21 转为真实契约。
- 无 update/delete 端点（与 incident 评论同一取舍：评论为过程记录，不做编辑审计面）。

### 问题知识沉淀

```http
POST /problems/{id}/knowledge
GET /problems/{id}/knowledge
Authorization: Bearer <accessToken>
Content-Type: application/json
```

把问题的现象/根因/临时措施/解决方案沉淀为知识草稿，并支持反查沉淀结果（2026-10-22 落地，补齐问题域与知识域的双向断链）。

- `POST` 请求体全字段可选：`{title?, category?, tags?}`。空值用问题单内容兜底——标题=问题标题、分类=`故障排查`、内容按 `问题现象 / 根因分析 / 临时措施 / 解决方案` 模板分段组装（空段跳过）；tags 自动追加 `问题沉淀` 与问题编号。
- **质量门**：问题既无根因也无解决方案时 `422`/`4220` 拒绝——没有可复用结论的空壳文章只会污染 RAG 语料。允许同一问题沉淀多篇（不同侧面，如「根因复盘」与「预防指南」）。
- **沉淀产物为 draft**：`is_published=false`，发布必须过知识评审流（submit→approve）——沉淀降低的是记录成本，不是审核成本。响应回显 `sourceProblemId`（来源追溯的权威字段）。
- `GET` `200` 返回 `{items, total}`，`items` 为知识文章 DTO（`status` 为 `draft`/`published`），**含草稿**——沉淀当事人需要立刻看到沉淀结果；创建时间倒序。空列表 `items: []`（非 null）。
- **权限语义**：入口挂在问题域，但动作语义是「创建知识」——写走 `knowledge:write`、反查走 `knowledge:read`，不因入口放宽。
- **存储**：`knowledge_articles.source_problem_id`（迁移 `20261022_knowledge_source_problem.sql`）。裸 ID 引用而非外键：沉淀后的知识是独立资产，问题删除不级联删知识；引用完整性由业务层校验。
- **跨租户隔离口径**：沉淀与反查都先校验问题归属，别家租户的 `problem_id` 统一 `404`/`4004`；知识仓储未装配时沉淀 `503`/`5003`（显式失败，不静默假成功）。回归见 `itsm-backend/router/problem_knowledge_route_test.go`。
- 关联读面：问题调查摘要 `GET /problem-investigation/problems/{id}/summary` 的 `KnowledgeArticles` 字段同源（同一 `source_problem_id`），不再是恒空数组。

### 服务请求 SLA

```http
GET /service-requests/{id}/sla
Authorization: Bearer <accessToken>
```

`200` 返回 `dto.ServiceRequestSLAResponse`：`{requestId, title, status, bound, slaDefinitionId, slaStatus, responseDeadline, resolutionDeadline, firstResponseAt, resolvedAt, pausedDurationSeconds}`。

- **绑定路径**（2026-10-21 起）：服务请求创建即按 `service_type="service_request"` 匹配 SLA 定义并写入 `sla_states`（与工单/事件/问题同一引擎同一多态表）。服务请求无 priority 字段，定义匹配不带优先级；无匹配定义即不绑定——`bound=false` + `slaStatus="unbound"` 是「没配 SLA」的如实回答，调用方不得与「SLA 正常」混淆。SLA monitor 周期扫描自动覆盖 service_request。
- **not-found 契约**（与该域 Get 一致）：跨租户/不存在返回 **HTTP 200 + 业务码 `404`**、无 `data`——不暴露资源存在性；非数字 ID `HTTP 400` + 业务码 `1001`；未认证 401。
- 事件域对照：事件 SLA 状态内联在 `GET /incidents/{id}` 详情响应（`slaState` 字段），无独立 `/sla` 端点；工单/问题有独立端点。四域统一走同一 `sla_states` 存储。

### 交付任务失败恢复

```http
POST /provisioning-tasks/{id}/retry
POST /provisioning-tasks/{id}/resolve
Authorization: Bearer <accessToken>
Content-Type: application/json
```

权限码均为 `service_request:write`——交付会真实创建云资源，读权限不携带这种副作用。

`retry` 请求体 `{reason?}`（可选，记录在命令 payload 的 `retryReason` 里，便于区分自动重试与人工重试）：

- 只对 `failed` 生效。`succeeded` → `409`/`4090`「已完成，不能重复执行」；`running` → `409`「正在执行中」；`pending` → `409`「当前状态不允许重试」（它已经在队列里，重试没有语义）。
- 尝试次数达上限（`maxAttempts`，当前 5）→ `409`「重试次数已达上限，请人工判定交付结果」，且不留下任何命令。
- **返回时 `status=pending`**：重试是「重新排队」，执行由队列 worker 领取，不是同步执行。
- 同一事务内把服务请求从 `failed` 拉回 `provisioning`（仅当仍是 `failed`，不覆盖人工已推进的状态）。
- 每次尝试使用独立幂等键 `provisioning_task:{id}:execute:{attempt}`。复用首次的键会被入箱层判重直接跳过——接口 `200` 但任务永远停在 `failed`。

`resolve` 请求体 `{outcome: "delivered"|"failed", note(必填)}`：

- 用于自动重试不可能成功的场景：provider 凭据吊销、配额封禁、地域下线，或资源已在平台外建好。
- `outcome=delivered` → 任务置 `succeeded`、服务请求置 `delivered`；`outcome=failed` → 任务置 `failed`、服务请求置 `failed` 且 `lastError` 记录说明。
- **`note` 必填**：没有依据说明的人工判定无法审计。缺 `note` 返回 `400`/`1001` 且任务状态零变化。
- 仅 `failed`/`pending` 可判定；`running` → `409`（执行中的任务即将回写自己的结果，人工判定会被覆盖），`succeeded` → `409`。
- 留痕字段：`resolvedByUserId` + `resolutionNote`，随任务响应返回。

任务响应（`GET /service-requests/{id}/provisioning-tasks`、`execute`/`retry`/`resolve`）新增：

| 字段 | 含义 |
|---|---|
| `attemptCount` | 已执行次数（含正在运行的那次）。重试本身不加，领取时 +1 |
| `maxAttempts` | 服务端下发的上限。`attemptCount >= maxAttempts` 时前端不应再显示「重试」 |
| `lastAttemptAt` | 最近一次开始执行的时间；`running` 任务超过 10 分钟未更新即视为无人持有，可被重新领取 |
| `resolvedByUserId` / `resolutionNote` | 人工接管的判定人与依据 |

执行权与状态守卫（`POST /provisioning-tasks/{id}/execute`）：已完成任务返回 `409`/`4090` 而非二次交付；并发执行只有一个能领到；worker 崩溃留下的 `running` 由 10 分钟租约回收自愈。存储侧新增 `attempt_count`/`last_attempt_at`/`resolved_by_user_id`/`resolution_note`（迁移 `20261021_provisioning_task_recovery.sql`）。

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
POST /changes/{id}/schedule
POST /changes/{id}/start
POST /changes/{id}/complete
POST /changes/{id}/fail
POST /changes/{id}/rollback
POST /changes/{id}/close
POST /changes/{id}/cancel
```

请求体（所有流转共用）：`{"comment": "...", "reason": "..."}`——两个字段等价，`comment`
为空时取 `reason`。审批流转的理由写进审批决定；`fail` / `rollback` / `complete` 的理由写进
**实施结果说明**（`outcomeNote`）。

#### 实施结果留痕（`/fail`、`/rollback`、`/complete`）

| 流转 | 权限码 | 原因 | 落点 |
|---|---|---|---|
| `POST /changes/{id}/fail` | `change:write` | **必填** | `outcomeNote` + 状态 `failed` |
| `POST /changes/{id}/rollback` | `change:rollback` | **必填** | `outcomeNote` + 状态 `rolled_back` |
| `POST /changes/{id}/complete` | `change:write` | 可选（完成说明） | `outcomeNote` + 状态 `completed` |

- 原因缺失 → `400`（业务码 `1002`），文案指出缺的是原因以及它会用在哪里。
- `outcome_note` 与状态在**同一条 UPDATE** 落库：不存在"状态已是 failed、原因是空"的窗口。
- `failed` 不是死胡同：`failed → scheduled`（重新排期）、`failed → rolled_back`（回滚）、
  `failed → cancelled` 都是合法迁移（`GET /changes/stats` 的 `failed` 计数由此才有真实来源；
  此前状态机允许 `failed` 但**没有任何 API 入口**写它，失败只能被报成 `completed` 或直接回滚）。

`GET /changes/{id}` 的响应新增 `outcomeNote` 字段（此前不存在，事后结论无处存放）。

#### 回滚清单与失败可回退门禁（`/rollback-steps`）

「失败可回退」此前只落在一次状态翻转上：`rollback_plan` 是一段自由文本，提交门禁只校验
非空；写了几步、走到第几步、每步成没成，系统一概不知。于是审计里最该被回答的三个问题
（回滚到哪了 / 哪一步没回成 / 谁确认的）全部无法回答，PIR 的「影响是否已复原」只能靠记忆填。

回滚清单把计划与执行事实收敛到同一行——`change_rollback_steps`，一行一步：

| 端点 | 权限码 | 作用 |
|---|---|---|
| `GET /changes/{id}/rollback-steps` | `change:read` | 按 `stepNo` 升序返回清单（含状态、备注、确认人、确认时间） |
| `PUT /changes/{id}/rollback-steps` | `change:write` | 覆盖式编排，请求体 `{"steps":["摘除流量","回滚镜像"]}`（顺序即序号，空数组=清空） |
| `POST /changes/{id}/rollback-steps/{stepId}` | `change:rollback` | 逐项确认，请求体 `{"status":"done\|skipped\|failed\|pending","note":"..."}` |

**状态语义**（`pending` / `done` / `skipped` / `failed`）：

- `skipped` 与 `failed` **必须带 `note`**（不适用依据 / 失败现象与残留影响）：
  没有理由的跳过在审计里与「漏做」无法区分，而漏做正是回滚最危险的形态。
- `failed` 也算「已确认」：回滚失败是真实结果，卡在 `pending` 只会逼人谎报完成。
- `status=pending` 表示**撤销误点**：清空确认人与时间，但保留备注作为复核线索。

**门禁与状态约束**：

| 规则 | 行为 |
|---|---|
| 清单存在且有 `pending` 步骤时 `POST /changes/{id}/rollback` | `422`（`ErrRollbackStepsUnresolved`），文案点出还差哪几步；状态不动 |
| 清单存在且全部已确认（含 `failed`） | 放行，`rolled_back` 与 `outcomeNote` 正常落库 |
| 清单为空 | 放行——应急回滚不能被「你还没编排清单」挡住；此类回滚不留逐项证据 |
| 清单读取失败 | 拒绝回滚（失败关闭）：读不到就无法证明清单已走完，而回滚不可逆 |
| 编排清单（`PUT`） | 仅 `draft`/`pending`/`approved`/`scheduled`；实施开始后 `409`（`ErrRollbackChecklistLocked`）。否则「清空清单 → 点回滚」就能绕过门禁 |
| 逐项确认（`POST`） | 仅 `in_progress`/`failed`；其余状态 `409` |
| 重复确认同一步 | `409`（`ErrRollbackStepAlreadyResolved`），文案说明已被谁于何时确认——不静默覆盖前人结论 |
| 跨租户 / 不存在的步骤 | `404`（统一口径，不暴露可枚举信号） |

`GET /changes/{id}` 的响应新增 `rollbackSteps` 数组；`resolved` 为派生字段
（`status !== "pending"`），客户端据此算进度，不必自己维护状态枚举。

> 与「回滚计划」的分工：`rollbackPlan`（计划页，可存 JSON 的策略/触发条件/责任人/验证方式）
> 描述**打算怎么做**；回滚清单描述**实际做了什么**。前端「编排回滚清单」会把计划里的
> `rollbackSteps[].title`（或纯文本的行）预填成清单，由人确认后落库——不做自动拆分，
> 因为清单会被当作执行依据逐项确认。

#### 审批与会签门禁（`POST /changes/{id}/approve`）

审批通过不是"这一票说了算"，而是按审批链（`change_approval_chains`）求值：

| 审批链形态 | 通过条件 |
|---|---|
| 单级或签（`serial` / `or`） | 任意一位审批人批准 |
| 会签（`parallel` / `all`） | 该层**全部**候选审批人批准 |
| N of M（`threshold` > 0） | 该层有效批准票数 ≥ 阈值 |
| 多级链 | **每一级**都满足各自阈值 |
| 没有层标记 `is_required` | 按全部层都是必需层处理（不存在"配了链却不用审"的放行口） |
| 未配置审批链 | 免审放行（标准变更预授权 / 紧急通道），由状态机与提交门禁决定 |

响应语义：

- **达阈值** → `200`，`data.status = "approved"`，状态推进与审批链收口同事务落库。
- **未达阈值** → `200`，本次审批决定**已写入审计表**，但 `data.status` 仍是 `pending`。
  这不是失败：一票已记录，等其余审批人。客户端应按"已记录、等待其他审批人"提示，
  不要重试（重复审批会被 `403`「不是待办审批人」拒绝）。
- **链上已有驳回票** → `409`（`ErrInvalidTransition`）。已否决的单子不能被"通过"翻回来。
- **不是该单待办审批人 / 已审批过** → `403`。
- **状态已被并发请求推进** → `409`。

终态流转（`reject` / `complete` / `cancel` / `rollback` / `close`）会在**同一事务**内把
残留的 `pending` 审批链节点置为 `obsolete`：收口失败则整体回滚，不会出现"变更已终态、
审批链仍挂着待办"的半写状态。

> **未绑定流程实例时的会签（`handled=false` 纯业务回退路径）**：
> 该路径的审批决定**没有落库位置**（`ProcessApprovalDecision` 的
> `process_instance_id` / `process_task_id` / `node_key` 都是流程事实，不能凭空造），
> 因此会签票无法累积。此时门禁只认"当前这一票"：
>
> - 单审批人链、或签、或这一票本身已达阈值 → 正常批准；
> - 会签 / N of M / 多级链未集齐 → **`409`（`ErrQuorumRequiresWorkflow`）**，
>   提示配置审批流程模板或改为单审批人链。
>
> 这是**失败关闭**（fail-closed）：不会静默退化成"一个人批掉整单"。需要会签/阈值的
> 租户必须为变更类型配置审批流程模板。

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

#### 创建 PIR 的状态门禁（`POST /changes/{id}/pir`）

PIR 是**实施后回顾**：只有真正实施过的变更才有可回顾的事实。允许创建的状态：

| 变更状态 | 是否可创建 PIR | 说明 |
|---|---|---|
| `completed` | ✅ | 正常路径 |
| `rolled_back` | ✅ | 回滚本身就是最需要复盘的结果 |
| `failed` | ✅ | 实施失败同理 |
| `draft` / `pending` / `approved` / `scheduled` / `in_progress` | ❌ | `409` +「尚未实施完成，不能做实施后回顾」 |

此前不校验状态，草稿和待审批的变更也能建 PIR，PIR 列表里混着还没开始的单，
「实施成功率 / 目标是否达成」这类指标直接失去意义。
同一变更只允许一条 PIR（重复创建 `409`，改用 `PUT /changes/pir/{id}`）。

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
- category: 分类过滤（精确匹配原始字符串，取值来自数据本身而不是枚举）
- status: 状态过滤，只接受 enabled | disabled
- q: 关键词，对 name / description 做包含匹配（与 `/search` 同一套口径），最长 100
- ciTypeId: 按关联 CI 类型过滤（正整数；0 或缺省表示不过滤）
- cloudServiceId: 按关联云服务过滤（正整数；0 或缺省表示不过滤）
```

响应是标准分页信封 `{items, total, page, pageSize, totalPages}`，`total` 为过滤后的全量条数；
`items` 元素是 `dto.ServiceCatalogResponse`（camelCase）。空结果序列化为 `[]`。
排序固定为 `created_at DESC, id ASC`。三个 URL 是同一个 handler，契约完全一致。

> 破坏性变更（2026-10-03 E4-6f）：集合键 `catalogs`→`items`、页长键 `size`→`pageSize`、
> 新增 `totalPages`，请求侧不再接受 `size`。见 UPGRADE.md §1.15。

> 服务端过滤（2026-10-16）：`q`/`ciTypeId`/`cloudServiceId` 之前只在 `/search` 或根本不存在，
> 列表端点因此把「对已加载的一页再做关键词过滤」当成搜索。现在筛选在 SQL 层完成，
> `total` 就是筛选后的全集条数，调用方不得再对 `items` 二次过滤后冒充全量。

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

### 创建服务目录

```http
POST /service-catalogs
Authorization: Bearer <accessToken>
需要权限: service_catalog:write

Request Body (dto.CreateServiceCatalogRequest):
- name: 必填，最长 255
- category: 必填，最长 100，自由文本（后端不做枚举校验，列表按原始字符串精确过滤）
- description: 最长 1000
- serviceType: 可选，oneof vm | rds | oss | network | storage | security | custom
- deliveryTime: 字符串承载的正整数「天数」，1-3650；缺省或空串按 1 天落库
- unit: 可选，oneof 月 | 次 | 用户
- price: 数字，缺省 0（免费/未定价）
- requiresApproval: 布尔指针，缺省为 true；必须显式传 false 才能创建免审批目录
- approvalLevel: 0-3，0 表示「未配置」
- slaResponseTime / slaResolutionTime / sortOrder / ciTypeId / cloudServiceId /
  formSchema / availableRegions / availableSpecs / status（enabled | disabled）
```

响应 `data` 是 `dto.ServiceCatalogResponse`。`requiresApproval`、`isActive`、`deliveryTime`
恒定返回；其余零值可选字段带 `omitempty`，缺席表示未配置，调用方不得用 `||` 兜底数字与布尔。

引用与取值校验（HTTP 400，业务码 1001，均为中文提示；同一规则只有一处文案，创建与更新一致）：

- `关联云服务时必须选择 CI 类型`
- `关联的 CI 类型不存在或未启用` / `关联的云服务不存在或未启用`（写入前按当前租户重新校验，
  跨租户 ID 一律拒绝）
- `deliveryTime 必须为正整数天数`（绑定层解析字符串）/ `交付天数必须在 1 到 3650 之间`（领域层范围校验）

### 更新服务目录

```http
PUT /service-catalogs/{id}
Authorization: Bearer <accessToken>
需要权限: service_catalog:write
```

请求体是 **PATCH 语义**（`dto.UpdateServiceCatalogRequest`，可写字段全部为指针）：

- 字段缺席 = 不修改，后端保持原值；
- 显式传 `""` / `0` / `false` = 真的清空或关闭（清除计价单位、清零价格、关闭审批都靠这条表达）；
- 因此前端表单必须把「用户清空输入框」提交成显式零值，省略字段会让页面显示与库里数据分叉。

`serviceType`、`unit`、`status` 的枚举/范围校验放在 service 层而不是绑定层：绑定层的 `oneof`
对非空指针仍会执行，会把「清空单位」这类合法 PATCH 拦成通用「请求参数错误」并丢失字段级提示。

关联字段的清空规则（2026-10-16 隔离栈实测修正）：

- `ciTypeId` / `cloudServiceId` 显式提交 `0` 表示解除关联，仓储层无条件写入实体值。
  此前仓储用 `> 0` 守卫，清空返回 HTTP 200 却把旧关联留在库里，页面显示「未关联」而新申请仍按旧 CI 类型履约。
- 云服务必须与 CI 类型成对存在：只把 `ciTypeId` 置 0 而保留 `cloudServiceId` 会被拒绝
  （HTTP 400 / code 1001「关联云服务时必须选择 CI 类型」），且一行不改；两者同时置 0 才合法。
- `approvers`、`availableRegions`、`availableSpecs` 目前是裸切片，只支持非空覆盖，
  **清空这三项尚无契约**，前端不得提交空数组来假装清除。

### 删除服务目录

```http
DELETE /service-catalogs/{id}
Authorization: Bearer <accessToken>
需要权限: service_catalog:delete
```

删除是软归档：目录转为 `disabled`，门户不再展示，历史申请记录保留。请求其他租户的目录返回
HTTP 404（业务码 4004），不泄露资源存在性。

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

文章响应同时携带**两条独立的状态轴**，缺一条就无法判断该文章能不能发布、卡在哪一步：

| 字段 | 来源 | 含义 |
|---|---|---|
| `status` | 由 `is_published` 派生 | `draft` / `published` —— 只回答「是否对外可见」 |
| `reviewStatus` | `review_status` | `draft` / `under_review` / `approved` / `rejected` —— 回答「是否过审」 |

`POST /knowledge/articles/{id}/publish` 要求 `review_status == approved`（见「发布与评审」），
历史数据中存在「已发布但未过评审」的文章，因此**不能用 `status` 反推评审态**，
也不能用评审态反推是否已发布。空值或未知值在响应里归一为 `draft`。
`2026-10-23` 前响应里没有 `reviewStatus`，API 消费者只能靠发起一次 publish 试错。

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

可选 `status` 字段（`draft` / `published`）与 `/publish` 是同一可见性投影，受同一发布门禁约束：
把文章改为 `published` 要求 `reviewStatus == approved`，否则返回 `409 / 4090`（与 `/publish` 一致），
不会留下「绕过评审直接发布」的第二条入口。改为 `draft`（撤回）不设门槛。
门禁读取服务端权威评审态，客户端自带的 `reviewStatus` 不能蒙混过关。

### 删除知识文章

```http
DELETE /knowledge/articles/{id}
Authorization: Bearer <accessToken>
```

### 发布与评审（发布门禁）

知识文章的对外可见性不是独立开关，而是评审结果的投影（`review_status → is_published`）。
发布端点对未通过评审的文章返回 `409 / 4090`，不再接受直接发布草稿——
否则作者可以绕过「不能自审」把未审内容放进 RAG。

```http
POST /knowledge/articles/{id}/review/submit      # 作者送审：draft|rejected → under_review
POST /knowledge/articles/{id}/review/approve     # 评审人通过：under_review → approved（自动发布 + 入索引）
POST /knowledge/articles/{id}/review/reject      # 评审人驳回：必须填写意见；撤回可见性并出索引
POST /knowledge/articles/{id}/review/withdraw    # 作者撤回：under_review → draft
POST /knowledge/articles/{id}/review/republish   # 已通过内容被修改后重新送审
GET  /knowledge/articles/{id}/reviews            # 评审流水（时间倒序）

POST /knowledge/articles/{id}/publish            # 仅 approved 可用；未过评审 409/4090
POST /knowledge/articles/{id}/unpublish          # 不设门槛：撤下内容永远是安全方向
```

| 当前 review_status | publish | 结果 |
|---|---|---|
| `draft` / `under_review` / `rejected` | `409` `4090` | 先 `review/submit`，由其他评审人 `review/approve` 后自动发布 |
| `approved`（未发布） | `200` | 恢复可见；已发布再调用为幂等，不重复入索引队列 |
| `approved`（已发布） | `200` | 幂等返回，不重复产生索引任务 |
| 任意（`unpublish`） | `200` | 撤回不要求评审状态 |

- **不能自审**：`approve` / `reject` 的操作人不能是文章作者（`403 / 2003`）。
- **驳回必填意见**：`review/reject` 空意见返回 `400 / 1002`。
- **可见性与索引同事务**：通过即入索引（RAG 可检索），驳回/撤回即出索引；
  草稿与评审中的文章不会出现在检索结果里。
- **向量索引能力未就绪时如实降级**：接了向量后端但 embedding 未配置时，发布/更新不会入
  `knowledge.vector.sync` 队列（避免注定失败的命令重试到死信）；更新已发布文章改为入 `delete`
  撤掉旧索引，防止检索返回过期内容。评审与发布本身不受影响，配置补齐后重新发布或修改即恢复索引。
  关键词检索后端不需要向量，不受该判定影响。

### 版本历史与恢复

```http
GET  /knowledge/articles/{id}/versions                       # 版本快照列表（版本号倒序）
GET  /knowledge/articles/{id}/versions/compare?from=&to=     # 两个版本对比
POST /knowledge/articles/{id}/versions/{version}/restore     # 恢复到指定版本
```

恢复是**内容变更**，不是单纯的可见性开关：

- 恢复前先把当前正文存为新版本快照，历史链不断；快照写入失败会**阻断恢复**（不静默吞错），
  避免覆写后旧正文无处可寻。
- 恢复后文章重置为 `reviewStatus = draft`、`isPublished = false`——旧正文未经重新评审不得继续对外可见，
  需由作者重新走 `review/submit → review/approve` 才能再次发布。
- 同一事务入向量 `delete` 任务清理旧索引，防止未经评审的旧内容继续被 RAG 检索命中。

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

权限 `sla:read`。响应 `data` 为 `dto.SLAStatsResponse`：`totalDefinitions`（租户内全部策略）、
`activeDefinitions`（`isActive=true`）、`totalViolations`、`openViolations`（`isResolved=false`）、
`overallComplianceRate`（0–100，保留一位小数）。计数走数据库 `Count`，不受列表分页上限影响；
2026-10-24 前统计值来自 `pageSize=1000` 的列表长度，超过 1000 条时恒等于 1000。
合规率优先按工单达标情况计算（有工单样本时），无工单样本时退化为违规已解决比例；
两者都无样本时为 `0`。任一计数查询失败返回错误（HTTP 500 / `code: 5001`），
不会把故障伪装成「零违规、零定义」。

### 获取 SLA 违规记录

```http
GET /sla/violations
Authorization: Bearer <accessToken>
```

### SLA 指标（未实现，显式 501）

```http
GET /api/v1/sla/metrics
Authorization: Bearer <accessToken>
```

权限 `sla:read`。该端点读取的 `sla_metrics` 表当前没有写入方（无指标采集任务），
因此不再返回空列表的 200 成功，而是返回 HTTP 501 / `code: 5010`：

```json
{
  "code": 5010,
  "message": "SLA 指标采集尚未实现：当前版本不写入 SLA 指标样本，请改用 GET /api/v1/sla/performance 或 POST /api/v1/sla/monitoring"
}
```

按需求改用：绩效聚合 `GET /api/v1/sla/performance`、实时快照 `POST /api/v1/sla/monitoring`、
全量汇总 `GET /api/v1/sla/stats`、窗口报表 `GET /api/v1/sla/compliance-report`。
`slaDefinitionId`、`metricType` 查询参数不再生效。

### 管理者统一服务质量报表

```http
GET /api/v1/reports/service-quality?days=30
Authorization: Bearer <accessToken>
```

权限 `dashboard:read`（报表资源统一收敛在 dashboard）。把散在各报表页与工作台的局部指标聚合成一次请求的管理视图：`kpi`（行级口径与工作台同源，`scope` 字段显式标注 `tenant`/`own`）、`sla`（合规率/风险/违约数）、`incidentDistribution`、`satisfaction`、`ticketTrend`、`ticketStats`（analytics 工单统计；analytics 未装配时该块显式缺省）。`days` 为统计窗口天数，默认 30、上限 90。

**口径版本契约**：响应携带 `metricVersion`（总体）与 `metricVersions`（各块独立）——任何指标的计算语义变更（窗口、过滤、分母、环比基线）都必须递增对应版本，下游管理层月报与对账脚本以版本号感知口径变化，禁止凭数字猜语义。空集合一律序列化为 `[]` 而非 `null`。导出通道 `POST /api/v1/analytics/export` 的审计由全局审计中间件承载（`export` 动作 + 请求体摘要入 `audit_logs`）。

### 获取 SLA 绩效聚合

```http
GET /api/v1/sla/performance?dimension=serviceType&startDate=2026-10-01&endDate=2026-10-24&page=1&pageSize=20
Authorization: Bearer <accessToken>
```

权限 `sla:read`。监控大屏「各服务类型绩效」「各优先级绩效」的数据源。`dimension` 只接受
`serviceType`（默认）/`priority`，非法值 400/1001；`serviceType`、`priority` 是可选的种群预过滤
（先限制统计范围再分页，不是对当前页做二次筛选）。`startDate`/`endDate` 可选，接受 RFC3339 或
裸日期 `YYYY-MM-DD`：裸日期按 UTC 解释，`startDate` 取当日 00:00:00、`endDate` 取当日
`23:59:59.999999999`（包含当天），与 `/changes/calendar`、MSP 报表的日期过滤同口径；非法值
400/1001，未传时窗口为最近 24 小时。响应 `data` 是标准分页信封附加 `dimension` 与 `truncated`：
`{items, total, page, pageSize, totalPages, dimension, truncated}`，行结构为
`SLAPerformanceRow`（`key`、`totalTickets`、`resolvedTickets`、`resolutionRate`、`metSlaTickets`、
`violatedTickets`、`complianceRate`、响应/解决的样本数与达标率、平均分钟数）。
`truncated=true` 表示窗口内工单数命中扫描上限、分母已被截断，指标只能按「至少这么多」解读。

### 获取 SLA 合规报表

```http
GET /api/v1/sla/compliance-report?startDate=2026-01-01T00:00:00Z&endDate=2026-01-31T23:59:59Z
Authorization: Bearer <accessToken>
```

权限 `sla:read`。`startDate`/`endDate` 必填，接受 RFC3339 或裸日期 `YYYY-MM-DD`：裸日期按 UTC
解释，`startDate` 取当日 00:00:00、`endDate` 取当日 `23:59:59.999999999`（包含当天）；两种写法
都不会退化成「统计全部时间」，非法日期返回 400 / code 1001。响应 `data` 为 `dto.SLAComplianceReport`：`totalTickets`、`metSla`、`violatedSla`、`complianceRate`、`avgResponseTime`、`avgResolutionTime`、`reportPeriod`。该端点与 `GET /api/v1/sla/stats` 口径不同（报表按窗口内的工单计率，stats 为全量口径），调用方不得在前者失败时静默改用后者数字冒充同一指标。

### SLA 定义（`/api/v1/sla/definitions` 家族）

```http
GET    /api/v1/sla/definitions?page=1&pageSize=100   # 列表，权限 sla:read
POST   /api/v1/sla/definitions                       # 创建，权限 sla:write
GET    /api/v1/sla/definitions/{id}                  # 详情，权限 sla:read
PUT    /api/v1/sla/definitions/{id}                  # 更新，权限 sla:write
DELETE /api/v1/sla/definitions/{id}                  # 删除，权限 sla:delete
```

`GET /sla` 与 `GET /sla/policies`（含 `/policies/:id` 全家族）是指向本家族的旧路径别名，返回同一份 DTO。

列表响应 `data` 是标准信封 `{items,total,page,pageSize,totalPages,hasNext,hasPrev}`；单对象响应 `data` 直接承载 `dto.SLADefinitionResponse`（见下）。列表只解析 `page`/`pageSize`，**不支持** `name`、`isActive` 等过滤参数，传了会被静默忽略；`pageSize` 上限 100（`common.MaxPageSize`），超限按 100 截断，调用方需用 `total` 与自己拿到的 `items.length` 比对，不能假定一次请求取回全量。

`dto.SLADefinitionResponse`：

```json
{
  "id": 1,
  "name": "Incident-P0-紧急",
  "description": "生产环境全面不可用",
  "serviceType": "incident",
  "priority": "urgent",
  "responseTime": 30,
  "resolutionTime": 240,
  "businessHours": {
    "timeZone": "Asia/Shanghai",
    "alwaysOn": true
  },
  "escalationLevels": [
    { "level": 1, "thresholdMinutes": 15, "notifyRoles": ["manager"] }
  ],
  "conditions": { "priority": ["urgent"] },
  "isActive": true,
  "tenantId": 1,
  "createdAt": "2026-10-15T10:00:00Z",
  "updatedAt": "2026-10-15T10:00:00Z"
}
```

- `responseTime`/`resolutionTime` 单位是**分钟**，创建时后端要求 `>= 1`；前端不得再做小时/分钟换算。
- `businessHours` 与 `escalationLevels` 是把存储层 JSON（`business_hours`、`escalation_rules.levels[]`，键为 `time_zone`/`work_days`/`is_24_7`/`threshold_minutes`/`notify_roles` 等 snake_case）投影后的 camelCase 视图。原始 map **不再出现在响应里**：`escalationRules` 字段已从 `SLADefinitionResponse` 移除，改用 `escalationLevels[]`。
- `businessHours` 缺省（`omitempty`）表示存储配置为空，前端应显示「未配置」；`alwaysOn: true` 表示 7×24，两者语义不同，不得合并成一个 hardcoded 展示值。
- `escalationLevels` 只保留 `levels[]` 中类型正确的层级，缺失或类型不符的条目被跳过，因此数组长度可能小于原始配置条数。
- `conditions` 仍是原始 map（当前没有稳定 schema），管理页不编辑。
- `priority` 词表：种子数据用 `urgent|high|medium|low`，模板用 `critical|high|medium|low`，后端优先级匹配前统一小写并把 `urgent` 归一到 `critical`。展示层需要同时接受两者。
- 创建请求体的 `isActive` 是非指针 `bool`：不传即按 `false` 落成停用状态。管理页创建时必须显式提交 `isActive`。

回归见 `itsm-backend/dto/sla_projection_test.go`（snake_case 键投影、空配置返回 nil）与
`itsm-frontend/src/app/(main)/admin/sla-definitions/__tests__/page.test.tsx`（分页信封、显式 `isActive`、优先级别名）。

### SLA 模板（`/api/v1/sla/templates` 家族）

```http
GET  /api/v1/sla/templates            # 目录 + 当前租户安装状态，权限 sla:read
GET  /api/v1/sla/templates/{key}      # 单模板详情，权限 sla:read
POST /api/v1/sla/templates/{key}/install  # 安装到当前租户，权限 sla:write
```

列表响应 `data` 是 `{items,total}`（目录在服务端固定，无分页参数）；单对象响应 `data` 直接承载
`dto.SLATemplateResponse`。

`dto.SLATemplateResponse`：

```json
{
  "key": "incident_p1_critical",
  "name": "事件 P1 紧急",
  "description": "核心业务系统不可用",
  "serviceType": "incident",
  "priority": "critical",
  "responseTime": 15,
  "resolutionTime": 240,
  "businessHours": { "timeZone": "Asia/Shanghai", "alwaysOn": true },
  "escalationLevels": [
    { "level": 1, "thresholdMinutes": 15, "notifyRoles": ["manager"] }
  ],
  "conditions": { "priority": ["critical"] },
  "industry": "incident",
  "recommended": true,
  "installed": true,
  "slaDefinitionId": 12,
  "installedAt": "2026-10-16T02:00:00Z"
}
```

- `businessHours` / `escalationLevels` 与 `SLADefinitionResponse` 共用 `dto.ProjectSLABusinessHours` /
  `ProjectSLAEscalationLevels`，因此模板预览和安装后的 SLA 定义是同一个形状；目录里的 snake_case
  存储键（`time_zone`/`is_24_7`/`threshold_minutes`/`notify_roles`）不会出现在响应里。
- `installed`/`slaDefinitionId`/`installedAt` 是**安装事实**，由服务端查 `sla_definitions` 得出，不是前端
  点过按钮的本地状态。识别依据是安装时写入的 `conditions.templateKey` 标记，并回落到旧约定
  `description` 前缀 `[Template] <key> - `；**不按名称前缀过滤**，管理员改掉显示名称后依然认得出是同一份安装。
- 安装是幂等的：已安装时返回 `wasAlreadyExist: true` 且不新建定义。幂等互斥只在单副本进程内
  （按 tenant+key 串行），多副本并发安装仍需 `sla_definitions` 上的 `(tenant_id, template_key)` 唯一约束才能彻底去重。
- `conditions` 是目录原始 map；安装时才追加 `templateKey`，目录响应里不含该标记。
- 模板的 `escalationLevels` 只是随定义一起写入的参考配置，**当前没有运行时消费者**；真正的升级动作在
  「升级规则」（`/admin/escalation-rules`）与「升级矩阵」（`/admin/escalation-matrices`）配置。
- 安装响应 `data` 是 `{templateKey,slaDefinitionId,created,wasAlreadyExist,message}`。
- 未知 `key` → HTTP 404 + 业务 code 4004；缺少租户上下文 → fail-closed，HTTP 401 + code 2001
  （不会返回「谁都没装过」的空列表）。

回归见 `itsm-backend/service/sla_template_install_test.go`（安装状态来自库、改名后仍幂等、并发只装一份、
租户隔离、租户自建定义不误判）与 `itsm-backend/handlers/sla_template/handler_test.go`（标准信封、投影形态、
HTTP 层幂等、404/401 语义），前端见 `src/app/(main)/admin/sla-templates/__tests__/page.test.tsx`。

## 升级矩阵接口

```http
GET /api/v1/escalation-matrices        # 需要 sla:read，返回本租户当前生效的矩阵
PUT /api/v1/escalation-matrices        # 需要 sla:write，整体覆盖
POST /api/v1/escalation-matrices/invalidate-cache  # 需要 sla:write，清读缓存
```

`data` 直接是「优先级 → 升级级别链」的 map，没有再包一层 `matrix`/`items`：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "critical": [
      {
        "level": 1,
        "afterMinutes": 15,
        "notifyRoles": ["manager"],
        "notifyUserIDs": [],
        "description": "15分钟未响应升级至部门经理"
      }
    ]
  }
}
```

- 优先级键与 `Ticket.Priority` 对齐，为 `critical` / `high` / `medium` / `low`；键顺序不保证，消费方按 key 取值。
- `afterMinutes` 是**距告警开始**累计的分钟数，不是距上一级升级的间隔。
- `notifyRoles` 取主角色词表原值（`manager`/`it_admin`/`admin`…），`notifyUserIDs` 是具体用户 ID。
  两者未配置时返回**空数组**而不是 `null`（2026-10-16 修正：原先 nil 切片直接序列化，前端每处取值都要再判空）。
  角色与用户都为空的级别不会发出通知。
- `notifyUserIDs` 的 JSON key 沿用 Go 缩写写法，未按接口 camelCase 规范改为 `notifyUserIds`；
  改名属破坏性契约变更，需要连同 `system_configs` 里已持久化的矩阵一起评估。
- 未配置自定义矩阵时返回内置默认矩阵，因此**空 map 不是「未配置」而是异常**，前端据此报错而不是当作空成功。
- 自定义矩阵持久化在 `system_configs`（key=`escalation.matrix`，category=`escalation`）；`PUT` 落库成功才更新
  读缓存，落库失败返回错误而不是假成功。
- HTTP 契约回归：`handlers/escalation_matrix/http_contract_test.go`（默认矩阵非 null、PUT/GET 覆盖往返、
  多租户隔离、invalidate-cache 后 store 路径不丢配置）。注意按 authz 编译期默认权限集 `end_user` 持有
  `sla:read`（可读矩阵与 SLA 展示面），但不持有 `sla:write`（PUT 一律 403/`2003`）。

## 变更评审接口

评审组成员名册（REVIEW / EREVIEW 双类别），驱动审批链引擎的 `review:` 解析器：

```http
GET    /api/v1/change-review/members?type=REVIEW   # 需要 change:read，type 仅 REVIEW/EREVIEW，非法值回落 REVIEW
POST   /api/v1/change-review/members               # 需要 change:write
PUT    /api/v1/change-review/members/{id}          # 需要 change:write（role / isActive）
DELETE /api/v1/change-review/members/{id}          # 需要 change:write
```

- `data` 直接是 `ChangeReviewMemberResponse` 数组（`{id,userId,userName,email,type,role,isActive,tenantId,createdAt}`），
  **没有**分页信封；`type` 过滤是精确匹配。
- `POST` 的 `userId` 必须是**本租户真实用户**：跨租户或不存在的用户 id → 400/`1001`（引用型资源缺失，
  与工单评论「提及用户无效」同口径）；同 user+type 已在名册 → 409/`4090`；两者都不再伪装 500（2026-10-18
  纠正，见 UPGRADE §1.45）。
- `PUT`/`DELETE` 对不存在或跨租户成员一律 404/`4004`（存在性与跨租户不可区分），不再伪装 500。
- 列表内部会按租户谓词读用户详情；历史脏数据（成员行指向他租用户）在列表中静默跳过。

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

列表查询参数（2026-10-18 起真正生效；此前请求 DTO 缺 `form` 标签，camelCase 参数被静默丢空，
过滤与分页形同不存在）：

```http
GET /api/v1/bpmn/process-instances?processDefinitionKey=&status=&businessKey=&page=1&pageSize=20
GET /api/v1/bpmn/process-definitions?key=&category=&isActive=&page=1&pageSize=20
GET /api/v1/bpmn/tasks?status=&processDefinitionKey=&page=1&pageSize=20
```

- `tenantId`、`userId`、`assignee`、`candidateUsers`、`candidateGroups` 不是可查询参数，一律来自
  认证上下文；`processDefinitionKey`/`status` 的取值见实例状态词表
  `running | suspended | completed | terminated`。
- 生命周期端点 `suspend`/`resume`/`terminate` 校验源状态：`suspend←running`、`resume←suspended`、
  `terminate←running|suspended`，非法迁移返回 **409/`4090`**（实例状态与审计均不变），实例不存在
  或跨租户返回 **404/`4004`**，不再对已完成的实例返回 200。
- `DELETE /api/v1/bpmn/process-definitions/{key}?version=` 同理：仍有 running/suspended 活跃实例
  → 409/`4090`；只剩历史实例 → 409/`4090`（`process_instances.process_definition_id` 是必填外键，
  删除会破坏历史），无实例的版本才真正删除。
- 响应中的 `endTime`/`suspendedTime` 是可选字段：时间尚未发生（running 未结束、非挂起无挂起时刻）
  时整个键省略，不再输出 Go 零值 `0001-01-01T00:00:00Z`；已结束/已挂起时按 RFC3339 返回。
  调用方应以「键是否存在」判断时间是否发生，不能把零值时间当成真实时间。

  以上均属调用方可见的行为变更，升级注意事项见 [UPGRADE.md](../UPGRADE.md) §1.46。

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

响应 `data` 为统一分页信封 `{items,total,page,pageSize,totalPages}`，`items` 元素为 `UserDetailResponse`；旧的 `users` 与 `pagination` 键已移除（`pagination.totalPage` 单数从未是后端字段）。破坏性变更见 [UPGRADE §1.84](../UPGRADE.md)。

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

### 自助账号面（仅需登录，不需要 `user:write`）

上面 `PUT /api/v1/users/{id}` 是**管理用户目录**的语义（`user:write`），最终用户改自己的
姓名或电话走它会得到 403；把 `user:write` 放开给全员又等于交出整个用户目录的写权限。
因此登录用户对自己账号的维护走独立权限面，路由在 `itsm-backend/router/account_routes.go`
注册在 `r` 层 `/api/v1/account`（与 `/auth/me` 同组，不进租户级 RBAC 分组），只挂
`AuthMiddleware`、**不挂** `RequirePermission`。

| Method + Path | 权限 | 说明 |
|:---|:---|:---|
| `GET /api/v1/auth/me` | 仅需登录 | 本人资料读侧（`UserDetailResponse`） |
| `PUT /api/v1/account/profile` | 仅需登录 | 更新本人 `name`/`email`/`phone`/`department` |
| `PUT /api/v1/account/change-password` | 仅需登录 | 改本人密码，必须带旧密码，成功后吊销该用户全部存量 token |

```http
PUT /api/v1/account/profile
Content-Type: application/json

{
  "name": "张伟",
  "email": "zhangwei@example.com",
  "phone": "13900000000",
  "department": "财务部"
}
```

- **目标用户只来自鉴权上下文**：handler 从中间件写入的 `user_id` 取，请求体和路径参数里的
  `userId` 不参与寻址，不存在改他人资料的路径。
- **字段白名单是唯一的提权屏障**：`dto.UpdateOwnProfileRequest` 只声明上表四个字段，
  `role`/`username`/`active`/`tenantId`/`mspRole` 在 JSON 绑定阶段被丢弃，因此该端点不具备
  自我提权或改变租户归属的能力。回归：`service/user_selfprofile_test.go`
  `TestUpdateOwnProfile_DoesNotEscalateRole`、`handlers/account/handler_test.go`
  `TestUpdateProfileIgnoresEscalationFields`。
- **可选字段是指针**：`phone`/`department` 区分「未传」（保持原值）与「显式 `""`（清空）」；
  只提交姓名不会抹掉电话。`name`/`email` 必填，邮箱留空一律拒绝——邮箱是登录身份，静默跳过
  会把账号改成无法登录。
- **邮箱唯一性按管理面口径**：跨租户全局唯一，冲突返回 409/`4090`「该邮箱已被其他账号使用」，
  不得退化成未分类错误（否则前端只显示「更新失败」）。
- 响应 `data` 是更新后的 `UserDetailResponse`（含只读的 `role`/`active`），前端必须以响应回填
  表单，而不是把用户输入的原始值拼进旧状态（服务端会 trim）。

| 场景 | HTTP | 业务码 |
|:---|:---|:---|
| `name`/`email` 缺失、邮箱格式非法 | 400 | `1001` |
| 姓名或邮箱全为空白 | 400 | `4000` |
| 缺少鉴权上下文（无 `user_id`） | 401 | `2002` |
| 账号服务未装配 | 503 | `5003` |
| 邮箱已被其他账号占用 | 409 | `4090` |
| 目标行不属于调用者租户 | 404 | `4004` |


## 用户组接口

```http
GET    /api/v1/groups?page&pageSize&search        # 组列表，权限 group:read
GET    /api/v1/groups/{id}                        # 组详情，权限 group:read
POST   /api/v1/groups                             # 新建组，权限 group:write
PUT    /api/v1/groups/{id}                        # 更新组，权限 group:write
DELETE /api/v1/groups/{id}                        # 删除组，权限 group:write
GET    /api/v1/groups/{id}/members?page&pageSize  # 组成员分页，权限 group:read
POST   /api/v1/groups/{id}/members                # 添加成员，权限 group:write，body {userId}
DELETE /api/v1/groups/{id}/members                # 移除成员，权限 group:write，body {userId}
```

`GET /groups` 与 `GET /groups/{id}/members` 的 `data` 均为 `{items,total,page,pageSize,totalPages}`：列表元素分别是 `GroupResponse`（`id`/`name`/`description`/`tenantId`/`createdAt`/`updatedAt`）与 `UserDTO`。不存在 `groups`、`members`、`pagination` 键——旧形状已在 [UPGRADE §1.84](../UPGRADE.md) 中退役。

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

### 权限字典

```http
GET /api/v1/permissions
Authorization: Bearer <accessToken>
# 权限 permission:read
```

`data` 为不分页的完整集合 `{items,total}`，`items[]` 为 `PermissionResponse`（`id`/`code`/`name`/`description`/`resource`/`action`/`tenantId`/`createdAt`），授权时传 `code`。此前同时返回的 `permissions` 纯码值数组已移除；该端点没有 `page`/`pageSize`/`totalPages`，不要按分页信封校验必填（[UPGRADE §1.84](../UPGRADE.md)）。

## 团队管理接口

```http
GET    /api/v1/org/teams?page&pageSize&search   # 团队列表（含成员投影），权限 team:read
GET    /api/v1/org/teams/{id}                   # 团队详情（含成员投影），权限 team:read
POST   /api/v1/org/teams                        # 新建团队，权限 team:write
PUT    /api/v1/org/teams/{id}                   # 更新团队，权限 team:write
DELETE /api/v1/org/teams/{id}                   # 删除团队（软删除），权限 team:write
GET    /api/v1/org/teams/{id}/members?page&pageSize  # 成员分页，权限 team:read
POST   /api/v1/org/teams/{id}/members           # 添加成员，权限 team:write
DELETE /api/v1/org/teams/{id}/members           # 移除成员，权限 team:write，body 带 userId
```

`GET /api/v1/teams` 是同一 handler 的旧路径别名，响应形状与 `/api/v1/org/teams` 完全一致。

### 列表与投影

列表响应 `data` 使用统一信封 `{items,total,page,pageSize,totalPages}`（破坏性变更见 UPGRADE.md §1.36，
此前是裸数组）。`total` 是**过滤后**的全量条数；`search` 在服务端对 `name`/`code`/`description` 做
不区分大小写的包含匹配，前端不得拿当前页数据二次过滤后冒充全量结果。`pageSize` 超过 100 由
`common.GetPaginationFromQuery` 回落到上限。

`Team` 投影（`handlers/common.Team`）：

```json
{
  "id": 1,
  "name": "运维一组",
  "code": "OPS-A",
  "description": "负责生产运维",
  "status": "active",
  "managerId": 10,
  "managerName": "爱丽丝",
  "tenantId": 7,
  "members": [
    { "id": 10, "username": "alice", "name": "爱丽丝", "email": "alice@example.com" }
  ],
  "createdAt": "2026-10-16T02:00:00Z",
  "updatedAt": "2026-10-16T02:00:00Z"
}
```

- `members` 由列表/详情查询一次性带出（Ent `WithUsers` 只取 `id/username/name/email` 四列），
  团队没有成员时是空数组 `[]`，不会缺省。响应里不再出现 Ent 的 `edges` 形态，也不会带出
  `password_hash` 等用户列。
- `managerName` 由服务端按 `managerId` 在当前租户内一次查回（优先 `name`，缺失回落 `username`），
  前端不得再拿用户目录比对 ID 推导姓名：目录被分页截断时负责人会显示成空。负责人未设置、
  或库里残留指向其他租户用户的 `manager_id` 时该字段是空串 `""`，不回带其他租户用户的姓名。
- 成员端点 `GET /teams/{id}/members` 的分页口径与 `Team.members` 一致：同一列清单、同一
  `username` 升序，只是额外提供 `total` 翻页。

### 更新语义（`PUT /api/v1/org/teams/{id}`）

请求体 `{name?, code?, description?, managerId?}`，可选字段按「未传 / 传零值」区分：

| 字段 | 未传 | 传值 |
|:---|:---|:---|
| `name` / `code` | 保持原值 | 非空则覆盖（空串等同于未传，两者都不允许把必填列清空） |
| `description` | 保持原值 | 显式 `""` **清空**描述 |
| `managerId` | 保持原负责人 | 显式 `0` **解除**负责人；正整数按当前租户校验后设置 |

前端编辑表单展示全部字段，因此按全量提交：清空负责人要显式发送 `managerId: 0`，省略键不会解除。
`managerId` 指向其他租户或停用用户返回 400 + code 4000，且原负责人不变。

### 成员关系语义

成员关系落在 `users.team_users` 外键上（Ent 的 O2M 边），**一个用户同一时刻只能归属一个团队**，
因此写操作不能靠 `AddUserIDs` 静默搬人：

| 场景 | 结果 |
|:---|:---|
| 添加已在该团队的成员 | HTTP 200，幂等，不产生第二份关系 |
| 添加已属于**其他团队**的用户 | HTTP 409 + code 4090，消息带上归属团队名，原归属不变 |
| 添加跨租户 / 已停用 / 不存在的用户 | HTTP 400 + code 4000 |
| 移除不属于该团队的用户 | HTTP 404 + code 4004 |
| 未知团队 id 的读/写/删 | HTTP 404 + code 4004（`DELETE /teams/{id}` 不再对不存在的团队返回成功） |

跨资源引用校验（负责人/成员不属于当前租户、已停用、不存在）走领域层的 `AppError(HTTP 400)`，
业务码是 **4000**；只有路径 ID 非数字、请求体绑定失败这类纯参数错误才是 **1001**。两者分属两层，
前端不得把它们合并成同一句「参数错误」。

`POST`/`DELETE /teams/{id}/members` 的 `data` 直接是操作后的**服务端成员列表**（`TeamMember[]`），
前端用它刷新，不自行拼接本地状态。请求体只有 `{"userId": 10}`。

### 归属校验与租户边界

- `managerId`（负责人）与 `userId`（成员）都是跨资源引用，写入前必须校验目标用户属于**当前租户**且
  处于启用状态；只校验团队自身的 tenant 不足以阻止跨租户关系注入。
- 缺少租户上下文一律 fail-closed：HTTP 401 + code 2001，不会退化成空列表。
- 跨租户访问他人团队（含成员端点）返回 404 + code 4004，不暴露资源是否存在。
- `team:read` / `team:write` 在 DBOnly 权限模式下必须有角色条目；此前 `admin` 角色缺这两条，
  团队管理对租户管理员恒为 403。

回归见 `itsm-backend/handlers/common/team_member_test.go`（列表信封与成员投影、服务端搜索、
分页、跨租户 404、幂等/冲突/参数拒绝、写响应回带服务端状态、fail-closed、负责人跨租户注入）与
`itsm-frontend/src/app/(main)/admin/teams/__tests__/page.test.tsx`（成员列渲染、总数口径、
服务端搜索与翻页、添加/移除/冲突原因透出、错误态重试）。

## 菜单管理接口

路由在 `itsm-backend/router/common_system_routes.go` 注册，HTTP 层在 `handlers/rbac/handler.go`，
业务规则在 `service/menu_service.go`，数据落在 `menus` 表（租户内 `(tenant_id, path)` 组合唯一，
见 `migrations/20261011_add_menus_tenant_path_unique.sql`）。写端点权限为
`system_config:write`、读端点为 `system_config:read`（DBOnly 模式下必须有对应角色条目）。

| Method + Path | 权限 | 说明 |
|:---|:---|:---|
| `GET /api/v1/menus` | `system_config:read` | 当前租户的**全部**菜单，按 `sortOrder` 升序 |
| `GET /api/v1/menus/{id}` | `system_config:read` | 单条菜单 |
| `POST /api/v1/menus` | `system_config:write` | 新建 |
| `PUT /api/v1/menus/{id}` | `system_config:write` | 更新，可选字段是指针 |
| `DELETE /api/v1/menus/{id}` | `system_config:write` | 删除叶子菜单 |
| `GET /api/v1/menus/export` | `system_config:read` | 导出为 seeder 规格（用 `parentPath` 表达层级，不是 `parentId`） |
| `POST /api/v1/menus/init` | `system_config:write` | 按内置基线补齐缺失菜单，返回 `{added,unchanged,totalAdded}` |
| `GET /api/v1/auth/menus` | 仅需登录 | 侧边栏数据源：按当前用户角色与权限码过滤后的菜单树 `{main,admin}` |

### 列表响应：标准信封，但一次给全量、不翻页

`GET /api/v1/menus` 的 `data` 是 `{items,total,page,pageSize,totalPages}`。这里
`page=1`、`pageSize=total`、`totalPages=1` 是**有意为之**：该端点返回的就是当前租户的完整菜单集合，
不接受 `page`/`pageSize` 请求参数。管理页在本地做搜索过滤与按 `parentId` 组装树形，是因为它拿到的
已经是全量数据，而不是在冒充服务端分页；调用方不得对该端点翻页。

`items` 中每条是 `MenuDTO`：`id`、`name`、`path`、`sortOrder`、`tenantId`、`isVisible`、`isEnabled`
恒在；`icon`、`description` 为空串时**整键省略**；`parentId`、`permissionCode` 为 NULL 时**整键省略**
（指针 + `omitempty`）。因此前端必须用「键是否存在」判断上级和权限码，不得写
`record.parentId ?? 0` 之类把「顶级」和「字段缺失」合并的 fallback。响应不含 Ent 的 `edges`，
也不含 `permission_code`、`tenant_id` 这类 snake_case 键。

### 上级菜单与顶级菜单

- 请求里的 `parentId: 0` 与未传同义，都表示**顶级菜单**；创建时不会写入 0 这类悬空外键，库里是 NULL。
- 上级必须是**同租户**的菜单，且不能是自身或自身的后代；违反时 HTTP 400 + code 4000，
  消息分别是 `上级菜单不存在或不属于当前租户` / `上级菜单不能是当前菜单自身` / `上级菜单不能是当前菜单的子菜单`，
  被拒绝的写入不落库。成环会让按 `parentId` 组装的整棵子树在页面上消失。
- `GET /api/v1/auth/menus` 的 `main` / `admin` 划分只看**顶级菜单**的路径前缀：
  `/admin`、`/system`、`/workflow` 开头进 `admin`，其余进 `main`；子树跟随其顶级菜单归属。

### `PUT` 的清空语义

可选字段都是指针，必须区分「未传」「传零值」「清空」：

| 字段 | 未传 | 显式传值 |
|:---|:---|:---|
| `name` / `path` | 保持原值 | 非空则覆盖；空串等同未传（两列必填，不允许被清空） |
| `icon` / `description` | 保持原值 | 显式 `""` **清除** |
| `permissionCode` | 保持原值 | 显式 `""` **清除权限码**（见下方安全含义） |
| `parentId` | 保持原上级 | 显式 `0` **升为顶级**（写 NULL），正整数按上面的规则校验 |
| `sortOrder` | 保持原值 | 按值覆盖，`0` 是合法排序值 |
| `isVisible` / `isEnabled` | 保持原值 | 按布尔覆盖 |

管理页展示全部字段，因此按全量提交并把「清除」表达为空值：移除上级要显式发 `parentId: 0`，
省略键不会解除上级。

**权限码清除是放权操作。** `permissionCode` 为空的菜单在权限过滤里视为「无权限要求」，
对租户内所有登录用户可见。后端在创建时就把权限码与其余字段写进同一条 `INSERT`，避免
「菜单已建、权限码写失败」留下全员可见的行；页面上的清除动作必须让操作者看到这一点。

### 状态码口径

| 场景 | HTTP + 业务码 |
|:---|:---|
| 缺少租户上下文 | 401 + code 2002，不返回空列表 |
| 非 `super_admin` 带 `?tenantId=` 查看他租户 | 403 + code 2003，且不查询 |
| `?tenantId=` 非正整数 | 400 + code 1001 |
| 同租户内 `path` 重复（含并发窗口，唯一索引兜底） | 409 + code 4090 `同级路径菜单已存在` |
| 上级菜单非法（跨租户 / 自身 / 后代） | 400 + code 4000 |
| 未知或跨租户的 `{id}`（读/改/删） | 404 + code 4004 `资源不存在或已被删除` |
| 删除仍有子菜单的菜单 | 409 + code 4090 `请先删除子菜单`，父子行都不受影响 |

领域校验走 `AppError(HTTP 400)`，业务码是 **4000**；只有路径 ID 非数字、`?tenantId=` 非法、
请求体绑定失败这类纯参数错误才是 **1001**。删除不存在的菜单不再返回成功。

### 创建端点的默认值

`POST /api/v1/menus` 未传 `sortOrder` 或传 `0` 时按 **100** 落库（与 `PUT` 不同，创建阶段无法表达
排序 0）；`isVisible` / `isEnabled` 未传按 `true`。`name`、`path` 为必填，缺失是 400 + code 1001。

回归见 `itsm-backend/handlers/rbac/handler_test.go`（列表信封与 camelCase 投影、`?tenantId=`
越权拒绝与 fail-closed、显式清空 payload 的指针到达 service、非法上级映射 400/4000、
删除冲突映射 409/4090、未知删除映射 404/4004）、
`itsm-backend/service/menu_service_test.go`（`TestCreateMenu_DuplicatePathReturnsConflict`、
`TestUpdateMenu_ParentAssignmentSemantics`、`TestDeleteMenu_StateAndTenantSemantics`）与
`itsm-frontend/src/app/(main)/admin/menus/__tests__/page.test.tsx`（从 `items` 渲染与统计口径、
搜索命中子项、错误态、编辑保存的显式清空 payload）。

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

> 资产不存在或跨租户返回 404/`4004`（不再伪装 200 空成功，见 UPGRADE §1.44）。

### 报废资产

```http
PUT /assets/{id}/retire
Authorization: Bearer <accessToken>
```

> 资产不存在或跨租户返回 404/`4004`（见 UPGRADE §1.44）。

### 删除资产

```http
DELETE /assets/{id}
Authorization: Bearer <accessToken>
```

> 删除是幂等可重放的：首次成功删除返回 200；重复删除（已删除/跨租户/不存在）
> 一律 404/`4004`，不再静默假装成功（见 UPGRADE §1.44）。

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

> `userIds` 只保留本租户真实存在的用户 id，跨租户或不存在的用户 id 静默跳过（不占席位、不报错）；席位不足与并发扣减冲突返回 409/`4090`（都是业务冲突而不是 500）；许可证不存在或跨租户返回 404/`4004`（见 UPGRADE §1.44）。

### 删除许可证

```http
DELETE /licenses/{id}
Authorization: Bearer <accessToken>
```

> 已删除/不存在/跨租户的重复删除一律 404/`4004`，不再静默假装成功（见 UPGRADE §1.44）。

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
| `tools[]` | CMDB 相关的 AI 工具定义（含 JSON-Schema 参数契约）；参数键统一为 camelCase（`ciId`/`ciNumber`/`ticketId`/`withRelations` 等），`list_cis` 的 `ciType` 枚举由租户自有 CI 类型动态生成 |

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

### 更新配置项生命周期

```http
PUT /api/v1/cmdb/cis/{id}/lifecycle
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "status": "scrapped",
  "version": 3,
  "retirementReason": "end_of_life",
  "remark": "2023 年采购，已过维保期"
}
```

- `status` 取值 `draft|online|maintenance|offline|scrapped`。迁移由后端状态机校验
  （`common/ci_lifecycle.go` 是唯一事实源）：`draft` 只能到 `online`；
  `online/maintenance/offline` 之间可互转并可退役到 `scrapped`；`scrapped`
  是软删除终态，不可恢复（读接口不再可见）。
- `version` 必填，为乐观锁：与库内不一致时返回 HTTP 409 / code `4090`，
  载荷携带 `{resourceId, currentVersion, serverVersion}`。
- `retirementReason` 仅在 `status=scrapped` 时必填，取值限受控词表
  `manual|decommissioned|replaced|discovery_missing|end_of_life`；缺失或词表外
  返回 HTTP 400 / code `1001`。`PUT /cmdb/items/{id}` 不允许把生命周期直写为
  `scrapped`，退役必须走本接口。
- 非法迁移返回 HTTP 409 / code `4090`；目标 CI 不存在、跨租户或已退役返回
  HTTP 404 / code `4004`。
- 批量入口 `PUT /api/v1/cmdb/cis/batch/lifecycle` 请求体为
  `{ids, status, remark?, retirementReason?}`，逐条执行、逐项返回
  `{successCount, failedCount, failedIds, errors}`，批量路径不做乐观锁。

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
接口边界；创建/更新请求仍可写入该字段。`secret://` 引用必须属于当前租户
（`secret://tenant-<tenantId>/...`），试点环境仅额外允许固定 `env://ITSM_ALIYUN`
allowlist；`PUT` 传空字符串可显式清空凭据引用。云表面的租户一律只取认证上下文
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

### CMDB 治理看板

```http
GET /api/v1/cmdb/governance/report
```

单租户 CMDB 健康快照，读面权限 `cmdb:read`（与对账同权）。数据面与 `GET /cmdb/reconciliation`
同源（云资源全集 + 权威 CI 全集），在其之上消费治理纯函数：

- `quality`：总数/活跃/退役（`lifecycle_status=scrapped`）/陈旧（同步时间超 7 天）/孤儿
  （无 owner、无 assignee、无任何来源）/不完整/完整率（百分比，0.01 精度）。
- `diff`：发现差异四类分桶——`add`（发现侧新增未建账）/`noop`（已关联指纹未变）/
  `retire_confirm`（上次发现的资源本批消失，待人工确认退役）/`duplicate`（同批重复，告警）。
- `cloudResources` / `cis`：两侧全集行数。

空租户返回全 0 快照（`diff` 四桶全在且为 0），不是 `null`。陈旧阈值固定 7 天
（两次周度发现的间隔上限）；租户级配置化待有真实需求再走 `system_configs`，不预造配置面。

契约锁：`handlers/cmdb/governance_report_test.go`（三面数据同源、空租户全 0、读取失败上抛）
与 `router/cmdb_governance_route_test.go`（真实路由三态：无 `cmdb:read` 403、跨租户只见
自己的快照、camelCase 响应键）。CLI 侧 `cmdb.governance report` 命令同步接入（路由覆盖率
棘轮 605 基线保持不变）。

### 云发现任务（Job 闭环）

`POST /api/v1/cmdb/discovery/jobs` 已从 fail-closed（503）转正：真实校验来源后落
`discovery_jobs` 行并入队出箱命令 `cmdb.cloud_discovery.run`（**同一事务**——不存在
没有命令的 queued 僵尸任务）。Worker 认领任务（租约 + fencing token）→ 按来源的
`serviceCodes × regions` 逐 scope 拉取云资源 → 与 `cloud_resources` 按 `identity_hash`
diff（create/update/unchanged）→ `discovery_results` 落 diff 证据 → 资源写回并维护
`first_seen_at/last_seen_at/missing_count`（连续缺失达来源 `staleThreshold` 置
`lifecycle_state=stale`）→ 终态 `succeeded/partial_failed/failed` + summary。

```http
POST   /api/v1/cmdb/discovery/jobs            {"sourceId":"ds_1","idempotencyKey":"可选"}
GET    /api/v1/cmdb/discovery/jobs            ?status=queued&page=1&pageSize=20
GET    /api/v1/cmdb/discovery/jobs/{id}       详情（progress/completedScopes/failedScopes/summary）
POST   /api/v1/cmdb/discovery/jobs/{id}/cancel   queued 直接取消；执行中记取消请求（scope 边界生效）
POST   /api/v1/cmdb/discovery/jobs/{id}/retry    failed/partial_failed 派生新任务（parentJobId 溯源）
```

权限：读 `cmdb:read`、写 `cmdb:write`。幂等语义：携带 `idempotencyKey` 重复创建返回
409（Detail 含既有任务 ID）；不携带则每次生成唯一键，手动重跑不受指纹去重限制。
来源不存在 404、来源/账号停用 400/403。租约字段（leaseOwner/leaseExpiresAt/
fencingToken/heartbeatAt）是执行器内部协调状态，不对外暴露。

### 云发现差异决策（apply / reject）

manual 对账策略下，执行器对「有变化」的资源只落一条 `status=pending` 的
`discovery_result`，**不改** `cloud_resources`（生产属性变更要人确认）。此前没有任何
写入面能让这条差异被应用或驳回——对账停在「算出了差异」，账合不上。

```http
POST /api/v1/cmdb/discovery/results/{id}/apply     # 应用差异（写回 CloudResource）
POST /api/v1/cmdb/discovery/results/{id}/reject    {"reason":"可选"}
GET  /api/v1/cmdb/discovery/results?job_id=42      # 列出差异（含 diff 与 status）
```

- **只有 `action=update` 且 `status=pending` 可被决策**：`create` 在发现时已落库，
  `delete` 根本不产生（缺失只推进 `missingCount`→`stale`，防误删）。
- **应用写的是 `cloud_resources` 的事实**（name/status/region/zone/tags/metadata +
  `sourceFingerprint` 换成被发现的那一版），**不写 CI**——CI 回写尚未接通，
  在这里假装写 CI 会让「配置项与云上一致」变成假承诺。
- **重复决策 409**：已 `applied`/`rejected` 再点不会产生第二次写入，也不当成功返回
  （否则会掩盖「这条差异早被驳回过」）。
- **资源已消失 409**：差异指向的 `identityHash` 在 `cloud_resources` 里找不到时
  提示重跑发现，不自动重建（重建是 `create` 语义，需要新的发现事实）。
- **快照过期 409**：差异是基于 `beforeHash` 那一版资源算出来的。若应用时资源的
  `sourceFingerprint` 已不等于 `beforeHash`（发现后又被另一轮发现自动应用、手工改动，
  或另一条差异先被应用），说明快照已过期——拒绝覆写并提示重跑发现，该行置 `apply_failed`
  （`errorCode=resource_changed`），避免用过期快照悄悄盖掉更新的事实。缺 `beforeHash`
  无法证明新鲜，同样拒绝。
- 应用写入失败时该行置 `apply_failed` 并留 `errorMessage`，不会静默留在 pending。
- 驳回理由可选，写入该行 `errorMessage`（状态为 `rejected` 时语义为决策说明）；
  操作者与请求体由全局审计中间件留痕。

权限：`cmdb:write`。跨租户访问按 404 处理（不暴露可枚举信号）。

### 云发现调度（`discovery_sources.schedule`）

`schedule` 从建表起就存在，但此前**没有任何消费者**——填了 cron 既不生效、也不报错，
云发现只能手动触发。现在由后台调度器每分钟消费一次：

- **表达式**：5 段 cron（`0 2 * * *`）、`@every 24h` 等描述符、ISO 8601 cycle（`R/PT6H`）。
  6 段带秒的 cron 会被拒绝（与 BPMN 定时表达式同一解析器，秒级巡检无业务意义）。
- **时区**：按租户 `tenants.timezone` 解释。`0 2 * * *` 是租户本地时间 2 点，
  不是 UTC 2 点。
- **到期判定**：以该来源**最近一次任务的入队时间**为锚点，而不是 `lastSuccessAt`——
  失败的任务也算跑过，否则一个持续失败的来源会被每分钟重试一次。停机后恢复只补跑
  一次（取锚点之后最近的触发点），不会把停机期间错过的每个触发点都补一遍。
- **在途守卫**：来源还有非终态任务（`queued`/`discovering`/`discovered`/`reconciling`）
  时跳过，避免长跑任务被 cron 堆成一队。
- **幂等**：调度任务按「来源 + 触发点」派生幂等键（`sched-<sourceId>-<fireAt unix>`），
  多副本同时扫描同一触发点时后到者拿 409 并按「已排上」处理，不产生重复任务。

来源读面新增两个字段，让「配了没反应」在保存那一刻就能看见：

```json
{
  "id": "ds_1",
  "schedule": "0 2 * * *",
  "nextRunAt": "2026-10-23T18:00:00Z",   // 下一次会排任务的时刻（UTC，按租户时区计算）
  "scheduleError": "无法识别的定时表达式: ..."  // 仅表达式非法时出现
}
```

`nextRunAt` 未配置 `schedule` 时省略；`scheduleError` 非空表示**该来源永远不会被自动
触发**，需修正表达式（调度器会跳过它并告警，不影响同租户其他来源）。修改 `schedule`
不会立即触发一次发现——新表达式从下一个匹配点开始生效。

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

工作台数字全部挂在 `/api/v1/dashboard` 下，统一要求 `dashboard:read`（写配置面要求 `dashboard:admin`）。
完整路由以 `router/dashboard_routes.go` 为准，本节只锁定 KPI 契约。

### 概览与 KPI 指标

```http
GET /api/v1/dashboard/overview
GET /api/v1/dashboard/kpi-metrics
```

两者返回同一组 `KPIMetric`（overview 里在 `data.kpiMetrics`）：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "pending-tickets",
      "title": "待处理工单",
      "value": 12,
      "unit": "个",
      "color": "#f59e0b",
      "trend": "stable",
      "change": 0,
      "changeType": "stable",
      "comparable": false,
      "scope": "own",
      "description": "当前待处理存量"
    }
  ]
}
```

- `scope` 是工单类计数的行级统计范围，由服务端按登录者角色推导，与 `GET /api/v1/tickets`、
  `GET /api/v1/tickets/stats` 共用同一判定（`datascope.ForTicketReader`）：管理角色为 `tenant`
  （本租户全部工单），一线角色为 `own`（本人创建或受理）。范围不一致时坐席会出现
  「卡片 12、下钻列表 0 条」。请求体/查询参数里的 `tenantId`、`userId` 一律不被采纳。
- `comparable` 表示环比是否有真实上期基线。`false` 时 `change` 恒为 `0`、`trend`/`changeType`
  恒为 `stable`，前端不得渲染「相比上期」——上期窗口没有样本不等于 0% 或 +100%。
- 待处理、处理中、超时是**时点存量快照**，库里没有上月末的状态快照，因此恒为 `comparable: false`；
  总数按「上月末累计」比较，已完成按「上月新建并完成 vs 本月新建并完成」同期队列比较。
- `target`/`alert` 为可选字段（响应里缺省即不出现），SLA 达成率目标 95、平均首次响应 4h、平均解决 8h。
- 趋势、团队负载、事件分布、高峰时段等图表区本期仍是**租户级**口径，不随 `scope` 收窄。

### 获取工单统计

```http
GET /api/v1/dashboard/ticket-stats
Authorization: Bearer <accessToken>
```

### 获取 SLA 统计

```http
GET /api/v1/dashboard/sla-stats
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

### 获取系统运行状态

```http
GET /system-configs/status
Authorization: Bearer <accessToken>
```

需要 `system_config:read` 权限。`GET /configs/status` 与 `GET /dashboard/stats/system` 返回同一个 DTO，
只有这一套采集口径。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "uptime": 244.7,
    "cpuUsage": 0.011,
    "memoryUsage": 0,
    "diskUsage": 98.88,
    "avgResponseTime": 38.33,
    "requestsPerSecond": 0.158,
    "errorRate": 0,
    "metricsWindowSeconds": 101.14,
    "dbConnections": 5,
    "dbSize": 22985751,
    "cacheHitRate": 34.72,
    "cores": 16,
    "goroutines": 40,
    "sampledAt": "2026-10-15T22:57:09+08:00",
    "availability": {
      "cpu": "measured",
      "memory": "unavailable",
      "disk": "measured",
      "http": "measured",
      "dbSize": "measured",
      "cache": "unconfigured"
    }
  }
}
```

- `availability` 取值 `measured` / `unavailable` / `unconfigured`；非 `measured` 的字段恒为 0，调用方必须按该标记渲染「未接入」，不得当成实测值参与评级。
- `metricsWindowSeconds` 是 `avgResponseTime`、`requestsPerSecond`、`errorRate` 覆盖的时间窗：区间足够时为「距上一次采样的增量」，区间不足 1s（多个轮询者共用同一基线）或首次采样时退回启动以来的累计口径，此时等于 `uptime`。
- `cacheHitRate` 只有在后端注入 Redis 客户端后才为 `measured`；`errorRate` 只统计 5xx。

### 默认配置的来源

默认配置不再提供初始化接口。`GET /system-configs` 在租户配置表为空时懒写入 24 项默认配置，
返回值即为写入后的列表；此前的 `GET /system-configs/init`（以及别名 `GET /configs/init`）已移除——
它按读权限守卫却整表写入密码策略、2FA 等安全配置，任何持 `system_config:read` 的账号一次 GET 即可落地。
写配置仍走 `PUT /system-configs/:id`（需 `system_config:write`）。

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
| `GET /api/v1/tickets/templates` | `{items, total}` | 模板全量返回；历史实现伪造 `page=1`、`pageSize=len(items)`，并在服务端写死 `PageSize=100` 造成第 101 条静默消失（2026-10-21 去掉截断，见「工单模板接口」） |
| `GET /api/v1/tickets/views` | `{items, total}` | 视图按租户全量返回（`service/ticket_view_service.go` 无 `Limit`），`total=len(items)` 诚实 |
| `GET /api/v1/tickets/{id}/comments` | `{items, total}` | 单工单评论全量返回，同上；分页若将来引入必须实装而非补键 |
| `GET /api/v1/tickets/assignment-rules` | `{items: AssignmentRuleResponse[], total}` | 按 `priority DESC, created_at DESC` 全量返回本租户规则；无任何查询参数，`total=len(items)` 诚实；读写整组接口需 `assignment_rule:read`/`assignment_rule:write`。`DELETE /{id}` 幂等可重放：条件删除 0 行（已删/跨租户/不存在）一律 404/`4004`，不再静默假装成功；`POST /assignment-rules/test` 的 `ruleId`/`ticketId` 均带租户谓词，跨租户引用与不存在同名 404/`4004`，他租工单不再被读取做匹配或回显动作（2026-10-18 纠正，见 UPGRADE §1.45） |
| `GET /api/v1/tickets/automation-rules` | `{items: AutomationRuleResponse[], total}` | 同上排序；item 带 `createdBy`/`creator`/`tenantId`，读取需 `automation_rule:read`（规则配置面自有码，不是 `system_config:read`，也不是最终用户为看流转进度而持有的 `workflow:read`，见 UPGRADE §1.41） |
| `GET /api/v1/tickets/{id}/attachments` | `{items: TicketAttachmentResponse[], total}` | 单工单附件按 `created_at DESC` 全量返回；读取需 `ticket:read` 且调用者是工单相关方或管理角色，跨租户工单是 404/4004 |
| `GET /api/v1/tickets/cc/my` | `{items: TicketCCRecordResponse[], total}` | 抄送给当前用户的记录，按 `tenant_id + user_id + is_active` 全量返回，排序 `added_at DESC, id ASC`（`id` 是并列键）；`user`/`addedBy` 是 `WorkflowUserInfo`，姓名键是 `fullName`，没有 `name` |
| `GET /api/v1/tickets/{id}/cc` | `{items: TicketCCRecordResponse[], total}` | 单工单抄送列表，同上排序；读取需 `ticket:read`（2026-10-20 起，此前误挂 `workflow:read`，见 UPGRADE）且调用者是工单相关方/审批人/管理角色。跨租户探测与工单不存在返回同一个 404/`4004`（不确认对方资源存在），同租户但非相关方的用户是 403/`2003`；三种失败都不携带 `items` 载荷（2026-10-04 由 500/`5001` 纠正，见 UPGRADE §1.20） |
| `GET /api/v1/tickets/assign-recommendations/{id}` | `{items: AssignmentRecommendation[], total}` | 按评分降序全量返回本租户的可用候选（`service/ticket_assignment_smart_service.go:116`：`user.TenantIDEQ` + `ActiveEQ(true)` 整表取回后排序），`total=len(items)` 诚实；item 是 `{userId,username,name,email,score,reason,workload,skills?,categories?}`，**没有** `userName`/`userEmail`/`userAvatar`/`factors`。读取需 `ticket:assign`（单工单分配动作，与 `POST /tickets/{id}/auto-assign` 同码：候选表带 email/workload/skills，不对只读角色开放；旧口径 `system_config:read` 见 UPGRADE §1.41）；工单不存在与跨租户返回同一个 404/`4004`（同样是 E4-5 本片由 500/`5001` 纠正），不泄漏对方租户候选 |
| `GET /api/v1/tickets/{id}/notifications` | `{items, total}` | 单工单通知按 ticket+tenant 全量返回（`service/ticket_notification_service.go:831` 无 `Limit`），handler 里 `total=len(items)` 诚实 |
| `GET /api/v1/msp/reports/customers` | `{items, total}` | 区间聚合，字段为 camelCase DTO |
| `GET /api/v1/msp/reports/performance` | `{items, total}` | 同上 |
| `GET /api/v1/msp/allocations/history` | `{items, total, page, pageSize, totalPages}` | 标准分页信封，元素是 `MSPAllocationDTO` |

### MSP 能力开关（部署形态）

`DEPLOYMENT_MODE=private` 时 `/api/v1/msp/*` 路由仍然注册，但整个路由族被门控拒绝。
响应是 HTTP 503 / `code: 5003`，`data` 显式给出能力状态：

```json
{
  "code": 5003,
  "message": "MSP 能力在当前部署形态下未启用",
  "data": { "capability": "msp", "state": "disabled" }
}
```

这不是 404：路由存在，只是当前部署形态没有启用该能力。调用方应据 `data.state` 区分
「未启用」与「客户/分配记录不存在」，不要重试创建；运维侧对应配置项是 `DEPLOYMENT_MODE`。
`saas`、`saas_msp` 或留空时 MSP 路由正常放行，原有身份与分配校验不变。

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

## 连接器健康检查

回答「这个连接器现在通不通」的只读接口，权限码 `connector:read`。

```http
GET /api/v1/connectors/health
Authorization: Bearer <accessToken>
```

`data` 结构：

| 字段 | 说明 |
|---|---|
| `tenantId` | 认证租户；响应内容不跨租户 |
| `items[]` | 本租户运行中的连接器实例（按 `name`、`provider` 排序） |
| `items[].name` / `provider` | 实例标识，前端按这两个字段定位健康结果 |
| `items[].ok` / `latencyMs` / `message` / `checkedAt` | 真实健康检查结果（拨号到第三方，5 秒超时） |
| `items[].extra.lastSendSuccessAt` / `lastSendFailureAt` / `lastSendError` | 最近一次发送成败，来自实例运行时记录 |
| `checkedAt` | 本次快照时间 |

- 只拨**当前租户**已装配的实例。健康检查会真实拨号，若返回全租户实例，等于给普通租户一个探测其他租户端点可达性的探针；同一约束适用于 `GET /connectors`、`GET /connectors/configs`、`GET /connectors/lifecycle` 以及 `POST /connectors/configs` 的响应。
- `items[].checkedAt` 由 Manager 在实际检查完成时统一落戳，不会出现 `0001-01-01T00:00:00Z`；市场 / 配置 / 生命周期视图的 `lastCheckedAt` 在从未检查过时省略字段。
- 实例以 `name`/`provider` 标识，租户号不再编进 key。历史响应是 `map["<tenantId>/<name>/<provider>"]`，调用方只能拿租户号拼键——前端就是在这里把租户号硬编码成了 `0`，导致状态列永远取不到结果。
- 该端点会触发真实拨号（每实例 5 秒超时），实例较多时不要轮询。

## 连接器凭据轮换

轮换一个已配置连接器实例的凭据（飞书的 `appSecret`、企微/钉钉的 `token`、Webhook 的签名密钥等）。权限码 `connector:write`。

```http
POST /api/v1/connectors/configs/{name}/rotate-secret
Authorization: Bearer <accessToken>
Content-Type: application/json

{ "newCredentials": { "appId": "cli_xxx", "appSecret": "new-secret" } }
```

`data`：`{ name, provider, rotated, rotatedAt, note }`。响应**不回显任何凭据值**（新旧都不回显）。

| 情形 | 结果 |
|---|---|
| 成功 | `200`；运行态与库内凭据同时换成新值，并写一条 `audit_log`（`action=rotate_secret`） |
| `newCredentials` 为空 | `400`，文案说明至少要提交一个字段 |
| 连接器实例不存在 | `404` |
| 新凭据无法初始化（必填项缺失 / 第三方拒绝） | `400`，**已保留原凭据**，运行态与库都不动 |
| 初始化成功但落库失败 | `5xx`，**运行态回滚为原配置**（旧实现是 Revoke，连接器会直接变不可用） |

- **旧凭据立即失效**，不存在「新旧并行」的宽限窗口：三家 IM 的发送与验签都只读一份凭据。要留退路，先在对方平台保留旧密钥，需要时用旧凭据再调一次 `POST /connectors/configs` 覆盖回来（历史实现里的 `gracePeriodHours` 从未被任何代码执行，已移除）。
- 审计只记录连接器、provider、操作人、时间与**凭据键名**，不写凭据值。
- `provider` / `type` / `settings` 一律沿用原实例，不能由请求体指定——否则等于允许借轮换请求改掉 `Init` 走哪个实现。
- 审计写失败不回滚轮换（密钥已在对方平台换掉，回滚只会造成更大不一致），但会以 error 级别留痕。

## 连接器市场安装与卸载（运行态闭环）

市场里的连接器类商品（`type=connector`）安装时不只是插一条记录，还要把实例真正
注册进运行态（`connector.Manager`）。这两件事可能一件成一件败，接口的返回语义就是
把失败如实交出来，而不是「记录写成功就算成功」。

```http
POST /api/v1/marketplace/items/{itemId}/install
Authorization: Bearer <accessToken>
```

`data` 是安装记录（与 `GET /marketplace/installations` 同结构），关键字段：

| 字段 | 含义 |
|---|---|
| `status` | `installing` / `active` / `disabled` / `failed` / `uninstalled` |
| `errorMessage` | 最近一次失败原因。连接器类商品**必看**：`failed` 时这里写着为什么起不来 |

| 情形 | 结果 |
|---|---|
| 连接器成功注册进运行态 | `200`，`status=active` |
| 技能或插件（`type=skill/plugin`）尚无分发、加载实现 | `501`，`code=5010`；不新增安装记录、不重试或激活历史记录、不增加安装计数 |
| 记录已建但连接器起不来（缺凭据 / provider 未注册 / 端点不可达） | `200`，`status=failed` + `errorMessage`，响应 `message` 提示补全配置 |
| 商品不存在 / 未发布 | `404` / `400` |
| 连接器已安装且 `active`/`disabled` | `200` 幂等返回原记录，不重复注册 |
| 上次停在 `installing`/`failed` | 视为**重试**：重新注册连接器，重复调用安装即是重试入口 |
| 卸载后重装（历史记录复用） | 同样重新走注册，不会因为「记录已存在」就直接标 `active` |
| 待装版本声明的依赖未满足（前置未安装 / 版本不符 / 自依赖） | `409`，`message` 逐项列出缺口，**不落安装记录** |

**失败不返回 4xx 的原因**：资源（安装记录）确实被创建了，`409`/`500` 会让人以为
什么都没发生。返回 `200 + failed` 让前端能显示「已安装但未就绪 + 原因」，用户补完
配置点重试即可，安装痕迹也不会丢。

**依赖未满足返回 409 的原因相反**：记录根本没创建，商品本身也是可安装的，缺的是
前置链条。这属于业务冲突而非参数错误，`409` 才能让调用方区分「我传错了」和
「环境没准备好」。

### 依赖闸门

`ItemVersion.dependencies`（`{组件名: 版本要求}`）是安装期的硬约束：

- 只看**待装版本**的声明；没有对应版本行、或依赖为空，视为未声明依赖（存量与种子商品照常可装）。
- 前置组件必须在**同一租户**内存在 `active` 或 `disabled` 的安装记录。`disabled` 算已安装（管理员主动停用，组件仍在租户内）；`installing`/`failed` 不算。
- 版本要求支持 `>=` `>` `<=` `<` `=` `==` 前缀，无前缀按精确匹配；版本号按 `x[.y[.z]]` 数字比较，忽略 `v` 前缀与 `-prerelease` 后缀。
- 缺口聚合成一条 `409`，`message` 逐项写明「未安装，需要 X」/「已安装 A，需要 B」/「组件不能依赖自身」。
- 校验发生在落安装记录**之前**，`installing`/`failed` 的重试同样要过闸门。

### 补配置即闭环

```http
PUT /api/v1/marketplace/installations/{itemId}/config
Authorization: Bearer <accessToken>
Content-Type: application/json

{ "credentials": { "appId": "cli_xxx", "appSecret": "..." } }
```

| 情形 | 结果 |
|---|---|
| 新配置让连接器成功起来 | `200`；写入配置，`installing`/`failed` 的记录一并转 `active` 并清空 `errorMessage` |
| 新配置无法初始化 | `409`，**库内保留上一份可用配置**，运行态仍是旧实例 |
| 技能或插件的历史安装记录 | `501`，`code=5010`；配置、状态和时间戳保持不变 |
| 安装记录不存在、已卸载或属于其他租户 | `404`，`code=4004` |

技能和插件仍可浏览目录；历史安装记录可查看、卸载，但历史 `active` 状态不代表已经加载运行，本次不批量改写存量数据。

顺序是**先让运行态用新配置起来、成功后才落库**。反序会在失败时留下「库是新凭据、
内存是旧凭据」的分裂状态：用户以为换了 token 已生效，实际仍在用旧的发消息。

### 卸载

```http
POST /api/v1/marketplace/items/{itemId}/uninstall
```

| 情形 | 结果 |
|---|---|
| 运行实例已注销 | `200`，记录转 `uninstalled` |
| 未安装 | `404` |
| 仍被同租户内 `active`/`disabled` 的已安装组件依赖 | `409`，`message` 列出依赖方（标题 + 名称 + 已装版本）；**记录保持原状态，连接器未 revoke** |
| 运行实例未能注销（同 `name` 的实例仍在） | `409`，**记录保持原状态**（不会标成 `uninstalled`） |

反向依赖检查排在 `Revoke` **之前**：先卸运行态再判定不允许卸载，会留下「记录仍
`active`、连接器已经没了」的分裂状态。判定依据是各依赖方**已安装版本**的依赖声明
（不看市场最新版本——新版本新增的依赖与存量安装无关，拿它判定会挡住无关卸载）；
`failed` 的依赖方不拦（未生效，其重试会重新过安装闸门）。

`Revoke` 以 `(tenant, name, provider)` 为键删实例。历史上 `provider` 若与安装时不一致，
键就不匹配、实例会残留——此时若照样标 `uninstalled`，界面显示已卸载而连接器仍在用
活着的凭据收发消息、仍能接收回调。因此卸载后按 `name` 复查运行态，残留即失败关闭。

## 商品目录管理（平台级）

目录（`marketplace_items`）是**全局表**：没有 `tenant_id`，所有租户共享同一份。因此
「安装」与「上架」是两件事、两个权限码：

| 动作 | 端点前缀 | 权限码 | 授予谁 |
|---|---|---|---|
| 浏览、安装/卸载/改配置 | `/api/v1/marketplace/*` | `marketplace:read/write` | 租户管理员 |
| 上架、下架、改元数据、维护版本 | `/api/v1/marketplace/catalog/*` | `marketplace_catalog:read/write` | 仅 `sysadmin` |

复用 `marketplace:write` 做目录写会让任一租户管理员改写全平台商品：把 `name` 指向未注册
provider，可让所有租户的安装按钮全部失败。因此目录写独立成面，且**租户管理员不会看到
对应菜单**（菜单权限码同为 `marketplace_catalog:read`）。

### 端点

| 方法与路径 | 说明 |
|---|---|
| `GET /marketplace/catalog/items` | 目录列表，**包含草稿与已下架**；`status` / `type` / `search` / `page` / `pageSize` |
| `POST /marketplace/catalog/items` | 创建（初始 `draft`，租户不可见） |
| `GET /marketplace/catalog/items/{id}` | 详情，内联返回 `versions` |
| `PATCH /marketplace/catalog/items/{id}` | 更新元数据（PATCH 语义：未提交字段不变） |
| `DELETE /marketplace/catalog/items/{id}` | 删除（仅未上架且从未被安装） |
| `POST /marketplace/catalog/items/{id}/publish` | 上架（`draft`/`deprecated` → `published`） |
| `POST /marketplace/catalog/items/{id}/unpublish` | 下架（`published` → `deprecated`） |
| `GET /marketplace/catalog/items/{id}/versions` | 版本列表（按发布时间倒序） |
| `POST /marketplace/catalog/items/{id}/versions` | 新增版本（默认 `draft`；`publish=true` 同时发布） |
| `POST /marketplace/catalog/items/{id}/versions/{version}/publish` | 发布版本并推进 `latestVersion` |

### 四条不变量（接口会拒绝违反它们）

1. **`name`/`type` 创建后不可改**。安装时 `name` 决定连接器 provider（去掉 `-connector`
   后缀后查注册表），改名等于让已安装租户指向另一个 provider。
2. **连接器类商品的 `name` 必须对应已注册 provider**。`type=connector` 且 provider 未注册
   时创建直接 `400`，响应里列出已注册清单——否则商品能上架、能被安装，但 `provision`
   永远失败，而错误只在租户点安装那一刻出现。标题/描述/图标/能力/配置 Schema 在未显式
   提供时取运行时 manifest，保证目录描述与代码能力一致。
3. **`latestVersion` 只能由「发布某个版本」派生**，不能手填。上架时若没有任何已发布版本
   返回 `409`；发布版本时按发布动作推进（而不是按版本号大小），因此**重新发布旧版本就是
   回滚**。安装记录写的是 `latestVersion`，这条不变量决定「装的到底是哪个版本」。
4. **下架不删除**。`deprecated` 条目对租户不可见、不能再安装，但已安装租户的卸载/改配置
   仍需要读它；删除被限制为「未上架 + 从未被安装」。

### 错误语义

| 情形 | 结果 |
|---|---|
| `name` 不合规（非小写字母/数字/连字符） | `400` |
| 连接器 provider 未注册 | `400`，文案含 provider 名与已注册清单 |
| 版本号非 `x.y.z`（允许 `-rc.1`），或连接器版本与运行时 manifest 不一致 | `400` |
| 同名商品 / 同版本号已存在 | `409` |
| 改 `name`/`type` | `409` |
| 上架但无已发布版本 | `409`，提示先发布版本 |
| 删除已上架条目 | `409`，提示先下架 |
| 删除已被安装的条目 | `409`，文案含安装数 |
| 条目或版本不存在 | `404` |

`name` 允许 `v` 前缀但入库前统一去掉：`v1.0.0` 与 `1.0.0` 在 `UNIQUE(item_id, version)`
眼里是两个版本，会让同一版本在目录里出现两行。

## 入站回调的安全与去重（飞书 / 企微 / 钉钉）

三家 IM 的外部回调都在 `public` 分组（无本系统鉴权），因此安全性完全由「验签 +
时间窗 + 一次性 nonce」决定，业务安全由「持久化去重」保证。三家走同一套语义：

| 步骤 | 说明 |
|---|---|
| 1. 验签 | 飞书 HMAC-SHA256(timestamp + nonce + body)、企微 SHA1 字典序拼接、钉钉 HMAC-SHA256(timestamp + "\n" + body)。失败 `403` |
| 2. 时间窗 | 请求时间戳必须落在 **±5 分钟**内，否则按重放拒绝 `403` |
| 3. 去重 | `(tenant_id, connector_name, event_id)` 唯一登记：飞书用 `timestamp:nonce`、企微/钉钉用消息/事件 ID |
| 4. 处理 | 首次请求继续处理；已登记且**未过期** → 视为重复，直接返回平台期望的成功响应 |
| 5. 标记或释放 | 处理成功写 `ok`/`ignored`；处理失败**释放该行**，让平台重投能重新接管 |

**去重是持久化的**（`connector_inbound_dedup` 表，非进程内 map）：进程重启、多副本
部署都不会丢去重窗口。表中 `response_status` 记录本次事件的处置结果
（`processing` / `ok` / `ignored`），可直接回答「平台说重投了，我这边到底处理成功没有」。

| 情形 | 结果 |
|---|---|
| 首次事件 | 正常处理 |
| 同一事件 TTL（5 分钟）内重投 | 判重复，返回平台期望的成功响应（不重复执行业务） |
| 同一事件在 TTL 后被重投 | 过期行可被覆盖，重新处理（不依赖清理任务是否跑过） |
| 业务处理失败 | 释放去重行并返回 `5xx`，平台重投时重新接管——**不会丢事件** |
| 去重存储不可用 | `500` fail-closed（放过去重等于允许同一事件被二次执行） |

去重表由后台任务每 10 分钟清理过期行（此前 `CleanupExpired` 只有单测调用，
表只增不减）。

### 企微入站解密（url_verification 与事件回调）

企微回调 body 是密文（`Encrypt`/`echostr`），由连接器用 `encoding_aes_key` 解密
（AES-256-CBC，43 位 base64 密钥补 `=` 解出 32 字节，IV 取密钥前 16 字节，PKCS#7
块大小 **32**——企微约定而非 AES 的 16）。解密在 `connector/builtin/wecom`（持有密钥），
handler 只做验签与透传：

- **URL 验证**（`echostr`）：`echostr` 本身是密文，解密后校验 `receiveid == corp_id`，
  返回**明文** echo。此前把密文原样回显，企微后台保存回调配置永远校验失败。
- **事件回调**（`Encrypt`）：解密 → 校验 `receiveid` → 解析 XML 为结构化字段
  （`MsgType`/`Content`/`UserID`/`MessageID`）。无 `MsgId` 的事件（如菜单点击）按
  `CreateTime:FromUserName:Event:EventKey` 合成去重键，不再因缺 ID 而漏去重。
- `encoding_aes_key` 未配置或 `receiveid` 与 `corp_id` 不符一律 fail-closed 拒绝。
- handler 入站日志只记 `message_id`，不再打印解密后的消息正文（隐私）。

仍是预览能力：入站解密与去重已可用，但「消息 → 工单/事件」的业务派发尚未接线。

### 飞书 OAuth 授权回调（state 防 CSRF）

与上面的 webhook 事件回调不同，OAuth 授权回调（`GET /api/v1/feishu/oauth/callback/:instance_id`）
换取的是用户授权 token 并写回连接器配置，安全性由 **一次性 state** 保证：

- `GET /api/v1/feishu/oauth/auth-url`（权限 `connector:write`）由服务端生成不可猜测的随机
  `state`，绑定发起时的租户/连接器实例/用户，随 `authUrl` 一起下发；**不再信任客户端传入的
  state，也不再回退到可枚举的 `tenant_id/user_id` 拼接**。
- 回调在换取 token **之前**校验并单次消费 `state`：缺失、伪造、过期（10 分钟）、重放或
  租户/实例绑定不符一律 `403`（业务码 `2003`）。命中即删除，正确绑定也只能用一次。
- state 存储是进程内 map（与 webhook 去重的内存回退同一局限）：多副本部署下回调落到另一副本
  会因查不到 state 而拒绝（fail-closed，安全但牺牲可用性），HA 需共享存储或会话粘滞。

## 通知渠道诊断

回答「为什么我没收到通知」的只读接口，权限码 `notification:read`。与投递审计互补：审计回答「发了但失败了」，渠道诊断回答「根本发不出去」。

```http
GET /api/v1/notifications/channels
Authorization: Bearer <accessToken>
```

`data.items[]` 每项包含：

| 字段 | 说明 |
|---|---|
| `channel` | 渠道名（连接器实例名，如 `email`/`feishu`；站内信为 `in_app`） |
| `provider` / `type` | 连接器提供方与类型 |
| `healthy` / `healthMessage` | 真实健康检查结果（拨号到第三方，5 秒超时） |
| `checkedAt` | 本次检查时间 |
| `lastSendSuccessAt` / `lastSendFailureAt` / `lastSendError` | 最近一次发送成败，来自连接器实例的运行时记录 |
| `recipientTarget` | 当前登录用户在该渠道的收件目标，已脱敏（如 `ch****om`） |
| `deliverable` | `healthy && recipientTarget != ""`；站内信恒为 `true` |
| `blockedReason` | 不可投递时的可读原因（未配置目标 / 连接器不健康） |

- `in_app` 恒排在首位且恒为可投递——它是账号有效即可达的兜底渠道，不依赖连接器。
- 健康检查只探测**当前租户**已装配的连接器（`connector.Manager.HealthCheckTenant`），不会借这个接口探测其他租户的端点。
- 接口只读，不触发真实发送；需要验证「能否送达」时先看这里，再用投递审计确认实际发送结果。

## 通知投递审计

`notification_deliveries` 由命令总线（`commandbus`）的投递处理器写入，记录每个「接收人 × 渠道」的投递状态、重试次数、供应商消息 ID 与失败原因。以下接口是该表的唯一读面，权限码 `notification:read`（重试需要 `notification:write`），全部按认证租户隔离。

```http
GET  /api/v1/notification-deliveries?channel=feishu&status=failed&recipientId=12&ticketId=88&from=2026-10-01&to=2026-10-20&page=1&pageSize=20
GET  /api/v1/notification-deliveries/stats?days=7
POST /api/v1/notification-deliveries/{id}/retry
```

- `from`/`to` 接受 `YYYY-MM-DD`（服务器本地时区，`to` 含当日）或 RFC3339；格式非法返回 **400/1001**，不返回 500。
- 列表响应为统一分页信封 `{items,total,page,pageSize,totalPages}`；`targetMasked` 是写入时脱敏后的目标（如 `us****01`），接口不回显明文收件地址。
- `stats` 按 `channel × status` 聚合，返回 `byStatus` 与 `byChannel[]`；`successRate` 的分母只计已定论的投递（`sent + failed`），`pending` 不计入，避免队列积压把成功率压低。
- `retry` 只接受 `status=failed`：`sent` 返回 **409**（无需重试），`pending` 返回 **409**（仍在队列中），跨租户或记录不存在返回 **404**。重试不修改旧记录——旧记录保留为失败证据，新命令由 worker 执行后写入一条新的投递记录，因此同一通知可能有多条投递记录（审计设计，不是重复写入）。
- `retry` 是**幂等**的：同一条投递反复点「重试」会生成多条命令、每条都真实发送一次，因此服务端按「重试命令谱系」拦截——已有 `pending`/`processing` 的重试 → **409**「已有在途重试」；上一次重试已 `succeeded` → **409**「已重试成功」；上一次重试 `dead_letter`/`cancelled`（确实没送出去）才放行。判定依据是重试命令的幂等键 `<原键>:retry:<纳秒>`，不新增投递状态（旧记录保持 `failed` 是刻意的失败证据）。

## 死信人工重放（命令重放与投递重试）

命令总线（`operational_commands`）里状态为 `dead_letter` 的命令表示「worker 重试耗尽仍未完成」。这类命令可能从未进入业务处理（payload 不全、handler 缺失、聚合已删除），也可能是发送环节失败。人工重放的入口分两层，**语义不同，不能互相当成替代**：

| 层 | 端点 | 作用 | 幂等语义 |
|---|---|---|---|
| 命令重放 | `POST /api/v1/admin/operations/commands/{id}/replay` | 把一条 `dead_letter`（或 `cancelled`）命令重置回 `pending` | **保留原 `idempotency_key` 与 `fencing_token`**，worker 仍按原身份去重，不会与历史成功的业务写入重复 |
| 批量重放 | `POST /api/v1/admin/operations/commands/bulk-replay` | 按 `{status,commandType,aggregateType,pageSize}` 匹配后批量重置 | 同上；单次上限 200 条（`maxBulkLimit` 1000），`status` 只允许 `dead_letter`/`cancelled` |
| 命令取消 | `POST /api/v1/admin/operations/commands/{id}/cancel` | 把一条 `pending`/`processing`/`dead_letter` 命令置为 `cancelled` | `fencing_token`+1、清 lease；`dead_letter` 可直接取消（重试耗尽后的自然终态，无需先 replay） |
| 批量取消 | `POST /api/v1/admin/operations/commands/bulk-cancel` | 按 `{status,commandType,aggregateType,pageSize}` 匹配后批量取消 | `status` 允许 `processing`/`pending`/`dead_letter`；终态（`succeeded` 等）拒绝 |
| 投递重试 | `POST /api/v1/notification-deliveries/{id}/retry` | 以**新幂等键**重新入箱一条失败投递 | 会真实再发一次消息；受上面的「重试谱系」幂等守卫保护 |

```http
GET  /api/v1/admin/operations/commands?status=dead_letter&commandType=notification.deliver&page=1&pageSize=20
POST /api/v1/admin/operations/commands/{id}/replay
POST /api/v1/admin/operations/commands/bulk-replay
POST /api/v1/admin/operations/commands/{id}/cancel
POST /api/v1/admin/operations/commands/bulk-cancel
```

- 权限 `system:write`（读列表与重放同权限组），全部按认证租户隔离；每次重放写一条 `audit_log`（`resource=operational_command`、`action=replay`、`body` 含 `previousStatus`）。
- 列表响应 `data` 为 `{items,total,page,pageSize,totalPages,summary,byType}`；`summary`/`byType` 是聚合附加字段，不是分页元数据。`summary` 给出 `pending/processing/deadLetter/cancelled/succeeded` 与 `stuckLeases`、`oldestWaitingAt`，是判断「积压还是死信」的依据。
- 批量重放返回 `{matchedIds,updated,skipped,pageSize}`：`skipped` 是在事务内被并发改状态（例如 worker 恰好复活了该命令）而未能命中的条数，不是错误。
- 重放不修改 payload，也不重置 `attempt`——`attempt` 是这条命令的历史尝试次数，重置会掩盖真实失败次数；需要清零的语义请新建命令。

产品入口：**管理 → 死信与重放**（`/admin/operations`），两个 Tab 分别对应命令死信与失败投递。此前只有 HTTP 端点，界面里唯一可点的「重放」在租户初始化抽屉里（仅覆盖产品基线安装这一类命令）。

## 值班排班

排班表（schedule）属于某个支持组，班次（shift）是具体人员的时间段；同一排班表内班次不允许时间重叠。规范入口为 `/api/v1/on-call`，权限码 `on_call:read` / `on_call:write`。历史入口 `/api/v1/email-intake/on-call/*` 仍然可用并共用同一服务实例，但不再新增能力（缺少排班表改名/停用/删除）。

```http
GET    /api/v1/on-call/schedules?groupId=5
POST   /api/v1/on-call/schedules
GET    /api/v1/on-call/schedules/{id}
PUT    /api/v1/on-call/schedules/{id}
DELETE /api/v1/on-call/schedules/{id}

GET    /api/v1/on-call/shifts?scheduleId=61&from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z
POST   /api/v1/on-call/shifts
PUT    /api/v1/on-call/shifts/{id}
DELETE /api/v1/on-call/shifts/{id}

GET    /api/v1/on-call/current?groupId=5&at=2026-10-20T02:00:00Z
```

- 排班表缺省时区 `Asia/Shanghai`、缺省状态 `active`；`groupId` 必须属于当前租户。
- 班次要求 `endAt > startAt`（否则 **400**）、值班人是该支持组的在职成员（否则 **400**）、与同排班表已有班次不重叠（否则 **409**）。
- 仍有班次的排班表不允许删除（**409**），避免历史派单失去可解释性；先删班次再删排班表。
- `current` 在无匹配班次时返回 **404**「该支持组当前没有值班人」；同一时刻命中多个班次时取最早开始的一个（交接班重叠的确定性规则）。

## 能力矩阵健康字段

`GET /api/v1/capabilities` 的每个条目新增 `health` 对象（`{ok,reason,checkedAt}`），`lastHealthCheckAt` 改为真实探测时间而非请求时刻。探测覆盖数据库连通性（全部能力）、连接器健康（`connector.*`）、租户是否启用身份提供方（`identity.oidc`）；探测失败时 `deploymentReady` 一并置 `false` 并给出 `degradedReason`。`identity.oidc` 此前误标为 `disabled`，现为 `pilot`（SSO/OIDC 已有 11 个端点），是否可用取决于租户配置。

## 能力矩阵 pilotReason（2026-10-22）

`GET /api/v1/capabilities` 对 `maturity=pilot` 的条目下发 `pilotReason`（字符串），回答两件事：**为什么它还不是 GA**，以及**晋升需要补哪块可验证证据**（理由必须以「晋升条件」结尾）。`ga` 条目不下发该字段。

```json
{
  "key": "cmdbDiscovery",
  "maturity": "pilot",
  "deploymentReady": false,
  "degradedReason": "deployment dependency is not ready",
  "pilotReason": "发现任务没有执行者：ClaimDiscoveryJob 仅被单测调用，队列无消费者，任务创建后不会被领取执行；……晋升条件：worker 消费 + 真实账号发现验收。"
}
```

- `degradedReason` 与 `pilotReason` 语义不同，**不可互相替代**：前者是「此刻为什么不可用」（会变，修好即消失），后者是「本来就还没承诺」（发布决策）。两者可能同时存在。
- 该字段是后端强制契约：`pilot` 缺理由、理由里没有「晋升条件」、或 `ga` 条目带理由，都会让 `handlers/capability` 的测试失败——过期理由等于对客户宣称一个已经不存在的问题。
- 同批晋升：`problem` 与 `template` 由 `pilot` 转 `ga`（问题域的 SLA/评论/关闭证据/KEDB 关联四项阻断均已闭环且装配进生产路径；模板的真实表面是 `/api/v1/tickets/templates` 的 8 个端点 + `/tickets/templates` 页，旧注释引用的通用 `/api/v1/templates` 从未存在）。
- 运营视图：**管理 → 能力矩阵**（`/admin/capabilities`）展示同一份矩阵（成熟度、部署就绪、探测结果、`pilotReason`），页面只读——成熟度是代码里的发布决策，不应由界面误点改变对外承诺。

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2024-01-01 | 初始版本 |

## 联系支持

如有问题，请联系技术支持团队。
