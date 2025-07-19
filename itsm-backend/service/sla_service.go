package service

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
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
func (s *SLAService) CreateSLADefinition(ctx context.Context, req *dto.SLADefinitionRequest, tenantID int, createdBy string) (*dto.SLADefinitionResponse, error) {
	definition, err := s.client.SLADefinition.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetServiceType(req.ServiceType).
		SetPriority(req.Priority).
		SetImpact(req.Impact).
		SetResponseTime(req.ResponseTime).
		SetResolutionTime(req.ResolutionTime).
		SetBusinessHours(req.BusinessHours).
		SetHolidays(req.Holidays).
		SetIsActive(req.IsActive).
		SetTenantID(tenantID).
		SetCreatedBy(createdBy).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to create SLA definition", "error", err)
		return nil, err
	}

	return s.toSLADefinitionResponse(definition), nil
}

// GetSLADefinitionByID 根据ID获取SLA定义
func (s *SLAService) GetSLADefinitionByID(ctx context.Context, id int, tenantID int) (*dto.SLADefinitionResponse, error) {
	definition, err := s.client.SLADefinition.Query().
		Where(sladefinition.IDEQ(id), sladefinition.TenantIDEQ(tenantID)).
		First(ctx)

	if err != nil {
		s.logger.Errorw("Failed to get SLA definition", "id", id, "error", err)
		return nil, err
	}

	return s.toSLADefinitionResponse(definition), nil
}

// ListSLADefinitions 获取SLA定义列表
func (s *SLAService) ListSLADefinitions(ctx context.Context, tenantID int, page, pageSize int) (*dto.SLADefinitionListResponse, error) {
	query := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count SLA definitions", "error", err)
		return nil, err
	}

	definitions, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(sladefinition.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		s.logger.Errorw("Failed to list SLA definitions", "error", err)
		return nil, err
	}

	responses := make([]dto.SLADefinitionResponse, len(definitions))
	for i, definition := range definitions {
		responses[i] = *s.toSLADefinitionResponse(definition)
	}

	return &dto.SLADefinitionListResponse{
		Definitions: responses,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// UpdateSLADefinition 更新SLA定义
func (s *SLAService) UpdateSLADefinition(ctx context.Context, id int, req *dto.SLADefinitionRequest, tenantID int) (*dto.SLADefinitionResponse, error) {
	definition, err := s.client.SLADefinition.UpdateOneID(id).
		SetName(req.Name).
		SetDescription(req.Description).
		SetServiceType(req.ServiceType).
		SetPriority(req.Priority).
		SetImpact(req.Impact).
		SetResponseTime(req.ResponseTime).
		SetResolutionTime(req.ResolutionTime).
		SetBusinessHours(req.BusinessHours).
		SetHolidays(req.Holidays).
		SetIsActive(req.IsActive).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to update SLA definition", "id", id, "error", err)
		return nil, err
	}

	return s.toSLADefinitionResponse(definition), nil
}

// DeleteSLADefinition 删除SLA定义
func (s *SLAService) DeleteSLADefinition(ctx context.Context, id int, tenantID int) error {
	err := s.client.SLADefinition.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete SLA definition", "id", id, "error", err)
		return err
	}
	return nil
}

// GetApplicableSLA 获取适用的SLA定义
func (s *SLAService) GetApplicableSLA(ctx context.Context, serviceType, priority, impact string, tenantID int) (*dto.SLADefinitionResponse, error) {
	definition, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantIDEQ(tenantID),
			sladefinition.ServiceTypeEQ(serviceType),
			sladefinition.PriorityEQ(priority),
			sladefinition.ImpactEQ(impact),
			sladefinition.IsActiveEQ(true),
		).
		First(ctx)

	if err != nil {
		s.logger.Errorw("Failed to get applicable SLA", "service_type", serviceType, "priority", priority, "impact", impact, "error", err)
		return nil, err
	}

	return s.toSLADefinitionResponse(definition), nil
}

// CheckSLAViolation 检查SLA违规
func (s *SLAService) CheckSLAViolation(ctx context.Context, ticketID int, ticketType string, tenantID int) error {
	// 获取工单信息
	ticket, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		First(ctx)

	if err != nil {
		s.logger.Errorw("Failed to get ticket for SLA check", "ticket_id", ticketID, "error", err)
		return err
	}

	// 获取适用的SLA定义
	sla, err := s.GetApplicableSLA(ctx, ticketType, string(ticket.Priority), "medium", tenantID)
	if err != nil {
		s.logger.Errorw("No applicable SLA found", "ticket_id", ticketID, "error", err)
		return err
	}

	// 计算工作时间
	elapsedMinutes := s.calculateElapsedMinutes(ticket.CreatedAt, time.Now(), sla.BusinessHours, sla.Holidays)

	// 检查响应时间违规
	if elapsedMinutes > sla.ResponseTime && ticket.Status == "open" {
		err = s.createSLAViolation(ctx, ticketID, ticketType, "response_time", sla, elapsedMinutes, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to create response time violation", "ticket_id", ticketID, "error", err)
		}
	}

	// 检查解决时间违规
	if elapsedMinutes > sla.ResolutionTime && (ticket.Status == "resolved" || ticket.Status == "closed") {
		err = s.createSLAViolation(ctx, ticketID, ticketType, "resolution_time", sla, elapsedMinutes, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to create resolution time violation", "ticket_id", ticketID, "error", err)
		}
	}

	return nil
}

// createSLAViolation 创建SLA违规记录
func (s *SLAService) createSLAViolation(ctx context.Context, ticketID int, ticketType, violationType string, sla *dto.SLADefinitionResponse, actualMinutes int, tenantID int) error {
	var expectedTime int
	if violationType == "response_time" {
		expectedTime = sla.ResponseTime
	} else {
		expectedTime = sla.ResolutionTime
	}

	overdueMinutes := actualMinutes - expectedTime
	if overdueMinutes <= 0 {
		return nil
	}

	_, err := s.client.SLAViolation.Create().
		SetTicketID(ticketID).
		SetTicketType(ticketType).
		SetViolationType(violationType).
		SetSLADefinitionID(sla.ID).
		SetSLAName(sla.Name).
		SetExpectedTime(expectedTime).
		SetActualTime(actualMinutes).
		SetOverdueMinutes(overdueMinutes).
		SetStatus("pending").
		SetViolationOccurredAt(time.Now()).
		SetTenantID(tenantID).
		SetCreatedBy("system").
		Save(ctx)

	return err
}

// calculateElapsedMinutes 计算工作时间内的经过分钟数
func (s *SLAService) calculateElapsedMinutes(startTime, endTime time.Time, businessHoursJSON, holidaysJSON string) int {
	// 这里简化实现，实际应该解析businessHours和holidays配置
	// 计算工作日内的分钟数
	duration := endTime.Sub(startTime)
	return int(duration.Minutes())
}

// GetSLAViolations 获取SLA违规列表
func (s *SLAService) GetSLAViolations(ctx context.Context, tenantID int, page, pageSize int, status string) (*dto.SLAViolationListResponse, error) {
	query := s.client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tenantID))

	if status != "" {
		query = query.Where(slaviolation.StatusEQ(status))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count SLA violations", "error", err)
		return nil, err
	}

	violations, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(slaviolation.FieldViolationOccurredAt)).
		All(ctx)

	if err != nil {
		s.logger.Errorw("Failed to list SLA violations", "error", err)
		return nil, err
	}

	responses := make([]dto.SLAViolationResponse, len(violations))
	for i, violation := range violations {
		responses[i] = *s.toSLAViolationResponse(violation)
	}

	return &dto.SLAViolationListResponse{
		Violations: responses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// UpdateSLAViolationStatus 更新SLA违规状态
func (s *SLAService) UpdateSLAViolationStatus(ctx context.Context, id int, status string, assignedTo string, resolutionNote string, tenantID int) error {
	update := s.client.SLAViolation.UpdateOneID(id).
		SetStatus(status)

	if assignedTo != "" {
		update = update.SetAssignedTo(assignedTo)
	}

	if status == "resolved" {
		now := time.Now()
		update = update.SetResolvedAt(now)
		if resolutionNote != "" {
			update = update.SetResolutionNote(resolutionNote)
		}
	}

	err := update.Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update SLA violation status", "id", id, "error", err)
		return err
	}

	return nil
}

// GenerateSLAMetrics 生成SLA指标
func (s *SLAService) GenerateSLAMetrics(ctx context.Context, tenantID int, period string, periodStart, periodEnd time.Time) error {
	// 这里实现SLA指标统计逻辑
	// 包括达标率、平均响应时间、平均解决时间等
	s.logger.Infow("Generating SLA metrics", "tenant_id", tenantID, "period", period)
	return nil
}

// GetSLAComplianceReport 获取SLA达标率报表
func (s *SLAService) GetSLAComplianceReport(ctx context.Context, tenantID int, periodStart, periodEnd time.Time) (*dto.SLAComplianceReportResponse, error) {
	// 实现SLA达标率报表逻辑
	report := &dto.SLAComplianceReportResponse{
		OverallComplianceRate: 0.0,
		ServiceTypeMetrics:    make(map[string]dto.ServiceMetrics),
		PriorityMetrics:       make(map[string]dto.PriorityMetrics),
		TrendData:             []dto.TrendData{},
		Period:                "custom",
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
	}

	return report, nil
}

// toSLADefinitionResponse 转换为SLA定义响应
func (s *SLAService) toSLADefinitionResponse(definition *ent.SLADefinition) *dto.SLADefinitionResponse {
	return &dto.SLADefinitionResponse{
		ID:             definition.ID,
		Name:           definition.Name,
		Description:    definition.Description,
		ServiceType:    definition.ServiceType,
		Priority:       definition.Priority,
		Impact:         definition.Impact,
		ResponseTime:   definition.ResponseTime,
		ResolutionTime: definition.ResolutionTime,
		BusinessHours:  definition.BusinessHours,
		Holidays:       definition.Holidays,
		IsActive:       definition.IsActive,
		TenantID:       definition.TenantID,
		CreatedBy:      definition.CreatedBy,
		CreatedAt:      definition.CreatedAt,
		UpdatedAt:      definition.UpdatedAt,
	}
}

// toSLAViolationResponse 转换为SLA违规响应
func (s *SLAService) toSLAViolationResponse(violation *ent.SLAViolation) *dto.SLAViolationResponse {
	response := &dto.SLAViolationResponse{
		ID:                  violation.ID,
		TicketID:            violation.TicketID,
		TicketType:          violation.TicketType,
		ViolationType:       violation.ViolationType,
		SLADefinitionID:     violation.SLADefinitionID,
		SLAName:             violation.SLAName,
		ExpectedTime:        violation.ExpectedTime,
		ActualTime:          violation.ActualTime,
		OverdueMinutes:      violation.OverdueMinutes,
		Status:              violation.Status,
		ViolationOccurredAt: violation.ViolationOccurredAt,
		TenantID:            violation.TenantID,
		CreatedBy:           violation.CreatedBy,
		CreatedAt:           violation.CreatedAt,
		UpdatedAt:           violation.UpdatedAt,
	}

	// 处理可选字段
	if violation.AssignedTo != "" {
		response.AssignedTo = &violation.AssignedTo
	}
	if !violation.ResolvedAt.IsZero() {
		response.ResolvedAt = &violation.ResolvedAt
	}
	if violation.ResolutionNote != "" {
		response.ResolutionNote = &violation.ResolutionNote
	}

	return response
}
