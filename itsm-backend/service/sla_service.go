package service

import (
	"context"
	"time"
	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
)

// SLAService SLA服务
type SLAService struct {
	client *ent.Client
}

// NewSLAService 创建SLA服务实例
func NewSLAService(client *ent.Client) *SLAService {
	return &SLAService{client: client}
}

// SLADefinition SLA定义
type SLADefinition struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	ServiceType     string    `json:"service_type"`
	Priority        string    `json:"priority"`
	ResponseTime    int       `json:"response_time"`
	ResolutionTime  int       `json:"resolution_time"`
	BusinessHours   map[string]interface{} `json:"business_hours"`
	IsActive        bool      `json:"is_active"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SLAViolation SLA违规记录
type SLAViolation struct {
	ID              int       `json:"id"`
	TicketID        int       `json:"ticket_id"`
	SLAID           int       `json:"sla_id"`
	ViolationType   string    `json:"violation_type"`
	ViolationTime   time.Time `json:"violation_time"`
	BreachDuration  int       `json:"breach_duration"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
	AssigneeID      int       `json:"assignee_id"`
	Notes           string    `json:"notes"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateSLADefinition 创建SLA定义
func (s *SLAService) CreateSLADefinition(ctx context.Context, sla *SLADefinition) error {
	_, err := s.client.SLADefinition.Create().
		SetName(sla.Name).
		SetDescription(sla.Description).
		SetServiceType(sla.ServiceType).
		SetPriority(sla.Priority).
		SetResponseTime(sla.ResponseTime).
		SetResolutionTime(sla.ResolutionTime).
		SetBusinessHours(sla.BusinessHours).
		SetIsActive(sla.IsActive).
		SetTenantID(sla.TenantID).
		Save(ctx)
	return err
}

// GetSLADefinition 获取SLA定义
func (s *SLAService) GetSLADefinition(ctx context.Context, id int) (*SLADefinition, error) {
	sla, err := s.client.SLADefinition.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &SLADefinition{
		ID:             sla.ID,
		Name:           sla.Name,
		Description:    sla.Description,
		ServiceType:    sla.ServiceType,
		Priority:       sla.Priority,
		ResponseTime:   sla.ResponseTime,
		ResolutionTime: sla.ResolutionTime,
		BusinessHours:  sla.BusinessHours,
		IsActive:       sla.IsActive,
		TenantID:       sla.TenantID,
		CreatedAt:      sla.CreatedAt,
		UpdatedAt:      sla.UpdatedAt,
	}, nil
}

// UpdateSLADefinition 更新SLA定义
func (s *SLAService) UpdateSLADefinition(ctx context.Context, id int, updates map[string]interface{}) error {
	update := s.client.SLADefinition.UpdateOneID(id)
	
	if name, ok := updates["name"].(string); ok {
		update.SetName(name)
	}
	if description, ok := updates["description"].(string); ok {
		update.SetDescription(description)
	}
	if responseTime, ok := updates["response_time"].(int); ok {
		update.SetResponseTime(responseTime)
	}
	if resolutionTime, ok := updates["resolution_time"].(int); ok {
		update.SetResolutionTime(resolutionTime)
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		update.SetIsActive(isActive)
	}

	_, err := update.Save(ctx)
	return err
}

// DeleteSLADefinition 删除SLA定义
func (s *SLAService) DeleteSLADefinition(ctx context.Context, id int) error {
	return s.client.SLADefinition.DeleteOneID(id).Exec(ctx)
}

// ListSLADefinitions 获取SLA定义列表
func (s *SLAService) ListSLADefinitions(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLADefinition, error) {
	query := s.client.SLADefinition.Query().Where(sladefinition.TenantID(tenantID))

	if serviceType, ok := filters["service_type"].(string); ok && serviceType != "" {
		query = query.Where(sladefinition.ServiceType(serviceType))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(sladefinition.Priority(priority))
	}
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where(sladefinition.IsActive(isActive))
	}

	slas, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*SLADefinition
	for _, sla := range slas {
		result = append(result, &SLADefinition{
			ID:             sla.ID,
			Name:           sla.Name,
			Description:    sla.Description,
			ServiceType:    sla.ServiceType,
			Priority:       sla.Priority,
			ResponseTime:   sla.ResponseTime,
			ResolutionTime: sla.ResolutionTime,
			BusinessHours:  sla.BusinessHours,
			IsActive:       sla.IsActive,
			TenantID:       sla.TenantID,
			CreatedAt:      sla.CreatedAt,
			UpdatedAt:      sla.UpdatedAt,
		})
	}

	return result, nil
}

// CheckSLACompliance 检查SLA合规性（简化版本）
func (s *SLAService) CheckSLACompliance(ctx context.Context, ticketID int) (*SLAViolation, error) {
	// 简化实现，返回nil表示无违规
	return nil, nil
}

// GetApplicableSLA 获取适用的SLA定义（简化版本）
func (s *SLAService) GetApplicableSLA(ctx context.Context, serviceType, priority string, tenantID int) (*SLADefinition, error) {
	sla, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.And(
				sladefinition.ServiceType(serviceType),
				sladefinition.Priority(priority),
				sladefinition.IsActive(true),
				sladefinition.TenantID(tenantID),
			),
		).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &SLADefinition{
		ID:             sla.ID,
		Name:           sla.Name,
		Description:    sla.Description,
		ServiceType:    sla.ServiceType,
		Priority:       sla.Priority,
		ResponseTime:   sla.ResponseTime,
		ResolutionTime: sla.ResolutionTime,
		BusinessHours:  sla.BusinessHours,
		IsActive:       sla.IsActive,
		TenantID:       sla.TenantID,
		CreatedAt:      sla.CreatedAt,
		UpdatedAt:      sla.UpdatedAt,
	}, nil
}

// GetSLAMetrics 获取SLA指标（简化版本）
func (s *SLAService) GetSLAMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) (map[string]interface{}, error) {
	// 简化实现，返回基础指标
	metrics := map[string]interface{}{
		"total_tickets":      0,
		"on_time_tickets":    0,
		"breached_tickets":   0,
		"compliance_rate":    100.0,
		"avg_response_time":  0,
		"avg_resolution_time": 0,
	}
	return metrics, nil
}

// GetSLAViolations 获取SLA违规列表（简化版本）
func (s *SLAService) GetSLAViolations(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAViolation, error) {
	// 简化实现，返回空列表
	return []*SLAViolation{}, nil
}

// UpdateViolationStatus 更新违规状态（简化版本）
func (s *SLAService) UpdateViolationStatus(ctx context.Context, id int, status, notes string) error {
	// 简化实现，直接返回成功
	return nil
}
