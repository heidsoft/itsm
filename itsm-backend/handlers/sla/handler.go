package sla

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
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
		ID:               s.ID,
		Name:             s.Name,
		Description:      s.Description,
		ServiceType:      s.ServiceType,
		Priority:         s.Priority,
		ResponseTime:     s.ResponseTime,
		ResolutionTime:   s.ResolutionTime,
		BusinessHours:    dto.ProjectSLABusinessHours(s.BusinessHours),
		EscalationLevels: dto.ProjectSLAEscalationLevels(s.EscalationRules),
		Conditions:       s.Conditions,
		IsActive:         s.IsActive,
		TenantID:         s.TenantID,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

// CreateSLADefinition handles POST /api/v1/sla/definitions
// @Summary 创建SLA定义
// @Description 新建一条SLA定义（响应/解决时限、服务类型、优先级、营业时间与升级规则）。需要 sla:write 权限；租户上下文缺失返回 401。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param body body dto.CreateSLADefinitionRequest true "SLA定义"
// @Success 200 {object} common.Response{data=dto.SLADefinitionResponse}
// @Failure 400 {object} common.Response "参数错误"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/definitions [post]
func (h *Handler) CreateSLADefinition(c *gin.Context) {
	var req dto.CreateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
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
		common.RespondError(c, err, "创建SLA定义失败")
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// GetSLADefinition handles GET /api/v1/sla/definitions/:id
// @Summary 获取SLA定义详情
// @Description 按 ID 查询当前租户的SLA定义。仅 ent.NotFoundError 映射为 404；DB 故障等后端错误返回 500，不伪装成「定义不存在」。兼容旧路径 /sla/policies/:id。
// @Tags SLA管理
// @Produce json
// @Param id path int true "SLA定义ID"
// @Success 200 {object} common.Response{data=dto.SLADefinitionResponse}
// @Failure 400 {object} common.Response "无效ID"
// @Failure 404 {object} common.Response "定义不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/definitions/{id} [get]
func (h *Handler) GetSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	res, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		// 真·不存在（ent.NotFoundError）→ 404；DB 故障等其余错误 → 500，
		// 不能无条件映射成 404 把后端故障伪装成「定义不存在」。
		common.RespondError(c, err, "查询SLA定义失败")
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// ListSLADefinitions handles GET /api/v1/sla/definitions
// @Summary 列出SLA定义
// @Description 分页列出当前租户的SLA定义，页长由平台统一夹紧。兼容 /sla 与 /sla/policies。
// @Tags SLA管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response "data 含 items/total/page/pageSize/totalPages"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/definitions [get]
func (h *Handler) ListSLADefinitions(c *gin.Context) {
	pg := common.GetPaginationFromQuery(c)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	list, total, err := h.svc.ListDefinitions(c.Request.Context(), tenantIDVal, pg.Page, pg.PageSize)
	if err != nil {
		common.RespondError(c, err, "查询SLA定义列表失败")
		return
	}

	dtos := make([]*dto.SLADefinitionResponse, 0, len(list))
	for _, item := range list {
		dtos = append(dtos, toSLADefinitionDTO(item))
	}

	common.SuccessWithPagination(c, dtos, pg.Page, pg.PageSize, int64(total))
}

// UpdateSLADefinition handles PUT /api/v1/sla/definitions/:id
// @Summary 更新SLA定义
// @Description 按 ID 局部更新SLA定义（仅提交过的字段生效）。前置读取仅 NotFoundError→404，DB 故障→500。兼容旧路径 /sla/policies/:id。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param id path int true "SLA定义ID"
// @Param body body dto.UpdateSLADefinitionRequest true "更新字段"
// @Success 200 {object} common.Response{data=dto.SLADefinitionResponse}
// @Failure 400 {object} common.Response "参数错误"
// @Failure 404 {object} common.Response "定义不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/definitions/{id} [put]
func (h *Handler) UpdateSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	var req dto.UpdateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	existing, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.RespondError(c, err, "查询SLA定义失败")
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
		common.RespondError(c, err, "更新SLA定义失败")
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// DeleteSLADefinition handles DELETE /api/v1/sla/definitions/:id
// @Summary 删除SLA定义
// @Description 按 ID 删除当前租户的SLA定义；不存在或跨租户返回 404，DB 故障返回 500。
// @Tags SLA管理
// @Produce json
// @Param id path int true "SLA定义ID"
// @Success 200 {object} common.Response
// @Failure 404 {object} common.Response "定义不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/definitions/{id} [delete]
func (h *Handler) DeleteSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteDefinition(c.Request.Context(), id, tenantIDVal); err != nil {
		common.RespondError(c, err, "删除SLA定义失败")
		return
	}

	common.Success(c, nil)
}

// CreateAlertRule handles POST /api/v1/sla/alert-rules
// @Summary 创建SLA告警规则
// @Description 为当前租户的SLA定义创建告警规则（阈值百分比、告警级别、通知渠道）。引用的SLA定义跨租户时拒绝。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param body body dto.CreateSLAAlertRuleRequest true "告警规则"
// @Success 200 {object} common.Response{data=dto.SLAAlertRuleResponse}
// @Failure 400 {object} common.Response "参数错误或跨租户引用"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-rules [post]
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var req dto.CreateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
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
		common.RespondError(c, err, "创建SLA告警规则失败")
		return
	}
	common.Success(c, res)
}

// ListAlertRules handles GET /api/v1/sla/alert-rules
// @Summary 列出SLA告警规则
// @Description 列出当前租户的SLA告警规则，可按 sla_definition_id 过滤。
// @Tags SLA管理
// @Produce json
// @Success 200 {object} common.Response{data=[]dto.SLAAlertRuleResponse}
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-rules [get]
func (h *Handler) ListAlertRules(c *gin.Context) {
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	filters := make(map[string]interface{})
	// 查询参数契约为 camelCase，与前端 API client 保持逐字段一致。
	if slaID := c.Query("slaDefinitionId"); slaID != "" {
		id, _ := strconv.Atoi(slaID)
		filters["sla_definition_id"] = id
	}
	res, err := h.svc.ListAlertRules(c.Request.Context(), tenantIDVal, filters)
	if err != nil {
		common.RespondError(c, err, "查询SLA告警规则列表失败")
		return
	}
	common.Success(c, res)
}

// GetAlertRule handles GET /api/v1/sla/alert-rules/:id
// @Summary 获取SLA告警规则
// @Description 按 ID 查询告警规则。仅 NotFoundError→404；DB 故障→500。
// @Tags SLA管理
// @Produce json
// @Param id path int true "告警规则ID"
// @Success 200 {object} common.Response{data=dto.SLAAlertRuleResponse}
// @Failure 404 {object} common.Response "规则不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-rules/{id} [get]
func (h *Handler) GetAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	res, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		// 真·不存在 → 404；DB 故障 → 500，不无条件伪装成「规则不存在」。
		common.RespondError(c, err, "查询SLA告警规则失败")
		return
	}
	common.Success(c, res)
}

// UpdateAlertRule handles PUT /api/v1/sla/alert-rules/:id
// @Summary 更新SLA告警规则
// @Description 局部更新告警规则。前置读取跨租户/不存在→404，DB 故障→500。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param id path int true "告警规则ID"
// @Param body body dto.UpdateSLAAlertRuleRequest true "更新字段"
// @Success 200 {object} common.Response{data=dto.SLAAlertRuleResponse}
// @Failure 404 {object} common.Response "规则不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-rules/{id} [put]
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	var req dto.UpdateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	existing, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal)
	if err != nil {
		common.RespondError(c, err, "查询SLA告警规则失败")
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
		common.RespondError(c, err, "更新SLA告警规则失败")
		return
	}
	common.Success(c, res)
}

// DeleteAlertRule handles DELETE /api/v1/sla/alert-rules/:id
// @Summary 删除SLA告警规则
// @Description 按 ID 删除当前租户的告警规则。
// @Tags SLA管理
// @Produce json
// @Param id path int true "告警规则ID"
// @Success 200 {object} common.Response
// @Failure 404 {object} common.Response "规则不存在或跨租户"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-rules/{id} [delete]
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteAlertRule(c.Request.Context(), id, tenantIDVal); err != nil {
		common.RespondError(c, err, "删除SLA告警规则失败")
		return
	}
	common.Success(c, nil)
}

// GetSLAMetrics handles GET /api/v1/sla/metrics
// @Summary 获取SLA指标
// @Description sla_metrics 表当前无采集写入方，读它只会返回空列表造成「已采集且无异常」的假象，故本端点显式返回 501 声明能力边界。
// @Tags SLA管理
// @Produce json
// @Failure 501 {object} common.Response "指标采集未接入"
// @Security BearerAuth
// @Router /api/v1/sla/metrics [get]
//
// sla_metrics 表没有写入方（无指标采集任务），读它只会返回空列表。空成功会让
// 调用方误以为「已采集且无异常」，因此按已声明边界返回 501。
func (h *Handler) GetSLAMetrics(c *gin.Context) {
	if _, ok := handlerctx.ResolveTenantID(c); !ok {
		return
	}
	common.Fail(c, common.NotImplementedCode,
		"SLA 指标采集尚未实现：当前版本不写入 SLA 指标样本，请改用 GET /api/v1/sla/performance 或 POST /api/v1/sla/monitoring")
}

// GetSLAViolations handles GET /api/v1/sla/violations
// @Summary 列出SLA违规记录
// @Description 分页列出当前租户的SLA违规，支持 resolved 等状态过滤；页长由平台统一夹紧。
// @Tags SLA管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response "data 为标准分页信封"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/violations [get]
func (h *Handler) GetSLAViolations(c *gin.Context) {
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	filters := make(map[string]interface{})
	if isResolved := c.Query("isResolved"); isResolved != "" {
		if val, err := strconv.ParseBool(isResolved); err == nil {
			filters["is_resolved"] = val
		}
	}
	if severity := c.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if violationType := c.Query("violationType"); violationType != "" {
		filters["violation_type"] = violationType
	}
	if slaID := c.Query("slaDefinitionId"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}

	pg := common.GetPaginationFromQuery(c)

	res, total, err := h.svc.GetSLAViolations(c.Request.Context(), tenantIDVal, pg.Page, pg.PageSize, filters)
	if err != nil {
		common.RespondError(c, err, "查询SLA违规记录失败")
		return
	}
	if res == nil {
		res = []*SLAViolation{}
	}
	common.SuccessWithPagination(c, res, pg.Page, pg.PageSize, int64(total))
}

// UpdateViolationStatus handles PUT /api/v1/sla/violations/:id
// @Summary 处理SLA违规
// @Description 将违规标记为已/未处理并记录处理说明；跨租户不可写，DB 故障返回 500。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param id path int true "违规ID"
// @Param body body dto.UpdateSLAViolationRequest true "处理状态与说明"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/violations/{id} [put]
func (h *Handler) UpdateViolationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	var req struct {
		IsResolved bool   `json:"isResolved"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	res, err := h.svc.UpdateSLAViolationStatus(c.Request.Context(), id, req.IsResolved, req.Notes, tenantIDVal)
	if err != nil {
		common.RespondError(c, err, "更新SLA违规状态失败")
		return
	}
	common.Success(c, res)
}

// GetSLAMonitoring handles POST /api/v1/sla/monitoring（兼容别名 /sla/monitor）
// @Summary 获取SLA监控大屏指标
// @Description 按强制时间窗口（startTime/endTime 为 RFC3339，缺省由服务层套用默认 30 天）返回监控指标快照。无请求体合法；无样本时如实报告 0，不做假成功。
// @Tags SLA管理
// @Accept json
// @Produce json
// @Param body body dto.SLAMonitoringRequest false "统计窗口（空体用默认）"
// @Success 200 {object} common.Response{data=dto.SLAMonitoringDashboard}
// @Failure 400 {object} common.Response "时间格式非法"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/monitoring [post]
//
// 请求体中的 startTime/endTime 必须是 RFC3339；缺省时由 Service 套用默认 30 天窗口。
// 历史实现接受 "30d"/"now" 这类伪值但仓储层从未使用它们，导致大屏指标与实际窗口无关。
// 无请求体（或空体）是合法查询，不能当成参数错误。
func (h *Handler) GetSLAMonitoring(c *gin.Context) {
	var req dto.SLAMonitoringRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

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
// @Summary SLA绩效分维度聚合
// @Description 按 serviceType / priority 聚合窗口内绩效，供监控大屏表格使用；serviceType、priority 为可选预过滤（限制统计种群，非结果二次筛选）。仅 camelCase 契约。
// @Tags SLA管理
// @Produce json
// @Param dimension query string true "聚合维度：serviceType 或 priority"
// @Param serviceType query string false "服务类型预过滤"
// @Param priority query string false "优先级预过滤"
// @Success 200 {object} common.Response "data 含逐维绩效行与命中扫描上限标记"
// @Failure 400 {object} common.Response "维度非法"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/performance [get]
//
// 监控大屏的「各服务类型绩效」「各优先级绩效」表格数据源。
// dimension 只接受 serviceType / priority；serviceType、priority 为可选预过滤，
// 它们限制统计种群而不是对已分页的结果集做二次筛选。
// 本路由是新增接口，查询参数只有一套 camelCase 契约，不再兼容 snake_case 别名。
func (h *Handler) GetSLAPerformance(c *gin.Context) {
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	dimension := strings.TrimSpace(c.Query("dimension"))
	if dimension == "" {
		dimension = SLADimensionServiceType
	}

	start, err := parseSLADateFilter(c.Query("startDate"), false)
	if err != nil {
		common.ParamError(c, "startDate 格式非法，请使用 YYYY-MM-DD 或 RFC3339")
		return
	}
	end, err := parseSLADateFilter(c.Query("endDate"), true)
	if err != nil {
		common.ParamError(c, "endDate 格式非法，请使用 YYYY-MM-DD 或 RFC3339")
		return
	}
	pg := common.GetPaginationFromQuery(c)

	res, err := h.svc.ListSLAPerformance(c.Request.Context(), tenantIDVal, SLAPerformanceQuery{
		Dimension:   dimension,
		Start:       start,
		End:         end,
		ServiceType: strings.TrimSpace(c.Query("serviceType")),
		Priority:    strings.TrimSpace(c.Query("priority")),
		Page:        pg.Page,
		PageSize:    pg.PageSize,
	})
	if err != nil {
		failSLAQuery(c, err, "获取SLA绩效数据失败")
		return
	}

	totalPages := 0
	if res.Total > 0 {
		totalPages = (res.Total + pg.PageSize - 1) / pg.PageSize
	}
	common.Success(c, gin.H{
		"items":      res.Items,
		"total":      res.Total,
		"page":       pg.Page,
		"pageSize":   pg.PageSize,
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

// parseSLADateFilter 解析查询参数的日期过滤，接受 RFC3339 或裸日期 YYYY-MM-DD。
// 裸日期按 UTC 解释：start 取当日 00:00:00，end 取当日 23:59:59.999999999，
// 与 /changes/calendar 的日期过滤口径一致；过去只收 RFC3339 会把同一语义的
// 两个过滤端点拆成两套 wire 格式。
func parseSLADateFilter(value string, endOfDay bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	day, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		day = day.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return day.UTC(), nil
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
// @Summary 检查单个工单的SLA合规性
// @Description 基于工单首次响应/解决时间与SLA截止时间判定合规，返回含 actual_response_minutes 的完整结果；判定出的违规会落库，合规则不写违规。
// @Tags SLA管理
// @Produce json
// @Param ticketId path int true "工单ID"
// @Success 200 {object} common.Response "data 为 SLAComplianceResult"
// @Failure 400 {object} common.Response "无效工单ID"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/check-compliance/{ticketId} [post]
// 直接把 Service 返回的 *SLAComplianceResult（含 actual_response_minutes）
// 透传给前端，而不是仅返回占位 message。
func (h *Handler) CheckSLACompliance(c *gin.Context) {
	ticketIDStr := c.Param("ticketId")
	ticketID, _ := strconv.Atoi(ticketIDStr)
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	res, err := h.svc.CheckSLACompliance(c.Request.Context(), ticketID, tenantIDVal)
	if err != nil {
		common.RespondError(c, err, "检查SLA合规性失败")
		return
	}
	common.Success(c, res)
}

// GetAlertHistory handles GET /api/v1/sla/alert-history
// @Summary 列出SLA告警历史
// @Description 分页列出告警历史，支持过滤；页长由平台统一夹紧（坏值回落默认而非整表读取）。
// @Tags SLA管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response{data=[]dto.SLAAlertHistoryResponse}
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/alert-history [get]
func (h *Handler) GetAlertHistory(c *gin.Context) {
	tenantIDVal, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	// filters 的 key 是仓储层/数据库列名（snake_case），不是 HTTP 契约；
	// 查询参数按 camelCase 读取。
	filters := make(map[string]interface{})
	if slaID := c.Query("slaDefinitionId"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}
	if alertRuleID := c.Query("alertRuleId"); alertRuleID != "" {
		if id, _ := strconv.Atoi(alertRuleID); id > 0 {
			filters["alert_rule_id"] = id
		}
	}
	if alertLevel := c.Query("alertLevel"); alertLevel != "" {
		filters["alert_level"] = alertLevel
	}

	pg := common.GetPaginationFromQuery(c)

	res, total, err := h.svc.GetAlertHistory(c.Request.Context(), tenantIDVal, pg.Page, pg.PageSize, filters)
	if err != nil {
		common.RespondError(c, err, "查询告警历史失败")
		return
	}
	if res == nil {
		res = []*SLAAlertHistory{}
	}
	common.SuccessWithPagination(c, res, pg.Page, pg.PageSize, int64(total))
}

// GetSLAStats handles GET /api/v1/sla/stats
// @Summary 获取SLA统计
// @Description 汇总当前租户的SLA定义、违规与合规统计。工单统计查询失败时整体报错，不以 0 顶替；样本为 0 时如实报告 0。
// @Tags SLA管理
// @Produce json
// @Success 200 {object} common.Response{data=dto.SLAStatsResponse}
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 500 {object} common.Response "统计查询失败"
// @Security BearerAuth
// @Router /api/v1/sla/stats [get]
func (h *Handler) GetSLAStats(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	stats, err := h.svc.GetSLAStats(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "获取SLA统计失败")
		return
	}

	common.Success(c, stats)
}

// GetSLAComplianceReport handles GET /api/v1/sla/compliance-report
// @Summary 获取SLA合规报告
// @Description 按日期范围（RFC3339 或 YYYY-MM-DD）汇总合规率与平均响应/解决时长；非法日期返回 400。
// @Tags SLA管理
// @Produce json
// @Param startDate query string false "开始日期"
// @Param endDate query string false "结束日期"
// @Success 200 {object} common.Response{data=dto.SLAComplianceReport}
// @Failure 400 {object} common.Response "日期格式非法"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/sla/compliance-report [get]
func (h *Handler) GetSLAComplianceReport(c *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	// Parse query parameters.
	// camelCase 是查询参数的契约写法，snake_case 仅兼容存量调用方。
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	if startDateStr == "" || endDateStr == "" {
		common.ParamError(c, "startDate and endDate are required")
		return
	}

	// Parse ISO 8601 timestamps
	startDate, err := parseSLADateFilter(startDateStr, false)
	if err != nil {
		common.ParamError(c, "invalid startDate format, use YYYY-MM-DD or RFC3339 (e.g., 2024-01-01T00:00:00Z)")
		return
	}
	endDate, err := parseSLADateFilter(endDateStr, true)
	if err != nil {
		common.ParamError(c, "invalid endDate format, use YYYY-MM-DD or RFC3339 (e.g., 2024-01-31T23:59:59Z)")
		return
	}

	report, err := h.svc.GetComplianceReport(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		common.RespondError(c, err, "生成SLA合规报告失败")
		return
	}

	common.Success(c, report)
}
