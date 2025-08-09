package service

import (
	"context"
	"encoding/json"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

type CMDBAdvancedService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewCMDBAdvancedService(client *ent.Client, logger *zap.SugaredLogger) *CMDBAdvancedService {
	return &CMDBAdvancedService{
		client: client,
		logger: logger,
	}
}

// CreateCIType 创建CI类型
func (s *CMDBAdvancedService) CreateCIType(ctx context.Context, req *dto.CreateCITypeRequest) (*dto.CITypeResponse, error) {
	// 验证名称唯一性
	exists, err := s.client.CIType.Query().
		Where(citype.Name(req.Name), citype.TenantID(req.TenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("Failed to check CI type existence: %v", err)
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("CI type with name '%s' already exists", req.Name)
	}

	// 将AttributeSchema转换为JSON字符串
	attributeSchemaJSON := ""
	if req.AttributeSchema != nil {
		jsonBytes, err := json.Marshal(req.AttributeSchema)
		if err != nil {
			s.logger.Errorf("Failed to marshal attribute schema: %v", err)
			return nil, err
		}
		attributeSchemaJSON = string(jsonBytes)
	}

	// 创建CI类型
	ciType, err := s.client.CIType.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetAttributeSchema(attributeSchemaJSON).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI type: %v", err)
		return nil, err
	}

	// 将JSON字符串解析回map
	var attributeSchema map[string]interface{}
	if ciType.AttributeSchema != "" {
		if err := json.Unmarshal([]byte(ciType.AttributeSchema), &attributeSchema); err != nil {
			s.logger.Errorf("Failed to unmarshal attribute schema: %v", err)
			attributeSchema = nil
		}
	}

	return &dto.CITypeResponse{
		ID:              ciType.ID,
		Name:            ciType.Name,
		Description:     ciType.Description,
		Icon:            ciType.Icon,
		AttributeSchema: attributeSchema,
		TenantID:        ciType.TenantID,
		CreatedAt:       ciType.CreatedAt,
		UpdatedAt:       ciType.UpdatedAt,
	}, nil
}

// CreateCIRelationship 创建CI关系
func (s *CMDBAdvancedService) CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest) (*dto.CIRelationshipResponse, error) {
	// 验证关系约束
	if err := s.validateRelationshipConstraints(ctx, req); err != nil {
		return nil, err
	}

	// 创建关系
	relationship, err := s.client.CIRelationship.Create().
		SetSourceCiID(req.SourceCIID).
		SetTargetCiID(req.TargetCIID).
		SetRelationshipTypeID(req.RelationshipTypeID).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI relationship: %v", err)
		return nil, err
	}

	return &dto.CIRelationshipResponse{
		ID:                 relationship.ID,
		SourceCIID:         relationship.SourceCiID,
		TargetCIID:         relationship.TargetCiID,
		RelationshipTypeID: relationship.RelationshipTypeID,
		TenantID:           relationship.TenantID,
		CreatedAt:          relationship.CreatedAt,
		UpdatedAt:          relationship.UpdatedAt,
	}, nil
}

// UpdateCILifecycleState 更新CI生命周期状态
func (s *CMDBAdvancedService) UpdateCILifecycleState(ctx context.Context, req *dto.UpdateCILifecycleStateRequest) error {
	// 直接更新CI的状态
	err := s.client.ConfigurationItem.UpdateOneID(req.CIID).
		SetStatus(req.State).
		Exec(ctx)

	if err != nil {
		s.logger.Errorf("Failed to update CI status: %v", err)
		return err
	}

	s.logger.Infof("Updated CI %d lifecycle state to %s", req.CIID, req.State)
	return nil
}

// BatchImportCIs 批量导入CI
func (s *CMDBAdvancedService) BatchImportCIs(ctx context.Context, req *dto.BatchImportCIsRequest) (*dto.BatchImportCIsResponse, error) {
	var successCount, failureCount int
	var errors []string

	for _, ciData := range req.CIs {
		_, err := s.client.ConfigurationItem.Create().
			SetName(ciData.Name).
			SetCiTypeID(ciData.CITypeID).
			SetStatus(ciData.Status).
			SetTenantID(req.TenantID).
			Save(ctx)

		if err != nil {
			failureCount++
			errors = append(errors, fmt.Sprintf("Failed to import CI %s: %v", ciData.Name, err))
			s.logger.Errorf("Failed to import CI %s: %v", ciData.Name, err)
		} else {
			successCount++
		}
	}

	return &dto.BatchImportCIsResponse{
		TotalCount:   len(req.CIs),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Errors:       errors,
	}, nil
}

// validateRelationshipConstraints 验证关系约束
func (s *CMDBAdvancedService) validateRelationshipConstraints(ctx context.Context, req *dto.CreateCIRelationshipRequest) error {
	// 检查源CI和目标CI是否存在
	sourceExists, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.SourceCIID)).
		Exist(ctx)
	if err != nil || !sourceExists {
		return fmt.Errorf("source CI %d does not exist", req.SourceCIID)
	}

	targetExists, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.TargetCIID)).
		Exist(ctx)
	if err != nil || !targetExists {
		return fmt.Errorf("target CI %d does not exist", req.TargetCIID)
	}

	// 检查关系是否已存在
	existingRel, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.SourceCiID(req.SourceCIID),
			cirelationship.TargetCiID(req.TargetCIID),
			cirelationship.RelationshipTypeID(req.RelationshipTypeID),
		).
		First(ctx)

	if err == nil && existingRel != nil {
		return fmt.Errorf("relationship already exists between CI %d and %d", req.SourceCIID, req.TargetCIID)
	}

	return nil
}
