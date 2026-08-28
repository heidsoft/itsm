package vector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	q "github.com/qdrant/go-client/qdrant"
)

type qdrantStore struct {
	client     *q.Client
	collection string
	vectorSize int
}

func newQdrant(collection string, cfg map[string]interface{}) (VectorStore, error) {
	if collection == "" {
		collection = stringConfig(cfg, "collection", "")
	}
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	addr := stringConfig(cfg, "addr", "localhost:6334")
	host, portText, ok := strings.Cut(addr, ":")
	if !ok {
		return nil, fmt.Errorf("addr must be host:port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, fmt.Errorf("invalid addr: %w", err)
	}
	client, err := q.NewClient(&q.Config{Host: host, Port: port, PoolSize: 1})
	if err != nil {
		return nil, err
	}
	return &qdrantStore{client: client, collection: collection, vectorSize: intConfig(cfg, "vector_size", 0)}, nil
}
func (s *qdrantStore) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if len(req.Vector) == 0 {
		return SearchResponse{}, fmt.Errorf("query vector is required")
	}
	if s.vectorSize > 0 && len(req.Vector) != s.vectorSize {
		return SearchResponse{}, fmt.Errorf("vector dimension: got %d, want %d", len(req.Vector), s.vectorSize)
	}
	filter, err := qdrantFilter(req.Filter)
	if err != nil {
		return SearchResponse{}, err
	}
	resp, err := s.client.GetPointsClient().Search(ctx, &q.SearchPoints{CollectionName: s.collection, Vector: req.Vector, Filter: filter, Limit: uint64(topK(req.TopK, DefaultTopK)), WithPayload: q.NewWithPayload(true)})
	if err != nil {
		return SearchResponse{}, fmt.Errorf("qdrant search: %w", err)
	}
	out := SearchResponse{Backend: "qdrant", Results: []SearchResult{}}
	for _, p := range resp.Result {
		meta := map[string]interface{}{}
		for k, v := range p.Payload {
			meta[k] = qdrantValue(v)
		}
		content, _ := meta["content"].(string)
		delete(meta, "content")
		out.Results = append(out.Results, SearchResult{ID: qdrantID(p.Id), Content: content, Score: float64(p.Score), Metadata: meta})
	}
	return out, nil
}
func (s *qdrantStore) Insert(ctx context.Context, req InsertRequest) error {
	points := make([]*q.PointStruct, 0, len(req.Chunks))
	for _, c := range req.Chunks {
		if s.vectorSize > 0 && len(c.Vector) != s.vectorSize {
			return fmt.Errorf("chunk %s vector dimension: got %d, want %d", c.ID, len(c.Vector), s.vectorSize)
		}
		payload := make(map[string]interface{}, len(c.Metadata)+1)
		for k, v := range c.Metadata {
			payload[k] = v
		}
		payload["content"] = c.Content
		values, err := q.TryValueMap(payload)
		if err != nil {
			return fmt.Errorf("encode chunk %s metadata: %w", c.ID, err)
		}
		points = append(points, &q.PointStruct{Id: q.NewID(c.ID), Vectors: q.NewVectors(c.Vector...), Payload: values})
	}
	wait := true
	_, err := s.client.Upsert(ctx, &q.UpsertPoints{CollectionName: s.collection, Wait: &wait, Points: points})
	if err != nil {
		return fmt.Errorf("qdrant upsert: %w", err)
	}
	return nil
}
func (s *qdrantStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	p := make([]*q.PointId, len(ids))
	for i, id := range ids {
		p[i] = q.NewID(id)
	}
	wait := true
	_, err := s.client.Delete(ctx, &q.DeletePoints{CollectionName: s.collection, Wait: &wait, Points: q.NewPointsSelector(p...)})
	return err
}
func (s *qdrantStore) Ping(ctx context.Context) error {
	_, err := s.client.GetQdrantClient().HealthCheck(ctx, &q.HealthCheckRequest{})
	return err
}
func (s *qdrantStore) Close() error { return s.client.Close() }
func qdrantFilter(values map[string]interface{}) (*q.Filter, error) {
	if len(values) == 0 {
		return nil, nil
	}
	f := &q.Filter{}
	for k, v := range values {
		switch x := v.(type) {
		case string:
			f.Must = append(f.Must, q.NewMatchKeyword(k, x))
		case bool:
			f.Must = append(f.Must, q.NewMatchBool(k, x))
		case int:
			f.Must = append(f.Must, q.NewMatchInt(k, int64(x)))
		case int64:
			f.Must = append(f.Must, q.NewMatchInt(k, x))
		case float64:
			f.Must = append(f.Must, q.NewMatchInt(k, int64(x)))
		default:
			return nil, fmt.Errorf("unsupported qdrant filter %s type %T", k, v)
		}
	}
	return f, nil
}
func qdrantID(id *q.PointId) string {
	if id == nil {
		return ""
	}
	if n := id.GetNum(); n != 0 {
		return strconv.FormatUint(n, 10)
	}
	return id.GetUuid()
}
func qdrantValue(v *q.Value) interface{} {
	if v == nil {
		return nil
	}
	switch x := v.Kind.(type) {
	case *q.Value_StringValue:
		return x.StringValue
	case *q.Value_IntegerValue:
		return x.IntegerValue
	case *q.Value_DoubleValue:
		return x.DoubleValue
	case *q.Value_BoolValue:
		return x.BoolValue
	default:
		return nil
	}
}
func init() { Register("qdrant", newQdrant) }
