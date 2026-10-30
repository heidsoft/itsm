package knowledge

import (
	"errors"
	"time"
)

// ErrArticleNotFound 文章不存在或不属于当前租户。
// 版本表无 tenant_id，租户隔离经文章归属校验实现，失败时返回本错误。
var ErrArticleNotFound = errors.New("article not found")

// Article representing a knowledge base article
type Article struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	AuthorID    int      `json:"authorId"`
	TenantID    int      `json:"tenantId"`
	IsPublished bool     `json:"isPublished"`

	// 评审状态机。ReviewStatus 为空按 "draft" 处理，
	// 兼容未回填该字段的历史响应。
	ReviewStatus  string     `json:"reviewStatus,omitempty"`
	ReviewerID    *int       `json:"reviewerId,omitempty"`
	ReviewedAt    *time.Time `json:"reviewedAt,omitempty"`
	ReviewComment string     `json:"reviewComment,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// RelevanceScore is set by SearchArticles when returning RAG results (0 if not from RAG).
	// Values are in [0, 1]; higher means more relevant.
	RelevanceScore float64 `json:"relevanceScore,omitempty"`

	// 知识可引用性 L1：时效性。三个时间字段为 nil 表示未设置，
	// 语义是「长期有效 / 不设复核」。
	ValidFrom          *time.Time `json:"validFrom,omitempty"`
	ValidUntil         *time.Time `json:"validUntil,omitempty"`
	LastReviewedAt     *time.Time `json:"lastReviewedAt,omitempty"`
	ReviewIntervalDays int        `json:"reviewIntervalDays,omitempty"`

	// 知识可引用性 L2：权威性。仅影响检索排序，不影响准入。
	AuthorityLevel int `json:"authorityLevel,omitempty"`

	// Freshness 为派生字段，仅在检索/详情响应中填充，用于前端提示「该内容已逾期未复核」。
	// 不参与写入，写入请以 ValidUntil / LastReviewedAt 等原始字段为准。
	Freshness string `json:"freshness,omitempty"`

	// 来源追溯：文章由哪个问题沉淀而来。nil = 手工创建。
	// 裸 ID 引用而非外键：知识是独立资产，问题删除不级联删知识。
	SourceProblemID *int `json:"sourceProblemId,omitempty"`

	// PublishedVersion 指向发布时刻落的版本快照：回答「线上是哪个版本」。
	// nil = 从未发布或已撤回。读取路径填充；发布/撤回由仓储层维护，
	// 客户端传入的值一律忽略（否则任何 PUT 都能改写线上版本的指向）。
	PublishedVersion *int `json:"publishedVersion,omitempty"`
}

// Category represents a knowledge category
type Category struct {
	Name string `json:"name"`
}

// ArticleVersion 文章版本快照。更新正文前自动创建，用于历史回溯与恢复。
// JSON 键与前端 ArticleVersion 契约对齐：createdByName/changeLog 是展示字段，
// 由仓储层在读取时补齐（changeLog 取自变更摘要，createdByName 查用户表）。
type ArticleVersion struct {
	Version       int       `json:"version"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	CreatedBy     int       `json:"createdBy"`
	CreatedByName string    `json:"createdByName"`
	ChangeLog     string    `json:"changeLog"`
	Summary       string    `json:"summary"`
	CreatedAt     time.Time `json:"createdAt"`
}
