# ADR-004：权限词表统一

> Status: proposed

## 状态

Proposed（2026-10-06）

## 背景

### 现状：三套并行权限词表

系统当前存在三套独立的权限标识集合，各自维护、缺乏统一治理：

| 词表 | 位置 | 用途 | 条目数 |
|:---|:---|:---|:---|
| 菜单基线 | `pkg/menubaseline/baseline.go` | 种子菜单的 `PermissionCode` 字段，决定菜单可见性 | 95 条 |
| 路由声明 | `router/*.go` 中的 `RequirePermission(resource, action)` | 路由级鉴权，决定接口是否可访问 | ~120 条 |
| 授权码空间 | `internal/authz/catalog.go` → `Definitions()` | DB 种子权限码，决定角色可授予哪些 `(resource, action)` | ~110 条 |

三套词表之间没有自动同步机制。当路由使用 `RequirePermission("ticket_type", "manage")` 而码空间只注册了 `ticket_type:write` 时，该路由对除 `super_admin` 外的所有角色永久 403（此类问题已在 2026-09-17 P0 越权收口批次中部分修复，但根因未消除）。

### 别名解析只在菜单层生效

`service/menu_service.go:574-619` 维护了两张硬编码别名表：

```go
// actionAliasMap（12 条）
"view" → "read", "use" → "read", "manage" → "admin",
"create" → "write", "update" → "write",
"approve" → "admin", "analyze" → "read", "audit" → "read",
"config" → "write", "request" → "read", "access" → "read"

// resourceAliasMap（9 条）
"service" → "service_catalog", "workflow" → "bpmn",
"helpdesk" → "ticket", "approval" → "permission",
"department" → "org", "team" → "org", "tenant" → "org",
"system" → "system_config", "report" → "report"
```

这两张表**只在 `GetUserMenus` 的菜单过滤路径中调用**（`menu_service.go:467/484-485`）。路由层的 `RequirePermission` → `AuthorizeResource` → `checkPermissionMatch`（`middleware/rbac.go:549/818/761`）是**纯精确匹配**，不做任何别名解析。

后果：菜单可见性与接口可达性使用不同的匹配规则。一个用户可能看到菜单但无法调用接口（菜单别名解析通过，路由精确匹配失败），或反过来。

### Action 方言碎片化

路由层实际使用的 action 至少存在 5 种方言：

| 方言 | 使用场景 | 路由示例 |
|:---|:---|:---|
| `read` / `write` / `delete` | 主流资源 CRUD | `RequirePermission("ticket", "read")` |
| `create` / `update` | 工单标签、分类、模板、部门、通知 | `RequirePermission("ticket_tag", "create")` |
| `manage` | 工单类型 | `RequirePermission("ticket_type", "manage")` |
| `approve` | 变更、服务请求、发布审批 | `RequirePermission("change", "approve")` |
| `archive` | 工单类型删除 | `RequirePermission("ticket_type", "archive")` |

`checkPermissionMatch` 只认精确 `(resource, action)` 对和 `admin` 通配。`create` 不等于 `write`，`manage` 不等于 `admin`，`approve` 不等于 `admin`。路由声明了 `manage`，码空间只注册了 `write`，则 403。

### Resource 重复注册

`authz/catalog.go` 中存在 6 组重复资源对——别名资源和规范资源同时作为独立权限注册：

| 别名资源 | 规范资源 | 重复行号 |
|:---|:---|:---|
| `service:read/write` | `service_catalog:read/write/delete` | 108-112 |
| `department:read/write` | `org:read/write` | 136-155 |
| `team:read/write` | `org:read/write` | 139-155 |
| `approval:read/write` | `permission:read` | 142-203 |
| `workflow:read/write` | `bpmn:read/write/delete` | 145-211 |
| `system:read/write` | `system_config:read/write` | 152-221 |

重复注册导致角色授权界面出现语义相同的两个选项（如「查看服务」和「查看服务目录」），管理员无法判断应授予哪一个，且两者互不通用——授予 `service:read` 不能访问要求 `service_catalog:read` 的路由。

### 2026-09-17 守卫补齐的遗留

`catalog.go:190-238` 的三次 P0 补码批次（注释标注「DB 已有但种子清单缺失」「路由已引用但码空间缺失」）本质都是同一根因的症状修复：路由声明和码空间没有共享单一事实来源，每次新增路由都可能引入新的 403 缺陷。

## 决策

### 1. 确立规范词表（Canonical Vocabulary）

**规范 Action 集**（封闭、不可随意扩展）：

| 规范 Action | 语义 | 吸收的方言 |
|:---|:---|:---|
| `read` | 查看/列表/详情 | view, use, analyze, audit, request, access |
| `write` | 创建+更新（非破坏性写入） | create, update, config |
| `delete` | 删除/归档/停用 | archive |
| `admin` | 资源级管理配置 | manage, approve |

`approve` 不作为独立 action 存在——审批是业务动作，由领域状态机校验；权限层面审批等同于对该资源的 `admin`（或单独建模为 `approve` 但需与 `admin` 互通，此选项待评估）。

**规范 Resource 集**：以 `authz/catalog.go` 为基础，每个业务概念只有一个规范名称。别名资源不再独立注册，改为在迁移阶段统一替换。

### 2. 别名解析下沉到中间件

将 `actionAliasMap` / `resourceAliasMap` 从 `service/menu_service.go` 提升到 `middleware/rbac.go` 的 `checkPermissionMatch` 入口。路由声明和菜单基线可以继续使用别名，但鉴权时统一解析为规范名后再匹配。

这保证：
- 菜单可见性和接口可达性使用相同的匹配规则
- 历史路由声明不需要一次性全量改名
- 新增代码鼓励使用规范名，别名只用于兼容

### 3. 消除 authz/catalog.go 重复注册

对 6 组重复资源对，保留规范资源、删除别名资源。已授予别名资源的角色数据在迁移脚本中自动转换为规范资源。

| 保留 | 删除 | 迁移 |
|:---|:---|:---|
| `service_catalog:*` | `service:*` | `role_permissions` 中 `service` → `service_catalog` |
| `org:*` | `department:*`, `team:*` | `role_permissions` 中 `department`/`team` → `org` |
| `permission:*` | `approval:*` | `role_permissions` 中 `approval` → `permission` |
| `bpmn:*` | `workflow:*` | `role_permissions` 中 `workflow` → `bpmn` |
| `system_config:*` | `system:*` | `role_permissions` 中 `system` → `system_config` |

### 4. 路由声明逐步对齐规范词表

不要求一次性修改所有路由。分阶段推进：

- **Phase 1**（本 ADR 批准后）：别名解析下沉中间件，消除 403 隐患；catalog.go 重复注册清理 + 数据迁移。
- **Phase 2**：新路由必须使用规范 `(resource, action)`；CI 守卫 `checkPermissionMatch` 增加「别名检测」，路由使用非规范 action 时告警（不阻断）。
- **Phase 3**：逐域迁移存量路由声明，从高频域（ticket、change、service_request）开始。每个域的迁移是一个独立 commit，包含路由改名 + 守卫基线更新。
- **Phase 4**：别名表冻结，不再新增映射。当所有路由和菜单基线都使用规范名后，别名表可移除。

### 5. CI 守卫增强

在 `middleware/precheck_freshness_test.go` 的现有守卫基础上增加：

- **G1 子集校验**：路由声明的 `(resource, action)` 经别名解析后 ⊆ `authz/catalog.go` 规范码空间。
- **G2 重复检测**：`catalog.go` 不允许同时注册别名资源和规范资源（如 `service:read` 和 `service_catalog:read` 不可共存）。
- **G3 方言告警**：路由使用非规范 action（`create`/`update`/`manage`/`approve`/`archive`）时输出告警日志，不阻断 CI。

## 后果

### 正面

- **消除 403 隐患**：路由声明和码空间的方言不匹配不再导致静默 403。
- **管理员体验清晰**：角色授权界面不再出现语义重复的选项。
- **可审计性提升**：权限码空间收敛后，权限授予、菜单可见性、路由可达性三者可交叉验证。
- **向后兼容**：别名解析下沉后，历史路由和菜单声明不需要立即修改。

### 负面

- **迁移风险**：`role_permissions` 数据迁移需要精确匹配租户、角色和权限码，迁移失败可能导致角色权限丢失。必须可回滚（保留旧码数据或提供反向迁移）。
- **别名表维护成本**：过渡期别名表同时服务于菜单和路由，新增映射需要评估影响范围。
- **Phase 3 工程量大**：全量路由改名涉及 ~120 条声明，需要逐域验证。

### 风险缓解

- Phase 1 的 catalog.go 清理必须配套数据迁移脚本和回滚 SQL。
- Phase 3 按域拆分，每个域独立 PR、独立回归测试。
- 别名表变更必须同步更新本 ADR 的映射清单和 CI 守卫基线。

## 关联

- [ADR-001](./adr-001-modular-monolith.md)：模块化单体决策，权限词表统一是其 RBAC 子系统的细化。
- [AGENTS.md](../../AGENTS.md)「身份、租户与数据范围」和「能力状态与失败语义」章节定义了权限系统的强制约束。
- `middleware/precheck_freshness_test.go`：现有路由-码空间守卫，本 ADR 的 G1-G3 增强基于此文件。
- `internal/authz/catalog.go`：权限码空间权威清单。
- `pkg/menubaseline/baseline.go`：菜单基线权威清单。
