package middleware

import (
	"context"
	"net/http"
	"strconv"

	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/pkg/tenantmode"

	"github.com/gin-gonic/gin"
)

// MSPMiddleware 在请求进入 MSP 路由族时构造 MSPContext：
//   1. 门控 mspEnabled（private 部署直接 404）
//   2. 解析 user_id / 加载用户与租户
//   3. 仅当租户类型为 msp_provider 且 msp_role 非空才认定 MSP 身份
//   4. 拉取有效 MSPAllocation，写入 AllowedCustomers
//   5. 若请求头带 X-Customer-Tenant-ID，验证其是否在分配列表中
//
// 注意：admin 角色不再自动授予 MSP 访问权，必须有真实的 msp_role 和
// allocation 才能访问客户数据（参见 msp_rbac.go 与 plan 文档 Task 9）。
func MSPMiddleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !mspEnabled {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "MSP routes are disabled in this deployment mode",
			})
			c.Abort()
			return
		}
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
		isMSP := false
		var mspRole string
		if tenantmode.IsMSPProviderTenantType(string(tenantObj.Type)) && u.MspRole != "" {
			isMSP = true
			mspRole = string(u.MspRole)
		}

		ctx := &MSPContext{
			IsMSP:     isMSP,
			MSPUserID: userID,
		}

		// 4. MSP 特殊处理
		if isMSP {
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

			allowedCustomers := make([]int, 0, len(allocations))
			for _, alloc := range allocations {
				if alloc.Edges.CustomerTenant != nil {
					allowedCustomers = append(allowedCustomers, alloc.Edges.CustomerTenant.ID)
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

		c.Set(MSPContextKey, ctx)
		c.Next()
	}
}

// MSPFilterByCustomer 根据 Go context 中存放的客户租户 ID 给 ent 查询加
// TenantIDEQ 过滤。非 MSP 请求或未限定客户时原样返回，让 Service 层继续走
// 自己注入的租户过滤。
func MSPFilterByCustomer(ctx context.Context, client *ent.Client, query *ent.TicketQuery) (*ent.TicketQuery, error) {
	tenantID, ok := GetMSPCustomerTenantID(ctx)
	if !ok || tenantID == 0 {
		return query, nil
	}
	return query.Where(ticket.TenantIDEQ(tenantID)), nil
}
