package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newVectorsTestDB 建立 sqlite 内存库的 vectors 简化表。
// 只含 Delete 所需列（tenant_id/object_type/object_id），pgvector 列在单测中不可用。
func newVectorsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INT NOT NULL,
			object_type TEXT NOT NULL,
			object_id INT NOT NULL,
			content TEXT,
			UNIQUE(tenant_id, object_type, object_id)
		)
	`)
	require.NoError(t, err)
	return db
}

func seedVector(t *testing.T, db *sql.DB, tenantID int, objectType string, objectID int, content string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO vectors (tenant_id, object_type, object_id, content)
		VALUES (?, ?, ?, ?)
	`, tenantID, objectType, objectID, content)
	require.NoError(t, err)
}

func countVectors(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&n))
	return n
}

func TestVectorStore_Delete(t *testing.T) {
	db := newVectorsTestDB(t)
	store := NewVectorStore(db)
	ctx := context.Background()

	seedVector(t, db, 1, "kb", 42, "文章A")
	seedVector(t, db, 1, "kb", 43, "文章B")
	seedVector(t, db, 2, "kb", 42, "租户2文章")

	// 删除目标租户/类型/ID 的行，不影响其他行
	require.NoError(t, store.Delete(ctx, 1, "kb", 42))
	assert.Equal(t, 2, countVectors(t, db))

	// 幂等：再次删除已不存在的条目不报错
	require.NoError(t, store.Delete(ctx, 1, "kb", 42))
	assert.Equal(t, 2, countVectors(t, db))

	// 租户隔离：租户2 的同一 object_id 必须保留
	require.NoError(t, store.Delete(ctx, 1, "kb", 43))
	assert.Equal(t, 1, countVectors(t, db))
}

func TestRAGService_RemoveArticle(t *testing.T) {
	db := newVectorsTestDB(t)
	rag := NewRAGService(nil, NewVectorStore(db), &MockEmbedder{}, zaptest.NewLogger(t).Sugar(), RAGConfig{
		UseVector:    true,
		UseKeyword:   false,
		HybridSearch: false,
		MaxResults:   10,
	})
	ctx := context.Background()

	seedVector(t, db, 1, "kb", 7, "残留向量")

	// 真实删除：软删除/取消发布文章后 vectors 表不再残留
	require.NoError(t, rag.RemoveArticle(ctx, 1, 7))
	assert.Equal(t, 0, countVectors(t, db))

	// 幂等：再次删除不报错
	require.NoError(t, rag.RemoveArticle(ctx, 1, 7))
}

func TestRAGService_RemoveArticle_DisabledOrNilVectors(t *testing.T) {
	ctx := context.Background()

	// vectors 为 nil（如关键字降级模式）→ 静默成功
	ragNoVectors := NewRAGService(nil, nil, nil, zaptest.NewLogger(t).Sugar(), RAGConfig{
		UseVector:  true,
		UseKeyword: true,
	})
	require.NoError(t, ragNoVectors.RemoveArticle(ctx, 1, 7))

	// useVector=false → 静默成功
	db := newVectorsTestDB(t)
	ragDisabled := NewRAGService(nil, NewVectorStore(db), nil, zaptest.NewLogger(t).Sugar(), RAGConfig{
		UseVector:  false,
		UseKeyword: true,
	})
	require.NoError(t, ragDisabled.RemoveArticle(ctx, 1, 7))
}
