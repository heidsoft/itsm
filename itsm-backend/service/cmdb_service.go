package service

import (
	"context"
	"fmt"
	"strconv" // 添加 strconv 包
	"time"    // 添加 time 包

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
		SetAttributes(req.Attributes).
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
		Attributes:  ci.Attributes,
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
		WithOutgoingRelationships().
		WithIncomingRelationships().
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
		Attributes:  ci.Attributes,
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
			Attributes:  ci.Attributes,
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
	if req.Attributes != nil {
		update = update.SetAttributes(req.Attributes)
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
		Attributes:  ci.Attributes,
		TenantID:    ci.TenantID,
		CreatedAt:   ci.CreatedAt,
		UpdatedAt:   ci.UpdatedAt,
	}, nil
}

// DeleteCI 删除配置项
func (s *CMDBService) DeleteCI(ctx context.Context, id int, tenantID int) error {
	err := s.client.ConfigurationItem.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorf("Failed to delete CI: %v", err)
		return err
	}

	s.logger.Infof("Deleted CI with ID: %d", id)
	return nil
}

// CreateCIAttributeDefinition 创建CI属性定义
func (s *CMDBService) CreateCIAttributeDefinition(ctx context.Context, req *dto.CIAttributeDefinitionRequest, tenantID int) (*dto.CIAttributeDefinitionResponse, error) {
	// 验证CI类型是否存在
	ciType, err := s.client.CIType.Get(ctx, req.CITypeID)
	if err != nil {
		s.logger.Errorf("CI type not found: %v", err)
		return nil, err
	}

	if ciType.TenantID != tenantID {
		return nil, fmt.Errorf("unauthorized access to CI type")
	}

	// 创建属性定义
	attr, err := s.client.CIAttributeDefinition.Create().
		SetName(req.Name).
		SetDisplayName(req.DisplayName).
		SetDescription(req.Description).
		SetDataType(req.DataType).
		SetIsRequired(req.IsRequired).
		SetIsUnique(req.IsUnique).
		SetDefaultValue(req.DefaultValue).
		SetValidationRules(req.ValidationRules).
		SetEnumValues(req.EnumValues).
		SetReferenceType(req.ReferenceType).
		SetDisplayOrder(req.DisplayOrder).
		SetIsSearchable(req.IsSearchable).
		SetCiTypeID(req.CITypeID).
		SetTenantID(tenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI attribute definition: %v", err)
		return nil, err
	}

	return s.convertToAttributeDefinitionResponse(attr), nil
}

// GetCITypeWithAttributes 获取CI类型及其属性定义
func (s *CMDBService) GetCITypeWithAttributes(ctx context.Context, ciTypeID int, tenantID int) (*dto.CITypeWithAttributesResponse, error) {
	ciType, err := s.client.CIType.Query().
		Where(
			citype.ID(ciTypeID),
			citype.TenantID(tenantID),
		).
		WithAttributeDefinitions(func(q *ent.CIAttributeDefinitionQuery) {
			q.Where(ciattributedefinition.IsActive(true)).
				Order(ent.Asc(ciattributedefinition.FieldDisplayOrder))
		}).
		First(ctx)

	if err != nil {
		s.logger.Errorf("Failed to get CI type with attributes: %v", err)
		return nil, err
	}

	// 转换属性定义
	var attrDefs []dto.CIAttributeDefinitionResponse
	for _, attr := range ciType.Edges.AttributeDefinitions {
		attrDefs = append(attrDefs, *s.convertToAttributeDefinitionResponse(attr))
	}

	return &dto.CITypeWithAttributesResponse{
		ID:                   ciType.ID,
		Name:                 ciType.Name,
		DisplayName:          ciType.DisplayName,
		Description:          ciType.Description,
		Category:             ciType.Category,
		Icon:                 ciType.Icon,
		IsSystem:             ciType.IsSystem,
		IsActive:             ciType.IsActive,
		AttributeDefinitions: attrDefs,
		CreatedAt:            ciType.CreatedAt,
		UpdatedAt:            ciType.UpdatedAt,
	}, nil
}

// ValidateCIAttributes 验证CI属性
func (s *CMDBService) ValidateCIAttributes(ctx context.Context, req *dto.ValidateCIAttributesRequest, tenantID int) (*dto.ValidateCIAttributesResponse, error) {
	// 获取CI类型的属性定义
	attrDefs, err := s.client.CIAttributeDefinition.Query().
		Where(
			ciattributedefinition.CiTypeID(req.CITypeID),
			ciattributedefinition.TenantID(tenantID),
			ciattributedefinition.IsActive(true),
		).
		All(ctx)

	if err != nil {
		s.logger.Errorf("Failed to get attribute definitions: %v", err)
		return nil, err
	}

	errors := make(map[string]string)
	warnings := make(map[string]string)
	normalizedAttrs := make(map[string]interface{})

	// 验证每个属性定义
	for _, attrDef := range attrDefs {
		value, exists := req.Attributes[attrDef.Name]

		// 检查必填字段
		if attrDef.IsRequired && (!exists || value == nil || value == "") {
			errors[attrDef.Name] = fmt.Sprintf("Field '%s' is required", attrDef.DisplayName)
			continue
		}

		// 如果字段不存在且有默认值，使用默认值
		if !exists && attrDef.DefaultValue != "" {
			normalizedAttrs[attrDef.Name] = s.parseDefaultValue(attrDef.DefaultValue, attrDef.DataType)
			continue
		}

		if exists && value != nil {
			// 验证数据类型和格式
			if validationErr := s.validateAttributeValue(value, attrDef); validationErr != nil {
				errors[attrDef.Name] = validationErr.Error()
			} else {
				normalizedAttrs[attrDef.Name] = value
			}
		}
	}

	return &dto.ValidateCIAttributesResponse{
		IsValid:              len(errors) == 0,
		Errors:               errors,
		Warnings:             warnings,
		NormalizedAttributes: normalizedAttrs,
	}, nil
}

// SearchCIsByAttributes 根据属性搜索CI
func (s *CMDBService) SearchCIsByAttributes(ctx context.Context, req *dto.CIAttributeSearchRequest, tenantID int) (*dto.ListCIsResponse, error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID))

	// 如果指定了CI类型，添加过滤条件
	if req.CITypeID > 0 {
		query = query.Where(configurationitem.CiTypeIDEQ(req.CITypeID))
	}

	// 由于 Ent 不直接支持 JSON 字段的复杂查询，我们先获取所有符合基本条件的 CI
	// 然后在应用层进行属性过滤
	cis, err := query.All(ctx)
	if err != nil {
		s.logger.Errorf("Failed to search CIs: %v", err)
		return nil, err
	}

	// 在应用层过滤属性
	var filteredCIs []*ent.ConfigurationItem
	for _, ci := range cis {
		if s.matchesAttributes(ci.Attributes, req.Attributes) {
			filteredCIs = append(filteredCIs, ci)
		}
	}

	// 应用分页
	start := req.Offset
	end := start + req.Limit
	if req.Limit <= 0 {
		end = len(filteredCIs)
	}
	if start > len(filteredCIs) {
		start = len(filteredCIs)
	}
	if end > len(filteredCIs) {
		end = len(filteredCIs)
	}

	pagedCIs := filteredCIs[start:end]

	// 转换为响应格式
	var ciResponses []*dto.CIResponse
	for _, ci := range pagedCIs {
		ciResponses = append(ciResponses, &dto.CIResponse{
			ID:          ci.ID,
			Name:        ci.Name,
			CITypeID:    ci.CiTypeID,
			Description: ci.Description,
			Status:      ci.Status,
			Attributes:  ci.Attributes,
			TenantID:    ci.TenantID,
			CreatedAt:   ci.CreatedAt,
			UpdatedAt:   ci.UpdatedAt,
		})
	}

	return &dto.ListCIsResponse{
		CIs:   ciResponses,
		Total: len(filteredCIs),
	}, nil
}

// 辅助方法：检查CI属性是否匹配搜索条件
func (s *CMDBService) matchesAttributes(ciAttrs map[string]interface{}, searchAttrs map[string]interface{}) bool {
	for key, searchValue := range searchAttrs {
		ciValue, exists := ciAttrs[key]
		if !exists {
			return false
		}

		// 简单的值比较，可以根据需要扩展为更复杂的匹配逻辑
		if fmt.Sprintf("%v", ciValue) != fmt.Sprintf("%v", searchValue) {
			return false
		}
	}
	return true
}

// 辅助方法：转换为属性定义响应格式
func (s *CMDBService) convertToAttributeDefinitionResponse(attr *ent.CIAttributeDefinition) *dto.CIAttributeDefinitionResponse {
	return &dto.CIAttributeDefinitionResponse{
		ID:              attr.ID,
		Name:            attr.Name,
		DisplayName:     attr.DisplayName,
		Description:     attr.Description,
		DataType:        attr.DataType,
		IsRequired:      attr.IsRequired,
		IsUnique:        attr.IsUnique,
		DefaultValue:    attr.DefaultValue,
		ValidationRules: attr.ValidationRules,
		EnumValues:      attr.EnumValues,
		ReferenceType:   attr.ReferenceType,
		DisplayOrder:    attr.DisplayOrder,
		IsSearchable:    attr.IsSearchable,
		IsSystem:        attr.IsSystem,
		IsActive:        attr.IsActive,
		CITypeID:        attr.CiTypeID,
		TenantID:        attr.TenantID,
		CreatedAt:       attr.CreatedAt,
		UpdatedAt:       attr.UpdatedAt,
	}
}

// 辅助方法：解析默认值
func (s *CMDBService) parseDefaultValue(defaultValue, dataType string) interface{} {
	switch dataType {
	case "number":
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			return val
		}
	case "boolean":
		if val, err := strconv.ParseBool(defaultValue); err == nil {
			return val
		}
	case "date":
		if val, err := time.Parse(time.RFC3339, defaultValue); err == nil {
			return val
		}
	}
	return defaultValue
}

// 辅助方法：验证属性值
func (s *CMDBService) validateAttributeValue(value interface{}, attrDef *ent.CIAttributeDefinition) error {
	switch attrDef.DataType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string value for %s", attrDef.DisplayName)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64:
			// Valid number types
		default:
			return fmt.Errorf("expected number value for %s", attrDef.DisplayName)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean value for %s", attrDef.DisplayName)
		}
	case "enum":
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string value for enum %s", attrDef.DisplayName)
		}
		// 检查是否在枚举值中
		for _, enumVal := range attrDef.EnumValues {
			if enumVal == strValue {
				return nil
			}
		}
		return fmt.Errorf("invalid enum value for %s", attrDef.DisplayName)
	}
	return nil
}
