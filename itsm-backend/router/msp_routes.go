package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/middleware"
)

// SetupMSPRoutes 注册 MSP（托管服务商）跨租户路由。
// 注意：MSP 跨租户访问，不需要租户中间件，直接挂在 /api/v1/msp 根组。
func SetupMSPRoutes(r *gin.Engine, config *RouterConfig) {
	if config.MSPHandler == nil {
		return
	}
	msp := r.Group("/api/v1/msp")
	msp.Use(middleware.AuthMiddleware(config.JWTSecret))
	msp.Use(middleware.RBACMiddleware(config.Client)) // 设置 client 到 context
	msp.Use(middleware.MSPMiddleware(config.Client))
	{
		// MSP 身份自述 - 不含客户数据，MSP 管理员语义见下
		// 保持 AUTH_ONLY（ADR-005 D3）：前端靠它判断展示工作台还是权限拒绝态，
		// 套 RequireMSPPermission 会让非 MSP 用户收到 403 而不是可判定的响应。
		// 它只返回调用者自身身份，数据端点仍逐请求带权限码。
		msp.GET("/status", config.MSPHandler.GetMSPStatus)
		msp.GET("/context", middleware.RequireMSPPermission("msp", "read"), config.MSPHandler.GetMSPContext)

		// 分配管理 - 需要 msp_allocation 权限
		msp.GET("/allocations", middleware.RequireMSPPermission("msp_allocation", "read"), config.MSPHandler.GetAllocations)
		msp.GET("/allocations/history", middleware.RequireMSPPermission("msp_allocation", "read"), config.MSPHandler.GetAllocationHistory)
		msp.POST("/allocations", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPHandler.CreateAllocation)
		msp.POST("/allocations/deallocate", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPHandler.Deallocate)

		// 客户管理 - 需要 msp_customer 权限
		msp.GET("/customers", middleware.RequireMSPPermission("msp_customer", "read"), config.MSPHandler.GetAllCustomers)
		msp.GET("/customers/:customer_tenant_id/tickets", middleware.RequireMSPPermission("msp_ticket", "read"), config.MSPHandler.GetCustomerTickets)

		// 工单分配 - 需要 msp_ticket:write 权限
		msp.POST("/tickets/:id/assign", middleware.RequireMSPPermission("msp_ticket", "write"), config.MSPHandler.AssignMSPTechnician)

		// 报表 - 需要 msp_report 权限
		msp.GET("/reports/customers", middleware.RequireMSPPermission("msp_report", "read"), config.MSPHandler.GetCustomerReports)
		msp.GET("/reports/performance", middleware.RequireMSPPermission("msp_report", "read"), config.MSPHandler.GetPerformanceReports)
	}
}
