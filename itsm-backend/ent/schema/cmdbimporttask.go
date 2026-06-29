package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CMDBImportTask holds the schema definition for the CMDBImportTask entity.
type CMDBImportTask struct {
	ent.Schema
}

// Fields of the CMDBImportTask.
func (CMDBImportTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_id").
			Comment("任务ID").
			NotEmpty().
			Unique(),
		field.String("file_url").
			Comment("导入文件地址").
			NotEmpty(),
		field.String("file_type").
			Comment("文件类型：xlsx/csv").
			NotEmpty(),
		field.String("update_mode").
			Comment("更新模式：skip/overwrite/merge").
			NotEmpty(),
		field.String("sheet_name").
			Comment("Excel Sheet名称").
			Optional(),
		field.Int("total_count").
			Comment("总行数").
			Default(0),
		field.Int("success_count").
			Comment("成功行数").
			Default(0),
		field.Int("failed_count").
			Comment("失败行数").
			Default(0),
		field.JSON("errors", []map[string]interface{}{}).
			Comment("错误详情").
			Optional(),
		field.String("status").
			Comment("任务状态：pending/processing/completed/failed").
			Default("pending"),
		field.String("error_message").
			Comment("全局错误信息").
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
	}
}

// Edges of the CMDBImportTask.
func (CMDBImportTask) Edges() []ent.Edge {
	return nil
}

// Indexes of the CMDBImportTask.
func (CMDBImportTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id").Unique(),
		index.Fields("tenant_id"),
		index.Fields("status"),
		index.Fields("operator_id"),
		index.Fields("created_at"),
	}
}
