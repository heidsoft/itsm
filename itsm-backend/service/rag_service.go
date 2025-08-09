package service

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"itsm-backend/ent"
	ka "itsm-backend/ent/knowledgearticle"
)

// RAGService provides minimal retrieval augmented generation over Knowledge Base
type RAGService struct {
	client   *ent.Client
	vectors  *VectorStore
	embedder Embedder
}

func NewRAGService(client *ent.Client, vectors *VectorStore, embedder Embedder) *RAGService {
	return &RAGService{client: client, vectors: vectors, embedder: embedder}
}

// Ask performs a simplified retrieval over knowledge articles: naive LIKE search and returns snippets
func (r *RAGService) Ask(ctx context.Context, tenantID int, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}
	results := []map[string]any{}
	seen := map[string]struct{}{}
	// vector search if available (kb only by default)
	if r.vectors != nil && r.embedder != nil {
		if vec, err := r.embedder.Embed(query); err == nil {
			if rows, err := r.vectors.SearchTopKByType(ctx, tenantID, "kb", vec, limit); err == nil {
				defer rows.Close()
				for rows.Next() {
					var objType string
					var objID int
					var content, source sql.NullString
					var distance float64
					if err := rows.Scan(&objType, &objID, &content, &source, &distance); err == nil {
						key := objType + ":" + strconv.Itoa(objID)
						if _, ok := seen[key]; ok {
							continue
						}
						seen[key] = struct{}{}
						item := map[string]any{
							"object_type": objType,
							"id":          objID,
							"snippet":     snippet(content.String, 200),
							"source":      source.String,
							"score":       1.0 - distance,
						}
						// enrich KB with title/category
						if objType == "kb" {
							if a, err := r.client.KnowledgeArticle.Get(ctx, objID); err == nil {
								item["title"] = a.Title
								item["category"] = a.Category
							}
						}
						results = append(results, item)
						if len(results) >= limit {
							return results, nil
						}
					}
				}
			}
		}
	}
	// keyword fallback/augment
	q := r.client.KnowledgeArticle.Query().Where(ka.TenantIDEQ(tenantID))
	if qq := strings.TrimSpace(query); qq != "" {
		q = q.Where(ka.Or(ka.TitleContainsFold(qq), ka.ContentContainsFold(qq)))
	}
	articles, err := q.Limit(limit).All(ctx)
	if err != nil {
		return results, nil
	}
	for _, a := range articles {
		key := "kb:" + strconv.Itoa(a.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		results = append(results, map[string]any{
			"object_type": "kb",
			"id":          a.ID,
			"title":       a.Title,
			"category":    a.Category,
			"snippet":     snippet(a.Content, 160),
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func snippet(s string, n int) string {
	if n <= 0 {
		n = 160
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
