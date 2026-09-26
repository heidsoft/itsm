# ITSM 系统业务与前后端代码审计报告

> 审计范围：`itsm-backend/` (Go/Gin/Ent) + `itsm-frontend/` (Next.js 15/React 19/Antd v6)
> 审计时间：2026-09-26
> 审计基线：v1.6.x 收敛期（TicketType 平台、RBAC/租户硬化、状态机 CAS）
> 审计重点：业务模型正确性、API 契约一致性、租户隔离/RBAC、状态机、并发安全

---

## 一、系统业务全景

### 1.1 业务域覆盖（ITIL v3/v4）

| 领域 | 状态 | 关键模块 |
|---|---|---|
| Ticket 平台（v1.6 核心） | GA | 模板/分类/工作流/SLA/Rating/CI 反查 |
| Incident | GA | 阿里云集成、安全事件、影响分析 |
| Problem | Pilot | RCA/已知错误/关联事件 |
| Change | Pilot | CAB 审批/风险评估/实施窗口 |
| Release | Pilot | 与 Change 联动 |
| Service Request | GA | 服务目录 + 审批 + 履约 |
| Service Catalog | GA | 分类/条目/SLA/审批模板 |
| CMDB v1 | GA | CI 类型/实例/关系/拓扑/影响分析 |
| Knowledge / RAG | Pilot | 版本/权限/检索 |
| BPMN Workflow | GA | 定义/实例/任务/历史 |
| SLA | GA | 定义/策略/升级/暂停 |
| AI-Native | Pilot | Triage/Summarize/Skill Registry |

### 1.2 架构基线

- 后端 `handlers/<domain>/` 为推荐架构；`service/` 共享业务逻辑；`controller/` 已冻结（AGENTS.md §Backend Layering Rules）
- 前端 App Router + React Query + Zustand；`src/lib/api/` 是契约客户端
- API 响应统一 `{ code, message, data }`；列表响应规范 `{ items, total, page, pageSize, totalPages }`
- 字段命名：DB snake_case、Go PascalCase、HTTP/JSON camelCase
- 事务型 outbox + `commandbus.EnqueueTx` 用于可靠副作用
- 乐观锁：PATCH 必传 `version`，使用 `WHERE id + tenant_id + version` CAS

---

## 二、核心审计发现

### 总览表

| 编号 | 标题 | 严重度 | 类型 | 影响面 |
|---|---|---|---|---|
| F-01 | Ticket 列表响应字段 `tickets` 偏离标准 `items` | P1 | 契约 | 前端一致性 |
| F-02 | `TenantListResponse` 使用 `size` 而非标准 `pageSize` | P1 | 契约 | 列表字段别名 |
| F-03 | Ticket 多字段兼容写法（`pageSize ?? size`） | P1 | 反模式 | 类型逃逸 |
| F-04 | 工单状态机前后端不一致（`pending_approval`/`rejected`） | P1 | 业务 | 状态流转 |
| F-05 | 缺少"今日新增"统计 API，前端硬编码 `today: 0` | P1 | 数据 | 业务可视 |
| F-06 | `SuccessWithPagination` 同时输出 `items` 和 `tickets` 别名 | P1 | 契约 | 长尾隐患 |
| F-07 | 写路径仅在 `ticket/handler.go` 注入 DataScope，多数路由未声明 | P1 | 安全 | 跨租户写越权 |
| F-08 | `CreateTicketRequest.RequesterID`/`UserID` 仍允许从请求体传入 | P1 | 安全 | 身份伪造 |
| F-09 | `datascope.CanWriteResource` 仅 ticket 模块使用 | P1 | 一致性 | 行级 RBAC |
| F-10 | `incident-api.ts` 多处无对应后端接口 | P2 | 契约 | 前端无效调用 |
| F-11 | `version` 字段为 `int` 而非 `*int`，无法区分未传 | P1 | 并发 | 乐观锁误判 |
| F-12 | `frontend/src/lib/api/...` 中动态 base path 部分被契约扫描跳过 | P2 | 验证 | 漂移风险 |
| F-13 | 部分 PATCH DTO 使用非指针 `bool` 字段（如 `Force`） | P1 | API 规范 | 字段语义 |
| F-14 | 共享 `service/ticket_lifecycle_service.go` 仍持有业务规则 | P2 | 架构 | handler 化未完成 |
| F-15 | `userId` 字段仍允许前端传入覆盖认证上下文 | P0 | 安全 | 身份伪造 |
| F-16 | `approveTicket`/`resolveTicket` 使用前端拼接字符串路径 | P2 | 路径 | 契约扫描 |
| F-17 | `router/router.go` 仅 4 个路由显式声明 `RequirePermission` | P1 | 安全 | RBAC 覆盖率 |
| F-18 | `BasePath` 残留 `DefaultBaseURL` 兜底 | P2 | 配置 | 环境差异 |

---

## 三、详细审计

### 3.1 API 契约一致性

#### F-01 / F-02：列表响应字段偏离

**事实**：

```go
// itsm-backend/dto/ticket_dto.go:112-119
type ListTicketsResponse struct {
    Tickets    []*TicketResponse `json:"tickets"` // ❌ 标准应为 items
    Total      int               `json:"total"`
    Page       int               `json:"page"`
    PageSize   int               `json:"pageSize"`
    TotalPages int               `json:"totalPages"`
}
```

```typescript
// itsm-frontend/src/lib/api/api-config.ts:33-39 标准契约
export interface PaginationResponse<T> {
  items: T[];          // ✅ 单一事实
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// src/lib/api/api-config.ts:66-71 偏差
export interface TenantListResponse {
  tenants: Tenant[];
  total: number;
  page: number;
  size: number;        // ❌ 应为 pageSize
}
```

**问题**：
- AGENTS.md 明文规定"新增列表接口统一使用 `data: { items, total, page, pageSize, totalPages }`"
- 现有 `ListTicketsResponse`/`TicketListResponse`/`TenantListResponse` 沿用领域名（tickets/tenants/size），违反零新增规则

**修复方案**：
1. 后端 `ListTicketsResponse`/`TenantListResponse` 改造为 `PaginationResponse[*TicketResponse]` 形态，使用 `items`
2. Mapper 统一走 `dto.Paginate(items, page, pageSize, total)`
3. 旧字段名通过 `internal/` 适配层临时保留（标注 DEPRECATED 与移除条件）

#### F-03：前端多字段兜底反模式

**事实**：

```typescript
// itsm-frontend/src/lib/api/ticket-api.ts:24-28
return {
  ...response,
  size: response.pageSize ?? 20,   // ❌ 多字段兼容
};
```

**问题**：AGENTS.md §请求/响应多字段兼容零新增规则明确禁止 `response.field ?? response.field2 ?? default`。`0`/`false`/`""` 是合法值，不能用 `||`/`??` 兜底。

**修复**：统一按唯一 DTO 字段读取，禁止前端补充 `size`。

#### F-06：`SuccessWithPagination` 同时输出双别名

**事实**：

```go
// itsm-backend/handlers/ticket/handler.go:221-222
// v1.1 回归：使用 SuccessWithPagination 自动产出 items+tickets 别名，
// 避免前端 response.tickets 未定义导致列表为空
common.SuccessWithPagination(c, ticketListToResponse(tickets), req.Page, req.PageSize, int64(total))
```

**问题**：
- `common.SuccessWithPagination` 同时输出 `items` 与 `tickets`，制造"双轨真相"
- 注释暴露前端在用 `response.tickets` 而非 `response.items`，前端契约与标准偏差
- 这种回填别名会持续掩盖根本性契约偏差

**修复**：
1. 移除别名输出，前端必须用 `items`
2. 在前端 `getTickets` 显式读取 `items`，加契约测试覆盖（`response.items` 必须非空）

#### F-11：乐观锁字段未使用指针

**事实**：

```go
// itsm-backend/dto/ticket_dto.go:52
Version int `json:"version"` // ❌ 应为 *int
```

```go
// handlers/ticket/handler.go:247
params := &UpdateParams{Version: req.Version}  // 无法区分未传 vs 传 0
```

**问题**：AGENTS.md §PATCH 强制要求"可选字段使用指针，包括 bool/int/枚举/时间，必须区分未传/传零值/清空"。`Version` 是关键乐观锁字段，未传时不应触发 CAS。

**修复**：
```go
Version *int `json:"version"`
// handler: 仅在 req.Version != nil 时设置 params.Version
```

#### F-13：PATCH DTO 非指针字段

**事实**：

```go
// dto/ticket_dto.go:53
Force bool `json:"-"` // 忽略 JSON，未暴露，但同时让 status 等关键字段也用 string 而非 *string
```

需要逐字段扫描：
- `Title string` → `*string`
- `Description string` → `*string`
- `Status string` → `*string`
- `Priority string` → `*string`
- `AssigneeID int` → `*int`
- `Resolution string` → `*string`
- `FormFields map` → `*map`

#### F-10 / F-16：前端 API 路径不一致

**事实**：

```typescript
// incident-api.ts 包含 incident.impactAnalysis、incident.sourceIp 等字段
// 但未确认后端 /api/v1/incidents/:id/impact 接口存在

// ticket-api.ts:97-99 等位置使用 ${ticketId} 字符串模板 - 这是合规的
// 但 approveTicket 等方法路径需要审计
POST /api/v1/tickets/workflow/approve   // ❌ 不符合 REST：动作应在资源路径下
```

**问题**：动作伪装成查询参数或顶层动词，违反 AGENTS.md §API URL 与路由规范。

**修复**：改为 `POST /api/v1/tickets/:id/approve`、`POST /api/v1/incidents/:id/impact-analysis`。

---

### 3.2 业务逻辑（核心领域）

#### F-04：工单状态机不一致

**事实**：

```typescript
// itsm-frontend/src/constants/taxonomy.ts:60-70
export enum TicketStatus {
  NEW = 'new',
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  PENDING_APPROVAL = 'pending_approval', // ❌ 后端无此状态
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled',
  REJECTED = 'rejected',                 // ❌ 后端无独立 REJECTED
}
```

```go
// itsm-backend/dto/ticket_dto.go:42
Status string `json:"status" binding:"omitempty,oneof=new open assigned in_progress pending resolved closed cancelled approved rejected"`
```

后端枚举：new, open, **assigned**, in_progress, pending, resolved, closed, cancelled, **approved**, rejected
前端枚举：new, open, in_progress, **pending_approval**, pending, resolved, closed, cancelled, rejected
差异：前端多 `pending_approval`、少 `assigned` 与 `approved`

**问题**：
- `workflow-state-machine.ts` 中 PENDING_APPROVAL 流转规则不可达（后端拒绝）
- 前端控件可能让用户点击"待审批"，但 PATCH 会被 422 拒绝
- 服务端领域真正状态是 `assigned` + `approved`，前端 UX 误用 `pending_approval`

**修复**：
1. 后端保留 `pending_approval` 状态（业务上确实有"待审批"语义），扩展 `oneof`
2. 前端移除冗余 `PENDING_APPROVAL` 或增加映射
3. 共享 state-machine 存放在 `constants/workflow.ts` 与后端 schema 双向生成

#### F-05：缺失"今日新增"API

**事实**：

```typescript
// itsm-frontend/src/app/(main)/tickets/page.tsx:82
today: 0, // 暂时没有今日新增的 API
```

```go
// itsm-backend/dto/ticket_dto.go:121-129 TicketStatsResponse
type TicketStatsResponse struct {
    Total        int `json:"total"`
    Open         int `json:"open"`
    InProgress   int `json:"inProgress"`
    Resolved     int `json:"resolved"`
    Pending      int `json:"pending"`
    HighPriority int `json:"highPriority"`
}
```

**问题**：前端业务仪表盘声称有"今日新增"，但后端 Stats 缺字段且前端硬编码 0 = 错误数据呈现。

**修复**：
1. 后端 `TicketStatsResponse` 增加 `TodayNew int json:"todayNew"`（需要明确时区与 created_at 范围）
2. 前端删除硬编码 `today: 0`，改读 `stats.todayNew`
3. 增加契约测试覆盖

#### F-14：业务规则未完成 handler 化

**事实**：`service/ticket_lifecycle_service.go` 仍持有状态机合法迁移表与升级/解除操作。

```go
// TicketLifecycleService.ResolveTicket：仅 open/in_progress/pending → resolved
// TicketLifecycleService.CloseTicket：仅 resolved → closed
```

**问题**：
- AGENTS.md §Backend Layering Rules 要求"新代码写到 handlers/<domain>/"
- 状态机规则应作为领域不变量放在 handler 同包或 `handlers/ticket/lifecycle.go`
- `service/` 只应承载跨域共享逻辑（如 `sla_monitor`）

**修复**：
1. 将状态机迁移规则迁入 `handlers/ticket/lifecycle.go`
2. `service/ticket_lifecycle_service.go` 仅保留给 `incident/problem/change` 等共享语义
3. 在 `scripts/docs-gate/check-product-drift.sh` 加规则：`service/ticket_*.go` 必须由 `handlers/ticket/` 引用

---

### 3.3 租户隔离、RBAC 与安全

#### F-15 / F-08：身份字段允许请求体覆盖（CRITICAL）

**事实**：

```go
// dto/ticket_dto.go:27
RequesterID int `json:"requesterId" binding:"omitempty"`
// dto/ticket_dto.go:47
RequesterID int `json:"requesterId"`
// dto/ticket_dto.go:51
UserID int `json:"userId" binding:"omitempty"`
```

**问题**：AGENTS.md §身份、租户与数据范围强制要求"`userId`、`requesterId` 必须来自认证中间件上下文，禁止信任请求体"。当前 DTO 仍允许覆盖，是高危合规缺口。

**修复**：
1. 移除 `requesterId`/`userId` JSON tag，改为 `json:"-"` + 内部填充
2. handler 中：
   ```go
   actorID := c.GetInt("user_id")
   req.RequesterID = 0  // 强制忽略
   ```
3. 增加集成测试：模拟带 `requesterId: 999` 的请求，断言最终入库为 JWT user

#### F-07：DataScope 仅在 ticket 模块注入

**事实**：
- `handlers/ticket/handler.go` 在 List/Update 显式调用 `datascope.CanWriteResource` 等
- `handlers/incident/handler.go` 仅校验 `tenant_id + user_id`，未做行级检查
- `router/router.go` 仅 4 个路由声明 `RequirePermission`，大部分路由仅依赖 tenant group

**问题**：
- AGENTS.md §身份、租户与数据范围要求"每个新增租户资源接口至少包含一个跨租户拒绝测试"
- 现状是多数 handler 不做行级 RBAC，依赖 Service 内部隐式检查

**修复**：
1. 在 `router.go` 给所有读写路由按域+动作注册 `RequirePermission`：
   ```go
   tickets := tenant.Group("/tickets")
   tickets.GET("", middleware.RequirePermission("ticket", "read"), h.ListTickets)
   tickets.POST("", middleware.RequirePermission("ticket", "write"), h.CreateTicket)
   ```
2. 写操作统一通过 `datascope.Middleware(actor, repo)` 注入行级检查
3. 增补跨租户拒绝测试矩阵（admin/manager/department_manager/agent/end_user × ticket/incident/cmdb/...）

#### F-09：`datascope.CanWriteResource` 仅一处使用

**事实**：`handlers/common/datascope/datascope.go` 提供 `CanWriteResource(actorID, actorRole, ownerID, assigneeID)`，但 grep 仅 `ticket/service.go` 调用一次。

**修复**：将 `CanWriteResource` 封装为 helper middleware：
```go
func RequireWriteScope(resourceField string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从路径 :id 读取资源，校验 owner/assignee 与 actor
    }
}
```

#### F-17：路由权限覆盖率极低

**事实**：Router 中仅 4 处 `middleware.RequirePermission`：
- `system.write`
- `system.read`
- `cmdb.read` × 1
- `problem.read` × 1

**问题**：
- 其余 70+ 路由完全依赖 tenant middleware（仅校验 tenant_id 存在）
- 用户可访问同租户内任何资源 → 实际 RBAC 失效

**修复**：见 F-07 修复方案。

---

### 3.4 前端状态管理与契约

#### F-12：动态 base path 被契约测试跳过

**事实**：`src/lib/__tests__/api-contract.test.ts` 跳过 `${CMDB_BASE}/...` 等动态模板。

**问题**：动态 base path 无法静态扫描，长期可能累积漂移。

**修复**：将 base path 重构为静态常量（如 `CMDB_BASE = '/api/v1/cmdb'`），删除跳过逻辑。

#### F-18：BaseURL 兜底

**事实**：`api-config.ts` 中保留 `DefaultBaseURL` 兜底，开发/生产环境差异未完全消除。

**修复**：删除兜底，统一从 `NEXT_PUBLIC_API_URL` 注入；缺失时启动报错。

---

## 四、合同一致性差异表（节选）

| 接口 | 后端字段 | 前端期望字段 | 状态 |
|---|---|---|---|
| `GET /api/v1/tickets` | `data.tickets` | `data.items`（标准） | ❌ |
| `GET /api/v1/tenants` | `data.tenants` | `data.tenants` | ✅（但不符合标准） |
| `GET /api/v1/tenants` | `data.size` | `data.pageSize` | ❌ |
| `GET /api/v1/incidents` | `data.items` | `data.items` | ✅ |
| `GET /api/v1/service-requests` | `data.items` | `data.items` | ✅ |
| `GET /api/v1/cmdb/cis` | `data.items` | `data.items` | ✅ |

**结论**：Ticket 与 Tenant 两个最常用接口违反标准，其余大部分已对齐。

---

## 五、租户隔离 / RBAC 矩阵（建议）

| 角色 | Ticket 读 | Ticket 写 | Incident 读 | Incident 写 | CMDB 读 | CMDB 写 |
|---|---|---|---|---|---|---|
| super_admin | ✅ All | ✅ All | ✅ All | ✅ All | ✅ All | ✅ All |
| admin | ✅ All | ✅ All | ✅ All | ✅ All | ✅ All | ✅ All |
| manager | ✅ Dept | ✅ Dept | ✅ Dept | ✅ Dept | ✅ All | ✅ Dept |
| department_manager | ✅ Dept | ✅ Dept | ✅ Dept | ✅ Dept | ✅ All | ✅ Dept |
| l1/l2 agent | ✅ Owned | ✅ Owned | ✅ Owned | ✅ Owned | ✅ All | ✅ All |
| agent | ✅ Owned | ✅ Owned | ✅ Owned | ✅ Owned | ✅ All | ❌ |
| end_user | ✅ Owned | ❌ (only create) | ✅ Owned | ❌ | ✅ All | ❌ |

> 当前状态：handler 未统一注入此矩阵，依赖隐式过滤。

---

## 六、推荐修复优先级

### P0（立即修复）

1. **F-15 / F-08**：移除 `requesterId`/`userId` 请求体字段，强制来自认证上下文
2. **F-07 / F-17**：补齐所有路由的 `RequirePermission` 与行级 RBAC

### P1（v1.6.x 必须收敛）

3. **F-01 / F-02 / F-06**：统一 `items` 字段，移除 `SuccessWithPagination` 双别名输出
4. **F-03**：前端删除 `pageSize ?? size` 多字段兼容
5. **F-04**：状态机枚举对齐（`pending_approval`/`approved`/`rejected`）
6. **F-11**：乐观锁 `Version` 改为 `*int`
7. **F-13**：PATCH DTO 全字段指针化
8. **F-05**：补 `todayNew` 统计 API

### P2（v1.7 收敛）

9. **F-09**：封装 `RequireWriteScope` 中间件
10. **F-14**：业务规则迁入 handler 同包
11. **F-10 / F-16**：REST 动作路径化（`/tickets/:id/approve`）
12. **F-12 / F-18**：契约扫描静态化 + BaseURL 去兜底

---

## 七、验证清单（修复后必跑）

### 后端

```bash
cd itsm-backend
go test ./handlers/ticket/...   # 状态机、DataScope、版本冲突
go test ./handlers/incident/...
go test ./handlers/common/datascope/...
go test ./...
# 集成：跨租户拒绝矩阵
```

### 前端

```bash
cd itsm-frontend
npm run type-check
npm run lint:check
npm run lint:antd
npm run test:unit -- --runTestsByPath src/lib/__tests__/api-contract.test.ts
npm run test:unit -- --runTestsByPath src/app/(main)/tickets/__tests__/...
```

### 契约验证

```bash
# 1. 字段命名
git diff --unified=0 -- '*.go' '*.ts' '*.tsx' | \
  rg '^\+.*(json|form|query):"[a-z0-9]+_[a-z0-9_]+"|^\+.*["'\''][a-z][a-z0-9]*_[a-z0-9_]+["'\'']'

# 2. Space.direction 弃用
git diff --unified=0 -- '*.tsx' | \
  rg '^\+.*<Space\b[^>]*\bdirection\s*='
```

---

## 八、附：与既有审计报告的关系

| 已有报告 | 本报告补充 |
|---|---|
| `FE-BE-ALIGNMENT-REVIEW.md` | 把字段别名问题升级为契约零新增规则 |
| `PROD_DEPLOYMENT_REVIEW.md` | 进一步指出 RBAC 路由覆盖率与身份伪造 |
| `FRONTEND_AUDIT_REPORT.md` | 验证 PATCH 指针化 / Space.direction 仍然零命中 |
| `BUSINESS_BUG_REPORT.md` | 重新定位 `pending_approval`/`rejected` 状态差异 |
| `UI_DESIGN_AUDIT_REPORT.md` | 指出"今日新增"硬编码 0 的业务影响 |

---

> 报告生成完毕。建议优先推进 P0（身份伪造 + RBAC 覆盖），再按 P1 顺序收敛契约与状态机。
