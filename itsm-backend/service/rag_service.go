package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	connectorVector "itsm-backend/connector/vector"
	"itsm-backend/ent"
	ka "itsm-backend/ent/knowledgearticle"
	"itsm-backend/handlers/common/knowledgeaccess"
)

// RAGService provides retrieval augmented generation over Knowledge Base
type RAGService struct {
	client       *ent.Client
	vectors      *VectorStore
	vectorStore  connectorVector.VectorStore
	embedder     Embedder
	logger       *zap.SugaredLogger
	useVector    bool             // Whether to use vector search
	useKeyword   bool             // Whether to use keyword fallback
	hybridSearch bool             // Whether to use hybrid (vector + keyword) search
	ontology     *OntologyService // 本体图检索：实体识别 + 关系扩展（可选注入）

	// knowledgeGuard 知识分类可见性守卫（知识可引用性 L0 权限边界）。
	// 阻断「同租户内任何用户都能通过 AI 助手问出受限知识」的越权路径。
	// nil 时守卫不生效（仅本地/测试环境允许），生产装配必须注入。
	knowledgeGuard *knowledgeaccess.Guard
}

// SetKnowledgeGuard 注入知识分类可见性守卫。
func (r *RAGService) SetKnowledgeGuard(g *knowledgeaccess.Guard) { r.knowledgeGuard = g }

// SetVectorStore installs the pluggable connector-backed store. The legacy
// PGVector store remains available during migration of existing installations.
func (r *RAGService) SetVectorStore(store connectorVector.VectorStore) {
	r.vectorStore = store
	if store != nil {
		r.useVector = true
		r.hybridSearch = true
	}
}

// SetOntologyService installs the ontology graph service. When installed,
// AskWithLLMStream recognizes business entities (tickets/incidents/CIs/...)
// in the user query and injects their 1-hop relation neighborhood into the
// LLM context and the UI sources list. nil-safe: absent means legacy behavior.
func (r *RAGService) SetOntologyService(svc *OntologyService) {
	r.ontology = svc
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
	svc := &RAGService{
		client:     client,
		vectors:    vectors,
		embedder:   embedder,
		logger:     logger,
		useVector:  useVector,
		useKeyword: cfg.UseKeyword,
		// hybridSearch only makes sense if vector search is available
		hybridSearch: cfg.HybridSearch && useVector,
	}
	// 默认装配守卫：只要持有 ent client 就启用分类级可见性管控。
	// 未注入 Viewer 的调用方会按匿名处理（仅放行未纳管分类），属 fail-closed。
	if client != nil {
		svc.knowledgeGuard = knowledgeaccess.NewGuard(client, logger)
	}
	return svc
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
				// 知识分类可见性（L0）：向量索引里可能残留受限分类文章，
				// 必须在这里拦截，否则会绕过 keywordSearch 的 SQL 层过滤。
				if !r.articleReadable(ctx, tenantID, a) {
					r.logger.Debugw("RAG: skip connector vector result, category not readable",
						"article_id", objID, "category", a.Category)
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
			// 知识分类可见性（L0）：向量索引里可能残留受限分类文章，
			// 必须在这里拦截，否则会绕过 keywordSearch 的 SQL 层过滤。
			if !r.articleReadable(ctx, tenantID, a) {
				r.logger.Debugw("RAGService: skip vector result, category not readable",
					"article_id", objID, "category", a.Category)
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
// articleReadable 判定单篇文章对当前访问者是否可读（分类可见性 L0）。
// 用于向量检索结果的后置过滤：向量索引是异步构建的，可能残留受限分类文章
// 或权限变更前的快照，无法依赖 SQL 层过滤兜底。
//
// 守卫未装配时恒为 true（保持旧行为）。
func (r *RAGService) articleReadable(ctx context.Context, tenantID int, a *ent.KnowledgeArticle) bool {
	if r.knowledgeGuard == nil || a == nil {
		return true
	}
	viewer, hasViewer := knowledgeaccess.ViewerFrom(ctx)
	if !hasViewer {
		r.logger.Debugw("RAG: 未注入 Viewer，按匿名处理", "tenant_id", tenantID, "article_id", a.ID)
		viewer = knowledgeaccess.Viewer{}
	}
	return r.knowledgeGuard.CanReadCategory(ctx, tenantID, viewer, a.Category, a.AuthorID)
}

// deniedCategories 返回当前访问者无权读取的知识分类。
//
// 知识可引用性 L0：RAG 检索此前只按 tenant_id + is_published + deleted_at 过滤，
// 同租户内任何用户都能通过 AI 助手问出受限分类（财务/HR/高管）的知识。
// 这里按 Viewer（userID + role）逐分类判定，返回应被排除的分类名。
//
// 守卫未装配时返回 nil（不做额外限制，保持旧行为）。
// Viewer 缺失时按匿名处理：所有已纳管分类一律排除（fail-closed）。
func (r *RAGService) deniedCategories(ctx context.Context, tenantID int) []string {
	if r.knowledgeGuard == nil {
		return nil
	}
	viewer, hasViewer := knowledgeaccess.ViewerFrom(ctx)
	if !hasViewer {
		r.logger.Debugw("RAG: 未注入 Viewer，按匿名处理，受限分类一律排除", "tenant_id", tenantID)
		viewer = knowledgeaccess.Viewer{}
	}

	restricted, err := r.knowledgeGuard.RestrictedCategories(ctx, tenantID)
	if err != nil {
		// 查询失败按最严处理：排除所有受限分类，绝不放行未知权限状态的内容
		r.logger.Warnw("RAG: 受限分类查询失败，按 fail-closed 排除全部受限分类",
			"tenant_id", tenantID, "error", err)
		restricted = nil
		if r.knowledgeGuard != nil {
			// 缓存失效场景下无法拿到集合，直接走保守策略：拒绝全部（返回哨兵）
			return []string{denyAllSentinel}
		}
	}

	denied := make([]string, 0, len(restricted))
	for cat := range restricted {
		// authorID 传 0：SQL 层无法逐条判断作者，作者豁免在 FilterArticles 里处理
		if !r.knowledgeGuard.CanReadCategory(ctx, tenantID, viewer, cat, 0) {
			denied = append(denied, cat)
		}
	}
	return denied
}

// denyAllSentinel 受限分类集合不可得时的拒绝哨兵，配合 CategoryNotIn 使用。
const denyAllSentinel = "\x00__deny_all__"

func (r *RAGService) keywordSearch(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if !r.useKeyword {
		return nil, fmt.Errorf("keyword search not available")
	}

	q := r.client.KnowledgeArticle.Query().
		// 可见性过滤：仅检索本租户、未软删除且已发布的文章，草稿不得进入 RAG 结果。
		Where(ka.TenantIDEQ(tenantID), ka.DeletedAtIsNil(), ka.IsPublished(true))

	// 知识分类可见性（L0 权限边界）：在 SQL 层排除无权访问的分类，
	// 保证 limit 语义准确（后置过滤会让召回数不足）。
	// 作者本人的文章即便落在受限分类也放行。
	viewer, _ := knowledgeaccess.ViewerFrom(ctx)
	if denied := r.deniedCategories(ctx, tenantID); len(denied) > 0 {
		if len(denied) == 1 && denied[0] == denyAllSentinel {
			// 权限状态未知，最保守：仅允许本分类体系外的空分类文章
			q = q.Where(ka.CategoryEQ(""))
		} else if viewer.UserID > 0 {
			// 作者豁免：自己写的文章即便落在受限分类也可见
			q = q.Where(ka.Or(
				ka.CategoryNotIn(denied...),
				ka.AuthorIDEQ(viewer.UserID),
			))
			r.logger.Debugw("RAG: 已按分类可见性过滤", "tenant_id", tenantID, "denied", denied, "user_id", viewer.UserID)
		} else {
			// 匿名/无用户上下文：不做作者豁免
			q = q.Where(ka.CategoryNotIn(denied...))
			r.logger.Debugw("RAG: 已按分类可见性过滤（匿名）", "tenant_id", tenantID, "denied", denied)
		}
	}

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
		// 知识库无匹配时仍尝试用 LLM 通用知识回答；仅当无网关时才退回模板。
		if gateway != nil {
			messages := []LLMMessage{
				{Role: "system", Content: "你是IT服务管理(ITSM)智能助手。当前问题在知识库中未检索到相关文章。请基于你的通用IT服务管理/IT运维知识直接回答用户；如合适，可简要说明你还能提供的帮助。回答需简洁专业，使用中文。"},
				{Role: "user", Content: query},
			}
			if resp, err := gateway.Chat(ctx, "", messages); err == nil {
				return strings.TrimSpace(resp), nil
			}
		}
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

	// 本体增强：识别 query 中的业务实体（工单号/事件号/CI 名等），
	// 做 1 跳关系扩展，实体卡注入 sources、图谱事实注入 prompt。
	// 查询失败仅降级（不注入），不影响 KB 主链路。
	var ontologyBlock string
	if r.ontology != nil {
		oc := r.ontology.ExtractAndExpand(ctx, tenantID, query)
		if !oc.Empty() {
			ontologyBlock = oc.PromptBlock()
			docs = append(oc.Sources(), docs...)
		}
	}

	// Emit sources first so the UI can show citations while the answer streams.
	onSources(docs)

	if len(docs) == 0 {
		// 知识库无匹配文章时：若已配置 LLM 网关，仍调用大模型用通用 ITSM 知识回答，
		// 而不是直接返回"无内容"模板——否则助手对通用问题（如"你能做什么"）完全失效。
		if gateway != nil {
			generalMessages := []LLMMessage{
				{Role: "system", Content: "你是IT服务管理(ITSM)智能助手。当前问题在知识库中未检索到相关文章。请基于你的通用IT服务管理/IT运维知识直接回答用户；如合适，可简要说明你还能提供的帮助（如协助创建或查询工单、检索知识库、解释SLA/变更/事件/CMDB等ITSM概念）。回答需简洁专业，使用中文。"},
				{Role: "user", Content: query},
			}
			if err := gateway.ChatStream(ctx, "", generalMessages, onDelta); err != nil {
				r.logger.Warnw("RAGService: general LLM answer failed, falling back to KB-empty template", "error", err)
			} else {
				return nil
			}
		}
		// 无 LLM 网关或 LLM 调用失败：给出静态引导模板
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
	// 本体图谱块：业务实体及其关系邻居（若有）
	if ontologyBlock != "" {
		contextBuilder.WriteString(ontologyBlock)
	}

	prompt := fmt.Sprintf(`%s
用户问题：%s

请根据以上知识库内容与业务实体图谱（如有），用简洁专业的中文回答用户问题。
回答涉及具体业务对象（工单/事件/CI/问题/发布）时，优先依据图谱上下文中的事实，给出对象编号、状态与关联关系。
如果知识库内容没有直接相关的信息，请说明"未在知识库中找到相关答案"。
可以在回答末尾用【文档X】形式引用来源，但不要重复输出文档全文。

回答：`, contextBuilder.String(), query)

	messages := []LLMMessage{
		{Role: "system", Content: "你是IT服务管理知识库助手，基于检索到的知识内容与CMDB/工单图谱事实回答用户问题。图谱中的对象编号、状态、关联关系是系统实时数据，可信度高于推测。"},
		{Role: "user", Content: prompt},
	}

	if err := gateway.ChatStream(ctx, "", messages, onDelta); err != nil {
		return fmt.Errorf("LLM stream failed: %w", err)
	}
	return nil
}

// maxToolRounds 工具循环上限，防止模型无限循环调用工具。
const maxToolRounds = 5

// AskWithLLMStreamWithTools 是 AskWithLLMStream 的工具增强版本：除检索知识库与本体图谱外，
// 还向 LLM 声明一组只读工具。模型若发起工具调用，通过 execTool 执行（由调用方负责
// RBAC 校验与审计），结果回填对话后继续生成，最终答案经 onDelta 流式下发。
//
// 不改动原 AskWithLLMStream 签名；当 gateway 为 nil 或 tools 为空时完全退化为原方法行为，
// 保证既有调用方零回归。execTool 返回的 error 会被序列化为工具结果（{"error": ...}），
// 让模型感知执行失败而不中断整条流。
func (r *RAGService) AskWithLLMStreamWithTools(
	ctx context.Context,
	tenantID int,
	query string,
	gateway *LLMGateway,
	maxResults int,
	tools []LLMTool,
	onSources func(sources []map[string]any),
	onDelta func(delta string),
	execTool func(name string, args map[string]any) (any, error),
) error {
	if gateway == nil || len(tools) == 0 {
		return r.AskWithLLMStream(ctx, tenantID, query, gateway, maxResults, onSources, onDelta)
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	if onSources == nil {
		onSources = func([]map[string]any) {}
	}

	// 检索（与 AskWithLLMStream 一致：KB + 本体增强）
	docs, err := r.Ask(ctx, tenantID, query, maxResults)
	if err != nil {
		return fmt.Errorf("retrieval failed: %w", err)
	}
	var ontologyBlock string
	if r.ontology != nil {
		oc := r.ontology.ExtractAndExpand(ctx, tenantID, query)
		if !oc.Empty() {
			ontologyBlock = oc.PromptBlock()
			docs = append(oc.Sources(), docs...)
		}
	}
	// 先发 sources，让前端在答案流式期间渲染引用/实体卡
	onSources(docs)

	var contextBuilder strings.Builder
	if len(docs) > 0 {
		contextBuilder.WriteString("基于以下知识库内容回答用户问题：\n\n")
		for i, doc := range docs {
			contextBuilder.WriteString(fmt.Sprintf("【文档%d】%v\n", i+1, doc["title"]))
			contextBuilder.WriteString(fmt.Sprintf("内容：%v\n\n", doc["snippet"]))
		}
	} else {
		contextBuilder.WriteString("知识库未检索到相关内容。如用户需要实时业务数据（工单/事件/CI 等），请优先使用工具查询。\n\n")
	}
	if ontologyBlock != "" {
		contextBuilder.WriteString(ontologyBlock)
	}

	prompt := fmt.Sprintf(`%s
用户问题：%s

请根据以上知识库内容与业务实体图谱（如有），用简洁专业的中文回答用户问题。
回答涉及具体业务对象（工单/事件/CI/问题/发布）时，优先依据图谱上下文中的事实，给出对象编号、状态与关联关系。
如用户需要实时业务数据（例如当前有哪些工单），请调用提供的工具查询后再回答，不要臆造数据。
如果知识库内容没有直接相关的信息，请说明"未在知识库中找到相关答案"。
可以在回答末尾用【文档X】形式引用来源，但不要重复输出文档全文。

回答：`, contextBuilder.String(), query)

	messages := []LLMMessage{
		{Role: "system", Content: `你是 IT 服务管理（ITSM）智能助手，基于检索到的知识库内容与 CMDB/工单图谱事实回答用户问题。请遵守以下约定：
1. 口语与错别字纠正：用户常把"工单"说成"工地"，"单子/报修/工单子"也指工单；"测试工单/探针工单"通常指 E2E 或探针产生的测试数据，可先用 list_tickets 查询并询问是否需要清理归档，不要当作未知概念去检索知识库。
2. 图谱中的对象编号、状态、关联关系是系统实时数据，可信度高于推测。需要实时数据（当前有哪些工单、事件统计、CI 列表等）时，必须优先调用已提供的工具获取，严禁编造数据。
3. 你可以调用 create_ticket 创建工单、create_ticket_type 创建工单类型、update_ticket 更新工单；这些写操作会进入审批流，调用后请明确告知用户"已提交、待人工审批（含 invocationId）"，并说明审批通过后才会正式生效。
4. CMDB 本体关联（故障→配置项→工单）：当用户报告某台设备/数据库/服务/网络故障（如"HIS-DB-01 连接超时""护士站电脑蓝屏""PACS 上传失败"）时，必须先定位受影响的配置项（CI），再把工单挂到该 CI 上，形成 ITSM↔CMDB 本体闭环。操作顺序：
   (a) 用 list_cis 定位 CI —— 支持 search 按名称/资产标签/序列号/型号/厂商/云资源ID 模糊匹配，也支持 ci_type 按类型过滤（server/database/application/network/storage/cloud_resource）。从返回结果中取 id 作为 ci_id。
   (b) 建单时带 ci_id：调用 create_ticket 时把 ci_id 一并传入，工单创建后会自动绑定到该配置项。
   (c) 若用户先报障建单、后才说清是哪台设备，用 link_ticket_ci 把已存在的工单补挂到 CI（需审批）。
   (d) 影响面分析：用 get_ci_tickets 查询某个 CI 上已关联的工单，判断是否为重复报障、该资产是否反复故障。这在回答"这台服务器最近怎么老出问题"类问题时是必做步骤。
   (e) 若 list_cis 未找到匹配 CI，可正常创建工单，并在回答中明确提示用户补充设备名称/资产编号，不要编造 ci_id。`},
		{Role: "user", Content: prompt},
	}

	// 工具循环：模型发起调用 → 执行 → 结果回填 → 再请求，直到模型给出最终回答
	for round := 0; round < maxToolRounds; round++ {
		var toolCalls []LLMToolCall
		if err := gateway.ChatStreamWithTools(ctx, "", messages, tools, onDelta, func(tcs []LLMToolCall) {
			toolCalls = tcs
		}); err != nil {
			return fmt.Errorf("LLM stream failed: %w", err)
		}
		if len(toolCalls) == 0 {
			// 无工具调用：本轮已流式输出最终答案
			return nil
		}

		// 回填 assistant 的工具调用消息与各工具的执行结果
		messages = append(messages, LLMMessage{Role: "assistant", ToolCalls: toolCalls})
		for _, tc := range toolCalls {
			var result any
			var execErr error
			if execTool != nil {
				result, execErr = execTool(tc.Name, parseToolArgs(tc.Arguments))
			} else {
				execErr = fmt.Errorf("no tool executor provided")
			}
			if execErr != nil {
				result = map[string]any{"error": execErr.Error()}
			}
			b, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				b = []byte(`{"error":"failed to serialize tool result"}`)
			}
			messages = append(messages, LLMMessage{Role: "tool", ToolCallID: tc.ID, Content: string(b)})
		}
		// 继续下一轮，让模型基于工具结果生成最终回答
	}
	return fmt.Errorf("tool loop exceeded max rounds (%d)", maxToolRounds)
}

// parseToolArgs 解析模型返回的工具参数 JSON；空串/非法 JSON 时返回空 map 或降级兜底，
// 保证工具执行不会因参数解析失败而 panic。
func parseToolArgs(s string) map[string]any {
	args := map[string]any{}
	if strings.TrimSpace(s) == "" {
		return args
	}
	if err := json.Unmarshal([]byte(s), &args); err != nil {
		// 非 JSON 对象：整段作为 value 传给工具，工具内部自行容错
		args["value"] = s
	}
	return args
}

// IndexArticle adds a knowledge article to all available vector stores.
// It writes to both the connector store and the legacy store and reports any
// partial failure so callers do not mistake an incomplete index for success.
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
			if r.vectors != nil {
				if compensationErr := r.vectors.Delete(ctx, tenantID, "kb", articleID); compensationErr != nil {
					r.logger.Errorw("RAGService: failed to compensate legacy vector after connector insert failure", "article_id", articleID, "tenant_id", tenantID, "error", compensationErr)
				}
			}
			return fmt.Errorf("connector vector insert: %w", err)
		}
	}
	if r.vectors != nil {
		if err := r.vectors.Upsert(ctx, tenantID, "kb", articleID, embedding, content, title); err != nil {
			r.logger.Errorw("RAGService: legacy vector upsert failed after connector insert", "article_id", articleID, "tenant_id", tenantID, "error", err)
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
