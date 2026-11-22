package skill_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/handlers/skill"
)

// TestUpsertDefinition_PostgreSQLDialect 在真实 PostgreSQL 的 JSONB + 唯一约束上
// 验证与 SQLite 相同的 UPSERT 语义。默认 skip：只有显式提供一次性测试库 DSN
// （库名必须以 itsm_init_test_ 开头）才运行，避免误连生产库。
//
//	用法：ITSM_TEST_POSTGRES_DSN='postgres://...@127.0.0.1:5432/itsm_init_test_<name>?sslmode=disable' \
//	  go test ./handlers/skill -run TestUpsertDefinition_PostgreSQLDialect -count=1
func TestUpsertDefinition_PostgreSQLDialect(t *testing.T) {
	dsn := os.Getenv("ITSM_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ITSM_TEST_POSTGRES_DSN 未设置；PostgreSQL 方言用例不纳入默认测试")
	}
	parsed, err := url.Parse(dsn)
	require.NoError(t, err, "PostgreSQL 测试 DSN 必须是 URL 形式")
	dbName := strings.TrimPrefix(parsed.Path, "/")
	require.True(t, strings.HasPrefix(dbName, "itsm_init_test_"),
		"拒绝在非一次性测试库上执行：got %q", dbName)

	admin, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close() })

	schema := fmt.Sprintf("skill_upsert_%d", time.Now().UnixNano())
	_, err = admin.ExecContext(context.Background(), "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
	})

	// 与迁移 023 等价的列类型及唯一约束；每个连接通过 DSN 固定 search_path，
	// 并发连接不会写到 public 或别的 schema。
	_, err = admin.ExecContext(context.Background(), `
CREATE TABLE `+pq.QuoteIdentifier(schema)+`.skill_definitions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    version VARCHAR(20) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(50) NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    required_permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    executor JSONB NOT NULL DEFAULT '{}'::jsonb,
    provider VARCHAR(100) NOT NULL DEFAULT '',
    author VARCHAR(100) NOT NULL DEFAULT '',
    checksum VARCHAR(80) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
)`)
	require.NoError(t, err)

	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	db, err := sql.Open("postgres", parsed.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(8)
	store := skill.NewDBDefinitionStore(db)

	row := concurrentUpsertRow("ai.pg.concurrent", "v1", "sha256:pg-stable", 0)
	const workers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := store.UpsertDefinition(context.Background(), row)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "PostgreSQL 并发同 checksum 不得产生 23505")
	}

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM skill_definitions`).Scan(&count))
	assert.Equal(t, 1, count)
	var checksum, version string
	require.NoError(t, db.QueryRow(`SELECT checksum, version FROM skill_definitions`).Scan(&checksum, &version))
	assert.Equal(t, "sha256:pg-stable", checksum)
	assert.Equal(t, "v1", version)

	// 串行重跑必须零变更，证明 DO UPDATE 的 WHERE 在 JSONB 方言同样生效。
	_, changed, err := store.UpsertDefinition(context.Background(), row)
	require.NoError(t, err)
	assert.False(t, changed)
}
