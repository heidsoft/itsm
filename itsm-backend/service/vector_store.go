package service

import (
	"context"
	"database/sql"
	"strconv"
)

type VectorStore struct{ db *sql.DB }

func NewVectorStore(db *sql.DB) *VectorStore { return &VectorStore{db: db} }

func (s *VectorStore) Upsert(ctx context.Context, tenantID int, objectType string, objectID int, embedding []float32, content string, source string) error {
	// pgx prefers []float32 -> vector; with database/sql we build string literal
	// Convert to SQL literal: '['1,2,3']' style for pgvector (space-separated)
	values := make([]byte, 0, len(embedding)*6)
	values = append(values, '[')
	for i, v := range embedding {
		if i > 0 {
			values = append(values, ',')
		}
		values = append(values, []byte(fmtFloat(v))...)
	}
	values = append(values, ']')
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO vectors(tenant_id, object_type, object_id, embedding, content, source)
        VALUES ($1,$2,$3,$4::vector,$5,$6)
        ON CONFLICT (tenant_id, object_type, object_id) DO UPDATE
        SET embedding = EXCLUDED.embedding, content = EXCLUDED.content, source = EXCLUDED.source;
    `, tenantID, objectType, objectID, string(values), content, source)
	return err
}

func (s *VectorStore) SearchTopK(ctx context.Context, tenantID int, query []float32, k int) (*sql.Rows, error) {
	if k <= 0 {
		k = 5
	}
	values := make([]byte, 0, len(query)*6)
	values = append(values, '[')
	for i, v := range query {
		if i > 0 {
			values = append(values, ',')
		}
		values = append(values, []byte(fmtFloat(v))...)
	}
	values = append(values, ']')
	return s.db.QueryContext(ctx, `
        SELECT object_type, object_id, content, source, (embedding <#> $1::vector) AS distance
        FROM vectors WHERE tenant_id = $2
        ORDER BY embedding <#> $1::vector
        LIMIT $3;
    `, string(values), tenantID, k)
}

// SearchTopKByType allows restricting vector search to a specific object_type (e.g., 'kb' or 'incident')
func (s *VectorStore) SearchTopKByType(ctx context.Context, tenantID int, objectType string, query []float32, k int) (*sql.Rows, error) {
	if k <= 0 {
		k = 5
	}
	values := make([]byte, 0, len(query)*6)
	values = append(values, '[')
	for i, v := range query {
		if i > 0 {
			values = append(values, ',')
		}
		values = append(values, []byte(fmtFloat(v))...)
	}
	values = append(values, ']')
	return s.db.QueryContext(ctx, `
        SELECT object_type, object_id, content, source, (embedding <#> $1::vector) AS distance
        FROM vectors WHERE tenant_id = $2 AND object_type = $3
        ORDER BY embedding <#> $1::vector
        LIMIT $4;
    `, string(values), tenantID, objectType, k)
}

func fmtFloat(f float32) string {
	// compact but precise enough for embeddings
	return strconv.FormatFloat(float64(f), 'f', 6, 64)
}
