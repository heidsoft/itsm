package service_catalog

import (
	"context"

	"itsm-backend/ent"
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

func (r *EntRepository) Get(ctx context.Context, id int) (*ServiceCatalog, error) {
	res, err := r.client.ServiceCatalog.Get(ctx, id)
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
		query = query.Where(servicecatalog.Status(filters.Status))
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

func (r *EntRepository) Update(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	update := r.client.ServiceCatalog.UpdateOneID(catalog.ID).
		SetName(catalog.Name).
		SetCategory(catalog.Category).
		SetDescription(catalog.Description).
		SetDeliveryTime(catalog.DeliveryTime).
		SetStatus(catalog.Status)
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

func (r *EntRepository) Delete(ctx context.Context, id int) error {
	return r.client.ServiceCatalog.DeleteOneID(id).Exec(ctx)
}

func (r *EntRepository) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
	query := r.client.ServiceCatalog.Query().
		Where(
			servicecatalog.TenantID(tenantID),
			servicecatalog.StatusEQ("enabled"),
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
