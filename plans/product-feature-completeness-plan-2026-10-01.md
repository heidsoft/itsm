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
| C2 `submitted` 死词表 | service/change 2、scenario3 2 | **生产缺陷**（legacy 写入非法枚举）→ 已按 N1 修复 |
| C3 BPMN 审批任务 4090 | scenario10 全部子用例、handlers/change 1 | **契约决策未定**→ 已按 N2 三态收敛，全绿 |
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

7. **知识检索响应 snake_case**（2026-10-02 复核仍在，行号位移）— `handlers/knowledge/handler.go:514-531` 返回 `map[string]interface{}`，键为 `is_published`、`relevance_score`、`created_at`、`updated_at`，违反 camelCase 单一契约，且未走 DTO/Mapper。同时前端 `KnowledgeSearchResult.articles[]` 声明的形状与后端实际 `{items:[…]}` 完全不对应（见 §8）。
8. **AI 审计可被客户端伪造** — `handlers/ai/service.go:540,557` 以 `model=""/confidence=0` 落库；`POST /ai/audit`（`handlers/ai/handler.go:533-591`）接受客户端自报 model/confidence/accepted。
9. **邮件 intake 先提交后 enqueue** — `handlers/email_intake/orchestrator.go:173-186` commit，`:364` 才入队，崩溃窗口丢命令；`service/ticket_service.go:171` 同模式。
10. **~~前端契约测试失败 2 处~~ → 已修（2026-10-02，见 §8）** — `menu-api.ts:95` 拼接 URL 改为静态 base 常量分支；`msp-api.ts:134` 指向不存在的后端路由，按能力开关显式下线而不是补假接口。实测 `npx jest src/lib/__tests__/api-contract.test.ts` = 3 passed / 0 mismatch。
11. **`/reports` 无视自身能力开关** — `app/(main)/reports/page.tsx:3,13` 渲染 `AdvancedReporting`，而 `product-capabilities.ts:55` 标 `advancedReporting:false` 且 reports 写侧在 allowlist 里。
12. **8 个 service 包测试文件未传 ctx**（2026-10-03 复核：现存 5 个） — 原列 `change/incident/problem/ci_relationship/release/root_cause/recommendation_service_test.go` 调 `context.Background()`，与 `.Scan(ctx)` 修复处同类。`problem_service_test.go` 已随台账 E4-13 作为死第二实现的测试删除（其状态机表测 `git mv` 迁到 `handlers/problem/problem_status_machine_test.go`，覆盖线上实现）；`recommendation_service_test.go` 实测在 HEAD 就已不存在，属本清单既有漂移、非本批删除。其余 5 个（change/incident/ci_relationship/release/root_cause）仍在。

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

26. ~~**`service/problem_service.go`（695 行）100% 死代码**~~ → **已实施（2026-10-03，台账 E4-13）** — 实测 `NewProblemService(` 零非测试调用方后整文件删除（695 行 + 其测试 467 行）；`tests/scenarios/` 两条业务流回归改打线上装配的 `problem.NewService(problem.NewEntRepository(client), logger)`，状态机表测迁入 `handlers/problem/`，租户谓词与迁移宽松度三例注入证明断言真的在咬。**这是 change 域 legacy 双轨的第二例，change 那一例仍在（台账 E4-12）。**
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

**N1/N2 收口（2026-10-01 同日，用户拍板「两个一起做，按三态方案落实」）**

上表 4 个失败函数已全部消除，`cd itsm-backend && go test ./...` 81 个包 ok、0 个 FAIL：

- **N1（C2）口径**：写入只允许枚举内的 `pending`，`submitted` 降级为「只读历史词表」。列表/日历按待审批过滤时用 `changeStatusFilter` 同时命中两种写法——原实现按 `submitted` 过滤，新写入的 `pending` 行在列表里不可见，这是第二个 C2 缺陷；迁移表两侧用 `normalizeChangeTransitionStatus` 归一到 `submitted` 词表，只归一源值会把 `draft -> pending`（提交审批）判成非法迁移。回归测试 `TestChangeService_LegacySubmittedStatusStaysReadableAndFilterable` 用 raw SQL 还原枚举校验生效前的存量行（Ent 的 `StatusValidator` 只在 builder 保存前触发，枚举外的值可读不可写），断言读取映射、过滤计数、统计三处；反证（临时删掉旧值 OR 分支）在 `change_service_test.go:680` 失败后恢复。
- **N2（C3）口径**：`BPMNApprovalBridge` 三个公开方法改为三态返回——`(true,nil)` 已桥接、`(false,nil)` 从未绑定（按业务键查不到任何流程实例）、`(false,err)` 已绑定但无操作待办（4090）。`bound` 以「存在任意状态流程实例」判定，实例被挂起或已结束时仍返回冲突，防止已交流程裁决的对象被业务直批绕过。调用方按域处置：`handlers/change` 与 `ticket_workflow_service.go` 旧路径在未绑定时回退审批链并记 `bpmn_handled=false`；BPMN 优先分支保持严格，但成因文案与错误码分离（桥接未接线 5003、工单未绑定 4090、已绑定无待办 4090，见 `bpmnBridgeUnavailable` / `bpmnTicketUnbound`）；服务请求与发布维持「审批必须由流程待办裁决」，仅把「已绑定无待办」的错误换成 bridge 的精确文案（HTTP 仍 409/4090，快照断言不变）。
- **scenario10**：此前 `service_request.NewService(..., entClient=nil, ...)` 不是生产装配，按 C3 保持 fail-closed 的口径它必然红。改为传入 ent client 并用新增的 `seedApprovalProcess` 播一个串行三级审批流程——只预建第一个待办，后续层级必须由真实 BPMN 引擎在完成当前待办时创建，三级指派人分别是 manager/it_admin/security_admin；7 个子用例转绿，服务目录→请求→审批→履约闭环首次真正覆盖生产装配。

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

1. ~~删除 `service/problem_service.go`（确认零调用方后 `git rm`，保留测试断言到 `handlers/problem`）~~ → **已实施（2026-10-03，台账 E4-13）**：按此口径完成——死实现删除、场景测改接线上实现、状态机断言迁入 `handlers/problem`。
2. change 状态机词表统一为 `pending`：转换表 key 改 `pending`、删 `:844-848` 归一化；`apiChangeStatus` 读映射**保留**并加存量数据说明。
3. 旧审批 HTTP 面下线（依赖收敛计划 §4 契约测试清单）。

## 4. 需要拍板的新增决策（编号 N，避开既有 D1-D8）

**N1：change 的 `submitted` 怎么处理（C2，阻塞 4 个测试）**
`persistedChangeStatus` 的 `pending→submitted` 反转是纯 bug。但修法有两种：
- (a) 删反转 + 转换表 key 改 `pending` + 删归一化 —— 词表一次收敛，但动到生产共享函数 `IsValidChangeStatusTransition`（`handlers/change` 4 处调用）。
- (b) 只删 `persistedChangeStatus` 的反转，保留 `submitted` key 与归一化 —— 改动最小，legacy 与 `handlers/change` 都能绿，但 `submitted` 幽灵继续留着。
建议 **(b)** 先解阻塞，把 (a) 留给批次 4。

> **已定并落地（2026-10-01）：按 (b) 的最小改动方向执行**——保留 `submitted` 迁移表词表与读映射，只删写入侧反转，另补两处的归一（过滤命中历史行、迁移表目标值）。批次 4 的 (a)（转换表 key 改 `pending`、删归一化）仍未做，`submitted` 作为只读历史词表继续存在。

**N2：无 BPMN 绑定时审批该 fail-closed 还是回退（C3，阻塞 scenario10）**
现状 `service/bpmn_approval_bridge_service.go:47,93,133` 无待办即返 4090 conflict，导致 service-catalog 多级审批在测试里全断。两个方向：
- **保留 fail-closed**：则 scenario10 与 `TestTransitionStatus_NoBoundInstanceFallsBack` 的期望要改成断言 4090，且必须给"未绑定流程"的租户一条明确的运营可用路径（配置流程绑定），否则生产上未配 BPMN 的租户审批全线不可用。
- **恢复回退**：无绑定时回落到审批链直判，4090 只在"有绑定但无待办"时出现。
我倾向 **恢复回退 + 精确区分两种 4090**，因为这直接决定默认初始化（未配流程）的租户能不能用。

> 与既有计划的关联：这实质是 `output/dev-improvement-plan-2026-09-27.md` §10 的 **D2（Legacy 审批读取端点是否在 Phase 1 移除，cutoff 2026-11-01）** 与 **D8（BPMN Service Task 配置读取失败降级：建议阻塞并告警，不用默认审批人）** 的同一处判定面。若按 D8 的"阻塞并告警"口径，N2 应选 fail-closed 分支，但需同时补齐"未绑定"与"已绑定无待办"两种可观测错误语义。

> **已定并落地（2026-10-01）：三态方案**——「未绑定」与「已绑定无待办」不再是同一个 4090，因此两条分支可以同时成立：change/ticket 未绑定时回退审批链（默认初始化、未配流程的租户可用），已绑定无待办仍 fail-closed（不破坏 D8 口径）；服务请求与发布按既有 P0-1 决策保持「必须由流程待办裁决」，未绑定也冲突。此处遗留的产品问题只有一个：**未接 BPMN 的租户能否审批服务请求**。本次按"不能"保持实现并把 scenario10 改为真实流程装配；若产品要放开，需要单独评审并同步改 `handlers/service_request` 的 fail-closed 注释与断言。

**N3：统一工作台在租户内的可见范围**
`GET /api/v1/workbench` 认证+租户域已确认无问题（§2 P0#4 已撤销）。开放问题是同一租户内 agent 能否看到他人待办：现有 `handlers/workbench/handler.go:41-45` 支持按 `assigneeId` 传参过滤，但**未校验该 ID 是否等于当前用户**，即缺省返回整租户工作项、且允许指定他人。需定义为"默认仅本人 + 需权限才可跨人"，或确认租户内全员可见本就是产品意图。

## 5. 复核中被推翻的子代理结论（记录以免复发）

梳理阶段共 5 条结论经复核**不成立**，已从清单中删除或修正：

1. ~~"`/api/v1/menus/*` 的端点权限 map 为空，读菜单无权限门"~~ → **错**。引用的 `handlers/rbac/routes.go:31-39` **文件不存在**（`ls handlers/rbac/` 只有 handler.go/handler_test.go/service.go）；真实注册在 `router/common_system_routes.go:146-151`，逐条带 `middleware.RequirePermission("system_config", "read"/"write")`。
2. ~~"`service/notification_service.go` 与 `ticket_comment_service.go` 无生产调用方、可退役"~~ → **错**。两者分别由 `internal/bootstrap/app.go:722`、`:375` 构造并驱动通知与评论 HTTP 面。真正零非测试调用方的只有 `NewProblemService` 与 `NewChangeService`（前者已于 2026-10-03 随台账 E4-13 删除，后者仍在，见台账 E4-12）。
3. ~~"工单评论 DTO `CreatedAt` 无 json tag 且从不赋值"~~ → **错**。`dto/ticket_comment_dto.go:34` 为 `CreatedAt time.Time `+"`json:\"createdAt\"`"+`，`:57` 由 `comment.CreatedAt` 赋值。
4. ~~"MCP `tools/call` 直接 `ExecuteTool`，绕开带 RBAC 3-gate + audit 的路径"~~ → **不成立**。全仓无 `tools/call` 端点（grep 零命中）；`handlers/ai/service.go:118` 的 `ExecuteTool` 是唯一工具执行入口，并由 `:407` 在同一服务内调用， cited 行 `handlers/ai/handler.go:419-442` 实际是 `AnalyzeTicketWithAudit`。
5. ~~"`GET /api/v1/workbench` 缺端点权限，是 RBAC 缺口"~~ → **观察属实、定性错误**。该路由确实无 `RequirePermission`，但按 `output/dev-improvement-plan-2026-09-27.md` §6 的三方裁定，身份自读类路由用 AuthMiddleware+租户上下文即为正确设计；已降级为产品口径问题（§4 N3）。

另有 1 条需分层理解：**"22 个端点无 RBAC"** 作为风险陈述方向正确，但清单必须按读/写与公开探针意图分层，不能一律当越权 —— `*/health`、`/ready`、`/metrics`、`/version` 属设计上公开。已抽查 `cli/info`、`mcp/info`、`marketplace/publish`、`release-management`、`collaboration/board` 在 `precheck_fallback.go` 与 `rbac_precheck_gen.go` 中命中 0 条，确无声明。

## 6. 排期建议

批次 0 与 1 合计约 4 天，可立刻开工且不动业务规则；批次 2、3 各约 2 天。
批次 4 不独立排期 —— 它就是 `output/dev-improvement-plan-2026-09-27.md` Phase 1/Phase 4 的内容，本文只提供现状核对；`service/problem_service.go` 退役已按该计划 Phase 4「清理」口径于 2026-10-03 完成（台账 E4-13）。
**N1、N2 必须先定**，否则批次 0 无法收口到全绿；N3 不阻塞批次 0，可并入既有 D5「统一工作台 MVP」的产品评审一起定。

## 7. 批次 0 最终状态（2026-10-01）

- N1、N2 已拍板并落地，批次 0 的 C2/C3 存量失败清零：`cd itsm-backend && go test ./...` 81 个包 ok、0 FAIL；`staticcheck ./handlers/ticket_workflow/... ./service/... ./handlers/change/... ./handlers/service_request/... ./tests/scenarios/...` 无告警，改动文件 `gofumpt -l` 为空。
- 真实生产入口核对：工单审批 `POST /api/v1/tickets/workflow/approve`（`router/ticket_routes.go:219` → `handlers/ticket_workflow/routes.go:189` → `service.TicketWorkflowService.ApproveTicket` → 三态 `BPMNApprovalBridge` → 审批链计数或流程待办）、变更审批（`handlers/change/service.go:1003` 起）、服务请求 `POST /api/v1/service-requests/:id/approval`、发布 `POST /api/v1/releases/:id/{approve,reject}` 均已由测试覆盖，不再只测孤立 helper。
- `handlers/ticket_workflow` 的 HTTP 层断言按三态拆成两个用例：`TestHandler_ApproveTicket_UnboundFallsBackToApprovalChain`（跨租户 404/4004 → 本租户 200 走审批链 → 重复提交 409「审批已处理」，并断言三种 action 的工单/审批状态与 `bpmn_handled=false` 审计标记）、`TestHandler_ApproveTicket_BoundWithoutActionableTaskIsConflict`（实例在、待办已被他人处理 → 409/4090 精确文案，工单/审批/流转记录零写入）。原 `TestHandler_ApproveTicket_RequiresBPMNTask` 把「未绑定」也当成冲突，与 N2 口径相反，已删除。
- 推送后 CI 实测（`gh run list --limit 6` 于 `8113a471`）：`backend-ci`、`test-coverage-guard` 绿；`docs-gate`、`ga-gate`、`Security Scan` 红，但与前一 commit `04fd4cc7` 计数完全一致（C.1 4 处 / C.3 19 条），属存量红、非本批引入。⚠️ 上一版这里写的「docs-gate 通过」只在本地成立：`make docs-gate` 默认 advisory，CI 传 `--strict`，且 `output/`、`itsm-rag/` 被 gitignore 导致链接本地可解析、CI 缺失。
- 存量红 ①② 已修（本批）：C.3 的 19 条内链改为指向仓库内文档或降级为行内代码并标注本地工作稿，C.1 的 `ga-gate.yml` 口令夹具补门禁认可的「开发环境…不得用于生产」注释；在 HEAD 干净 worktree 里 `run-all.sh --strict` 得 7 total / 0 failed。③ gosec / npm audit / `ga-gate` 的 Start core stack 失败仍未处理，需单独排期。
- 仍待拍板：仅 **N3**（统一工作台跨人可见范围）。批次 1 只完成了其中的 api-contract 验收（见 §8），批次 2-4 未开工。

## 8. 执行记录：列表信封契约收敛（2026-10-02）

用户直接下达的四项排期中第 2 项（"列表信封契约收敛（含 api-contract-check 由红转绿并加响应键断言）"）。
第 1 项 fresh-install-gate 见 `495b3b40`。

**代码侧**：`common.ListResponse.MarshalJSON` 删除反射领域别名与嵌套 `pagination`，成为唯一平铺
五键生产者；`ListChanges`（原 `gin.H{"changes":…}`）、知识文章列表（原重复 DTO、缺 `totalPages`）、
活跃告警（缺 `totalPages`）改走标准生产者；`TicketTypeListResponse.Types`、`IncidentListResponse`
旧形状、`KnowledgeArticleListResponse` 删除；事件列表/活跃告警读侧 `size` → `pageSize`，前端
`IncidentAPI` 的 `pageSize→size` 翻译器与 BPMN 两处 `pagination.total` 读取移除。真实生产入口：
`handlers/{change,incident,knowledge,ticket,ticket_type}/handler.go` → `common.SuccessWithPagination`。

**守卫（本次新增，双向棘轮：新增违规与基线过期都失败）** — `itsm-backend/tests/contract/list_envelope_ratchet_test.go`：

| 轴 | 判据 | 基线（2026-10-02 实测） |
|:--|:--|:--|
| `TestListEnvelopeRatchet` | List 信封出现 `items` 之外的集合键 | 30 |
| `TestListEnvelopeKeys` | List 信封缺 `page`/`pageSize`/`totalPages` | 11 |
| `TestListEnvelopePagingAliases` | List 信封带 `size`/`limit`/`offset`/`totalCount` 别名 | 9（CMDB 6 处 `DefaultQuery("size")` + 服务目录 + 通知） |

真实路由键集合断言：`handlers/incident/list_contract_test.go`（`TestList_CanonicalEnvelope`、
`TestList_SizeParamIsNotContract`）、`handlers/knowledge/list_envelope_test.go`
（`TestListArticles_CanonicalEnvelope`，含租户 99 空结果返回 `[]` 而非 `null`）。

**门禁由红转绿（实测）**：
- `api-contract-check` 唯一失败项是 swagger 新鲜度 job。重新生成后定义 344→174（移除 171 个全是
  Ent 生成模型/枚举，新增 1 个 `dto.IncidentListResponse`），路径数 159 不变但集合换 6 条：删
  `/api/v1/projects`、`/api/v1/projects/{id}`、不存在的 `/api/v1/departments/{id}`，补 5 个真实注册的
  部门路由注解；悬空 `$ref` 0，仍有 6 个 `ent.*` 定义由市场/安装路由泄漏（既有债务，未在本次处理）。
- 前端 `src/lib/__tests__/api-contract.test.ts` 实测 3 passed / 0 mismatch（原 1 failed）。P1 #10 关闭。
  ⚠️ 与批次 1 验收口径「allowlist 条目数不增」有 1 条偏离：`msp-api.ts` 的
  `GET /api/v1/msp/allocations/history` 从未被 `router/msp_routes.go` 注册，属悬空接口而非命名漂移，
  按 AGENTS.md 允许的「产品开关 + 明确原因」通道登记 `mspAllocationHistory:false` 并隐藏 MSP
  「分配历史」Tab。选择下线该能力面而不是补假接口；重新开放前提写在 `product-capabilities.ts` 注释里
  （`msp_allocations` 需补 `deallocation_reason`/`created_by`，并新增带租户与 RBAC 校验的历史查询）。

**复核确认仍未修**：P1 #7 知识检索响应 snake_case 依旧存在，行号已随本批改动位移到
`handlers/knowledge/handler.go:514-531`（`is_published`/`relevance_score`/`created_at`/`updated_at`，
`map[string]interface{}` 未走 DTO）。顺带实测：前端 `KnowledgeSearchResult.articles[]`
（`{article, score, highlights}`）与后端实际返回（`{items:[扁平文章字段]}`，无 `score` 包裹）
完全不对应，`KnowledgeIntegration.tsx` 与 `useKnowledgeBase.ts` 拿不到声明的字段——批次 2 第 1 步
需同时修后端 DTO 与前端声明，而不是只改一侧。

**验证命令**：
```bash
cd itsm-backend && go test ./tests/contract/... -run TestListEnvelope
cd itsm-backend && go test ./handlers/incident/... ./handlers/knowledge/... ./common/... ./dto/...
cd itsm-backend && go run github.com/swaggo/swag/cmd/swag init -d . -g main.go -o docs --parseDependency --parseInternal && git diff --exit-code -- docs/
cd itsm-frontend && npx jest src/lib/__tests__/api-contract.test.ts && npm run type-check
```

## 9. 执行记录：流程绑定 key 单一来源 + verify 报错可读化（2026-10-02）

用户排期第 3 项。真实生产入口：`POST /api/v1/departments/:id/init-processes`
（`router` 注册于 `handlers/bpmn/process_trigger.go:63`，要求 `department:write`）→
`ProcessTriggerHandler.InitDepartmentProcesses` → `service.ProcessBindingService.InitDepartmentDefaultBindings`。

**实测到的缺陷（修复前）**：同一份「部门类型 → 流程 key」存在三处实现且已漂移——
`service/bpmn_process_binding_service.go` 的私有 switch（真实入口消费）、
`service/scenario/scenarios.go` 的部门模板目录、`service/department_process_service.go`
（`DepartmentProcessService`，零调用方、零测试的重复实现，含第三份 scenario→业务类型映射）。
三处引用的 11 个 key 中只有 `incident_emergency_flow`、`change_normal_flow`、
`release_approval_flow` 在 `service/bpmn/*.bpmn` 有载体；其余 8 个
（`change_emergency_flow`、`release_test_flow`、`change_requirement_flow`、`expense_approval_flow`、
`budget_approval_flow`、`procurement_flow`、`leave_approval_flow`、`recruitment_approval_flow`）从未存在。
旧循环对缺失一律 `continue`
且仍返回 nil，因此财务/HR 部门初始化实际零绑定却提示成功——违反「禁止把未实现伪装成空成功结果」。
另外两份死清单还带 `tech_review_flow`、`onboarding_approval_flow` 与 operations 的 `standard_change`：
前两者既无载体也没有登记的业务类型映射，第三条会新增一条与 `change_release` 同 key 的绑定，
真实入口从来不含它们，故按「对齐现有生产行为」删除，不擅自扩面。

**收敛结果**：
- 唯一来源 = `service/scenario` 目录，条目补齐 `businessType`/`businessSubType`/`category`，
  内容与真实入口原行为逐条对齐（未擅自增删场景）。
- 可用性不再手抄：新增 `service.BuiltinProcessTemplateKeys()`，直接枚举 `go:embed bpmn/*.bpmn`，
  与 `LoadAndDeployTemplates` 的部署集合同源。
- 私有 switch 与 `service/department_process_service.go` 删除（`git rm`，254 行零引用重复实现）。
- `InitDepartmentDefaultBindings` 区分三类跳过：无模板载体 / 有载体但当前租户未部署（IsActive+IsLatest）/
  已存在绑定。仅「一条未建且未命中已有绑定」时 fail-closed 返回错误并列出缺的 key，重复调用仍幂等。
- `pkg/seeder/initialization_adapter.go` `verifyWorkflowTemplates`：`err != nil || !exists` 合并分支
  配 `err=%w` 会打印 `err=%!w(<nil>)`；现拆为查询失败（带 tenant 与 businessType/businessSubType）
  与定义缺失（说明需要 `service/bpmn/*.bpmn` 载体，否则 workflow-core 回滚）两条。

**守卫**：`service/bpmn_department_bindings_guard_test.go` 对 ready（3）/unready（8）两集合做双向基线
（补模板或删模板都必须显式改基线），并用 enttest 覆盖四种结果；`service/scenario/scenarios_test.go`
锁清单形状（业务类型词表封闭、部门内标识唯一、category 与部门类型一致、priority>0、场景已登记）。
`pkg/seeder/process_bindings_manifest_test.go` 顶部「部门级默认绑定另行处理」的说明已改为指向本守卫。

**待拍板（新增 N4）**：财务/HR/需求变更等场景的产品形态——(a) 为 8 个 key 补独立 BPMN 模板，
还是 (b) 把 `service_request` 类场景指向已部署的通用 `service_request_flow`（种子清单已用
`ticket/service_request → service_request_flow`）。本批只做诚实失败，不代替产品决策，因此这些部门
当前初始化会返回明确的「无 BPMN 载体」错误而不是静默成功。

**验证命令**：
```bash
cd itsm-backend && go test ./service/ -run "TestDepartmentProcessCatalog_ReadinessBaseline|TestInitDepartmentDefaultBindings"
cd itsm-backend && go test ./service/scenario/ ./pkg/seeder/ ./handlers/bpmn/
cd itsm-backend && ~/go/bin/staticcheck ./service/... ./pkg/seeder/... && ~/go/bin/gofumpt -l service/ pkg/seeder/
```

---

## 10. 执行记录：会话真相统一到后端（2026-10-02，排期第 4 项）

### 10.1 实测根因

- **登出在生产里不可达**：`POST /api/v1/auth/logout` 挂在 `AuthMiddleware` 之后，而 access token 只有 15 分钟。用过期 access token 直接打到后端 → HTTP 401 / 业务码 2001，handler 从未执行。`itsm-frontend/src/app/api/[...path]/route.ts` 的代理在此之上又按「cookie 里是否存在三段式字符串」的形状校验自造 `{code:2001}` 401（连 `auth-token=1` 这种非 JWT 标记都算有效），于是过期凭证被当有效会话、Next 边缘中间件再把已登录用户从 `/login` 反复弹走。
- **登出从不吊销 refresh token**：旧实现只调 access 吊销，浏览器 cookie 清掉后服务端那枚 7 天凭证照旧能换新 access token。
- **无 Redis 时吊销检查静默跳过**：`s.redis == nil` 直接当作「没吊销」，单副本/开发环境可无限重放旧 refresh token。
- **四套互不接线的吊销机制**：`middleware.tokenRevocationStore`（仅 access）、`handlers/common` 的 Redis-only refresh 黑名单、从未被生产装配构造的 `service.TokenBlacklistService`（264 行死代码）、TTL 只有 1 小时的 `MinIssuedAt`。
- 顺带核实的旧语义：刷新链路本来就会下发新 `refresh_token` cookie，但**没有原子认领也没有已用检查**，旧值在 7 天内可无限重放；`MinIssuedAt` 的 TTL 只有 1 小时，短于 7 天 refresh 生命周期，且刷新链路从未读它。因此「单次使用」此前并不存在，本批是新建语义而非修好已有语义。

### 10.2 后端改动与回归

`POST /api/v1/auth/logout` 移出鉴权中间件：无条件清两类 cookie，再尽力吊销请求携带的两类 token，吊销存储不可用返回 5003（如实说明「cookie 已清但服务端未吊销」），无凭证重复登出幂等 200。`RevokeRefreshToken` 用 Redis `SET NX` 做原子认领（检查与标记之间无竞态），键名取 JWT 的 SHA-256 摘要（吊销列表存的是仍未过期的凭证，明文进 keyspace 等于泄漏给任何能读 Redis 键的工具）；`MinIssuedAt` TTL 提到 8 天以覆盖 7 天 refresh 生命周期并被刷新链路真正消费；改密/停用/降权统一走 `middleware.InvalidateUserTokens`。新增 `GET /api/v1/auth/session` → `{user, tenants, expiresIn}`，`/auth/login` 与 `/auth/refresh` 响应新增 `expiresIn`；cookie 名称与生命周期统一由 `middleware.AccessTokenCookie/RefreshTokenCookie/AccessTokenTTL/RefreshTokenTTL` 派生，`Set-Cookie Max-Age`、JWT 过期与契约测试同源。

**实测**：`go test ./middleware/... ./router/... ./handlers/auth/... ./service/...` 绿（含过期 access + 有效 refresh cookie 的登出回归、同一枚 refresh cookie 续签第二次 401、cookie 名称与 Max-Age 断言）；`go build ./...`、`staticcheck`、`gofumpt` 通过；全量 `go test ./...` 与修复前的存量失败基线逐条比对**完全一致（13 个包，零新增）**——`tests` 那批失败是场景库缺 `sla_states` 与 `ticket_automation_rules` 的 fixture 存量问题（本批未触碰自动化与 SLA）。

### 10.3 前端改动与实测

新增 `src/lib/api/session-api.ts` 作为登录态唯一来源，三态 `authenticated | unauthenticated | unavailable`：401/403 才是后端判定的未登录，5xx、业务码非 0、响应缺 `user`、网络故障一律 unavailable——既不把瞬时故障当登出把人踢走，也不停在假登录态。删除的六处本地推断：JS 写入的 `auth-token=1` 标记 cookie（`token-storage.isAuthenticated()` 只检查它非空，等于把「曾经登录成功过」当身份）、落地页与服务请求客户端扫 `document.cookie`、`_hasConfirmedSession` 回放持久化 Zustand、边缘中间件按 JWT 形状放行、代理自造 `{code:2001}` 401、`/auth/me` + `/auth/tenants` 双探活失败即「保留会话」（曾伪造 `tenantId:1` 与 `new Date().toISOString()` 塞进 store）。`middleware.ts` 只做遗留菜单 307 重定向；`(main)` 布局在无法确认会话时渲染可重试的「暂时无法确认登录状态」。

续签改由后端 `expiresIn` 驱动并**强制单飞**——单飞不是优化而是必需：refresh token 单次可用后，多个并行 401 各自续签会让后到的请求拿旧凭证认领失败，反而把有效会话判成过期。登出带 `keepalive`（调用方随后整页跳转会取消普通 fetch，服务端就收不到吊销请求），吊销失败返回 `revoked=false` 并由上层告警。删除的死代码与假能力：`lib/auth/jwt-decoder.ts`、`components/layout/RouteGuard.tsx`、`components/providers/Providers.tsx`（守卫挂在永不为真的 `AuthService.getToken()` 上）、废弃客户端 `authApi.refreshToken()/validateToken()` 与 legacy `/api/v1/refresh-token` 前端调用、按浏览器时钟的 10 分钟定时续签、登录页「记住我」（凭证在 httpOnly cookie 里，勾选框从未改变服务端窗口）。SSO 与第三方登录改为回调后回读会话端点，读不到如实报错而不是假装登录。

删掉 store 里那个从未真正承载凭证的 `token` 字段（真凭证只在 httpOnly cookie，JS 读不到）时暴露出一个**静默失效的既有功能**：站内通知页与 Header 都用 `user?.id && token` 给 WebSocket 连接设门禁，而这个 `token` 自 cookie 化起就恒为 `undefined`，于是通知页的 WS 从来没能连上（`notificationWS.connect(userId, token)` 的第二形参也早已无用，认证走 cookie + `POST /api/v1/ws/ticket` 短期票据）。现改为只按 `user?.id`（会话端点确认后的身份）门禁，`connect(userId)` 单参。同时把 zustand persist 的默认浅合并换成显式 `merge`：旧版本写进 `localStorage['auth-storage']` 的 `user`/`isAuthenticated`/`token` 会被默认合并原样复活，等于「一份过期的用户对象 + 一个曾登录过的痕迹」继续伪装成会话；现在 hydration 只采纳 `currentTenant`，身份一律由会话端点重新给出，因此也不再需要运行时的「后端确认过」标记（该标记本批删除，无其他消费者）。

**实测**：`npx jest --coverage=false` = 200 套件全绿 / 3418 通过 / 13 跳过；`npm run type-check` 0 错误；`npm run lint:antd` 0 命中；`npm run lint:check` 0 error（12 条存量 warning 全在未触碰文件：`improvements/[id]`、`sla-monitor`、`workflow/dashboard`、`workflow/instances`、`SmartAssignmentModal`、`ProblemInvestigationTab`、`WorkflowInstanceDetail`、`pwa.ts`、`security.ts`）；`api-contract.test.ts` 仍 0 mismatch。新增 `src/lib/api/__tests__/session-api.test.ts` 锁三态映射、并行续签只发一次请求、单飞锁释放、登出 keepalive；`auth-service.test.ts` 锁「登录成功仍须会话端点确认」「后端接受凭证但会话查询失败时抛错不谎报」「unavailable 不动 store」；`auth.test.ts` 断言 `isAuthenticated/getAccessToken/setAccessToken` 等本地推断入口已不存在；`auth-store.test.ts` 新增 hydration 回归——注入含 `token/isAuthenticated/user` 的旧版 `auth-storage` 后 `rehydrate()`，断言 `isAuthenticated=false`、`user=null`、状态里不存在 `token` 键、仅 `currentTenant.code==='t7'` 被恢复，另一条断言 `login()` 之后再 `rehydrate()` 不会清掉刚确认的会话。

### 10.4 遗留（不在本批擅自扩大范围）

1. 前端设计文档仍描述已删除的会话形态：`itsm-frontend/docs/class-diagram.mermaid:5`、`docs/sequence-diagram.mermaid:11-13`、`docs/system_design.md:97,127,216,237-239,393-394` 写着 4 参数的 `login(username, password, tenantCode?, rememberMe?)` 与 `{access_token, refresh_token, user, tenant}` 登录响应。`rememberMe` 与响应体里的 token 都不存在（`dto/auth_dto.go:19,50` 的 `AccessToken` 是 `json:"-"`），属上一批 cookie 化时就欠下的文档账，本批只核对未重写。
2. 一批测试/脚本夹具仍在读会话真相不存在的位置：`itsm-frontend/tests/e2e/comprehensive-e2e.spec.ts:322,342`、`tests/e2e/business-flows/sla-monitoring.spec.ts`（5 处）、`tests/e2e/flows/flow-auth-journey.spec.ts:62` 与 `flow-ai-chat-stream.spec.ts:30`（读 `localStorage['access_token']`）、`scripts/regression_p1p2.py:60`、`scripts/stage4_runner.py:35`、`scripts/stage5_runner.py:23`、`scripts/test_permissions_and_menus.sh:52` 都取 `data.access_token`；`tests/{browser-e2e-verify*,screenshot-*,verify-designer-fix*}.cjs` 还用 `document.cookie = 'auth-token=…'` 自造登录标记。这些读取在 cookie 化后就已经恒为 `undefined`（不是本批造成），但本批删掉了 `auth-token` 标记 cookie 与本地推断，会让「靠标记 cookie 放行」的夹具彻底失效，应改走 `tests/e2e/auth-utils.ts` 的真实表单登录；`itsm-cli/src/commands/login.tsx:24-25` 同理存的是 `result.token/refresh_token`，需单独核实 CLI 登录是否已不可用。legacy `/api/v1/refresh-token`（`router/router.go:304`）仍注册且有契约测试，本批只移除前端调用方。
3. **需拍板**：登录与切租户时旧 refresh token 未被吊销（会留下孤儿凭证直到 7 天到期）。要修需引入用户维度会话账本（`user_sessions` 表），涉及真实业务库迁移，按约定先给方案再施工，本批不做。
4. `http-client` 的 SSR 形态内存 `token` 字段（`setToken/getAuthToken`）生产已无写入方，仅剩非浏览器调用方语义，按后续清理项处理。
5. `GET /api/v1/auth/me` 仍注册着（注释写「供 middleware 验证」，但没有任何 middleware 调它），实际所有者已收敛为 `/auth/session`。为避免与 RBAC 预检生成物继续不同步，本批保留该路由未摘。

**影响面**：登出/刷新/改密的行为契约已同步 `docs/api-reference.md`、`UPGRADE.md` §1.9（含前端删除清单与「客户端必须持久化新签发 cookie、续签必须单飞」）与 swagger；`scripts/docs-gate/product-surface-baseline.txt` 同批 `service_go_files` 333→332（删死代码）、`bootstrap_app_lines` 1811→1815（吊销存储接线，属收敛而非扩散）；前端删除的三个文件不在任何棘轮键统计内，无需改基线。

## 11. 执行记录：e2e 隔离栈与工单读权限收敛（2026-10-02）

对应排期「先起重建栈再修，端到端验证」这一条。两项决策由用户拍板：建一次性隔离栈再验（不碰 prod 容器与 `itsm_prod`），以及把隔离守卫做成**跑前自动证明**而不是文档约定。

### 11.1 为什么 URL 白名单不够

`itsm-frontend/next.config.ts:10` 与 `src/app/api/[...path]/route.ts:4` 的代理 upstream 默认值都是 `http://localhost:8090`。本机实测：`127.0.0.1:8090` 由 `itsm-backend-prod`（`DB_NAME=itsm_prod`）占用，`[::1]:8090` 由另一个会话起的宿主原生二进制占用。因此「测试只打环回 `:3000`」并不能保证写入落在预期的后端——`POST /api/v1/users` 会经代理进哪一个后端，只取决于 Node 解析 `localhost` 的顺序。结论：守卫必须**行为可证明**（写的行要在本栈数据库里数出来），不能只做地址匹配。

### 11.2 隔离栈与 proof 契约

`scripts/e2e-isolated-stack.sh` + `docker-compose.e2e.yml`（叠加在 dev compose 上）做三件事：

1. 新 project（`itsm-e2e-$E2E_RUN_ID`）+ 唯一容器名 + 独占端口（`127.0.0.1:18090`/`127.0.0.1:3001`）；postgres/redis 不发布宿主端口，验证与清理一律 `docker exec` 进容器，避免任何脚本误连本机其他 postgres。被测后端镜像现场构建并断言运行中的容器镜像 ID 等于刚构建的那个，禁止复用 dev/prod 镜像。
2. canary 证明：经 `:3001` 代理取 CSRF → 代理登录 → `POST /api/v1/users` 建随机名用户 → `docker exec` 进本栈 postgres 数出 `e2e-canary-%` 恰好 1 行。代理若指向别的后端，本栈库里就是 0 行，脚本立刻红。
3. 把证明结果写成 0600 的 proof（`$TMPDIR/itsm-e2e-proofs/<project>.json`，含本栈随机 admin 口令；仓库内不出现可用口令）。

compose 合并语义这一层踩过坑，记下来以免复发：`!reset` 只接受空值、会**丢弃**传入的列表项，`!override` 才整体替换序列。backend/frontend 的 `ports`/`volumes` 需要 `!override`（基础文件里是 `8090:8090`，与本机 prod 直接抢端口），postgres/redis 的端口与卷用 `!reset []`。

Playwright 侧 `tests/e2e/harness.ts` 的 `isolatedBaseURL()` 是硬门禁：`ITSM_E2E_ISOLATED_STACK=1` + 显式 `PLAYWRIGHT_BASE_URL`（不允许有默认值）+ 环回白名单 + proof 未过期 + 地址与 proof 逐字符一致（`localhost` ≠ `127.0.0.1`，本机它们可能解析到不同后端）+ proof 的 `proxyUpstream` 不是环回地址。`playwright.config.ts` 在隔离模式下不再由宿主起 `:3000` dev server。

实测八条分支：正向一次 proof-backed cookie 会话经 `:3001` 读到 `{items,total}` 信封并通过；四种篡改输入（无 proof、地址换成宿主 `:3000`、`proxyUpstream=http://localhost:8090`、proof 过期）逐条被拒且报错可读。这八条已固化成离线回归 `tests/e2e/isolated-stack-guard.spec.ts`（自己造 proof 文件，8 passed，不需要真实栈，可在 CI 跑）。

顺带修掉一个自己造的错：`down` 曾打印「已拆除」而容器与卷一个没少。根因是 `itsm-init` 的 `${E2E_ADMIN_PASSWORD:?}` 在 compose **解析阶段**校验，拆栈不带该变量整条命令失败，而脚本把 stderr 和退出码一起吞了。现在拆栈/查状态先用 up 时留存的口令文件补一个只为让解析通过的值，失败输出可见、退出即报错，并按 `label=com.docker.compose.project` 复核残留，有残留就拒绝删 proof；`up` 里的预清理同样复核，并拒绝把拆栈占位值当真实口令起栈。实测修复后一次 `down` 清掉 5 容器 + 4 卷 + 网络，重复 `down` 幂等退出 0。

### 11.3 工单读权限与筛选（在隔离栈里才测得出来）

全新空卷栈没有存量工单，权限与筛选断言第一次变得可归因。实测数据（经 `:3001` 前端代理）：

- 修复前：technician 名下 23 条却看到 45 条（全租户）。
- 修复后：technician A `{total:3,items:3}`、technician B `{total:2,items:2}`、admin `{total:5}`，`foreign` 数组为空；把 B 的一张工单指派给 A 后，A 的可见范围正确扩到 4 条并含 `ds-b-assigned-a`。
- 筛选下传：`assigneeId=A` 返回 1 条且受理人全等；`requesterId=B&type=incident` 返回 2 条无杂项；`categoryId=999999`（不存在）返回 0 条而不是全量。

两处根因与修法见 `CHANGELOG.md` `[Unreleased]` Fixed 前两条；行级谓词的证据链是 `handlers/ticket/repository_impl_datascope_test.go`（enttest 打真实 `repository/ticket`）+ `handler_test.go`（mock 仓储真的执行谓词，经真实路由断言 2/1/3 与缺 `user_id` 的 fail-closed），两组在修复前实测为红。

### 11.4 本批实测到的其他事实与遗留

1. **不带 `assigneeId` 建单会被自动指派给 bootstrap admin（user 1）**。来源是 `service/ticket_service.go:428` 的 `assignmentSmartService` 兜底：工单无受理人时由智能指派决定，空库里落到 1 号用户。隔离栈 25 条工单里 18 条 `assignee_id=1`。这不是本批改动造成的，但会让「我的工单」类视图与行级 `OwnedOrAssigned` 语义在空库/新建租户下失真，需要单独判断是配置缺省（该由工单类型/队列决定）还是兜底本身错。
2. 隔离栈镜像残留：`itsm-e2e-backend:02oct`、`itsm-e2e-init:02oct` 各 145MB。保留是为了下一次 `E2E_SKIP_BUILD=1` 提速；不要就 `docker rmi itsm-e2e-backend:02oct itsm-e2e-init:02oct`。
3. 一次性验证脚本 `/tmp/verify-datascope.sh` 刻意不进仓库：它依赖 `docker exec` 进指定容器名、并直接读 proof 目录，属于人工取证工具而不是可复用门禁。仓库里被固化的只有 Go 测试与 `isolated-stack-guard.spec.ts`。
4. `staticcheck ./handlers/ticket/` 有 2 处存量 `SA4023`（`handler.go:675,698` 对 `ExportTickets`/`ImportTickets` 的 nil 接口比较），不在本批 diff 里，未顺手改。`gofumpt -l` 对本批四个 Go 文件 0 命中。
5. `cd itsm-backend && go test ./...` 本批实测退出 0（82 个包 ok、无 FAIL）；`npm run type-check` 0 错误。
6. 下一步仍是 #25：`tests/e2e/roles/*.spec.ts` 七个夹具还在用固定口令演示账号（`user1/user123`、`tenant1admin/ta123`、`security1` 的 `password:'test'`）与 `data.access_token`/`Bearer` 写法，隔离栈里必然登不上——这正是本批把夹具底座（cookie 会话 + 缓存 + proof 守卫）建好之后要收的账。以及 #26：常驻栈/生产库里可能已被历史夹具写入的用户与工单，需要只读盘点后再决定清理，按约定先给方案。

**影响面**：新增 `scripts/e2e-isolated-stack.sh`、`docker-compose.e2e.yml`、`itsm-frontend/tests/e2e/{harness.ts,isolated-stack-guard.spec.ts}`；改 `playwright.config.ts`、`tests/e2e/fixtures/auth.ts`、`handlers/ticket/{handler.go,handler_test.go,repository_impl.go}`；文档同步 `CHANGELOG.md`、`README.md`「开发与测试」、`docs/dev-commands-reference.md` §2.5（新增小节，后续编号顺延）与本计划 §11。无新增 API 契约字段（六个过滤参数 DTO 早已声明，本批只是让它真的生效），因此不动 `docs/api-reference.md`。
