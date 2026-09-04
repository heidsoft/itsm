package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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

// CIRelationshipTypeMeta 内置关系类型元数据。
// 这是 CMDB 关系词表的**单一受控源**（single source of truth）：
// API /relationship-types 端点、AI 工具参数、ontology 正则反向映射都必须从这里派生，
// 禁止在其他文件再硬编码第二份关系清单（2026-09-04 CMDB AI-Native 评审 P0-2 收口）。
type CIRelationshipTypeMeta struct {
	Type        CIRelationshipType
	Name        string             // 中文展示名
	Description string             // 语义说明
	Direction   string             // uni-directional / bi-directional
	Icon        string             // 前端图标名
	Reverse     CIRelationshipType // 语义反向类型（用于入边方向描述；自反类型为自身）
}

// CIRelationshipTypeVocabulary 全部 13 种内置关系类型的受控词表。
// 注意：depends_on 与 impacted_by 互为反向、impacts 的反向也是 impacted_by，
// 与 ontology_service 既有行为保持一致（ 收口时未改变运行语义）。
var CIRelationshipTypeVocabulary = []CIRelationshipTypeMeta{
	{Type: DependsOn, Name: "依赖", Description: "源CI依赖目标CI", Direction: "uni-directional", Icon: "link", Reverse: ImpactedBy},
	{Type: Hosts, Name: "托管", Description: "源CI托管目标CI", Direction: "uni-directional", Icon: "server", Reverse: HostedOn},
	{Type: HostedOn, Name: "承载于", Description: "源CI运行或部署在目标CI上", Direction: "uni-directional", Icon: "hard-drive", Reverse: Hosts},
	{Type: ConnectsTo, Name: "连接到", Description: "源CI连接目标CI", Direction: "bi-directional", Icon: "network", Reverse: ConnectsTo},
	{Type: RunsOn, Name: "运行于", Description: "源CI运行在目标CI上", Direction: "uni-directional", Icon: "play", Reverse: RunsOn},
	{Type: Contains, Name: "包含", Description: "源CI包含目标CI", Direction: "uni-directional", Icon: "box", Reverse: PartOf},
	{Type: PartOf, Name: "组成部分", Description: "源CI是目标CI的一部分", Direction: "uni-directional", Icon: "component", Reverse: Contains},
	{Type: Impacts, Name: "影响", Description: "源CI故障会影响目标CI", Direction: "uni-directional", Icon: "activity", Reverse: ImpactedBy},
	{Type: ImpactedBy, Name: "受影响于", Description: "源CI受目标CI故障影响", Direction: "uni-directional", Icon: "activity", Reverse: DependsOn},
	{Type: Owns, Name: "拥有", Description: "源CI拥有目标CI", Direction: "uni-directional", Icon: "key", Reverse: OwnedBy},
	{Type: OwnedBy, Name: "被拥有", Description: "源CI被目标CI拥有", Direction: "uni-directional", Icon: "key", Reverse: Owns},
	{Type: Uses, Name: "使用", Description: "源CI使用目标CI能力", Direction: "uni-directional", Icon: "plug", Reverse: UsedBy},
	{Type: UsedBy, Name: "被使用", Description: "源CI能力被目标CI使用", Direction: "uni-directional", Icon: "plug", Reverse: Uses},
}

// IsValidCIRelationshipType 判断关系类型是否在受控词表内。
func IsValidCIRelationshipType(v string) bool {
	for _, m := range CIRelationshipTypeVocabulary {
		if string(m.Type) == v {
			return true
		}
	}
	return false
}

// ReverseCIRelationshipType 返回语义反向关系类型；未知类型原样返回（保持既有行为）。
func ReverseCIRelationshipType(v string) string {
	for _, m := range CIRelationshipTypeVocabulary {
		if string(m.Type) == v {
			return string(m.Reverse)
		}
	}
	return v
}

// CIRelationshipTypeDirectionLabel 返回关系类型的方向语义描述（入边/出边展示用）。
func CIRelationshipTypeDirectionLabel(v string) string {
	switch v {
	case string(DependsOn), string(ImpactedBy), string(HostedOn), string(PartOf), string(OwnedBy), string(UsedBy):
		return "依赖/受影响方向"
	case string(Impacts), string(Hosts), string(Contains), string(Owns), string(Uses), string(RunsOn):
		return "影响/承载方向"
	}
	return "双向"
}

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
		// 租户ID
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(), // 必填：存量数据已由 migrations/20260610_cmdb_tenant_id_backfill.sql 回填
		// 关系类型（受控词表见 CIRelationshipTypeVocabulary，13 种内置类型；
		// 写入侧由 ci_relationship_service 校验，禁止自由字符串）
		field.String("relationship_type").
			Comment("关系类型: depends_on, hosts, hosted_on, connects_to, runs_on, contains, part_of, impacts, impacted_by, owns, owned_by, uses, used_by").
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
		index.Fields("tenant_id"),
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
