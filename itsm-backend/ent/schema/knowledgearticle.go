package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
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
		field.Int("view_count").Comment("浏览次数").Default(0),
		field.Int("like_count").Comment("点赞次数").Default(0),
		// ---- 知识可引用性 L1：时效性（正确性边界）----
		// 三个时间字段为 nil 表示未设置，语义由 knowledgeaccess.FreshnessGuard 统一解释：
		//   valid_from      非 nil 且当前时间早于它 → 尚未生效，不得被 RAG 引用
		//   valid_until     非 nil 且当前时间达到它 → 已失效，不得被 RAG 引用
		//   last_reviewed_at 配合 review_interval_days 判定复核是否逾期
		// 均为 Optional+Nillable 是刻意的：存量与新建文章默认为「永久有效、不设复核」，
		// 保证向后兼容，只有管理员显式声明时效的知识才进入管控。
		field.Time("valid_from").Comment("生效时间（早于该时间不得被 RAG 引用；空=立即生效）").Optional().Nillable(),
		field.Time("valid_until").Comment("失效时间（达到该时间后不得被 RAG 引用；空=长期有效）").Optional().Nillable(),
		field.Time("last_reviewed_at").Comment("最近一次内容复核时间；空=从未复核").Optional().Nillable(),
		field.Int("review_interval_days").Comment("复核周期（天）；0=不设复核要求").Default(0),
		// ---- 知识可引用性 L2：权威性（排序质量）----
		// 0=普通 10=部门推荐 20=官方标准 30=唯一真相源。仅影响检索排序与并列时的取舍，
		// 不参与准入判定——权威性低不等于不可引用，因此它不能承担正确性职责。
		field.Int("authority_level").Comment("权威等级：0 普通 / 10 部门推荐 / 20 官方标准 / 30 唯一真相源").Default(0),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Comment("软删除时间").Optional().Nillable(),
	}
}

func (KnowledgeArticle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_likes", KnowledgeArticleLike.Type),
		edge.To("versions", KnowledgeArticleVersion.Type),
		edge.To("sessions", KnowledgeArticleSession.Type),
	}
}
