package router

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 阻塞驱动：任何查询都等到 ctx 结束才返回，模拟 DB 高负载下查询无限挂起。
type readinessBlockingDriver struct{}

func (readinessBlockingDriver) Open(string) (driver.Conn, error) {
	return readinessBlockingConn{}, nil
}

type readinessBlockingConn struct{}

func (readinessBlockingConn) Prepare(string) (driver.Stmt, error) {
	return readinessBlockingStmt{}, nil
}
func (readinessBlockingConn) Close() error              { return nil }
func (readinessBlockingConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (c readinessBlockingConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type readinessBlockingStmt struct{}

func (readinessBlockingStmt) Close() error  { return nil }
func (readinessBlockingStmt) NumInput() int { return -1 }
func (readinessBlockingStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("not implemented")
}
func (readinessBlockingStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errors.New("not implemented")
}

var registerReadinessDriverOnce sync.Once

func openReadinessBlockingDB(t *testing.T) *sql.DB {
	t.Helper()
	registerReadinessDriverOnce.Do(func() {
		sql.Register("readiness_blocking", readinessBlockingDriver{})
	})
	db, err := sql.Open("readiness_blocking", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// 回归锁：探针必须自带超时上界。修复前查询直接沿用父 ctx，父 ctx 给 30s
// 就挂 30s（远超上界断言而红）；修复后内部 5s 截止，快速返回 not-ready。
func TestCheckInitializationReadinessBoundsQueryTime(t *testing.T) {
	db := openReadinessBlockingDB(t)
	parentCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	result := checkInitializationReadiness(parentCtx, db)
	elapsed := time.Since(start)

	require.False(t, result.Ready)
	require.Contains(t, result.Reason, "context deadline exceeded")
	require.Less(t, elapsed, readinessQueryTimeout+5*time.Second,
		"探针必须在内部超时上界附近返回，不得沿用父 ctx 的长 deadline")
}

// 父 ctx 取消必须立即传导（探针随请求取消而放弃，不继续占连接）。
func TestCheckInitializationReadinessHonorsParentCancellation(t *testing.T) {
	db := openReadinessBlockingDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	result := checkInitializationReadiness(ctx, db)
	elapsed := time.Since(start)

	require.False(t, result.Ready)
	require.Contains(t, result.Reason, "context canceled")
	require.Less(t, elapsed, 2*time.Second)
}
