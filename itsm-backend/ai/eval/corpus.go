package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
)

// 本文件把 RAG 评测从「期望答案即候选」升级到「真实语料 + 真实检索器」。
//
// ## 为什么需要语料文件
//
// 此前数据集里的 expected_doc_ids 是虚构的（KB-001、KB-020……），库里根本
// 不存在这些文档。于是任何真实检索器在 CI 里的命中率必然为 0，而用期望
// 答案当候选又让指标恒为满分——两头都不成立：既测不出真实质量，又给人
// 「召回率有 CI 保护」的错觉。
//
// ## 为什么按标题引用而不是按 ID
//
// 按 ID 引用会把数据集钉死在某个库的具体主键上，换一个环境（生产库、
// 另一个租户）数据集就失效了。按**标题**引用让数据集与库的 ID 解耦：同一
// 份数据集既能在 CI 的内存库里跑，也能在接了真实知识库的生产验收里跑。
//
// ## 为什么关键词检索就够
//
// service.RAGService 的关键词路径走的是**真实 SQL**（ent 的
// KnowledgeArticle 查询，带租户谓词、发布状态、软删与时效过滤），只依赖
// ent client——不需要 pgvector、不需要外部服务。CI 里用 sqlite 内存库即可
// 跑通与生产同一条检索代码路径。向量检索需要 embedding 服务，那是生产
// 验收的事。

// RAGCorpusArticle 是评测语料里的一篇知识文章。
//
// 它是**真实的知识文章形状**：标题、正文、分类、标签、发布状态——与
// ent/schema/knowledgearticle.go 的字段对应，而不是「KB-001」这种占位 ID。
type RAGCorpusArticle struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	// Tags 是逗号分隔字符串——与生产的 KnowledgeArticle.tags 字段同形，
	// 语料因此能直接灌进真实表，不需要在评测层做形状转换。
	Tags string `json:"tags"`
	// TenantID 为 0 时用语料级的默认租户（评测统一用 1）。
	TenantID int `json:"tenant_id,omitempty"`
	// Published 默认 true：未发布的文章不得进入 RAG 结果（生产语义）。
	Published *bool `json:"published,omitempty"`
}

// RAGCorpus 是一组评测语料。
type RAGCorpus struct {
	Articles []RAGCorpusArticle `json:"articles"`
}

// IsPublished 返回文章是否发布（默认 true，与生产「草稿不进 RAG」一致）。
func (a RAGCorpusArticle) IsPublished() bool {
	if a.Published == nil {
		return true
	}
	return *a.Published
}

// NormalizeRef 归一化引用文本，用于比较（去空白、统一大小写）。
//
// 归一化是必要的：真实库的标题可能有全角空格、大小写差异，若直接比较会
// 产生「明明召回了却算未命中」的假阴性——那会让召回率被低估，掩盖真实问题。
func NormalizeRef(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// LoadRAGCorpus 从 JSONL 载入语料（每行一篇）。
//
// 语料与用例分离：语料是「库里有什么」，用例是「该检索到什么」。两者混在
// 一个文件里会导致「为了命中而写语料」的自我循环——语料必须独立于断言存在。
func LoadRAGCorpus(path string) (*RAGCorpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open corpus: %w", err)
	}
	defer f.Close()

	corpus := &RAGCorpus{}
	dec := json.NewDecoder(f)
	for {
		var a RAGCorpusArticle
		if err := dec.Decode(&a); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("decode corpus: %w", err)
		}
		if strings.TrimSpace(a.Title) == "" {
			return nil, fmt.Errorf("corpus article without title")
		}
		corpus.Articles = append(corpus.Articles, a)
	}
	if len(corpus.Articles) == 0 {
		return nil, fmt.Errorf("corpus is empty: %s", path)
	}
	return corpus, nil
}

// CorpusTitles 返回语料里所有标题（归一化后），用于检查期望引用是否指向
// **真实存在的语料**——指向不存在的引用会让期望永远无法命中，门禁恒 0，
// 那是一个「看起来在测、实际测不到」的配置错误。
func (c *RAGCorpus) CorpusTitles() map[string]struct{} {
	out := make(map[string]struct{}, len(c.Articles))
	for _, a := range c.Articles {
		out[NormalizeRef(a.Title)] = struct{}{}
	}
	return out
}

// RealRAGRetriever 用真实的 service.RAGService 检索语料。
//
// 候选是**真实检索器的产出**：真实 SQL、真实租户谓词、真实发布/软删/时效
// 过滤。它返回命中文章的标题（归一化形式），因为期望按标题引用。
type RealRAGRetriever struct {
	svc    *service.RAGService
	client *ent.Client
	limit  int

	// queries 记录真实检索实际发起的次数。
	//
	// 这是「候选是否真来自检索」的**机制性**判据：真实检索每次都要查库，
	// 而「返回期望答案」的 stub 一次都不查。此前用数据形状（候选集合 ≠ 期望
	// 集合）来判，在期望全部指向语料里的文章时两者不可区分——真实检索恰好
	// 全召回与抄答案会得到完全相同的候选列表。改用查询计数就不依赖数据形状。
	queries int
}

// QueryCount 返回真实检索发起的查询次数。
func (r *RealRAGRetriever) QueryCount() int { return r.queries }

// NewRealRAGRetriever 在内存库里建好语料并装配真实检索服务。
//
// dsn 用共享缓存的内存库名（不同测试用不同名，避免互相污染）。
func NewRealRAGRetriever(dsn string, corpus *RAGCorpus, logger *zap.SugaredLogger) (*RealRAGRetriever, error) {
	// 用 ent.Open 而非 enttest.Open：enttest 需要 testing.TB（含私有方法，
	// 外部无法实现），而本文件要能被 cmd/ai-eval 这样的非测试代码复用。
	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open eval corpus db: %w", err)
	}
	ctx := context.Background()
	// 建表：与生产同一份 ent schema（内存库，无需迁移账本）。
	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("create eval corpus schema: %w", err)
	}

	// 租户与作者必须先存在（外键）。
	tenantID := 1
	if _, err := client.Tenant.Create().SetName("eval").SetCode("eval").Save(ctx); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	author, err := client.User.Create().
		SetUsername("eval-author").
		SetName("eval-author").
		SetEmail("eval-author@example.com").
		SetPasswordHash("x").
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create author: %w", err)
	}

	for _, a := range corpus.Articles {
		tid := a.TenantID
		if tid == 0 {
			tid = tenantID
		}
		if _, err := client.KnowledgeArticle.Create().
			SetTitle(a.Title).
			SetContent(a.Content).
			SetCategory(a.Category).
			SetTags(a.Tags).
			SetAuthorID(author.ID).
			SetTenantID(tid).
			SetIsPublished(a.IsPublished()).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("create article %q: %w", a.Title, err)
		}
	}

	// 纯关键词检索：不需要向量库与 embedding（那是生产验收的事）。
	cfg := service.RAGConfig{
		UseVector:           false,
		UseKeyword:          true,
		HybridSearch:        false,
		SimilarityThreshold: 0.0,
		MaxResults:          10,
	}
	svc := service.NewRAGService(client, nil, nil, logger, cfg)

	return &RealRAGRetriever{svc: svc, client: client}, nil
}

// Close 释放内存库。
func (r *RealRAGRetriever) Close() {
	if r.client != nil {
		_ = r.client.Close()
	}
}

// RetrievedTitles 返回本次检索命中的**原始标题**（未归一化）。
//
// 供「候选真实性」校验使用：真实检索的产物必然是语料里真实存在的文章标题，
// 而外部注入的 stub 可能返回任意字符串。
func (r *RealRAGRetriever) RetrievedTitles(ctx context.Context, c RAGCase) []string {
	limit := c.TopK
	if limit <= 0 {
		limit = 3
	}
	items, err := r.svc.Ask(ctx, c.TenantID, c.Query, limit)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if title, ok := it["title"].(string); ok && title != "" {
			out = append(out, title)
		}
	}
	return out
}

// Retrieve 返回命中的文章标题（归一化）。
//
// 这是**真实检索**的产出，不是期望答案：候选集合由 SQL 与真实文章决定。
func (r *RealRAGRetriever) Retrieve(ctx context.Context, c RAGCase) []string {
	return r.RetrieveWithLimit(ctx, c)
}

// RetrieveWithLimit 同 Retrieve，但显式指定 topK（0 = 用 case 自带）。
func (r *RealRAGRetriever) RetrieveWithLimit(ctx context.Context, c RAGCase) []string {
	limit := c.TopK
	if limit <= 0 {
		limit = 3
	}
	r.queries++
	items, err := r.svc.Ask(ctx, c.TenantID, c.Query, limit)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		title, _ := it["title"].(string)
		if title == "" {
			continue
		}
		out = append(out, NormalizeRef(title))
	}
	return out
}

// VerifyRetrievedAreRealCorpus 校验检索命中的标题都真实存在于语料。
//
// 这是「候选真实性」对**真实检索器**的正确判据：命中的必须是语料里真实存在
// 的文章标题。
//
// 为什么不用「候选集合 ≠ 期望集合」来判真实检索：top_k=1 且期望只有 1 篇
// 时，真实检索的产物必然就等于期望集合——此时「候选==期望」与「抄答案」
// 在数据上不可区分，用它判真实检索会误杀正确的召回。
//
// 区别在于：**真实检索不会凭空造出语料里没有的标题**，而抄答案的 stub 只会
// 返回期望数组里的东西。两者在 top_k=1 时可能重合，但在 top_k>1 时必然
// 分道扬镳——所以「命中标题都在语料里」是稳定可靠的判据。
//
// 外部注入的 stub 则相反：它的候选可能完全不存在于语料（编造的 ID），或
// 直接等于期望数组（自我实现）。这两种情况由 DetectGoldenDocIDs 判定。
func VerifyRetrievedAreRealCorpus(titles []string, corpus *RAGCorpus) (bool, string) {
	known := corpus.CorpusTitles()
	for _, t := range titles {
		if _, ok := known[NormalizeRef(t)]; !ok {
			return false, fmt.Sprintf("命中标题 %q 不在语料里——候选不是真实检索的产出", t)
		}
	}
	return true, "命中标题均存在于语料"
}

// DefaultCorpusPath 返回默认语料文件路径。
func DefaultCorpusPath() string {
	return filepath.Join("datasets", "corpus", "rag_corpus.jsonl")
}
