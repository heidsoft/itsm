package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ApprovalConfig 流程级审批配置。
// DB 列由 migrations/20260621_process_definition_approval_sla_config.sql 创建（JSONB），
// 本 struct 与该迁移 COMMENT 中登记的 Schema 一一对应——新增字段必须同步迁移注释，
// 保证读改写（read-modify-write）不丢未知字段。
type ApprovalConfig struct {
	RequireApproval  bool             `json:"require_approval,omitempty"`
	ApprovalType     string           `json:"approval_type,omitempty"` // single | parallel | sequential | conditional
	Approvers        []int            `json:"approvers,omitempty"`     // 流程级兜底用户 ID 列表（主要走 BPMN 节点级 candidateGroups）
	AutoApproveRoles []string         `json:"auto_approve_roles,omitempty"`
	EscalationRules  []EscalationRule `json:"escalation_rules,omitempty"`
}

// EscalationRule 审批升级规则（当前仅透传存储，尚未消费）。
type EscalationRule struct {
	AfterHours int    `json:"after_hours,omitempty"`
	ToUser     int    `json:"to_user,omitempty"`
	ToGroup    string `json:"to_group,omitempty"`
	Notify     bool   `json:"notify,omitempty"`
}

// ProcessDefinition holds the schema definition for the BPMN Process Definition entity.
type ProcessDefinition struct {
	ent.Schema
}

// Fields of the ProcessDefinition.
func (ProcessDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			Comment("流程定义Key，BPMN标准").
			NotEmpty(),
		field.String("name").
			Comment("流程定义名称").
			NotEmpty(),
		field.Text("description").
			Comment("流程描述").
			Optional(),
		field.String("version").
			Comment("版本号").
			Default("1.0.0"),
		field.String("category").
			Comment("流程分类").
			Default("default"),
		field.JSON("bpmn_xml", []byte{}).
			Comment("BPMN XML定义内容"),
		field.JSON("approval_config", &ApprovalConfig{}).
			Comment("流程级审批配置（require_approval/approval_type/approvers/auto_approve_roles/escalation_rules），列由 20260621 迁移创建").
			Optional(),
		field.JSON("sla_config", map[string]interface{}{}).
			Comment("流程级 SLA 配置（response_time_hours/resolution_time_hours/business_hours_only 等），列由 20260621 迁移创建，当前仅透传存储").
			Optional(),
		field.JSON("process_variables", map[string]interface{}{}).
			Comment("流程变量定义").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.Bool("is_latest").
			Comment("是否最新版本").
			Default(true),
		field.Int("deployment_id").
			Comment("部署ID").
			Positive(),
		field.String("deployment_name").
			Comment("部署名称").
			Optional(),
		field.Time("deployed_at").
			Comment("部署时间").
			Default(time.Now),
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

// Edges of the ProcessDefinition.
func (ProcessDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("process_instances", ProcessInstance.Type).
			Comment("流程实例"),
		edge.To("bindings", ProcessBinding.Type).
			Comment("流程绑定"),
		edge.To("version_changelogs", ProcessVersionChangelog.Type).
			Comment("版本变更日志"),
		edge.From("deployment", ProcessDeployment.Type).
			Ref("definitions").
			Field("deployment_id").
			Required().
			Unique(),
	}
}

// Indexes of the ProcessDefinition.
func (ProcessDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "key", "version").
			Unique(),
		index.Fields("tenant_id", "key"),
		index.Fields("deployment_id"),
		index.Fields("is_active"),
		index.Fields("is_latest"),
	}
}
