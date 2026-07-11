package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

// MSPRoleManager 顶层 MSP 角色，常量集中放这里便于引用。
const MSPRoleManager = "provider_admin"

// mspRoleToRBACRoleMap 把 MSP 原生角色映射到 RBAC 角色，供 RequireMSPPermission
// 在 hasResourcePermission 前做转换。已带 msp_ 前缀的角色直接放行。
var mspRoleToRBACRoleMap = map[string]string{
	"provider_admin": "msp_manager",
	"provider_agent": "msp_tech",
	"customer_user":  "end_user",
}

// GetMSPRBACRole 把 MSP 原生角色转换成 RBAC 角色；未识别的回落到 msp_viewer。
func GetMSPRBACRole(mspRole string) string {
	if rbacRole, ok := mspRoleToRBACRoleMap[mspRole]; ok {
		return rbacRole
	}
	return "msp_viewer"
}

// RequireMSPAccess 阻挡非 MSP 用户的访问。仅在 MSP 路由族内挂载。
func RequireMSPAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContext(c)
		if !exists || !mspCtx.IsMSP {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "仅MSP员工可访问",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireMSPManager 在 RequireMSPAccess 基础上额外要求 Role == provider_admin。
// 用于租户开通、计费查询、跨客户批量操作等高风险面。
func RequireMSPManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContext(c)
		if !exists || !mspCtx.IsMSP {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "仅MSP员工可访问",
			})
			c.Abort()
			return
		}
		if mspCtx.Role != MSPRoleManager {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "仅MSP经理可执行此操作",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidateCustomerTenantHeader 强制 MSP 请求必须带 X-Customer-Tenant-ID，
// 并校验该 ID 在 AllowedCustomers 列表中。常用于需要锁定单一客户的 API。
func ValidateCustomerTenantHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContext(c)
		if !exists {
			c.Next()
			return
		}

		header := c.GetHeader("X-Customer-Tenant-ID")
		if header == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "MSP请求必须提供 X-Customer-Tenant-ID 头",
			})
			c.Abort()
			return
		}

		tenantID, err := strconv.Atoi(header)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "客户租户ID格式错误",
			})
			c.Abort()
			return
		}

		hasAccess := false
		for _, allowedID := range mspCtx.AllowedCustomers {
			if allowedID == tenantID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "无权访问指定客户租户",
			})
			c.Abort()
			return
		}

		mspCtx.CustomerTenantID = &tenantID
		c.Set(MSPContextKey, mspCtx)
		c.Next()
	}
}

// RequireMSPPermission 是 RBAC + 客户分配联合检查的统一入口。
// 检查顺序：MSP 身份 → RBAC 资源权限 → 客户分配（如有 X-Customer-Tenant-ID）。
func RequireMSPPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContext(c)
		if !exists || !mspCtx.IsMSP {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "仅MSP员工可访问",
			})
			c.Abort()
			return
		}

		rbacRole := mspCtx.Role
		if !strings.HasPrefix(rbacRole, "msp_") {
			rbacRole = GetMSPRBACRole(rbacRole)
		}

		tenantIDInterface, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "租户信息缺失",
			})
			c.Abort()
			return
		}
		tenantID := tenantIDInterface.(int)

		clientInterface, exists := c.Get("client")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "客户端缺失",
			})
			c.Abort()
			return
		}
		client := clientInterface.(*ent.Client)

		if !hasResourcePermission(client, rbacRole, resource, action, tenantID) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "权限不足",
				"data": gin.H{
					"required_resource": resource,
					"required_action":   action,
					"msp_role":          mspCtx.Role,
					"rbac_role":         rbacRole,
				},
			})
			c.Abort()
			return
		}

		targetCustomerHeader := c.GetHeader("X-Customer-Tenant-ID")
		if targetCustomerHeader != "" {
			targetTenantID, convErr := strconv.Atoi(targetCustomerHeader)
			if convErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    http.StatusBadRequest,
					"message": "目标客户租户ID格式错误",
				})
				c.Abort()
				return
			}

			hasAccess := false
			for _, allowedID := range mspCtx.AllowedCustomers {
				if allowedID == targetTenantID {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    http.StatusForbidden,
					"message": "MSP员工无权访问此客户租户",
					"data": gin.H{
						"target_customer":   targetTenantID,
						"allowed_customers": mspCtx.AllowedCustomers,
					},
				})
				c.Abort()
				return
			}
			mspCtx.CustomerTenantID = &targetTenantID
			c.Set(MSPContextKey, mspCtx)
		}

		c.Next()
	}
}
