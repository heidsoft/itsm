package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/user"

	"github.com/gin-gonic/gin"
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
func (s *IncidentService) CreateIncident(ctx context.Context, req *dto.CreateIncidentRequest, reporterID, tenantID int) (*dto.Incident, error) {
	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}

	// 确保事务回滚
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 验证报告人是否存在
	_, err = tx.User.Query().Where(user.ID(reporterID)).Only(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("报告人不存在: %w", err)
	}

	// 验证处理人是否存在（如果指定）
	if req.AssigneeID != nil {
		_, err = tx.User.Query().Where(user.ID(*req.AssigneeID)).Only(ctx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("处理人不存在: %w", err)
		}
	}

	// 验证配置项是否存在（如果指定）
	if req.ConfigurationItemID != nil {
		_, err = tx.ConfigurationItem.Query().Where(configurationitem.ID(*req.ConfigurationItemID)).Only(ctx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("配置项不存在: %w", err)
		}
	}

	// 生成事件编号
	incidentNumber, err := s.generateIncidentNumber(ctx, tx, tenantID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("生成事件编号失败: %w", err)
	}

	// 创建事件
	incident, err := tx.Incident.Create().
		SetIncidentNumber(incidentNumber).
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetStatus(dto.IncidentStatusNew).
		SetPriority(req.Priority).
		SetReporterID(reporterID).
		SetNillableAssigneeID(req.AssigneeID).
		SetNillableConfigurationItemID(req.ConfigurationItemID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建事件失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 异步发送通知
	go s.sendIncidentNotification(incident, "created")

	s.logger.Infof("事件创建成功: %s, 报告人: %d, 租户: %d", incidentNumber, reporterID, tenantID)

	return dto.ToIncidentResponse(incident), nil
}

// GetIncidents 获取事件列表
func (s *IncidentService) GetIncidents(ctx context.Context, req *dto.GetIncidentsRequest) (*dto.IncidentListResponse, error) {
	query := s.client.Incident.Query().
		Where(incident.TenantID(req.TenantID))

	// 添加过滤条件
	if req.Status != "" {
		query = query.Where(incident.Status(req.Status))
	}
	if req.Priority != "" {
		query = query.Where(incident.Priority(req.Priority))
	}
	if req.ReporterID > 0 {
		query = query.Where(incident.ReporterID(req.ReporterID))
	}
	if req.AssigneeID > 0 {
		query = query.Where(incident.AssigneeID(req.AssigneeID))
	}
	if req.ConfigurationItemID > 0 {
		query = query.Where(incident.ConfigurationItemID(req.ConfigurationItemID))
	}

	// 获取总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事件总数失败: %w", err)
	}

	// 分页查询
	incidents, err := query.
		Order(ent.Desc(incident.FieldCreatedAt)).
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事件列表失败: %w", err)
	}

	// 转换为DTO
	incidentDTOs := make([]dto.Incident, len(incidents))
	for i, incident := range incidents {
		incidentDTO := *dto.ToIncidentResponse(incident)

		// 如果有关联的配置项，获取配置项详细信息
		if incident.ConfigurationItemID > 0 {
			ci, err := s.client.ConfigurationItem.Query().
				Where(configurationitem.ID(incident.ConfigurationItemID)).
				Where(configurationitem.TenantID(req.TenantID)).
				Only(ctx)
			if err == nil {
				// 获取配置项类型信息
				ciType, err := s.client.CIType.Query().
					Where(citype.ID(ci.CiTypeID)).
					Only(ctx)
				if err == nil {
					incidentDTO.ConfigurationItem = &dto.ConfigurationItemInfo{
						ID:          ci.ID,
						Name:        ci.Name,
						Type:        ciType.Name,
						Status:      ci.Status,
						Description: ci.Description,
					}
				}
			}
		}

		incidentDTOs[i] = incidentDTO
	}

	return &dto.IncidentListResponse{
		Incidents: incidentDTOs,
		Total:     total,
		Page:      req.Page,
		Size:      req.Size,
	}, nil
}

// GetIncident 获取单个事件
func (s *IncidentService) GetIncident(ctx context.Context, incidentID, tenantID int) (*dto.Incident, error) {
	incident, err := s.client.Incident.Query().
		Where(incident.ID(incidentID)).
		Where(incident.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("事件不存在或无权限: %w", err)
	}

	// 转换为DTO
	incidentDTO := dto.ToIncidentResponse(incident)

	// 如果有关联的配置项，获取配置项详细信息
	if incident.ConfigurationItemID > 0 {
		ci, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.ID(incident.ConfigurationItemID)).
			Where(configurationitem.TenantID(tenantID)).
			Only(ctx)
		if err == nil {
			// 获取配置项类型信息
			ciType, err := s.client.CIType.Query().
				Where(citype.ID(ci.CiTypeID)).
				Only(ctx)
			if err == nil {
				incidentDTO.ConfigurationItem = &dto.ConfigurationItemInfo{
					ID:          ci.ID,
					Name:        ci.Name,
					Type:        ciType.Name,
					Status:      ci.Status,
					Description: ci.Description,
				}
			}
		}
	}

	return incidentDTO, nil
}

// UpdateIncident 更新事件
func (s *IncidentService) UpdateIncident(ctx context.Context, incidentID int, req *dto.UpdateIncidentRequest, tenantID int) (*dto.Incident, error) {
	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 验证事件是否存在
	_, err = tx.Incident.Query().
		Where(incident.ID(incidentID)).
		Where(incident.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("事件不存在或无权限: %w", err)
	}

	// 验证处理人是否存在（如果指定）
	if req.AssigneeID != nil {
		_, err = tx.User.Query().Where(user.ID(*req.AssigneeID)).Only(ctx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("处理人不存在: %w", err)
		}
	}

	// 验证配置项是否存在（如果指定）
	if req.ConfigurationItemID != nil {
		_, err = tx.ConfigurationItem.Query().Where(configurationitem.ID(*req.ConfigurationItemID)).Only(ctx)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("配置项不存在: %w", err)
		}
	}

	// 构建更新操作
	update := tx.Incident.UpdateOneID(incidentID)

	if req.Title != nil {
		update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Status != nil {
		update.SetStatus(*req.Status)
		// 如果状态变为已解决，设置解决时间
		if *req.Status == dto.IncidentStatusResolved {
			now := time.Now()
			update.SetResolvedAt(now)
		}
		// 如果状态变为已关闭，设置关闭时间
		if *req.Status == dto.IncidentStatusClosed {
			now := time.Now()
			update.SetClosedAt(now)
		}
	}
	if req.Priority != nil {
		update.SetPriority(*req.Priority)
	}
	if req.AssigneeID != nil {
		update.SetAssigneeID(*req.AssigneeID)
	}
	if req.ConfigurationItemID != nil {
		update.SetConfigurationItemID(*req.ConfigurationItemID)
	}

	// 执行更新
	updatedIncident, err := update.Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新事件失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 异步发送通知
	go s.sendIncidentNotification(updatedIncident, "updated")

	s.logger.Infof("事件更新成功: %d, 租户: %d", incidentID, tenantID)

	return dto.ToIncidentResponse(updatedIncident), nil
}

// CloseIncident 关闭事件
func (s *IncidentService) CloseIncident(ctx context.Context, incidentID, tenantID int) error {
	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 验证事件是否存在
	incident, err := tx.Incident.Query().
		Where(incident.ID(incidentID)).
		Where(incident.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("事件不存在或无权限: %w", err)
	}

	// 检查事件状态
	if incident.Status == dto.IncidentStatusClosed {
		tx.Rollback()
		return fmt.Errorf("事件已经关闭")
	}

	now := time.Now()
	_, err = tx.Incident.UpdateOneID(incidentID).
		SetStatus(dto.IncidentStatusClosed).
		SetClosedAt(now).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("关闭事件失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 异步发送通知
	go s.sendIncidentNotification(incident, "closed")

	s.logger.Infof("事件关闭成功: %d, 租户: %d", incidentID, tenantID)

	return nil
}

// GetIncidentStats 获取事件统计
func (s *IncidentService) GetIncidentStats(ctx context.Context, tenantID int) (*dto.IncidentStatsResponse, error) {
	// 获取各种状态的事件数量
	total, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取总事件数失败: %w", err)
	}

	open, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.StatusIn(dto.IncidentStatusNew, dto.IncidentStatusInProgress, dto.IncidentStatusWaitingCustomer)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取开放事件数失败: %w", err)
	}

	resolved, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.Status(dto.IncidentStatusResolved)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取已解决事件数失败: %w", err)
	}

	closed, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.Status(dto.IncidentStatusClosed)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取已关闭事件数失败: %w", err)
	}

	urgent, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.Priority(dto.IncidentPriorityUrgent)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取紧急事件数失败: %w", err)
	}

	high, err := s.client.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.Priority(dto.IncidentPriorityHigh)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取高优先级事件数失败: %w", err)
	}

	return &dto.IncidentStatsResponse{
		TotalIncidents:        total,
		OpenIncidents:         open,
		ResolvedIncidents:     resolved,
		ClosedIncidents:       closed,
		UrgentIncidents:       urgent,
		HighPriorityIncidents: high,
	}, nil
}

// generateIncidentNumber 生成事件编号
func (s *IncidentService) generateIncidentNumber(ctx context.Context, tx *ent.Tx, tenantID int) (string, error) {
	// 获取当前年份
	year := time.Now().Year()

	// 获取该租户今年的事件数量
	count, err := tx.Incident.Query().
		Where(incident.TenantID(tenantID)).
		Where(incident.CreatedAtGTE(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))).
		Count(ctx)
	if err != nil {
		return "", err
	}

	// 生成编号格式：INC-YYYY-XXXXX
	incidentNumber := fmt.Sprintf("INC-%d-%05d", year, count+1)
	return incidentNumber, nil
}

// sendIncidentNotification 发送事件通知（模拟）
func (s *IncidentService) sendIncidentNotification(incident *ent.Incident, action string) {
	// 这里应该实现真正的通知逻辑
	// 例如：发送邮件、推送消息、WebSocket通知等
	s.logger.Infof("发送事件通知: 事件ID=%d, 动作=%s, 处理人ID=%v",
		incident.ID, action, incident.AssigneeID)
}

// GetCurrentUserID 从上下文获取当前用户ID
func (s *IncidentService) GetCurrentUserID(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, fmt.Errorf("用户ID不存在")
	}
	return userID.(int), nil
}

// GetCurrentTenantID 从上下文获取当前租户ID
func (s *IncidentService) GetCurrentTenantID(c *gin.Context) (int, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0, fmt.Errorf("租户ID不存在")
	}
	return tenantID.(int), nil
}

// GetConfigurationItemsForIncident 获取可关联的配置项列表
func (s *IncidentService) GetConfigurationItemsForIncident(ctx context.Context, tenantID int, search, ciType, status string) ([]dto.ConfigurationItemInfo, error) {
	// 构建查询条件
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantID(tenantID))

	// 添加搜索条件
	if search != "" {
		query = query.Where(configurationitem.NameContains(search))
	}

	// 添加状态过滤
	if status != "" {
		query = query.Where(configurationitem.Status(status))
	}

	// 限制返回数量，避免数据过多
	query = query.Limit(50)

	// 执行查询
	cis, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询配置项失败: %w", err)
	}

	// 转换为响应格式
	var ciInfos []dto.ConfigurationItemInfo
	for _, ci := range cis {
		// 获取配置项类型信息
		ciType, err := s.client.CIType.Query().
			Where(citype.ID(ci.CiTypeID)).
			Only(ctx)
		if err == nil {
			ciInfos = append(ciInfos, dto.ConfigurationItemInfo{
				ID:          ci.ID,
				Name:        ci.Name,
				Type:        ciType.Name,
				Status:      ci.Status,
				Description: ci.Description,
			})
		}
	}

	return ciInfos, nil
}
