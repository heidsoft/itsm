package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
)

type CMDBService struct {
	client *ent.Client
}

func NewCMDBService(client *ent.Client) *CMDBService {
	return &CMDBService{client: client}
}

// CreateCI 创建配置项
func (s *CMDBService) CreateCI(ctx context.Context, req *CreateCIRequest) (*ent.ConfigurationItem, error) {
	create := s.client.ConfigurationItem.
		Create().
		SetName(req.Name).
		SetCiType(req.CiType).
		SetCiTypeID(req.CiTypeID).
		SetStatus(req.Status).
		SetEnvironment(req.Environment).
		SetCriticality(req.Criticality).
		SetTenantID(req.TenantID)

	if req.AssetTag != nil {
		create = create.SetAssetTag(*req.AssetTag)
	}
	if req.SerialNumber != nil {
		create = create.SetSerialNumber(*req.SerialNumber)
	}
	if req.Location != nil {
		create = create.SetLocation(*req.Location)
	}
	if req.AssignedTo != nil {
		create = create.SetAssignedTo(*req.AssignedTo)
	}
	if req.OwnedBy != nil {
		create = create.SetOwnedBy(*req.OwnedBy)
	}
	if req.DiscoverySource != nil {
		create = create.SetDiscoverySource(*req.DiscoverySource)
	}
	if req.Attributes != nil {
		create = create.SetAttributes(*req.Attributes)
	}

	return create.Save(ctx)
}

// GetCI 获取配置项
func (s *CMDBService) GetCI(ctx context.Context, id int, tenantID int) (*ent.ConfigurationItem, error) {
	return s.client.ConfigurationItem.
		Query().
		Where(configurationitem.ID(id), configurationitem.TenantID(tenantID)).
		WithOutgoingRelations().
		WithIncomingRelations().
		Only(ctx)
}

// ListCIs 列出配置项，返回列表和总数
func (s *CMDBService) ListCIs(ctx context.Context, req *ListCIsRequest) ([]*ent.ConfigurationItem, int, error) {
	query := s.client.ConfigurationItem.Query()

	if req.TenantID > 0 {
		query = query.Where(configurationitem.TenantID(req.TenantID))
	}
	if req.CiType != "" {
		query = query.Where(configurationitem.CiType(req.CiType))
	}
	if req.Status != "" {
		query = query.Where(configurationitem.Status(req.Status))
	}
	if req.Environment != "" {
		query = query.Where(configurationitem.Environment(req.Environment))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := query.
		Offset(req.Offset).
		Limit(req.Limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// CreateRelationship 创建CI关系（SEC-004: 校验 source/target CI 租户归属 + 设置 tenant_id）
func (s *CMDBService) CreateRelationship(ctx context.Context, req *CreateRelationshipRequest) (*ent.CIRelationship, error) {
	// SEC-004 修复：校验 source CI 属于当前租户
	sourceCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.SourceCIID), configurationitem.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("源CI不存在或不属于当前租户")
		}
		return nil, fmt.Errorf("查询源CI失败: %w", err)
	}
	_ = sourceCI

	// SEC-004 修复：校验 target CI 属于当前租户
	targetCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.TargetCIID), configurationitem.TenantID(req.TenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("目标CI不存在或不属于当前租户")
		}
		return nil, fmt.Errorf("查询目标CI失败: %w", err)
	}
	_ = targetCI

	create := s.client.CIRelationship.
		Create().
		SetRelationshipType(req.Type).
		SetSourceCiID(req.SourceCIID).
		SetTargetCiID(req.TargetCIID)

	// SEC-001 修复：创建关系时设置 tenant_id
	if req.TenantID > 0 {
		create = create.SetTenantID(req.TenantID)
	}

	if req.Description != nil {
		create = create.SetDescription(*req.Description)
	}

	return create.Save(ctx)
}

// DeleteRelationship 删除CI关系
func (s *CMDBService) DeleteRelationship(ctx context.Context, id int) error {
	return s.client.CIRelationship.DeleteOneID(id).Exec(ctx)
}

// UpdateRelationship 更新CI关系
func (s *CMDBService) UpdateRelationship(ctx context.Context, id int, req *UpdateRelationshipRequest) (*ent.CIRelationship, error) {
	update := s.client.CIRelationship.UpdateOneID(id)
	if req.Type != "" {
		update = update.SetRelationshipType(req.Type)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	return update.Save(ctx)
}

// CountCIs 统计配置项数量
func (s *CMDBService) CountCIs(ctx context.Context, tenantID int) (int, error) {
	q := s.client.ConfigurationItem.Query()
	if tenantID > 0 {
		q = q.Where(configurationitem.TenantID(tenantID))
	}
	return q.Count(ctx)
}

// ListRelationships 获取CI关系列表（SEC-001: 添加租户隔离过滤）
func (s *CMDBService) ListRelationships(ctx context.Context, tenantID int, ciID *int) ([]*ent.CIRelationship, error) {
	query := s.client.CIRelationship.Query()

	// SEC-001 修复：强制租户隔离，防止跨租户数据泄露
	if tenantID > 0 {
		query = query.Where(cirelationship.TenantID(tenantID))
	}

	if ciID != nil {
		// 如果指定了CIID，查询与该CI相关的所有关系
		query = query.Where(
			cirelationship.Or(
				cirelationship.SourceCiID(*ciID),
				cirelationship.TargetCiID(*ciID),
			),
		)
	}

	return query.All(ctx)
}

// GetCITopology 获取CI拓扑（SEC-001: 递归查询添加租户隔离过滤）
func (s *CMDBService) GetCITopology(ctx context.Context, ciID int, tenantID int, depth int) (*CITopology, error) {
	ci, err := s.GetCI(ctx, ciID, tenantID)
	if err != nil {
		return nil, err
	}

	topology := &CITopology{
		CI:       ci,
		Children: make([]*CITopology, 0),
	}

	if depth > 0 {
		query := s.client.CIRelationship.Query().
			Where(cirelationship.SourceCiID(ciID))

		// SEC-001 修复：递归查询中也添加租户隔离过滤
		if tenantID > 0 {
			query = query.Where(cirelationship.TenantID(tenantID))
		}

		children, err := query.WithTargetCi().All(ctx)
		if err != nil {
			return nil, err
		}

		for _, rel := range children {
			if rel.Edges.TargetCi != nil {
				childTopology, err := s.GetCITopology(ctx, rel.Edges.TargetCi.ID, tenantID, depth-1)
				if err != nil {
					continue
				}
				topology.Children = append(topology.Children, childTopology)
			}
		}
	}

	return topology, nil
}

// 请求结构体 — 以下类型别名指向 dto 包，保持向后兼容性
// 调用方可继续使用 service.CreateCIRequest 等，实际引用 dto 包定义

// CreateCIRequest 创建配置项请求（已迁移至 dto.CMDBCreateCIRequest）
type CreateCIRequest = dto.CMDBCreateCIRequest

// ListCIsRequest 列表查询请求（已迁移至 dto.CMDBListCIsRequest）
type ListCIsRequest = dto.CMDBListCIsRequest

// CreateRelationshipRequest 创建 CI 关系请求（已迁移至 dto.CMDBCreateRelationshipRequest）
type CreateRelationshipRequest = dto.CMDBCreateRelationshipRequest

// UpdateRelationshipRequest 更新 CI 关系请求（已迁移至 dto.CMDBUpdateRelationshipRequest）
type UpdateRelationshipRequest = dto.CMDBUpdateRelationshipRequest

// CITopology CI 拓扑结构
type CITopology struct {
	CI       *ent.ConfigurationItem `json:"ci"`
	Children []*CITopology          `json:"children"`
}
