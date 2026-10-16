package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SSOIdentity 外部身份（provider + subject）到本地用户的绑定关系。
//
// 与 password_reset_tokens 同一口径：provider_id / user_id 是普通整型外键列，
// 不建 ent edge——避免为了 SSO 去改动 User 的关系图（edge.From 双向契约）。
// 租户隔离由 tenant_id 列承担（两张表都含 tenant_id，无需登记 tenant_guard 豁免）。
type SSOIdentity struct{ ent.Schema }

func (SSOIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("provider_id").Comment("ssoproviders.id"),
		field.Int("user_id").Comment("users.id"),
		field.String("subject").NotEmpty().MaxLen(255).
			Comment("IdP 令牌里的 sub 声明，提供方内唯一"),
		field.String("external_username").Optional().MaxLen(255),
		field.String("external_email").Optional().MaxLen(255),
		field.Time("last_login_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SSOIdentity) Indexes() []ent.Index {
	return []ent.Index{
		// 同一提供方内 subject 唯一：回调时凭 (provider, subject) 定位本地用户。
		index.Fields("tenant_id", "provider_id", "subject").Unique(),
		// 同一用户在同一提供方只保留一条绑定：解绑/重绑不累积历史行。
		index.Fields("tenant_id", "user_id", "provider_id").Unique(),
		index.Fields("provider_id"),
	}
}
