// Package msp 是 MSP 多租户服务域的 HTTP handler 层（域切片架构）。
// 自 controller/msp_controller.go 迁移而来（2026-09-02），
// 业务逻辑仍由 service.MSPAllocationService / service.TicketService 承载，
// 本包只做参数解析与响应封装。
package msp

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler MSP HTTP handler
type Handler struct {
	mspAllocationService *service.MSPAllocationService
	ticketService        *service.TicketService
	logger               *zap.SugaredLogger
}

// NewHandler 创建 MSP handler 实例
func NewHandler(
	mspAllocationService *service.MSPAllocationService,
	ticketService *service.TicketService,
	logger *zap.SugaredLogger,
) *Handler {
	return &Handler{
		mspAllocationService: mspAllocationService,
		ticketService:        ticketService,
		logger:               logger,
	}
}

// parseDateOrZero 解析 YYYY-MM-DD 格式日期，失败时返回零值
func parseDateOrZero(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

// GetMSPStatus 获取当前用户的 MSP 状态
func (h *Handler) GetMSPStatus(c *gin.Context) {
	mspCtx, exists := middleware.GetMSPContext(c)

	userRole, _ := c.Get("user_role")
	isAdmin := userRole == "super_admin" || userRole == "admin"

	if exists && mspCtx.IsMSP {
		common.Success(c, dto.MSPStatusResponse{
			IsMSP:     true,
			MSPUserID: mspCtx.MSPUserID,
			Role:      mspCtx.Role,
			IsAdmin:   isAdmin,
		})
		return
	}

	if isAdmin {
		common.Success(c, dto.MSPStatusResponse{
			IsMSP:   false,
			IsAdmin: true,
			Message: "管理员模式：可配置MSP功能",
		})
		return
	}

	common.Success(c, dto.MSPStatusResponse{
		IsMSP:   false,
		IsAdmin: false,
		Message: "非MSP用户",
	})
}

// GetMSPContext 获取当前 MSP 上下文
func (h *Handler) GetMSPContext(c *gin.Context) {
	mspCtx, exists := middleware.GetMSPContext(c)
	if !exists || !mspCtx.IsMSP {
		common.Success(c, dto.MSPContext{
			IsMSP: false,
		})
		return
	}

	common.Success(c, dto.MSPContext{
		IsMSP:             mspCtx.IsMSP,
		MSPUserID:        mspCtx.MSPUserID,
		CustomerTenantID:  mspCtx.CustomerTenantID,
		Role:              mspCtx.Role,
		AllowedCustomers:  mspCtx.AllowedCustomers,
	})
}

// GetAllocations 获取 MSP 分配列表
func (h *Handler) GetAllocations(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	allocations, err := h.mspAllocationService.ListByMSPUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to list allocations", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "查询分配列表失败")
		return
	}

	common.Success(c, dto.MSPAllocationListResponse{
		Allocations: allocations,
		Total:       len(allocations),
	})
}

// CreateAllocation 创建 MSP 分配
func (h *Handler) CreateAllocation(c *gin.Context) {
	var req dto.CreateAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	operatorID := c.GetInt("user_id")
	if operatorID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}
	roleVal, _ := c.Get("role")
	operatorRole, _ := roleVal.(string)

	alloc, err := h.mspAllocationService.Create(c.Request.Context(), req.MSPUserID, req.CustomerTenantID, req.Role, operatorRole)
	if err != nil {
		h.logger.Errorw("Failed to create allocation", "error", err, "operator", operatorID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, alloc)
}

// Deallocate 解除 MSP 分配
func (h *Handler) Deallocate(c *gin.Context) {
	var req dto.DeallocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	operatorID := c.GetInt("user_id")
	if operatorID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	err := h.mspAllocationService.Deactivate(c.Request.Context(), req.MSPUserID, req.CustomerTenantID)
	if err != nil {
		h.logger.Errorw("Failed to deallocate", "error", err, "operator", operatorID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "分配已解除"})
}

// GetAllCustomers 获取当前 MSP 员工可访问的所有客户
func (h *Handler) GetAllCustomers(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	customers, err := h.mspAllocationService.GetMSPCustomers(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorw("Failed to list customers", "error", err, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "查询客户列表失败")
		return
	}

	customerDTOs := make([]*dto.CustomerDTO, 0, len(customers))
	for _, customer := range customers {
		customerDTOs = append(customerDTOs, &dto.CustomerDTO{
			ID:   customer.ID,
			Code: customer.Code,
			Name: customer.Name,
		})
	}

	common.Success(c, dto.MSPCustomerListResponse{
		Customers: customerDTOs,
		Total:     len(customerDTOs),
	})
}

// GetCustomerTickets 获取指定客户的工单（MSP 视角）
func (h *Handler) GetCustomerTickets(c *gin.Context) {
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

	tickets, err := h.ticketService.GetCustomerTicketsForMSP(c.Request.Context(), userID, customerTenantID, &status, page, pageSize)
	if err != nil {
		h.logger.Errorw("Failed to get customer tickets", "error", err, "customer_tenant_id", customerTenantID)
		common.Fail(c, common.InternalErrorCode, "查询工单失败")
		return
	}

	common.Success(c, gin.H{
		"tickets": tickets,
		"total":   len(tickets),
	})
}

// AssignMSPTechnician 为工单分配 MSP 技术员
func (h *Handler) AssignMSPTechnician(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "工单ID无效")
		return
	}

	var req dto.AssignMSPTechnicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	assignerID := c.GetInt("user_id")
	if assignerID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}

	ticket, err := h.ticketService.AssignMSPTechnician(c.Request.Context(), ticketID, req.CustomerTenantID, assignerID)
	if err != nil {
		h.logger.Errorw("Failed to assign MSP technician", "error", err, "ticket_id", ticketID)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, ticket)
}

// GetCustomerReports 获取客户服务报表
func (h *Handler) GetCustomerReports(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		common.Fail(c, common.ParamErrorCode, "start_date和end_date为必填参数")
		return
	}

	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "用户未认证")
		return
	}
	mspUserID := userID

	dateFrom, err := parseDateOrZero(startDate)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "start_date格式错误(YYYY-MM-DD)")
		return
	}
	dateTo, err := parseDateOrZero(endDate)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "end_date格式错误(YYYY-MM-DD)")
		return
	}

	reports, err := h.ticketService.GetMSPCustomerReports(c.Request.Context(), mspUserID, dateFrom, dateTo)
	if err != nil {
		h.logger.Errorw("Failed to get customer reports", "error", err)
		common.Fail(c, common.InternalErrorCode, "生成报表失败")
		return
	}

	common.Success(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}

// GetPerformanceReports 获取 MSP 员工绩效报表
func (h *Handler) GetPerformanceReports(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	mspUserIDStr := c.Query("msp_user_id")

	if startDate == "" || endDate == "" {
		common.Fail(c, common.ParamErrorCode, "start_date和end_date为必填参数")
		return
	}

	dateFrom, err := parseDateOrZero(startDate)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "start_date格式错误(YYYY-MM-DD)")
		return
	}
	dateTo, err := parseDateOrZero(endDate)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "end_date格式错误(YYYY-MM-DD)")
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
		mspUserID, err = strconv.Atoi(mspUserIDStr)
		if err != nil {
			common.Fail(c, common.ParamErrorCode, "msp_user_id格式错误")
			return
		}
	}

	reports, err := h.ticketService.GetMSPPerformanceReports(c.Request.Context(), mspUserID, dateFrom, dateTo)
	if err != nil {
		h.logger.Errorw("Failed to get performance reports", "error", err)
		common.Fail(c, common.InternalErrorCode, "生成绩效报表失败")
		return
	}

	common.Success(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}
