package skill_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/handlers/skill"
)

// 这个用例不是「多跑几轮碰运气」的并发测试：它在驱动层把另一写入方精确插入
// 到 UpsertDefinition 的 SELECT 与 INSERT 之间，确定性地复现 check-then-act。
// 旧实现必然撞 UNIQUE (tenant_id, code)；DB 层 UPSERT 则把它判成「无需写入」。

var raceDriverSeq uint64

type raceDriver struct {
	base      driver.Driver
	helper    *sql.DB
	insertSQL string
	args      []any
	fired     atomic.Bool
}

func (d *raceDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return &raceConn{Conn: conn, driver: d}, nil
}

type raceConn struct {
	driver.Conn
	driver *raceDriver
}

// QueryContext 先把结果集读进内存（确保 SELECT 快照已取到），再让同一 code 的
// 行由另一个连接落库，最后把原结果集交付调用方。调用方看到的仍是「查无此行」。
func (c *raceConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, fmt.Errorf("underlying sqlite conn does not implement QueryerContext")
	}
	rows, err := queryer.QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(query, "SELECT checksum FROM skill_definitions") ||
		!c.driver.fired.CompareAndSwap(false, true) {
		return rows, nil
	}
	buffered, err := bufferDriverRows(rows)
	if err != nil {
		return nil, err
	}
	if _, err := c.driver.helper.ExecContext(ctx, c.driver.insertSQL, c.driver.args...); err != nil {
		return nil, fmt.Errorf("inject concurrent writer: %w", err)
	}
	return buffered, nil
}

type bufferedDriverRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func bufferDriverRows(rows driver.Rows) (*bufferedDriverRows, error) {
	defer rows.Close()
	out := &bufferedDriverRows{cols: append([]string(nil), rows.Columns()...)}
	for {
		values := make([]driver.Value, len(out.cols))
		if err := rows.Next(values); err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, err
		}
		out.data = append(out.data, values)
	}
}

func (r *bufferedDriverRows) Columns() []string { return r.cols }
func (r *bufferedDriverRows) Close() error      { return nil }
func (r *bufferedDriverRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

func TestUpsertDefinition_CheckThenActRaceIsClosedInSQL(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "race") +
		"?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL"
	helper, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = helper.Close() })
	createDefinitionTable(t, helper)

	row := concurrentUpsertRow("ai.race", "v1", "sha256:race", 0)
	drv := &raceDriver{
		base:      &sqlite3.SQLiteDriver{},
		helper:    helper,
		insertSQL: `INSERT INTO skill_definitions (code, version, title, checksum, tenant_id) VALUES (?, ?, ?, ?, ?)`,
		args:      []any{row.Code, row.Version, row.Title, row.Checksum, row.TenantID},
	}
	driverName := fmt.Sprintf("sqlite3-race-%d-%d", time.Now().UnixNano(), atomic.AddUint64(&raceDriverSeq, 1))
	sql.Register(driverName, drv)

	db, err := sql.Open(driverName, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1) // SELECT 与 INSERT 必须走同一个被包装的连接
	store := skill.NewDBDefinitionStore(db)

	_, changed, err := store.UpsertDefinition(context.Background(), row)
	require.NoError(t, err, "SELECT 之后同键行落库，UPSERT 不得报唯一约束错误")
	assert.False(t, changed, "同 checksum 的并发输家必须报告零变更")
	assert.Equal(t, 1, countDefinitionRows(t, helper), "竞态只应保留一行")
}
