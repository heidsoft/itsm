package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConfigurationItem holds the schema definition for the ConfigurationItem entity.
type ConfigurationItem struct {
	ent.Schema
}

// Fields of the ConfigurationItem.
func (ConfigurationItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("ci_number").
			Comment("CI唯一业务编号，格式 CI-YYYYMM-NNNNNN，由服务层发号器生成（全局唯一，跨租户协调）").
			Optional().
			Nillable(),
		field.String("name").
			Comment("CI名称").
			NotEmpty(),
		field.String("description").
			Comment("CI描述").
			Optional(),
		field.Int("ci_type_id").
			Comment("CI类型ID").
			Positive(),
		field.String("ci_type").
			Comment("CI类型").
			Default("server"),
		field.String("status").
			Comment("运行状态").
			Default("active"),
		field.String("environment").
			Comment("环境").
			Default("production"),
		field.String("criticality").
			Comment("重要性级别").
			Default("medium"),

		// 基础属性
		field.String("asset_tag").
			Comment("资产标签").
			Optional(),
		field.String("serial_number").
			Comment("序列号").
			Optional(),
		field.String("model").
			Comment("型号").
			Optional(),
		field.String("vendor").
			Comment("厂商").
			Optional(),
		field.String("location").
			Comment("位置").
			Optional(),
		field.String("assigned_to").
			Comment("分配给").
			Optional(),
		field.String("owned_by").
			Comment("拥有者").
			Optional(),
		field.String("ownership_mode").
			Comment("ownership模式: managed(平台管理)/customer(客户托管)/sla(SLA覆盖)").
			Default("managed"),
		field.Time("local_modified_at").
			Comment("客户本地修改时间，用于三方合并冲突检测").
			Optional(),

		// 发现属性
		field.String("discovery_source").
			Comment("发现源").
			Optional(),
		field.Time("last_discovered").
			Comment("最后发现时间").
			Optional(),
		field.String("source").
			Comment("数据来源（manual/discovery/import）").
			Optional(),
		field.String("source_id").Comment("规范发现来源ID").Optional(),
		field.String("canonical_cloud_account_id").Comment("云厂商不可变账号ID").Optional(),
		field.Time("source_last_seen_at").Comment("来源最近确认时间").Optional(),
		field.Int("source_missing_count").Comment("连续完整快照缺失次数").Default(0).NonNegative(),
		field.String("source_fingerprint").Comment("来源内容指纹").Optional(),

		// 扩展属性
		field.JSON("attributes", map[string]interface{}{}).
			Comment("扩展属性").
			Optional(),

		// 云资源属性
		field.String("cloud_provider").
			Comment("云厂商").
			Optional(),
		field.String("cloud_account_id").
			Comment("云账号ID").
			Optional(),
		field.String("cloud_region").
			Comment("云Region").
			Optional(),
		field.String("cloud_zone").
			Comment("云Zone").
			Optional(),
		field.String("cloud_resource_id").
			Comment("云资源ID").
			Optional(),
		field.String("cloud_resource_type").
			Comment("云资源类型").
			Optional(),
		field.JSON("cloud_metadata", map[string]interface{}{}).
			Comment("云资源元数据").
			Optional(),
		field.JSON("cloud_tags", map[string]interface{}{}).
			Comment("云资源标签").
			Optional(),
		field.JSON("cloud_metrics", map[string]interface{}{}).
			Comment("云资源监控指标").
			Optional(),
		field.Time("cloud_sync_time").
			Comment("云资源同步时间").
			Optional(),
		field.String("cloud_sync_status").
			Comment("云资源同步状态").
			Optional(),
		field.Int("cloud_resource_ref_id").
			Comment("云资源引用ID").
			Optional(),

		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("version").
			Comment("版本号，用于乐观锁").
			Default(1),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),

		// 生命周期管理
		field.String("lifecycle_status").
			Comment("生命周期状态").
			Default("online"),
		field.Time("effective_at").
			Comment("生效时间").
			Optional(),
		field.Time("expire_at").
			Comment("失效时间").
			Optional(),
	}
}

// Edges of the ConfigurationItem.
func (ConfigurationItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ci_type_ref", CIType.Type).
			Ref("cis").
			Unique().
			Field("ci_type_id").
			Required(),
		edge.From("cloud_resource_ref", CloudResource.Type).
			Ref("cis").
			Unique().
			Field("cloud_resource_ref_id"),
		// 与工单的关系
		edge.To("tickets", Ticket.Type),
		// 与事件的关系
		edge.To("incidents", Incident.Type),
		// CI之间的关系 - 出边 (作为源)
		edge.To("outgoing_relations", CIRelationship.Type),
		// CI之间的关系 - 入边 (作为目标)
		edge.To("history", ConfigurationItemHistory.Type).
			Comment("CI变更历史"),
		edge.To("tags", CITag.Type).
			Comment("CI标签"),
		edge.To("incoming_relations", CIRelationship.Type),
	}
}

// Indexes of the ConfigurationItem.
func (ConfigurationItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ci_number").Unique(),
		index.Fields("ci_type"),
		index.Fields("ci_type_id"),
		index.Fields("status"),
		index.Fields("environment"),
		index.Fields("cloud_provider"),
		index.Fields("cloud_account_id"),
		index.Fields("cloud_region"),
		index.Fields("cloud_resource_id"),
		index.Fields("tenant_id", "source_id"),
		index.Fields("tenant_id", "serial_number").Unique(),
		index.Fields("ownership_mode"),
	}
}
