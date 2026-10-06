# ADR-006：权限判定链路规范

> Status: proposed

## 状态

Proposed（2026-10-06）

## 背景

ITSM 后端的权限判定由两条入口路径与四层判定逻辑组成，但在实际演进中暴露出链路不透明、日志不完整、缓存一致性风险等问题：

### 1. 两条入口路径

```
RBACMiddleware (rbac.go:352)
  ├─ 每条认证请求必经
  ├─ 加载用户实体、物化角色（主角色 + m2m 附加角色）
  ├─ 调用 hasPermission → SmartCheckPermission
  └─ 路径预检：/auth/menus、/capabilities 硬编码直通

RequirePermission (rbac.go:549)
  ├─ 每条业务路由显式声明
  ├─ 调用 AuthorizeResource
  └─ AuthorizeResource → loadPermissionsByMode → checkPermissionMatch
```

两条路径最终都汇聚到 `AuthorizeResource`（rbac.go:818），但 `SmartCheckPermission` 的四层判定逻辑与 `AuthorizeResource` 的直接判定逻辑并存，导致判定链路不透明。

### 2. SmartCheckPermission 四层判定（smart_permission.go:107）

```
L1: Auth whitelist（硬编码 map：/auth/login, /auth/register, /health）
  ↓ 未命中
L2: Database ACL（endpoint_acls 表，按租户缓存，TTL 5 分钟）
  ├─ 加载：loadACLsFromDB（Ent 查询，租户隔离）
  ├─ 缓存：aclCache[tenantID]，过期时间 5 分钟
  └─ 匹配：matchACL（精确 + 通配符，按 priority 排序）
  ↓ 未命中
L3: URL auto-inference（ResourceActionMap + methodToAction）
  ├─ getPermissionFromPath：精确匹配 → 通配符匹配（最具体优先）
  ├─ methodToAction：GET→read, POST/PUT/PATCH→write, DELETE→delete
  └─ checkRolePermissionFromDB → AuthorizeResourceForRole
  ↓ 未命中
L4: Role defaults（RoleDefaultPermissions，DBOnly 模式下禁用）
  └─ PermissionConfig.Mode == DBOnly → return false
```

### 3. AuthorizeResource 直接判定（rbac.go:818）

```
super_admin 任一角色命中 → return true
  ↓
for each role:
  loadPermissionsByMode(ctx, client, role, tenantID)
  ├─ DBOnly:
  │   ├─ unavailable (DB 不可用) → return nil (fail-closed)
  │   ├─ unconfigured (DB 无该角色行) → RoleDefaultPermissions 兜底
  │   └─ configured (DB 有该角色行) → 返回 DB 权限集（含空集=显式撤销）
  ├─ HardcodeOnly → RoleDefaultPermissions
  ├─ Merge → DB ∪ defaults（并集去重）
  └─ Fallback → DB first, then defaults
  ↓
checkPermissionMatch(permissions, resource, action)
  ├─ *:* → true
  ├─ resource:* → true
  ├─ resource:admin → true
  └─ resource:action → true
```

### 4. 已识别的问题

**A. 判定链路不透明**

`SmartCheckPermission` 的四层判定中，只有 L1/L2/L3 的部分分支有 Debug 日志，L4 完全无日志。当权限被拒绝时，无法从日志判断是哪一层拒绝、为何拒绝。`RequirePermission` deny 时有 Warn 日志（rbac.go:597），但不包含判定链路信息。

**B. L1 whitelist 硬编码不可配置**

`authWhitelist` map（smart_permission.go:164）硬编码 4 条路径，新增公开端点必须改代码。`hasPermission`（rbac.go:646）又硬编码 `/auth/menus` 和 `/capabilities` 直通，与 L1 重复且分散。

**C. L2 endpoint_acls 表生产使用情况不明**

`endpoint_acls` 表存在且有完整的加载/缓存/匹配逻辑，但生产环境是否实际使用该表存储 ACL 规则不明。若未使用，L2 层形同虚设。

**D. L3 URL inference 的动作推断局限**

`methodToAction` 按 HTTP 方法推断动作（GET→read, POST→write），但动作型接口（如 `POST /tickets/:id/assign`）无法按方法推断，必须依赖 `ResourceActionMap` 的显式映射。若映射缺失，L3 会推断出错误的 action。

**E. super_admin 直通在多处重复**

`hasPermission`（rbac.go:655）、`SmartCheckPermission`（smart_permission.go:118）、`AuthorizeResource`（rbac.go:822）三处都检查 `super_admin` 直通，逻辑重复且不一致风险。

**F. 缓存一致性风险**

权限缓存（`permissionCache`）与 ACL 缓存（`aclCache`）都支持本地失效 + Redis 广播，但无强一致性保证。多副本部署下，缓存失效广播延迟可能导致不同实例的判定结果短暂不一致。

**G. DBOnly 三态语义复杂**

`loadPermissionsFromDBDBOnlyState` 区分 unavailable/unconfigured/configured 三态，unconfigured 走 `RoleDefaultPermissions` 兜底。但"unconfigured"的判定逻辑（DB 查询无该角色行）与"角色存在但未分配权限"难以区分，可能导致意外兜底。

## 决策

### D1. 权限判定链路必须可观测

**规则**：
- `SmartCheckPermission` 的每一层判定（L1-L4）必须记录结构化日志，包含 `tenant_id`、`user_id`、`roles`、`resource`、`action`、`layer`、`result`（granted/denied/skip）
- 权限拒绝必须记录拒绝原因（未命中哪一层、ResourceActionMap 是否包含该路径）
- 日志级别：L1/L2/L3 grant = Debug，L4 deny = Info，所有 deny = Warn
- 日志不得包含密码、JWT、API key 等敏感信息

**迁移**：
- 在 `SmartCheckPermission` 的每一层入口与出口补日志
- `RequirePermission` deny 日志补 `layer` 字段，标明是哪一层拒绝

### D2. L1 whitelist 必须集中管理

**规则**：
- 所有公开端点（不需要认证）必须在 `authWhitelist` map 中声明
- `hasPermission` 中的硬编码直通（`/auth/menus`、`/capabilities`）必须移入 `authWhitelist` 或论证为何不需要
- 新增公开端点必须修改 `authWhitelist`，禁止在 `hasPermission` 或其他位置硬编码

**迁移**：
- 审查 `hasPermission` 中的硬编码直通，移入 `authWhitelist` 或补充论证注释
- CI 守卫：禁止在 `hasPermission` 中新增硬编码路径直通

### D3. L2 endpoint_acls 的使用必须明确

**规则**：
- 若生产环境使用 `endpoint_acls` 表，必须在文档中说明使用场景与配置方式
- 若生产环境不使用 `endpoint_acls` 表，必须在代码注释中标明 L2 层为保留能力、未启用，并考虑是否移除或简化

**当前状态**：待确认生产环境是否使用 `endpoint_acls` 表。

### D4. L3 URL inference 必须与 ResourceActionMap 保持同步

**规则**：
- 所有动作型接口（非 CRUD 的 REST 端点，如 `/tickets/:id/assign`）必须在 `ResourceActionMap` 中显式映射
- `methodToAction` 的推断逻辑只用于标准 CRUD 端点（`/api/v1/{resource}` 与 `/api/v1/{resource}/:id`）
- 新增路由后必须执行 `go run ./cmd/authz-gen` 重新生成 `rbac_precheck_gen.go`，确保 `ResourceActionMap` 完整

**迁移**：
- 审查所有路由，确认动作型接口在 `ResourceActionMap` 中有显式映射
- CI 守卫 `TestPrecheckMapIsFresh` 与 `TestNoPermissionDeclarationIsDropped` 必须通过

### D5. super_admin 直通必须收敛到单一位置

**规则**：
- `super_admin` 直通逻辑只在 `AuthorizeResource`（rbac.go:818）中检查一次
- `hasPermission` 与 `SmartCheckPermission` 中的 `super_admin` 检查移除，统一委托 `AuthorizeResource`

**迁移**：
- 移除 `hasPermission`（rbac.go:655）与 `SmartCheckPermission`（smart_permission.go:118）中的 `super_admin` 检查
- 确保 `AuthorizeResource` 的 `super_admin` 直通逻辑正确

### D6. 缓存失效必须保证最终一致性

**规则**：
- 权限缓存与 ACL 缓存的失效必须通过 Redis 广播保证多副本最终一致
- 缓存 TTL 不得超过 5 分钟，避免长时间持有过期权限
- 关键权限变更（如角色权限修改、用户角色调整）必须同步失效相关缓存

**当前状态**：已有本地失效 + Redis 广播机制（`InvalidateRolePermissionCache`），但无强一致性保证。生产环境需确认 Redis 广播是否正常工作。

### D7. DBOnly 三态语义必须文档化

**规则**：
- `loadPermissionsFromDBDBOnlyState` 的三态（unavailable/unconfigured/configured）必须在文档中明确说明
- "unconfigured"的判定逻辑必须清晰：DB 查询无该角色行 vs 角色存在但未分配权限
- 生产环境上线前必须确认所有角色的 `role_permissions` 行已初始化，避免意外走 `RoleDefaultPermissions` 兜底

**迁移**：
- 在 `loadPermissionsFromDBDBOnlyState` 函数注释中补充三态语义说明
- 初始化/seed 流程必须为所有内置角色创建 `role_permissions` 行（即使为空集）

## 后果

### 正面

1. **可观测性**：权限判定链路每一层都有日志，排查权限问题时可快速定位
2. **一致性**：super_admin 直通收敛到单一位置，消除重复逻辑
3. **可维护性**：whitelist 集中管理，新增公开端点只需修改一处
4. **安全性**：缓存失效保证最终一致，DBOnly 三态语义清晰

### 负面

1. **日志量增加**：每层判定都记录日志会增加日志量，需配置合适的日志级别
2. **迁移工作量**：需审查并补齐所有缺失的日志、whitelist、ResourceActionMap 映射
3. **缓存一致性成本**：Redis 广播增加网络开销，但可接受

### 中性

1. **L2 endpoint_acls 的定位**：需确认生产环境是否使用，若不使用可考虑移除
2. **DBOnly unconfigured 兜底**：保留作为新装/小租户的容错，但必须文档化

## 关联

- ADR-004：权限词表统一（动作词表收敛为 read/write/delete/admin）
- ADR-005：路由鉴权规范（路由必须显式声明 RequirePermission）
- `middleware/smart_permission.go`：SmartCheckPermission 四层判定
- `middleware/rbac.go`：AuthorizeResource、RequirePermission、loadPermissionsByMode
- `middleware/precheck_codegen.go` / `rbac_precheck_gen.go`：ResourceActionMap 代码生成
- `internal/authz/roles.go`：RoleDefaultPermissions 单一真源
