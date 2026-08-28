package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	connectorVector "itsm-backend/connector/vector"
	"itsm-backend/ent"
	ka "itsm-backend/ent/knowledgearticle"
)

// RAGService provides retrieval augmented generation over Knowledge Base
type RAGService struct {
	client       *ent.Client
	vectors      *VectorStore
	vectorStore  connectorVector.VectorStore
	embedder     Embedder
	logger       *zap.SugaredLogger
	useVector    bool // Whether to use vector search
	useKeyword   bool // Whether to use keyword fallback
	hybridSearch bool // Whether to use hybrid (vector + keyword) search
}

// SetVectorStore installs the pluggable connector-backed store. The legacy
// PGVector store remains available during migration of existing installations.
func (r *RAGService) SetVectorStore(store connectorVector.VectorStore) {
	r.vectorStore = store
	if store != nil {
		r.useVector = true
		r.hybridSearch = true
	}
}

// RAGConfig holds RAG service configuration
type RAGConfig struct {
	UseVector           bool
	UseKeyword          bool
	HybridSearch        bool
	SimilarityThreshold float64
	MaxResults          int
}

// DefaultRAGConfig returns default RAG configuration
func DefaultRAGConfig() RAGConfig {
	return RAGConfig{
		UseVector:           true,
		UseKeyword:          true,
		HybridSearch:        true,
		SimilarityThreshold: 0.7,
		MaxResults:          5,
	}
}

// NewRAGService creates a new RAG service with configuration
func NewRAGService(client *ent.Client, vectors *VectorStore, embedder Embedder, logger *zap.SugaredLogger, cfg RAGConfig) *RAGService {
	useVector := cfg.UseVector && vectors != nil && embedder != nil
	return &RAGService{
		client:     client,
		vectors:    vectors,
		embedder:   embedder,
		logger:     logger,
		useVector:  useVector,
		useKeyword: cfg.UseKeyword,
		// hybridSearch only makes sense if vector search is available
		hybridSearch: cfg.HybridSearch && useVector,
	}
}

// NewRAGServiceWithAutoConfig creates a RAG service with automatic configuration detection
func NewRAGServiceWithAutoConfig(client *ent.Client, vectors *VectorStore, embedder Embedder, logger *zap.SugaredLogger) *RAGService {
	cfg := DefaultRAGConfig()
	// Check if vector store is actually available
	// vectors is never nil but may be non-functional if the vectors table doesn't exist
	// embedder may be valid but fail if no API key is configured
	vectorAvailable := vectors != nil && embedder != nil
	if vectorAvailable {
		// Test if embedder works (requires valid API key)
		if testEmbed, ok := embedder.(interface {
			Embed(string) ([]float32, error)
		}); ok {
			if _, err := testEmbed.Embed("test"); err != nil {
				logger.Warnw("RAGService: embedder not functional", "error", err)
				vectorAvailable = false
			}
		}
	}
	if !vectorAvailable {
		cfg.UseVector = false
		cfg.HybridSearch = false
		logger.Warn("RAGService: vector store or embedder not available, falling back to keyword search")
	}
	// Test if vector search is actually functional
	if cfg.UseVector && vectors != nil {
		if err := vectors.TestConnection(); err != nil {
			logger.Warnw("RAGService: vector table not available, disabling vector search", "error", err)
			cfg.UseVector = false
			cfg.HybridSearch = false
		}
	}
	return NewRAGService(client, vectors, embedder, logger, cfg)
}

// Ask performs retrieval augmented generation over knowledge articles
func (r *RAGService) Ask(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}

	r.logger.Debugw("RAGService Ask called",
		"query", query,
		"tenantID", tenantID,
		"hybridSearch", r.hybridSearch,
		"useVector", r.useVector,
		"useKeyword", r.useKeyword,
		"limit", limit)

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
	if r.vectorStore != nil {
		var embedding []float32
		if r.embedder != nil {
			var err error
			embedding, err = r.embedder.Embed(query)
			if err != nil {
				r.logger.Warnw("RAGService: embedding unavailable; trying keyword vector backend", "error", err)
			}
		}
		response, err := r.vectorStore.Search(ctx, connectorVector.SearchRequest{Vector: embedding, Query: query, TopK: limit, Filter: map[string]interface{}{"tenantID": tenantID, "objectType": "kb"}})
		if err != nil {
			// Fallback to legacy store if connector fails
			if r.vectors != nil && r.embedder != nil {
				r.logger.Warnw("RAGService: connector vector search failed, falling back to legacy store", "error", err)
			} else {
				return nil, fmt.Errorf("vector connector search: %w", err)
			}
		} else {
			results := make([]map[string]any, 0, len(response.Results))
			for _, hit := range response.Results {
				objID, err := strconv.Atoi(hit.ID)
				if err != nil {
					continue
				}
				a, err := r.client.KnowledgeArticle.Query().Where(ka.IDEQ(objID), ka.TenantIDEQ(tenantID), ka.DeletedAtIsNil(), ka.IsPublished(true)).Only(ctx)
				if err != nil {
					continue
				}
				results = append(results, map[string]any{"object_type": "kb", "id": objID, "title": a.Title, "category": a.Category, "snippet": snippet(hit.Content, 200), "score": hit.Score, "search_type": response.Backend})
			}
			return results, nil
		}
	}

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

		// Enrich with knowledge article metadata.
		// 可见性过滤：仅保留存在、未软删除且已发布的文章；否则跳过该条结果，
		// 避免向量索引残留（软删除/未发布文章）泄漏到检索结果。
		if objType == "kb" {
			a, err := r.client.KnowledgeArticle.Query().
				Where(ka.IDEQ(objID), ka.TenantIDEQ(tenantID), ka.DeletedAtIsNil(), ka.IsPublished(true)).
				Only(ctx)
			if err != nil {
				r.logger.Debugw("RAGService: skip vector result, article not visible", "article_id", objID, "error", err)
				continue
			}
			item["title"] = a.Title
			item["category"] = a.Category
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

	q := r.client.KnowledgeArticle.Query().
		// 可见性过滤：仅检索本租户、未软删除且已发布的文章，草稿不得进入 RAG 结果。
		Where(ka.TenantIDEQ(tenantID), ka.DeletedAtIsNil(), ka.IsPublished(true))
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
		return "知识库中暂无相关内容。请尝试更换关键词或补充上下文，或联系知识管理员补充相关文章。", nil
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

	response, err := gateway.Chat(ctx, "", messages)
	if err != nil {
		return "", fmt.Errorf("LLM response generation failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// AskWithLLMStream performs RAG and streams the LLM answer. It first retrieves
// relevant documents (returned as sources so the caller can render citations),
// then streams the generated answer through onDelta. If gateway is nil or the
// LLM call fails, the function falls back to concatenating snippets so the
// caller still has something to display.
func (r *RAGService) AskWithLLMStream(
	ctx context.Context,
	tenantID int,
	query string,
	gateway *LLMGateway,
	maxResults int,
	onSources func(sources []map[string]any),
	onDelta func(delta string),
) error {
	if maxResults <= 0 {
		maxResults = 5
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	if onSources == nil {
		onSources = func([]map[string]any) {}
	}

	docs, err := r.Ask(ctx, tenantID, query, maxResults)
	if err != nil {
		return fmt.Errorf("retrieval failed: %w", err)
	}

	// Emit sources first so the UI can show citations while the answer streams.
	onSources(docs)

	if len(docs) == 0 {
		onDelta("知识库中暂无相关内容。以下为可能的原因与建议：\n\n")
		onDelta("1. 知识库尚未录入相关文章——请联系知识管理员补充运维手册或FAQ；\n")
		onDelta("2. 您的问题关键词与文章标题不匹配——请尝试更换关键词或补充上下文，如产品名、错误码、症状等；\n")
		onDelta("3. 知识文章未发布或处于草稿状态——请在知识库页面检查文章状态；\n")
		onDelta("4. 当前租户下未创建任何知识文章——可访问「知识库 → 新建文章」录入内容后重试。\n\n")
		onDelta("您也可以直接联系运维团队或访问服务目录获取人工支持。")
		return nil
	}

	// Fallback path when there is no LLM gateway: return concatenated snippets.
	if gateway == nil {
		var b strings.Builder
		b.WriteString("知识库检索结果如下：\n\n")
		for i, doc := range docs {
			b.WriteString(fmt.Sprintf("【%d】%v\n", i+1, doc["title"]))
			if snip, ok := doc["snippet"].(string); ok && snip != "" {
				b.WriteString(snip)
				b.WriteString("\n\n")
			}
		}
		onDelta(b.String())
		return nil
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("基于以下知识库内容回答用户问题：\n\n")
	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("【文档%d】%v\n", i+1, doc["title"]))
		contextBuilder.WriteString(fmt.Sprintf("内容：%v\n\n", doc["snippet"]))
	}

	prompt := fmt.Sprintf(`%s
用户问题：%s

请根据以上知识库内容，用简洁专业的中文回答用户问题。
如果知识库内容没有直接相关的信息，请说明"未在知识库中找到相关答案"。
可以在回答末尾用【文档X】形式引用来源，但不要重复输出文档全文。

回答：`, contextBuilder.String(), query)

	messages := []LLMMessage{
		{Role: "system", Content: "你是IT服务管理知识库助手，基于检索到的知识回答用户问题。"},
		{Role: "user", Content: prompt},
	}

	if err := gateway.ChatStream(ctx, "", messages, onDelta); err != nil {
		return fmt.Errorf("LLM stream failed: %w", err)
	}
	return nil
}

// IndexArticle adds a knowledge article to all available vector stores.
// It writes to both the connector store and the legacy store simultaneously
// to keep dual-write consistent. Failures are logged but non-fatal to avoid
// blocking article publishing when one store is unavailable.
func (r *RAGService) IndexArticle(ctx context.Context, tenantID int, articleID int, title, content string) error {
	// If both vector and embedder are disabled, skip silently
	if !r.useVector || (r.vectorStore == nil && r.vectors == nil) {
		r.logger.Debugw("RAGService: vector indexing disabled")
		return nil
	}

	// Generate embedding once; both stores share the same vector
	var embedding []float32
	if r.embedder != nil {
		var err error
		embedding, err = r.embedder.Embed(title + "\n" + content)
		if err != nil {
			r.logger.Warnw("RAGService: failed to generate embedding, skipping all vector stores", "article_id", articleID, "error", err)
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	} else {
		// No embedder: keyword fallback can still index if legacy store is available
		r.logger.Debugw("RAGService: no embedder, skipping vector indexing", "article_id", articleID)
		return nil
	}

	insertReq := connectorVector.InsertRequest{
		Chunks: []connectorVector.ChunkInput{{
			ID:      fmt.Sprintf("tenant:%d:kb:%d", tenantID, articleID),
			Content: content,
			Vector:  embedding,
			Metadata: map[string]interface{}{
				"tenantID":   tenantID,
				"objectType": "kb",
				"source":     title,
			},
		}},
	}

	// Dual-write: write to both connector and legacy simultaneously
	if r.vectorStore != nil {
		if err := r.vectorStore.Insert(ctx, insertReq); err != nil {
			return fmt.Errorf("connector vector insert: %w", err)
		}
	}
	if r.vectors != nil {
		if err := r.vectors.Upsert(ctx, tenantID, "kb", articleID, embedding, content, title); err != nil {
			return fmt.Errorf("legacy vector upsert: %w", err)
		}
	}

	return nil
}

// RemoveArticle removes a knowledge article from the vector store.
// 真实删除：软删除/取消发布文章时调用，物理移除 vectors 表中的残留向量，
// 使检索侧不再依赖 enrichment 阶段的兜底过滤。幂等：条目不存在时静默成功。
func (r *RAGService) RemoveArticle(ctx context.Context, tenantID int, articleID int) error {
	// Clean connector store
	if r.vectorStore != nil {
		if err := r.vectorStore.Delete(ctx, []string{fmt.Sprintf("tenant:%d:kb:%d", tenantID, articleID)}); err != nil {
			r.logger.Warnw("RAGService: failed to remove article from connector vector store", "article_id", articleID, "tenant_id", tenantID, "error", err)
		}
	}
	// Also clean legacy store if available
	if !r.useVector || r.vectors == nil {
		return nil
	}
	if err := r.vectors.Delete(ctx, tenantID, "kb", articleID); err != nil {
		r.logger.Warnw("RAGService: failed to remove article vector from legacy store", "article_id", articleID, "tenant_id", tenantID, "error", err)
		return fmt.Errorf("failed to remove article vector: %w", err)
	}
	r.logger.Infow("RAGService: article vector removed", "article_id", articleID, "tenant_id", tenantID)
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
