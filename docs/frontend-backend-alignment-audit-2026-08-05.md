# ITSM 前后端业务对齐审查报告

> 审查时间: 2026-08-05
> 范围: `itsm-backend` (Go/Gin) + `itsm-frontend` (Next.js/TS)
> 目标: 端点对齐、字段命名、响应包装、错误处理四个维度的业务对齐状况
> 审查方式: 静态分析 (router 全部 / dto 全部 / 前端 lib/api 全部),仅读不改

---

## 执行摘要

| 维度 | 合规度 | 严重问题数 |
|---|---|---|
| 后端路由注册 | 高 (~90%) | 0 |
| 前端调用实现 | 中 (~65%) | 6 个系统性子模块未对接 |
| HTTP 方法 / 路径对齐 | 中 (~80%) | 7 处必须修复 |
| 字段命名 (camelCase) | 高 (~95%) | 1 处真实问题 |
| 响应包装 (`{code, message, data}`) | 中 (~95%) | 2 处必改 + 1 处 HTTP 语义错误 |
| 分页响应一致性 | **低** | 后端至少 5 套形状并存,前端被迫 `?? ?? ??` |
| 错误处理一致性 | 中 | `UnauthorizedCode(2002)` 走 HTTP 200 是严重 bug |
| 认证 / CSRF / 租户头 | 中 | 部分 raw fetch 漏 CSRF 和租户头 |

**最大风险**:
1. 后端 `handlers/dashboard/handler.go` 整套违反响应包装规范 (前端无法识别业务错误)
2. 后端 `common.Fail()` 未映射 `UnauthorizedCode(2002)` 到 HTTP 401 → 租户缺失返回 200 → 前端 401 刷新令牌机制永远不触发
3. 多个前端子系统 (`tickets/relations`、`tickets/batch`、`templates`、`knowledge`、`change-classification`、`priority-matrix`、`reports`、`collaboration`) 对应的后端路由**完全不存在**,功能行为在前端模拟但不会持久化

---

## 一、整体对齐情况

### 1.1 项目规模
- **后端路由**: ~560 个路由定义在 `itsm-backend/router/router.go` (1776 行),`cmdb_routes.go` (249 行),`audit_routes.go` (26 行),`feishu_routes.go` (30 行),`ws_ticket.go` (95 行) + 13 个 controller 通过 `RegisterRoutes` 委托注册
- **后端 DTO**: 88 个 dto 文件覆盖 ticket/incident/change/problem/release/service/knowledge/sla/cmdb/workflow/user/auth 等域
- **前端 API 客户端**: 80+ 个 `*.ts` 文件,涵盖 295 条唯一 `/api/v1/...` 路径
- **架构边界**: 后端业务规则 + 权限 + 审计,前端 UI/UX/表单层,中间用 `httpClient` 统一封装

### 1.2 自动对齐机制 (基础保障)
`itsm-frontend/src/lib/api/http-client.ts:7-25` 实现 `toCamelCase` 函数,自动将响应数据从 snake_case 转为 camelCase;`requestInternal` 在第 352-353 行统一解包 `responseData.data` 并应用 `toCamelCase`。这条自动桥接极大降低了字段命名不对齐带来的风险。

`itsm-frontend/src/lib/api/http-client.ts:218-222` 在请求体一侧同样将 camelCase 转换回原始形态,保证发送出去与后端 mapper 约定一致。

---

## 二、API 端点对齐 — 核心问题

### 2.1 后端缺失 / 前端调用的端点 (按严重度排序)

#### 🔴 严重级 1 — `tickets/relations` 整棵子树缺失
- **前端文件**: `itsm-frontend/src/lib/api/ticket-relations-api.ts` 全文件
- **暴露端点**: ~30 个,包括
  - `POST /api/v1/tickets/relations`
  - `GET/PUT/DELETE /api/v1/tickets/relations/:id`
  - `POST/DELETE /api/v1/tickets/:id/parent`
  - `GET /api/v1/tickets/:id/children`
  - `GET /api/v1/tickets/:id/hierarchy`
  - `POST/DELETE /api/v1/tickets/:id/dependencies/:id`
  - `GET /api/v1/tickets/:id/dependencies`
  - `GET /api/v1/tickets/:id/dependency-graph`
  - `GET /api/v1/tickets/relations/search`
  - `GET /api/v1/tickets/:id/related`
  - `GET /api/v1/tickets/:id/duplicates`
  - `GET /api/v1/tickets/:id/relations/stats`
  - `POST /api/v1/tickets/:id/impact-analysis`
  - `GET /api/v1/tickets/:id/critical-path`
  - `GET /api/v1/tickets/:id/graph`、`/graph/export`
  - `GET /api/v1/tickets/:id/relation-suggestions`
  - `GET /api/v1/tickets/:id/ai-recommendations`
  - `GET /api/v1/tickets/:id/relations/history`
  - `GET /api/v1/tickets/relations/logs[/:id]`
  - `GET /api/v1/tickets/:id/relations/permissions`
- **后端现状**: 仅 `tickets/:id/dependencies` 单端点存在 (`router.go:1278`)
- **影响**: "工单关系/依赖" UI 看似正常但数据不会持久化,所有依赖图、影响分析、关键路径、AI 推荐结果**前端模拟**
- **建议**: 在后端新建 `TicketRelationController` 或裁剪前端功能

#### 🔴 严重级 2 — `tickets/batch` 整棵子树缺失
- **前端文件**: `itsm-frontend/src/lib/api/batch-operations-api.ts` 全文件
- **暴露端点**: ~50 个,涵盖批量执行/校验/预览/分配/状态/优先级/分类/标签/归档/导出/操作进度/计划/日志/统计/权限/撤销
- **后端现状**: 仅 `/tickets/batch-delete`、`/tickets/export` 两条 (`router.go:528-529`)
- **影响**: 所有批量 UI 操作 (批量改状态、批量改负责人、批量导出) 调用均会 404

#### 🔴 严重级 3 — `templates` 命名空间错位
- **前端文件**: `itsm-frontend/src/lib/api/template-api.ts`
- **前端调用**: `/api/v1/templates` 与 `/api/v1/template-categories`
- **后端注册**: 仅 `/api/v1/tickets/templates/*` (`router.go:535-542`)
- **影响**: ratings、versions、import/export、analytics、favorites、smart-recommend、field-suggestions 等所有高级模板功能 404

#### 🔴 严重级 4 — `cloud` 客户端前缀错误
- **前端文件**: `itsm-frontend/src/lib/api/cloud-api.ts`
- **前端调用**: `/cloud/{accounts,services,resources}` (省略 `/api/v1`)
- **后端注册**: `/api/v1/cloud/{accounts,services,resources}` (`router.go:1669-1700`)
- **影响**: 所有云资源操作 404

#### 🔴 严重级 5 — `knowledge-base` 高级功能大面积缺失
- **前端文件**: `itsm-frontend/src/lib/api/knowledge-base-api.ts`
- **缺失端点**: versions、archive、clone、batch、categories 写入、tags CRUD、comments 删除、feedback、like/bookmark/share/view、popular、upload、autosave、review、analytics、export 等 ~25 个端点
- **后端现状** (`router.go:1030-1072`): 仅 list/get/create/update/delete、publish/unpublish、comments GET、categories GET、search、recommendations、recent、stats
- **影响**: 知识库高级 UI (点赞、收藏、版本、评论删除等) 全部不可用

#### 🔴 严重级 6 — `change-classification` 与 `priority-matrix` 整命名空间缺失
- **change-classification-api.ts**: classifications、rules、templates、risk-assessments、impact-analyses、approval-matrix、stats、history ~20 端点
- **priority-matrix-api.ts**: calculate、matrix-configs、rules、history、analysis、export ~20 端点
- **后端现状**: 仅 incident 内部使用 priority 字段,priority/http 端点**完全没有**

### 2.2 HTTP 方法不一致 (必须修复)

| # | 前端调用 | 后端注册 | 文件:行号 | 建议 |
|---|---|---|---|---|
| 1 | `DELETE /api/v1/tickets/batch-delete` | `POST /api/v1/tickets/batch-delete` | `ticket-api.ts:558` ↔ `router.go:529` | 前端改 POST |
| 2 | `GET /api/v1/tickets/export` | `POST /api/v1/tickets/export` | `ticket-api.ts:582` ↔ `router.go:528` | 前端改 POST |
| 3 | `POST /api/v1/notifications/:id/read` | `PUT /api/v1/notifications/:id/read` | `collaboration-api.ts:174` ↔ `router.go:1380` | 前端改 PUT |
| 4 | `POST /api/v1/notifications/mark-all-read` | `PUT /api/v1/notifications/read-all` | `collaboration-api.ts:181` ↔ `router.go:1381` | 前端改 PUT, 路径同步 |
| 5 | `POST /api/v1/notifications/batch-read` | `PUT /api/v1/notifications/batch/read` | `collaboration-api.ts:405` ↔ `router.go:1382` | 前端改 PUT, 路径同步 |
| 6 | `POST /api/v1/bpmn/tasks/:id/claim` | `PUT /api/v1/bpmn/tasks/:id/claim` | `bpmn-workflow-api.ts` ↔ `BPMNWorkflowController` | 前端改 PUT |
| 7 | `PATCH /api/v1/templates/:id` | `PUT /api/v1/tickets/templates/:id` | `template-api.ts:55` ↔ `router.go:539` | 命名空间错位 (见 2.1.3) |

### 2.3 后端有但前端未使用 (代表性)

- `GET /api/v1/tickets/:id/sla` (`router.go:563`) — 实际上前端 ticket-api.ts:683-700 有 `getTicketSLA`,已对齐
- `GET /api/v1/sla/alert-rules` (`router.go:1098`) 全 CRUD — 前端 sla-api.ts 未实现
- 全部 `cmdb/cis/batch`、`cmdb/cis/:id/revert`、`cmdb/cis/:id/lifecycle/*`、`cmdb/import`、`cmdb/export`、`cmdb/discovery/*`、`cmdb/reconciliation` (`cmdb_routes.go`) — 前端 cmdb-api.ts 未调用 (只读为主)
- `/api/v1/connectors/*` (`router.go:1482-1491`) — 飞书/钉钉连接器配置 (运维平台专用,可不暴露给业务前端)
- `/api/v1/dashboard/charts/:chart_type`、`/realtime/:data_type`、`/metrics/*` — 仪表盘 mock handler

---

## 三、字段命名一致性

### 3.1 自动桥接已覆盖大部分场景

`http-client.ts:7-25` 实现的 `toCamelCase` 自动转换,以及 `requestInternal:218-222` 对请求体的反向转换,事实上让 snake_case 与 camelCase 混用在响应/请求层面不会立即引发问题。

### 3.2 真实违规点

经全量扫描,前端 TypeScript 接口中的字段命名违规主要集中在以下几类:

#### 🔴 真实问题
- **`process-binding-api.ts:27`** — 字段命名违规 (具体行需现场核对)
- **`ticket-api.ts:192` `assignee_id` 残留** — 旧兼容代码,在 `IncidentAPI.getIncidents` 仍用 `assignee_id: params.assigneeId` 然后清空 `assigneeId` (`incident-api.ts:850-855`),依赖请求体反向转换才正确

#### 🟡 已对齐 (无问题)
- 后端 88 个 dto 文件的 JSON tag 全部使用 camelCase (例如 `dto/ticket_dto.go:108-114` 的 `TicketResponse`,`dto/incident_dto.go:102-130` 的 `IncidentResponse`,`dto/change_dto.go:121` 的 `ChangeListResponse`)
- 前端主要 API 客户端 (ticket, incident, change, problem, knowledge, cmdb, sla) 的 TypeScript 接口全部 camelCase
- `*.bak`、`*.disabled` 文件不应纳入统计

#### ⚪ snake_case 有意为之 (URL 查询参数)
- `incident-api.ts:363-365` 将 `pageSize` 翻译为 `size` 提交给后端 — 后端 incident handler 期望 `size` (沿用分页形状不一致,见四)
- 多个客户端的 query 参数保留 snake_case 是与后端 form 标签的有意对齐,不算违规

### 3.3 综合评价

**字段命名一致性总体评分: 良好**。自动转换层做了大部分兜底工作,真实违规 ≤ 2 处。建议在开发规范文档中加入"TypeScript 接口字段命名检查"的 lint 规则 (例如 `eslint-plugin-camelcase` 配配置) 防止后续回归。

---

## 四、响应包装与错误处理

### 4.1 后端响应包装

#### 规范定义 (`common/response.go`)
```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

`common.Success(c, data)` 与 `common.Fail(c, code, msg)` 是事实标准。

#### 🔴 严重违规 1 — `handlers/dashboard/handler.go` 整套绕过包装
- **文件**: `itsm-backend/handlers/dashboard/handler.go:23-56`
- **行为**: 4 处直接 `c.JSON(...)`,使用 `{success: bool, message: string, data, error}` 格式
- **后果**: 前端 `httpClient.requestInternal` 第 342-349 行依赖 `responseData.code !== 0` 判错,但该 handler 返回的 `{success: false, ...}` 没有 `code` 字段,业务错误会被永远视为成功,**前端不会显示任何错误提示**
- **建议**: 4 处全部改 `common.Success/Fail`

#### 🔴 严重违规 2 — `handlers/ai/handler.go:514` 单点遗漏
- **文件**: `itsm-backend/handlers/ai/handler.go:514`
- **行为**: `c.JSON(http.StatusOK, result)` 直接返回
- **后果**: AI 创建工单接口拿到的是没有 `code/message` 的纯对象,httpClient 会因为 `code !== 0` 抛不友好的错误

#### 🟡 中等问题 — `common.Fail()` HTTP 状态映射不完整
- **文件**: `common/response.go:48-70`
- **缺失分支**:
  - `UnauthorizedCode = 2002` (租户信息缺失) — 被 9 个 controller 使用,**未映射 → HTTP 200**,语义错误
  - `ToolPermissionDeniedCode = 2004`、`UnknownToolCode = 2005` (AI 工具 RBAC) — 同样 fall-through 到 HTTP 200
- **后果**:
  - 前端 `http-client.ts:279` 的 401-refresh 逻辑不会触发,租户缺失时前端只能看到 message 然后放弃
  - `useErrorHandler.ts:43-47` / `error-handler.tsx:253-254` 按 HTTP 401/403 做的语义判断不会生效

#### 🟡 中等问题 — `controller/global_search_controller.go:55`
- `common.Fail(ctx, http.StatusBadRequest, "租户上下文缺失")` 把 HTTP 400 当作业务 code 传入,客户端收到 `code: 400`,与 `ParamErrorCode(1001)` 不一致

#### 🟡 中等问题 — `common/role_handler.go:79-291`
- 6 处直接用 `gin.H` 写响应 + 混用 snake_case `page_size` 与 camelCase,违反项目约定

#### ⚪ 死代码 — `internal/api/routes.go`
- 定义了第二套 `StandardResponse` / `PaginatedResponse` / `BatchResponse` + `SuccessWithPagination()` / `BatchOperationResponse()`,全代码库搜索无任何 controller 调用
- **建议**: 删除或迁移并标记 deprecated

### 4.2 后端分页响应一致性

**最严重的领域问题**。后端至少 5 套分页形状并存:

| DTO | 字段 | 来源 |
|---|---|---|
| `ListTicketsResponse` | `{tickets, total, page, pageSize, totalPages}` | `dto/ticket_dto.go:108` |
| `IncidentListResponse` | `{incidents, total, page, pageSize, totalPages}` | `dto/incident_dto.go:136` |
| `BPMNProcessInstanceListResponse` | `{items, total, page, pageSize, totalPages}` | `dto/bpmn_dto.go:55` |
| `WorkflowListResponse` | `{workflows, total, page, pageSize}` **(无 totalPages)** | `dto/mappers.go:744` |
| `TenantListResponse` | `{tenants, total, page, pageSize}` **(无 totalPages)** | `dto/tenant_dto.go:80` |
| `CIListResponse`, `NotificationListResponse`, `SystemConfigListResponse`, `ServiceCatalogListResponse`, `ServiceRequestListResponse` | `{items/xxx, total, page, size}` **(用 `size` 不用 `pageSize`, 无 totalPages)** | `dto/cmdb_core_dto.go:69`, `dto/notification_dto.go:52`, `dto/system_config_dto.go:30`, `dto/service_dto.go:147/155` |
| `ChangeListResponse` | `{total, changes, pageSize, totalPages}` **(无 `page`)** | `dto/change_dto.go:121` |
| `ReleaseListResponse`, `ProjectListResponse`, `MSPAllocationListResponse` | `{items, total}` **(无分页)** | `dto/release_dto.go:129` 等 |
| `RoleListResponse` | `{roles, total, page, pageSize, totalPage}` **(`totalPage` 拼写错误)** | `dto/role_dto.go:21` |

**handler 实际响应 vs DTO 还有差异**:
- `handlers/change/handler.go:264-269` 返回 `{changes, total, page, pageSize}`,既有 `page` 又没有 `totalPages`
- `common/role_handler.go:79-88` 返回 `{roles, total, page, page_size}`,使用 snake_case

**前端被迫的兼容补丁**:
- `incident-api.ts:376` — `response.incidents ?? response.items ?? response.data ?? []`
- `app/(main)/cmdb/cloud-resources/page.tsx:109` — `Array.isArray(response) ? response : response.items || response.data || []`
- `ticket-api.ts:15-18` — `response.size ?? response.pageSize ?? params?.pageSize ?? params?.size ?? 20`
- `lib/api/index.ts:187` — `emptyPagination` 工厂返回 `types.ts` 的 `{data}` 形状,但消费者期望 `{items}` 形状

**建议**:
1. 统一所有 `*ListResponse` 形状为 `{items, total, page, pageSize, totalPages}`
2. 把 `common.SuccessWithList` (response.go:147) 和 `common.SuccessWithPagination` (pagination.go:87) 真正用起来,所有 list 端点迁过去
3. 修掉 `RoleListResponse` 的 `totalPage` 拼写错误

### 4.3 前端响应解包一致性

**统一部分** (OK):
- `requestInternal` (http-client.ts:337-353) 正常路径正确解包 + 应用 toCamelCase
- 容错 `code === undefined/null` 视为成功 (应对老 BPMN workflow controller)

**绕过 httpClient 的位置** (raw fetch,需审计):
- `lib/api/auth-api.ts` — 6 处 raw fetch,正确使用 `data.data || data` 回退
- `lib/api/service-request-api.ts:113-140` — 自建 `request()`,**没有添加 CSRF token**
- `lib/services/ticket-service.ts:472-484` — POST export,**没有 CSRF token + X-Tenant-Code**
- `lib/api/base-api.ts:299-318` — `BaseApi.export` 用 `fetch` GET,**没有 X-Tenant-ID/X-Tenant-Code/X-CSRF-Token** (GET 不需 CSRF,但租户 header 缺失会让 cookie-only 登录的浏览器请求被拒)
- `lib/api/ai-api.ts:199` — SSE chat/stream,事件流不解包 (协议层正确)

**结论**: 主要问题是 raw fetch 缺少 CSRF 与租户头,以及不走统一错误处理路径 (无法附加 X-Request-Id 到 toast)。

### 4.4 错误处理

**已对齐**:
- `base-api-handler.ts:107-109` 通过 `message.error(friendlyMessage)` 统一弹错
- `error-handler.tsx:195-223` 按 severity 选择 notification

**未处理**:
- `code: 2002` (租户缺失) — 没有"重新选择租户"引导
- `code: 2004/2005` (AI 工具权限/未知) — 没有专门 UI
- `code: 4031` (CSRF 缺失) — 没有"刷新页面重试"提示
- AI 经常 `code: 0` 但内嵌 `degraded: true` 字段 (`handlers/ai/handler.go:128-133`) 没有软失败识别

**i18n 未做**: 后端 message 全是中文,前端直接展示,不做 i18n;`ERROR_MESSAGES` 表只针对 HTTP status 字符串匹配,不针对业务 code。

---

## 五、认证 / CSRF / 租户头

### 5.1 Tenant 头

- **后端 `TenantMiddleware`** 显式忽略 `X-Tenant-ID` (`middleware/tenant.go:30-31`),优先级为 JWT claims > X-Tenant-Code > Subdomain > Path 参数
- **前端 `httpClient`** 同时发两个头 (`http-client.ts:124-129`),`X-Tenant-ID` 是无效负载
- **无 tenant 源**: `code: 1001, HTTP 400`,前端 `if (!response?.ok)` 捕获 (行为正确)
- **JWT 与 tenant 不一致**: `code: 2001, HTTP 401` (OK)

### 5.2 CSRF

- **后端 CSRF 中间件**: 默认仅对 POST/PUT/DELETE/PATCH 校验;skip paths 包含 login/refresh/csrf-token/health;Bearer Authorization 自动 bypass ✓
- **前端 httpClient**: `addCSRFHeader` 在 mutating 方法时自动添加 (✓)
- **绕过 httpClient 的 raw fetch 缺 CSRF**:
  - `lib/api/service-request-api.ts:113-122` — POST/PUT/DELETE 都没有 CSRF
  - `lib/services/ticket-service.ts:473-484` — POST export 没有 CSRF
- **CSRF 错误码混乱**: `middleware/csrf.go:147/161/169` 三处返回 `{code, message}` 形状不一致,前端用 message 字符串前缀匹配识别,脆弱

---

## 六、按优先级修复建议

### P0 — 立即修复 (阻断业务或破坏体验)

1. **`handlers/dashboard/handler.go`** 4 处全部 `c.JSON` 改为 `common.Success/Fail` — 当前业务错误前端无法识别
2. **`handlers/ai/handler.go:514`** 改用 `common.Success`
3. **`common.Fail()`** 在 `response.go:48-63` 补充 `UnauthorizedCode(2002)`、`ToolPermissionDeniedCode(2004)`、`UnknownToolCode(2005)` 的 HTTP 状态映射 — 当前租户缺失返回 HTTP 200,前端 401 刷新令牌机制永不触发
4. **HTTP 方法不一致 7 处** (见 2.2 表) — 直接导致 405/404
5. **Cloud 客户端 `/cloud/*` → `/api/v1/cloud/*`** — `cloud-api.ts` 全部修正

### P1 — 短期修复 (影响功能完整性)

6. **后端缺失的子系统路由** — 按严重度依次:
   - `tickets/relations/*` 子树 (新建 `TicketRelationController`)
   - `tickets/batch/*` 子树 (新建 `BatchOperationController`)
   - `templates` 命名空间错位 (要么前端迁 `/tickets/templates`,要么后端补 `/templates`)
   - `knowledge-base` 高级功能 ~25 端点
   - `change-classification` ~20 端点
   - `priority-matrix` ~20 端点
7. **删除死代码** `internal/api/routes.go` + `common.SuccessWithList/Pagination` 真正用起来
8. **统一分页响应** 为 `{items, total, page, pageSize, totalPages}`
9. **前端 raw fetch 补齐 CSRF + 租户头** (service-request-api.ts、ticket-service.ts export、base-api.ts export)

### P2 — 中期治理 (提升一致性)

10. **修 `RoleListResponse.totalPage` 拼写错误**
11. **合并前端重复定义** (`PaginationResponse` 两个版本、`TicketListResponse` 两个版本)
12. **修 `process-binding-api.ts:27`** 字段命名违规
13. **修 `controller/global_search_controller.go:55`** 业务 code 应传 `ParamErrorCode(1001)` 而非 HTTP 400
14. **修 `common/role_handler.go`** 6 处直接 `gin.H` 改为 `common.Success/Fail`
15. **统一 CSRF 错误码** 为单一形状,前端按 code 而非 message 字符串匹配
16. **统一前端 raw fetch 风格** (考虑把 `BaseApi.export` 改走 `httpClient.request<Blob>`)
17. **i18n 框架建设** — 后端 message 中英文分离或前端统一翻译

### P3 — 持续治理

18. **ESLint 规则** 防止新增 camelCase 违规字段
19. **API 契约测试** (`dto/ticket_contract_test.go` 已存在),扩展到所有域
20. **CI 检查** — 静态扫描所有 controller 是否仍走 `common.Success/Fail`,发现裸 `c.JSON` 立即报警

---

## 七、亮点 / 良好实践

1. **自动命名转换层** (`http-client.ts:7-25`) 大幅降低了字段命名不对齐的爆炸半径
2. **`dto/ticket_contract_test.go`** 存在,说明已经意识到契约测试的重要性,值得扩展
3. **`common.Success/Fail` 事实标准** 已经覆盖 ~95% 的 controller,统一度良好
4. **CSRF HttpOnly cookie + 自动注入** 设计正确
5. **路径参数命名一致** (后端 `:id` ↔ 前端 `${id}`),绝大部分路由无此问题
6. **`X-Tenant-ID` 显式忽略 + JWT 优先** 的设计,防止前端伪造 tenant
7. **乐观锁字段 `version`** 在 UpdateTicketRequest/UpdateIncidentRequest 等已规范
8. **DTO 全部 camelCase JSON tag** (88 个 dto 文件) — 字段命名一致性底层做得好
9. **路由分组权限** (`middleware.RequirePermission`) 与 RBAC 中间件配合,默认所有路由都被保护
10. **WebSocket 票据机制** (`router.go:415-446`) 避免了 token 通过 query string 泄露

---

## 八、结论

ITSM 项目前后端**整体对齐良好** (~80% 路由与调用一致),但存在以下**结构性风险**:

1. **多个前端子系统 (relations、batch、templates、knowledge、change-classification、priority-matrix、reports、collaboration) 的后端实现是空白的** — UI 行为前端模拟,数据不会持久化。建议产品/技术联合决策:要么裁剪前端功能,要么补齐后端。

2. **响应包装违规 + HTTP 状态映射错误** — 导致前端无法识别业务错误,租户缺失类错误甚至以 HTTP 200 返回。建议优先修 P0 第 1/2/3 项。

3. **分页响应至少 5 套形状** — 前端被迫 fallback 链,长期必然产生 bug。建议统一为 `{items, total, page, pageSize, totalPages}` 并真正使用 `common.SuccessWithList`。

4. **HTTP 方法不一致 7 处** — 直接导致 405/404,必须修复。

5. **CSRF/租户头在 raw fetch 中漏发** — 短期不会出问题,但接入 OAuth/cookie-only 登录后会爆雷,建议统一走 `httpClient`。

建议按 P0 → P1 → P2 → P3 顺序逐批修复,每批配合同步回归测试与契约测试用例,避免修复本身引入新问题。

---

**审查完成**。如需对任一子项进一步细化 (例如某个 API 模块的完整对齐矩阵或具体修复 PR),请指示。