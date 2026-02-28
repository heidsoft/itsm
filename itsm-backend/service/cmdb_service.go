package service

import (
	"context"

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
		SetStatus(req.Status).
		SetEnvironment(req.Environment).
		SetCriticality(req.Criticality).
		SetTenantID(1) // 默认租户ID

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
func (s *CMDBService) GetCI(ctx context.Context, id int) (*ent.ConfigurationItem, error) {
	return s.client.ConfigurationItem.
		Query().
		Where(configurationitem.ID(id)).
		WithOutgoingRelations().
		WithIncomingRelations().
		Only(ctx)
}

// ListCIs 列出配置项
func (s *CMDBService) ListCIs(ctx context.Context, req *ListCIsRequest) ([]*ent.ConfigurationItem, error) {
	query := s.client.ConfigurationItem.Query()

	if req.CiType != "" {
		query = query.Where(configurationitem.CiType(req.CiType))
	}
	if req.Status != "" {
		query = query.Where(configurationitem.Status(req.Status))
	}
	if req.Environment != "" {
		query = query.Where(configurationitem.Environment(req.Environment))
	}

	return query.
		Offset(req.Offset).
		Limit(req.Limit).
		All(ctx)
}

// CreateRelationship 创建CI关系
func (s *CMDBService) CreateRelationship(ctx context.Context, req *CreateRelationshipRequest) (*ent.CIRelationship, error) {
	create := s.client.CIRelationship.
		Create().
		SetRelationshipType(req.Type).
		SetSourceCiID(req.ParentID).
		SetTargetCiID(req.ChildID)

	if req.Description != nil {
		create = create.SetDescription(*req.Description)
	}

	return create.Save(ctx)
}

// ListRelationships 获取CI关系列表
func (s *CMDBService) ListRelationships(ctx context.Context, tenantID int, ciID *int) ([]*ent.CIRelationship, error) {
	query := s.client.CIRelationship.Query()

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

// GetCITopology 获取CI拓扑
func (s *CMDBService) GetCITopology(ctx context.Context, ciID int, depth int) (*CITopology, error) {
	ci, err := s.GetCI(ctx, ciID)
	if err != nil {
		return nil, err
	}

	topology := &CITopology{
		CI:       ci,
		Children: make([]*CITopology, 0),
	}

	if depth > 0 {
		children, err := s.client.CIRelationship.
			Query().
			Where(cirelationship.SourceCiID(ciID)).
			WithTargetCi().
			All(ctx)
		if err != nil {
			return nil, err
		}

		for _, rel := range children {
			if rel.Edges.TargetCi != nil {
				childTopology, err := s.GetCITopology(ctx, rel.Edges.TargetCi.ID, depth-1)
				if err != nil {
					continue
				}
				topology.Children = append(topology.Children, childTopology)
			}
		}
	}

	return topology, nil
}

// 请求结构体
type CreateCIRequest struct {
	Name            string                  `json:"name"`
	CiType          string                  `json:"ci_type"`
	Status          string                  `json:"status"`
	Environment     string                  `json:"environment"`
	Criticality     string                  `json:"criticality"`
	AssetTag        *string                 `json:"asset_tag,omitempty"`
	SerialNumber    *string                 `json:"serial_number,omitempty"`
	Location        *string                 `json:"location,omitempty"`
	AssignedTo      *string                 `json:"assigned_to,omitempty"`
	OwnedBy         *string                 `json:"owned_by,omitempty"`
	DiscoverySource *string                 `json:"discovery_source,omitempty"`
	Attributes      *map[string]interface{} `json:"attributes,omitempty"`
}

type ListCIsRequest struct {
	CiType      string `json:"ci_type,omitempty"`
	Status      string `json:"status,omitempty"`
	Environment string `json:"environment,omitempty"`
	Offset      int    `json:"offset"`
	Limit       int    `json:"limit"`
}

type CreateRelationshipRequest struct {
	ParentID    int     `json:"parent_id"`
	ChildID     int     `json:"child_id"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
}

type CITopology struct {
	CI       *ent.ConfigurationItem `json:"ci"`
	Children []*CITopology          `json:"children"`
}
