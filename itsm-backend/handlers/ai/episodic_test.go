package ai

import (
	"context"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/knowledgeaccess"

	"github.com/stretchr/testify/require"
)

// 情景记忆（历史工单处置经验）的守卫测试。
//
// 情景层与短期层最大的区别是：**它读的是业务数据**。工单有行级可见范围，
// 把用户无权查看的工单处置喂给 LLM 等于绕过权限外泄内容。所以本文件的第一
// 组守卫是权限边界，第二组才是语义与标注。

// episodeEnv 造一个带工单的租户环境。
type episodeEnv struct {
	client   *ent.Client
	tenantID int
	userA    int
	userB    int
}

func newEpisodeEnv(t *testing.T) *episodeEnv {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ai_episode?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("t").SetCode("t").
		SetDomain("t.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ua, err := client.User.Create().SetUsername("ua").SetName("ua").
		SetEmail("ua@e.com").SetPasswordHash("x").SetActive(true).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	ub, err := client.User.Create().SetUsername("ub").SetName("ub").
		SetEmail("ub@e.com").SetPasswordHash("x").SetActive(true).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	return &episodeEnv{client: client, tenantID: tenant.ID, userA: ua.ID, userB: ub.ID}
}

// seedTicket 建一张工单。
func (e *episodeEnv) seedTicket(t *testing.T, tkTitle, tkStatus, resolution string, requester, assignee int) int {
	t.Helper()
	ctx := context.Background()
	tk, err := e.client.Ticket.Create().
		SetTitle(tkTitle).
		SetDescription(tkTitle + " 描述").
		SetStatus(tkStatus).
		SetType("incident").
		SetPriority("medium").
		SetTicketNumber("T-" + tkTitle).
		SetRequesterID(requester).
		SetAssigneeID(assignee).
		SetTenantID(e.tenantID).
		SetResolution(resolution).
		Save(ctx)
	require.NoError(t, err)
	return tk.ID
}

// TestEpisodeRecallRespectsDataScope 锁行级可见范围。
//
// 这是情景层最关键的一条：agent（OwnedOrAssigned 档）不得召回别人的工单
// 处置经验。管理角色（全租户档）才可以。
func TestEpisodeRecallRespectsDataScope(t *testing.T) {
	env := newEpisodeEnv(t)
	// 这张工单属于 userB：userA 既不是创建人也不是受理人。
	env.seedTicket(t, "CDN 回源超时", common.TicketStatusResolved,
		"调整回源超时到 60 秒", env.userB, env.userB)

	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.entClient = env.client
	ctx := context.Background()

	// agent 档：看不到 userB 的工单 → 无经验。
	episodes := svc.recallEpisodes(ctx, env.tenantID, env.userA, "agent", "CDN 回源超时怎么办")
	require.Empty(t, episodes, "agent 不得召回他人工单的处置经验")

	// 管理档：全租户可见 → 召回得到。
	episodes = svc.recallEpisodes(ctx, env.tenantID, env.userA, "admin", "CDN 回源超时怎么办")
	require.Len(t, episodes, 1, "管理角色应能召回同租户的历史经验")
	require.Equal(t, "调整回源超时到 60 秒", episodes[0].Resolution)
}

// TestEpisodeRecallOwnTicketVisible 锁本人工单对本人可见。
func TestEpisodeRecallOwnTicketVisible(t *testing.T) {
	env := newEpisodeEnv(t)
	env.seedTicket(t, "磁盘空间不足", common.TicketStatusResolved,
		"清理日志并扩容 100G", env.userA, env.userA)

	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.entClient = env.client

	episodes := svc.recallEpisodes(context.Background(), env.tenantID, env.userA, "agent", "磁盘空间不足")
	require.Len(t, episodes, 1, "本人工单的处置经验应可召回")
	require.Equal(t, "清理日志并扩容 100G", episodes[0].Resolution)
}

// TestEpisodeRecallRejectsCrossTenant 锁跨租户隔离。
func TestEpisodeRecallRejectsCrossTenant(t *testing.T) {
	env := newEpisodeEnv(t)
	env.seedTicket(t, "网络抖动", common.TicketStatusResolved, "更换网线", env.userA, env.userA)

	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.entClient = env.client

	episodes := svc.recallEpisodes(context.Background(), env.tenantID+999, env.userA, "admin", "网络抖动")
	require.Empty(t, episodes, "跨租户不得召回任何工单经验")
}

// TestEpisodeRecallOnlyResolvedWithConclusion 锁「经验」的门槛。
//
// 未解决的工单没有经验价值；没有处置结论的工单更不是经验。把进行中的描述
// 当成「以前怎么解决的」喂给模型，是会误导的。
func TestEpisodeRecallOnlyResolvedWithConclusion(t *testing.T) {
	env := newEpisodeEnv(t)
	env.seedTicket(t, "待处理问题", common.TicketStatusOpen, "", env.userA, env.userA)
	env.seedTicket(t, "已解决但无结论", common.TicketStatusResolved, "   ", env.userA, env.userA)
	env.seedTicket(t, "已解决有结论", common.TicketStatusClosed, "重启服务后恢复", env.userA, env.userA)

	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.entClient = env.client

	episodes := svc.recallEpisodes(context.Background(), env.tenantID, env.userA, "admin", "已解决有结论")
	require.Len(t, episodes, 1)
	require.Equal(t, "已解决有结论", episodes[0].Title,
		"只有已解决且有处置结论的工单才是经验")
	require.Equal(t, "重启服务后恢复", episodes[0].Resolution)
}

// TestEpisodeRecallSkipsSoftDeleted 锁软删工单不得进经验。
func TestEpisodeRecallSkipsSoftDeleted(t *testing.T) {
	env := newEpisodeEnv(t)
	id := env.seedTicket(t, "已删除工单", common.TicketStatusResolved, "处置结论", env.userA, env.userA)
	ctx := context.Background()
	_, err := env.client.Ticket.UpdateOneID(id).SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.entClient = env.client

	episodes := svc.recallEpisodes(ctx, env.tenantID, env.userA, "admin", "已删除工单")
	require.Empty(t, episodes, "软删工单不得进入情景召回")
}

// TestEpisodeResultsCarrySource 锁来源标注。
//
// 情景经验必须能被调用方与模型区分出来：object_type=ticket_experience +
// fromMemory + memorySource，且带工单号（模型被要求引用工单号）。
func TestEpisodeResultsCarrySource(t *testing.T) {
	episodes := []episodeCandidate{{
		TicketID: 42, TicketNumber: "T-1001", Title: "CDN 回源超时",
		Resolution: "调整回源超时到 60 秒", ResolutionCategory: "配置调整",
	}}
	results := episodesToResults(episodes)
	require.Len(t, results, 1)
	r := results[0]
	require.Equal(t, "ticket_experience", r["object_type"])
	require.Equal(t, true, r["fromMemory"])
	require.Equal(t, "ticket_experience", r["memorySource"])
	require.Equal(t, "T-1001", r["ticket_number"])
	require.Equal(t, "调整回源超时到 60 秒", r["snippet"])
}

// TestEpisodePromptBlockMarksSource 锁 prompt 里的来源标注。
//
// 不标注的话模型会把历史工单结论说成「知识库文档里写了」——伪造溯源。
func TestEpisodePromptBlockMarksSource(t *testing.T) {
	block := episodePromptBlock([]episodeCandidate{{
		TicketNumber: "T-1001", Title: "CDN 回源超时", Resolution: "调整回源超时到 60 秒",
	}})
	require.Contains(t, block, "历史工单处置经验")
	require.Contains(t, block, "非知识库文档")
	require.Contains(t, block, "T-1001")
	require.Contains(t, block, "调整回源超时到 60 秒")

	require.Empty(t, episodePromptBlock(nil), "无经验时不得产出 prompt 片段")
}

// TestEpisodeKeywordExtraction 锁候选关键词抽取。
//
// 关键约束：工单标题是**多词短语**（「CDN 回源超时」），而用户问的是
// 「CDN 回源超时怎么办」——候选里必须含能命中标题的组合（去掉疑问词后的
// 相邻词组），同时不能只有单词（太泛）。
func TestEpisodeKeywordExtraction(t *testing.T) {
	got := episodeKeywords("CDN 回源超时怎么办")
	require.NotEmpty(t, got)
	require.Contains(t, got, "CDN 回源超时", "必须保留能命中标题的多词组合")

	got2 := episodeKeywords("请问磁盘空间不足怎么处理")
	require.Contains(t, got2, "磁盘空间不足", "客套与疑问前缀应被去掉")

	// 泛用词与空输入不产生候选。
	require.Empty(t, episodeKeywords("它"))
	require.Empty(t, episodeKeywords(""))
	require.Empty(t, episodeKeywords("怎么办"))
}

// TestChatRoleFromContext 锁角色来源。
//
// 角色必须来自认证上下文（与 RAG 分类可见性同一套身份），上下文无身份时
// 返回空串——DataScope 对空角色取最保守档，不会放宽到全租户。
func TestChatRoleFromContext(t *testing.T) {
	ctx := knowledgeaccess.WithViewer(context.Background(),
		knowledgeaccess.Viewer{UserID: 7, Role: "agent"})
	require.Equal(t, "agent", chatRole(ctx))
	require.Empty(t, chatRole(context.Background()), "无身份上下文时角色应为空")
}
