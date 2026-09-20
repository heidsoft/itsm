# Raw SQL 治理清单（v1.0 GA 准入）

> 本文档跟踪所有非 Ent ORM 的直接 SQL 调用，作为 `P0-1 raw SQL 治理` 任务的源数据。
>
> **基线扫描时间**：2026-06-27
> **扫描命令**：`grep -rEn "ExecContext|QueryContext" itsm-backend/controller/ itsm-backend/service/ | grep -v _test.go`
> **基线调用点总数**：28
> **GA 准入阈值**：高风险 5 模块 ≤ 0，其他文件 ≤ 旧值 × 30%（即 ≤ 8）

## 1. 文件级汇总

| 文件 | 调用点 | 风险等级 | 整改要求 |
|------|--------|----------|----------|
| `service/problem_investigation_service.go` | 13 | 高 | 必须全部替换为 Ent ORM |
| `service/vector_store.go` | 5 | 低 | pgvector 扩展必须 raw SQL，允许保留 |
| `service/change_approval_service.go` | 4 | 高 | 必须全部替换为 Ent ORM |
| `service/ticket_type_service.go` | 3 | 中 | 复杂 JOIN 优先，必要时保留并参数化 |
| `service/ai_telemetry.go` | 2 | 中 | 报表类查询，可保留但需参数化 |
| `controller/ticket_workflow_controller.go` | 1 | 中 | 下沉到 service 层后整改 |
| **合计** | **28** | — | — |

## 2. 高风险 5 模块（必须 v1.0 GA 前归零）

> 注意：第 5 个高风险模块在 controller 层，扫描发现 1 处 raw SQL，需要下移到 service 层后整改。
> 原计划的 5 个高风险 controller 文件（approval/bpmn_workflow/incident/cmdb/sla_policy）经复查均**未直接使用 raw SQL**，而是走 Ent 或 HTTP query params。
> 因此高风险模块从 controller 改为 service，调整为以下 5 个：

| 模块 | 文件 | 调用点 | 治理负责人 |
|------|------|--------|------------|
| 变更审批 | `service/change_approval_service.go` | 4 | 后端 agent |
| 问题调查 | `service/problem_investigation_service.go` | 13 | 后端 agent |
| 工单类型 | `service/ticket_type_service.go` | 3 | 后端 agent |
| AI 遥测 | `service/ai_telemetry.go` | 2 | 后端 agent |
| Ticket Workflow | `controller/ticket_workflow_controller.go` | 1 | 后端 agent |

## 3. 调用点明细

### 3.1 `service/problem_investigation_service.go`（13 处）

| 行号 | 调用类型 | SQL 描述 | 整改方案 |
|------|----------|----------|----------|
| 129 | ExecContext | `UPDATE problems SET status = 'in_progress' ...` | 改 `client.Problem.UpdateOneID(...).SetStatus(...)` |
| 212 | ExecContext | 动态 UPDATE | 改 Ent + predicate |
| 219 | ExecContext | `UPDATE problems SET status = 'resolved' ...` | 改 Ent Update |
| 342 | ExecContext | 动态 UPDATE | 改 Ent + predicate |
| 507 | QueryContext | RCA 列表查询 | 改 `client.RootCauseAnalysis.Query()` |
| 536 | QueryContext | RCA 关联 | 改 Ent + With 关联 |
| 565 | QueryContext | 多表 JOIN | 评估 Ent 是否支持，必要时保留但加 `internal/sqlx` 包装 |
| 667 | ExecContext | 动态 UPDATE | 改 Ent Update |
| 691 | ExecContext | `DELETE FROM problem_root_cause_analyses` | 改 `client.RootCauseAnalysis.DeleteOneID()` |
| 775 | ExecContext | 动态 UPDATE | 改 Ent Update |
| 799 | ExecContext | `DELETE FROM problem_solutions` | 改 `client.ProblemSolution.DeleteOneID()` |
| 843 | ExecContext | INSERT | 改 `client.X.Create()` |
| 854 | ExecContext | INSERT | 改 `client.X.Create()` |

### 3.2 `service/vector_store.go`（5 处，允许保留）

| 行号 | 调用类型 | SQL 描述 | 说明 |
|------|----------|----------|------|
| 26 | ExecContext | `CREATE EXTENSION IF NOT EXISTS vector` | pgvector 扩展必须 raw |
| 31 | ExecContext | CREATE TABLE 嵌入表 | pgvector DDL 必须 raw |
| 62 | ExecContext | CREATE INDEX 嵌入索引 | pgvector 索引必须 raw |
| 84 | QueryContext | 向量相似度查询 | pgvector 操作必须 raw |
| 106 | QueryContext | 嵌入元数据查询 | 评估 Ent 是否支持 |

### 3.3 `service/change_approval_service.go`（4 处）

| 行号 | 调用类型 | SQL 描述 | 整改方案 |
|------|----------|----------|----------|
| 158 | ExecContext | `DELETE FROM change_approval_chains` | 改 `client.ChangeApprovalChain.DeleteOneID()` |
| 176 | ExecContext | 动态 INSERT | 改 `client.X.Create()` |
| 216 | QueryContext | 链查询 | 改 `client.ChangeApprovalChain.Query().WithUser()` |
| 276 | QueryContext | 历史查询 | 改 `client.X.Query()` |

### 3.4 `service/ticket_type_service.go`（3 处）

| 行号 | 调用类型 | SQL 描述 | 整改方案 |
|------|----------|----------|----------|
| 246 | ExecContext | 动态 UPDATE | 改 Ent Update |
| 391 | QueryContext | 复杂查询 | 改 `client.TicketType.Query().Where(...)` |
| 442 | ExecContext | DELETE | 改 `client.TicketType.DeleteOneID()` |

### 3.5 `service/ai_telemetry.go`（2 处）

| 行号 | 调用类型 | SQL 描述 | 整改方案 |
|------|----------|----------|----------|
| 45 | ExecContext | INSERT 遥测 | 改 `client.AITelemetry.Create()` |
| 94 | QueryContext | 统计查询 | 改 `client.AITelemetry.Query().Aggregate()` |

### 3.6 `controller/ticket_workflow_controller.go`（1 处）

| 行号 | 调用类型 | SQL 描述 | 整改方案 |
|------|----------|----------|----------|
| 430 | QueryContext | 工单工作流查询 | 下沉到 `service/ticket_workflow_service.go` 后改 Ent |

## 4. 整改策略

### 4.1 优先级

1. **P0**（高风险 5 模块）：v1.0 GA 前必须归零
2. **P1**（pgvector 5 处）：v1.0 GA 允许保留，但需 `// ALLOWED: pgvector` 注释 + 单元测试
3. **P2**（其他）：v1.1 处理

### 4.2 整改模板

```go
// BEFORE
rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM tickets WHERE tenant_id = $1", tenantID)

// AFTER (Ent ORM)
rows, err := s.client.Ticket.Query().
    Where(ticket.TenantIDEQ(tenantID)).
    Select(ticket.FieldID, ticket.FieldName).
    All(ctx)
```

### 4.3 不能替换时的包装

```go
// internal/sqlx/wrapper.go
// SafeQuery 执行参数化 raw SQL，自动注入 tenant_id 过滤
// ALLOWED 只用于：pgvector、复杂报表、Ent 不支持的操作
func SafeQuery(ctx context.Context, db *sql.DB, sql string, args ...any) (*sql.Rows, error) {
    // 1. 检查 SQL 含 tenant_id 过滤
    // 2. 参数化校验
    // 3. 慢查询日志
    // 4. 行级权限检查
}
```

## 5. 验收标准

- [ ] 高风险 5 模块 raw SQL 调用点 = 0
- [ ] `grep -rEn "ExecContext|QueryContext" itsm-backend/{controller,service} | grep -v _test.go | wc -l` ≤ 8
- [ ] `go vet ./...` 全绿
- [ ] `go test ./...` 全绿
- [ ] 保留的 raw SQL 全部带 `// ALLOWED: <reason>` 注释

## 6. 实施 PR 拆分

| PR | 范围 | 估时 |
|----|------|------|
| `refactor(sql): problem_investigation_service 改 Ent` | §3.1 全部 13 处 | 2d |
| `refactor(sql): change_approval_service 改 Ent` | §3.3 全部 4 处 | 0.5d |
| `refactor(sql): ticket_type_service 改 Ent` | §3.4 全部 3 处 | 0.5d |
| `refactor(sql): ai_telemetry 改 Ent` | §3.5 全部 2 处 | 0.5d |
| `refactor(ticket): ticket_workflow_controller 下沉 SQL` | §3.6 | 0.5d |
| `chore(sqlx): vector_store 添加 ALLOWED 注释` | §3.2 | 0.2d |

---

**文档版本**：v1.0-rc.1
**下次刷新**：每个 PR 合入后刷新调用点计数
