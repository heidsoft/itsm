package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	ka "itsm-backend/ent/knowledgearticle"
)

// RAGService provides retrieval augmented generation over Knowledge Base
type RAGService struct {
	client        *ent.Client
	vectors       *VectorStore
	embedder      Embedder
	logger        *zap.SugaredLogger
	useVector     bool // Whether to use vector search
	useKeyword    bool // Whether to use keyword fallback
	hybridSearch  bool // Whether to use hybrid (vector + keyword) search
}

// RAGConfig holds RAG service configuration
type RAGConfig struct {
	UseVector          bool
	UseKeyword         bool
	HybridSearch       bool
	SimilarityThreshold float64
	MaxResults         int
}

// DefaultRAGConfig returns default RAG configuration
func DefaultRAGConfig() RAGConfig {
	return RAGConfig{
		UseVector:          true,
		UseKeyword:         true,
		HybridSearch:       true,
		SimilarityThreshold: 0.7,
		MaxResults:         5,
	}
}

// NewRAGService creates a new RAG service with configuration
func NewRAGService(client *ent.Client, vectors *VectorStore, embedder Embedder, logger *zap.SugaredLogger, cfg RAGConfig) *RAGService {
	return &RAGService{
		client:        client,
		vectors:       vectors,
		embedder:      embedder,
		logger:        logger,
		useVector:     cfg.UseVector && vectors != nil && embedder != nil,
		useKeyword:    cfg.UseKeyword,
		hybridSearch:  cfg.HybridSearch,
	}
}

// NewRAGServiceWithAutoConfig creates a RAG service with automatic configuration detection
func NewRAGServiceWithAutoConfig(client *ent.Client, vectors *VectorStore, embedder Embedder, logger *zap.SugaredLogger) *RAGService {
	cfg := DefaultRAGConfig()
	// Check if vector store is actually available
	if vectors == nil || embedder == nil {
		cfg.UseVector = false
		cfg.HybridSearch = false
		logger.Warn("RAGService: vector store or embedder not available, falling back to keyword search")
	}
	return NewRAGService(client, vectors, embedder, logger, cfg)
}

// Ask performs retrieval augmented generation over knowledge articles
func (r *RAGService) Ask(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}

	results := []map[string]any{}
	seen := map[string]struct{}{}

	// Collect results from different sources
	if r.hybridSearch {
		// Hybrid search: vector + keyword
		vectorResults, err := r.vectorSearch(ctx, tenantID, query, limit)
		if err != nil {
			r.logger.Warnw("RAGService: vector search failed", "error", err)
		} else {
			for _, item := range vectorResults {
				key := fmt.Sprintf("%s:%d", item["object_type"], item["id"])
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					results = append(results, item)
				}
			}
		}

		if len(results) < limit {
			keywordResults, err := r.keywordSearch(ctx, tenantID, query, limit-len(results))
			if err != nil {
				r.logger.Warnw("RAGService: keyword search failed", "error", err)
			} else {
				for _, item := range keywordResults {
					key := fmt.Sprintf("%s:%d", item["object_type"], item["id"])
					if _, ok := seen[key]; !ok {
						seen[key] = struct{}{}
						results = append(results, item)
					}
				}
			}
		}
	} else if r.useVector {
		// Vector-only search
		results, err := r.vectorSearch(ctx, tenantID, query, limit)
		if err != nil {
			r.logger.Warnw("RAGService: vector search failed, falling back to keyword", "error", err)
			return r.keywordSearch(ctx, tenantID, query, limit)
		}
		return results, nil
	} else if r.useKeyword {
		// Keyword-only search
		return r.keywordSearch(ctx, tenantID, query, limit)
	}

	return results, nil
}

// vectorSearch performs similarity search using vectors
func (r *RAGService) vectorSearch(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if !r.useVector || r.vectors == nil || r.embedder == nil {
		return nil, fmt.Errorf("vector search not available")
	}

	// Generate embedding for query
	embedding, err := r.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	rows, err := r.vectors.SearchTopKByType(ctx, tenantID, "kb", embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	results := []map[string]any{}
	for rows.Next() {
		var objType string
		var objID int
		var content, source sql.NullString
		var distance float64
		if err := rows.Scan(&objType, &objID, &content, &source, &distance); err != nil {
			r.logger.Warnw("RAGService: failed to scan vector result", "error", err)
			continue
		}

		// Calculate similarity score (1 - normalized distance)
		similarity := 1.0 - distance
		if similarity < 0 {
			similarity = 0
		}

		item := map[string]any{
			"object_type": objType,
			"id":          objID,
			"snippet":     snippet(content.String, 200),
			"source":      source.String,
			"score":       similarity,
			"search_type": "vector",
		}

		// Enrich with knowledge article metadata
		if objType == "kb" {
			if a, err := r.client.KnowledgeArticle.Get(ctx, objID); err == nil {
				item["title"] = a.Title
				item["category"] = a.Category
			}
		}

		results = append(results, item)
	}

	return results, nil
}

// keywordSearch performs full-text search using LIKE
func (r *RAGService) keywordSearch(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if !r.useKeyword {
		return nil, fmt.Errorf("keyword search not available")
	}

	q := r.client.KnowledgeArticle.Query().Where(ka.TenantIDEQ(tenantID))
	if qq := strings.TrimSpace(query); qq != "" {
		// Use OR for broader search
		q = q.Where(ka.Or(
			ka.TitleContainsFold(qq),
			ka.ContentContainsFold(qq),
		))
	}

	articles, err := q.Limit(limit).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	results := []map[string]any{}
	for _, a := range articles {
		// Calculate simple relevance score based on match location
		score := 0.5 // base score
		titleLower := strings.ToLower(a.Title)
		queryLower := strings.ToLower(query)
		if strings.Contains(titleLower, queryLower) {
			score = 0.9
		}

		results = append(results, map[string]any{
			"object_type": "kb",
			"id":          a.ID,
			"title":       a.Title,
			"category":    a.Category,
			"snippet":     snippet(a.Content, 160),
			"score":       score,
			"search_type": "keyword",
		})
	}

	return results, nil
}

// AskWithLLM performs RAG with LLM-generated answer
func (r *RAGService) AskWithLLM(ctx context.Context, tenantID int, query string, gateway *LLMGateway, maxResults int) (string, error) {
	if gateway == nil {
		return "", fmt.Errorf("LLM gateway not configured")
	}

	if maxResults <= 0 {
		maxResults = 5
	}

	// Get relevant documents
	docs, err := r.Ask(ctx, tenantID, query, maxResults)
	if err != nil {
		return "", err
	}

	if len(docs) == 0 {
		return "未找到相关知识库文章。", nil
	}

	// Build context from retrieved documents
	var contextBuilder strings.Builder
	contextBuilder.WriteString("基于以下知识库内容回答用户问题：\n\n")

	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("【文档%d】%s\n", i+1, doc["title"]))
		contextBuilder.WriteString(fmt.Sprintf("内容：%s\n\n", doc["snippet"]))
	}

	// Build prompt
	prompt := fmt.Sprintf(`%s
用户问题：%s

请根据以上知识库内容，用简洁专业的中文回答用户问题。
如果知识库内容没有直接相关的信息，请说明"未在知识库中找到相关答案"。
请只输出回答内容，不要引用来源。

回答：`, contextBuilder.String(), query)

	messages := []LLMMessage{
		{Role: "system", Content: "你是IT服务管理知识库助手，基于检索到的知识回答用户问题。"},
		{Role: "user", Content: prompt},
	}

	response, err := gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		return "", fmt.Errorf("LLM response generation failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// IndexArticle adds a knowledge article to the vector store
func (r *RAGService) IndexArticle(ctx context.Context, tenantID int, articleID int, title, content string) error {
	if !r.useVector || r.embedder == nil {
		r.logger.Debugw("RAGService: vector indexing disabled")
		return nil
	}

	// Generate embedding for article content
	embedding, err := r.embedder.Embed(title + "\n" + content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Upsert to vector store
	if r.vectors != nil {
		if err := r.vectors.Upsert(ctx, tenantID, "kb", articleID, embedding, content, title); err != nil {
			return fmt.Errorf("failed to upsert vector: %w", err)
		}
	}

	return nil
}

// RemoveArticle removes a knowledge article from the vector store
func (r *RAGService) RemoveArticle(ctx context.Context, tenantID int, articleID int) error {
	if !r.useVector || r.vectors == nil {
		return nil
	}

	// Vector store doesn't have delete, but we could add one
	// For now, just log
	r.logger.Debugw("RAGService: article removal from vector store not yet implemented")
	return nil
}

// GetStats returns RAG service statistics
func (r *RAGService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"use_vector":    r.useVector,
		"use_keyword":   r.useKeyword,
		"hybrid_search": r.hybridSearch,
	}
}

// CheckHealth checks RAG service health
func (r *RAGService) CheckHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}

	// Check vector store
	if r.useVector {
		if r.vectors != nil {
			health["vector_store"] = "connected"
		} else {
			health["vector_store"] = "not configured"
		}
	} else {
		health["vector_store"] = "disabled"
	}

	// Check embedder
	if r.embedder != nil {
		health["embedder"] = "available"
	} else {
		health["embedder"] = "not configured"
	}

	// Test embedding generation
	if r.embedder != nil {
		_, err := r.embedder.Embed("health check")
		if err != nil {
			health["embedder"] = fmt.Sprintf("error: %v", err)
			health["status"] = "degraded"
		}
	}

	return health
}

// snippet extracts a preview from content
func snippet(s string, n int) string {
	if n <= 0 {
		n = 160
	}
	if len(s) <= n {
		return s
	}
	// Try to cut at a sentence boundary
	cut := s[:n]
	lastPeriod := strings.LastIndex(cut, "。")
	lastNewline := strings.LastIndex(cut, "\n")
	cutPos := lastPeriod
	if lastNewline > cutPos {
		cutPos = lastNewline
	}
	if cutPos > n/2 {
		return s[:cutPos+1] + "..."
	}
	return s[:n] + "..."
}
