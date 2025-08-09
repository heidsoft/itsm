package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

type CMDBService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewCMDBService(client *ent.Client, logger *zap.SugaredLogger) *CMDBService {
	return &CMDBService{
		client: client,
		logger: logger,
	}
}

// CreateCI 创建配置项
func (s *CMDBService) CreateCI(ctx context.Context, req *dto.CreateCIRequest) (*dto.CIResponse, error) {
	ci, err := s.client.ConfigurationItem.Create().
		SetName(req.Name).
		SetCiTypeID(req.CITypeID).
		SetDescription(req.Description).
		SetStatus(req.Status).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI: %v", err)
		return nil, err
	}

	return &dto.CIResponse{
		ID:          ci.ID,
		Name:        ci.Name,
		CITypeID:    ci.CiTypeID,
		Description: ci.Description,
		Status:      ci.Status,
		TenantID:    ci.TenantID,
		CreatedAt:   ci.CreatedAt,
		UpdatedAt:   ci.UpdatedAt,
	}, nil
}

// GetCI 获取配置项详情
func (s *CMDBService) GetCI(ctx context.Context, id int, tenantID int) (*dto.CIResponse, error) {
	ci, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		First(ctx)

	if err != nil {
		s.logger.Errorf("Failed to get CI: %v", err)
		return nil, err
	}

	return &dto.CIResponse{
		ID:          ci.ID,
		Name:        ci.Name,
		CITypeID:    ci.CiTypeID,
		Description: ci.Description,
		Status:      ci.Status,
		TenantID:    ci.TenantID,
		CreatedAt:   ci.CreatedAt,
		UpdatedAt:   ci.UpdatedAt,
	}, nil
}

// ListCIs 获取配置项列表
func (s *CMDBService) ListCIs(ctx context.Context, req *dto.ListCIsRequest) (*dto.ListCIsResponse, error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantID(req.TenantID))

	// 添加过滤条件
	if req.CITypeID != 0 {
		query = query.Where(configurationitem.CiTypeID(req.CITypeID))
	}

	if req.Status != "" {
		query = query.Where(configurationitem.Status(req.Status))
	}

	// 分页
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	cis, err := query.All(ctx)
	if err != nil {
		s.logger.Errorf("Failed to list CIs: %v", err)
		return nil, err
	}

	// 转换为响应格式
	var ciResponses []*dto.CIResponse
	for _, ci := range cis {
		ciResponses = append(ciResponses, &dto.CIResponse{
			ID:          ci.ID,
			Name:        ci.Name,
			CITypeID:    ci.CiTypeID,
			Description: ci.Description,
			Status:      ci.Status,
			TenantID:    ci.TenantID,
			CreatedAt:   ci.CreatedAt,
			UpdatedAt:   ci.UpdatedAt,
		})
	}

	return &dto.ListCIsResponse{
		CIs:   ciResponses,
		Total: len(ciResponses),
	}, nil
}

// UpdateCI 更新配置项
func (s *CMDBService) UpdateCI(ctx context.Context, id int, req *dto.UpdateCIRequest) (*dto.CIResponse, error) {
	update := s.client.ConfigurationItem.UpdateOneID(id)

	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}
	if req.Status != "" {
		update = update.SetStatus(req.Status)
	}

	ci, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("Failed to update CI: %v", err)
		return nil, err
	}

	return &dto.CIResponse{
		ID:          ci.ID,
		Name:        ci.Name,
		CITypeID:    ci.CiTypeID,
		Description: ci.Description,
		Status:      ci.Status,
		TenantID:    ci.TenantID,
		CreatedAt:   ci.CreatedAt,
		UpdatedAt:   ci.UpdatedAt,
	}, nil
}

// DeleteCI 删除配置项
func (s *CMDBService) DeleteCI(ctx context.Context, id int, tenantID int) error {
	err := s.client.ConfigurationItem.DeleteOneID(id).
		Where(configurationitem.TenantID(tenantID)).
		Exec(ctx)

	if err != nil {
		s.logger.Errorf("Failed to delete CI: %v", err)
		return err
	}

	return nil
}

// CreateCIAttributeDefinition 创建CI属性定义
func (s *CMDBService) CreateCIAttributeDefinition(ctx context.Context, req *dto.CIAttributeDefinitionRequest, tenantID int) (*dto.CIAttributeDefinitionResponse, error) {
	attrDef, err := s.client.CIAttributeDefinition.Create().
		SetName(req.Name).
		SetDisplayName(req.DisplayName).
		SetType(req.DataType).
		SetRequired(req.IsRequired).
		SetUnique(req.IsUnique).
		SetDefaultValue(req.DefaultValue).
		SetValidationRules(fmt.Sprintf("%v", req.ValidationRules)).
		SetCiTypeID(req.CITypeID).
		SetTenantID(tenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI attribute definition: %v", err)
		return nil, err
	}

	return s.convertToAttributeDefinitionResponse(attrDef), nil
}

// GetCITypeWithAttributes 获取CI类型及其属性定义
func (s *CMDBService) GetCITypeWithAttributes(ctx context.Context, ciTypeID int, tenantID int) (*dto.CITypeWithAttributesResponse, error) {
	ciType, err := s.client.CIType.Query().
		Where(citype.ID(ciTypeID), citype.TenantID(tenantID)).
		First(ctx)

	if err != nil {
		s.logger.Errorf("Failed to get CI type: %v", err)
		return nil, err
	}

	// 获取属性定义
	attrDefs, err := s.client.CIAttributeDefinition.Query().
		Where(ciattributedefinition.CiTypeID(ciTypeID)).
		All(ctx)

	if err != nil {
		s.logger.Errorf("Failed to get attribute definitions: %v", err)
		return nil, err
	}

	// 转换为响应格式
	var attrDefResponses []dto.CIAttributeDefinitionResponse
	for _, attrDef := range attrDefs {
		attrDefResponses = append(attrDefResponses, *s.convertToAttributeDefinitionResponse(attrDef))
	}

	return &dto.CITypeWithAttributesResponse{
		ID:                   ciType.ID,
		Name:                 ciType.Name,
		DisplayName:          ciType.Name,
		Description:          ciType.Description,
		Icon:                 ciType.Icon,
		IsActive:             ciType.IsActive,
		AttributeDefinitions: attrDefResponses,
		CreatedAt:            ciType.CreatedAt,
		UpdatedAt:            ciType.UpdatedAt,
	}, nil
}

// convertToAttributeDefinitionResponse 转换为属性定义响应
func (s *CMDBService) convertToAttributeDefinitionResponse(attr *ent.CIAttributeDefinition) *dto.CIAttributeDefinitionResponse {
	return &dto.CIAttributeDefinitionResponse{
		ID:              attr.ID,
		Name:            attr.Name,
		DisplayName:     attr.DisplayName,
		Description:     "", // 简化处理，因为schema中没有Description字段
		DataType:        attr.Type,
		IsRequired:      attr.Required,
		IsUnique:        attr.Unique,
		DefaultValue:    attr.DefaultValue,
		ValidationRules: map[string]interface{}{}, // 简化处理
		IsActive:        attr.IsActive,
		CITypeID:        attr.CiTypeID,
		TenantID:        attr.TenantID,
		CreatedAt:       attr.CreatedAt,
		UpdatedAt:       attr.UpdatedAt,
	}
}
