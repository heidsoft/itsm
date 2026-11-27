# 变更单绑定 BPMN 流程

适用于负责变更流程配置和客户 PoC 验收的租户管理员、变更经理。以下操作按 2026-11-27 主分支 `021f08c0e` 的实现编写。

## 配置租户内的变更路由

1. 在目标租户的工作流管理中创建或选择流程定义，完成部署，记录流程定义 Key。先在独立 QA 租户验证自定义流程的审批和状态推进。
2. 用持有 `bpmn:read` 的账号调用 `GET /api/v1/process-bindings?businessType=change`，保存原规则供回滚。普通、标准、紧急变更的子类型分别为 `normal`、`standard`、`emergency`。
3. 用持有 `bpmn:write` 的账号调用 `POST /api/v1/process-bindings`。下例把高风险普通变更路由到示例 Key `change_high_risk_review`。请替换为本租户已部署的定义。

```json
{
  "businessType": "change",
  "businessSubType": "normal",
  "processDefinitionKey": "change_high_risk_review",
  "departmentId": 0,
  "teamId": 0,
  "conditions": { "risk_level": "high" },
  "priority": 100,
  "isActive": true
}
```

实例启动时，后端仅检索当前租户的启用绑定，按匹配得分排序。同分时数字较大的 `priority` 优先，配置时请避免同分同优先级。变更启动仅提供变更类型、风险和优先级给路由服务。部门、团队、场景和类别字段请留空或填零。管理后台「部门流程」创建带部门范围的绑定，当前变更启动链路无法命中该范围。

没有匹配规则时，紧急变更使用 `change_emergency_flow`，其它类型使用 `change_normal_flow`。自定义定义不可用时，Worker 会检查内置兜底定义，紧急变更可继续回退到普通流程。验收应核对实际定义 Key，以便识别走了兜底的情况。

## PoC 验收与撤回

| 操作 | 通过标准 |
| --- | --- |
| 新建高风险普通变更 | 后台出现关联该变更的流程实例，定义 Key 为本次配置的 Key。 |
| 新建普通变更对照样本 | 用其它风险等级确认原有匹配规则或内置普通流程生效。 |
| 新建紧急变更 | 在默认模板均已部署、且没有适用自定义规则时，实例引用紧急流程。 |
| 从变更页面执行适用的审批和推进操作 | 变更状态、流程待办和审批审计一致。 |
| 在另一个 QA 租户创建同类型、同风险变更 | 按该租户自己的规则选择定义。 |
| 删除本次新建的测试绑定 | 用 `DELETE /api/v1/process-bindings/:id` 删除该绑定，并新建对照变更确认回到原规则。已有实例继续使用其已启动的流程。 |

保存变更后，流程由 Worker 异步启动。验收证据应记录变更 ID、流程实例 ID、实际定义 Key 和必要的审批审计。实例未出现时，按变更 ID 检查 `workflow.start` 命令处理状态及定义部署状态。

## 实现入口与回归命令

`POST /api/v1/changes` → 变更 Handler → 领域 Service → `CreateWithWorkflowCommand` 在同一事务保存变更和命令 → Worker 读取租户内变更 → `ResolveForChange` → BPMN 启动。配置入口由 `handlers/bpmn/process_trigger.go` 注册。

在 `itsm-backend` 下运行：

```bash
go test ./internal/services -run '^TestWorkflowStart_ChangeRouting' -count=1
go test ./handlers/change -run '^TestEntRepositoryCreateWithWorkflowCommandIsAtomic$' -count=1
```

上述测试覆盖 Worker 路由、租户隔离和变更写入与命令的事务关系。浏览器操作、部署状态及客户审批链需在目标 QA 租户单独验收。
