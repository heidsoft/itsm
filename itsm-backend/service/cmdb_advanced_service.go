package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"
)

type CMDBAdvancedService struct {
	client *ent.Client
}

func NewCMDBAdvancedService(client *ent.Client) *CMDBAdvancedService {
	return &CMDBAdvancedService{client: client}
}

// CreateCIType 创建CI类型
func (s *CMDBAdvancedService) CreateCIType(ctx context.Context, req *dto.CreateCITypeRequest) (*dto.CITypeResponse, error) {
	ciType, err := s.client.CIType.Create().
		SetName(req.Name).
		SetDisplayName(req.DisplayName).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetIcon(req.Icon).
		SetAttributeSchema(req.AttributeSchema).
		SetValidationRules(req.ValidationRules).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create CI type: %w", err)
	}

	return &dto.CITypeResponse{
		ID:              ciType.ID,
		Name:            ciType.Name,
		DisplayName:     ciType.DisplayName,
		Description:     ciType.Description,
		Category:        ciType.Category,
		Icon:            ciType.Icon,
		AttributeSchema: ciType.AttributeSchema,
		ValidationRules: ciType.ValidationRules,
		IsSystem:        ciType.IsSystem,
		IsActive:        ciType.IsActive,
		CreatedAt:       ciType.CreatedAt,
		UpdatedAt:       ciType.UpdatedAt,
	}, nil
}

// CreateCIRelationship 创建CI关系
func (s *CMDBAdvancedService) CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest) (*dto.CIRelationshipResponse, error) {
	// 验证关系类型是否允许这种CI类型组合
	relType, err := s.client.CIRelationshipType.Get(ctx, req.RelationshipTypeID)
	if err != nil {
		return nil, fmt.Errorf("relationship type not found: %w", err)
	}

	// 获取源和目标CI的类型
	sourceCi, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.SourceCIID)).
		WithCiType().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("source CI not found: %w", err)
	}

	targetCi, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.TargetCIID)).
		WithCiType().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("target CI not found: %w", err)
	}

	// 验证关系类型约束
	if !s.validateRelationshipConstraints(relType, sourceCi.Edges.CiType.Name, targetCi.Edges.CiType.Name) {
		return nil, fmt.Errorf("relationship not allowed between these CI types")
	}

	relationship, err := s.client.CIRelationship.Create().
		SetSourceCIID(req.SourceCIID).
		SetTargetCIID(req.TargetCIID).
		SetRelationshipTypeID(req.RelationshipTypeID).
		SetProperties(req.Properties).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return &dto.CIRelationshipResponse{
		ID:                 relationship.ID,
		SourceCIID:         relationship.SourceCIID,
		TargetCIID:         relationship.TargetCIID,
		RelationshipTypeID: relationship.RelationshipTypeID,
		Properties:         relationship.Properties,
		Status:             relationship.Status,
		EffectiveFrom:      relationship.EffectiveFrom,
		EffectiveTo:        relationship.EffectiveTo,
		CreatedAt:          relationship.CreatedAt,
	}, nil
}

// GetServiceMap 获取服务图谱
func (s *CMDBAdvancedService) GetServiceMap(ctx context.Context, ciID int, depth int, tenantID int) (*dto.ServiceMapResponse, error) {
	nodes := make(map[int]*dto.ServiceMapNode)
	edges := make([]*dto.ServiceMapEdge, 0)

	// 递归获取关联的CI
	err := s.buildServiceMap(ctx, ciID, depth, tenantID, nodes, &edges, make(map[int]bool))
	if err != nil {
		return nil, err
	}

	// 转换为数组
	nodeList := make([]*dto.ServiceMapNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, node)
	}

	return &dto.ServiceMapResponse{
		Nodes: nodeList,
		Edges: edges,
	}, nil
}

// AnalyzeImpact 影响分析
func (s *CMDBAdvancedService) AnalyzeImpact(ctx context.Context, ciID int, changeType string, tenantID int) (*dto.ImpactAnalysisResponse, error) {
	// 获取所有受影响的CI
	affectedCIs := make([]*dto.AffectedCI, 0)

	// 向上分析（依赖此CI的其他CI）
	upstreamCIs, err := s.getUpstreamCIs(ctx, ciID, tenantID)
	if err != nil {
		return nil, err
	}

	// 向下分析（此CI依赖的其他CI）
	downstreamCIs, err := s.getDownstreamCIs(ctx, ciID, tenantID)
	if err != nil {
		return nil, err
	}

	// 计算影响级别
	for _, ci := range upstreamCIs {
		impactLevel := s.calculateImpactLevel(ci, changeType)
		affectedCIs = append(affectedCIs, &dto.AffectedCI{
			CIID:        ci.ID,
			CIName:      ci.Name,
			CIType:      ci.Edges.CiType.Name,
			ImpactLevel: impactLevel,
			Direction:   "upstream",
		})
	}

	for _, ci := range downstreamCIs {
		impactLevel := s.calculateImpactLevel(ci, changeType)
		affectedCIs = append(affectedCIs, &dto.AffectedCI{
			CIID:        ci.ID,
			CIName:      ci.Name,
			CIType:      ci.Edges.CiType.Name,
			ImpactLevel: impactLevel,
			Direction:   "downstream",
		})
	}

	return &dto.ImpactAnalysisResponse{
		SourceCIID:  ciID,
		ChangeType:  changeType,
		AffectedCIs: affectedCIs,
		TotalCount:  len(affectedCIs),
		AnalyzedAt:  time.Now(),
	}, nil
}

// UpdateCILifecycleState 更新CI生命周期状态
func (s *CMDBAdvancedService) UpdateCILifecycleState(ctx context.Context, req *dto.UpdateCILifecycleRequest) error {
	// 记录状态变更
	_, err := s.client.CILifecycleState.Create().
		SetCIID(req.CIID).
		SetState(req.NewState).
		SetSubState(req.SubState).
		SetReason(req.Reason).
		SetChangedBy(req.ChangedBy).
		SetMetadata(req.Metadata).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to record lifecycle state: %w", err)
	}

	// 更新CI的当前状态
	err = s.client.ConfigurationItem.UpdateOneID(req.CIID).
		SetLifecycleState(req.NewState).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update CI lifecycle state: %w", err)
	}

	return nil
}

// BatchImportCIs 批量导入CI
func (s *CMDBAdvancedService) BatchImportCIs(ctx context.Context, req *dto.BatchImportCIRequest) (*dto.BatchImportResponse, error) {
	results := &dto.BatchImportResponse{
		Total:     len(req.CIs),
		Succeeded: 0,
		Failed:    0,
		Errors:    make([]string, 0),
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	for i, ciData := range req.CIs {
		_, err := tx.ConfigurationItem.Create().
			SetName(ciData.Name).
			SetDisplayName(ciData.DisplayName).
			SetDescription(ciData.Description).
			SetCiTypeID(ciData.CITypeID).
			SetSerialNumber(ciData.SerialNumber).
			SetAssetTag(ciData.AssetTag).
			SetStatus(ciData.Status).
			SetLifecycleState(ciData.LifecycleState).
			SetBusinessService(ciData.BusinessService).
			SetOwner(ciData.Owner).
			SetEnvironment(ciData.Environment).
			SetLocation(ciData.Location).
			SetAttributes(ciData.Attributes).
			SetDiscoverySource(map[string]interface{}{
				"source":   "batch_import",
				"batch_id": req.BatchID,
			}).
			SetTenantID(req.TenantID).
			Save(ctx)

		if err != nil {
			results.Failed++
			results.Errors = append(results.Errors, fmt.Sprintf("Row %d: %v", i+1, err))
		} else {
			results.Succeeded++
		}
	}

	if results.Failed == 0 {
		err = tx.Commit()
		if err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return results, nil
}

// 辅助方法
func (s *CMDBAdvancedService) validateRelationshipConstraints(relType *ent.CIRelationshipType, sourceType, targetType string) bool {
	// 检查源CI类型约束
	if len(relType.SourceCiTypes) > 0 {
		allowed := false
		for _, allowedType := range relType.SourceCiTypes {
			if allowedType == sourceType {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	// 检查目标CI类型约束
	if len(relType.TargetCiTypes) > 0 {
		allowed := false
		for _, allowedType := range relType.TargetCiTypes {
			if allowedType == targetType {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

func (s *CMDBAdvancedService) buildServiceMap(ctx context.Context, ciID, depth, tenantID int, nodes map[int]*dto.ServiceMapNode, edges *[]*dto.ServiceMapEdge, visited map[int]bool) error {
	if depth <= 0 || visited[ciID] {
		return nil
	}

	visited[ciID] = true

	// 获取CI信息
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(ciID)).
		WithCiType().
		Only(ctx)
	if err != nil {
		return err
	}

	// 添加节点
	nodes[ciID] = &dto.ServiceMapNode{
		ID:       ci.ID,
		Name:     ci.Name,
		Type:     ci.Edges.CiType.Name,
		Status:   ci.Status,
		Category: ci.Edges.CiType.Category,
	}

	// 获取关联关系
	relationships, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.Or(
				cirelationship.SourceCIID(ciID),
				cirelationship.TargetCIID(ciID),
			),
			cirelationship.TenantID(tenantID),
		).
		WithRelationshipType().
		All(ctx)

	if err != nil {
		return err
	}

	// 处理关系和递归
	for _, rel := range relationships {
		*edges = append(*edges, &dto.ServiceMapEdge{
			SourceID:         rel.SourceCIID,
			TargetID:         rel.TargetCIID,
			RelationshipType: rel.Edges.RelationshipType.Name,
			Properties:       rel.Properties,
		})

		// 递归处理相关CI
		var nextCIID int
		if rel.SourceCIID == ciID {
			nextCIID = rel.TargetCIID
		} else {
			nextCIID = rel.SourceCIID
		}

		err = s.buildServiceMap(ctx, nextCIID, depth-1, tenantID, nodes, edges, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *CMDBAdvancedService) getUpstreamCIs(ctx context.Context, ciID, tenantID int) ([]*ent.ConfigurationItem, error) {
	// 获取依赖此CI的其他CI（向上依赖）
	return s.client.ConfigurationItem.Query().
		Where(
			configurationitem.HasOutgoingRelationshipsWith(
				cirelationship.TargetCIID(ciID),
				cirelationship.TenantID(tenantID),
			),
		).
		WithCiType().
		All(ctx)
}

func (s *CMDBAdvancedService) getDownstreamCIs(ctx context.Context, ciID, tenantID int) ([]*ent.ConfigurationItem, error) {
	// 获取此CI依赖的其他CI（向下依赖）
	return s.client.ConfigurationItem.Query().
		Where(
			configurationitem.HasIncomingRelationshipsWith(
				cirelationship.SourceCIID(ciID),
				cirelationship.TenantID(tenantID),
			),
		).
		WithCiType().
		All(ctx)
}

func (s *CMDBAdvancedService) calculateImpactLevel(ci *ent.ConfigurationItem, changeType string) string {
	// 根据CI类型、业务重要性等计算影响级别
	switch ci.Edges.CiType.Category {
	case "infrastructure":
		return "high"
	case "application":
		return "medium"
	default:
		return "low"
	}
}
