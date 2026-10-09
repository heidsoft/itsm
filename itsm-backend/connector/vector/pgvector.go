package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type pgVectorStore struct {
	db            *sql.DB
	table, metric string
	dimension     int
}

func newPGVector(collection string, cfg map[string]interface{}) (VectorStore, error) {
	dsn := stringConfig(cfg, "dsn", "")
	if dsn == "" {
		return nil, fmt.Errorf("dsn is required")
	}
	table := stringConfig(cfg, "table_name", collection)
	if table == "" {
		table = "vectors"
	}
	if err := validIdentifier(table); err != nil {
		return nil, err
	}
	metric := strings.ToLower(stringConfig(cfg, "metric", "cosine"))
	if metric != "cosine" && metric != "l2" && metric != "ip" {
		return nil, fmt.Errorf("unsupported metric %q", metric)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &pgVectorStore{db: db, table: table, metric: metric, dimension: intConfig(cfg, "dimension", 0)}, nil
}

func (s *pgVectorStore) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if len(req.Vector) == 0 {
		return SearchResponse{}, fmt.Errorf("query vector is required")
	}
	if s.dimension > 0 && len(req.Vector) != s.dimension {
		return SearchResponse{}, fmt.Errorf("vector dimension: got %d, want %d", len(req.Vector), s.dimension)
	}
	op := map[string]string{"cosine": "<=>", "l2": "<->", "ip": "<#>"}[s.metric]
	where, filterArgs, err := searchFilterClauses(req.Filter, 1)
	if err != nil {
		return SearchResponse{}, err
	}
	args := []interface{}{pgvector.NewVector(req.Vector)}
	args = append(args, filterArgs...)
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, topK(req.TopK, DefaultTopK))
	query := fmt.Sprintf("SELECT id, content, metadata, embedding %s $1 AS distance FROM %s%s ORDER BY embedding %s $1 LIMIT $%d", op, s.table, clause, op, len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("pgvector search: %w", err)
	}
	defer rows.Close()
	out := SearchResponse{Backend: "pgvector", Results: []SearchResult{}}
	for rows.Next() {
		var id, content string
		var raw []byte
		var distance float64
		if err := rows.Scan(&id, &content, &raw, &distance); err != nil {
			return SearchResponse{}, fmt.Errorf("scan pgvector result: %w", err)
		}
		meta := map[string]interface{}{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &meta); err != nil {
				return SearchResponse{}, fmt.Errorf("decode metadata: %w", err)
			}
		}
		score := 1 - distance
		if s.metric == "l2" {
			score = 1 / (1 + distance)
		} else if s.metric == "ip" {
			score = -distance
		}
		out.Results = append(out.Results, SearchResult{ID: id, Content: content, Score: score, Metadata: meta})
	}
	return out, rows.Err()
}

// searchFilterClauses 把过滤器键值编译为 metadata JSONB 文本投影谓词。
// pgvector 表仅含 id/content/embedding/metadata 四列：过滤键若按裸列名拼
// WHERE（如 metadata 里不存在的 tenantID）会因列不存在必然报错，检索静默降级。
// 值统一按文本比较：JSON 数字/字符串经 ->> 投影为文本后与 %v 格式化值相等即命中。
// 键必须通过 validIdentifier（会内嵌进单引号 SQL 文本），非法键 fail closed。
func searchFilterClauses(filter map[string]interface{}, argOffset int) ([]string, []interface{}, error) {
	where := make([]string, 0, len(filter))
	args := make([]interface{}, 0, len(filter))
	for key, val := range filter {
		if err := validIdentifier(key); err != nil {
			return nil, nil, err
		}
		args = append(args, fmt.Sprintf("%v", val))
		where = append(where, fmt.Sprintf("metadata->>'%s' = $%d", key, argOffset+len(args)))
	}
	return where, args, nil
}

func (s *pgVectorStore) Insert(ctx context.Context, req InsertRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := fmt.Sprintf("INSERT INTO %s (id,content,embedding,metadata) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET content=EXCLUDED.content,embedding=EXCLUDED.embedding,metadata=EXCLUDED.metadata", s.table)
	for _, c := range req.Chunks {
		if s.dimension > 0 && len(c.Vector) != s.dimension {
			return fmt.Errorf("chunk %s vector dimension: got %d, want %d", c.ID, len(c.Vector), s.dimension)
		}
		m, e := json.Marshal(c.Metadata)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, q, c.ID, c.Content, pgvector.NewVector(c.Vector), m); e != nil {
			return fmt.Errorf("insert chunk %s: %w", c.ID, e)
		}
	}
	return tx.Commit()
}

func (s *pgVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	args := make([]interface{}, len(ids))
	p := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		p[i] = fmt.Sprintf("$%d", i+1)
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", s.table, strings.Join(p, ",")), args...)
	return err
}
func (s *pgVectorStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *pgVectorStore) Close() error                   { return s.db.Close() }
func init()                                             { Register("pgvector", newPGVector) }
