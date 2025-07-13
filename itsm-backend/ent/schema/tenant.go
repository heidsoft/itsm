package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// TenantStatus 租户状态枚举
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"    // 活跃
	TenantStatusSuspended TenantStatus = "suspended" // 暂停
	TenantStatusExpired   TenantStatus = "expired"   // 过期
	TenantStatusDeleted   TenantStatus = "deleted"   // 删除
)

// TenantType 租户类型
type TenantType string

const (
	TenantTypeTrial        TenantType = "trial"        // 试用版
	TenantTypeStandard     TenantType = "standard"     // 标准版
	TenantTypeProfessional TenantType = "professional" // 专业版
	TenantTypeEnterprise   TenantType = "enterprise"   // 企业版
)

// Tenant 租户模型
type Tenant struct {
	ent.Schema
}

func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("租户名称"),
		field.String("code").
			Unique().
			NotEmpty().
			MaxLen(50).
			Comment("租户代码"),
		field.String("domain").
			Optional().
			MaxLen(100).
			Comment("自定义域名"),
		field.Enum("status").
			Values(
				string(TenantStatusActive),
				string(TenantStatusSuspended),
				string(TenantStatusExpired),
				string(TenantStatusDeleted),
			).
			Default(string(TenantStatusActive)).
			Comment("租户状态"),
		field.Enum("type").
			Values(
				string(TenantTypeTrial),
				string(TenantTypeStandard),
				string(TenantTypeProfessional),
				string(TenantTypeEnterprise),
			).
			Default(string(TenantTypeTrial)).
			Comment("租户类型"),
		field.JSON("settings", map[string]interface{}{}).
			Optional().
			Comment("租户配置"),
		field.JSON("quota", map[string]interface{}{}).
			Optional().
			Comment("资源配额"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("过期时间"),
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

func (Tenant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
		edge.To("tickets", Ticket.Type),
		edge.To("service_catalogs", ServiceCatalog.Type),
		edge.To("service_requests", ServiceRequest.Type),
		edge.To("subscriptions", Subscription.Type),
		edge.To("configuration_items", ConfigurationItem.Type),
		edge.To("knowledge_articles", KnowledgeArticle.Type),
		edge.To("workflows", Workflow.Type),
		// CMDB扩展
		edge.To("ci_types", CIType.Type),
		edge.To("ci_relationship_types", CIRelationshipType.Type),
		edge.To("ci_relationships", CIRelationship.Type),
		edge.To("ci_lifecycle_states", CILifecycleState.Type),
		edge.To("ci_change_records", CIChangeRecord.Type),
		// 新增：CI属性定义
		edge.To("ci_attribute_definitions", CIAttributeDefinition.Type),
	}
}

// Indexes of the Tenant.
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code"),
		index.Fields("status"),
		index.Fields("type"),
		index.Fields("domain"),
		index.Fields("created_at"),
		// 复合索引
		index.Fields("status", "type"),
	}
}
