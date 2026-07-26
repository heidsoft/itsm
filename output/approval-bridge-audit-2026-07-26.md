# BPMN 审批桥接与前端操作审计报告

**审计日期**: 2026-07-26
**审计范围**: BPMNApprovalBridge 复用、工单 delegate 同步、P0-2/P0-3
**修复状态**: ✅ 全部行动项已于 2026-07-26 修复完成并通过验证（见文末「六、修复记录」）

---

## 一、BPMNApprovalBridge 复用分析

### 1.1 当前复用状态

| 业务类型 | BusinessType | 是否复用 Bridge | 审批实现方式 |
|---------|-------------|----------------|-------------|
| 工单 (Ticket) | `"ticket"` | ✅ 是 | Bridge + 自有审批记录 |
| 变更 (Change) | `"change"` | ✅ 是 | Bridge + 审批链/审批记录 |
| **服务请求 (ServiceRequest)** | `"service_request"` | ❌ **否** | 自实现3级审批链，直接操作 DB |
| **发布 (Release)** | `"release"` | ❌ **否** | 无审批机制，直接 SetStatus |

### 1.2 服务请求审批实现

**文件**: `handlers/service_request/service.go` - `ApplyApproval()`

- **路由**: `POST /service-requests/:id/approval`
- **问题**: 自实现硬编码三段式审批链（manager → IT → security），直接操作 `ServiceRequestApproval` 表
- **未复用原因**: Service 结构体无 `approvalBridge` 字段

### 1.3 发布审批实现

**文件**: `controller/release_controller.go` - `ApproveRelease()`

- **路由**: `POST /releases/:id/approve`
- **问题**: 极其简陋 — 直接将状态从 `draft` 改为 `scheduled`，无审批人验证、无审批记录
- **未复用原因**: ReleaseService 结构体无 `approvalBridge` 字段

### 1.4 复用可行性评估

**接口完全兼容**，`BPMNApprovalBridge.CompleteBusinessApprovalTask` 签名：

```go
func (b *BPMNApprovalBridge) CompleteBusinessApprovalTask(
    ctx context.Context, tenantID, actorUserID int,
    businessType string, businessID int,
    action, comment string,
) (bool, error)
```

- `businessType` 已是 `string` 类型，`BusinessTypeServiceRequest` 和 `BusinessTypeRelease` 枚举已存在
- BPMN 流程绑定表已有 `release` 和 `service_request` 的绑定定义

**推荐改动**:

1. **服务请求** (高优先级): 在 `Service` 中注入 Bridge，在 `ApplyApproval()` 步骤 5 前插入桥接调用
2. **发布** (中优先级): 补充审批人校验 + Bridge 桥接

---

## 二、工单 Delegate 与 BPMN DelegateTask 同步分析

### 2.1 现状

| 层级 | 实现 | 状态 |
|-----|------|------|
| **业务层** | `ticket_workflow_service.go:398-452` - 创建新审批记录 | ✅ 已实现 |
| **BPMN 层** | `bpmn_process_engine.go:2025-2044` - DelegateTask | ❌ **未被调用** |

### 2.2 问题详情

**业务层 Delegate** (`ticket_workflow_service.go`):
```go
// 第 439-451 行: 创建新的审批记录
if req.Action == "delegate" && req.DelegateToUserID != nil {
    _, err = txClient.TicketApproval.Create().
        SetTicketID(req.TicketID).
        SetLevel(approval.Level).
        SetApproverID(*req.DelegateToUserID).
        SetStatus(string(dto.ApprovalStatusPending)).
        Save(ctx)
}
```

**BPMN DelegateTask** (`bpmn_process_engine.go`):
```go
func (s *bpmnTaskService) DelegateTask(ctx context.Context, taskID string, newAssignee string) error {
    // 更新 ProcessTask 表的 assignee 和 status
    task.TaskVariables["delegated_from"] = task.Assignee
    task.TaskVariables["delegated_time"] = time.Now().Format(time.RFC3339)
    _, err = s.client.ProcessTask.UpdateOne(task).
        SetAssignee(newAssignee).
        SetStatus("delegated").
        Save(ctx)
}
```

### 2.3 结论

**工单 delegate 未同步 BPMN DelegateTask**，两者独立运行：
- 业务层创建新的 `TicketApproval` 记录
- BPMN 流程任务仍是原审批人

**影响**: 当工单绑定 BPMN 流程时，delegate 后 BPMN 侧任务仍是原审批人，业务侧 delegate 记录无效。

---

## 三、P0-2: 前端启动实例入口

### 3.1 现状

| 页面 | 状态 | 说明 |
|-----|------|------|
| `/workflow` (旧版) | ⚠️ 有但已废弃 | "发起流程"按钮存在，页面已标记 `@deprecated` |
| `/admin/workflows` (新版) | ❌ **无** | 只有 CRUD，无启动实例入口 |
| `/workflow/instances` | ❌ **无** | 纯监控，无启动入口 |
| `/workflow/designer` | ❌ **无** | 纯设计器，无启动入口 |

### 3.2 位置

- **旧入口**: `itsm-frontend/src/app/(main)/workflow/page.tsx` 第 974-980 行
- **调用链**: `handleStartWorkflowSubmit()` → `WorkflowAPI.startWorkflow()` → `POST /api/v1/bpmn/process-instances`

### 3.3 建议

在 `/admin/workflows/page.tsx` 或 `/workflow/instances/page.tsx` 添加"发起流程"入口。

---

## 四、P0-3: 破坏性操作确认弹窗

### 4.1 P0 级问题（直接调 API 无确认）

| # | 位置 | 问题 |
|---|------|------|
| 1 | `workflow/automation/page.tsx:88-96` | `handleDeleteRule` 直接调用 `deleteRule(id)` 无 Popconfirm |
| 2 | `problem/ProblemAssociationsTab.tsx:77-86` | `handleRemove` 直接调用 `removeAssociation()` 无 Popconfirm |

### 4.2 已有的保护工具

- `useConfirmDialog` Hook: `lib/templates/ui.tsx:189-249`
- `createActionColumn`: `lib/templates/ui.tsx:97-162` (自动包裹 delete)
- `BatchActionBar`: `components/business/BatchActionBar.tsx` (danger 需配合 confirmTitle)

### 4.3 建议修复

1. **P0-1**: 在 `workflow/automation/page.tsx` 的删除按钮包裹 `<Popconfirm>`
2. **P0-2**: 在 `problem/ProblemAssociationsTab.tsx` 的移除按钮包裹 `<Popconfirm>`
3. **P1 加固**: `BatchActionBar` 对 `danger: true` 强制走 Popconfirm

---

## 五、行动项

| 优先级 | 事项 | 预计改动量 | 状态 |
|--------|------|-----------|------|
| P0 | 服务请求审批复用 BPMNApprovalBridge | 小 | ✅ 已完成 |
| P0 | 发布审批补充审批人 + Bridge | 中 | ✅ 已完成 |
| P0 | 工单 delegate 同步 BPMN DelegateTask | 中 | ✅ 已完成 |
| P0 | 工作流自动化规则删除加 Popconfirm | 小 | ✅ 已完成 |
| P0 | 问题关联移除加 Popconfirm | 小 | ✅ 已完成 |
| P1 | 前端添加启动实例入口 | 中 | ✅ 已完成 |

---

## 六、修复记录（2026-07-26）

### 6.1 后端修复

| 事项 | 改动文件 | 说明 |
|------|---------|------|
| 服务请求审批桥接 | `handlers/service_request/service.go`、`internal/bootstrap/app.go` | `Service` 注入 `approvalBridge`，`ApplyApproval()` 在推进业务审批链前先调用 `CompleteBusinessApprovalTask`；无关联流程实例时回退旧审批链，桥接失败则 fail-closed 中止审批 |
| 发布审批加固 | `service/release_service.go`、`controller/release_controller.go` | 新增 `ApplyReleaseApproval`：草稿态门禁、租户内有效用户校验、创建人禁止自审、Bridge 桥接（fail-closed）；approve→scheduled / reject→cancelled |
| 工单 delegate 同步 | `service/ticket_workflow_service.go`、`service/bpmn_approval_bridge_service.go` | 新增 `DelegateBusinessApprovalTask`，delegate 时同步 BPMN `DelegateTask`（任务改派新审批人、状态置为 delegated）；待办状态集包含 delegated，新审批人可继续完成任务，原审批人失去授权 |

### 6.2 前端修复

| 事项 | 改动文件 | 说明 |
|------|---------|------|
| 自动化规则删除确认 | `workflow/automation/page.tsx` | 删除按钮包裹 `<Popconfirm>`，danger 确认后才调用 `deleteRule` |
| 问题关联移除确认 | `problem/ProblemAssociationsTab.tsx` | 移除按钮包裹 `<Popconfirm>` |
| 启动实例入口 | `workflow/instances/page.tsx` | 新增「发起流程」按钮 + 弹窗（流程定义选择/businessKey/JSON 变量）；修正旧页面传 `id` 的问题，改为传流程定义 `code`（key） |

### 6.3 回归测试

| 测试文件 | 覆盖内容 | 结果 |
|---------|---------|------|
| `service/bpmn_approval_bridge_service_test.go` | 新增 3 个 delegate 用例：无实例回退、改派后新审批人可完成/原审批人被拒、未授权操作人 fail-closed | ✅ 通过 |
| `service/release_approval_test.go` | 新建 8 个用例：审批/拒绝状态流转、自审禁止、非草稿拒绝、无效审批人、租户隔离、Bridge 桥接、fail-closed | ✅ 通过 |
| `handlers/service_request/handler_bpmn_bridge_test.go` | 新建 2 个用例：审批桥接完成 BPMN 任务且业务链同步推进、未授权操作人 fail-closed（任务与业务状态均不变） | ✅ 通过 |

### 6.4 全量验证

- 后端：`cd itsm-backend && go test ./...` ✅ 全部通过
- 前端：`cd itsm-frontend && npx tsc --noEmit` ✅ 0 错误
