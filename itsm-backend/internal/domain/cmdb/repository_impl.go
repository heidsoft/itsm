package cmdb

import (
	"context"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Map ent CI to domain CI
func toCIDomain(e *ent.ConfigurationItem) *ConfigurationItem {
	if e == nil {
		return nil
	}
	return &ConfigurationItem{
		ID:           e.ID,
		Name:         e.Name,
		Description:  e.Description,
		Type:         e.Type,
		Status:       e.Status,
		Location:     e.Location,
		SerialNumber: e.SerialNumber,
		Model:        e.Model,
		Vendor:       e.Vendor,
		CITypeID:     e.CiTypeID,
		TenantID:     e.TenantID,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

// Map ent CIType to domain CIType
func toTypeDomain(e *ent.CIType) *CIType {
	if e == nil {
		return nil
	}
	return &CIType{
		ID:              e.ID,
		Name:            e.Name,
		Description:     e.Description,
		Icon:            e.Icon,
		Color:           e.Color,
		AttributeSchema: e.AttributeSchema,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func (r *EntRepository) CreateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	e, err := r.client.ConfigurationItem.Create().
		SetName(ci.Name).
		SetDescription(ci.Description).
		SetType(ci.Type).
		SetStatus(ci.Status).
		SetLocation(ci.Location).
		SetSerialNumber(ci.SerialNumber).
		SetModel(ci.Model).
		SetVendor(ci.Vendor).
		SetCiTypeID(ci.CITypeID).
		SetTenantID(ci.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) GetCI(ctx context.Context, id int, tenantID int) (*ConfigurationItem, error) {
	e, err := r.client.ConfigurationItem.Query().
		Where(configurationitem.ID(id), configurationitem.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) ListCIs(ctx context.Context, tenantID int, page, size int, ciTypeID int, status string) ([]*ConfigurationItem, int, error) {
	q := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID))
	if ciTypeID > 0 {
		q = q.Where(configurationitem.CiTypeID(ciTypeID))
	}
	if status != "" {
		q = q.Where(configurationitem.Status(status))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(configurationitem.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var results []*ConfigurationItem
	for _, e := range es {
		results = append(results, toCIDomain(e))
	}
	return results, total, nil
}

func (r *EntRepository) UpdateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	e, err := r.client.ConfigurationItem.UpdateOneID(ci.ID).
		SetName(ci.Name).
		SetDescription(ci.Description).
		SetType(ci.Type).
		SetStatus(ci.Status).
		SetLocation(ci.Location).
		SetSerialNumber(ci.SerialNumber).
		SetModel(ci.Model).
		SetVendor(ci.Vendor).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) DeleteCI(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.ConfigurationItem.Delete().
		Where(configurationitem.ID(id), configurationitem.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	total, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID)).Count(ctx)
	active, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("active")).Count(ctx)
	inactive, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("inactive")).Count(ctx)
	maintenance, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("maintenance")).Count(ctx)

	// Simple type distribution
	dist := make(map[string]int)
	// We might need a aggregation query for this, but for now we can do a simplified version or skip if too complex for Ent raw

	return &Stats{
		TotalCount:       total,
		ActiveCount:      active,
		InactiveCount:    inactive,
		MaintenanceCount: maintenance,
		TypeDistribution: dist,
	}, nil
}

// CITypes
func (r *EntRepository) CreateCIType(ctx context.Context, ct *CIType) (*CIType, error) {
	e, err := r.client.CIType.Create().
		SetName(ct.Name).
		SetDescription(ct.Description).
		SetIcon(ct.Icon).
		SetColor(ct.Color).
		SetAttributeSchema(ct.AttributeSchema).
		SetIsActive(ct.IsActive).
		SetTenantID(ct.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toTypeDomain(e), nil
}

func (r *EntRepository) GetCIType(ctx context.Context, id int, tenantID int) (*CIType, error) {
	e, err := r.client.CIType.Query().
		Where(citype.ID(id), citype.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toTypeDomain(e), nil
}

func (r *EntRepository) ListCITypes(ctx context.Context, tenantID int) ([]*CIType, error) {
	es, err := r.client.CIType.Query().Where(citype.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var results []*CIType
	for _, e := range es {
		results = append(results, toTypeDomain(e))
	}
	return results, nil
}

// Relationships
func (r *EntRepository) CreateRelationship(ctx context.Context, rel *CIRelationship) (*CIRelationship, error) {
	e, err := r.client.CIRelationship.Create().
		SetSourceCiID(rel.SourceCIID).
		SetTargetCiID(rel.TargetCIID).
		SetRelationshipTypeID(rel.RelationshipTypeID).
		SetDescription(rel.Description).
		SetTenantID(rel.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CIRelationship{
		ID:                 e.ID,
		SourceCIID:         e.SourceCiID,
		TargetCIID:         e.TargetCiID,
		RelationshipTypeID: e.RelationshipTypeID,
		Description:        e.Description,
		TenantID:           e.TenantID,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}, nil
}

func (r *EntRepository) GetRelationships(ctx context.Context, ciID int) ([]*CIRelationship, error) {
	es, err := r.client.CIRelationship.Query().
		Where(cirelationship.Or(
			cirelationship.SourceCiID(ciID),
			cirelationship.TargetCiID(ciID),
		)).All(ctx)
	if err != nil {
		return nil, err
	}
	var results []*CIRelationship
	for _, e := range es {
		results = append(results, &CIRelationship{
			ID:                 e.ID,
			SourceCIID:         e.SourceCiID,
			TargetCIID:         e.TargetCiID,
			RelationshipTypeID: e.RelationshipTypeID,
			Description:        e.Description,
			TenantID:           e.TenantID,
			CreatedAt:          e.CreatedAt,
			UpdatedAt:          e.UpdatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) DeleteRelationship(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.CIRelationship.Delete().
		Where(cirelationship.ID(id), cirelationship.TenantID(tenantID)).
		Exec(ctx)
	return err
}
