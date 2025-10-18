package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentalert"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentmetric"
	"itsm-backend/ent/incidentrule"
	"itsm-backend/ent/incidentruleexecution"

	"go.uber.org/zap"
)

type IncidentService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewIncidentService(client *ent.Client, logger *zap.SugaredLogger) *IncidentService {
	return &IncidentService{
		client: client,
		logger: logger,
	}
}

// CreateIncident 创建事件
func (s *IncidentService) CreateIncident(ctx context.Context, req *dto.CreateIncidentRequest, tenantID int) (*dto.IncidentResponse, error) {
	s.logger.Infow("Creating incident", "title", req.Title, "tenant_id", tenantID)

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

	incidentEntity, err := s.client.Incident.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetStatus("new").
		SetPriority(req.Priority).
		SetSeverity(req.Severity).
		SetIncidentNumber(incidentNumber).
		SetReporterID(1). // 假设当前用户ID为1
		SetCategory(req.Category).
		SetSubcategory(req.Subcategory).
		SetImpactAnalysis(req.ImpactAnalysis).
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
		if *req.Status == "resolved" {
			now := time.Now()
			updateQuery.SetResolvedAt(now)
		}
		// 如果状态变更为closed，设置关闭时间
		if *req.Status == "closed" {
			now := time.Now()
			updateQuery.SetClosedAt(now)
		}
	}
	if req.Priority != nil {
		updateQuery.SetPriority(*req.Priority)
	}
	if req.Severity != nil {
		updateQuery.SetSeverity(*req.Severity)
	}
	if req.Category != nil {
		updateQuery.SetCategory(*req.Category)
	}
	if req.Subcategory != nil {
		updateQuery.SetSubcategory(*req.Subcategory)
	}
	if req.AssigneeID != nil {
		updateQuery.SetAssigneeID(*req.AssigneeID)
	}
	if req.ImpactAnalysis != nil {
		updateQuery.SetImpactAnalysis(req.ImpactAnalysis)
	}
	if req.RootCause != nil {
		updateQuery.SetRootCause(req.RootCause)
	}
	if req.ResolutionSteps != nil {
		updateQuery.SetResolutionSteps(req.ResolutionSteps)
	}
	if req.Metadata != nil {
		updateQuery.SetMetadata(req.Metadata)
	}

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

// DeleteIncident 删除事件
func (s *IncidentService) DeleteIncident(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting incident", "id", id, "tenant_id", tenantID)

	err := s.client.Incident.DeleteOneID(id).
		Where(incident.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("incident not found")
		}
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

	event, err := s.client.IncidentEvent.Create().
		SetIncidentID(req.IncidentID).
		SetEventType(req.EventType).
		SetEventName(req.EventName).
		SetDescription(req.Description).
		SetStatus(req.Status).
		SetSeverity(req.Severity).
		SetData(req.Data).
		SetOccurredAt(occurredAt).
		SetUserID(req.UserID).
		SetSource(req.Source).
		SetMetadata(req.Metadata).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
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
			if incidentEntity.ResolvedAt != nil {
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
	incidentEntity, err := s.client.Incident.Query().
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
	incidentEntity, err = s.client.Incident.UpdateOneID(req.IncidentID).
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
			"reason":          req.Reason,
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
			"reason":          req.Reason,
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

// 辅助方法
func (s *IncidentService) generateIncidentNumber(ctx context.Context, tenantID int) (string, error) {
	// 获取当前年份和月份
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 查询当月的事件数量
	count, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.CreatedAtGTE(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)),
			incident.CreatedAtLT(time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to count incidents: %w", err)
	}

	// 生成事件编号
	incidentNumber := fmt.Sprintf("INC-%04d%02d-%06d", year, month, count+1)
	return incidentNumber, nil
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
		Where(incident.IDEQ(incidentID)).
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

// 转换为响应DTO
func (s *IncidentService) toIncidentResponse(incident *ent.Incident) *dto.IncidentResponse {
	return &dto.IncidentResponse{
		ID:                  incident.ID,
		Title:               incident.Title,
		Description:        incident.Description,
		Status:              incident.Status,
		Priority:            incident.Priority,
		Severity:            incident.Severity,
		IncidentNumber:      incident.IncidentNumber,
		ReporterID:          incident.ReporterID,
		AssigneeID:          incident.AssigneeID,
		ConfigurationItemID: incident.ConfigurationItemID,
		Category:            incident.Category,
		Subcategory:         incident.Subcategory,
		ImpactAnalysis:      incident.ImpactAnalysis,
		RootCause:           incident.RootCause,
		ResolutionSteps:     incident.ResolutionSteps,
		DetectedAt:          incident.DetectedAt,
		ResolvedAt:          incident.ResolvedAt,
		ClosedAt:            incident.ClosedAt,
		EscalatedAt:         incident.EscalatedAt,
		EscalationLevel:     incident.EscalationLevel,
		IsAutomated:         incident.IsAutomated,
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
		UserID:      event.UserID,
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
		AcknowledgedAt: alert.AcknowledgedAt,
		ResolvedAt:     alert.ResolvedAt,
		AcknowledgedBy: alert.AcknowledgedBy,
		Metadata:       alert.Metadata,
		TenantID:       alert.TenantID,
		CreatedAt:     alert.CreatedAt,
		UpdatedAt:     alert.UpdatedAt,
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