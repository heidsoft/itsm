package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// CIRelationshipService CI关系服务
type CIRelationshipService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCIRelationshipService 创建CI关系服务
func NewCIRelationshipService(client *ent.Client, logger *zap.SugaredLogger) *CIRelationshipService {
	return &CIRelationshipService{
		client: client,
		logger: logger,
	}
}

// CreateCIRelationship 创建CI关系
func (s *CIRelationshipService) CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest, tenantID int) (*dto.CIRelationshipResponse, error) {
	// 检查源CI是否存在
	sourceCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(req.SourceCIID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("source CI not found")
		}
		s.logger.Errorw("Failed to get source CI", "error", err, "ci_id", req.SourceCIID)
		return nil, fmt.Errorf("failed to get source CI: %w", err)
	}

	// 检查目标CI是否存在
	targetCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(req.TargetCIID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("target CI not found")
		}
		s.logger.Errorw("Failed to get target CI", "error", err, "ci_id", req.TargetCIID)
		return nil, fmt.Errorf("failed to get target CI: %w", err)
	}

	// 检查关系是否已存在
	exists, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.SourceCiIDEQ(req.SourceCIID),
			cirelationship.TargetCiIDEQ(req.TargetCIID),
			cirelationship.RelationshipTypeEQ(string(req.RelationshipType)),
			cirelationship.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check relation existence", "error", err)
		return nil, fmt.Errorf("failed to check relation existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("relationship already exists between CI %d and CI %d with type %s",
			req.SourceCIID, req.TargetCIID, req.RelationshipType)
	}

	// 创建关系
	create := s.client.CIRelationship.Create().
		SetRelationshipType(string(req.RelationshipType)).
		SetSourceCiID(req.SourceCIID).
		SetTargetCiID(req.TargetCIID).
		SetTenantID(tenantID)

	if req.Strength != "" {
		create.SetStrength(cirelationship.Strength(req.Strength))
	}
	if req.ImpactLevel != "" {
		create.SetImpactLevel(cirelationship.ImpactLevel(req.ImpactLevel))
	}
	if req.Description != "" {
		create.SetDescription(req.Description)
	}
	if req.Metadata != nil {
		create.SetMetadata(req.Metadata)
	}
	if req.IsDiscovered != nil {
		create.SetIsDiscovered(*req.IsDiscovered)
	}

	relation, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI relationship", "error", err,
			"source_ci_id", req.SourceCIID, "target_ci_id", req.TargetCIID, "type", req.RelationshipType)
		return nil, fmt.Errorf("failed to create CI relationship: %w", err)
	}

	// 加载关联的CI信息
	relation, err = s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(relation.ID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to load created relation", "error", err, "relation_id", relation.ID)
		return nil, fmt.Errorf("failed to load created relation: %w", err)
	}

	s.logger.Infow("CI relationship created successfully", "relation_id", relation.ID,
		"source_ci_id", sourceCI.ID, "target_ci_id", targetCI.ID, "type", req.RelationshipType)
	return dto.ToCIRelationshipResponse(relation), nil
}

// GetCIRelationshipByID 根据ID获取CI关系
func (s *CIRelationshipService) GetCIRelationshipByID(ctx context.Context, id, tenantID int) (*dto.CIRelationshipResponse, error) {
	relation, err := s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(id), cirelationship.TenantIDEQ(tenantID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get CI relationship", "error", err, "relation_id", id)
		return nil, fmt.Errorf("failed to get CI relationship: %w", err)
	}

	return dto.ToCIRelationshipResponse(relation), nil
}

// ListCIRelationshipsByCIID 根据CI ID获取关系列表
func (s *CIRelationshipService) ListCIRelationshipsByCIID(ctx context.Context, ciID, tenantID int, direction string) ([]*dto.CIRelationshipResponse, error) {
	query := s.client.CIRelationship.Query().
		Where(cirelationship.TenantIDEQ(tenantID), cirelationship.IsActiveEQ(true))

	if direction == "outgoing" {
		query = query.Where(cirelationship.SourceCiIDEQ(ciID))
	} else if direction == "incoming" {
		query = query.Where(cirelationship.TargetCiIDEQ(ciID))
	} else {
		query = query.Where(
			cirelationship.Or(
				cirelationship.SourceCiIDEQ(ciID),
				cirelationship.TargetCiIDEQ(ciID),
			),
		)
	}

	relations, err := query.
		WithSourceCi().
		WithTargetCi().
		Order(ent.Desc(cirelationship.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI relationships", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to list CI relationships: %w", err)
	}

	return dto.ToCIRelationshipResponseList(relations), nil
}

// ListAllCIRelationships 获取所有CI关系列表
func (s *CIRelationshipService) ListAllCIRelationships(ctx context.Context, tenantID int, page, pageSize int, relationType string) (*dto.CIRelationshipListResponse, error) {
	query := s.client.CIRelationship.Query().
		Where(cirelationship.TenantIDEQ(tenantID), cirelationship.IsActiveEQ(true))

	if relationType != "" {
		query = query.Where(cirelationship.RelationshipTypeEQ(relationType))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI relationships", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count CI relationships: %w", err)
	}

	relations, err := query.
		WithSourceCi().
		WithTargetCi().
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(cirelationship.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI relationships", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list CI relationships: %w", err)
	}

	return &dto.CIRelationshipListResponse{
		Items: dto.ToCIRelationshipResponseList(relations),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// UpdateCIRelationship 更新CI关系
func (s *CIRelationshipService) UpdateCIRelationship(ctx context.Context, id, tenantID int, req *dto.UpdateCIRelationshipRequest) (*dto.CIRelationshipResponse, error) {
	update := s.client.CIRelationship.UpdateOneID(id).
		Where(cirelationship.TenantIDEQ(tenantID))

	if req.Strength != nil {
		update.SetStrength(cirelationship.Strength(*req.Strength))
	}
	if req.ImpactLevel != nil {
		update.SetImpactLevel(cirelationship.ImpactLevel(*req.ImpactLevel))
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Metadata != nil {
		update.SetMetadata(*req.Metadata)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	relation, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI relationship", "error", err, "relation_id", id)
		return nil, fmt.Errorf("failed to update CI relationship: %w", err)
	}

	// 重新加载关联信息
	relation, err = s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(relation.ID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to load updated relation", "error", err, "relation_id", relation.ID)
		return nil, fmt.Errorf("failed to load updated relation: %w", err)
	}

	s.logger.Infow("CI relationship updated successfully", "relation_id", relation.ID, "tenant_id", tenantID)
	return dto.ToCIRelationshipResponse(relation), nil
}

// DeleteCIRelationship 删除CI关系
func (s *CIRelationshipService) DeleteCIRelationship(ctx context.Context, id, tenantID int) error {
	err := s.client.CIRelationship.DeleteOneID(id).
		Where(cirelationship.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete CI relationship", "error", err, "relation_id", id)
		return fmt.Errorf("failed to delete CI relationship: %w", err)
	}

	s.logger.Infow("CI relationship deleted successfully", "relation_id", id, "tenant_id", tenantID)
	return nil
}

// GetCIImpactAnalysis 获取CI影响分析
func (s *CIRelationshipService) GetCIImpactAnalysis(ctx context.Context, ciID, tenantID int, maxDepth int) (*dto.CIImpactAnalysisResponse, error) {
	// 检查CI是否存在
	_, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get CI for impact analysis", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	// 使用广度优先搜索遍历影响链
	visited := make(map[int]bool)
	queue := []struct {
		ciID  int
		depth int
		path  []int
	}{{ciID: ciID, depth: 0, path: []int{ciID}}}

	impactedCIs := []*dto.ImpactedCI{}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.ciID] {
			continue
		}
		visited[current.ciID] = true

		// 如果不是根节点，加入影响列表
		if current.depth > 0 {
			ci, err := s.client.ConfigurationItem.Query().
				Where(configurationitem.IDEQ(current.ciID), configurationitem.TenantIDEQ(tenantID)).
				First(ctx)
			if err != nil {
				s.logger.Warnw("Failed to get impacted CI", "error", err, "ci_id", current.ciID)
				continue
			}
			impactedCIs = append(impactedCIs, &dto.ImpactedCI{
				CI:    dto.ToCIResponse(ci),
				Depth: current.depth,
				Path:  current.path,
			})
		}

		// 如果达到最大深度，停止遍历
		if current.depth >= maxDepth {
			continue
		}

		// 查找所有直接受影响的CI（当前CI是源，关系类型是影响类）
		relations, err := s.client.CIRelationship.Query().
			Where(
				cirelationship.SourceCiIDEQ(current.ciID),
				cirelationship.TenantIDEQ(tenantID),
				cirelationship.IsActiveEQ(true),
				cirelationship.RelationshipTypeIn("impacts", "depends_on", "uses"),
			).
			WithTargetCi().
			All(ctx)
		if err != nil {
			s.logger.Errorw("Failed to get outgoing impact relations", "error", err, "ci_id", current.ciID)
			continue
		}

		// 将受影响的CI加入队列
		for _, rel := range relations {
			if !visited[rel.TargetCiID] {
				newPath := make([]int, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, rel.TargetCiID)
				queue = append(queue, struct {
					ciID  int
					depth int
					path  []int
				}{ciID: rel.TargetCiID, depth: current.depth + 1, path: newPath})
			}
		}
	}

	return &dto.CIImpactAnalysisResponse{
		SourceCIID:    ciID,
		ImpactedCIs:   impactedCIs,
		TotalImpacted: len(impactedCIs),
	}, nil
}
