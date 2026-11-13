package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// nextval 桩驱动：模拟 PG 序列 incident_number_seq 的原子递增。
type incidentSeqStubDriver struct{}

func (incidentSeqStubDriver) Open(string) (driver.Conn, error) {
	return &incidentSeqStubConn{}, nil
}

type incidentSeqStubConn struct {
	mu  sync.Mutex
	seq int64
}

func (c *incidentSeqStubConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *incidentSeqStubConn) Close() error              { return nil }
func (c *incidentSeqStubConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (c *incidentSeqStubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "nextval('incident_number_seq')") {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	c.mu.Lock()
	c.seq++
	v := c.seq
	c.mu.Unlock()
	return &incidentSeqStubRows{value: v}, nil
}

type incidentSeqStubRows struct {
	value int64
	done  bool
}

func (r *incidentSeqStubRows) Columns() []string { return []string{"nextval"} }
func (r *incidentSeqStubRows) Close() error      { return nil }
func (r *incidentSeqStubRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.value
	return nil
}

var registerIncidentSeqStubOnce sync.Once

func openIncidentSeqStubDB(t *testing.T) *sql.DB {
	t.Helper()
	registerIncidentSeqStubOnce.Do(func() {
		sql.Register("incident_seq_stub", incidentSeqStubDriver{})
	})
	db, err := sql.Open("incident_seq_stub", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// 收敛锁：云信号/邮件建单路径的发号必须走 incident_number_seq 单一号源，
// 与 HTTP 创建（handlers/incident）、BPMN 服务任务共用同一序列。
func TestIncidentNumberUsesSharedPGSequence(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)
	svc.SetRawDB(openIncidentSeqStubDB(t))

	now := time.Now()
	first, err := svc.generateIncidentNumber(context.Background())
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("INC-%04d%02d-000001", now.Year(), int(now.Month())), first)

	second, err := svc.generateIncidentNumber(context.Background())
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("INC-%04d%02d-000002", now.Year(), int(now.Month())), second)
}

// 未注入 rawDB 的测试夹具：唯一后缀兜底仍保持 INC- 前缀且互不相同。
func TestIncidentNumberFallbackWithoutRawDB(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)

	first, err := svc.generateIncidentNumber(context.Background())
	require.NoError(t, err)
	require.Contains(t, first, "INC-")
	second, err := svc.generateIncidentNumber(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, first, second)
}

// 退役锁：独立 Redis 号源（sequence:incident:*）不得复活——两套号源互撞是
// 本次收敛修复的根因。
func TestIncidentRedisNumberSourceRetired(t *testing.T) {
	source, err := os.ReadFile("incident_service.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "sequence:incident:",
		"incident 发号已收敛到 incident_number_seq，禁止复活独立 Redis 号源")
	require.NotContains(t, string(source), "generateIncidentNumberWithDB",
		"MAX+1 备用号源已退役，禁止复活")
}
