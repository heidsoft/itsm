package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CMDBExportTask holds the schema definition for the CMDBExportTask entity.
type CMDBExportTask struct {
	ent.Schema
}

// Fields of the CMDBExportTask.
func (CMDBExportTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_id").
			Comment("任务ID").
			NotEmpty().
			Unique(),
		field.JSON("filters", map[string]interface{}{}).
			Comment("导出过滤条件").
			Optional(),
		field.JSON("export_fields", []string{}).
			Comment("导出字段列表").
			Optional(),
		field.String("export_type").
			Comment("导出类型：xlsx/csv").
			NotEmpty(),
		field.String("file_url").
			Comment("导出文件地址").
			Optional(),
		field.Int64("file_size").
			Comment("文件大小（字节）").
			Optional(),
		field.Int("total_count").
			Comment("导出的CI数量").
			Default(0),
		field.String("status").
			Comment("任务状态：pending/processing/completed/failed").
			Default("pending"),
		field.String("error_message").
			Comment("错误信息").
			Optional(),
		field.Int("operator_id").
			Comment("操作人ID").
			Positive(),
		field.String("operator_name").
			Comment("操作人名称").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("started_at").
			Comment("开始时间").
			Optional(),
		field.Time("completed_at").
			Comment("完成时间").
			Optional(),
		field.Time("expires_at").
			Comment("下载链接过期时间").
			Optional(),
	}
}

// Edges of the CMDBExportTask.
func (CMDBExportTask) Edges() []ent.Edge {
	return nil
}

// Indexes of the CMDBExportTask.
func (CMDBExportTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id").Unique(),
		index.Fields("tenant_id"),
		index.Fields("status"),
		index.Fields("operator_id"),
		index.Fields("created_at"),
	}
}
