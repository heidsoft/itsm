package service

import (
	"context"
	"encoding/json"

	"itsm-backend/ent"
	"itsm-backend/ent/fielddefinition"
	"itsm-backend/ent/fieldvalue"
)

// FieldValueService 动态字段值的共享服务，工单先接入，服务请求以后接入。
type FieldValueService struct {
	client *ent.Client
}

func NewFieldValueService(client *ent.Client) *FieldValueService {
	return &FieldValueService{client: client}
}

// CreateValues 把提交的 values（fieldName -> 原始值）跟 (defEntityType, defEntityID) 下的
// 字段定义匹配，快照 name/label/顺序后写入 field_values，挂在 (valueEntityType, valueEntityID) 上。
// values 里不匹配任何字段定义的 key 会被忽略（例如 presetTypeId 这类路由元数据）。
// 多条 insert 包在一个事务里：中途某一条失败（比如瞬时 DB 错误）不应该留下"插了一半"的
// 半成品提交记录——field_values 代表的是一次完整的表单提交，要么整体成功要么整体不落库。
func (s *FieldValueService) CreateValues(ctx context.Context, tenantID int, defEntityType string, defEntityID int, valueEntityType string, valueEntityID int, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	defs, err := s.client.FieldDefinition.Query().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(defEntityType),
			fielddefinition.EntityID(defEntityID),
			fielddefinition.IsActive(true),
		).
		Order(ent.Asc(fielddefinition.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	for _, def := range defs {
		raw, ok := values[def.Name]
		if !ok {
			continue
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			return rollback(tx, err)
		}
		defID := def.ID
		_, err = tx.FieldValue.Create().
			SetTenantID(tenantID).
			SetEntityType(valueEntityType).
			SetEntityID(valueEntityID).
			SetFieldDefinitionID(defID).
			SetFieldName(def.Name).
			SetFieldLabel(def.Label).
			SetSortOrder(def.SortOrder).
			SetValue(encoded).
			Save(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}

// AdHocFieldValue 是没有对应 field_definitions 行的自描述字段值——
// 用于前端静态预设（代码里写死、不对应数据库模板）提交自定义字段的场景。
type AdHocFieldValue struct {
	Name      string
	Label     string
	SortOrder int
	Value     interface{}
}

// CreateAdHocValues 直接按调用方提供的 name/label 写入 field_values，跳过
// CreateValues 那种"先查 field_definitions 再匹配"的步骤——静态预设没有
// field_definitions 行可以匹配。
func (s *FieldValueService) CreateAdHocValues(ctx context.Context, tenantID int, valueEntityType string, valueEntityID int, fields []AdHocFieldValue) error {
	if len(fields) == 0 {
		return nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	for _, f := range fields {
		encoded, err := json.Marshal(f.Value)
		if err != nil {
			return rollback(tx, err)
		}
		_, err = tx.FieldValue.Create().
			SetTenantID(tenantID).
			SetEntityType(valueEntityType).
			SetEntityID(valueEntityID).
			SetFieldName(f.Name).
			SetFieldLabel(f.Label).
			SetSortOrder(f.SortOrder).
			SetValue(encoded).
			Save(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}

// FieldValueDTO 展示用的已解析字段值。
type FieldValueDTO struct {
	Name  string      `json:"name"`
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// ListValues 按提交时快照的顺序返回 (tenantID, entityType, entityID) 的字段值。
func (s *FieldValueService) ListValues(ctx context.Context, tenantID int, entityType string, entityID int) ([]FieldValueDTO, error) {
	rows, err := s.client.FieldValue.Query().
		Where(
			fieldvalue.TenantID(tenantID),
			fieldvalue.EntityType(entityType),
			fieldvalue.EntityID(entityID),
		).
		Order(ent.Asc(fieldvalue.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]FieldValueDTO, 0, len(rows))
	for _, row := range rows {
		var value interface{}
		if len(row.Value) > 0 {
			if err := json.Unmarshal(row.Value, &value); err != nil {
				continue // 损坏的值跳过展示，不阻塞整个响应
			}
		}
		result = append(result, FieldValueDTO{
			Name:  row.FieldName,
			Label: row.FieldLabel,
			Value: value,
		})
	}
	return result, nil
}
