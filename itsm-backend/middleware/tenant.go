package middleware

import (
	"errors"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TenantContext struct {
	TenantID int
	Tenant   *ent.Tenant
}

const TenantContextKey = "tenant_context"

// TenantTrustConfig controls whether X-Tenant-Code / Subdomain / Path may override
// the tenant context when JWT claims are missing or different.
//
// Default policy refuses all header overrides. Header-based overrides are only
// honored for SuperAdmin roles, and the resolved tenant must match the JWT
// claims.tenant_id whenever one is present.
type TenantTrustConfig struct {
	// AllowHeaderOverride allows any caller to drive the tenant context via X-Tenant-Code.
	// Production should keep this false; opening it requires audit + alerting.
	AllowHeaderOverride bool
	// SuperAdminRoles still may drive X-Tenant-Code override when AllowHeaderOverride=false,
	// for break-glass operations that must be audited.
	SuperAdminRoles []string
}

// TrustConfig is the process-level security policy for tenant resolution.
// Tests may mutate it; production code should treat it as read-only.
var TrustConfig = TenantTrustConfig{
	AllowHeaderOverride: false,
	SuperAdminRoles:     []string{"super_admin"},
}

func isSuperAdmin(c *gin.Context) bool {
	role, _ := c.Get("role")
	s, _ := role.(string)
	if s == "" {
		return false
	}
	for _, r := range TrustConfig.SuperAdminRoles {
		if r == s {
			return true
		}
	}
	return false
}

// TenantMiddleware resolves the tenant context for the incoming request.
//
// Source priority:
//  1. JWT claims.tenant_id (strong, always enforced when present)
//  2. X-Tenant-Code header (allowed only for SuperAdmin, otherwise rejected)
//  3. Host subdomain (e.g. acme.itsm.example.com -> acme)
//  4. Path param :tenant
//
// The resolved tenant entity must pass status and expiry checks regardless of
// the source, and must equal the JWT tenant when the request carries one.
func TenantMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantEntity *ent.Tenant
		var err error
		var source string

		// 1) JWT claims.tenant_id is the strongest source.
		claimsTenantID := c.GetInt("tenant_id")
		if claimsTenantID > 0 {
			tenantEntity, err = client.Tenant.Get(c.Request.Context(), claimsTenantID)
			if err != nil {
				zap.S().Warnw("jwt tenant_id not found",
					"jwt_tenant_id", claimsTenantID,
					"user_id", c.GetInt("user_id"),
				)
			}
			if tenantEntity != nil {
				source = "jwt"
			}
		}

		// 2) Header takes effect only when JWT did not produce a tenant AND the
		// caller is allowed to override. Default: refuse plain header overrides.
		if tenantEntity == nil {
			headerCode := c.GetHeader("X-Tenant-Code")
			if headerCode != "" {
				if !TrustConfig.AllowHeaderOverride && !isSuperAdmin(c) {
					common.Fail(c, common.AuthFailedCode, "X-Tenant-Code 不可信，请通过 JWT 登录")
					c.Abort()
					return
				}
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(headerCode)).
					First(c.Request.Context())
				if tenantEntity != nil {
					source = "header"
				}
			}
		}

		// 3) Subdomain: only when no JWT and no header produced a tenant.
		if tenantEntity == nil {
			if code := extractTenantFromHost(c.Request.Host); code != "" {
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(code)).
					First(c.Request.Context())
				if tenantEntity != nil {
					source = "subdomain"
				}
			}
		}

		// 4) Path parameter :tenant (e.g. /api/v1/tenants/{tenant}/...).
		if tenantEntity == nil {
			if code := c.Param("tenant"); code != "" {
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(code)).
					First(c.Request.Context())
				if tenantEntity != nil {
					source = "path"
				}
			}
		}

		if tenantEntity == nil {
			common.Fail(c, common.ParamErrorCode, "租户信息缺失")
			c.Abort()
			return
		}

		// 5) Header/Subdomain/Path must never disagree with JWT.tenant_id.
		if claimsTenantID > 0 && tenantEntity.ID != claimsTenantID {
			zap.S().Warnw("tenant mismatch rejected",
				"resolved_tenant_id", tenantEntity.ID,
				"jwt_tenant_id", claimsTenantID,
				"source", source,
				"user_id", c.GetInt("user_id"),
			)
			common.Fail(c, common.AuthFailedCode, "租户不匹配")
			c.Abort()
			return
		}

		// 6) Status/expiry checks run for every source, including JWT-derived ones.
		// This closes the suspended-tenant-with-live-jwt bypass.
		if tenantEntity.Status != "active" {
			common.Forbidden(c, "租户已被暂停或过期")
			c.Abort()
			return
		}
		if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
			common.Forbidden(c, "租户已过期")
			c.Abort()
			return
		}

		tenantCtx := &TenantContext{
			TenantID: tenantEntity.ID,
			Tenant:   tenantEntity,
		}
		c.Set(TenantContextKey, tenantCtx)
		c.Set("tenant_id", tenantEntity.ID)
		c.Set("tenant_source", source)

		c.Next()
	}
}

// extractTenantFromHost parses the leading subdomain of the Host header,
// e.g. tenant1.itsm.example.com -> tenant1. Returns "" when the host is too
// short to be a tenant subdomain.
func extractTenantFromHost(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		return parts[0]
	}
	return ""
}

// GetTenantContext returns the per-request TenantContext. The bool reports
// whether the middleware actually ran for this request.
func GetTenantContext(c *gin.Context) (*TenantContext, bool) {
	value, exists := c.Get(TenantContextKey)
	if !exists {
		return nil, false
	}
	tenantCtx, ok := value.(*TenantContext)
	return tenantCtx, ok
}

// GetTenantID returns the tenant id stamped by the middleware, or an error if
// the middleware never ran (e.g. unauthenticated public route).
func GetTenantID(c *gin.Context) (int, error) {
	tenantCtx, exists := GetTenantContext(c)
	if !exists {
		return 0, errors.New("租户上下文不存在")
	}
	return tenantCtx.TenantID, nil
}

// GetUserID returns the user id stamped by AuthMiddleware.
func GetUserID(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("用户ID不存在")
	}
	if id, ok := userID.(int); ok {
		return id, nil
	}
	return 0, errors.New("用户ID类型错误")
}
