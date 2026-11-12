package service

import (
	"context"
	"testing"

	connectorVector "itsm-backend/connector/vector"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type recordingVectorStore struct {
	inserts []connectorVector.InsertRequest
	deletes [][]string
}

func (s *recordingVectorStore) Search(context.Context, connectorVector.SearchRequest) (connectorVector.SearchResponse, error) {
	return connectorVector.SearchResponse{}, nil
}

func (s *recordingVectorStore) Insert(_ context.Context, req connectorVector.InsertRequest) error {
	s.inserts = append(s.inserts, req)
	return nil
}

func (s *recordingVectorStore) Delete(_ context.Context, ids []string) error {
	s.deletes = append(s.deletes, ids)
	return nil
}

func (s *recordingVectorStore) Ping(context.Context) error { return nil }
func (s *recordingVectorStore) Close() error               { return nil }

func TestVectorIndexReady(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	t.Run("nil service", func(t *testing.T) {
		var rag *RAGService
		assert.False(t, rag.HasVectorBackend())
		assert.False(t, rag.VectorIndexReady())
	})

	t.Run("no vector backend", func(t *testing.T) {
		rag := NewRAGService(nil, nil, &MockEmbedder{}, logger, RAGConfig{UseVector: true})
		assert.False(t, rag.HasVectorBackend())
		assert.False(t, rag.VectorIndexReady())
	})

	t.Run("vector search disabled", func(t *testing.T) {
		rag := NewRAGService(nil, nil, &MockEmbedder{}, logger, RAGConfig{UseVector: true})
		rag.SetVectorStore(&recordingVectorStore{})
		rag.useVector = false
		assert.False(t, rag.VectorIndexReady())
	})

	t.Run("embedder not configured", func(t *testing.T) {
		rag := NewRAGService(nil, nil, &OpenAIEmbedder{}, logger, RAGConfig{UseVector: true})
		rag.SetVectorStore(&recordingVectorStore{})
		assert.True(t, rag.HasVectorBackend())
		assert.False(t, rag.VectorIndexReady())
	})

	t.Run("embedder without readiness contract is assumed usable", func(t *testing.T) {
		rag := NewRAGService(nil, nil, &MockEmbedder{}, logger, RAGConfig{UseVector: true})
		rag.SetVectorStore(&recordingVectorStore{})
		assert.True(t, rag.VectorIndexReady())
	})

	t.Run("configured openai embedder", func(t *testing.T) {
		rag := NewRAGService(nil, nil, NewOpenAIEmbedderWithConfig("test-key", "http://127.0.0.1:1", ""), logger, RAGConfig{UseVector: true})
		rag.SetVectorStore(&recordingVectorStore{})
		assert.True(t, rag.VectorIndexReady())
	})
}

func TestIndexArticle_SkipsWhenEmbeddingNotConfigured(t *testing.T) {
	store := &recordingVectorStore{}
	rag := NewRAGService(nil, nil, &OpenAIEmbedder{}, zaptest.NewLogger(t).Sugar(), RAGConfig{UseVector: true})
	rag.SetVectorStore(store)

	require.NoError(t, rag.IndexArticle(context.Background(), 7, 42, "runbook", "restart safely"))
	assert.Empty(t, store.inserts, "未配置的 embedding 不应触发索引写入")
}

func TestIndexArticle_WritesWhenEmbeddingReady(t *testing.T) {
	store := &recordingVectorStore{}
	rag := NewRAGService(nil, nil, &MockEmbedder{}, zaptest.NewLogger(t).Sugar(), RAGConfig{UseVector: true})
	rag.SetVectorStore(store)

	require.NoError(t, rag.IndexArticle(context.Background(), 7, 42, "runbook", "restart safely"))
	require.Len(t, store.inserts, 1)
	require.Len(t, store.inserts[0].Chunks, 1)
	assert.Equal(t, "tenant:7:kb:42", store.inserts[0].Chunks[0].ID)
	assert.NotEmpty(t, store.inserts[0].Chunks[0].Vector)
}
