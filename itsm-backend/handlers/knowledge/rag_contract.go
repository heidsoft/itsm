package knowledge

import "context"

// 知识域对 RAG 能力的窄契约（依赖倒置）。
//
// 历史上本域直接持有共享层 RAG 实现的具体指针类型，形成「域包 → 共享 service 包」
// 的反向依赖：共享层任何改动都会波及知识域，域包也无法在不引入共享层的
// 前提下做单元测试。这里把依赖收敛成知识域真实消费的 5 个方法，实现仍由
// internal/bootstrap 装配的共享层 RAG 实现提供（编译期断言见
// rag_contract_test.go）。本批为批次 2 开局，只抽契约、不搬实现、零行为变更。
//
// 契约边界：新增方法前先确认知识域确实需要，禁止把共享层的大接口原样搬进域包。

// RAGRetriever 是知识检索读面（Search）所需的最小契约。
type RAGRetriever interface {
	Ask(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error)
}

// RAGIndexer 是向量索引同步写面与就绪探测所需的最小契约。
// outbox worker 只依赖这一组方法，不牵连读面。
type RAGIndexer interface {
	HasVectorBackend() bool
	VectorIndexReady() bool
	IndexArticle(ctx context.Context, tenantID int, articleID int, title, content string) error
	RemoveArticle(ctx context.Context, tenantID int, articleID int) error
}

// RAGService 是 KnowledgeService 需要的组合契约（读面 + 索引就绪探测）。
type RAGService interface {
	RAGRetriever
	RAGIndexer
}
