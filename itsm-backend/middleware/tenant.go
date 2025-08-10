package middleware

import (
	"errors"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"net/http"
	"strconv"
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
		// 优先从头部读取租户代码
		tenantCode := c.GetHeader("X-Tenant-Code")
		var tenantEntity *ent.Tenant
		var err error
		// 1) 如果没有提供租户代码，尝试从 JWT 的 tenant_id 或 X-Tenant-ID 解析
		if tenantCode == "" {
			claimedTenantID := c.GetInt("tenant_id")
			if claimedTenantID <= 0 {
				// 尝试从头部 X-Tenant-ID 读取
				if s := c.GetHeader("X-Tenant-ID"); s != "" {
					if id, convErr := strconv.Atoi(s); convErr == nil {
						claimedTenantID = id
					}
				}
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
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1001,
				"message": "租户信息缺失",
				"data":    nil,
			})
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
				c.JSON(http.StatusNotFound, gin.H{
					"code":    1002,
					"message": "租户不存在",
					"data":    nil,
				})
				c.Abort()
				return
			}
		}

		// 检查租户状态
		if tenantEntity.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    1003,
				"message": "租户已被暂停或过期",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查租户是否过期
		if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
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
