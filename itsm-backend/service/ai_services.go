package service

import (
	"context"
	"database/sql"
	"strconv"
)

// SimilarIncidents returns topK incidents similar to the query text
func SimilarIncidents(ctx context.Context, vectors *VectorStore, embedder Embedder, tenantID int, query string, k int) ([]map[string]any, error) {
	if vectors == nil || embedder == nil {
		return []map[string]any{}, nil
	}
	vec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}
	rows, err := vectors.SearchTopKByType(ctx, tenantID, "incident", vec, k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var objType string
		var objID int
		var content, source sql.NullString
		var distance float64
		if err := rows.Scan(&objType, &objID, &content, &source, &distance); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id":        objID,
			"snippet":   content.String,
			"ref":       source.String,
			"score":     1.0 - distance,
			"object":    objType,
			"object_id": strconv.Itoa(objID),
		})
	}
	return out, nil
}
