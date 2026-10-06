package approval

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		common.ParamErrorWithErr(c, err, "无效的工作流ID")
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
		common.RespondError(c, err, "迁移审批工作流失败")
		return
	}
	common.Success(c, result)
}

// CreateWorkflow 创建审批工作流
func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req dto.CreateApprovalWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	response, err := h.approvalService.CreateWorkflow(c.Request.Context(), &req, tid)
	if err != nil {
		common.RespondError(c, err, "创建工作流失败")
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	response, err := h.approvalService.UpdateWorkflow(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.RespondError(c, err, "更新工作流失败")
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
		common.RespondError(c, err, "删除工作流失败")
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

	pg := common.GetPaginationFromQuery(c)

	workflows, total, err := h.approvalService.ListWorkflows(c.Request.Context(), filter, tid, pg.Page, pg.PageSize)
	if err != nil {
		common.RespondError(c, err, "获取工作流列表失败")
		return
	}

	common.Success(c, map[string]interface{}{
		"items":    workflows,
		"total":    total,
		"page":     pg.Page,
		"pageSize": pg.PageSize,
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
		// service 用 fmt.Errorf("...: %w", err) 包装 *ent.NotFoundError，
		// 由 common.RespondError 经 classifyError 映射到 NotFoundCode(4004) +
		// 「资源不存在或已被删除」safeMsg；其他驱动层错误统一 5001，
		// 原始 err.Error() 仅入 zap。
		common.RespondError(c, err, "获取工作流失败")
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	response, err := h.approvalService.UpdateWorkflow(c.Request.Context(), id, &req, tid)
	if err != nil {
		common.RespondError(c, err, "更新工作流失败")
		return
	}

	common.Success(c, response)
}

// GetApprovalRecords 获取审批记录
// Deprecated: This API is deprecated. Use GET /api/v1/bpmn/tasks instead.
// Sunset: Sat, 01 Nov 2026 00:00:00 GMT
func (h *Handler) GetApprovalRecords(c *gin.Context) {
	zap.S().Warnw("Deprecated API called", "path", c.FullPath(), "method", "GET /approval-records", "successor", "GET /api/v1/bpmn/tasks")
	c.Header("Deprecation", "true")
	c.Header("Sunset", "Sat, 01 Nov 2026 00:00:00 GMT")
	c.Header("Link", `</api/v1/bpmn/tasks>; rel="successor-version"`)

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
		pg := common.GetPaginationFromQuery(c)
		req.Page = pg.Page
		req.PageSize = pg.PageSize
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	records, total, err := h.approvalService.GetApprovalRecords(c.Request.Context(), &req, tid)
	if err != nil {
		common.RespondError(c, err, "获取审批记录失败")
		return
	}

	common.Success(c, map[string]interface{}{
		"items":    records,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
	})
}

// legacyApprovalRetiredCode is the stable business code for retired submissions.
const legacyApprovalRetiredCode = 4100

// SubmitApproval keeps both legacy routes visible but permanently rejects writes.
// Deprecated: use POST /api/v1/bpmn/tasks/:id/decisions with a BPMN task ID.
func (h *Handler) SubmitApproval(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}
	uid, ok := userID(c)
	if !ok {
		return
	}

	// Do not parse or look up legacy IDs: no payload may reactivate this path,
	// reveal record existence, or implicitly migrate historical approval data.
	zap.S().Warnw("Legacy approval submission rejected", "path", c.FullPath(),
		"tenantId", tid, "actorId", uid, "errorClass", "legacy_approval_retired")
	c.Header("Deprecation", "true")
	c.Header("Link", `</api/v1/bpmn/tasks/:id/decisions>; rel="successor-version"`)
	c.AbortWithStatusJSON(http.StatusGone, common.Response{
		Code:    legacyApprovalRetiredCode,
		Message: "旧审批提交已退役，请使用 BPMN 用户任务审批接口；历史记录仍可查询，不会自动迁移或删除。请使用 BPMN 任务 ID，勿复用旧审批记录 ID。",
	})
}
