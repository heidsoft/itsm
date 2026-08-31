# 双工作流引擎收敛计划（v1.7）

> 创建：2026-09-01
> 背景：`workflow_engine.go`（JSON 定义驱动，旧）与 `bpmn_process_engine.go`（BPMN，新）并存，
> `ticket_workflow_service.go`（48.9KB）横跨两者。2026-08-31 审核确认 BPMN 引擎四大旧伤
> （并行/包容网关、CompleteTask 事务化、租户 fail-closed、版本漂移）已全部修复，
> BPMN 引擎能力全面领先，具备收敛条件。

## 现状盘点（2026-08-31 实测）

| 项 | 旧引擎（workflow_engine.go 18.3KB） | 新引擎（bpmn_process_engine.go 108KB） |
|---|---|---|
| 定义格式 | JSON（WorkflowDefinition） | BPMN 2.0 XML |
| 网关 | 仅条件转换 | 排他/并行/包容 三类（fork+join） |
| 任务完成 | 无事务保证 | 单 ent.Tx 原子化 |
| 租户 | 无强制上下文 | requireBPMNTenantContext fail-closed |
| 版本 | workflow_version_service（简版） | 完整版本服务（激活/回滚/对比/兼容性评估） |
| 监控 | workflow_monitor | bpmn_monitoring（性能/瓶颈/时间线） |
| 设计器 | 无 | bpmn-js 流程设计器 + AI 生成 + Lint |
| 调用方 | workflow_controller + workflow_monitor/approval/task | 9 域 handler + 审批桥 + SLA + 触发器 |

## 冻结纪律（即刻生效）

1. **旧引擎只修缺陷、不加新能力**（已在类型与构造函数上标记 `Deprecated`，go vet 会对新引用告警）
2. 新功能一律走 BPMN 引擎；若必须动旧引擎，先在本计划登记理由
3. PR 评审时发现 `NewWorkflowEngine` 新增调用点应打回

## 收敛批次（v1.7 内完成）

### 批次 A：只读面迁移（低风险）
- [ ] `workflow_monitor.go` 指标查询改走 `bpmn_metrics_service` / `bpmn_monitoring_service`
- [ ] `workflow_controller.go` 的定义列表/详情接口输出对齐 BPMN 版本服务数据
- 验收：旧引擎无 HTTP 入口读 monitor 数据；BPMN 仪表盘指标不回归

### 批次 B：工单状态机桥接收敛（中风险）
- [ ] `ticket_workflow_service.go` 中横跨两引擎的分支收敛为 BPMN 单路径
- [ ] `workflow_approval.go` 逻辑并入 `bpmn_approval_bridge_service`（会签/或签已在 BPMN 侧）
- 验收：工单创建/审批/完成的 E2E 流程走 BPMN 引擎；旧引擎引用计数为零

### 批次 C：删除旧引擎（需 v1.7 末尾单独发版说明）
- [ ] 删除 `workflow_engine.go` + `workflow_monitor.go` + `workflow_approval.go` + `workflow_task.go` 中旧引擎专属代码
- [ ] `workflow_version_service.go` 与 `bpmn_version_service.go` 二选一保留（保留 BPMN 版）
- [ ] 数据迁移检查：`workflow` / `workflowinstance` ent 表存量实例是否需要导出归档
- 验收：全仓无 `WorkflowEngine` 类型引用；`go build ./...` + 全量测试通过
- 风险控制：批次 C 拆独立 PR，回滚只需 revert 单提交

## CI 门禁（v1.7 期间）

- backend-ci 增加检查：`workflow_engine.go`/`NewWorkflowEngine` 的 diff 若非纯删除或缺陷修复注释，标记 needs-review（可先人工执行，脚本化排到 v1.7 前半段）

## 关联
- 审核依据：`output/workflow-designer-cmdb-review-2026-08-31.md`（第四节 P2-d）
- 同批治理：app.go 拆分（P2-I）、bpmn_process_engine.go 108KB 拆分（P2-e）——引擎收敛完成后再拆新引擎，避免白做
