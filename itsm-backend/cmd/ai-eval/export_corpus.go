package main

// export_corpus.go 实现 -export-corpus：从生产库导出指定租户的已发布知识
// 文章为评测语料格式（rag_corpus.jsonl 同形），作为验收语料准备的起点。
//
// 它只导出**语料**（库里有什么），不产出期望引用——「该检索到什么」必须由
// 知识库维护者人工标注。自动生成的期望（比如按检索结果回填）会把「检索器
// 的现状」当成「正确答案」，验收就失去意义了。见
// docs/operations/ai-eval-corpus-guide.md。

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"itsm-backend/ai/eval"
	"itsm-backend/config"
	"itsm-backend/database"

	_ "github.com/lib/pq"
)

// exportCorpus 导出语料并返回导出条数。
func exportCorpus(tenantID int, outPath string, limit int) (int, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return 0, fmt.Errorf("读配置: %w", err)
	}
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		return 0, fmt.Errorf("连数据库: %w", err)
	}
	defer db.Close()

	// 只读连接，直接用 ent 生成的查询？不行——这里没有 ent client（InitDB 只给
	// *sql.DB），但导出只需要四列。用原生 SQL 保持命令零额外装配：生产库的
	// 连接配置（含 RLS 角色口径）由 database.InitDB 统一处理。
	rows, err := db.Query(`
		SELECT title, content, category, tags, tenant_id
		FROM knowledge_articles
		WHERE tenant_id = $1
		  AND is_published = true
		  AND deleted_at IS NULL
		ORDER BY id
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return 0, fmt.Errorf("查询知识文章: %w", err)
	}
	defer rows.Close()

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, fmt.Errorf("建输出目录: %w", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return 0, fmt.Errorf("建语料文件: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	count := 0
	for rows.Next() {
		var a eval.RAGCorpusArticle
		var tenant int
		if err := rows.Scan(&a.Title, &a.Content, &a.Category, &a.Tags, &tenant); err != nil {
			return count, fmt.Errorf("读行: %w", err)
		}
		a.TenantID = tenant
		published := true
		a.Published = &published
		if err := enc.Encode(a); err != nil {
			return count, fmt.Errorf("写行: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, err
	}
	return count, nil
}

// runExport 是 -export-corpus 模式的入口（main 解析到该 flag 后转到这里）。
func runExport(args []string) error {
	fs := flag.NewFlagSet("export-corpus", flag.ContinueOnError)
	tenant := fs.Int("tenant", 1, "导出哪个租户的文章")
	out := fs.String("out-corpus", "", "语料输出路径（默认 <工作目录>/rag_corpus.jsonl）")
	limit := fs.Int("limit", 500, "最多导出多少篇")
	if err := fs.Parse(args); err != nil {
		return err
	}
	outPath := *out
	if outPath == "" {
		outPath = "rag_corpus.jsonl"
	}
	n, err := exportCorpus(*tenant, outPath, *limit)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("租户 %d 没有可导出的已发布文章（语料不能为空，装载会拒绝）", *tenant)
	}
	fmt.Printf("已导出 %d 篇到 %s\n", n, outPath)
	fmt.Println("下一步：人工标注 rag.jsonl 的 expected_refs（见 docs/operations/ai-eval-corpus-guide.md）")
	return nil
}
