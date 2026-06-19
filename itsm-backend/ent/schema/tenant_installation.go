package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TenantInstallation holds the schema definition for the TenantInstallation entity.
type TenantInstallation struct {
	ent.Schema
}

// Fields of the TenantInstallation.
func (TenantInstallation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").
			Comment("租户ID"),
		field.Int("item_id").
			Comment("关联的MarketplaceItem ID"),
		field.String("installed_version").
			Comment("安装的版本号"),
		field.Enum("status").
			Values("installing", "active", "disabled", "failed", "uninstalled").
			Default("installing").
			Comment("安装状态"),
		field.JSON("config", map[string]interface{}{}).
			Optional().
			Comment("租户的组件配置信息，敏感字段加密存储"),
		field.Bool("auto_upgrade").
			Default(true).
			Comment("是否自动升级到最新版本"),
		field.String("installed_by").
			Optional().
			Comment("安装者用户ID"),
		field.Time("installed_at").
			Default(time.Now).
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.Time("last_updated_at").
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.Time("last_used_at").
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.String("error_message").
			Optional().
			Comment("安装/运行错误信息"),
		field.Time("created_at").
			Default(time.Now).
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
	}
}

// Edges of the TenantInstallation.
func (TenantInstallation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("item", MarketplaceItem.Type).
			Ref("installations").
			Unique().
			Field("item_id").
			Required(),
	}
}

// Indexes of the TenantInstallation.
func (TenantInstallation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "item_id").
			Unique(),
		index.Fields("tenant_id", "status"),
		index.Fields("status", "installed_at"),
	}
}
