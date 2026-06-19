package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ItemVersion holds the schema definition for the ItemVersion entity.
type ItemVersion struct {
	ent.Schema
}

// Fields of the ItemVersion.
func (ItemVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int("item_id").
			Comment("关联的MarketplaceItem ID"),
		field.String("version").
			Comment("版本号，语义化版本如v1.0.0"),
		field.String("changelog").
			Optional().
			Comment("版本更新日志"),
		field.String("download_url").
			Optional().
			Comment("安装包下载地址，第三方组件使用"),
		field.String("manifest_path").
			Optional().
			Comment("内置组件的Manifest路径，官方组件使用"),
		field.String("min_system_version").
			Optional().
			Comment("该版本最低支持的系统版本"),
		field.JSON("dependencies", map[string]string{}).
			Optional().
			Comment("依赖的其他组件，格式为组件名:版本要求"),
		field.Enum("status").
			Values("draft", "published", "deprecated", "withdrawn").
			Default("draft").
			Comment("版本状态"),
		field.Int("download_count").
			Default(0).
			Comment("该版本下载次数"),
		field.Time("released_at").
			Default(time.Now).
			SchemaType(map[string]string{
				dialect.MySQL: "datetime",
			}),
		field.JSON("config_schema", map[string]interface{}{}).
			Optional().
			Comment("该版本的配置Schema，覆盖item级别的Schema"),
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

// Edges of the ItemVersion.
func (ItemVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("item", MarketplaceItem.Type).
			Ref("versions").
			Unique().
			Field("item_id").
			Required(),
	}
}

// Indexes of the ItemVersion.
func (ItemVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("item_id", "version").
			Unique(),
		index.Fields("status", "released_at"),
	}
}
