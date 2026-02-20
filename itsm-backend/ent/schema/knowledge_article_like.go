package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// KnowledgeArticleLike 用户点赞记录
type KnowledgeArticleLike struct{ ent.Schema }

func (KnowledgeArticleLike) Fields() []ent.Field {
	return []ent.Field{
		field.Int("article_id").Comment("文章ID").Positive(),
		field.Int("user_id").Comment("用户ID").Positive(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("点赞时间").Default(time.Now),
	}
}

func (KnowledgeArticleLike) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("article", KnowledgeArticle.Type).
			Ref("user_likes").
			Field("article_id").
			Required().
			Unique(),
	}
}
