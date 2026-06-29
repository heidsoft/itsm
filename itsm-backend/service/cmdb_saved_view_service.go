package service

import (
	"context"
	"encoding/json"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cmdbsavedview"

	"go.uber.org/zap"
)
	"go.uber.org/zap"
)

// CMDBSavedViewService CMDB保存视图服务
type CMDBSavedViewService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCMDBSavedViewService 创建CMDB保存视图服务
func NewCMDBSavedViewService(client *ent.Client, logger *zap.SugaredLogger) *CMDBSavedViewService {
	return &CMDBSavedViewService{
		client: client,
		logger: logger,
	}
}

// CreateSavedView 创建保存视图
func (s *CMDBSavedViewService) CreateSavedView(ctx context.Context, req *dto.CreateCISavedViewRequest, tenantID int, creatorID int, creatorName string) (*dto.CISavedView, error) {
	// 检查同名视图是否已存在
	exists, err := s.client.CMDBSavedView.Query().
		Where(
			cmdbsavedview.TenantID(tenantID),
			cmdbsavedview.Name(req.Name),
			cmdbsavedview.CreatorID(creatorID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check saved view existence", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("检查视图是否存在失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("已存在同名的视图")
	}

	// 转换过滤条件为JSON
	filtersJSON, err := toJSONMap(&req.Filters)
	if err != nil {
		s.logger.Errorw("Failed to marshal filters to JSON", "error", err)
		return nil, fmt.Errorf("序列化过滤条件失败: %w", err)
	}

	// 创建视图
	view, err := s.client.CMDBSavedView.Create().
		SetName(req.Name).
		SetNillableDescription(&req.Description).
		SetFilters(filtersJSON).
		SetNillableSortBy(&req.SortBy).
		SetSortOrder(req.SortOrder).
		SetIsPublic(req.IsPublic).
		SetCreatorID(creatorID).
		SetCreatorName(creatorName).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create saved view", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("创建视图失败: %w", err)
	}

	s.logger.Infow("Saved view created successfully", "view_id", view.ID, "name", view.Name, "tenant_id", tenantID)
	return s.convertToViewDTO(view), nil
}

// GetSavedView 获取视图详情
func (s *CMDBSavedViewService) GetSavedView(ctx context.Context, id int, tenantID int) (*dto.CISavedView, error) {
	view, err := s.client.CMDBSavedView.Query().
		Where(
			cmdbsavedview.ID(id),
			cmdbsavedview.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("视图不存在")
		}
		s.logger.Errorw("Failed to get saved view", "error", err, "view_id", id)
		return nil, fmt.Errorf("获取视图失败: %w", err)
	}

	return s.convertToViewDTO(view), nil
}

// ListSavedViews 获取视图列表
func (s *CMDBSavedViewService) ListSavedViews(ctx context.Context, tenantID int, userID int, includePublic bool, page, pageSize int) (*dto.ListResponse[dto.CISavedView], error) {
	query := s.client.CMDBSavedView.Query().
		Where(cmdbsavedview.TenantID(tenantID))

	// 权限过滤：只能看自己创建的和公开的
	conditions := []cmdbsavedview.Predicate{
		cmdbsavedview.CreatorID(userID),
	}
	if includePublic {
		conditions = append(conditions, cmdbsavedview.IsPublic(true))
	}
	query = query.Where(cmdbsavedview.Or(conditions...))

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count saved views", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计视图数量失败: %w", err)
	}

	// 查询列表
	views, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(cmdbsavedview.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list saved views", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("获取视图列表失败: %w", err)
	}

	// 转换为DTO
	items := make([]dto.CISavedView, len(views))
	for i, view := range views {
		items[i] = *s.convertToViewDTO(view)
	}

	return &dto.ListResponse[dto.CISavedView]{
		Items: items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// UpdateSavedView 更新视图
func (s *CMDBSavedViewService) UpdateSavedView(ctx context.Context, id int, tenantID int, userID int, req *dto.UpdateCISavedViewRequest) (*dto.CISavedView, error) {
	// 检查视图是否存在且属于当前用户
	view, err := s.client.CMDBSavedView.Query().
		Where(
			cmdbsavedview.ID(id),
			cmdbsavedview.TenantID(tenantID),
			cmdbsavedview.CreatorID(userID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("视图不存在或没有权限修改")
		}
		s.logger.Errorw("Failed to get saved view for update", "error", err, "view_id", id)
		return nil, fmt.Errorf("获取视图失败: %w", err)
	}

	update := s.client.CMDBSavedView.UpdateOne(view)

	if req.Name != nil && *req.Name != "" {
		// 检查新名称是否与其他视图重名
		exists, err := s.client.CMDBSavedView.Query().
			Where(
				cmdbsavedview.TenantID(tenantID),
				cmdbsavedview.Name(*req.Name),
				cmdbsavedview.CreatorID(userID),
				cmdbsavedview.IDNEQ(id),
			).
			Exist(ctx)
		if err != nil {
			s.logger.Errorw("Failed to check duplicate view name", "error", err, "view_id", id, "name", *req.Name)
			return nil, fmt.Errorf("检查名称是否重复失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("已存在同名的视图")
		}
		update.SetName(*req.Name)
	}

	if req.Description != nil {
		update.SetDescription(*req.Description)
	}

	if req.Filters != nil {
		filtersJSON, err := toJSONMap(req.Filters)
		if err != nil {
			s.logger.Errorw("Failed to marshal filters to JSON", "error", err)
			return nil, fmt.Errorf("序列化过滤条件失败: %w", err)
		}
		update.SetFilters(filtersJSON)
	}

	if req.SortBy != nil {
		update.SetSortBy(*req.SortBy)
	}

	if req.SortOrder != nil {
		update.SetSortOrder(*req.SortOrder)
	}

	if req.IsPublic != nil {
		update.SetIsPublic(*req.IsPublic)
	}

	// 保存更新
	updatedView, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update saved view", "error", err, "view_id", id)
		return nil, fmt.Errorf("更新视图失败: %w", err)
	}

	s.logger.Infow("Saved view updated successfully", "view_id", id, "tenant_id", tenantID)
	return s.convertToViewDTO(updatedView), nil
}

// DeleteSavedView 删除视图
func (s *CMDBSavedViewService) DeleteSavedView(ctx context.Context, id int, tenantID int, userID int) error {
	// 检查视图是否存在且属于当前用户
	view, err := s.client.CMDBSavedView.Query().
		Where(
			cmdbsavedview.ID(id),
			cmdbsavedview.TenantID(tenantID),
			cmdbsavedview.CreatorID(userID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("视图不存在或没有权限删除")
		}
		s.logger.Errorw("Failed to get saved view for delete", "error", err, "view_id", id)
		return fmt.Errorf("获取视图失败: %w", err)
	}

	// 删除视图
	err = s.client.CMDBSavedView.DeleteOne(view).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete saved view", "error", err, "view_id", id)
		return fmt.Errorf("删除视图失败: %w", err)
	}

	s.logger.Infow("Saved view deleted successfully", "view_id", id, "tenant_id", tenantID)
	return nil
}

// convertToViewDTO 转换为视图DTO
func (s *CMDBSavedViewService) convertToViewDTO(view *ent.CMDBSavedView) *dto.CISavedView {
	// 转换过滤条件
	var filters dto.CISearchFilter
	err := fromJSONMap(view.Filters, &filters)
	if err != nil {
		s.logger.Warnw("Failed to unmarshal saved view filters", "error", err, "view_id", view.ID)
	}

	return &dto.CISavedView{
		ID:          view.ID,
		Name:        view.Name,
		Description: view.Description,
		Filters:     filters,
		SortBy:      view.SortBy,
		SortOrder:   view.SortOrder,
		IsPublic:    view.IsPublic,
		CreatorID:   view.CreatorID,
		CreatorName: view.CreatorName,
		TenantID:    view.TenantID,
		CreatedAt:   view.CreatedAt,
		UpdatedAt:   view.UpdatedAt,
	}
}

// toJSONMap 将任意对象转换为map[string]interface{}
func toJSONMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	return m, err
}

// fromJSONMap 将map[string]interface{}转换为对象
func fromJSONMap(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
