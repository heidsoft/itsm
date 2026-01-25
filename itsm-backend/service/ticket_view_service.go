package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketview"
	"time"

	"go.uber.org/zap"
)

type TicketViewService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketViewService(client *ent.Client, logger *zap.SugaredLogger) *TicketViewService {
	return &TicketViewService{
		client: client,
		logger: logger,
	}
}

// ListTicketViews 获取工单视图列表
func (s *TicketViewService) ListTicketViews(
	ctx context.Context,
	tenantID int,
	userID *int,
) ([]*dto.TicketViewResponse, error) {
	query := s.client.TicketView.Query().
		Where(ticketview.TenantID(tenantID))

	// 如果指定了用户ID，只返回该用户创建的或共享的视图
	if userID != nil {
		query = query.Where(
			ticketview.Or(
				ticketview.CreatedBy(*userID),
				ticketview.IsShared(true),
			),
		)
	}

	views, err := query.
		Order(ent.Desc(ticketview.FieldCreatedAt)).
		WithCreator().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list ticket views", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list ticket views: %w", err)
	}

	responses := make([]*dto.TicketViewResponse, 0, len(views))
	for _, view := range views {
		var creator *ent.User
		if view.Edges.Creator != nil {
			creator = view.Edges.Creator
		} else {
			creator, _ = s.client.User.Get(ctx, view.CreatedBy)
		}
		responses = append(responses, dto.ToTicketViewResponse(view, creator))
	}

	return responses, nil
}

// GetTicketView 获取工单视图详情
func (s *TicketViewService) GetTicketView(
	ctx context.Context,
	viewID, tenantID int,
) (*dto.TicketViewResponse, error) {
	view, err := s.client.TicketView.Query().
		Where(
			ticketview.ID(viewID),
			ticketview.TenantID(tenantID),
		).
		WithCreator().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket view not found: %w", err)
	}

	var creator *ent.User
	if view.Edges.Creator != nil {
		creator = view.Edges.Creator
	} else {
		creator, _ = s.client.User.Get(ctx, view.CreatedBy)
	}

	return dto.ToTicketViewResponse(view, creator), nil
}

// CreateTicketView 创建工单视图
func (s *TicketViewService) CreateTicketView(
	ctx context.Context,
	req *dto.CreateTicketViewRequest,
	userID, tenantID int,
) (*dto.TicketViewResponse, error) {
	s.logger.Infow("Creating ticket view", "name", req.Name, "user_id", userID, "tenant_id", tenantID)

	view, err := s.client.TicketView.Create().
		SetName(req.Name).
		SetNillableDescription(req.Description).
		SetFilters(req.Filters).
		SetColumns(req.Columns).
		SetSortConfig(req.SortConfig).
		SetGroupConfig(req.GroupConfig).
		SetIsShared(req.IsShared).
		SetCreatedBy(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket view", "error", err)
		return nil, fmt.Errorf("failed to create ticket view: %w", err)
	}

	// 获取创建人信息
	creator, _ := s.client.User.Get(ctx, userID)

	return dto.ToTicketViewResponse(view, creator), nil
}

// UpdateTicketView 更新工单视图
func (s *TicketViewService) UpdateTicketView(
	ctx context.Context,
	viewID int,
	req *dto.UpdateTicketViewRequest,
	userID, tenantID int,
) (*dto.TicketViewResponse, error) {
	s.logger.Infow("Updating ticket view", "view_id", viewID, "user_id", userID, "tenant_id", tenantID)

	// 验证视图是否存在且属于当前租户
	view, err := s.client.TicketView.Query().
		Where(
			ticketview.ID(viewID),
			ticketview.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket view not found: %w", err)
	}

	// 验证权限：只有创建人可以更新，或者管理员可以更新共享视图
	if view.CreatedBy != userID {
		return nil, fmt.Errorf("permission denied: only creator can update this view")
	}

	updateQuery := s.client.TicketView.UpdateOneID(viewID)

	if req.Name != nil {
		updateQuery.SetName(*req.Name)
	}
	if req.Description != nil {
		updateQuery.SetNillableDescription(req.Description)
	}
	if req.Filters != nil {
		updateQuery.SetFilters(req.Filters)
	}
	if req.Columns != nil {
		updateQuery.SetColumns(req.Columns)
	}
	if req.SortConfig != nil {
		updateQuery.SetSortConfig(req.SortConfig)
	}
	if req.GroupConfig != nil {
		updateQuery.SetGroupConfig(req.GroupConfig)
	}
	if req.IsShared != nil {
		updateQuery.SetIsShared(*req.IsShared)
	}

	updateQuery.SetUpdatedAt(time.Now())

	updatedView, err := updateQuery.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket view", "error", err)
		return nil, fmt.Errorf("failed to update ticket view: %w", err)
	}

	// 获取创建人信息
	creator, _ := s.client.User.Get(ctx, updatedView.CreatedBy)

	return dto.ToTicketViewResponse(updatedView, creator), nil
}

// DeleteTicketView 删除工单视图
func (s *TicketViewService) DeleteTicketView(
	ctx context.Context,
	viewID, userID, tenantID int,
) error {
	s.logger.Infow("Deleting ticket view", "view_id", viewID, "user_id", userID, "tenant_id", tenantID)

	// 验证视图是否存在且属于当前租户
	view, err := s.client.TicketView.Query().
		Where(
			ticketview.ID(viewID),
			ticketview.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("ticket view not found: %w", err)
	}

	// 验证权限：只有创建人可以删除
	if view.CreatedBy != userID {
		return fmt.Errorf("permission denied: only creator can delete this view")
	}

	err = s.client.TicketView.DeleteOneID(viewID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete ticket view", "error", err)
		return fmt.Errorf("failed to delete ticket view: %w", err)
	}

	return nil
}

// ShareTicketView 共享工单视图
func (s *TicketViewService) ShareTicketView(
	ctx context.Context,
	viewID int,
	req *dto.ShareTicketViewRequest,
	userID, tenantID int,
) error {
	s.logger.Infow("Sharing ticket view", "view_id", viewID, "user_id", userID, "tenant_id", tenantID)

	// 验证视图是否存在且属于当前租户
	view, err := s.client.TicketView.Query().
		Where(
			ticketview.ID(viewID),
			ticketview.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("ticket view not found: %w", err)
	}

	// 验证权限：只有创建人可以共享
	if view.CreatedBy != userID {
		return fmt.Errorf("permission denied: only creator can share this view")
	}

	// 更新共享状态
	_, err = s.client.TicketView.UpdateOneID(viewID).
		SetIsShared(true).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to share ticket view", "error", err)
		return fmt.Errorf("failed to share ticket view: %w", err)
	}

	// 注意：团队共享需要创建 ticket_view_team 表来记录共享关系
	// 当前实现仅标记为共享，未来版本可完善

	return nil
}

