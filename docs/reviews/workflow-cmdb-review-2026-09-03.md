# 流程与 CMDB 功能评审（2026-09-03）

## 范围与结论

本次检查仓库生产接线、领域实现与自动化测试；没有连接生产环境，不能据此判定部署健康或完整 ITIL 业务验收通过。工作区原有的分页、工单、事件、标准变更与 bootstrap 修改保留。

| 能力 | 代码状态与证据 | 判断 |
| --- | --- | --- |
| BPMN 定义、实例、任务、审批决策 | `handlers/bpmn/handler.go`、`workflow.go` 注册真实入口，bootstrap 注入 `CustomProcessEngine` | 已接线；本次修复审批 ID 解释不一致 |
| CI 类型、实例、关系、拓扑、影响分析 | `router/cmdb_routes.go` 注册，使用真实 Ent 服务 | 已接线；本次修复错误映射和影响分析关联隔离 |
| CMDB 云发现 | `GetDiscoveryCapability` 检查 adapter、secret resolver、worker、租户云账号 | 存在 disabled/unready/unconfigured/ready 分级；部署是否 ready 尚未验证 |
| 流程与 CMDB 大规模运行 | 深度限制、BFS、租户条件已有；仍存在以下缺口 | 不能宣称已达到企业生产完整验收标准 |

## 本次修复

1. **审批任务身份一致性**：`POST /api/v1/bpmn/tasks/:id/decisions` 原先先按 BPMN 字符串 ID 读取配置，却按数字数据库 ID 完成。当另一任务的 `task_id` 恰好等于目标主键时，会校验错误节点的拒绝意见要求。现在与任务详情和完成接口一致：数字使用主键，非数字使用 BPMN task ID。真实引擎/SQLite 回归覆盖冲突与跨租户拒绝。
2. **CMDB 错误语义**：拓扑、影响分析对不存在和租户不可见的根 CI 均返回 HTTP 404 / code 4004；数据库故障保持 HTTP 500 / code 5001，但不再把 SQL/Ent 原始错误拼入响应。影响分析保留错误链，支持类型化判断。
3. **影响分析关联隔离**：关联工单、事件需要独立租户过滤；返回关系边只能连接已加载的本租户节点。回归数据包含正常关联和历史/导入可能遗留的跨租户异常关联。

真实调用链：

- Router → `bpmn.Handler.RegisterRoutes` → `WorkflowHandler.SubmitTaskDecision` → `TaskService.GetTaskByID/GetTask`、`CompleteTaskByID/CompleteTask` → Ent / 流程引擎。
- Router → `cmdb.Handler` → `cmdb.Service` 转发 → `cmdb.ProductionService.GetCITopology/GetCIImpactAnalysis` → `service.CIRelationshipService` → Ent 租户查询。

## 后续优先事项

- **P1：图查询规模预算**。生产拓扑与影响分析限制最大深度 10，但每层关系使用 `All`，没有总节点/边预算或分页。高扇出图仍可能消耗大量资源。应设计明确的截断/分页契约与压力测试，避免静默省略影响范围。
- **P1：流程引擎实例复用**。`TaskService()` 创建新服务对象，`CompleteTaskByID` 再新建引擎，与不变量文档要求的复用不一致。自定义表达式/回调注入状态的保留需要独立回归验证和统一引擎生命周期设计。
- **P1：测试迁移缺口**。`service/auth_service_ext_test.go:27,43` 引用不存在的 `AuthService`，当前实现为 `handlers/auth.Service`；阻塞整个 service 测试包编译。需迁移测试及其私有字段 fixture，不能简单跳过或添加假类型。
- **P2：不变量文档漂移**。`docs/architecture/workflow-cmdb-invariants.md` 声称拓扑超限返回 400、建议深度 5，实际 Handler 接受非法深度后回退、service 静默限制到 10；部分列出的测试路径不存在。应在明确 API 行为后同步文档和契约测试。

## 验证

新增测试通过真实 Handler 和数据库，不依赖孤立 helper 来证明租户隔离。

- 修复前回归：审批 ID 冲突、CMDB 404/原始错误泄漏测试失败；关联隔离测试返回了 tenant 2 工单标题，确认缺陷。
- `cd itsm-frontend && npm run test:unit -- --runTestsByPath src/lib/__tests__/api-contract.test.ts`：通过，3 项断言。
- `cd itsm-frontend && npm run type-check`：通过。
- `cd itsm-backend && go test ./service -run 'Test.*(BPMN|ApprovalDecision|CIImpactAnalysis|CITopology|CIRelationship)' -count=1`：无法编译，原因是上述 `AuthService` 存量引用。
- `cd itsm-backend && go test ./...`：全量验证发现 `dto.TestTicketTypeListResponseUsesCamelCaseJSON` 失败，响应包含禁止的 `types` 别名。该字段及自定义 MarshalJSON 已在本次开始时的工作区改动中存在，本次未修改该文件。
- `cd itsm-backend && go test ./handlers/cmdb ./handlers/bpmn -count=1`：最终通过，包含新增生产 Handler/真实数据库回归。
- `git diff --check`：通过；已审查本次最终实现和测试 diff。
- 全量执行另发现 `handlers/standard_change` 失败：`TestGetStandardChange_NotFound`、`TestUpdateStandardChange_NotFound`（404 被映射为 500）；`TestDeleteStandardChange_SoftDelete`（删除后查不到记录）；`TestDeleteStandardChange_NotFound`（404 被返回为 200）；`TestGetCategories_Distinct`（空结果类型断言 panic）。这些测试不经过本次修改的 BPMN/CI relationship 调用链，且标准变更文件在任务开始时已有未提交修改，本次未触碰。

- 全量执行的 `internal/contracts.TestApprovalWritePathInventoryHasNoDrift` 还发现审批写入清单引用已不存在的 `service/service_request_service.go`。属于领域迁移后清单未同步，本次未修改该清单或服务请求实现。

没有执行上线、迁移或生产数据写入。
