package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// ApprovalController 审批流程控制器
type ApprovalController struct {
	approvalService *service.ApprovalService
}

// NewApprovalController 创建审批流程控制器实例
func NewApprovalController(approvalService *service.ApprovalService) *ApprovalController {
	return &ApprovalController{
		approvalService: approvalService,
	}
}

// CreateWorkflow 创建审批工作流
func (c *ApprovalController) CreateWorkflow(ctx *gin.Context) {
	var req dto.CreateApprovalWorkflowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.approvalService.CreateWorkflow(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// UpdateWorkflow 更新审批工作流
func (c *ApprovalController) UpdateWorkflow(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return
	}

	var req dto.UpdateApprovalWorkflowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.approvalService.UpdateWorkflow(ctx.Request.Context(), id, &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// DeleteWorkflow 删除审批工作流
func (c *ApprovalController) DeleteWorkflow(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	err = c.approvalService.DeleteWorkflow(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "工作流已删除"})
}

// ListWorkflows 获取审批工作流列表
func (c *ApprovalController) ListWorkflows(ctx *gin.Context) {
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	filters := make(map[string]interface{})
	if ticketType := ctx.Query("ticket_type"); ticketType != "" {
		filters["ticket_type"] = ticketType
	}
	if priority := ctx.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if isActive := ctx.Query("is_active"); isActive != "" {
		filters["is_active"] = isActive == "true"
	}

	page := 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	pageSize := 20
	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	workflows, total, err := c.approvalService.ListWorkflows(ctx.Request.Context(), filters, tenantID.(int), page, pageSize)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取工作流列表失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]interface{}{
		"items":     workflows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetWorkflow 获取审批工作流详情
func (c *ApprovalController) GetWorkflow(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	workflow, err := c.approvalService.GetWorkflow(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, workflow)
}

// GetApprovalRecords 获取审批记录
func (c *ApprovalController) GetApprovalRecords(ctx *gin.Context) {
	var req dto.GetApprovalRecordsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 尝试从查询参数获取
		if ticketIDStr := ctx.Query("ticket_id"); ticketIDStr != "" {
			if ticketID, err := strconv.Atoi(ticketIDStr); err == nil {
				req.TicketID = &ticketID
			}
		}
		if workflowIDStr := ctx.Query("workflow_id"); workflowIDStr != "" {
			if workflowID, err := strconv.Atoi(workflowIDStr); err == nil {
				req.WorkflowID = &workflowID
			}
		}
		if status := ctx.Query("status"); status != "" {
			req.Status = &status
		}
		req.Page = 1
		if pageStr := ctx.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				req.Page = p
			}
		}
		req.PageSize = 20
		if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				req.PageSize = ps
			}
		}
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	records, total, err := c.approvalService.GetApprovalRecords(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取审批记录失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]interface{}{
		"items":     records,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
	})
}

// SubmitApproval 提交审批
func (c *ApprovalController) SubmitApproval(ctx *gin.Context) {
	var req struct {
		TicketID        int    `json:"ticket_id" binding:"required"`
		ApprovalID      int    `json:"approval_id" binding:"required"`
		Action          string `json:"action" binding:"required,oneof=approve reject delegate"`
		Comment         string `json:"comment"`
		DelegateToUserID *int  `json:"delegate_to_user_id,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// 获取当前用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 用户信息缺失")
		return
	}

	err := c.approvalService.SubmitApproval(
		ctx.Request.Context(),
		req.ApprovalID,
		userID.(int),
		req.Action,
		req.Comment,
		req.DelegateToUserID,
		tenantID.(int),
	)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "提交审批失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "审批已提交"})
}

