package middleware

import (
	"errors"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"

	"github.com/gin-gonic/gin"
)

type TenantContext struct {
	TenantID int
	Tenant   *ent.Tenant
}

const TenantContextKey = "tenant_context"

// TenantMiddleware 租户中间件
func TenantMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从头部读取租户代码
		tenantCode := c.GetHeader("X-Tenant-Code")
		var tenantEntity *ent.Tenant
		var err error
		// 1) 如果没有提供租户代码，尝试从 JWT 的 tenant_id 获取（不信任请求头）
		if tenantCode == "" {
			claimedTenantID := c.GetInt("tenant_id")
			if claimedTenantID <= 0 {
				// 不再从 X-Tenant-ID 请求头获取，防止租户绕过
				// JWT 中的 tenant_id 是唯一的可信来源
			}
			if claimedTenantID > 0 {
				tenantEntity, err = client.Tenant.Get(c.Request.Context(), claimedTenantID)
				if err == nil {
					tenantCode = tenantEntity.Code
				}
			}
		}
		// 2) 如果仍未获取到，尝试从子域名提取
		if tenantCode == "" {
			host := c.Request.Host
			tenantCode = extractTenantFromHost(host)
		}
		// 3) 如果仍未获取到，尝试从路径参数获取
		if tenantCode == "" {
			tenantCode = c.Param("tenant")
		}
		// 4) 若依然缺失，则报错
		if tenantCode == "" && tenantEntity == nil {
			common.Fail(c, common.ParamErrorCode, "租户信息缺失")
			c.Abort()
			return
		}
		// 按租户代码查询（若之前未通过 ID 查询到）
		if tenantEntity == nil {
			tenantEntity, err = client.Tenant.
				Query().
				Where(tenant.CodeEQ(tenantCode)).
				First(c.Request.Context())
			if err != nil {
				common.NotFound(c, "租户不存在")
				c.Abort()
				return
			}
		}

		claimedTenantID := c.GetInt("tenant_id")
		if claimedTenantID > 0 && claimedTenantID != tenantEntity.ID {
			common.Fail(c, common.AuthFailedCode, "租户不匹配")
			c.Abort()
			return
		}

		// 检查租户状态
		if tenantEntity.Status != "active" {
			common.Forbidden(c, "租户已被暂停或过期")
			c.Abort()
			return
		}

		// 检查租户是否过期
		if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
			common.Forbidden(c, "租户已过期")
			c.Abort()
			return
		}

		// 设置租户上下文
		tenantCtx := &TenantContext{
			TenantID: tenantEntity.ID,
			Tenant:   tenantEntity,
		}
		c.Set(TenantContextKey, tenantCtx)
		c.Set("tenant_id", tenantEntity.ID)

		c.Next()
	}
}

// extractTenantFromHost 从主机名提取租户代码
// 例如：tenant1.itsm.example.com -> tenant1
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
