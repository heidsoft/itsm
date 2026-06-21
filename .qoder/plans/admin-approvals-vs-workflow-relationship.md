# `/admin/approvals` 审批管理与 `/workflow` 工作流的关系调研

## Context（背景）

用户在前端访问 `http://127.0.0.1:3000/admin/approvals`（审批管理）和 `http://127.0.0.1:3000/workflow`（工作流控制台）时，发现两个页面的配置内容看起来完全不同，且找不到任何"关联"或"绑定"入口。本任务通过对前后端代码的真实读取，给出准确的架构关系说明，并明确为什么没有关联配置。

## 结论（结论先行）

**这两套是完完全全互相独立的系统，仅共享"工作流"这三个字的概念。**
前端看不到关联配置不是配置入口缺失，而是后端根本没把它们接起来——它们在数据库表、服务、API 路由上都是隔离的两条线。实际上后端并列存在**三套**流程类子系统，不是两套。

---

## 1. 后端三套并行的流程类子系统

| 模块 | 前端页面 | 后端定位 | 核心数据表 | 服务/引擎 |
|---|---|---|---|---|
| **审批模板管理** | `/admin/approvals` | 轻量级审批节点模板 | `approval_workflows` + `approval_records` | `ApprovalService`（[service/approval_service.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/approval_service.go)） |
| **工单流转** | `/approvals/pending`、工单详情内的状态操作 | 工单生命周期状态机 | `ticket_approvals` | `TicketWorkflowService`（[service/ticket_workflow_service.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/ticket_workflow_service.go)） |
| **BPMN 工作流** | `/workflow`、`/workflow/designer`、`/workflow/ticket-approval` | 完整 BPMN 2.0 流程引擎 | `workflows` + `workflowinstances` + `workflowtasks` + `workflowversions` | `ProcessEngine` + 23 个 [bpmn_*.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn_*.go) 服务 |

## 2. 数据、API、服务三层互不关联的证据

### 2.1 数据库层完全隔离
- [`ent/schema/approval_workflow.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/ent/schema/approval_workflow.go) 字段：`name / ticket_type / priority / nodes JSON / status / is_active / tenant_id` —— **没有 `process_key` 或 `workflow_id` 字段**
- [`ent/schema/workflow.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/ent/schema/workflow.go) 字段：`name / type / definition JSON / version / is_active / tenant_id / department_id` —— **没有反向关联审批模板的字段**
- 两表之间**无外键、无 trigger_id、无 process_instance_id**

### 2.2 API 路由挂载完全分离
- [router/router.go:386-394](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/router.go#L386-L394) 注册 `/api/v1/approval-workflows` → `ApprovalController`
- [router/router.go:399-410](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/router.go#L399-L410) 注册 `/api/v1/tickets/workflow/*` → `TicketWorkflowController`
- [router/router.go:1149-1151](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/router.go#L1149-L1151) 把 BPMN 引擎注册到 `/api/v1/bpmn/*`（注意：**不是** `/api/v1/workflow/*`）
- `/api/v1/workflow/*` 只有一条快捷别名路由（`router.go:1154-1160` 的注释自承是"简化路由"）

### 2.3 服务层无任何交叉调用
通过严格字符串搜索交叉验证：
- 在 [`approval_service.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/approval_service.go) 全文搜 `bpmn.|BPMNService|ProcessEngine` → **0 处真正调用**（15 处匹配全部是 `ApprovalWorkflow` 子串误命中）
- 在 [`bpmn_process_trigger_service.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn_process_trigger_service.go) 搜 `approval|Approval` → **0 处**
- 整个 23 个 `bpmn_*.go` 文件全部不引用 `ApprovalWorkflow` / `ApprovalRecord` / `ApprovalService`

### 2.4 构造函数互不注入
- `NewApprovalService(client, logger)` — 只注入 Ent 客户端
- `NewBPMNWorkflowController(processEngine, versionService)` — 只注入 BPMN 引擎和版本服务
- `NewTicketWorkflowService(...)` — 同样不注入 ApprovalService 或 ProcessEngine

## 3. 前端表现对应关系

| 前端页面 | 调用 API | 数据特征 |
|---|---|---|
| [`/admin/approvals/page.tsx`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/admin/approvals/page.tsx) | `/api/v1/approval-workflows` | 节点数组 `nodes[]`（level/approver_type/approver_ids/approval_mode/timeout_hours） |
| [`/workflow/page.tsx`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/workflow/page.tsx) | `/api/v1/bpmn/process-definitions`（见 [`workflow-api.ts:105`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/api/workflow-api.ts#L105)） | `bpmn_xml`、category、version、instances_count |
| [`/workflow/ticket-approval`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/workflow/ticket-approval/page.tsx) | 同上 | BPMN 设计器，输出 XML，最终写入 `workflows` 表 |
| [`/approvals/page.tsx`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/approvals/page.tsx) | `/api/v1/tickets?status=pending` 等 | 聚合待审批业务对象（工单/变更/服务请求） |

`/admin/approvals` 配置的"节点模板"存进 `approval_workflows.nodes`；`/workflow` 配置的"BPMN XML"存进 `workflows.definition`。**两边连"跳转到对方"的下拉/链接都没有**——因为后端没有 `workflow_id` / `process_key` 字段可以让前端挂关联入口。

## 4. 给用户的一句话答复

> 你看到"看不到关联"是对的——**因为后端目前就是两套（实际三套）独立系统**：
> - `/admin/approvals` 是审批**节点模板**配置器（`ApprovalService` + `approval_workflows` 表）
> - `/workflow` 是独立的 **BPMN 2.0 引擎**（`ProcessEngine` + 23 个 `bpmn_*.go` + `workflows` 表）
> - 两者既不共享数据库表，也不互相调用，API 路由也分开挂载
>
> 如果要让"审批管理"驱动 BPMN 流程，需要先在数据表加关联字段、再在服务层注入 `ProcessEngine`、最后在前端补"绑定流程"入口。

## 5. 未来若要做关联（仅思路，未实施）

如果将来要让 `/admin/approvals` 能触发 BPMN 流程，技术上需要：

1. **数据层**：`approval_workflows` 加 `bpmn_process_key`（可空），`approval_records` 加 `process_instance_id`
2. **服务层**：`ApprovalService.SubmitApproval` 在命中节点审批时调用 `ProcessEngine.StartProcess`；可选中间表 `approval_workflow_process` 支持多对多
3. **API 层**：`/api/v1/approval-workflows/:id` 返回中携带 `linked_process`
4. **前端层**：`/admin/approvals` 详情页增加"BPMN 流程绑定"区块（未绑定显示选择器，已绑定显示跳转 `/workflow/designer`）
5. **兼容性**：`bpmn_process_key` 为空的模板继续走原"纯模板"逻辑

## 关键文件清单（仅供查阅，不修改）

- 前端：[`itsm-frontend/src/app/(main)/admin/approvals/page.tsx`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/admin/approvals/page.tsx)
- 前端：[`itsm-frontend/src/app/(main)/workflow/page.tsx`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/app/(main)/workflow/page.tsx)
- 前端：[`itsm-frontend/src/lib/api/workflow-api.ts`](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/api/workflow-api.ts)
- 后端：[`itsm-backend/router/router.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/router.go)（386-394、399-410、1148-1160 行）
- 后端：[`itsm-backend/controller/approval_controller.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/controller/approval_controller.go)
- 后端：[`itsm-backend/controller/bpmn_workflow_controller.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/controller/bpmn_workflow_controller.go)
- 后端：[`itsm-backend/service/approval_service.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/approval_service.go)
- 后端：[`itsm-backend/service/bpmn_process_engine.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn_process_engine.go)
- Schema：[`itsm-backend/ent/schema/approval_workflow.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/ent/schema/approval_workflow.go)
- Schema：[`itsm-backend/ent/schema/workflow.go`](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/ent/schema/workflow.go)

## 验证方式

本次调研为只读分析，不涉及代码改动。验证手段：
1. 交叉搜索 `approval_service.go` / `bpmn_*.go` / `approval_controller.go` / `bpmn_workflow_controller.go` 之间的相互引用
2. 阅读 `router.go` 的路由注册段确认路径挂载
3. 阅读 `ent/schema/` 两类工作流的实体定义确认字段无交叉
4. 阅读前端 `page.tsx` 和 `workflow-api.ts` 确认调用 API 端点

如需进一步对某一假设进行实操验证（例如手动创建一条审批模板与一条 BPMN 流程，确认两边互不感知），可作为后续任务启动。