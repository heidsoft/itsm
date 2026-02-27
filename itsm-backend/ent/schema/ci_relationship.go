package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// CIRelationshipType CI关系类型枚举
type CIRelationshipType string

const (
	// DependsOn 依赖关系 - A依赖于B
	DependsOn CIRelationshipType = "depends_on"
	// Hosts 托管关系 - A托管于B (如应用部署在服务器)
	Hosts CIRelationshipType = "hosts"
	// HostedOn 所属关系 - A所属B (如服务器在机架)
	HostedOn CIRelationshipType = "hosted_on"
	// ConnectsTo 连接关系 - A连接到B
	ConnectsTo CIRelationshipType = "connects_to"
	// RunsOn 运行关系 - A运行在B上
	RunsOn CIRelationshipType = "runs_on"
	// Contains 包含关系 - A包含B
	Contains CIRelationshipType = "contains"
	// PartOf 组成部分 - A是B的一部分
	PartOf CIRelationshipType = "part_of"
	// Impacts 影响关系 - A影响B
	Impacts CIRelationshipType = "impacts"
	// ImpactedBy 受影响于 - A受B影响
	ImpactedBy CIRelationshipType = "impacted_by"
	// Owns 拥有关系 - A拥有B
	Owns CIRelationshipType = "owns"
	// OwnedBy 所属关系 - A被B拥有
	OwnedBy CIRelationshipType = "owned_by"
	// Uses 使用关系 - A使用B
	Uses CIRelationshipType = "uses"
	// UsedBy 被使用关系 - A被B使用
	UsedBy CIRelationshipType = "used_by"
)

// RelationshipStrength 关系强度
type RelationshipStrength string

const (
	StrengthCritical RelationshipStrength = "critical" // 关键依赖
	StrengthHigh     RelationshipStrength = "high"     // 强依赖
	StrengthMedium   RelationshipStrength = "medium"   // 中等依赖
	StrengthLow      RelationshipStrength = "low"      // 弱依赖
)

// ImpactLevel 影响程度
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical" // 致命影响
	ImpactHigh     ImpactLevel = "high"     // 严重影响
	ImpactMedium   ImpactLevel = "medium"   // 一般影响
	ImpactLow      ImpactLevel = "low"      // 轻微影响
)

// CIRelationship holds the schema definition for CI relationships.
type CIRelationship struct {
	ent.Schema
}

// Fields of the CIRelationship.
func (CIRelationship) Fields() []ent.Field {
	return []ent.Field{
		// 关系类型
		field.String("relationship_type").
			Comment("关系类型: depends_on, hosts, hosted_on, connects_to, runs_on, contains, part_of, impacts, owned_by, owns, uses, used_by").
			NotEmpty(),
		// 源CI ID (关系发起方)
		field.Int("source_ci_id").
			Comment("源CI ID"),
		// 目标CI ID (关系接收方)
		field.Int("target_ci_id").
			Comment("目标CI ID"),
		// 关系强度
		field.Enum("strength").
			Values("critical", "high", "medium", "low").
			Default("medium").
			Comment("关系强度"),
		// 影响程度
		field.Enum("impact_level").
			Values("critical", "high", "medium", "low").
			Default("medium").
			Comment("影响程度"),
		// 是否启用
		field.Bool("is_active").
			Default(true).
			Comment("是否启用"),
		// 是否为自动发现的关系
		field.Bool("is_discovered").
			Default(false).
			Comment("是否自动发现"),
		// 关系描述
		field.String("description").
			Optional().
			Comment("关系描述"),
		// 元数据 (存储额外属性)
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("关系元数据"),
		// 创建时间
		field.Time("created_at").
			Default(time.Now),
		// 更新时间
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the CIRelationship.
func (CIRelationship) Edges() []ent.Edge {
	return []ent.Edge{
		// 源CI
		edge.From("source_ci", ConfigurationItem.Type).
			Ref("outgoing_relations").
			Unique().
			Field("source_ci_id").
			Required(),
		// 目标CI
		edge.From("target_ci", ConfigurationItem.Type).
			Ref("incoming_relations").
			Unique().
			Field("target_ci_id").
			Required(),
	}
}

// Indexes of the CIRelationship.
func (CIRelationship) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("relationship_type"),
		index.Fields("source_ci_id"),
		index.Fields("target_ci_id"),
		index.Fields("strength"),
		index.Fields("impact_level"),
		index.Fields("is_active"),
		// 复合索引：防止重复关系
		index.Fields("source_ci_id", "target_ci_id", "relationship_type").Unique(),
	}
}
