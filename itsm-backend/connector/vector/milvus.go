package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type milvusStore struct {
	client     client.Client
	collection string
	dimension  int
	metric     entity.MetricType
}

func newMilvus(collection string, cfg map[string]interface{}) (VectorStore, error) {
	if collection == "" {
		collection = stringConfig(cfg, "collection", "")
	}
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	metric := entity.MetricType(strings.ToUpper(stringConfig(cfg, "metric_type", "COSINE")))
	if metric != entity.L2 && metric != entity.IP && metric != entity.COSINE {
		return nil, fmt.Errorf("unsupported metric_type %q", metric)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	defer cancel()
	c, err := client.NewClient(ctx, client.Config{Address: stringConfig(cfg, "addr", "localhost:19530")})
	if err != nil {
		return nil, err
	}
	return &milvusStore{client: c, collection: collection, dimension: intConfig(cfg, "dimension", 0), metric: metric}, nil
}
func (s *milvusStore) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if len(req.Vector) == 0 {
		return SearchResponse{}, fmt.Errorf("query vector is required")
	}
	if s.dimension > 0 && len(req.Vector) != s.dimension {
		return SearchResponse{}, fmt.Errorf("vector dimension: got %d, want %d", len(req.Vector), s.dimension)
	}
	expr, err := milvusExpr(req.Filter)
	if err != nil {
		return SearchResponse{}, err
	}
	sp, _ := entity.NewIndexFlatSearchParam()
	found, err := s.client.Search(ctx, s.collection, nil, expr, []string{"content", "metadata"}, []entity.Vector{entity.FloatVector(req.Vector)}, "embedding", s.metric, topK(req.TopK, DefaultTopK), sp)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("milvus search: %w", err)
	}
	out := SearchResponse{Backend: "milvus", Results: []SearchResult{}}
	if len(found) == 0 {
		return out, nil
	}
	res := found[0]
	contents, _ := res.Fields.GetColumn("content").(*entity.ColumnVarChar)
	metadata, _ := res.Fields.GetColumn("metadata").(*entity.ColumnVarChar)
	for i := 0; i < res.ResultCount; i++ {
		idv, e := res.IDs.Get(i)
		if e != nil {
			return SearchResponse{}, e
		}
		content := ""
		if contents != nil {
			content, _ = contents.GetAsString(i)
		}
		meta := map[string]interface{}{}
		if metadata != nil {
			raw, _ := metadata.GetAsString(i)
			_ = json.Unmarshal([]byte(raw), &meta)
		}
		out.Results = append(out.Results, SearchResult{ID: fmt.Sprint(idv), Content: content, Score: float64(res.Scores[i]), Metadata: meta})
	}
	return out, nil
}
func (s *milvusStore) Insert(ctx context.Context, req InsertRequest) error {
	ids := make([]string, len(req.Chunks))
	contents := make([]string, len(req.Chunks))
	metas := make([]string, len(req.Chunks))
	vectors := make([][]float32, len(req.Chunks))
	for i, c := range req.Chunks {
		if s.dimension > 0 && len(c.Vector) != s.dimension {
			return fmt.Errorf("chunk %s vector dimension: got %d, want %d", c.ID, len(c.Vector), s.dimension)
		}
		raw, err := json.Marshal(c.Metadata)
		if err != nil {
			return err
		}
		ids[i], contents[i], metas[i], vectors[i] = c.ID, c.Content, string(raw), c.Vector
	}
	_, err := s.client.Upsert(ctx, s.collection, "", entity.NewColumnVarChar("id", ids), entity.NewColumnVarChar("content", contents), entity.NewColumnVarChar("metadata", metas), entity.NewColumnFloatVector("embedding", s.dimension, vectors))
	return err
}
func (s *milvusStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = strconv.Quote(id)
	}
	return s.client.Delete(ctx, s.collection, "", "id in ["+strings.Join(quoted, ",")+"]")
}
func (s *milvusStore) Ping(ctx context.Context) error {
	state, err := s.client.CheckHealth(ctx)
	if err != nil {
		return err
	}
	if !state.IsHealthy {
		return fmt.Errorf("milvus is unhealthy")
	}
	return nil
}
func (s *milvusStore) Close() error { return s.client.Close() }
func milvusExpr(f map[string]interface{}) (string, error) {
	parts := make([]string, 0, len(f))
	for k, v := range f {
		if !identifierRE.MatchString(k) {
			return "", fmt.Errorf("invalid filter field %q", k)
		}
		switch x := v.(type) {
		case string:
			parts = append(parts, k+" == "+strconv.Quote(x))
		case bool:
			parts = append(parts, fmt.Sprintf("%s == %t", k, x))
		case int, int64, float64:
			parts = append(parts, fmt.Sprintf("%s == %v", k, x))
		default:
			return "", fmt.Errorf("unsupported milvus filter %s type %T", k, v)
		}
	}
	return strings.Join(parts, " && "), nil
}
func init() { Register("milvus", newMilvus) }
