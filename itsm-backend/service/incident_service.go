package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"
	"itsm-backend/ent/incidentrule"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

type IncidentService struct {
	priorityMatrixService *PriorityMatrixService
	client                *ent.Client
	logger                *zap.SugaredLogger
	sequenceService       *SequenceService
	processTriggerService ProcessTriggerServiceInterface
}

func NewIncidentService(client *ent.Client, logger *zap.SugaredLogger) *IncidentService {
	return &IncidentService{
		client: client,
		logger: logger,
	}
}

// SetProcessTriggerService 设置流程触发服务
func (s *IncidentService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

// SetSequenceService 设置序列服务（用于 incident_number 生成）
func (s *IncidentService) SetPriorityMatrixService(pms *PriorityMatrixService) {
	s.priorityMatrixService = pms
}

func (s *IncidentService) SetSequenceService(seq *SequenceService) {
	s.sequenceService = seq
}

// CreateIncident 创建事件
func (s *IncidentService) CreateIncident(ctx context.Context, req *dto.CreateIncidentRequest, tenantID, userID int) (*dto.IncidentResponse, error) {
	s.logger.Infow("Creating incident", "title", req.Title, "tenant_id", tenantID, "user_id", userID)

	// 生成事件编号
	incidentNumber, err := s.generateIncidentNumber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate incident number: %w", err)
	}

	// 设置检测时间
	detectedAt := time.Now()
	if req.DetectedAt != nil {
		detectedAt = *req.DetectedAt
	}

	// 计算优先级
	priority := req.Priority
	if priority == "" && s.priorityMatrixService != nil {
		impact := req.Impact
		if impact == "" {
			impact = "medium"
		}
		urgency := req.Urgency
		if urgency == "" {
			urgency = "medium"
		}
		
		calculatedPriority, err := s.priorityMatrixService.CalculatePriority(tenantID, impact, urgency)
		if err != nil {
			s.logger.Warnw("Failed to calculate priority, using default medium", "error", err)
			priority = "medium"
		} else {
			priority = calculatedPriority
		}
	}
	
	// 如果最终priority还是空，使用默认值
	if priority == "" {
		priority = "medium"
	}

	incidentEntity, err := s.client.Incident.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetStatus("new").
		SetPriority(priority).
		SetSeverity(req.Severity).
		SetImpact(req.Impact).
		SetUrgency(req.Urgency).
		SetIncidentNumber(incidentNumber).
		SetReporterID(userID).
		SetCategory(req.Category).
		SetSubcategory(req.Subcategory).
		SetImpactAnalysis(dto.StructToMap(req.ImpactAnalysis)).
		SetSource(req.Source).
		SetMetadata(req.Metadata).
		SetDetectedAt(detectedAt).
		SetIsAutomated(false).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident", "error", err)
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}
	if err != nil {
		s.logger.Errorw("Failed to create incident", "error", err)
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}

	// 设置可选的字段
	if req.ConfigurationItemID != nil {
		incidentEntity, err = s.client.Incident.UpdateOneID(incidentEntity.ID).
			SetConfigurationItemID(*req.ConfigurationItemID).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update incident configuration item", "error", err)
		}
	}

	if req.AssigneeID != nil {
		incidentEntity, err = s.client.Incident.UpdateOneID(incidentEntity.ID).
			SetAssigneeID(*req.AssigneeID).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update incident assignee", "error", err)
		}
	}

	// 记录事件创建活动
	_, err = s.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  incidentEntity.ID,
		EventType:   "creation",
		EventName:   "事件创建",
		Description: fmt.Sprintf("事件 %s 已创建", incidentNumber),
		Status:      "active",
		Severity:    "info",
		Source:      "system",
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create incident event", "error", err)
	}

	// 执行事件规则
	go s.executeIncidentRules(context.Background(), incidentEntity.ID, tenantID)

	// 触发BPMN工作流（异步执行，不阻塞事件创建）
	if s.processTriggerService != nil {
		go func() {
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForIncident(workflowCtx, incidentEntity.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to trigger workflow for incident", "error", err, "incident_id", incidentEntity.ID)
			}
		}()
	}

	s.logger.Infow("Incident created successfully", "id", incidentEntity.ID, "number", incidentNumber)
	return s.toIncidentResponse(incidentEntity), nil
}

// GetIncident 获取事件
func (s *IncidentService) GetIncident(ctx context.Context, id int, tenantID int) (*dto.IncidentResponse, error) {
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(id),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		s.logger.Errorw("Failed to get incident", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	return s.toIncidentResponse(incidentEntity), nil
}

// ListIncidents 获取事件列表
func (s *IncidentService) ListIncidents(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*dto.IncidentResponse, int, error) {
	query := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID))

	// 应用过滤器
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where(incident.StatusEQ(status))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(incident.PriorityEQ(priority))
	}
	if severity, ok := filters["severity"].(string); ok && severity != "" {
		query = query.Where(incident.SeverityEQ(severity))
	}
	if category, ok := filters["category"].(string); ok && category != "" {
		query = query.Where(incident.CategoryEQ(category))
	}
	if source, ok := filters["source"].(string); ok && source != "" {
		query = query.Where(incident.SourceEQ(source))
	}
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		// 关键词搜索：标题、描述、事件编号
		query = query.Where(
			incident.Or(
				incident.TitleContains(keyword),
				incident.DescriptionContains(keyword),
				incident.IncidentNumberContains(keyword),
			),
		)
	}
	if assigneeID, ok := filters["assignee_id"].(int); ok && assigneeID > 0 {
		query = query.Where(incident.AssigneeIDEQ(assigneeID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count incidents", "error", err)
		return nil, 0, fmt.Errorf("failed to count incidents: %w", err)
	}

	// 分页查询
	incidents, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(incident.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list incidents", "error", err)
		return nil, 0, fmt.Errorf("failed to list incidents: %w", err)
	}

	responses := make([]*dto.IncidentResponse, len(incidents))
	for i, incidentEntity := range incidents {
		responses[i] = s.toIncidentResponse(incidentEntity)
	}

	return responses, total, nil
}

// UpdateIncident 更新事件
func (s *IncidentService) UpdateIncident(ctx context.Context, id int, req *dto.UpdateIncidentRequest, tenantID int) (*dto.IncidentResponse, error) {
	s.logger.Infow("Updating incident", "id", id, "tenant_id", tenantID)

	// 获取当前事件实体
	currentIncident, err := s.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	// 版本检查（乐观锁）- 除非明确强制更新
	if !req.Force && req.Version > 0 && currentIncident.Version != req.Version {
		return nil, common.NewVersionConflictError(
			"事件",
			id,
			req.Version,
			currentIncident.Version,
		)
	}

	// 如果要更新状态，验证状态转换是否合法
	if req.Status != nil {
		// 验证状态转换
		if !isValidIncidentStatusTransition(currentIncident.Status, *req.Status) {
			return nil, fmt.Errorf("invalid status transition from '%s' to '%s'", currentIncident.Status, *req.Status)
		}
	}

	// 计算优先级：如果用户没有显式传入Priority，但修改了Impact或Urgency，则自动重新计算
	priority := req.Priority
	if priority == nil && s.priorityMatrixService != nil && (req.Impact != nil || req.Urgency != nil) {
		// 使用新的Impact或现有Impact
		impact := currentIncident.Impact
		if req.Impact != nil {
			impact = *req.Impact
		}
		
		// 使用新的Urgency或现有Urgency
		urgency := currentIncident.Urgency
		if req.Urgency != nil {
			urgency = *req.Urgency
		}
		
		calculatedPriority, err := s.priorityMatrixService.CalculatePriority(tenantID, impact, urgency)
		if err != nil {
			s.logger.Warnw("Failed to calculate priority during update, keeping current value", "error", err)
		} else {
			priority = &calculatedPriority
		}
	}

	updateQuery := s.client.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID)).
		SetUpdatedAt(time.Now())

	if req.Title != nil {
		updateQuery.SetTitle(*req.Title)
	}
	if req.Description != nil {
		updateQuery.SetDescription(*req.Description)
	}
	if req.Status != nil {
		updateQuery.SetStatus(*req.Status)
		// 如果状态变更为resolved，设置解决时间
		if *req.Status == common.IncidentStatusResolved {
			now := time.Now()
			updateQuery.SetResolvedAt(now)
		}
		// 如果状态变更为closed，设置关闭时间
		if *req.Status == common.IncidentStatusClosed {
			now := time.Now()
			updateQuery.SetClosedAt(now)
		}
	}
	if priority != nil {
		updateQuery.SetPriority(*priority)
	}
	if req.Severity != nil {
		updateQuery.SetSeverity(*req.Severity)
	}
	if req.Impact != nil {
		updateQuery.SetImpact(*req.Impact)
	}
	if req.Urgency != nil {
		updateQuery.SetUrgency(*req.Urgency)
	}
	if req.AssigneeID != nil {
		updateQuery.SetAssigneeID(*req.AssigneeID)
	}
	if req.ImpactAnalysis != nil {
		updateQuery.SetImpactAnalysis(dto.StructToMap(req.ImpactAnalysis))
	}
	if req.RootCause != nil {
		updateQuery.SetRootCause(dto.StructToMap(req.RootCause))
	}
	if req.ResolutionSteps != nil {
		updateQuery.SetResolutionSteps(dto.StructSliceToMapSlice(req.ResolutionSteps))
	}
	if req.Metadata != nil {
		updateQuery.SetMetadata(req.Metadata)
	}

	// 自动增加版本号
	updateQuery.AddVersion(1)

	incidentEntity, err := updateQuery.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		s.logger.Errorw("Failed to update incident", "error", err)
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	// 记录事件更新活动
	_, err = s.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  id,
		EventType:   "update",
		EventName:   "事件更新",
		Description: "事件信息已更新",
		Status:      "active",
		Severity:    "info",
		Source:      "system",
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create incident event", "error", err)
	}

	s.logger.Infow("Incident updated successfully", "id", id)
	return s.toIncidentResponse(incidentEntity), nil
}

// DeleteIncident deletes an incident with cascade cleanup of related data.
// Tenant validation is performed before any deletion to prevent cross-tenant access.
func (s *IncidentService) DeleteIncident(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting incident", "id", id, "tenant_id", tenantID)

	// First verify the incident belongs to the current tenant
	incidentEntity, err := s.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("cross-tenant access denied: incident not found")
		}
		return fmt.Errorf("failed to query incident: %w", err)
	}

	// Delete cascade records with tenant filter to prevent cross-tenant deletion
	// Delete incident_events
	_, err = s.client.IncidentEvent.Delete().
		Where(incidentevent.IncidentIDEQ(id), incidentevent.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil && !ent.IsNotFound(err) {
		s.logger.Errorw("Failed to delete incident events", "error", err)
		return fmt.Errorf("failed to delete incident events: %w", err)
	}

	// Delete incident_alerts
	_, err = s.client.IncidentAlert.Delete().
		Where(incidentalert.IncidentIDEQ(id), incidentalert.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil && !ent.IsNotFound(err) {
		s.logger.Errorw("Failed to delete incident alerts", "error", err)
		return fmt.Errorf("failed to delete incident alerts: %w", err)
	}

	// Delete incident_metrics
	_, err = s.client.IncidentMetric.Delete().
		Where(incidentmetric.IncidentIDEQ(id), incidentmetric.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil && !ent.IsNotFound(err) {
		s.logger.Errorw("Failed to delete incident metrics", "error", err)
		return fmt.Errorf("failed to delete incident metrics: %w", err)
	}

	// Delete the incident itself
	err = s.client.Incident.DeleteOne(incidentEntity).
		Where(incident.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete incident", "error", err)
		return fmt.Errorf("failed to delete incident: %w", err)
	}

	s.logger.Infow("Incident deleted successfully", "id", id)
	return nil
}

// CreateIncidentEvent 创建事件活动记录
func (s *IncidentService) CreateIncidentEvent(ctx context.Context, req *dto.CreateIncidentEventRequest, tenantID int) (*dto.IncidentEventResponse, error) {
	s.logger.Infow("Creating incident event", "incident_id", req.IncidentID, "type", req.EventType)

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}

	eventBuilder := s.client.IncidentEvent.Create().
		SetIncidentID(req.IncidentID).
		SetEventType(req.EventType).
		SetEventName(req.EventName).
		SetDescription(req.Description).
		SetStatus(req.Status).
		SetSeverity(req.Severity).
		SetData(req.Data).
		SetOccurredAt(occurredAt).
		SetSource(req.Source).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if req.UserID != nil {
		eventBuilder.SetUserID(*req.UserID)
	}

	event, err := eventBuilder.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident event", "error", err)
		return nil, fmt.Errorf("failed to create incident event: %w", err)
	}

	s.logger.Infow("Incident event created successfully", "id", event.ID)
	return s.toIncidentEventResponse(event), nil
}

// CreateIncidentAlert 创建事件告警
func (s *IncidentService) CreateIncidentAlert(ctx context.Context, req *dto.CreateIncidentAlertRequest, tenantID int) (*dto.IncidentAlertResponse, error) {
	s.logger.Infow("Creating incident alert", "incident_id", req.IncidentID, "type", req.AlertType)

	triggeredAt := time.Now()
	if req.TriggeredAt != nil {
		triggeredAt = *req.TriggeredAt
	}

	alert, err := s.client.IncidentAlert.Create().
		SetIncidentID(req.IncidentID).
		SetAlertType(req.AlertType).
		SetAlertName(req.AlertName).
		SetMessage(req.Message).
		SetSeverity(req.Severity).
		SetStatus("active").
		SetChannels(req.Channels).
		SetRecipients(req.Recipients).
		SetTriggeredAt(triggeredAt).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident alert", "error", err)
		return nil, fmt.Errorf("failed to create incident alert: %w", err)
	}

	s.logger.Infow("Incident alert created successfully", "id", alert.ID)
	return s.toIncidentAlertResponse(alert), nil
}

// CreateIncidentMetric 创建事件指标
func (s *IncidentService) CreateIncidentMetric(ctx context.Context, req *dto.CreateIncidentMetricRequest, tenantID int) (*dto.IncidentMetricResponse, error) {
	s.logger.Infow("Creating incident metric", "incident_id", req.IncidentID, "type", req.MetricType)

	measuredAt := time.Now()
	if req.MeasuredAt != nil {
		measuredAt = *req.MeasuredAt
	}

	metric, err := s.client.IncidentMetric.Create().
		SetIncidentID(req.IncidentID).
		SetMetricType(req.MetricType).
		SetMetricName(req.MetricName).
		SetMetricValue(req.MetricValue).
		SetUnit(req.Unit).
		SetMeasuredAt(measuredAt).
		SetTags(req.Tags).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident metric", "error", err)
		return nil, fmt.Errorf("failed to create incident metric: %w", err)
	}

	s.logger.Infow("Incident metric created successfully", "id", metric.ID)
	return s.toIncidentMetricResponse(metric), nil
}

// GetIncidentMonitoring 获取事件监控数据
func (s *IncidentService) GetIncidentMonitoring(ctx context.Context, req *dto.IncidentMonitoringRequest, tenantID int) (*dto.IncidentMonitoringResponse, error) {
	s.logger.Infow("Getting incident monitoring data", "tenant_id", tenantID)

	// 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w", err)
	}

	query := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.CreatedAtGTE(startTime),
			incident.CreatedAtLTE(endTime),
		)

	// 应用过滤器
	if req.IncidentID != nil {
		query = query.Where(incident.IDEQ(*req.IncidentID))
	}
	if req.Category != nil {
		query = query.Where(incident.CategoryEQ(*req.Category))
	}
	if req.Priority != nil {
		query = query.Where(incident.PriorityEQ(*req.Priority))
	}
	if req.Status != nil {
		query = query.Where(incident.StatusEQ(*req.Status))
	}

	// 获取事件列表
	incidents, err := query.All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incidents", "error", err)
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}

	// 计算统计数据
	totalIncidents := len(incidents)
	var openIncidents, resolvedIncidents, closedIncidents, criticalIncidents, highPriorityIncidents int
	var totalResolutionTime float64
	var resolvedCount int

	for _, incidentEntity := range incidents {
		switch incidentEntity.Status {
		case "new", "in_progress":
			openIncidents++
		case "resolved":
			resolvedIncidents++
			if !incidentEntity.ResolvedAt.IsZero() {
				resolutionTime := incidentEntity.ResolvedAt.Sub(incidentEntity.CreatedAt).Hours()
				totalResolutionTime += resolutionTime
				resolvedCount++
			}
		case "closed":
			closedIncidents++
		}

		if incidentEntity.Severity == "critical" {
			criticalIncidents++
		}
		if incidentEntity.Priority == "high" || incidentEntity.Priority == "urgent" {
			highPriorityIncidents++
		}
	}

	// 计算平均解决时间
	var averageResolutionTime float64
	if resolvedCount > 0 {
		averageResolutionTime = totalResolutionTime / float64(resolvedCount)
	}

	// 计算解决率
	var resolutionRate float64
	if totalIncidents > 0 {
		resolutionRate = float64(resolvedIncidents+closedIncidents) / float64(totalIncidents) * 100
	}

	// 计算升级率
	var escalationRate float64
	if totalIncidents > 0 {
		var escalatedCount int
		for _, incidentEntity := range incidents {
			if incidentEntity.EscalationLevel > 0 {
				escalatedCount++
			}
		}
		escalationRate = float64(escalatedCount) / float64(totalIncidents) * 100
	}

	// 构建响应
	response := &dto.IncidentMonitoringResponse{
		TotalIncidents:        totalIncidents,
		OpenIncidents:         openIncidents,
		ResolvedIncidents:     resolvedIncidents,
		ClosedIncidents:       closedIncidents,
		CriticalIncidents:     criticalIncidents,
		HighPriorityIncidents: highPriorityIncidents,
		AverageResolutionTime: averageResolutionTime,
		ResolutionRate:        resolutionRate,
		EscalationRate:        escalationRate,
	}

	// 转换事件列表
	response.Incidents = make([]dto.IncidentResponse, len(incidents))
	for i, incidentEntity := range incidents {
		response.Incidents[i] = *s.toIncidentResponse(incidentEntity)
	}

	return response, nil
}

// EscalateIncident 升级事件
func (s *IncidentService) EscalateIncident(ctx context.Context, req *dto.IncidentEscalationRequest, tenantID int) (*dto.IncidentEscalationResponse, error) {
	s.logger.Infow("Escalating incident", "incident_id", req.IncidentID, "level", req.EscalationLevel)

	// 获取事件
	_, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(req.IncidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found")
		}
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}

	// 更新事件升级信息
	now := time.Now()
	incidentEntity, err := s.client.Incident.UpdateOneID(req.IncidentID).
		SetEscalationLevel(req.EscalationLevel).
		SetEscalatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to escalate incident", "error", err)
		return nil, fmt.Errorf("failed to escalate incident: %w", err)
	}

	// 记录升级活动
	_, err = s.CreateIncidentEvent(ctx, &dto.CreateIncidentEventRequest{
		IncidentID:  req.IncidentID,
		EventType:   "escalation",
		EventName:   "事件升级",
		Description: fmt.Sprintf("事件升级到级别 %d: %s", req.EscalationLevel, req.Reason),
		Status:      "active",
		Severity:    "high",
		Data: map[string]interface{}{
			"escalation_level": req.EscalationLevel,
			"reason":           req.Reason,
		},
		Source: "system",
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create escalation event", "error", err)
	}

	// 创建升级告警
	_, err = s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: req.IncidentID,
		AlertType:  "escalation",
		AlertName:  "事件升级告警",
		Message:    fmt.Sprintf("事件 %s 已升级到级别 %d", incidentEntity.IncidentNumber, req.EscalationLevel),
		Severity:   "high",
		Channels:   []string{"email", "sms"},
		Recipients: []string{"manager@company.com"},
		Metadata: map[string]interface{}{
			"escalation_level": req.EscalationLevel,
			"reason":           req.Reason,
		},
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create escalation alert", "error", err)
	}

	// 构建响应
	response := &dto.IncidentEscalationResponse{
		ID:              incidentEntity.ID,
		IncidentID:      req.IncidentID,
		EscalationLevel: req.EscalationLevel,
		Reason:          req.Reason,
		Status:          "active",
		NotifiedUsers:   req.NotifyUsers,
		AutoAssigned:    req.AutoAssign,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.logger.Infow("Incident escalated successfully", "incident_id", req.IncidentID, "level", req.EscalationLevel)
	return response, nil
}

// generateIncidentNumber 生成事件编号，优先使用 Redis 序列
func (s *IncidentService) generateIncidentNumber(ctx context.Context, tenantID int) (string, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	expiredAt := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	key := fmt.Sprintf("sequence:incident:%d%02d", year, month)

	// 优先使用 Redis 序列（原子递增，避免并发重复）
	if s.sequenceService != nil {
		seq, err := s.sequenceService.GetNextSequenceWithExpiry(ctx, key, expiredAt)
		if err != nil {
			s.logger.Warnw("Redis sequence failed for incident, fallback to DB", "error", err)
		} else {
			return fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq), nil
		}
	}

	// 备用方案：数据库查询
	return s.generateIncidentNumberWithDB(ctx, tenantID, year, month)
}

// generateIncidentNumberWithDB 使用数据库查询生成事件编号（备用方案）
// 修复：使用 IncidentNumberContains 过滤标准格式（INC-YYYYMM-NNNNNN），
// 避免旧格式（INC-001 等）干扰序列计算
func (s *IncidentService) generateIncidentNumberWithDB(ctx context.Context, tenantID int, year, month int) (string, error) {
	prefix := fmt.Sprintf("INC-%04d%02d-", year, month)

	incidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.IncidentNumberContains(prefix),
		).
		All(ctx)

	maxSeq := 0
	if err != nil {
		s.logger.Warnw("Query incident numbers failed, starting from 0", "error", err)
	} else {
		for _, inc := range incidents {
			num := inc.IncidentNumber
			// 解析 INC-YYYYMM-NNNNNN 格式，只取最后的数字序列
			for i := len(num) - 1; i >= 0; i-- {
				if num[i] == '-' {
					var seq int
					if _, err := fmt.Sscanf(num[i+1:], "%d", &seq); err == nil {
						if seq > maxSeq {
							maxSeq = seq
						}
					}
					break
				}
			}
		}
	}

	return fmt.Sprintf("INC-%04d%02d-%06d", year, month, maxSeq+1), nil
}

func (s *IncidentService) executeIncidentRules(ctx context.Context, incidentID int, tenantID int) {
	s.logger.Infow("Executing incident rules", "incident_id", incidentID)

	// 获取激活的事件规则
	rules, err := s.client.IncidentRule.Query().
		Where(
			incidentrule.TenantIDEQ(tenantID),
			incidentrule.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident rules", "error", err)
		return
	}

	// 获取事件信息
	incidentEntity, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get incident", "error", err)
		return
	}

	// 执行每个规则
	for _, rule := range rules {
		if s.evaluateRuleConditions(rule.Conditions, incidentEntity) {
			s.executeRuleActions(ctx, rule, incidentEntity, tenantID)
		}
	}
}

func (s *IncidentService) evaluateRuleConditions(conditions map[string]interface{}, incident *ent.Incident) bool {
	// 简化的条件评估逻辑
	if priority, ok := conditions["priority"].([]string); ok {
		for _, p := range priority {
			if incident.Priority == p {
				return true
			}
		}
	}
	if severity, ok := conditions["severity"].([]string); ok {
		for _, s := range severity {
			if incident.Severity == s {
				return true
			}
		}
	}
	if status, ok := conditions["status"].(string); ok {
		return incident.Status == status
	}
	return false
}

func (s *IncidentService) executeRuleActions(ctx context.Context, rule *ent.IncidentRule, incident *ent.Incident, tenantID int) {
	s.logger.Infow("Executing rule actions", "rule_id", rule.ID, "incident_id", incident.ID)

	// 记录规则执行
	execution, err := s.client.IncidentRuleExecution.Create().
		SetRuleID(rule.ID).
		SetIncidentID(incident.ID).
		SetStatus("running").
		SetStartedAt(time.Now()).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create rule execution", "error", err)
		return
	}

	// 执行动作
	for _, action := range rule.Actions {
		if actionType, ok := action["type"].(string); ok {
			switch actionType {
			case "escalate":
				s.executeEscalationAction(ctx, action, incident, tenantID)
			case "notify":
				s.executeNotificationAction(ctx, action, incident, tenantID)
			case "assign":
				s.executeAssignmentAction(ctx, action, incident, tenantID)
			}
		}
	}

	// 更新规则执行状态
	_, err = s.client.IncidentRuleExecution.UpdateOneID(execution.ID).
		SetStatus("completed").
		SetCompletedAt(time.Now()).
		SetResult("Rule executed successfully").
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update rule execution", "error", err)
	}

	// 更新规则执行次数
	_, err = s.client.IncidentRule.UpdateOneID(rule.ID).
		SetExecutionCount(rule.ExecutionCount + 1).
		SetLastExecutedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update rule execution count", "error", err)
	}
}

func (s *IncidentService) executeEscalationAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	if level, ok := action["level"].(int); ok {
		_, err := s.EscalateIncident(ctx, &dto.IncidentEscalationRequest{
			IncidentID:      incident.ID,
			EscalationLevel: level,
			Reason:          "自动升级",
			NotifyUsers:     []int{},
			AutoAssign:      false,
		}, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to execute escalation action", "error", err)
		}
	}
}

func (s *IncidentService) executeNotificationAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	channels := []string{"email"}
	if ch, ok := action["channels"].([]string); ok {
		channels = ch
	}

	recipients := []string{"admin@company.com"}
	if rec, ok := action["recipients"].([]string); ok {
		recipients = rec
	}

	message := "事件需要关注"
	if msg, ok := action["message"].(string); ok {
		message = msg
	}

	_, err := s.CreateIncidentAlert(ctx, &dto.CreateIncidentAlertRequest{
		IncidentID: incident.ID,
		AlertType:  "notification",
		AlertName:  "规则触发通知",
		Message:    message,
		Severity:   "medium",
		Channels:   channels,
		Recipients: recipients,
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to execute notification action", "error", err)
	}
}

func (s *IncidentService) executeAssignmentAction(ctx context.Context, action map[string]interface{}, incident *ent.Incident, tenantID int) {
	if assigneeID, ok := action["assignee_id"].(int); ok {
		_, err := s.UpdateIncident(ctx, incident.ID, &dto.UpdateIncidentRequest{
			AssigneeID: &assigneeID,
		}, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to execute assignment action", "error", err)
		}
	}
}

// isValidIncidentStatusTransition 检查事件状态转换是否合法
// Incident状态转换规则:
// new -> acknowledged, assigned, in_progress
// acknowledged -> in_progress, closed
// assigned -> in_progress, escalated, closed
// in_progress -> resolved, escalated, pending, closed
// escalated -> in_progress, closed
// resolved -> closed, in_progress (reopen)
// closed -> (不允许转换到其他状态)
func isValidIncidentStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		common.IncidentStatusNew:          {common.IncidentStatusAcknowledged, common.IncidentStatusAssigned, common.IncidentStatusInProgress},
		common.IncidentStatusAcknowledged: {common.IncidentStatusInProgress, common.IncidentStatusClosed},
		common.IncidentStatusAssigned:     {common.IncidentStatusInProgress, common.IncidentStatusEscalated, common.IncidentStatusClosed},
		common.IncidentStatusInProgress:   {common.IncidentStatusResolved, common.IncidentStatusEscalated, common.IncidentStatusClosed},
		common.IncidentStatusTriaged:      {common.IncidentStatusInProgress, common.IncidentStatusEscalated, common.IncidentStatusClosed},
		common.IncidentStatusEscalated:    {common.IncidentStatusInProgress, common.IncidentStatusClosed},
		common.IncidentStatusResolved:     {common.IncidentStatusClosed, common.IncidentStatusInProgress}, // 重新打开
		common.IncidentStatusClosed:       {},                                                             // 已关闭不允许转换
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		// 未知状态，允许转换（保守策略）
		return true
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

// 转换为响应DTO
func (s *IncidentService) toIncidentResponse(incident *ent.Incident) *dto.IncidentResponse {
	var impactAnalysis *dto.ImpactAnalysis
	if incident.ImpactAnalysis != nil {
		impactAnalysis = &dto.ImpactAnalysis{}
		dto.MapToStruct(incident.ImpactAnalysis, impactAnalysis)
	}

	var rootCause *dto.RootCause
	if incident.RootCause != nil {
		rootCause = &dto.RootCause{}
		dto.MapToStruct(incident.RootCause, rootCause)
	}

	var resolutionSteps []dto.ResolutionStep
	if incident.ResolutionSteps != nil {
		dto.MapSliceToStructSlice(incident.ResolutionSteps, &resolutionSteps)
	}

	return &dto.IncidentResponse{
		ID:                  incident.ID,
		Title:               incident.Title,
		Description:         incident.Description,
		Status:              incident.Status,
		Priority:            incident.Priority,
		Severity:            incident.Severity,
		Impact:              incident.Impact,
		Urgency:             incident.Urgency,
		IncidentNumber:      incident.IncidentNumber,
		ReporterID:          incident.ReporterID,
		AssigneeID:          &incident.AssigneeID,
		ConfigurationItemID: &incident.ConfigurationItemID,
		Category:            incident.Category,
		Subcategory:         incident.Subcategory,
		ImpactAnalysis:      impactAnalysis,
		RootCause:           rootCause,
		ResolutionSteps:     resolutionSteps,
		Version:             incident.Version,
		DetectedAt:          incident.DetectedAt,
		ResolvedAt:          &incident.ResolvedAt,
		ClosedAt:            &incident.ClosedAt,
		EscalatedAt:         &incident.EscalatedAt,
		EscalationLevel:     incident.EscalationLevel,
		IsAutomated:         incident.IsAutomated,
		IsMajorIncident:     incident.IsMajorIncident,
		Source:              incident.Source,
		Metadata:            incident.Metadata,
		TenantID:            incident.TenantID,
		CreatedAt:           incident.CreatedAt,
		UpdatedAt:           incident.UpdatedAt,
	}
}

func (s *IncidentService) toIncidentEventResponse(event *ent.IncidentEvent) *dto.IncidentEventResponse {
	return &dto.IncidentEventResponse{
		ID:          event.ID,
		IncidentID:  event.IncidentID,
		EventType:   event.EventType,
		EventName:   event.EventName,
		Description: event.Description,
		Status:      event.Status,
		Severity:    event.Severity,
		Data:        event.Data,
		OccurredAt:  event.OccurredAt,
		UserID:      &event.UserID,
		Source:      event.Source,
		Metadata:    event.Metadata,
		TenantID:    event.TenantID,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
}

func (s *IncidentService) toIncidentAlertResponse(alert *ent.IncidentAlert) *dto.IncidentAlertResponse {
	return &dto.IncidentAlertResponse{
		ID:             alert.ID,
		IncidentID:     alert.IncidentID,
		AlertType:      alert.AlertType,
		AlertName:      alert.AlertName,
		Message:        alert.Message,
		Severity:       alert.Severity,
		Status:         alert.Status,
		Channels:       alert.Channels,
		Recipients:     alert.Recipients,
		TriggeredAt:    alert.TriggeredAt,
		AcknowledgedAt: &alert.AcknowledgedAt,
		ResolvedAt:     &alert.ResolvedAt,
		AcknowledgedBy: &alert.AcknowledgedBy,
		Metadata:       alert.Metadata,
		TenantID:       alert.TenantID,
		CreatedAt:      alert.CreatedAt,
		UpdatedAt:      alert.UpdatedAt,
	}
}

func (s *IncidentService) toIncidentMetricResponse(metric *ent.IncidentMetric) *dto.IncidentMetricResponse {
	return &dto.IncidentMetricResponse{
		ID:          metric.ID,
		IncidentID:  metric.IncidentID,
		MetricType:  metric.MetricType,
		MetricName:  metric.MetricName,
		MetricValue: metric.MetricValue,
		Unit:        metric.Unit,
		MeasuredAt:  metric.MeasuredAt,
		Tags:        metric.Tags,
		Metadata:    metric.Metadata,
		TenantID:    metric.TenantID,
		CreatedAt:   metric.CreatedAt,
		UpdatedAt:   metric.UpdatedAt,
	}
}

// GetIncidentStats 获取事件统计信息

// AcknowledgeIncident 流转事件状态到 acknowledged
func (s *IncidentService) AcknowledgeIncident(ctx context.Context, id, userID, tenantID int) error {
	now := time.Now()
	return s.client.Incident.UpdateOneID(id).
		SetStatus(common.IncidentStatusAcknowledged).
		SetUpdatedAt(now).
		Exec(ctx)
}

// ResolveIncident 流转事件状态到 resolved
func (s *IncidentService) ResolveIncident(ctx context.Context, id, userID, tenantID int, resolution, rootCause string) error {
	now := time.Now()
	return s.client.Incident.UpdateOneID(id).
		SetStatus(common.IncidentStatusResolved).
		SetUpdatedAt(now).
		Exec(ctx)
}

// CloseIncident 流转事件状态到 closed
func (s *IncidentService) CloseIncident(ctx context.Context, id, userID, tenantID int, closeNotes string) error {
	now := time.Now()
	return s.client.Incident.UpdateOneID(id).
		SetStatus(common.IncidentStatusClosed).
		SetUpdatedAt(now).
		Exec(ctx)
}

func (s *IncidentService) GetIncidentStats(ctx context.Context, tenantID int) (*dto.IncidentStatsResponse, error) {
	s.logger.Infow("Getting incident stats", "tenant_id", tenantID)

	// 获取总事件数
	totalIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total incidents", "error", err)
		return nil, fmt.Errorf("failed to count total incidents: %w", err)
	}

	// 获取开放事件数（new, in_progress）
	openIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID), incident.StatusIn("new", "in_progress")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count open incidents", "error", err)
		return nil, fmt.Errorf("failed to count open incidents: %w", err)
	}

	// 获取关键事件数（severity = critical）
	criticalIncidents, err := s.client.Incident.Query().
		Where(incident.TenantIDEQ(tenantID), incident.SeverityEQ("critical")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count critical incidents", "error", err)
		return nil, fmt.Errorf("failed to count critical incidents: %w", err)
	}

	// 获取主要事件数（使用 severity = critical 或 priority = high/urgent 作为主要事件）
	majorIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.Or(
				incident.SeverityEQ("critical"),
				incident.PriorityIn("high", "urgent"),
			),
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count major incidents", "error", err)
		return nil, fmt.Errorf("failed to count major incidents: %w", err)
	}

	// 获取已解决的事件，计算平均解决时间
	resolvedIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.StatusEQ("resolved"),
			incident.ResolvedAtNotNil(),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get resolved incidents", "error", err)
		return nil, fmt.Errorf("failed to get resolved incidents: %w", err)
	}

	var totalResolutionTime float64
	var totalAcknowledgeTime float64
	resolvedCount := len(resolvedIncidents)
	acknowledgedCount := 0

	for _, inc := range resolvedIncidents {
		if !inc.ResolvedAt.IsZero() && !inc.DetectedAt.IsZero() {
			resolutionTime := inc.ResolvedAt.Sub(inc.DetectedAt).Hours()
			totalResolutionTime += resolutionTime
		}
		// 使用 detected_at 到 created_at 的时间差作为确认时间（简化实现）
		if !inc.DetectedAt.IsZero() && !inc.CreatedAt.IsZero() {
			acknowledgeTime := inc.DetectedAt.Sub(inc.CreatedAt).Hours()
			if acknowledgeTime > 0 {
				totalAcknowledgeTime += acknowledgeTime
				acknowledgedCount++
			}
		}
	}

	var avgResolutionTime float64
	if resolvedCount > 0 {
		avgResolutionTime = totalResolutionTime / float64(resolvedCount)
	}

	var mtta float64
	if acknowledgedCount > 0 {
		mtta = totalAcknowledgeTime / float64(acknowledgedCount)
	}

	var mttr float64 = avgResolutionTime

	return &dto.IncidentStatsResponse{
		TotalIncidents:    totalIncidents,
		OpenIncidents:     openIncidents,
		CriticalIncidents: criticalIncidents,
		MajorIncidents:    majorIncidents,
		AvgResolutionTime: avgResolutionTime,
		MTTA:              mtta,
		MTTR:              mttr,
	}, nil
}

// GetIncidentEvents 获取指定事件的活动记录
func (s *IncidentService) GetIncidentEvents(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentEventResponse, error) {
	s.logger.Infow("Getting incident events", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的活动记录
	events, err := s.client.IncidentEvent.Query().
		Where(
			incidentevent.IncidentIDEQ(incident.ID),
			incidentevent.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident events: %w", err)
	}

	responses := make([]dto.IncidentEventResponse, len(events))
	for i, event := range events {
		responses[i] = *s.toIncidentEventResponse(event)
	}

	return responses, nil
}

// GetIncidentAlerts 获取指定事件的告警
func (s *IncidentService) GetIncidentAlerts(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentAlertResponse, error) {
	s.logger.Infow("Getting incident alerts", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的告警
	alerts, err := s.client.IncidentAlert.Query().
		Where(
			incidentalert.IncidentIDEQ(incident.ID),
			incidentalert.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident alerts: %w", err)
	}

	responses := make([]dto.IncidentAlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = *s.toIncidentAlertResponse(alert)
	}

	return responses, nil
}

// GetIncidentMetrics 获取指定事件的指标
func (s *IncidentService) GetIncidentMetrics(ctx context.Context, incidentID int, tenantID int) ([]dto.IncidentMetricResponse, error) {
	s.logger.Infow("Getting incident metrics", "incident_id", incidentID, "tenant_id", tenantID)

	// 验证事件是否存在且属于该租户
	incident, err := s.client.Incident.Query().
		Where(
			incident.ID(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("incident not found or not accessible")
		}
		return nil, fmt.Errorf("failed to verify incident: %w", err)
	}

	// 获取事件的指标
	metrics, err := s.client.IncidentMetric.Query().
		Where(
			incidentmetric.IncidentIDEQ(incident.ID),
			incidentmetric.TenantIDEQ(tenantID),
		).
		Order(ent.Desc("created_at")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident metrics: %w", err)
	}

	responses := make([]dto.IncidentMetricResponse, len(metrics))
	for i, metric := range metrics {
		responses[i] = *s.toIncidentMetricResponse(metric)
	}

	return responses, nil
}

// triggerWorkflowForIncident 为事件触发工作流
func (s *IncidentService) triggerWorkflowForIncident(ctx context.Context, incidentID int, tenantID int) error {
	// 获取事件信息
	inc, err := s.client.Incident.Query().
		Where(
			incident.IDEQ(incidentID),
			incident.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get incident: %w", err)
	}

	// 构建流程变量
	variables := map[string]interface{}{
		"incident_id":     inc.ID,
		"incident_number": inc.IncidentNumber,
		"title":           inc.Title,
		"description":     inc.Description,
		"priority":        inc.Priority,
		"severity":        inc.Severity,
		"status":          inc.Status,
		"category":        inc.Category,
		"reporter_id":     inc.ReporterID,
		"assignee_id":     inc.AssigneeID,
	}

	// 根据严重程度选择不同的流程
	// 注意: incident_general_flow 不存在，使用 incident_emergency_flow 作为默认
	processKey := "incident_emergency_flow"
	if inc.Severity == "critical" || inc.Priority == "urgent" {
		processKey = "incident_emergency_flow"
	}

	// 触发流程
	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeIncident,
		BusinessID:           incidentID,
		ProcessDefinitionKey: processKey,
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", inc.ReporterID),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerService.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow(
		"Workflow triggered for incident",
		"incident_id", incidentID,
		"process_instance_id", resp.ProcessInstanceID,
		"process_key", processKey,
	)

	return nil
}

// GetWorkflowStatus 获取事件关联的流程状态
func (s *IncidentService) GetWorkflowStatus(ctx context.Context, incidentID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	businessKey := fmt.Sprintf("incident:%d", incidentID)

	// 直接查询流程实例
	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("未找到事件关联的流程实例")
		}
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	processDefName := ""
	if processInstance.Edges.Definition != nil {
		processDefName = processInstance.Edges.Definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     processInstance.ID,
		ProcessDefinitionKey:  processInstance.ProcessDefinitionKey,
		ProcessDefinitionName: processDefName,
		BusinessKey:           processInstance.BusinessKey,
		Status:                s.mapProcessStatus(processInstance.Status),
		CurrentActivityID:     processInstance.CurrentActivityID,
		CurrentActivityName:   processInstance.CurrentActivityName,
		StartTime:             processInstance.StartTime,
		EndTime:               &processInstance.EndTime,
	}, nil
}

// mapProcessStatus 映射流程状态
func (s *IncidentService) mapProcessStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}
