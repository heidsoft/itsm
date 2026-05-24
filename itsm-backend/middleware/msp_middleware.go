package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"

	"github.com/gin-gonic/gin"
)

// MSPContextKey MSP 上下文键名
const MSPContextKey = "msp_context"

// TenantTypeMSP MSP服务提供商类型
const TenantTypeMSP = "msp"

// MSPRoleManager MSP经理角色
const MSPRoleManager = "provider_admin"

// MSPContext MSP 访问上下文
type MSPContext struct {
	IsMSP            bool
	MSPUserID        int
	CustomerTenantID *int
	Role             string
	AllowedCustomers []int
}

// GetMSPContext 获取MSP上下文
func GetMSPContext(c *gin.Context) (*MSPContext, bool) {
	val, exists := c.Get(MSPContextKey)
	if !exists {
		return nil, false
	}
	ctx, ok := val.(*MSPContext)
	return ctx, ok
}

// MSPMiddleware MSP 访问控制中间件
func MSPMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从上下文中获取当前用户 ID (由 AuthMiddleware 设置)
		userIDVal, exists := c.Get("user_id")
		if !exists {
			// 非认证用户，继续后续处理（可能由 AuthMiddleware 处理）
			c.Next()
			return
		}

		userID, ok := userIDVal.(int)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "用户ID格式错误",
			})
			c.Abort()
			return
		}

		// 2. 查询用户及其租户信息
		u, err := client.User.Query().
			Where(user.IDEQ(userID)).
			WithTenant(func(q *ent.TenantQuery) {
				q.Select(tenant.FieldID, tenant.FieldCode, tenant.FieldName, tenant.FieldType)
			}).
			Only(c.Request.Context())
		if err != nil {
			if ent.IsNotFound(err) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    http.StatusUnauthorized,
					"message": "用户不存在",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "查询用户失败",
				})
			}
			c.Abort()
			return
		}

		tenants := u.Edges.Tenant
		if tenants == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "用户租户信息缺失",
			})
			c.Abort()
			return
		}

		tenantObj := tenants

		// 3. 判断是否为 MSP 员工
		// 条件：租户类型为 "msp" 且用户有 msp_role 字段
		// 注意：admin 权限不能绕过 MSP 访问控制，MSP 客户数据访问需要有效的分配验证
		isMSP := false
		var mspRole string

		// MSP 员工判断：用户必须属于 MSP 租户且有 msp_role 字段
		if tenantObj.Type == TenantTypeMSP && u.MspRole != "" {
			isMSP = true
			mspRole = string(u.MspRole)
		}

		ctx := &MSPContext{
			IsMSP:     isMSP,
			MSPUserID: userID,
		}

		// 4. MSP 特殊处理
		if isMSP {
			// 4.1 查询该 MSP 员工的所有有效分配（未解除的）
			allocations, err := client.MSPAllocation.Query().
				Where(
					mspallocation.MspUserIDEQ(userID),
					mspallocation.DeassignedAtIsNil(),
				).
				WithCustomerTenant(func(q *ent.TenantQuery) {
					q.Select(tenant.FieldID, tenant.FieldCode, tenant.FieldName)
				}).
				All(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "查询MSP分配失败",
				})
				c.Abort()
				return
			}

			// 4.2 构建允许访问的客户租户列表
			allowedCustomers := make([]int, 0, len(allocations))
			for _, alloc := range allocations {
				if len(alloc.Edges.CustomerTenant) > 0 {
					allowedCustomers = append(allowedCustomers, alloc.Edges.CustomerTenant[0].ID)
				}
			}

			ctx.Role = mspRole
			ctx.AllowedCustomers = allowedCustomers

			// 4.3 如果请求头中包含目标客户租户，验证权限
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

				// 验证该 MSP 员工是否有权访问此客户
				hasAccess := false
				for _, allowedID := range allowedCustomers {
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
							"msp_user_id":       userID,
							"target_customer":   targetTenantID,
							"allowed_customers": allowedCustomers,
						},
					})
					c.Abort()
					return
				}
				ctx.CustomerTenantID = &targetTenantID
			}
		}

		// 5. 将 MSP 上下文存入 Gin Context
		c.Set(MSPContextKey, ctx)
		c.Next()
	}
}

// GetMSPContextFromGin 从 Gin Context 获取 MSP 上下文
func GetMSPContextFromGin(c *gin.Context) (*MSPContext, bool) {
	val, exists := c.Get(MSPContextKey)
	if !exists {
		return nil, false
	}
	ctx, ok := val.(*MSPContext)
	return ctx, ok
}

// RequireMSPAccess 要求 MSP 权限的中间件
// 用于只有 MSP 员工可访问的路由
func RequireMSPAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContextFromGin(c)
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

// RequireMSPManager 要求 MSP Manager 角色
func RequireMSPManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContextFromGin(c)
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

// ValidateCustomerTenantHeader 验证 X-Customer-Tenant-ID 头（用于需要指定客户的路由）
func ValidateCustomerTenantHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		mspCtx, exists := GetMSPContextFromGin(c)
		if !exists {
			c.Next()
			return
		}

		// 如果是 MSP 员工，必须提供客户租户 ID
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

		// 验证是否有权访问该客户
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

		// 更新上下文中的客户租户 ID
		mspCtx.CustomerTenantID = &tenantID
		c.Set(MSPContextKey, mspCtx)
		c.Next()
	}
}

// MSPCustomerTenantIDKey 用于在context中传递客户租户ID的key
const MSPCustomerTenantIDKey = "msp_customer_tenant_id"

// SetMSPCustomerTenantID 设置客户租户ID到context
func SetMSPCustomerTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, MSPCustomerTenantIDKey, tenantID)
}

// GetMSPCustomerTenantID 从context获取客户租户ID
func GetMSPCustomerTenantID(ctx context.Context) (int, bool) {
	val := ctx.Value(MSPCustomerTenantIDKey)
	if val == nil {
		return 0, false
	}
	tenantID, ok := val.(int)
	return tenantID, ok
}

// MSPFilterByCustomer 根据当前 MSP 上下文自动过滤查询（用于 Service 层）
func MSPFilterByCustomer(ctx context.Context, client *ent.Client, query *ent.TicketQuery) (*ent.TicketQuery, error) {
	tenantID, ok := GetMSPCustomerTenantID(ctx)
	if !ok || tenantID == 0 {
		return query, nil // 非 MSP 或无指定客户，不过滤
	}

	// 只查询指定客户租户的工单
	return query.Where(ticket.TenantIDEQ(tenantID)), nil
}

// GetMSPContextFromContext 从 Go context 获取 MSP 上下文（兼容旧接口）
func GetMSPContextFromContext(ctx context.Context) (*MSPContext, bool) {
	tenantID, ok := GetMSPCustomerTenantID(ctx)
	if !ok || tenantID == 0 {
		return nil, false
	}
	return &MSPContext{
		IsMSP:            true,
		CustomerTenantID: &tenantID,
	}, true
}

// MSP角色到RBAC角色的映射
var mspRoleToRBACRoleMap = map[string]string{
	"provider_admin": "msp_manager",
	"provider_agent": "msp_tech",
	"customer_user":  "end_user",
}

// GetMSPRBACRole 将用户的 MSP 角色转换为 RBAC 角色
func GetMSPRBACRole(mspRole string) string {
	if rbacRole, ok := mspRoleToRBACRoleMap[mspRole]; ok {
		return rbacRole
	}
	// 默认返回 msp_viewer
	return "msp_viewer"
}

// RequireMSPPermission 要求 MSP 权限的中间件
// 支持资源级别的权限检查和客户级别验证
func RequireMSPPermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 MSP 上下文
		mspCtx, exists := GetMSPContextFromGin(c)
		if !exists || !mspCtx.IsMSP {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "仅MSP员工可访问",
			})
			c.Abort()
			return
		}

		// 2. 获取用户的 RBAC 角色
		// 注意：mspCtx.Role 已经是 RBAC 角色（如 msp_manager），不需要再转换
		// 只有当 Role 是原生 MSP 角色（如 provider_admin）时才需要转换
		rbacRole := mspCtx.Role
		if !strings.HasPrefix(rbacRole, "msp_") {
			// 如果不是 RBAC 角色格式，进行转换
			rbacRole = GetMSPRBACRole(rbacRole)
		}

		// 3. 获取租户ID
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

		// 4. 获取客户端
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

		// 5. 检查 RBAC 权限
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

		// 6. 如果请求头中包含目标客户租户，验证客户级别权限
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

			// 验证该 MSP 员工是否有权访问此客户
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
			// 更新上下文中的客户租户 ID
			mspCtx.CustomerTenantID = &targetTenantID
			c.Set(MSPContextKey, mspCtx)
		}

		c.Next()
	}
}
