package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"itsm-backend/database"
)

type VectorStore struct{ db *sql.DB }

// VectorSearchResult is fully materialized before its tenant-scoped SQL
// connection is released. It prevents callers from holding *sql.Rows beyond
// the RLS session lifetime.
type VectorSearchResult struct {
	ObjectType string
	ObjectID   int
	Content    string
	Source     string
	Distance   float64
}

func NewVectorStore(db *sql.DB) *VectorStore { return &VectorStore{db: db} }

// TestConnection checks if the vectors table exists and is accessible
func (s *VectorStore) TestConnection() error {
	// Simple query to verify the vectors table exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM vectors LIMIT 1").Scan(&count)
	return err
}

// EnsureExtension verifies that the bootstrap migration provisioned pgvector
// storage. It deliberately performs no DDL: long-running application
// instances must not mutate schema or race each other at startup.
func (s *VectorStore) EnsureExtension(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("vector store database is not configured")
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector')`).Scan(&exists); err != nil {
		return fmt.Errorf("check pgvector extension: %w", err)
	}
	if !exists {
		return fmt.Errorf("pgvector extension is not installed; run bootstrap migration 020")
	}
	if err := s.TestConnection(); err != nil {
		return fmt.Errorf("vectors storage is not ready; run bootstrap migration 020: %w", err)
	}
	return nil
}

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

// CountByTenant 返回指定租户的向量数。
// 调用方必须从已认证上下文中获取 tenantID,不得使用跨租户范围。
func (s *VectorStore) CountByTenant(ctx context.Context, tenantID int) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vectors WHERE tenant_id = $1`, tenantID).Scan(&count)
	return count, err
}

// Delete removes a vector entry by tenant + object identity.
// Used by RAGService.RemoveArticle so soft-deleted / unpublished articles are
// physically removed from the vectors table instead of lingering as stale
// hits that only enrichment-time filtering can hide.
func (s *VectorStore) Delete(ctx context.Context, tenantID int, objectType string, objectID int) error {
	_, err := s.db.ExecContext(ctx, `
        DELETE FROM vectors WHERE tenant_id = $1 AND object_type = $2 AND object_id = $3
    `, tenantID, objectType, objectID)
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

// SearchTopKByTypeResults is the tenant-safe search API for new callers.
// It scans all rows while the scoped RLS connection is checked out.
func (s *VectorStore) SearchTopKByTypeResults(ctx context.Context, tenantID int, objectType string, query []float32, k int) ([]VectorSearchResult, error) {
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
	const statement = `
        SELECT object_type, object_id, content, source, (embedding <#> $1::vector) AS distance
        FROM vectors WHERE tenant_id = $2 AND object_type = $3
        ORDER BY embedding <#> $1::vector
        LIMIT $4;
    `
	return database.WithTenantSQL(ctx, s.db, tenantID, func(q database.SQLExecutor) ([]VectorSearchResult, error) {
		rows, err := q.QueryContext(ctx, statement, string(values), tenantID, objectType, k)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := make([]VectorSearchResult, 0, k)
		for rows.Next() {
			var result VectorSearchResult
			var content, source sql.NullString
			if err := rows.Scan(&result.ObjectType, &result.ObjectID, &content, &source, &result.Distance); err != nil {
				return nil, err
			}
			result.Content, result.Source = content.String, source.String
			results = append(results, result)
		}
		return results, rows.Err()
	})
}

func fmtFloat(f float32) string {
	// compact but precise enough for embeddings
	return strconv.FormatFloat(float64(f), 'f', 6, 64)
}
