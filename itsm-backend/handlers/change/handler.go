package change

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Map domain to DTO
func toDTO(c *Change) *dto.ChangeResponse {
	if c == nil {
		return nil
	}
	res := &dto.ChangeResponse{
		ID:                 c.ID,
		ChangeNumber:       c.ChangeNumber,
		Title:              c.Title,
		Description:        c.Description,
		Justification:      c.Justification,
		Type:               dto.ChangeType(c.Type),
		Status:             dto.ChangeStatus(c.Status),
		Priority:           dto.ChangePriority(c.Priority),
		ImpactScope:        dto.ChangeImpact(c.ImpactScope),
		RiskLevel:          dto.ChangeRisk(c.RiskLevel),
		AssigneeID:         c.AssigneeID,
		CreatedBy:          c.CreatedBy,
		TenantID:           c.TenantID,
		PlannedStartDate:   c.PlannedStartDate,
		PlannedEndDate:     c.PlannedEndDate,
		ActualStartDate:    c.ActualStartDate,
		ActualEndDate:      c.ActualEndDate,
		ImplementationPlan: c.ImplementationPlan,
		RollbackPlan:       c.RollbackPlan,
		AffectedCIs:        c.AffectedCIs,
		RelatedTickets:     c.RelatedTickets,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
	if c.Assignee != nil {
		res.AssigneeName = &c.Assignee.Name
	}
	if c.CreatedByUser != nil {
		res.CreatedByName = c.CreatedByUser.Name
	}
	return res
}

// CreateChange handles POST /api/v1/changes
//
//	@Summary	创建变更单
//	@Description	创建新的变更请求，初始状态为 draft（草稿）。类型支持 normal / standard / emergency。
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	request	body	dto.CreateChangeRequest	true	"创建变更请求"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes [post]
func (h *Handler) CreateChange(c *gin.Context) {
	var req dto.CreateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	changeEntity := &Change{
		Title:              req.Title,
		Description:        req.Description,
		Justification:      req.Justification,
		Type:               string(req.Type),
		Status:             "draft",
		Priority:           string(req.Priority),
		ImpactScope:        string(req.ImpactScope),
		RiskLevel:          string(req.RiskLevel),
		CreatedBy:          userID,
		TenantID:           tenantID,
		PlannedStartDate:   req.PlannedStartDate,
		PlannedEndDate:     req.PlannedEndDate,
		ImplementationPlan: req.ImplementationPlan,
		RollbackPlan:       req.RollbackPlan,
		AffectedCIs:        req.AffectedCIs,
		RelatedTickets:     req.RelatedTickets,
	}

	res, err := h.svc.CreateChange(c.Request.Context(), changeEntity)
	if err != nil {
		common.InternalError(c, "创建变更失败: "+err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// GetChange handles GET /api/v1/changes/:id
//
//	@Summary	获取变更详情
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	404	{object}	common.Response
//	@Router	/api/v1/changes/{id} [get]
func (h *Handler) GetChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetChange(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Change not found")
		return
	}

	common.Success(c, toDTO(res))
}

// GetApprovalSummary handles GET /api/v1/changes/:id/approval-summary
//
//	@Summary	获取变更审批摘要
//	@Description	返回该变更的审批链进度摘要（各节点状态与意见）
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/approval-summary [get]
func (h *Handler) GetApprovalSummary(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	summary, err := h.svc.GetApprovalSummary(c.Request.Context(), id, tenantID)
	if err != nil {
		common.InternalError(c, "获取审批摘要失败: "+err.Error())
		return
	}

	common.Success(c, summary)
}

// GetRiskAssessment handles GET /api/v1/changes/:id/risk-assessment
//
//	@Summary	获取变更风险评估
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response{data=dto.ChangeRiskAssessment}
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/risk-assessment [get]
//	@Router	/api/v1/changes/{id}/risk [get]
func (h *Handler) GetRiskAssessment(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)

	ra, err := h.svc.GetRisk(c.Request.Context(), id, tenantID)
	if err != nil {
		common.InternalError(c, "获取风险评估失败: "+err.Error())
		return
	}

	if ra == nil {
		common.Success(c, nil)
		return
	}

	common.Success(c, dto.ChangeRiskAssessment{
		ID:                 ra.ID,
		ChangeID:           ra.ChangeID,
		RiskLevel:          dto.ChangeRisk(ra.RiskLevel),
		RiskDescription:    ra.RiskDescription,
		ImpactAnalysis:     ra.ImpactAnalysis,
		MitigationMeasures: ra.MitigationMeasures,
		ContingencyPlan:    ra.ContingencyPlan,
		RiskOwner:          ra.RiskOwner,
		RiskReviewDate:     ra.RiskReviewDate,
		CreatedAt:          ra.CreatedAt,
		UpdatedAt:          ra.UpdatedAt,
	})
}

// UpdateRisk handles PUT /api/v1/changes/:id/risk
//
//	@Summary	更新变更风险评估
//	@Description	RiskLevel 仅接受 low / medium / high
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	dto.ChangeRiskAssessment	true	"风险评估内容"
//	@Success	200	{object}	common.Response{data=dto.ChangeRiskAssessment}
//	@Failure	400	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/risk [put]
func (h *Handler) UpdateRisk(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	var req dto.ChangeRiskAssessment
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}
	if req.RiskLevel != dto.ChangeRiskLow &&
		req.RiskLevel != dto.ChangeRiskMedium &&
		req.RiskLevel != dto.ChangeRiskHigh {
		common.ParamError(c, "Invalid risk level")
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDVal.(int)
	assessment, err := h.svc.UpdateRisk(c.Request.Context(), &RiskAssessment{
		ChangeID:           id,
		TenantID:           tenantID,
		RiskLevel:          string(req.RiskLevel),
		RiskDescription:    req.RiskDescription,
		ImpactAnalysis:     req.ImpactAnalysis,
		MitigationMeasures: req.MitigationMeasures,
		ContingencyPlan:    req.ContingencyPlan,
		RiskOwner:          req.RiskOwner,
		RiskReviewDate:     req.RiskReviewDate,
	})
	if err != nil {
		common.InternalError(c, "更新风险评估失败: "+err.Error())
		return
	}
	common.Success(c, dto.ChangeRiskAssessment{
		ID:                 assessment.ID,
		ChangeID:           assessment.ChangeID,
		RiskLevel:          dto.ChangeRisk(assessment.RiskLevel),
		RiskDescription:    assessment.RiskDescription,
		ImpactAnalysis:     assessment.ImpactAnalysis,
		MitigationMeasures: assessment.MitigationMeasures,
		ContingencyPlan:    assessment.ContingencyPlan,
		RiskOwner:          assessment.RiskOwner,
		RiskReviewDate:     assessment.RiskReviewDate,
		CreatedAt:          assessment.CreatedAt,
		UpdatedAt:          assessment.UpdatedAt,
	})
}

// GetCMDBImpactSummary handles GET /api/v1/changes/:id/cmdb-impact
//
//	@Summary	获取变更的 CMDB 影响摘要
//	@Description	返回受影响配置项（AffectedCIs）的汇总影响信息
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/cmdb-impact [get]
func (h *Handler) GetCMDBImpactSummary(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	summary, err := h.svc.GetCMDBImpactSummary(c.Request.Context(), id, tenantID)
	if err != nil {
		common.InternalError(c, "获取CMDB影响摘要失败: "+err.Error())
		return
	}

	common.Success(c, summary)
}

// ListChanges handles GET /api/v1/changes
//
//	@Summary	获取变更列表
//	@Description	分页查询变更，支持状态、关键词与风险等级过滤；含行级数据权限过滤
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	page	query	int	false	"页码（默认 1）"
//	@Param	pageSize	query	int	false	"每页数量（默认 10）"
//	@Param	status	query	string	false	"状态过滤"	Enums(draft,pending,approved,rejected,scheduled,in_progress,completed,failed,rolled_back,cancelled,closed)
//	@Param	search	query	string	false	"标题/描述关键词"
//	@Param	riskLevel	query	string	false	"风险等级过滤（同时兼容 risk_level）"	Enums(low,medium,high)
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes [get]
func (h *Handler) ListChanges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")
	search := c.Query("search")
	// 支持 risk_level 与 riskLevel 两种命名，前端一般发 camelCase
	riskLevel := c.Query("risk_level")
	if riskLevel == "" {
		riskLevel = c.Query("riskLevel")
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得，下传给 service 判定 DataScope。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	list, total, err := h.svc.ListChanges(c.Request.Context(), tenantID, page, pageSize, status, search, riskLevel, currentUserID, currentRole)
	if err != nil {
		common.InternalError(c, "查询变更列表失败: "+err.Error())
		return
	}

	// 空列表必须序列化为 [] 而非 null，否则前端列表页渲染失败
	dtos := make([]dto.ChangeResponse, 0, len(list))
	for _, item := range list {
		dtos = append(dtos, *toDTO(item))
	}

	common.Success(c, gin.H{
		"changes":  dtos,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// UpdateChange handles PUT /api/v1/changes/:id
//
//	@Summary	更新变更单
//	@Description	部分更新：仅提交字段生效
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	dto.UpdateChangeRequest	true	"更新变更请求"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id} [put]
func (h *Handler) UpdateChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}

	// First get existing
	existing, err := h.svc.GetChange(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Change not found")
		return
	}

	// Update fields if present in request
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Justification != nil {
		existing.Justification = *req.Justification
	}
	if req.Type != nil {
		existing.Type = string(*req.Type)
	}
	if req.Priority != nil {
		existing.Priority = string(*req.Priority)
	}
	if req.ImpactScope != nil {
		existing.ImpactScope = string(*req.ImpactScope)
	}
	if req.RiskLevel != nil {
		existing.RiskLevel = string(*req.RiskLevel)
	}
	if req.PlannedStartDate != nil {
		existing.PlannedStartDate = req.PlannedStartDate
	}
	if req.PlannedEndDate != nil {
		existing.PlannedEndDate = req.PlannedEndDate
	}
	if req.ImplementationPlan != nil {
		existing.ImplementationPlan = *req.ImplementationPlan
	}
	if req.RollbackPlan != nil {
		existing.RollbackPlan = *req.RollbackPlan
	}
	if req.AffectedCIs != nil {
		existing.AffectedCIs = req.AffectedCIs
	}
	if req.RelatedTickets != nil {
		existing.RelatedTickets = req.RelatedTickets
	}

	res, err := h.svc.UpdateChange(c.Request.Context(), existing)
	if err != nil {
		common.InternalError(c, "更新变更失败: "+err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// SubmitApproval handles POST /api/v1/changes/:id/approvals
//
//	@Summary	提交变更审批记录
//	@Description	为变更添加审批意见；会触发审批链推进（待审批时自动置为 pending）
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	dto.CreateChangeApprovalRequest	true	"审批意见"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/approvals [post]
func (h *Handler) SubmitApproval(c *gin.Context) {
	changeID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.CreateChangeApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}
	req.ChangeID = changeID

	record := &ApprovalRecord{
		ChangeID:   req.ChangeID,
		ApproverID: req.ApproverID,
		Comment:    req.Comment,
	}

	res, err := h.svc.SubmitApproval(c.Request.Context(), record, tenantID)
	if err != nil {
		common.InternalError(c, "提交审批失败: "+err.Error())
		return
	}

	common.Success(c, res)
}

// SubmitChange handles POST /api/v1/changes/:id/submit
//
//	@Summary	提交变更进入审批
//	@Description	将 draft 状态的变更提交审批，创建审批链并把状态推进为 pending
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	dto.SubmitChangeRequest	false	"提交参数（可为空 body）"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	409	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/submit [post]
func (h *Handler) SubmitChange(c *gin.Context) {
	changeID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	var req dto.SubmitChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}

	res, err := h.svc.SubmitChange(c.Request.Context(), changeID, tenantID, userID, &req)
	if err != nil {
		common.InternalError(c, "提交变更失败: "+err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// GetStats handles GET /api/v1/changes/stats
//
//	@Summary	获取变更统计
//	@Description	返回当前租户各状态的变更数量汇总
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	common.Response{data=dto.ChangeStatsResponse}
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	res, err := h.svc.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取统计信息失败: "+err.Error())
		return
	}
	// Map domain stats -> DTO so the response shape stays governed by dto.ChangeStatsResponse
	// (Project rule: Controller must return DTO, never the domain struct directly.)
	common.Success(c, toStatsDTO(res))
}

// toStatsDTO maps the change.Stats domain struct to dto.ChangeStatsResponse.
func toStatsDTO(s *Stats) *dto.ChangeStatsResponse {
	if s == nil {
		return &dto.ChangeStatsResponse{}
	}
	return &dto.ChangeStatsResponse{
		Total:      s.Total,
		Pending:    s.Pending,
		Approved:   s.Approved,
		Scheduled:  s.Scheduled,
		InProgress: s.InProgress,
		Completed:  s.Completed,
		Failed:     s.Failed,
		RolledBack: s.RolledBack,
		Rejected:   s.Rejected,
		Cancelled:  s.Cancelled,
	}
}

// TransitionStatus handles status transition actions
// POST /api/v1/changes/:id/approve|reject|start|complete|rollback|cancel
//
//	@Summary	变更状态流转
//	@Description	按路径尾段执行状态流转：approve→approved、reject→rejected、schedule→scheduled、start→in_progress、complete→completed、close→closed、rollback→rolled_back、cancel→cancelled。审批类操作需要 change:approve 权限。状态机违规或 CAS 并发冲突返回 409。
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	object	false	"备注（{comment 或 reason}）"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	403	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	409	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/approve [post]
//	@Router	/api/v1/changes/{id}/reject [post]
//	@Router	/api/v1/changes/{id}/schedule [post]
//	@Router	/api/v1/changes/{id}/start [post]
//	@Router	/api/v1/changes/{id}/complete [post]
//	@Router	/api/v1/changes/{id}/close [post]
//	@Router	/api/v1/changes/{id}/rollback [post]
//	@Router	/api/v1/changes/{id}/cancel [post]
func (h *Handler) TransitionStatus(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	// Determine target status from the last path segment
	path := c.FullPath() // e.g. /api/v1/changes/:id/approve
	parts := strings.Split(path, "/")
	action := parts[len(parts)-1]
	statusMap := map[string]string{
		"approve":  "approved",
		"reject":   "rejected",
		"schedule": "scheduled",
		"start":    "in_progress",
		"complete": "completed",
		"close":    "closed",
		"rollback": "rolled_back",
		"cancel":   "cancelled",
	}
	targetStatus, ok := statusMap[action]
	if !ok {
		common.ParamError(c, "Unknown action: "+action)
		return
	}

	var body struct {
		Comment string `json:"comment"`
		Reason  string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	comment := strings.TrimSpace(body.Comment)
	if comment == "" {
		comment = strings.TrimSpace(body.Reason)
	}

	res, err := h.svc.TransitionStatus(c.Request.Context(), id, tenantID, userID, targetStatus, comment)
	if err != nil {
		// 按错误语义分流：业务规则拒绝与并发冲突不得再伪装成 500。
		// 否则客户端无法与真实故障区分，重试逻辑无从实现，告警也会被无谓污染。
		switch {
		case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrConcurrentModification):
			common.Conflict(c, err.Error(), nil)
		case errors.Is(err, ErrNotApprover):
			common.Fail(c, common.ForbiddenCode, err.Error())
		case errors.Is(err, ErrChangeNotFound):
			common.Fail(c, common.NotFoundCode, "变更不存在")
		default:
			common.InternalError(c, "状态转换失败: "+err.Error())
		}
		return
	}
	common.Success(c, toDTO(res))
}

// AssignChange handles POST /api/v1/changes/:id/assign
//
//	@Summary	分配变更处理人
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	object	true	"分配对象（{assigneeId: int, 必填}）"
//	@Success	200	{object}	common.Response{data=dto.ChangeResponse}
//	@Failure	400	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/assign [post]
func (h *Handler) AssignChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req struct {
		AssigneeID int `json:"assigneeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "assignee_id is required")
		return
	}

	existing, err := h.svc.GetChange(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Change not found")
		return
	}
	existing.AssigneeID = &req.AssigneeID
	res, err := h.svc.UpdateChange(c.Request.Context(), existing)
	if err != nil {
		common.InternalError(c, "分配变更失败: "+err.Error())
		return
	}
	common.Success(c, toDTO(res))
}

// GetApprovals handles GET /api/v1/changes/:id/approvals
//
//	@Summary	获取变更审批历史
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/approvals [get]
func (h *Handler) GetApprovals(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	history, err := h.svc.GetApprovalHistory(c.Request.Context(), id, tenantID)
	if err != nil {
		common.InternalError(c, "获取审批历史失败: "+err.Error())
		return
	}
	common.Success(c, history)
}

// DeleteChange handles DELETE /api/v1/changes/:id
//
//	@Summary	删除变更单
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id} [delete]
func (h *Handler) DeleteChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := h.svc.DeleteChange(c.Request.Context(), id, tenantID); err != nil {
		common.InternalError(c, "删除变更失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{"message": "deleted"})
}

// GetCalendar handles GET /api/v1/changes/calendar
//
//	@Summary	获取变更日历视图
//	@Description	按时间窗口返回变更排期，用于变更日历展示
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	startDate	query	string	true	"开始日期（YYYY-MM-DD）"
//	@Param	endDate	query	string	true	"结束日期（YYYY-MM-DD）"
//	@Param	status	query	string	false	"状态过滤"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/calendar [get]
func (h *Handler) GetCalendar(c *gin.Context) {
	var req dto.ChangeCalendarRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamError(c, "Invalid query parameters: "+err.Error())
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetCalendarView(c.Request.Context(), tenantID, req.StartDate, req.EndDate, req.Status)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, res)
}

// ==================== PIR (Post-Implementation Review) Handlers ====================

// CreatePIR handles POST /api/v1/changes/:id/pir
//
//	@Summary	创建实施后评审（PIR）
//	@Description	为已完成（completed）的变更创建实施后评审记录；同一变更仅允许一份 PIR，重复创建返回 409
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Param	request	body	dto.CreateChangePIRRequest	true	"PIR 内容"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	409	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/pir [post]
func (h *Handler) CreatePIR(c *gin.Context) {
	changeID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	var req dto.CreateChangePIRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}
	req.ChangeID = changeID

	pir, err := h.svc.CreatePIR(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "已存在") {
			common.Conflict(c, err.Error(), nil)
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, pir)
}

// GetPIR handles GET /api/v1/changes/:id/pir
//
//	@Summary	获取变更的 PIR
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"变更ID"
//	@Success	200	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/{id}/pir [get]
func (h *Handler) GetPIR(c *gin.Context) {
	changeID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	pir, err := h.svc.GetPIRByChange(c.Request.Context(), changeID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "无PIR记录") {
			common.NotFoundWithErr(c, err, "resource not found")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, pir)
}

// ListPIRs handles GET /api/v1/changes/pirs
//
//	@Summary	获取 PIR 列表
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	page	query	int	false	"页码（默认 1）"
//	@Param	pageSize	query	int	false	"每页数量（默认 10）"
//	@Param	result	query	string	false	"评审结论过滤"
//	@Success	200	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/pirs [get]
func (h *Handler) ListPIRs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	result := c.Query("result")

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	pirs, err := h.svc.ListPIRs(c.Request.Context(), tenantID, page, pageSize, result)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, pirs)
}

// UpdatePIR handles PUT /api/v1/changes/pir/:id
//
//	@Summary	更新 PIR
//	@Description	注意：路径参数 id 是 PIR 记录 ID，不是变更 ID
//	@Tags	变更管理
//	@Accept	json
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"PIR 记录ID"
//	@Param	request	body	dto.UpdateChangePIRRequest	true	"PIR 更新内容"
//	@Success	200	{object}	common.Response
//	@Failure	400	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/pir/{id} [put]
func (h *Handler) UpdatePIR(c *gin.Context) {
	pirID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateChangePIRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body: "+err.Error())
		return
	}

	pir, err := h.svc.UpdatePIR(c.Request.Context(), pirID, &req, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			common.NotFoundWithErr(c, err, "resource not found")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, pir)
}

// DeletePIR handles DELETE /api/v1/changes/pir/:id
//
//	@Summary	删除 PIR
//	@Description	注意：路径参数 id 是 PIR 记录 ID，不是变更 ID
//	@Tags	变更管理
//	@Produce	json
//	@Security	BearerAuth
//	@Param	id	path	int	true	"PIR 记录ID"
//	@Success	200	{object}	common.Response
//	@Failure	404	{object}	common.Response
//	@Failure	500	{object}	common.Response
//	@Router	/api/v1/changes/pir/{id} [delete]
func (h *Handler) DeletePIR(c *gin.Context) {
	pirID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := h.svc.DeletePIR(c.Request.Context(), pirID, tenantID); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			common.NotFoundWithErr(c, err, "resource not found")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "PIR deleted"})
}
