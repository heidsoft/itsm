package middleware

import (
	"context"
	"strings"
	"sync"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/endpointacl"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// =============================================================================
// Smart Permission Checker - Enterprise RBAC Architecture
//
// Permission Check Priority:
//   L1. Auth whitelist (no permission needed) - e.g., /api/v1/auth/login
//   L2. Database ACL (dynamic configuration) - endpoint_acls table
//   L3. URL auto-inference (fallback for REST endpoints) - /api/v1/{resource}
//   L4. Role-based hardcoded defaults (last resort)
//
// Features:
//   - Multi-tenant support
//   - Hot-reload capability
//   - URL pattern matching with wildcard support
//   - Auto-inference for REST endpoints
// =============================================================================

// EndpointACL represents a cached ACL entry
type EndpointACL struct {
	PathPattern string
	Method      *string // nil means all methods
	Resource    string
	Action      string
	Priority    int
	IsWhitelist bool
}

// ACL Cache with TTL
type cachedACL struct {
	acls      []EndpointACL
	expiresAt time.Time
}

// ACL Configuration
type ACLConfig struct {
	EnableHotReload bool          // Enable dynamic ACL reloading
	CacheTTL        time.Duration // Cache TTL (default: 5 minutes)
	ReloadInterval  time.Duration // Hot reload interval (default: 1 minute)
}

var (
	aclCache     = make(map[int]*cachedACL) // tenant_id -> cached ACLs
	aclCacheLock sync.RWMutex
	aclConfig    = ACLConfig{
		EnableHotReload: true,
		CacheTTL:        5 * time.Minute,
		ReloadInterval:  1 * time.Minute,
	}
)

// =============================================================================
// Public Functions
// =============================================================================

// SetACLConfig sets the ACL configuration
func SetACLConfig(cfg ACLConfig) {
	aclConfig = cfg
	if cfg.CacheTTL == 0 {
		aclConfig.CacheTTL = 5 * time.Minute
	}
	if cfg.ReloadInterval == 0 {
		aclConfig.ReloadInterval = 1 * time.Minute
	}
	// Invalidate all caches when config changes
	InvalidateACLCache()
}

// InvalidateACLCache invalidates all ACL caches (for hot-reload)
func InvalidateACLCache() {
	aclCacheLock.Lock()
	defer aclCacheLock.Unlock()
	clear(aclCache)
	zap.S().Info("ACL cache invalidated")
}

// InvalidateTenantACLCache invalidates ACL cache for a specific tenant
func InvalidateTenantACLCache(tenantID int) {
	aclCacheLock.Lock()
	defer aclCacheLock.Unlock()
	delete(aclCache, tenantID)
	zap.S().Infow("Tenant ACL cache invalidated", "tenant_id", tenantID)
}

// =============================================================================
// Smart Permission Check
// =============================================================================

// SmartCheckPermission checks permission using 4-layer fallback
// This is the main entry point for permission checking
//
// P0-4 修复：移除 raw DB 通道，ACL 与角色权限统一走 Ent（保留租户过滤与 RLS），
// 并使用请求 context（超时/取消随请求传播），不再使用 context.Background()。
func SmartCheckPermission(c *gin.Context, client *ent.Client, role string, method, path string, tenantID int) bool {
	ctx := c.Request.Context()

	// Phase 1 多角色：优先取 gin 上下文中物化的全部生效角色（主角色+m2m附加角色），
	// 未注入时退化为单角色（兼容旧链路与单测）。
	roles := GetContextRoles(c)
	if len(roles) == 0 && role != "" {
		roles = []string{role}
	}

	// super_admin 任一角色命中即直通（与 AuthorizeResource 语义一致）。
	for _, r := range roles {
		if r == "super_admin" {
			return true
		}
	}

	// L1: Check auth whitelist (public endpoints)
	if isAuthWhitelist(path, method) {
		zap.S().Debugw("Auth whitelist match", "path", path, "method", method)
		return true
	}

	// L2: Check database ACL (dynamic configuration)
	for _, r := range roles {
		if checkDatabaseACL(ctx, client, r, method, path, tenantID) {
			return true
		}
	}

	// L3: Check URL auto-inference (REST endpoints)
	for _, r := range roles {
		if checkURLInference(ctx, client, r, method, path, tenantID) {
			return true
		}
	}

	// L4: Fallback to role-based hardcoded permissions
	// Phase 1 修复：L4 直接读编译期 RolePermissions map，不感知 PermissionConfig.Mode，
	// 生产（DBOnly fail-closed）下无 DB 权限的角色仍可能经 L4 拿到权限，违反 fail-closed 语义。
	// 现按模式分流：DBOnly 下硬编码完全不参与授权，未命中 L1-L3 即拒绝。
	if PermissionConfig.Mode == PermissionConfigModeDBOnly {
		return false
	}
	for _, r := range roles {
		if checkRoleBasedPermission(r, method, path) {
			return true
		}
	}
	return false
}

// =============================================================================
// L1: Auth Whitelist
// =============================================================================

// Auth whitelist endpoints - no permission required
var authWhitelist = map[string]bool{
	"/api/v1/auth/login":    true,
	"/api/v1/auth/register": true,
	"/health":               true,
	"/api/v1/health":        true,
}

func isAuthWhitelist(path, method string) bool {
	return authWhitelist[path]
}

// =============================================================================
// L2: Database ACL Check
// =============================================================================

func checkDatabaseACL(ctx context.Context, client *ent.Client, role, method, path string, tenantID int) bool {
	if client == nil {
		return false
	}

	// Get cached ACLs or load from database
	acls := getCachedACLs(ctx, client, tenantID)
	if acls == nil {
		return false
	}

	// Check each ACL in priority order
	for _, acl := range acls {
		if matchACL(acl, method, path) {
			// Check whitelist
			if acl.IsWhitelist {
				zap.S().Debugw("Database ACL whitelist match",
					"path", path, "method", method)
				return true
			}

			// Found matching ACL - check role permission
			if checkRolePermissionFromDB(ctx, client, role, acl.Resource, acl.Action, tenantID) {
				zap.S().Debugw("Database ACL permission granted",
					"path", path, "method", method,
					"resource", acl.Resource, "action", acl.Action)
				return true
			}
			// ACL matched but no role permission - deny
			zap.S().Debugw("Database ACL matched but no role permission",
				"path", path, "method", method,
				"resource", acl.Resource, "action", acl.Action, "role", role)
			return false
		}
	}

	// No ACL matched - return false (let L3 handle it)
	return false
}

func getCachedACLs(ctx context.Context, client *ent.Client, tenantID int) []EndpointACL {
	aclCacheLock.RLock()
	if cached, exists := aclCache[tenantID]; exists {
		if time.Now().Before(cached.expiresAt) {
			aclCacheLock.RUnlock()
			return cached.acls
		}
	}
	aclCacheLock.RUnlock()

	// Load from database
	acls := loadACLsFromDB(ctx, client, tenantID)
	if acls == nil {
		return nil
	}

	// Cache the result
	aclCacheLock.Lock()
	aclCache[tenantID] = &cachedACL{
		acls:      acls,
		expiresAt: time.Now().Add(aclConfig.CacheTTL),
	}
	aclCacheLock.Unlock()

	return acls
}

// loadACLsFromDB 通过 Ent 加载租户 ACL（P0-4：替代 raw SQL，
// 保持 tenant_id 过滤并使用请求 ctx）。
func loadACLsFromDB(ctx context.Context, client *ent.Client, tenantID int) []EndpointACL {
	if client == nil || tenantID <= 0 {
		return nil
	}

	rows, err := client.EndpointACL.Query().
		Where(
			endpointacl.TenantIDEQ(tenantID),
			endpointacl.IsActiveEQ(true),
		).
		Order(ent.Desc(endpointacl.FieldPriority)).
		All(ctx)
	if err != nil {
		zap.S().Warnw("Failed to load ACLs from DB",
			"tenant_id", tenantID, "error", err)
		return nil
	}

	var acls []EndpointACL
	for _, row := range rows {
		acl := EndpointACL{
			PathPattern: row.PathPattern,
			Resource:    row.Resource,
			Action:      row.Action,
			Priority:    row.Priority,
		}
		// method 为空字符串表示匹配所有 HTTP 方法（等价旧 schema 的 NULL）
		if row.Method != "" {
			m := row.Method
			acl.Method = &m
		}
		// Check if this is a whitelist endpoint (auth endpoints with NULL method)
		acl.IsWhitelist = isKnownWhitelistPath(acl.PathPattern)
		acls = append(acls, acl)
	}

	if len(acls) == 0 {
		zap.S().Warnw("No ACLs found in database",
			"tenant_id", tenantID)
		return nil
	}

	zap.S().Infow("Loaded ACLs from database",
		"tenant_id", tenantID, "count", len(acls))
	return acls
}

// isKnownWhitelistPath checks if a path is a known whitelist endpoint
func isKnownWhitelistPath(path string) bool {
	switch path {
	case "/api/v1/auth/login",
		"/api/v1/auth/register",
		"/health",
		"/api/v1/health":
		return true
	default:
		return false
	}
}

func matchACL(acl EndpointACL, method, path string) bool {
	// Check path pattern
	pathMatches := false

	// Exact match
	if acl.PathPattern == path {
		pathMatches = true
	}

	// Wildcard match (e.g., /api/v1/tickets/*)
	if strings.HasSuffix(acl.PathPattern, "*") {
		prefix := strings.TrimSuffix(acl.PathPattern, "*")
		if strings.HasPrefix(path, prefix) {
			pathMatches = true
		}
	}

	if !pathMatches {
		return false
	}

	// Check method (nil means all methods)
	if acl.Method != nil && *acl.Method != "" && *acl.Method != method {
		return false
	}

	return true
}

// =============================================================================
// L3: URL Auto-Inference
// =============================================================================

// REST URL pattern: /api/v1/{resource}/*
// Examples: /api/v1/tickets, /api/v1/incidents/123
func checkURLInference(ctx context.Context, client *ent.Client, role, method, path string, tenantID int) bool {
	// 优先使用显式端点映射。动作型接口不能只按 HTTP 方法推断，
	// 例如 POST /tickets/:id/assign 对应 ticket:assign，而不是 ticket:write。
	if permission := getPermissionFromPath(method, path); permission != nil {
		if checkRolePermissionFromDB(ctx, client, role, permission.Resource, permission.Action, tenantID) {
			zap.S().Debugw("URL permission mapping granted",
				"path", path, "method", method,
				"resource", permission.Resource, "action", permission.Action)
			return true
		}
	}

	// Extract resource from URL
	// Format: /api/v1/{resource}[/*]
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "api" || parts[2] != "v1" {
		return false
	}

	resource := parts[3]
	action := methodToAction(method)

	// Check role has this resource:action permission
	if hasResourcePermissionFromRole(role, resource, action) {
		zap.S().Debugw("URL inference permission granted",
			"path", path, "method", method,
			"inferred_resource", resource, "inferred_action", action)
		return true
	}

	return false
}

func methodToAction(method string) string {
	switch method {
	case "GET":
		return "read"
	case "POST", "PUT", "PATCH":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return "read"
	}
}

// =============================================================================
// L4: Role-Based Hardcoded Permissions (Fallback)
// =============================================================================

// hasResourcePermissionFromRole checks if a role has a specific resource:action permission
// This uses the hardcoded RolePermissions map as fallback
func hasResourcePermissionFromRole(role, resource, action string) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, perm := range permissions {
		if perm.Resource == "*" && perm.Action == "*" {
			return true
		}
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*" || perm.Action == "admin") {
			return true
		}
	}

	return false
}

// checkRoleBasedPermission checks permission using hardcoded RolePermissions
func checkRoleBasedPermission(role, method, path string) bool {
	// Get resource and action from ResourceActionMap
	perm := getPermissionFromPath(method, path)
	if perm == nil {
		// No mapping found - deny by default
		zap.S().Warnw("No permission mapping found",
			"role", role, "method", method, "path", path)
		return false
	}

	return hasResourcePermissionFromRole(role, perm.Resource, perm.Action)
}

// =============================================================================
// Helper Functions
// =============================================================================

// checkRolePermissionFromDB checks if a role has permission for a resource:action
// using the database-driven approach
// SEC-005 修复：真正查询数据库获取角色权限，而非仅使用硬编码权限
// P0-4：透传请求 ctx，替代 loadACLsFromDB/loadPermissions 链路中的 context.Background()
func checkRolePermissionFromDB(ctx context.Context, client *ent.Client, role, resource, action string, tenantID int) bool {
	// Phase 1 统一判定：委托 rbac.go 的单一真源 AuthorizeResourceForRole，
	// 消除与 hasResourcePermission 的双头语义分叉（sysadmin 直通不一致等）。
	return AuthorizeResourceForRole(ctx, client, role, resource, action, tenantID)
}

// GetResourceAndActionFromPath extracts resource and action from a URL path
// This is useful for logging and debugging
// P0-4 修复：移除硬编码 aclCache[1]（跨租户读取 tenant 1 缓存），
// 仅保留静态 URL 推断，不再依赖任何租户的 ACL 缓存。
func GetResourceAndActionFromPath(method, path string) (resource, action string) {
	// URL auto-inference
	parts := strings.Split(path, "/")
	if len(parts) >= 4 && parts[1] == "api" && parts[2] == "v1" {
		return parts[3], methodToAction(method)
	}

	// Fallback
	return "", ""
}

// =============================================================================
// ServiceNow-style ACL Evaluation
// =============================================================================

// EvaluateACLScript evaluates an ACL condition script
// This would be used for advanced scenarios where simple path matching isn't enough
// Example: "user can only see tickets assigned to them"
type ACLScriptContext struct {
	UserID       int
	TenantID     int
	Role         string
	ResourceID   interface{}
	ResourceType string
}

func EvaluateACLScript(script string, ctx ACLScriptContext) bool {
	if script == "" {
		return true
	}

	// SEC-003 修复：使用 ACL 表达式引擎替代空实现
	engine := NewACLExpressionEngine()
	variables := map[string]interface{}{
		"ctx.user_id":       ctx.UserID,
		"ctx.tenant_id":     ctx.TenantID,
		"ctx.role":          ctx.Role,
		"ctx.resource_id":   ctx.ResourceID,
		"ctx.resource_type": ctx.ResourceType,
	}

	result := engine.Evaluate(script, variables)

	zap.S().Debugw("ACL 脚本评估完成",
		"script_length", len(script),
		"user_id", ctx.UserID,
		"resource_type", ctx.ResourceType,
		"result", result)

	return result
}
