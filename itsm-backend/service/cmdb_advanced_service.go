package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
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
	ciType, err := s.client.CIType.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetAttributeSchema(req.AttributeSchema).
		SetIsSystem(req.IsSystemDefined). // 修改为 SetIsSystem
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI type: %v", err)
		return nil, err
	}

	return &dto.CITypeResponse{
		ID:              ciType.ID,
		Name:            ciType.Name,
		Description:     ciType.Description,
		Icon:            ciType.Icon,
		AttributeSchema: ciType.AttributeSchema,
		IsSystemDefined: ciType.IsSystem, // 修改为 IsSystem
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
		SetProperties(req.Properties).
		SetEffectiveFrom(req.EffectiveFrom).
		SetNillableEffectiveTo(req.EffectiveTo). // 使用 SetNillableEffectiveTo
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
		Properties:         relationship.Properties,
		EffectiveFrom:      relationship.EffectiveFrom,
		EffectiveTo:        &relationship.EffectiveTo, // 添加 & 取地址符号
		TenantID:           relationship.TenantID,
		CreatedAt:          relationship.CreatedAt,
		UpdatedAt:          relationship.UpdatedAt,
	}, nil
}

// GetServiceMap 获取服务图谱
func (s *CMDBAdvancedService) GetServiceMap(ctx context.Context, req *dto.GetServiceMapRequest) (*dto.ServiceMapResponse, error) {
	// 获取根CI
	rootCI, err := s.client.ConfigurationItem.Get(ctx, req.RootCIID)
	if err != nil {
		s.logger.Errorf("Failed to get root CI: %v", err)
		return nil, err
	}

	// 构建服务图谱
	serviceMap, err := s.buildServiceMap(ctx, rootCI, req.Depth, req.TenantID)
	if err != nil {
		return nil, err
	}

	return &dto.ServiceMapResponse{
		RootCI:   serviceMap.RootCI,
		Nodes:    serviceMap.Nodes,
		Edges:    serviceMap.Edges,
		Metadata: serviceMap.Metadata,
	}, nil
}

// AnalyzeImpact 影响分析
func (s *CMDBAdvancedService) AnalyzeImpact(ctx context.Context, req *dto.AnalyzeImpactRequest) (*dto.ImpactAnalysisResponse, error) {
	// 获取源CI
	sourceCI, err := s.client.ConfigurationItem.Get(ctx, req.SourceCIID)
	if err != nil {
		s.logger.Errorf("Failed to get source CI: %v", err)
		return nil, err
	}

	// 获取上游影响
	upstreamImpacts, err := s.getUpstreamCIs(ctx, req.SourceCIID, req.Depth, req.TenantID)
	if err != nil {
		return nil, err
	}

	// 获取下游影响
	downstreamImpacts, err := s.getDownstreamCIs(ctx, req.SourceCIID, req.Depth, req.TenantID)
	if err != nil {
		return nil, err
	}

	// 计算影响级别
	impactLevel := s.calculateImpactLevel(upstreamImpacts, downstreamImpacts)

	return &dto.ImpactAnalysisResponse{
		SourceCI:          sourceCI,
		UpstreamImpacts:   upstreamImpacts,
		DownstreamImpacts: downstreamImpacts,
		ImpactLevel:       impactLevel,
		AnalysisTime:      time.Now(),
	}, nil
}

// UpdateCILifecycleState 更新CI生命周期状态
func (s *CMDBAdvancedService) UpdateCILifecycleState(ctx context.Context, req *dto.UpdateCILifecycleStateRequest) error {
	// 创建生命周期状态记录
	_, err := s.client.CILifecycleState.Create().
		SetCiID(req.CIID).
		SetState(req.State).
		SetReason(req.Reason).
		SetChangedBy(req.ChangedBy).
		SetChangedAt(req.ChangedAt).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("Failed to create CI lifecycle state: %v", err)
		return err
	}

	// 更新CI的当前状态
	err = s.client.ConfigurationItem.UpdateOneID(req.CIID).
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
			SetAttributes(ciData.Attributes).
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

// buildServiceMap 构建服务图谱
func (s *CMDBAdvancedService) buildServiceMap(ctx context.Context, rootCI *ent.ConfigurationItem, depth int, tenantID int) (*dto.ServiceMapData, error) {
	nodes := []*dto.ServiceMapNode{{
		ID:       rootCI.ID,
		Name:     rootCI.Name,
		Type:     "root",
		Level:    0,
		Metadata: rootCI.Attributes,
	}}

	var edges []*dto.ServiceMapEdge
	visited := make(map[int]bool)
	visited[rootCI.ID] = true

	// 递归构建图谱
	err := s.buildServiceMapRecursive(ctx, rootCI.ID, 0, depth, tenantID, &nodes, &edges, visited)
	if err != nil {
		return nil, err
	}

	return &dto.ServiceMapData{
		RootCI: &dto.ServiceMapNode{
			ID:       rootCI.ID,
			Name:     rootCI.Name,
			Type:     "root",
			Level:    0,
			Metadata: rootCI.Attributes,
		},
		Nodes: nodes,
		Edges: edges,
		Metadata: map[string]interface{}{
			"totalNodes": len(nodes),
			"totalEdges": len(edges),
			"maxDepth":   depth,
		},
	}, nil
}

// buildServiceMapRecursive 递归构建服务图谱
func (s *CMDBAdvancedService) buildServiceMapRecursive(ctx context.Context, ciID, currentLevel, maxDepth, tenantID int, nodes *[]*dto.ServiceMapNode, edges *[]*dto.ServiceMapEdge, visited map[int]bool) error {
	if currentLevel >= maxDepth {
		return nil
	}

	// 获取相关关系
	relationships, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.Or(
				cirelationship.SourceCiID(ciID),
				cirelationship.TargetCiID(ciID),
			),
			cirelationship.TenantID(tenantID),
		).
		WithSourceCi().
		WithTargetCi().
		All(ctx)

	if err != nil {
		return err
	}

	for _, rel := range relationships {
		var nextCIID int
		var nextCI *ent.ConfigurationItem
		var edgeDirection string

		if rel.SourceCiID == ciID {
			nextCIID = rel.TargetCiID
			nextCI = rel.Edges.TargetCi
			edgeDirection = "outgoing"
		} else {
			nextCIID = rel.SourceCiID
			nextCI = rel.Edges.SourceCi
			edgeDirection = "incoming"
		}

		if !visited[nextCIID] {
			visited[nextCIID] = true

			// 添加节点
			*nodes = append(*nodes, &dto.ServiceMapNode{
				ID:       nextCI.ID,
				Name:     nextCI.Name,
				Type:     "ci",
				Level:    currentLevel + 1,
				Metadata: nextCI.Attributes,
			})

			// 递归处理下一级
			err = s.buildServiceMapRecursive(ctx, nextCIID, currentLevel+1, maxDepth, tenantID, nodes, edges, visited)
			if err != nil {
				return err
			}
		}

		// 添加边
		*edges = append(*edges, &dto.ServiceMapEdge{
			ID:       rel.ID,
			SourceID: rel.SourceCiID,
			TargetID: rel.TargetCiID,
			Type:     edgeDirection,
			Metadata: rel.Properties,
		})
	}

	return nil
}

// getUpstreamCIs 获取上游CI
func (s *CMDBAdvancedService) getUpstreamCIs(ctx context.Context, ciID, depth, tenantID int) ([]*dto.ImpactedCI, error) {
	var upstreamCIs []*dto.ImpactedCI
	visited := make(map[int]bool)

	err := s.getUpstreamCIsRecursive(ctx, ciID, 0, depth, tenantID, &upstreamCIs, visited)
	return upstreamCIs, err
}

// getUpstreamCIsRecursive 递归获取上游CI
func (s *CMDBAdvancedService) getUpstreamCIsRecursive(ctx context.Context, ciID, currentLevel, maxDepth, tenantID int, upstreamCIs *[]*dto.ImpactedCI, visited map[int]bool) error {
	if currentLevel >= maxDepth {
		return nil
	}

	// 获取指向当前CI的关系（上游）
	relationships, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.TargetCiID(ciID),
			cirelationship.TenantID(tenantID),
		).
		WithSourceCi().
		All(ctx)

	if err != nil {
		return err
	}

	for _, rel := range relationships {
		if !visited[rel.SourceCiID] {
			visited[rel.SourceCiID] = true

			*upstreamCIs = append(*upstreamCIs, &dto.ImpactedCI{
				CI:           rel.Edges.SourceCi,
				ImpactLevel:  currentLevel + 1,
				Relationship: rel,
			})

			// 递归获取更上游的CI
			err = s.getUpstreamCIsRecursive(ctx, rel.SourceCiID, currentLevel+1, maxDepth, tenantID, upstreamCIs, visited)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getDownstreamCIs 获取下游CI
func (s *CMDBAdvancedService) getDownstreamCIs(ctx context.Context, ciID, depth, tenantID int) ([]*dto.ImpactedCI, error) {
	var downstreamCIs []*dto.ImpactedCI
	visited := make(map[int]bool)

	err := s.getDownstreamCIsRecursive(ctx, ciID, 0, depth, tenantID, &downstreamCIs, visited)
	return downstreamCIs, err
}

// getDownstreamCIsRecursive 递归获取下游CI
func (s *CMDBAdvancedService) getDownstreamCIsRecursive(ctx context.Context, ciID, currentLevel, maxDepth, tenantID int, downstreamCIs *[]*dto.ImpactedCI, visited map[int]bool) error {
	if currentLevel >= maxDepth {
		return nil
	}

	// 获取从当前CI出发的关系（下游）
	relationships, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.SourceCiID(ciID),
			cirelationship.TenantID(tenantID),
		).
		WithTargetCi().
		All(ctx)

	if err != nil {
		return err
	}

	for _, rel := range relationships {
		if !visited[rel.TargetCiID] {
			visited[rel.TargetCiID] = true

			*downstreamCIs = append(*downstreamCIs, &dto.ImpactedCI{
				CI:           rel.Edges.TargetCi,
				ImpactLevel:  currentLevel + 1,
				Relationship: rel,
			})

			// 递归获取更下游的CI
			err = s.getDownstreamCIsRecursive(ctx, rel.TargetCiID, currentLevel+1, maxDepth, tenantID, downstreamCIs, visited)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// calculateImpactLevel 计算影响级别
func (s *CMDBAdvancedService) calculateImpactLevel(upstreamCIs, downstreamCIs []*dto.ImpactedCI) string {
	totalImpacted := len(upstreamCIs) + len(downstreamCIs)

	switch {
	case totalImpacted == 0:
		return "无影响"
	case totalImpacted <= 5:
		return "低影响"
	case totalImpacted <= 15:
		return "中等影响"
	case totalImpacted <= 30:
		return "高影响"
	default:
		return "严重影响"
	}
}
