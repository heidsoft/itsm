# ITSM 系统Bug报告

## Bug概览

本报告通过代码审查发现了多个潜在bug，分为严重级别：

**总计发现：** 12个Bug
- 🔴 高危：3个
- 🟡 中危：5个
- 🟢 低危：4个

---

## 🔴 高危Bug

### Bug #1: 审核工作流缺少事务保护

**文件：** `itsm-backend/service/knowledge_review_service.go:75-115`

**严重程度：** 🔴 高危

**问题描述：**
`ApproveArticle` 方法中存在数据不一致风险。文章状态更新和审核记录更新不在同一事务中，如果文章状态更新成功但审核记录更新失败，会导致数据不一致。

**代码片段：**
```go
// 更新文章状态为已发布
_, err = article.Update().
    SetStatus("published").
    SetIsPublished(true).
    Save(ctx)
if err != nil {
    return fmt.Errorf("更新文章状态失败: %w", err)
}

// 更新审核记录（独立操作，无事务保护）
_, err = s.client.KnowledgeArticleReview.Update().
    Where(...).
    Save(ctx)
if err != nil {
    s.logger.Warnw("Failed to update review record", "error", err)
    // 仅记录日志，不影响主流程
}
```

**影响：**
- 文章状态为已发布，但审核记录未更新
- 审核历史不完整
- 数据一致性受损

**修复建议：**
使用事务包装两个更新操作：
```go
tx, err := s.client.Tx(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

// 在事务中执行两个更新
// ...

return tx.Commit()
```

---

### Bug #2: 并发场景下的异步goroutine context问题

**文件：** `itsm-backend/service/change_service.go:197-207`

**严重程度：** 🔴 高危

**问题描述：**
创建新的context.Background()可能导致租户隔离信息丢失。在goroutine中使用context.Background()而不是传递原始context，会丢失trace ID、租户信息等上下文。

**代码片段：**
```go
// 触发审批流程（异步）
if s.approvalService != nil {
    go func() {
        approvalCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        // 这里丢失了租户隔离信息
        records, err := s.approvalService.TriggerApproval(approvalCtx, approvalReq)
    }()
}
```

**影响：**
- 审计日志缺少trace信息
- 租户隔离可能失效
- 问题排查困难

**修复建议：**
使用context.WithoutCancel(ctx)或传递必要信息：
```go
approvalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
```

---

### Bug #3: goroutine泄漏风险

**文件：** `itsm-backend/service/problem_service.go:47-53`

**严重程度：** 🔴 高危

**问题描述：**
启动goroutine执行工作流触发，但没有等待机制。如果服务关闭，这些goroutine可能被强制终止，导致资源泄漏或操作不完整。

**代码片段：**
```go
// 触发BPMN工作流（异步执行，不阻塞问题创建）
if s.processTriggerService != nil {
    go func() {
        workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := s.triggerWorkflowForProblem(workflowCtx, problem.ID, tenantID); err != nil {
            s.logger.Warnw("Failed to trigger workflow for problem", "error", err)
        }
    }()
}
```

**影响：**
- goroutine泄漏
- 服务关闭时可能丢失操作
- 资源未正确释放

**修复建议：**
使用sync.WaitGroup或worker pool管理goroutine生命周期。

---

## 🟡 中危Bug

### Bug #4: 权限验证缺失

**文件：** `itsm-backend/service/knowledge_review_service.go:75-115`

**严重程度：** 🟡 中危

**问题描述：**
`ApproveArticle` 方法未验证审核人是否有审核权限。任何登录用户都可以审核文章。

**代码片段：**
```go
func (s *KnowledgeReviewService) ApproveArticle(ctx context.Context, articleID, reviewerID, tenantID int, comment string) error {
    // 未验证reviewerID是否有审核权限
    article, err := s.client.KnowledgeArticle.Query().
        Where(...).
        Only(ctx)
    // ...
}
```

**影响：**
- 权限绕过
- 未授权审核

**修复建议：**
添加权限验证：
```go
if !s.hasReviewPermission(ctx, reviewerID, tenantID) {
    return fmt.Errorf("无审核权限")
}
```

---

### Bug #5: 缺少租户隔离验证

**文件：** `itsm-backend/service/change_approval_service.go:45-75`

**严重程度：** 🟡 中危

**问题描述：**
使用原生SQL插入审批记录时，未验证approver是否属于当前租户。

**代码片段：**
```go
// 验证审批人是否存在
approver, err := s.client.User.Get(ctx, req.ApproverID)
if err != nil {
    return nil, fmt.Errorf("approver not found: %w", err)
}
// 未验证approver.TenantID == tenantID
```

**影响：**
- 跨租户数据访问风险
- 租户隔离失效

**修复建议：**
```go
approver, err := s.client.User.Query().
    Where(user.ID(req.ApproverID), user.TenantID(tenantID)).
    Only(ctx)
```

---

### Bug #6: SLA监控服务N+1查询问题

**文件：** `itsm-backend/service/ticket_sla_service.go:180-205`

**严重程度：** 🟡 中危

**问题描述：**
在循环中为每个工单单独查询SLA定义，导致N+1查询性能问题。

**代码片段：**
```go
for _, t := range tickets {
    // 每次循环都查询一次数据库
    slaDef, err := s.getSLADefinition(ctx, tenantID, t.Type, t.Priority)
    if err != nil {
        continue
    }
    // ...
}
```

**影响：**
- 性能下降
- 数据库压力大

**修复建议：**
预加载所有SLA定义，使用map缓存。

---

### Bug #7: 变更审批缺少租户验证

**文件：** `itsm-backend/service/change_approval_service.go:50-75`

**严重程度：** 🟡 中危

**问题描述：**
创建审批记录时未验证change_id是否属于当前租户。

**代码片段：**
```go
// 验证变更是否存在
_, err := s.client.Change.Get(ctx, req.ChangeID)
// 未验证TenantID
```

**影响：**
- 跨租户操作风险

**修复建议：**
```go
_, err := s.client.Change.Query().
    Where(change.ID(req.ChangeID), change.TenantID(tenantID)).
    Only(ctx)
```

---

### Bug #8: 问题调查服务SQL拼接风险

**文件：** `itsm-backend/service/problem_investigation_service.go:180-210`

**严重程度：** 🟡 中危

**问题描述：**
使用fmt.Sprintf拼接SQL WHERE子句，虽然使用了参数化查询，但拼接逻辑复杂，容易出错。

**代码片段：**
```go
query := fmt.Sprintf(", status = $%d", argIndex)
args = append(args, *req.Status)
```

**影响：**
- SQL语法错误风险
- 维护困难

**修复建议：**
使用Ent ORM或查询构建器。

---

## 🟢 低危Bug

### Bug #9: 错误处理不一致

**文件：** `itsm-backend/service/knowledge_review_service.go:110-111`

**严重程度：** 🟢 低危

**问题描述：**
审核记录更新失败只记录日志，不影响主流程，可能导致审核历史不完整。

**修复建议：**
返回错误或在事务中回滚。

---

### Bug #10: 时间计算精度问题

**文件：** `itsm-backend/service/problem_trend_service.go:100-130`

**严重程度：** 🟢 低危

**问题描述：**
使用`Sub().Minutes()`计算时间差可能丢失精度，应使用`Sub().Round(time.Minute)`。

---

### Bug #11: 排序算法效率低

**文件：** `itsm-backend/service/problem_trend_service.go:135-155`

**严重程度：** 🟢 低危

**问题描述：**
使用冒泡排序，时间复杂度O(n²)，建议使用sort.Slice。

---

### Bug #12: 缺少空值检查

**文件：** `itsm-backend/service/workflow_integration_service.go:70-100`

**严重程度：** 🟢 低危

**问题描述：**
访问prob.RootCause等字段前未检查是否为空字符串。

---

## 修复优先级建议

**立即修复（本周）：**
- Bug #1: 审核工作流事务问题
- Bug #4: 权限验证缺失
- Bug #5: 租户隔离验证

**短期修复（2周内）：**
- Bug #2: Context传递问题
- Bug #3: Goroutine管理
- Bug #6: N+1查询优化

**中期修复（1个月内）：**
- Bug #7, #8: 租户验证完善
- Bug #9-12: 低危问题

---

## 测试建议

### 单元测试补充
- 审核工作流并发测试
- 租户隔离边界测试
- Context传递测试

### 集成测试
- 多租户场景测试
- 并发审核测试
- SLA监控性能测试

---

## 总结

本次审查发现了12个潜在bug，其中3个高危bug需要立即修复。主要问题集中在：
1. **事务完整性** - 审核流程缺少事务保护
2. **并发安全** - Goroutine和Context管理不当
3. **租户隔离** - 部分服务缺少租户验证
4. **权限控制** - 审核权限验证缺失

建议按照优先级逐步修复，并补充相应的测试用例。
