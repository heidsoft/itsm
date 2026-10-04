package service

import (
	"context"
	"fmt"
	"strconv"
)

// SimilarIncidents returns topK incidents similar to the query text
func SimilarIncidents(ctx context.Context, vectors *VectorStore, embedder Embedder, tenantID int, query string, k int) ([]map[string]any, error) {
	if vectors == nil || embedder == nil {
		return nil, fmt.Errorf("向量搜索服务未配置：VectorStore 或 Embedder 为空")
	}
	vec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("文本向量化失败: %w", err)
	}
	vectorResults, err := vectors.SearchTopKByTypeResults(ctx, tenantID, "incident", vec, k)
	if err != nil {
		return nil, fmt.Errorf("向量搜索失败: %w", err)
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
