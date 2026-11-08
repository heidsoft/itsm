# 工作流跨域联调验收签字清单

> 状态：**待人工验收**。本文件不代替验收结论，签字栏在验收人填写前一律留空。
> 能力矩阵 `workflow` 在签字完成前保持 **Pilot**，不得据代码改动晋升 GA。

## 1. 背景与边界

`workflow` 能力此前停在 Pilot，剩余四项阻断为：

| # | 阻断项 | 本批闭环情况 | 代码事实源 |
| --- | --- | --- | --- |
| 1 | ServiceTask 副作用与流程完成同事务 | 已闭环：输出变量、副作用完成凭据（`_serviceTaskExecutedOccurrences`）、receipt、审计、补偿登记、节点完成与流程推进收进同一个 `ent.Tx`；崩溃回滚后按凭据缺失重放 handler 并复用同一幂等键 | `itsm-backend/service/bpmn_service_task_command_handler.go`、`service/bpmn_service_task_transaction_test.go`、`service/bpmn_service_task_convergence_test.go` |
| 2 | 补偿内置模板与触发入口 | 已闭环：内置模板 `service/bpmn/compensation_recovery_flow.bpmn` 含补偿结束事件、补偿边界事件与 `isForCompensation` 处理器；管理面 `TriggerCompensation` 支持 dry-run/apply、按边界事件解析范围、逆序补偿、occurrence 幂等、触发留痕与审计同事务；CLI `cmd/bpmn-compensation` 默认 dry-run | `itsm-backend/service/bpmn_compensation_trigger.go`、`cmd/bpmn-compensation/main.go`、`service/bpmn_compensation_trigger_test.go` |
| 3 | 存量 pending 待办迁移工具 | 已闭环：`BuildPendingMigrationPlan` 只读计划 + `ApplyPendingMigration` 默认 dry-run、显式 `-apply` 单事务执行；只自动处理孤儿实例与显式指定目标用户的未指派待办，陈旧待办仅 review；幂等键 + 状态复核 + 审计 + 租户过滤 | `itsm-backend/service/bpmn_pending_migration.go`、`cmd/pending-task-migration/main.go`、`service/bpmn_pending_migration_test.go` |
| 4 | 真实租户跨域联调验收签字 | **未闭环**：本批只交付自动化证据与验收流程，真实生产租户的执行与签字必须由人完成 | 本文件、`scripts/acceptance/workflow-cross-domain-acceptance.sh`、`itsm-backend/service/bpmn_cross_tenant_e2e_test.go` |

自动化能证明的是「代码层语义与租户隔离」；**不能**证明「真实租户的业务数据、外部系统副作用、组织审批链、通知渠道在目标环境上按预期工作」。因此第 4 项不可由脚本或 AI 代签。

## 2. 验收前置条件

| 项 | 要求 |
| --- | --- |
| 环境 | 目标环境已部署本批版本，且 `GET /api/v1/capabilities` 的 `workflow` 条目可读 |
| 账号 | 至少两个**不同租户**的真实账号，各自具备工单创建/处理、问题创建、流程实例读权限 |
| 版本 | 记录后端镜像 tag / commit、前端构建号、数据库迁移版本 |
| 窗口 | 写链路会在目标环境创建带 `ACCEPTANCE-` 前缀的测试数据，须在维护窗口内执行并留清理记录 |
| 备份 | 执行前完成数据库备份（或确认可回滚），并记录备份标识 |
| 凭据 | 通过环境变量传入，禁止写入仓库、日志或证据文件 |

## 3. 自动化证据（先跑，作为签字附件）

### 3.1 代码层回归

```bash
cd itsm-backend
go test ./service/ -run 'TestServiceTask|TestCompensation|TestTriggerCompensation|TestBuiltinCompensationTemplate|TestBuildPendingMigration|TestApplyPendingMigration|TestCrossTenant' -count=1
```

必须全绿。覆盖：完成事务原子回滚 / 提交后幂等 / 崩溃重放复用幂等键、补偿触发与逆序幂等、内置模板节点守卫、迁移 dry-run 零写入 / apply 原子幂等 / 租户隔离、跨租户跨业务域真实起流。

### 3.2 目标环境证据脚本

```bash
ITSM_BASE_URL=http://<host> \
TENANT_A_USER=<user> TENANT_A_PASS=<pass> TENANT_A_CODE=<code> \
TENANT_B_USER=<user> TENANT_B_PASS=<pass> TENANT_B_CODE=<code> \
EVIDENCE_DIR=output/acceptance \
bash scripts/acceptance/workflow-cross-domain-acceptance.sh
```

默认只读。需要写链路证据时追加 `ACCEPTANCE_ALLOW_WRITE=1` 并提供
`ACCEPTANCE_TICKET_PAYLOAD` / `ACCEPTANCE_PROBLEM_PAYLOAD`（JSON，含各自必填字段，标题带 `ACCEPTANCE-` 前缀）。
脚本产出 `workflow-cross-domain-<UTC时间戳>.json` 与 `.md`：**SKIP 不计通过**，签字前必须补做或书面说明豁免理由。

## 4. 逐项验收清单

| # | 检查点 | 执行方式 / 判定标准 | 结果 | 证据位置 | 验收人 |
| --- | --- | --- | --- | --- | --- |
| A1 | 六业务域创建均触发 `workflow.start` | 工单 / 事件 / 变更 / 服务请求 / 发布 / 问题各建一单；异步起流后按 `businessKey` 查到 `running` 实例 | 待填写 | 待填写 | 待填写 |
| A2 | 工单生命周期动作推进 BPMN | 详情页按 `state-v2` 的 `availableActions` 显隐；接单/解决/关闭后 `tickets.status` 与实例 `currentActivityId` 同步变化，无 `ticket=resolved / inst=running` 悬挂 | 待填写 | 待填写 | 待填写 |
| A3 | 审批节点不代签 | 流程停在审批节点时业务动作返回可行动的 409；他人已认领任务不得被系统推进；完成审批后正常放行 | 待填写 | 待填写 | 待填写 |
| B1 | ServiceTask 完成凭据与流程推进同提交 | 实例变量里 `_serviceTaskReceipts`、`_serviceTaskExecutedOccurrences` 与流程完成同时可见；人工重放同一命令不重复发通知/Webhook | 待填写 | 待填写 | 待填写 |
| B2 | 崩溃恢复语义 | 模拟 worker 在完成事务提交前被杀；恢复后副作用按同一幂等键重放一次，输出变量不丢、补偿范围不重登 | 待填写 | 待填写 | 待填写 |
| C1 | 补偿内置模板与流程内触发 | 部署 `compensation_recovery_flow`；走到 `EndEvent_Compensate` 时按完成顺序逆序补偿已登记 ServiceTask，各补一次 | 待填写 | 待填写 | 待填写 |
| C2 | 补偿显式触发入口 | `cmd/bpmn-compensation` 先 dry-run 再 `-apply`；按边界事件 `Boundary_Compensate` 解析范围、留痕与审计同事务、重复触发零副作用 | 待填写 | 待填写 | 待填写 |
| D1 | 存量待办迁移 dry-run | `cmd/pending-task-migration -tenant-id <A>` 输出计划且零写入；动作清单含幂等键，陈旧待办只列 review | 待填写 | 待填写 | 待填写 |
| D2 | 存量待办迁移 apply | 指定孤儿实例执行 `-apply`；待办取消 + 实例终结 + 审计在同一提交；重复运行全部 skipped；正常在途实例不受影响 | 待填写 | 待填写 | 待填写 |
| E1 | 跨租户零串写 | A 会话读 B 的实例/待办/审计一律 403/404；A 的迁移、补偿、工单动作均不改变 B 的数据 | 待填写 | 待填写 | 待填写 |
| E2 | 跨域数据关联 | 工单/问题/变更在 CMDB CI、SLA、通知、审计上的关联在两租户内各自成立，无跨租户引用 | 待填写 | 待填写 | 待填写 |
| F1 | 失败与降级 | 通知渠道不可用、AI 侧车降级、命令死信重放时主链路可恢复且不产生重复副作用；死信有可操作文案 | 待填写 | 待填写 | 待填写 |
| F2 | 观测与审计 | 流程实例时间线、审计日志、命令总线摘要可交叉对账；操作人、原因、租户、时间齐全 | 待填写 | 待填写 | 待填写 |
| G1 | 清理 | 所有 `ACCEPTANCE-` 前缀测试数据已按外键顺序清理或显式标记保留；备份/回滚记录齐全 | 待填写 | 待填写 | 待填写 |

## 5. 判定与晋升条件

- **通过**：A1–G1 全部「通过」，第 3 节自动化证据全绿且无未豁免 SKIP，证据文件与本报告一并归档。
- **有条件通过**：须逐条写明遗留项、影响面、补偿措施与跟进批次，且不得包含 A2 / B1 / E1 任一失败。
- **不通过**：任一安全边界项（A3 / B1 / C2 / D2 / E1）失败。

只有「通过」且完成第 6 节签字后，才允许在**后续独立批次**里：把能力矩阵 `workflow` 从 `MaturityPilot` 改为 `MaturityGA`、删除 `pilotReason`、同步 README 成熟度表与商业能力契约、调整相应守卫测试。本批不执行这些动作。

## 6. 人工验收签字（留空）

| 项目 | 填写 |
| --- | --- |
| 验收人 |  |
| 角色 / 部门 |  |
| 验收日期 |  |
| 目标环境 / 后端版本 |  |
| 证据文件（JSON / MD 路径） |  |
| 自动化回归命令与结果 |  |
| 结论（通过 / 不通过 / 有条件通过） |  |
| 遗留问题与跟进批次 |  |
| 签字 |  |

> 签字栏为空即表示**尚未验收**。任何自动化产出、代码合并或 CI 绿灯都不构成签字。

## 7. 参考

- 能力矩阵事实源：`itsm-backend/handlers/capability/handler.go`（`workflow` 条目 `pilotReason`）
- 商业能力契约：`docs/product/itsm-commercial-capability-contract.md`（`BPMN/审批` 行）
- 变更记录：`CHANGELOG.md` `[Unreleased]`
