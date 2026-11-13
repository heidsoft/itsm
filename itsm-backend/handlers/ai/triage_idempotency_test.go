package ai_test

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/ai"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件是 prod F2 的回归锁。
//
// 实测现象：每次进入工单详情页都会重新 POST /api/v1/ai/triage（同一工单累计 5 次），
// AI 建议的分类在「存储」与「网络」之间漂移，采纳按钮刷新后仍然显示，
// 且每次点击都改写一次审计底账（PATCH /api/v1/ai/audit-logs/5）。
//
// 根因：triage 请求体不带 ticketId，后端无法按工单去重，每次都重跑模型并新增一条底账。
//
// 修复：请求体带 ticketId 时按工单幂等——已有结论直接复用，不再调用模型、不再新增底账。

// countingTriageAdvisor 统计被调用的次数，用于断言「没有重复调模型」。
type countingTriageAdvisor struct {
	calls int
}

func (c *countingTriageAdvisor) SuggestForTenant(_ context.Context, title, description string, _ int) dto.TriageResult {
	c.calls++
	return dto.TriageResult{
		Category:      "storage",
		Priority:      "medium",
		Confidence:    0.82,
		Explanation:   "关键词命中存储",
		Model:         "test-model",
		PromptVersion: "v1",
	}
}

// newTriageIdempotencyEnv 构造带真实 ent 仓储的分诊服务。
func newTriageIdempotencyEnv(t *testing.T) (*ai.Service, *countingTriageAdvisor) {
	t.Helper()
	dsn := "file:ai_triage_idem_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })

	repo := ai.NewEntRepository(client)
	advisor := &countingTriageAdvisor{}
	// 直接注入计数替身：验证「是否真的重跑了模型」。
	svc := ai.NewService(repo, zap.NewNop().Sugar(), nil, nil, nil, nil, nil, advisor, nil, nil)
	return svc, advisor
}

// TestTriageTicketIsIdempotentPerTicket 钉住核心语义：
// 同一工单连查两次，只分析一次、只落一条结论，第二次复用既有结论。
func TestTriageTicketIsIdempotentPerTicket(t *testing.T) {
	svc, advisor := newTriageIdempotencyEnv(t)
	ctx := context.Background()

	first, err := svc.TriageTicket(ctx, 1, 7, "磁盘告警", "存储卷使用率 95%", "", "", 42)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, 1, advisor.calls, "首次分诊必须调用模型")

	firstMap := first.(map[string]interface{})
	assert.Equal(t, "storage", firstMap["suggestions"].(map[string]interface{})["category"])

	second, err := svc.TriageTicket(ctx, 1, 7, "磁盘告警", "存储卷使用率 95%", "", "", 42)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 1, advisor.calls,
		"同一工单再次分诊不得重复调用模型（prod 实测 F2：每次进详情都重跑一次）")

	secondMap := second.(map[string]interface{})
	assert.Equal(t, true, secondMap["cached"], "第二次必须命中缓存标记")
	assert.Equal(t,
		firstMap["suggestions"].(map[string]interface{})["category"],
		secondMap["suggestions"].(map[string]interface{})["category"],
		"复用结论必须与首次一致，分类不得在多次进入详情页之间漂移")

	// 换一个工单：必须重新分析，不得跨工单串结论。
	_, err = svc.TriageTicket(ctx, 1, 7, "网络抖动", "内网丢包", "", "", 43)
	require.NoError(t, err)
	assert.Equal(t, 2, advisor.calls, "不同工单必须各自分析")
}

// TestTriageTicketWithoutTicketIdStillWorks 保护旧调用方：
// 不带 ticketId 的请求（AI 技能、其它入口）行为不变，每次都真实分析。
func TestTriageTicketWithoutTicketIdStillWorks(t *testing.T) {
	svc, advisor := newTriageIdempotencyEnv(t)
	ctx := context.Background()

	_, err := svc.TriageTicket(ctx, 1, 7, "标题", "描述", "", "", 0)
	require.NoError(t, err)
	_, err = svc.TriageTicket(ctx, 1, 7, "标题", "描述", "", "", 0)
	require.NoError(t, err)
	assert.Equal(t, 2, advisor.calls, "不带 ticketId 时不得改变既有行为")
}
