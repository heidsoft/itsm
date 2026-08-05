package service

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/fielddefinition"
)

// FieldDefinitionService 动态字段定义的共享服务，工单模板、服务目录项共用。
type FieldDefinitionService struct {
	client *ent.Client
}

func NewFieldDefinitionService(client *ent.Client) *FieldDefinitionService {
	return &FieldDefinitionService{client: client}
}

// FieldDefinitionInput 创建/替换字段定义时的输入。
type FieldDefinitionInput struct {
	Name      string
	Label     string
	FieldType string
	Required  bool
	Options   []interface{}
	SortOrder int
}

// ReplaceDefinitions 删除 (tenantID, entityType, entityID) 下所有既有字段定义，
// 按 defs 的顺序重新插入。字段值（field_values）已经快照了 name/label/顺序，
// 不依赖这里的行 ID 存续，所以用"删除重建"而不是逐字段 diff。
func (s *FieldDefinitionService) ReplaceDefinitions(ctx context.Context, tenantID int, entityType string, entityID int, defs []FieldDefinitionInput) ([]*ent.FieldDefinition, error) {
	if err := s.DeleteDefinitions(ctx, tenantID, entityType, entityID); err != nil {
		return nil, err
	}
	result := make([]*ent.FieldDefinition, 0, len(defs))
	for i, d := range defs {
		sortOrder := d.SortOrder
		if sortOrder == 0 {
			sortOrder = i
		}
		options := d.Options
		if options == nil {
			options = []interface{}{}
		}
		created, err := s.client.FieldDefinition.Create().
			SetTenantID(tenantID).
			SetEntityType(entityType).
			SetEntityID(entityID).
			SetName(d.Name).
			SetLabel(d.Label).
			SetFieldType(d.FieldType).
			SetRequired(d.Required).
			SetOptions(options).
			SetSortOrder(sortOrder).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, created)
	}
	return result, nil
}

// ListDefinitions 按 sort_order 返回 (tenantID, entityType, entityID) 下的字段定义。
func (s *FieldDefinitionService) ListDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) ([]*ent.FieldDefinition, error) {
	return s.client.FieldDefinition.Query().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(entityType),
			fielddefinition.EntityID(entityID),
			fielddefinition.IsActive(true),
		).
		Order(ent.Asc(fielddefinition.FieldSortOrder)).
		All(ctx)
}

// DeleteDefinitions 删除 (tenantID, entityType, entityID) 下所有字段定义。
func (s *FieldDefinitionService) DeleteDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) error {
	_, err := s.client.FieldDefinition.Delete().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(entityType),
			fielddefinition.EntityID(entityID),
		).
		Exec(ctx)
	return err
}
