package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiscoveryJob holds the schema definition for the DiscoveryJob entity.
type DiscoveryJob struct {
	ent.Schema
}

// Fields of the DiscoveryJob.
func (DiscoveryJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("source_id").
			Comment("发现源ID").
			NotEmpty(),
		field.String("status").
			Comment("任务状态（queued/discovering/discovered/reconciling/succeeded/partial_failed/failed/cancelled）").
			Default("queued"),
		field.String("operation").Comment("作业类型").Default("full_discovery"),
		field.String("idempotency_key").Comment("客户端幂等键").Optional(),
		field.String("request_fingerprint").Comment("规范化请求指纹").Optional(),
		field.JSON("source_snapshot", map[string]interface{}{}).Comment("不可变来源快照").Optional(),
		field.JSON("scope_snapshot", map[string]interface{}{}).Comment("不可变发现范围快照").Optional(),
		field.JSON("completed_scopes", []string{}).Comment("完整覆盖范围").Optional(),
		field.JSON("failed_scopes", []string{}).Comment("失败范围").Optional(),
		field.String("snapshot_generation").Comment("快照代次").Optional(),
		field.Int("requested_by").Comment("请求人ID").Optional(),
		field.Time("queued_at").Comment("入队时间").Optional(),
		field.Time("heartbeat_at").Comment("最近心跳").Optional(),
		field.String("lease_owner").Comment("租约所有者").Optional(),
		field.Time("lease_expires_at").Comment("租约过期时间").Optional(),
		field.Int64("fencing_token").Comment("租约隔离令牌").Default(0).NonNegative(),
		field.Int("attempt").Comment("当前尝试次数").Default(0).NonNegative(),
		field.Int("parent_job_id").Comment("重试来源作业ID").Optional(),
		field.Int("max_attempts").Comment("最大尝试次数").Default(3).Positive(),
		field.Int("progress").Comment("进度百分比").Default(0).Range(0, 100),
		field.String("error_code").Comment("稳定错误码").Optional(),
		field.String("error_message").Comment("净化后的错误摘要").Optional(),
		field.Time("cancel_requested_at").Comment("取消请求时间").Optional(),
		field.Time("started_at").
			Comment("开始时间").
			Optional(),
		field.Time("finished_at").
			Comment("结束时间").
			Optional(),
		field.JSON("summary", map[string]interface{}{}).
			Comment("任务摘要").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
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

// Edges of the DiscoveryJob.
func (DiscoveryJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source", DiscoverySource.Type).
			Ref("jobs").
			Unique().
			Field("source_id").
			Required(),
		edge.To("results", DiscoveryResult.Type),
	}
}

// Indexes of the DiscoveryJob.
func (DiscoveryJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("source_id"),
		index.Fields("status"),
		index.Fields("tenant_id", "operation", "source_id", "idempotency_key"),
		index.Fields("status", "lease_expires_at"),
	}
}
