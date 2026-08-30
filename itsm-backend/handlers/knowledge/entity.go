package knowledge

import (
	"time"
)

// Article representing a knowledge base article
type Article struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	AuthorID    int       `json:"authorId"`
	TenantID    int       `json:"tenantId"`
	IsPublished bool      `json:"isPublished"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	// RelevanceScore is set by SearchArticles when returning RAG results (0 if not from RAG).
	// Values are in [0, 1]; higher means more relevant.
	RelevanceScore float64 `json:"relevanceScore,omitempty"`

	// 知识可引用性 L1：时效性。三个时间字段为 nil 表示未设置，
	// 语义是「长期有效 / 不设复核」，与本次改动前的行为一致。
	ValidFrom          *time.Time `json:"validFrom,omitempty"`
	ValidUntil         *time.Time `json:"validUntil,omitempty"`
	LastReviewedAt     *time.Time `json:"lastReviewedAt,omitempty"`
	ReviewIntervalDays int        `json:"reviewIntervalDays,omitempty"`

	// 知识可引用性 L2：权威性。仅影响检索排序，不影响准入。
	AuthorityLevel int `json:"authorityLevel,omitempty"`

	// Freshness 为派生字段，仅在检索/详情响应中填充，用于前端提示「该内容已逾期未复核」。
	// 不参与写入，写入请以 ValidUntil / LastReviewedAt 等原始字段为准。
	Freshness string `json:"freshness,omitempty"`
}

// Category represents a knowledge category
type Category struct {
	Name string `json:"name"`
}
