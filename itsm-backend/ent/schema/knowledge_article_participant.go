package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// KnowledgeArticleParticipant 协作参与者
type KnowledgeArticleParticipant struct {
	ent.Schema
}

func (KnowledgeArticleParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.Int("session_id").Comment("会话ID").Positive(),
		field.Int("user_id").Comment("用户ID").Positive(),
		field.Int("cursor_position").Comment("光标位置").Default(0),
		field.Bool("is_active").Comment("是否活跃").Default(true),
		field.Time("joined_at").Comment("加入时间").Default(time.Now),
		field.Time("last_activity").Comment("最后活动时间").Optional(),
	}
}

func (KnowledgeArticleParticipant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", KnowledgeArticleSession.Type).
			Ref("participants"),
		edge.From("user", User.Type).
			Ref("article_participations"),
	}
}

func (KnowledgeArticleParticipant) Indexes() []ent.Index {
	return []ent.Index{
		// 索引: session_id
		index.Fields("session_id"),
		// 索引: user_id
		index.Fields("user_id"),
		// 联合索引: session_id + user_id
		index.Fields("session_id", "user_id"),
	}
}
