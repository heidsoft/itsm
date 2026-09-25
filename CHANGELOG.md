# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Security

- **登录/刷新响应令牌收敛** — access token 和 refresh token 不再通过 JSON 响应返回，改为仅通过 HttpOnly cookie 下发，防止 XSS 窃取
- **MSP 跨租户访问加固** — `handlers/msp` 的 `GetCustomerTickets` 与 `AssignMSPTechnician` 现在校验 `AllowedCustomers`，MSP operator 只能访问已授权客户租户，越权访问 fail-closed
- **Webhook 出站 Header 注入防护** — BPMN Webhook Connector 不再透传用户配置的 `Host`、`X-Forwarded-Host`、`X-Forwarded-For`、`X-Real-IP` 等 12 个敏感 header，防止 SSRF 虚拟主机绕过与 IP 伪造
- **加密密钥派生升级** — `EncryptionService` 从确定性 `SHA256(secret)` 升级为 HKDF-SHA256 + 随机盐 + 版本字节；新密文格式 `version(1) || salt_len(2) || salt || nonce(12) || ciphertext`，旧密文自动 fallback 解密（迁移期兼容），为未来密钥轮换奠定基础
- **审计日志敏感字段掩码扩充** — `MaskSensitiveFields` 新增 `signing_secret`、`corp_secret`、`agent_secret`、`encrypt_key`、`app_key`、`bot_token` 等 connector 密钥字段的正则规则，防止审计日志泄露连接器凭据

### Tooling

- 租户管理页新增「初始化」列与状态抽屉：按需加载并缓存每租户的产品基线安装状态（基线就绪/未就绪组件、安装命令状态与尝试次数、模板版本、命令错误），`dead_letter` 可一键重放安装（复用运维命令重放入口、保留原幂等键）；状态接口新增 `commandId` 以支撑重放
- e2e 登录工具适配令牌 Cookie 化（不再读取已废弃的 `accessToken`、移除硬编码口令改走 `E2E_ADMIN_PASSWORD`/`ADMIN_PASSWORD`），并新增 `tests/e2e/tenant-provisioning.spec.ts` 固化「新建租户 → outbox 安装基线 → 状态就绪」全链路
- 产品表面棘轮基线上调 `service_go_files` 326→327、`bootstrap_app_lines` 1776→1805：分别对应组件 checksum 的内嵌 BPMN 摘要（`service/bpmn_template_digest.go`）与租户开通 outbox 接线（`POST /api/v1/tenants` 的 `tenant.bootstrap.install` 主链路），理由已登记在 `scripts/docs-gate/product-surface-baseline.txt`；部署与回归记录见 [docs/testing/deploy-regression-2026-09-24.md](./docs/testing/deploy-regression-2026-09-24.md)
- `initialize -action audit-tenants`：逐租户只读基线审计，输出每组件 verified/error 的 JSON 差异报告，作为存量前滚修复方案的输入（写入拦截 + total_changes 不变的测试锁死只读）
- **Gate C.6 产品口径漂移守卫**（[docs-gate/check-product-drift.sh](./scripts/docs-gate/check-product-drift.sh)）— 把"口径不一致 = 构建失败"作为机器守卫生效，对应 2026-09-22 审计的 5 项无守卫平面（C.6.1 成熟度双向闭合 / C.6.2 领域清单走目录 / C.6.3 零路由域包 / C.6.4 表面棘轮 / C.6.5 覆盖率口径披露）。CI 默认 hard；存量债务（变更管理口径冲突、department/root_cause/dashboard 三个孤儿包）已在 [product-drift-waivers.txt](./scripts/docs-gate/product-drift-waivers.txt) 登记 owner+到期日，**10-31 到期未解决自动反向 FAIL**。详见 [output/product-drift-overdesign-audit-2026-09-22.md](./output/product-drift-overdesign-audit-2026-09-22.md) §5
- **AGENTS.md 不再硬编码领域清单** — `handlers/<domain>/` 域包清单以目录为准，文档不再漂移；Ghost domain `cab` 已移除
- **ROADMAP 披露前端覆盖率口径** — 显式声明 jest `collectCoverageFrom` 仅含 `src/lib/**`，避免 80% 门槛被误读为产品覆盖率
- **删除空目录 `itsm-backend/handlers/team/`** — 未跟踪、无生产代码，纯负担

### Fixed

- **修复市场页恒为空** — 生产初始化新增 `marketplace-items` 组件：写入 5 条内置市场条目（连接器/技能/插件，状态 published、纳入版本账本 checksum 与逐项验证），存量环境可前滚补齐；此前生产 DAG 无该组件、列表按 published 过滤后恒为空。市场列表/详情同步改为返回 DTO（`MarketplaceItemResponse`，camelCase），不再直接序列化 Ent 模型
- 修复工单/事件 SLA 暂停与恢复的假成功：工单侧此前直接返回"SLA已暂停"却完全不落库，事件侧返回"尚未接入"，而底层 `SLAMonitorService.PauseSLA/ResumeSLA` 早已是真实现；现两域统一接线，暂停真实落库并记录原因、恢复顺延截止时间，重复暂停/未暂停恢复按 409 冲突、跨租户按 404 拒绝
- 知识/问题评论占位接口不再伪装成功：评论能力尚未落地（无存储模型），统一显式返回 5003 unready；此前知识侧返回假的评论对象、问题侧用空列表假装"暂无评论"
- 仪表盘事件/变更指标去掉模拟值：`AvgResolutionTime=240 分钟`、`SuccessRate=95.5%` 写死的假数据改为真实领域统计（事件解决时长均值、变更 completed/(completed+failed)），且指标改从事件/变更域取数而非工单表；无数据时如实为 0
- 修复 SLA 定义整段跳过导致存量环境前滚被永久卡死：改为按租户按条目 reconcile 补齐清单缺失条目、不覆盖已有条目（2026-09-25 演练库前滚实证：verify 要求 `Incident-P0-紧急`，而"已有任一行即跳过"一直拒绝创建，组件永远过不了自校验）
- 新增数据修复迁移停用两代旧命名的 SLA 定义（`SLA-P0-*`、`SLA-服务请求`、`SLA-变更`、`[Template] *`）：只按封闭名称清单置 `is_active=false`，不删行不改名，历史工单/SLA 违规引用继续可解析且保留原 SLA 条款，新工单不再可选；幂等可回滚，`NOTICE` 如实报告本次翻转行数
- 修复开通过产品基线的租户无法删除：`system-baseline-*` 基线归属账号只是模板行审计归属、不可登录，不再计入「租户下还有用户」删除守卫；真实用户仍阻止删除
- 修复新建租户接口 500：产品基线安装的 outbox 命令归属目标租户，而创建请求运行在操作者租户上下文，被租户写护栏按跨租户插入拦截（浏览器回归实测发现）；跨租户的 bootstrap/seed 类命令入队改走显式 system context 并留审计，测试 fixture 同步注册生产安全拦截器，杜绝只测裸客户端漏掉真实入口
- teams 与工单类型改为按租户按条目 reconcile：存量租户的部分集合可前滚补齐（此前"任一行存在即整段跳过"会让缺口永远补不上），不覆盖客户改名
- 租户创建不再留下"已建但无基线"的孤儿租户：`POST /api/v1/tenants` 在同一事务内投递 `tenant.bootstrap.install` outbox 命令，入队失败连同租户一起回滚；worker 消费后按命令租户重新加载校验并安装产品基线（复用平台初始化同一套组件，不复制第二套实现），新增 `GET /api/v1/tenants/:id/initialization` 只读接口如实报告逐组件验证结果与命令状态
- 修复生产环境数据库迁移失败问题：部分迁移脚本内嵌事务控制语句导致整批迁移中止
- 修复初始化失败被静默吞掉：迁移发现、账本调和与执行任一环节失败都会中止启动；组件写入、事务内校验、租约 fencing 与成功账本现在同一次提交，失败整体回滚，不再残留半成品基线
- 修复独立 `initialize -action verify` 依赖上一次 Apply 内存基线的问题：验证基线改为从代码内产品清单确定性派生，缺失菜单/权限/角色授权、跨租户授权和悬空授权均可被检出
- 修复审批组未随生产初始化创建：`identity-rbac` 现在写入审批组并纳入验证，JSON 种子配置中的 `groups` 段不再被合并逻辑丢弃
- 修复 SLA 告警规则按硬编码名称匹配：改为按租户实际 SLA 定义派生，引用缺失直接失败（此前静默跳过且日志谎报创建数量）
- 修复部门 `parent_code` 未落库导致组织树平铺：现在按 code 补齐父子关系，可前滚修复已安装环境且不覆盖客户改名
- 修复多租户基线种子撞全局唯一键：ticket_categories.code 与 tags.code 从全局 `Unique()` 改为 `index.Fields("tenant_id","code").Unique()`，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致；迁移脚本 `20260923_tenant_scope_baseline_unique_keys.sql` 先放旧约束 → 显式检测 `(tenant_id, code)` 重复 → 重建组合索引；迁移失败立刻停止，绝不留下半索引状态
- 修复租户基线种子运行时引用 `default` 租户硬编码：`Seeder.baselineTenant(ctx)` 抽象替出，平台初始化时仍落到 default 租户，按租户 provisioning 时落到目标租户；新增 `system-baseline-<id>` 不可登录审计账号（bcrypt 永不可通过的 `!` 哈希）让 template 表的 `user_id` 引用有主；避免「customer data 假扮 product template」的复制陷阱
- 修复 `SeedAll` 中新增种子函数签名升级后未对应 `attempt` 包装的 4 行旧调用：CI/types/standard-changes/ticket-tags 与原 ticket-types 一致用 `attempt("...", fn)` 包裹，失败仅 sugar.Errorw 不阻断，与 SeedAll 容错继续哲学一致
- 修复服务目录子项父目录缺失时被静默跳过（约 12/15 子项丢失）：现在初始化失败并给出具体条目，验证也逐条检查子项存在性
- 修复菜单初始化每次运行都重置运维设置的隐藏/停用状态
- 修复种子配置合并遗漏 `groups` 段：JSON 种子配置中的审批组定义此前被静默丢弃，现按整段替换内置默认并保留未覆盖段（含回归测试）
- 修复生产部署登录失败：docker-compose 默认 RLS 模式从 `enforce` 改为 `off`，避免未携带租户上下文的公共路由（登录/注册）返回 401
- **修复通知重复投递** — `notification_delivery_command_handler` 在 connector `Send` 成功后，若更新 `NotificationDelivery` 状态为 `sent` 失败，不再返回 error 触发 worker 重试（消息已送达），改为记录告警并返回 nil，避免同一通知被重复发送
- **修复 BPMN 任务完成审计丢失** — `CustomProcessEngine.CompleteTask` 现在将任务完成审计（`ProcessAuditLog`）写入与任务状态更新同一事务内，审计写入失败则整体回滚，杜绝"任务已完成但无审计记录"的不一致状态
- **修复单条连接器配置解密失败导致全部连接器不可用** — `PersistentConfigStore.LoadAll` 逐条容错：单条解密或反序列化失败跳过并记录 ID，不影响其余连接器正常加载；新增 `LoadAllWithFailures` 返回失败 ID 列表供运维排查
- **修复批量关闭/更新工单部分失败中断** — `BatchCloseTickets` 与 `BatchUpdatePriority` 改为逐条容错，单条状态机校验失败不阻塞其余工单，结果通过 `BatchResult` 返回成功数与失败 ID 列表
- **优化工单分析查询** — `GetTicketAnalytics` 使用 `Select` 仅加载 status/priority/created_at/updated_at 四列，合并二次查询为单次遍历，减少内存占用和 DB IO

### Changed

- `config/seed/default.json` 不再整段替换服务目录清单，内置 Go 清单成为唯一权威来源（原 8 条与内置子项引用冲突的目录已移除，"权限变更"并入账户申请子项）
- 初始化组件 checksum 现在覆盖代码内产品清单（权限目录、菜单规格、内置角色/审批组/角色授权、工单类型、BPMN 模板内容），修改这些定义会被版本账本察觉，无需手工升版
- 生产部署配置：`RLS_MODE` 默认值调整为 `off`（与后端安全默认对齐）。已配置 `.env.prod` 的部署不受影响

---

## [1.6.10] - 2026-09-20

### Added

- **BPMN 定时事件** — 完整实现 Timer Event 基础设施：调度引擎、BPMN 引擎集成、设计器属性面板、任务到期计时（dueDate）和定时启动事件（Timer Start Event），附带 Prometheus 监控指标和任务超时扫描
- **审批链运行时能力** — 新增加签（allowAddApprover）、委派（allowDelegate）、拒绝策略配置和动态层级适配，均通过审批链配置驱动、运行时生效
- **运维命令批处理** — 支持 Outbox 命令的批量重放与取消
- **钉钉/企微入站回调** — 入站消息回调 API 及幂等去重
- **BPMN 模板重载 API** — `POST /api/v1/bpmn/workflow-templates/:key/reload`，从已发布版本创建新部署，版本号自动递增
- **AI 工作流模板治理** — 租户隔离的模板目录 API 和 `/admin/workflows` 管理 UI，支持草稿、版本递增、发布前 Lint、发布与停用
- **CMDB ontology 自描述端点** — `GET /api/v1/cmdb/ontology` 返回租户 CMDB 的机器可读描述（CI 类型、关系词汇、AI 工具定义），AI Agent 可运行时发现 CMDB 契约
- **CI 可读编号** — 每个新建配置项获得 `CI-YYYYMM-NNNNNN` 唯一编号，支持按编号精确查询
- **一键演示数据集** — `make dev-seed-demo` 生成 8 个事件、2 个问题、3 个变更和 5 篇知识文章的演示数据
- **10 个管理页面使用指南** — 新增 `UsageGuideCard` 组件，为管理页面提供基于实际代码逻辑的操作指引
- **Swagger 文档重建 + CI 新鲜度门禁** — 重新生成 158 个 OpenAPI 路径，CI 自动检测文档漂移

### Changed

- **BREAKING: RBAC 授权平面收敛** — 统一权限码、补全预检映射、实现路由→权限码自动生成，关闭未授权写入漏洞；管理员/技术员角色数据库权限空集修复
- **BREAKING: 审批架构迁移** — 统一消费 BPMN 任务，消除双审批实例，ApprovalRecord 标记废弃，清理冗余 DTO
- **BREAKING: API 响应字段统一 camelCase** — Ticket/Incident/SLA/BPMN 响应不再暴露 snake_case 字段，外部集成需同步迁移
- **BREAKING: SLA/BPMN 查询参数 camelCase** — `ticketType`/`customerTier`/`startDate`/`endDate`/`timeRange` 替代原有 snake_case 参数
- **BREAKING: SLA/BPMN 监控端点标准信封** — 统一返回 `{code, message, data}` 格式
- **BREAKING: CI 关系类型受控词汇表** — 关系类型统一由 ent schema 管理，未知类型返回 400 而非静默持久化
- **BREAKING: Ant Design `direction` → `orientation`** — `Space` 组件统一使用 v6 API
- **服务请求 BPMN 集成** — 服务请求审批走 Outbox 事务投递
- **工单/事件写入路径加固** — 乐观锁、NULL 时间戳处理、错误语义修正、升级操作行级安全校验
- **变更/发布/DataScope 写入路径** — 行级安全校验补全
- **流程设计器修复** — 属性写入丢失、版本解析、菜单 404、模板完整性校验、Lint 审批节点校验
- **CSRF 全链路修复** — Cookie 安全属性与 Token 传递对齐
- **通知偏好** — 接入按事件类型偏好 API，修复虚假 API 处理和接收者广播排除
- **查询参数契约收尾** — 统一 camelCase，关闭残留 snake_case 不一致
- **角色域修复** — 移除虚假控件，API 契约对齐 camelCase
- **生产部署加固** — 数据库迁移与索引对齐、开发/生产隔离、Schema 变更脱敏、迁移脚本检查、`VERSION` 不可变要求、诊断端口绑定 loopback
- **基础设施加固** — AI vector/telemetry schema 版本化迁移、HTTP 响应缓存规范化、租户级 Raw SQL 执行器覆盖、Incident/CI 编号使用结构化审计事务
- **前端 UX 审计** — 工单列表列宽、创建按钮尺寸、折叠侧边栏 flyout、看板卡片溢出、工单分析页空白、管理后台仪表盘虚假指标等 20+ 项修复

### Fixed

- **AI 助手知识库无命中时丢失产品自知上下文** — RAG 降级链路现在注入 AI-Native ITSM 的真实能力边界
- **AI Native 工作流生产闭环** — AI BPMN 生成/预览/模板推荐走认证租户与 workflow 权限
- **AI 生成 BPMN 导入失败** — 自动为缺失 DI 的 BPMN XML 注入布局信息
- **表达式引擎 `avg()` 除零 panic** — 空数组或全零数组返回 0
- **会签投票逻辑修复** — `counterSign` 模式下投票结果计数修正
- **流程设计器属性丢失** — 保存时 `extensionElements` 丢失，6 处属性写入不一致统一修复
- **服务请求审批链双缺陷** — 配置未生效和审批人重复
- **审批人自动分配层级丢失** — `autoAssign=true` 时未携带 `level` 字段
- **流程管理 5 项修复** — 版本 API 路径、semver 解析、菜单 404、列表分页、模板导入 DI 补全
- **工单分配搜索崩溃** — `description` 字段缺失时 crash
- **三个用户可见 bug** — 工单列表分页失效、事件详情页关联 tab crash、变更审批评论丢失
- **杂项修复** — SLA 预测窗口校验、WebSocket 重连指数退避、知识文章链接 404

### Security

- RBAC 未授权写入关闭（incident/change/problem/service-request/release/knowledge），含批量回归测试
- CSRF 全链路修复（Cookie 属性 + Token 传递）
- DataScope / 事件 / 发布 / 工单升级写入路径行级安全校验
- 生产发布门禁加固：高/危依赖和 gosec 发现阻断发布

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

- [Development Guide](./docs/install.md)
- [Deployment Guide](./docs/deployment-optimization.md)
- [API Documentation](./docs/api-reference.md)

---

*Thank you to all contributors for your support!*
