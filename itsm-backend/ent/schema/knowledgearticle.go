package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type KnowledgeArticle struct{ ent.Schema }

func (KnowledgeArticle) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").Comment("文章标题").NotEmpty(),
		field.Text("content").Comment("文章内容").Optional(),
		field.String("category").Comment("分类").Optional(),
		field.String("tags").Comment("标签").Optional(),
		field.Int("author_id").Comment("作者ID").Positive(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_published").Comment("是否发布").Default(false),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (KnowledgeArticle) Edges() []ent.Edge { return nil }
