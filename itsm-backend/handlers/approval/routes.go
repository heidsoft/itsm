package approval

import (
	"errors"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// tenantID 提取租户上下文；沿用 handlerctx 契约（401 语义），
// 与旧 controller 的 getIntFromContext + AuthFailedCode 行为等价。
func tenantID(c *gin.Context) (int, bool) {
	return handlerctx.RequireTenantID(c)
}

// userID 提取当前用户上下文
func userID(c *gin.Context) (int, bool) {
	uid := c.GetInt("user_id")
	if uid <= 0 {
		common.Fail(c, common.AuthFailedCode, "无效的用户ID")
		return 0, false
	}
	return uid, true
}

func pathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工作流ID: "+err.Error())
		return 0, false
	}
	return id, true
}

// MigrateWorkflowToBPMN 迁移审批工作流到 BPMN
func (h *Handler) MigrateWorkflowToBPMN(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	tid, ok := tenantID(c)
	if !ok {
		return
	}
	dryRun := c.Query("dryRun") == "true"
	result, err := h.approvalService.MigrateWorkflowToBPMN(c.Request.Context(), id, tid, dryRun)
	if err != nil {
		common.InternalError(c, "迁移审批工作流失败: "+err.Error())
		return
	}
	common.Success(c, result)
}

// CreateWorkflow 创建审批工作流
func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req dto.CreateApprovalWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	response, err := h.approvalService.CreateWorkflow(c.Request.Context(), &req, tid)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "创建工作流失败: "+err.Error())
		return
	}

	common.Success(c, response)
}

// UpdateWorkflow 更新审批工作流
func (h *Handler) UpdateWorkflow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	var req dto.UpdateApprovalWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	response, err := h.approvalService.UpdateWorkflow(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "更新工作流失败: "+err.Error())
		return
	}

	common.Success(c, response)
}

// DeleteWorkflow 删除审批工作流
func (h *Handler) DeleteWorkflow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	if err := h.approvalService.DeleteWorkflow(c.Request.Context(), id, tid); err != nil {
		common.Fail(c, common.InternalErrorCode, "删除工作流失败: "+err.Error())
		return
	}

	common.Success(c, map[string]string{"message": "工作流已删除"})
}

// ListWorkflows 获取审批工作流列表
func (h *Handler) ListWorkflows(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	// 强类型过滤条件，取代 map[string]interface{}
	filter := &dto.WorkflowListFilter{}
	if ticketType := c.Query("ticketType"); ticketType != "" {
		filter.TicketType = ticketType
	}
	if priority := c.Query("priority"); priority != "" {
		filter.Priority = priority
	}
	if isActive := c.Query("isActive"); isActive != "" {
		val := isActive == "true"
		filter.IsActive = &val
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	pageSize := 20
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	workflows, total, err := h.approvalService.ListWorkflows(c.Request.Context(), filter, tid, page, pageSize)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取工作流列表失败: "+err.Error())
		return
	}

	common.Success(c, map[string]interface{}{
		"items":    workflows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetWorkflow 获取审批工作流详情
func (h *Handler) GetWorkflow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	workflow, err := h.approvalService.GetWorkflow(c.Request.Context(), id, tid)
	if err != nil {
		// 检查是否是"未找到"错误
		if err.Error() == "ent: not found" || strings.Contains(err.Error(), "not found") {
			c.JSON(404, common.Response{
				Code:    404,
				Message: "审批工作流不存在",
				Data:    nil,
			})
			return
		}
		common.Fail(c, common.InternalErrorCode, "获取工作流失败: "+err.Error())
		return
	}

	common.Success(c, workflow)
}

// PatchWorkflow 部分更新审批工作流
func (h *Handler) PatchWorkflow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	var req dto.UpdateApprovalWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	response, err := h.approvalService.UpdateWorkflow(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "更新工作流失败: "+err.Error())
		return
	}

	common.Success(c, response)
}

// GetApprovalRecords 获取审批记录
func (h *Handler) GetApprovalRecords(c *gin.Context) {
	var req dto.GetApprovalRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 尝试从查询参数获取
		if ticketIDStr := c.Query("ticketId"); ticketIDStr != "" {
			if ticketID, err := strconv.Atoi(ticketIDStr); err == nil {
				req.TicketID = &ticketID
			}
		}
		if workflowIDStr := c.Query("workflowId"); workflowIDStr != "" {
			if workflowID, err := strconv.Atoi(workflowIDStr); err == nil {
				req.WorkflowID = &workflowID
			}
		}
		if status := c.Query("status"); status != "" {
			req.Status = &status
		}
		req.Page = 1
		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				req.Page = p
			}
		}
		req.PageSize = 20
		if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				req.PageSize = ps
			}
		}
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	records, total, err := h.approvalService.GetApprovalRecords(c.Request.Context(), &req, tid)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "获取审批记录失败: "+err.Error())
		return
	}

	common.Success(c, map[string]interface{}{
		"items":    records,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

// SubmitApproval 提交审批
func (h *Handler) SubmitApproval(c *gin.Context) {
	var req struct {
		TicketID         int    `json:"ticketId" binding:"required"`
		ApprovalID       int    `json:"approvalId" binding:"required"`
		Action           string `json:"action" binding:"required,oneof=approve reject delegate"`
		Comment          string `json:"comment"`
		DelegateToUserID *int   `json:"delegateToUserId,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	uid, ok := userID(c)
	if !ok {
		return
	}

	if err := h.approvalService.SubmitApproval(
		c.Request.Context(),
		req.ApprovalID,
		uid,
		req.Action,
		req.Comment,
		req.DelegateToUserID,
		tid,
	); err != nil {
		// 按领域哨兵错误分流：并发冲突/越级 → 409，记录不存在 → 404，
		// 非指定审批人 → 403；仅真实故障保留 500（T-2 观察项修复）。
		switch {
		case errors.Is(err, service.ErrApprovalRecordProcessed),
			errors.Is(err, service.ErrApprovalOutOfOrder):
			common.Conflict(c, err.Error(), nil)
		case errors.Is(err, service.ErrApprovalRecordNotFound):
			common.Fail(c, common.NotFoundCode, "审批记录不存在")
		case errors.Is(err, service.ErrApprovalNotAuthorized):
			common.Fail(c, common.ForbiddenCode, "当前用户不是该审批记录的指定审批人")
		default:
			common.Fail(c, common.InternalErrorCode, "提交审批失败: "+err.Error())
		}
		return
	}

	common.Success(c, map[string]string{"message": "审批已提交"})
}
