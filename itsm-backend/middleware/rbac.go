package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"itsm-backend/common"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/user"
	"itsm-backend/internal/authz"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Permission 权限结构
type Permission struct {
	Resource string `json:"resource"` // 资源名称，如 "ticket", "user", "dashboard"
	Action   string `json:"action"`   // 操作类型，如 "read", "write", "delete", "admin"
}

type rbacRoleContextKey struct{}

// WithRBACRole 将认证中间件确认的角色写入标准请求 context，供服务层执行细粒度权限检查。
func WithRBACRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, rbacRoleContextKey{}, role)
}

// RBACRoleFromContext 返回认证中间件写入的角色。缺失时调用方必须 fail closed。
func RBACRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(rbacRoleContextKey{}).(string)
	return role, ok && strings.TrimSpace(role) != ""
}

// cachedPermission 带过期时间的缓存条目
type cachedPermission struct {
	permissions []Permission
	expiresAt   time.Time
}

// DefaultPermissionCacheTTL 默认权限缓存TTL（5分钟）
const DefaultPermissionCacheTTL = 5 * time.Minute

// PermissionCache 权限缓存（带TTL）
var (
	permissionCache     = make(map[string]*cachedPermission)
	permissionCacheLock sync.RWMutex
	permissionCacheTTL  = DefaultPermissionCacheTTL
)

// SetPermissionCacheTTL 设置权限缓存TTL
func SetPermissionCacheTTL(ttl time.Duration) {
	permissionCacheLock.Lock()
	permissionCacheTTL = ttl
	permissionCacheLock.Unlock()
}

// RoleDefaultPermissions 返回角色的编译期默认权限（DBOnly unconfigured 兜底、
// HardcodeOnly/Merge/Fallback 模式及菜单/auth 展示面的唯一来源）。
//
// 这里不独立维护硬编码表，单一真源是
// internal/authz.RolePermissionDefaults()（内置角色派生自播种码集本身，
// 兜底与播种不可能漂移；security/msp_* 等非播种角色在 authz 侧显式列出）。
func RoleDefaultPermissions(role string) []Permission {
	codes, ok := authz.RolePermissionDefaults()[role]
	if !ok {
		return nil
	}
	perms := make([]Permission, 0, len(codes))
	for _, code := range codes {
		resource, action, valid := authz.SplitPermissionCode(code)
		if !valid {
			continue
		}
		perms = append(perms, Permission{Resource: resource, Action: action})
	}
	return perms
}

// PermissionConfigMode 权限配置模式
type PermissionConfigMode int

const (
	// PermissionConfigModeDBOnly 仅使用数据库权限
	PermissionConfigModeDBOnly PermissionConfigMode = iota
	// PermissionConfigModeHardcodeOnly 仅使用硬编码权限
	PermissionConfigModeHardcodeOnly
	// PermissionConfigModeMerge 合并数据库和硬编码权限（并集）
	PermissionConfigModeMerge
	// PermissionConfigModeFallback 先数据库，失败则使用硬编码（默认）
	PermissionConfigModeFallback
)

// PermissionConfig 权限配置
var PermissionConfig = struct {
	// Mode 权限加载模式
	Mode PermissionConfigMode
	// EnableCache 是否启用缓存
	EnableCache bool
}{
	Mode:        PermissionConfigModeDBOnly, // 企业级交付：仅使用数据库权限，支持多租户差异化
	EnableCache: true,
}

// logPermissionLoadFailure 权限加载失败必须可观测：本函数只在 DB/上下文错误时调用，
// 空权限集（显式撤销）不在此列。错误原文只落安全分类，避免把 SQL 细节带进日志。
func logPermissionLoadFailure(roleName string, tenantID int, err error) {
	class := "db_error"
	switch {
	case errors.Is(err, context.Canceled):
		class = "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		class = "context_deadline"
	}
	zap.S().Warnw("RBAC: role permission load failed, request denied without caching",
		"role", roleName, "tenant_id", tenantID, "error_class", class)
}

// cacheRolePermissions 写入角色权限缓存。
// 加载失败（DB 报错、请求 ctx 取消）必须与「DB 显式配置了空权限集」区分：
// 空集在 DBOnly 下表示撤销该角色全部权限，是合法状态、可以缓存；
// 而失败时缓存空集会把整个 role+tenant 在 TTL 窗口内锁死为 deny-all。
func cacheRolePermissions(cacheKey string, perms []Permission, loadErr error) {
	if loadErr != nil && len(perms) == 0 {
		return
	}
	permissionCacheLock.Lock()
	permissionCache[cacheKey] = &cachedPermission{
		permissions: perms,
		expiresAt:   time.Now().Add(permissionCacheTTL),
	}
	permissionCacheLock.Unlock()
}

// errRoleInactive 表示角色行存在但被显式停用（roles.is_active=false）。
//
// 它与「角色行不存在」是两种不同状态，必须区分：
//   - 角色行不存在 → DBOnly 下视为「已撤销/迁移异常」，按空权限集 fail-closed；
//   - 角色行存在但停用 → 同样是显式撤销，也按空权限集 fail-closed，
//     但不能被误判成「未初始化」而回退到编译期默认权限（否则停用形同虚设）。
var errRoleInactive = errors.New("role inactive")

// lookupEffectiveRole 是「角色是否生效」的唯一判定源。
//
// 停用状态刻意不在 SQL 里用 role.IsActive 过滤：过滤后停用行会退化成
// NotFound，与「角色行不存在」不可区分，DBOnly 路径就会把「显式停用」
// 当成「未配置」并回退到编译期默认权限——停用反而变成放行。
//
// 平台超管 super_admin 的直通语义由 AuthorizeResource 单独拥有，
// 不经由本函数；原因见该函数注释。
func lookupEffectiveRole(ctx context.Context, client *ent.Client, roleName string, tenantID int) (*ent.Role, error) {
	if client == nil {
		return nil, nil
	}
	r, err := client.Role.Query().
		Where(
			role.Code(roleName),
			role.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if !r.IsActive {
		return nil, errRoleInactive
	}
	return r, nil
}

// loadRolePermissionsFromDB 从数据库加载角色的权限
// 直接从 role_permissions 联表查询，支持多租户
// ctx 由调用方传入（请求链路为请求 ctx），不再使用 context.Background()
func loadRolePermissionsFromDB(ctx context.Context, client *ent.Client, roleName string, tenantID int) ([]Permission, error) {
	// 如果 client 为 nil，直接返回空权限
	if client == nil {
		return nil, nil
	}

	cacheKey := roleName + "_" + strconv.Itoa(tenantID)

	// 先检查缓存（包括TTL检查）
	permissionCacheLock.RLock()
	if cached, exists := permissionCache[cacheKey]; exists {
		if time.Now().Before(cached.expiresAt) {
			permissionCacheLock.RUnlock()
			return cached.permissions, nil
		}
	}
	permissionCacheLock.RUnlock()

	// 从数据库加载: 通过 role_permissions 联表查询
	var perms []Permission
	var loadErr error

	// 首先查找角色ID。复用这一次查询做生效性判定：角色行存在但已停用时
	// 按显式撤销处理（空权限集 + 错误），既不回退默认权限、也不写入缓存
	// （停用是管理动作，缓存里不能留下可继续放行的权限集）。
	roleEntity, err := lookupEffectiveRole(ctx, client, roleName, tenantID)
	if errors.Is(err, errRoleInactive) {
		return nil, errRoleInactive
	}

	if err == nil && roleEntity != nil {
		roleID := roleEntity.ID

		// 直接查询 role_permissions 联表获取该角色的权限
		rolePerms, err := client.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(roleID), rolepermission.TenantID(tenantID)).
			All(ctx)
		if err != nil {
			loadErr = err
		}

		if err == nil && len(rolePerms) > 0 {
			// 提取permission_id列表
			permIDs := make([]int, len(rolePerms))
			for i, rp := range rolePerms {
				permIDs[i] = rp.PermissionID
			}

			// 查询 permissions 表获取权限详情（加 tenant 过滤）
			permsData, err := client.Permission.Query().
				Where(permission.IDIn(permIDs...), permission.TenantID(tenantID)).
				All(ctx)

			if err != nil {
				loadErr = err
			} else {
				for _, p := range permsData {
					if perm, ok := permissionFromEntRow(p); ok {
						perms = append(perms, perm)
					}
				}
			}
		}
	} else if err != nil && !ent.IsNotFound(err) {
		loadErr = err
	}

	cacheRolePermissions(cacheKey, perms, loadErr)
	if loadErr != nil {
		logPermissionLoadFailure(roleName, tenantID, loadErr)
	}

	return perms, loadErr
}

// permissionFromEntRow 把权限行转换为人工 RBAC 判定使用的 (resource, action)。
//
// 运行期策略信封里的 IsActive=false 表示管理员已停用该码。此前只有 AI 主体
// 判定链（authz.Registry）读取该字段，人工路径直接读 Resource/Action 行，
// 于是「停用」只挡住 AI，人类用户仍然放行——与权限管理界面承诺的语义相反。
//
// 兼容边界：存量纯文本描述仍按旧语义放行，避免升级后整租户权限失效；
// 以 "{" 开头但解不开的信封按 fail-closed 处理，与 Registry 的严格解码一致。
func permissionFromEntRow(p *ent.Permission) (Permission, bool) {
	row := Permission{Resource: p.Resource, Action: p.Action}
	raw := strings.TrimSpace(p.Description)
	if !strings.HasPrefix(raw, "{") {
		return row, true
	}
	_, policy, err := authz.DecodeDefinition(raw, p.Resource, p.Action)
	if err != nil || !policy.IsActive {
		zap.S().Warnw("权限定义已停用或无法解码，人工 RBAC 路径跳过该码",
			"code", p.Code, "tenant_id", p.TenantID, "error", err)
		return row, false
	}
	return row, true
}

// invalidatePermissionCache 使指定角色的缓存失效
func invalidatePermissionCache(roleName string, tenantID int) {
	cacheKey := roleName + "_" + strconv.Itoa(tenantID)
	permissionCacheLock.Lock()
	delete(permissionCache, cacheKey)
	permissionCacheLock.Unlock()
}

// InvalidateAllPermissionCaches 使所有权限缓存失效
func InvalidateAllPermissionCaches() {
	permissionCacheLock.Lock()
	clear(permissionCache)
	permissionCacheLock.Unlock()
}

// loadPermissionsFromDB 从 role_permission + permission（旧表）加载角色权限。
// 注意：权限定义的权威表是 Permission（ent/schema/permission.go）；
// permission_definition 表（ent/schema/permission_definition.go）目前
// 尚无任何消费方，如需启用须同步改造本函数与
// AssignPermissions 的写入侧。若 role_permission 无数据，则 fallback 到
// loadRolePermissionsFromDB（旧 Permission 表直查）。
// ctx 由调用方传入（请求链路为请求 ctx），不再使用 context.Background()
func loadPermissionsFromDB(ctx context.Context, client *ent.Client, roleName string, tenantID int) ([]Permission, error) {
	// 如果 client 为 nil，直接返回空权限（将使用默认权限）
	if client == nil {
		return nil, nil
	}

	cacheKey := roleName + "_" + strconv.Itoa(tenantID)

	// 先检查缓存（包括TTL检查）
	permissionCacheLock.RLock()
	if cached, exists := permissionCache[cacheKey]; exists {
		if time.Now().Before(cached.expiresAt) {
			permissionCacheLock.RUnlock()
			return cached.permissions, nil
		}
	}
	permissionCacheLock.RUnlock()

	// 从 role_permission 关联 + permission 权威表加载（见函数头注释）
	var perms []Permission
	var loadErr error

	// 首先查找角色ID
	roleEntity, err := client.Role.Query().
		Where(
			role.Code(roleName),
			role.TenantID(tenantID),
		).
		Only(ctx)

	if err == nil && roleEntity != nil {
		roleID := roleEntity.ID

		// 查询role_permission表获取该角色的权限定义ID
		rolePerms, err := client.RolePermission.Query().
			Where(rolepermission.RoleIDEQ(roleID), rolepermission.TenantID(tenantID)).
			All(ctx)
		if err != nil {
			loadErr = err
		}

		if err == nil && len(rolePerms) > 0 {
			// 提取permission_id列表
			permIDs := make([]int, len(rolePerms))
			for i, rp := range rolePerms {
				permIDs[i] = rp.PermissionID
			}

			// 查询 permissions 表获取权限详情（加 tenant 过滤）
			permsData, err := client.Permission.Query().
				Where(permission.IDIn(permIDs...), permission.TenantID(tenantID)).
				All(ctx)

			if err != nil {
				loadErr = err
			} else {
				for _, p := range permsData {
					if perm, ok := permissionFromEntRow(p); ok {
						perms = append(perms, perm)
					}
				}
			}
		}
	} else if err != nil && !ent.IsNotFound(err) {
		loadErr = err
	}

	// 如果仍然没有权限，fallback到旧的方式
	if len(perms) == 0 {
		fallbackPerms, fallbackErr := loadRolePermissionsFromDB(ctx, client, roleName, tenantID)
		perms = append(perms, fallbackPerms...)
		if fallbackErr != nil {
			loadErr = fallbackErr
		}
	}

	cacheRolePermissions(cacheKey, perms, loadErr)
	if loadErr != nil {
		logPermissionLoadFailure(roleName, tenantID, loadErr)
	}

	return perms, loadErr
}

// loadPermissionsFromDBDBOnlyState DBOnly 模式下区分三种状态，让 loadPermissionsByMode
// 决定是 fail-closed（unavailable / configured）、走硬编码兜底（unconfigured）。
//
// 三态语义（详见 loadPermissionsByMode 注释）：
//   - permissionDBOnlyUnavailable   : DB 不可用（含查询报错/ctx 取消），调用方必须 fail-closed
//   - permissionDBOnlyUnconfigured  : DB 可用但角色行不存在，走硬编码兜底
//   - permissionDBOnlyConfigured    : DB 角色行存在，perms 以 DB 为准（空集=显式撤销，fail-closed）
//
// ctx 由调用方传入（请求链路为请求 ctx），不再使用 context.Background()。
func loadPermissionsFromDBDBOnlyState(ctx context.Context, client *ent.Client, roleName string, tenantID int) (permissionDBOnlyState, []Permission) {
	if client == nil {
		// DB 不可用，fail-closed（保留 baseline 测试
		// TestSmartCheckPermission_DBOnlyFailClosed / TestDBOnlyPermissionModeDoesNotUseHardcodedFallback）
		return permissionDBOnlyUnavailable, nil
	}

	roleEntity, err := lookupEffectiveRole(ctx, client, roleName, tenantID)
	switch {
	case errors.Is(err, errRoleInactive):
		// 角色行存在但被显式停用：这是管理员的显式撤销，语义等同「权限集为空」，
		// 必须 fail-closed 且**不得**回退到编译期默认权限基线——
		// 若落到 unconfigured 分支，停用反而会重新发放默认权限（形同虚设）。
		// 这里复用上面那次角色行查询，不额外增加 DB 往返。
		return permissionDBOnlyConfigured, nil
	case ent.IsNotFound(err) || (err == nil && roleEntity == nil):
		// 只有整个租户还没有任何 Role 行时才视为“未初始化”并使用默认基线；
		// 租户已有角色但当前 role 行缺失，视为该角色被撤销/迁移异常，
		// 必须按显式空权限集 fail-closed，不能重新发放默认权限。
		roleCount, countErr := client.Role.Query().
			Where(role.TenantIDEQ(tenantID)).
			Count(ctx)
		if countErr != nil {
			logPermissionLoadFailure(roleName, tenantID, countErr)
			return permissionDBOnlyUnavailable, nil
		}
		if roleCount > 0 {
			return permissionDBOnlyConfigured, nil
		}
		return permissionDBOnlyUnconfigured, nil
	case err != nil:
		// 查询本身失败（连接错误、ctx 取消、同 code 多行）：不能当成「未配置」走硬编码，
		// 否则瞬时故障变成 fail-open。
		logPermissionLoadFailure(roleName, tenantID, err)
		return permissionDBOnlyUnavailable, nil
	}

	// 角色行存在：以 DB 实际权限集合为准（空集=显式撤销，仍 fail-closed）
	perms, loadErr := loadPermissionsFromDB(ctx, client, roleName, tenantID)
	if loadErr != nil {
		return permissionDBOnlyUnavailable, nil
	}
	return permissionDBOnlyConfigured, perms
}

// permissionDBOnlyState 三态枚举，供 loadPermissionsByMode DBOnly 分支分流。
type permissionDBOnlyState int

const (
	permissionDBOnlyUnavailable permissionDBOnlyState = iota
	permissionDBOnlyUnconfigured
	permissionDBOnlyConfigured
)

//go:generate go run ../cmd/authz-gen

// ResourceActionMap 路径预检映射 = 路由声明生成条目 + 显式族级回退策略。
//
// 路由条目的单一真源是 router/ 与 handlers/ 中的 RequirePermission 家族声明：
// 生成物在 rbac_precheck_gen.go（禁手改）；族级通配/别名策略在 precheck_fallback.go。
// 新增/修改路由权限声明后必须重新生成：cd itsm-backend && go run ./cmd/authz-gen
// （忘生成会被 TestPrecheckMapIsFresh / TestRoutePrecheckAlignment 守卫拦下。）
var ResourceActionMap = buildResourceActionMap()

// buildResourceActionMap 合并生成条目与回退策略。同键冲突时以生成条目（声明）优先。
func buildResourceActionMap() map[string]map[string]Permission {
	m := precheckRoutePermissions()
	for method, pairs := range precheckFallbackPolicies {
		if m[method] == nil {
			m[method] = make(map[string]Permission, len(pairs))
		}
		for pattern, perm := range pairs {
			if _, exists := m[method][pattern]; !exists {
				m[method][pattern] = perm
			}
		}
	}
	return m
}

// RBACMiddleware RBAC权限控制中间件
func RBACMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 调试日志
		zap.S().Infow(
			"RBACMiddleware: received request",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)

		// 将 Ent 客户端放入上下文，供资源级别检查使用（例如工单所有权校验）
		if client != nil {
			c.Set("client", client)
		}
		// 获取用户信息
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			zap.S().Warnw(
				"RBACMiddleware: user_id not found in context",
				"path", c.Request.URL.Path,
			)
			common.Fail(c, common.AuthFailedCode, "用户未认证")
			c.Abort()
			return
		}

		userID, ok := userIDInterface.(int)
		if !ok {
			zap.S().Warnw(
				"RBACMiddleware: user_id format error",
				"path", c.Request.URL.Path,
				"user_id_interface", userIDInterface,
			)
			common.Fail(c, common.AuthFailedCode, "用户ID格式错误")
			c.Abort()
			return
		}

		// 获取租户ID
		// 特殊路径：认证相关端点不需要租户ID检查
		authPaths := map[string]bool{
			"/api/v1/auth/me":      true,
			"/api/v1/auth/tenants": true,
			"/api/v1/auth/menus":   true,
			"/api/v1/auth/profile": true,
		}
		isAuthPath := authPaths[c.Request.URL.Path]

		tenantIDInterface, exists := c.Get("tenant_id")
		if !exists {
			// 对于认证端点，尝试从JWT claim获取租户ID
			if isAuthPath {
				// 仅从JWT的tenant_id claim获取，不信任请求头
				if jwtTenantID, ok := c.Get("tenant_id"); ok {
					if tid, ok := jwtTenantID.(int); ok && tid > 0 {
						c.Set("tenant_id", tid)
						tenantIDInterface = tid
					}
				}
				// 如果仍然没有租户ID，拒绝请求而非默认分配
				if tenantIDInterface == nil {
					zap.S().Warnw(
						"RBACMiddleware: tenant_id not found in context for auth path",
						"path", c.Request.URL.Path,
						"user_id", userID,
					)
					common.Fail(c, common.AuthFailedCode, "租户信息缺失")
					c.Abort()
					return
				}
			} else {
				zap.S().Warnw(
					"RBACMiddleware: tenant_id not found in context",
					"path", c.Request.URL.Path,
					"user_id", userID,
				)
				common.Fail(c, common.AuthFailedCode, "租户信息缺失")
				c.Abort()
				return
			}
		}
		tenantID, ok := tenantIDInterface.(int)
		if !ok {
			zap.S().Warnw(
				"RBACMiddleware: tenant_id format error",
				"path", c.Request.URL.Path,
				"tenant_id_interface", tenantIDInterface,
			)
			common.Fail(c, common.AuthFailedCode, "租户ID格式错误")
			c.Abort()
			return
		}

		// service 主体短路（2026-11-18）。
		//
		// service 主体是**进程身份**而非登录用户：JWT 里 user_id 恒为 0，
		// 它的权限来自签发时固定的只读载体角色（audit_admin），由
		// AuthMiddleware 校验过（角色必须是 ServicePrincipalRole、
		// user_id=0、tenant>0）。
		//
		// 此前本中间件无条件按 user_id 查用户表，service 主体必然查不到
		// → 一律 401「用户不存在」。后果是 service token 无法访问任何
		// /api/v1/* 业务路由（含 MCP 通道）—— Phase B1 落地了主体身份，
		// 但 RBAC 这一环没打通，既有测试只覆盖到 AuthMiddleware 就停了。
		//
		// 这里短路而非「造一条用户记录」：service 主体不该出现在用户表里
		// （它不是人，无法登录、无法被指派工单、无法承担职责分立）。
		// 短路后仍由 RequirePermission 按其载体角色判定权限，
		// 并由 Gate 2.5（authz 主体零信任）决定它能否执行具体动作。
		if authz.PrincipalFromContext(c.Request.Context()) == authz.PrincipalService {
			c.Next()
			return
		}

		// 从数据库获取用户最新角色信息
		// 使用请求 ctx，随请求超时/取消传播
		// 多角色：附带加载 user.roles（m2m 附加角色），权限判定取并集。
		userEntity, err := client.User.Query().
			Where(user.ID(userID)).
			WithRoles().
			Only(c.Request.Context())
		if err != nil {
			zap.S().Warnw(
				"RBACMiddleware: user not found in DB",
				"path", c.Request.URL.Path,
				"user_id", userID,
				"error", err.Error(),
			)
			common.Fail(c, common.AuthFailedCode, "用户不存在")
			c.Abort()
			return
		}

		// 物化全部生效角色（角色 code 列表）到上下文，供 RequirePermission/
		// SmartCheckPermission 统一按并集判定。附加角色仅取启用且同租户的。
		if extraRoles := userEntity.Edges.Roles; len(extraRoles) > 0 {
			roleCodes := make([]string, 0, len(extraRoles))
			for _, r := range extraRoles {
				// 附加角色不可授予平台超管：历史上租户可伪造该码，触发字符串直通。
				// super_admin 只能来自 users.role 主角色，不能通过 user_roles 叠加。
				if r != nil && r.IsActive && r.TenantID == tenantID && r.Code != domainrole.SuperAdmin {
					roleCodes = append(roleCodes, r.Code)
				}
			}
			if len(roleCodes) > 0 {
				c.Set("roles", roleCodes)
			}
		}

		// 检查用户是否被禁用
		if !userEntity.Active {
			zap.S().Warnw(
				"RBACMiddleware: user is disabled",
				"path", c.Request.URL.Path,
				"user_id", userID,
			)
			common.Fail(c, common.ForbiddenCode, "用户已被禁用")
			c.Abort()
			return
		}

		// 从JWT中获取角色信息
		roleInterface, exists := c.Get("role")
		if !exists {
			zap.S().Warnw(
				"RBACMiddleware: role not found in context",
				"path", c.Request.URL.Path,
				"user_id", userID,
			)
			common.Fail(c, common.AuthFailedCode, "角色信息缺失")
			c.Abort()
			return
		}

		role, ok := roleInterface.(string)
		if !ok {
			zap.S().Warnw(
				"RBACMiddleware: role format error",
				"path", c.Request.URL.Path,
				"role_interface", roleInterface,
			)
			common.Fail(c, common.AuthFailedCode, "角色格式错误")
			c.Abort()
			return
		}

		// 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 检查权限（从数据库加载权限）
		if !hasPermission(client, role, method, path, userID, tenantID, c) {
			zap.S().Warnw(
				"RBACMiddleware: permission denied",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"user_id", userID,
				"role", role,
			)
			common.Fail(c, common.ForbiddenCode, "权限不足")
			c.Abort()
			return
		}

		// 调试日志：RBAC检查通过
		zap.S().Infow(
			"RBACMiddleware: access granted",
			"path", c.Request.URL.Path,
			"user_id", userID,
			"role", role,
			"tenant_id", tenantID,
		)

		// 将用户实体信息存储到上下文中
		c.Set("user_entity", userEntity)

		c.Next()
	}
}

// RequirePermission 要求特定权限的中间件
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok {
			common.Fail(c, common.AuthFailedCode, "用户角色信息缺失")
			c.Abort()
			return
		}
		roleStr, ok := role.(string)
		if !ok {
			common.Fail(c, common.AuthFailedCode, "用户角色信息格式错误")
			c.Abort()
			return
		}

		// 获取租户ID
		tenantIDInterface, ok := c.Get("tenant_id")
		if !ok {
			common.Fail(c, common.AuthFailedCode, "租户信息缺失")
			c.Abort()
			return
		}
		tenantID, ok := tenantIDInterface.(int)
		if !ok {
			common.Fail(c, common.AuthFailedCode, "租户信息格式错误")
			c.Abort()
			return
		}

		// 获取客户端
		clientInterface, ok := c.Get("client")
		if !ok {
			common.Fail(c, common.InternalErrorCode, "客户端缺失")
			c.Abort()
			return
		}
		client, ok := clientInterface.(*ent.Client)
		if !ok {
			common.Fail(c, common.InternalErrorCode, "客户端类型错误")
			c.Abort()
			return
		}

		// 多角色：判定取全部生效角色（主角色 + m2m 附加角色）并集。
		// 无 "roles" 键时退化为仅主角色（兼容未物化角色的旧链路）。
		if !AuthorizeResource(c.Request.Context(), client, GetContextRoles(c), resource, action, tenantID) {
			// deny 路径补 Warn 日志，便于排查权限问题
			// 路径预检的 deny 在 RBACMiddleware 已有日志，这里覆盖路由级 RequirePermission
			zap.S().Warnw(
				"RequirePermission: denied",
				"resource", resource,
				"action", action,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"role", roleStr,
				"tenant_id", tenantID,
				"roles", GetContextRoles(c),
			)
			common.Fail(c, common.ForbiddenCode, "权限不足")
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(WithRBACRole(c.Request.Context(), roleStr))

		c.Next()
	}
}

// RequireRole enforces that the authenticated user role is one of the allowed roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	normalized := make([]string, 0, len(allowedRoles))
	for _, r := range allowedRoles {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(r)))
	}
	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 2003, "message": "缺少角色信息"})
			c.Abort()
			return
		}
		role, _ := roleAny.(string)
		role = strings.ToLower(strings.TrimSpace(role))
		for _, ar := range normalized {
			if role == ar {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"code": 2003, "message": "无权限执行该操作"})
		c.Abort()
	}
}

// hasPermission 检查用户是否有权限访问指定资源
// Uses Smart Permission Checker (4-layer fallback architecture)
func hasPermission(client *ent.Client, role, method, path string, userID, tenantID int, c *gin.Context) bool {
	// 仅 super_admin 硬编码直通（与 smart_permission.go checkRolePermissionFromDB 语义统一）。
	// sysadmin 不短路：DBOnly 模式下判定走 AuthorizeResourceForRole → loadPermissionsByMode，
	// configured 态尊重 role_permissions（空集=显式撤销），收回/降级在机制上可生效。
	// sysadmin 的播种码集现直接取 catalog.AllCodes()（权威投影），与 Definitions() 不再有差集。
	if role == "super_admin" {
		return true
	}

	// 特殊路径：允许所有认证用户访问自己的菜单
	// 菜单服务会根据用户权限过滤菜单，这里不需要额外权限检查
	if method == "GET" && (path == "/api/v1/auth/menus" || strings.HasPrefix(path, "/api/v1/auth/menus?")) {
		return true
	}
	// Capability discovery is available to every authenticated tenant member.
	// It only returns readiness metadata and already filters actions by role;
	// allowing discovery prevents a circular dependency where the UI needs a
	// capability permission before it can learn which capabilities are usable.
	if method == "GET" && path == "/api/v1/capabilities" {
		return true
	}
	// WS 票据颁发仅凭已认证身份（票据一次性且仅限本人，与 router 守卫测试豁免理由一致）；
	// 不豁免则 DBOnly fail-closed 拒绝全部非 super_admin 角色，实时通知整体失效。
	if method == "POST" && path == "/api/v1/ws/ticket" {
		return true
	}

	// 使用智能权限检查器（4层兜底架构）
	// 移除 database.GetRawDB() 直连，ACL 查询统一走 Ent 客户端
	return SmartCheckPermission(c, client, role, method, path, tenantID)
}

// HasResourcePermission 检查角色是否有指定资源的操作权限（导出供 AI 工具 RBAC 校验复用）
// AI 工具执行前的 Gate 2 校验入口
// ctx 由调用方传入（请求链路为请求 ctx）
func HasResourcePermission(ctx context.Context, client *ent.Client, role, resource, action string, tenantID int) bool {
	return hasResourcePermission(ctx, client, role, resource, action, tenantID)
}

// hasResourcePermission 检查角色是否有指定资源的操作权限（支持多种配置模式）
// 统一判定：委托 AuthorizeResourceForRole，单一真源。
func hasResourcePermission(ctx context.Context, client *ent.Client, role, resource, action string, tenantID int) bool {
	return AuthorizeResourceForRole(ctx, client, role, resource, action, tenantID)
}

// loadPermissionsByMode 根据配置模式加载权限
//
// DBOnly 模式下三态语义（loadPermissionsFromDBDBOnlyState 返回）：
//   - unavailable : DB 不可用（client==nil / 查询报错），fail-closed，禁止任何授权
//   - unconfigured: DB 可用但不存在该角色行（典型：未走 RBAC 后台初始化即上线的小租户），
//     走 authz 单一真源默认权限兜底，避免非 super_admin 全量 403
//   - configured  : DB 角色行存在，授权集合以 DB 为准（含空集=显式撤销，fail-closed）
//     兜底仅在「unconfigured」分支触发，「unavailable」分支仍 fail-closed，
//     保持既有 TestSmartCheckPermission_DBOnlyFailClosed /
//     TestDBOnlyPermissionModeDoesNotUseHardcodedFallback 等 fail-closed 测试不破。
func loadPermissionsByMode(ctx context.Context, client *ent.Client, role string, tenantID int) []Permission {
	// 非 DBOnly 模式（开发/测试环境）没有「角色行不存在即未配置」的分支，
	// 停用角色会经由 RoleDefaultPermissions 重新拿到编译期默认权限——
	// 停用必须真的停用，所以这里先做一次生效性判定：停用即返回空集。
	// 生产 DBOnly 路径不在此处判定，避免每次授权判定都多查一次库
	// （DBOnlyState 内部本来就查角色行，判定已在那条路径上完成）。
	if client != nil && PermissionConfig.Mode != PermissionConfigModeDBOnly {
		if _, err := lookupEffectiveRole(ctx, client, role, tenantID); errors.Is(err, errRoleInactive) {
			return nil
		}
	}
	switch PermissionConfig.Mode {
	case PermissionConfigModeDBOnly:
		state, perms := loadPermissionsFromDBDBOnlyState(ctx, client, role, tenantID)
		switch state {
		case permissionDBOnlyConfigured:
			// DB 显式配置（含空集合=显式撤销），尊重 DB
			return perms
		case permissionDBOnlyUnconfigured:
			// DB 未配置：authz 单一真源默认权限兜底，避免新装/小租户下非 super_admin 全量 403
			if defaults := RoleDefaultPermissions(role); len(defaults) > 0 {
				return defaults
			}
			return nil
		default:
			// unavailable：DB 不可用，fail-closed（不返回任何授权）
			return nil
		}
	case PermissionConfigModeHardcodeOnly:
		return RoleDefaultPermissions(role)
	case PermissionConfigModeMerge:
		// 合并数据库和默认权限（并集）
		dbPerms, _ := loadPermissionsFromDB(ctx, client, role, tenantID)
		defaultPerms := RoleDefaultPermissions(role)
		if len(defaultPerms) == 0 {
			return dbPerms
		}
		if len(dbPerms) == 0 {
			return defaultPerms
		}
		// 合并去重
		permMap := make(map[string]Permission)
		for _, p := range dbPerms {
			permMap[p.Resource+":"+p.Action] = p
		}
		for _, p := range defaultPerms {
			key := p.Resource + ":" + p.Action
			if _, exists := permMap[key]; !exists {
				permMap[key] = p
			}
		}
		result := make([]Permission, 0, len(permMap))
		for _, p := range permMap {
			result = append(result, p)
		}
		return result
	case PermissionConfigModeFallback:
		fallthrough
	default:
		// 默认（仅 dev/test 环境）：先数据库（role_permission+permission 表），为空则使用默认权限
		dbPerms, _ := loadPermissionsFromDB(ctx, client, role, tenantID)
		if len(dbPerms) > 0 {
			return dbPerms
		}
		return RoleDefaultPermissions(role)
	}
}

// checkPermissionMatch 检查权限是否匹配
func checkPermissionMatch(permissions []Permission, resource, action string) bool {
	if len(permissions) == 0 {
		return false
	}

	for _, perm := range permissions {
		// 检查通配符权限
		if perm.Resource == "*" && perm.Action == "*" {
			return true
		}
		if perm.Resource == "*" && perm.Action == action {
			return true
		}
		if perm.Resource == resource && perm.Action == "*" {
			return true
		}
		// 资源管理员权限包含该资源下的具体业务动作。
		if perm.Resource == resource && perm.Action == "admin" {
			return true
		}
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}

	return false
}

// InvalidateRolePermissionCache 使指定角色-租户的权限缓存失效（导出供外部调用）
// 本地失效后向 Redis 广播，多副本部署下其它实例同步失效。
func InvalidateRolePermissionCache(roleName string, tenantID int) {
	if PermissionConfig.EnableCache {
		invalidatePermissionCache(roleName, tenantID)
		broadcastPermissionInvalidation(roleName, tenantID)
	}
}

// InvalidateAllPermissionCachesEx 使所有权限缓存失效（导出供外部调用）
func InvalidateAllPermissionCachesEx() {
	if PermissionConfig.EnableCache {
		InvalidateAllPermissionCaches()
	}
}

// =============================================================================
// 统一权限判定核心
//
// 所有权限判定的单一真源：rbac.go 的 hasResourcePermission 与 smart_permission.go 的
// checkRolePermissionFromDB 都委托到这里，避免两处各自决定 super_admin/sysadmin 是否直通。
// 多角色按并集判定。
// =============================================================================

// AuthorizeResource 统一判定：一组角色（并集语义）在指定租户内是否拥有 resource:action 权限。
// roles 为该用户的全部生效角色（主角色 + m2m 附加角色）；任一角色命中即授权。
// 语义与既有单角色判定完全一致：
//   - super_admin 任一角色命中即直通；
//   - 其余角色按 PermissionConfig.Mode 加载权限（生产 DBOnly fail-closed）。
func AuthorizeResource(ctx context.Context, client *ent.Client, roles []string, resource, action string, tenantID int) bool {
	// 空 resource/action 一律拒绝（fail-closed）。
	// 这不是理论防御：checkPermissionMatch 的通配分支在空值下会产生意外放行——
	// 漏配 Resource 的工具定义（ToolDefinition.Resource == ""）会让持有
	// "*:<action>" 码的角色通过 "*" && perm.Action == action 分支。
	// 在入口显式拒绝，把「人为漏配」的后果固定为拒绝，而不是依赖对通配符
	// 叠加效果的推理。已验证所有生产调用方均传非空值，此分支不改变既有行为。
	if resource == "" || action == "" {
		return false
	}
	if len(roles) == 0 {
		return false
	}
	for _, r := range roles {
		if r == "super_admin" {
			return true
		}
	}
	for _, r := range roles {
		permissions := loadPermissionsByMode(ctx, client, r, tenantID)
		if checkPermissionMatch(permissions, resource, action) {
			return true
		}
	}
	return false
}

// AuthorizeResourceForRole 单角色便捷入口（兼容旧调用方）。
// 委托 AuthorizeResource：空值拒绝与 super_admin 判定收敛在单一入口，
// 避免「单角色路径放行空值、多角色路径拒绝空值」的语义分叉。
func AuthorizeResourceForRole(ctx context.Context, client *ent.Client, role, resource, action string, tenantID int) bool {
	return AuthorizeResource(ctx, client, []string{role}, resource, action, tenantID)
}

// GetContextRoles 从 gin 上下文提取用户全部生效角色（主角色 + 附加角色）。
// RBACMiddleware 会物化 "roles" 键；未注入时退化为仅主角色（兼容旧链路）。
func GetContextRoles(c *gin.Context) []string {
	primary := ""
	if v, ok := c.Get("role"); ok {
		if s, ok := v.(string); ok {
			primary = s
		}
	}
	var extra []string
	if v, ok := c.Get("roles"); ok {
		if list, ok := v.([]string); ok {
			extra = list
		}
	}
	// 去重：附加角色不得与主角色重复
	seen := make(map[string]bool, len(extra)+1)
	roles := make([]string, 0, len(extra)+1)
	if primary != "" && !seen[primary] {
		seen[primary] = true
		roles = append(roles, primary)
	}
	for _, r := range extra {
		if r != "" && !seen[r] {
			seen[r] = true
			roles = append(roles, r)
		}
	}
	return roles
}

// getPermissionFromPath 从路径获取权限信息
func getPermissionFromPath(method, path string) *Permission {
	methodMap, exists := ResourceActionMap[method]
	if !exists {
		return nil
	}

	// 精确匹配
	if perm, exists := methodMap[path]; exists {
		return &perm
	}

	// 通配符匹配。多个规则命中时选择最具体的规则，避免
	// /tickets/* 抢先覆盖 /tickets/*/assign 这类动作权限。
	var matched *Permission
	bestSpecificity := -1
	for pattern, perm := range methodMap {
		if matchPath(pattern, path) {
			specificity := len(strings.ReplaceAll(pattern, "*", ""))
			if specificity > bestSpecificity {
				permission := perm
				matched = &permission
				bestSpecificity = specificity
			}
		}
	}

	return matched
}

// matchPath 匹配路径（支持通配符）
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// 末尾 * 保持历史语义：匹配剩余任意层级。
	if strings.Count(pattern, "*") == 1 && strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	// 中间 * 匹配单个路径段，例如 /tickets/*/assign。
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i := range patternParts {
		if patternParts[i] != "*" && patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}
