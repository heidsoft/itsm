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

// TenantMiddleware 租户中间件
//
// 来源优先级:JWT claims.tenant_id (强) > X-Tenant-Code > Subdomain > Path
// 任一来源成功后,必须再做 status/expires 校验;当 JWT 带 tenant_id 时,最终结果
// 必须与之相等,否则拒绝。
//
// 备注:`X-Tenant-ID` 被刻意忽略(由更早的修复注释确认),不要把它加回来。
func TenantMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantEntity *ent.Tenant
		var err error
		var source string

		// 1) JWT claims 中的 tenant_id 是最强来源。
		// JWT 一旦锁定,后续任何来源都不能与之不一致。
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

		// 2) Header:仅在 JWT 没拿到实体时作为补充。
		// JWT 已经锁定后,Header 结果必须 == JWT (稍后统一校验)。
		if tenantEntity == nil {
			if code := c.GetHeader("X-Tenant-Code"); code != "" {
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(code)).
					First(c.Request.Context())
				if err != nil {
					if ent.IsNotFound(err) {
						common.NotFound(c, "租户不存在")
						c.Abort()
						return
					}
					zap.S().Errorw("tenant lookup failed", "source", "header", "error", err)
					common.Fail(c, common.InternalErrorCode, "租户查询失败")
					c.Abort()
					return
				}
				if tenantEntity != nil {
					source = "header"
				}
			}
		}

		// 3) Subdomain:仅在没有 JWT 且未配 Header 时作为兜底(同源多租户 SaaS 部署)。
		if tenantEntity == nil {
			if code := extractTenantFromHost(c.Request.Host); code != "" {
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(code)).
					First(c.Request.Context())
				if err != nil {
					if ent.IsNotFound(err) {
						common.NotFound(c, "租户不存在")
						c.Abort()
						return
					}
					zap.S().Errorw("tenant lookup failed", "source", "subdomain", "error", err)
					common.Fail(c, common.InternalErrorCode, "租户查询失败")
					c.Abort()
					return
				}
				if tenantEntity != nil {
					source = "subdomain"
				}
			}
		}

		// 4) Path 参数:例如 /api/v1/tenants/{tenant}/...。
		if tenantEntity == nil {
			if code := c.Param("tenant"); code != "" {
				tenantEntity, err = client.Tenant.
					Query().
					Where(tenant.CodeEQ(code)).
					First(c.Request.Context())
				if err != nil {
					if ent.IsNotFound(err) {
						common.NotFound(c, "租户不存在")
						c.Abort()
						return
					}
					zap.S().Errorw("tenant lookup failed", "source", "path", "error", err)
					common.Fail(c, common.InternalErrorCode, "租户查询失败")
					c.Abort()
					return
				}
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

		// 5) JWT 与最终结果不一致 → 拒绝。
		// 这条对所有 header/subdomain/path 来源都生效,阻断跨租户越权。
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

		// 6) 状态/有效期:跨所有来源都必须校验,防止 suspended/expired 租户的 live JWT 仍可访问。
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

// extractTenantFromHost 从主机名提取租户代码
// 例如:tenant1.itsm.example.com -> tenant1
func extractTenantFromHost(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		return parts[0]
	}
	return ""
}

// GetTenantContext 获取租户上下文
func GetTenantContext(c *gin.Context) (*TenantContext, bool) {
	value, exists := c.Get(TenantContextKey)
	if !exists {
		return nil, false
	}
	tenantCtx, ok := value.(*TenantContext)
	return tenantCtx, ok
}

// GetTenantID 获取租户ID
func GetTenantID(c *gin.Context) (int, error) {
	tenantCtx, exists := GetTenantContext(c)
	if !exists {
		return 0, errors.New("租户上下文不存在")
	}
	return tenantCtx.TenantID, nil
}

// GetUserID 获取用户ID
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
