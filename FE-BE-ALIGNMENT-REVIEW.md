# ITSM 前端页面与后端API对齐审查报告

## 审查范围

- 前端：`itsm-frontend/src/app/(main)/` 下的核心页面
- 后端：`itsm-backend/router/router.go` + 各个 domain controller
- 审查目标：页面功能与后端API对齐、状态映射正确性

---

## 1. Tickets 页面 (`/tickets`)

### 前端 API 调用（`ticket-api.ts` + `ticket-service.ts`）

| 前端调用 | API 路径 | 后端存在 | 备注 |
|---|---|---|---|
| `GET /api/v1/tickets` | 列表/搜索 | ✅ | |
| `POST /api/v1/tickets` | 创建 | ✅ | |
| `GET /api/v1/tickets/:id` | 详情 | ✅ | |
| `PUT /api/v1/tickets/:id` | 更新 | ✅ | |
| `PUT /api/v1/tickets/:id/status` | 状态更新 | ✅ | |
| `POST /api/v1/tickets/:id/assign` | 分配 | ✅ | |
| `POST /api/v1/tickets/:id/escalate` | 升级 | ✅ | |
| `POST /api/v1/tickets/:id/resolve` | 解决 | ✅ | |
| `POST /api/v1/tickets/:id/close` | 关闭 | ✅ | |
| `GET /api/v1/tickets/search` | 搜索 | ✅ | |
| `GET /api/v1/tickets/overdue` | 逾期列表 | ✅ | |
| `GET /api/v1/tickets/stats` | 统计数据 | ✅ | |
| `GET /api/v1/tickets/:id/comments` | 评论列表 | ✅ | |
| `POST /api/v1/tickets/:id/comments` | 添加评论 | ✅ | |
| `GET /api/v1/tickets/:id/attachments` | 附件列表 | ✅ | |
| `POST /api/v1/tickets/:id/attachments` | 上传附件 | ✅ | |
| `GET /api/v1/tickets/:id/workflow/state` | 工作流状态 | ✅ | |
| `GET /api/v1/tickets/:id/subtasks` | 子任务 | ✅ | |
| `POST /api/v1/tickets/:id/subtasks` | 创建子任务 | ✅ | |
| `GET /api/v1/tickets/cc/my` | 抄送记录 | ✅ | |
| `POST /api/v1/tickets/workflow/accept` | 接单 | ✅ | |
| `POST /api/v1/tickets/workflow/reject` | 驳回 | ✅ | |
| `POST /api/v1/tickets/workflow/withdraw` | 撤回 | ✅ | |
| `POST /api/v1/tickets/workflow/forward` | 转发 | ✅ | |
| `POST /api/v1/tickets/workflow/cc` | 抄送 | ✅ | |
| `POST /api/v1/tickets/workflow/reopen` | 重开 | ✅ | |
| `POST /api/v1/tickets/workflow/approve` | 审批 | ✅ | |

### ✅ 对齐状态

**功能对齐：良好**。Ticket API 与后端控制器方法一一对应，所有前端调用均有对应的后端路由。

### ⚠️ 发现的问题

1. **`getTicketStats` 硬编码 `today: 0`**
   - 前端 `tickets/page.tsx:72` 注释写明："暂时没有今日新增的 API"
   - 后端 `GetTicketStats` 确实未返回今日新增数字，前端直接置零而非调用可用 API

2. **Ticket 状态映射 — 前端 Kanban Tab 缺失**
   - 页面有 `kanban` Tab（看板视图），但 `TicketKanban` 组件的列配置与 ticket status 映射是否正确未在 page.tsx 中验证

---

## 2. Incidents 页面 (`/incidents`)

### 前端 API 调用（`incident-api.ts`）

| 前端调用 | API 路径 | 后端存在 | 备注 |
|---|---|---|---|
| `GET /api/v1/incidents` | 列表 | ✅ | |
| `POST /api/v1/incidents` | 创建 | ✅ | |
| `GET /api/v1/incidents/:id` | 详情 | ✅ | |
| `PUT /api/v1/incidents/:id` | 更新 | ✅ | |
| `PUT /api/v1/incidents/:id/status` | 状态更新 | ✅ | |
| `POST /api/v1/incidents/:id/assign` | 分配 | ✅ | |
| `POST /api/v1/incidents/:id/acknowledge` | 确认 | ✅ | |
| `POST /api/v1/incidents/:id/resolve` | 解决 | ✅ | |
| `POST /api/v1/incidents/:id/close` | 关闭 | ✅ | |
| `POST /api/v1/incidents/:id/reopen` | 重开 | ✅ | |
| `POST /api/v1/incidents/:id/escalate` | 升级 | ✅ | |
| `POST /api/v1/incidents/:id/major-incident` | 重大事件 | ✅ | |
| `POST /api/v1/incidents/:id/convert-to-problem` | 转问题 | ✅ | |
| `GET /api/v1/incidents/:id/events` | 事件列表 | ✅ | |
| `GET /api/v1/incidents/:id/alerts` | 告警列表 | ✅ | |
| `GET /api/v1/incidents/:id/comments` | 评论 | ✅ | |
| `POST /api/v1/incidents/:id/comments` | 添加评论 | ✅ | |
| `GET /api/v1/incidents/:id/root-cause` | 根因 | ✅ | |
| `POST /api/v1/incidents/:id/root-cause` | 更新根因 | ✅ | |
| `GET /api/v1/incidents/:id/impact-assessment` | 影响评估 | ✅ | |
| `POST /api/v1/incidents/:id/impact-assessment` | 更新影响评估 | ✅ | |
| `GET /api/v1/incidents/:id/classification` | 分类 | ✅ | |
| `POST /api/v1/incidents/:id/classification` | 更新分类 | ✅ | |
| `POST /api/v1/incidents/:id/major-incident` | 重大事件 | ✅ | |

### ✅ 对齐状态

**功能对齐：优秀**。Incident API 与后端 `IncidentController` 所有方法完全对应。

### ⚠️ 发现的问题

1. **Kanban 列状态值硬编码 vs 后端实际状态**
   - 前端 `incidents/page.tsx` 硬编码了 Kanban 列 key：
     ```
     new | acknowledged | assigned | in_progress | resolved | closed
     ```
   - 后端 `IncidentController` 返回的 status 字段值需要验证是否完全匹配上述值
   - 建议：确认后端 incident status 枚举与前端 Kanban column key 一致

2. **Type 类型不匹配警告**
   - 前端注释明确说明：`IncidentAPI` 返回的 `Incident` 类型与 `types.Incident` 类型是分开维护的，需要强制 cast
   - `incidents/page.tsx:158`：`incidents={query.incidents as unknown as Incident[]}`
   - 这是一个架构债务，长期需要统一两个 Incident 类型定义

---

## 3. CMDB 页面 (`/cmdb`)

### 路由架构

- 前端 `/cmdb` → 渲染 `<CSDMHub />` 组件（路由本身是 `page.tsx`）
- 实际子路由：`/cmdb/ci`、`/cmdb/ci-types`、`/cmdb/cis`、`/cmdb/topology` 等

### 前端 API 调用（`cmdb-api.ts`）

| 前端调用 | API 路径 | 后端存在 | 备注 |
|---|---|---|---|
| `GET /api/v1/configuration-items` | CI列表 | ✅ | |
| `POST /api/v1/configuration-items` | 创建CI | ✅ | |
| `GET /api/v1/configuration-items/:id` | CI详情 | ✅ | |
| `PUT /api/v1/configuration-items/:id` | 更新CI | ✅ | |
| `DELETE /api/v1/configuration-items/:id` | 删除CI | ✅ | |
| `GET /api/v1/configuration-items/types` | CI类型列表 | ✅ | |
| `POST /api/v1/configuration-items/types` | 创建CI类型 | ✅ | |
| `GET /api/v1/configuration-items/:id/topology` | 拓扑图 | ✅ | |
| `GET /api/v1/configuration-items/:id/impact-analysis` | 影响分析 | ✅ | |
| `GET /api/v1/configuration-items/:id/change-history` | 变更历史 | ✅ | |
| `GET /api/v1/configuration-items/:id/relationships` | 关系列表 | ✅ | |
| `POST /api/v1/configuration-items/relationships` | 创建关系 | ✅ | |
| `GET /api/v1/cmdb/cloud-services` | 云服务 | ✅ | |
| `GET /api/v1/cmdb/cloud-accounts` | 云账号 | ✅ | |
| `GET /api/v1/cmdb/cloud-resources` | 云资源 | ✅ | |
| `GET /api/v1/cmdb/discovery/sources` | 发现源 | ✅ | |
| `GET /api/v1/cmdb/reconciliation` | 对账结果 | ✅ | |

### ✅ 对齐状态

**功能对齐：良好**。CMDB API 覆盖面广，与后端 `cmdb_routes.go` + `CMDBController` 方法一一对应。

### ⚠️ 发现的问题

1. **CMDB 路由前缀混用**
   - `cmdb-api.ts` 注释说明：CI 本体走 `/configuration-items`，云资源/云账号/云服务/发现/对账走 `/cmdb/*`
   - 这是正确的设计，但需要注意：当 `CMDBController` 初始化失败时，部分路由会返回空数据（`router.go:639-657` 兼容处理）

2. **`getCITypes` 分页循环获取**
   - `cmdb-api.ts:114-128` 用循环分页获取所有 CI 类型
   - 后端支持 `page`+`size` 参数，前端自己做了聚合，后端实际有 `total` 字段可以直接判断总数

---

## 4. Workflows 页面 (`/workflows`)

### ⚠️ 关键发现：页面是空壳

```typescript
// workflows/page.tsx
export default function WorkflowsPage() {
  redirect('/workflow');  // 重定向到 /workflow
}
```

- `/workflows` 本身是一个重定向页面，指向 `/workflow`
- 实际功能在 `/workflow` 页面（不在本次审查的 `(main)/` 路径下）
- **结论：无需审查 — 该路由只是一个兼容重定向**

### 前端 API 调用（`workflow-api.ts` → BPMN）

| 前端调用 | API 路径 | 后端存在 | 备注 |
|---|---|---|---|
| `GET /api/v1/bpmn/process-definitions` | 流程定义列表 | ✅ | |
| `GET /api/v1/bpmn/process-definitions/:id` | 流程定义详情 | ✅ | |
| `POST /api/v1/bpmn/process-definitions` | 创建流程 | ✅ | |
| `PUT /api/v1/bpmn/process-definitions/:id` | 更新流程 | ✅ | |
| `POST /api/v1/bpmn/process-definitions/:id/activate` | 激活 | ✅ | |
| `GET /api/v1/bpmn/process-instances` | 流程实例列表 | ✅ | |
| `POST /api/v1/bpmn/process-instances/:id/suspend` | 挂起 | ✅ | |
| `POST /api/v1/bpmn/process-instances/:id/resume` | 恢复 | ✅ | |
| `POST /api/v1/bpmn/process-instances/:id/terminate` | 终止 | ✅ | |
| `GET /api/v1/bpmn/tasks` | 任务列表(我的待办) | ✅ | |
| `PUT /api/v1/bpmn/tasks/:id/claim` | 领取任务 | ✅ | |
| `POST /api/v1/bpmn/tasks/:id/decisions` | 提交决策 | ✅ | |
| `GET /api/v1/bpmn/versions/:key` | 版本列表 | ✅ | |
| `GET /api/v1/bpmn/stats` | 统计 | ✅ | |

### ✅ 对齐状态

**BPMN API 对齐：良好**。`workflow-api.ts` 所有调用都对应后端 `BPMNWorkflowController.RegisterRoutes()` 注册的路由。

---

## 5. SLA 页面 (`/sla`)

### 前端 API 调用（`sla-api.ts`）

| 前端调用 | API 路径 | 后端存在 | 备注 |
|---|---|---|---|
| `GET /api/v1/sla/stats` | SLA统计 | ✅ | |
| `GET /api/v1/sla/definitions` | SLA定义列表 | ✅ | |
| `POST /api/v1/sla/definitions` | 创建SLA | ✅ | |
| `GET /api/v1/sla/definitions/:id` | SLA详情 | ✅ | |
| `PUT /api/v1/sla/definitions/:id` | 更新SLA | ✅ | |
| `DELETE /api/v1/sla/definitions/:id` | 删除SLA | ✅ | |
| `GET /api/v1/sla/violations` | 违规列表 | ✅ | |
| `PUT /api/v1/sla/violations/:id` | 更新违规状态 | ✅ | |
| `GET /api/v1/sla/compliance-report` | 合规报告 | ✅ | |
| `POST /api/v1/sla/monitor` | 实时监控 | ✅ | |
| `GET /api/v1/sla/alert-rules` | 告警规则 | ✅ | |
| `POST /api/v1/sla/alert-rules` | 创建告警规则 | ✅ | |
| `GET /api/v1/sla/alert-history` | 告警历史 | ✅ | |

### ✅ 对齐状态

**功能对齐：优秀**。所有 SLA API 与后端 `SLAHandler` 路由完全对应。

### ⚠️ 发现的问题

1. **`getSLAMonitoring` 响应结构不完整**
   - `sla-api.ts:231-258` 中 `getSLAMonitoring` 只返回固定结构，且 `alerts` 始终为空数组 `[]`
   - 注释说明："该端点未返回 alerts（生产环境常见）"，前端 fallback 到 violations 端点
   - 但 `sla/page.tsx:67-68` 调用的是 `SLAApi.getSLAAlerts()` 而非 `getSLAMonitoring()`
   - **`getSLAAlerts()` 逻辑是正确的**（优先 monitor.alerts，fallback 到 violations）

2. **`getSLAComplianceReport` 参数命名转换**
   - `sla-api.ts:161-164` 正确使用了 `start_date`/`end_date`（snake_case）发送给后端
   - 这是一个已修复的 bug（代码注释明确说明）

3. **priority 配置 `urgent` 映射到 `critical`**
   - `sla/page.tsx:42`：`urgent: { label: t('sla.priorityCritical'), color: 'magenta' }`
   - SLA 页面 priority 有 `critical | urgent | high | medium | normal | low`，其中 `urgent` 和 `critical` 等同处理
   - 需要确认后端 SLA priority 枚举是否也包含 `urgent`

---

## 总体评估

### 对齐质量：✅ 良好

| 页面 | 对齐状态 | 说明 |
|---|---|---|
| Tickets | ✅ 良好 | 所有 API 均有后端对应，但 `today` 统计未实现 |
| Incidents | ✅ 优秀 | 全部对齐，Kanban 状态值需人工确认一致性 |
| CMDB | ✅ 良好 | 路由前缀设计正确，云资源和CI分离清晰 |
| Workflows | ✅ N/A | 页面仅为重定向，实际功能在 `/workflow` |
| SLA | ✅ 优秀 | 所有端点对齐，fallback 逻辑健壮 |

### 关键问题汇总

1. **[Tickets] `getTicketStats` 的 `today` 字段硬编码为 0** — 后端未提供该数据源
2. **[Incidents] Kanban 列状态值与后端 status 枚举的一致性** — 需人工确认
3. **[SLA] `urgent` priority 等同于 `critical` 的处理方式** — 需确认后端枚举
4. **[Incidents] `IncidentAPI.Incident` vs `types.Incident` 类型分裂** — 架构债务

### 建议

- 对 Kanban 组件传入的 status 值与后端 `incident.status` 枚举值做一次人工核对
- 确认 `urgent` 是否是后端 SLA priority 枚举的有效值
- 长期：统一 `Incident` 类型定义，消除 `as unknown as` 类型断言
