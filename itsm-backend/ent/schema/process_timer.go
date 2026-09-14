package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProcessTimer struct{ ent.Schema }

func (ProcessTimer) Fields() []ent.Field {
	return []ent.Field{
		field.String("timer_id").
			Comment("全局唯一 Timer ID").
			NotEmpty(),
		field.String("timer_type").
			Comment("Timer 类型：start / intermediate / boundary").
			NotEmpty(),
		field.String("process_definition_key").
			Comment("流程定义 Key").
			NotEmpty(),
		field.Int("process_instance_id").
			Comment("流程实例 ID（start timer 无此字段）").
			Optional(),
		field.String("activity_id").
			Comment("绑定的活动 ID（boundary timer 必填）").
			Optional(),
		field.String("timer_expression").
			Comment("ISO 8601 duration 或 cron 表达式，支持 ${variable} 占位符").
			NotEmpty(),
		field.String("expression_type").
			Comment("表达式类型：duration / cron / date").
			NotEmpty(),
		field.Time("fire_at").
			Comment("计划触发时间"),
		field.Time("fired_at").
			Comment("实际触发时间（审计用）").
			Optional(),
		field.String("status").
			Comment("状态：pending / fired / cancelled / failed").
			Default("pending"),
		field.Int("version").
			Comment("乐观锁版本号").
			Default(1),
		field.String("idempotency_key").
			Comment("幂等 key = timer_id + fire_at + tenant_id").
			NotEmpty(),
		field.Int("retry_count").
			Comment("已重试次数").
			Default(0),
		field.Int("max_retries").
			Comment("最大重试次数").
			Default(3),
		field.Time("last_fire_attempt").
			Comment("上次触发尝试时间").
			Optional(),
		field.String("failure_reason").
			Comment("失败原因").
			Optional().
			MaxLen(2000),
		field.JSON("context_variables", map[string]interface{}{}).
			Comment("触发时注入流程的变量").
			Optional(),
		field.Float("total_duration_seconds").
			Comment("总时长（从 timer_expression 解析，用于 SLA 暂停/恢复）").
			Optional(),
		field.Float("elapsed_seconds").
			Comment("已消耗时间（暂停时计算）").
			Default(0),
		field.String("pause_state").
			Comment("暂停状态：running / paused").
			Default("running"),
		field.Int("parent_timer_id").
			Comment("恢复时创建的新 timer 指向原始 timer").
			Optional(),
		field.Int("tenant_id").
			Comment("租户 ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (ProcessTimer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("process_instance", ProcessInstance.Type).
			Ref("timers").
			Field("process_instance_id").
			Unique(),
	}
}

func (ProcessTimer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("timer_id").
			Unique(),
		index.Fields("idempotency_key").
			Unique(),
		index.Fields("status", "fire_at"),
		index.Fields("tenant_id", "status"),
		index.Fields("process_instance_id"),
		index.Fields("pause_state", "status"),
	}
}
