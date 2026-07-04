package controller

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// getIntFromContext safely extracts an int from gin context with type assertion.
// Returns the value and true if successful, or 0 and false if the key is missing or wrong type.
func getIntFromContext(c *gin.Context, key string) (int, bool) {
	val, exists := c.Get(key)
	if !exists {
		return 0, false
	}
	v, ok := val.(int)
	return v, ok
}

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
// @Summary 创建审批工作流
// @Description 创建新的审批工作流定义
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param request body dto.CreateApprovalWorkflowRequest true "工作流信息"
// @Success 200 {object} common.Response
// @Router /api/v1/approval-workflows [post]
func (c *ApprovalController) CreateWorkflow(ctx *gin.Context) {
	var req dto.CreateApprovalWorkflowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	response, err := c.approvalService.CreateWorkflow(ctx.Request.Context(), &req, tid)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// UpdateWorkflow 更新审批工作流
// @Summary 更新审批工作流
// @Description 更新已有的审批工作流定义
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "工作流ID"
// @Param request body dto.UpdateApprovalWorkflowRequest true "工作流信息"
// @Success 200 {object} common.Response
// @Router /api/v1/approval-workflows/{id} [put]
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

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	response, err := c.approvalService.UpdateWorkflow(ctx.Request.Context(), id, &req, tid)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// DeleteWorkflow 删除审批工作流
// @Summary 删除审批工作流
// @Description 删除指定的审批工作流
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "工作流ID"
// @Success 200 {object} common.Response
// @Router /api/v1/approval-workflows/{id} [delete]
func (c *ApprovalController) DeleteWorkflow(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return
	}

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	err = c.approvalService.DeleteWorkflow(ctx.Request.Context(), id, tid)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "工作流已删除"})
}

// ListWorkflows 获取审批工作流列表
// @Summary 获取审批工作流列表
// @Description 获取所有审批工作流，支持分页和过滤
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param ticket_type query string false "工单类型"
// @Param priority query string false "优先级"
// @Param is_active query bool false "是否激活"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} common.Response
// @Router /api/v1/approval-workflows [get]
func (c *ApprovalController) ListWorkflows(ctx *gin.Context) {
	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	// 强类型过滤条件，取代 map[string]interface{}
	filter := &dto.WorkflowListFilter{}
	if ticketType := ctx.Query("ticket_type"); ticketType != "" {
		filter.TicketType = ticketType
	}
	if priority := ctx.Query("priority"); priority != "" {
		filter.Priority = priority
	}
	if isActive := ctx.Query("is_active"); isActive != "" {
		val := isActive == "true"
		filter.IsActive = &val
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

	workflows, total, err := c.approvalService.ListWorkflows(ctx.Request.Context(), filter, tid, page, pageSize)
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

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	workflow, err := c.approvalService.GetWorkflow(ctx.Request.Context(), id, tid)
	if err != nil {
		// 检查是否是"未找到"错误
		if err.Error() == "ent: not found" || strings.Contains(err.Error(), "not found") {
			ctx.JSON(404, common.Response{
				Code:    404,
				Message: "审批工作流不存在",
				Data:    nil,
			})
			return
		}
		common.Fail(ctx, common.InternalErrorCode, "获取工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, workflow)
}

// PatchWorkflow 部分更新审批工作流
func (c *ApprovalController) PatchWorkflow(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return
	}

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	var req dto.UpdateApprovalWorkflowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	response, err := c.approvalService.UpdateWorkflow(ctx.Request.Context(), id, &req, tid)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
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

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	records, total, err := c.approvalService.GetApprovalRecords(ctx.Request.Context(), &req, tid)
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
		TicketID         int    `json:"ticketId" binding:"required"`
		ApprovalID       int    `json:"approvalId" binding:"required"`
		Action           string `json:"action" binding:"required,oneof=approve reject delegate"`
		Comment          string `json:"comment"`
		DelegateToUserID *int   `json:"delegateToUserId,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tid, ok := getIntFromContext(ctx, "tenant_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的租户ID")
		return
	}

	// 获取当前用户ID
	uid, ok := getIntFromContext(ctx, "user_id")
	if !ok {
		common.Fail(ctx, common.AuthFailedCode, "无效的用户ID")
		return
	}

	err := c.approvalService.SubmitApproval(
		ctx.Request.Context(),
		req.ApprovalID,
		uid,
		req.Action,
		req.Comment,
		req.DelegateToUserID,
		tid,
	)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "提交审批失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "审批已提交"})
}
