# ITSM 开源商业化 P0-P2 建设蓝图

## 1. 目标与产品结论

目标是把当前“能力广但首次价值、跨域闭环和证据分散”的系统，收敛为可重复安装、可安全升级、可演示核心差异、可持续验收的企业级开源 ITSM。

本蓝图不新增平行业务引擎，遵守以下边界：

- BPMN/process definition、instance、task、variable、history 是审批与流程编排的权威机制；通用 `ApprovalChain` 只作为可配置模板或兼容入口，不再形成第二套运行时审批事实。
- `handlers/<domain>` 与 `controller/ + service/` 按现有路由归属演进，不跨包直接调用 repository implementation。
- 所有 HTTP DTO 使用 camelCase，Controller 不暴露 Ent。
- 所有租户资源、后台任务、回调、导入导出和 AI/RAG 查询均 fail closed。
- AI 默认提供建议，不静默执行高风险动作；所有采纳、拒绝和工具调用可审计。
- 初始化只安装 P0/T0 产品模板，不创建 R0 客户业务数据；演示数据独立且显式启用。

## 2. 现状证据与关键缺口

### 2.1 已有基础

- `config/seed/default.json` 已声明 8 个服务目录模板、服务请求/变更审批工作流和默认 process binding。
- 初始化引擎已有 component DAG、账本、lease、heartbeat、fencing、校验和与 readiness 基础。
- 事件、问题、变更、服务请求已有领域服务和一定数量测试；前端已有 business-flows、flows、roles、multi-tenant、golden 套件。
- CMDB 已有商业化专项蓝图和阿里云发现/对账基础。
- 连接器已有 manifest/checksum/requiredPermissions、部分 lifecycle 与 marketplace 基础。
- OperationalCommand outbox 已承担工作流启动和通知等可靠异步命令。

### 2.2 当前阻断

1. 服务目录模板被放在 `extension-core`，初始化 readiness 未证明“目录 + 流程绑定 + 可申请”整体可用。
2. `seedServiceCatalog` 以“租户存在任意目录记录”整体跳过，缺失模板不会自动补齐，也无法表达客户覆盖与系统托管字段。
3. 通用 ApprovalChain、approval workflow、BPMN task/decision 并存，若不冻结权威边界会产生状态分叉。
4. 生命周期、权限和租户测试分散，缺少同一套状态矩阵和跨租户 CRUD/direct-ID/export/background 认证。
5. CMDB、变更和事件分别具备能力，但影响快照、变更门禁、事件关联和审计结果尚未成为一个产品闭环。
6. 连接器配置、密钥解析、健康检查、生命周期和审计仍需统一用例边界。
7. 发布证据是历史快照，版本源可能不一致；缺少当前 revision 的一键诊断与首值验收。
8. AI 建议缺少统一的 suggestion/evaluation 产品契约与可运营指标。

## 3. 依赖图与实施顺序

```text
PR-0 商业化契约、审批写路径清单与证据基线
 └─ PR-0.5 托管模板 expand schema、回填与账本迁移
     ├─ PR-1 服务目录首次价值认证（P0）
     └─ PR-2a/2b/2c ITIL 状态机、关联租户门禁、真实 PG RLS（P0）

PR-1 + PR-2a + PR-2b + PR-2c ──> PR-3 CMDB→变更→事件影响闭环（P1）
PR-0 + PR-2c ───────────────────> PR-4 连接器治理统一（P1）
PR-1 + PR-2c + PR-4 ───────────> PR-5 可重复发布与诊断（P1）
PR-0 + PR-2a + PR-2c ──────────> PR-6 AI 建议与评估闭环（P2）
PR-3 + PR-4 + PR-5 ────────────> PR-7 商业化 Core Golden Gate
PR-6 ──────────────────────────> PR-8 AI Beta Gate
```

PR-1 与 PR-2a 可并行；PR-3 与 PR-4 可并行；Core Golden Gate 不依赖 AI 质量功能，AI disabled/degraded 时仍必须满足租户、审计和 fallback 安全基线。

## 4. 分步建设计划

### PR-0：冻结商业化能力契约与证据基线

**分支**：`codex/commercial-00-contracts`

**目标**：建立 capability、lifecycle、tenant、audit、readiness 和版本证据的共同语言，防止六条工作流各自发明状态。

**任务**

- 建立 ITIL 状态矩阵：incident、problem、change、service_request 的状态、允许动作、角色、必填字段、终态与重开规则。
- 建立跨租户资源矩阵：list/search/get/create/update/delete/export/background/callback 每一类的预期行为。
- 盘点每一条 ApprovalChain、ApprovalWorkflow、ServiceRequestApproval、ChangeApproval、ProcessApprovalDecision 写路径，标记 owner、读者、在途记录和 API/UI 入口。
- 冻结审批权威边界并形成可执行迁移：先禁止创建新的 legacy runtime instance，再以 feature flag 做 dual-read shadow compare；在途实例采用 drain/grandfather，映射完成前不强制切换；定义一致性 invariant、切换点和 rollback window。
- 定义 capability contract：`buildCapability`、`deploymentReadiness`、`tenantReadiness`、`actorPermission`、`degradedReasons`。
- 定义版本事实优先级和 evidence manifest：Git SHA、release version、前后端版本、镜像 digest、迁移版本、模板版本、测试证据时间。

**验证**

- 契约文档覆盖六条工作流的状态、租户、审计和故障语义。
- 静态测试能发现新增路由未进入 ACL/tenant 矩阵。

**回滚**：纯契约和测试清单，可直接回滚；不得引入运行时行为变化。

### PR-0.5：托管模板数据模型与初始化账本迁移

**分支**：`codex/commercial-005-managed-template-schema`

**目标**：为 PR-1 的逐项 reconcile 提供可升级、可冲突、可回滚的数据事实，避免仅以名称猜测系统/客户 ownership。

**任务**

- 对 ServiceCatalog、ProcessBinding 及必要模板增加 `sourceType/sourceKey/sourceVersion/lastAppliedHash/managedFields/localModifiedAt` 或独立 managed-record ledger；定义 system-owned、customer-owned、mergeable 字段。
- expand migration 后按确定性 fingerprint 回填；无法判定的记录标记 conflict，不自动接管。
- 在冲突报告清零后建立 tenant + source key 唯一约束；客户自定义记录不要求 source key。
- 定义三方合并：last applied spec、当前租户值、新 spec；客户改过系统字段时进入 degraded/manual resolution。
- 拆分 `extension-core` 账本：定义 ledger alias/adoption、checksum/version bump、旧新 initializer 互斥、default/new/existing tenant rollout、canary/resume 和 ownership 移交完成点。
- 每个 schema 变化包含 expand/backfill/read switch/write switch/contract 阶段、N-1 兼容、rollback cutoff 和恢复演练。

**退出标准**：PR-1 不再依赖名称作为永久稳定键；既有成功 `extension-core` 安装可无重复地 adopt 新 component；旧二进制在 contract 前仍可运行。

### PR-1：服务目录首次价值与初始化认证（P0）

**分支**：`codex/commercial-01-first-value`，依赖 PR-0.5。当前按租户+名称逐项补缺属于 PR-1a 兼容修复，只解决“任一记录导致整体跳过”，不代表托管模板升级完成。

**目标**：全新开源安装完成后，管理员无需手工造数据即可让终端用户提交第一个服务请求；生产初始化不创建业务请求。

**任务**

- 将服务目录模板从笼统 `extension-core` 拆为可独立计划、应用、验证和 readiness 展示的 `service-catalog-core` component；只硬依赖 identity-rbac、workflow-core、sla-core。CMDB enrichment 使用独立可选 component/capability，不要求初始化引擎支持隐式可选依赖。
- 用 PR-0.5 的 stable source key 对 8 个默认模板逐项三方 reconcile；保留客户自定义记录并按字段 ownership 处理冲突。
- 为模板增加最小可用 form schema、交付说明、SLA/流程绑定语义和清晰的启用状态；不写死租户用户 ID 作为审批人。
- 将 `requiresApproval` 映射到 BPMN service-request binding；ProcessDefinition、deployment、binding 同样按托管 stable key reconcile，不得因存在任意 binding 而整体跳过。初始化验证必须证明定义已部署、binding 可解析、候选角色存在。
- 明确 ApprovalChain 兼容策略：若 UI 仍展示审批链，展示 BPMN 模板的只读/转换视图，不创建第二套运行时 decision。
- tenant readiness 验证：目录数量、稳定键完整性、至少一个免审批目录、至少一个审批目录、service_request process binding、必要角色/权限。前端 route/build/API 可见性属于 deployment smoke，不由数据库 initializer 伪造。
- 增加 fresh-install/upgrade 集成测试：default tenant、新租户、既有租户、组件拆分 adoption、部分失败恢复；空库初始化两次结果相同；删除一个托管模板后重跑可补齐；客户自定义目录不被覆盖；R0 service_request/process instance 保持为零。
- 增加浏览器 first-value 流：end user 浏览目录 → 打开动态表单 → 提交 → 免审批项进入 fulfillment，审批项生成 BPMN task → 刷新仍可见。

**验证命令**

```bash
cd itsm-backend
GOTOOLCHAIN=auto go test ./pkg/seeder ./internal/initialization ./handlers/service_catalog ./handlers/service_request -count=1
cd ../itsm-frontend
npm run type-check
npx playwright test tests/e2e/flows/flow-4-service-request.spec.ts --project=chromium --workers=1
```

**退出标准**

- fresh install 的 readiness 明确报告 service catalog 可用；重复初始化不重复、不覆盖客户数据。
- 用户能完成第一个服务请求，审批状态只由 BPMN 运行时驱动。
- 任一关键绑定缺失时 fail closed/degraded，不能显示“ready”。

**回滚**：保留旧 component 读取兼容一个版本；模板升级只 forward-fix，不删除客户记录。

### PR-2a/2b/2c：ITIL 状态机与跨租户权限集成矩阵（P0）

**分支**：按 `codex/commercial-02a-lifecycle`、`02b-association-tenant-guards`、`02c-pg-rls-background` 拆分，避免一个 PR 同时改四域、RLS 和后台执行。

**目标**：事件、问题、变更、服务请求的生命周期和租户隔离可由集成测试证明，而不是依赖前端按钮约束。

**任务**

- PR-2a 逐域小 PR 建立后端 transition validator，先 contract tests 后 feature-flagged behavior switch；覆盖正常、非法跳转、终态、重开、并发冲突、必填字段和角色门禁。
- PR-2b 对 incident↔problem/change、service request↔catalog/CI、change↔CI/release 增加双边租户校验，并覆盖 list/search/direct-ID/update/delete/link/export。
- PR-2c 使用真实 PostgreSQL 双角色 CI：app role 无 BYPASSRLS、admin/system role 受 allowlist；`RLS_MODE=enforce` 下验证缺 tenant context fail closed、连接池 tenant reset 和逐表 CRUD/direct-ID/link/export。SQLite placeholder 不得作为发布证据。
- PR-2c 中每个 command handler 先用 aggregate type/id 读取权威 tenant，与 command.tenant_id 比对，再进入 tenant-scoped reread；不匹配进入 dead letter 和 security audit。SLA/watcher 采用 system scan→tenant context handoff，所有 bypass 有 allowlist/reason/audit。
- 统一错误契约：无权访问不泄漏对象存在性；非法状态与权限/租户错误可区分并审计。

**验证命令**

```bash
cd itsm-backend
GOTOOLCHAIN=auto go test ./handlers/incident ./handlers/problem ./handlers/change ./handlers/service_request ./middleware ./tests/multi_tenant -count=1
cd ../itsm-frontend
PLAYWRIGHT_ENABLE_MULTI_TENANT=1 npx playwright test --project=multi-tenant --workers=1
npm run test:e2e:business
```

**退出标准**：四域状态矩阵 100% 有正/反例；真实 PG RLS 证据而非 SQLite sanity；跨租户 direct-ID 和关联写入均 fail closed；伪造/错误 tenant command 无业务副作用。

**回滚**：2a 保留旧读兼容和 feature flag；2b 只收紧越权写入不可回退为放行；2c 的 RLS policy/role 变更使用 expand、shadow、enforce 三阶段并附连接池恢复演练。

### PR-3：CMDB—变更—事件影响分析闭环（P1）

**分支**：`codex/commercial-03-impact-loop`

**目标**：用 CMDB 关系回答“这次变更影响什么、失败后关联哪些事件”，形成可演示的核心差异。

**任务**

- 复用 CMDB impact traversal，增加最大深度、最大节点、关系 allowlist、cycle detection 和 tenant predicate。
- 变更提交时生成版本化 impact snapshot，保存完整受控输入/hash、根 CI、关系图摘要、受影响服务/CI、risk policy version、process definition/binding version 和时间；process instance/task 以不可变外键引用。CI/关系变化导致 snapshot stale 时必须重新提交或显式豁免。
- 风险规则将关键 CI、影响范围、活跃事件、SLA/维护窗口纳入确定性评分；AI 只能追加建议和置信度。
- 变更实施失败/回滚时支持创建或关联事件，并保留 change↔incident↔CI 审计链。
- CMDB 数据不足时明确 degraded/unknown，不以“零影响”替代未知。
- 增加 topology、change detail、incident detail 三端可追溯入口和导出。

**退出标准**：跨租户图遍历不可见；循环图有界；同一审批快照可复现；未知数据不会被标为低风险。

**迁移/回滚**：snapshot 表先 expand，旧读路径兼容；shadow 生成比对后切读写；contract 前允许关闭新门禁，但已生成快照不删除。

### PR-4：连接器生命周期、密钥、健康检查和审计统一（P1）

**分支**：`codex/commercial-04-connector-governance`，依赖 PR-2c 的 command tenant invariant。

**目标**：所有官方和第三方连接器通过同一治理边界运行。

**任务**

- 收敛 installed → configured → enabled → healthy/degraded → disabled → uninstalled 状态机和后端 validator。
- 统一 secret reference/provider 接口；API/日志/audit/error 永不返回明文，只返回 masked metadata、版本和 lastRotatedAt。
- 健康检查通过受限 command/outbox 执行，具备 timeout、rate limit、重试边界、结构化错误和审计；不把“HTTP 可达”当作权限完整。
- manifest checksum/version/requiredPermissions/minVersion 与安装记录绑定；升级前兼容性检查，卸载前引用检查。
- callback/webhook 校验签名、重放窗口、大小限制、tenant mapping 和 idempotency key。
- 为 Feishu、DingTalk、WeCom、Webhook 运行相同 contract suite。

**退出标准**：无连接器可绕过 secret、tenant、permission、audit；健康失败可诊断且不泄密；重复 callback 不重复执行业务动作。

**迁移/回滚**：secret 与 installation schema 采用 expand/backfill/dual-read/write-switch；明文列只在轮换完成和恢复演练后 contract；生命周期切换保留一个版本兼容映射。

### PR-5：可重复发布、演示数据、升级迁移与健康诊断（P1）

**分支**：`codex/commercial-05-release-reproducibility`

**目标**：贡献者和客户可用一条命令获得可识别版本、可验证健康、可选择演示数据、可升级回滚的环境。

**任务**

- 提供统一 `make`/CLI 入口：preflight、build、up、initialize、verify、demo-seed、diagnose、backup、upgrade-plan。
- demo seed 独立、显式确认、命名空间化、可重复、可清理；生产镜像默认不执行。
- 输出 release evidence manifest，绑定 Git SHA、镜像 digest、迁移/模板版本、compose config hash、测试证据。
- 限定首期支持 N-1→N、声明 Postgres/架构/downtime/RTO/RPO 矩阵；升级执行 preflight → 可恢复性校验过的 backup → expand → app/template upgrade → verify → contract，并实际执行 restore smoke。
- `diagnose` 汇总但脱敏：容器、端口、health/readiness、init ledger、migration drift、queue backlog、connector/AI degraded capability。
- 将 first-value、角色、跨租户和 golden flow 纳入发布 gate。

**退出标准**：全新环境与上一个受支持版本升级均可重复；错误 env/secret/init/migration 会 fail closed；证据可归档比较。

### PR-6：AI 建议模型版本、置信度、反馈与评估仪表盘（P2）

**分支**：`codex/commercial-06-ai-evaluation`

**目标**：把 AI 从“调用成功”升级为可测量、可审计、可运营的决策支持能力。

**任务**

- 定义统一 Suggestion/Audit DTO：task、input fingerprint、provider/model、prompt/skill version、confidence/calibration、output、fallback、latency、cost、tenant/user、status。
- 支持 suggested → accepted/rejected/edited/expired 状态和 operator feedback；采纳动作再次经过权限与领域校验。
- 建立脱敏 evaluation dataset 与离线指标：分类准确率、top-k、RAG hit/attribution、人工采纳率、编辑距离、失败率、延迟和成本。
- 仪表盘按 tenant、skill、model、version、时间展示质量和漂移，不暴露跨租户 prompt/业务内容。
- 低置信度、provider timeout、禁用和预算耗尽均走确定性 fallback。

**退出标准**：每条建议可回答“谁、何时、哪个模型/模板、置信度、是否采纳、结果如何”；无法审计的建议不得触发动作。

**迁移/回滚**：Suggestion/Audit 表先 expand；旧 AI 接口 dual-write/shadow compare；动作切换可关闭，审计记录只追加不回删；跨租户数据按不可逆泄漏风险处理。

### PR-7：商业化 Core Golden Gate 与发布签字

**分支**：`codex/commercial-07-golden-gate`

**目标**：将六条工作流合并成每次发布可重复执行的门禁。

**任务**

- 建立 fresh install、first value、ITIL lifecycle、cross-tenant、CMDB impact、connector lifecycle、upgrade 七条 core golden journey；AI disabled/degraded 也必须安全且可诊断。
- 归档机器可判定 evidence manifest，绑定 revision/image digest、断言 schema、保留期和签名；至少证明无 R0 seed、BPMN 唯一 decision、audit correlation ID、outbox terminal state、跨租户拒绝且无 side effect、升级前后业务计数/checksum。
- 定义发布负责人、安全负责人、数据库负责人和产品负责人签字项。
- 区分 blocking、degraded、experimental capability，UI 由后端 capability contract 驱动。

**退出标准**：所有 P0/P1 阻断项关闭；不依赖 PR-6。

### PR-8：AI Beta Gate

**依赖**：PR-6，不阻塞 Core Golden Gate。

**任务**：增加 AI feedback golden journey、离线质量阈值、漂移/成本门禁；AI 权限、租户、审计和 deterministic fallback 属于安全基线，即使 beta 也不得豁免。

## 5. 全局反模式

- 只 Seed 数据但不验证流程绑定和首个用户结果。
- 因“表里已有一条”而跳过整类模板升级。
- 用前端隐藏菜单替代权限和租户校验。
- 新建第二套审批/工作流/连接器状态机。
- 在 Controller 启动不可恢复 goroutine 执行发现、健康检查或 AI 动作。
- 用 HTTP 2xx、Toast 或日志代表业务完成。
- 将 demo/customer 数据放入生产初始化。
- AI 输出直接改变高风险状态，或未知 CMDB 数据被当作零影响。
- 历史认证文档被当作当前 revision 的发布证据。

## 6. 计划变更协议

- 新发现的 P0 可插入当前步骤之前，但必须说明依赖、迁移、回滚和证据影响。
- 不得跳过 PR-0 的权威边界；发现双轨事实源时先冻结写路径，再迁移读路径。
- 每个 PR 只拥有列出的领域文件；跨 PR 公共契约变更先回写本蓝图。
- 步骤完成时记录 Git SHA、验证命令、未验证层和兼容窗口；不能以“代码已写”代替退出标准。
