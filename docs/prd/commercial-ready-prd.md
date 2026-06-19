# ITSM 商用就绪修复 PRD

**文档编号**: ITSM-PRD-2026-001
**版本**: v1.0
**日期**: 2026-06-14
**作者**: 许清楚（Xu），产品经理
**状态**: Draft

---

## 1. 产品目标

### 1.1 核心目标

将 ITSM 系统从当前「可运行但不可商用」状态推进至「商用就绪」状态，确保系统在**安全性、数据一致性、功能完整性**三个维度满足企业级交付标准。

### 1.2 成功指标

| 维度 | 当前状态 | 目标状态 |
|------|---------|---------|
| 安全漏洞 | 3 个 P0 级安全漏洞 | 0 个 P0/P1 级安全漏洞 |
| 数据一致性 | 审批流程无事务保护 | 关键写操作全部事务化 |
| 功能完整度 | Dashboard/AI/组织架构等 API 404 | 核心模块 API 全部可用 |
| 架构合规 | TicketWorkflowService 全量原生 SQL | 统一使用 Ent ORM |
| 代码质量 | 日志混用、RBAC 重复代码 ~70 行 | 日志统一、重复代码消除 |

---

## 2. 用户故事

### 2.1 安全审计员

> 作为一个安全审计员，我需要确保所有多租户数据查询都经过 `tenantID` 过滤，以便不同租户之间不存在数据泄露风险。

> 作为一个安全审计员，我需要 BPMN 网关条件评估在失败时默认拒绝而非默认通过，以便攻击者无法通过构造无效表达式绕过权限网关。

> 作为一个安全审计员，我需要高级 ACL 脚本评估真正执行权限校验逻辑而非直接放行，以便细粒度权限控制生效。

### 2.2 运维工程师

> 作为一个运维工程师，我需要工单审批流程中的多个数据库操作在事务中执行，以便在异常情况下数据不会处于不一致状态。

> 作为一个运维工程师，我需要 BPMN 流程变量的合并操作具有原子性，以便并发完成任务时不会丢失变量更新。

> 作为一个运维工程师，我需要 ServiceTask 真正执行配置的业务逻辑而非仅打印日志，以便自动化流程能正确运转。

### 2.3 系统管理员

> 作为一个系统管理员，我需要 Dashboard API 返回正确的统计数据，以便我可以在首页总览系统运行状况。

> 作为一个系统管理员，我需要角色权限检查真正查询数据库而非回退到硬编码，以便我通过管理界面配置的权限变更能实时生效。

> 作为一个系统管理员，我需要角色状态字段在后端存在，以便我能区分和管理启用/禁用的角色。

### 2.4 普通用户

> 作为一个普通用户，我需要前端搜索和筛选操作有 debounce 防抖，以便快速输入时不会产生大量无效请求。

> 作为一个普通用户，我需要 AI 助手能够正确检索知识库内容，以便我能通过自然语言获取帮助。

> 作为一个 security 角色（安全审批人）用户，我需要能够查看分配给我的工单、通知和知识库，以便我能完成审批工作。

---

## 3. 需求池

### 3.1 P0 - 安全漏洞（必须修复，阻断商用）

#### SEC-001: CMDB 关系列表跨租户数据泄露

| 属性 | 内容 |
|------|------|
| 需求编号 | SEC-001 |
| 需求名称 | CMDB ListRelationships 缺失 tenantID 过滤 |
| 严重程度 | P0 - 安全漏洞 |
| 模块归属 | CMDB / 后端 |
| 详细描述 | `CMDBService.ListRelationships`（`service/cmdb_service.go:141`）接收 `tenantID` 参数但未用于查询过滤。查询 `CIRelationship` 时仅按 `ciID` 过滤，导致任何租户用户可获取其他租户的 CI 关系数据。 |
| 验收标准 | 1. `ListRelationships` 查询必须包含 `cirelationship.TenantID(tenantID)` 条件<br>2. 编写单元测试验证跨租户查询返回空结果<br>3. 同步修复 `GetCITopology` 中递归查询同样缺失 tenantID 过滤的问题 |

#### SEC-002: BPMN 条件评估失败默认通过

| 属性 | 内容 |
|------|------|
| 需求编号 | SEC-002 |
| 需求名称 | BPMN evaluateCondition 评估失败默认 return true |
| 严重程度 | P0 - 安全漏洞 |
| 模块归属 | BPMN 流程引擎 / 后端 |
| 详细描述 | `CustomProcessEngine.evaluateCondition`（`service/bpmn_process_engine.go:530`）在表达式评估返回错误时 `return true`，攻击者可构造无效表达式绕过网关条件，使流程走向非预期分支。`GatewayEngine.evaluateCondition`（`service/bpmn_gateway_engine.go:315`）存在类似问题——无法解析的表达式默认 `return false`，但调用方 `evaluateExclusiveGatewayConditions` 在无任何条件匹配时仍选择第一个输出流作为兜底。 |
| 验收标准 | 1. 条件评估失败时默认 `return false`（安全侧兜底）<br>2. 评估失败记录 Error 级别日志（含表达式内容和错误信息）<br>3. 排他网关无条件满足且无默认流时，返回错误而非选择第一个输出<br>4. 编写测试验证无效表达式不会导致网关放行 |

#### SEC-003: ACL 脚本评估空实现

| 属性 | 内容 |
|------|------|
| 需求编号 | SEC-003 |
| 需求名称 | EvaluateACLScript 空实现直接 return true |
| 严重程度 | P0 - 安全漏洞 |
| 模块归属 | 权限系统 / 后端 |
| 详细描述 | `EvaluateACLScript`（`middleware/smart_permission.go:443`）是高级 ACL 场景的核心，但当前实现无论脚本内容如何都 `return true`。这意味着所有依赖 ACL 脚本的细粒度权限控制（如"用户只能查看自己创建的工单"）完全失效。 |
| 验收标准 | 1. 实现基于沙箱的 ACL 脚本执行引擎，支持基础比较运算（==、!=、>、<）、逻辑运算（&&、||）和上下文变量引用<br>2. 脚本为空字符串时仍 return true（无 ACL 即放行）<br>3. 脚本执行异常时 return false（安全侧兜底）<br>4. 编写单元测试覆盖典型 ACL 脚本场景 |

---

### 3.2 P1 - 架构偏离与数据一致性（必须修复，影响商用质量）

#### ARCH-001: TicketWorkflowService 绕过 Ent ORM

| 属性 | 内容 |
|------|------|
| 需求编号 | ARCH-001 |
| 需求名称 | TicketWorkflowService 全量使用原生 SQL 绕过 Ent ORM |
| 严重程度 | P1 - 架构偏离 |
| 模块归属 | 工单流转 / 后端 |
| 详细描述 | `TicketWorkflowService`（`service/ticket_workflow_service.go`）全量使用 `rawDB.ExecContext` 执行原生 SQL，绕过 Ent ORM 层。这导致：（1）无法享受 ORM 的类型安全和自动迁移；（2）SQL 语句与数据模型解耦，字段变更易遗漏；（3）与项目中其他服务（均使用 Ent）风格不一致。 |
| 验收标准 | 1. 所有原生 SQL 操作改为 Ent ORM 调用<br>2. 移除 `rawDB *sql.DB` 依赖<br>3. 保留必要的 Ent 不支持的复杂查询作为注释说明<br>4. 所有现有功能测试通过 |

#### DATA-001: 工单审批流程缺少事务保护

| 属性 | 内容 |
|------|------|
| 需求编号 | DATA-001 |
| 需求名称 | 工单审批流程多个 SQL 操作缺少事务保护 |
| 严重程度 | P1 - 数据一致性 |
| 模块归属 | 工单流转 / 后端 |
| 详细描述 | `TicketWorkflowService.ApproveTicket`（`service/ticket_workflow_service.go:235`）中，更新审批记录 → 创建委派审批 → 统计待审批数量 → 更新工单状态 → 取消其他待审批 → 创建流转记录，共 6 个写操作没有任何事务包裹。任一步骤失败都会导致数据不一致（如审批记录已更新但工单状态未变更）。 |
| 验收标准 | 1. `AcceptTicket`、`RejectTicket`、`ApproveTicket`、`ResolveTicket`、`CloseTicket`、`ReopenTicket` 等写操作均需事务包裹<br>2. 使用 Ent 的事务 API（`client.Tx()`）<br>3. 事务失败时回滚所有变更并返回明确错误<br>4. 编写测试验证事务回滚场景 |

#### DATA-002: BPMN 变量合并非原子操作

| 属性 | 内容 |
|------|------|
| 需求编号 | DATA-002 |
| 需求名称 | BPMN 流程变量合并非原子操作 |
| 严重程度 | P1 - 数据一致性 |
| 模块归属 | BPMN 流程引擎 / 后端 |
| 详细描述 | `CustomProcessEngine.CompleteTask`（`service/bpmn_process_engine.go:304`）中，变量合并为先读取 `instance.Variables`，再 `for` 循环覆盖，最后写入数据库。两个并发完成任务可能同时读到旧变量，后写入者覆盖前者的更新，导致变量丢失。 |
| 验收标准 | 1. 变量合并操作在事务中执行<br>2. 使用数据库行锁或乐观锁防止并发覆盖<br>3. 编写并发测试验证变量不会丢失 |

#### FUNC-001: ServiceTask 仅打印日志

| 属性 | 内容 |
|------|------|
| 需求编号 | FUNC-001 |
| 需求名称 | ServiceTask 仅 fmt.Printf 不执行实际逻辑 |
| 严重程度 | P1 - 功能缺失 |
| 模块归属 | BPMN 流程引擎 / 后端 |
| 详细描述 | `CustomProcessEngine.handleElement`（`service/bpmn_process_engine.go:393`）中，遇到 ServiceTask 时仅执行 `fmt.Printf("自动执行服务任务: %s\n", ...)`，然后直接跳到下一个步骤。实际上系统已有 `bpmn.CallbackRegistry` 和各类 Handler（Ticket/Incident/Change/Approval/Notification/Webhook），但 `handleElement` 没有调用它们。 |
| 验收标准 | 1. `handleElement` 遇到 ServiceTask 时调用 `ProcessCallbackService.HandleCallback`<br>2. 根据任务类型路由到对应的 Handler<br>3. Handler 执行失败时流程进入错误状态而非继续推进<br>4. 移除 `fmt.Printf` 调试输出 |

#### SEC-004: CI 关系创建无 tenantID 校验

| 属性 | 内容 |
|------|------|
| 需求编号 | SEC-004 |
| 需求名称 | CreateRelationship 不校验 source/target CI 的 tenantID |
| 严重程度 | P1 - 安全漏洞 |
| 模块归属 | CMDB / 后端 |
| 详细描述 | `CMDBService.CreateRelationship`（`service/cmdb_service.go:100`）直接创建关系，不校验 `source_ci_id` 和 `target_ci_id` 是否属于当前租户。攻击者可通过指定其他租户的 CI ID 建立跨租户关联关系。 |
| 验收标准 | 1. 创建关系前校验 source CI 和 target CI 均属于当前 tenantID<br>2. 校验失败返回 403 错误<br>3. `CreateRelationshipRequest` 增加 `TenantID` 字段 |

#### SEC-005: checkRolePermissionFromDB 未真正查 DB

| 属性 | 内容 |
|------|------|
| 需求编号 | SEC-005 |
| 需求名称 | checkRolePermissionFromDB 未查询数据库权限 |
| 严重程度 | P1 - 权限失效 |
| 模块归属 | 权限系统 / 后端 |
| 详细描述 | `checkRolePermissionFromDB`（`middleware/smart_permission.go:389`）函数名暗示从数据库查询权限，但实际仅做了超级管理员判断和硬编码权限查找，没有查询 `role_permissions` 表。这导致管理员通过界面配置的权限变更无法在 ACL 层生效。 |
| 验收标准 | 1. 函数内增加从 `role_permissions` 表查询权限的逻辑<br>2. 查询结果与硬编码权限合并（或根据 PermissionConfig.Mode 决定策略）<br>3. 编写测试验证数据库权限配置生效 |

#### CODE-001: RBAC 重复代码

| 属性 | 内容 |
|------|------|
| 需求编号 | CODE-001 |
| 需求名称 | rbac.go 重复代码约 70 行 |
| 严重程度 | P1 - 代码质量 |
| 模块归属 | 权限系统 / 后端 |
| 详细描述 | `middleware/rbac.go` 中 `loadRolePermissionsFromDB`（第 357-426 行）和 `loadPermissionsFromDB`（第 445-519 行）存在约 70 行重复代码，两个函数都执行了相同的角色查询 → 权限联表查询 → 缓存写入逻辑。 |
| 验收标准 | 1. 提取公共的权限加载和缓存逻辑<br>2. `loadPermissionsFromDB` 调用提取后的公共方法<br>3. 移除 `loadRolePermissionsFromDB`，统一入口<br>4. 现有测试全部通过 |

#### FUNC-002: getDefaultAssigntee 基于中文关键词

| 属性 | 内容 |
|------|------|
| 需求编号 | FUNC-002 |
| 需求名称 | getDefaultAssigntee 基于中文关键词匹配，脆弱不可靠 |
| 严重程度 | P1 - 功能脆弱 |
| 模块归属 | BPMN 流程引擎 / 后端 |
| 详细描述 | `CustomProcessEngine.getDefaultAssigntee`（`service/bpmn_process_engine.go:467`）通过 `strings.Contains(taskName, "审批")` 等中文关键词匹配来决定任务分配人。这在国际化环境或非中文命名场景下完全失效。 |
| 验收标准 | 1. 任务分配策略改为基于 BPMN 定义中的 `assignee`/`candidateUsers`/`candidateGroups` 属性<br>2. 仅在 BPMN 未定义分配策略时，使用基于角色配置的任务分配规则（而非中文关键词）<br>3. 中文关键词匹配逻辑标记为 deprecated 并计划移除 |

---

### 3.3 P2 - 代码规范与体验优化（建议修复，不阻断商用）

#### CODE-002: 日志规范不一致

| 属性 | 内容 |
|------|------|
| 需求编号 | CODE-002 |
| 需求名称 | 日志规范不一致（fmt.Printf/log.Printf/zap 混用） |
| 严重程度 | P2 - 代码规范 |
| 模块归属 | 全局 / 后端 |
| 详细描述 | 项目混用三种日志方式：`fmt.Printf`（如 `bpmn_process_engine.go:393`、`bpmn_variable_service.go:564`）、`log.Printf`（如 `cmd/` 目录）和 `zap`（推荐方式）。`fmt.Printf` 不经过日志框架，无法控制级别、格式和输出目标。 |
| 验收标准 | 1. 全局替换 `fmt.Printf` 为 `zap.S().Infof`/`Warnf`/`Errorf`<br>2. 全局替换 `log.Printf` 为 `zap` 调用<br>3. 新增 linter 规则禁止直接使用 fmt/log 包输出日志 |

#### CODE-003: DTO 分离不一致

| 属性 | 内容 |
|------|------|
| 需求编号 | CODE-003 |
| 需求名称 | DTO 分离不一致 |
| 严重程度 | P2 - 代码规范 |
| 模块归属 | 全局 / 后端 |
| 详细描述 | 部分模块（如 CMDB）在 `service` 包内定义 Request 结构体，部分模块（如 Ticket）在 `dto` 包定义。缺乏统一规范，增加维护成本。 |
| 验收标准 | 1. 制定 DTO 规范：所有 Request/Response 结构体统一定义在 `dto` 包<br>2. 迁移 CMDB 等 service 内联结构体到 `dto` 包<br>3. 不影响现有 API 接口兼容性 |

#### UX-001: 前端 filters 无 debounce

| 属性 | 内容 |
|------|------|
| 需求编号 | UX-001 |
| 需求名称 | 前端 filters 无 debounce、批量删除逐条调用 |
| 严重程度 | P2 - 用户体验 |
| 模块归属 | 前端 |
| 详细描述 | 工单列表等页面的筛选条件输入时每次按键都触发 API 请求，缺少 debounce 防抖。批量删除操作逐条调用 API 而非批量接口。 |
| 验收标准 | 1. 所有 filters 组件添加 300ms debounce<br>2. 批量删除改为调用批量 API（如后端无批量接口则后端先补充）<br>3. 用户快速输入时不会产生超过 1 次/300ms 的请求 |

#### UX-002: 角色状态字段后端缺失

| 属性 | 内容 |
|------|------|
| 需求编号 | UX-002 |
| 需求名称 | 角色状态字段后端缺失，前端 inactiveRoles 始终为 0 |
| 严重程度 | P2 - 功能不完整 |
| 模块归属 | 角色管理 / 前后端 |
| 详细描述 | 前端角色管理页面（`admin/roles/page.tsx`）展示了 inactiveRoles 统计，但后端 Role 实体没有 `status` 字段，导致 inactiveRoles 始终为 0，无法区分启用/禁用角色。 |
| 验收标准 | 1. 后端 Role 实体增加 `is_active`（或 `status`）字段<br>2. 数据库迁移脚本添加字段并设置默认值为 active<br>3. 角色 CRUD API 支持状态更新<br>4. 前端 inactiveRoles 正确反映禁用角色数量 |

---

### 3.4 P1 - API 缺失（历史测试遗留）

#### API-001: Dashboard API 404

| 属性 | 内容 |
|------|------|
| 需求编号 | API-001 |
| 需求名称 | GET /api/v1/dashboard 返回 404 |
| 严重程度 | P1 - 功能缺失 |
| 模块归属 | Dashboard / 后端 |
| 详细描述 | 测试报告 B5：`GET /api/v1/dashboard` 返回 404。后端已有 `dashboard_controller.go` 和 `dashboard_service.go`，但路由可能未注册或路径不匹配。 |
| 验收标准 | 1. `GET /api/v1/dashboard` 返回 200 和正确的统计数据<br>2. 包含工单/事件/变更等核心统计指标<br>3. 支持租户级数据隔离 |

#### API-002: AI 助手 RAG 检索 0 命中

| 属性 | 内容 |
|------|------|
| 需求编号 | API-002 |
| 需求名称 | AI 助手 RAG 检索 0 命中（向量库未初始化） |
| 严重程度 | P1 - 功能缺失 |
| 模块归属 | AI 助手 / 后端 |
| 详细描述 | 测试报告 B6：AI 助手的 RAG 检索始终返回 0 命中结果，向量数据库未初始化或未连接。 |
| 验收标准 | 1. 系统启动时自动初始化向量库连接<br>2. 知识库文章创建/更新时自动同步向量索引<br>3. RAG 检索返回相关结果<br>4. 向量库不可用时优雅降级（返回提示信息而非 0 命中） |

#### API-003: 组织架构 API 404

| 属性 | 内容 |
|------|------|
| 需求编号 | API-003 |
| 需求名称 | /api/v1/{departments,tags,teams} 全部 404 |
| 严重程度 | P1 - 功能缺失 |
| 模块归属 | 组织架构 / 后端 |
| 详细描述 | 测试报告 B7：部门、标签、团队三个 API 端点全部返回 404。前端已有对应页面但无法获取数据。 |
| 验收标准 | 1. `GET /api/v1/departments` 返回部门列表<br>2. `GET /api/v1/tags` 返回标签列表<br>3. `GET /api/v1/teams` 返回团队列表<br>4. 支持租户级数据隔离和 CRUD 操作 |

#### API-004: AI 智能工单 API 404

| 属性 | 内容 |
|------|------|
| 需求编号 | API-004 |
| 需求名称 | AI 智能工单创建和总结 API 404 |
| 严重程度 | P1 - 功能缺失 |
| 模块归属 | AI 助手 / 后端 |
| 详细描述 | 测试报告 B9：AI 智能创建工单和 AI 工单总结 API 返回 404。 |
| 验收标准 | 1. `POST /api/v1/ai/ticket/create` 支持自然语言创建工单<br>2. `POST /api/v1/ai/ticket/{id}/summarize` 支持工单总结<br>3. AI 功能不可用时返回明确错误信息而非 404 |

#### API-005: 服务目录申请表单缺字段

| 属性 | 内容 |
|------|------|
| 需求编号 | API-005 |
| 需求名称 | 服务目录申请表单缺 compliance_ack/expire_at 字段 |
| 严重程度 | P1 - 功能不完整 |
| 模块归属 | 服务目录 / 前后端 |
| 详细描述 | 测试报告 B10：服务目录申请表单缺少合规确认（compliance_ack）和到期时间（expire_at）字段，不满足企业合规要求。 |
| 验收标准 | 1. 服务请求 DTO 增加 `compliance_ack`（bool）和 `expire_at`（time）字段<br>2. 前端申请表单增加对应输入项<br>3. `compliance_ack` 为 false 时阻止提交 |

#### API-006: security 角色权限过严

| 属性 | 内容 |
|------|------|
| 需求编号 | API-006 |
| 需求名称 | security1 角色权限过严（无法查看工单/通知/知识库） |
| 严重程度 | P1 - 权限配置 |
| 模块归属 | 权限系统 / 后端 |
| 详细描述 | 测试报告 B12：security 角色用户无法查看工单、通知、知识库，导致安全审批人无法正常工作。当前硬编码权限已有修复，但数据库权限配置可能未同步更新。 |
| 验收标准 | 1. security 角色在数据库中配置 ticket:read、notification:read/write、knowledge:read 权限<br>2. security 角色用户可正常查看分配给自己的工单<br>3. security 角色用户可查看通知和知识库 |

---

## 4. 需求优先级矩阵

```
紧急度 ↑
  │  SEC-001  SEC-002  SEC-003
  │  SEC-004  SEC-005
  │
  │  DATA-001  DATA-002  ARCH-001
  │  FUNC-001  FUNC-002
  │  API-001~006
  │
  │  CODE-001  CODE-002  CODE-003
  │  UX-001    UX-002
  │
  └──────────────────────────────→ 影响范围
```

**修复顺序建议**：

1. **第一批（P0 安全漏洞）**: SEC-001 → SEC-002 → SEC-003
2. **第二批（P1 安全+数据一致性）**: SEC-004 → SEC-005 → DATA-001 → DATA-002
3. **第三批（P1 架构+功能）**: ARCH-001 → FUNC-001 → FUNC-002 → CODE-001
4. **第四批（P1 API 缺失）**: API-001 → API-003 → API-005 → API-006 → API-002 → API-004
5. **第五批（P2 规范+体验）**: CODE-002 → CODE-003 → UX-001 → UX-002

---

## 5. 待确认问题

| 编号 | 问题 | 影响范围 | 建议方 |
|------|------|---------|-------|
| Q-001 | ACL 脚本引擎的实现深度：是支持完整的脚本语言（如 JavaScript/Lua 沙箱），还是仅支持有限的表达式语法？ | SEC-003 | 产品经理 + 架构师 |
| Q-002 | `TicketWorkflowService` 迁移到 Ent ORM 后，审批流程中涉及的 `ticket_approvals` 和 `ticket_workflow_records` 表是否有对应的 Ent Schema？如果没有，是否需要先创建？ | ARCH-001 | 架构师 |
| Q-003 | AI 助手的向量数据库选型：当前使用什么向量库？是否需要支持多种向量库后端？ | API-002 | 架构师 |
| Q-004 | 组织架构（departments/tags/teams）是全新模块还是已有 Schema 但缺路由？这决定了实现工作量。 | API-003 | 架构师 |
| Q-005 | BPMN 条件评估默认改为 `return false` 后，现有流程定义中是否存在「无条件默认分支」依赖 `return true` 的隐式逻辑？需要回归测试。 | SEC-002 | QA |
| Q-006 | ServiceTask 调用 CallbackRegistry 后，如果 Handler 执行时间较长（如外部 API 调用），是否需要支持异步执行？ | FUNC-001 | 架构师 |
| Q-007 | `getDefaultAssigntee` 移除中文关键词匹配后，现有使用中文任务名称的流程定义是否需要迁移？ | FUNC-002 | 产品经理 |
| Q-008 | security 角色权限修复是仅更新硬编码映射，还是需要同时提供数据库迁移脚本更新已有租户的权限配置？ | API-006 | 架构师 |

---

## 附录 A: 需求编号索引

| 编号 | 名称 | 级别 | 模块 |
|------|------|------|------|
| SEC-001 | CMDB ListRelationships 缺失 tenantID 过滤 | P0 | CMDB |
| SEC-002 | BPMN evaluateCondition 评估失败默认 true | P0 | BPMN |
| SEC-003 | EvaluateACLScript 空实现 | P0 | 权限 |
| SEC-004 | CreateRelationship 无 tenantID 校验 | P1 | CMDB |
| SEC-005 | checkRolePermissionFromDB 未查 DB | P1 | 权限 |
| ARCH-001 | TicketWorkflowService 绕过 Ent ORM | P1 | 工单流转 |
| DATA-001 | 审批流程缺少事务保护 | P1 | 工单流转 |
| DATA-002 | BPMN 变量合并非原子操作 | P1 | BPMN |
| FUNC-001 | ServiceTask 仅打印日志 | P1 | BPMN |
| FUNC-002 | getDefaultAssigntee 中文关键词匹配 | P1 | BPMN |
| CODE-001 | rbac.go 重复代码 ~70 行 | P1 | 权限 |
| CODE-002 | 日志规范不一致 | P2 | 全局 |
| CODE-003 | DTO 分离不一致 | P2 | 全局 |
| UX-001 | 前端 filters 无 debounce | P2 | 前端 |
| UX-002 | 角色状态字段后端缺失 | P2 | 角色管理 |
| API-001 | Dashboard API 404 | P1 | Dashboard |
| API-002 | AI RAG 检索 0 命中 | P1 | AI |
| API-003 | 组织架构 API 404 | P1 | 组织架构 |
| API-004 | AI 智能工单 API 404 | P1 | AI |
| API-005 | 服务目录申请表单缺字段 | P1 | 服务目录 |
| API-006 | security 角色权限过严 | P1 | 权限 |

---

## 附录 B: 参考文档

- 架构评审报告: `docs/review/architecture-review-2026-06-14.md`
- 问题源码位置:
  - `service/cmdb_service.go` (SEC-001, SEC-004)
  - `service/bpmn_process_engine.go` (SEC-002, FUNC-001, FUNC-002, DATA-002)
  - `middleware/smart_permission.go` (SEC-003, SEC-005)
  - `service/ticket_workflow_service.go` (ARCH-001, DATA-001)
  - `service/bpmn_gateway_engine.go` (SEC-002)
  - `middleware/rbac.go` (CODE-001)
