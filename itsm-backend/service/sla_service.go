package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/slametric"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type SLAService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewSLAService(client *ent.Client, logger *zap.SugaredLogger) *SLAService {
	return &SLAService{
		client: client,
		logger: logger,
	}
}

// CreateSLADefinition 创建SLA定义
func (s *SLAService) CreateSLADefinition(ctx context.Context, req *dto.CreateSLADefinitionRequest, tenantID int) (*dto.SLADefinitionResponse, error) {
	s.logger.Infow("Creating SLA definition", "name", req.Name, "tenant_id", tenantID)

	slaDefinition, err := s.client.SLADefinition.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetServiceType(req.ServiceType).
		SetPriority(req.Priority).
		SetResponseTime(req.ResponseTime).
		SetResolutionTime(req.ResolutionTime).
		SetBusinessHours(req.BusinessHours).
		SetEscalationRules(req.EscalationRules).
		SetConditions(req.Conditions).
		SetIsActive(req.IsActive).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create SLA definition", "error", err)
		return nil, fmt.Errorf("failed to create SLA definition: %w", err)
	}

	s.logger.Infow("SLA definition created successfully", "id", slaDefinition.ID)
	return s.toSLADefinitionResponse(slaDefinition), nil
}

// GetSLADefinition 获取SLA定义
func (s *SLAService) GetSLADefinition(ctx context.Context, id int, tenantID int) (*dto.SLADefinitionResponse, error) {
	slaDefinition, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.IDEQ(id),
			sladefinition.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("SLA definition not found")
		}
		s.logger.Errorw("Failed to get SLA definition", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get SLA definition: %w", err)
	}

	return s.toSLADefinitionResponse(slaDefinition), nil
}

// ListSLADefinitions 获取SLA定义列表
func (s *SLAService) ListSLADefinitions(ctx context.Context, tenantID int, page, size int) ([]*dto.SLADefinitionResponse, int, error) {
	query := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID))

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count SLA definitions", "error", err)
		return nil, 0, fmt.Errorf("failed to count SLA definitions: %w", err)
	}

	// 分页查询
	slaDefinitions, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(sladefinition.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list SLA definitions", "error", err)
		return nil, 0, fmt.Errorf("failed to list SLA definitions: %w", err)
	}

	responses := make([]*dto.SLADefinitionResponse, len(slaDefinitions))
	for i, sla := range slaDefinitions {
		responses[i] = s.toSLADefinitionResponse(sla)
	}

	return responses, total, nil
}

// UpdateSLADefinition 更新SLA定义
func (s *SLAService) UpdateSLADefinition(ctx context.Context, id int, req *dto.UpdateSLADefinitionRequest, tenantID int) (*dto.SLADefinitionResponse, error) {
	s.logger.Infow("Updating SLA definition", "id", id, "tenant_id", tenantID)

	updateQuery := s.client.SLADefinition.UpdateOneID(id).
		Where(sladefinition.TenantIDEQ(tenantID)).
		SetUpdatedAt(time.Now())

	if req.Name != nil {
		updateQuery.SetName(*req.Name)
	}
	if req.Description != nil {
		updateQuery.SetDescription(*req.Description)
	}
	if req.ServiceType != nil {
		updateQuery.SetServiceType(*req.ServiceType)
	}
	if req.Priority != nil {
		updateQuery.SetPriority(*req.Priority)
	}
	if req.ResponseTime != nil {
		updateQuery.SetResponseTime(*req.ResponseTime)
	}
	if req.ResolutionTime != nil {
		updateQuery.SetResolutionTime(*req.ResolutionTime)
	}
	if req.BusinessHours != nil {
		updateQuery.SetBusinessHours(req.BusinessHours)
	}
	if req.EscalationRules != nil {
		updateQuery.SetEscalationRules(req.EscalationRules)
	}
	if req.Conditions != nil {
		updateQuery.SetConditions(req.Conditions)
	}
	if req.IsActive != nil {
		updateQuery.SetIsActive(*req.IsActive)
	}

	slaDefinition, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("SLA definition not found")
		}
		s.logger.Errorw("Failed to update SLA definition", "error", err)
		return nil, fmt.Errorf("failed to update SLA definition: %w", err)
	}

	s.logger.Infow("SLA definition updated successfully", "id", id)
	return s.toSLADefinitionResponse(slaDefinition), nil
}

// DeleteSLADefinition 删除SLA定义
func (s *SLAService) DeleteSLADefinition(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting SLA definition", "id", id, "tenant_id", tenantID)

	err := s.client.SLADefinition.DeleteOneID(id).
		Where(sladefinition.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("SLA definition not found")
		}
		s.logger.Errorw("Failed to delete SLA definition", "error", err)
		return fmt.Errorf("failed to delete SLA definition: %w", err)
	}

	s.logger.Infow("SLA definition deleted successfully", "id", id)
	return nil
}

// CreateSLAViolation 创建SLA违规记录
func (s *SLAService) CreateSLAViolation(ctx context.Context, req *dto.CreateSLAViolationRequest, tenantID int) (*dto.SLAViolationResponse, error) {
	s.logger.Infow("Creating SLA violation", "ticket_id", req.TicketID, "type", req.ViolationType)

	violation, err := s.client.SLAViolation.Create().
		SetSLADefinitionID(req.SLADefinitionID).
		SetTicketID(req.TicketID).
		SetViolationType(req.ViolationType).
		SetViolationTime(time.Now()).
		SetDescription(req.Description).
		SetSeverity(req.Severity).
		SetIsResolved(false).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create SLA violation", "error", err)
		return nil, fmt.Errorf("failed to create SLA violation: %w", err)
	}

	s.logger.Infow("SLA violation created successfully", "id", violation.ID)
	return s.toSLAViolationResponse(violation), nil
}

// ListSLAViolations 获取SLA违规记录列表
func (s *SLAService) ListSLAViolations(ctx context.Context, tenantID int, page, size int, slaDefinitionID *int, ticketID *int) ([]*dto.SLAViolationResponse, int, error) {
	query := s.client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tenantID))

	if slaDefinitionID != nil {
		query = query.Where(slaviolation.SLADefinitionIDEQ(*slaDefinitionID))
	}
	if ticketID != nil {
		query = query.Where(slaviolation.TicketIDEQ(*ticketID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count SLA violations", "error", err)
		return nil, 0, fmt.Errorf("failed to count SLA violations: %w", err)
	}

	// 分页查询
	violations, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(slaviolation.FieldViolationTime)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list SLA violations", "error", err)
		return nil, 0, fmt.Errorf("failed to list SLA violations: %w", err)
	}

	responses := make([]*dto.SLAViolationResponse, len(violations))
	for i, violation := range violations {
		responses[i] = s.toSLAViolationResponse(violation)
	}

	return responses, total, nil
}

// UpdateSLAViolation 更新SLA违规记录
func (s *SLAService) UpdateSLAViolation(ctx context.Context, id int, req *dto.UpdateSLAViolationRequest, tenantID int) (*dto.SLAViolationResponse, error) {
	s.logger.Infow("Updating SLA violation", "id", id, "tenant_id", tenantID)

	updateQuery := s.client.SLAViolation.UpdateOneID(id).
		Where(slaviolation.TenantIDEQ(tenantID)).
		SetUpdatedAt(time.Now())

	if req.Description != nil {
		updateQuery.SetDescription(*req.Description)
	}
	if req.Severity != nil {
		updateQuery.SetSeverity(*req.Severity)
	}
	if req.IsResolved != nil {
		updateQuery.SetIsResolved(*req.IsResolved)
		if *req.IsResolved {
			now := time.Now()
			updateQuery.SetResolvedAt(now)
		}
	}
	if req.ResolutionNotes != nil {
		updateQuery.SetResolutionNotes(*req.ResolutionNotes)
	}

	violation, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("SLA violation not found")
		}
		s.logger.Errorw("Failed to update SLA violation", "error", err)
		return nil, fmt.Errorf("failed to update SLA violation: %w", err)
	}

	s.logger.Infow("SLA violation updated successfully", "id", id)
	return s.toSLAViolationResponse(violation), nil
}

// CreateSLAMetric 创建SLA指标
func (s *SLAService) CreateSLAMetric(ctx context.Context, req *dto.CreateSLAMetricRequest, tenantID int) (*dto.SLAMetricResponse, error) {
	s.logger.Infow("Creating SLA metric", "type", req.MetricType, "name", req.MetricName)

	metric, err := s.client.SLAMetric.Create().
		SetSLADefinitionID(req.SLADefinitionID).
		SetMetricType(req.MetricType).
		SetMetricName(req.MetricName).
		SetMetricValue(req.MetricValue).
		SetUnit(req.Unit).
		SetMeasurementTime(time.Now()).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create SLA metric", "error", err)
		return nil, fmt.Errorf("failed to create SLA metric: %w", err)
	}

	s.logger.Infow("SLA metric created successfully", "id", metric.ID)
	return s.toSLAMetricResponse(metric), nil
}

// GetSLAMetrics 获取SLA指标
func (s *SLAService) GetSLAMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*dto.SLAMetricResponse, error) {
	s.logger.Infow("Getting SLA metrics", "tenant_id", tenantID)

	query := s.client.SLAMetric.Query().
		Where(slametric.TenantIDEQ(tenantID))

	// 应用过滤器
	if slaDefID, ok := filters["sla_definition_id"].(int); ok && slaDefID > 0 {
		query = query.Where(slametric.SLADefinitionIDEQ(slaDefID))
	}
	if metricType, ok := filters["metric_type"].(string); ok && metricType != "" {
		query = query.Where(slametric.MetricTypeEQ(metricType))
	}

	// 获取最近30天的数据
	startTime := time.Now().AddDate(0, 0, -30)
	query = query.Where(slametric.MeasurementTimeGTE(startTime))

	// 排序
	query = query.Order(ent.Desc(slametric.FieldMeasurementTime))

	metrics, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA metrics", "error", err)
		return nil, fmt.Errorf("failed to get SLA metrics: %w", err)
	}

	responses := make([]*dto.SLAMetricResponse, len(metrics))
	for i, m := range metrics {
		responses[i] = s.toSLAMetricResponse(m)
	}

	return responses, nil
}

// UpdateSLAViolationStatus 更新SLA违规状态
func (s *SLAService) UpdateSLAViolationStatus(ctx context.Context, id int, isResolved bool, notes string, tenantID int) (*dto.SLAViolationResponse, error) {
	s.logger.Infow("Updating SLA violation status", "id", id, "is_resolved", isResolved)

	violation, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.IDEQ(id),
			slaviolation.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("违规记录不存在")
		}
		return nil, fmt.Errorf("查询违规记录失败: %w", err)
	}

	// 更新状态
	update := violation.Update().
		SetIsResolved(isResolved)
	if isResolved {
		update = update.SetResolvedAt(time.Now())
	}
	if notes != "" {
		update = update.SetResolutionNotes(notes)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新违规状态失败: %w", err)
	}

	return s.toSLAViolationResponse(updated), nil
}

// GetSLAViolations 获取SLA违规列表
func (s *SLAService) GetSLAViolations(ctx context.Context, tenantID int, filters map[string]interface{}, page, size int) ([]*dto.SLAViolationResponse, int, error) {
	s.logger.Infow("Getting SLA violations", "tenant_id", tenantID)

	query := s.client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tenantID))

	// 应用过滤器
	if isResolved, ok := filters["is_resolved"].(bool); ok {
		query = query.Where(slaviolation.IsResolvedEQ(isResolved))
	}
	if severity, ok := filters["severity"].(string); ok && severity != "" {
		query = query.Where(slaviolation.SeverityEQ(severity))
	}
	if violationType, ok := filters["violation_type"].(string); ok && violationType != "" {
		query = query.Where(slaviolation.ViolationTypeEQ(violationType))
	}
	if slaDefID, ok := filters["sla_definition_id"].(int); ok && slaDefID > 0 {
		query = query.Where(slaviolation.SLADefinitionIDEQ(slaDefID))
	}

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count violations: %w", err)
	}

	// 分页
	violations, err := query.
		Order(ent.Desc(slaviolation.FieldViolationTime)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA violations", "error", err)
		return nil, 0, fmt.Errorf("failed to get SLA violations: %w", err)
	}

	responses := make([]*dto.SLAViolationResponse, len(violations))
	for i, v := range violations {
		responses[i] = s.toSLAViolationResponse(v)
	}

	return responses, total, nil
}

// GetSLAMonitoring 获取SLA监控数据
func (s *SLAService) GetSLAMonitoring(ctx context.Context, req *dto.SLAMonitoringRequest, tenantID int) (*dto.SLAMonitoringResponse, error) {
	s.logger.Infow("Getting SLA monitoring data", "tenant_id", tenantID)

	// 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
	}

	// 获取SLA定义
	var slaDefinition *ent.SLADefinition
	if req.SLADefinitionID != nil {
		slaDefinition, err = s.client.SLADefinition.Query().
			Where(
				sladefinition.IDEQ(*req.SLADefinitionID),
				sladefinition.TenantIDEQ(tenantID),
			).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get SLA definition: %w", err)
		}
	} else {
		// 获取默认SLA定义
		slaDefinition, err = s.client.SLADefinition.Query().
			Where(
				sladefinition.TenantIDEQ(tenantID),
				sladefinition.IsActiveEQ(true),
			).
			First(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default SLA definition: %w", err)
		}
	}

	// 获取指标数据
	metrics, err := s.client.SLAMetric.Query().
		Where(
			slametric.SLADefinitionIDEQ(slaDefinition.ID),
			slametric.TenantIDEQ(tenantID),
			slametric.MeasurementTimeGTE(startTime),
			slametric.MeasurementTimeLTE(endTime),
		).
		Order(ent.Desc(slametric.FieldMeasurementTime)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA metrics", "error", err)
		return nil, fmt.Errorf("failed to get SLA metrics: %w", err)
	}

	// 获取违规记录
	violations, err := s.client.SLAViolation.Query().
		Where(
			slaviolation.SLADefinitionIDEQ(slaDefinition.ID),
			slaviolation.TenantIDEQ(tenantID),
			slaviolation.ViolationTimeGTE(startTime),
			slaviolation.ViolationTimeLTE(endTime),
		).
		Order(ent.Desc(slaviolation.FieldViolationTime)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get SLA violations", "error", err)
		return nil, fmt.Errorf("failed to get SLA violations: %w", err)
	}

	// 计算合规率
	totalTickets, err := s.client.Ticket.Query().
		Where(
			ticket.SLADefinitionIDEQ(slaDefinition.ID),
			ticket.TenantIDEQ(tenantID),
			ticket.CreatedAtGTE(startTime),
			ticket.CreatedAtLTE(endTime),
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count tickets", "error", err)
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	complianceRate := 100.0
	if totalTickets > 0 {
		complianceRate = float64(totalTickets-len(violations)) / float64(totalTickets) * 100
	}

	// 计算平均响应时间和解决时间
	var avgResponseTime, avgResolutionTime float64
	if len(metrics) > 0 {
		var totalResponseTime, totalResolutionTime float64
		var responseCount, resolutionCount int

		for _, metric := range metrics {
			if metric.MetricType == "response_time" {
				totalResponseTime += metric.MetricValue
				responseCount++
			} else if metric.MetricType == "resolution_time" {
				totalResolutionTime += metric.MetricValue
				resolutionCount++
			}
		}

		if responseCount > 0 {
			avgResponseTime = totalResponseTime / float64(responseCount)
		}
		if resolutionCount > 0 {
			avgResolutionTime = totalResolutionTime / float64(resolutionCount)
		}
	}

	// 构建响应
	response := &dto.SLAMonitoringResponse{
		SLADefinitionID:       slaDefinition.ID,
		SLAInfo:               *s.toSLADefinitionResponse(slaDefinition),
		ComplianceRate:        complianceRate,
		AverageResponseTime:   avgResponseTime,
		AverageResolutionTime: avgResolutionTime,
		TotalTickets:          totalTickets,
		ViolatedTickets:       len(violations),
	}

	// 转换指标和违规记录
	response.Metrics = make([]dto.SLAMetricResponse, len(metrics))
	for i, metric := range metrics {
		response.Metrics[i] = *s.toSLAMetricResponse(metric)
	}

	response.Violations = make([]dto.SLAViolationResponse, len(violations))
	for i, violation := range violations {
		response.Violations[i] = *s.toSLAViolationResponse(violation)
	}

	return response, nil
}

// CheckSLACompliance 检查SLA合规性
func (s *SLAService) CheckSLACompliance(ctx context.Context, ticketID int, tenantID int) error {
	s.logger.Infow("Checking SLA compliance", "ticket_id", ticketID)

	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.IDEQ(ticketID),
			ticket.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if ticketEntity.SLADefinitionID == 0 {
		s.logger.Infow("Ticket has no SLA definition", "ticket_id", ticketID)
		return nil
	}

	// 获取SLA定义
	slaDefinition, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.IDEQ(ticketEntity.SLADefinitionID),
			sladefinition.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SLA definition: %w", err)
	}

	now := time.Now()

	// 检查响应时间合规性
	if !ticketEntity.SLAResponseDeadline.IsZero() && now.After(ticketEntity.SLAResponseDeadline) {
		if ticketEntity.FirstResponseAt.IsZero() {
			// 创建响应时间违规记录
			_, err = s.CreateSLAViolation(ctx, &dto.CreateSLAViolationRequest{
				SLADefinitionID: slaDefinition.ID,
				TicketID:        ticketID,
				ViolationType:   "response_time",
				Description:     "响应时间超时",
				Severity:        "high",
			}, tenantID)
			if err != nil {
				s.logger.Errorw("Failed to create response time violation", "error", err)
			}
		}
	}

	// 检查解决时间合规性
	if !ticketEntity.SLAResolutionDeadline.IsZero() && now.After(ticketEntity.SLAResolutionDeadline) {
		if ticketEntity.ResolvedAt.IsZero() {
			// 创建解决时间违规记录
			_, err = s.CreateSLAViolation(ctx, &dto.CreateSLAViolationRequest{
				SLADefinitionID: slaDefinition.ID,
				TicketID:        ticketID,
				ViolationType:   "resolution_time",
				Description:     "解决时间超时",
				Severity:        "critical",
			}, tenantID)
			if err != nil {
				s.logger.Errorw("Failed to create resolution time violation", "error", err)
			}
		}
	}

	return nil
}

// 辅助方法：转换为响应DTO
func (s *SLAService) toSLADefinitionResponse(sla *ent.SLADefinition) *dto.SLADefinitionResponse {
	return &dto.SLADefinitionResponse{
		ID:              sla.ID,
		Name:            sla.Name,
		Description:     sla.Description,
		ServiceType:     sla.ServiceType,
		Priority:        sla.Priority,
		ResponseTime:    sla.ResponseTime,
		ResolutionTime:  sla.ResolutionTime,
		BusinessHours:   sla.BusinessHours,
		EscalationRules: sla.EscalationRules,
		Conditions:      sla.Conditions,
		IsActive:        sla.IsActive,
		TenantID:        sla.TenantID,
		CreatedAt:       sla.CreatedAt,
		UpdatedAt:       sla.UpdatedAt,
	}
}

func (s *SLAService) toSLAViolationResponse(violation *ent.SLAViolation) *dto.SLAViolationResponse {
	return &dto.SLAViolationResponse{
		ID:              violation.ID,
		SLADefinitionID: violation.SLADefinitionID,
		TicketID:        violation.TicketID,
		ViolationType:   violation.ViolationType,
		ViolationTime:   violation.ViolationTime,
		Description:     violation.Description,
		Severity:        violation.Severity,
		IsResolved:      violation.IsResolved,
		ResolvedAt:      &violation.ResolvedAt,
		ResolutionNotes: violation.ResolutionNotes,
		TenantID:        violation.TenantID,
		CreatedAt:       violation.CreatedAt,
		UpdatedAt:       violation.UpdatedAt,
	}
}

func (s *SLAService) toSLAMetricResponse(metric *ent.SLAMetric) *dto.SLAMetricResponse {
	return &dto.SLAMetricResponse{
		ID:              metric.ID,
		SLADefinitionID: metric.SLADefinitionID,
		MetricType:      metric.MetricType,
		MetricName:      metric.MetricName,
		MetricValue:     metric.MetricValue,
		Unit:            metric.Unit,
		MeasurementTime: metric.MeasurementTime,
		Metadata:        metric.Metadata,
		TenantID:        metric.TenantID,
		CreatedAt:       metric.CreatedAt,
		UpdatedAt:       metric.UpdatedAt,
	}
}
