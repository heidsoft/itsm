package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

type VectorStore struct{ db *sql.DB }

func NewVectorStore(db *sql.DB) *VectorStore { return &VectorStore{db: db} }

// TestConnection checks if the vectors table exists and is accessible
func (s *VectorStore) TestConnection() error {
	// Simple query to verify the vectors table exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM vectors LIMIT 1").Scan(&count)
	return err
}

// EnsureExtension 确保 pgvector 扩展已安装，并初始化 vectors 表。
// 如果扩展不可用，将返回错误（调用方应降级为关键字搜索）。
func (s *VectorStore) EnsureExtension(ctx context.Context) error {
	// 尝试创建 pgvector 扩展（幂等操作，已存在时不报错）
	_, err := s.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("pgvector 扩展不可用: %w", err)
	}
	// 确保 vectors 表存在
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS vectors (
			id           SERIAL PRIMARY KEY,
			tenant_id    INT NOT NULL,
			object_type  TEXT NOT NULL,
			object_id    INT NOT NULL,
			embedding    vector(1536),
			content      TEXT,
			source       TEXT,
			created_at   TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(tenant_id, object_type, object_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("初始化 vectors 表失败: %w", err)
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
