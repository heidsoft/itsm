package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

// SubscriptionStatus 订阅状态
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"   // 活跃
	SubscriptionStatusExpired  SubscriptionStatus = "expired"  // 过期
	SubscriptionStatusCanceled SubscriptionStatus = "canceled" // 取消
)

// Subscription 订阅模型
type Subscription struct {
	ent.Schema
}

func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").
			Positive().
			Comment("租户ID"),
		field.String("plan_name").
			NotEmpty().
			MaxLen(50).
			Comment("订阅计划名称"),
		field.Float("monthly_price").
			Positive().
			Comment("月费价格"),
		field.Float("yearly_price").
			Positive().
			Comment("年费价格"),
		field.Enum("status").
			Values(
				string(SubscriptionStatusActive),
				string(SubscriptionStatusExpired),
				string(SubscriptionStatusCanceled),
			).
			Default(string(SubscriptionStatusActive)).
			Comment("订阅状态"),
		field.Time("starts_at").
			Comment("开始时间"),
		field.Time("expires_at").
			Comment("过期时间"),
		field.JSON("features", []string{}).
			Optional().
			Comment("功能特性列表"),
		field.JSON("quota", map[string]interface{}{}).
			Optional().
			Comment("资源配额"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

func (Subscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).
			Ref("subscriptions").
			Field("tenant_id").
			Required().
			Unique(),
	}
}
