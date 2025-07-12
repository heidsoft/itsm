package middleware

import (
	"context"
	"itsm-backend/ent"
	"net/http"
	"strconv"

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

		if tenantCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "租户信息缺失"})
			c.Abort()
			return
		}

		// 查询租户信息
		tenant, err := client.Tenant.
			Query().
			Where(tenant.CodeEQ(tenantCode)).
			First(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "租户不存在"})
			c.Abort()
			return
		}

		// 检查租户状态
		if tenant.Status != string(TenantStatusActive) {
			c.JSON(http.StatusForbidden, gin.H{"error": "租户已被暂停或过期"})
			c.Abort()
			return
		}

		// 设置租户上下文
		tenantCtx := &TenantContext{
			TenantID: tenant.ID,
			Tenant:   tenant,
		}
		c.Set(TenantContextKey, tenantCtx)

		c.Next()
	}
}

func extractTenantFromHost(host string) string {
	// 实现从主机名提取租户代码的逻辑
	// 例如：tenant1.itsm.example.com -> tenant1
	return ""
}
