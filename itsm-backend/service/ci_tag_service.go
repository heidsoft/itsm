package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citag"

	"go.uber.org/zap"
)

// CITagService CI标签服务
type CITagService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCITagService 创建CI标签服务
func NewCITagService(client *ent.Client, logger *zap.SugaredLogger) *CITagService {
	return &CITagService{
		client: client,
		logger: logger,
	}
}

// CreateCITag 创建CI标签
func (s *CITagService) CreateCITag(ctx context.Context, req *dto.CreateCITagRequest, tenantID int) (*dto.CITagResponse, error) {
	// 检查标签是否已存在
	exists, err := s.client.CITag.Query().
		Where(
			citag.TenantIDEQ(tenantID),
			citag.KeyEQ(req.Key),
			citag.ValueEQ(req.Value),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check CI tag existence", "error", err, "tenant_id", tenantID, "key", req.Key, "value", req.Value)
		return nil, fmt.Errorf("failed to check tag existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("tag with key '%s' and value '%s' already exists", req.Key, req.Value)
	}

	tag, err := s.client.CITag.Create().
		SetKey(req.Key).
		SetValue(req.Value).
		SetColor(req.Color).
		SetDescription(req.Description).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI tag", "error", err, "tenant_id", tenantID, "key", req.Key)
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	s.logger.Infow("CI tag created successfully", "tag_id", tag.ID, "tenant_id", tenantID, "key", tag.Key)
	return dto.ToCITagResponse(tag), nil
}

// GetCITagByID 根据ID获取CI标签
func (s *CITagService) GetCITagByID(ctx context.Context, id, tenantID int) (*dto.CITagResponse, error) {
	tag, err := s.client.CITag.Query().
		Where(
			citag.IDEQ(id),
			citag.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("tag not found")
		}
		s.logger.Errorw("Failed to get CI tag", "error", err, "tag_id", id)
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return dto.ToCITagResponse(tag), nil
}

// ListCITags 获取CI标签列表
func (s *CITagService) ListCITags(ctx context.Context, tenantID int, page, pageSize int, search string) (*dto.CITagListResponse, error) {
	query := s.client.CITag.Query().Where(citag.TenantIDEQ(tenantID))

	if search != "" {
		query = query.Where(
			citag.Or(
				citag.KeyContains(search),
				citag.ValueContains(search),
				citag.DescriptionContains(search),
			),
		)
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI tags", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count tags: %w", err)
	}

	tags, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(citag.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI tags", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return &dto.CITagListResponse{
		Items: dto.ToCITagResponseList(tags),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// UpdateCITag 更新CI标签
func (s *CITagService) UpdateCITag(ctx context.Context, id, tenantID int, req *dto.UpdateCITagRequest) (*dto.CITagResponse, error) {
	update := s.client.CITag.UpdateOneID(id).
		Where(citag.TenantIDEQ(tenantID))

	if req.Key != nil {
		update.SetKey(*req.Key)
	}
	if req.Value != nil {
		update.SetValue(*req.Value)
	}
	if req.Color != nil {
		update.SetColor(*req.Color)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}

	tag, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI tag", "error", err, "tag_id", id)
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	s.logger.Infow("CI tag updated successfully", "tag_id", tag.ID, "tenant_id", tenantID)
	return dto.ToCITagResponse(tag), nil
}

// DeleteCITag 删除CI标签
func (s *CITagService) DeleteCITag(ctx context.Context, id, tenantID int) error {
	// 检查标签是否被CI使用
	count, err := s.client.CITag.Query().
		Where(citag.IDEQ(id), citag.TenantIDEQ(tenantID)).
		QueryCis().
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check tag usage", "error", err, "tag_id", id)
		return fmt.Errorf("failed to check tag usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete tag that is used by %d CIs", count)
	}

	err = s.client.CITag.DeleteOneID(id).
		Where(citag.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete CI tag", "error", err, "tag_id", id)
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	s.logger.Infow("CI tag deleted successfully", "tag_id", id, "tenant_id", tenantID)
	return nil
}
