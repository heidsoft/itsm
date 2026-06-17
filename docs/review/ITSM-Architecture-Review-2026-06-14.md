# ITSM 系统架构评审报告

**评审日期**: 2026-06-14  
**评审人**: 高见远（Gao），架构师  
**系统版本**: 基于 main 分支当前代码  
**技术栈**: Go/Gin + Ent ORM（后端） | Next.js/TypeScript + Ant Design（前端）

---

## 1. 执行摘要

### 整体评价

ITSM 系统作为一套面向企业级交付的 IT 服务管理平台，功能覆盖面广（工单/事件/问题/变更管理、CMDB、BPMN 工作流引擎、RBAC 权限体系、SLA 监控、AI 辅助等），架构思路基本正确——采用分层架构、多租户隔离、ORM 优先策略。然而，在深度实现层面存在若干架构风险和一致性问题，需要优先治理。

**架构成熟度评分: 3.0 / 5.0**

| 维度 | 评分 | 说明 |
|------|------|------|
| 功能完整性 | 4.0/5 | 覆盖 ITIL 核心流程，BPMN 引擎可用但受限 |
| 代码质量 | 2.5/5 | 大量原始 SQL、调试输出、空实现 |
| 安全性 | 2.5/5 | 跨租户泄露风险、表达式引擎默认通过 |
| 可维护性 | 2.5/5 | 单文件过大、重复逻辑、缺失抽象 |
| 可扩展性 | 3.0/5 | 接口设计合理但网关/服务任务实现不完整 |
| 测试覆盖 | 2.0/5 | 有集成测试但核心模块单元测试不足 |

### 关键发现（Top 5）

1. **🔴 P0 安全漏洞**: `CMDB.ListRelationships` 未过滤 `tenantID`，跨租户数据可直接泄露
2. **🔴 P0 安全漏洞**: BPMN 表达式引擎 `evaluateCondition` 在评估失败时默认 `return true`，攻击者可构造无效表达式绕过网关条件
3. **🟠 P1 架构偏离**: `TicketWorkflowService` 全量使用原生 SQL，绕过 Ent ORM，与项目规范严重不一致
4. **🟠 P1 数据一致性**: 工单审批流程中多个 SQL 操作缺少事务保护，并发场景下审批结果可能不一致
5. **🟡 P2 功能缺陷**: BPMN 引擎仅支持 ExclusiveGateway，不支持并行网关和包容网关，限制了业务流程表达能力

---

## 2. 工作流模块评估

### 架构设计评分: ⭐⭐⭐ (3/5)

**理由**: 接口分层（ProcessEngine → DefinitionService/InstanceService/TaskService）设计合理，内置表达式引擎和审计服务是亮点；但 ServiceTask 空实现、网关类型单一、变量并发不安全等问题显著降低了引擎的可用性。

### 优点

| # | 优点 | 位置 |
|---|------|------|
| 1 | **接口抽象良好**: `ProcessEngine` 接口清晰分离定义/实例/任务三层子服务 | `bpmn_process_engine.go:23-36` |
| 2 | **内置表达式引擎**: 基于 `expr-lang/expr` 实现完整的表达式评估，支持自定义函数注册 | `expression_engine.go:14-17` |
| 3 | **审计日志完整**: 流程启动/暂停/恢复/终止/任务完成均有审计记录 | `bpmn_process_engine.go:254,629,674,729` |
| 4 | **会签支持**: 实现了并行/串行会签、投票机制、阈值判定 | `bpmn_process_engine.go:1604-1802` |
| 5 | **任务操作丰富**: Claim/Delegate/Escalate/Retry/BatchAssign 等任务生命周期操作齐全 | `bpmn_process_engine.go:1351-1536` |
| 6 | **BPMN 模板预置**: 提供 incident_emergency、change_normal 等 5 个业务模板 | `service/bpmn/` |
| 7 | **ServiceTask Handler 体系**: 通过 `CallbackRegistry` 实现了可扩展的 Handler 注册机制 | `bpmn_service_task_adapter.go` |

### 问题列表

| 级别 | 问题描述 | 位置 | 影响 |
|------|----------|------|------|
| **P0** | `evaluateCondition` 表达式评估失败时默认 `return true`，可被利用绕过网关条件 | `bpmn_process_engine.go:552` | 安全风险：恶意构造无效表达式可绕过所有条件网关 |
| **P1** | `CompleteTask` 中变量合并是非原子操作：先读 `instance.Variables` → 合并 → 写回，无乐观锁/事务 | `bpmn_process_engine.go:303-317` | 并发场景下变量可能丢失 |
| **P1** | `handleElement` 中 ServiceTask 仅 `fmt.Printf` 打印日志并继续推进，未调用 Handler | `bpmn_process_engine.go:393-394` | 自动化任务不会真正执行，流程自动推进到下一步 |
| **P1** | 仅支持 ExclusiveGateway，不支持 ParallelGateway、InclusiveGateway | `bpmn_process_engine.go:390` | 无法表达并行审批、会签分支等业务场景 |
| **P1** | `getDefaultAssigntee` 基于中文关键词匹配角色，脆弱且不可靠 | `bpmn_process_engine.go:467-508` | 任务名变更即导致分配错误，且 "处理" 重复检查 |
| **P2** | `fmt.Printf` 调试输出应替换为结构化日志 | `bpmn_process_engine.go:393` | 生产环境日志不规范 |
| **P2** | 无 BPMN 模型校验机制（如死循环检测、孤立节点检测） | 全局 | 非法流程定义可导致引擎无限递归 |
| **P2** | `GetProcessInstanceHistory` 返回空数组，未实现 | `bpmn_process_engine.go:1186-1188` | 流程历史追踪不可用 |
| **P2** | `getInstanceStatistics` 加载全量实例到内存中统计，大数据量下性能极差 | `bpmn_process_engine.go:1207` | OOM 风险 |
| **P2** | `CompleteTaskByID` 每次调用都 `NewCustomProcessEngine` 创建新引擎实例 | `bpmn_process_engine.go:1280` | 资源浪费，可能遗漏已注册的函数 |

### 改进建议

1. **表达式引擎安全加固**（P0 → 立即修复）:
   - `evaluateCondition` 失败时应 `return false`（拒绝策略），而非默认通过
   - 增加表达式沙箱白名单，限制可调用的函数和变量

2. **变量合并原子化**（P1）:
   - 使用 Ent 的事务机制包裹读-改-写操作
   - 或使用乐观锁（version 字段）防止并发覆盖

3. **ServiceTask 真正执行**（P1）:
   - `handleElement` 遇到 ServiceTask 时，调用已注册的 `CallbackRegistry` 执行实际逻辑
   - 执行失败时应支持重试/错误边界事件

4. **网关类型扩展**（P1 → Q2 规划）:
   - 实现 ParallelGateway：同时激活所有出边，等待所有分支完成后汇聚
   - 实现 InclusiveGateway：条件为真的出边全部激活

5. **任务分配策略重构**（P1）:
   - 废弃中文关键词匹配，改为基于 BPMN 定义中的 `candidateGroups`/`candidateUsers` 属性
   - 支持「任务分配规则表」配置，替代硬编码

---

## 3. CMDB 配置管理模块评估

### 架构设计评分: ⭐⭐⭐ (3/5)

**理由**: 核心 CI CRUD 功能可用，关系类型定义丰富（13 种），拓扑查询使用 BFS 避免了递归栈溢出；但 `cmdb_service.go` 中的 `ListRelationships` 跨租户泄露是严重安全问题，DTO 未分离到标准目录，搜索能力薄弱。

### 优点

| # | 优点 | 位置 |
|---|------|------|
| 1 | **CI 模型灵活**: 支持 `attributes` JSON 字段存储扩展属性，适配不同 CI 类型 | `configurationitem.go` |
| 2 | **关系类型丰富**: 定义 13 种标准关系类型（depends_on/hosts/connects_to 等） | `ci_relationship_service.go:496-512` |
| 3 | **拓扑图 BFS 遍历**: 使用队列而非递归，避免了栈溢出，且已限制深度上限 3 | `ci_relationship_service.go:299-384` |
| 4 | **影响分析完整**: 支持上游/下游影响分析、关键依赖识别、风险等级计算 | `ci_relationship_service.go:393-471` |
| 5 | **关联工单/事件**: 影响分析中关联了受影响的工单和事件，提供业务影响视角 | `ci_relationship_service.go:428-465` |
| 6 | **批量创建关系**: 支持批量创建 CI 关系 | `ci_relationship_service.go:474-493` |
| 7 | **前端组件拆分合理**: CSDMHub/CIList/CIDetail/CIRelationshipManager/TopologyGraph 各司其职 | 前端 `components/cmdb/` |

### 问题列表

| 级别 | 问题描述 | 位置 | 影响 |
|------|----------|------|------|
| **P0** | `ListRelationships` 未过滤 `tenantID`，任何租户用户可查看所有租户的 CI 关系 | `cmdb_service.go:141-155` | **跨租户数据泄露** |
| **P1** | `GetCITopology` 递归查询无深度保护上限（虽然 `CIRelationshipService` 的 BFS 版本已限制 depth=3，但 `CMDBService.GetCITopology` 仍接受任意 depth 参数） | `cmdb_service.go:158-191` | 恶意请求 depth=100 可导致大量 DB 查询 |
| **P1** | `ListCIs` 缺少全文搜索/模糊查询能力（仅有精确过滤 ciType/status/environment） | `cmdb_service.go:67-97` | 用户无法按名称/描述搜索 CI |
| **P1** | `CreateRelationship`/`DeleteRelationship` 未校验 CI 的 tenantID 一致性 | `cmdb_service.go:100-117` | 可创建跨租户的 CI 关系 |
| **P2** | 请求结构体定义在 `cmdb_service.go` 中，未分离到 `dto/` 目录 | `cmdb_service.go:194-236` | 与项目规范不一致 |
| **P2** | CI 类型管理（CIType）与 CI 实例管理分离度不够，缺少 CIType 属性模板 | 架构层面 | 无法按类型动态扩展 CI 表单 |
| **P2** | `BatchCreateCIRelationships` 逐条创建，无事务保护 | `ci_relationship_service.go:481-493` | 部分失败时数据不一致 |
| **P2** | `BatchCreateCIRelationships` 使用 `log.Printf` 而非 zap 日志 | `ci_relationship_service.go:489` | 日志规范不一致 |

### 改进建议

1. **修复 ListRelationships 跨租户泄露**（P0 → 立即修复）:
   ```go
   query = query.Where(cirelationship.HasSourceCiWith(configurationitem.TenantID(tenantID)))
   ```
   或者在 CI 关系表上增加 `tenant_id` 字段。

2. **增加搜索能力**（P1）:
   - 添加 `keyword` 参数，在 `name`、`serial_number`、`asset_tag` 字段上做 `Contains` 模糊查询
   - 长期考虑集成 Elasticsearch 做全文检索

3. **深度保护硬上限**（P1）:
   - `GetCITopology` 中增加 `if depth > 5 { depth = 5 }` 硬上限
   - 同时限制返回节点总数（如最大 500 个）

4. **关系创建租户校验**（P1）:
   - 在 `CreateRelationship` 中校验 sourceCI 和 targetCI 属于同一 tenantID

5. **DTO 分离**（P2）:
   - 将 `CreateCIRequest`、`ListCIsRequest` 等结构体移至 `dto/cmdb_dto.go`

---

## 4. RBAC 权限体系评估

### 架构设计评分: ⭐⭐⭐ (3/5)

**理由**: 4 层权限检查架构设计先进（L1 白名单 → L2 DB ACL → L3 URL 推断 → L4 硬编码兜底），多模式切换灵活；但 L2 层 `checkRolePermissionFromDB` 实际上回退到硬编码（并未真正查 DB），`EvaluateACLScript` 空实现直接 return true，降低了整体安全性。文件过大（1186 行）且存在重复逻辑。

### 优点

| # | 优点 | 位置 |
|---|------|------|
| 1 | **4 层兜底架构**: L1 白名单 → L2 DB ACL → L3 URL 推断 → L4 硬编码，保证权限不遗漏 | `smart_permission.go:107-126` |
| 2 | **多模式支持**: DBOnly/HardcodeOnly/Merge/Fallback 四种模式适配不同部署环境 | `rbac.go:331-342` |
| 3 | **权限缓存 + TTL**: 内存 Map 缓存 + 5 分钟 TTL + 手动失效，性能可接受 | `rbac.go:36-44` |
| 4 | **细粒度角色定义**: 9 个基础角色 + 5 个 MSP 角色，权限映射详尽 | `rbac.go:54-328` |
| 5 | **工单级权限控制**: `checkTicketPermission` 实现 end_user 只能访问自己的工单 | `rbac.go:1084-1118` |
| 6 | **ACL 热重载**: 支持运行时动态刷新 ACL 缓存 | `smart_permission.go:86-99` |
| 7 | **ResourceActionMap**: API 路径 → resource:action 的完整映射表，支持通配符 | `rbac.go:521-669` |

### 问题列表

| 级别 | 问题描述 | 位置 | 影响 |
|------|----------|------|------|
| **P0** | `EvaluateACLScript` 空实现（直接 return true），高级 ACL 场景如"只能看到分配给自己的工单"永远通过 | `smart_permission.go:443-458` | 安全策略形同虚设 |
| **P1** | `checkRolePermissionFromDB`（L2 层）实际回退到硬编码 `hasResourcePermissionFromRole`，并未真正查 DB | `smart_permission.go:389-401` | DB ACL 配置无效，L2 层名不副实 |
| **P1** | `loadRolePermissionsFromDB` 和 `loadPermissionsFromDB` 逻辑几乎相同（重复代码约 70 行） | `rbac.go:357-518` | 维护成本高，修改一处容易遗漏另一处 |
| **P1** | `rbac.go` 1186 行单文件过于庞大，应按职责拆分 | `rbac.go` 全文件 | 可读性差、协作冲突 |
| **P1** | `checkTicketOwnership` 每次都查询 DB，无缓存，高并发下成为瓶颈 | `rbac.go:1152-1170` | 性能问题 |
| **P1** | `isKnownWhitelistPath` 硬编码白名单路径，与 L1 的 `authWhitelist` 重复 | `smart_permission.go:262-272` vs `smart_permission.go:133-138` | 维护不一致风险 |
| **P1** | ACL 缓存按 `tenantID` 隔离但未按 `role` 隔离，同租户不同角色可能共享缓存 | `smart_permission.go:59` | 权限判断可能不准确 |
| **P2** | `AssignPermissions` 先清空再重建策略，存在短暂无权限窗口 | `role_service.go:390-412` | 并发请求可能在清空后、重建前被拒绝 |
| **P2** | 无权限变更审计日志 | `role_service.go` 全文件 | 无法追溯谁在何时修改了什么权限 |
| **P2** | `generateCodeFromName` 截断为 20 字符可能冲突 | `role_service.go:552-561` | 中文名截断后不同名称可能生成相同 code |
| **P2** | 前端 `PERMISSION_MODULES`（20 种）与后端 `InitDefaultPermissions`（22 种资源类型）不完全对齐 | 前后端对比 | 权限矩阵不完整 |
| **P2** | 前端角色页面 `inactiveRoles` 始终为 0（后端无 status 字段） | 前端 `admin/roles/page.tsx` | UI 数据不准确 |
| **P2** | 删除角色无关联用户检查（虽然后端 `DeleteRole` 有检查，但前端缺少提示） | 前端 `admin/roles/page.tsx` | 用户体验不完整 |

### 改进建议

1. **修复 EvaluateACLScript**（P0）:
   - 短期：默认 `return false`（安全策略），非空脚本需要显式实现
   - 长期：集成表达式引擎（复用 `ExpressionEngine`）执行 ACL 脚本

2. **修复 L2 层 checkRolePermissionFromDB**（P1）:
   - 应真正查询 `role_permissions` + `permissions` 表
   - 复用 `loadPermissionsFromDB` 获取权限列表后调用 `checkPermissionMatch`

3. **拆分 rbac.go**（P1）:
   - `rbac_middleware.go`: 中间件主逻辑 + RBACMiddleware
   - `rbac_permissions.go`: RolePermissions 映射 + ResourceActionMap
   - `rbac_cache.go`: 缓存管理逻辑
   - `rbac_resource.go`: 资源级别权限检查（checkTicketPermission 等）

4. **合并重复的 DB 加载函数**（P1）:
   - 保留 `loadPermissionsFromDB`，删除 `loadRolePermissionsFromDB`
   - 或让前者调用后者避免重复

5. **权限变更审计**（P2）:
   - 在 `AssignPermissions` 中增加审计日志记录
   - 记录：谁、何时、对哪个角色、增加了/删除了哪些权限

---

## 5. 跨模块问题

### 5.1 架构一致性问题

| # | 问题 | 影响 | 严重程度 |
|---|------|------|----------|
| 1 | **ORM 使用不一致**: `TicketWorkflowService` 全量使用 raw SQL，其他模块（CMDB/BPMN/RBAC）使用 Ent ORM | 维护成本高、类型安全丢失、新人理解困难 | P1 |
| 2 | **日志规范不一致**: BPMN 引擎混用 `fmt.Printf` 和 `zap`，CMDB 使用 `log.Printf`，RBAC 使用 `zap.S()` | 生产环境日志不可控 | P2 |
| 3 | **DTO 分离不一致**: CMDB 请求结构体定义在 service 文件中，TicketWorkflow 使用 `dto/` 包 | 代码组织混乱 | P2 |
| 4 | **上下文传值方式不一致**: BPMN 通过 `ctx.Value("bpmn_tenant_id")`，RBAC 通过 `c.Set("tenant_id")` | 键名冲突风险、类型不安全 | P2 |
| 5 | **租户隔离粒度不一致**: CI 关系缺少 tenantID 字段（需通过 JOIN CI 表过滤），工单有 tenantID 字段 | 部分查询性能差、安全漏洞风险 | P1 |

### 5.2 数据一致性问题

| # | 问题 | 影响 | 严重程度 |
|---|------|------|----------|
| 1 | **工单审批无事务**: 审批通过后更新工单状态 + 取消其他审批不是原子操作 | 并发审批可能导致数据不一致 | P1 |
| 2 | **BPMN 变量合并无事务**: CompleteTask 读-改-写 instance.Variables 非原子 | 并发完成任务时变量可能丢失 | P1 |
| 3 | **权限分配先删后建**: AssignPermissions 清空 → 重建之间有短暂无权限窗口 | 瞬间请求可能被拒绝 | P2 |
| 4 | **前后端字段名不一致**: TicketID/TicketIDAlt 兼容性处理 | 前后端契约不清晰 | P2 |

### 5.3 前端共性问题

| # | 问题 | 位置 | 严重程度 |
|---|------|------|----------|
| 1 | Workflow 页面 `filters` 变化触发 `loadWorkflows` 无 debounce | `workflow/page.tsx` | P2 |
| 2 | Workflow 统计数据从前端列表计算而非后端 API，分页下不准确 | `workflow/page.tsx` | P2 |
| 3 | 批量删除逐条调用 API，应使用批量接口 | `workflow/page.tsx` | P2 |
| 4 | `workflowConsoleEntries` 中部分路径可能未实现 | `workflow/page.tsx` | P2 |
| 5 | 角色状态字段后端缺失，前端按启用处理 | `admin/roles/page.tsx` | P2 |

---

## 6. 技术债务清单

| # | 技术债务 | 优先级 | 预估工作量 | 模块 |
|---|----------|--------|------------|------|
| 1 | `cmdb_service.ListRelationships` 未过滤 tenantID（跨租户泄露） | **P0** | 0.5 天 | CMDB |
| 2 | `evaluateCondition` 失败默认 return true（安全漏洞） | **P0** | 0.5 天 | Workflow |
| 3 | `EvaluateACLScript` 空实现 return true | **P0** | 2 天 | RBAC |
| 4 | `TicketWorkflowService` 原生 SQL 重写为 Ent ORM | **P1** | 5 天 | Workflow |
| 5 | 工单审批流程添加事务保护 | **P1** | 2 天 | Workflow |
| 6 | BPMN 变量合并原子化 | **P1** | 1 天 | Workflow |
| 7 | ServiceTask 真正执行 Handler 而非 fmt.Printf | **P1** | 3 天 | Workflow |
| 8 | `checkRolePermissionFromDB` 真正查 DB | **P1** | 1 天 | RBAC |
| 9 | `rbac.go` 拆分为 4 个文件 | **P1** | 1 天 | RBAC |
| 10 | 合并 `loadRolePermissionsFromDB` 和 `loadPermissionsFromDB` 重复逻辑 | **P1** | 1 天 | RBAC |
| 11 | `checkTicketOwnership` 增加缓存 | **P1** | 1 天 | RBAC |
| 12 | `ListCIs` 增加模糊搜索 | **P1** | 1 天 | CMDB |
| 13 | CI 关系创建增加 tenantID 校验 | **P1** | 0.5 天 | CMDB |
| 14 | BPMN ParallelGateway/InclusiveGateway 支持 | **P1** | 10 天 | Workflow |
| 15 | `getDefaultAssigntee` 重构为基于配置的分配策略 | **P1** | 3 天 | Workflow |
| 16 | `root_cause_analysis_service.go`: TODO 集成 AI 分析服务 | **P2** | 5 天 | AI |
| 17 | `sla_policy_service.go`: TODO 实现完整业务时间计算 | **P2** | 3 天 | SLA |
| 18 | `cloud_discovery_service.go`: 注释掉的 AWS SDK 配置 | **P2** | 2 天 | CMDB |
| 19 | 日志规范统一（fmt.Printf → zap） | **P2** | 2 天 | 全局 |
| 20 | DTO 统一分离到 `dto/` 目录 | **P2** | 1 天 | CMDB |
| 21 | 前后端权限模块对齐 | **P2** | 2 天 | RBAC |
| 22 | 前端 filters debounce + 批量删除接口 | **P2** | 1 天 | Workflow |

---

## 7. 改进路线图

### Q3 2026（7-9 月）：安全加固 & 紧急修复

**主题**: 消除 P0 安全漏洞，修复核心数据一致性问题

| 周次 | 任务 | 负责模块 | 优先级 |
|------|------|----------|--------|
| W1-W2 | 修复 `ListRelationships` tenantID 过滤 + CI 关系 tenantID 校验 | CMDB | P0 |
| W2-W3 | 修复 `evaluateCondition` 默认拒绝策略 + 表达式沙箱 | Workflow | P0 |
| W3-W4 | 修复 `EvaluateACLScript` 默认拒绝 + 基础表达式执行 | RBAC | P0 |
| W4-W5 | 工单审批流程事务保护 + BPMN 变量合并原子化 | Workflow | P1 |
| W5-W8 | `TicketWorkflowService` 逐步从 raw SQL 迁移到 Ent ORM | Workflow | P1 |
| W6-W8 | `checkRolePermissionFromDB` 修复 + rbac.go 拆分 + 合并重复函数 | RBAC | P1 |

**交付物**: 安全漏洞全部修复、核心操作事务化、ORM 一致性恢复

### Q4 2026（10-12 月）：功能完善 & 架构优化

**主题**: 补全核心功能缺失，提升引擎可用性

| 周次 | 任务 | 负责模块 | 优先级 |
|------|------|----------|--------|
| W1-W4 | ServiceTask 真正执行 Handler + 错误边界事件 | Workflow | P1 |
| W3-W5 | 任务分配策略重构（配置化替代中文关键词匹配） | Workflow | P1 |
| W5-W8 | BPMN ParallelGateway + InclusiveGateway 实现 | Workflow | P1 |
| W1-W3 | `ListCIs` 模糊搜索 + 前后端权限模块对齐 | CMDB+RBAC | P1 |
| W3-W5 | `checkTicketOwnership` 缓存 + 权限变更审计日志 | RBAC | P1-P2 |
| W6-W8 | 日志规范统一 + DTO 分离 + 前端 debounce/批量删除 | 全局 | P2 |

**交付物**: BPMN 引擎支持并行/包容网关、任务分配配置化、搜索能力增强

### Q1 2027（1-3 月）：技术债务清理 & 质量提升

**主题**: 清理遗留技术债务，提升测试覆盖率

| 周次 | 任务 | 负责模块 | 优先级 |
|------|------|----------|--------|
| W1-W3 | AI 分析服务集成（root_cause_analysis） | AI | P2 |
| W3-W5 | SLA 业务时间计算完善 | SLA | P2 |
| W5-W6 | AWS SDK 配置恢复（cloud_discovery） | CMDB | P2 |
| W1-W8 | 核心模块单元测试补充（BPMN 引擎 / RBAC / CMDB） | 全局 | P2 |
| W6-W8 | CI 关系表增加 tenant_id 字段 + 数据迁移 | CMDB | P2 |

**交付物**: 技术债务清零、测试覆盖率 > 60%、多租户隔离完善

---

## 附录 A: 评审方法说明

本次评审采用以下方法：

1. **静态代码分析**: 逐文件阅读关键源码，识别安全漏洞、架构缺陷、代码异味
2. **架构模式比对**: 以 ITIL/ITSM 行业标准和 Go 项目最佳实践为基准
3. **数据流追踪**: 追踪关键操作（启动流程/审批/权限检查）的完整调用链
4. **前后端一致性检查**: 比对前后端字段定义、权限模块、API 路径

## 附录 B: 评审范围

### 后端文件（已评审）

| 文件 | 行数 | 关注点 |
|------|------|--------|
| `service/bpmn_process_engine.go` | 1802 | BPMN 引擎核心 |
| `service/ticket_workflow_service.go` | 717 | 工单流转 |
| `service/bpmn_service_task_adapter.go` | 136 | ServiceTask 适配 |
| `service/expression_engine.go` | 330 | 表达式引擎 |
| `service/cmdb_service.go` | 236 | CMDB 核心 |
| `service/ci_relationship_service.go` | 757 | CI 关系服务 |
| `middleware/rbac.go` | 1186 | RBAC 中间件 |
| `middleware/smart_permission.go` | 459 | 智能权限 |
| `service/role_service.go` | 562 | 角色服务 |

### 前端文件（已评审）

| 文件 | 行数 | 关注点 |
|------|------|--------|
| `app/(main)/workflow/page.tsx` | 1358 | 工作流管理 |
| `app/(main)/admin/roles/page.tsx` | 670 | 角色管理 |
| `app/(main)/admin/permissions/page.tsx` | 728 | 权限配置 |
| `components/cmdb/CSDMHub.tsx` | - | CMDB 中心 |
| `components/cmdb/CIList.tsx` | - | CI 列表 |
| `components/cmdb/CIDetail.tsx` | - | CI 详情 |
| `components/cmdb/TopologyGraph.tsx` | - | 拓扑图 |

---

*本报告基于代码仓库当前快照生成，部分问题可能在后续迭代中已修复。建议每季度进行一次架构评审，持续跟踪改进进展。*
