# Feature Specification: ITSM v1.0 GA 角色驱动的全产品测试方案

**Feature Branch**: `001-role-based-testing`

**Created**: 2026-06-13

**Status**: Draft

**Input**: User description: "你是测试专家，基于角色视角制定完整产品测试方案 + 使用 specify-cli 完善"

**Source Document**: `docs/ITSM-基于角色视角的产品测试方案.md`（v1.0，2026-06-13）— 本规范是其 Spec Kit 化版本。

---

## Clarifications

### Session 2026-06-13

- Q: GA 前 `/reports/cmdb-quality`、`/reports/sla-trend`、`/reports/incident-mttr` 等无后端路由的报表菜单如何处置？ → A: GA 前隐藏菜单，路由延后至 post-GA 实现。
  - **DB 现状核查（重要更正）**：`menus` 表仅 25 条记录，顶级 `/reports`（id=10）下无子菜单；之前列出的 `/reports/cmdb-quality` 等仅出现在前端 `src/components/layout/sidebar/menu-config.ts` 的静态 fallback 配置中，并未注入数据库。
  - 影响：
    - 前端 `menu-config.ts` MUST 移除 `/reports/cmdb-quality`、`/reports/sla-trend`、`/reports/incident-trends`、`/reports/change-success`、`/reports/problem-efficiency`、`/tickets/templates` 等后端无对应路由的子项；或将顶级 `/reports` 也临时下架。
    - 数据库菜单 seed/迁移脚本 MUST 保持当前 25 条不再新增孤立报表项。
    - 新增 FR-1001/1002/1003。
  - 验收：`/api/v1/auth/menus` 返回的所有菜单项 MUST 100% 对应前端可达页面 + 后端可达 GET 路由；并通过 SC-204 校验。
  - 风险登记 R-05 状态：mitigation 为"GA 前清理 menu-config.ts 静态项 + post-GA 补后端"。

- Q: `/api/v1/templates/tickets`、`/api/v1/incident-rules`、`/api/v1/process-instances` 路径不可达。应如何处置？ → A: 以数据库菜单为权威源，前端不再硬编码菜单；冒烟脚本与文档使用真实路由。
  - **真实情况核查**：
    - `/api/v1/templates/tickets` 在代码与 DB 菜单中均未实际引用（仅 `menu-config.ts` 中的 `/tickets/templates` 前端路由 → 无对应后端 API），属于 §六冒烟矩阵的猜测路径。
    - `/api/v1/incident-rules` 在代码中未发现引用，同上。
    - `/api/v1/process-instances` 实际有效路径为 `/api/v1/bpmn/process-instances`（已在 `prd/工作流模块详细设计.md` 与 `controller/bpmn_workflow_controller.go` 注册）；前端 `workflow-engine-service.ts` 使用相对 baseUrl 的 `/process-instances`，由 baseUrl 决定真实前缀。
  - 决议：
    - 冒烟脚本与本 spec 中相关条目 MUST 改用 `/api/v1/bpmn/process-instances`（process）与 `/api/v1/workflow/instances`（workflow）的真实路径。
    - 对未实现的 `templates/tickets` 与 `incident-rules`，从测试矩阵移除，列入 post-GA 路线图。
    - 强化 FR-1002：菜单项的可达性由 DB + auth/menus + 真实路由三方校验，而非前端静态配置。
    - 新增 FR-1004：前端 `menu-config.ts` 中与 DB 菜单不一致的冗余项 MUST 在 GA 前清理；菜单加载以 DB 为唯一权威源。
  - 验收：
    - `/api/v1/bpmn/process-instances` GET 返回 `code=0`。
    - `menu-config.ts` 与 DB `menus` 表 diff 为空（除已知顶级布局差异）。
    - `docs/ITSM-基于角色视角的产品测试方案.md` 第六节"已知不可达"列表对应消除或重新解释。

---

## User Scenarios & Testing *(mandatory)*

> 测试方案是"被实施的产品"。每个用户故事 = 一个独立可交付的角色测试套件，单独跑也能给出 GA 通过/不通过的局部结论。

### User Story 1 - end_user 提单到关闭闭环验证 (Priority: P1)

终端用户登录 ITSM，提交工单 → 跟踪进度 → 浏览知识库 → 关闭并评分。需保证最普通的"用户报障"路径在 GA 后第一天就完全可用。

**Why this priority**: 终端用户是日活量最大、风险面最广的角色；提单失败=系统对外不可用。

**Independent Test**: 仅以 `user1 / user123` 登录，跑 `EU-01..EU-10` 即可独立判断"普通用户视角是否交付可用"。

**Acceptance Scenarios**:

1. **Given** end_user 已登录，**When** `POST /api/v1/tickets` 携带合法 `title/description/priority`，**Then** 返回 `code=0` 且 `data.ticketNumber` 形如 `TKT-YYYYMM-NNNNNN`。
2. **Given** end_user 已创建 N 张工单，**When** `GET /api/v1/tickets`，**Then** 仅返回 `requester_id == self` 的工单，不泄露其他用户工单。
3. **Given** end_user 试图 `PUT /api/v1/tickets/{otherId}`，**When** 该工单 `requester_id != self`，**Then** 返回 401/403 且不修改数据。
4. **Given** end_user 访问 `/api/v1/audit-logs` 或 `/api/v1/configuration-items`，**Then** 返回 401/403。
5. **Given** 工单已被 engineer 解决，**When** end_user 调用 `POST /api/v1/tickets/{id}/rating`，**Then** 评分入库且工单进入 `closed`。

---

### User Story 2 - engineer 处理人全生命周期 (Priority: P1)

工程师视角的工单/事件/变更处置，是"业务能否真正闭环"的判定标准。

**Why this priority**: 业务价值落地点。一线/二线工程师无法操作=GA 无意义。

**Independent Test**: 以 `engineer1 / eng123`（需补 seed 或脚本建账号）登录，跑 `EN-01..EN-10`。

**Acceptance Scenarios**:

1. **Given** engineer 接到分配，**When** 推进状态 `open → in_progress → resolved`，**Then** 状态机不可逆地切换且每步写 `audit-logs`。
2. **Given** 事件已解决，**When** engineer 调用 `POST /api/v1/incidents/{id}/root-cause` 并创建 problem，**Then** 形成 `incident → problem` 引用关系。
3. **Given** problem 已确认，**When** engineer 创建 change，**Then** change 进入审批流且关联 problem_id。
4. **Given** change 审批通过，**When** engineer 推进 `approved → in_progress → completed` 并修改 CI，**Then** CI 字段变更被审计且 `affected_ci_ids` 反向可查。
5. **Given** AI Triage 给出建议，**When** engineer `POST /api/v1/ai/audit { accepted: true }`，**Then** `/ai/metrics` 中 `acceptance_rate` 上升。

---

### User Story 3 - super_admin 跨租户与平台治理 (Priority: P1)

超级管理员负责租户/角色/菜单/连接器/AI 审计等"系统级"操作，必须在 GA 前 100% 通过。

**Why this priority**: 误操作即跨租户事故；GA 准入门槛硬指标。

**Independent Test**: 以 `admin / admin123` 登录跑 `SA-01..SA-10`。

**Acceptance Scenarios**:

1. **Given** super_admin 登录，**When** `GET /api/v1/auth/menus`，**Then** 返回 ≥80 个菜单项且包含所有顶级模块。
2. **Given** super_admin，**When** `GET /api/v1/readiness/ga`，**Then** 返回 `12/12 ready`。
3. **Given** 新建租户后切换上下文，**When** 使用新租户 token 调 `/api/v1/users`，**Then** 用户列表与原租户隔离。
4. **Given** super_admin 调整 `/api/v1/roles/{id}/permissions`，**When** 受影响用户复登录，**Then** 新权限实时生效。
5. **Given** super_admin 调用 `/api/v1/connectors/{id}/lifecycle` 安装 → enable → health → disable → uninstall，**Then** 每步 `code=0` 且 health 返回 `ready`。

---

### User Story 4 - approver 审批人多级审批 (Priority: P2)

审批人对 change/service_request 实施多级审批；审批不通=变更不能落地。

**Why this priority**: 影响变更/服务请求闭环但不影响"系统能用"。

**Independent Test**: 以 `manager1 / mgr123` 登录跑 `AP-01..AP-06`。

**Acceptance Scenarios**:

1. **Given** approver 待办列表，**When** 同意第一级审批，**Then** 实例进入下一审批节点。
2. **Given** 多级审批中任意节点驳回，**Then** 整个工作流标记 `rejected` 并写审计。
3. **Given** 节点 timeout 短于实测等待时间，**Then** 自动升级或挂起，且 `escalation_history` 有记录。

---

### User Story 5 - sd_manager 服务台主管运营视图 (Priority: P2)

服务台主管负责工作台总览、SLA 监控、自动分派规则、强制升级。

**Why this priority**: 主管功能缺失不阻塞业务流，但影响 SLA 达成率。

**Independent Test**: 以 `sd_manager` 角色账号跑 `SD-01..SD-07`。

**Acceptance Scenarios**:

1. **Given** sd_manager，**When** `GET /api/v1/dashboard/overview`，**Then** 返回今日工单/事件/SLA 风险三类计数。
2. **Given** 设置自动分派规则，**When** 新工单匹配规则，**Then** 自动写入 `assignee_id`。
3. **Given** SLA 风险预警，**When** `POST /api/v1/sla/monitoring`，**Then** 返回风险列表并触发连接器通知。

---

### User Story 6 - security 安全管理员只读审计 (Priority: P2)

安全管理员只读审计日志，不能改业务/配置/用户。

**Why this priority**: 合规需要，但不阻塞业务流。

**Independent Test**: 以 `security1 / security123` 登录跑 `SE-01..SE-06`。

**Acceptance Scenarios**:

1. **Given** security 角色，**When** `GET /api/v1/audit-logs`，**Then** 返回 200。
2. **Given** security 角色，**When** `PUT /api/v1/users/{id}` 或 `PUT /api/v1/sla/definitions/{id}`，**Then** 返回 403。
3. **Given** 多次失败登录，**Then** 审计日志中含 `login_failed` 记录可被 security 检索。

---

### User Story 7 - tenant_admin 租户隔离 (Priority: P2)

租户管理员仅能管理自身租户的数据，跨租户调用必须为空或 403。

**Why this priority**: 多租户隔离=合规底线；GA 安全审查项。

**Independent Test**: 用 `tenant1admin / ta123`（需补 seed）跑 `TA-01..TA-06` + `FLOW-9`。

**Acceptance Scenarios**:

1. **Given** tenant1admin 登录，**When** `GET /api/v1/tickets`，**Then** 不包含 tenant2 工单。
2. **Given** 客户端篡改 `tenant_id` 字段提交，**When** 服务端处理，**Then** 强制使用 token 中 tenant_id，篡改无效。

---

### Edge Cases

- 同一工单被两位 engineer 并发推进状态 → 后者收到乐观锁/版本冲突错误，工单状态保持一致。
- AI Triage 调用 LLM 超时（>10s）→ 返回降级建议，写入 `ai_audit.timeout=true`，不阻塞工单创建。
- 连接器配置 webhook URL 指向内网（如 `http://127.0.0.1:22`）→ 拒绝或限制内网地址（A10 SSRF）。
- Redis NOAUTH/不可用 → fallback 到内存限流和内存序列号生成，应用不崩溃但有 warning 日志。
- 一次提单包含 ≥10MB 附件 → 校验文件大小并拒绝，错误信息明确可读。
- 在毫秒级双击"关闭工单"按钮 → 第二次返回 `already_closed` 而非 500。
- 跨租户工单评论引用 → 返回空数据集，不返回任何对方租户内容。
- 删除一个仍有 in_progress 工单的租户 → 阻断或软删除并保留审计追溯。
- 知识库文章状态为 `draft` → end_user 在 `/knowledge-articles` 不可见。
- 终端用户调用 `GET /api/v1/users/{otherUserId}` → 仅返回最小公开信息，不返回 phone/email。

---

## Requirements *(mandatory)*

### Functional Requirements

#### FR-100 系列：认证与会话

- **FR-101**: 系统 MUST 支持账号密码登录，登录成功返回 `{code:0, data:{access_token, refresh_token, user, menus}}`。
- **FR-102**: 系统 MUST 在所有受保护 API 上强制 JWT 校验，过期/篡改/缺失 Token 一律 401。
- **FR-103**: 系统 MUST 在登录失败 ≥5 次/min 时触发限流，且写 `audit-logs.action=login_failed`。
- **FR-104**: 系统 MUST 在 `/api/v1/auth/menus` 中按当前用户权限返回菜单子集，super_admin 应能看到 ≥80 项。

#### FR-200 系列：工单/事件/问题/变更

- **FR-201**: 用户 MUST 能创建工单，创建后自动生成 `ticket_number` 形如 `TKT-YYYYMM-NNNNNN` 且全局唯一。
- **FR-202**: 系统 MUST 维护工单状态机 `open → in_progress → resolved → closed`，禁止逆向跳转（`reopen` 除外）。
- **FR-203**: 系统 MUST 支持 `incident → problem → change` 引用关系，并在审计中可追溯。
- **FR-204**: 标准变更 MUST 跳过人工审批，普通变更 MUST 走多级审批。
- **FR-205**: 系统 MUST 在工单关闭时支持评分 1-5，并支持服务满意度分析。
- **FR-206**: 关闭/重开必须写审计且需要权限 `ticket:close|ticket:reopen`。

#### FR-300 系列：CMDB 与服务目录

- **FR-301**: CMDB MUST 支持 CI 类型与属性扩展（`/configuration-items/types`）。
- **FR-302**: CMDB MUST 支持 `/configuration-items/{id}/impact-analysis` 影响分析，返回下游受影响 CI 列表。
- **FR-303**: 服务目录条目 MUST 支持 `published / archived` 状态切换，end_user 仅可见 `published`。

#### FR-400 系列：SLA 与流程

- **FR-401**: 系统 MUST 提供 6+ 默认 SLA 定义覆盖 P0-P3、service_request、change。
- **FR-402**: 系统 MUST 提供 `POST /api/v1/sla/monitoring` 返回当前风险工单列表。
- **FR-403**: SLA 违约 MUST 触发连接器通知（飞书/钉钉/企业微信/邮件之一），写 `slaalerthistory`。
- **FR-404**: 审批工作流 MUST 支持 `manager_timeout` 超时升级。

#### FR-500 系列：AI / 知识

- **FR-501**: AI Triage MUST 在 P95 < 3s 内返回建议（受 LLM 影响可单独基线）。
- **FR-502**: `POST /api/v1/ai/audit` MUST 接受 `{ ticketId, suggestion, accepted }` 并入库。
- **FR-503**: `/api/v1/ai/metrics` MUST 返回 `acceptance_rate / total_calls`。
- **FR-504**: `/api/v1/ai/rag/ask` 在无匹配 KB 时 MUST 优雅降级到关键字搜索而非 500。

#### FR-600 系列：RBAC / 多租户

- **FR-601**: 系统 MUST 强制租户隔离：服务端忽略客户端 `tenant_id` 字段，强制使用 token 中值。
- **FR-602**: end_user MUST 仅看到 `requester_id == self` 的工单。
- **FR-603**: security MUST 拥有 `audit-logs:read` 但不得拥有任何业务写权限。
- **FR-604**: 角色 `super_admin / security / end_user` MUST 在默认 seed 中存在；`engineer / manager / tenant_admin` 业务角色 MUST 可通过 seed 或脚本快速创建。

#### FR-700 系列：连接器与集成

- **FR-701**: 系统 MUST 提供 5 个内置连接器（飞书、钉钉、企业微信、Console、Webhook）的 lifecycle 接口。
- **FR-702**: 连接器 MUST 拒绝 webhook URL 指向内网地址（A10 SSRF 防护）。
- **FR-703**: 连接器 health 检查 MUST 返回 `ready / degraded / unhealthy` 三态。

#### FR-800 系列：可观测性 / 准入

- **FR-801**: `GET /api/v1/health` MUST 在 200ms 内返回 200。
- **FR-802**: `GET /api/v1/readiness/ga` MUST 列出 12 个核心模块及其 `ready/not_ready` 状态。
- **FR-803**: 关键操作（登录、工单状态变更、审批、角色变更、租户操作、连接器变更）MUST 全部留 `audit-logs`。

#### FR-900 系列：安全基线（OWASP Top 10 映射）

- **FR-901** (A01): 任意角色越权调用 MUST 返回 403。
- **FR-902** (A02): JWT 必须签名校验通过；任意位篡改 → 401。
- **FR-903** (A03): 工单/CI 字段中的 SQL/HTML 注入串 MUST 被参数化存储且渲染时转义。
- **FR-904** (A05): 生产 `SERVER_MODE=release`、`ITSM_CORS_ALLOWED_ORIGINS` 显式白名单、`REDIS_PASSWORD` 必填或 fallback 透明。
- **FR-905** (A07): 暴力登录 MUST 触发限流。
- **FR-906** (A08): 客户端无法通过修改请求体跨租户写数据。
- **FR-907** (A09): 关键操作有审计；登录失败、权限拒绝有专门记录。
- **FR-908** (A10): 连接器 webhook 内网地址 MUST 拒绝。

> 标记不明项（NEEDS CLARIFICATION）：
>
> - ~~**FR-NC-001**~~：✅ 已澄清（2026-06-13）→ GA 前清理前端 `menu-config.ts` 中 `/reports/*` 子项；后端报表路由延后至 post-GA。详见 `## Clarifications`。
> - ~~**FR-NC-002**~~：✅ 已澄清（2026-06-13）→ 菜单以 DB 为权威源，前端 `menu-config.ts` 与 DB `menus` 表对齐；`process-instances` 真实路径为 `/api/v1/bpmn/process-instances`；`templates/tickets`、`incident-rules` 移出 GA 测试矩阵。

#### FR-1000 系列：菜单与导航（GA 范围裁剪）

- **FR-1001**: 前端 `src/components/layout/sidebar/menu-config.ts` MUST 不包含 `/reports/cmdb-quality`、`/reports/sla-trend`、`/reports/incident-trends`、`/reports/change-success`、`/reports/problem-efficiency` 等无后端 API 的子菜单；GA 前清理或将顶级 `/reports` 同步下架。
- **FR-1002**: 任何 GA 镜像中保留的菜单项 MUST 同时满足：(a) DB `menus` 表存在，(b) 前端有对应可达页面，(c) 该页面所需后端 API 返回 `code=0`。
- **FR-1003**: 报表能力 MUST 列入 post-GA 路线图，由独立 spec 跟踪；本 spec 不再覆盖。
- **FR-1004**: 菜单加载以 DB `menus` 表为唯一权威源；前端静态 `menu-config.ts` 仅作为本地开发兜底，与 DB diff 在 GA 前 MUST 收敛为零。

### Key Entities

- **Tenant**：多租户根；属性 `name, code, domain, status`；删除级联限制。
- **User**：归属租户，挂角色；属性 `username, email, name, role, active, tenant_id`。
- **Role / Permission**：RBAC 核心；角色对资源 `(resource, action)` 授权。
- **Ticket**：工单；属性 `title, ticket_number, priority, status, requester_id, assignee_id, tenant_id, affected_ci_ids`。
- **Incident / Problem / Change**：工单子类；通过引用关系串成 ITIL 闭环。
- **ConfigurationItem (CI)**：CMDB 条目；含 `ci_type_id`、关系（依赖/被依赖）、影响分析。
- **KnowledgeArticle**：状态 `draft / published / archived`；可被工单引用。
- **ServiceCatalog / ServiceRequest**：服务目录条目 + 服务请求实例。
- **SLA Definition / Monitoring / AlertHistory**：定义、监控、告警历史。
- **ApprovalWorkflow / Instance / Node**：多级审批；含 `manager_timeout`。
- **Connector**：飞书/钉钉/企业微信/Console/Webhook；含 lifecycle 状态。
- **AIAudit**：`ticketId, suggestion, accepted, latencyMs, timeout`。
- **AuditLog**：贯穿全模块的操作留痕；不可篡改。

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

#### SC-1xx 通过率

- **SC-101**: P1 角色 (end_user / engineer / super_admin) 全部场景 100% 通过；任何 P0 缺陷阻塞 GA。
- **SC-102**: P2 角色 (approver / sd_manager / security / tenant_admin) ≥90% 场景通过。
- **SC-103**: 跨角色 E2E `FLOW-1..FLOW-10` 100% 通过；FLOW-9（多租户隔离）零跨租户泄露。

#### SC-2xx 自动化与覆盖

- **SC-201**: 后端 `go test ./...` 100% 通过（17 包）。
- **SC-202**: 前端 `npm run test:unit` 100% 通过（≥526 测试），覆盖率 `lib/api/* / lib/services/*` ≥60%。
- **SC-203**: TypeScript type-check 0 错误；ESLint 0 errors，warnings ≤500。
- **SC-204**: API 冒烟脚本覆盖 ≥25 端点，全部 `code=0`。

#### SC-3xx 性能基线

- **SC-301**: `/api/v1/tickets` 列表 P95 < 300ms @ 50 RPS。
- **SC-302**: `/api/v1/tickets` 创建 P95 < 500ms @ 20 RPS。
- **SC-303**: `/api/v1/dashboard/overview` P95 < 600ms。
- **SC-304**: AI Triage P95 < 3s（独立基线，受 LLM 影响）。
- **SC-305**: 24h soak：错误率 < 0.1%，无内存泄漏（RSS 增长 < 5%/h）。

#### SC-4xx 安全 / 准入

- **SC-401**: OWASP Top 10 用例全部通过（A01-A10）。
- **SC-402**: `/api/v1/readiness/ga` 返回 12/12 ready。
- **SC-403**: 已知 P0 缺陷 0；P1 缺陷 ≤2 且具备 workaround。

#### SC-5xx 用户体验

- **SC-501**: 一次端到端"提单 → 解决 → 关闭 → 评分"在 5 分钟内完成（操作员熟练度基准）。
- **SC-502**: AI Triage 建议接受率 ≥30%（GA 后 30 天内统计）。
- **SC-503**: end_user 提单成功率 ≥99%（不含人为参数错误）。

---

## Assumptions

- ITSM 部署在私有环境，PostgreSQL/Redis 自托管，外部 LLM 通过 `llm.endpoint` 接入。
- 测试数据使用默认 seed（admin/user1/security1）+ 至少 2 个租户用于隔离测试；缺失的 `engineer1 / manager1 / tenant1admin` 通过 seed 补丁或前置脚本建账号。
- 性能基线在 4C8G 标准容器/单节点上测得；生产硬件可放宽 SC-3xx 但不可放宽 SC-1xx/SC-4xx。
- AI 模块依赖外部 LLM（MiniMax）可用；离线场景启用降级策略而非阻塞主流程。
- 现有自动化基线已就绪：`go test ./...` 全过、`npm run test:unit` 526/526 全过、`comprehensive-e2e.spec.ts` 13/13 全过。
- Spec Kit 工件位于 `.specify/`（仓库根），后续 `/speckit-plan` 与 `/speckit-tasks` 将基于本 spec 生成。
- 本规范不取代 `docs/ITSM-基于角色视角的产品测试方案.md`，而是其 Spec Kit 化版本：操作细节（命令、矩阵、附录）以源文档为准；可测试需求与判定准入以本规范为准。

---

## 附录 A — 角色 × 权限速查（核心 API）

| 角色 | tickets | incidents | changes | configuration-items | audit-logs | users | tenants | sla |
|---|---|---|---|---|---|---|---|---|
| super_admin | RWD | RWD | RWD | RWD | R | RWD | RWD | RWD |
| tenant_admin | RWD | RWD | RWD | RWD | R(本租户) | RWD | R(本租户) | RWD |
| sd_manager | RWD | RWD | R | R | R | R | - | RW |
| engineer | RW(分配) | RW(分配) | RW(分配) | RW | - | R | - | R |
| approver | R | R | RW(审批) | R | - | R | - | R |
| security | R | R | R | R | R | R | R | R |
| end_user | RW(自有) | R(自有) | - | - | - | R(自己) | - | - |

> R=read，W=write，D=delete；空=无权限。

## 附录 B — Spec Kit 工件清单

- `specs/001-role-based-testing/spec.md`（本文档）
- `specs/001-role-based-testing/plan.md`（待 `/speckit-plan` 生成）
- `specs/001-role-based-testing/tasks.md`（待 `/speckit-tasks` 生成）
- `specs/001-role-based-testing/checklists/*.md`（待 `/speckit-checklist` 生成）
- 源文档：`docs/ITSM-基于角色视角的产品测试方案.md`（操作细节、API 矩阵、安全/性能/排期）

---

**Spec Status**: Draft → Ready（NC-001/002 已澄清，可进入 `/speckit-plan`）。
