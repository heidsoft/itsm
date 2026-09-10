# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

- **dev/prod 部署隔离修复（2026-09-10）** — ①`docker-compose.prod.yml` 顶部固定 `name: itsm-prod`，此前 dev/prod 两个 compose 文件共用目录名派生的项目名 `itsm`，`make prod-*` 与 README 的默认写法会与 dev 栈同服务名互相顶掉容器（dev up 后 prod 502，反之亦然；prod 卷因已 `external` 锁名而数据无损）；②prod 后端宿主机诊断端口改为可配置 `BACKEND_DIAG_PORT`（默认仍 `127.0.0.1:8090`），dev 栈同机运行 8090 时设 8091 即可避免 `Bind for 127.0.0.1:8090 failed`；`.env.prod.example` 同步说明。修复后实测 prod 7 容器全 healthy、`/api/v1/readyz` 200、admin 登录签发 token 正常，dev 栈不受影响。
- **prod 索引执行缺口核实闭环（2026-09-10）** — 对 `itsm-postgres-prod` 实测：`migrations/add_missing_indexes.sql` 41 条语句中 39 条生效（2 条为有意注释禁用），`changes`/`problems` 从"仅 PK"变为各 5 索引、`incidents` 12 索引，外部审计 #13/#14 的"脚本已写但从未执行"缺口在 prod 已闭环。残留（非本脚本范围）：68 张表仍仅 PK、60 张有 `tenant_id` 列但无租户前缀索引，需另立索引批次评估。

- **钉钉/企微入站回调 + 持久化入站去重 + 连接器健康度与凭据轮换（2026-09-09）** — ①新增 `handlers/dingtalk` / `handlers/wecom` 入站回调 handler（公开路由 `POST /api/v1/{dingtalk,wecom}/webhook/:instance_id`，解析/验签下沉到 `connector.builtin.*` 的 Receiver 接口：钉钉 HMAC-SHA256、企微 msg_signature=SHA1(token,timestamp,nonce,encrypt)，验签失败写 audit_log）；②新增 `connector.InboundDedup` 持久化入站去重器，用 `connector_inbound_dedups` 表的 `(tenant_id, connector_name, event_id)` UNIQUE 约束替代 in-memory nonce map，重启/多实例部署不再丢去重窗口（TTL 默认 5 分钟），feishu handler 同步接入；③`connector_configs` 新增 `last_success_at` / `last_failure_at`，`Manager.Send` 记录最近一次发送成败并截断 last_error，新增租户级 `TenantHealth` 端点避免普通租户看到全租户实例；④新增 `RotateSecret` 凭据轮换端点（grace period 0-168h 可反悔 + 全程 audit_log）。补 dingtalk/wecom receiver 验签单测与去重器单测。

- **Schema 变化脱敏副本 + 迁移 lint（2026-09-09）** — 新增 `migration/pii` 包与 `cmd/migration-lint` CLI：①`pii.New(strategy)` 注解挂在 ent schema 字段上（已为 `user.email/phone/name/password_hash` 落地 email/name/phone/api_key 策略）；②`pii.ExtractFromLoadedSchema` + `PolicyFromDescriptors` 抽取出 (table,column)→Strategy 映射；③`MaskedCopySQL` 生成 `create table mask_<t> as select` 风格的脱敏副本 SQL，email/phone/id_card 用 HMAC + 截断 16 hex，name 用 `regexp_replace` 保留首字、address 数字替换为 *，api_key/free_text 直接置 NULL；④`CheckNotNullDefaults` 解析 ALTER TABLE ADD COLUMN 校验 NOT NULL 必须带 DEFAULT，缺则 lint 退出码 1。SQLite 真机 round-trip 测试覆盖 up → 兼容读 → down → 数据保留 4 阶段。补 17 个单测覆盖 DDL 解析、annotation 提取、mask 表达式、quoted dollar tag、字典序 trim 等边界。

- **运维命令后台补齐：批量 replay / cancel + 命令类型汇总 + 滞留 lease 撤离（2026-09-09）** — `handlers/operations` 服务新增 `BulkReplay` / `BulkCancel` / `summaryByCommandType` 三个能力，handler 暴露 `POST /api/v1/admin/operations/commands/bulk-replay`、`POST /bulk-cancel`，List 接口支持 `commandType` / `aggregateType` 过滤并返回按 CommandType 的 `failureRate`、`stuckLeases` 计数。批量操作单事务内整批回滚或全提交，避免半生效歧义；每条命令分别写 `bulk_replay` / `bulk_cancel` audit_log。Lease 过期的 processing 命令才能撤离（`leaseExpired=true`），不会被正在跑的命令误杀。补 4 个 service 单测 + 2 个 handler 单测。
- **incident 外部告警渠道全部走 outbox（替换 fire-and-forget）** — `service/incident_alerting_service.go` 的 `CreateIncidentAlert` 不再 `go s.sendAlertNotifications(...)`；改为入箱 `commandbus.CommandDeliverIncidentAlert`，worker 调 `IncidentAlertingService.DeliverExternalAlert` 重载 alert 后实际触发 email/sms/slack/webhook。告警缺失时返回 nil（避免 worker 一直重试到 dead_letter）。新增 `IncidentAlertDeliveryCommandHandler` 并在 bootstrap 注册。
- **service_request 履约接入 outbox** — `service/provisioning_service.go` 的 `CreateTaskFromServiceRequest` 不再只创建 provisioning_task；现在同一事务里入箱 `commandbus.CommandExecuteProvisioningTask`，worker 调 `ExecuteTask` 完成实际交付。idempotency_key 按 task ID 锁定，replay 不会重复执行；手动 `POST /provisioning-tasks/:id/execute` 保留作为运维强制重试入口。新增 `ProvisioningTaskCommandHandler` 并在 bootstrap 注册。补 2 个原子性单测。
- **V1 fire-and-forget 兜底标记 deprecated** — `service/ticket_service.go:471/484/2494`、`service/incident_service.go:391/406` 仍是 V1 兼容分支；本轮仅替换 incident_alerting 的兜底，其余保留并加注释：主路径走 outbox，V1 分支在 `outboxEnabled=false` 时才生效，留作单元测试/旧组装兼容。后续 v1.7 视生产装配稳定度再清理。

- **三处用户可见缺陷修复（2026-09-07）** — ①AI 表单生成请求报错——`a2ui-api.ts` 裸 fetch 绕过 httpClient 缺少 CSRF/租户头被服务端拒绝，现对齐请求头（保留裸 fetch，因 A2UI 响应为根级 `messages` 结构与 httpClient 的 `data` 解析不兼容）；②审计日志用户列只显示 `#userId`——后端已返回 `userName` 但前端未消费，现优先显示姓名、缺失回退 ID；③BPMN 工单流转进度不显示处理人——`enrichBpmnProcessState` 此前仅取 `assignee`，任务只配 `candidate_users`/`candidate_groups` 时处理人为空，现按 ID/username/email 三形式解析候选人并经 GroupResolver 展开候选组（不可解析项降级跳过），补 3 个单测。
- **服务请求审批架构四项改进（2026-09-07）** — 在 P1 双缺陷修复基础上的架构级收口：①**角色词表单一源**——新增 `domain/role` 包统一 `users.role`/`roles.code`/审批 fallback 三套词表（`IsAdminLike`/`IsServiceRequestApprover` 判定函数 + 词表契约测试），service_request/capability/datascope 三处硬编码已迁移；②**RBAC 播种固化**——seeder 补 `service_request:approve` 权限定义（此前 DB 直改、代码缺失）、Roles 清单补 manager/it_admin、三个审批角色进 `rolePermissionMap`（幂等可重跑），修复「SQL 直改待固化」的漂移隐患；③**存量审批链自愈**——新增 `PendingApprovalRepairer` 启动任务，对 pending 请求按修复后逻辑重算审批人（幂等：已决策记录不动、结果相同不写回），配 3 条单测；④**审批路由语义收口**——`POST /:id/approval(s)` 从 `service_request:write` 改为专用的 `service_request:approve`，审批权限与建单/编辑权限解耦，为「只有审批权没有处理权」的角色模型铺路。
- **服务请求审批链双缺陷修复（2026-09-07）** — 多场景深测揪出并修复两个 P1：①审批人解析缺级回退——`resolveApproversForStep` 注释承诺「同部门→同角色→super_admin」三级回退，实际 `FindActiveUsersByRole` 仅做部门过滤，跨部门申请时审批链退化为 admin 独审，现补齐第二级「租户内同角色」回退；②审批角色 RBAC 权限整体缺失——`users.role`（manager/it_admin/security_admin）与 `roles.code`、资格判定 fallback 词表三处漂移，且 `service_request:approve` 权限定义后从未分配，导致三审批角色全部被 middleware 403，现已对齐词表并补齐 3 角色 × read/write/approve 权限。修复后全生命周期复验通过：ECS 申请建单→三级跨角色审批→provisioning→delivered。
- **E2E 验证收尾 + lint 重复 ID 规则补全（2026-09-07）** — 四项架构调整经 prod 栈端到端实测：incident 建单的 AutoAssign serviceTask 正常执行（原 dead_letter）；service_request 全流程带变量驱动走完（原 Feedback 静默卡死）；显式 `workflowDefinitionKey` 成功路由到指定流程（原 payload 恒空）。lint 补「重复 sequenceFlow id = error」规则（E2E 复验时发现该规则此前误报为已落地），并经镜像升级后复验拦截生效。前端模板库移除无消费方的 `ticketTypeCode` 死代码，改为 `TICKET_TYPE_TO_TEMPLATE` 码表映射 + 单测。常驻回归门禁 `TestBuiltinTemplates_Gate`（16 模板 lint 0 error + parser 富化守卫）落库。
- **内置 BPMN 模板完整性修复（16/16 全绿）** — 基于 2026-09-07 全量模板实测（12 处真实图缺口、24 处 outgoing 声明悬空、2 个 XML 语法损坏、2 处重复 sequenceFlow id、37 个 serviceTask 缺 `service_task_type`）：修复 cloud 两文件的 `ingoing→incoming` 拼写；为 service_request/ticket_general/ticket_assignment/problem/release/incident 及对应中文模板的断点逐一补线（反馈回环、升级回处理、SLA 告警收尾、审批拒绝路径等，均按业务语义选 targetRef）；重命名 problem_cn/service_request_cn 的重复 flow id；为全部缺失的 serviceTask 注入 `metaData service_task_type`。修复后脚本化复验：XML 可解析、声明↔连线一致、无非 endEvent 死端、无重复 ID，lint 门禁 16 文件 0 errors。
- **工作流引擎防静默卡死（P0-1 根治）** — `executeStep` 在 sourceRef 无出边且非结束事件时，先按元素声明的 `<bpmn:outgoing>` fallback 匹配一次 sequenceFlow（兼容历史模板声明悬空），仍无可走则将实例显式置为 `suspended` 并返回错误，彻底移除 `return nil` 静默成功路径。parser 后处理新增逐元素 outgoing 声明索引（`BPMNProcess.OutgoingDecls`）与 serviceTask 的 `metaData service_task_type` 解析（此前该元数据从未进内存）。
- **ServiceTask 处理器寻址修复（P0-2 根治）** — `serviceTaskReference` 现在 `service_task_type` 拥有最高优先级，`##WebService` 等 BPMN 标准实现标注（面向外部引擎）不再劫持内部 handler 寻址；`CallbackRegistry.RegisterHandler` 同时以 handler ID 与任务类型（`generic_task`/`incident_task`/`ticket_task` 等）双键注册；命令路径移除恒失败的 `GetHandler(task.GetType())` fallback（其值恒为 `"ServiceTask"`，只会掩盖配置错误并以误导性报错失败）。
- **workflowDefinitionKey 全链路修复（P0-3 根治）** — `handlers/ticket` 的 `CreateParams` 新增字段并在 handler 构造与 DTO 重建两处透传（此前双层丢弃，请求显式指定的流程 key 从未到达 outbox payload）；`ticket_service` 优先级语义改为「请求显式指定 > TicketType 配置 > 绑定解析」，消除与 resolver 注释的矛盾。
- **内置模板部署 lint 门禁（P0-4 根治）** — `LoadAndDeployTemplates` 部署前逐个过 lint，坏模板告警跳过而非阻断启动，并汇总返回失败清单；新增 lint 规则：outgoing 声明与 sequenceFlow `sourceRef` 一致性校验（悬空声明=error）、非结束事件出边完整性校验（死端=error）、重复 sequenceFlow id 检测。此前 2 个 XML 损坏的 cloud 模板曾直接入库。

- **审批链动态级别适配（i3 P0）** — `ApprovalChainStep` 新增 `conditionPriorities`（优先级白名单，大小写不敏感）、`conditionAmountMin` / `conditionAmountMax`（金额闭区间）三个条件字段，全部为空时与旧行为兼容。当工单 Priority/Amount 不匹配条件时整个 level 被标记为 `Skipped=true`、Status=`satisfied`，PendingLevel 自动跳过该层，避免「金额不匹配却被 block」。同一 level 内多 step 用 OR 语义合并（任一匹配即适用）。DTO/Mapper/CreateApprovalChain/UpdateApprovalChain 同步透传，覆盖 10 个表驱动单元测试（含大小写、边界、组合、OR 语义、向后兼容）。
- **审批人自动分配不再静默丢级（i2 P0）** — `TriggerApproval` 在 dynamic role 解析失败时记录完整的 `node_index/level/assignee_type/assignee_value/ticket_id/workflow_id` 日志，并先尝试降级到租户管理员（按 `super_admin → tenant_admin → admin → workflow_admin` 顺序）再决定跳过。`ApprovalTriggerRequest` 新增 `departmentId/teamId/projectId/approverFallback` 字段，`ticket_service` 显式从工单 `DepartmentID` 填充。`enrichApproverContext` / `resolveTenantAdminApprover` 对 nil client/req 完全 fail closed。
- **BPMN 模板重载 API（i4 P1）** — 新增 `POST /api/v1/bpmn/workflow-templates/:key/reload`，从当前 `is_latest=true` 已发布版本创建新的 `process_deployment` + `process_definition`，版本号自动 `bumpMinorVersion`（major 不可解析时回到 1.0.0），旧版本降级 `is_latest=false`，事务保证原子性。受 `workflow.update` 权限保护。响应 `ReloadWorkflowTemplateResponse` 含 `previousVersion/newVersion/deploymentId/processDefinitionId/source/reloadedAt`。

- Infrastructure hardening: AI vector/telemetry schema is now owned by versioned migration `020_add_ai_vector_observability_storage`; HTTP response caching stores raw bytes, canonicalizes query keys, and invalidates PATCH writes; knowledge-vector updates are persisted through the operational-command outbox before a worker reloads authoritative article state to index it; tenant-scoped raw-SQL executors now cover dashboard aggregates, approval-chain reads/writes, and independent approval-record writes; and global Incident/CI numbering now uses an explicit, structured-audited system-numbering transaction instead of direct raw DB transactions.

- Hardened the production release gate: production Compose now requires an explicit immutable `VERSION`, binds the backend diagnostic port to loopback, and no longer advertises an unconfigured host TLS port. High and critical dependency findings and gosec findings now fail the security workflow. The ACL manifest generator also recognizes multiline permission middleware and preserves 100% route coverage.
- Strict docs-gate now passes all five checks with filename-safe scanners and real release evidence anchors. Jest uses a timer-independent MessageChannel test adapter so React suites exit without native `MESSAGEPORT` handles. The production RLS stream adds migration `019_align_rls_tenant_variable` to align legacy policies with the runtime `app.current_tenant` GUC.
- Production smoke deployment rebuilt backend/frontend/AI images at `1.6.9`; nginx now forwards WebSocket upgrades through the API proxy. Frontend API and notification tests are isolated from CSRF bootstrap side effects, and the coverage gate is set to the measured 64.5% branch baseline while legacy UI coverage is expanded.
- Production Compose now assigns `itsm-frontend:${VERSION}` explicitly, matching backend/Worker/init/AI image tagging and preventing automatic `latest` or project-generated frontend image drift.

### Added

- **AI 工作流模板治理** — 新增租户隔离的模板目录 API 和 `/admin/workflows` 管理 UI，支持草稿创建/编辑、版本递增、版本历史、发布前 BPMN Lint、发布与停用；发布后的历史版本不可直接覆盖，模板管理动作受 `workflow` 权限保护。

- **CMDB ontology self-description endpoint (`GET /api/v1/cmdb/ontology`)** — Returns a machine-readable description of the tenant's CMDB: every CI type (with its parsed `attributeSchema` JSON and type-level attribute definitions), the full governed relationship vocabulary (13 types with name / description / direction / reverse / icon), the lifecycle + environment vocabularies, and the CMDB-scoped AI tools with their JSON-Schema argument definitions (where `list_cis.ci_type` enum is derived from the tenant's own CI types instead of a hard-coded list). This closes the biggest AI-Native gap: an agent can now discover the CMDB contract at runtime instead of guessing field names. Single-type attribute-definition failures degrade gracefully (log + empty list) rather than failing the whole introspection.
- **`ci_number` human-readable unique key for configuration items** — Every newly created CI (single and batch paths) receives `CI-YYYYMM-NNNNNN`, generated by the same three-tier mechanism used for incident numbers: Redis sequence with monthly sharding and global collision probing, DB `FOR UPDATE SKIP LOCKED` fallback, then a random suffix. The column carries a global unique index (deliberately not tenant-scoped, so numbers are unique across tenants) and is backfilled for historical rows by migration `018_backfill_ci_number`. `GET /api/v1/cmdb/cis` accepts `ciNumber` as an exact-match filter, and `CIResponse` exposes `ciNumber` so agents can address assets by stable business key rather than auto-increment id.
- **Frontend governed relationship vocabulary (`src/lib/cmdb/relationship-vocabulary.ts`)** — Single source for CI relationship types on the frontend, mirroring the backend's 13-type vocabulary, with `loadRelationshipVocabulary()` to override the static defaults from `GET /api/v1/cmdb/relationship-types` (fail-soft) and `relationshipLabel()` / `relationshipMeta()` accessors.

- **Core-domain Swagger coverage rebuilt + CI freshness gate** — Added swaggo annotations to the incident (36 handlers), problem (18), and change (21, including the 8-way `TransitionStatus` action routes and dual `risk-assessment`/`risk` paths) domains. Regenerated `itsm-backend/docs/` with swag v1.16.6: 158 OpenAPI paths (previous committed file was stale with 210 paths, many referencing routes removed by the controller→handlers migration). New `make swagger-gen` regenerates docs using the swag version pinned by `go.mod`. CI `api-contract-check.yml` gains a `swagger-docs-freshness` job that regenerates and fails on drift; docs-gate C.6 smoke-checks core-domain paths.
- **One-command demo dataset (`make dev-seed-demo`)** — New seed CLI (`go run -tags seed_demo .` in `itsm-backend/`) plus `config/seed/demo.json` seeding 8 incidents (spanning new → escalated → closed), 2 problems, 3 changes, and 5 knowledge articles under the default tenant. Business-record seeding only activates when `ITSM_SEED_CONFIG` points to a config containing the records arrays (production `default.json` stays record-free). Idempotent via fixed `INC-DEMO-xxxx` / `PRB-DEMO-xxxx` / `CHG-DEMO-xxxx` numbers. `ProblemSeed` / `ChangeSeed` structs gained optional `problem_number` / `change_number` fields; `config.yaml` `database.port` now honors `${DB_PORT:5432}` for host-to-container connections.
- **In-page usage guides for 10 admin pages** — New collapsible `UsageGuideCard` component (`src/components/common/UsageGuideCard.tsx`) ships grounded, button-level instructions on `/admin/cab`, `/admin/tickets/assignment-rules`, `/admin/tickets/automation-rules`, `/admin/connectors`, `/admin/vector-store`, `/admin/escalation-matrices`, `/admin/sla-templates`, `/admin/system-config`, `/admin/cmdb-types` and `/admin/workflows`. Every step was verified against the actual source of truth (assignment-rule JSON evaluator, `VECTOR_STORE_CONFIG` env handling, connector provision/test flow, idempotent SLA template install, CI-type inheritance semantics), replacing "方法论化" copy with concrete how-to-use guidance.

### Changed

- **BREAKING: CI relationship types are now governed by a single controlled vocabulary** — `relationship_type` previously drifted across five places: ent schema (13 documented values), `handlers/cmdb/production_service.go` (hard-coded 10 values, missing `impacted_by` / `owned_by` / `used_by`), `service/ontology_service.go` (its own reverse-relation map), the AI tool schema (a 6-value `enum`), and the frontend `RelationType` enum (12 values overlapping only partially). The vocabulary now lives in the ent schema as `CIRelationshipType` constants plus a `RelationshipTypeMeta` table; the relationship-types endpoint, ontology service reverse-mapping, AI tool enum, and frontend vocabulary are all derived from it. `POST`/`PUT` on CI relationships now validate the type and reject unknown values with `400` instead of persisting them silently — integrations writing custom relationship types must register them in the schema first.

- **BREAKING: API response fields are now camelCase across controllers and incident handlers** — Ticket, incident, SLA, and BPMN responses no longer expose snake_case fields such as `deleted_count`, `assigned_count`, `page_size`, `workflow_steps`, or `avg_resolution_time`. The new contract uses `deletedCount`, `assignedCount`, `pageSize`, `workflowSteps`, and `avgResolutionTime` to match the existing `dto.IncidentStats` / `dto.IncidentMetrics` types and the OpenAPI schema. External integrations that read snake_case fields must migrate in this release; the frontend `toCamelCase` compatibility layer in `src/lib/api/http-client.ts` is retained for one release as defense-in-depth and will be removed in the next minor.

- **BREAKING: SLA and BPMN monitoring query parameters are now camelCase** — `GET /api/v1/sla-policies/match` expects `ticketType` and `customerTier` instead of `ticket_type` and `customer_tier`; `GET /api/v1/sla-policies/compliance-rate` expects `startDate` / `endDate`; BPMN monitoring endpoints expect `timeRange` / `startTime` / `endTime`. The frontend `src/lib/api/sla-api.ts` sends camelCase directly. Existing callers using snake_case query strings must update their requests.

- **BREAKING: SLA templates and BPMN monitoring endpoints now return the standard envelope** — `controller/sla_template_controller.go` and `controller/bpmn_monitoring_controller.go` return `{code, message, data}` via `common.Success` / `common.Fail`. The previous ad-hoc `{"message": ..., "data": ...}` envelope is gone. HTTP status-code semantics are preserved (400 → `ParamErrorCode`, 404 → `NotFoundCode`, 500 → `InternalErrorCode`); only the body shape changed.

- **BREAKING: Ant Design `direction` prop removed from `Space` / `Steps` components** — Five pages (`admin/cab`, `ai/audit`, `email-intake/on-call`, `admin/config-inheritance`, `workflow/ticket-approval`) now use `orientation="vertical"` instead of `direction="vertical"`, matching the Ant Design v6 API. A new `itsm-frontend/tools/check-antd-direction.sh` CI guard fails the build if the legacy prop reappears in `src/`.

### Fixed

- **AI 助手在知识库无命中时丢失产品自知上下文** — 聊天原本把“未检索到文章”当作不知道自己服务哪个系统，导致针对本产品 AI 能力的提问退化成通用 ITSM 话术或反问用户系统名称。RAG 的普通与流式降级链路现在注入 AI-Native ITSM 的真实能力边界：明确 Pilot 状态、已实现的 RAG/分诊/摘要/分析/BPMN/受控工具能力、配置与权限依赖，以及尚未上线的跨源 AIOps 与自动修复；模型不可用的静态回退同样保留该说明。回归测试覆盖无知识命中时的提示词和静态降级文案。

- **AI Native 工作流生产闭环** — AI BPMN 生成、预览和模板推荐现在走认证租户与 workflow 权限；模板推荐复用内置 BPMN 模板目录，自动部署成功后回填真实 `processDefinitionId`/版本，Lint 错误阻断部署，AI/模板服务不可用返回明确的 `5003`，并补充模板与部署关联回归测试。
- **AI 生成 BPMN 应用到画布报“导入失败: no diagram to display”** — LLM 经常只输出 `<bpmn:process>` 流程节点而不带 `<bpmndi:BPMNDiagram>` DI 部分，bpmn-js `importXML` 会在渲染阶段抛错。新增 `src/components/workflow/bpmnAutoLayout.ts`，在 `import.parse.complete` 高优先级监听器（1500，比默认 1000 更早运行）里用 moddle + dagre 为缺失 DI 的 Definitions 自动注入 `BPMNDiagram / BPMNPlane / BPMNShape / BPMNEdge`（含合理坐标与 waypoint），再交给原 `importDefinitions` 渲染。包含 8 个单元测试覆盖已有 DI / 注入 DI / 空 Process / 非法 rootElements 等分支。

- **Structured business workflow candidates** — BPMN AI generation now returns a reviewable candidate contract with typed required form fields, domain ontology bindings, amount-tier approval rules for expense/procurement, SLA metadata, and an explicit confirmation gate. Server-side validation rejects malformed approval expressions before any automatic deployment; the existing workflow designer remains the only publication surface.

- **AI provider consistency across ITSM domains** — The Python guidance sidecar now applies the same runtime `LLM_PROVIDER`, `LLM_MODEL`, endpoint, key, and timeout configuration as the Go LLM gateway, including MiniMax's Anthropic-compatible Messages API. BPMN AI generation and preview no longer hard-code `gpt-4o`; they retain the deployment-configured provider model so MiniMax and local deployments work consistently. When vector embeddings are unavailable, knowledge retrieval continues through its existing permission-filtered keyword fallback rather than making an unsafe cross-tenant fallback.

- **Ticket-types table columns were unbounded and cramped** — only 3 of 10 columns declared a width, so the rest shared the remaining space arbitrarily and long codes/workflow keys squeezed neighbors. Every column now has an explicit width, long-text columns (name/code/workflow) enable `ellipsis`, and the table scrolls horizontally with the actions column pinned right.
- **Create-button size inconsistent across list pages** — `BusinessPageTemplate`'s primary action (used by incidents, problems, changes, assets) rendered with `size="small"` while every other page on the site uses the default button size; the template now matches the site-wide default.
- **Collapsed sidebar flyout showed icons without text** — menu labels are JSX, so antd could not derive the collapsed-state tooltip, and submenu flyout popups (portaled to `body`) inherited the inline-collapsed `title-content` hide rules. Items now pass an explicit string `title`, and a global override restores `.ant-menu-title-content` inside `.ant-menu-submenu-popup`.
- **Ticket kanban cards stretched unbounded by long content** — the six status columns split the 24-column grid evenly (`Col span={4}`), so on narrow screens columns collapsed to a few dozen pixels and long titles/descriptions (especially unbroken number/URL runs) pushed cards past the viewport with no way to read them. Columns are now fixed-width (300px) with a horizontally scrolling board, each column body scrolls vertically within `calc(100vh - 300px)`, and card title/description are hard-clamped to 2 lines via CSS `line-clamp` with a full-text tooltip. Unknown priority values fall back to a neutral badge instead of throwing on `undefined.color`, and the kanban search no longer crashes on missing `description`.
- **Ticket analytics rendered blank** — two stacked defects: the `/tickets` "分析" tab had no content branch at all (selecting it rendered nothing), and the standalone `/tickets/analytics` page had its `Tabs items` array emptied during an earlier refactor while all data fetching/remapping remained. The tab now navigates to the dedicated page (including `?tab=analytics` deep links), and the page rebuilds its overview (KPI row, created/resolved trend line, status pie and priority bar charts with percentage tooltips, processing-efficiency cards) plus team-performance and hot-category tables and the existing deep-analytics component; unavailable backend breakdowns (type distribution, team/category detail) render explicit `Empty` states instead of silent zeros.
- **Admin/workflow browser-test batch (production-stack defects)** —
  - _`/admin/approval-chains` crash_: the page read `response.data` via `httpClient.getPaginated`, but the backend returns the standard `{items,total,...}` envelope which `httpClient` already unwraps — the pagination object itself has no `.data`, producing `Cannot read properties of undefined (reading 'map')`. The page now reads `items` directly, and step rendering guards missing `steps` (`(chain.steps ?? []).map`).
  - _`/workflow/instances` total > 0 but empty list_: `WorkflowApi.getInstances` only consumed the legacy bare-array/`data` shape while the backend now returns `data: {items,total,page,pageSize,totalPages}`; the client reads `items` first with legacy-shape fallback. Related query binding was aligned to the camelCase contract (`pageSize`) in `common/pagination.go`, the audit-log/notification/process-trigger DTOs, and `bpmn-workflow-api.ts`, whose double-unwrap helpers were removed because `httpClient` already unwraps the envelope (this also made `process-definitions` reads silently return `undefined`).
  - _Workflow template import failures_: several built-in templates in `src/lib/workflow-templates.ts` shipped BPMN XML without a `BPMNDiagram` DI section, so bpmn-js threw "no diagram to display" on import, and the 请假审批 template had sequence flows but no `BPMNEdge` shapes — it rendered with nodes but no connection lines. Templates are now generated with complete DI (node shapes + edges).
  - _Duplicate workflow designers_: the legacy `/workflow/ticket-approval` parallel designer (577 lines) was removed and replaced with a server-side `redirect('/workflow/designer')` so old menu/bookmark links keep working; the workflow home quick-entry card now points at the designer directly.
  - _Squeezed list columns_: `/audit-logs`, `/notifications` (all + unread tabs) and `/workflow/audit` now declare explicit column widths with `ellipsis` on long-text columns, and notification list rows stop overflowing because flex children no longer default to non-shrinking `min-width:auto`.
  - _`/admin` dashboard fabricated metrics_: SystemOverview dropped invented trend badges (+12%), made-up progress values and a fake alert card; the four cards now trace to live APIs (`user-stats`, running instances, service-catalog total, ticket-stats) via `Promise.allSettled`, render `—` + an explicit "暂无数据" badge when a source is unavailable, and `SystemInfo` reads the real version from `GET /api/v1/version` (which now returns the standard `{code,message,data}` envelope — the previous raw `gin.H` body made the frontend unwrap `undefined`), plus honest Apache-2.0/repo/help links instead of a non-existent license field.
  - _`/admin/groups` style outlier_: replaced the last remaining `BusinessStatsGrid` usage (which contained two junk cards: a duplicate of the total and the page title) with the `Card + Statistic` pattern the sibling users/roles pages use.
  - _`/admin/permissions` misleading default and field offset_: the matrix previously rendered a fabricated all-enabled snapshot before any role was selected; it now starts fully disabled until a role loads. Missing CSS for `.enterprise-tree` left tree titles content-width, so switches/tags did not align across rows — global CSS now stretches `.ant-tree-title` to the row, and the antd v6 `<Spin tip>` (which renders nothing standalone) became a nested spinner with a label.

- **Browser-test defect batch (20 fixes across 35 modules)** — Closed the findings from the full-module browser regression pass:
  - _Sidebar/menu_: collapsed sidebar no longer drops flyout submenus (`popupOverflow`); breadcrumb root renders "Home" instead of the raw `/` path.
  - _Notifications_: header badge now reads the server-authoritative `GET /api/v1/notifications/unread-count` instead of counting a nonexistent `status` field (every notification used to render unread); the list request now sends the backend's real `size` param (it silently ignored `pageSize`); raw `{title,message,read}` contract is mapped via the shared `toTicketNotification()` mapper in the page, panel, and WebSocket handler; notification search no longer crashes on undefined content.
  - _Tickets_: detail-page related tab no longer crashes spreading a non-iterable; AI ticket creation no longer races auth-store rehydration; attachment upload uses relative paths through nginx instead of hard-coded `localhost:8090`; list columns (number/status/priority/created/updated) now sort server-side via a whitelist-validated `sortBy`/`sortOrder` contract.
  - _ITIL flows_: change approve/reject now persists the approval comment to the `change_approvals` record and closes the pending row on reject (previously only `approved` updated and `comment` was dropped); service-request approval sends the CSRF token (was 403); problems list honors the `search` param that the backend ignored; creating a problem no longer crashes on the post-save redirect; SLA edit form loads real response/resolution targets; service-catalog category filter enum matches the backend vocabulary; dashboard satisfaction rate no longer renders `-Infinity`/`Infinity`.
  - _Backend identity contract_: Known Error and Problem Investigation create DTOs dropped `binding:"required"` from `createdBy`/`tenantId`/`investigatorId` — these are derived from the auth context server-side, and requiring client self-reported identity made every UI creation call fail with 400.
  - _Write-operation idempotency_: change detail lifecycle actions and service-catalog approval actions now guard with a synchronous in-flight ref (button `loading` only takes effect after the render commit, leaving a double-submit window — approved fired 3× in testing); the server-side half is already enforced by `UpdateStatusCAS` + `WHERE status='pending'` on approval records, proven by `handlers/change/transition_semantics_test.go`.
  - _i18n_: incident status/source enums are translated; HTML entity `&#x2F;` is unescaped in rendered text; the permissions matrix resolves module labels through an explicit code→key map (11 of 23 modules rendered raw keys because the page upper-cased permission codes like `ticket`→`TICKET` while the dictionary uses `TICKETS`/`KNOWLEDGE_BASE`/`WORKFLOWS`), plus the missing en-US `TICKET_CATEGORY`/`TICKET_TYPE`/`LICENSE` entries; guarded by `admin/permissions/__tests__/permissions-i18n.test.ts` (48 assertions).

- **CMDB graph engine rendered raw English keys as edge labels** — `getRelationshipLabel` mapped relationship types through the legacy `RelationType` enum from `src/types/cmdb.ts`, whose 12 values only overlap the backend's 13-type vocabulary on six entries. Edges of type `hosts`, `connects_to`, `part_of`, `impacts`, `impacted_by`, `owns`, and `used_by` fell through to the `|| type` fallback and displayed as raw snake_case keys in the topology view. The label lookup now goes through `relationship-vocabulary.ts`. (The graph engine currently has no production call sites — this aligns it with the real API contract ahead of wiring it up.)
- **CMDB relationship endpoints returned 500 for invalid input** — `CreateCIRelationship` / `UpdateCIRelationship` mapped every error to `500 InternalErrorCode`, including client mistakes. They now use sentinel errors (`ErrInvalidRelationshipType`, `ErrCIRelationshipNotFound`, `ErrCIRelationshipSelfReference`, `ErrCIRelationshipDuplicate`) and map them to `400` / `404`, matching the state-machine error semantics used by the incident and approval modules.

- **Incident priority keyword dedup** — Removed the duplicate `"critical"` entry from the `inferIncidentPriority` keyword list in `handlers/incident/handler.go` so that `autoPriorityByKeyword("critical incident")` returns `"urgent"` exactly once instead of double-matching.
- **Handler layering violation in `GET /api/v1/incidents/stats`** — The handler no longer reaches into `ent.Client` from a Gin request context. It delegates to `service.GetStats` → `repository.GetStats`, where a single `COUNT(*) FILTER` aggregation replaces the previous seven concurrent goroutine queries and reduces database roundtrips from 7 to 1 (verified via `pprof`).
- **`avgResolutionTime` is now a real aggregated value** — `GET /api/v1/incidents/stats` computes `AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60)` for incidents with a non-null `resolved_at` (per tenant), instead of returning a hard-coded `0` for every tenant.
- **super_admin navigation restored** — `hasPermission` on the frontend now honors the `*` and `resource:*` wildcards that the backend issues to super_admin, so the admin section (users, roles, permissions, menus, tenants, workflow) renders again. `/auth/me` now returns the `permissions` list and `AuthGuard` forwards it on session restore, fixing menus disappearing after a page refresh.
- **Change status CAS** — Change status transitions use a single conditional `UPDATE` (`UpdateStatusCAS`) instead of read-validate-write, eliminating lost-update races between concurrent approvers.
- **Problem lifecycle error semantics** — Invalid problem state transitions (e.g. closing from `investigating`) now return `409/4090` business-conflict errors instead of `500` internal errors, matching the incident module's lifecycle error pattern.
- **Change approval chain cleanup SQL** — The terminal-state cleanup statement no longer references the non-existent `updated_at` column of `change_approval_chains`, removing a per-transition error log.
- **Command bus permanent failures** — Commands whose aggregate no longer exists (e.g. workflow/notification commands left behind by a deleted change) are dead-lettered immediately instead of retrying eight times; the shared in-memory test database is now isolated per test case.
- **Frontend dev server CSS parsing** — Removed 34 invalid `:global()` usages from `globals.css` that broke compilation under Lightning CSS (Next.js Turbopack); dev containers now persist `node_modules` and the `.next` cache in named volumes, cutting container restart recovery from ~170s to ~38s.
- **Production volume naming** — `docker-compose.prod.yml` pins the existing `postgres`/`redis` data volumes as `external` so switching the compose project name no longer mounts fresh empty volumes ("disappearing data"); MinIO stays auto-created since it is an optional `--profile storage` service with no historical volume.
- **Swagger global API info restored** — `swag init -g main.go` only reads general-info annotations (`@title` / `@version` / `@BasePath` / `@securityDefinitions`) from the main file, so the block previously placed in `swagger_meta.go` was silently dropped and the generated document had no title, base path, or BearerAuth security definition. The annotations moved to the `main.go` package comment (build tags untouched) and `swagger_meta.go` now only holds the blank `docs` import. The problem domain's annotations also gained the missing `@Security BearerAuth`, bringing core-domain operations to 100% security coverage.
- **Registration API contract** — Fixed the `fullName`/`displayName` mismatch between the register endpoint contract and consumers.

- **CMDB topology edges rendered without arrowheads** — `cmdb/topology` was passing `markerEnd` as a React Flow marker object; version 11.11.4's internal `getMarkerId()` concatenates every object field (`color=#1890ff&type=arrowclosed`) into the SVG `<marker id>` and its HTML-escaped `=`/`&` variants silently break the `<path marker-end="url(#…)">` reference, so edges drew as bare lines. Fix: pass `markerEnd` as a plain string id (`cmdb-arrow-<strength>`), inject the matching `<marker>` defs through a global hidden `<svg>` (bypassing React Flow's own `MarkerDefinitions`), and force antd v6 `Card` body to inherit `height: 100%` (topology canvas was previously collapsing to `0px`). Verified via DOM: 5 `cmdb-arrow-*` markers present, edge path emits a clean `url('#cmdb-arrow-medium')`.
- **AI ChatStream fake tool calls under MiniMax** — The MiniMax Anthropic-compatible provider only implements `Chat`, not `ChatStreamWithTools`, so the gateway silently degrades to plain streaming while the system prompt still instructs the model to call `list_tickets` / `list_cis` / `create_ticket`. The model then emits "正在调用 list_tickets 工具..." as plain text and the chat hangs forever. `LLMGateway.SupportsToolCalling()` now exposes the capability probe and `handlers/ai/service.ChatStream` skips tool injection when the bound provider cannot execute them, so the conversation falls back to a clean knowledge-only answer. Regression test: `TestLLMGateway_SupportsToolCalling_ReflectsProviderCapability` (4 subtests).
- **`/changes/create` route 404 and Sidebar submenu clicks silently dropped** — Two adjacent defects. (a) `route-config.ts` declared the change-create menu entry against `/changes/create`, but the Next.js page lives at `app/(main)/changes/new/page.tsx`; path corrected to `/changes/new`. (b) `MenuItems.tsx` relied on antd v6 `items[].onClick` alone for SubMenu children, which is dropped when the parent title-click and item-click race in inline mode, so `变更管理 → 新建变更` never navigated. Child and top-level labels now attach `onClick` directly on the label element and prefer `item.path` over `item.key`, mirroring the parent-item pattern so every SubMenu child reliably routes.
- **AI triage confidence value invisible** — `AISuggestionPanel` renders the confidence via `<Progress size="small" />` and antd v6 hides the inner label by default at this size; the user saw a colored bar with no percentage. An explicit `format={(p)=>\`${p}%\`}` now keeps the number visible regardless of theme.

### Tests

- **Business flow regression suite** — Added `output/dev_business_flow_test.py` (27 assertions: full incident lifecycle, problem lifecycle, change reject path, state-machine negative cases returning 409 not 500, forged-token negative cases, notification/dashboard linkage) alongside the 63-item multi-domain integration suite; both run green against the dev stack.

- **Bug-fix regression coverage for the seven v1.6.x hardening fixes** — Added unit tests: `handlers/incident/repository_impl_test.go` covers Bug 1 (`autoPriorityByKeyword` single-match) and Bugs 3+4 (`service.GetStats` table-driven, no direct `ent.Client` access, `avgResolutionTime > 0` after resolved events, cross-tenant isolation); `controller/sla_template_controller_test.go` covers Bug 2 (standard `{code, message, data}` envelope, status-code mapping, no raw `ctx.JSON`); `controller/sla_policy_controller_test.go` covers Bug 6 (static source check that no snake_case `ctx.Query("...")` remains and the camelCase counterparts are present); `controller/ticket_controller_test.go` covers Bug 5 (`BatchDeleteTickets` returns `deletedCount` and never `deleted_count`); `itsm-frontend/tools/check-antd-direction.sh` is the Bug 7 CI guard that fails if `direction="vertical"` returns to `src/`. All new tests pass under `go test ./...`.

### Documentation

- Reworked the open-source capability guide around user roles, executable business journeys, TicketType-driven dynamic forms, maturity boundaries, deployment acceptance, and explicit non-goals.
- Updated README authentication and ticket examples for HttpOnly cookie sessions, CSRF protection, camelCase DTOs, string priorities, and runtime TicketType selection.
- Resynchronized the roadmap with the v1.6.x hardening line and corrected obsolete BPMN goroutine claims now that ticket and incident production wiring uses persistent command/outbox execution.

---

## [1.6.9] - 2026-08-20

### Added

- **Problem Management expansion** — Added Problem trends, hotspots, SLA lookup, and per-problem comment read/write endpoints so operators can analyze recurring issues and discuss root cause without leaving the Problem record.
- **Incident comment deletion** — Added `DELETE /api/v1/incidents/:id/comments/:commentId` for tenant-scoped comment cleanup, mirroring the existing create/list semantics.
- **Knowledge recommendations** — Replaced the previous stub implementations of `/api/v1/knowledge/recommendations` and `/api/v1/knowledge/recent` with tenant-scoped queries that return real published articles with proper permission filtering.

### Fixed

- **Frontend Edge Runtime compatibility** — Replaced `Buffer.from(...)` calls in `src/middleware.ts` and `src/app/api/[...path]/route.ts` with an `atob()`-based JWT decoder so the Next.js middleware no longer raises `Code generation from strings disallowed for this context` in Edge Runtime (was breaking `itsm-frontend-prod` with HTTP 500).

### CI

- **Backend lint toolchain** — Pinned `gofumpt` to `v0.7.0` and cached `~/go/bin` between CI runs to make formatting results reproducible and shave install time.

### Tests

- **CMDB Service layer coverage** — Added 9 table-driven test functions covering CloudService / CloudAccount / CloudResource CRUD, Reconciliation (bound / unbound / orphan / unlinked / mixed), and Discovery operations via a `mockRepository` that isolates the service from the database.

---

## [1.6.8] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.12 to include the latest TLS, X.509, and MIME-header security fixes required by the production vulnerability gate.

---

## [1.6.7] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.10 as an intermediate production security baseline.

### Fixed

- **SLA calendar enforcement** — SLA deadlines now apply each definition's configured business calendar consistently across creation, lookup, and overdue detection; the prior unused implementation was removed.
- **Backend quality gate** — Removed obsolete change-status validation code so the release static-analysis gate has no dead-code findings.

---

## [1.6.6] - 2026-08-03

### Fixed

- **Cookie-only authentication test contract** — Updated API integration and HTTP client tests to assert credentialed browser requests without exposing HttpOnly session cookies through JavaScript `Authorization` headers.

---

## [1.6.5] - 2026-08-03

### Security

- **Production dependency remediation** — Updated frontend production dependencies and lockfile overrides for known Axios, DOMPurify, lodash, PostCSS, Sharp, and UUID advisories; `npm audit --omit=dev --audit-level=high` now reports zero vulnerabilities.
- **Package-manager integrity** — Removed the obsolete `pnpm-lock.yaml`; the frontend is governed solely by the committed npm lockfile, matching the release CI configuration.

---

## [1.6.4] - 2026-08-03

### Fixed

- **CI formatting gate** — Applied the repository's `gofumpt` formatting standard to backend source and tests, restoring the required backend release pipeline gate.

---

## [1.6.3] - 2026-08-03

### Fixed

- **Production authentication hardening** — Browser sessions now use HttpOnly, SameSite=Lax cookies exclusively. Access and refresh tokens are no longer exposed in JSON responses or read by frontend JavaScript; refresh token rotation is performed through the cookie transport.
- **Secure proxy cookie handling** — Authentication cookies now receive the `Secure` attribute when the request is HTTPS or terminated by a trusted HTTPS reverse proxy.
- **CSRF refresh coverage** — Both supported refresh routes are explicitly covered by the CSRF session-refresh exception.
- **Initialization readiness** — `/api/v1/readyz` now requires the latest registered post-schema migration rather than a stale, hard-coded migration version.
- **Workflow validation contract** — Removed the frontend call to an unimplemented validation endpoint; workflow designer preflight validation is deterministic and local, while create/update remain server-validated.

### Changed

- **Release metadata** — Frontend package, backend version endpoint, system configuration endpoint, and GA-readiness report now identify the release as `1.6.3`.

---

## [1.6.0] - 2026-08-01

### Fixed

- **Critical: Form Boundary Issue** - Fixed TicketTypeFormModal where approval/SLA tab fields were outside `<Form>` component, causing form validation failures.
- **Critical: Auth State Persistence** - Removed `isAuthenticated` from Zustand persist partialize to prevent false login state after browser refresh.
- **High: Timer Leak in Export/Import** - Fixed memory leaks in TicketCategoryExport and TicketCategoryImport with proper useRef cleanup on unmount and error paths.
- **High: ITIL Status Guard** - Added readonly protection on change/edit and release forms to prevent editing approved/completed records.
- **High: Suspense Boundary** - Added Suspense wrapper for useSearchParams in tickets page to fix CSR bailout.
- **High: Query Key Normalization** - Fixed useTicketsQuery to only include request params in queryKey, not response data.
- **High: NaN Route Guard** - Added Number.isFinite check for marketplace/[id] route parameter.
- **High: Error Handling** - Fixed workflow-api to throw errors instead of silently returning empty arrays.
- **Medium: useFormMemory** - Added debounce (500ms), enabled flag for edit mode, and dayjs serialization.
- **Medium: CIEditorForm** - Added min/max/precision props to InputNumber for numeric fields.
- **Medium: ChangeDetail Null Guard** - Added null check for createdAt before dayjs conversion.

### Changed

- **Ant Design destroyOnHidden** - Added to ProblemInvestigationTab modals.
- **SchemaField Type** - Added validation property with minValue/maxValue/precision/pattern.

---

## [1.5.2] - 2026-07-31

### Fixed

- **SLA Compliance Statistics** - Fixed negative compliance numbers caused by mismatched time scopes between total ticket count (30 days) and violated ticket count (all time). Backend now uses `HasTicketWith` edge predicate for consistent scope.
- **Timezone Inconsistency** - Fixed dashboard activity timestamps using `time.RFC3339` format instead of `Format("2006-01-02 15:04:05")` which browsers parse as UTC.
- **TicketDetail Duplicate Request** - Fixed double API call on ticket detail page by removing `fetchTicket`/`fetchSLAInfo` from useEffect dependency arrays.
- **Login Error Message** - Login page now shows actual backend error messages (e.g., "invalid credentials") instead of generic "登录失败".

### Changed

- **Ant Design TabPane Deprecation Migrated** - Converted all `Tabs.TabPane` / `<TabPane>` usage to `items` prop pattern across 8 files: analytics, applications, NotificationCenter, TicketTypeFormModal, IncidentManagement, FieldDesigner, profile, dashboard.
- **Space direction → orientation** - Migrated 6 instances of deprecated `Space direction="vertical"` to `orientation="vertical"`.
- **destroyOnClose → destroyOnHidden** - Migrated 1 instance in ApprovalTimeline component.
- **alert()/confirm() → antd message/modal** - Replaced native browser dialogs in marketplace and installations pages with antd `App.useApp()` message/modal.
- **console.log Cleanup** - Removed debug console.log from TicketDetail and BPMNDesigner.

---

## [1.5.0] - 2026-07-30

### Added

- **Connector/Skill Manifest Hardening** - All official connector manifests now declare `version`, `requiredPermissions`, and a deterministic SHA-256 `checksum`; registration is fail-closed (incomplete manifests are rejected at startup). Skill manifests share the same validation and checksum convention. Market API exposes `isOfficial` / `requiredPermissions` / `checksum`.
- **Post-Schema Migrations 008-010** - Initialization ledger (008), PostgreSQL RLS tenant isolation (009), ticket types in `itil-core` transaction (010).
- **Bootstrap Token** - One-time hashed bootstrap token with TTL, concurrent-consumption protection, replay defense, and break-glass flow for first-admin creation.
- **Endpoint ACL Manifest** - Versioned ACL manifest with 100% static coverage gate over protected routes (route-ACL-permission-menu).
- **Fencing Token Hardening** - Owner/token/lease re-verified inside the committing transaction; stale-writer prevention proven via PostgreSQL fault-injection tests.
- **Audit Routes** - New audit trail API endpoints with tenant isolation support
- **CI Attribute Validator** - Moved from handlers to service layer for better separation of concerns
- **Operator Context** - Enhanced audit trail with operator context tracking

### Fixed

- **Ant Design v6 Select Compatibility** - Replaced deprecated `<Select><Option>` child pattern with `options` prop across 100+ files. Fixes issue where clicking Select dropdowns had no response in antd v6. Affected modules: Ticket, Incident, Problem, Change, CMDB, Workflow, SLA, Service Catalog, Admin, and Reports pages.

### Changed

- **CMDB API Route Convergence** - `/api/v1/cmdb/*` is now the canonical prefix for all CMDB endpoints (CIs, CI types, relationships, relationship types, topology, impact analysis, change history, stats). Frontend API clients (`cmdb-api.ts`, `cmdb-relationship.ts`) have been switched to the canonical prefix. Added `GET /api/v1/cmdb/relationship-types` to the canonical tree. Note: change history is `GET /api/v1/cmdb/cis/:id/history` (the old `change-history` suffix only exists on the deprecated alias).
- **CMDB Multi-Tenant Isolation** - Added `tenant_id` field to CI relationships, configuration item history, and discovery sources. Added tenant-aware backfill migration.
- **CI Attribute Validation** - Migrated from `handlers/cmdb/attribute_validation.go` to `service/ci_attribute_validator.go`.

### Deprecated

- **`/api/v1/configuration-items/*` routes** - Kept as a compatibility alias for clients not yet upgraded. No new endpoints will be added under this prefix; removal will be evaluated after a regression period.

### Migration Notes

- **CMDB tenant backfill**: Run `itsm-backend/migrations/20260610_cmdb_tenant_id_backfill.sql` on existing databases to populate the new `tenant_id` fields.
- **Preset CI types**: Enable with `itsm-backend/migrations/20260611_enable_preset_ci_types.sql`.
- **Audit routes**: New endpoints are protected by existing JWT + RBAC + tenant middleware.

---

## [1.0.0] - 2026-03-07

### Added

#### Core ITIL Modules
- **Ticket Management** - Complete ticket lifecycle management with creation, assignment, tracking, and closure; built-in SLA management, priority handling, comments and attachments
- **Incident Management** - Incident discovery, logging, classification, escalation; real-time monitoring and alerts
- **Problem Management** - Root Cause Analysis (RCA), Known Error Database, problem resolution tracking
- **Change Management** - Change requests, risk assessment, multi-level approval workflows

#### Service & Knowledge Base
- **Service Catalog** - Service request templates, self-service portal, SLA management
- **Knowledge Base** - RAG intelligent search, knowledge categorization, FAQ management, vector retrieval

#### Workflow Engine
- **BPMN Workflow** - Visual process designer, approval workflow automation
- **Task Management** - Workflow task assignment and tracking

#### AI-Powered Features
- **Intelligent Classification** - Auto-identify ticket type, priority, impact scope
- **Auto-Summary** - AI-generated ticket/incident summaries
- **RAG Knowledge Base** - Vector search-based intelligent knowledge recommendation
- **Smart Suggestions** - Recommended solutions, similar tickets

#### User & Permissions
- **Multi-Tenant Architecture** - Complete tenant isolation and management
- **Role-Based Access Control** - RBAC permission system, fine-grained access control
- **User Management** - User CRUD, team and department management

#### SLA Monitoring
- **SLA Definition** - Service Level Agreement configuration
- **Real-Time Monitoring** - SLA compliance rate tracking
- **Alert Rules** - SLA violation alerts and notifications

### Technical Stack

| Category | Technology |
|----------|------------|
| Backend | Go 1.25+ / Gin / Ent ORM |
| Frontend | Next.js 15 / React 19 / TypeScript / Ant Design 6 |
| Database | PostgreSQL 17 / Redis 7 |
| Deployment | Docker / Docker Compose |

### Quick Start

```bash
# Docker Compose (Recommended)
git clone https://github.com/heidsoft/itsm.git
cd itsm
make dev-up

# Access
# Frontend: http://localhost:3000
# Backend: http://localhost:8090
# API Docs: http://localhost:8090/swagger

# Login
# Username: admin
# Password: admin123
```

### Known Limitations

- Mobile PWA features are under development
- Enterprise integration features (LDAP/SSO) planned for future releases

### Documentation

- [Development Guide](./docs/getting-started/install.md)
- [Deployment Guide](./docs/DEPLOYMENT_OPTIMIZATION.md)
- [API Documentation](./docs/api/API_REFERENCE.md)

---

*Thank you to all contributors for your support!*
