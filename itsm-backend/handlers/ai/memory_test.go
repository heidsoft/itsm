package ai

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
)

// AI 记忆层的守卫测试。
//
// 记忆层的价值全在「让多轮对话里的上下文真正参与回答」。但记忆也是最容易
// 出事的地方：它可能跨用户泄漏、可能把「聊过」冒充成「库里写着」、可能
// 反而让检索变差。下面每条守卫对应一类真实事故。

// memoryTestRepo 造一个带历史消息的会话。
type memoryTestRepo struct {
	Repository
	messages []*Message
	convID   int
	tenantID int
}

func (m *memoryTestRepo) GetMessages(_ context.Context, conversationID int, tenantID int) ([]*Message, error) {
	if conversationID != m.convID || tenantID != m.tenantID {
		return nil, nil
	}
	return m.messages, nil
}

// TestMemoryExtractsEntityFromHistory 锁实体抽取。
//
// 用户问「它怎么修」时，历史里的工单号是最该拿去检索的东西——它是当前
// query 里没有、但历史里有的语义锚点。
func TestMemoryExtractsEntityFromHistory(t *testing.T) {
	turns := []*Message{
		{Role: "user", Content: "之前那个 TKT-20260831 的工单现在什么状态？"},
		{Role: "assistant", Content: "该工单处于处理中。"},
	}
	queries := extractMemoryQueries(turns, 3)
	require.Contains(t, queries, "TKT-20260831",
		"历史里的工单号必须被抽出为补充检索词")

	// 事件号、CI 号同样识别。
	for _, entity := range []string{"INC-1001", "CI-88", "SLA-2026"} {
		got := extractMemoryQueries([]*Message{{Role: "user", Content: "关于 " + entity + " 的问题"}}, 3)
		require.Contains(t, got, entity, "实体 %s 必须被抽出", entity)
	}
}

// TestMemorySkipsAssistantMessages 锁只从用户消息抽词。
//
// 助手的回答里常有系统生成的措辞（「已为您查询…」），它们不是用户的检索
// 意图，抽出来只会污染检索。
func TestMemorySkipsAssistantMessages(t *testing.T) {
	turns := []*Message{
		{Role: "user", Content: "数据库连接超时怎么处理"},
		{Role: "assistant", Content: "建议检查 CHG-99999 变更记录"},
	}
	queries := extractMemoryQueries(turns, 3)
	require.NotContains(t, queries, "CHG-99999",
		"助手消息里的实体不得进入补充检索词")
}

// TestMemoryProducesNoQueryWithoutHistory 锁无历史时不编造。
//
// 记忆读不到或历史为空时，必须返回空补充词——凭空造一个检索词比没有记忆
// 更糟（会把无关内容塞进上下文）。
func TestMemoryProducesNoQueryWithoutHistory(t *testing.T) {
	require.Empty(t, extractMemoryQueries(nil, 3))
	require.Empty(t, extractMemoryQueries([]*Message{}, 3))

	repo := &memoryTestRepo{}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	mem := svc.recallConversationMemory(context.Background(), 7, 1, 0)
	require.False(t, mem.Used)
	require.Empty(t, mem.Queries)
}

// TestMemoryNotUsedForNewConversation 锁新会话不触发记忆读。
//
// convID==0 是新会话，没有历史可读；此时不该做无意义的仓储查询。
func TestMemoryNotUsedForNewConversation(t *testing.T) {
	repo := &memoryTestRepo{}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	mem := svc.recallConversationMemory(context.Background(), 7, 1, 0)
	require.False(t, mem.Used)
	require.Empty(t, mem.Turns)
}

// TestMemoryRespectsTenantBoundary 锁记忆不跨租户。
//
// 记忆来自仓储查询，仓储带租户谓词；这一条锁「传错租户就读不到」——
// 万一将来有人把租户参数写死或漏传，这里会红。
func TestMemoryRespectsTenantBoundary(t *testing.T) {
	repo := &memoryTestRepo{
		convID:   5,
		tenantID: 7,
		messages: []*Message{{Role: "user", Content: "TKT-1 的状态"}},
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// 正确租户读到。
	mem := svc.recallConversationMemory(context.Background(), 7, 1, 5)
	require.True(t, mem.Used)
	require.NotEmpty(t, mem.Queries)

	// 别的租户读不到（仓储按租户过滤）。
	mem = svc.recallConversationMemory(context.Background(), 8, 1, 5)
	require.False(t, mem.Used)
	require.Empty(t, mem.Queries)
}

// TestMemoryLimitIsBounded 锁记忆规模有上限。
//
// 记忆不是越多越好：整段历史塞进检索既慢又会挤掉当前 query 的命中。
func TestMemoryLimitIsBounded(t *testing.T) {
	repo := &memoryTestRepo{
		convID:   5,
		tenantID: 7,
	}
	for i := 0; i < 30; i++ {
		repo.messages = append(repo.messages, &Message{
			Role: "user", Content: "问题 TKT-" + itoa(1000+i) + " 怎么处理",
		})
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	mem := svc.recallConversationMemory(context.Background(), 7, 1, 5)

	require.LessOrEqual(t, len(mem.Turns), maxMemoryTurns, "回读轮次必须限量")
	require.LessOrEqual(t, len(mem.Queries), maxMemoryQueries, "补充检索词必须限量")
}

// TestMemoryQueryExtractionSkipsGenericWords 锁抽词跳过泛用词。
//
// 「怎么处理」「怎么办」这类词几乎不出现在知识文章标题里，带上只会稀释
// 检索——真正有检索价值的是「CDN 缓存预热」这种实义短语。
func TestMemoryQueryExtractionSkipsGenericWords(t *testing.T) {
	turns := []*Message{{Role: "user", Content: "请问怎么处理这个问题"}}
	queries := extractMemoryQueries(turns, 3)
	for _, q := range queries {
		require.NotEqual(t, "怎么处理这个问题", q, "泛用词短语不得原样进入检索")
	}
}

// TestMergeMemoryResultsMarksSource 锁记忆条目的来源标注。
//
// 记忆召回的条目必须标 fromMemory 与 memoryQuery：调用方要能区分
// 「库里写着」与「我们之前聊过」。不标就是让两者冒充同一权威来源。
func TestMergeMemoryResultsMarksSource(t *testing.T) {
	primary := []map[string]any{
		{"object_type": "kb", "id": 1, "title": "主路命中"},
	}
	memory := []map[string]any{
		{"object_type": "kb", "id": 2, "title": "记忆命中"},
		{"object_type": "kb", "id": 1, "title": "主路命中"}, // 重复，应被去重
	}
	merged := mergeMemoryResults(primary, memory, "TKT-1")

	require.Len(t, merged, 2, "重复项必须去重")
	// 主路在前（权威优先）。
	require.Equal(t, "主路命中", merged[0]["title"])
	// 记忆项带标注。
	require.Equal(t, true, merged[1]["fromMemory"])
	require.Equal(t, "TKT-1", merged[1]["memoryQuery"])
	// 主路项不得被标成记忆。
	_, marked := merged[0]["fromMemory"]
	require.False(t, marked, "主路结果不得被标为记忆来源")
}

// TestMemoryIsolatedFromRealRepository 锁记忆仓储的租户谓词真实生效。
//
// 这条用真实 ent 内存库：mock 测不出谓词写错（mock 直接返回零值），
// 必须让 SQL 真跑一次。
func TestMemoryIsolatedFromRealRepository(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ai_memory_repo?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("ta").SetCode("ta").Save(ctx)
	require.NoError(t, err)
	userA, err := client.User.Create().SetUsername("ua").SetName("ua").
		SetEmail("ua@example.com").SetPasswordHash("x").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	conv, err := repo.CreateConversation(ctx, &Conversation{
		Title: "会话", UserID: userA.ID, TenantID: tenantA.ID,
	})
	require.NoError(t, err)
	_, err = repo.CreateMessage(ctx, &Message{
		ConversationID: conv.ID, Role: "user", Content: "TKT-777 的问题",
	})
	require.NoError(t, err)

	svc := NewService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// 本租户本用户读到记忆。
	mem := svc.recallConversationMemory(ctx, tenantA.ID, userA.ID, conv.ID)
	require.True(t, mem.Used, "属主应能读到自己的会话记忆")

	// 别的租户读不到。
	mem = svc.recallConversationMemory(ctx, tenantA.ID+999, userA.ID, conv.ID)
	require.False(t, mem.Used, "跨租户不得读到会话记忆")
}
