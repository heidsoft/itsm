package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		// ---- 评审状态机（最小闭环）----
		// 流转：draft --submit--> under_review --approve--> published
		//                ↑                    └--reject--> rejected --resubmit--> under_review
		//                            approved --withdraw--> draft（作者撤回）
		//
		// 为什么要有独立状态而不是复用 is_published：
		// is_published 是"是否对外可见"的布尔投影，评审是"谁批准的"过程事实。
		// 用布尔表达评审会导致「已发布但没人审过」与「审过并发布」不可区分，
		// 审计时无法回答"这篇知识是谁批的"。
		//
		// 为什么 Default 是 draft 而不是 published：
		// 存量文章在加字段前多为 is_published=true。若默认 draft，
		// 加字段瞬间它们会从"已发布"变成"未评审"，造成 RAG 检索大面积掉量。
		// 因此存量按 is_published 反向初始化（见迁移 SQL），新文章才从 draft 起。
		field.Enum("review_status").
			Values("draft", "under_review", "approved", "rejected").
			Comment("评审状态：draft 草稿 / under_review 评审中 / approved 已通过 / rejected 已驳回").
			Default("draft"),
		field.Int("reviewer_id").Comment("最近一次评审人ID；空=未评审").Optional().Nillable(),
		field.Time("reviewed_at").Comment("最近一次评审时间；空=未评审").Optional().Nillable(),
		field.Text("review_comment").Comment("最近一次评审意见").Optional(),
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
		edge.To("review_records", KnowledgeArticleReview.Type),
	}
}

// KnowledgeArticleReview 记录每一次评审动作（提交/通过/驳回/撤回）。
//
// 为什么不把评审历史塞进文章本身：文章的 review_* 字段只保存**最近一次**，
// 而「谁在什么时候驳回过我、驳回理由是什么」是知识治理的核心审计需求，
// 压成单行就答不出来了。
type KnowledgeArticleReview struct{ ent.Schema }

func (KnowledgeArticleReview) Fields() []ent.Field {
	return []ent.Field{
		field.Int("article_id").Comment("文章ID").Positive(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		// from_status/to_status 记录流转前后状态，驳回-重提这类往返才能被还原。
		// 不用枚举而用 String：这里存的是**历史值**，将来枚举扩展时
		// 老记录里的值仍需可读，用 String 避免"读老行报 InvalidValue"。
		field.String("from_status").Comment("流转前状态").Default("draft"),
		field.String("to_status").Comment("流转后状态"),
		field.String("action").Comment("动作：submit/approve/reject/withdraw"),
		field.Int("operator_id").Comment("操作人ID").Positive(),
		field.Text("comment").Comment("评审意见").Optional(),
		field.Time("created_at").Comment("操作时间").Default(time.Now),
	}
}

func (KnowledgeArticleReview) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("article", KnowledgeArticle.Type).
			Ref("review_records").
			Field("article_id").
			Required().
			Unique(),
	}
}

func (KnowledgeArticleReview) Indexes() []ent.Index {
	return []ent.Index{
		// 同一文章同一动作只留一条未消费记录时用；此处仅建查询索引，
		// 唯一性留给业务层按 (article_id, to_status, action) 判定。
		index.Fields("article_id", "created_at"),
		index.Fields("tenant_id", "article_id"),
	}
}
