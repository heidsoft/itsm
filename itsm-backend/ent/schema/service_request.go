package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// ServiceRequestStatus defines the status enum for service request
type ServiceRequestStatus string

const (
	ServiceRequestStatusPending    ServiceRequestStatus = "pending"     // 待处理
	ServiceRequestStatusInProgress ServiceRequestStatus = "in_progress" // 处理中
	ServiceRequestStatusCompleted  ServiceRequestStatus = "completed"   // 已完成
	ServiceRequestStatusRejected   ServiceRequestStatus = "rejected"    // 已拒绝
)

// ServiceRequest holds the schema definition for the ServiceRequest entity.
type ServiceRequest struct {
	ent.Schema
}

// Fields of the ServiceRequest.
func (ServiceRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int("catalog_id").
			Positive().
			Comment("服务目录ID"),
		field.Int("requester_id").
			Positive().
			Comment("申请人ID"),
		field.Enum("status").
			Values(
				string(ServiceRequestStatusPending),
				string(ServiceRequestStatusInProgress),
				string(ServiceRequestStatusCompleted),
				string(ServiceRequestStatusRejected),
			).
			Default(string(ServiceRequestStatusPending)).
			Comment("请求状态"),
		field.String("reason").
			Optional().
			MaxLen(500).
			Comment("申请原因"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

// Edges of the ServiceRequest.
func (ServiceRequest) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联服务目录
		edge.From("catalog", ServiceCatalog.Type).
			Ref("service_requests").
			Field("catalog_id").
			Required().
			Unique(),
		// 关联申请人
		edge.From("requester", User.Type).
			Ref("service_requests").
			Field("requester_id").
			Required().
			Unique(),
	}
}

// Indexes of the ServiceRequest.
func (ServiceRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("catalog_id"),
		index.Fields("requester_id"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("requester_id", "status"),
		index.Fields("catalog_id", "status"),
	}
}