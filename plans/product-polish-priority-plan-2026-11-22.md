# 产品打磨优先级任务清单（2026-11-22）

> 目标：把前端 UI 细节与后端功能实践的问题排成可执行队列，明确每项由 Codex 委派还是本会话自做，尽快把产品打磨到位。
>
> 证据基线：`HEAD = b3bfd7486` + 68 项并行 WIP（只读核验，本清单不包含任何已由 WIP 修复的内容）。
> 来源：`output/backend-module-audit-2026-11-21.md`、`output/api-halfbaked-audit-2026-11-20.md`、`output/repo-organization-audit-2026-11-21.md` 的逐项复核 + 前端交互模式扫描（见 §2）。

## 分工原则

| 归属 | 判据 |
|:--|:--|
| **Codex 委派** | 改动面大但模块独立、可独立验证（测试先行可红可绿）、不触碰权限判定链/审计字段/共树提交协议；单次 prompt 打包全部判据与验收标准 |
| **本会话自做** | 触碰 authz/审计/迁移/菜单基线等高风险判据；需要并行协议记账与门禁归因的提交；跨前后端契约联动；需要人工核 diff 的语义决策 |
| **暂不动** | 需要产品/数据模型决策（如 engineer_skills 写入契约）、或需真实环境验收（k6、真实租户联调） |

---

## 1. P0 — 直接打生产 / 门禁已红

| # | 问题 | 证据 | 动作 | 归属 |
|:--|:--|:--|:--|:--|
| P0-1 | **`itsm-worker` 缺 `AI_SERVICE_TOKEN`（四处全缺）**：`docker-compose.prod.yml:370` 环境段无该变量（backend/ai-service 均 `${AI_SERVICE_TOKEN:?...}`）；`deploy/docker-compose.saas.yml`、`saas-host.yml` 同缺；k8s `01-configmap/02-secrets/21-worker.yaml` 出现 0 次。worker 还缺 `GUIDANCE_URL` → 回落 `localhost:8091`（`internal/bootstrap/app.go:751-757`），容器内不可达 → sidecar fail-closed 401，worker 侧 AI 分诊全挂 | 三份 compose + k8s 四处补 `AI_SERVICE_TOKEN` + `GUIDANCE_URL`，与 backend 段同一 `${:?}` 语义；k8s 补 envFrom | **自做**（部署面 + 敏感变量，须核 diff 防泄漏） |
| P0-2 | **swagger 漂移已进 HEAD**：436 个 `@Router` 注解 vs tracked `docs/swagger.json` 199 paths；**158 个注解 op 不在 swagger.json**（128 个来自已提交文件：knowledge 20、sla 19、ticket 34、sso 11…）。WIP 还在批量补注解（tenant +118 行、rbac +260 行）→ `api-contract-check` freshness 作业按设计为红 | 收口时机：等 tenant/rbac WIP 定稿后跑 `make swagger-gen` 一次回流 + 提交 | **自做**（与 WIP 强耦合，时机敏感） |
| P0-3 | **`scripts/parallel/` 已 staged 未入 HEAD**：`git ls-tree HEAD` 为空，协议脚本随时可能随索引重置消失 | 单独 commit 入库（显式 `git add scripts/parallel/`），与相关会话对账后执行 | **自做**（并行协议本体） |

## 2. P1 — 功能语义错误 / 用户可见

| # | 问题 | 证据 | 动作 | 归属 |
|:--|:--|:--|:--|:--|
| P1-1 | **事件转问题不绑 SLA**：`root_cause_analysis_service.go:222-302` 事务内仅 `Problem.Create` + `IncidentEvent`，零 SLA 调用（该 service 无 slaSvc 依赖）；正常建问题走 `handlers/problem/service.go:112-136 bindSLAOnCreate` → 转换后 `/problems/:id/sla` 必然 `bound=false` | 在转换事务内接入 `bindSLAOnCreate` 同款逻辑；补「转换后 SLA 已绑定」回归 | **Codex**（跨服务调用边界清晰、测试可独立红绿；prompt 附 bindSLAOnCreate 位置与事务边界判据） |
| P1-2 | **智能分派门禁是空的**：`ticket_assignment_service.go:396-411` 两门禁恒 true；`:322` 技能硬编码 `["general"]` → 40% 权重技能分失真；`engineer_skills` 无写入方、无 API、DTO 零消费者；auto-assign 路由线上可达（`router/ticket_routes.go:275-276`） | 两步：① 先出数据模型设计（engineer_skills 写入契约 + 配置入口）② 再启用门禁。**本轮只做 ①设计 + 在 UI/文档标注「技能分暂不可信」** | **自做设计**；实现待设计评审 |
| P1-3 | **死 handler + 前后端契约漂移（原报告口径需修正）**：`handlers/incident/handler.go:1762-1803 GetIncidentMetrics`（含硬编码 99.9、租户级 sla_violations）**零路由注册**；生产路由 `router/incident_routes.go:41` 绑 `GetMetrics`（返回指标数组）。前端 `incident-api.ts:270-277` 快照类型与 `:905-910 getIncidentMetricsData` 同样零消费 | 删死 handler `GetIncidentMetrics` + 前端零消费的快照类型/函数；或反向接线（产品决策）。**先删死代码是安全方向** | **Codex**（删代码 +契约测试，判据明确：两处零引用需先 grep 复核） |
| P1-4 | **ticket_rating 非法日期静默忽略**：`handlers/ticket_rating/service.go:216-229` 无 else 分支；`dto/ticket_rating_dto.go:46-52` 无 binding 标签 → 报表可被放大到全历史 | DTO 加 binding 校验 → 400；service 补错误分支 | **Codex**（小而独立，带回归测试） |
| P1-5 | **public 组写操作缺审计**：`POST /auth/register`、`ForgotPassword`、`ResetPassword`、`refresh-token` 及三个 webhook 无 `AuditLog.Create`（`middleware/audit.go` 的 shouldAudit 只覆盖已挂载组，public 组绕过） | 逐端点补审计（复用 `RecordLoginAudit` 模式） | **Codex**（模式统一；prompt 给 `handlers/common/service.go:73-149` 审计模板） |

## 3. P2 — 工程质量 / 治理

| # | 问题 | 证据 | 动作 | 归属 |
|:--|:--|:--|:--|:--|
| P2-1 | **分页三类缺口**：① BPMN 引擎列表 `dto/bpmn_engine_dto.go:63,93` PageSize 无 binding → `Limit(可>1000)`，信封回显却夹到 100（自相矛盾）；② user 列表上限 1000 vs 平台 100（`dto/user_dto.go:67` 绕过 `common/pagination.go` 夹紧）；③ `handlers/common/handler.go:235/649/660` users/tags/audit-logs 回退路径完全无分页 | 三处统一到 `common.ValidatePagination` / 平台夹紧 | **Codex**（三处独立、判据=对齐 `common/pagination.go` 既有语义，各配回归） |
| P2-2 | **死代码退役**：`KnowledgeIntegrationService`（零调用方，仍进棘轮基线）、`dashboard_service.go:1118/1149` 下划线 SLA helper（零外部调用） | 删代码 + 更新 `scripts/docs-gate/service-prod-baseline.txt` 棘轮基线 | **Codex**（删前先 grep 复核零引用；棘轮基线同步是机械操作） |
| P2-3 | **后端根松散文件归位**：`migrate_fresh.go`、`seed_demo.go`、`swagger_meta.go`、`migrate_ticket_workflow.sql` 仍 tracked；129MB `scenarios.test` 在磁盘未跟踪（`.gitignore:23 *.test` 误吞，本应入库或显式忽略） | 迁入 `cmd/`/`scripts/` + 单独 commit + 全量编译验证；`scenarios.test` 定去留 | **自做**（涉及构建入口与 gitignore，需全量编译验证） |
| P2-4 | **skill seed 并发残余**：`handlers/skill/persistence.go:104-160` SELECT-then-INSERT 非原子、无 ON CONFLICT；seed 失败只 log 不阻断（`app.go:1137-1139`）→ 多副本首启仍可能 23505 | 加 ON CONFLICT（ent 未开 sql/upsert，需 `OnConflict` API 或捕获 23505 重查） | **Codex**（单点改动 + 并发回归测试；prompt 附 `connector/alert/repository.go:62` 的既有约束注记） |
| P2-5 | **`internal/layering` 计数红恶化**：基线 `shared_services_go_files=340` vs 实测 **348**（4 个由 WIP 测试推高）；既有红 332/340 不代调，但 348>340 的上界红是新形态 | 等 WIP 定稿后统一基线回写（与 `entgen-hardening` 会话对账） | **自做**（计数型门禁 + 并行对账） |
| P2-6 | **19 项孤儿 WIP 归属对账**（repo-org 报告 §3.2）：BPMN/看板批、RBAC/租户、`analytics-service.ts` 等无会话认领 | 只读清单逐项问对应会话；各会话自行认领补 `--claim` | **自做**（协议治理） |

## 4. 前端 UI 细节（交互缺陷模式扫描，31 项候选，全部为已提交存量、不在 WIP 中）

> 方法论：`itsm-frontend-interaction` 技能 10 类反模式；机检：`type-check` **exit 0 全绿**、`lint:antd` **exit 1 超基线 3 类 9 处**（基线只许降不许升 → 当前为回归）。

### 4.1 P0 — 高置信 + 高流量影响（本会话自做，逐条验证可复现再修）

| # | 缺陷 | 证据 | 归属理由 |
|:--|:--|:--|:--|
| F-P0-1 | **共享模板搜索每键一发请求**：`useDebounce` 算了没用（eslint 报 unused），`onChange` 直调 `onSearch`；Enter 与 onChange 叠加可能双发。波及 incidents/changes/problems/assets/workbench/ChangeList/ProblemList **6+ 页面** | `BusinessPageTemplate.tsx:245-263` | 共享外壳、影响面最大；修法涉及防抖+双发两层，须核每个消费方 → 自做 |
| F-P0-2 | **知识 AI 搜索 Enter 双发**：`onSearch`+`onPressEnter` 同绑，antd v6 源码证实按一次 Enter 发两次 POST `/ai/rag/search`；`enterButton` cloneElement 分支不注入 loading，按钮期间可连点 | `knowledge/page.tsx:165,369,375` | 已核 antd 源码的确定性缺陷；涉及 AI 接口双倍消耗 → 自做 |
| F-P0-3 | **CMDB 删除静默吞错**：`catch(() => {})`，用户无任何失败反馈；根因 `useDeleteCIMutation` 无 onError | `CIList.tsx:133` + `useCMDB.ts:313` | 单点、判据清晰（对齐同文件批量删除 allSettled 范式）→ **Codex** |
| F-P0-4 | **工单列表无错误态**：最高流量页加载失败只 toast，表格显示「暂无数据」——失败伪装空态，违反判定红线 | `TicketList.tsx:71-81` + `useTickets.ts:56,109` | 高流量页但修法模板化（loadError 分流）→ **Codex** |
| F-P0-5 | **事件详情首帧闪「未找到」**：loading 初值 false → 首帧渲染 404 分支；`ProblemDetail` 已修同款可对照 | `IncidentDetail.tsx:95,506-514` | 一行初值修复 + 回归 → **Codex** |

### 4.2 P1 — 数据正确性 / 失败伪装空态族

| # | 缺陷 | 证据 | 归属 |
|:--|:--|:--|:--|
| F-P1-1 | changes 四路 fetch（列表/看板/统计/日历）全无 seq 守卫——`problems/page.tsx` 已有 `listSeqRef` 正确范式未对齐 | `changes/page.tsx:249,316` 等 | **Codex**（照抄 problems 范式） |
| F-P1-2 | 详情页深链竞态族：IncidentDetail / ProblemDetail / ChangeDetail 的 loadData 无 seq、id 切换不清 per-id 数据 | 三组件 useEffect | **Codex**（generation ref 套路统一，SKILL 有标准代码） |
| F-P1-3 | 失败伪装空态族：marketplace catch→置空数组、admin/users、approvals loadLegacy 单源 `.catch(()=>({items:[]}))`、tickets 统计卡 catch→显示 0 | 4 处 | **Codex**（统一 loadError 分流模板） |
| F-P1-4 | workbench 错误态无重试：传了 `error` 没传 `onRetry` | `workbench/page.tsx:290` | **Codex**（传参补全） |
| F-P1-5 | tenants 搜索无防抖无乱序守卫（同仓 teams/roles 已有 300ms 防抖范式） | `admin/tenants/page.tsx:109,125-159` | **Codex** |
| F-P1-6 | TicketList / useTickets 缺 seq 乱序守卫（防抖已有，仅竞态段） | `useTickets.ts` | **Codex** |

### 4.3 P2 — 对齐与规范

| # | 缺陷 | 证据 | 归属 |
|:--|:--|:--|:--|
| F-P2-1 | **lint:antd 基线回归 9 处**：`Alert.message`×6、`Space.direction`×2、`valueStyle`×1（基线 0，只许降不许升 → CI 语义上已是红） | 9 处文件均不在 WIP | **Codex**（机械替换 + lint:antd 验收） |
| F-P2-2 | 防重入对齐 inFlightRef：approvals 三提交路径、四个 create 页仅靠 useState loading | 对照 `service-catalog/approvals` 范式 | **Codex** |
| F-P2-3 | i18n 同屏混用 4 处：BusinessPageTemplate 硬编码与 `t()` 同屏、knowledge 标签、CIList、IncidentDetail 错误文案 | 4 处 | **Codex**（复用既有 i18n 键，禁新增硬编码） |
| F-P2-4 | 死代码/传参被忽略：incidents 不可达空态分支、`TicketListProps.pageSize` 声明未用 | 2 处 | **Codex** |
| F-P2-5 | **结构性测试覆盖缺口**：incidents 0/4、problems 0/6、knowledge 0/6、changes 主列表、cmdb/cis、**tickets 主列表** 零测试锁定——交互缺陷高发区几乎无回归锁 | 目录统计 | 分批：**Codex** 给已修缺陷补回归（每修必锁）；页面级全量补测暂缓 |

### 4.4 P3 — 低置信待人工确认（暂列不动）

- ProblemDetail 深链竞态可见症状、`useIncidentsQuery` loading 初值 false（需确认驱动页面是否渲染空态）、BusinessPageTemplate 刷新整页 reload（产品口径问题）、pageSize 被忽略（当前无症状）

## 5. 暂不动（需产品/数据模型决策或真实环境）

- `engineer_skills` 写入契约与配置入口（P1-2 前置）— 设计先行
- PDF 导出重新开放 — 需支持中文的渲染器（后端审计 §四.4）
- 真实租户跨域联调验收签字（`docs/acceptance/workflow-cross-domain-signoff.md`）
- LLM 档 AI Evaluator 真实质量验收（`cmd/ai-eval -mode llm`）
- 连接器三渠道真实联调（能力矩阵维持预览）
- i18n 全量英文化（v1.7 项，~83% 页面硬编码，非本轮）

## 6. 执行顺序建议（最终版）

**第 0 步 · 立即（自做，生产直击）**
1. P0-1 `itsm-worker` 补 `AI_SERVICE_TOKEN` + `GUIDANCE_URL`（三 compose + k8s，核 diff 防泄漏）
2. P0-3 `scripts/parallel/` 入库

**第 1 步 · Codex 批次 A（前端 P0，单次 prompt 打包 3 项小修）**
F-P0-3（CMDB 吞错）+ F-P0-4（工单错误态）+ F-P0-5（事件首帧闪现）——每项配回归测试，验收 `type-check` + 定向 jest + eslint

**第 2 步 · Codex 批次 B（前端 P1，两批）**
- B1：F-P1-1/2（seq 守卫族，照抄 problems/ChangeDetail 范式）
- B2：F-P1-3/4/5/6（失败伪装空态族 + tenants/useTickets）

**第 3 步 · 自做（前端 P0 两个高风险项）**
F-P0-1 共享模板防抖（波及 6+ 页，逐消费方核）+ F-P0-2 AI 搜索双发（已核 antd 源码）

**第 4 步 · Codex 批次 C（后端 P1 小件打包）**
P1-4 ticket_rating 日期 400 + P1-5 public 组审计 + P2-4 skill seed ON CONFLICT

**第 5 步 · Codex 批次 D（后端 P1 大件 + P2）**
- D1：P1-1 事件转问题绑 SLA（事务边界判据写进 prompt）
- D2：P1-3 死 handler + 前端零消费类型清理（先 grep 复核零引用）
- D3：P2-1 分页三缺口 + P2-2 死代码退役（棘轮基线同步）

**第 6 步 · Codex 批次 E（前端 P2 规范件打包）**
F-P2-1 antd 废弃 9 处 + F-P2-2 inFlightRef 对齐 + F-P2-3 i18n 混用 + F-P2-4 死代码

**时机敏感（自做）**
- P0-2 swagger 回流（等 tenant/rbac WIP 定稿 → `make swagger-gen` 一次回流）
- P2-5 layering 计数基线回写、P2-6 孤儿 WIP 对账（与并行会话协同）

**设计先行（自做）**
- P1-2 智能分派数据模型设计（engineer_skills 写入契约）→ 评审后再排实现

**每批 Codex 产出的固定验收**（本会话执行，不信任自报）：`git diff` 核范围 → 敏感信息扫描 → `go test ./<pkg>/` / 定向 jest → `gofmt`/eslint → 门禁归因（`gate-at-head.sh` 区分既有红）→ 精确暂存提交（并行协议 note）。

## 7. Codex 委派模板（省额度）

单次 `codex exec` prompt 必含：① 任务边界与文件范围 ② 项目判据（TDD 先红、`handlers/<domain>/` 分层、统一响应 `common.Fail`、并行协议禁 `git add -A`）③ 验收命令（`go test ./<pkg>/`、`gofmt`、`git diff --check`）④ 「只改列出的文件，不碰其他 WIP」。产出由本会话跑门禁 + 核 diff 后提交。
