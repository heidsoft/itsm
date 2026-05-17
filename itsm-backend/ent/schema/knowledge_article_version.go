package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// KnowledgeArticleVersion 文章版本历史
type KnowledgeArticleVersion struct {
	ent.Schema
}

func (KnowledgeArticleVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int("article_id").Comment("文章ID").Positive(),
		field.Int("version").Comment("版本号").Positive(),
		field.String("title").Comment("文章标题").NotEmpty(),
		field.Text("content").Comment("文章内容").Optional(),
		field.String("category").Comment("分类").Optional(),
		field.String("tags").Comment("标签").Optional(),
		field.Int("author_id").Comment("作者ID").Positive(),
		field.String("change_summary").Comment("变更摘要").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
	}
}

func (KnowledgeArticleVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("article", KnowledgeArticle.Type).
			Ref("versions"),
	}
}

func (KnowledgeArticleVersion) Indexes() []ent.Index {
	return []ent.Index{
		// 联合唯一索引: 文章ID + 版本号
		index.Fields("article_id", "version").Unique(),
	}
}
