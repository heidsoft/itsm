# 🛣️ ITSM Roadmap

> **Source of truth for what is shipping, what is shipping next, and
> what is parked.** Updated as part of every release. Last synced: 2026-10-29（v1.6.16–v1.6.20 五版本条目级回填完成——此前该区间只有 CHANGELOG 无 ROADMAP 条目；v1.6.21–v1.6.34 的增量以 `CHANGELOG.md` 为准，待下个发布节点统一回填。同日：v1.7 两项提前收口确认——AI Evaluator CI gate（v1.6.16 落地，四指标阈值已进 backend-ci）与 Skill Registry UI（v1.6.16 落地）。P0 数据库结构护栏迁移 `20261027_prod_schema_guards.sql` 落地 29 索引/唯一键）。
>
> Cross-references:
> - Design outlines: [docs/prd/](./docs/prd)（当前仅 1 篇）+ [plans/](./plans)（专项计划与审计）
> - v1.0 GA readiness: 历史验收文档已于 2026-10-12 移除（`Status: historical`，结论不得复用于当前版本）；当前发布判断见 [文档状态与事实源](./docs/documentation-governance.md)
> - Architecture: [docs/architecture/](./docs/architecture)
> - Open issues & milestones: GitHub [Issues](https://github.com/heidsoft/itsm/issues) and [Projects](https://github.com/heidsoft/itsm/projects)

### 当前未发布改进（2026-10-29）

- 应用市场：技能、插件缺少加载器时拒绝安装和配置，保留历史记录的查看与卸载；路由回归覆盖租户、权限及连接器原有流程，服务回归覆盖各安装状态下不写入数据。
- AI 评测：摘要评分改为调用候选生成器，空摘要、无关摘要和混合质量输出均有回归测试；仍是离线评测基础，不作为生产模型验收。
- 知识库：堵住更新文章与恢复历史版本两条绕过评审的发布旁路，恢复版本重置为草稿并同事务清理旧索引。
- CMDB：对账应用差异前校验资源快照新鲜度，指纹已变化时拒绝覆写（409），避免用过期快照盖掉更新的云上事实。
- 报表：`slaCompliance` 改按权威 SLA 违规记录计算，不再用解决率冒充合规率；工作台读取 SLA 失败上抛而非静默填零。
- 流程引擎：部署带定时启动的流程时，定时器注册失败不再用 `fmt.Printf` 静默吞掉，改为结构化告警（含流程 key/租户/错误），与设计器发布口径一致；部署本身不阻断，重新部署可重建时间表。
- 身份与连接器：SSO（OIDC）发起按语义分类失败（503/404/400）；飞书 OAuth 回调补一次性 state 防 CSRF；企微入站回调真正解密（URL 验证返回明文 echo、事件解出结构化字段），入站日志不再打印消息正文；钉钉入站验签缺头一律拒绝（无签名探活不再放行）并改常量时间比较。
- 用户管理：删除与停用语义分离——删除写 `deleted_at` 软删并统一屏蔽读路径（列表/搜索/关联查询/登录），不可再被"启用"复活；停用保持原语义（可见可启用）。存量 `active=false` 数据按停用对待，不回填。
- CLI 工程：`bin/itsm.js` 构建陈旧守卫——src 比 dist 新时以 exit 70 拒绝执行并给出修复指令，消除「改 src 忘 build 静默跑旧编译产物」对 agent 判定的误导；dist 继续跟踪（fresh clone 直接可跑的政策不变）。
- 上述连接器与身份能力仍为预览：入站解密/验签/去重已可用，但「消息→工单」业务派发、真实交付机制和生产候选评测尚未完成。

---

## 🎯 North Star

**Become the de-facto open-source AI-Native ITSM for enterprises that need
ServiceNow-class workflows without the lock-in or the footprint.**

Concretely that means:
1. **Process completeness** across the ITIL core, measured by executable business journeys rather than menu count.
2. **AI that earns its seat** — classification, summarization, RAG, and
   impact analysis that are measurable, not vibes.
3. **Native integration surface** — Feishu / DingTalk / WeCom / Webhook
   ship as first-class connectors, not bolt-ons.
4. **Operational discipline** — coverage, observability, security, and
   release hygiene as defaults, not afterthoughts.

---

## 📅 Release Timeline

| Version | Target | Theme | Status |
|:---|:---|:---|:---|
| **v1.0 GA** | 2026-Q2 | ITIL core + AI-Native scaffolding + private deploy | ✅ Shipped |
| **v1.6.x**   | 2026-Q3 | TicketType platform + reliability + RBAC/tenant hardening | 🟡 In progress |
| **v1.7**     | 2026-Q4 | Connector productionization + AI evaluator + business E2E | 🟢 Planned |
| **v2.0**     | 2027-Q2 | Coverage 70% + AI auto-triage GA + MSP billing + multi-region | 🔵 Roadmap |
| **v3.0**     | 2027-Q4 | Self-hostable AI inference + Plugin marketplace v2 + agent ecosystem | ⚪ Parked |

---

## 🟢 v1.0 GA — Shipped (2026-Q2)

**Theme:** Get the foundation right.

### Capability

- [x] **ITIL core flows** — ticket / incident / problem / change / release / service request
- [x] **Service catalog** — request templates, approval routing, SLA binding
- [x] **BPMN workflow engine** — process definitions, instances, user tasks,
      variable persistence, candidateGroups-driven approval (replaces the
      old dual-track approval system)
- [x] **CMDB v1** — CI types, configurations items, relationships, impact
      analysis, cloud discovery scaffold
- [x] **Knowledge base** — articles, versioning, RAG retrieval
- [x] **SLA** — multi-level policies, escalation matrix, alert rules
- [x] **AI capabilities (scaffold)** — Guidance-Harness-Skill framework,
      LLM Gateway, Triage / Summarize / KB skills
- [x] **RBAC + multi-tenant** — roles, permissions, menu gating, MSP mode
- [x] **Deployment** — Docker Compose (private / saas / saas_msp), GHCR
      images, multi-platform Release zip

### Quality

- [x] GA gate (4 checks): backend tests, frontend build, compose health,
      E2E smoke (11 core APIs)
- [x] Staticcheck + gofumpt + ESLint + tsc
- [x] Dependabot weekly scans
- [x] Security policy + Code of Conduct

### 后续持续治理项

- 🟡 关键业务旅程的服务层、集成与 E2E 覆盖继续提升。
- ✅ 超大 Controller 按现有领域边界渐进拆分——**已完成**：`controller/` 已清空（仅余空目录），新代码统一进入 `handlers/<domain>`；拆分过程中同步完成 `router.go` 按域拆分与六域接口化样板。
- 🟡 连接器从生命周期框架推进到真实渠道生产验收——钉钉 / 企微入站已落地，待真实渠道联调与飞书补齐。

---

## 🟡 v1.6.x — In Progress (2026-Q3)

**Theme:** Cover the seams and harden the foundation.

### 已落地

- [x] **TicketType 平台化** — 类型持久化、动态字段、创建快照、Preset Library、归档恢复和管理 UI。
- [x] **统一绑定解析** — Ticket 创建从已解析 TicketType 执行 Workflow、SLA 与 Assignment。
- [x] **权限与审计** — TicketType 独立管理/归档/Preset 安装权限，ACL manifest 覆盖；Preset 安装、归档恢复和绑定变更独立审计。
- [x] **可靠异步执行** — 工单与事件的流程启动进入持久化 command/outbox；关键通知具备 outbox、租约、重试和死信基础。
- [x] **租户与输入防线** — 覆盖跨租户、禁用类型、非法动态字段与非法绑定引用的回归测试。
- [x] **发布与安全加固** — HttpOnly cookie、初始化 migration ledger、PostgreSQL RLS、Endpoint ACL、依赖与运行时安全基线。
- [x] **分层迁移收尾** — legacy controller 全部退役，新代码统一进入 `handlers/<domain>` 垂直分层；swagger 路由冲突、菜单/认证契约断裂等收敛问题清零。
- [x] **状态机与错误语义加固** — 变更状态推进 CAS 并发防护；问题/事件状态机违规返回 409 业务语义而非 500；`super_admin` 通配权限链路（登录 / `/auth/me` / 前端判定）对齐。
- [x] **可靠执行补强** — commandbus 对聚合已删除的命令立即死信；审批链收口 SQL 缺列修复；业务流程回归套件（63 项集成 + 27 项生命周期深度）全绿。
- [x] **开源体验补强** — `make dev-seed-demo` 一键演示数据集（事件/问题/变更/知识库，幂等），README 快速开始接入；产品定位明示 **v1.6.x 界面中文优先**，完整界面 i18n 规划至 v1.7。
- [x] **行级权限守卫（写路径对称）** — incident 生命周期（ack/resolve/close/reopen/escalate）、ticket 四操作、change 非审批路径、release 写路径统一补 owner/role 守卫；四域 Update/Delete 接入 DataScope 校验；RBAC ResourceActionMap 补齐 releases 显式映射（`controller/` 已清空，仅余空目录）。
- [x] **错误语义与可观测性统一** — 行级拒绝 403 被 handler 兜底吞成 500 的根因修复，四域错误映射统一走 `common.RespondError`；AI 持久化失败经 zap 与计数器暴露，静默失败不再不可观测。
- [x] **租户隔离与上下文防线** — 租户/用户上下文缺失统一为 401（`TenantIDOrUnauthorized`）；change / cloud / BPMN 等域 30+ 处无保护类型断言与 `tenantID=0` fail-open 跨租户风险清零；`tenant_id` 豁免单一源 + 启动扫表 guard 接入长驻进程与 cmd/cmdb 路径。
- [x] **迁移与升级安全** — `migrations/` 目录即真相（discovery + 重写）、migration ledger 调和、PII 脱敏注解与 `migration-lint` CLI、升级前 preflight 与发布证据；索引缺口 batch2 补齐及 adoption 日期边界缺陷修复。
- [x] **连接器入站能力** — 钉钉 / 企微入站回调、持久化入站去重（`connector_inbound_dedups`）、连接器健康度与凭据轮换。
- [x] **工作流引擎加固** — 出边 fallback 声明、ServiceTask metaData 寻址、handler 双键注册、内置模板 16/16 lint 零错误、`workflowDefinitionKey` 全链路透传与 lint 门禁加固；工作流模板管理与 BPMN 前端集成。
- [x] **CMDB AI-Native P0/P1** — 关系词表（13 种关系单一源）、本体端点、`ci_number` 全局唯一序列；List/Search 合并、AI 工具与影响解释；CMDB 前端完成 React Query 迁移。
- [x] **CMDB 数据治理骨架（v1.6.15）** — 退役状态机（`online -> retiring -> retired` 显式转移 + 受控 reason 词表）、差异分类（`add/noop/retire_confirm/duplicate`）、治理质量指标（Active/Retired/Stale/Orphan/Incomplete/CompletenessPct）；21 用例全绿，零 Repository 接口扩张。
- [x] **架构收敛与可维护性** — `router.go` 巨石按域拆分为独立 routes 文件；user / tenant / rbac / notification / application / cloud 六域接口化并补冒烟测试（作为 58 域迁移样板）；包名与目录名统一、分层守卫增加包名≠目录名检查；双 BPMN 引擎死代码清理。
- [x] **Pilot 能力生产化（2026-10-21）** — 通知域晋升 GA：投递审计读面（`GET /notification-deliveries` + 失败重试）与渠道诊断（`GET /notifications/channels`：渠道装配/健康/收件目标三合一）补齐，"没收到通知"三类根因可定位；新增 `connector.Manager.HealthCheckTenant` 只探测当前租户，避免租户级只读接口成为跨租户连通性探针。问题域修两处生产阻断（关闭必须有解决方案+根因，否则 409；「转为已知错误」的 `problemId` 此前被 json 静默丢弃，现已落 `problem` 边并回显），但因 `GET /problems/:id/sla` 仍显式 503 而维持 Pilot。逐项阻断项见 `output/pilot-to-ga-2026-10-21.md`。
- [x] **授权平面收敛（批次 1–5）** — A 类越权写收口（bpmn/流程触发等 39+ 条写路由补挂权限门，任务面 `task:*` 与流程面 `bpmn:*` 分权）；B 类权限码词表统一（95 种未定义码收敛到既有码空间）；C 类预检映射全量对齐（112 处声明/预检错配清零）；批次 5 治本：**路由声明成为权限单一真源**——预检映射路由条目由 `cmd/authz-gen` 从声明 AST 生成（698 条），族级回退策略显式化（120 条），4 道守卫（写路由必挂门 / 声明码⊆码空间 / 声明-预检对齐 / 生成物新鲜度）构成防漂移闭环；admin/technician DBOnly 空集修复 + seeder 同义动作奇偶补齐。
- [x] **性能与运维** — 变更列表与 RAG 向量检索两处热路径 N+1 修复；prod 备份自动化（`scripts/prod-backup.sh` + 恢复演练 + launchd 每日调度）、compose 项目名隔离（`itsm-prod` / `itsm`）与诊断端口参数化。
- [x] **变更审批-状态原子性（P0）** — `handlers/change.SubmitApproval` 把审批记录写入与目标状态 CAS 推进合并到同一 `*sql.Tx`：`SubmitApprovalRecordTx` 仓储方法以 `record.ChangeID + tenantID + status='draft' expected` 一次提交，任一子步失败整笔回滚；同时拒绝审批人不在审批链上的提交，避免越权。回归 `handlers/change/submit_approval_atomicity_test.go` 5 例：原子成功提交 / 事务失败零副作用 / CAS 守卫不会回退状态 / 拒绝非链上审批人 / 时间戳由调用方注入不被时钟漂移。
- [x] **工单关联写入面（P0）** — 补齐 `PATCH /api/v1/tickets/:id/relations` 路由并接入 `service.UpdateTicketAssociations`：父子环、跨租户、缺失父工单三类拒绝都返回 `*common.BusinessError`（422/4004/4004）而非 500/5001；缺省 `tenantID<=0` fail-closed，读端点透出的字段不含底层 ent 错误串。回归 `router/ticket_relations_write_route_test.go` 7 例打在真实 `SetupRoutes` 上：同租户写入反射 / 跨租户 404 且零副作用 / 父子环 422 保留原状态 / 不存在 404 / 非法 body 与非法 ID 均为 400/1001 / childrenTree 反映父子关系；既有 `TestTicketRelationsRoutesTenantScope` 仍绿。同批 `dto.ChangeListResponse` 复原以修复 `service/change_service.go` 残留死引用（`go build ./...` 与 `go vet` 退出码 0）。
- [x] **P0-2 R2-a 权限奇偶守卫红态前置** — `TestBuiltinRolePermissionCodes_Guard` 范围从手工枚举 `{admin, technician}` 扩到 `domainrole.All` ∩ `middleware.RolePermissions` 同名键（结构性扩围在 `a852b45dc`）；按 `plans/product-remediation-plan-2026-10-03.md` §2 R2-a 约束分两档：硬档 `admin`/`technician` 差异即红，待拍板档 `manager`/`agent`/`sysadmin`/`end_user` 以 `t.Skipf` 子测试登记差异清单（实测 21/8/`*:*`/16 缺），等 §3 N4 拍板后让守卫真正红出。台账 E4-41 伴生残留（`middleware.RolePermissions` 留不留 / `security_admin`/`it_admin` 同名缺口）交 N4、N5 拍板。
- [x] **P0-2 N4/N5/N6 拍板落地：权限码集扩权 + 双权威硬编码兜底表移除 + demo 页清理（2026-10-05）** — N4 选**扩权**：`authz.BuiltinRolePermissionCodes()` 按 R2-a 实测差集补齐 `manager` 26 / `agent` 7 / `end_user` 16 码（agent 的 `alerts:read` 按拼写收敛口径不授，路由预检实测零条 `alerts:*` 声明）；N5 选**移除**：`middleware.RolePermissions` 380 行硬编码表删除，DBOnly `unconfigured` 兜底、`HardcodeOnly`/`Merge`/`Fallback`、菜单与 `/auth/me` 展示面统一改由 `authz.RolePermissionDefaults()` 派生（内置角色直接派生自播种码集，兜底与播种结构性同源）；legacy `security` 与 `msp_*` 五个运行时角色默认集在 authz 显式保留；`sysadmin` 兜底从 `*:*` 收敛为枚举口径（与 configured 一致）。N6：删除 `/agent-ops-demo` 纯静态演示页。守卫 `TestBuiltinRolePermissionCodes_Guard` 的 R2-a `t.Skipf` 待拍板档同批退役，替换为 N4 扩权回归锁 + 单一真源锁（兜底默认 ⊄ DB 码集即红）。行为契约见 `UPGRADE.md` §1.31。实测证据：`go test ./internal/authz/ ./handlers/auth/ ./handlers/common/ ./middleware/ ./tests/contract/ ./service/` 全绿、`npm run type-check` 绿；`pkg/seeder`/`tests/parity` 因他域在途 `seeder.go` 编辑暂缓，落位后补跑。
- [x] **P0-2 E4-9b 后续切片：AI 审计日志端点页长/信封/双写响应三缺陷同片收口（E4-45）** — `GET /api/v1/ai/audit-logs` 修复 ① handler 私有 `queryInt` 无上界 ② `service/ai_evaluator.go` 私有 `auditMaxPageSize=200` 第三套 owner ③ 四键手拼信封无 `totalPages` 且回显未采纳值 ④ `handlerctx.ResolveTenantID` 拒绝后 handler 再 `common.Fail` 写第二份响应（拼接 JSON）。修法：HTTP 入口走 `common.GetPaginationFromQuery` 单点、service 侧仅 `common.ValidatePagination` 兜非 HTTP 调用方、信封走 `common.SuccessWithPagination`、`common/pagination.go:69` 裸 `100` 换 `MaxPageSize`、handler 拒绝分支只 `return`、前端类型复用 `PaginationResponse<AIAuditEntry>` 并删三处 `??/||` 兜底。回归 `tests/contract/ai_audit_logs_envelope_test.go` 8 例（五键精确集合 / 缺省 20 / 越界不越平台界 / 回显等于采纳值 4 输入 / 第二页余数 / 租户收敛 / 缺租户只写一份 / 空结果 `[]`），负证明在 HEAD 独立工作区跑同一夹具 ⇒ 7 例转红、唯一常绿例锁既有租户收敛；`go test ./handlers/ai/... ./service/... ./tests/contract/... ./common/...` 与前端 `npm run type-check` 全绿。同批台账新登记 E4-46（同一双写形态 24 处/6 文件）与 E4-47（`page_size_owner_ratchet_test.go` 只扫 `DefaultQuery` 漏掉的多处自建 owner，含 bpmn dashboard 真实整表读取通道），均交下一批裁。`UPGRADE.md` §1.25 写 `pageSize` 在 `(100,200]` 的行为变化与新增 `totalPages` 字段。
- [x] **P0-2 BPMN 审计与流程实例列表的页长收给单一所有者（E4-47⑤）** — 三个端点（`/bpmn/dashboard/audit-logs`、`/bpmn/monitoring/audit-logs`、`/bpmn/monitoring/instances/status`）原各自 `DefaultQuery`/`strconv.Atoi` 下传且无上界，最终落到 `service/bpmn_audit_service.go` 的条件式分页分支 `if req.Page > 0 && req.PageSize > 0 { Offset/Limit }`：条件不成立时 `LIMIT` 子句消失 ⇒ 单租户 25 条种子被整表读出。修法：HTTP 入口改走 `common.GetPaginationFromQuery` 单点、`ListProcessInstancesStatus` 自建缺省 1/20 删除、service 侧只留 `common.ValidatePagination` 兜非 HTTP 调用方。回归 `handlers/bpmn/audit_pagination_contract_test.go` 6 例（缺省 20 / 越界不越平台界 / 非数字回落 / 负值回落 / 末页余数 / 跨租户收敛），负证明在 HEAD 独立工作区跑同一夹具 ⇒ 6 例转红 3 例（pageSize 整表读取 + 原值执行）。`UPGRADE.md` §1.26 写 `pageSize>100` 由「按原值执行 + 回显 100」变「回落 20」与 `pageSize=abc/-5` 由「整表读取」变「回落 20」，`docs/api-reference.md` BPMN 监控章节同批补页长口径。同批台账新登记 E4-48（`/bpmn/dashboard/audit-logs` 与 `/bpmn/monitoring/audit-logs` 是同一用例两套 HTTP 表面，两条都无 `RequirePermission`、dashboard 那条直接序列化 Ent 模型响应键为 snake_case 且把租户 ID 发给前端，唯一生产消费方 `/workflow/audit` 页面因此四列读不到值），交下批。
- [x] **P0-2 同源响应双写收口（E4-46）** — `handlerctx.ResolveTenantID` / `RequireTenantID` 失败时委托 `middleware.AbortIfTenantError`（`middleware/msp_tenant_resolver.go`），后者对 401/403/500 三种情形都**已写体并 abort**；但 `handlers/` 仍有 24 处「拒绝后 handler 又 `common.Fail` 再写一份」的形态，客户端按 `{code,message,data}` 严格解析时报 `invalid character '{' after top-level value`。本批一次成片收掉 **24 处 / 6 文件**：仅保留 `if !ok { return }`，删除 `common.Fail` / `common.FailWithErr` / `common.ParamError` 等任何再写响应；并在 `common/handlerctx/handlerctx.go` 的 `ResolveTenantID` / `RequireTenantID` 函数注释里写明「失败时已写响应并 abort；调用方在 `!ok` 分支禁止再写响应，只能 `return`」。新增 `tests/contract/handlerctx_no_double_write_test.go` 作为双向 AST 棘轮（扫描 `handlers/*.go` 里 `tenantID, ok := handlerctx.Resolve*` / `userID, ok := handlerctx.Resolve*` / `tenantID, ok := handlerctx.Require*` 形态的 `!ok` 分支体，并对 `handlerctx.Resolve*` / `Require*` 的契约注释做 token 校验）。`UPGRADE.md` §1.27 记变更范围表 + 集成检查 + 未触碰区域（`handlers/sla_template/handler.go:32-34` 1 处未收口，`handlers/dashboard_handler.go:528-530` 的 `!ok` 分支体不是裸 `return` 待读全函数后再裁；跨域统一文案「未授权访问 vs 租户上下文缺失」是同型问题但本批不动）。坐标：`handlers/ticket_type/handler.go` 9、`handlers/ai/handler.go` 4、`handlers/operations/handler.go` 4、`handlers/ticket_rating/handler.go` 3、`handlers/timer/handler.go` 3、`handlers/skill/handler.go` 1。

- [x] **P0-2 流程实例寻址键收敛（ga-gate 全链路 E2E 404 根因）** — `/api/v1/bpmn/process-instances/:id` 家族此前双寻址键：仅 GET 详情按数字 Ent 主键寻址，suspend/resume/terminate/variables/approval-history 与 monitoring 表面全按 `PI-*` 业务键寻址，列表返回的实例键回打详情必然 404。现全家族收敛为 PI 业务键单一所有者（数字 ID 不再是任何端点寻址入口，fail-closed 404），`POST` 启动响应从裸序列化 Ent 模型改为 `BPMNProcessInstanceResponse` DTO（snake_case → camelCase 契约修正）。回归 `service/bpmn_process_instance_get_test.go`（PI 命中/跨租户拒绝/数字 ID 必须 404）+ `bpmn_process_instance_history_test.go`；前端 `WorkflowApi.getInstances/getInstance/startWorkflow` 删除多字段兜底、审批中心深链改传 `processInstanceKey`。`UPGRADE.md` §1.28、`docs/api-reference.md` 工作流章节同批。

- [x] **P0-2 BPMN 流程审计单表面收敛（E4-48）** — 「流程审计日志读取」此前有两套已注册 HTTP 表面（`/bpmn/dashboard/audit-logs` 与 `/bpmn/monitoring/audit-logs` 委托同一 `BPMNAuditService.QueryAuditLogs`）、两条都无 `RequirePermission`（任何已认证用户可读全量流程审计）、dashboard 那条直接序列化 Ent 模型（snake_case 响应键且把 `tenant_id` 发给前端），唯一生产消费方 `/workflow/audit` 页面因此四列恒空、只能靠 `normalizeAuditLog` 同键自复制兜底并自报 `tenantId`。修法：保留 dashboard 为唯一表面，删除 monitoring 路由 + `MonitoringHandler.GetAuditLogs` + `BPMNMonitoringService.GetAuditLogs/AuditLogRequest`；两条 dashboard 审计路由挂 `RequirePermission("bpmn","read")` 并再生成 `rbac_precheck_gen`（+2 条声明）；新增 `dto.ProcessAuditLogResponse`（camelCase 21 键、无 tenantId）；同批删除死路由 `/audit-logs/timeline`（参数名与 handler 读取键从不匹配、恒 1001、零消费方）。回归 `handlers/bpmn/audit_single_surface_contract_test.go`（重复表面 404 / camelCase 精确键集合 / 200-403-401 权限三态，打在真实 `RegisterRoutes` 上）+ `audit_pagination_contract_test.go` 租户断言改读 camelCase；错误泄漏棘轮基线 `monitoring.go` 9→8。前端 `ProcessAuditLog`/`QueryAuditLogsRequest` 删 `tenantId`、`getUserActivity` 不再发送该参数、页面删兜底。`UPGRADE.md` §1.29、`docs/api-reference.md` BPMN 监控章节同批改写。遗留另账：timeline 端点 `ProcessTimelineEntry` 与页面时间线读取字段（`action` vs `eventType`）不同构。

- [x] **v1.6.11 收口——错误泄漏全量清零 + CMDB ontology 自描述锁契约 + 错误语义强制 helper** — ① 错误泄漏 sweep 全量清零 91→0：应用域 14 文件 60 处 + 基线残留 10 文件 31 处，合计 91 处 `common.<fail>(ctx, err.Error())` 形态全部收敛为 `ParamErrorWithErr`/`RespondError`/`NotFoundWithErr`/`BadRequestWithErr`/`AuthFailedWithErr`/`ForbiddenWithErr` 安全 helper；② `common.Response` 新增 `AuthFailedWithErr`(强制 401/2001) 与 `ForbiddenWithErr`(强制 403/2003) 语义强制 helper——`FailWithErr.classifyError` 会把 `*ent.NotFoundError` 归类为 4004/404、把版本冲突归类为 4090/409，对「登录/注册/密码重置」与「跨租户 switch-tenant」语义错误；`handlers/auth/handler.go` 5 处与 `handlers/user/handler.go` 2 处改走新 helper；③ `tests/contract/error_leak_ratchet_test.go` baseline 清零（24 文件条目全部移除），自检条件放宽到 baseline 非空时启用，ratchet 锁仍覆盖 Fail/FailWithData/ParamError/ValidationErrorResponse/AuthFailed/Forbidden/NotFound/InternalError 全系列防止回退；④ CMDB ontology 五键契约 lock：`/api/v1/cmdb/ontology` 新增 `handlers/cmdb/ontology_test.go` 13 用例（version/ciTypes/relationshipTypes/enums/aiTools 五键精确集合、13 条受控词表与 `ent/schema.CIRelationshipTypeVocabulary` 同源、枚举值域与 `common.CILifecycleStatus*` 常量同源、缺租户 fail-closed 401/2001、跨租户拒绝、toolRegistry=nil 时省略整段 aiTools、`parseAttributeSchemaOrRaw` 三态、`listAllCITypes` 空+单页、handlerctx 拒绝后 handler 不再写一次响应）；`dto.CMDBOntologyResponse.AITools` 改为 `*[]CMDBOntologyTool, omitempty`，避免 LLM Agent 把 `aiTools: null` 误判为能力已就绪；⑤ 前端 `src/lib/cmdb/relationship-vocabulary.ts` 三处词表与后端 `ent/schema/ci_relationship.go` 对齐（`impacted_by.icon alert-triangle → activity` / `owned_by.name 归属于 → 被拥有 且 icon user → key` / `used_by.icon share-2 → plug`，13 条受控词表全部一一对应）；⑥ 验证矩阵：`go test ./... -count=1 -short` 全绿（含 handlers/auth/handlers/user/handlers/cmdb/tests/contract/router/CMDB ontology 13 用例 + E4-48 4 用例 + TestErrorLeakRatchet），`DOCS_GATE_ARGS="--strict" bash scripts/docs-gate/run-all.sh` 7/7（CI workflow `.github/workflows/docs-gate.yml` 默认 hard 模式自 2026-09-23 commit `70afc2f51` 启用），`npm run type-check` 0 error。发版就绪

- [x] **v1.6.16（2026-10-07）——弱口令策略接线 + Skill Registry UI + AI Evaluator CI gate + 回调审计脱敏** — ① `pkg/credential` 反孤儿接入：v1.6.14 提交的 `policy.go`（`MinAdminPasswordLength=12`）此前零 caller，`default_credentials_guard`/`pkg/seeder`/`scripts/init_data` 三处接入弱/短口令双检查；② Skill Registry UI（v1.7 E4 提前收口）：`/admin/skills` 管理面板（列表/筛选/manifest viewer/注册/更新/promote/disable），菜单基线同步；③ AI Evaluator CI gate（v1.7 E1 提前收口）：`ai/eval` ROUGE-L stub 升级为 rune-level LCS 标准实现，`backend-ci.yml` 显式追加 `./ai/eval/...` step + `ai-eval.log` artifact，四指标阈值（triage top-1 ≥0.85 / ROUGE-L ≥0.6 / RAG hit ≥0.7 / ROC AUC ≥0.75）进 CI；④ 钉钉/企微 `recordAuditFailure` 把整个 headers map（含 signature/aes_key）marshal 进审计行——审计反而成密钥泄露通道，新增 `connector/redact_headers.go`（11 个敏感键 substring 匹配覆写 `***`）。

- [x] **v1.6.17（2026-10-11）——角色模型重设 + 四组越权/并发缺陷收口** — ① 角色三层词表单一源 `domain/role`（Tier 1 主角色/Tier 2 ITIL 实践/Tier 3 MSP），legacy 角色退役迁移（`users.role=security→security_admin`、BPMN 指派位按映射改写、无成员连 role_permissions 删除）；② `roleRank` 词表陈旧致 admin 可铸 `sysadmin` 等价超管——层级收敛到 `domain/role.Rank` 单一源 + Create/Update 双层 `IsBuiltinCode` 白名单；③ 租户 admin 可跨租户管控任意租户（`/tenants` CRUD 无归属校验）——三写端点补 `ensureTenantScope`，status 白名单堵 `deleted` 绕过；④ DeleteUser 不吊销 token、可删三员/超管——补 `InvalidateUserTokens` + `ensureTargetRankNotAboveCaller`；⑤ SLA 矩阵升级多副本重复通知——级别推进改 claim-once 条件更新；⑥ 变更审批桥接先提交后 CAS 的半写状态——桥接与业务 CAS 同事务；⑦ 服务请求会签并发丢票——审批记录加 `version` 乐观锁 + 事务内重读合并重试；⑧ menus 补 (tenant,path) 唯一约束（ent schema 组合爆炸原因注释在案，勿再试）。

- [x] **v1.6.18（2026-10-12）——CMDB CI 生命周期状态机 + it_admin 派单权限** — `PUT /cmdb/cis/:id/lifecycle` 从 query 参数改为 JSON body，迁移矩阵收敛 `common/ci_lifecycle.go` 单一源（draft→online 单向、scrapped 终态不可逆），非法迁移 409/4090；`version` 乐观锁（冲突载荷含 currentVersion/serverVersion）；退役复用 `retireCIInTx` 全量软删路径 + 受控理由词表（manual/decommissioned/replaced/discovery_missing/end_of_life）；it_admin 补 `ticket:assign`（权威源漏码致 DBOnly 下 IT 管理员无法派单）。

- [x] **v1.6.19（2026-10-17）——SSO（OIDC）整批落地 + 历史文档清理** — 提供方管理（`sso:read/write`）/登录发起回调（挂登录限流）/自助绑定三链路 + 前端登录页闭环（`initiateSSOLogin` 信封解包 bug 修正——旧实现读顶层 `data.redirectUrl` 恒 undefined，既有单测按错误契约 mock 把 bug 钉成通过）+ 装配守卫（生产装配缺失时整套 SSO 静默 404 无人能测出）；修复 main 分支编译断裂（SSO 生成物被引用未入库，origin/main 检出即编译失败）；历史文档清理两轮（10 份 superseded 文档出仓 + mkdocs nav 同步 + 7 处守卫豁免软引用修复）；知识评审状态机最小闭环（`review_status` 枚举 + `knowledge_article_reviews` 审计流水 + 显式流转白名单，409 语义）；docs-gate CJK 假阳性修复（macOS BSD awk collation——脚本顶 `export LC_ALL=C`）；菜单孤儿子树回填 `backfillParentMenus`（父码不满足被剔除时存活子菜单不再整棵丢失）。

- [x] **v1.6.20（2026-10-23）——连接器域五批收口 + 变更域晋升「可用」** — ① 通知域晋升 GA：投递审计读面 `GET /notification-deliveries`（失败重试按「操作谱系幂等」`<原键>:retry:<纳秒>` 防连点双发，LIKE `_` 通配符误判已修）+ 渠道诊断 `GET /notifications/channels`（渠道装配/健康/收件目标三合一）；② 连接器健康面跨租户泄露收口（`HealthCheckAll`→`HealthCheckTenant`，四读面统一；前端健康列键硬编码租户号 0 从未显示过真实结果一并修复）；③ 飞书回调持久化去重接线（`SetInboundDedup` 注入后零读取，实际仍在用内存 map——重启/多副本重复执行同步任务；补 `CleanupExpired` 生产调用点 + 过期覆盖 + `Forget` 失败释放）；④ 连接器凭据轮换从死代码接线 + 市场安装运行态闭环（Revoke 失败不标 uninstalled 防凭据仍活的分裂态）；⑤ 死信人工重放产品化（`/admin/operations` 双 Tab：命令死信四指标 + 失败投递，替代只能 curl）；⑥ 变更域经会签门禁修复（回退路径「一票即批准」）/回滚清单逐项确认/实施留痕（`/fail` 入口 + 失败/回滚原因必填）/终态审批链同事务晋升「可用」；⑦ 问题域 SLA 与评论接入后晋升 GA 候选；⑧ 企微/钉钉连接器 Disabled→Pilot 转正（部署就绪由 READY 环境变量申报）；⑨ 知识发布门禁（未过评审不可发布）+ 问题→知识沉淀闭环前后端；⑩ 能力矩阵 pilotReason 两批纠偏（已修好的阻断项不再当理由对外声明，`TestPilotReasonsDoNotCiteClosedBlockers` 禁词表防复发；`report` 行补录）。


### 当前收敛项

> 2026-09-12 复核：以下各项均按“已完成部分 / 剩余缺口”标注，避免把部分进展误读为闭环。

- [x] **业务旅程 E2E** — 固化 TicketType 安装与绑定、工单创建、Workflow 实例/任务/历史、SLA、Assignment、审计的完整断言。（2026-10-27 收口：两层固化已齐——① `tests/scenarios` 十个业务旅程回归（工单生命周期/事件-问题-知识链/变更审批链/CMDB 影响分析/SLA 违规/租户隔离/连接器 E2E/AI 分诊基线/outbox 重试死信/服务目录全环）共 87 个子测试，在 backend-ci 的 `TESTABLE_PKGS`（`go list ./...` 全量，仅排 migrations）里每次 push 常驻；② ga-gate 在真实 compose 栈上跑 API 冒烟 + TicketType HTTP-worker 全链路 Playwright E2E（`playwright.ticket-type.config.ts`）。fresh-install-gate 每日从空卷证明全新安装链路。）
  - 已完成：服务层 TicketType 安装到审计六环节全链路断言；工作流节点 e2e spec 入库。
  - 剩余：跨服务 E2E 固化与 CI 常驻。
- [x] **可靠执行统一** — 将剩余 ITIL 域从非可靠触发路径迁移到 command/outbox，并提供积压、重放和死信运维。（2026-10-27 代码清点收口：六个 ITIL 域的流程启动全部在创建事务内 enqueue `CommandStartBPMN`——ticket/incident（`service/ticket_service.go:397`、`service/incident_service.go:432`）、change（`handlers/change/repository_impl.go:268`）、service_request（`handlers/service_request/repository_impl.go:255`）、release（`service/release_service.go:164`）；problem 域不经「创建即启动实例」路径，其生命周期动作走 `BPMNApprovalBridge.AdvanceBusinessWorkflowWithClient` 与业务写入同事务的桥接模式（`handlers/problem/service.go:623`），不存在非可靠触发路径。运维面：死信/积压四指标 + 单条/批量重放 + 失败投递重试已在 `/admin/operations` 产品化，命令类型汇总与批量 replay/cancel 齐备。）
  - 已完成：incident 告警与 provisioning 履约接入 outbox；运维命令批量 replay/cancel 与命令类型汇总。
  - 剩余：其余 ITIL 域迁移。
- [ ] **生产数据升级门禁** — 对每次 Schema 变化执行脱敏 PostgreSQL 副本迁移、兼容、回滚与耗时验证。
  - 已完成：migration ledger 调和、preflight、`migration-lint` 与发布证据。
  - 剩余：脱敏副本实跑与回滚演练常态化。
- [ ] **Connector marketplace 生产化** — Feishu、DingTalk、WeCom、Webhook 的真实渠道健康检查、验签、重放与密钥治理。
  - 已完成：钉钉 / 企微入站回调、持久化去重、健康度与凭据轮换。
  - 剩余：真实渠道联调验收；飞书渠道补齐。
- [ ] **AI Audit/Evaluator** — 对建议保留接受/拒绝反馈，并形成可重复的质量基线。
  - 已完成：AI 审计上报链路打通；评测集去占位。
  - 剩余：接受/拒绝反馈闭环与 CI 质量基线门禁。
- [ ] **CMDB 数据治理** — 发现 Job、Diff、调和、退役、质量指标与规模测试。
  - 已完成：AI-Native P0/P1（见上）；v1.6.15 落地数据治理纯函数骨架（退役状态机 online -> retiring -> retired + 受控 reason 词表 + DiffReconciliation 4 类分类 + ComputeQualityMetrics），21 用例全绿，零 Repository 接口扩张。
  - 剩余：service 层 GetReconciliation 接入 diff 与 metrics 字段；handler 暴露 `/cmdb/governance/report` + 退役 PUT；规模测试与发现 Job 闭环工作流。

### 发布门禁

- [x] 后端全量测试、静态分析、前端类型检查/构建、API 契约和 Endpoint ACL 均有自动化入口。
- [ ] 每个候选版本保留 Git SHA、镜像 digest、数据库版本、迁移结果、E2E 与恢复演练证据。
  - 已完成：迁移 preflight 与发布证据、prod 备份与恢复演练脚本具备。
  - 剩余：固化为每个候选版本的门禁清单，并留存执行产物。
- [ ] 生产放行继续按部署环境验收，不能由仓库中的历史“全部通过”报告替代。
  - 补充：仓库内历史测试/部署报告一律标注 `Status: superseded`（见 `docs/documentation-governance.md`），只作证据索引，不作放行依据。

---

## 🟢 v1.7 — Planned (2026-Q4)

**Theme:** AI earns its seat, integrations go live.

### Engineering

- [ ] **AI Evaluator v1** — classification accuracy ≥85%, summarization
      ROUGE ≥0.6, RAG hit-rate ≥70%. Regression suite in CI.
- [ ] **AI telemetry** — capture prompt/response/cost/latency for every
      skill invocation; dashboard at `/api/v1/ai/audit`.
- [ ] **Knowledge base RAG v2** — chunking strategy improvements,
      re-ranking, hybrid search (BM25 + vector).
- [ ] **Skill registry v1** — declarative skill manifests, hot-pluggable
      pipeline, registry UI.

### Product

- [ ] **Feishu / DingTalk / WeCom native connectors** — end-to-end:
      account / approval / IM notification / webhook relay.
- [ ] **Auto-triage (human-in-the-loop)** — AI suggests category,
      assignee, SLA tier; engineer accepts with one click.
- [ ] **SLA forecast skill** — predict SLA breach risk per ticket,
      surface on dashboards.
- [ ] **Full UI i18n** — the UI is Chinese-first through v1.6.x; extract
      the remaining hardcoded strings (currently ~83% of pages) into
      `src/lib/i18n` message catalogs and ship an en-US locale with a
      per-user language switch.

### Quality

- [ ] **Backend coverage** 40% → **55%** overall.
- [ ] **Performance budgets** — k6 baselines for top 10 endpoints,
      enforced in CI.
- [ ] **Trivy + govulncheck** — daily scans, high-severity blockers.

---

## 🔵 v2.0 — Roadmap (2027-Q2)

**Theme:** MSP-friendly, AI-assisted, multi-region.

### Engineering

- [ ] **Coverage 55% → 70%**.
- [ ] **Service decomposition** — split monolithic `itsm-backend` into
      `core` + `workflow` + `ai` + `cmdb` services along bounded contexts.
- [ ] **Event-driven architecture** — Watermill is already in deps;
      promote to first-class pub/sub for incident events.
- [ ] **Multi-region active-active** — Redis Streams + region-aware
      routing.

### Product

- [ ] **MSP billing** — usage metering, invoicing, allocation reports.
- [ ] **AI auto-triage (full)** — replaces the human-in-the-loop step
      from v1.7 with confidence-based auto-accept.
- [ ] **Impact analysis skill** — given a change, predict affected CIs,
      tickets, and downstream SLAs.
- [ ] **Plugin marketplace v2** — signed plugins, sandboxed execution,
      revenue share for authors.

### Quality

- [ ] **SOC 2 Type II readiness** — control mapping, evidence collection,
      audit-ready logging.
- [ ] **Customer-managed keys (BYOK)** for LLM Gateway.

---

## ⚪ v3.0 — Parked (2027-Q4)

**Theme:** Self-hostable AI, agent ecosystem.

- Self-hostable LLM inference (Ollama, vLLM, llama.cpp) — drop the
  external OpenAI dependency for privacy-sensitive deployments.
- Agent marketplace — third-party agents that can act on the ITSM
  data model under strict RBAC.
- Mobile PWA with offline-first ticket intake.
- Multilingual UI (zh-CN baseline; en-US, ja-JP, ko-KR planned).

---

## 🛠️ Always-On Tracks

These don't belong to a single release; they ship incrementally:

### Testing & Quality

- Incremental coverage gate (60% on new code) — landed
- End-to-end smoke on every PR — landed v1.0
- Frontend visual regression — planned v1.7
- Property-based tests for critical parsers (BPMN XML, RAG chunking)
  — planned v1.7

### Security

- CodeQL + Trivy + govulncheck — landed
- Quarterly threat-model review
- Annual pen-test

### Open-Source Governance

- Issue triage SLA (48h first response, 14d close-or-fix) — ongoing governance target
- Monthly community digest
- Quarterly maintainer rotation review

### Developer Experience

- `make dev-*` unified dev environment (already landed v1.0)
- `itsm-cli` for ops (deploy/seed/inspect) — landed v1.0
- `itsm-skill` for OpenClaw / Codex agents — landed v1.0
- Container image size reduction (distroless base) — planned v1.7

---

## 📊 Key Metrics

We track these on every release. Numbers below are post-v1.0 GA baseline
and the **target** for the next major release.

| Metric | Historical v1.0 baseline | v1.7 target | v2.0 target |
|:---|---:|---:|---:|
| Backend coverage | ~2% | 55% | 70% |
| Frontend coverage（门槛口径 `src/lib/**`，阻断） | ~10% | ≥80%（现行门槛） | ≥85% |
| Frontend coverage（产品口径 `src/**`，只报告） | 待首次产出（`npm run test:coverage:product`） | 30% | 60% |
| E2E smoke coverage | 11 APIs | 25 APIs | 50 APIs |
| Mean PR → first review | TBD | < 48h | < 24h |
| Mean issue → first response | TBD | < 48h | < 24h |
| AI triage accuracy | — | 85% | 92% |
| Open stale issues | varies | < 30 | < 15 |

> **2026-09-12 实测基线**（防止与上表历史口径混淆）：`go test -cover ./handlers/...` 为 **26.2%**。
> 该口径仅覆盖 `handlers/` 垂直分层包；历史 v1.0 基线的 ~2% 为 `service` + `controller` 口径，
> 两者不可直接比较。覆盖率数据应由 CI 生成并写入，避免手抄导致漂移。
>
> **前端覆盖率口径**（同样为防止误读）：`itsm-frontend/jest.config.js` 的 `collectCoverageFrom`
> 仅包含 `src/lib/**`，不含 `src/app/**` 与 `src/components/**`。因此上表 Frontend coverage 与
> 80% 门槛都是 **`src/lib` 的局部口径**，不等于产品代码覆盖率；把口径扩到页面/组件，
> 或明确按局部口径对外表述，两者必须二选一（由 Gate C.6.5 守卫）。
>
> **2026-10-17 决策：双口径并存，不再二选一。** `src/lib/**` 只占产品代码约 30%（45k/150k 行），
> 把阻断门槛扩到 `src/**` 会让门槛从 80% 断崖到 30% 量级——那不是更诚实，而是失去约束力
> （30% 几乎拦不住回归），还会诱导「渲染即通过」的凑数断言。因此：
> **`src/lib/**` 继续作为阻断门槛（80%），另以 `npm run test:coverage:product`
> 产出 `src/**` 产品口径数字（只报告、无 threshold）**，用于观察 UI 覆盖是否在推进。
> 上表两行分别对应这两个口径，缺一即口径漂移（C.6.5 守卫：存在该脚本时 ROADMAP 必须同时披露 `src/**`）。

---

## 🤝 How to Influence the Roadmap

1. **File an issue** with the `feature-request` template and link to
   the milestone you think it belongs in.
2. **Vote** on issues with 👍 — we sort milestone backlogs by reactions.
3. **Propose a major change** via the RFC process:
   `docs/rfcs/0000-template.md`.
4. **Pick up a "good first issue"** — every track has at least one.

---

## 📜 Changelog

Major releases are tracked in [CHANGELOG.md](./CHANGELOG.md) and via
GitHub [Releases](https://github.com/heidsoft/itsm/releases).
