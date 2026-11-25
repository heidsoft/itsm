# CloudMesh ITSM 当前产品现状（证据基线）

> Status: current · 基线日期：2026-11-23 · 仓库 HEAD：`a1d4dfba1`（`main`，本地领先 `origin/main` 3 个提交）
>
> **证据模式**：本轮为静态代码/测试审阅 + 仓库内既有产物核对。**未执行** `go test ./...`、`npm test`、Playwright E2E，未部署、未打 tag、未创建 PR。所有“Implemented”结论指“代码与测试证据支持该能力存在并接线”，不等价于“本轮运行验证通过”。
>
> **工作树边界**：`website/index.html` 存在用户 WIP 修改，本轮只读、未触碰。

---

## 0. 结论摘要

CloudMesh 已不是早期原型，而是一个**工程底座显著强于商业交付能力**的成熟代码库：

- **强项**：核心 ITIL 主链路（工单/事件/问题/变更/服务请求/SLA/CMDB/知识）具备真实状态机、租户谓词、CAS 乐观锁、事务型 outbox；AI 工具调用治理（Gate 2.5 零信任主体 + 强制人工审批 + 执行期重校验）达到企业级设计水准；命令总线具备 fencing / 退避 / 死信 / 重放全套语义并有回归测试。
- **弱项**：**没有一个被客户验证的付费需求**；社区版/企业版边界是文档里明确记录的开放问题；销售与定价文档是占位符；CI 最近 100 次运行中 39 次失败且 docs-gate 不是分支保护必需检查；版本号在至少 7 个表面互相矛盾；K8s 仅为脚手架；入站 IM 消息到工单的业务派发未实现。
- **一句话判断**：**产品深度足够支撑 PoC 与技术评审，但尚不足以支撑可承诺、可收费、可复制交付。**

按用户要求的状态词表（Implemented / Partial / Missing / Broken / Unknown）逐域判定见第 3 节。

---

## 1. 可复核的度量基线

| 指标 | 数值 | 复核方式 |
|:---|:---|:---|
| Go 源文件（不含 `ent/`） | 4,681 | `find itsm-backend -name '*.go' -not -path '*/ent/*' \| wc -l` |
| Go 测试文件 | 914 | `find itsm-backend -name '*_test.go' \| wc -l` |
| Ent schema / 生成代码 | 137 / 1,118 | `ls itsm-backend/ent/schema/*.go`、`find itsm-backend/ent -name '*.go'` |
| SQL 迁移脚本 | 93 | `ls itsm-backend/migrations/*.sql \| wc -l` |
| `handlers/<domain>/` 域包 | 70 | `find itsm-backend/handlers -mindepth 1 -maxdepth 1 -type d` |
| 带 `service.go` / `repository*.go` 的域 | 43 / 34 | `find itsm-backend/handlers -name ...` |
| 路由注册（方法级） | 755 | `rg '\.(GET\|POST\|PUT\|PATCH\|DELETE)\(' itsm-backend/router --glob '*.go'` 排除 `_test.go` |
| 前端页面 / 测试 / API client | 173 / 308 / 70 | `find`/`ls`，见第 1 节来源 |
| CI 工作流 | 17 | `ls .github/workflows/` |
| 最新 tag / CHANGELOG 已发布版 | `v1.6.60` / `1.6.60`（2026-11-22） | `git tag --list --sort=-v:refname`、`CHANGELOG.md:62` |

> 路由数为静态文本计数，含测试辅助注册之外的真实方法注册；与运行时实际生效路由数可能有少量差异。

---

## 2. 判定方法

遵循仓库 `AGENTS.md` 与本任务的证据纪律：

1. **禁止按文件名、菜单名、接口名或文档宣称推断完成度。**
2. 对每个域追踪生产调用链：`Router → Handler → Service → Repository`，确认真实 HTTP 入口与中间件（Auth/RBAC/Tenant/Audit）。
3. 至少读到一个业务规则实现（状态机校验、租户谓词、事务边界、幂等键）才算 Implemented 候选。
4. 有测试但只覆盖孤立 helper、生产路由未接线 → 不得判 Implemented。
5. 证据不足则明确标 `Unknown`，不用推测填充。

状态定义：

| 状态 | 含义 |
|:---|:---|
| Implemented | 代码存在、生产接线、有业务规则与回归测试证据 |
| Partial | 存在真实实现但缺关键闭环、或部分路径仍为骨架/未接线 |
| Missing | 仓库内未发现对应实现 |
| Broken | 有实现但存在明确缺陷/自相矛盾，导致不可用或不可信 |
| Unknown | 本轮未取得足够证据，需进一步验证 |

---

## 3. 目标能力地图逐域现状

### A. ITSM Core

| 能力 | 状态 | 最强证据 |
|:---|:---|:---|
| 工单 Service Desk | **Implemented** | `handlers/ticket/{handler,service,repository,repository_impl,contracts,stats}.go`；`router/ticket_routes.go:15-72`；`service.go:201,364,377,417,464,771` 调 `common.IsValidTicketStatusTransition`；`repository_impl.go:440-442` CAS + tx hook；`handlers/ticket/handler_test.go` + `router/ticket_*_route_test.go` |
| 事件管理 Incident | **Implemented** | `router/incident_routes.go:12-33`（escalate/ack/resolve/close/reopen/assign）；`service.go:410` 状态机；`service.go:204-205` `client.Tx` + `:276` `commandbus.EnqueueTx`；`repository.go:22` id+tenant+version |
| 服务请求 Service Request | **Implemented** | `router/service_request_routes.go:15-49`；`service.go:1390-1405` 操作态白名单；`repository_impl.go:105,188,263` tx + EnqueueTx；`service_request_cas_conflict_test.go`、`service_request_outbox_test.go` |
| 问题管理 Problem | **Implemented** | `router/problem_routes.go:12-36`；`service.go:717-732` `problemStatusTransitions` 单一来源；`problem_status_machine_test.go`、`knowledge_visibility_test.go` |
| 变更管理 Change | **Implemented** | `router/change_routes.go:12-56`（submit/approve/schedule/start/complete/rollback/cancel、approval/risk/PIR）；`repository.go:21` `UpdateStatusCAS(id, tenantID, expected, target)`；`approval_quorum_gate_test.go`、`update_status_cas_test.go` |
| 发布管理 Release | **Implemented（分层偏离）** | `handlers/release/` 仅 `handler.go`/`contracts.go`，业务逻辑在 `internal/services/release_service.go:449-492`（状态白名单 + 行级 guard + CAS）；`router/release_routes.go:12-23` |
| SLA 管理 | **Implemented** | `router/sla_routes.go:13-58`；`service/sla/engine.go` 权威时间引擎；`handlers/sla/compliance_violation_test.go`；违规按权威记录计算（见 ROADMAP 2026-11-08） |
| 知识管理 Knowledge | **Implemented** | `router/knowledge_routes.go:13-48`；`repository.go:17-31` tenant + visibility；`outbox.go:31-62` 向量同步事务；`publish_bypass_test.go` 堵住旁路 |
| 服务目录 Service Catalog | **Implemented** | `router/service_catalog_routes.go:12-25`；tenant-scoped `repository.go:9-13`； CRUD/search/stats 测试齐全 |
| CMDB / 配置管理 | **Implemented（深度不足）** | `router/cmdb_routes.go:19-204`（CI/类型/属性/视图/导入导出/关系/云账号/对账/认证）；`repository_impl.go:1114-1149` tx + EnqueueTx。但类模型/IRE/业务服务语义缺失，见 `docs/product/servicenow-gap-analysis.md` |
| 资产管理 Asset | **Implemented（分层偏离）** | `handlers/asset/routes.go:12-26`；`service.go:37-46,77` 直接用 Ent（无 repository 层）；`cross_tenant_http_test.go` |
| 审批中心 Approval | **Partial** | `handlers/approval/` 无 service/repository，逻辑在 `internal/services/approval_service.go:39-49,92`；路由挂在 `router/ticket_routes.go:206-233`；`handlers/approval/routes.go` 是 handler 方法而非路由注册（命名误导） |
| 通知 Notification | **Implemented** | `router/common_system_routes.go:203-227`；`internal/services/notification_outbox.go:25-62`；`notification_delivery.go` schema 一命令一投递；`notification_delivery_command_handler.go:60-65` 去重、`:272-339` 脱敏 |

**核心结论**：12 个核心域中 11 个 Implemented、1 个 Partial。C.6.3 门禁（`scripts/docs-gate/check-product-drift.sh:318-410`）当前无豁免、扫描结果 0 个未接线域包。

### B. IT Operations（目标能力，非现状承诺）

| 能力 | 状态 | 证据与边界 |
|:---|:---|:---|
| 监控告警接入 | **Partial / Unknown** | 存在 `router/alert_routes.go`、`handlers/cloudsignal/`；未核实到从告警源到事件/工单的稳定闭环与生产验收 |
| 事件聚合与关联 | **Partial** | ROADMAP 2026-11-07 记录 BPMN 自动建事件收敛单一发号器、事件根因分析 Tab 接权威契约；端到端效果未运行验证 |
| 自动化巡检 / 运维任务执行 | **Partial** | `handlers/automation_rule/`、`handlers/operations/`、`handlers/provisioning/` 与 durable command 存在；具体任务种类与真实环境执行未核实 |
| 数据库运维治理 | **Partial（骨架）** | CHANGELOG 2026-11-17 明确 DDL 流程为“权限骨架与工具注册，不是端到端可用”，G1 未完成 |
| 多云资源管理 | **Partial** | `handlers/cmdb/` 云账号/发现/对账 + `connector/` 有实现；ROADMAP 记录发现水位线与新鲜度校验；生产多账号验收未知 |
| 应用部署与运行环境 | **Unknown** | `handlers/application/` 存在，但本轮未核实到部署执行闭环 |

> 该节整体证据强度低于 A 节。按证据纪律标记为 **Unknown 优先**，不得当作已具备的产品能力对外承诺。

### C. AI-Native

| 能力 | 状态 | 最强证据 |
|:---|:---|:---|
| LLM Gateway | **Implemented** | `internal/services/llm_gateway.go:32-66,118-139,149-249,256,283,332`；`:126-139` 拒绝伪造模型名；每消费者有确定性降级 |
| RAG / 知识检索 | **Implemented（v2 Partial）** | `rag_service.go:440,532,714,765,859` 全路径 tenant + 发布态；`:658,723,785-801` 向量结果二次可见性过滤；`:811-844` 分类拒绝 fail-closed。缺 chunking / 真重排（ROADMAP:203-209，“hybrid”是开关而非 BM25 融合） |
| AI Evaluator | **Partial** | 4 指标阈值在 `ai/eval/eval_test.go:362,379,409,430`（triage top1≥0.85、summarize ROUGE-L≥0.6、RAG hit≥0.8、prediction AUC≥0.75），CI step `backend-ci.yml:300-310`；真实模型档是手工命令 `cmd/ai-eval/main.go:28`，CI 不消费。ROADMAP:142 仍写 hit≥0.7（漂移） |
| Skill Registry | **Implemented** | `handlers/skill/handler.go:593,685,724`：manifest 权限校验 + 市场授权 + 决策审计（含失败）；内置 12 技能 `builtin_skills.go:25-38` |
| Tool Execution / HITL 治理 | **Implemented（最强项）** | `handlers/ai/service.go:251-348` Gate 2 + Gate 2.5；`:298-299` 提示词注入改不了 DB 权限；`:334-346` 写工具强制审批、双审批人不可自批；`repository_impl.go:400-402` `EnqueueTxIdempotent`；`tool_executor.go:69-139` 执行期重验主体与策略指纹 |
| Agent Runtime / Registry | **Partial** | 生产侧是固定 12 技能集 + `maxToolRounds=5` 的有界工具循环（`rag_service.go:1170-1293`），不是通用 Agent 注册表；`itsm-agent/` 是独立的 GitHub issue→PR 开发机器人，不接后端 router/worker；`itsm-skill/index.js` 是 60 行开发 CLI |
| Agent Memory | **Missing** | `itsm-backend/memory/` 仅含遗留 Python venv，无源码 |
| Agent Observability | **Partial** | `handlers/ai/control_plane.go:15-34,148-235` 汇总工具面/队列/策略/拒绝；审计字段 `entity.go:43-72` 含 `permissionCheck`/`roleSnapshot`/`policyVersion`/`runId`；无统一 Trace 产品面 |
| Context Engineering / ActionFlow | **Partial** | 阶段散落在 `investigation.go:17-31`（Context→Agent→Policy→Suggestion→Audit）与 durable outbox；无统一声明式 Trigger→Verification 编排器；触发器是 HTTP 请求而非事件 |
| Auto-triage 产品化 | **Partial** | 建议接受度有遥测，ROADMAP:217 仍未勾选 human-in-the-loop auto-triage |

### D. Governance

| 能力 | 状态 | 最强证据 |
|:---|:---|:---|
| 身份与租户上下文 | **Implemented** | `middleware/auth.go:166-224,349-359` 身份只来自签名 claims；`middleware/tenant.go:25-30,144-157` 忽略 `X-Tenant-ID`，任何提示来源必须等于 JWT tenant，缺失 401 fail-closed |
| RBAC / 权限目录 | **Implemented** | `middleware/rbac.go:657-724`；`router/router.go:435-438` 全局挂载；`route_permission_guard_test.go:144-261` AST 扫描所有读写路由；`permission_code_catalog_guard_test.go:30-119` 保证权限码在目录中 |
| 多租户隔离 | **Implemented（应用层）/ Partial（RLS）** | 约 119 个仓库文件带 tenant 谓词；`tests/scenarios/scenario6_tenant_isolation_test.go:29-198`、`tests/rbac/cross_tenant_test.go:37-571`；RLS 仅 `changes`/`vectors` 两表试点（`tests/multi_tenant/rls_matrix_test.go:1-25`） |
| 审计 | **Implemented（有边界）** | `middleware/audit.go:362-405,230-237` 审计置于链首，覆盖写操作与 4xx/5xx，4KB 截断 + `MaskSensitiveFields`；但审计写是独立事务，失败只计数与降级日志，不阻断业务（`:340-350`） |
| MSP | **Implemented** | `router/msp_routes.go` + `msp_route_registration_test.go`；`handlers/msp/allocation_service_impl.go:170` 审计 |
| SSO / OIDC | **Implemented** | `router/sso_routes.go:22-48`、`handlers/auth/sso_oidc.go`（Discovery/JWKS/PKCE/state/nonce）；`sso_routes_test.go:175-426` 端到端 mock IdP。缺 nonce 按设计仅告警 |
| 初始化 / 基线 | **Partial** | 版本化+lease/fencing+校验和+凭据保护已实现（`internal/initialization/engine.go:20-34,141-223`、`sql_store.go:43-115,365-400`、`default_credentials_guard.go`）；但字段级三方合并未实现（`itsm-backend/docs/initialization-data-classification.md:79-81`），6 个基线 schema 仍属“第三阶段待确认”（`docs/engineering/system-baseline-design.md:161`） |
| 升级与迁移 | **Partial** | `UPGRADE.md` + 93 个 SQL 迁移 + `cmd/migration-lint`；但 `UPGRADE.md:5` 称稳定版 1.6.20，与 HEAD/tag 漂移 |

### E. Industry Solutions

| 行业方案 | 状态 | 证据 |
|:---|:---|:---|
| 物流仓储 / 制造 IT-OT / 电商 / MSP NOC / 园区设施 | **Missing（产品化）/ Hypothesis（需求）** | 仅有 `docs/product/industry-verticals-analysis-2026-08-01.md` 的能力基线与购买动机假设；无行业包、无行业模板、无客户验证 |

---

## 4. 已核实的生产调用链

```
itsm-backend/main.go
  → internal/bootstrap/app.go                  # gin.New() :1396, handlers :418-1335
    → router/router.go:288 SetupRoutes
      → /api/v1 public group :362 / auth group :425
        → Auth → RBAC → CSRF :435-443
          → TenantMiddleware :533
            → RequirePermission（逐路由）
              → handlers/<domain>/handler.go
                → handlers/<domain>/service.go（或 internal/services/<x>_service.go）
                  → handlers/<domain>/repository_impl.go
                    → Ent / PostgreSQL
```

典型实例：

- **工单**：`router/ticket_routes.go:17` `POST /tickets` → `handlers/ticket/handler.go` → `service.go:201/364/...` 状态机 → `repository_impl.go:440-442` CAS。
- **AI 工具调用**：`handlers/skill/handler.go:593` `POST /:code/invoke` → manifest 权限 → `handlers/ai/service.go:251-348` 双闸门 → `EnqueueTxIdempotent` → worker 执行期重验。
- **通知**：业务事务内 `notification_outbox.go:25-62` → `operational_commands` → delivery handler 去重与投递。

**分层偏离登记**（不是重复实现，但需收敛）：`release`、`approval`、`notification` 的业务逻辑仍在 `internal/services/`；`asset` service 直接用 Ent；`service_request` 无 repository 接口。旧的扁平 `service/` 已是 8 个子目录、97 个文件的 `service/{approver,marketplace,lifecycle,common,scenario,bpmn,cloud,sla}`，不是与 handlers 并行的第二套实现。

---

## 5. 质量与交付现状

### 5.1 CI 与测试

- 17 个工作流；`backend-ci.yml:49-140` lint 实际用 **gofumpt + staticcheck**（不是 golangci-lint；后者只在 `Makefile:112` 出现）。
- `backend-ci.yml:242-323` 测试含 AI evaluator gate（`:300-310`，显式处理 `PIPESTATUS[0]` 防止 `tee` 掩盖绿），之后跑全量 coverage。
- `frontend-ci.yml:48-67` 跑 lint/type-check/Jest/build，**不含 E2E**；E2E 只在 `ga-gate.yml:106` 跑单个 `ticket-type-full-chain.spec.ts`。
- 仓库内真实 Playwright 用户流存在：`tests/e2e/happy-path.spec.ts:39-82` 覆盖建单→接受→解决→关闭并断言 `status==='closed'`。
- **历史证据快照**：`output/github-ci-review-2026-11-14.md:27` 记录最近 100 次 Actions = 56 success / **39 failure** / 5 cancelled；`:32-45` 列明 gofumpt 90 文件、Swagger 陈旧、2 个 Jest 断言、RLS 角色断言、Trivy HIGH CVE、star-history 直推失败。该报告是 2026-11-14 快照，其中“main 领先 7 个提交”已过时（当前领先 3）。
- 仓库内测试产物：`itsm-frontend/junit.xml:2` 3,363 tests / 0 failures，但时间戳 **2026-08-16**（陈旧）；`test-results/junit.xml:2` 6 tests / 0 failures（2026-11-23 局部）；`ticket-type-full-chain.xml:2` 1 test / 1 failure（2026-10-03）。
- `CHANGELOG.md:10-37` 自报“后端 126 包 ok；前端 3467 pass”，同时明确登记 **7 个后端 / 9 个前端存量失败**（在干净 HEAD 可复现）。
- `docs/testing/coverage-baseline.json:2-3` Go 13.6% / Jest 0.47%，来自 v1.1，文档自述为历史下限而非当前测量。

### 5.2 部署

- Docker Compose 体系完整：dev(19)/prod(25)/demo(2)/e2e(14)/fresh-gate(3) + saas(15)；`Dockerfile.prod` 多阶段、非 root、固定镜像。
- K8s 为**起步脚手架**：`deploy/k8s/README.md:1-10,32-45` 明示未覆盖 AI sidecar、MinIO、监控、HPA/PDB、多副本 Postgres/Redis；无 ITSM Helm/Kustomize（仅 OpenViking 自带 vendored chart）。
- 升级路径实质存在（`UPGRADE.md` 285KB + 迁移脚本），但版本口径漂移。

### 5.3 文档门禁

- `scripts/docs-gate/run-all.sh:4-27` 共 **8 道门**（C.1–C.8）。C.6/C.7 默认 hard，C.1–C.5 默认 advisory（`--strict` 才 hard）。
- `.github/workflows/docs.yml:58-62` 以 `strict: false` 调用；`docs-gate.yml:198` 文案仍写“All 6 gates block”（与 8 道门不符）。
- `output/github-ci-review-2026-11-14.md:14-16` 记录分支保护仅 4 个必需检查（backend-ci lint/build/test、frontend-ci Frontend），**docs-gate 不在其中**，因此“hard”门实际不阻断合并。

### 5.4 版本口径漂移（Broken）

| 表面 | 声称版本 |
|:---|:---|
| `git tag` 最大值 / CHANGELOG 最新发布 | `v1.6.60` / `1.6.60`（2026-11-22） |
| `ROADMAP.md:31` | tag 账本停在 v1.6.37、prod 部署 v1.6.48 |
| `UPGRADE.md:5` | 稳定 1.6.20 |
| `docs/documentation-governance.md:25` | 最新 v1.6.9 |
| `scripts/version-drift-baseline.json:3-10` | 规范 v1.6.59，其他表面 1.6.8/1.6.9/1.6.19 |
| `.env.prod:76` / `output/build-v1.6.71-20261123.log` | v1.6.70 / 构建 v1.6.71 |
| `README.md:9` Go 徽章 1.25.13 vs `go.mod:3` | 1.25.13 / **1.26.10** |

2026-11-22 的守卫只登记历史漂移、阻止新增，未做对账。

---

## 6. 商业化现状

| 维度 | 现状 | 证据 |
|:---|:---|:---|
| Community / Enterprise / Industry 边界 | **未定义** | `docs/product/itsm-commercial-capability-contract.md:213-214` 明确把“开源版 vs 商业版边界”“社区版/企业版分界”列为 OPEN QUESTION #6/#7 |
| 首个目标客户 / 部署形态 / 首个 IM 渠道 | **未决策** | 同文件 `:209-211` 问题 #1/#2/#3 |
| 销售与定价物料 | **占位符** | `docs/business/{SALES_PLAYBOOK,POC_PLAYBOOK,IDEAL_CUSTOMER_PROFILE,PRICING_AND_PACKAGING}.md:4` 均写“PLACEHOLDER — 不得虚构客户/价格”；建议测试价私有化 10–30 万起、PoC 1.5–3 万，仅为建议 |
| 付费需求证据 | **无** | 全仓未发现客户名、LOI、签署 PoC 或付费承诺；所有相关表述均为“不得虚构”红线 |
| 官网 | 静态定位页 | `website/index.html:295-296,394-408` 为 Apache-2.0 定位 + 联系方式，无版本/定价矩阵（本轮只读）。产品站 `cloudmesh.top` 的实际线上内容本轮未抓取 |
| ServiceNow 差距 | **仅 CMDB** | `docs/product/servicenow-gap-analysis.md` 只覆盖 CMDB：无类模型/CSDM、无 IRE、生命周期状态未分、关系缺业务服务语义、数据质量未闭环 |

**需求证据等级**：本轮全部候选需求只能归为 `Observed Problem`（仓库内有缺陷/缺口证据）或 `Hypothesis`（行业分析假设）。**没有任何一条可归为 `Verified Demand`。**

---

## 7. 关键矛盾与事实漂移

| # | 矛盾 | 影响 |
|:---|:---|:---|
| 1 | 版本号 7 个表面互相矛盾（§5.4） | 客户无法判断该部署哪个版本；升级承诺不可信 |
| 2 | ROADMAP 称 tag 停 v1.6.37、prod v1.6.48，实际 tag 到 v1.6.60 | 路线图进度失真，发布判断不可依赖 |
| 3 | docs-gate C.6/C.7“hard”但不在分支保护必需检查 | 文档漂移可合入 |
| 4 | ROADMAP RAG hit≥0.7 vs 测试 ≥0.8 | 评测门槛表述错误 |
| 5 | ROADMAP 称 workflow 维持 Preview，但 CHANGELOG 多处描述“全绿/可用”；跨域人工签字栏全空 | 成熟度结论不能由脚本代替人工签字 |
| 6 | README Go 1.25.13 vs go.mod 1.26.10 | 开发环境与构建环境不一致 |
| 7 | `docs/product/README.md:16` 声明本目录不维护并行总控计划，根目录 `ROADMAP.md` 为唯一事实源 | 本轮新增的 `docs/product/roadmap.md` 必须定位为“ subordinate 四周执行计划”，不得复制根路线图 |

---

## 8. 本轮未验证项（Unknown 清单）

1. **运行时行为**：未启动服务、未跑任何测试套件；所有 Implemented 结论为静态证据结论。
2. **真实租户跨域验收**：`docs/acceptance/workflow-cross-domain-signoff.md` 9 个签字字段为空，当前库仅 1 租户 / 1 admin，不具备双租户验收条件。
3. **生产连接器的真实租户交付**：飞书/企微/钉钉只完成验签/解密/去重，未与真实租户做生产验收（capability pilotReason 明确记录）。
4. **数据库 RLS 全覆盖**：仅 2 表试点，生产隔离依赖应用层谓词。
5. **性能 / 容量 / 灾备**：未做压测、未验证 pg 灾难恢复演练结果。
6. **官网线上内容**：未访问 `https://cloudmesh.top/`。
7. **B 节 IT Operations 多项能力**：只有 handler/route 存在性证据，无端到端闭环证据。

---

## 9. 主要证据索引

- 入口与路由：`itsm-backend/main.go`、`internal/bootstrap/app.go`、`router/router.go`
- 核心域：`itsm-backend/handlers/{ticket,incident,problem,change,service_request,sla,cmdb,knowledge}/`
- AI 治理：`itsm-backend/handlers/ai/service.go`、`handlers/ai/tool_executor.go`、`handlers/skill/handler.go`、`handlers/mcp/service.go`
- 可靠性：`itsm-backend/internal/commandbus/commandbus.go`、`ent/schema/operational_command.go`、`internal/services/timer_scheduler.go`、`internal/services/notification_delivery_command_handler.go`
- 初始化：`itsm-backend/internal/initialization/engine.go`、`internal/initialization/sql_store.go`、`itsm-backend/docs/initialization-data-classification.md`
- 质量：`output/github-ci-review-2026-11-14.md`、`CHANGELOG.md`、`scripts/docs-gate/run-all.sh`
- 商业：`docs/product/itsm-commercial-capability-contract.md`、`docs/business/`、`docs/product/industry-verticals-analysis-2026-08-01.md`

---

## 10. 2026-11-24 跟进：版本口径对账（Slice A）

第 5.4 节是 2026-11-23 的基线快照。2026-11-24 已执行 Slice A，以下表面已统一到 **v1.6.60**：`git tag`、CHANGELOG 最新发布、`.env.prod.example`、`router.defaultAppVersion`、`handlers/mcp/adapters.go` 兜底、`router/ga_readiness.go`、`main.go` swagger `@version`（含重生成的 `docs/{docs.go,swagger.json,swagger.yaml}`）、`itsm-frontend/package.json` 与 lock、`docs/documentation-governance.md`、`UPGRADE.md`、`ROADMAP.md`、`deploy/k8s/*` 示例标签；README 与 README.en 的 Go 徽章对齐 `go.mod` 的 1.26.10；ROADMAP 的 RAG 阈值表述改为 ≥0.8；`docs-gate.yml` 的“6 gates”改为“8 gates”。

守卫已扩展为 10 个版本表面 + 2 个 Go 徽章，`node scripts/check-version-consistency.js --canonical=v1.6.60 --strict` 通过，5 个测试（含逐表面反向注入）全绿。

**仍未解决 / 不由本批控制**：

1. 本地未入库的 `.env.prod` 仍为 `VERSION=v1.6.70`，`output/build-v1.6.71-20261123.log` 构建 v1.6.71 —— 需发布负责人按部署流程人工确认，本批未触碰（属人工审批边界）。
2. `docs-gate` C.6/C.8 仍因并行会话 WIP 红灯；分支保护必需检查仍只有 4 项，`docs-gate` 与新增的 `Version Truth` 作业需在 GitHub 仓库设置中人工加入，本批无法远程配置。
3. `CHANGELOG.md` 被并行会话暂存，本批未写入 `[Unreleased]` 条目，需在并行会话提交后补记（不得把两个主题混进同一 commit）。
