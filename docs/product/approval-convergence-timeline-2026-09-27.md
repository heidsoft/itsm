# 审批/自动化收敛时间线（2026-09-27）

## 当前状态

### ✅ 已完成（2026-09-26）
- **审批事务原子性**（commit `ae60259a`）：Release 和 ServiceRequest 的审批使用事务感知桥接（`CompleteBusinessApprovalTaskWithClient`），业务失败时 BPMN 任务完成回滚
- **回归测试覆盖**：重复审批拒绝 + 业务失败回滚
- **旧入口标记废弃**：`ApprovalService.SubmitApproval` 拒绝所有写动作，两条旧 HTTP 提交入口返回 410

### ⚠️ P1 缺口（未解决）
1. **handler `Validate()` 全是 no-op**：未知 action 静默返回 `Success: true`，应 fail-closed
2. **`ticket_handler.getTenantID` 信任 payload**：回落 `variables["tenant_id"]` 自报归属，跨租户风险
3. **表达式方言 4 套并存**：expr-lang / ACL 自研 / process_routing min_/max_ / ApprovalConditionConfig 三元组

## 收敛时间线

### Phase 1：止血（✅ 已完成 2026-09-26）
- [x] 审批事务原子性修复
- [x] 旧入口标记废弃
- [x] 回归测试覆盖

### Phase 2：P1 缺口修复（建议 2026-10 第一周）
- [ ] handler `Validate()` 实现：未知 action 返回业务错误，不再静默成功
- [ ] `ticket_handler.getTenantID` 移除 payload 信任：强制从认证上下文取 tenant_id
- [ ] 表达式方言统一：收敛到 `service/expression_engine.go`（expr-lang），其他 3 套标记废弃

### Phase 3：原语层（建议 2026-10 第二周）
- [ ] 封闭原语层：`assign` / `notify` / `escalate` / `set_field` / `transition_status`
- [ ] 泛化 `TicketAutomationRule` → `AutomationPolicy`（加 trigger / scope / version）
- [ ] 策略层只做副作用，不改路由（路由仍归 BPMN sequence flow）

### Phase 4：三层 UI 收敛（建议 2026-10 第三周）
- [ ] 流程设计器：只展示 BPMN 流程路由
- [ ] 审批节点设计器：只展示审批人/条件/动作
- [ ] 策略中心：统一入口 `/workflow/automation`，泛化后的 AutomationPolicy

## 验收标准

### Phase 2 验收
```bash
# handler Validate 测试
(cd itsm-backend && go test ./handlers/ticket_workflow -run TestValidate)

# tenantID 隔离测试
(cd itsm-backend && go test ./handlers/ticket -run TestCrossTenant)

# 表达式统一测试
(cd itsm-backend && go test ./service -run TestExpression)
```

### Phase 3 验收
```bash
# 原语层测试
(cd itsm-backend && go test ./service/automation -run TestPrimitives)

# 泛化策略测试
(cd itsm-backend && go test ./service/automation -run TestAutomationPolicy)
```

### Phase 4 验收
- [ ] 前端 E2E：流程设计器 → 审批节点 → 策略中心全链路
- [ ] 文档：三层心智模型用户指南

## 风险与依赖

- **风险**：表达式方言统一可能影响现有 BPMN 流程条件表达式，需回归测试
- **依赖**：Phase 3 依赖 Phase 2 完成；Phase 4 依赖 Phase 3 完成
- **回滚方案**：每阶段独立可回滚，不引入跨阶段依赖

## 参考

- [[approval-automation-convergence-design]]：8 套机制盘点 + 三层心智模型设计
- [[p1-improvement-plan-2026-09-26]]：P1 缺口详细清单
- commit `ae60259a`：审批事务原子性修复
