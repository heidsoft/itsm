package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"

	"go.uber.org/zap"
)

// CIAttributeDefinitionService CI属性定义服务
type CIAttributeDefinitionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCIAttributeDefinitionService 创建CI属性定义服务
func NewCIAttributeDefinitionService(client *ent.Client, logger *zap.SugaredLogger) *CIAttributeDefinitionService {
	return &CIAttributeDefinitionService{
		client: client,
		logger: logger,
	}
}

// CreateCIAttributeDefinition 创建CI属性定义
func (s *CIAttributeDefinitionService) CreateCIAttributeDefinition(ctx context.Context, req *dto.CreateCIAttributeDefinitionRequest, tenantID int) (*dto.CIAttributeDefinitionResponse, error) {
	attr, err := s.client.CIAttributeDefinition.Create().
		SetName(req.Name).
		SetDisplayName(req.DisplayName).
		SetType(req.Type).
		SetRequired(req.Required).
		SetUnique(req.Unique).
		SetDefaultValue(req.DefaultValue).
		SetValidationRules(req.ValidationRules).
		SetCiTypeID(req.CiTypeID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI attribute definition", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("failed to create CI attribute definition: %w", err)
	}

	s.logger.Infow("CI attribute definition created successfully", "attr_id", attr.ID, "tenant_id", tenantID, "name", attr.Name)
	return dto.ToCIAttributeDefinitionResponse(attr), nil
}

// GetCIAttributeDefinitionByID 根据ID获取CI属性定义
func (s *CIAttributeDefinitionService) GetCIAttributeDefinitionByID(ctx context.Context, id, tenantID int) (*dto.CIAttributeDefinitionResponse, error) {
	attr, err := s.client.CIAttributeDefinition.Query().
		Where(ciattributedefinition.IDEQ(id), ciattributedefinition.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get CI attribute definition", "error", err, "attr_id", id)
		return nil, fmt.Errorf("failed to get CI attribute definition: %w", err)
	}

	return dto.ToCIAttributeDefinitionResponse(attr), nil
}

// ListCIAttributeDefinitionsByCITypeID 根据CI类型ID获取属性定义列表
func (s *CIAttributeDefinitionService) ListCIAttributeDefinitionsByCITypeID(ctx context.Context, ciTypeID, tenantID int) ([]*dto.CIAttributeDefinitionResponse, error) {
	attrs, err := s.client.CIAttributeDefinition.Query().
		Where(
			ciattributedefinition.CiTypeIDEQ(ciTypeID),
			ciattributedefinition.TenantIDEQ(tenantID),
			ciattributedefinition.IsActiveEQ(true),
		).
		Order(ent.Asc(ciattributedefinition.FieldName)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI attribute definitions", "error", err, "ci_type_id", ciTypeID)
		return nil, fmt.Errorf("failed to list CI attribute definitions: %w", err)
	}

	return dto.ToCIAttributeDefinitionResponseList(attrs), nil
}

// UpdateCIAttributeDefinition 更新CI属性定义
func (s *CIAttributeDefinitionService) UpdateCIAttributeDefinition(ctx context.Context, id, tenantID int, req *dto.UpdateCIAttributeDefinitionRequest) (*dto.CIAttributeDefinitionResponse, error) {
	update := s.client.CIAttributeDefinition.UpdateOneID(id).
		Where(ciattributedefinition.TenantIDEQ(tenantID))

	if req.DisplayName != nil {
		update.SetDisplayName(*req.DisplayName)
	}
	if req.Type != nil {
		update.SetType(*req.Type)
	}
	if req.Required != nil {
		update.SetRequired(*req.Required)
	}
	if req.Unique != nil {
		update.SetUnique(*req.Unique)
	}
	if req.DefaultValue != nil {
		update.SetDefaultValue(*req.DefaultValue)
	}
	if req.ValidationRules != nil {
		update.SetValidationRules(*req.ValidationRules)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	attr, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI attribute definition", "error", err, "attr_id", id)
		return nil, fmt.Errorf("failed to update CI attribute definition: %w", err)
	}

	s.logger.Infow("CI attribute definition updated successfully", "attr_id", attr.ID, "tenant_id", tenantID)
	return dto.ToCIAttributeDefinitionResponse(attr), nil
}

// DeleteCIAttributeDefinition 删除CI属性定义
func (s *CIAttributeDefinitionService) DeleteCIAttributeDefinition(ctx context.Context, id, tenantID int) error {
	err := s.client.CIAttributeDefinition.DeleteOneID(id).
		Where(ciattributedefinition.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete CI attribute definition", "error", err, "attr_id", id)
		return fmt.Errorf("failed to delete CI attribute definition: %w", err)
	}

	s.logger.Infow("CI attribute definition deleted successfully", "attr_id", id, "tenant_id", tenantID)
	return nil
}
