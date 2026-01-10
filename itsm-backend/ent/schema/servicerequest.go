package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ServiceRequest struct{ ent.Schema }

func (ServiceRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Int("catalog_id").Comment("服务目录ID").Positive(),
		field.Int("ci_id").Comment("关联CI ID").Optional(),
		field.Int("requester_id").Comment("申请人ID").Positive(),
		field.String("status").Comment("状态").Default("submitted"),
		field.String("title").Comment("请求标题").Optional(),
		field.Text("reason").Comment("申请原因").Optional(),

		// V0 关键字段（最小集合）
		field.JSON("form_data", map[string]any{}).Comment("表单数据").Optional(),
		field.String("cost_center").Comment("成本中心").Optional(),
		field.String("data_classification").Comment("数据分级：public|internal|confidential").Default("internal"),
		field.Bool("needs_public_ip").Comment("是否需要公网访问").Default(false),
		field.JSON("source_ip_whitelist", []string{}).Comment("源IP白名单").Optional(),
		field.Time("expire_at").Comment("到期时间").Optional(),
		field.Bool("compliance_ack").Comment("合规条款确认").Default(false),

		// 审批进度（三段审批）
		field.Int("current_level").Comment("当前审批级别").Default(1),
		field.Int("total_levels").Comment("总审批级别数").Default(3),

		field.Text("last_error").Comment("最近一次错误信息").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (ServiceRequest) Edges() []ent.Edge { return nil }

func (ServiceRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "created_at"),
		index.Fields("tenant_id", "requester_id", "created_at"),
		index.Fields("tenant_id", "status", "created_at"),
		index.Fields("tenant_id", "ci_id"),
	}
}
