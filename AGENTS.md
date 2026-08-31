# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project Overview

ITSM (IT Service Management) system with a Go/Gin backend and Next.js/TypeScript frontend. Features include:

- Ticket/Incident/Problem/Change management
- Service Catalog
- Knowledge Base with RAG
- BPMN Workflow engine
- SLA monitoring and escalation
- AI-powered triage and summarization

## Product Direction

This project is building an enterprise-grade, open-source, AI-Native ITSM platform for the China market. The long-term benchmark is ServiceNow-class process capability, but with lighter private deployment, stronger local enterprise integration, and open extensibility.

Core product goals:

- Cover complete ITIL v3/v4 service management workflows: ticket, incident, problem, change, release, service request, service catalog, SLA, knowledge, and CMDB.
- Make workflow customization a first-class capability through BPMN, process binding, form/config templates, and auditable task execution.
- Build AI into the service management lifecycle instead of adding a chatbot beside it: triage, summarization, knowledge retrieval, impact analysis, workflow recommendation, audit review, and controlled tool invocation.
- Prepare for Feishu, WeCom, DingTalk, Webhook, connector marketplace, skill marketplace, plugin marketplace, and CLI-driven operations.
- Support private deployment, SaaS, and SaaS + MSP modes without forking the core data model.

When making architecture choices, prefer enterprise correctness, auditability, tenant isolation, and extensibility over quick feature-only shortcuts.

## Current Product Stage

The repository is past v1.0 GA foundation work and is moving through v1.1 hardening:

- v1.0 delivered ITIL core flows, BPMN workflow engine, CMDB v1, knowledge/RAG scaffold, SLA, RBAC, multi-tenant/MSP foundations, Docker Compose, GHCR images, and basic AI/connector scaffolding.
- v1.1 focus is coverage backfill, controller splitting, connector marketplace v1, RBAC hardening, AI audit console, and integration test coverage.
- v1.5+ focus is measurable AI evaluator, Feishu/DingTalk/WeCom production connectors, Skill registry, performance budgets, and stronger security scans.

For new work, align with the roadmap rather than creating parallel mechanisms. If a feature overlaps with workflow, connector, AI skill, or marketplace direction, extend the existing extension point.

## Development Commands

### Frontend (itsm-frontend)

```bash
cd itsm-frontend
npm install              # Install dependencies
npm run dev              # Start dev server (http://localhost:3000)
npm run build            # Production build
npm run lint             # Lint with auto-fix
npm run lint:check       # Lint check only
npm run lint:antd        # 检查 Ant Design 废弃/旧版 API（前端改动必跑）
npm run type-check       # TypeScript type check
npm test                 # Run all tests
npm run test:unit        # Unit tests only
npm run test:integration # Integration tests only
npm run test:e2e         # E2E tests
```

### Backend (itsm-backend)

```bash
cd itsm-backend
go run main.go           # Start server (http://localhost:8090)
go build -o itsm-backend main.go # Binary build
./itsm-backend           # Run binary
go test ./...            # Run all tests
# Database migrations (use build tags)
go run -tags migrate main.go
go run -tags create_user main.go
```

### Environment Setup

```bash
# Configure .env file in itsm-backend/
LOG_LEVEL=info
DB_PASSWORD=your_password
JWT_SECRET=your-jwt-secret
ADMIN_PASSWORD=admin123
```

## Architecture

### System Boundaries

- **itsm-backend** is the source of truth for domain rules, permissions, workflow execution, tenant isolation, audit logs, and integration contracts.
- **itsm-frontend** is the operator/admin/user experience layer. It should not duplicate business rules that belong in backend services.
- **itsm-ai-service / guidance_sidecar / RAG services** provide AI assistance and retrieval. They must remain observable and fail safely.
- **itsm-agent / itsm-skill / itsm-cli** are future-facing extension surfaces. Do not hard-code behavior into the core app if it clearly belongs in a connector, skill, or CLI operation.
- **connector marketplace** integrations must go through lifecycle, health check, config, permission, and audit boundaries instead of ad hoc HTTP calls from controllers.

### Backend Structure

- **handlers/<domain>/** - **Target architecture.** Domain-sliced vertical slices. Each package owns: `handler.go` (HTTP), `service.go` (business logic), `repository.go` + `repository_impl.go` (data access), `entity.go` (domain entities/DTOs). Existing domains: ai, cab, capability, change, cmdb, dashboard, email_intake, incident, knowledge, known_error, operations, problem, service_catalog, service_request, skill, sla, standard_change. Shared helpers live in `handlers/common/` and `handlers/shared/`.
- **service/** - Business logic used by both legacy `controller/` and `handlers/<domain>/`. Do not add new business logic here without a clear owner.
- **controller/** - **Legacy horizontal layering.** Hosts thin HTTP facades that delegate to `service/`. This directory is frozen for new code. Existing controllers are gradually migrated to `handlers/<domain>/` as part of normal development.
- **ent/schema/** - Database schema definitions (Ent ORM)
- **middleware/** - Auth, logging, CORS, tenant isolation
- **dto/** - Request/response DTOs
- **cache/** - Redis integration
- **router/** - Route registration

### Backend Layering Rules

1. **New code goes to `handlers/<domain>/`.** If the domain does not yet have a `handlers/<domain>/` package, create one. Do not add new files to `controller/`.
2. **Follow the existing pattern.** When extending a domain that already uses `controller/` + `service/` (e.g., analytics, department), continue that pattern only for that domain — do not introduce a new `handlers/<domain>/` package alongside it.
3. **No dual implementation.** Do not implement the same domain endpoint in both `controller/` and `handlers/<domain>/`. Pick one and migrate if needed.
4. **Migration is opportunistic.** When working on a domain that lives in `controller/` and the change is more than trivial, evaluate migrating it to `handlers/<domain>/` as part of the same change. Use `git mv` to preserve history.
5. **Rationale.** `handlers/<domain>/` is the long-term target because it aligns with product goals (MSP multi-tenant, marketplace, skill registry, team autonomy) and does not scale with horizontal layering as the domain count grows.

### Frontend Structure

- **src/app/** - Next.js App Router pages and layouts
- **src/app/(main)/** - Protected page routes
- **src/components/** - Reusable UI components, organized by business domain (`ticket/`, `incident/`, `cmdb/`, ...) plus shared `ui/`, `common/`, `layout/`
- **src/lib/** - API clients (`lib/api/`), business hooks (`lib/hooks/`), frontend services (`lib/services/`), Zustand stores (`lib/store/`), utilities (`lib/utils/`)
- **src/hooks/** - Global custom React hooks
- **src/types/** - Shared TypeScript type definitions

### API Response Format

All APIs return `{ code: number, message: string, data: any }`:

- `code: 0` = success
- `code: 1001+` = param errors
- `code: 2001` = auth failed
- `code: 5001` = internal error

### Key Services

- **BPMN Workflow**: `service/bpmn_*`, uses `nitram509/lib-bpmn-engine`
- **RAG/Knowledge**: `service/rag_service.go`, `service/vector_store.go`
- **AI Features**: `service/llm_gateway.go`, `service/triage_service.go`
- **SLA**: `service/sla_monitor_service.go`, `service/escalation_service.go`

### Domain Ownership

- Ticket, incident, problem, change, release, and service request are separate business domains. Reuse shared helpers, but do not collapse their lifecycle rules into one generic ticket abstraction unless the existing code already does so.
- BPMN/process execution is the orchestration layer. Approval chains, service catalog fulfillment, SLA escalation, and AI suggestions should integrate through workflow/process records where possible.
- CMDB is not an asset table. Preserve CI type, CI instance, relationship, topology, discovery source, impact analysis, and reconciliation concepts.
- Knowledge/RAG features must keep source attribution, versioning, and permission filtering. Do not return knowledge content across tenant or permission boundaries.
- MSP and multi-tenant behavior must be considered for every new table, query, API, menu item, and background job.

## Important Patterns

### Backend

- Use `common.Success(c, data)` / `common.Fail(c, code, msg)` for responses
- Use `zap.S()` for logging, not `fmt.Println()`
- Controllers call services, never access DB directly
- Ent schemas in `ent/schema/*.go` generate CRUD
- Keep controller methods thin: bind/validate request, call service, map DTO, return response.
- Put transaction boundaries in service/domain layers, not in frontend or controller glue.
- New domain tables must include tenant/account ownership fields where applicable and must be protected by tenant-aware queries.
- Prefer idempotent seed/migration/init logic. Default initialization must create product templates/configuration, not fake customer business data.
- Any high-risk action triggered by AI, connector, workflow automation, or bulk operation must create an audit record.

### 生产入口与领域归属（强制）

- 修改功能前必须从实际注册的 Router 或 Worker handler 追踪生产调用链，确认 `route -> controller/handler -> service -> repository`；仅修改未接线 helper 不算完成
- 新增实现前必须搜索同名 Route、Handler、Service、Repository 和 CommandType；禁止在 `controller/service` 与 `handlers/<domain>` 两套架构中重复实现同一用例
- 一个用例只能有一个业务规则所有者。跨领域调用使用公开 service/command 接口，禁止访问其他领域的 `repository_impl`、Ent 查询细节或私有状态机
- Handler/Controller 只负责鉴权上下文、绑定校验、调用一个用例、错误映射和 DTO 输出；不得直接访问 Ent、SQL、连接器、LLM 或 BPMN 引擎
- 完成报告必须指出真实生产入口和调用链；测试只覆盖孤立 helper、但生产路由未接线时，不得声称功能完成

### 身份、租户与数据范围（强制）

- `tenantId`、`userId`、requester、creator、actor、operator 等身份必须来自认证中间件上下文，禁止信任请求体、查询参数或客户端 Header 中的同名字段
- 缺少租户上下文必须 fail closed；禁止默认租户、回退到 tenant 1、无租户查询或用全局管理员角色名称推断跨租户权限
- tenant scope 必须覆盖 Get/Only/Exist/Count/List/Update/Delete、关联表、聚合统计、唯一性检查、后台扫描、导入导出、Raw SQL、缓存 key 和异步消费者
- 写入关联 ID 前必须验证目标资源属于同一租户且当前用户有权引用；只校验主资源 tenant 不足以防止跨租户关系注入
- MSP 场景必须明确 operator tenant、customer tenant 和 delegated scope；跨租户系统任务只能使用显式 system context，并记录原因、范围与审计
- 数据库 RLS 是纵深防御，不能替代应用层 tenant predicate；两层均应 fail closed
- 每个新增租户资源接口至少包含一个跨租户拒绝测试，覆盖真实 Router/Handler 入口

### PATCH、状态流转与并发控制（强制）

- PATCH/部分更新 DTO 的可选字段使用指针，包括 `bool`、`int`、枚举和时间，必须区分“未传”“传零值”“清空字段”
- 禁止用非指针 DTO、truthy 判断或默认值覆盖来推断字段是否提交
- 所有 ITIL 状态迁移必须由领域 service 校验允许的 source/target、权限、必填数据和副作用；前端禁用按钮不是状态机
- 高并发可修改资源使用显式 `version` 乐观锁或等价条件更新，写入必须包含 `WHERE id + tenant_id + version` 并检查 affected rows
- read-then-unconditional-write 不属于乐观锁；版本冲突必须返回明确 conflict，不得静默覆盖或伪装成 not-found
- SLA、升级、审批窗口和生命周期耗时使用数据库中的权威时间戳；禁止依赖浏览器时间、UI 状态或消息到达时间
- 状态流转测试至少覆盖成功、非法迁移、权限拒绝和版本冲突

### 事务、Outbox 与可靠副作用（强制）

当业务写入会触发通知、审批任务、BPMN、连接器、Webhook、邮件、CMDB 同步或 AI 作业时，必须使用事务型 durable command/outbox：

- 在同一个 service/domain 事务中写入业务状态、历史/审计记录和 `operational_commands`；使用现有 `commandbus.EnqueueTx` 或 `EnqueueSQLTx`
- 禁止业务提交后再 enqueue，禁止 fire-and-forget goroutine，禁止用同步网络调用代替 durable command
- 数据库事务内禁止调用 LLM、连接器、HTTP、邮件或远程 BPMN；事务只保存完成业务不变量所需的持久状态
- command payload 只保存必要 ID 和最小不可变快照，不得包含 connector secret、token、密码或完整敏感正文
- consumer 必须根据 command tenant 重新加载权威资源、接收人和 connector，并再次执行 tenant/RBAC/启用状态校验；不得信任 payload 自报归属
- 一个 recipient + channel 对应一个可追踪 delivery；通用 Notification、领域通知和 Delivery audit 不得相互伪造或混用
- 事务测试必须证明：enqueue 失败会回滚业务写入，业务失败不会遗留 command，重复请求不会产生重复副作用

### 幂等、重试与可恢复性（强制）

- 幂等 key 必须包含 tenant、command type、aggregate type/id、业务事件 occurrence/version，并按需包含 recipient/channel；禁止用显示文本、当前时间或 retry count 作为身份
- Worker 必须具备 claim/lease、fencing token、有限指数退避、最大重试次数、dead-letter 状态和 operator replay 路径
- Provider 支持 idempotency token/message ID 时必须透传 command idempotency key，覆盖“外部成功但本地 audit 写入失败”的重试场景
- retry 不得创建重复 audit/delivery；执行外部调用前先检查已成功的 delivery 记录
- 永久非法 payload、跨租户资源或缺失 handler 必须进入可观察失败/dead-letter，禁止吞错并标记成功
- replay 必须保留原始幂等身份并新增操作审计，禁止通过生成新 key 绕过重复保护

### 能力状态与失败语义（强制）

- 产品能力必须区分 disabled、unconfigured、unready/degraded、ready；禁止把未实现、未配置或依赖不可用伪装成空成功结果
- capability flag 只控制产品呈现和显式契约禁用，不能跳过后端权限、租户或审计检查
- 缺少 connector/LLM/BPMN 配置属于 operational unavailable，不得返回用户参数错误；非法状态迁移属于 conflict，资源不存在与无权访问按安全策略明确映射
- Controller/Handler 必须稳定映射 validation、unauthorized、forbidden、not-found、conflict、unavailable 和 internal error；禁止把所有错误返回 500 或泄漏原始 SQL/provider 错误
- 降级路径必须返回可观测状态和确定性 fallback，记录 provider/model/能力版本及安全错误分类；禁止静默吃掉失败

### Context、日志与可观测性（强制）

- 请求链路必须传递 `c.Request.Context()`，Worker 必须传递带 tenant/command 信息的 context；业务路径禁止用 `context.Background()` 丢失取消、超时和租户信息
- 外部调用必须设置超时并响应 context cancellation；禁止无限等待或自行启动无法停止的 goroutine
- 使用结构化 `zap` 日志，至少携带适用的 request ID、tenant ID、actor ID、aggregate type/ID、command ID 和稳定 error class
- 日志、metric label 和 trace attribute 不得包含密码、JWT、API key、connector secret、完整工单正文、私有 prompt 或未脱敏接收地址
- Provider 原始错误必须净化后再写日志、audit 或响应；运维需要保留稳定错误码、attempt、last safe error、时间戳和 dead-letter 状态
- Metrics label 必须低基数，禁止使用 ticket ID、user ID、URL 原文、错误全文等高基数字段作为 label
- 禁止 `fmt.Println`、裸 `console.log` 或吞错；临时调试输出在提交前必须删除

### Schema、迁移与初始化（强制）

- 新 tenant-owned 表通常必须包含 `tenant_id`、时间戳、业务状态/所有权字段，以及匹配真实查询的索引；业务唯一键默认按 tenant 组合唯一
- 外键、Ent edge、级联/限制/软删除行为必须明确设计；不得依赖 ORM 默认行为猜测数据生命周期
- 修改 `ent/schema` 后必须重新生成 Ent 代码并审查 generated diff；禁止手改 generated 文件
- 生产 schema 变更使用显式、可版本化迁移，不能依赖服务启动时自动建表；迁移必须考虑已有数据、冲突检测、回滚或前滚恢复
- 新增唯一约束前必须提供存量重复检测与修复方案；只在空数据库验证通过不算完成
- 初始化/seed 必须可重复执行、带版本和 checksum；只能创建产品模板/基线配置，禁止写入伪造客户业务数据
- 初始化 Worker 必须使用 lease/fencing 防止并发执行者重复提交；恢复或重试不得重复创建租户基线数据

### Frontend

- Use App Router (no Pages Router)
- API calls via `src/lib/api/*.ts` classes
- Global state with Zustand in `src/lib/store/`
- Tailwind CSS for all styling
- Treat backend API DTOs as the contract. Do not patch around backend field bugs in UI code without also fixing the DTO/mapper.
- Keep operational screens dense and scannable: tables, filters, status, owners, timestamps, and actions should be easy to compare.
- Prefer feature-local components/hooks under the route module for domain-specific UI; use shared components only when reuse is real.
- User-facing enterprise workflows must show loading, empty, error, permission-denied, and success states.

### 前端服务端状态与操作一致性（强制）

- 服务端数据优先通过现有 React Query hooks/API client 管理；禁止在同一领域并行维护 React Query、Zustand 和组件 local state 三份服务器真相
- Query key 必须包含所有影响结果的 tenant scope、过滤、排序和分页参数；mutation 成功后精确 invalidate/update 所属资源，禁止用整站 reload 掩盖缓存问题
- 请求竞态必须由 query cancellation、AbortSignal 或请求身份解决；禁止让旧响应覆盖新过滤条件的结果
- Zustand 只保存跨页面 UI/会话状态或明确的客户端状态，不复制可重新查询的领域对象列表
- 列表 `rowKey` 使用后端稳定 ID，不得用数组 index；分页、排序和过滤由后端负责时，前端不得再对当前页做二次业务排序后冒充全量结果
- 创建/更新/删除操作必须防重复提交，并区分 processing、success、conflict、permission denied 和 unavailable；禁止所有失败只显示“操作失败”
- 破坏性、高风险或不可逆动作必须展示对象身份与影响范围并二次确认；批量操作还必须显示选中数量、失败明细和可重试结果
- 权限 Hook、菜单和按钮只负责展示；即使 UI 隐藏，API 仍必须由后端授权。前端不得根据角色名称推导超出后端返回权限的能力
- URL 中可分享的列表状态（搜索、过滤、分页或选中视图）应使用稳定 camelCase query contract；不得在刷新后悄然回到不同的数据范围

## AI-Native Engineering Rules

- AI suggestions are decision support by default, not silent authority. For triage, impact analysis, workflow recommendation, and automation, keep confidence, prompt/template version, model/provider, accepted/rejected status, and operator feedback.
- LLM calls must go through the LLM gateway or existing AI service abstraction. Do not call model providers directly from random controllers or frontend components.
- Always provide deterministic fallback behavior for AI failure, timeout, low confidence, or disabled provider.
- Prompts and skills should be versioned, testable, and auditable. Avoid burying large prompts in controller code.
- RAG must respect tenant, RBAC, knowledge visibility, and article version state before retrieval and before final response.
- Tool invocation by AI must pass explicit permission checks and produce audit logs.

## Workflow And ITIL Rules

- Workflow changes must preserve process definition, process instance, task, variable, execution history, and audit log integrity.
- Do not introduce a second approval engine when BPMN/process binding can represent the behavior.
- ITIL lifecycle transitions must be explicit and validated in services. Avoid frontend-only status transitions.
- SLA timers and escalations must be computed from authoritative timestamps and policy bindings, not from UI-derived state.
- Change management must preserve risk, CAB/approval, release window, implementation result, and PIR concepts.
- Problem management must preserve root cause, workaround, known error, and linked incident relationships.

## CMDB Rules

- CI type/schema, CI instance, CI relationship, relationship type, discovery source, import/export task, saved view, topology, and impact analysis are separate concerns.
- Every CI mutation should preserve history or auditability where the existing model supports it.
- Discovery/import/reconciliation must be idempotent and source-aware. Avoid creating duplicate CIs from repeated discovery runs.
- Relationship and topology APIs should enforce tenant isolation and permission checks.
- Impact analysis should traverse CI relationships deliberately and protect against unbounded recursion.

## Connector, Skill, And Plugin Rules

- New enterprise integrations should be implemented as connectors with lifecycle states: installed, configured, enabled, health-checked, disabled/uninstalled where supported.
- Connector secrets must never be returned to the frontend. Return masked metadata and health status only.
- Feishu, WeCom, DingTalk, and Webhook integrations should share marketplace/lifecycle abstractions instead of one-off endpoint designs.
- Skills should be declarative where possible: manifest, inputs, outputs, permissions, audit behavior, and evaluation hooks.
- Plugins must not bypass RBAC, tenant isolation, audit logging, or API response contracts.
- CLI operations should call stable backend APIs or documented service commands. Do not let CLI logic become a hidden business-rule fork.

## Security And Compliance

- Never commit real secrets, tokens, customer data, production database dumps, or unmasked connector credentials.
- Authentication, RBAC, menu permissions, endpoint ACL, and tenant filters must be considered together. Hiding a menu is not authorization.
- Cross-tenant access must fail closed. Add tests when touching tenant-scoped queries.
- Logs must not expose passwords, JWTs, API keys, connector secrets, prompt secrets, or private ticket content unless explicitly designed as protected audit content.
- File upload, import, connector callback, webhook, and AI tool invocation endpoints are high-risk surfaces and need validation, size limits, and audit logs.

## Configuration

- Frontend: `NEXT_PUBLIC_API_URL` env var (default: <http://localhost:8090>)
- Backend: `config.yaml` or environment variables
- Backend runs on port 8090, frontend on 3000

## Build Tags for Backend

- `migrate` - Run database migrations
- `create_user` - Create test user
- Default (no tag) - Run normal server

## Testing

- Backend: Table-driven tests with `stretchr/testify`, use `enttest.NewClient()` for DB
- Frontend: Jest + React Testing Library, mock API calls

### Verification Expectations

- For backend service/controller changes, run the narrow package tests first, then `cd itsm-backend && go test ./...` when practical.
- For frontend type/API/UI changes, run `cd itsm-frontend && npm run type-check` and the relevant Jest or Playwright test.
- For contract changes, update backend DTOs/mappers, frontend API clients/types, and tests in the same change.
- For workflow, CMDB, connector, AI, auth, tenant, or deployment changes, include at least one regression test or a documented manual verification path.
- For production deployment changes, validate with explicit env file usage and health checks.

### 回归测试与完成判定（强制）

- 测试必须覆盖真实生产 Router/Handler/Worker 入口；只测试未接线 helper、mock service 或复制出的逻辑不能证明功能可用
- 每项业务变更先定义不变量，再至少覆盖成功路径和最高风险失败路径；租户、权限、事务、并发、幂等或状态机变更不得只有 happy path
- Bug 修复必须添加能够在修复前失败、修复后通过的回归测试；禁止只改实现或只更新 snapshot
- Controller 契约测试必须断言 HTTP status、业务 code、camelCase DTO 和敏感字段缺失；不得只断言响应包含某段字符串
- 数据库相关测试必须验证 rollback 和 tenant scope；异步副作用必须验证 duplicate、retry、dead-letter 或 fencing 中与改动相关的行为
- 测试数据必须显式区分 tenant A/B、actor、资源归属和权限，避免共享 fixture 无意中绕过隔离
- 先运行最窄的相关测试和静态检查，再运行领域/全量套件；不得用全量套件噪声替代对改动范围的精确证明
- 如全量测试存在存量失败，必须报告准确命令、失败测试和为何与本次改动无关，同时证明修改包/模块的窄测试通过；禁止声称“全部通过”
- 完成前必须运行 `git diff --check` 并审查最终 diff，确认没有 generated 噪声、调试代码、临时 allowlist、密钥或无关用户文件

## API 响应规范

### Controller必须返回DTO，禁止直接返回Ent模型

- ✅ `common.Success(c, dto.ToTicketResponse(ticket))`
- ❌ `common.Success(c, ticket)` // ticket是*ent.Ticket

### 使用已有的Mapper函数

- `dto.ToTicketResponse()` / `ToTicketResponseList()`
- `dto.ToIncidentResponse()` / `ToIncidentResponseList()`
- `dto.ToUserDetailResponse()` / `ToUserDetailResponseList()`
- `dto.ToTenantResponse()` / `ToTenantResponseList()`

### List响应必须通过Service层转换

- Service层返回 `*TicketListResponse` 而非 `[]*ent.Ticket`
- Controller层使用Mapper包装单个对象

### API 契约单一事实来源（强制）

每个接口只能有一套正式契约。后端 DTO、Mapper 和 Router 是服务端实现依据，前端 `src/lib/api/` 类型与调用必须逐字段匹配；禁止让页面、Hook 或组件自行猜测接口结构。

- 成功响应统一为 `{ code: 0, message: string, data: T }`；业务成功不得返回 `code: 200`、`success: true` 等第二套标识
- 失败响应仍遵循 `{ code, message, data }`，HTTP status 表达传输层结果，业务 `code` 表达应用错误；两者职责不得混用
- 单对象响应的 `data` 直接承载 DTO，不得额外嵌套 `{ data: { data: ... } }`
- 新增列表接口统一使用 `data: { items, total, page, pageSize, totalPages }`；领域名只出现在 item 类型中，不得把 `items` 改成 `tickets`、`incidents`、`records` 或 `list`
- 请求分页字段统一为 `page`、`pageSize`；响应统计字段统一为 `total`、`totalPages`，禁止同时支持 `size`、`limit`、`offset`、`totalCount` 等别名，除非接口本身采用经过评审的另一种分页协议
- 请求 DTO、响应 DTO、查询参数、前端类型、Mock 和测试夹具必须在同一变更中更新
- API 字段新增、删除、改名或类型变化属于契约变更，必须先更新 DTO/Mapper 和契约测试，再更新前端调用；禁止只修前端适配

标准列表响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "pageSize": 20,
    "totalPages": 0
  }
}
```

### API URL 与路由规范（强制）

- 后端注册路由是 URL 契约的唯一来源；前端路径必须与实际 Gin Router 的 HTTP method、完整前缀、静态段和动态段一致
- API 统一使用 `/api/v1` 前缀；资源路径使用复数、全小写 `kebab-case`，例如 `/api/v1/service-requests/:id`
- 动态资源标识放在路径段中：后端使用 `:id`，前端使用包含 `${ticketId}` 的模板字面量；禁止用字符串拼接隐藏路径结构
- CRUD 使用 HTTP method 表达动作：`GET /tickets/:id`、`POST /tickets`、`PATCH /tickets/:id`、`DELETE /tickets/:id`；禁止新增 `/getTicket`、`/create-ticket` 等 RPC 风格路径
- 非 CRUD 领域动作采用资源下的明确动作段并保持审计语义，例如 `POST /changes/:id/approve`；不得把动作伪装成查询参数
- 查询参数只用于过滤、排序、搜索和分页，字段必须使用 camelCase；禁止同时发送 `pageSize` 与 `size` 或 `assigneeId` 与 `assignee_id`
- 页面和组件不得直接拼 API URL 或调用裸 `fetch/axios`；统一通过 `src/lib/api/` 客户端，并复用模块内静态 base path 常量
- 禁止在多个 API client 为同一后端资源维护不同 base path；跨领域调用应复用资源所属 API client
- 修改或新增路由时，必须同时更新 Router、权限/RBAC 注册、前端 API client、Mock、契约测试和相关文档；隐藏菜单不能替代后端路由授权

前端 API 路径应保持静态可分析：

```typescript
const TICKETS_PATH = '/api/v1/tickets';

httpClient.get<TicketResponse>(`${TICKETS_PATH}/${ticketId}`);
```

禁止无法被契约测试可靠解析的写法：

```typescript
httpClient.get('/api/v1/' + resource + '/' + id);
```

### 请求/响应多字段兼容零新增（强制）

禁止用多个候选字段“兼容”不确定契约。以下写法会掩盖后端 DTO 错误并制造永久双轨，均不得新增：

```typescript
// ❌ 禁止
const items = response.items ?? response.incidents ?? response.data ?? [];
const pageSize = response.pageSize ?? response.size ?? 20;
const total = response.total ?? response.totalCount ?? items.length;
const id = response.id ?? response.ID;
```

必须按唯一 DTO 读取：

```typescript
// ✅ 正确
const { items, total, page, pageSize, totalPages } = response;
```

- 禁止同一请求同时发送新旧字段，或同一响应同时返回多个同义字段
- 禁止在通用工具中做递归 camelCase/snake_case 猜测、`data.data` 自动解包或按字段存在性推断响应版本
- 禁止使用 `||` 给数字、布尔值和字符串字段兜底，因为 `0`、`false`、空字符串可能是合法值；可选字段应在 DTO 中明确定义，并只对该字段使用 `??`
- 默认值只能用于契约明确声明为可选的字段，不能用默认值隐藏必填字段缺失
- 历史接口迁移必须使用明确、局部、可删除的 adapter，写明旧版本、移除条件和跟踪事项；adapter 不得进入页面、Hook、领域 Store 或通用响应工具
- 若后端与前端不一致，应修复后端 DTO/Mapper 或前端声明中的错误一方，并添加回归测试；不得用多字段 fallback 让两套错误格式继续共存

### API 契约验证（强制）

涉及 Router、Controller DTO、查询参数或 `src/lib/api/` 的变更，至少执行：

```bash
cd itsm-frontend
npm run test:unit -- --runTestsByPath src/lib/__tests__/api-contract.test.ts
npm run type-check
```

- 契约测试必须使用完整路径和准确 HTTP method，禁止用 `stringContaining`、宽泛正则或只断言部分 URL
- 动态 base path 导致契约扫描器无法解析时，优先改成静态常量；不得直接加入跳过列表
- `KNOWN_UNMATCHED_FRONTEND_PATHS`、禁用契约或其他 allowlist 只能用于有产品开关且有明确原因的临时例外，不得用于绕过真实路由漂移
- 当前路由契约测试主要校验 URL；新增或修改接口还必须增加请求 DTO、响应 DTO 和分页字段的行为测试

### 字段命名规范

- Ent/Go 字段使用 Go 导出命名（`AssigneeID`、`TicketNumber`），对应数据库列使用 snake_case（`assignee_id`、`ticket_number`）
- 所有 HTTP 接口字段统一使用 camelCase（`assigneeId`、`ticketNumber`），前端不得继续传播 snake_case
- 请求 DTO、响应 DTO、JSON tag、query/form 参数、动态 Map key 都必须使用 camelCase
- Mapper 负责持久化模型与 API DTO 之间的转换；禁止为省事直接序列化 Ent 模型

## Docker 部署规范

### 生产环境启动（必须显式传入env-file）

```bash
# 正确：显式传入 --env-file
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 错误：缺少环境变量文件
docker-compose -f docker-compose.prod.yml up -d
```

### 常见问题排查

```bash
# 检查容器状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 检查容器日志
docker logs <container> --tail 30

# 检查容器网络
docker inspect <container> --format '{{json .NetworkSettings.Networks}}' | jq -r 'keys[]'

# 从容器内测试API
docker exec <container> wget -qO- http://localhost:8090/api/v1/health
```

### 网络隔离问题

生产容器与开发容器可能运行在不同网络：

- `itsm_itsm-network` - 开发网络
- `itsm_itsm-prod-network` - 生产网络
如遇DNS解析失败，先检查容器所在网络。

## TypeScript 开发规范

### Ant Design 组件导入

```tsx
// ❌ 错误：使用点链式访问 CompoundedComponent
Form.Input
Form.TextArea
Form.Select

// ✅ 正确：使用命名导入
import { Input, TextArea, Select } from 'antd';
```

### 组件迁移（Ant Design 升级）

| 旧API | 新API |
|-------|-------|
| `Space direction="vertical"` | `Space orientation="vertical"` |
| `Tabs.TabPane` | `items` 属性数组 |
| `Form.useForm()` | `const [form] = Form.useForm()` |

### Ant Design `Space.direction` 零新增规则（强制）

Ant Design 已弃用 `Space` 组件的 `direction` 属性。所有新代码和修改过的代码必须使用 `orientation`，禁止继续新增或复制 `Space direction={...}`。

```tsx
// ❌ 禁止：deprecated API
<Space direction="vertical" />
<Space direction={isMobile ? 'vertical' : 'horizontal'} />

// ✅ 正确
<Space orientation="vertical" />
<Space orientation={isMobile ? 'vertical' : 'horizontal'} />
```

- 本规则只针对 `Space.direction`；其他组件仍按其当前类型定义和官方 API 使用，不得机械替换同名但未弃用的属性
- 修改包含 `Space` 的旧组件时，应同步将该组件内现存的 `direction` 迁移为 `orientation`
- 禁止使用 props 展开、类型断言或封装别名规避弃用检查
- Code Review 必须检查新增行是否出现 `<Space ... direction=` 或传给 `Space` 的 `direction` 属性

提交前执行增量检查：

```bash
git diff --unified=0 -- '*.tsx' '*.jsx' | rg '^\+.*<Space\b[^>]*\bdirection\s*='
```

命中即视为违规，必须改为 `orientation`；完成后运行 `cd itsm-frontend && npm run type-check`。

### Ant Design v6 旧 API 零新增规则（强制）

本项目使用 Ant Design v6。新代码禁止继续使用 v4/v5 遗留或已弃用 API；修改旧组件时，应一并迁移该组件内触达的旧 API：

| 禁止写法 | 统一写法 |
|---------|---------|
| `Modal/Drawer visible={...}` | `open={...}` |
| `destroyOnClose` | `destroyOnHidden` |
| `Card bodyStyle={...}` | `styles={{ body: ... }}` |
| `Dropdown overlay={...}` | `menu={...}` |
| `Select dropdownRender={...}` | `popupRender={...}` |
| `Select onDropdownVisibleChange` | `onOpenChange` |
| `Tabs.TabPane` | `Tabs items` |

- 不得通过 `as any`、props 展开或自建兼容包装器继续传递废弃属性
- `visible` 等名称可用于项目自有组件，但 Ant Design 组件必须使用当前 API；Review 时应识别组件归属，不能机械全局替换
- 新增或修改 Ant Design 代码后必须运行 `cd itsm-frontend && npm run lint:antd && npm run type-check`
- 发现 `lint:antd` 未覆盖的新弃用告警时，应同步扩展 `tools/check-antd-legacy.sh`，防止同类问题再次进入

### Ant Design 反馈 API 与上下文（强制）

禁止直接使用静态 `message`、`notification` 和 `Modal.confirm/info/success/error/warning`。静态方法无法可靠继承主题、locale 和 App 上下文，统一从 `App.useApp()` 或项目已有 Hook 获取实例：

```tsx
// ❌ 禁止
import { message, Modal } from 'antd';
message.success('保存成功');
Modal.confirm({ title: '确认删除？' });

// ✅ 正确
import { App } from 'antd';

const { message, modal, notification } = App.useApp();
message.success('保存成功');
modal.confirm({ title: '确认删除？' });
```

- 非 React 模块不得直接导入静态反馈 API；应由调用方注入反馈能力，或通过项目统一的反馈/错误处理抽象返回结果
- 测试使用项目已有的 Ant Design `App` 测试 wrapper，不得为了方便退回静态 API
- 禁止在局部页面重复嵌套 `<App>` 修复上下文；优先复用根布局中的 `AntdProvider`/`App`，确有独立根节点时除外

### TypeScript 类型安全零新增规则（强制）

项目已启用 `strict`。新代码和修改行禁止使用类型逃逸掩盖契约问题：

- 禁止新增显式 `any`、`as any`、`@ts-ignore`、`@ts-nocheck` 或无理由的非空断言 `!`
- 外部/动态数据先使用 `unknown`，再通过类型守卫、schema 校验或判别联合收窄
- Ant Design 表格、表单、Select 等使用官方泛型和事件类型，禁止用 `columns as any`、`values: any` 绕过错误
- API client 必须声明准确的请求/响应泛型；禁止用 `(response as any)?.data ?? response.items ?? response` 猜测多套响应结构
- 测试需要构造非法输入时，可在最小范围使用明确注释的断言；生产代码不得复制测试中的类型逃逸
- 遇到类型错误应修复 DTO、Mapper、API 类型或组件泛型的根因，不得仅以“让 type-check 通过”为目标加入断言

### React Hook 正确性（强制）

- 禁止新增 `eslint-disable react-hooks/exhaustive-deps`；依赖不稳定时用 `useCallback`、`useMemo`、拆分 effect 或调整数据流解决
- Effect 必须清理 timer、subscription、AbortController、observer 和事件监听器，避免卸载后更新状态
- 基于旧状态计算新状态时使用函数式更新，例如 `setItems(previous => ...)`
- 不得在条件、循环、嵌套函数或普通工具函数中调用 Hook

### 富文本与 HTML 渲染安全（强制）

- 禁止新增未经净化的 `dangerouslySetInnerHTML`
- 知识文章、评论、工单描述、AI 输出和外部连接器内容必须先经过项目统一的 `SafeContent` 或 DOMPurify 净化
- 禁止用正则、自制字符串替换或“内容来自后端”作为跳过净化的理由
- 净化策略变更必须包含 XSS 回归测试，至少覆盖 `script`、事件属性、`javascript:` URL 和危险 SVG

### 字段命名一致性

```typescript
// ❌ 错误：与后端API不匹配
threshold_percent
notify_owners

// ✅ 正确：与API响应类型一致
thresholdPercentage
notifyOwners
```

### 类型检查

```bash
cd itsm-frontend && npm run type-check
```

## 数据结构约定

### 前后端字段命名规范

| 层级 | 命名风格 | 示例 |
|------|---------|------|
| 后端 Go/Ent 字段 | PascalCase | `AssigneeID`, `TicketNumber`, `CreatedAt` |
| Ent/数据库列名 | snake_case | `assignee_id`, `ticket_number`, `created_at` |
| 后端 DTO JSON tag | camelCase | `assigneeId`, `ticketNumber`, `createdAt` |
| 后端 DTO 响应 | camelCase | `assigneeId`, `ticketNumber`, `createdAt` |
| 前端 TypeScript | camelCase | `assigneeId`, `ticketNumber`, `createdAt` |
| 数据库字段 | snake_case | `assignee_id`, `ticket_number` |

### 核心规则

1. **后端 → 前端**：DTO 必须使用 camelCase 响应
2. **前端 → 后端**：API 请求 payload 使用 camelCase
3. **数据库**：仅数据库列名、SQL 和持久化配置使用 snake_case；Go/Ent 字段本身使用 PascalCase
4. **Mapper 转换**：DTO 层负责 snake_case → camelCase 转换

### API 字段契约（单一事实来源）

- 所有 HTTP/JSON 交互字段统一使用 `camelCase`，包括请求体、响应体、JSON tag、查询参数、表单字段、排序/过滤字段、前端类型定义和运行时对象 key
- `snake_case` 仅允许出现在数据库列名、SQL、Ent 的 storage key/field 配置、数据库迁移及与明确要求 snake_case 的外部系统适配器内部
- Controller 不直接暴露 Ent 模型；进入接口边界前必须通过 DTO/Mapper 完成字段转换
- 新增接口时，先定义前端期望的 `camelCase` DTO，再在后端 Mapper 中完成 `snake_case` 到 `camelCase` 的映射
- 如历史接口仍返回 `snake_case`，应视为待修复兼容问题，不得复制到新 DTO、前端类型或新接口中；兼容逻辑必须集中在明确标注的适配层，并附迁移说明

### Snake_case 零新增规则（强制）

新代码不得在 API 与前端边界新增 snake_case。以下写法均属于违规：

- Go DTO 中出现 `json:"assignee_id"`、`form:"assignee_id"`、`query:"assignee_id"`
- Gin 中读取或输出 snake_case 参数，例如 `c.Query("assignee_id")`、`gin.H{"ticket_number": value}`
- `map[string]any`、事件 payload、Webhook payload 或审计 metadata 使用 snake_case key（外部协议明确要求时除外，且必须封装在适配器内）
- TypeScript interface/type、对象字面量、表单字段名、`URLSearchParams`、表格 `dataIndex` 或状态 Store 使用 `assignee_id`
- 为兼容旧接口而同时长期返回 `assigneeId` 和 `assignee_id`

正确示例：

```go
type AssignTicketRequest struct {
    AssigneeID string `json:"assigneeId" form:"assigneeId"`
}

common.Success(c, gin.H{
    "ticketNumber": ticket.TicketNumber,
    "assigneeId":   ticket.AssigneeID,
})
```

```typescript
const query = new URLSearchParams({ assigneeId });
const payload = { ticketNumber, assigneeId };
```

外部系统确实要求 snake_case 时，必须在 connector/adapter 边界完成双向转换：核心 service、DTO 和前端仍使用 camelCase。禁止把外部协议字段直接泄漏到领域模型或公共 API。

### 提交前命名检查（强制）

涉及 API、DTO、前端类型或接口调用的变更，提交前必须检查本次新增行，而不是只依赖类型检查：

```bash
git diff --unified=0 -- '*.go' '*.ts' '*.tsx' | rg "^\\+.*(json|form|query):\\\"[a-z0-9]+_[a-z0-9_]+\\\"|^\\+.*[\\\"'][a-z][a-z0-9]*_[a-z0-9_]+[\\\"']"
```

- 命中后逐项确认；只有数据库/SQL/迁移、测试夹具中的数据库字段、或封装后的外部协议适配器可以保留
- 对允许的命中，应在代码附近注明边界原因，避免后续 Agent 误用
- 检查无违规后，仍需运行 `npm run type-check`、相关前端测试及后端相关包测试
- Code Review 必须将“新增 snake_case 是否逃出持久化/适配器边界”列为独立检查项

### 请求体与响应体示例

```json
{
  "serviceCatalogId": "svc_123",
  "assigneeId": "user_001",
  "notifyOwners": true,
  "thresholdPercentage": 85
}
```

对应数据库/Schema 字段可为：`service_catalog_id`、`assignee_id`、`notify_owners`、`threshold_percentage`。

### 响应类型驼峰约定

所有 API 响应类型必须使用驼峰命名：

```typescript
// ✅ 正确：使用驼峰命名的响应类型
interface TicketResponse {
  id: string;
  ticketNumber: string;
  title: string;
  assigneeId: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

// ❌ 错误：使用蛇形命名
interface TicketResponse {
  id: string;
  ticket_number: string;  // 不符合规范
  assignee_id: string;   // 不符合规范
}
```

### DTO Mapper 实现规范

```go
// ✅ 正确：在 DTO 层完成转换
func ToTicketResponse(ticket *ent.Ticket) *TicketResponse {
    return &TicketResponse{
        ID:           ticket.ID.String(),
        TicketNumber: ticket.TicketNumber,
        Title:        ticket.Title,
        AssigneeID:   ticket.AssigneeID,      // Ent 字段 snake_case
        Status:       ticket.Status,            // Ent 字段 snake_case
        CreatedAt:    ticket.CreatedAt.Format(), // 转换为字符串
        UpdatedAt:    ticket.UpdatedAt.Format(),
    }
}

// ❌ 错误：直接返回 Ent 模型
func GetTicket(c echo.Context) error {
    ticket, _ := svc.GetTicket(...)
    return c.JSON(200, ticket) // 泄漏 Ent 模型
}
```

## 文件命名规范

### 命名边界原则

- 接口字段命名与文件命名是两套规则：JSON/API 字段使用 `camelCase`，文件名按语言与框架约定执行
- 不要因为接口字段是 `camelCase`，就把 Go 后端文件命名成 `ticketService.go`
- 不要因为数据库字段是 `snake_case`，就把前端 TypeScript 文件命名成 `ticket_service.ts`
- 优先保持“看文件名就知道它属于哪一层、承担什么职责”

### 后端 (Go)

| 类型 | 命名风格 | 示例 |
|------|---------|------|
| Controller | `*_controller.go` | `ticket_controller.go` |
| Service | `*_service.go` | `ticket_service.go` |
| DTO | `*_dto.go` | `ticket_dto.go` |
| Schema | `*.go` (ent) | `ticket.go` |
| Middleware | `*_middleware.go` | `auth_middleware.go` |
| Repository | `*_repository.go` / `repository_impl.go` | `ticket_repository.go` |
| Router | `*_router.go` / `router.go` | `ticket_router.go` |

后端补充规则：

- Go 文件统一使用 `snake_case`
- 同一资源尽量使用统一前缀：`ticket_controller.go`、`ticket_service.go`、`ticket_dto.go`
- 避免无语义后缀：`ticket_handler_new.go`、`ticket_final.go`、`ticket_temp.go`
- 若文件承载的是实现细分，可使用职责后缀：`ticket_query_service.go`、`ticket_mapper.go`

### 前端 (TypeScript/Next.js)

| 类型 | 命名风格 | 示例 |
|------|---------|------|
| 页面 | `page.tsx` | `tickets/page.tsx` |
| 组件 | `*.tsx` (PascalCase) | `TicketList.tsx` |
| 工具函数 | `*.ts` (camelCase) | `formatDate.ts` |
| API 客户端 | `*Api.ts` | `TicketApi.ts` |
| 类型定义 | `*.ts` | `types/ticket.ts` |
| Hooks | `use*.ts` | `useTicket.ts` |
| Store | `*Store.ts` / `use*Store.ts` | `ticketStore.ts` |

前端补充规则：

- React 组件文件使用 `PascalCase`
- Hook、工具函数、store、API client 文件使用 `camelCase`
- Next.js 路由目录使用 `kebab-case`，动态路由使用 `[id]` 形式
- 同一模块的页面、组件、API、类型命名尽量围绕同一个业务词根，例如 `ticket` / `Ticket`
- 避免混用风格：不要同时出现 `TicketList.tsx`、`ticket-list.tsx`、`ticket_list.tsx`

### 目录结构命名

```
src/
├── app/                    # Next.js App Router
│   └── (main)/            # 路由组 (括号命名)
│       └── tickets/       # 页面目录 (kebab-case)
│           ├── page.tsx
│           └── [ticketId]/
│               └── page.tsx
├── components/            # 组件目录 (按业务域分子目录)
│   ├── ticket/            # 领域组件目录
│   │   └── TicketList.tsx # 组件文件 (PascalCase)
│   └── ui/                # 通用组件
├── hooks/                 # 全局 Hooks
├── types/                 # 类型定义
└── lib/
    ├── api/               # API 客户端
    ├── hooks/             # 业务 Hooks
    ├── services/          # 前端服务层
    ├── store/             # Zustand stores
    └── utils/             # 工具函数
```

### 禁止的命名方式

- ❌ `TicketListComponent.tsx` → ✅ `TicketList.tsx`
- ❌ `ticket_service.ts` → ✅ `ticketService.ts`
- ❌ `get_tickets.go` → ✅ `ticket_service.go`
- ❌ `APIUtils.ts` → ✅ `apiUtils.ts`
- ❌ `ticketService.go` → ✅ `ticket_service.go`
- ❌ `ticket_list.tsx` → ✅ `TicketList.tsx`
- ❌ `ticket-list-api.ts` → ✅ `ticketApi.ts` 或 `TicketApi.ts`

## 工程要求摘要

### 必须遵守的规则

1. **DTO 返回**：Controller 必须返回 DTO，禁止返回 Ent 模型
2. **字段命名**：后端 DTO 使用 camelCase，前端使用 camelCase
3. **文件命名**：遵循上述命名规范（Go 用 snake_case，TypeScript 用 PascalCase/camelCase）
4. **Mapper 转换**：DTO 层负责 snake_case → camelCase 转换
5. **命名边界**：接口字段、数据库字段、文件名各自遵循本层规则，禁止相互污染
6. **租户隔离**：新增查询、后台任务、导入导出、AI/RAG、连接器回调必须考虑 tenant/MSP 边界
7. **审计优先**：AI 建议、流程流转、审批、连接器动作、批量操作、高风险变更必须可追踪
8. **扩展优先**：流程、连接器、Skill、插件、CLI 方向已有扩展点时，优先扩展现有机制
9. **测试闭环**：改业务规则必须配套服务层/接口层/前端契约验证，不能只靠手工点击
10. **安全默认**：权限、密钥、日志、上传、Webhook、AI 工具调用默认按企业级高风险面处理
