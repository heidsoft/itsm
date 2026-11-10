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
				"message": "Accessible only to MSP employees",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// IsMSPManagerRole 判断 MSP 原生角色是否具备服务商管理员权限。
// 这是「MSP 管理员」的唯一口径：RequireMSPManager 的放行条件与 GET /msp/status
// 返回给前端的 isAdmin 展示位必须同源，否则 UI 会把只有读权限的 provider_agent
// 导航进管理页，再在每个写操作上收到 403。
func IsMSPManagerRole(mspRole string) bool {
	return mspRole == MSPRoleManager
}

// RequireMSPManager 在 RequireMSPAccess 基础上额外要求 Role == provider_admin。
// 用于租户开通、计费查询、跨客户批量操作等高风险面。
func RequireMSPManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContext(c)
		if !exists || !mspCtx.IsMSP {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "Accessible only to MSP employees",
			})
			c.Abort()
			return
		}
		if !IsMSPManagerRole(mspCtx.Role) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "Only MSP managers can perform this operation",
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
				"message": "MSP requests must provide the X-Customer-Tenant-ID header",
			})
			c.Abort()
			return
		}

		tenantID, err := strconv.Atoi(header)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "Customer tenant ID format error",
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
				"message": "Not authorized to access the specified customer tenant",
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
				"message": "Accessible only to MSP employees",
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
				"message": "Tenant information is missing",
			})
			c.Abort()
			return
		}
		tenantID := tenantIDInterface.(int)

		clientInterface, exists := c.Get("client")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Client missing",
			})
			c.Abort()
			return
		}
		client := clientInterface.(*ent.Client)

		if !hasResourcePermission(c.Request.Context(), client, rbacRole, resource, action, tenantID) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "Insufficient permissions",
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
					"message": "The target customer tenant ID format is incorrect.",
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
					"message": "MSP employees do not have access to this customer tenant",
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
