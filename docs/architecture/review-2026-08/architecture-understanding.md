# ITSM 系统架构评审与优化建议（2026-08-29）

- **评审方法**：`risk-quality-reviewer` + `system-modeler`（架构技能包），证据优先，所有结论标注文件路径。
- **范围**：当前状态（current-state）评审，不含目标态设计。
- **配套图表**：`itsm-containers.dsl`（C4 容器视图）、`backend-dual-layering.dot`（双分层图）、`risk-map.dot`（风险地图）。

## 摘要

架构骨架是健康的：**模块化单体 + PG 事务型 outbox + 租户双重防御 + 成熟 CI/安全流水线**，与 `ADR-001-modular-monolith` 一致。主要债务集中在**中间层结构**：新旧两套分层在 5 个域上重叠、42 个文件穿透分层直访数据库、少数巨型文件承载了过多逻辑。以下按优先级给出风险与改进建议。

---

## 1. 现状架构（证据）

### 1.1 部署拓扑（置信度：高）

生产环境（`docker-compose.prod.yml`）7 容器：nginx → frontend(Next.js:3000) / backend(Go:8090) + worker（同镜像，`ITSM_PROCESS_MODE=worker`）→ PostgreSQL 17(pgvector) + Redis 7；MinIO 可选。**无独立消息队列**，可靠副作用由 `operational_commands` outbox + worker 消费承担（`internal/commandbus/commandbus.go`、`docs/architecture/operational-command-outbox.md`）。生产 compose 使用 `${VAR:?}` 强制注入密钥，做法正确。

### 1.2 后端结构（置信度：高）

| 观察 | 证据 |
|---|---|
| 双分层并存：`controller/` 100 文件 vs `handlers/` 19 个领域目录（仅 7 个有完整五件套） | 目录清单 |
| 5 域重叠：incident、cmdb、knowledge、problem、sla 在两套分层均有实现 | `controller/*` 与 `handlers/*` 并存 |
| 路由实况：cmdb / knowledge / problem / sla 的新层 handler 已在 `router/router.go` 注册（如 :1062-1083）；**`handlers/incident` 无任何路由**，`app.go:765` 仅以 `_ = incident.NewEntRepository` 压制未使用导入 | `router/router.go:1062+`、`internal/bootstrap/app.go:765` |
| ticket 域仍全量在旧分层（15 个 controller，`ticket_service.go` 2754 行） | `controller/ticket*` |
| 装配集中：`internal/bootstrap/app.go` 1481 行、161 处手工 `service.New*/controller.New*` | 直接统计 |
| 巨型文件：`bpmn_process_engine.go` 3060 行、`incident_service.go` 2132 行、`router.go` 2004 行、`cmdb_controller.go` 1879 行 | `wc -l` |
| 分层穿透：29 个 controller 文件、13 个 handler 文件直接出现 Ent 客户端调用（如 `incident_controller.go`、`known_error/handler.go`） | grep `.Client/.Query/.Create` |
| 规模：1857 个 Go 文件、130 个 Ent schema、`service/` 281 文件（上帝目录） | 目录统计 |
| 强项：36 个中间件（租户、MSP、RBAC、ACL 表达式引擎、Redis 限流、令牌撤销）；254 个测试文件，集成测试连真实 PG 验证 RLS 与黄金路径（`integration/golden_path_test.go`） | 文件存在 |

### 1.3 前端结构（置信度：高）

- 40 个一级业务路由（`src/app/(main)/`），972 个 ts/tsx 文件，252 个组件。
- 统一 `httpClient`（`src/lib/api/http-client.ts`）+ 78 个领域 API client；**仍有 8 处组件/页面直接 `fetch`**（如 `components/collaboration`、`app/admin/approvals`）。
- 状态管理方向健康：仅 3 个精简 Zustand store（auth/layout/recent-visit），服务器状态主体已迁往 React Query（`@tanstack/react-query` v5 + 15+ useQuery hooks）。
- 测试资产扎实：199 个测试文件、契约测试 `src/lib/__tests__/api-contract.test.ts`、58 个 Playwright spec。
- 缺口：**类型手工同步无 codegen**（`src/types/` 35 文件）；巨型组件（`WorkflowNodeInspector.tsx` 1745 行、`BPMNDesigner.tsx` 1356 行、`IncidentDetail.tsx` 1203 行）；**未集成 i18n**（package.json 无 i18next/next-intl）。

### 1.4 辅助服务（置信度：高/中）

- `itsm-ai-service`（FastAPI：triage/RCA/risk）、`itsm-agent`（Go，含 9.8MB 预编译二进制）、`itsm-rag`（Python，独立 compose）**均未出现在任何主 compose 编排中**。
- 后端自带 `service/llm_gateway.go` + `rag_service.go`（782 行），与上述资产存在功能重叠。功能归属与调用关系属于**未知项**，需要一次所有权决策。

---

## 2. 质量属性评估

| 属性 | 评级 | 依据 |
|---|---|---|
| 安全性 / 租户隔离 | 良好 | 应用层 tenant predicate + PG RLS 纵深防御、RLS 集成测试、生产密钥强制注入、TruffleHog 扫描 |
| 可靠性（副作用） | 良好 | 事务型 outbox + worker、幂等/重试规则在 AGENTS.md 强制并有文档 |
| 可测试性 | 中等 | 单元/契约/真实 PG 集成测试俱全，但新分层 `handlers/` 测试偏薄（对比 service/ 109 个测试文件） |
| 可维护性 | 偏低 | 双分层重叠、42 文件穿透边界、巨型文件、1481 行装配函数 |
| 契约一致性 | 中等 | 有契约测试与 4 层单一事实源规范，但类型手工同步，漂移只能靠测试兜底 |
| 可扩展性 | 中等偏上 | 单体 + worker 拆分合理；连接器/技能市场脚手架存在；暂无消息队列是合理克制而非缺陷 |
| 可观测性 | 中等 | zap 结构化日志 + Prometheus 栈；未见分布式追踪证据 |

---

## 3. 风险登记册（按优先级）

| # | 风险 | 等级 | 核心证据 |
|---|---|---|---|
| R1 | 五域双入口并存，同一用例可能落在两套实现，行为/状态机分裂 | 高 | `backend-dual-layering.dot` |
| R2 | incident 域迁移半途：新层零路由（编译占位），旧层 1736 行 controller 承载全部流量，违反 `domain-ownership.md` 迁移定义 | 高 | `app.go:765` |
| R3 | 42 个 controller/handler 文件直访 Ent，事务与审计边界难以保证 | 高 | grep 证据 |
| R4 | 巨型文件与超长装配函数：理解成本、合并冲突、回归风险 | 中高 | `bpmn_process_engine.go` 3060 行等 |
| R5 | `service/` 上帝目录（281 文件，混合 BPMN 引擎/审批/命令处理器） | 中 | 目录统计 |
| R6 | 前端类型与后端 DTO 手工同步，契约漂移风险 | 中 | 无 codegen 配置 |
| R7 | 巨型前端组件（1745 行属性面板等），可维护性与回归风险 | 中 | 行数统计 |
| R8 | 浮动 AI 资产：3 个未编排服务 + 后端网关功能重叠，所有权不明 | 中 | compose 无定义 |
| R9 | 未集成 i18n，与中国市场企业级产品方向不一致 | 低中 | 依赖清单 |
| R10 | dev compose 硬编码凭据（仅开发环境，可接受但应显式标注） | 低 | `docker-compose.dev.yml` |

---

## 4. 优化建议（按优先级）

### 第一批：立即可做（约 1-2 周）

1. **了结 incident 域迁移决策**（对应 R2）。按 `domain-ownership.md` 已有的"逐端点迁移后一次切路由"策略二选一：a) 为 `handlers/incident` 接线并一次性切换路由、删除旧实现；b) 若暂不迁移，删除死代码并在 `domain-ownership.md` 标注实际状态。当前"编译占位"状态是最差选项。
2. **冻结双分层扩张**（对应 R1、R5）。用 CI 检查（如 `go-arch-lint` 或简单脚本）禁止向 4 个已切路由域（cmdb/knowledge/problem/sla）的 legacy controller 新增端点；`service/` 新增文件需评审说明。
3. **拆分装配函数**（对应 R4 局部）。将 `app.go` 按域拆为 `bootstrap_ticket.go`、`bootstrap_cmdb.go` 等，目标单文件 <300 行；不引入 DI 框架，保持手工装配的可读性。

### 第二批：本季度（对应 v1.1 加固）

4. **分层边界 lint**（对应 R3）。配置 `depguard`/`go-arch-lint`：`controller/` 与 `handlers/*/` 包禁止导入 `ent` 客户端包（测试文件豁免），存量违规建白名单并逐域清账。这是把 AGENTS.md 已有规则变成机器强制的最直接一步。
5. **拆分巨型业务文件**（对应 R4）。优先 `bpmn_process_engine.go`（按解析/执行/任务/历史拆）与 `ticket_service.go`（按用例拆）；每次拆分配套既有测试回归。
6. **契约生成或强化**（对应 R6）。选项 a：后端导出 OpenAPI → 生成前端类型（orval 等）；选项 b（更轻）：把 `api-contract.test.ts` 从 URL 校验升级为 DTO schema 级断言，并在后端加 DTO 快照测试。
7. **补齐新分层测试**（对应可测试性）。对已切路由的 4 域按"每域至少一条真实 Router 入口的跨租户拒绝测试"补齐，与 AGENTS.md 强制条款对齐。

### 第三批：半年视角

8. **AI 资产所有权 ADR**（对应 R8）。建议收敛路径：LLM 能力统一走后端 `llm_gateway`，RAG 走 `itsm-rag`（编排进 compose）或并入后端 `rag_service`；`itsm-ai-service`/`itsm-agent` 若不在路线图则归档。产出 `ADR-003`。
9. **i18n 引入**（对应 R9）。选 `next-intl`，先覆盖导航、菜单、核心列表页文案；与 `i18n-guide` 技能规范对齐。
10. **巨型前端组件拆分**（对应 R7）。`WorkflowNodeInspector` 按节点类型拆子面板；`TicketDetail`/`IncidentDetail` 按标签页拆。
11. **分布式追踪**（可观测性）。OpenTelemetry 贯穿 Gin → service → worker，与现有 Prometheus 栈整合；请求链路携带 tenant/actor 属性（注意低基数）。

### 不建议做

- **不要**现在引入消息队列：PG outbox + worker 在当前规模下是自洽的，预置 MQ 只会增加运维面。
- **不要**为拆微服务而拆：模块化单体 + 领域切分符合 ADR-001，先把领域边界在单体内做干净。

---

## 5. 未知项与待验证

| 未知项 | 验证方式 |
|---|---|
| cmdb/knowledge/problem/sla 的 legacy controller 仍承载哪些端点 | 逐路由对比 `router.go` 与 controller `RegisterRoutes` |
| ticket 域 React Query 迁移实际覆盖率 | 抽样 `app/(main)/tickets` 页面数据流 |
| `itsm-ai-service` 是否仍被后端调用 | 全局搜索后端对其端口/路径的引用 |
| 13 个直访 DB 的 handler 是否构成真实租户风险 | 逐个审查事务与租户谓词 |

## 6. 交付物

| 文件 | 回答的问题 |
|---|---|
| `architecture-understanding.md` | 本报告：现状、质量评估、风险、建议 |
| `itsm-containers.dsl` | 系统由哪些可部署单元组成、如何连接（Qoder Canvas 可预览） |
| `backend-dual-layering.dot` | 新旧分层在哪些域重叠、路由实际挂在哪 |
| `risk-map.dot` | 结构性原因如何导致质量影响 |

*维护说明：本报告基于 2026-08-29 main 分支快照（含工作区未提交改动 `app.go`、`sequence_service.go` 的存在性观察）；迁移进展变化后应更新第 3 节风险状态。*
