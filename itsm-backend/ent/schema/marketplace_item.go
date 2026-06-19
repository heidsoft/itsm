package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MarketplaceItem holds the schema definition for the MarketplaceItem entity.
type MarketplaceItem struct {
	ent.Schema
}

// Fields of the MarketplaceItem.
func (MarketplaceItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Unique().
			Comment("组件唯一标识名称"),
		field.Enum("type").
			Values("connector", "skill", "plugin").
			Comment("组件类型：连接器/技能/插件"),
		field.String("title").
			Comment("组件显示名称"),
		field.String("provider").
			Comment("提供商名称"),
		field.String("description").
			Optional().
			Comment("组件描述"),
		field.String("long_description").
			Optional().
			Comment("详细描述"),
		field.String("icon_url").
			Optional().
			Comment("图标URL"),
		field.JSON("screenshots", []string{}).
			Optional().
			Comment("截图URL列表"),
		field.JSON("tags", []string{}).
			Optional().
			Comment("标签列表"),
		field.Float("rating").
			Default(0.0).
			Comment("评分，0-5"),
		field.Int("install_count").
			Default(0).
			Comment("安装次数"),
		field.String("latest_version").
			Comment("最新版本号"),
		field.String("min_system_version").
			Optional().
			Comment("最低支持的系统版本"),
		field.Enum("status").
			Values("draft", "reviewing", "published", "rejected", "deprecated").
			Default("draft").
			Comment("发布状态"),
		field.Bool("is_official").
			Default(false).
			Comment("是否是官方组件"),
		field.Bool("is_free").
			Default(true).
			Comment("是否免费"),
		field.Float("price").
			Default(0.0).
			Comment("价格，付费组件使用"),
		field.String("category").
			Optional().
			Comment("分类，如办公协同/监控告警/AI能力等"),
		field.JSON("capabilities", []string{}).
			Optional().
			Comment("组件支持的能力列表"),
		field.JSON("required_permissions", []string{}).
			Optional().
			Comment("组件需要的系统权限列表"),
		field.JSON("config_schema", map[string]interface{}{}).
			Optional().
			Comment("配置JSON Schema"),
		field.String("author_id").
			Optional().
			Comment("作者用户ID"),
		field.String("author_name").
			Optional().
			Comment("作者名称"),
		field.String("homepage").
			Optional().
			Comment("项目主页"),
		field.String("repository").
			Optional().
			Comment("代码仓库地址"),
		field.String("license").
			Optional().
			Comment("开源协议"),
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

// Edges of the MarketplaceItem.
func (MarketplaceItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("versions", ItemVersion.Type),
		edge.To("installations", TenantInstallation.Type),
	}
}

// Indexes of the MarketplaceItem.
func (MarketplaceItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("type", "category"),
		index.Fields("status", "is_official"),
		index.Fields("rating", "install_count"),
	}
}
