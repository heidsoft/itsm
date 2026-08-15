# Workflow / CMDB Invariants

> **Status**: current. 自 v1.5 起强制。
> **迁移来源**：[`docs/review/architecture-review-2026-06-14.md`](../review/architecture-review-2026-06-14.md) 第 2、3 节（已识别但尚未迁移）。
> **维护人**：Workflow 域 owner / CMDB 域 owner。

本文档把所有"代码评审 / 故障复盘中反复出现、并且当前架构仍然必须满足"的不变量集中维护。新增评审报告若再次发现同类问题，应先更新本文件，再讨论 PR。

---

## 1. BPMN 工作流不变量

### 1.1 表达式求值必须采用拒绝策略（fail-closed）

`evaluateCondition` / `evaluateExpression` 在解析或求值失败时，必须返回 `false` 并写入审计日志；**禁止**默认返回 `true`。

**理由**：`bpmn_process_engine.go:552`（2026-06-14 评审 P0）暴露网关可被恶意构造表达式绕过。

**回归测试**：

- `handlers/bpmn/expression_engine_test.go` 中存在 `TestEvaluateCondition_InvalidExpression_ReturnsFalse`
- `handlers/bpmn/expression_engine_test.go` 中存在 `TestEvaluateCondition_Panic_ReturnsFalse`

**失败即视为 P0 阻断**。

### 1.2 变量合并必须是原子的

`CompleteTask`、`SetVariable` 等写路径必须：

1. 在 Ent 事务内完成"读取 instance.Variables → 合并 → 写回"；
2. 或者通过 `process_instance.version` 乐观锁拒绝并发覆盖。

**禁止**：

```go
// 反例：非原子读写
vars := instance.Variables
vars[k] = v
client.ProcessInstance.UpdateOne(instance).SetVariables(vars).Save(ctx)
```

### 1.3 ServiceTask 必须真正执行 Handler

`handleElement` 在遇到 ServiceTask 时：

1. 必须调用已注册的 `CallbackRegistry`；
2. 失败必须支持重试 / 错误边界事件；
3. 不允许 `fmt.Printf` 后直接推进流程。

### 1.4 网关支持范围

必须实现并测试：

- ExclusiveGateway（已有）
- ParallelGateway（v1.5 跟踪项）
- InclusiveGateway（v1.5 跟踪项）

未实现的网关类型在审批模板里禁止使用；模板 import 时必须 fail-closed。

### 1.5 任务分配

- **禁止**继续使用中文关键词匹配的 `getDefaultAssigntee`；
- 分配策略必须基于 BPMN 定义的 `candidateGroups` / `candidateUsers` 或显式"任务分配规则表"配置。

### 1.6 `CompleteTaskByID` 必须复用同一引擎实例

禁止每次 `NewCustomProcessEngine`，否则已注册的函数、handler 都会丢失。当前实现在 `bpmn_process_engine.go:1280`。

---

## 2. CMDB 不变量

### 2.1 跨租户隔离强制（最高优先级）

下列查询/写入必须强制过滤 `tenant_id`，缺失即视为 P0 安全漏洞：

| 操作 | 位置 | 备注 |
|---|---|---|
| `ListRelationships` | `service/cmdb_service.go` | 评审报告 P0；2026-06 后已修，但回归测试必须保留 |
| `GetCITopology` | `service/cmdb_service.go` | depth 必须有限 |
| `ListCIs` | `service/cmdb_service.go` | 增加 tenant 谓词 |
| `CreateRelationship` | `service/cmdb_service.go` | 校验两端 CI 的 tenant 一致 |
| `DeleteRelationship` | `service/cmdb_service.go` | 同上 |

回归测试要求：

- 跨租户 A 创建 CI、CI 关系；
- 租户 B 调用 `ListRelationships`、`GetCITopology` 必须返回空 / 404，不得泄露；
- 跨租户 `CreateRelationship` 必须返回 403 或 404，且不写入。

### 2.2 拓扑查询深度上限

`GetCITopology` 接受的 `depth` 参数必须硬上限（建议 5），超出时返回 400 而非静默截断。BFS 实现已限制 depth=3，新代码必须沿用 BFS，禁止递归实现。

### 2.3 关系两端租户一致性

`CreateRelationship` 必须验证 `sourceCi.TenantID == targetCi.TenantID == caller.TenantID`。任意一项不一致即拒绝。

### 2.4 DTO 与 schema 分离

请求/响应结构必须位于 `dto/`，禁止在 `service/cmdb_service.go` 内联定义。CI list 返回必须使用 `*ConfigurationItemListResponse` 五元组（见 `static-analysis-gates.md` §5.5）。

### 2.5 影响分析必须有界

影响分析（impact-analysis）必须：

1. 设置递归深度硬上限（建议 ≤ 8）；
2. 遇到环必须早返回，不得无限循环；
3. 跨租户 CI 必须被排除；
4. 单次返回节点数必须分页（默认 200）。

回归测试位于 `handlers/cmdb/topology_invariant_test.go`，覆盖：自环 / 50 层链 / 环早返回 / 跨租户隔离。

---

## 3. 引用与变更

- 新增不变量必须在此文件登记，并在对应测试文件中添加 `TestInvariant_<Name>`。
- 修改不变量必须链接到对应 PR/issue，并标注"v?.? 起变更"。
- 与 [`domain-ownership.md`](./domain-ownership.md) 冲突时，以本文件为准。
