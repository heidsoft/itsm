package vector

import "context"

const DefaultTopK = 10

// VectorStore stores already-generated embeddings. Embedding generation belongs
// to the caller so backends remain provider agnostic.
type VectorStore interface {
	Search(context.Context, SearchRequest) (SearchResponse, error)
	Insert(context.Context, InsertRequest) error
	Delete(context.Context, []string) error
	Ping(context.Context) error
	Close() error
}

type SearchRequest struct {
	Vector []float32
	Query  string
	TopK   int
	Filter map[string]interface{}
}

type SearchResponse struct {
	Results []SearchResult
	Backend string
}

type SearchResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]interface{}
}

type InsertRequest struct{ Chunks []ChunkInput }

type ChunkInput struct {
	ID       string
	Content  string
	Vector   []float32
	Metadata map[string]interface{}
}
