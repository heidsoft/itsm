package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"
	"itsm-backend/ent/citype"

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
	exists, err := s.client.CIType.Query().
		Where(citype.IDEQ(req.CiTypeID), citype.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate CI type: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("CI type not found")
	}
	if req.Type == "enum" && len(req.EnumValues) == 0 {
		return nil, fmt.Errorf("enumValues is required for enum attributes")
	}

	attr, err := s.client.CIAttributeDefinition.Create().
		SetName(req.Name).
		SetDisplayName(req.DisplayName).
		SetDescription(req.Description).
		SetType(req.Type).
		SetRequired(req.Required).
		SetUnique(req.Unique).
		SetDefaultValue(req.DefaultValue).
		SetValidationRules(req.ValidationRules).
		SetEnumValues(req.EnumValues).
		SetReferenceType(req.ReferenceType).
		SetDisplayOrder(req.DisplayOrder).
		SetGroupName(req.GroupName).
		SetPlaceholder(req.Placeholder).
		SetHelpText(req.HelpText).
		SetIsSearchable(req.IsSearchable).
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
	typeChain, err := s.resolveTypeChain(ctx, ciTypeID, tenantID)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]*ent.CIAttributeDefinition)
	order := make([]string, 0)
	for _, typeID := range typeChain {
		attrs, queryErr := s.client.CIAttributeDefinition.Query().
			Where(
				ciattributedefinition.CiTypeIDEQ(typeID),
				ciattributedefinition.TenantIDEQ(tenantID),
				ciattributedefinition.IsActiveEQ(true),
			).
			Order(ent.Asc(ciattributedefinition.FieldDisplayOrder), ent.Asc(ciattributedefinition.FieldName)).
			All(ctx)
		if queryErr != nil {
			return nil, fmt.Errorf("failed to list CI attribute definitions: %w", queryErr)
		}
		for _, attr := range attrs {
			if _, seen := byName[attr.Name]; !seen {
				order = append(order, attr.Name)
			}
			byName[attr.Name] = attr
		}
	}

	result := make([]*dto.CIAttributeDefinitionResponse, 0, len(order))
	for _, name := range order {
		result = append(result, dto.ToCIAttributeDefinitionResponse(byName[name]))
	}
	return result, nil
}

func (s *CIAttributeDefinitionService) resolveTypeChain(ctx context.Context, ciTypeID, tenantID int) ([]int, error) {
	visited := make(map[int]struct{})
	reversed := make([]int, 0, 4)
	currentID := ciTypeID
	for currentID != 0 {
		if _, exists := visited[currentID]; exists {
			return nil, fmt.Errorf("CI type inheritance cycle detected")
		}
		visited[currentID] = struct{}{}
		current, err := s.client.CIType.Query().
			Where(citype.IDEQ(currentID), citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("CI type not found")
			}
			return nil, fmt.Errorf("failed to resolve CI type inheritance: %w", err)
		}
		reversed = append(reversed, current.ID)
		if current.ParentTypeID == nil {
			break
		}
		currentID = *current.ParentTypeID
	}
	chain := make([]int, len(reversed))
	for i := range reversed {
		chain[len(reversed)-1-i] = reversed[i]
	}
	return chain, nil
}

// UpdateCIAttributeDefinition 更新CI属性定义
func (s *CIAttributeDefinitionService) UpdateCIAttributeDefinition(ctx context.Context, id, tenantID int, req *dto.UpdateCIAttributeDefinitionRequest) (*dto.CIAttributeDefinitionResponse, error) {
	update := s.client.CIAttributeDefinition.UpdateOneID(id).
		Where(ciattributedefinition.TenantIDEQ(tenantID))

	if req.DisplayName != nil {
		update.SetDisplayName(*req.DisplayName)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
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
	if req.EnumValues != nil {
		update.SetEnumValues(*req.EnumValues)
	}
	if req.ReferenceType != nil {
		update.SetReferenceType(*req.ReferenceType)
	}
	if req.DisplayOrder != nil {
		update.SetDisplayOrder(*req.DisplayOrder)
	}
	if req.GroupName != nil {
		update.SetGroupName(*req.GroupName)
	}
	if req.Placeholder != nil {
		update.SetPlaceholder(*req.Placeholder)
	}
	if req.HelpText != nil {
		update.SetHelpText(*req.HelpText)
	}
	if req.IsSearchable != nil {
		update.SetIsSearchable(*req.IsSearchable)
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
