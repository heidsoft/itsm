package sla

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// Map domain definition to DTO
func toSLADefinitionDTO(s *SLADefinition) *dto.SLADefinitionResponse {
	if s == nil {
		return nil
	}
	return &dto.SLADefinitionResponse{
		ID:              s.ID,
		Name:            s.Name,
		Description:     s.Description,
		ServiceType:     s.ServiceType,
		Priority:        s.Priority,
		ResponseTime:    s.ResponseTime,
		ResolutionTime:  s.ResolutionTime,
		BusinessHours:   s.BusinessHours,
		EscalationRules: s.EscalationRules,
		Conditions:      s.Conditions,
		IsActive:        s.IsActive,
		TenantID:        s.TenantID,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// CreateSLADefinition handles POST /api/v1/sla/definitions
func (h *Handler) CreateSLADefinition(c *gin.Context) {
	var req dto.CreateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	tenantIDVal := c.GetInt("tenant_id")
	def := &SLADefinition{
		Name:            req.Name,
		Description:     req.Description,
		ServiceType:     req.ServiceType,
		Priority:        req.Priority,
		ResponseTime:    req.ResponseTime,
		ResolutionTime:  req.ResolutionTime,
		BusinessHours:   req.BusinessHours,
		EscalationRules: req.EscalationRules,
		Conditions:      req.Conditions,
		IsActive:        req.IsActive,
		TenantID:        tenantIDVal,
	}

	res, err := h.svc.CreateDefinition(c.Request.Context(), def)
	if err != nil {
		common.InternalError(c, "创建SLA定义失败: "+err.Error())
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// GetSLADefinition handles GET /api/v1/sla/definitions/:id
func (h *Handler) GetSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	res, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.NotFound(c, "SLA Definition not found")
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// ListSLADefinitions handles GET /api/v1/sla/definitions
func (h *Handler) ListSLADefinitions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size := queryIntParam(c, "pageSize", "size", 10)
	tenantIDVal := c.GetInt("tenant_id")

	list, total, err := h.svc.ListDefinitions(c.Request.Context(), tenantIDVal, page, size)
	if err != nil {
		common.InternalError(c, "查询SLA定义列表失败: "+err.Error())
		return
	}

	var dtos []*dto.SLADefinitionResponse
	for _, item := range list {
		dtos = append(dtos, toSLADefinitionDTO(item))
	}

	common.Success(c, gin.H{
		"items":    dtos,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// UpdateSLADefinition handles PUT /api/v1/sla/definitions/:id
func (h *Handler) UpdateSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	var req dto.UpdateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	existing, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.NotFound(c, "SLA Definition not found")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.ServiceType != nil {
		existing.ServiceType = *req.ServiceType
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.ResponseTime != nil {
		existing.ResponseTime = *req.ResponseTime
	}
	if req.ResolutionTime != nil {
		existing.ResolutionTime = *req.ResolutionTime
	}
	if req.BusinessHours != nil {
		existing.BusinessHours = req.BusinessHours
	}
	if req.EscalationRules != nil {
		existing.EscalationRules = req.EscalationRules
	}
	if req.Conditions != nil {
		existing.Conditions = req.Conditions
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	res, err := h.svc.UpdateDefinition(c.Request.Context(), existing)
	if err != nil {
		common.InternalError(c, "更新SLA定义失败: "+err.Error())
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// DeleteSLADefinition handles DELETE /api/v1/sla/definitions/:id
func (h *Handler) DeleteSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	if err := h.svc.DeleteDefinition(c.Request.Context(), id, tenantIDVal); err != nil {
		common.InternalError(c, "删除SLA定义失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// CreateAlertRule handles POST /api/v1/sla/alert-rules
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var req dto.CreateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	tenantIDVal := c.GetInt("tenant_id")
	rule := &SLAAlertRule{
		SLADefinitionID:      req.SLADefinitionID,
		Name:                 req.Name,
		ThresholdPercentage:  req.ThresholdPercentage,
		AlertLevel:           req.AlertLevel,
		NotificationChannels: req.NotificationChannels,
		IsActive:             req.IsActive,
		TenantID:             tenantIDVal,
	}
	res, err := h.svc.CreateAlertRule(c.Request.Context(), rule)
	if err != nil {
		common.InternalError(c, "创建SLA告警规则失败: "+err.Error())
		return
	}
	common.Success(c, res)
}

// ListAlertRules handles GET /api/v1/sla/alert-rules
func (h *Handler) ListAlertRules(c *gin.Context) {
	tenantIDVal := c.GetInt("tenant_id")
	filters := make(map[string]interface{})
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		id, _ := strconv.Atoi(slaID)
		filters["sla_definition_id"] = id
	}
	res, err := h.svc.ListAlertRules(c.Request.Context(), tenantIDVal, filters)
	if err != nil {
		common.InternalError(c, "查询SLA告警规则列表失败: "+err.Error())
		return
	}
	common.Success(c, res)
}

// GetAlertRule handles GET /api/v1/sla/alert-rules/:id
func (h *Handler) GetAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	res, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.NotFound(c, "SLA Alert Rule not found")
		return
	}
	common.Success(c, res)
}

// UpdateAlertRule handles PUT /api/v1/sla/alert-rules/:id
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	var req dto.UpdateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	existing, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.NotFound(c, "SLA Alert Rule not found")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.ThresholdPercentage != nil {
		existing.ThresholdPercentage = *req.ThresholdPercentage
	}
	if req.AlertLevel != nil {
		existing.AlertLevel = *req.AlertLevel
	}
	if req.NotificationChannels != nil {
		existing.NotificationChannels = *req.NotificationChannels
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	res, err := h.svc.UpdateAlertRule(c.Request.Context(), existing)
	if err != nil {
		common.InternalError(c, "更新SLA告警规则失败: "+err.Error())
		return
	}
	common.Success(c, res)
}

// DeleteAlertRule handles DELETE /api/v1/sla/alert-rules/:id
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	if err := h.svc.DeleteAlertRule(c.Request.Context(), id, tenantIDVal); err != nil {
		common.InternalError(c, "删除SLA告警规则失败: "+err.Error())
		return
	}
	common.Success(c, nil)
}

// GetSLAMetrics handles GET /api/v1/sla/metrics
func (h *Handler) GetSLAMetrics(c *gin.Context) {
	tenantIDVal := c.GetInt("tenant_id")
	filters := make(map[string]interface{})
	// 查询参数契约为 camelCase，与前端 API client 保持逐字段一致。
	if slaID := c.Query("slaDefinitionId"); slaID != "" {
		id, _ := strconv.Atoi(slaID)
		filters["sla_definition_id"] = id
	}
	if metricType := c.Query("metricType"); metricType != "" {
		filters["metric_type"] = metricType
	}

	res, err := h.svc.GetSLAMetrics(c.Request.Context(), tenantIDVal, filters)
	if err != nil {
		failSLAQuery(c, err, "获取SLA指标失败")
		return
	}
	common.Success(c, gin.H{
		"metrics": res,
		"count":   len(res),
	})
}

// queryParam 优先读 camelCase（前端 HTTP 契约统一使用 camelCase），
// 再回退 snake_case，避免同名双轨参数。
func queryParam(c *gin.Context, camel, snake string) string {
	if v := c.Query(camel); v != "" {
		return v
	}
	return c.Query(snake)
}

// queryIntParam 优先读 camelCase 分页字段 pageSize，兼容存量 size 调用方。
func queryIntParam(c *gin.Context, camel, snake string, fallback int) int {
	if v := queryParam(c, camel, snake); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// GetSLAViolations handles GET /api/v1/sla/violations
func (h *Handler) GetSLAViolations(c *gin.Context) {
	tenantIDVal := c.GetInt("tenant_id")
	filters := make(map[string]interface{})
	if isResolved := queryParam(c, "isResolved", "is_resolved"); isResolved != "" {
		if val, err := strconv.ParseBool(isResolved); err == nil {
			filters["is_resolved"] = val
		}
	}
	if severity := queryParam(c, "severity", "severity"); severity != "" {
		filters["severity"] = severity
	}
	if violationType := queryParam(c, "violationType", "violation_type"); violationType != "" {
		filters["violation_type"] = violationType
	}
	if slaID := queryParam(c, "slaDefinitionId", "sla_definition_id"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size := queryIntParam(c, "pageSize", "size", 20)

	res, total, err := h.svc.GetSLAViolations(c.Request.Context(), tenantIDVal, page, size, filters)
	if err != nil {
		common.InternalError(c, "查询SLA违规记录失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{
		"items":    res,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// UpdateViolationStatus handles PUT /api/v1/sla/violations/:id
func (h *Handler) UpdateViolationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal := c.GetInt("tenant_id")

	var req struct {
		IsResolved bool   `json:"isResolved"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	res, err := h.svc.UpdateSLAViolationStatus(c.Request.Context(), id, req.IsResolved, req.Notes, tenantIDVal)
	if err != nil {
		common.InternalError(c, "更新SLA违规状态失败: "+err.Error())
		return
	}
	common.Success(c, res)
}

// GetSLAMonitoring handles POST /api/v1/sla/monitoring（兼容别名 /sla/monitor）
//
// 请求体中的 startTime/endTime 必须是 RFC3339；缺省时由 Service 套用默认 30 天窗口。
// 历史实现接受 "30d"/"now" 这类伪值但仓储层从未使用它们，导致大屏指标与实际窗口无关。
// 无请求体（或空体）是合法查询，不能当成参数错误。
func (h *Handler) GetSLAMonitoring(c *gin.Context) {
	var req dto.SLAMonitoringRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	tenantIDVal := c.GetInt("tenant_id")

	start, err := parseSLATimeParam(req.StartTime)
	if err != nil {
		common.ParamError(c, "startTime 格式非法，请使用 RFC3339，例如 2026-01-01T00:00:00Z")
		return
	}
	end, err := parseSLATimeParam(req.EndTime)
	if err != nil {
		common.ParamError(c, "endTime 格式非法，请使用 RFC3339，例如 2026-01-31T23:59:59Z")
		return
	}

	res, err := h.svc.GetSLAMonitoring(c.Request.Context(), tenantIDVal, start, end)
	if err != nil {
		failSLAQuery(c, err, "获取SLA监控数据失败")
		return
	}
	common.Success(c, res)
}

// GetSLAPerformance handles GET /api/v1/sla/performance
//
// 监控大屏的「各服务类型绩效」「各优先级绩效」表格数据源。
// dimension 只接受 serviceType / priority；serviceType、priority 为可选预过滤，
// 它们限制统计种群而不是对已分页的结果集做二次筛选。
// 本路由是新增接口，查询参数只有一套 camelCase 契约，不再兼容 snake_case 别名。
func (h *Handler) GetSLAPerformance(c *gin.Context) {
	tenantIDVal := c.GetInt("tenant_id")

	dimension := strings.TrimSpace(c.Query("dimension"))
	if dimension == "" {
		dimension = SLADimensionServiceType
	}

	start, err := parseSLATimeParam(c.Query("startDate"))
	if err != nil {
		common.ParamError(c, "startDate 格式非法，请使用 RFC3339，例如 2026-01-01T00:00:00Z")
		return
	}
	end, err := parseSLATimeParam(c.Query("endDate"))
	if err != nil {
		common.ParamError(c, "endDate 格式非法，请使用 RFC3339，例如 2026-01-31T23:59:59Z")
		return
	}
	page, err := strconv.Atoi(defaultString(c.Query("page"), "1"))
	if err != nil || page < 1 {
		common.ParamError(c, "page 必须为正整数")
		return
	}
	pageSize, err := strconv.Atoi(defaultString(c.Query("pageSize"), "20"))
	if err != nil || pageSize < 1 {
		common.ParamError(c, "pageSize 必须为正整数")
		return
	}

	res, err := h.svc.ListSLAPerformance(c.Request.Context(), tenantIDVal, SLAPerformanceQuery{
		Dimension:   dimension,
		Start:       start,
		End:         end,
		ServiceType: strings.TrimSpace(c.Query("serviceType")),
		Priority:    strings.TrimSpace(c.Query("priority")),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		failSLAQuery(c, err, "获取SLA绩效数据失败")
		return
	}

	totalPages := 0
	if res.Total > 0 {
		totalPages = (res.Total + res.PageSize - 1) / res.PageSize
	}
	common.Success(c, gin.H{
		"items":      res.Items,
		"total":      res.Total,
		"page":       res.Page,
		"pageSize":   res.PageSize,
		"totalPages": totalPages,
		"dimension":  dimension,
		"truncated":  res.Truncated,
	})
}

// bindOptionalJSON 绑定可选的 JSON 请求体：无体/空体返回 nil（使用全默认值），
// 只有真正格式错误或字段类型不合法才返回错误。
func bindOptionalJSON(c *gin.Context, target any) error {
	if c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}
	if err := c.ShouldBindJSON(target); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// parseSLATimeParam 解析 RFC3339 时间参数；空串返回零值，由 Service 套用默认窗口。
func parseSLATimeParam(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// failSLAQuery 把 service 层的稳定错误映射为 HTTP status 与业务 code，
// 并把原始错误只写日志、不外泄给客户端。
func failSLAQuery(c *gin.Context, err error, publicMsg string) {
	switch {
	case errors.Is(err, ErrTenantRequired):
		common.AuthFailed(c, "缺少租户上下文，拒绝访问")
	case errors.Is(err, ErrInvalidWindow):
		common.ParamError(c, "endTime 必须晚于 startTime")
	case errors.Is(err, ErrInvalidDimension):
		common.ParamError(c, "dimension 只支持 serviceType 或 priority")
	default:
		common.FailWithErr(c, err, publicMsg)
	}
}

// CheckSLACompliance handles POST /api/v1/sla/check-compliance/:ticketId
// P1-07 修复：直接把 Service 返回的 *SLAComplianceResult（含 actual_response_minutes）
// 透传给前端，而不是仅返回占位 message。
func (h *Handler) CheckSLACompliance(c *gin.Context) {
	ticketIDStr := c.Param("ticketId")
	ticketID, _ := strconv.Atoi(ticketIDStr)
	tenantIDVal := c.GetInt("tenant_id")

	res, err := h.svc.CheckSLACompliance(c.Request.Context(), ticketID, tenantIDVal)
	if err != nil {
		common.InternalError(c, "检查SLA合规性失败: "+err.Error())
		return
	}
	common.Success(c, res)
}

// GetAlertHistory handles GET /api/v1/sla/alert-history
func (h *Handler) GetAlertHistory(c *gin.Context) {
	tenantIDVal := c.GetInt("tenant_id")
	filters := make(map[string]interface{})
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}
	if alertRuleID := c.Query("alert_rule_id"); alertRuleID != "" {
		if id, _ := strconv.Atoi(alertRuleID); id > 0 {
			filters["alert_rule_id"] = id
		}
	}
	if alertLevel := c.Query("alert_level"); alertLevel != "" {
		filters["alert_level"] = alertLevel
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	res, total, err := h.svc.GetAlertHistory(c.Request.Context(), tenantIDVal, page, size, filters)
	if err != nil {
		common.InternalError(c, "查询告警历史失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{
		"items":    res,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// GetSLAStats handles GET /api/v1/sla/stats
func (h *Handler) GetSLAStats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	stats, err := h.svc.GetSLAStats(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取SLA统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}

// GetSLAComplianceReport handles GET /api/v1/sla/compliance-report
func (h *Handler) GetSLAComplianceReport(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	// Parse query parameters.
	// camelCase 是查询参数的契约写法，snake_case 仅兼容存量调用方。
	startDateStr := queryParam(c, "startDate", "start_date")
	endDateStr := queryParam(c, "endDate", "end_date")

	if startDateStr == "" || endDateStr == "" {
		common.ParamError(c, "startDate and endDate are required")
		return
	}

	// Parse ISO 8601 timestamps
	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		common.ParamError(c, "invalid startDate format, use ISO 8601 (e.g., 2024-01-01T00:00:00Z)")
		return
	}
	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		common.ParamError(c, "invalid endDate format, use ISO 8601 (e.g., 2024-01-31T23:59:59Z)")
		return
	}

	report, err := h.svc.GetComplianceReport(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		common.InternalError(c, "生成SLA合规报告失败: "+err.Error())
		return
	}

	common.Success(c, report)
}
