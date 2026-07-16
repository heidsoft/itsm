package service_catalog

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/servicecatalog"
)

// EntRepository implements Repository using Ent
type EntRepository struct {
	client *ent.Client
}

// NewEntRepository creates a new EntRepository
func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) Create(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	entFunc := r.client.ServiceCatalog.Create().
		SetName(catalog.Name).
		SetCategory(catalog.Category).
		SetDescription(catalog.Description).
		SetDeliveryTime(catalog.DeliveryTime).
		SetStatus(catalog.Status).
		SetIsActive(catalog.Status == "enabled").
		SetTenantID(catalog.TenantID)
	if catalog.CITypeID > 0 {
		entFunc = entFunc.SetCiTypeID(catalog.CITypeID)
	}
	if catalog.CloudServiceID > 0 {
		entFunc = entFunc.SetCloudServiceID(catalog.CloudServiceID)
	}

	res, err := entFunc.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(res), nil
}

func (r *EntRepository) Get(ctx context.Context, tenantID int, id int) (*ServiceCatalog, error) {
	res, err := r.client.ServiceCatalog.Query().
		Where(
			servicecatalog.ID(id),
			servicecatalog.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(res), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceCatalog, int, error) {
	query := r.client.ServiceCatalog.Query().
		Where(servicecatalog.TenantID(tenantID))

	if filters.Category != "" {
		query = query.Where(servicecatalog.Category(filters.Category))
	}
	if filters.Status != "" {
		switch filters.Status {
		case "enabled":
			query = query.Where(servicecatalog.StatusIn("enabled", "active"))
		case "disabled":
			query = query.Where(servicecatalog.StatusIn("disabled", "inactive"))
		default:
			query = query.Where(servicecatalog.Status(filters.Status))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	entList, err := query.
		Order(ent.Desc(servicecatalog.FieldCreatedAt)).
		Offset((filters.Page - 1) * filters.Size).
		Limit(filters.Size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var list []*ServiceCatalog
	for _, item := range entList {
		list = append(list, r.toDomain(item))
	}
	return list, total, nil
}

func (r *EntRepository) Update(ctx context.Context, tenantID int, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	update := r.client.ServiceCatalog.UpdateOneID(catalog.ID).
		Where(servicecatalog.TenantID(tenantID)).
		SetName(catalog.Name).
		SetCategory(catalog.Category).
		SetDescription(catalog.Description).
		SetDeliveryTime(catalog.DeliveryTime).
		SetStatus(catalog.Status).
		SetIsActive(catalog.Status == "enabled")
	if catalog.CITypeID > 0 {
		update = update.SetCiTypeID(catalog.CITypeID)
	}
	if catalog.CloudServiceID > 0 {
		update = update.SetCloudServiceID(catalog.CloudServiceID)
	}

	res, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(res), nil
}

func (r *EntRepository) Delete(ctx context.Context, tenantID int, id int) error {
	return r.client.ServiceCatalog.UpdateOneID(id).
		Where(servicecatalog.TenantID(tenantID)).
		SetStatus("disabled").
		SetIsActive(false).
		Exec(ctx)
}

func (r *EntRepository) NameExists(ctx context.Context, tenantID int, name string, excludeID int) (bool, error) {
	query := r.client.ServiceCatalog.Query().
		Where(servicecatalog.TenantIDEQ(tenantID), servicecatalog.NameEqualFold(name))
	if excludeID > 0 {
		query = query.Where(servicecatalog.IDNEQ(excludeID))
	}
	return query.Exist(ctx)
}

func (r *EntRepository) ValidateReferences(ctx context.Context, tenantID, ciTypeID, cloudServiceID int) error {
	if ciTypeID > 0 {
		exists, err := r.client.CIType.Query().
			Where(citype.IDEQ(ciTypeID), citype.TenantIDEQ(tenantID), citype.IsActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("failed to validate CI type: %w", err)
		}
		if !exists {
			return fmt.Errorf("CI type not found or inactive")
		}
	}
	if cloudServiceID > 0 {
		exists, err := r.client.CloudService.Query().
			Where(cloudservice.IDEQ(cloudServiceID), cloudservice.TenantIDEQ(tenantID), cloudservice.IsActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("failed to validate cloud service: %w", err)
		}
		if !exists {
			return fmt.Errorf("cloud service not found or inactive")
		}
	}
	return nil
}

func (r *EntRepository) Count(ctx context.Context, tenantID int, filters ListFilters) (int, error) {
	query := r.client.ServiceCatalog.Query().Where(servicecatalog.TenantID(tenantID))
	if filters.Status != "" {
		switch filters.Status {
		case "enabled":
			query = query.Where(servicecatalog.StatusIn("enabled", "active"))
		case "disabled":
			query = query.Where(servicecatalog.StatusIn("disabled", "inactive"))
		default:
			query = query.Where(servicecatalog.Status(filters.Status))
		}
	}
	return query.Count(ctx)
}

func (r *EntRepository) CountByCategory(ctx context.Context, tenantID int) (map[string]int, error) {
	catalogs, err := r.client.ServiceCatalog.Query().
		Where(servicecatalog.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, cat := range catalogs {
		result[cat.Category]++
	}
	return result, nil
}

func (r *EntRepository) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
	query := r.client.ServiceCatalog.Query().
		Where(
			servicecatalog.TenantID(tenantID),
			servicecatalog.StatusIn("enabled", "active"),
		)

	// 关键词搜索：name OR description
	if keyword != "" {
		query = query.Where(
			servicecatalog.Or(
				servicecatalog.NameContains(keyword),
				servicecatalog.DescriptionContains(keyword),
			),
		)
	}

	// 分类过滤
	if filters.Category != "" {
		query = query.Where(servicecatalog.Category(filters.Category))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (filters.Page - 1) * filters.Size
	catalogs, err := query.
		Order(ent.Desc(servicecatalog.FieldCreatedAt)).
		Offset(offset).
		Limit(filters.Size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var list []*ServiceCatalog
	for _, item := range catalogs {
		list = append(list, r.toDomain(item))
	}

	return list, total, nil
}

func (r *EntRepository) toDomain(e *ent.ServiceCatalog) *ServiceCatalog {
	return &ServiceCatalog{
		ID:             e.ID,
		Name:           e.Name,
		Category:       e.Category,
		Description:    e.Description,
		DeliveryTime:   e.DeliveryTime,
		CITypeID:       e.CiTypeID,
		CloudServiceID: e.CloudServiceID,
		Status:         e.Status,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}
