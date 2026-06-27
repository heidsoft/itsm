# 审批节点类型语义（Approval Node Semantics）

> 适用于 ITSM 流程设计器中「流程配置」Tab 的 `approval_config.approval_type` 字段，
> 以及 BPMN Designer 节点属性面板中每个 `userTask` 的审批行为约定。
> 版本：v1.0  /  日期：2026-06-21

## 1. 字段位置

| 字段 | 类型 | 存储位置 | 层级 |
| --- | --- | --- | --- |
| `approval_config.approval_type` | `single` \| `parallel` \| `sequential` \| `conditional` | `process_definitions.approval_config` JSONB | **流程级** |
| `approval_config.approvers` | `string[]`（用户 ID） | `process_definitions.approval_config` JSONB | **流程级兌底**（可选） |
| `BPMN userTask candidateGroups` | CSV 字符串，组名 | BPMN XML 属性 | **节点级** |
| `BPMN userTask candidateUsers` | CSV 字符串，用户名 | BPMN XML 属性 | **节点级** |
| 展开后 `process_task.candidate_users` | CSV 字符串 | `process_tasks.candidate_users` | 运行时 |

> ⚠️ **重要架构决策**：审批组是**节点级**的，不应存储在 `process_definitions.approval_config.approver_groups`。
> 该字段如出现则会被忽略，正确链路是：
> 前端在节点 inspector 中选组 → 写入 BPMN XML `<userTask candidateGroups="..."/>` → 后端 `bpmn_xml_parser` 解析 → 存入 `process_task.candidate_groups` → 任务分配时由 `GroupResolver` 展开为具体用户。

## 2. 四种审批类型的语义

### 2.1 `single`（单人审批）

| 项 | 说明 |
| --- | --- |
| 行为 | 任一候选审批人通过 → 节点结束（one-of） |
| 失败 | 任一审批人拒绝 → 节点整体失败 |
| 适用 | 简单审批，单一决策者足够 |
| BPMN 落地 | `userTask` + `candidateUsers="alice,bob"` 任一领取并通过即可 |
| 状态机 | `pending → claimed(by X) → completed` 或 `rejected` |
| 完成条件 | 收集到任意一个 `approve` 票 |

### 2.2 `sequential`（串行审批）

| 项 | 说明 |
| --- | --- |
| 行为 | 按顺序逐个审批，前一人通过后才轮到下一人 |
| 失败 | 任意一人拒绝 → 节点整体失败 |
| 适用 | 层级审批：直属经理 → 部门负责人 → IT 安全 |
| BPMN 落地 | **必须使用多个串联的 `userTask`**，每个对应一级审批人 |
| 状态机 | `step1 → step2 → step3 → completed` |
| 完成条件 | 链路上每一级都 `approve`，中途任一 `reject` 即终止 |
| 当前限制 | ITSM 引擎目前不内置串行路由，需要在 BPMN 图中显式建模；`approval_config.approval_type=sequential` 用于前端校验和审计 |

### 2.3 `parallel`（并行审批）

| 项 | 说明 |
| --- | --- |
| 行为 | 所有候选审批人需**同时**通过，节点才完成 |
| 失败 | 任一审批人拒绝 → 节点整体失败 |
| 适用 | 多方会签：财务 + 法务 + 安全同时审 |
| BPMN 落地 | BPMN 原生 Multi-Instance 模式（MI parallel）+ `completionCondition=...` |
| ITSM 引擎 | 已有内置会签能力：`/api/v1/bpmn/tasks/:id/counter-sign` 系列 API |
| 完成条件 | N 个并行子任务全部 `approve`（或达到阈值） |

### 2.4 `conditional`（条件审批）

| 项 | 说明 |
| --- | --- |
| 行为 | 根据运行时表达式（SpEL/JS）动态选择审批人/审批路径 |
| 适用 | 「金额 > 10000 → 触发 CEO 审批」之类规则 |
| BPMN 落地 | 配合 BPMN `exclusiveGateway` + `conditionExpression` |
| ITSM 引擎 | 通过 `process_variables` 在 `process_instance` 上做条件分支 |
| 当前限制 | 表达式以 BPMN 网关为准；`approval_config.approval_type=conditional` 仅作语义标注 |

## 3. 与 candidateGroups 的协作

`approval_type` 决定的是「**通过条件**」，`candidateGroups` 决定的是「**谁可以领**」。
两者正交且都是**节点级**（`approval_type` 可以理解为节点的默认行为兑底）：

```
审批人来源 (节点级) = BPMN candidateUsers
                    ∪ BPMN candidateGroups 展开后的组成员
审批人来源 (流程级兌底) = process_definitions.approval_config.approvers
```

> 运行时链路：后端 `bpmn_process_engine.createUserTask` 看到 `userTask.candidateGroups` 非空时，
> 会调用 `bpmn.GroupResolver.ExpandGroupsToUsers(tenantID, candidateGroups)` 按租户拉取所有匹配组及其成员，
> 把组员用户名合入 `expandedCandidateUsers`，再写入 `process_task.candidate_users`。
> ListUserTasks 会以 OR 查询命中 `assignee / candidate_users / candidate_groups`。

## 4. 状态机图

```
                 ┌─────────────────────────┐
                 │   created (engine)      │
                 └───────────┬─────────────┘
                             │
                             ▼
            ┌────────────────────────────────┐
            │  pending (等待领取/候选人可见) │
            └───┬───────────────────┬───────┘
                │ claim              │ auto-assign
                ▼                   ▼
   ┌────────────────────┐   ┌────────────────────┐
   │ claimed(by user X) │   │  assigned          │
   └─────┬──────────┬───┘   └─────┬──────────┬───┘
         │ approve  │ reject     │ approve  │ reject
         ▼          ▼            ▼          ▼
     completed   rejected    completed   rejected
```

会签模式（parallel）下，每个子任务都进入 `claimed` 状态，
整体完成条件由 `approval_type` 决定。

## 5. 「我的待办」如何命中这些任务

`GET /api/v1/bpmn/tasks`（未传 `user_id` / `assignee` / `candidate_users`）时，
后端会以 JWT 中的 `user_id` 做 OR 查询，覆盖：

1. `assignee == currentUserIdStr`
2. `candidate_users` 包含 `currentUserIdStr` / `username` / `email`
3. `candidate_groups` 包含 `currentUserBelongedGroupsCSV`（展开后的组名 CSV）

返回的列表对应前端 `/approvals/pending` 页面中「我作为候选组员（BPMN）」Tab。

## 6. 配置建议矩阵

| 场景 | `approval_type`（流程级兌底） | `candidateGroups`（节点级 BPMN 属性） | `approvers`（流程级兌底） |
| --- | --- | --- | --- |
| 任何经理审批 | `single` | `managers` | (留空) |
| 经理→总监 | `sequential` | `managers`, `directors` | (BPMN 中分多级 userTask) |
| 三人会签 | `parallel` | `finance,legal,security` | (留空) |
| 金额 > 10万触发 CEO | `conditional` | `managers` | BPMN exclusiveGateway 上挂表达式 |

> 表中 `approval_type` 列为「流程级兌底」。业务上你可以以流程为单位设置默认值，
> 以节点为粒度在 BPMN 中表达该节点的精确语义。

## 7. 后续待办

- [ ] 串行审批由引擎自动生成多 userTask（替代手工建模）
- [ ] 条件审批的表达式可视化编辑器
- [ ] 审批类型在 `process_instance.history` 中的可视化回放