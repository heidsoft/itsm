package service

import (
	"context"
	"fmt"
	"strconv"
)

// SimilarIncidents returns topK incidents similar to the query text
func SimilarIncidents(ctx context.Context, vectors *VectorStore, embedder Embedder, tenantID int, query string, k int) ([]map[string]any, error) {
	if vectors == nil || embedder == nil {
		return nil, fmt.Errorf("Vector search service is not configured: VectorStore or Embedder is empty")
	}
	vec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("Text vectorization failed: %w", err)
	}
	vectorResults, err := vectors.SearchTopKByTypeResults(ctx, tenantID, "incident", vec, k)
	if err != nil {
		return nil, fmt.Errorf("Vector search failed: %w", err)
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
