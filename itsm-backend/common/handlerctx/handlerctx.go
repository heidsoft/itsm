// Package handlerctx centralises the duplicated request-context
// boilerplate that was previously copy-pasted in every legacy
// controller (controller/*.go) and every domain handler
// (handlers/<domain>/handler.go).
//
// Why a sub-package:
//
//   - the original home (common) is imported by middleware, which means
//     adding helpers here that also depend on middleware would create
//     an import cycle
//   - keeping helpers in a leaf package means every layer (controller,
//     service, middleware, handler) can import this without risk
//
// Contract:
//
//   - helpers read strictly from the request-scoped context values
//     injected by the auth / tenant middleware (never from query string
//     or body)
//   - helpers write the standard error response on failure and return
//     (zero, false) so the caller can early-return
//   - helpers use structured zap logging where appropriate
//
// These helpers are intentionally thin wrappers; business logic must
// remain in service/ or handlers/<domain>/service.go.
package handlerctx

import (
	"strconv"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"itsm-backend/ent"
	"itsm-backend/middleware"
)

// ResolveTenantID extracts the tenant id from the request context. The
// tenant id is injected by middleware.ResolveRequestTenantID, which
// in turn derives it from the JWT principal. On failure the helper
// delegates to middleware.AbortIfTenantError so the HTTP status code
// matches the existing middleware contract (401 for missing tenant
// context, 403 for MSP customer denied, 500 otherwise).
//
// Usage:
//
//	tenantID, ok := handlerctx.ResolveTenantID(c)
//	if !ok {
//	    return
//	}
func ResolveTenantID(c *gin.Context) (int, bool) {
	tenantID, err := middleware.ResolveRequestTenantID(c)
	if err == nil {
		return tenantID, true
	}
	zap.S().Warnw("Failed to resolve tenant ID", "error", err)
	if middleware.AbortIfTenantError(c, err) {
		return 0, false
	}
	return 0, false
}

// RequireTenantID 是 handlers/<domain>/handler.go 中重复出现的
// `tenantID := c.GetInt("tenant_id"); if tenantID == 0 { ... return }` 模板的
// helper 版本。区别于 ResolveTenantID，本它只检查 gin context 中是否存在有效
// tenant_id（不涉及 JWT 主体解析），因此适配那些仅要求上下文存在 tenant_id
// 的接口。返回 true 表示成功获取 >0 的 tenant_id 并已写入响应，否则 false。
//
// Usage:
//
//	tenantID, ok := handlerctx.RequireTenantID(c)
//	if !ok {
//	    return
//	}
func RequireTenantID(c *gin.Context) (int, bool) {
	tenantID := c.GetInt("tenant_id")
	if tenantID <= 0 {
		c.AbortWithStatusJSON(401, gin.H{
			"code":    2001,
			"message": "Tenant ID missing",
		})
		return 0, false
	}
	return tenantID, true
}

// GetEntClient extracts the *ent.Client stored in the Gin context by
// the DB middleware. On any failure it writes a 500 InternalErrorCode
// response and returns false.
//
// This is the long-standing boilerplate:
//
//	client, exists := ctx.Get("client")
//	if !exists {
//	    common.Fail(ctx, common.InternalErrorCode, "数据库客户端未找到")
//	    return
//	}
//	entClient, ok := client.(*ent.Client)
//	if !ok || entClient == nil {
//	    common.Fail(ctx, common.InternalErrorCode, "数据库客户端无效")
//	    return
//	}
//
// Usage:
//
//	entClient, ok := handlerctx.GetEntClient(c)
//	if !ok {
//	    return
//	}
func GetEntClient(c *gin.Context) (*ent.Client, bool) {
	client, exists := c.Get("client")
	if !exists {
		c.AbortWithStatusJSON(500, gin.H{
			"code":    5001,
			"message": "数据库客户端未找到",
		})
		return nil, false
	}
	entClient, ok := client.(*ent.Client)
	if !ok || entClient == nil {
		zap.S().Errorw("Invalid database client in request context")
		c.AbortWithStatusJSON(500, gin.H{
			"code":    5001,
			"message": "数据库客户端无效",
		})
		return nil, false
	}
	return entClient, true
}

// ResolveResourceIDAndTenant 是 controller 端高频组合：
// ParseResourceID + ResolveTenantID。两个步骤中任一失败 helper 都已
// 写好错误响应并 abort 请求链。
//
// 用法：
//
//	id, tenantID, ok := handlerctx.ResolveResourceIDAndTenant(ctx, "事件")
//	if !ok {
//	    return
//	}
func ResolveResourceIDAndTenant(c *gin.Context, resourceName string) (int, int, bool) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(400, gin.H{
			"code":    1001,
			"message": "无效的" + resourceName + "ID",
		})
		return 0, 0, false
	}
	tenantID, ok := ResolveTenantID(c)
	if !ok {
		return 0, 0, false
	}
	return id, tenantID, true
}

// ResolveResourceIDUserAndTenant 是 lifecycle / alert / acknowledge 接口
// 共有的三段样板：URL :id 解析 + 当前用户 ID + 当前租户 ID。
// 任何步骤失败时 helper 已经写好错误响应并 abort 调用链。
//
// 用法：
//
//	id, userID, tenantID, ok := handlerctx.ResolveResourceIDUserAndTenant(ctx, "事件")
//	if !ok {
//	    return
//	}
func ResolveResourceIDUserAndTenant(c *gin.Context, resourceName string) (int, int, int, bool) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(400, gin.H{
			"code":    1001,
			"message": "无效的" + resourceName + "ID",
		})
		return 0, 0, 0, false
	}
	userID, err := middleware.GetUserID(c)
	if err != nil || userID <= 0 {
		c.AbortWithStatusJSON(401, gin.H{
			"code":    2001,
			"message": "获取用户ID失败",
		})
		return 0, 0, 0, false
	}
	tenantID, ok := ResolveTenantID(c)
	if !ok {
		return 0, 0, 0, false
	}
	return id, userID, tenantID, true
}