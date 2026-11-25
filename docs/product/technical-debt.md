# CloudMesh ITSM 技术债清单

> Status: current · 基线日期：2026-11-23 · 依赖：[current-state.md](./current-state.md)、[gap-analysis.md](./gap-analysis.md)
>
> 收录标准：有仓库内证据、会影响交付质量/可维护性/可信度的债务。**按“是否阻塞最近一次可信交付”分组**，不按代码量排序。
>
> 债务 ≠ 缺陷：部分条目是已知并被门禁监测的存量（如 Ant Design 冻结基线），不应在本轮无差别清理。

---

## 1. T0：阻塞可信交付（必须在四周计划内处理）

### TD-01 版本与事实源漂移

- **证据**：`git tag` 最大 `v1.6.60`；`ROADMAP.md:31` 写 tag v1.6.37 / prod v1.6.48；`UPGRADE.md:5` 写 1.6.20；`docs/documentation-governance.md:25` 写 v1.6.9；`scripts/version-drift-baseline.json` 写 v1.6.59；`.env.prod:76` 写 v1.6.70；`output/build-v1.6.71-20261123.log` 构建 v1.6.71；`README.md:9` Go 1.25.13 vs `itsm-backend/go.mod:3` 1.26.10。
- **影响**：交付版本、`/version`、发布说明、升级文档四者不可互证；合同附件无法引用。
- **风险**：客户验收争议；错误升级包；安全补丁是否内含无法判断。
- **修复**：单一版本来源（tag + CHANGELOG + 运行时 `/version`）；修正 6 处文档/配置；CI 加跨表面一致性断言；README Go 版本同步。
- **工作量**：S–M。**人工审批**：生产 `.env.prod` 改动的发布需按既定部署流程，本轮不得直接改生产配置。

### TD-02 CI 红灯存量与门禁不阻断

- **证据**：`output/github-ci-review-2026-11-14.md:27`（56 success / 39 failure / 5 cancelled）、`:32-45`（gofumpt 90 文件、Swagger 生成产物落后、2 个 Jest 过期断言、RLS 角色断言、Trivy HIGH CVE + govulncheck、star-history GH006）；`scripts/docs-gate/run-all.sh:21-27`（C.1–C.5 advisory）；`.github/workflows/docs.yml:58-62` `strict:false`；分支保护仅 4 项必需检查。
- **影响**：绿灯不代表可发布；新失败容易淹没在存量噪声里。
- **风险**：带 CVE 的镜像进入私有化交付；文档漂移合入无人察觉。
- **修复**：按“基线存量 / 本轮新增”分类红灯并登记；处理 gofumpt/Swagger/Jest 三类确定性项；Trivy HIGH 修复或书面风险接受；docs-gate 纳入必需检查或显式记录豁免。
- **工作量**：M（gofumpt 涉及并行会话 WIP，需单独批次）。

#### CI 失败分类（2026-11-14 快照，2026-11-24 对账）

`output/github-ci-review-2026-11-14.md:27` 给出 56 success / 39 failure / 5 cancelled；仓库只逐项记录了其中高频失败，无法对 39 项逐一复核。按可证据化的项分类如下：

| 项 | 分类 | 状态 | 依据 / 边界 |
|:---|:---|:---|:---|
| Swagger 生成产物落后注解 | 存量基线 | **已修（本批）** | `make swagger-gen` 重生成；连续两次生成 SHA-256 一致（`39e9b678…`） |
| gofumpt 90 文件 | 存量基线 | 待修（独立批次） | 涉及并行会话 WIP，未擅自批量改写 |
| 2 个 Jest 过期断言 | 存量基线 | 待确认 | CI review 称当轮已修，但仓库无当前全量绿色产物 |
| RLS 角色断言 | 存量基线 | 待修 | 需真实 Postgres 栈复现，属数据库迁移改动 |
| Trivy HIGH CVE + govulncheck | 存量基线 | 待修 / 书面风险接受 | 本批未取得当前扫描结果 |
| star-history GH006 直推失败 | 存量基线 | 待修 | 定时任务发布链路，非产品代码 |
| docs-gate C.6：`shared_services_go_files=359>345`、`bootstrap_app_lines=2289>2275` | 基线已有（并行会话 WIP） | 未调基线 | 本批前后 `--strict` 均同样失败，不代改他人棘轮 |
| docs-gate C.8：新增 `bpmn_timer_runtime.go` | 基线已有（并行会话 WIP） | 未调基线 | 同上 |
| 前端 `customers/page.tsx` 缺 `historicalNames` | 基线已有（并行会话 WIP） | 未修 | 本批未碰该文件；`npm run type-check` 因它失败 |

> 因此**不能**把“本批验证通过”表述为“全量绿灯”：本批自有检查全绿，仓库整体仍有他人 WIP 造成的红灯。

### TD-03 文档自相矛盾与门禁失效

- **证据**：`docs-gate.yml:198` 写“All 6 gates block”，实际 `run-all.sh` 为 8 道；`ROADMAP.md:142` 写 RAG hit≥0.7，`ai/eval/eval_test.go:409` 断言 ≥0.8；`docs/product/README.md:16` 声明本目录不维护并行总控计划，但本轮需新增 subordinate 四周计划；workflow 跨域签字栏 9 项全空却被多处描述为“全绿/可用”。
- **影响**：自动化结论与文档结论冲突，读者无法判断哪个是事实源。
- **修复**：统一为“测试代码 > CHANGELOG > ROADMAP > 其他文档”的证据优先级；修正 6 gates → 8 gates；RAG 阈值改 0.8；四周计划明确 subordinate 定位；保持 workflow = Preview，不代签。
- **工作量**：S。

---

## 2. T1：影响私有化交付与升级承诺

### TD-04 基线字段级三方合并未实现

- **证据**：`itsm-backend/docs/initialization-data-classification.md:79-81` 明确 `managed_fields` / `stable_key_aliases` / `local_modified_at` 列存在但停留 DDL 默认值，字段级 reconciliation 与三方 merge **未实现**；`docs/engineering/system-baseline-design.md:161,206-212` 将 6 个 schema 标为“第三阶段，待确认后实施”。
- **已实现部分**（不要否定）：`internal/initialization/engine.go:20-34,141-223` 版本 + lease/fencing；`sql_store.go:43-115,365-400` managed-record ledger 与安装同事务；`default_credentials_guard.go` 生产弱口令致命；seeder 保留既有 admin 凭据与客户编辑。
- **影响**：记录级登记可防重复创建，但无法防升级覆盖客户对同一记录的字段级修改。
- **风险**：承诺“跨版本无损升级”时无法兑现。
- **修复**：先决策是否进入合同承诺；若是，按设计文档第三阶段实现字段级三方合并 + 6 schema + 迁移计划（**生产迁移需人工审批**）。
- **工作量**：L。

### TD-05 RLS 仅两表试点

- **证据**：`tests/multi_tenant/rls_matrix_test.go:1-25` 明示 PostgreSQL RLS pilot-only，覆盖 `changes`/`vectors`。
- **影响**：数据库层纵深防御不足，隔离完全依赖应用层谓词（约 119 个仓库文件）。
- **风险**：复杂查询/新接入点漏写 tenant 谓词时无第二道防线；等保类要求可能无法满足。
- **修复**：按风险分级扩表；保持应用层 predicate 不动；补充 raw SQL / 缓存 key / 异步消费者覆盖。
- **工作量**：L。**生产数据库变更需人工审批**。

### TD-06 审计写独立事务、失败不阻断

- **证据**：`middleware/audit.go:340-350` 审计写失败仅计数 + fallback log，不阻断业务；`:301-315` 用 detached context。
- **影响**：高风险操作可能在审计缺失的情况下成功，合规叙事有缺口。
- **风险**：审计断链在合规审查中被判不通过。
- **修复**：对高风险动作（权限变更、生产发布、AI 工具写、删除）改为同事务写审计；低风险保留异步并增加可观测失败计数与告警。
- **工作量**：M。

### TD-07 K8s 仅脚手架

- **证据**：`deploy/k8s/README.md:1-10` 自述“起步脚手架”；`:32-45` 未覆盖 AI sidecar、MinIO、监控、HPA/PDB、多副本 Postgres/Redis；镜像 tag 1.6.9、占位 secret；无 ITSM Helm/Kustomize。
- **影响**：无法承诺私有化 HA 与商业 SLA。
- **风险**：客户按文档上生产后单点故障。
- **修复**：明确“参考部署，非 HA 方案”的文档定位；若要商业承诺，另立 Helm/Kustomize + PDB/HPA + 有状态服务方案批次。
- **工作量**：L。

---

## 3. T2：能力与架构债

### TD-08 通用 Agent Runtime 与声明式 ActionFlow 缺失

- **证据**：生产 AI 能力是固定 12 技能（`builtin_skills.go:25-38`）+ `maxToolRounds=5`（`rag_service.go:1170-1293`）；全仓 `handlers/ai` 无 `ActionFlow` / `ContextEngineer` 符号；`itsm-agent/` 是独立 GitHub PR 机器人，不接后端 router/worker；`itsm-skill/index.js` 60 行开发 CLI 且含硬编码路径；`itsm-backend/memory/` 只有遗留 venv。
- **影响**：用户提供的 Trigger→Context→Agent→Policy→Approval→Action→Verification→Audit 管线只有散落阶段，没有可声明、可复用的编排单元；Verification 阶段尤其弱。
- **风险**：AI 能力难以跨客户复用；每个新场景都要改代码。
- **修复**：先定义 Pipeline manifest schema 与最小 Verification 语义，再在现有 `investigation.go` + durable outbox 上收口；不要在无客户需求时先造通用运行时。
- **工作量**：L（且当前为 Hypothesis，建议延后）。

### TD-09 连接器入站业务派发缺失 + 单进程去重回退

- **证据**：`handlers/wecom/handler.go:91-94`、`handlers/dingtalk/handler.go:94-96` 只 log+ACK；`handlers/feishu/handler.go:292-316` 非 `task_event` 返回 `ignored`；`:322-326` 承认单进程 in-memory dedup 回退；manifest 声明 `CapCreateTicket` 但只有出站建单。
- **影响**：连接器投资停留在“安全入口”，不产生业务闭环。
- **风险**：宣称“IM 集成可用”会被客户现场验证打脸。
- **修复**：见 GAP-03 / REQ-003；去重必须使用持久存储，不能依赖单进程内存。
- **工作量**：M。

### TD-10 AI 评测真实模型档手工 + 指标口径漂移

- **证据**：`backend-ci.yml:300-310` 仅 4 个离线阈值；`cmd/ai-eval/main.go:28` 明确 CI 不消费 LLM 档，连接探测失败即拒绝出报告；`ROADMAP.md:142` hit≥0.7 vs `eval_test.go:409` ≥0.8。
- **影响**：无法用 CI 证明生产模型质量；对外引用 0.7 会与代码不符。
- **风险**：AI 能力验收缺少可重复证据。
- **修复**：把 LLM 档接入受控 CI（带成本上限与密钥隔离）或明确定位为“发布前人工门禁”；修正 ROADMAP 阈值。
- **工作量**：M。

### TD-11 CMDB 深度不足

- **证据**：`docs/product/servicenow-gap-analysis.md:19-89`：CIType 是标签而非类（无 `class_code`/`parent_class_id`/`class_path`）、无 IRE、生命周期状态未分、关系缺业务服务语义与置信度、数据质量规则引擎缺失；CHANGELOG 2026-11-18 记录行级权限与 business_service 仍待决策。
- **影响**：影响分析只能走技术依赖，无法跨到业务线。
- **风险**：对标 ServiceNow 的投标要求无法满足。
- **修复**：按该文档演进阶段分批；先决策是否把 IRE 作为企业版卖点（与 GAP-01 联动）。
- **工作量**：L。

### TD-12 分层收敛未完成

- **证据**：`release`/`approval`/`notification` 的业务逻辑仍在 `internal/services/`；`handlers/asset/service.go:37-46,77` 直接用 Ent 无 repository；`service_request` 无 repository 接口；`handlers/approval/routes.go` 是 handler 方法却命名为 routes。
- **影响**：`AGENTS.md` 的 `handlers/<domain>/` 目标架构尚未完全落地；新贡献者容易在两处改同一逻辑。
- **风险**：重复实现、规则漂移（当前 C.6.3 未发现重复用例，但偏离存在）。
- **修复**：按域 opportunistic 迁移（`git mv` 保历史），每次只在触达该域时做；重命名 `approval/routes.go` 需评估引用面。
- **工作量**：按域 S–M，整体 L。

---

## 4. T3：前端与体验债

### TD-13 Hook 设计基线偏移

- **证据**：`useIncidentsQuery.ts` 具备 requestId guard、errorMessage ref、H3 gating，但 `:132` 把 `message` 放入 deps（AGENTS.md Pattern C 反模式），`:101-103` 吞掉 user enrichment 错误；`useIncidentBatchOps.tsx` 符合 Pattern D/E 与 H2/H4/H6；`useTickets.ts` 有 requestId guard 但 `batchDeleteTickets:208` 无并发上限（H6）、`:4` 静态 `message` import、`:152,168` `ticketData: any`。
- **影响**：批量删除大列表可能耗尽浏览器连接池并误判 429；错误被吞导致用户无感知。
- **风险**：中（功能可用但体验与可靠性下降）。
- **修复**：先修 `useTickets.ts` 三项（并发池 ≤5、App.useApp 反馈、显式类型），再统一 `useIncidentsQuery` deps；补 requestId race 回归测试。
- **工作量**：S–M。

### TD-14 API 契约的未解析面

- **证据**：`src/lib/__tests__/api-contract.test.ts` 的 `KNOWN_UNMATCHED_FRONTEND_PATHS` 为 **0**（好消息），但 `UNRESOLVED_CALL_BASELINE` 18 条、`MATCHER_FALSE_POSITIVES` 1 条；`src/config/product-capabilities.ts:42-47` 5 个 disabled 契约（changeClassification、knowledgeAdvancedActions、notificationTemplateManagement、advancedTicketRelations、rootCauseWorkflowActions）；`incident-api.ts:439,445` 用 `API_URLS.INCIDENT(id)` 工厂而非静态常量。
- **影响**：契约扫描并非全覆盖；动态 base path 让部分调用无法被静态验证。
- **风险**：新增路由时漏改前端调用不易被发现。
- **修复**：逐条消化 18 条 baseline（优先改为静态常量）；disabled 契约逐条确认是产品决策还是遗漏；不要把 allowlist 当永久豁免。
- **工作量**：M。

### TD-15 Ant Design v6 冻结基线

- **证据**：`tools/check-antd-legacy.sh` + `tools/antd-legacy-baseline.txt`；实跑 12 类全部通过，`bodyStyle` 4/4、`Spin.tip` 7/7 为冻结存量；`rg '<Space[^>]*\bdirection\s*='` = 0。
- **影响**：仍有 11 处旧 API，但已被 ratchet 锁定，新增为零。
- **风险**：低（受控存量）。
- **修复**：**不要无差别清理**；仅在触达组件时按 `AGENTS.md` 迁移表顺手改造并缩紧基线。

### TD-16 移动端与无障碍覆盖薄弱

- **证据**：`useAccessibility.tsx`（339 行）、`useTableKeyboardNav.ts` 与测试真实存在；89 个 `.tsx` 含 aria/role；但 `useResponsive.ts` 在 `src/app`/`src/components` **零消费者**；`tests/e2e/responsive-layout.spec.ts:26,40` 仅 2 个断言；未发现 axe 类 E2E。
- **影响**：移动端主要依赖 CSS，缺乏 JS 级适配验证；无障碍无自动化回归。
- **风险**：低–中（取决于交付是否包含移动端）。
- **修复**：先确认交付范围；若包含，则补 `useResponsive` 实际消费点与 axe 门禁。

---

## 5. 明确“本轮不修”的项（避免范围失控）

| 项 | 理由 |
|:---|:---|
| 全量重生成 Ent 代码 | `AGENTS.md` 与用户指令明确禁止；历史上曾导致开发环境长时间卡住。只做精准增量生成。 |
| 一次性替换 90 个 gofumpt 文件 | 涉及并行会话 WIP；单列批次，先 `session-ledger.py enter`。 |
| 通用 Agent Runtime / Memory / ActionFlow | 无 Verified Demand，属 Hypothesis；先收敛商业边界与客户输入。 |
| ServiceNow 全量对标 / IRE | 高成本、高复杂度，需投标或客户输入触发。 |
| 全量 RLS 铺开 | 需生产 DB 变更与人工审批；先完成风险分级与设计评审。 |
| `website/index.html` 用户 WIP | 用户工作，只读。 |
| 生产部署、打 tag、创建 PR | 需显式人工审批/用户请求；本轮未执行。 |

---

## 6. 债务与差距映射

| 债务 | 对应差距 | 建议批次 |
|:---|:---|:---|
| TD-01/TD-02/TD-03 | GAP-02 / GAP-04 | 四周计划 Slice A（W1） |
| TD-09 | GAP-03 | 四周计划 Slice B（W2–W3） |
| TD-01 + 能力矩阵 | GAP-01 | 四周计划 Slice C（W4） |
| TD-04/TD-05/TD-07 | GAP-09 / GAP-10 / GAP-11 | 合同触发后另立批次 |
| TD-08/TD-10/TD-11/TD-12/TD-13… | GAP-05…GAP-08 | P1，按客户需求排序 |
