package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// CITypeService CI类型服务
type CITypeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCITypeService 创建CI类型服务
func NewCITypeService(client *ent.Client, logger *zap.SugaredLogger) *CITypeService {
	return &CITypeService{
		client: client,
		logger: logger,
	}
}

// CreateCIType 创建CI类型
func (s *CITypeService) CreateCIType(ctx context.Context, req *dto.CreateCITypeRequest, tenantID int) (*dto.CITypeResponse, error) {
	ciType, err := s.client.CIType.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetColor(req.Color).
		SetAttributeSchema(req.AttributeSchema).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI type", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("failed to create CI type: %w", err)
	}

	s.logger.Infow("CI type created successfully", "ci_type_id", ciType.ID, "tenant_id", tenantID, "name", ciType.Name)
	return dto.ToCITypeResponse(ciType), nil
}

// GetCITypeByID 根据ID获取CI类型
func (s *CITypeService) GetCITypeByID(ctx context.Context, id, tenantID int) (*dto.CITypeResponse, error) {
	ciType, err := s.client.CIType.Query().
		Where(citype.IDEQ(id), citype.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", id)
		return nil, fmt.Errorf("failed to get CI type: %w", err)
	}

	return dto.ToCITypeResponse(ciType), nil
}

// ListCITypes 获取CI类型列表
func (s *CITypeService) ListCITypes(ctx context.Context, tenantID int, page, pageSize int, search string) (*dto.CITypeListResponse, error) {
	query := s.client.CIType.Query().Where(citype.TenantIDEQ(tenantID), citype.IsActiveEQ(true))

	if search != "" {
		query = query.Where(citype.NameContains(search))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI types", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count CI types: %w", err)
	}

	ciTypes, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(citype.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI types", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list CI types: %w", err)
	}

	return &dto.CITypeListResponse{
		Items: dto.ToCITypeResponseList(ciTypes),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// UpdateCIType 更新CI类型
func (s *CITypeService) UpdateCIType(ctx context.Context, id, tenantID int, req *dto.UpdateCITypeRequest) (*dto.CITypeResponse, error) {
	update := s.client.CIType.UpdateOneID(id).
		Where(citype.TenantIDEQ(tenantID))

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Icon != "" {
		update.SetIcon(req.Icon)
	}
	if req.Color != "" {
		update.SetColor(req.Color)
	}
	if req.AttributeSchema != "" {
		update.SetAttributeSchema(req.AttributeSchema)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	ciType, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI type", "error", err, "ci_type_id", id)
		return nil, fmt.Errorf("failed to update CI type: %w", err)
	}

	s.logger.Infow("CI type updated successfully", "ci_type_id", ciType.ID, "tenant_id", tenantID)
	return dto.ToCITypeResponse(ciType), nil
}

// DeleteCIType 删除CI类型
func (s *CITypeService) DeleteCIType(ctx context.Context, id, tenantID int) error {
	// 检查是否有关联的CI
	count, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.CiTypeIDEQ(id), configurationitem.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check CI type usage", "error", err, "ci_type_id", id)
		return fmt.Errorf("failed to check CI type usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete CI type with %d associated configuration items", count)
	}

	err = s.client.CIType.DeleteOneID(id).
		Where(citype.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete CI type", "error", err, "ci_type_id", id)
		return fmt.Errorf("failed to delete CI type: %w", err)
	}

	s.logger.Infow("CI type deleted successfully", "ci_type_id", id, "tenant_id", tenantID)
	return nil
}
