package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// L9 回归锁：BPMN 审计读面不得再把整张 process_audit_logs 拉进内存。
// HTTP 入口的 startTime/endTime 完全由 query 决定（默认 7 天但可传 100 年），
// 且 failInternal 会把 service 错误统一兜底成 500，因此这里验证「夹紧 + LIMIT」。

func TestBPMNAuditWindow_ClampsUnboundedRange(t *testing.T) {
	s := NewBPMNAuditService(nil, zaptest.NewLogger(t).Sugar())
	end := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(-100, 0, 0)

	gotStart, gotEnd := s.clampBPMNAuditWindow(start, end, "test")

	require.Equal(t, end, gotEnd)
	require.Equal(t, time.Duration(maxBPMNAuditWindowDays)*24*time.Hour, gotEnd.Sub(gotStart))
	assert.True(t, gotStart.After(start), "100 年窗口必须被夹紧")
}

func TestBPMNAuditWindow_InvertedAndZeroAndNormal(t *testing.T) {
	s := NewBPMNAuditService(nil, zaptest.NewLogger(t).Sugar())
	end := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)

	t.Run("inverted collapses to empty window", func(t *testing.T) {
		gotStart, gotEnd := s.clampBPMNAuditWindow(end.AddDate(0, 0, 1), end, "test")
		assert.Equal(t, end, gotStart)
		assert.Equal(t, end, gotEnd)
	})

	t.Run("zero start defaults to max window", func(t *testing.T) {
		gotStart, gotEnd := s.clampBPMNAuditWindow(time.Time{}, end, "test")
		assert.Equal(t, end, gotEnd)
		assert.Equal(t, time.Duration(maxBPMNAuditWindowDays)*24*time.Hour, gotEnd.Sub(gotStart))
	})

	t.Run("normal window is untouched", func(t *testing.T) {
		start := end.AddDate(0, 0, -7)
		gotStart, gotEnd := s.clampBPMNAuditWindow(start, end, "test")
		assert.Equal(t, start, gotStart)
		assert.Equal(t, end, gotEnd)
	})
}

// 守卫：逐处比对三个曾经裸 .All(ctx) 的方法，不能只数全文出现次数。
func TestBPMNAuditReadPaths_EachHasRowLimit(t *testing.T) {
	src, err := os.ReadFile("bpmn_audit_service.go")
	require.NoError(t, err)
	text := string(src)

	for _, name := range []string{"GetProcessTimeline", "GetUserActivity", "GetActivityStatistics"} {
		body := bpmnAuditFuncBody(text, name)
		require.NotEmpty(t, body, "找不到方法 %s", name)
		require.Contains(t, body, "Limit(maxBPMNAuditRows)", "%s 缺少行数上限", name)
	}
	require.Contains(t, bpmnAuditFuncBody(text, "QueryAuditLogs"), "clampBPMNAuditWindow",
		"分页查询也必须夹紧时间窗口，否则 Count 仍全表扫")
}

func bpmnAuditFuncBody(src, name string) string {
	marker := "func (s *BPMNAuditService) " + name + "("
	i := strings.Index(src, marker)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	if j := strings.Index(rest[len(marker):], "\nfunc "); j >= 0 {
		return rest[:len(marker)+j]
	}
	return rest
}

func TestBPMNAudit_GetUserActivityClampsCenturyWindow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	s := NewBPMNAuditService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	end := time.Now().UTC()
	stamps := []time.Time{
		end.AddDate(0, 0, -400), // 超出 366 天夹紧窗口，不得返回
		end.AddDate(0, 0, -5),
		end.AddDate(0, 0, -1),
	}
	for _, ts := range stamps {
		_, err := client.ProcessAuditLog.Create().
			SetProcessInstanceID(1).
			SetProcessInstanceKey("inst-1").
			SetProcessDefinitionKey("def-1").
			SetProcessDefinitionID(1).
			SetActivityID("act-1").
			SetActivityType("userTask").
			SetAction("completed").
			SetUserID(7).
			SetTenantID(3).
			SetTimestamp(ts).
			Save(ctx)
		require.NoError(t, err)
	}

	logs, err := s.GetUserActivity(ctx, 7, 3, end.AddDate(-100, 0, 0), end)
	require.NoError(t, err)
	assert.Len(t, logs, 2, "100 年窗口夹紧到 366 天后，400 天前的记录不得入结果")
}
