package middleware

import (
	"errors"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"net/http"
	"strings"
	"time"

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
		// 从请求头或子域名获取租户信息
		tenantCode := c.GetHeader("X-Tenant-Code")
		if tenantCode == "" {
			// 从子域名提取租户代码
			host := c.Request.Host
			tenantCode = extractTenantFromHost(host)
		}

		// 如果还是没有，尝试从路径参数获取
		if tenantCode == "" {
			tenantCode = c.Param("tenant")
		}

		if tenantCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1001,
				"message": "租户信息缺失",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 查询租户信息
		tenantEntity, err := client.Tenant.
			Query().
			Where(tenant.CodeEQ(tenantCode)).
			First(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    1002,
				"message": "租户不存在",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查租户状态
		if tenantEntity.Status != tenant.StatusActive {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    1003,
				"message": "租户已被暂停或过期",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查租户是否过期
		if tenantEntity.ExpiresAt != nil && tenantEntity.ExpiresAt.Before(time.Now()) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    1004,
				"message": "租户已过期",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 设置租户上下文
		tenantCtx := &TenantContext{
			TenantID: tenantEntity.ID,
			Tenant:   tenantEntity,
		}
		c.Set(TenantContextKey, tenantCtx)

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
