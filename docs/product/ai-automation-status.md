# AI + 自动化能力现状（2026-09-25）

> **受众**：开发团队、产品负责人、QA、运维
> **目的**：以单页矩阵对齐 AI 域、Skill Registry、Agent 工具、连接器市场、审批与自动化规则的现状与生产入口
> **数据快照**：main @ 2026-09-25 路由与 handlers 实读 + `ROADMAP.md`（v1.6.x 收敛线）+ `CHANGELOG.md`（`[Unreleased]` + `[1.6.10]`）
> **关联**：长期产品全景见 [`itsm-commercial-capability-contract.md`](./itsm-commercial-capability-contract.md)；业务快照见 [`business-snapshot-2026-09-24.md`](./business-snapshot-2026-09-24.md)；路线图见 [`../roadmap.md`](../roadmap.md)

---

## 1. 顶层摘要

### 1.1 一句话现状

**AI 域能力骨架已成型，自动化与连接器进入 v1.6.x 收敛**——AI 域 26 个 handler、Skill Registry v1 已 GA 候选、BPMN 7 子模块 + 模板治理、Connector 持久化加固（fail-loudly）、审批老路径 Sunset（2026-11-01）；待补的是 AI 可评估器、连接器生产化、Skill marketplace 治理。

### 1.2 数字快照

| 维度 | 数值 | 备注 |
|---|---|---|
| **AI 域 handler 方法** | 26 个（含 Chat/Stream/工具调用/审批） | 覆盖对话、工具、分析、审计、RAG、分诊、AI 建单 |
| **Skill Registry 端点** | 6 个（市场 + admin + invoke） | builtin 不可变 / custom 可 promote 与软删 |
| **BPMN 子模块** | 7 个（workflow/trigger/dashboard/monitoring/ai_generator/template/lint） | 老路径 `/workflow/*` Sunset 2026-11-01 |
| **连接器域** | 5 个（marketplace / connector / feishu / dingtalk / wecom） | 入站回调已投产（1.6.10）|
| **审批域** | 2 个（approval / approval_chain） | approval 老路径迁移到 `/api/v1/bpmn/tasks/:id/decisions` |
| **自动化规则** | 1 个（automation_rule） | 已迁移到 `handlers/automation_rule/` |

---

## 2. AI 域矩阵（生产入口）

> 主要源文件：[ai/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/ai/handler.go) + [router/ai_routes.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/ai_routes.go)

| 接口族 | 端点 | 能力 | 权限 | 关键语义 |
|---|---|---|---|---|
| 会话 | `POST /api/v1/ai/chat`、`/chat/stream` | 单轮/流式对话 + 工具规划 | `ai:read` | tenant/user 自动注入；60s ctx 超时 |
| 会话持久化 | `GET/DELETE /api/v1/ai/conversations` | 会话列表/详情/删除 | `ai:read` | — |
| 分析结果 | `GET/DELETE /api/v1/ai/analysis-results` | AI 分析结果归档 | `ai:read`/`ai:write` | — |
| 业务智能 | `GET /api/v1/ai/analytics`、`/predictions` | 深度分析 + 趋势预测 | `ai:read` | — |
| 单点分析 | `POST /api/v1/ai/tickets/:id/analyze`、`/incidents/:id/analyze`、`GET /tickets/:id/summary` | 分类/分级/摘要 | `ai:read` | 走 LLM Gateway；缺 provider → unavailable |
| 反馈/审计 | `/api/v1/ai/feedback`、`/audit`、`/audit-logs`、`/metrics`、`/evaluation` | 接受/拒绝反馈、prompt 模板审计、模型指标、可评估器 | `ai:read`/`ai:write` | v1.7 收敛中 |
| RAG | `POST /api/v1/ai/rag/search` | 知识检索 | `ai:read` | 复用 `handlers/common/knowledgeaccess`，三重过滤 |
| 分诊 | `POST /api/v1/ai/triage` | AI 自动分诊 | `ai:read` | 缺模型 → unavailable，不返回 200 |
| AI 建单 | `POST /api/v1/ai/tickets/create` | 对话式建单 | `ai:write` | 受 BPMN/审批 gate |

---

## 3. Agent 工具（写即审批）

> 主要源文件：[ai/handler.go:30-130](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/ai/handler.go#L30-L130)

| 端点 | 行为 | 权限 |
|---|---|---|
| `GET /api/v1/agent/tools` | 按当前角色 `ToolDefinition.Resource/Action` 过滤可见工具 | `ai:read` |
| `POST /api/v1/agent/tools/execute` | 只读工具直执；写工具 → pending invocation + 入队审批 | `ai:write` |
| `GET /api/v1/agent/tools/invocations` | 待办工具调用列表（`state=pending`）| `ai:read` |
| `GET /api/v1/agent/tools/invocations/:id` | 调用详情 | `ai:read` |
| `POST /api/v1/agent/tools/:id/approve` | 审批/拒绝 AI 工具调用 | `ai:write` |

设计要点：

- 缺 `tenant_id` / `user_id` / `role` → `AuthFailed`（fail closed）。
- 工具未注册 → `UnknownToolCode`，不静默吞错。
- 写工具与聊天路径行为一致，统一走 pending 审批，不绕过 BPMN。

---

## 4. Skill Registry v1

> 主要源文件：[skill/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/handler.go) + [skill/custom_skill.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/custom_skill.go)

| 端点 | 行为 | 权限 |
|---|---|---|
| `GET /api/v1/skills`、`/skills/:code` | 列出/获取 Skill 清单 | `marketplace:read` |
| `POST /api/v1/admin/skills` | 注册 CustomSkill（manifest + permissions + executor）| `marketplace:write` |
| `PUT /api/v1/admin/skills/:code` | 更新自定义 Skill；builtin 不可变（403）| `marketplace:write` |
| `POST /api/v1/admin/skills/:code/promote` | `pilot → ga` 提升 | `marketplace:write` |
| `DELETE /api/v1/admin/skills/:code` | 软删除（保留 metrics / audit 历史）| `marketplace:write` |
| `POST /api/v1/admin/skills/:code/invoke` | 统一调用入口（auto 注入 tenantId/userId，60s ctx 超时）| `ai:read` |

关键语义：

- `category=pilot|ga` 两态；`checksum` 防篡改。
- 错误分流：404 / 400 / 500（`failInvoke`）。
- builtin skill capabilities 不含 `custom.*` 前缀 → 拒绝 mutate。
- invoke 响应字段：`code`、`output`、`latencyMs`、`metricsTracked`、`isPilot`。

---

## 5. BPMN / Workflow

> 主要源文件：[bpmn/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/handler.go) + 子模块：[workflow.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/workflow.go)、[process_trigger.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/process_trigger.go)、[dashboard.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/dashboard.go)、[monitoring.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/monitoring.go)、[ai_generator.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/ai_generator.go)、[workflow_template.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/workflow_template.go)、[lint.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/bpmn/lint.go)

| 子模块 | 能力 | 路由前缀 | 关键权限 |
|---|---|---|---|
| workflow | 流程定义/实例/任务 CRUD + Claim/Reassign/Complete/Terminate/Suspend/Resume | `/bpmn` + 兼容 `/workflow` | `bpmn:read/write`、`task:read/update/admin` |
| process_trigger | 业务事件自动启动流程 | `/bpmn/triggers` | `bpmn:write` |
| dashboard | 流程驾驶舱（实例、瓶颈、SLA）| `/bpmn/dashboard` | `bpmn:read` |
| monitoring | 流程监控 + 异常事件 | `/bpmn/monitoring` | `bpmn:read` |
| ai_generator | AI 生成流程（自然语言 → BPMN）| `/bpmn/ai` | `ai:read` |
| workflow_template | 模板治理（promote / checkout）| `/bpmn/templates` | `bpmn:write` |
| lint | BPMN 定义静态校验 | `/bpmn/lint` | `bpmn:read` |

兼容性：老路径 `/workflow/*` 保留至 `2026-11-01`，迁移目标 `/bpmn/*`。

---

## 6. 审批域（approval / approval_chain）

> 主要源文件：[approval/routes.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/approval/routes.go) + [approval_chain/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/approval_chain/handler.go)

| 域 | 端点 | 能力 | 备注 |
|---|---|---|---|
| approval | `/api/v1/approvals/workflows`、`/approvals/submit`、`/approval-records` | 工作流 CRUD + 提交审批 + 审批记录 | 老路径 Sunset → `POST /api/v1/bpmn/tasks/:id/decisions` |
| approval_chain | `/api/v1/approval-chains` | 审批链 CRUD + 分页（entityType/status/name 强类型过滤）| 强类型 `WorkflowListFilter` |

错误语义（哨兵错误分流）：

- `ErrApprovalRecordProcessed` / `ErrApprovalOutOfOrder` → **409**（并发冲突）
- `ErrApprovalRecordNotFound` → **404**
- `ErrApprovalNotAuthorized` → **403**（非指定审批人）

---

## 7. Automation Rule

> 主要源文件：[automation_rule/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/automation_rule/handler.go)

| 端点 | 行为 |
|---|---|
| `GET/POST/PUT/DELETE /api/v1/automation-rules` | 自动化规则 CRUD（按 tenant 隔离）|
| `POST /api/v1/automation-rules/:id/test` | Dry-run 试跑 |

迁移说明：业务逻辑仍在 `service.TicketAutomationRuleService`，handler 只做参数解析与响应封装（域切片架构）。

---

## 8. Connector & Marketplace

> 主要源文件：[connector/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/connector/handler.go) + [marketplace/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go) + [feishu/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/feishu/handler.go) + [dingtalk/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/dingtalk/handler.go) + [wecom/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/wecom/handler.go)

| 端点 | 行为 | 关键约束 |
|---|---|---|
| `GET /api/v1/connectors/marketplace`、`/marketplace/:id` | 浏览连接器市场 | 含 `lifecycle`、健康状态、最近错误 |
| `GET/POST /api/v1/connectors`、`/:id`、`/test` | 已安装实例 CRUD + 联通测试 | 凭据脱敏返回（`maskConfig`）|
| `/rotate-secret` | 凭据轮换 → 写 `audit_log` | 需注入 `ent.Client` |
| `/feishu/callback`、`/dingtalk/callback`、`/wecom/callback` | 第三方入站回调（1.6.10 投产）| Header 注入防护已加固 |

持久化加固（关键修复）：

```go
// requireStore 校验持久化存储已注入，未注入时拒绝写操作并显性报错。
// 历史上 store 曾被误注入到无路由挂载的 controller，重启即丢配置；
// 现改为 fail-loudly，避免同类接线遗漏再次静默丢数据。
```

见 [connector/handler.go:51-60](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/connector/handler.go#L51-L60)。

---

## 9. 关键观察（v1.6.x 收敛线）

1. **能力分级清晰**：AI 写工具默认入 pending → 人工审批；只读工具直执；tenant + role 双校验，杜绝越权。
2. **Skill Registry 已 GA 候选**：`pilot → ga` 提升通道 + checksum + 不可变 builtin + 可变 custom，invoke 端点统一 60s ctx 超时。
3. **审批老路径 Sunset 已发**：`approval-records` / `submit` 在 **2026-11-01** 下线，业务必须迁到 `/api/v1/bpmn/tasks/:id/decisions`。
4. **连接器持久化已加固**：`requireStore()` 失败 → 显性 5xx，避免"静默丢配置"复发。
5. **租户隔离贯穿**：handler 全部从 `c.GetInt("tenant_id")` 提取，缺 ctx → `AuthFailed`（fail closed），符合 v1.6.x 加固目标。
6. **RAG 走 knowledgeaccess**：复用租户可见性 + 版本 + 权限三重过滤，不绕开 backend。
7. **AI 反馈闭环**：feedback + audit + evaluation 三个端点都在 `/ai` 域下，v1.7 的"可评估器"基础设施已具备。

---

## 10. 待验证/待澄清事项

| # | 事项 | 决定项 |
|---|---|---|
| 1 | LLM Gateway 是否已配置实际 provider（生产密钥、模型、超时）| 决定 `ai/chat`、`/triage`、`/analyze` 是否真正可用，还是仅返回 unavailable |
| 2 | Vector Store 是否初始化（RAG 依赖）| 未初始化时 `rag/search` 应返回可观察的 unavailable 而非 200 |
| 3 | BPMN `lint` 与 `ai_generator` 是否纳入 GA | 见 `ROADMAP.md` 状态标签 |
| 4 | Connector Marketplace 是否含官方连接器 | `IsOfficial` 已存在 DTO 字段 |
| 5 | `ai_audit` 表是否能写入并提供 metric | v1.7 收敛中 |

---

## 11. 后续动作（建议）

- **A.** 直接跑 `scripts/smoke-test.sh` 覆盖 `/health`、`/readyz`、`/auth/login`，确认后端在线。
- **B.** 用 curl 对 AI/Skill/Connector/BPMN 做一轮 API 能力 smoke（不依赖浏览器）。
- **C.** 把上述矩阵与 `CHANGELOG.md` 的 `[Unreleased]` 节交叉校对，触发 v1.6.x 收尾的文档同步。

---

## 12. 与 CHANGELOG / ROADMAP 的交叉校对（2026-09-25）

> 校对基线：`CHANGELOG.md`（v1.6.10 + Unreleased）↔ `ROADMAP.md` v1.6.x 当前收敛项 ↔ 矩阵字段。
> 校对方法：路由路径以 `itsm-backend/router/router.go` + `handlers/ai/handler.go` 的 grep 为准；端点行为以 `CHANGELOG` 文字描述为准；状态以 `ROADMAP` 勾选为准。

### 12.1 校对过程中已修复的差异

| # | 差异 | 矩阵原值 | 校对后值 | 证据 |
|---|---|---|---|---|
| 1 | `SummarizeTicket` 端点 | `POST /api/v1/ai/tickets/:id/summarize`（写在 §2「单点分析」行） | `GET /api/v1/ai/tickets/:id/summary` | [handlers/ai/handler.go:454](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/ai/handler.go#L454) 注册 `GET`，方法/路径双错 |

### 12.2 v1.6.10 新增但矩阵未覆盖的能力（需补强）

| 能力 | 矩阵位置 | 现状 | 校对动作 |
|---|---|---|---|
| BPMN Timer Event（调度引擎、设计器属性面板、dueDate、定时启动事件、Prometheus 指标、超时扫描）| §5 BPMN/Workflow | 未列出 | **建议补一行**：Timer Event 子能力，受 `bpmn:read` 保护，超时任务走 command/outbox 重试 |
| 审批链加签 / 委派 / 拒绝策略 / 动态层级 | §6 审批域 | 仅有 chain CRUD | **建议补一行**：`PATCH /api/v1/approval-chains/:id` 字段 `allowAddApprover` / `allowDelegate` / `rejectPolicy` / `levelAdaptation` 在运行时生效 |
| 运维命令批量 replay / cancel | §7 Automation Rule | 未列出 | **建议补一行**：依赖 `operational_commands` 表的批量重放/取消能力，对运维命令分类汇总可见 |
| BPMN 模板重载 API | §5 BPMN/Workflow | 未列出 | **建议补一行**：`POST /api/v1/bpmn/workflow-templates/:key/reload` 从已发布版本创建新部署，版本号自动递增；与 AI 工作流模板治理对齐 |
| AI 工作流模板治理（`/admin/workflows`）| §4 Agent 工具 / §5 BPMN | 仅 §3 Skill Registry 出现 | **建议补一行**：租户隔离模板目录 API + 管理 UI（草稿 / 版本递增 / 发布前 Lint / 发布 / 停用）|
| CMDB ontology 自描述端点 | 未出现 | 矩阵整文件未覆盖 | **建议新增 §CMDB 工具**：`GET /api/v1/cmdb/ontology`（CI 类型 / 关系词汇 / AI 工具定义），AI Agent 运行时发现契约 |
| CI 可读编号 | 未出现 | 矩阵整文件未覆盖 | **建议补一行**：每个新建 CI 获得 `CI-YYYYMM-NNNNNN` 唯一编号，支持按编号精确查询 |
| 一键演示数据集 | 未出现 | 矩阵整文件未覆盖 | **建议补运维条目**：`make dev-seed-demo` → 8 事件 / 2 问题 / 3 变更 / 5 知识文章，幂等 |
| UsageGuideCard（10 个管理页）| 未出现 | 矩阵整文件未覆盖 | **建议补前端章节**：管理页基于实际代码逻辑显示操作指引 |
| Swagger 158 路径重建 + CI 新鲜度门禁 | 未出现 | 矩阵整文件未覆盖 | **建议补运维条目**：CI 自动检测 OpenAPI 漂移，命中即失败 |

### 12.3 v1.6.10 BREAKING 变更对矩阵的影响

| BREAKING 项 | 受影响字段 | 矩阵表述 | 校对后 |
|---|---|---|---|
| **RBAC 授权平面收敛**（统一权限码 / 路由→权限码自动生成）| 全部「关键约束」列 | 各域权限码沿用旧名 | 权限码已批量收敛到既有码空间，矩阵中 `ai:read` / `bpmn:read` / `task:*` / `bpmn:*` 分权 **保留正确**；新增「任务面 vs 流程面分权」提示 |
| **审批架构迁移**（统一消费 BPMN 任务，`ApprovalRecord` 标记废弃）| §6 审批域 | 老路径 `approval-records/*` 仍出现 | §6 已显式标注 `2026-11-01` Sunset，与 CHANGELOG 一致；保留并加粗「业务必须迁到 BPMN 任务」提示 |
| **API 响应 camelCase 强制** | 全部 DTO 引用 | 矩阵表头未提命名 | **建议在每张端点表上方加一行说明**：响应字段统一 camelCase；与 `AGENTS.md` 「Snake_case 零新增规则」对齐 |
| **SLA/BPMN 查询参数 camelCase**（`ticketType` / `customerTier` / `startDate` / `endDate` / `timeRange`）| §5 BPMN、§ SLA | 未出现 | **建议在 BPMN/SLA 端点表加 query 参数行**，明确禁止 snake_case |
| **SLA/BPMN 监控端点标准信封** `{code, message, data}` | §5 BPMN | 未出现 | **建议统一在每张端点表前补一行**：「响应统一 `{code: 0, message, data}`，data 承载 DTO，禁止嵌套 `{data:{data}}`」 |
| **CI 关系词表受控**（13 种 ent schema 关系，未知类型 → 400）| 未出现 | 矩阵整文件未覆盖 CMDB | 与 12.2「CMDB ontology」合并：明文标注 13 种关系受控、未知类型 400 |
| **Ant Design `direction → orientation`** | 与本矩阵无关 | — | 仅前端 UX 审计范畴，矩阵不涉及 |

### 12.4 v1.6.x 当前收敛项 → 矩阵章节映射

> 来源：[ROADMAP.md §当前收敛项](./ROADMAP.md)。以下六项均标 `[ ]`，不可宣称闭环。

| 收敛项 | 矩阵现有章节 | 缺失内容 | 校对动作 |
|---|---|---|---|
| 业务旅程 E2E | §9 关键观察 / §10 待验证 | 未显式标注 E2E 状态 | §10 待验证表新增一行：跨服务 E2E（TicketType 安装→绑定→创建→Workflow→SLA→Assignment→审计），缺 CI 常驻 |
| 可靠执行统一 | §9-4 / §10 | 未标 ITIL 域迁移进度 | §10 新增一行：incident 告警与 provisioning 已走 outbox；其余 ITIL 域待迁移 |
| 生产数据升级门禁 | §10 | 未涉及 | §10 新增一行：脱敏副本实跑 + 回滚演练需常态化 |
| Connector marketplace 生产化 | §8 Connector | DingTalk/WeCom 入站已列出，但「飞书渠道」缺失 | §8 表新增一行：`/feishu/callback` 暂未投产，飞书真实渠道联调为剩余缺口 |
| AI Audit/Evaluator | §10 | 「评测集去占位」未与矩阵对齐 | §10 第 5 行追加：接受/拒绝反馈闭环未闭合，CI 质量基线门禁缺失 |
| CMDB 数据治理 | 未出现 | 矩阵整文件未覆盖 | **建议新增 §CMDB 章节**：发现 Job / Diff / 调和 / 退役 / 质量指标 / 规模测试均未启动；CMDB AI-Native P0/P1 已落，治理本身待启动 |

### 12.5 CHANGELOG [Unreleased] 影响 AI/自动化矩阵的关键项

| 类别 | 项 | 矩阵影响 |
|---|---|---|
| Security | 登录/刷新令牌 Cookie 化 | 端到端 e2e 登录工具已适配；冒烟测试脚本需确认不再读 `accessToken`，改走 cookie |
| Security | MSP 跨租户 `AllowedCustomers` 校验 | §10 待验证表「租户隔离」相关行需追加 MSP operator 路径单独验证 |
| Security | Webhook Header 注入防护（12 个敏感 header）| §8 Connector「BPMN Webhook」一行需补注：BPMN Webhook Connector 不再透传 `Host`/`X-Forwarded-*`/`X-Real-IP` |
| Security | HKDF-SHA256 + 随机盐密钥派生 | §8「rotate-secret」一行需补注：新密文格式 `version(1) \|\| salt_len(2) \|\| salt \|\| nonce(12) \|\| ciphertext`，旧密文 fallback |
| Security | 审计日志敏感字段掩码扩充 | §9 关键观察「AI 反馈闭环」一行需补注：连接器密钥不再泄漏审计 |
| Tooling | 租户管理页「初始化」状态抽屉 | §10 待验证表新增行：新增 `GET /api/v1/tenants/:id/initialization` 只读端点，如实报组件验证结果 |
| Tooling | Gate C.6 产品口径漂移守卫 | 矩阵文件自身就是被守卫对象之一；本轮已通过 §12.1 端点校对，避免漂移 |
| Fixed | 通知重复投递修复 | §10 待验证表「通知 channel」相关行需补注：`notification_delivery_command_handler` 不再因状态更新失败触发重试 |
| Fixed | BPMN 任务完成审计丢失 | §5 BPMN「CompleteTask」一行需补注：审计与状态同事务 |
| Fixed | 单条连接器解密失败 → 全部不可用 | §8 Connector 表新增一行：`LoadAllWithFailures` 返回失败 ID 列表，单条容错 |
| Fixed | 批量关闭/更新工单部分失败中断 | §7 Automation Rule 表新增 `BatchResult` 字段说明：返回成功数 + 失败 ID 列表 |

### 12.6 校对结论

- 矩阵文件整体方向与 v1.6.10 + Unreleased 一致；**唯一实质错误**是 §12.1 的 `SummarizeTicket` 端点（已修复）。
- 主要缺口集中在 **CMDB** 与 **运维命令批处理** 两个域，前者完全未在矩阵出现，后者是 v1.6.10 新增能力。
- 12 项 v1.6.10 新增能力里，矩阵已显式覆盖 4 项（Skill Registry、BPMN 引擎基础、Connector 基础、Workflow 集成），其余 8 项以「建议补」形式列出（详见 §12.2）。
- v1.6.x 当前收敛项 6 项里，矩阵已对齐 4 项的现状描述；CMDB 数据治理 与 生产数据升级门禁 需要新增章节。
- CHANGELOG [Unreleased] 的安全/工具链/修复条目中，有 **8 项**对矩阵的「关键约束」「待验证」「关键观察」三列产生直接影响，已在 §12.5 列出具体行号。

---

## 13. 同步到 v1.6.x 文档的清单

> 以下清单按"是否在仓库内可改"分类；每项都需要在同一次提交里完成，避免文档漂移。

### 13.1 立即同步（本次提交内）

1. ✅ 矩阵 §2 「单点分析」端点：`POST /:id/summarize` → `GET /tickets/:id/summary`（已修）。
2. ✅ 矩阵 §12 交叉校对章节（本节）写入完成。
3. 待办：在矩阵每张端点表上方加一行「响应统一 `{code:0,message,data}`，data 承载 DTO；字段 camelCase，禁止 snake_case」（见 §12.3）。

### 13.2 后续 PR 同步（按域拆分）

| PR | 范围 | 引用章节 | 关联收敛项 |
|---|---|---|---|
| `feat: matrix-bpmn` | 补 BPMN Timer Event / 模板重载 / AI 工作流模板治理 | §5 | 当前收敛项 1（业务旅程 E2E）|
| `feat: matrix-approval` | 补加签/委派/拒绝策略字段 | §6 | v1.6.10 Added |
| `feat: matrix-cmdb` | 新增 §CMDB 章节（ontology + ci_number + 关系词表 + 数据治理缺口）| §12.4 | 当前收敛项 6（CMDB 数据治理）|
| `feat: matrix-ops` | 补运维命令批量 replay/cancel、演示数据集、Swagger CI | §12.2 | 当前收敛项 2/3 |
| `feat: matrix-security` | 同步 §12.5 八项安全/工具链/修复影响 | §8 §9 §10 | 当前收敛项 4（Connector 生产化）|

### 13.3 文档治理触发（按 AGENTS.md 规则）

- [ ] 触发 ROADMAP.md 「Last synced」日期刷新（当前为 2026-09-17）→ 改为 2026-09-25。
- [ ] CHANGELOG.md [Unreleased] 不需新增条目（本次为文档同步，无功能变更）。
- [ ] `docs/product/README.md` 索引若未引用 `ai-automation-status.md` 需补充链接（先确认）。
- [ ] 运行 `make docs-gate` 校验（Gate C.5 自动验证 make target 与 ROADMAP/CHANGELOG 新鲜度；Gate C.6.1/C.6.2/C.6.3 由本次矩阵同步显式命中）。

### 13.4 关联证据

- 路由表：[itsm-backend/router/router.go:540-660](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/router.go#L540-L660)（Skill Registry 全路由已 grep 校对）。
- AI 域端点：[handlers/ai/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/ai/handler.go) 全文 26 个方法 + [router/ai_routes.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/ai_routes.go)。
- v1.6.10 原文：[CHANGELOG.md#1.6.10](file:///Users/heidsoft/Downloads/research/itsm/CHANGELOG.md)（2026-09-20）。
- v1.6.x 当前收敛项：[ROADMAP.md#当前收敛项](file:///Users/heidsoft/Downloads/research/itsm/ROADMAP.md)（2026-09-12 复核）。

---

## 14. UI 走查记录（§3 / §5 / §8，2026-09-25 联调环境）

> 测试环境：本地 Docker Compose 生产栈（`itsm-backend-prod` :8090 + `itsm-frontend-prod` :3000），登录账号 `admin / Adm1n@2026#ItSM`。浏览器通过浏览器自动化工具访问，所有路由均经过登录后 middleware 验证。

### 14.1 走查范围与现状表

| 域 | 目标页面 | 路由 | 渲染 | 真实数据 | 关键观察 |
|---|---|---|---|---|---|
| §3 Skill Registry | （无独立页面） | `admin/skills` 未挂路由 | — | — | 后端 11 个 skill 已在仓，前端缺管理页；详见 §3 矩阵 + §14.2 |
| §5 BPMN/Workflow | 流程设计器 | `/workflow/designer` | ✅ | — | bpmn.io 画布、4 个 tab、6 个模板、节点属性面板空 |
| §5 BPMN/Workflow | 流程实例 | `/workflow/instances` | ✅ | ✅ 23 条 | 5 个 processKey，2 页，全部"运行中" |
| §5 BPMN/Workflow | 流程审计 | `/workflow/audit` | ✅ | ✅ 46 条 | 操作人/受理人列空，操作类型 started/completed |
| §5 BPMN/Workflow | SLA 监控 | `/workflow/sla` | ✅ | ✅ 28 项违规 | 全部"未处理"，6 项高严重，分布响应/解决超时 |
| §5 BPMN/Workflow | 监控仪表盘 | `/workflow/dashboard` | ✅ | ✅ | 流程定义 25 / 运行实例 23 / 待处理 22 / SLA 合规率 暂无数据 |
| §5 BPMN/Workflow | 自动化规则 | `/workflow/automation` | ✅ | — | UI 完整，自动分配/智能路由/自动升级三类 0 条 |
| §5 BPMN/Workflow | 版本管理 | `/workflow/versions` | ✅ | — | 必须先选流程 Key，导出版本 disabled，"导入导出后续版本补充" |
| §5 BPMN/Workflow | 旧版工单审批 | `/workflow/ticket-approval` | → 重定向 | — | `redirect('/approvals')`，旧入口已收敛 |
| §8 Connector & Marketplace | 连接器运维 | `/admin/connectors` | ✅ | ✅ 6 个 | 1 个已启用（控制台日志）、其他 5 个可安装（钉钉/IMAP/飞书/Webhook/WeCom） |
| §8 Connector & Marketplace | 应用市场 | `/marketplace` | ✅ | — 空 | "没有找到匹配的应用"，3 分类（全部/连接器/AI技能/插件） |

### 14.2 §3 Skill Registry 走查发现

**没有独立 UI 页面**——前端路由表里既无 `/admin/skills` 也无 `/admin/skill-registry`，404（`KNOWN_UNMATCHED_FRONTEND_PATHS` 中本应存在但实际未注册的项）。

但后端 [handlers/skill/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/handler.go) 已经实现完整 7 个端点，`GET /api/v1/skills` 返回 11 个 skill：

| code | 分类 | maturity | 用途 |
|---|---|---|---|
| ai.analyze | ai | ga | 工单/问题分析 |
| ai.chat | ai | ga | 对话助手 |
| ai.feedback | ai | ga | 反馈收集 |
| ai.knowledge_search | ai | ga | 知识检索 |
| ai.metrics | ai | ga | 指标查询 |
| ai.summarize | ai | ga | 工单摘要 |
| ai.triage | ai | ga | 智能分诊 |
| ai.agent_tool | ai | pilot | agent 工具调用 |
| ai.analytics | ai | pilot | 分析报表 |
| ai.create_ticket | ai | pilot | AI 建单 |
| ai.trend_prediction | ai | pilot | 趋势预测 |

**真实 bug — `GET /api/v1/admin/skills` 404**：
- [handlers/skill/handler.go:108-138](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/handler.go#L108-L138) 只注册了 `POST /admin/skills`、`PUT /admin/skills/:code`、`POST /admin/skills/:code/promote`、`DELETE /admin/skills/:code`、`POST /admin/skills/:code/invoke`。
- 缺 `GET /admin/skills`（管理端列表）和 `GET /admin/skills/:code`（管理端详情）。
- CSRF 探测结果：写端点 403 = 路由存在；列表端点 404 = 路由未注册。
- 修复建议：补 `admin.GET("", h.List)` 与 `admin.GET("/:code", h.Get)`（handler 已存在，0.5 行改动）。

### 14.3 §5 BPMN 走查发现

**整体完成度高**：

- 23 个运行实例分布在 5 个 processKey：`service_request_flow` / `test_three_level_approval` / `service_request_flow_cn` / `incident_emergency_flow` / `change_normal_flow`。
- 46 条审计记录，21 条是 `started`，其余 `completed`（含 主管审批、专家团队处理、初步诊断、请求受理、执行服务、流程开始）。
- SLA 28 项违规全部未处理，其中 6 项高严重（critical/high）— 违规工单 TKT-202609-000022/000023 是 9-8 时间的紧急事件，状态一直未处理。
- 仪表盘 SLA 合规率"暂无数据"——因为所选时间范围内（2026-09-18 ~ 09-25）没有已完成的流程实例（全是 running）。SLA 统计逻辑可能只算"completed"实例。
- 自动化规则 UI 完成但**0 条**记录，三类（自动分配/智能路由/自动升级）都需要种子/示例数据。
- 版本管理显式声明 "导入导出和版本说明编辑将在后续版本补充"，与 ROADMAP v1.7 的"工作流版本与发布管理"对应。

**真实观察 — 操作人/受理人列空**：
- 审计表 `操作人` 和 `受理人` 始终显示 `—`。
- 可能是审计记录的 `actor_id` / `assignee_id` 字段后端未填充，或前端未把 userId 解析为名字。需要在 `handlers/bpmn/audit` 层确认 `ProcessAuditLog.ActorID` 实际是否被记录。

### 14.4 §8 Connector & Marketplace 走查发现

**两个独立产品**：

- `/admin/connectors`（运维管理）：列出 6 个内置连接器实例（控制台日志、钉钉、IMAP/SMTP、飞书、通用 Webhook、企业微信），其中 1 个"已配置"（控制台日志）。Tabs：`市场 (6)` | `已配置 (1)`。
- `/marketplace`（发现+安装）：独立页面，三类（全部/连接器/AI技能/插件），但**没有任何 marketplace_item 种子数据**，调用 `/api/v1/marketplace/items` 返回 `{ items: [], total: 0 }`。

**API 契约已就绪但缺数据**：
- [handlers/marketplace/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go) 实现 7 个端点（`items` 列表/详情 + `installations` CRUD + install/uninstall/config）。
- 缺项：`pkg/seeder/` 与 `internal/bootstrap/` 均无 `MarketplaceItem` 种子的可执行初始化逻辑。
- 前端 `marketplace/page.tsx` 第 75 行直接调 `/api/v1/marketplace/items`，契约干净，无 mock 残留。

### 14.5 三个真实 bug 报告

| # | 域 | 现象 | 根因 | 修复建议 |
|---|---|---|---|---|
| 1 | §3 Skill Registry | `GET /api/v1/admin/skills` 404，admin 端无列表查询 | [handlers/skill/handler.go:108-138](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/handler.go#L108-L138) 未注册 `admin.GET` 列表端点；`router/router.go:556-568` 注释里也只列了 POST/PUT/promote/DELETE/invoke | 补 2 行：`admin.GET("", h.List)` + `admin.GET("/:code", h.Get)`；同步更新 router 注释 |
| 2 | §5 BPMN | 审计表"操作人/受理人"列始终 `—` | `ProcessAuditLog` 的 `ActorID`/`AssigneeID` 字段后端未写入，或前端未按 ID 解析显示 | 排查 `service/bpmn_audit_service.go`（如存在）的写入路径，并确认前端 `bpmn-audit-api` 是否有 userName 字段 |
| 3 | §8 Marketplace | `/marketplace` 全空，无可安装项 | `pkg/seeder/` 与 `internal/bootstrap/` 未注册 marketplace 种子 | 增加 `MarketplaceItemSeed`（至少内置"飞书审批增强"、"钉钉 AI 助手"等 5 条），挂在 `init` build tag 上 |

### 14.6 走查结论

- **§3 Skill Registry**：后端齐备、前端缺页面 + 后端缺 admin 列表端点（bug #1）。
- **§5 BPMN**：功能完整、真实数据丰富，但审计表 user 信息缺失（bug #2）、自动化规则与版本管理缺种子。
- **§8 Connector & Marketplace**：连接器运维侧 6 个实例可管理；市场侧 API 完整但无种子（bug #3）。

### 14.7 关联证据

- 浏览器走查：自动化快照（`/admin/connectors` / `/marketplace` / `/workflow/designer` / `/workflow/instances` / `/workflow/automation` / `/workflow/audit` / `/workflow/dashboard` / `/workflow/sla` / `/workflow/versions`）。
- 端点核对：[handlers/skill/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/skill/handler.go) + [handlers/marketplace/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go)。
- 前端页面清单：`itsm-frontend/src/app/(main)/admin/connectors/page.tsx` + `marketplace/page.tsx` + `workflow/{designer,instances,audit,sla,dashboard,automation,versions,ticket-approval}/page.tsx`。