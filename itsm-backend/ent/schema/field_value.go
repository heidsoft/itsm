package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FieldValue 动态字段提交的值，挂在具体的工单/服务请求实例上。
// entity_type 取值：ticket | service_request
// 冗余快照 field_name/field_label/sort_order：定义被改名/删除不影响历史值的展示。
type FieldValue struct {
	ent.Schema
}

func (FieldValue) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.String("entity_type").Comment("值归属的实体类型: ticket | service_request").NotEmpty(),
		field.Int("entity_id").Comment("归属实体ID（工单ID 或 服务请求ID）").Positive(),
		field.Int("field_definition_id").Comment("指回 field_definitions，可空，定义被删不影响历史值").Optional().Nillable(),
		field.String("field_name").Comment("提交时快照的字段名").NotEmpty(),
		field.String("field_label").Comment("提交时快照的显示名").NotEmpty(),
		field.Int("sort_order").Comment("提交时快照的顺序").Default(0),
		field.JSON("value", json.RawMessage{}).Comment("字段值，JSON 编码，原始类型（数字/字符串/布尔/数组）").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
	}
}

func (FieldValue) Edges() []ent.Edge { return nil }

func (FieldValue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "entity_type", "entity_id"),
	}
}
