# ITSM 商用就绪架构设计文档

**文档编号**: ITSM-ARCH-2026-001
**版本**: v1.0
**日期**: 2026-06-14
**作者**: 高见远（Gao），架构师
**状态**: Draft

---

## 1. 待确认问题决策（Q-001 ~ Q-008）

### Q-001: ACL 脚本引擎的实现深度

**决策：采用有限表达式语法，不引入完整脚本语言沙箱**

| 维度 | 说明 |
|------|------|
| 决策 | 实现基于 ANTLR/手写递归下降的有限表达式引擎，支持比较运算（==, !=, >, <, >=, <=）、逻辑运算（&&, \|\|, !）、上下文变量引用（ctx.user_id, ctx.tenant_id, ctx.resource_id） |
| 理由 | 1）完整 JS/Lua 沙箱引入 CGO 或重量级依赖，攻击面大；2）ITSM ACL 场景仅需"属性比较+逻辑组合"，如 `ctx.user_id == ctx.resource.owner_id`；3）有限表达式可静态分析、审计、做权限边界检查；4）与现有 `ExpressionEngine`（`service/expression_engine.go`）风格一致，可复用其变量解析和函数注册机制 |
| 风险 | 未来如果需要更复杂的 ACL 逻辑（如循环、函数调用），需要扩展表达式引擎或引入沙箱 |

### Q-002: Ent Schema 覆盖情况

**决策：需要新建 `TicketApproval` 和 `TicketWorkflowRecord` 两个 Ent Schema，并修改 `CIRelationship` 添加 `tenant_id` 字段**

| 表名 | Ent Schema 状态 | 说明 |
|------|----------------|------|
| `tickets` | 已有 `ent/schema/ticket.go` | 可直接使用 |
| `ticket_approvals` | **缺失** | 现有 `approval_record.go` Schema 结构与 `ticket_approvals` 表不同（字段名、关联关系不一致），需要新建 `TicketApproval` Schema 对齐原生 SQL 使用的表结构 |
| `ticket_workflow_records` | **缺失** | 需要新建 `TicketWorkflowRecord` Schema |
| `ci_relationships` | 已有但**缺 tenant_id** | `ent/schema/ci_relationship.go` 没有 `tenant_id` 字段，需新增 |
| `departments` | 已有 | 完整 |
| `tags` | 已有 | 完整 |
| `teams` | 已有 | 完整 |
| `roles` | 已有，但**缺 is_active/status 字段** | `ent/schema/role.go` 没有 `is_active` 或 `status` 字段 |

**迁移策略**：
1. 先创建缺失的 Ent Schema 并执行 `go generate ./ent`
2. `TicketWorkflowService` 迁移时，逐步将 `rawDB.ExecContext` 替换为 Ent 调用
3. `CIRelationship` 添加 `tenant_id` 字段需要数据库迁移脚本设置默认值

### Q-003: 向量库选型

**决策：继续使用 pgvector（PostgreSQL 扩展），不引入独立向量库**

| 维度 | 说明 |
|------|------|
| 决策 | 保持当前 pgvector 方案，优化 `VectorStore` 初始化逻辑 |
| 理由 | 1）`service/vector_store.go` 已基于 pgvector 实现 Upsert/SearchTopK，逻辑完整；2）`service/embed_pipeline.go` 已实现知识库/事件向量化管道；3）当前问题是**未初始化**（vectors 表可能不存在）而非选型问题；4）引入 Milvus/Qdrant 等独立向量库增加运维复杂度，对 ITSM 场景不值得 |
| 修复方案 | 1）启动时检测 pgvector 扩展和 vectors 表是否存在，不存在则自动创建；2）`EmbeddingPipeline` 增加定时/触发式索引同步；3）向量库不可用时 RAG 接口返回友好降级信息 |

### Q-004: 组织架构模块现状

**决策：Schema 已有，路由已有但路径不匹配，Service 层部分缺失**

| 组件 | 状态 | 说明 |
|------|------|------|
| Ent Schema | 已有 | `department.go`、`tag.go`、`team.go` 结构完整，含 tenant_id |
| 路由 | 已有 | `/api/v1/org/departments/*`、`/api/v1/org/teams`、`/api/v1/system/tags` 已注册 |
| 兼容路由 | 已有 | `/api/v1/departments`、`/api/v1/teams`、`/api/v1/tags` 作为只读别名已注册（router.go:1001-1006） |
| Service | 通过 `CommonHandler` 实现 | `ListDepartments`、`ListTeams`、`ListTags`、`GetDepartmentTree`、`CreateDepartment` 等 |
| 问题 | 前端请求的 CRUD 路径与后端不完全对齐 | 需要确认前端实际请求路径 |

**结论**：组织架构 API 404 的问题很可能是前端请求路径与后端路由不匹配，而非功能缺失。需排查前端请求的实际 URL，可能只需调整前端或补充路由别名。

### Q-005: BPMN 条件评估默认改为 return false 的影响

**决策：改为 return false 是安全的，但需要排查现有流程定义**

| 维度 | 说明 |
|------|------|
| 风险评估 | 1）`CustomProcessEngine.evaluateCondition`（line 530-556）中，无条件分支默认 `return true`（正确行为）；2）条件评估失败时当前 `return true`，改为 `return false` 后，**无效表达式的流程将无法通过网关**——这是期望的安全行为；3）`GatewayEngine.evaluateCondition`（line 315-356）失败时 `return false`（已正确），但 `evaluateExclusiveGatewayConditions` 在无匹配时选择第一个输出流（不安全） |
| 修复策略 | 1）`evaluateCondition` 失败时改为 `return false` 并记录 Error 日志；2）排他网关无条件匹配时返回错误而非选择第一个输出流；3）排查所有已部署的 BPMN 流程定义，确认无依赖"评估失败=通过"的逻辑 |
| 回归测试 | 需对所有流程执行冒烟测试 |

### Q-006: ServiceTask 异步执行

**决策：当前阶段仅支持同步执行，架构预留异步扩展能力**

| 维度 | 说明 |
|------|------|
| 决策 | FUNC-001 修复时先实现同步调用 CallbackRegistry，不引入异步队列 |
| 理由 | 1）`CallbackRegistry`（`service/bpmn/bpmn_callback_registry.go`）的 `HandleCallback` 是同步实现；2）现有 Handler（Ticket/Incident/Change/Approval/Notification/Webhook）均为同步执行；3）异步需要引入消息队列（如 Redis Streams），增加架构复杂度；4）可后续通过 `ServiceTask` 配置 `async: true` 标志来扩展 |
| 扩展预留 | 在 `handleElement` 中调用 `CallbackRegistry.HandleCallback` 时，如果 Handler 返回特定错误码（如 `ErrAsyncProcessing`），将任务状态标记为 `processing`，等待回调完成 |

### Q-007: 中文关键词迁移

**决策：保留中文关键词匹配作为 deprecated 兼容，新增基于 BPMN 属性的分配策略**

| 维度 | 说明 |
|------|------|
| 决策 | 1）优先使用 BPMN 定义中的 `assignee`/`candidateUsers`/`candidateGroups`；2）其次使用基于角色的分配规则（配置在数据库中）；3）中文关键词匹配标记为 `// Deprecated:` 并打印 Warn 日志 |
| 理由 | 1）现有 `*_cn.bpmn` 流程文件（如 `release_approval_flow_cn.bpmn`、`problem_management_flow_cn.bpmn`、`incident_emergency_flow_cn.bpmn`）中的 UserTask 已定义 `name` 属性为中文，但 `createUserTask` 方法已支持 `task.Assignee`/`task.CandidateUsers`/`task.CandidateGroups`（line 402-441）；2）直接删除中文匹配可能导致已有流程无法分配任务；3）渐进式迁移更安全 |
| 迁移计划 | 1）V1：添加 BPMN 属性优先 + 中文兼容；2）V2：为现有 `_cn.bpmn` 文件补充 `assignee`/`candidateGroups` 属性；3）V3：移除中文关键词匹配 |

### Q-008: security 角色权限修复

**决策：同时更新硬编码映射和数据库迁移脚本**

| 维度 | 说明 |
|------|------|
| 决策 | 两处都要修复：1）更新 `middleware/rbac.go` 中 `RolePermissions["security"]` 的硬编码权限，增加 `ticket:read`、`notification:read`、`notification:write`、`knowledge:read`；2）新增数据库迁移脚本更新 `role_permissions` 表中 security 角色（id=106）的权限配置 |
| 理由 | 1）`migrations/20260501_enable_rbac_from_db.sql` 中 security 角色权限（line 448-489）缺少 ticket、notification、knowledge 权限；2）`rbac.go` 硬编码同样缺少这些权限；3）仅修一处会导致数据库权限模式下和硬编码回退模式下行为不一致 |
| 验证 | security 角色用户可查看分配给自己的工单、通知和知识库 |

---

## 2. 实现方案 + 框架选型

### SEC-001: CMDB 关系列表跨租户数据泄露

**修改文件**:
- `service/cmdb_service.go` — `ListRelationships`、`GetCITopology`
- `ent/schema/ci_relationship.go` — 添加 `tenant_id` 字段

**核心变更思路**:
1. `CIRelationship` Schema 添加 `tenant_id` 字段（Positive, Optional，兼容存量数据）
2. 数据库迁移脚本为存量数据填充 tenant_id（从 source_ci 或 target_ci 的 tenant_id 继承）
3. `ListRelationships` 添加 `cirelationship.TenantID(tenantID)` 过滤条件
4. `GetCITopology` 递归查询添加 tenant_id 过滤
5. `CreateRelationship` 设置 tenant_id

**依赖**: 需要先修改 Ent Schema 并执行 `go generate`

### SEC-002: BPMN 条件评估失败默认通过

**修改文件**:
- `service/bpmn_process_engine.go` — `evaluateCondition` 方法
- `service/bpmn_gateway_engine.go` — `evaluateCondition` 和 `evaluateExclusiveGatewayConditions`

**核心变更思路**:
1. `CustomProcessEngine.evaluateCondition` 评估失败时 `return false`，日志级别从 Warn 改为 Error
2. `GatewayEngine.evaluateCondition` 已返回 false，但 `evaluateExclusiveGatewayConditions` 无条件匹配时选第一个输出流（line 293-296），改为返回错误
3. 添加单元测试验证无效表达式不会导致网关放行

### SEC-003: ACL 脚本评估空实现

**修改文件**:
- `middleware/smart_permission.go` — `EvaluateACLScript` 函数
- 新增 `middleware/acl_expression_engine.go` — 表达式解析执行引擎
- 新增 `middleware/acl_expression_engine_test.go` — 测试

**核心变更思路**:
1. 实现有限表达式语法解析器，支持：比较运算（==, !=, >, <, >=, <=）、逻辑运算（&&, \|\|, !）、变量引用（ctx.user_id, ctx.tenant_id, ctx.role, ctx.resource_id, ctx.resource_type）
2. 表达式示例：`ctx.user_id == ctx.resource_id`、`ctx.role == "admin" && ctx.tenant_id == 1`
3. 解析失败或执行异常时 `return false`
4. 空字符串仍 `return true`

**依赖包**: 无新增，基于手写递归下降解析器

### SEC-004: CI 关系创建无 tenantID 校验

**修改文件**:
- `service/cmdb_service.go` — `CreateRelationship`
- `ent/schema/ci_relationship.go` — 添加 `tenant_id`

**核心变更思路**:
1. `CreateRelationshipRequest` 增加 `TenantID` 字段
2. 创建关系前查询 source CI 和 target CI 是否均属于当前 tenantID
3. 校验失败返回 403
4. 创建关系时设置 tenant_id

### SEC-005: checkRolePermissionFromDB 未真正查 DB

**修改文件**:
- `middleware/smart_permission.go` — `checkRolePermissionFromDB`

**核心变更思路**:
1. 函数签名增加 `client *ent.Client` 参数（通过 context 或全局传递）
2. 使用 `loadRolePermissionsFromDB`（`middleware/rbac.go`）的查询逻辑，从 `role_permissions` 表查询
3. 查询结果与硬编码权限按 `PermissionConfig.Mode` 合并
4. 当前 `PermissionConfig.Mode` 为 `PermissionConfigModeDBOnly`，仅使用数据库权限

### ARCH-001: TicketWorkflowService 绕过 Ent ORM

**修改文件**:
- `service/ticket_workflow_service.go` — 全面重构
- `ent/schema/ticket_approval.go` — 新增
- `ent/schema/ticket_workflow_record.go` — 新增
- `dto/ticket_workflow_dto.go` — 可能需要调整

**核心变更思路**:
1. 新建 `TicketApproval` Ent Schema 对齐 `ticket_approvals` 表结构
2. 新建 `TicketWorkflowRecord` Ent Schema 对齐 `ticket_workflow_records` 表结构
3. 逐步替换 `rawDB.ExecContext` 为 Ent 调用：
   - `getTicket` → `client.Ticket.Get`
   - `createWorkflowRecord` → `client.TicketWorkflowRecord.Create`
   - `ApproveTicket` 中的审批记录操作 → `client.TicketApproval` 查询/更新
4. 移除 `rawDB *sql.DB` 依赖
5. 执行 `go generate ./ent`

**依赖**: 必须先于 DATA-001 完成，因为事务保护需要基于 Ent 的 `client.Tx()`

### DATA-001: 工单审批流程缺少事务保护

**修改文件**:
- `service/ticket_workflow_service.go` — 与 ARCH-001 合并实现

**核心变更思路**:
1. 在 ARCH-001 完成后的 Ent 版本基础上，为每个写操作方法添加 `client.Tx()` 事务
2. `AcceptTicket`、`RejectTicket`、`ApproveTicket`、`ResolveTicket`、`CloseTicket`、`ReopenTicket` 均包裹事务
3. 事务失败时 `tx.Rollback()` 并返回明确错误
4. 编写测试验证回滚场景

**依赖**: ARCH-001（必须先完成 Ent 迁移）

### DATA-002: BPMN 变量合并非原子操作

**修改文件**:
- `service/bpmn_process_engine.go` — `CompleteTask` 方法

**核心变更思路**:
1. 使用 Ent 事务包裹变量读取→合并→写入操作
2. 在事务内使用 `FOR UPDATE` 行锁（Ent 支持 `WithTx` + 原生 SQL hint，或通过 `client.Tx()` + 乐观锁版本号字段）
3. 乐观锁方案：为 `ProcessInstance` 添加 `version` 字段，更新时检查版本号
4. 并发写入时若版本不匹配则重试

### FUNC-001: ServiceTask 仅打印日志

**修改文件**:
- `service/bpmn_process_engine.go` — `handleElement` 方法

**核心变更思路**:
1. `handleElement` 遇到 ServiceTask 时，调用 `ProcessCallbackService.HandleCallback`
2. 根据 `serviceTask.ID`（任务定义 ID）路由到对应 Handler
3. 构建 `CallbackRequest`，填充 `ProcessInstanceID`、`ActivityType`（从 serviceTask 的 extensionElements 或 ID 推断）、`Result.OutputVars`
4. Handler 执行失败时，将流程实例状态设为 `error` 而非继续推进
5. 移除 `fmt.Printf`

**依赖**: `ProcessCallbackService` 和 `CallbackRegistry` 已实现完整，有 Ticket/Incident/Change/Approval/Notification/Webhook Handler

### FUNC-002: getDefaultAssigntee 基于中文关键词

**修改文件**:
- `service/bpmn_process_engine.go` — `getDefaultAssigntee`、`createUserTask`

**核心变更思路**:
1. `createUserTask` 已正确处理 BPMN 属性（assignee/candidateUsers/candidateGroups）和流程变量（requester_id/triggered_by/assignee_id）
2. `getDefaultAssigntee` 改为先查询数据库中的任务分配规则（`ticket_assignment_rules` 表），再按角色匹配
3. 中文关键词匹配标记为 `// Deprecated:`，仅在上述策略都失败时作为兜底，并打印 Warn 日志
4. 新增基于 `task.TaskDefinitionKey` 的分配策略（而非 taskName）

### CODE-001: RBAC 重复代码

**修改文件**:
- `middleware/rbac.go` — `loadRolePermissionsFromDB`、`loadPermissionsFromDB`

**核心变更思路**:
1. 提取公共方法 `loadPermissionsFromDBCommon(client, roleName, tenantID, tableName string) []Permission`
2. `loadRolePermissionsFromDB` 调用公共方法
3. `loadPermissionsFromDB` 调用公共方法 + fallback 逻辑
4. 考虑合并两个函数为一个（`loadPermissionsFromDB` 已包含 fallback 到 `loadRolePermissionsFromDB`）

### CODE-002: 日志规范不一致

**修改文件**:
- 全局替换 `fmt.Printf` → `zap.S().Infof`/`Warnf`/`Errorf`
- 全局替换 `log.Printf` → `zap` 调用
- 重点文件：`service/bpmn_process_engine.go`、`service/bpmn_variable_service.go`、`cmd/` 目录

### CODE-003: DTO 分离不一致

**修改文件**:
- `service/cmdb_service.go` — 将内联 Request 结构体迁移到 `dto/cmdb_dto.go`
- 统一规范文档

**核心变更思路**: CMDB 的 `CreateCIRequest`、`ListCIsRequest`、`CreateRelationshipRequest`、`UpdateRelationshipRequest` 迁移到 `dto` 包，原位置保留类型别名以兼容

### UX-001: 前端 filters 无 debounce

**修改文件**:
- 前端工单列表等页面组件

**核心变更思路**: 使用 lodash.debounce 或 ahooks 的 useDebounce，300ms 防抖

### UX-002: 角色状态字段后端缺失

**修改文件**:
- `ent/schema/role.go` — 添加 `is_active` 字段
- 数据库迁移脚本
- 角色 CRUD API

**核心变更思路**: Role Schema 添加 `is_active bool default true`，API 支持状态更新

### API-001: Dashboard API 404

**修改文件**:
- 检查路由注册和 Service 初始化

**核心变更思路**: 路由已注册（router.go:1187-1203），`DashboardHandler` 已有。问题可能是 `DashboardHandler` 未正确初始化注入到 `RouterConfig`。排查 `main.go` 中的初始化链路。

### API-002: AI 助手 RAG 检索 0 命中

**修改文件**:
- `service/vector_store.go` — 添加初始化检查
- `service/rag_service.go` — 降级处理
- `main.go` 或启动逻辑 — 初始化向量库

**核心变更思路**:
1. 启动时检测 pgvector 扩展，不存在则自动 `CREATE EXTENSION vector`
2. 检测 vectors 表，不存在则自动创建
3. 初始化完成后执行一次 `EmbeddingPipeline.RunOnce`
4. RAG 检索失败时返回友好提示而非 0 命中

### API-003: 组织架构 API 404

**修改文件**:
- 可能仅需前端修正路径

**核心变更思路**: 后端路由已注册 `/api/v1/departments`（GET）、`/api/v1/teams`（GET）、`/api/v1/tags`（GET）作为兼容别名（router.go:1001-1006）。问题可能是前端 POST 请求到这些路径但后端仅注册了 GET。需要补充 POST/PUT/DELETE 路由。

### API-004: AI 智能工单 API 404

**修改文件**:
- `handlers/ai/` 目录 — 路由注册
- `router/router.go` — 检查 AI 路由

**核心变更思路**: 当前 AI 路由已有 `/api/v1/ai/*`。检查 `POST /api/v1/ai/ticket/create` 和 `POST /api/v1/ai/ticket/{id}/summarize` 是否已注册。当前路由中有 `aiGroup.POST("/triage", ...)` 可用于工单创建，`aiGroup.GET("/tickets/:id/summary", ...)` 已有。

### API-005: 服务目录申请表单缺字段

**修改文件**:
- `dto/service_request_dto.go` 或相关 DTO
- 前端申请表单

**核心变更思路**: DTO 增加 `compliance_ack bool` 和 `expire_at *time.Time`，前端增加对应输入项

### API-006: security 角色权限过严

**修改文件**:
- `middleware/rbac.go` — `RolePermissions` 硬编码
- 新增数据库迁移脚本

**核心变更思路**: 详见 Q-008 决策

---

## 3. 文件列表及相对路径

### 3.1 后端修改文件

#### Ent Schema（新增/修改）
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `ent/schema/ticket_approval.go` | 新增 | 对齐 ticket_approvals 表 |
| `ent/schema/ticket_workflow_record.go` | 新增 | 对齐 ticket_workflow_records 表 |
| `ent/schema/ci_relationship.go` | 修改 | 添加 tenant_id 字段 |
| `ent/schema/role.go` | 修改 | 添加 is_active 字段 |

#### Service 层
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `service/cmdb_service.go` | 修改 | SEC-001/SEC-004: tenant_id 过滤和校验 |
| `service/ticket_workflow_service.go` | 重构 | ARCH-001/DATA-001: Ent 迁移 + 事务 |
| `service/bpmn_process_engine.go` | 修改 | SEC-002/FUNC-001/FUNC-002/DATA-002 |
| `service/bpmn_gateway_engine.go` | 修改 | SEC-002: 条件评估修复 |
| `service/vector_store.go` | 修改 | API-002: 初始化检查 |
| `service/embed_pipeline.go` | 修改 | API-002: 触发式同步 |
| `service/rag_service.go` | 修改 | API-002: 降级处理 |

#### Middleware 层
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `middleware/smart_permission.go` | 修改 | SEC-003/SEC-005: ACL 引擎和 DB 查询 |
| `middleware/acl_expression_engine.go` | 新增 | SEC-003: ACL 表达式解析引擎 |
| `middleware/acl_expression_engine_test.go` | 新增 | SEC-003: ACL 引擎测试 |
| `middleware/rbac.go` | 修改 | CODE-001/API-006: 去重+权限修复 |

#### DTO 层
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `dto/cmdb_dto.go` | 修改 | CODE-003: 迁移内联结构体 |
| `dto/service_request_dto.go` | 修改 | API-005: 增加字段 |
| `dto/ticket_approval_dto.go` | 可能修改 | ARCH-001: 对齐新 Schema |

#### Controller/Router 层
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `router/router.go` | 修改 | API-003/API-004: 补充路由 |
| `controller/cmdb_controller.go` | 修改 | SEC-004: 传递 tenantID |

#### 迁移脚本
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `migrations/20260614_security_role_permissions.sql` | 新增 | API-006: security 角色权限 |
| `migrations/20260614_ci_relationship_tenant_id.sql` | 新增 | SEC-001: CIRelationship 添加 tenant_id |
| `migrations/20260614_role_is_active.sql` | 新增 | UX-002: Role 添加 is_active |

#### 测试文件
| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `service/cmdb_service_test.go` | 新增 | SEC-001/SEC-004 测试 |
| `service/bpmn_process_engine_test.go` | 新增 | SEC-002/FUNC-001 测试 |
| `service/ticket_workflow_service_test.go` | 修改 | DATA-001 事务测试 |

### 3.2 前端修改文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| 工单列表页面组件 | 修改 | UX-001: 添加 debounce |
| 角色管理页面 | 修改 | UX-002: is_active 状态展示 |
| 服务目录申请表单 | 修改 | API-005: 增加 compliance_ack/expire_at |

---

## 4. 数据结构和接口变更

### 4.1 Ent Schema 变更

#### CIRelationship — 新增 tenant_id
```go
field.Int("tenant_id").
    Comment("租户ID").
    Positive().
    Optional(), // 兼容存量数据，迁移脚本填充后改为 Required
```

#### Role — 新增 is_active
```go
field.Bool("is_active").
    Comment("是否启用").
    Default(true),
```

#### TicketApproval — 新增 Schema
```go
type TicketApproval struct {
    ent.Schema
}

Fields:
- ticket_id      int     // 工单ID
- level          int     // 审批级别
- level_name     string  // 级别名称
- approver_id    int     // 审批人ID
- status         string  // pending/approved/rejected/cancelled
- action         string  // approve/reject/delegate
- comment        string  // 审批意见
- delegate_to_user_id  *int  // 委派目标
- tenant_id      int     // 租户ID
- created_at     time.Time
- updated_at     time.Time
- processed_at   *time.Time
```

#### TicketWorkflowRecord — 新增 Schema
```go
type TicketWorkflowRecord struct {
    ent.Schema
}

Fields:
- ticket_id      int
- action         string  // accept/reject/approve/resolve/close/reopen/forward/cc/withdraw/delegate
- from_status    *string
- to_status      *string
- operator_id    int
- from_user_id   *int
- to_user_id     *int
- comment        string
- reason         string
- metadata       map[string]interface{} // JSON
- tenant_id      int
- created_at     time.Time
```

### 4.2 DTO 变更

#### CreateRelationshipRequest — 增加 TenantID
```go
type CreateRelationshipRequest struct {
    SourceCIID  int     `json:"source_ci_id"`
    TargetCIID  int     `json:"target_ci_id"`
    Type        string  `json:"type"`
    Description *string `json:"description,omitempty"`
    TenantID    int     `json:"tenant_id"` // 新增
}
```

#### ServiceRequestDTO — 增加合规字段
```go
type CreateServiceRequestRequest struct {
    // ... 现有字段
    ComplianceAck bool       `json:"compliance_ack"` // 新增
    ExpireAt      *time.Time `json:"expire_at"`       // 新增
}
```

### 4.3 接口签名变更

#### EvaluateACLScript — 增加表达式引擎
```go
// 旧签名
func EvaluateACLScript(script string, ctx ACLScriptContext) bool

// 新签名（内部增加表达式解析）
func EvaluateACLScript(script string, ctx ACLScriptContext) bool
// 内部调用 aclExpressionEngine.Evaluate(script, variables)
```

#### checkRolePermissionFromDB — 增加 Ent Client
```go
// 旧签名
func checkRolePermissionFromDB(db DBQuerier, role, resource, action string, tenantID int) bool

// 新签名
func checkRolePermissionFromDB(db DBQuerier, client *ent.Client, role, resource, action string, tenantID int) bool
```

---

## 5. 任务列表

### 第一批（P0 安全漏洞）

| 编号 | 任务名称 | PRD编号 | 依赖 | 复杂度 | 实现步骤 | 涉及文件 |
|------|---------|---------|------|--------|---------|---------|
| T1-1 | CIRelationship Schema 添加 tenant_id | SEC-001 | 无 | M | 1) 修改 `ent/schema/ci_relationship.go` 添加 tenant_id 字段<br>2) 执行 `go generate ./ent`<br>3) 创建迁移脚本为存量数据填充 tenant_id | `ent/schema/ci_relationship.go`, `migrations/20260614_ci_relationship_tenant_id.sql` |
| T1-2 | CMDB ListRelationships 添加 tenantID 过滤 | SEC-001 | T1-1 | S | 1) `ListRelationships` 添加 `cirelationship.TenantID(tenantID)` 条件<br>2) `GetCITopology` 递归查询添加 tenant_id 过滤<br>3) 编写单元测试 | `service/cmdb_service.go`, `service/cmdb_service_test.go` |
| T1-3 | BPMN evaluateCondition 修复默认 false | SEC-002 | 无 | S | 1) `CustomProcessEngine.evaluateCondition` 评估失败改 `return false`，日志改 Error<br>2) `GatewayEngine.evaluateExclusiveGatewayConditions` 无条件匹配时返回错误<br>3) 编写测试 | `service/bpmn_process_engine.go`, `service/bpmn_gateway_engine.go` |
| T1-4 | ACL 表达式引擎实现 | SEC-003 | 无 | L | 1) 新建 `middleware/acl_expression_engine.go`，实现递归下降解析器<br>2) 支持 ==, !=, >, <, >=, <=, &&, \|\|, ! 和 ctx 变量引用<br>3) 修改 `EvaluateACLScript` 调用新引擎<br>4) 空字符串 return true，解析/执行异常 return false<br>5) 编写单元测试覆盖典型场景 | `middleware/acl_expression_engine.go`, `middleware/acl_expression_engine_test.go`, `middleware/smart_permission.go` |

### 第二批（P1 安全+数据一致性）

| 编号 | 任务名称 | PRD编号 | 依赖 | 复杂度 | 实现步骤 | 涉及文件 |
|------|---------|---------|------|--------|---------|---------|
| T2-1 | CreateRelationship 添加 tenantID 校验 | SEC-004 | T1-1 | S | 1) `CreateRelationshipRequest` 增加 TenantID 字段<br>2) 创建前查询 source/target CI 是否属于当前租户<br>3) 校验失败返回 403<br>4) 创建关系时设置 tenant_id | `service/cmdb_service.go`, `controller/cmdb_controller.go` |
| T2-2 | checkRolePermissionFromDB 实现数据库查询 | SEC-005 | 无 | M | 1) 函数签名增加 `*ent.Client` 参数<br>2) 使用 `loadRolePermissionsFromDB` 逻辑查询 role_permissions 表<br>3) 根据 `PermissionConfig.Mode` 合并策略处理<br>4) 编写测试 | `middleware/smart_permission.go` |
| T2-3 | 新建 TicketApproval 和 TicketWorkflowRecord Ent Schema | ARCH-001 | 无 | M | 1) 创建 `ent/schema/ticket_approval.go`<br>2) 创建 `ent/schema/ticket_workflow_record.go`<br>3) 执行 `go generate ./ent`<br>4) 验证 Schema 与现有表结构对齐 | `ent/schema/ticket_approval.go`, `ent/schema/ticket_workflow_record.go` |
| T2-4 | TicketWorkflowService Ent 迁移 | ARCH-001 | T2-3 | L | 1) 逐步替换 `rawDB.ExecContext` 为 Ent 调用<br>2) `getTicket` → `client.Ticket.Get`<br>3) `createWorkflowRecord` → `client.TicketWorkflowRecord.Create`<br>4) `ApproveTicket` 中的审批操作 → Ent 调用<br>5) 移除 `rawDB *sql.DB` 依赖<br>6) 所有现有功能测试通过 | `service/ticket_workflow_service.go` |
| T2-5 | 工单审批流程添加事务保护 | DATA-001 | T2-4 | M | 1) 在 Ent 版本基础上为写操作添加 `client.Tx()`<br>2) `AcceptTicket`、`RejectTicket`、`ApproveTicket`、`ResolveTicket`、`CloseTicket`、`ReopenTicket` 均包裹事务<br>3) 事务失败 Rollback 并返回明确错误<br>4) 编写回滚场景测试 | `service/ticket_workflow_service.go` |
| T2-6 | BPMN 变量合并原子操作 | DATA-002 | 无 | M | 1) 使用 Ent 事务包裹变量合并<br>2) ProcessInstance 添加 version 字段实现乐观锁<br>3) 更新时检查版本号，不匹配则重试<br>4) 编写并发测试 | `service/bpmn_process_engine.go`, `ent/schema/process_instance.go` |

### 第三批（P1 架构+功能）

| 编号 | 任务名称 | PRD编号 | 依赖 | 复杂度 | 实现步骤 | 涉及文件 |
|------|---------|---------|------|--------|---------|---------|
| T3-1 | ServiceTask 调用 CallbackRegistry | FUNC-001 | 无 | M | 1) `handleElement` 中 ServiceTask 分支调用 `ProcessCallbackService.HandleCallback`<br>2) 根据 serviceTask.ID/Name 路由到 Handler<br>3) 构建 `CallbackRequest` 填充必要字段<br>4) Handler 失败时流程进入 error 状态<br>5) 移除 `fmt.Printf` | `service/bpmn_process_engine.go` |
| T3-2 | getDefaultAssigntee 去中文关键词 | FUNC-002 | 无 | M | 1) 新增基于 TaskDefinitionKey 的分配策略<br>2) 查询 ticket_assignment_rules 表获取分配规则<br>3) 中文关键词匹配标记 Deprecated + Warn 日志<br>4) BPMN 属性优先 > 流程变量 > 数据库规则 > 中文关键词兜底 | `service/bpmn_process_engine.go` |
| T3-3 | RBAC 重复代码消除 | CODE-001 | 无 | S | 1) 提取 `loadPermissionsFromDBCommon` 公共方法<br>2) `loadRolePermissionsFromDB` 调用公共方法<br>3) `loadPermissionsFromDB` 调用公共方法 + fallback<br>4) 现有测试通过 | `middleware/rbac.go` |

### 第四批（P1 API 缺失）

| 编号 | 任务名称 | PRD编号 | 依赖 | 复杂度 | 实现步骤 | 涉及文件 |
|------|---------|---------|------|--------|---------|---------|
| T4-1 | Dashboard API 修复 | API-001 | 无 | S | 1) 排查 `DashboardHandler` 初始化链路<br>2) 确认 `RouterConfig.DashboardHandler` 非 nil<br>3) 验证 `GET /api/v1/dashboard` 返回 200 | `main.go`, `router/router.go` |
| T4-2 | 组织架构 API 路由补全 | API-003 | 无 | M | 1) 为 `/api/v1/departments` 补充 POST/PUT/DELETE 路由<br>2) 为 `/api/v1/teams` 补充 POST/PUT/DELETE 路由<br>3) 为 `/api/v1/tags` 补充 POST/PUT/DELETE 路由<br>4) 支持租户级数据隔离 | `router/router.go` |
| T4-3 | 服务目录申请表单增加合规字段 | API-005 | 无 | S | 1) DTO 增加 `compliance_ack` 和 `expire_at`<br>2) 后端校验 `compliance_ack` 为 true<br>3) 前端增加对应输入项 | `dto/service_request_dto.go`, 前端组件 |
| T4-4 | security 角色权限修复 | API-006 | 无 | S | 1) `rbac.go` 中 `RolePermissions["security"]` 增加 ticket:read、notification:read/write、knowledge:read<br>2) 新增迁移脚本更新 role_permissions 表<br>3) 验证 security 角色用户可查看工单/通知/知识库 | `middleware/rbac.go`, `migrations/20260614_security_role_permissions.sql` |
| T4-5 | AI RAG 检索修复 | API-002 | 无 | M | 1) 启动时检测 pgvector 扩展和 vectors 表，不存在则自动创建<br>2) 初始化后执行一次 `EmbeddingPipeline.RunOnce`<br>3) RAG 检索失败时返回友好降级信息<br>4) 编写集成测试 | `service/vector_store.go`, `service/rag_service.go`, `main.go` |
| T4-6 | AI 智能工单 API 补全 | API-004 | 无 | M | 1) 注册 `POST /api/v1/ai/ticket/create` 路由<br>2) 注册 `POST /api/v1/ai/ticket/:id/summarize` 路由（当前已有 GET 版本）<br>3) 实现 AI 工单创建和总结逻辑<br>4) AI 不可用时返回明确错误 | `handlers/ai/`, `router/router.go` |

### 第五批（P2 规范+体验）

| 编号 | 任务名称 | PRD编号 | 依赖 | 复杂度 | 实现步骤 | 涉及文件 |
|------|---------|---------|------|--------|---------|---------|
| T5-1 | 日志规范统一 | CODE-002 | 无 | M | 1) 全局替换 `fmt.Printf` 为 `zap` 调用<br>2) 全局替换 `log.Printf` 为 `zap` 调用<br>3) 添加 linter 规则 | 全局后端文件 |
| T5-2 | DTO 分离统一 | CODE-003 | 无 | S | 1) CMDB 内联 Request 结构体迁移到 `dto/cmdb_dto.go`<br>2) 原位置保留类型别名兼容<br>3) 统一 DTO 规范文档 | `service/cmdb_service.go`, `dto/cmdb_dto.go` |
| T5-3 | 前端 filters 添加 debounce | UX-001 | 无 | S | 1) 工单列表筛选组件添加 300ms debounce<br>2) 批量删除改为调用批量 API | 前端组件 |
| T5-4 | 角色状态字段添加 | UX-002 | 无 | M | 1) `ent/schema/role.go` 添加 `is_active` 字段<br>2) 数据库迁移脚本添加字段并设置默认值<br>3) 角色 CRUD API 支持状态更新<br>4) 前端 inactiveRoles 正确统计 | `ent/schema/role.go`, `migrations/20260614_role_is_active.sql`, 角色相关 API, 前端 |

---

## 6. 依赖包列表

### Go 依赖（新增）
| 包名 | 用途 | 版本建议 |
|------|------|---------|
| 无新增 | 当前依赖已满足所有需求 | — |

> 注：ACL 表达式引擎采用手写递归下降解析器，无需引入 ANTLR 等重量级依赖。向量库使用 pgvector，无需新增独立向量库客户端。

### Node 依赖（新增）
| 包名 | 用途 | 版本建议 |
|------|------|---------|
| lodash.debounce | UX-001 前端防抖 | ^4.17.21 |

---

## 7. 共享知识

### 7.1 跨文件约定

| 约定 | 说明 |
|------|------|
| tenant_id 过滤 | 所有多租户查询必须包含 `tenantID` 过滤条件，新 Ent Schema 必须包含 `tenant_id` 字段 |
| Ent ORM 优先 | 禁止新增原生 SQL 查询，所有数据库操作使用 Ent ORM，如遇 Ent 不支持的操作需在代码注释中说明 |
| 日志规范 | 使用 `zap.S().Infof`/`Warnf`/`Errorf`，禁止 `fmt.Printf`/`log.Printf` |
| DTO 规范 | 所有 Request/Response 结构体统一定义在 `dto` 包 |
| 事务规范 | 写操作（Create/Update/Delete）涉及多表时必须使用 `client.Tx()` |
| 安全兜底 | 条件评估、ACL 校验、权限检查失败时一律 deny（return false），而非 allow |
| 错误返回 | 业务错误使用 `fmt.Errorf` 包装，不要吞掉错误仅打印日志 |

### 7.2 命名规范

| 场景 | 规范 | 示例 |
|------|------|------|
| Ent Schema | PascalCase，与表名对应 | `TicketApproval` → `ticket_approvals` |
| DTO | PascalCase + Request/Response 后缀 | `CreateTicketApprovalRequest` |
| Service 方法 | PascalCase 动词开头 | `ApproveTicket`、`ListRelationships` |
| 数据库迁移 | 日期前缀 | `20260614_security_role_permissions.sql` |

### 7.3 注意事项

1. **ARCH-001 和 DATA-001 强依赖**：必须先完成 Ent Schema 创建（T2-3）和 ORM 迁移（T2-4），才能添加事务保护（T2-5）
2. **CIRelationship 无 tenant_id**：Schema 中缺少此字段，且 `ListRelationships` 方法签名接收 `tenantID` 参数但未使用，修复时需同步处理
3. **CallbackRegistry 已完整**：`service/bpmn/bpmn_callback_registry.go` 有 8 个 Handler，FUNC-001 只需在 `handleElement` 中调用即可
4. **PermissionConfig.Mode 为 DBOnly**：当前系统已切换到仅数据库权限模式，`checkRolePermissionFromDB` 必须真正查 DB
5. **security 角色在数据库中的 ID 为 106**：迁移脚本中硬编码了角色 ID，需注意多租户场景下不同租户的 role ID 可能不同
6. **ProcessInstance 添加 version 字段**：DATA-002 的乐观锁方案需要修改 Schema，需评估对现有数据的影响

---

## 8. 待明确事项

| 编号 | 事项 | 影响 | 建议 |
|------|------|------|------|
| O-1 | 前端实际请求的组织架构 API 路径 | API-003 | 需前端同学确认 `/api/v1/departments` 是 GET 还是 POST 请求 404 |
| O-2 | `ticket_approvals` 表结构是否与 `approval_records` 表相同 | ARCH-001 | 如果不同，新建 Schema 需要对齐 `ticket_approvals` 而非 `approval_records` |
| O-3 | 向量库 `vectors` 表是否已创建 | API-002 | 如果表不存在，自动创建需要 DDL 权限 |
| O-4 | ProcessInstance 添加 version 字段的影响评估 | DATA-002 | 需要确认现有 ProcessInstance 数据量，以及添加字段的迁移策略 |
| O-5 | ACL 表达式语法的最终规范 | SEC-003 | 需产品经理确认 ACL 脚本的具体语法和示例 |
| O-6 | 多租户场景下 security 角色权限的迁移策略 | API-006 | 当前迁移脚本仅处理 tenant_id=1，其他租户需要单独处理 |
| O-7 | DashboardHandler 未注入到 RouterConfig 的根本原因 | API-001 | 可能是初始化顺序问题或依赖缺失 |
| O-8 | AI 智能工单创建/总结的后端实现是否需要调用 LLM | API-004 | 如果需要 LLM 调用，需要确认 API Key 和配额 |
