package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MSPController MSP 管理控制器
type MSPController struct {
	mspAllocationService *service.MSPAllocationService
	ticketService        *service.TicketService
	logger               *zap.SugaredLogger
}

// NewMSPController 创建 MSP 控制器实例
func NewMSPController(
	mspAllocationService *service.MSPAllocationService,
	ticketService *service.TicketService,
	logger *zap.SugaredLogger,
) *MSPController {
	return &MSPController{
		mspAllocationService: mspAllocationService,
		ticketService:        ticketService,
		logger:               logger,
	}
}

// ==================== MSP 状态与上下文 ====================

// GetMSPStatus 获取当前用户的 MSP 状态
// @Summary 获取MSP状态
// @Description 返回当前用户是否是MSP员工
// @Tags MSP管理
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/msp/status [get]
func (mc *MSPController) GetMSPStatus(c *gin.Context) {
	mspCtx, exists := middleware.GetMSPContext(c)
	if !exists {
		common.Success(c, gin.H{
			"is_msp": false,
			"message": "非MSP用户",
		})
		return
	}
	common.Success(c, gin.H{
		"is_msp": mspCtx.IsMSP,
		"msp_user_id": mspCtx.MSPUserID,
		"role": mspCtx.Role,
	})
}

// GetMSPContextHandler 获取当前 MSP 上下文
// @Summary 获取MSP上下文
// @Description 返回MSP员工的分配信息和允许访问的客户列表
// @Tags MSP管理
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/msp/context [get]
func (mc *MSPController) GetMSPContext(c *gin.Context) {
	mspCtx, exists := middleware.GetMSPContext(c)
	if !exists || !mspCtx.IsMSP {
		common.Success(c, gin.H{
			"is_msp": false,
		})
		return
	}

	common.Success(c, gin.H{
		"is_msp":           mspCtx.IsMSP,
		"msp_user_id":     mspCtx.MSPUserID,
		"customer_tenant_id": mspCtx.CustomerTenantID,
		"role":            mspCtx.Role,
		"allowed_customers": mspCtx.AllowedCustomers,
	})
}

// ==================== MSP 分配管理 ====================

// GetAllocations 获取 MSP 分配列表
// @Summary 获取分配列表
// @Description 获取当前MSP员工的所有有效分配
// @Tags MSP管理
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页大小"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/allocations [get]
func (mc *MSPController) GetAllocations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	allocations, err := mc.mspAllocationService.ListByMSPUser(
		c.Request.Context(),
		userID.(int),
	)
	if err != nil {
		mc.logger.Errorw("Failed to list allocations", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "查询分配列表失败")
		return
	}

	common.Success(c, gin.H{
		"allocations": allocations,
		"total":       len(allocations),
	})
}

// CreateAllocation 创建 MSP 分配（仅 MSP Manager）
// @Summary 创建分配
// @Description MSP经理为MSP员工分配客户租户
// @Tags MSP管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.CreateAllocationRequest true "创建分配请求"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/allocations [post]
func (mc *MSPController) CreateAllocation(c *gin.Context) {
	var req dto.CreateAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	// 获取当前操作人（MSP Manager）
	operatorID, _ := c.Get("user_id")
	roleVal, _ := c.Get("role")
	operatorRole, _ := roleVal.(string)

	alloc, err := mc.mspAllocationService.Create(
		c.Request.Context(),
		req.MSPUserID,
		req.CustomerTenantID,
		req.Role,
		operatorRole, // 传递操作者角色用于权限检查
	)
	if err != nil {
		mc.logger.Errorw("Failed to create allocation", "error", err, "operator", operatorID)
		common.Fail(c, common.InternalErrorCode, "创建分配失败: "+err.Error())
		return
	}

	// TODO: 发送通知（可选）
	// mc.notificationService.SendMSPAllocationCreated(...)

	common.Success(c, alloc)
}

// Deallocate 解除 MSP 分配
// @Summary 解除分配
// @Description 解除MSP员工与客户租户的关联
// @Tags MSP管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.DeallocateRequest true "解除分配请求"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/allocations/deallocate [post]
func (mc *MSPController) Deallocate(c *gin.Context) {
	var req dto.DeallocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	operatorID, _ := c.Get("user_id")

	err := mc.mspAllocationService.Deactivate(
		c.Request.Context(),
		req.MSPUserID,
		req.CustomerTenantID,
	)
	if err != nil {
		mc.logger.Errorw("Failed to deallocate", "error", err, "operator", operatorID)
		common.Fail(c, common.InternalErrorCode, "解除分配失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message": "分配已解除",
	})
}

// ==================== MSP 客户管理 ====================

// GetAllCustomers 获取当前 MSP 员工可访问的所有客户
// @Summary 获取客户列表
// @Description 返回当前MSP员工有权访问的客户租户列表
// @Tags MSP管理
// @Produce json
// @Security Bearer
// @Success 200 {object} common.Response
// @Router /api/v1/msp/customers [get]
func (mc *MSPController) GetAllCustomers(c *gin.Context) {
	userID, _ := c.Get("user_id")

	customers, err := mc.mspAllocationService.GetMSPCustomers(
		c.Request.Context(),
		userID.(int),
	)
	if err != nil {
		mc.logger.Errorw("Failed to list customers", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "查询客户列表失败")
		return
	}

	common.Success(c, gin.H{
		"customers": customers,
		"total":     len(customers),
	})
}

// GetCustomerTickets 获取指定客户的工单（MSP 视角）
// @Summary 获取客户工单
// @Description 获取指定客户租户的工单列表（需要X-Customer-Tenant-ID头）
// @Tags MSP管理
// @Produce json
// @Security Bearer
// @Param customer_tenant_id path int true "客户租户ID"
// @Param status query string false "工单状态过滤"
// @Param page query int false "页码"
// @Param page_size query int false "每页大小"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/customers/{customer_tenant_id}/tickets [get]
func (mc *MSPController) GetCustomerTickets(c *gin.Context) {
	customerTenantID, err := strconv.Atoi(c.Param("customer_tenant_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "客户租户ID无效")
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	mspCtx, _ := middleware.GetMSPContext(c)
	userID := mspCtx.MSPUserID

	tickets, err := mc.ticketService.GetCustomerTicketsForMSP(
		c.Request.Context(),
		userID,
		customerTenantID,
		&status,
		page,
		pageSize,
	)
	if err != nil {
		mc.logger.Errorw("Failed to get customer tickets", "error", err, "customer_tenant_id", customerTenantID)
		common.Fail(c, common.InternalErrorCode, "查询工单失败")
		return
	}

	common.Success(c, gin.H{
		"tickets": tickets,
		"total":   len(tickets),
	})
}

// AssignMSPTechnician 为工单分配 MSP 技术员
// @Summary 分配技术员
// @Description MSP经理为工单分配技术员（自动设置managed_by_user_id）
// @Tags MSP管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "工单ID"
// @Param request body dto.AssignMSPTechnicianRequest true "分配请求"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/tickets/{id}/assign [post]
func (mc *MSPController) AssignMSPTechnician(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID无效")
		return
	}

	var req dto.AssignMSPTechnicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	assignerID, _ := c.Get("user_id")

	ticket, err := mc.ticketService.AssignMSPTechnician(
		c.Request.Context(),
		ticketID,
		req.CustomerTenantID,
		assignerID.(int),
	)
	if err != nil {
		mc.logger.Errorw("Failed to assign MSP technician", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, "分配失败: "+err.Error())
		return
	}

	common.Success(c, ticket)
}

// ==================== MSP 报表 ====================

// GetCustomerReports 获取客户服务报表
// @Summary 获取客户报表
// @Description 获取指定时间范围内的客户服务报表
// @Tags MSP管理
// @Produce json
// @Security Bearer
// @Param start_date query string true "开始日期 (YYYY-MM-DD)"
// @Param end_date query string true "结束日期 (YYYY-MM-DD)"
// @Param customer_tenant_id query int false "客户租户ID（不填则返回所有客户）"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/reports/customers [get]
func (mc *MSPController) GetCustomerReports(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	customerTenantIDStr := c.Query("customer_tenant_id")

	if startDate == "" || endDate == "" {
		common.Fail(c, common.ParamErrorCode, "start_date和end_date为必填参数")
		return
	}

	var customerTenantID *int
	if customerTenantIDStr != "" {
		id, err := strconv.Atoi(customerTenantIDStr)
		if err != nil {
			common.Fail(c, common.ParamErrorCode, "customer_tenant_id格式错误")
			return
		}
		customerTenantID = &id
	}

	reports, err := mc.ticketService.GetMSPCustomerReports(
		c.Request.Context(),
		startDate,
		endDate,
		customerTenantID,
	)
	if err != nil {
		mc.logger.Errorw("Failed to get customer reports", "error", err)
		common.Fail(c, common.InternalErrorCode, "生成报表失败")
		return
	}

	common.Success(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}

// GetPerformanceReports 获取 MSP 员工绩效报表
// @Summary 获取绩效报表
// @Description 获取MSP员工绩效报表
// @Tags MSP管理
// @Produce json
// @Security Bearer
// @Param start_date query string true "开始日期"
// @Param end_date query string true "结束日期"
// @Param msp_user_id query int false "MSP用户ID（不填则返回当前用户）"
// @Success 200 {object} common.Response
// @Router /api/v1/msp/reports/performance [get]
func (mc *MSPController) GetPerformanceReports(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	mspUserIDStr := c.Query("msp_user_id")

	if startDate == "" || endDate == "" {
		common.Fail(c, common.ParamErrorCode, "start_date和end_date为必填参数")
		return
	}

	var mspUserID int
	if mspUserIDStr == "" {
		mspCtx, _ := middleware.GetMSPContext(c)
		if !mspCtx.IsMSP {
			common.Fail(c, common.ParamErrorCode, "非MSP用户")
			return
		}
		mspUserID = mspCtx.MSPUserID
	} else {
		var err error
		mspUserID, err = strconv.Atoi(mspUserIDStr)
		if err != nil {
			common.Fail(c, common.ParamErrorCode, "msp_user_id格式错误")
			return
		}
	}

	reports, err := mc.ticketService.GetMSPPerformanceReports(
		c.Request.Context(),
		startDate,
		endDate,
		mspUserID,
	)
	if err != nil {
		mc.logger.Errorw("Failed to get performance reports", "error", err)
		common.Fail(c, common.InternalErrorCode, "生成绩效报表失败")
		return
	}

	common.Success(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}
