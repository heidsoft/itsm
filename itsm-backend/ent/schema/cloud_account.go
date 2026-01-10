package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CloudAccount holds the schema definition for the CloudAccount entity.
type CloudAccount struct {
	ent.Schema
}

// Fields of the CloudAccount.
func (CloudAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("provider").
			Comment("云厂商标识（aliyun/huawei/tencent/azure/onprem）").
			NotEmpty(),
		field.String("account_id").
			Comment("云账号ID").
			NotEmpty(),
		field.String("account_name").
			Comment("云账号名称").
			NotEmpty(),
		field.String("credential_ref").
			Comment("凭据引用（不存明文）").
			Optional(),
		field.JSON("region_whitelist", []string{}).
			Comment("可用Region白名单").
			Optional(),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
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

// Edges of the CloudAccount.
func (CloudAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("resources", CloudResource.Type),
	}
}

// Indexes of the CloudAccount.
func (CloudAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("provider", "account_id").
			Unique(),
	}
}
