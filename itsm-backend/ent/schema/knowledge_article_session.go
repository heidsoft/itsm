package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// KnowledgeArticleSession 实时协作会话
type KnowledgeArticleSession struct {
	ent.Schema
}

func (KnowledgeArticleSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("article_id").Comment("文章ID").Positive(),
		field.Int("user_id").Comment("用户ID").Positive(),
		field.String("session_token").Comment("会话Token").MaxLen(64),
		field.Enum("status").Values("active", "idle", "inactive").Comment("状态"),
		field.Time("last_heartbeat").Comment("最后心跳时间").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
	}
}

func (KnowledgeArticleSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("article", KnowledgeArticle.Type).
			Ref("sessions"),
		edge.From("user", User.Type).
			Ref("article_sessions"),
		edge.To("participants", KnowledgeArticleParticipant.Type),
	}
}

func (KnowledgeArticleSession) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一索引: session_token
		index.Fields("session_token").Unique(),
		// 索引: article_id
		index.Fields("article_id"),
		// 索引: user_id
		index.Fields("user_id"),
	}
}