package service

import (
	"context"
	"strconv"
)

// SimilarIncidents returns topK incidents similar to the query text
func SimilarIncidents(ctx context.Context, vectors *VectorStore, embedder Embedder, tenantID int, query string, k int) ([]map[string]any, error) {
	if vectors == nil || embedder == nil {
		return []map[string]any{}, nil
	}
	vec, err := embedder.Embed(query)
	if err != nil {
		// Embedding failed
		return []map[string]any{}, nil
	}
	vectorResults, err := vectors.SearchTopKByTypeResults(ctx, tenantID, "incident", vec, k)
	if err != nil {
		// 如果向量搜索失败（例如pgvector扩展未安装），降级为空结果
		return []map[string]any{}, nil
	}
	out := []map[string]any{}
	for _, result := range vectorResults {
		out = append(out, map[string]any{
			"id":        result.ObjectID,
			"snippet":   result.Content,
			"ref":       result.Source,
			"score":     1.0 - result.Distance,
			"object":    result.ObjectType,
			"object_id": strconv.Itoa(result.ObjectID),
		})
	}
	return out, nil
}
