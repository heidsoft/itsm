package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"

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
func (s *IncidentService) CreateIncident(ctx context.Context, req *dto.CreateIncidentRequest, reporterID int, tenantID int) (*ent.Incident, error) {
	// 生成事件编号
	incidentNumber := s.generateIncidentNumber()

	// 创建事件
	create := s.client.Incident.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(incident.Priority(req.Priority)).
		SetSource(incident.Source(req.Source)).
		SetType(incident.Type(req.Type)).
		SetIncidentNumber(incidentNumber).
		SetIsMajorIncident(req.IsMajor).
		SetReporterID(reporterID).
		SetTenantID(tenantID).
		SetDetectedAt(time.Now())

	if req.AssigneeID != nil {
		create = create.SetAssigneeID(*req.AssigneeID)
	}

	// 设置阿里云相关字段
	if req.AlibabaCloudInstanceID != "" {
		create = create.SetAlibabaCloudInstanceID(req.AlibabaCloudInstanceID)
	}
	if req.AlibabaCloudRegion != "" {
		create = create.SetAlibabaCloudRegion(req.AlibabaCloudRegion)
	}
	if req.AlibabaCloudService != "" {
		create = create.SetAlibabaCloudService(req.AlibabaCloudService)
	}
	if req.AlibabaCloudAlertData != nil {
		create = create.SetAlibabaCloudAlertData(req.AlibabaCloudAlertData)
	}
	if req.AlibabaCloudMetrics != nil {
		create = create.SetAlibabaCloudMetrics(req.AlibabaCloudMetrics)
	}

	// 设置安全事件相关字段
	if req.SecurityEventType != "" {
		create = create.SetSecurityEventType(req.SecurityEventType)
	}
	if req.SecurityEventSourceIP != "" {
		create = create.SetSecurityEventSourceIP(req.SecurityEventSourceIP)
	}
	if req.SecurityEventTarget != "" {
		create = create.SetSecurityEventTarget(req.SecurityEventTarget)
	}
	if req.SecurityEventDetails != nil {
		create = create.SetSecurityEventDetails(req.SecurityEventDetails)
	}

	incidentEntity, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create incident", "error", err)
		return nil, fmt.Errorf("创建事件失败: %w", err)
	}

	// 关联配置项
	if len(req.AffectedConfigurationItemIDs) > 0 {
		err = s.associateConfigurationItems(ctx, incidentEntity.ID, req.AffectedConfigurationItemIDs)
		if err != nil {
			s.logger.Errorw("Failed to associate configuration items", "error", err)
		}
	}

	s.logger.Infow("Incident created successfully", "incident_id", incidentEntity.ID, "reporter_id", reporterID)
	return incidentEntity, nil
}

// CreateIncidentFromAlibabaCloudAlert 从阿里云告警创建事件
func (s *IncidentService) CreateIncidentFromAlibabaCloudAlert(ctx context.Context, req *dto.AlibabaCloudAlertRequest, reporterID int, tenantID int) (*ent.Incident, error) {
	// 根据告警级别确定优先级
	priority := s.mapAlertLevelToPriority(req.AlertLevel)

	// 确定事件类型
	incidentType := s.mapServiceToIncidentType(req.Service)

	createReq := &dto.CreateIncidentRequest{
		Title:                  req.AlertName,
		Description:            req.AlertDescription,
		Priority:               priority,
		Source:                 "alibaba_cloud",
		Type:                   incidentType,
		IsMajor:                req.AlertLevel == "critical",
		AlibabaCloudInstanceID: req.InstanceID,
		AlibabaCloudRegion:     req.Region,
		AlibabaCloudService:    req.Service,
		AlibabaCloudAlertData:  req.AlertData,
		AlibabaCloudMetrics:    req.Metrics,
	}

	return s.CreateIncident(ctx, createReq, reporterID, tenantID)
}

// CreateIncidentFromSecurityEvent 从安全事件创建事件
func (s *IncidentService) CreateIncidentFromSecurityEvent(ctx context.Context, req *dto.SecurityEventRequest, reporterID int, tenantID int) (*ent.Incident, error) {
	// 根据严重程度确定优先级
	priority := s.mapSeverityToPriority(req.Severity)

	createReq := &dto.CreateIncidentRequest{
		Title:                 req.EventName,
		Description:           req.EventDescription,
		Priority:              priority,
		Source:                "security_event",
		Type:                  "security",
		IsMajor:               req.Severity == "critical" || req.Severity == "high",
		SecurityEventType:     req.EventType,
		SecurityEventSourceIP: req.SourceIP,
		SecurityEventTarget:   req.Target,
		SecurityEventDetails:  req.EventDetails,
	}

	return s.CreateIncident(ctx, createReq, reporterID, tenantID)
}

// CreateIncidentFromCloudProductEvent 从云产品事件创建事件
func (s *IncidentService) CreateIncidentFromCloudProductEvent(ctx context.Context, req *dto.CloudProductEventRequest, reporterID int, tenantID int) (*ent.Incident, error) {
	// 根据产品类型确定事件类型
	incidentType := s.mapProductToIncidentType(req.Product)

	createReq := &dto.CreateIncidentRequest{
		Title:                  req.EventName,
		Description:            req.EventDescription,
		Priority:               "medium", // 默认中等优先级
		Source:                 "cloud_product",
		Type:                   incidentType,
		IsMajor:                false,
		AlibabaCloudInstanceID: req.InstanceID,
		AlibabaCloudRegion:     req.Region,
		AlibabaCloudService:    req.Product,
		AlibabaCloudAlertData:  req.EventData,
	}

	return s.CreateIncident(ctx, createReq, reporterID, tenantID)
}

// GetIncidentByID 根据ID获取事件详情
func (s *IncidentService) GetIncidentByID(ctx context.Context, id int) (*ent.Incident, error) {
	incidentEntity, err := s.client.Incident.Query().
		Where(incident.ID(id)).
		WithReporter().
		WithAssignee().
		WithAffectedConfigurationItems().
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("事件不存在")
		}
		s.logger.Errorw("Failed to get incident", "incident_id", id, "error", err)
		return nil, fmt.Errorf("获取事件失败: %w", err)
	}

	return incidentEntity, nil
}

// ListIncidents 获取事件列表
func (s *IncidentService) ListIncidents(ctx context.Context, req *dto.ListIncidentsRequest, tenantID int) (*dto.ListIncidentsResponse, error) {
	query := s.client.Incident.Query().Where(incident.TenantID(tenantID))

	// 应用筛选条件
	if req.Status != "" {
		query = query.Where(incident.StatusEQ(incident.Status(req.Status)))
	}
	if req.Priority != "" {
		query = query.Where(incident.PriorityEQ(incident.Priority(req.Priority)))
	}
	if req.Source != "" {
		query = query.Where(incident.SourceEQ(incident.Source(req.Source)))
	}
	if req.Type != "" {
		query = query.Where(incident.TypeEQ(incident.Type(req.Type)))
	}
	if req.AssigneeID > 0 {
		query = query.Where(incident.AssigneeID(req.AssigneeID))
	}
	if req.IsMajor != nil {
		query = query.Where(incident.IsMajorIncident(*req.IsMajor))
	}
	if req.Keyword != "" {
		query = query.Where(incident.Or(
			incident.TitleContains(req.Keyword),
			incident.DescriptionContains(req.Keyword),
			incident.IncidentNumberContains(req.Keyword),
		))
	}

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事件总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	incidents, err := query.
		WithReporter().
		WithAssignee().
		Limit(req.PageSize).
		Offset(offset).
		Order(ent.Desc(incident.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	return &dto.ListIncidentsResponse{
		Incidents: s.convertToIncidentResponses(incidents),
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// UpdateIncident 更新事件
func (s *IncidentService) UpdateIncident(ctx context.Context, id int, req *dto.UpdateIncidentRequest) (*ent.Incident, error) {
	update := s.client.Incident.UpdateOneID(id)

	if req.Title != nil {
		update = update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.Priority != nil {
		update = update.SetPriority(incident.Priority(*req.Priority))
	}
	if req.Type != nil {
		update = update.SetType(incident.Type(*req.Type))
	}
	if req.AssigneeID != nil {
		update = update.SetAssigneeID(*req.AssigneeID)
	}
	if req.IsMajor != nil {
		update = update.SetIsMajorIncident(*req.IsMajor)
	}

	incidentEntity, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update incident", "incident_id", id, "error", err)
		return nil, fmt.Errorf("更新事件失败: %w", err)
	}

	return incidentEntity, nil
}

// UpdateIncidentStatus 更新事件状态
func (s *IncidentService) UpdateIncidentStatus(ctx context.Context, id int, req *dto.UpdateIncidentStatusRequest) (*ent.Incident, error) {
	update := s.client.Incident.UpdateOneID(id).SetStatus(incident.Status(req.Status))

	// 根据状态设置相应的时间字段
	switch req.Status {
	case "assigned":
		update = update.SetConfirmedAt(time.Now())
	case "resolved":
		update = update.SetResolvedAt(time.Now())
	case "closed":
		update = update.SetClosedAt(time.Now())
	}

	incidentEntity, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update incident status", "incident_id", id, "error", err)
		return nil, fmt.Errorf("更新事件状态失败: %w", err)
	}

	// 记录状态变更日志
	s.logStatusChange(ctx, id, req.Status, req.ResolutionNote, req.SuspendReason)

	return incidentEntity, nil
}

// GetIncidentMetrics 获取事件指标
func (s *IncidentService) GetIncidentMetrics(ctx context.Context, tenantID int) (*dto.IncidentManagementMetrics, error) {
	// 获取总事件数
	totalIncidents, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取开放事件数
	openIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.StatusIn(incident.StatusNew, incident.StatusAssigned, incident.StatusInProgress),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取紧急事件数
	criticalIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.PriorityEQ(incident.PriorityCritical),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取重大事件数
	majorIncidents, err := s.client.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.IsMajorIncident(true),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.IncidentManagementMetrics{
		TotalIncidents:    totalIncidents,
		OpenIncidents:     openIncidents,
		CriticalIncidents: criticalIncidents,
		MajorIncidents:    majorIncidents,
		AvgResolutionTime: 4.5, // 模拟数据：4.5小时
		MTTA:              0.5, // 模拟数据：30分钟
		MTTR:              4.0, // 模拟数据：4小时
	}, nil
}

// 辅助方法

func (s *IncidentService) generateIncidentNumber() string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("INC-%d", timestamp)
}

func (s *IncidentService) associateConfigurationItems(ctx context.Context, incidentID int, ciIDs []int) error {
	incidentEntity, err := s.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return err
	}

	// 获取配置项
	configurationItems, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDIn(ciIDs...)).
		All(ctx)
	if err != nil {
		return err
	}

	// 建立关联
	return incidentEntity.Update().
		AddAffectedConfigurationItems(configurationItems...).
		Exec(ctx)
}

func (s *IncidentService) mapAlertLevelToPriority(alertLevel string) string {
	switch alertLevel {
	case "critical":
		return "critical"
	case "warning":
		return "high"
	case "info":
		return "medium"
	default:
		return "medium"
	}
}

func (s *IncidentService) mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func (s *IncidentService) mapServiceToIncidentType(service string) string {
	switch service {
	case "ecs", "rds", "slb":
		return "infrastructure"
	case "cdn", "vpc", "nat":
		return "network"
	case "redis", "mongodb":
		return "database"
	case "oss", "nas":
		return "storage"
	default:
		return "cloud_service"
	}
}

func (s *IncidentService) mapProductToIncidentType(product string) string {
	switch product {
	case "ecs", "rds", "slb":
		return "infrastructure"
	case "cdn", "vpc", "nat":
		return "network"
	case "redis", "mongodb":
		return "database"
	case "oss", "nas":
		return "storage"
	default:
		return "cloud_service"
	}
}

func (s *IncidentService) logStatusChange(ctx context.Context, incidentID int, status, resolutionNote, suspendReason string) {
	// 这里可以记录状态变更日志到数据库
	s.logger.Infow("Incident status changed",
		"incident_id", incidentID,
		"status", status,
		"resolution_note", resolutionNote,
		"suspend_reason", suspendReason)
}

func (s *IncidentService) convertToIncidentResponses(incidents []*ent.Incident) []dto.IncidentResponse {
	responses := make([]dto.IncidentResponse, len(incidents))
	for i, incident := range incidents {
		responses[i] = dto.IncidentResponse{
			ID:              incident.ID,
			Title:           incident.Title,
			Description:     incident.Description,
			Status:          string(incident.Status),
			Priority:        string(incident.Priority),
			Source:          string(incident.Source),
			Type:            string(incident.Type),
			IncidentNumber:  incident.IncidentNumber,
			IsMajorIncident: incident.IsMajorIncident,
			DetectedAt:      &incident.DetectedAt,
			ConfirmedAt:     incident.ConfirmedAt,
			ResolvedAt:      incident.ResolvedAt,
			ClosedAt:        incident.ClosedAt,
			CreatedAt:       incident.CreatedAt,
			UpdatedAt:       incident.UpdatedAt,
		}

		// 设置报告人信息
		if incident.Edges.Reporter != nil {
			responses[i].Reporter = &dto.UserResponse{
				ID:   incident.Edges.Reporter.ID,
				Name: incident.Edges.Reporter.Name,
			}
		}

		// 设置处理人信息
		if incident.Edges.Assignee != nil {
			responses[i].Assignee = &dto.UserResponse{
				ID:   incident.Edges.Assignee.ID,
				Name: incident.Edges.Assignee.Name,
			}
		}
	}
	return responses
}
