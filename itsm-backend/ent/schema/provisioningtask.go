package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProvisioningTask holds the schema definition for the ProvisioningTask entity.
// M2：云资源自动交付任务（先实现可运行骨架，后续接入阿里云真实API）。
type ProvisioningTask struct{ ent.Schema }

func (ProvisioningTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Int("service_request_id").Comment("服务请求ID").Positive(),

		field.String("provider").Comment("云厂商：alicloud").Default("alicloud"),
		field.String("resource_type").Comment("资源类型：ecs|rds|oss|vpc|iam").Default("ecs"),

		// 原始请求/解析后的交付命令（JSONB）
		field.JSON("payload", map[string]any{}).Comment("交付任务payload（来自服务请求表单）").Optional(),
		field.JSON("result", map[string]any{}).Comment("交付结果（资源ID等）").Optional(),

		field.String("status").Comment("状态：pending|running|succeeded|failed").Default("pending"),
		field.Text("error_message").Comment("失败原因").Optional(),

		// 尝试计数（2026-10-21）：没有计数时「失败」是个终态——既无法判断重试了几次，
		// 也无法阻止无限重试。ExecuteTask 每次领取任务 +1，重试上限由
		// domain/provisioning.MaxTaskAttempts 约束。
		field.Int("attempt_count").Comment("已执行次数（含正在运行的那次）").Default(0).Min(0),
		field.Time("last_attempt_at").Comment("最近一次开始执行的时间").Optional().Nillable(),

		// 人工接管（2026-10-21）：provider 永久性故障（凭据吊销、配额封禁）或资源已在
		// 平台外建好时，交付不能永远卡在 failed。接管必须留痕：谁判定的、依据是什么。
		field.Int("resolved_by_user_id").Comment("人工接管人ID").Optional(),
		field.Text("resolution_note").Comment("人工接管说明（判定依据/工单号/资源ID）").Optional(),

		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProvisioningTask) Edges() []ent.Edge { return nil }

func (ProvisioningTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "service_request_id"),
		index.Fields("tenant_id", "status", "created_at"),
	}
}
