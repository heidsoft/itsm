package middleware

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"itsm-backend/ent"

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

// DBQuerier interface for database access
type DBQuerier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

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
func SmartCheckPermission(c *gin.Context, db DBQuerier, client *ent.Client, role string, method, path string, tenantID int) bool {
	// L1: Check auth whitelist (public endpoints)
	if isAuthWhitelist(path, method) {
		zap.S().Debugw("Auth whitelist match", "path", path, "method", method)
		return true
	}

	// L2: Check database ACL (dynamic configuration)
	if checkDatabaseACL(db, client, role, method, path, tenantID) {
		return true
	}

	// L3: Check URL auto-inference (REST endpoints)
	if checkURLInference(role, method, path) {
		return true
	}

	// L4: Fallback to role-based hardcoded permissions
	return checkRoleBasedPermission(role, method, path)
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

func checkDatabaseACL(db DBQuerier, client *ent.Client, role, method, path string, tenantID int) bool {
	if db == nil {
		return false
	}

	// Get cached ACLs or load from database
	acls := getCachedACLs(db, tenantID)
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
			if checkRolePermissionFromDB(client, role, acl.Resource, acl.Action, tenantID) {
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

func getCachedACLs(db DBQuerier, tenantID int) []EndpointACL {
	aclCacheLock.RLock()
	if cached, exists := aclCache[tenantID]; exists {
		if time.Now().Before(cached.expiresAt) {
			aclCacheLock.RUnlock()
			return cached.acls
		}
	}
	aclCacheLock.RUnlock()

	// Load from database
	acls := loadACLsFromDB(db, tenantID)
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

func loadACLsFromDB(db DBQuerier, tenantID int) []EndpointACL {
	ctx := context.Background()

	// Query endpoint_acls table - handle both old and new schema
	// Old schema: no is_whitelist column
	// New schema: has is_whitelist column
	rows, err := db.QueryContext(ctx, `
		SELECT path_pattern, method, resource, action, priority
		FROM endpoint_acls
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY priority DESC
	`, tenantID)
	if err != nil {
		zap.S().Warnw("Failed to load ACLs from DB",
			"tenant_id", tenantID, "error", err)
		return nil
	}
	defer rows.Close()

	var acls []EndpointACL
	for rows.Next() {
		var acl EndpointACL
		var method sql.NullString
		if err := rows.Scan(&acl.PathPattern, &method, &acl.Resource, &acl.Action, &acl.Priority); err != nil {
			zap.S().Warnw("Failed to scan ACL row", "error", err)
			continue
		}
		if method.Valid {
			acl.Method = &method.String
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
func checkURLInference(role, method, path string) bool {
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
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
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
func checkRolePermissionFromDB(client *ent.Client, role, resource, action string, tenantID int) bool {
	// Super admin has all permissions
	if role == "super_admin" || role == "sysadmin" {
		return true
	}

	// SEC-005: 通过 loadPermissionsByMode 从数据库加载权限
	// 该函数根据 PermissionConfig.Mode 决定查询策略：
	//   - DBOnly: 仅数据库，数据库为空时 fallback 硬编码
	//   - HardcodeOnly: 仅硬编码
	//   - Merge: 数据库 + 硬编码并集
	//   - Fallback: 先数据库，失败则硬编码
	permissions := loadPermissionsByMode(client, role, tenantID)

	// 使用统一的权限匹配逻辑检查
	if checkPermissionMatch(permissions, resource, action) {
		return true
	}

	return false
}

// GetResourceAndActionFromPath extracts resource and action from a URL path
// This is useful for logging and debugging
func GetResourceAndActionFromPath(method, path string) (resource, action string) {
	// Try L2: Check database ACL first
	aclCacheLock.RLock()
	if cached, ok := aclCache[1]; ok && cached != nil {
		for _, acl := range cached.acls {
			if matchACL(acl, method, path) {
				aclCacheLock.RUnlock()
				return acl.Resource, acl.Action
			}
		}
	}
	aclCacheLock.RUnlock()

	// Try L3: URL auto-inference
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
