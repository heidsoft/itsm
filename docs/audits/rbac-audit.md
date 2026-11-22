# RBAC、权限数据与初始化基线审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`

## 1. 数据模型职责边界（已确认）

| 模型 | 职责 | 证据 |
|:---|:---|:---|
| `Permission` | 权限码运行期真源，含 `resource`/`action`/策略信封，`tenant_id` 必填 | `ent/schema/permission.go:17-53`、`internal/authz/registry.go:184-252` |
| `RolePermission` | 纯 M2M，`(role_id,permission_id,tenant_id)` 唯一索引 | `ent/schema/role_permission.go:16-35` |
| `Role` | 角色身份 + `data_scope` + `is_system`；授权边 | `ent/schema/role.go:17-59` |
| `Menu` | 纯展示/筛选元数据，`permission_code` 无 FK，**不参与 API 授权** | `ent/schema/menu.go:15-68`、`menu_service_msp_menu_test.go:135-141` |

`PermissionDefinition` 是死模型（`rbac.go:237-242` 注释自认"目前尚无任何消费方"），且唯一键 `(resource,action)` 不含租户。
Go 目录 `authz.Definitions()` 与 DB `permissions` 的分工是显式设计，`Registry.ValidateBootstrap` 校验子集关系（`registry.go:255-284`），不是重复权限体系。

## 2. 授权决策链

`router/router.go:425-443`（Audit → Auth → RBAC → CSRF）→ `AuthMiddleware`（token/revocation/MinIssuedAt）→ `RBACMiddleware` 每请求重载用户与角色（`rbac.go:531-562`）→ `SmartCheckPermission` L3 URL 推断 → `RequirePermission`（`rbac.go:635-703`）→ `AuthorizeResource`（`:904-929`）→ DB 加载（`loadPermissionsFromDB`）→ `checkPermissionMatch`（`:847-873`）。
`data_scope` 由领域服务执行，中间件不处理——这是分层选择，本身不是 fail-open。

## 3. 发现项

```yaml
id: AUDIT-020
priority: P1
category: authz/policy
status: resolved
title: 权限停用（IsActive=false）未作用于人工 RBAC 判定路径
file: internal/authz/policy.go:131-135；middleware/rbac.go:847-873
evidence: IsActive 注释承诺"关闭后该码对鉴权不可见"；DeletePermission 仅把信封 IsActive=false；Registry.Lookup/AuthorizePrincipal 尊重它，但只服务 AI 执行链；人工路径直接读 Permission.Resource/Action 行，从不解码信封
reproduction: 管理员停用权限 X，角色仍持 role_permissions 行，人工用户仍可通过 RequirePermission(X.resource, X.action)；仅 AI 主体被拒
impact: "停用/删除"不能撤销人工访问，与契约相反
fix: 在 loadPermissionsFromDB 解码信封并跳过 IsActive=false；策略写入时失效中间件缓存；加真实路由契约测试
approval: 权限语义变更，需人工审批
```

```yaml

已验证修复（基准 c580b179）：middleware/rbac.go 的 permissionFromEntRow 在人工判定路径解码
permissions.description 的信封并跳过 IsActive=false；纯文本存量行保持旧语义，损坏信封 fail-closed。
UpdatePermissionPolicy 与 DeletePermission 同步调用 InvalidateAllPermissionCachesEx 清理人工权限缓存。
回归 middleware/permission_active_test.go 用真实 ent 库 + AuthorizeResource 锁住停用即拒绝、
重新启用恢复、纯文本兼容、损坏信封拒绝；反向注入后断言变红。go test ./middleware/ 全包通过。
跨副本广播仍未解决（AUDIT-022）。

id: AUDIT-021
priority: P1
category: authz/dbonly
status: confirmed (代码路径) / suspected (可达性)
title: DBOnly 下 L3 URL 推断回退到编译期角色默认值
file: middleware/smart_permission.go:355-386,407-423；middleware/rbac.go:148-150
evidence: checkURLInference 在 getPermissionFromPath 返回 nil 时调用 hasResourcePermissionFromRole→RoleDefaultPermissions，无 Mode==DBOnly 门禁；L4 有该门禁；route_precheck_alignment_test.go:40 明确跳过未映射路径，故没有全局守卫强制所有路由进 map
impact: 对既无预检映射又无路由级 RequirePermission 的路径，DB 撤销可能不生效；多数路由有 RequirePermission 二次判定，实际可利用性未证实
fix: 在 checkURLInference 的回退分支加 DBOnly 门禁或改走 DB 感知的 AuthorizeResource；补真实 ent client + 未映射路径的负例
approval: 权限判定变更，需人工审批
```

```yaml
id: AUDIT-022
priority: P1
category: authz/cache
status: confirmed
title: 三类缓存只有一类广播失效；Endpoint ACL 无失效调用点
file: middleware/smart_permission.go:82-96；internal/authz/registry.go:122-134；middleware/permission_cache_broadcast.go:43-73
evidence: 角色权限缓存有 Redis 广播（rbac.go:875-882 + role_service_impl.go:297,409,497）；Registry 与 ACL 缓存仅本地清空；InvalidateACLCache 全仓无生产调用方；EnableHotReload/ReloadInterval 无消费者
impact: 多副本策略变更最长 5 分钟不一致；ACL 路由映射变更最长 5 分钟不生效
fix: 把广播扩展到 Registry/ACL 两类缓存，或在写入路径调用 InvalidateTenantACLCache；删除无效的 hot-reload 配置
```

```yaml
id: AUDIT-023
priority: P2
category: data-integrity
status: confirmed
title: permissions / roles 缺 (tenant_id, code) DB 唯一约束
file: ent/schema/permission.go:17-53；ent/schema/role.go:17-59
evidence: RolePermission 有唯一索引，permissions/roles 没有；唯一性靠 Exist-then-Create；Registry 用 TestRegistry_DuplicateCodeFailsClosed 在读侧兜底
impact: 并发或手工插入重复码会导致整个租户权限目录 fail-closed（可用性，不是提权）
fix: 版本化迁移加唯一索引 + 存量重复检测/清理，冲突映射 409
approval: 数据库结构改动，需审批
```

```yaml
id: AUDIT-024
priority: P2
category: seeder
status: confirmed
title: seedMenuAndPermissionFixes 回退清单写入 resource="groups" 的纯文本权限
file: pkg/seeder/seeder.go:2113-2151
evidence: catalog.go:132 定义 group:read 的 resource 为 "group"；回退清单用 "groups" 且明文描述；正常流程因 seedPermissions 先执行而不会触发
impact: 若某 provisioning 路径跳过 seedPermissions，可写出 code≠resource:action 的行，Registry 对整租户 fail-closed
fix: 删除该回退清单（四个码已在 Definitions()）或从 catalog 派生并写信封
```

```yaml
id: AUDIT-025
priority: P2
category: seeder/admin-edit
status: confirmed (设计如此)
title: 重复初始化会覆盖托管菜单的 name/icon/sort/permission_code
file: pkg/seeder/seeder.go:1966-1992
evidence: 只保护 IsVisible/IsEnabled；其余字段每次 upsert 覆盖；菜单只影响显示，不影响 API 授权
fix: 要么把 permission_code 纳入操作员所有权，要么明确文档化保护字段清单
```

## 4. 已验证的正例（不要重复整改）

- sysadmin 全量派生（`roles.go:488-499` + `catalog_test.go:74-107`）。
- 权限策略信封在重复初始化时跳过管理员修改（`seeder.go:1872-1894`）。
- 角色授权只增不减，唯一收缩通道是显式退役清单（`roles.go:501-533`、`seeder.go:2244-2272`）。
- 角色移除即时生效：token revocation + 缓存失效 + fail-closed（`user_service.go:374-437`、`permission_cache_poison_test.go:114-132`）。
- 加载失败不污染缓存（`permission_cache_poison_test.go:19-62`）。
- 跨租户角色/权限关联拒绝（`handler_test.go:544-562`）。
- 21 个已知死码由棘轮锁住只降不升（`catalog_dead_code_advisory_test.go:18-23`）。

## 5. 缺失的负例测试

- 跨租户批量用户操作集成测试（目前只有 mock）。
- DBOnly + 真实 ent client + 未映射路径的 L3 回退负例。
- 人工 RBAC 路径对 `IsActive=false` 的 403 断言。
- 跨实例缓存广播的 publish/subscribe 端到端。
- `AssignPermissions` 的 Delete+CreateBulk 失败回滚测试。
