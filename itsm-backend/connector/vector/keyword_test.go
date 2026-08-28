package vector

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeywordSearchScoring(t *testing.T) {
	s, err := New("keyword", "", nil)
	require.NoError(t, err)
	require.NoError(t, s.Insert(context.Background(), InsertRequest{Chunks: []ChunkInput{{ID: "1", Content: "database database outage recovery"}, {ID: "2", Content: "printer paper jam"}}}))
	out, err := s.Search(context.Background(), SearchRequest{Query: "database outage", TopK: 2})
	require.NoError(t, err)
	require.Equal(t, "keyword", out.Backend)
	require.Len(t, out.Results, 1)
	require.Equal(t, "1", out.Results[0].ID)
	require.Positive(t, out.Results[0].Score)
}
func TestKeywordInsertSearchRoundTrip(t *testing.T) {
	s, _ := New("keyword", "", map[string]interface{}{"top_k": 1})
	require.NoError(t, s.Insert(context.Background(), InsertRequest{Chunks: []ChunkInput{{ID: "a", Content: "VPN connection guide", Metadata: map[string]interface{}{"tenant_id": 1}}, {ID: "b", Content: "VPN other tenant", Metadata: map[string]interface{}{"tenant_id": 2}}}}))
	out, err := s.Search(context.Background(), SearchRequest{Query: "VPN", Filter: map[string]interface{}{"tenant_id": 1}})
	require.NoError(t, err)
	require.Len(t, out.Results, 1)
	require.Equal(t, "a", out.Results[0].ID)
}
func TestKeywordDelete(t *testing.T) {
	s, _ := New("keyword", "", nil)
	require.NoError(t, s.Insert(context.Background(), InsertRequest{Chunks: []ChunkInput{{ID: "gone", Content: "delete me"}}}))
	require.NoError(t, s.Delete(context.Background(), []string{"gone"}))
	out, err := s.Search(context.Background(), SearchRequest{Query: "delete"})
	require.NoError(t, err)
	require.Empty(t, out.Results)
}
