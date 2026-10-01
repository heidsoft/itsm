# 产品功能完整性梳理与改进计划（2026-10-01）

> 口径：**以实测为准，不信文档与门禁结论**。本文每条判断都给出 `file:line` 或可复现命令。
> 结论来源：全量后端测试实测 + 4 个领域并行代码梳理（ITIL 核心域 / CMDB·知识·AI / 平台运维域 / 前端），
> 梳理阶段的子代理结论中有 5 条经复核**推翻或改判**，已在 §5 逐条记录（含一条引用了不存在文件的结论），避免后续按错误前提排期。

## 0. 与既有计划的关系（先读这段）

本仓已有两条在跑的规划主线，本文**不另起第三条**：

| 文档 | 定位 | 与本文关系 |
|:--|:--|:--|
| `output/dev-improvement-plan-2026-09-27.md` | 架构主线 Phase 1-4（编排统一 BPMN、Mixin 抽取、SLA 计算面收敛、清理+门禁），7.5 周，PM+SA 已评审 | **仍是架构轨道的唯一事实来源。** 本文批次 4 不重述其内容，只做现状核对；其 §6「明确不做的事项」对本文具有约束力 |
| `plans/scope-convergence-plan-2026-09-28.md` | 收敛执行计划（F8/B2 等批次，含 §8 待用户决策） | 本文批次 0 是其"门禁恢复"的延续 |

因此本文的**净新增内容**只有三块：
1. §1 实测基线与 6 个失败簇的根因定位（此前未有一手实测记录）；
2. §2 的**能力级待完善清单**（占位/未接线/假状态/契约漂移），这是既有两份计划都没逐项盘过的；
3. §4 的三个新决策点（编号 N1-N3，避开既有 D1-D8）。

已受既有裁定约束、本文不再主张的事项：读接口权限补全（§6 判为误报）、RAG 双底座合并、各域独立编号、`requester_id`/`created_by` 合并、ServiceRequest 强接 WorkItemCore、单表 Task 基类。

## 1. 实测基线

```bash
cd itsm-backend && go build ./...                       # 干净
cd itsm-backend && go test ./...                        # 76 ok / 9 FAIL，16 个失败测试函数
cd itsm-frontend && npm run type-check                  # 绿
cd itsm-frontend && npx jest src/lib/__tests__/api-contract.test.ts   # 1 failed / 2 passed
```

失败聚类（全部已定位到根因）：

| 簇 | 失败点 | 性质 |
|:--|:--|:--|
| C1 枚举类型化断言漂移 | handlers/change 3、integration 1、service/release 5 | **测试漂移**，生产值正确 |
| C2 `submitted` 死词表 | service/change 2、scenario3 2 | **生产缺陷**（legacy 写入非法枚举） |
| C3 BPMN 审批任务 4090 | scenario10 全部子用例、handlers/change 1 | **契约决策未定** |
| C4 `slaStore` 未注入 panic | service/sla 1、scenario5 1 | **生产缺陷**（缺 fail-closed） |
| C5 sqlite `table is locked` | scenario2 1 | **测试隔离** |
| C6 authz 生成物过期 | middleware 1 | **生成物漂移**，本次已重新生成 |

## 2. 待完善问题清单

### P0 门禁红灯 / 生产功能实际不可用

1. **变更「提交审批」在 legacy 路径上写非法枚举** — `ent/schema/change.go:38-50` 枚举含 `pending` 不含 `submitted`，但 `service/change_service.go:309-313` `persistedChangeStatus()` 在写入前把 `pending` 反映射成 `submitted`，`:723` 真实调用。`service.NewChangeService` 零生产调用方，但 `tests/scenarios/scenario3:98,150`、`scenario6:105` 仍走它 → 审批链+回滚业务流回归整段红。
2. **SLA 违规扫描 nil  panic** — `internal/bootstrap/app.go:394` 事后 `SetSLAStore()` 注入；未注入时 `service/sla/store.go:255` 直接解引用 nil `*Store`。4 个服务（sla_monitor/ticket/dashboard/ticket_sla）同模式。
3. **RBAC 预检生成物过期** — `middleware/rbac_precheck_gen.go` 与路由声明不同步（`projects` 已删、`admin/skills`、`menus/export`、`workbench` 后加）。`TestWriteRoutesRequirePermission` 仍绿（写路由都有声明），所以风险是**读侧端点校验映射过期**，不是普遍越权。
4. **~~`GET /api/v1/workbench` 无端点权限~~ → 已撤销，非缺陷.** `router/workbench_routes.go:13` 确实无 `RequirePermission`，但 `handlers/workbench/handler.go:24-33` 从**认证上下文**取 `tenantID` 且缺失即 fail-closed，并挂在 Auth+RBAC+Tenant 组内。这与 `output/dev-improvement-plan-2026-09-27.md` §6 的三方裁定一致：*"common_system 的 AuthMiddleware 路由是身份自读的正确设计，读接口权限补全属误报"*。残留的唯一开放问题是"租户内跨角色可见范围"，属产品口径（见 §4 N3），不是安全缺口。
5. **Delivery 状态无读接口** — `NotificationDelivery` 只写不读（全仓无 `NotificationDelivery.Query`）。outbox 的 `last_safe_error` 只进 zap（`service/notification_delivery_command_handler.go:129-166`）。运维无法回答"通知到底发出去没有"。
6. **Skill 注册表纯内存** — `service/skill_registry.go:53-59` 是 Go map，`handlers/skill/handler.go:161-442` 的注册/提升/删除**重启即丢**，且无 DB 持久层。

### P1 契约漂移与假状态

7. **知识检索响应 snake_case** — `handlers/knowledge/handler.go:519-529` 返回 `map[string]interface{}`，键为 `is_published`、`relevance_score`、`created_at`、`updated_at`，违反 camelCase 单一契约，且未走 DTO/Mapper。
8. **AI 审计可被客户端伪造** — `handlers/ai/service.go:540,557` 以 `model=""/confidence=0` 落库；`POST /ai/audit`（`handlers/ai/handler.go:533-591`）接受客户端自报 model/confidence/accepted。
9. **邮件 intake 先提交后 enqueue** — `handlers/email_intake/orchestrator.go:173-186` commit，`:364` 才入队，崩溃窗口丢命令；`service/ticket_service.go:171` 同模式。
10. **前端契约测试失败 2 处** — `menu-api.ts:95` 拼接 URL、`msp-api.ts:134` 指向不存在的后端路由（实测 `npx jest api-contract.test.ts` = 1 failed / 2 passed）。
11. **`/reports` 无视自身能力开关** — `app/(main)/reports/page.tsx:3,13` 渲染 `AdvancedReporting`，而 `product-capabilities.ts:55` 标 `advancedReporting:false` 且 reports 写侧在 allowlist 里。
12. **8 个 service 包测试文件未传 ctx** — `change/incident/problem/ci_relationship/release/root_cause/recommendation_service_test.go` 调 `context.Background()`，与 `.Scan(ctx)` 修复处（`problem_service.go:128`）同类。

### P2 占位与未接线（假成功风险最高的一组）

| # | 能力 | 状态 | 证据 |
|:--|:--|:--|:--|
| 13 | 工单导出 | 注册路由返回 not implemented | `handlers/ticket/service.go:308-310` |
| 14 | 工单导入 | 占位且**无路由**（unwired） | `handlers/ticket/service.go:313-315` |
| 15 | 问题评论 / 知识评论 | 注册路由恒 5003 | `handlers/problem/handler.go:926,964`；`handlers/knowledge/handler.go:415-421` |
| 16 | `GET /tickets/types` | 硬编码 6 项假清单，遮蔽真实 `/ticket-types` CRUD | `router/ticket_routes.go:52-61` |
| 17 | CMDB 云发现 | job 端点恒 503，`CommandRunCMDBCloudDiscovery` 注册但**从未 enqueue**，Azure provider 仅打日志 | `handlers/cmdb/handler.go:919-928`；`service/cloud_discovery_service.go:333-339` |
| 18 | 服务请求交付命令 | `CommandDeliverServiceRequest` 既未注册也未入队 | `internal/commandbus/commandbus.go:37` |
| 19 | 知识文章版本 | 快照逻辑只在 legacy `service/knowledge_service.go:377-464`，生产更新路径不写版本 | `handlers/knowledge/repository_impl.go:138` |
| 20 | 工单自动分配/升级/SLA 告警 | 只 log+计数（自认 "Simplification"） | `service/ticket_automation_service.go:103-118` |
| 21 | 事件自动化规则 | 同上，条件为硬编码子集 | `handlers/incident/service.go:447-477` |
| 22 | 知识浏览量/评分/推荐 | 未实现或以代理值冒充 | `service/knowledge_integration_service.go:488`；`handlers/knowledge/service.go:283-288`；`handler.go:559` |
| 23 | 前端 skills 管理 UI | 后端有 `/api/v1/admin/skills`，前端无界面 | — |
| 24 | `/admin/tags` 菜单 | 菜单已种，页面不存在 → 必然 404 | `pkg/menubaseline/baseline.go` |
| 25 | 对账修复动作 | 只有 diff 查询，无 bind/resolve 端点 | `handlers/cmdb/service.go:476-527` |

### P3 架构双轨（本次梳理新增的实测发现）

26. **`service/problem_service.go`（695 行）100% 死代码** — `NewProblemService` 零非测试调用方，活逻辑在 `handlers/problem/service.go`。**这是 change 域 legacy 双轨的第二例，且比 change 更彻底。**
27. **change 域状态机词表未收敛** — `IsValidChangeStatusTransition` 三张转换表 key 是 `submitted`，靠 `:844-848` 把 `pending` 归一化才能与 `handlers/change` 协作。
28. **`service/change_service.go` 的 `apiChangeStatus` 读映射不是死代码** — 枚举只在**写入**校验（`ent/change/change.go:201` 注释确认 `StatusValidator` 仅 builder save 前调用），存量库若有 `submitted` 行，删读映射会让它以 `submitted` 出现在 API 上。
29. 其余沿用收敛计划已有条目：`ticket_workflow` 在 handler 里跑裸 SQL（`handlers/ticket_workflow/routes.go:353-416`，实为 middleware-free 而非无守卫）、`dashboard_service.go:276` 直查 Ent、`handlers/approval/handler.go:16` 仍挂 legacy `service.ApprovalService`。

## 3. 开发改进计划

原则：**先让门禁真绿，再消假状态，最后收双轨**。每批独立可验证、独立可提交。

### 批次 0：恢复门禁可信（1 天）— 2026-10-01 执行，0.1/0.2/0.3/0.4/0.5 完成

| 步 | 内容 | 验收 | 结果 |
|:--|:--|:--|:--|
| 0.1 | C1 枚举断言漂移 9 处改断言类型（不改生产） | `go test ./handlers/change/ ./integration/ ./service/ -run 'Change\|Release'` 绿 | ✅ 9 处改为 `entrelease.StatusDraft`/`entchange.StatusPending` 等生成常量；因测试内已有同名局部变量（`release.ID`、`change.ID`），导入统一加 `entrelease`/`entchange` 别名 |
| 0.2 | 提交已重新生成的 `rbac_precheck_gen.go`，并把 `cmd/authz-gen` 挂进 pre-push | `go test ./middleware/ -run TestPrecheckMapIsFresh` 绿 | ✅ 绿（删 `projects*`，补 `admin/skills`、`admin/skills/:code`、`menus/export`） |
| 0.3 | `sla.Store` 构造期校验 client 非 nil，4 个服务的 `SetSLAStore` 改构造器必填或 nil 时 fail-closed 返回错误 | scenario5 与 `TestSLAMonitorService_CheckSLAViolations_Empty` 绿 | ✅ 采用 fail-closed 分支：`SLAMonitorService.requireSLAStore()` 在 `CheckSLAViolations`/`GetDashboardMetrics` 入口返回 5003，不 panic 也不伪装空结果；含「撤守卫即 panic」的回归测试。**未**改成构造器必填——`SetSLAStore` 是既有装配契约，改动面波及 4 个服务与 bootstrap，留给批次 3 双轨收敛 |
| 0.4 | scenario2 sqlite 串行化（`testDSN()` 加共享 cache / 单连接） | scenario 包无 `table is locked` | ⚠️ **计划的根因判断有误**。实测不是 DSN 同名争抢：`scenarioDSN()` 已用原子计数器分配互不相同的库名，且场景内无 `t.Parallel()`；`_busy_timeout` 加了也无效，因为报的是 `SQLITE_LOCKED`（共享缓存表级锁），不是驱动会重试的 `SQLITE_BUSY`。真因是 `service/incident_service.go:396,412` 在未开 outbox 时走 fire-and-forget goroutine，用 `context.Background()` 继续写同一块共享内存库，与下一步事务争抢 `incidents` 表。修法：scenario2/scenario6 按 `internal/bootstrap/app.go:457-458` 补 `EnableWorkflowOutbox()+EnableRulesOutbox()`。连续 6 轮 `-count=1` 无复现 |
| 0.5 | 补 `cmd/authz-gen` 到 pre-push / CI，防止生成物再次漂移 | 三个 precheck 测试同时绿 | ✅ `TestPrecheckMapIsFresh`/`TestRoutePrecheckAlignment`/`TestWriteRoutesRequirePermission` 全绿 |

**批次 0 执行中新暴露并一并修掉的 2 类问题**（原清单未列，此前被 panic 中断掩盖、测试根本没跑到）：

1. `TestSLAMonitorService_GetSLAComplianceByDefinition`、`TestTicketSLAService_GetOverdueTicketsUsesPersistedDeadline` — fixture 只写工单内嵌 `sla_definition_id`/`sla_*_deadline`，而 Phase 3 读路径以 `sla_states` 为权威源，于是要么返回空、要么走 `getOverdueTicketsInline` 回退分支。后者意味着名叫 `UsesPersistedDeadline` 的用例实际验的是**非**持久化路径。已补 `sla_states` 行并按真实语义收紧断言（条数、`slaDefinitionID`、合规率 100）。
2. `TestChangeService_*` 的类型化断言与 `submitted` fixture 属批次 0 范围外的 C2，未动。

C2（change `submitted`）与 C3（4090 契约）**不进批次 0**，见 §4 决策点。

**批次 0 收口状态**（`go test -count=1 $(go list ./... | grep -v migrations)`，与 CI 同一包集）：

失败测试函数从 16 个降到 4 个，且全部落在 C2/C3 两个待拍板决策上：

| 剩余失败 | 数量 | 阻塞于 |
|:--|:--|:--|
| `TestChangeService_UpdateChange_StatusTransition`、`TestChangeService_SubmitChange_Success`、`TestScenario3_*`、`TestScenario6_*`（change 子用例） | 2+2+1 | **N1** |
| `TestTransitionStatus_NoBoundInstanceFallsBack`、`TestScenario10_*`（7 个子用例） | 1+7 | **N2** |

`middleware`、`integration`、`handlers/change`（除 N2 一个）、`release`、SLA 全部子包均已绿。

### 批次 1：消灭假成功（2-3 天）

对每个占位能力二选一：**接真实现**，或**显式 unready 且不注册路由**。禁止"注册了但恒失败"。

1. 撤掉 `GET /tickets/types` 硬编码 stub，统一走 `/ticket-types`。
2. 工单导出/导入：实现，或从路由摘除并在能力清单标 unready。
3. 问题/知识评论：落地评论模型，或下线路由（当前 5003 可保留，但需在 `product-capabilities` 显式登记）。
4. 工单/事件自动化规则：接 command/outbox（复用 `EnqueueTx`），不再 log+计数。
5. 知识浏览量/评分/推荐：按仪表盘去模拟值的同一口径改为真实统计或如实为 0。
6. CMDB 云发现：把 `CommandRunCMDBCloudDiscovery` 的入队接上，或连 job 端点一起下线。
7. `CommandDeliverServiceRequest`：注册 handler 并入队，否则删除常量。
8. 前端：移除 `/reports` 的高级操作入口直到 `advancedReporting` 为 true；补 skills 管理 UI 或标灰；`/admin/tags` 菜单改指向已有 `/tags`。

验收：`npx jest api-contract.test.ts` 0 mismatch 且 allowlist 条目数不增；每个下线能力在 `product-capabilities.ts` 有显式登记。

### 批次 2：契约与安全（2 天）

1. `handlers/knowledge/handler.go:519-529` 响应改 camelCase DTO + 契约测试。
2. AI 审计：model/provider/confidence/latency 由服务端观测注入，拒绝客户端自报值；`POST /ai/audit` 只接受服务端已存在的调用 ID。
3. email intake / ticket 同步：enqueue 移入业务事务（`EnqueueSQLTx`），加"enqueue 失败回滚业务写入"事务测试。
4. 8 个测试文件补 ctx 传递。

### 批次 3：可观测性缺口（2 天）

1. Delivery 读面：`GET /api/v1/admin/notifications/deliveries`（按 aggregate/tenant/状态过滤）+ 命令详情内嵌 delivery；outbox `last_safe_error` 净化后落库可查。
2. Skill 注册表持久化：新增 `skill_definition` 表（tenant+name 唯一、version、checksum），注册/提升走 DB + 审计，重启不丢。
3. 知识文章版本：在生产更新路径写 `KnowledgeArticleVersion` 快照（legacy 逻辑迁移过来），补历史/回滚测试。

### 批次 4：双轨收敛（另立排期，改动面大）

1. 删除 `service/problem_service.go`（确认零调用方后 `git rm`，保留测试断言到 `handlers/problem`）。
2. change 状态机词表统一为 `pending`：转换表 key 改 `pending`、删 `:844-848` 归一化；`apiChangeStatus` 读映射**保留**并加存量数据说明。
3. 旧审批 HTTP 面下线（依赖收敛计划 §4 契约测试清单）。

## 4. 需要拍板的新增决策（编号 N，避开既有 D1-D8）

**N1：change 的 `submitted` 怎么处理（C2，阻塞 4 个测试）**
`persistedChangeStatus` 的 `pending→submitted` 反转是纯 bug。但修法有两种：
- (a) 删反转 + 转换表 key 改 `pending` + 删归一化 —— 词表一次收敛，但动到生产共享函数 `IsValidChangeStatusTransition`（`handlers/change` 4 处调用）。
- (b) 只删 `persistedChangeStatus` 的反转，保留 `submitted` key 与归一化 —— 改动最小，legacy 与 `handlers/change` 都能绿，但 `submitted` 幽灵继续留着。
建议 **(b)** 先解阻塞，把 (a) 留给批次 4。

**N2：无 BPMN 绑定时审批该 fail-closed 还是回退（C3，阻塞 scenario10）**
现状 `service/bpmn_approval_bridge_service.go:47,93,133` 无待办即返 4090 conflict，导致 service-catalog 多级审批在测试里全断。两个方向：
- **保留 fail-closed**：则 scenario10 与 `TestTransitionStatus_NoBoundInstanceFallsBack` 的期望要改成断言 4090，且必须给"未绑定流程"的租户一条明确的运营可用路径（配置流程绑定），否则生产上未配 BPMN 的租户审批全线不可用。
- **恢复回退**：无绑定时回落到审批链直判，4090 只在"有绑定但无待办"时出现。
我倾向 **恢复回退 + 精确区分两种 4090**，因为这直接决定默认初始化（未配流程）的租户能不能用。

> 与既有计划的关联：这实质是 `output/dev-improvement-plan-2026-09-27.md` §10 的 **D2（Legacy 审批读取端点是否在 Phase 1 移除，cutoff 2026-11-01）** 与 **D8（BPMN Service Task 配置读取失败降级：建议阻塞并告警，不用默认审批人）** 的同一处判定面。若按 D8 的"阻塞并告警"口径，N2 应选 fail-closed 分支，但需同时补齐"未绑定"与"已绑定无待办"两种可观测错误语义。

**N3：统一工作台在租户内的可见范围**
`GET /api/v1/workbench` 认证+租户域已确认无问题（§2 P0#4 已撤销）。开放问题是同一租户内 agent 能否看到他人待办：现有 `handlers/workbench/handler.go:41-45` 支持按 `assigneeId` 传参过滤，但**未校验该 ID 是否等于当前用户**，即缺省返回整租户工作项、且允许指定他人。需定义为"默认仅本人 + 需权限才可跨人"，或确认租户内全员可见本就是产品意图。

## 5. 复核中被推翻的子代理结论（记录以免复发）

梳理阶段共 5 条结论经复核**不成立**，已从清单中删除或修正：

1. ~~"`/api/v1/menus/*` 的端点权限 map 为空，读菜单无权限门"~~ → **错**。引用的 `handlers/rbac/routes.go:31-39` **文件不存在**（`ls handlers/rbac/` 只有 handler.go/handler_test.go/service.go）；真实注册在 `router/common_system_routes.go:146-151`，逐条带 `middleware.RequirePermission("system_config", "read"/"write")`。
2. ~~"`service/notification_service.go` 与 `ticket_comment_service.go` 无生产调用方、可退役"~~ → **错**。两者分别由 `internal/bootstrap/app.go:722`、`:375` 构造并驱动通知与评论 HTTP 面。真正零非测试调用方的只有 `NewProblemService` 与 `NewChangeService`。
3. ~~"工单评论 DTO `CreatedAt` 无 json tag 且从不赋值"~~ → **错**。`dto/ticket_comment_dto.go:34` 为 `CreatedAt time.Time `+"`json:\"createdAt\"`"+`，`:57` 由 `comment.CreatedAt` 赋值。
4. ~~"MCP `tools/call` 直接 `ExecuteTool`，绕开带 RBAC 3-gate + audit 的路径"~~ → **不成立**。全仓无 `tools/call` 端点（grep 零命中）；`handlers/ai/service.go:118` 的 `ExecuteTool` 是唯一工具执行入口，并由 `:407` 在同一服务内调用， cited 行 `handlers/ai/handler.go:419-442` 实际是 `AnalyzeTicketWithAudit`。
5. ~~"`GET /api/v1/workbench` 缺端点权限，是 RBAC 缺口"~~ → **观察属实、定性错误**。该路由确实无 `RequirePermission`，但按 `output/dev-improvement-plan-2026-09-27.md` §6 的三方裁定，身份自读类路由用 AuthMiddleware+租户上下文即为正确设计；已降级为产品口径问题（§4 N3）。

另有 1 条需分层理解：**"22 个端点无 RBAC"** 作为风险陈述方向正确，但清单必须按读/写与公开探针意图分层，不能一律当越权 —— `*/health`、`/ready`、`/metrics`、`/version` 属设计上公开。已抽查 `cli/info`、`mcp/info`、`marketplace/publish`、`release-management`、`collaboration/board` 在 `precheck_fallback.go` 与 `rbac_precheck_gen.go` 中命中 0 条，确无声明。

## 6. 排期建议

批次 0 与 1 合计约 4 天，可立刻开工且不动业务规则；批次 2、3 各约 2 天。
批次 4 不独立排期 —— 它就是 `output/dev-improvement-plan-2026-09-27.md` Phase 1/Phase 4 的内容，本文只提供现状核对；`service/problem_service.go` 退役可直接挂到该计划 Phase 4「清理」。
**N1、N2 必须先定**，否则批次 0 无法收口到全绿；N3 不阻塞批次 0，可并入既有 D5「统一工作台 MVP」的产品评审一起定。
