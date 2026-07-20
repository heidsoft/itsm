package change

import (
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
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
	return res
}

// CreateChange handles POST /api/v1/changes
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
	}

	res, err := h.svc.CreateChange(c.Request.Context(), changeEntity)
	if err != nil {
		common.InternalError(c, "创建变更失败: "+err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// GetChange handles GET /api/v1/changes/:id
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

// GetCMDBImpactSummary handles GET /api/v1/changes/:id/cmdb-impact
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

	list, total, err := h.svc.ListChanges(c.Request.Context(), tenantID, page, pageSize, status, search, riskLevel)
	if err != nil {
		common.InternalError(c, "查询变更列表失败: "+err.Error())
		return
	}

	var dtos []dto.ChangeResponse
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
	if err := c.ShouldBindJSON(&req); err != nil {
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
		"start":    "in_progress",
		"complete": "completed",
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

	res, err := h.svc.TransitionStatus(c.Request.Context(), id, tenantID, userID, targetStatus)
	if err != nil {
		common.InternalError(c, "状态转换失败: "+err.Error())
		return
	}
	common.Success(c, toDTO(res))
}

// AssignChange handles POST /api/v1/changes/:id/assign
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
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, res)
}

// ==================== PIR (Post-Implementation Review) Handlers ====================

// CreatePIR handles POST /api/v1/changes/:id/pir
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
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, pir)
}

// GetPIR handles GET /api/v1/changes/:id/pir
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
			common.NotFound(c, err.Error())
			return
		}
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, pir)
}

// ListPIRs handles GET /api/v1/changes/pirs
func (h *Handler) ListPIRs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	result := c.Query("result")

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	pirs, err := h.svc.ListPIRs(c.Request.Context(), tenantID, page, pageSize, result)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, pirs)
}

// UpdatePIR handles PUT /api/v1/changes/pir/:id
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
			common.NotFound(c, err.Error())
			return
		}
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, pir)
}

// DeletePIR handles DELETE /api/v1/changes/pir/:id
func (h *Handler) DeletePIR(c *gin.Context) {
	pirID, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := h.svc.DeletePIR(c.Request.Context(), pirID, tenantID); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			common.NotFound(c, err.Error())
			return
		}
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "PIR deleted"})
}
