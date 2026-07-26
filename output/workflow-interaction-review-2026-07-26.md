# 工作流引擎 / 工作流操作 前端交互审计报告

> 审计对象：`itsm-frontend` 工作流（Workflow / BPMN）模块的**前端交互**
> 审计方法：子代理全量枚举 + 本人逐行核实关键源码（标 ✅ 者已 Read/grep 源码坐实）
> 日期：2026-07-26

---

## 0. 一句话结论

**内核是真的，链路是断的。** BPMN 设计器（基于 `bpmn-js` 18.6.2）是真实可拖拽/连线/存 XML 的生产级实现，是本模块质量最高的部分；但它**没有被业务审批流程消费**——用户日常审批走的是另一套硬编码域端点，且前端**根本没有"启动流程实例"的入口**。此外存在 3–4 套重复/死实现的设计器与审批机制、多处破坏性操作缺确认弹窗。

判定：**BPMN 引擎目前是一个"可演示但未被业务消费"的独立 playground。**

---

## 1. 🔴 P0 级问题（已逐行核实源码）

### P0-1. BPMN 引擎与实际审批流程完全割裂（最致命）✅

> **修复状态（2026-07-26）：已完成（阶段1–3）**
> - 阶段1（前端）：审批中心 `approvals/page.tsx`、`approvals/pending/page.tsx` 改为消费 `GET /bpmn/tasks` + `POST /bpmn/tasks/:id/decisions`，BPMN 待办为权威审批入口。
> - 阶段2（后端）：新增 `service/bpmn_approval_bridge_service.go`（BPMNApprovalBridge），工单 `ApproveTicket` 与变更 `TransitionStatus` 审批前先桥接完成绑定的 BPMN 待办任务；无绑定实例回退纯业务审批，桥接失败则失败关闭（中止业务审批，防双轨分叉）。
> - 阶段3（测试补强）：`service/ticket_workflow_bpmn_bridge_test.go`（工单端到端/拒绝/fail-closed）、`handlers/change/service_bpmn_bridge_test.go`（变更端到端/fail-closed/回退）、`service/bpmn_approval_bridge_service_test.go`（桥接单测含租户隔离）共 10 项回归全绿。
> - 遗留候选：服务请求 `/service-requests/:id/approval` 与发布 `/releases/:id/approve` 审批尚未桥接；工单 `delegate` 未同步 BPMN `DelegateTask`。

真正被产品使用的审批中心 `approvals/page.tsx`，审批动作全部直连各业务域硬编码端点：

```
tickets/workflow/approve        (工单)
changes/{id}/approve            (变更)
service-requests/{id}/approval  (服务请求)
incidents/{id}/acknowledge      (事件——本质不是审批)
```

**无一条走 BPMN 任务完成接口 `/api/v1/bpmn/tasks/:id/complete`**（grep 核实，`approvals/page.tsx:142/152/159/205/210/215/220/242`）。

后果：在设计器里画好、部署、版本化的 BPMN 流程，**对真实工单/变更/服务请求审批毫无影响**。"配置即流程"的承诺没有兑现——改了流程图，线上审批链不变。这是工作流模块存在意义的根本问题。

### P0-2. 前端没有"启动流程实例"的入口 ✅
- `startWorkflow` / `process-instances` 在整个 `src/app` 与 `src/components` **grep 0 命中**（仅测试与 hook 引用）。
- 列表页 `workflow/page.tsx:933-935` 的"发起流程"按钮：
  ```tsx
  onClick={() => router.push('/workflow/instances')}   // 只是跳转到实例列表
  ```
  **点了不启动任何实例，只是换个页面** → 死按钮语义。

后果：即便流程设计得再完整，用户在 UI 上也无法手动发起一个实例。BPMN 实例只能靠（不存在的）后端自动触发产生，进一步坐实 P0-1 的"未被消费"。

### P0-3. 破坏性操作缺确认弹窗 ✅
`workflow/instances/page.tsx` **全页 0 个** `Modal.confirm` / `Popconfirm`（grep 核实），而其中：
- **终止实例**（`:247` `terminateWorkflow(record.id, '前端终止')`）——破坏性，直接执行，仅 `message.success` 反馈。
- 暂停 / 恢复 / 批量删除实例——同样无二次确认。

对照：`workflow/page.tsx` 的删除定义（`:311`）、部署（`:331`）、挂起都有 `modal.confirm`，唯独**定义"激活"（`:459-461` `handleActivateWorkflow` 直接执行）无确认**，与同页其他动作策略不一致。

后果：终止一个正在跑的实例是不可逆动作，误点即中断业务流程，无任何挽回提示。

---

## 2. 🟠 P1 级问题（结构性重复 / 缺失）

### P1-1. 路由与实现严重多头（12 + 7 入口）✅
grep 实测路由清单：

**工作流路由（12 个 page.tsx）**
```
/workflow                 主列表 + 内嵌 BPMNDesigner（文件头自标 @deprecated）
/workflow/designer        真正完整的设计器（本模块最佳）
/workflow/ticket-approval 又一套 BPMNDesigner 用法
/workflow/instances       实例监控（+ 残留 page.tsx.bak）
/workflow/versions        版本治理
/workflow/dashboard /audit /automation /bottlenecks /sla   监控子页
/workflows                纯 redirect('/workflow') 兼容壳 ✅
/admin/workflows          旧版重复列表（无设计器集成）
```

**审批入口（7 个，4 套机制并存）**
```
/approvals /approvals/pending          实际使用（走域端点，非 BPMN）
/admin/approvals                        /api/v1/approval-workflows（已废弃仍在）
/admin/approval-chains                  审批链（又一套机制）
/workflow/ticket-approval               BPMN 工单审批设计器
/service-catalog/approvals /settings/approvals   视图分身
```

→ 同一能力（工作流、审批）3–4 套实现互不打通，是长期维护地狱与用户困惑之源。

### P1-2. 设计器 3 套死代码 + 1 个 .bak
- `WorkflowNodePalette.tsx`：自定义 emoji 拖拽面板，拖拽事件 `application/bpmn-node` 与 `BPMNDesigner` 不对接，无页面 import → 死代码。
- `TicketApprovalWorkflowDesigner.tsx`：手搓审批设计器，全仓仅自身引用 → 死代码。
- `WorkflowEngine.tsx`：`@deprecated`，`setTimeout` + 硬编码 mock，无 API → 演示残留。
- `workflow/instances/page.tsx.bak` ✅：仓库残留备份文件。

### P1-3. 任务侧关键 UI 缺失（承接 P0-1）
BPMN 后端已实现 `listMyTasks / claimMyTask / completeNode / vote / createCounterSignTasks`，但**无任何 .tsx 页面消费**：
- ❌ 无"BPMN 我的待办 / 任务详情表单"界面
- ❌ 无"驳回 / 转办(reassign)"UI（后端也无 reassign 方法）
- ❌ 表单绑定仅停留在节点属性的 `formKey` 字符串，**无表单设计器、无按 formKey 动态渲染任务表单**

### P1-4. 监控子页可能显示"假数据"
`WorkflowAPI` 多个方法是前端 mock：`validateWorkflow` 恒返回 `{isValid:true}`、`getWorkflowStats/getNodeStats/getBottleneckAnalysis` 返回空/0、`getTemplates/saveAsTemplate` 返回占位。→ `dashboard/bottlenecks/sla` 子页可能呈现假指标，误导运营决策。

### P1-5. API 客户端分裂为 ≥8 个文件
`workflow-api.ts`（页面用）、`bpmn-workflow-api.ts`（"完整版"，仅 service 用）、`workflow-definition/instance/version/node/stats/countersign-api.ts`。`WorkflowDefinition` 类型在 3 处各定义一套 → 极易漂移。

---

## 3. 🟡 P2 级（一致性 / 无障碍）

- **版本治理无确认**：`workflow/versions` 的删除/回滚/比较均直接执行，仅 `message.success`。
- **图标按钮 a11y 不齐**：`WorkflowNodeInspector`、`/admin/workflows` 的 `Eye/Edit/Copy` 图标按钮缺 `aria-label`（仅 Tooltip，非 a11y 替代）；而 `/workflow` 操作列已正确加 `aria-label` → 实现不一致。
- **冗余依赖**：`@bpmn-io/properties-panel@3.31.0` 在 `package.json` 但属性面板用的是自定义 `WorkflowNodeInspector`，该依赖未被使用。
- **"部署=激活"语义混淆**：`deployProcessDefinition` 内部实际就是 `activateWorkflow`（后端无独立部署概念），术语上"部署/激活"两个按钮易误解。

---

## 4. 🟢 做得好的部分（应保留 / 可复用）

1. **`BPMNDesigner.tsx` + `WorkflowDesigner.tsx` + `WorkflowNodeInspector.tsx`** ✅：真实 bpmn.io 编辑器，拖拽/连线/属性编辑齐全，覆盖用户任务/服务任务/网关/事件全类型；属性面板支持审批语义（单人/会签/阈值/驳回策略）、候选组、条件表达式；保存为 BPMN XML 并真实持久化（`updateWorkflow({ bpmnXml })`）。**本模块质量最高、最接近生产**。
2. **`WorkflowToolbar.tsx`**：保存/保存并部署/重新部署带 `saving`/`deploying` loading，部署前 `validateWorkflow` 阻断错误，交互完整。
3. **`/workflow/instances` 执行历史 Timeline**：动作色标 + 用户 + 耗时 + 评论 + 变量变更，审计体验好。
4. **共享 UI 基建**：`LoadingEmptyError` / `ManagementPageHeader` / `StatsOverview` / `FilterToolbarCard` 在较新页面统一复用，一致性优于旧页。

---

## 5. 修复路线图

### P0（1–2 周，堵住根本断层与安全隐患）
1. **打通 BPMN ↔ 业务审批**（P0-1）：~~让 `/approvals` 的审批动作经 `process-binding` 找到绑定流程 → 调 `/api/v1/bpmn/tasks/:id/complete`~~ **已完成**：前端审批中心走 BPMN 决策接口；后端域审批端点经 `BPMNApprovalBridge` 桥接流程任务（BPMN-first + 失败关闭 + 无绑定回退）。剩余：服务请求/发布审批桥接、delegate 同步。
2. **补"启动实例"UI**（P0-2）：让"发起流程"按钮真正调用 `startWorkflow`（弹出业务键/变量表单），或在工单/变更创建时按绑定自动 `startWorkflow`。
3. **给破坏性操作加确认 + loading**（P0-3）：实例终止/暂停/恢复、定义激活、版本删除统一 `Modal.confirm`（`okType:'danger'`）+ 按钮 loading。

### P1（2–4 周，收敛重复与补齐任务侧）
4. 路由收敛：`/workflow` 为唯一入口，下线 `/admin/workflows`、`/workflow/ticket-approval`，删除 3 套死设计器 + `.bak`。
5. 审批机制收敛：明确 `approval-workflows`/`approval-chains`/BPMN 三选一为主，其余标记退役。
6. 补任务侧 UI：BPMN 我的待办 + 任务表单动态渲染（按 formKey）+ 驳回/转办。
7. 用真实接口替换 `validateWorkflow`/`stats`/`bottleneck` 的 mock，避免假数据。
8. 合并 8 个 API 客户端与重复类型定义为单一 `workflow-api` + 单一类型源。

### P2（持续）
9. 版本操作加确认；图标按钮补 `aria-label`；移除未用的 `@bpmn-io/properties-panel` 依赖；澄清"部署 vs 激活"术语。

---

## 6. 可追踪量化基线

| 指标 | 现状 | 目标 |
|---|---|---|
| 工作流路由入口数 | 12（+7 审批） | 1 工作流 + 1 审批主入口 |
| 审批机制套数 | 4（域端点/approval-workflows/approval-chains/BPMN） | 1 主 + 明确退役其余 |
| 设计器实现套数 | 4（1 真 + 3 死） | 1 |
| 前端可启动实例 | ❌ 无入口 | ✅ 有启动表单 |
| 审批经过 BPMN 引擎 | ✅ 工单/变更已桥接（审批中心 100% 走 BPMN 决策接口；服务请求/发布待桥接） | 明确 100% 或明确"不经过"并删入口 |
| 破坏性操作有确认弹窗 | 部分（终止/激活/版本删除缺失） | 100% |
| BPMN 任务侧 UI（待办/表单/驳回/转办） | ❌ 缺失 | ✅ 齐备 |
| 残留 .bak / mock 方法 | 1 .bak + ≥6 mock | 0 |
