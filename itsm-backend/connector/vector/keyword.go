package vector

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type keywordStore struct {
	mu          sync.RWMutex
	chunks      map[string]ChunkInput
	defaultTopK int
}

func newKeyword(_ string, cfg map[string]interface{}) (VectorStore, error) {
	return &keywordStore{chunks: make(map[string]ChunkInput), defaultTopK: intConfig(cfg, "top_k", DefaultTopK)}, nil
}
func (s *keywordStore) Insert(ctx context.Context, req InsertRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range req.Chunks {
		if c.ID != "" {
			s.chunks[c.ID] = c
		}
	}
	return nil
}
func (s *keywordStore) Delete(ctx context.Context, ids []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.chunks, id)
	}
	return nil
}
func (s *keywordStore) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if err := ctx.Err(); err != nil {
		return SearchResponse{}, err
	}
	queryTerms := tokenize(req.Query)
	response := SearchResponse{Backend: "keyword", Results: []SearchResult{}}
	if len(queryTerms) == 0 {
		return response, nil
	}
	s.mu.RLock()
	docs := make([]ChunkInput, 0, len(s.chunks))
	for _, c := range s.chunks {
		if matches(c.Metadata, req.Filter) {
			docs = append(docs, c)
		}
	}
	s.mu.RUnlock()
	df := map[string]int{}
	docTerms := make([]map[string]int, len(docs))
	for i, d := range docs {
		docTerms[i] = frequencies(tokenize(d.Content))
		for term := range docTerms[i] {
			df[term]++
		}
	}
	for i, d := range docs {
		score := 0.0
		for _, term := range queryTerms {
			tf := docTerms[i][term]
			if tf > 0 {
				score += (1+math.Log(float64(tf)))*math.Log(1+float64(len(docs))/float64(1+df[term])) + 0.25
			}
		}
		if score > 0 {
			response.Results = append(response.Results, SearchResult{ID: d.ID, Content: d.Content, Score: score, Metadata: d.Metadata})
		}
	}
	sort.SliceStable(response.Results, func(i, j int) bool {
		if response.Results[i].Score == response.Results[j].Score {
			return response.Results[i].ID < response.Results[j].ID
		}
		return response.Results[i].Score > response.Results[j].Score
	})
	limit := topK(req.TopK, s.defaultTopK)
	if len(response.Results) > limit {
		response.Results = response.Results[:limit]
	}
	return response, nil
}
func (s *keywordStore) Ping(context.Context) error { return nil }
func (s *keywordStore) Close() error               { return nil }
func tokenize(v string) []string {
	return strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}
func frequencies(terms []string) map[string]int {
	m := map[string]int{}
	for _, t := range terms {
		m[t]++
	}
	return m
}
func matches(metadata, filter map[string]interface{}) bool {
	for k, v := range filter {
		if metadata[k] != v {
			return false
		}
	}
	return true
}

func init() { Register("keyword", newKeyword) }
