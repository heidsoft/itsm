package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CICertificationTask holds the schema definition for the CI data certification task.
//
// ServiceNow CMDB Health 的 Compliance 维度：谁在何时确认某 CI 仍准确。
// 单次确认落成一条 status=confirmed 的任务记录，形成可审计的历史序列；
// 待办（pending）与逾期（overdue）由 due_at 派生，不单独存枚举状态。
// CI 表上的 last_confirmed_at/confirmed_by 是本表最近一条 confirmed 的冗余，
// 供列表/治理看板排序，避免每次读都聚合任务表。
type CICertificationTask struct {
	ent.Schema
}

// Fields of the CICertificationTask.
func (CICertificationTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ci_id").
			Comment("被认证的配置项ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.String("status").
			Comment("任务状态：pending 待办 / confirmed 已确认 / overdue 已逾期（由 due_at 派生）").
			Default("pending"),
		field.Int("assignee_id").
			Comment("认证责任人用户ID").
			Optional(),
		field.Int("confirmed_by").
			Comment("实际确认人用户ID").
			Optional(),
		field.Time("confirmed_at").
			Comment("实际确认时间").
			Optional(),
		field.Time("due_at").
			Comment("认证截止时间（逾期判定基准）").
			Optional(),
		field.String("evidence").
			Comment("认证依据/备注（人工填写）").
			Optional().
			MaxLen(2000),
		field.String("source").
			Comment("任务来源：manual 人工发起 / schedule 周期任务 / discovery 发现触发").
			Default("manual"),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the CICertificationTask.
func (CICertificationTask) Edges() []ent.Edge {
	return nil
}

// Indexes of the CICertificationTask.
// Ent v0.14.6 没有 index.Partial；entsql.IndexWhere 需要重新生成迁移元数据。
// 本批保持普通联合索引，pending 唯一约束以 20261126_certification_pending_unique.sql
// 为唯一事实源。enttest 测试客户端不带该约束，并发窗口测试只在 PostgreSQL
// 或显式安装等价部分唯一索引的 SQLite 上有意义。
func (CICertificationTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "ci_id"),
		index.Fields("tenant_id", "status"),
		index.Fields("tenant_id", "due_at"),
		index.Fields("ci_id", "confirmed_at"),
	}
}
