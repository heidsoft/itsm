package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConnectorInboundDedup 持久化入站回调去重表，覆盖 Feishu/DingTalk/WeCom/Webhook。
// 单一 (tenant_id, connector_name, event_id) 唯一；expires_at 后允许覆盖。
// 用 DB 持久化替代 in-memory map，重启 / 多实例部署不再丢去重窗口。
type ConnectorInboundDedup struct {
	ent.Schema
}

func (ConnectorInboundDedup) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("connector_name").MaxLen(64),
		field.String("event_id").MaxLen(256),
		field.Time("received_at").Default(time.Now),
		field.Time("expires_at"),
		field.String("payload_hash").MaxLen(64),
		field.String("response_status").MaxLen(32),
	}
}

func (ConnectorInboundDedup) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：(tenant, connector, event_id) 三元组
		index.Fields("tenant_id", "connector_name", "event_id").Unique(),
		// 用于后台扫描过期行
		index.Fields("expires_at"),
	}
}
