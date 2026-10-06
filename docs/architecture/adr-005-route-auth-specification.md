# ADR-005：路由鉴权规范

> Status: proposed

## 状态

Proposed（2026-10-06）

## 背景

ITSM 后端的路由鉴权架构由三层路由组 + 四层权限判定组成，但在实际演进中暴露出若干缺口与不一致：

### 1. 三层路由组架构

```
public (/api/v1)
  └─ 无认证中间件，公开端点（login/register/health）

auth (/api/v1)
  ├─ AuthMiddleware（JWT 验证）
  ├─ RBACMiddleware（加载用户角色到上下文）
  └─ 部分路由缺少 RequirePermission

tenant (extends auth)
  ├─ TenantMiddleware（租户隔离）
  ├─ AuditMiddleware（操作审计）
  └─ 所有业务路由，要求 RequirePermission

msp (/api/v1/msp)
  ├─ AuthMiddleware + RBACMiddleware
  ├─ MSPMiddleware（跨租户访问）
  └─ 部分路由缺少 RequirePermission
```

### 2. 四层权限判定（SmartCheckPermission）

```
L1: auth whitelist（/auth/login, /auth/register, /health）
  ↓ 未命中
L2: DB ACL（role_permissions 表，租户差异化）
  ↓ 未命中
L3: URL auto-inference（ResourceActionMap，代码生成）
  ↓ 未命中
L4: role defaults（硬编码默认权限，DBOnly 模式下禁用）
```

生产环境 `PermissionConfig.Mode = DBOnly`（`middleware/rbac.go:108`），L4 完全不参与授权，未命中 L1-L3 即拒绝。

### 3. 已识别的缺口

**A. auth 组路由缺少 RequirePermission**

```go
// router/router.go:376
auth.GET("/capabilities", capability.Handler)  // 无 RequirePermission

// router/router.go:416
auth.POST("/ws/ticket", ...)  // 无 RequirePermission
```

这两条路由只经过 AuthMiddleware + RBACMiddleware，任何已登录用户均可访问，未按资源/动作粒度授权。

**B. MSP /status 路由缺少 RequirePermission**

```go
// router/msp_routes.go:22
msp.GET("/status", config.MSPHandler.GetMSPStatus)  // 无 RequirePermission
```

MSP 组其他路由均使用 `RequireMSPPermission`，唯独 `/status` 只挂了组级中间件，任何 MSP 租户的已登录用户均可访问。

**C. CMDB 组级 RequirePermission 的隐式继承**

```go
// router/cmdb_routes.go:19-20
configurationItems := auth.Group("/configuration-items")
configurationItems.Use(middleware.RequirePermission("cmdb", "read"))  // 组级基线

// 后续路由部分显式声明，部分依赖组级
configurationItems.GET("/stats", ...)  // 依赖组级 cmdb:read
configurationItems.POST("/types", middleware.RequirePermission("cmdb", "write"), ...)  // 显式覆盖
```

组级 `Use(RequirePermission)` 作为基线可接受，但 Gin 的中间件链会在组级之后继续执行路由级中间件，导致权限判定逻辑分散、不易审计。

**D. 动作词表不一致（同 ADR-004）**

路由声明中混用 `read/write/delete`（主流）、`create/update`（6+ 路由）、`manage`（ticket_type）、`approve`（change/service_request/release）、`archive`（ticket_type），与 `internal/authz/catalog.go` 的码空间定义不完全对齐。

**E. ResourceActionMap 代码生成的新鲜度风险**

`middleware/rbac_precheck_gen.go` 由 `cmd/authz-gen` 从路由声明扫描生成，若路由变更后忘记重新生成，预检映射与真实路由不同步，非 super_admin 用户的权限判定会出现偏差。已有 CI 守卫 `TestPrecheckMapIsFresh`（`middleware/precheck_freshness_test.go`）兜底，但扫描器本身对某些路由形态（如函数形参传入的 `*gin.RouterGroup`）解析不完整，历史上曾静默丢弃 21 个端点（2026-09-27 至 2026-10-04 期间）。

## 决策

### D1. 所有租户作用域路由必须显式声明 RequirePermission

**规则**：`tenant` 组下的每一条路由必须显式调用 `RequirePermission(resource, action)`，禁止依赖组级中间件作为唯一授权点。

**理由**：
- 组级 `Use(RequirePermission)` 作为"最低权限基线"可接受，但不能替代路由级显式声明
- 路由级声明是 ResourceActionMap 代码生成的数据源，缺失声明 = 预检映射不完整
- 显式声明提升可读性，审计时一眼可见每条路由的权限要求

**迁移**：
- 审查所有 `tenant` 组路由，补齐缺失的 RequirePermission
- 组级 `Use(RequirePermission)` 保留作为基线，但路由级必须显式声明（可重复声明相同权限）

### D2. auth 组路由必须枚举并论证

**规则**：`auth` 组下不经过 TenantMiddleware 的路由必须满足以下条件之一：
1. 公开端点（移入 `public` 组）
2. 跨租户系统级端点（如 `/capabilities`、`/readiness/ga`），必须在路由注册处注释论证为何不需要租户隔离
3. WebSocket 票据颁发端点（`/ws/ticket`），必须论证为何不需要 RequirePermission

**当前状态**：
- `/capabilities`：返回系统能力开关，不涉及租户数据，可保留在 auth 组，但应补充 RequirePermission("system", "read") 或论证为何不需要
- `/ws/ticket`：票据颁发端点，已登录用户均可访问，但票据本身有 TTL 且一次性使用，风险可控；应补充 RequirePermission 或论证
- `/readiness/ga`：已有 RequirePermission("system", "read")，符合规范

### D3. MSP 组路由必须统一使用 RequireMSPPermission

**规则**：`msp` 组下所有路由必须显式调用 `RequireMSPPermission(resource, action)`，禁止依赖组级中间件。

**迁移**：
- `/msp/status` 补齐 `RequireMSPPermission("msp", "read")` 或论证为何不需要

### D4. ResourceActionMap 代码生成必须与路由声明保持同步

**规则**：
- 新增/修改/删除路由权限声明后，必须执行 `go run ./cmd/authz-gen` 重新生成 `rbac_precheck_gen.go`
- CI 守卫 `TestPrecheckMapIsFresh` 必须通过
- CI 守卫 `TestNoPermissionDeclarationIsDropped` 必须通过（兜住扫描器解析不完整的风险）

**当前状态**：已有 CI 守卫，但扫描器对某些路由形态（如 `SetupTicketTypeRoutes(tenant *gin.RouterGroup)` 函数形参传入的组根）解析不完整，历史上曾静默丢弃声明。扫描器需持续增强以支持更多路由注册约定。

### D5. 动作词表必须对齐 ADR-004 规范集

**规则**：路由声明中的 action 必须使用 ADR-004 规范的四个标准词：`read`、`write`、`delete`、`admin`。

**迁移**：
- `create` → `write`
- `update` → `write`
- `manage` → `admin`
- `approve` → `admin`（或按 ADR-004 的 Phase 2 决策）
- `archive` → `write`（或按 ADR-004 的 Phase 2 决策）

### D6. 权限判定链路必须可观测

**规则**：
- SmartCheckPermission 的每一层判定（L1-L4）必须记录结构化日志，包含 tenant_id、user_id、role、resource、action、layer、result
- 权限拒绝必须记录拒绝原因（未命中哪一层、ResourceActionMap 是否包含该路径）
- 日志不得包含密码、JWT、API key 等敏感信息

**当前状态**：已有部分日志（`zap.S().Debugw("Auth whitelist match", ...)`），但 L2/L3 层判定日志不完整。

## 后果

### 正面

1. **安全收敛**：所有租户作用域路由显式授权，消除"隐式允许"风险
2. **可审计性**：路由注册处一眼可见每条路由的权限要求，无需追踪中间件链
3. **CI 守卫闭环**：代码生成 + 新鲜度测试 + 扫描完整性测试，三重兜底
4. **词表统一**：与 ADR-004 对齐，消除动作方言碎片化

### 负面

1. **迁移工作量**：需审查并补齐所有缺失的 RequirePermission，预计影响 20+ 路由文件
2. **扫描器维护成本**：`cmd/authz-gen` 需持续增强以支持新路由注册约定
3. **日志量增加**：权限判定链路完整日志会增加日志量，需配置合适的日志级别

### 中性

1. **组级 RequirePermission 的定位**：保留作为基线，但不再作为唯一授权点
2. **auth 组路由的论证义务**：新增 auth 组路由必须注释论证为何不需要租户隔离

## 关联

- ADR-004：权限词表统一（动作词表收敛为 read/write/delete/admin）
- `middleware/rbac.go`：PermissionConfigMode、RequirePermission、AuthorizeResource
- `middleware/smart_permission.go`：SmartCheckPermission 四层判定
- `middleware/precheck_freshness_test.go`：CI 守卫（TestPrecheckMapIsFresh、TestNoPermissionDeclarationIsDropped）
- `router/router.go`：三层路由组架构
- `router/msp_routes.go`：MSP 跨租户路由
- `router/cmdb_routes.go`：CMDB 组级 RequirePermission 示例
