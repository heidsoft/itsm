package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IncidentStatus 事件状态枚举
type IncidentStatus string

const (
	IncidentStatusNew        IncidentStatus = "new"         // 新建
	IncidentStatusAssigned   IncidentStatus = "assigned"    // 已分配
	IncidentStatusInProgress IncidentStatus = "in_progress" // 处理中
	IncidentStatusResolved   IncidentStatus = "resolved"    // 已解决
	IncidentStatusClosed     IncidentStatus = "closed"      // 已关闭
	IncidentStatusSuspended  IncidentStatus = "suspended"   // 挂起
)

// IncidentPriority 事件优先级枚举
type IncidentPriority string

const (
	IncidentPriorityLow      IncidentPriority = "low"      // 低
	IncidentPriorityMedium   IncidentPriority = "medium"   // 中
	IncidentPriorityHigh     IncidentPriority = "high"     // 高
	IncidentPriorityCritical IncidentPriority = "critical" // 紧急
)

// IncidentSource 事件来源枚举
type IncidentSource string

const (
	IncidentSourceServiceDesk   IncidentSource = "service_desk"   // 服务台
	IncidentSourceMonitoring    IncidentSource = "monitoring"     // 监控系统
	IncidentSourceSecurity      IncidentSource = "security"       // 安全中心
	IncidentSourceUserReport    IncidentSource = "user_report"    // 用户报告
	IncidentSourceAlibabaCloud  IncidentSource = "alibaba_cloud"  // 阿里云
	IncidentSourceCloudProduct  IncidentSource = "cloud_product"  // 云产品
	IncidentSourceSecurityEvent IncidentSource = "security_event" // 安全事件
)

// IncidentType 事件类型枚举
type IncidentType string

const (
	IncidentTypeInfrastructure IncidentType = "infrastructure" // 基础设施
	IncidentTypeApplication    IncidentType = "application"    // 应用服务
	IncidentTypeSecurity       IncidentType = "security"       // 安全事件
	IncidentTypeNetwork        IncidentType = "network"        // 网络问题
	IncidentTypeDatabase       IncidentType = "database"       // 数据库
	IncidentTypeStorage        IncidentType = "storage"        // 存储
	IncidentTypeCloudService   IncidentType = "cloud_service"  // 云服务
)

// Incident 事件模型
type Incident struct {
	ent.Schema
}

func (Incident) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			NotEmpty().
			MaxLen(255).
			Comment("事件标题"),
		field.Text("description").
			Optional().
			Comment("事件描述"),
		field.Enum("status").
			Values(
				string(IncidentStatusNew),
				string(IncidentStatusAssigned),
				string(IncidentStatusInProgress),
				string(IncidentStatusResolved),
				string(IncidentStatusClosed),
				string(IncidentStatusSuspended),
			).
			Default(string(IncidentStatusNew)).
			Comment("事件状态"),
		field.Enum("priority").
			Values(
				string(IncidentPriorityLow),
				string(IncidentPriorityMedium),
				string(IncidentPriorityHigh),
				string(IncidentPriorityCritical),
			).
			Default(string(IncidentPriorityMedium)).
			Comment("事件优先级"),
		field.Enum("source").
			Values(
				string(IncidentSourceServiceDesk),
				string(IncidentSourceMonitoring),
				string(IncidentSourceSecurity),
				string(IncidentSourceUserReport),
				string(IncidentSourceAlibabaCloud),
				string(IncidentSourceCloudProduct),
				string(IncidentSourceSecurityEvent),
			).
			Comment("事件来源"),
		field.Enum("type").
			Values(
				string(IncidentTypeInfrastructure),
				string(IncidentTypeApplication),
				string(IncidentTypeSecurity),
				string(IncidentTypeNetwork),
				string(IncidentTypeDatabase),
				string(IncidentTypeStorage),
				string(IncidentTypeCloudService),
			).
			Comment("事件类型"),
		field.String("incident_number").
			Unique().
			NotEmpty().
			MaxLen(50).
			Comment("事件编号"),
		field.Bool("is_major_incident").
			Default(false).
			Comment("是否重大事件"),
		field.Int("reporter_id").
			Positive().
			Comment("报告人ID"),
		field.Int("assignee_id").
			Optional().
			Nillable().
			Positive().
			Comment("处理人ID"),
		field.Int("tenant_id").
			Positive().
			Comment("租户ID"),
		// 阿里云相关字段
		field.String("alibaba_cloud_instance_id").
			Optional().
			MaxLen(100).
			Comment("阿里云实例ID"),
		field.String("alibaba_cloud_region").
			Optional().
			MaxLen(50).
			Comment("阿里云地域"),
		field.String("alibaba_cloud_service").
			Optional().
			MaxLen(100).
			Comment("阿里云服务名称"),
		field.JSON("alibaba_cloud_alert_data", map[string]interface{}{}).
			Optional().
			Comment("阿里云告警原始数据"),
		field.JSON("alibaba_cloud_metrics", map[string]interface{}{}).
			Optional().
			Comment("阿里云监控指标数据"),
		// 安全事件相关字段
		field.String("security_event_type").
			Optional().
			MaxLen(100).
			Comment("安全事件类型"),
		field.String("security_event_source_ip").
			Optional().
			MaxLen(50).
			Comment("安全事件源IP"),
		field.String("security_event_target").
			Optional().
			MaxLen(200).
			Comment("安全事件目标"),
		field.JSON("security_event_details", map[string]interface{}{}).
			Optional().
			Comment("安全事件详细信息"),
		// 时间字段
		field.Time("detected_at").
			Default(time.Now).
			Comment("检测时间"),
		field.Time("confirmed_at").
			Optional().
			Nillable().
			Comment("确认时间"),
		field.Time("resolved_at").
			Optional().
			Nillable().
			Comment("解决时间"),
		field.Time("closed_at").
			Optional().
			Nillable().
			Comment("关闭时间"),
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

func (Incident) Edges() []ent.Edge {
	return []ent.Edge{
		// 租户关联
		edge.From("tenant", Tenant.Type).
			Ref("incidents").
			Field("tenant_id").
			Required().
			Unique(),
		// 报告人
		edge.From("reporter", User.Type).
			Ref("reported_incidents").
			Field("reporter_id").
			Required().
			Unique(),
		// 处理人
		edge.From("assignee", User.Type).
			Ref("assigned_incidents").
			Field("assignee_id").
			Unique(),
		// 关联的配置项
		edge.To("affected_configuration_items", ConfigurationItem.Type),
		// 关联的问题
		edge.To("related_problems", Ticket.Type),
		// 关联的变更
		edge.To("related_changes", Ticket.Type),
		// 状态变更日志
		edge.To("status_logs", StatusLog.Type),
		// 评论
		edge.To("comments", Ticket.Type),
	}
}

// Indexes of the Incident.
func (Incident) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("source"),
		index.Fields("type"),
		index.Fields("reporter_id"),
		index.Fields("assignee_id"),
		index.Fields("created_at"),
		index.Fields("incident_number"),
		index.Fields("is_major_incident"),
		// 租户相关索引
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "status"),
		index.Fields("tenant_id", "priority"),
		// 阿里云相关索引
		index.Fields("alibaba_cloud_instance_id"),
		index.Fields("alibaba_cloud_region"),
		index.Fields("alibaba_cloud_service"),
		// 安全事件相关索引
		index.Fields("security_event_type"),
		index.Fields("security_event_source_ip"),
		// 复合索引
		index.Fields("status", "priority"),
		index.Fields("source", "type"),
		index.Fields("tenant_id", "source"),
		index.Fields("tenant_id", "is_major_incident"),
	}
}
