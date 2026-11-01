package ai

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
)

// 会话读面/写面的**所有权边界**守卫。
//
// 此前 GetConversation 只校验 tenant_id、GetMessages 更是连租户谓词都没有
// （message 表按 conversation_id 过滤即可）。会话是用户私有的对话内容
// （ListConversations 本来就按 userID 过滤），于是同租户内 A 用户只要
// 拿到 B 用户的 conversation_id，就能读到对方的整段对话——租户是资源边界，
// 用户是所有权边界，两者都要校验。
//
// 这些断言用真实 ent 内存库跑：谓词写错时 mock 测不出来（mock 直接返回
// 零值），必须让 SQL 真的执行。

func newConversationGuardRepo(t *testing.T) (*EntRepository, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ai_conversation_guard?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("tenant-a").SetCode("tenant-a").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("tenant-b").SetCode("tenant-b").Save(ctx)
	require.NoError(t, err)

	userA1, err := client.User.Create().SetUsername("a1").SetName("a1").
		SetEmail("a1@example.com").SetPasswordHash("x").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	userA2, err := client.User.Create().SetUsername("a2").SetName("a2").
		SetEmail("a2@example.com").SetPasswordHash("x").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	// 租户 B 的用户存在即可（会话归属在 tenant 维度校验，不需要他的 id）。
	_, err = client.User.Create().SetUsername("b1").SetName("b1").
		SetEmail("b1@example.com").SetPasswordHash("x").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	return repo, tenantA.ID, userA1.ID, userA2.ID
}

// TestGetConversationRejectsOtherUserInSameTenant 锁同租户跨用户不可读。
func TestGetConversationRejectsOtherUserInSameTenant(t *testing.T) {
	ctx := context.Background()
	repo, tenantA, userA1, userA2 := newConversationGuardRepo(t)

	conv, err := repo.CreateConversation(ctx, &Conversation{
		Title: "a1 的私密对话", UserID: userA1, TenantID: tenantA,
	})
	require.NoError(t, err)
	_, err = repo.CreateMessage(ctx, &Message{
		ConversationID: conv.ID, Role: "user", Content: "只有 a1 能看的敏感内容",
	})
	require.NoError(t, err)

	// 属主可读。
	got, err := repo.GetConversation(ctx, conv.ID, tenantA, userA1)
	require.NoError(t, err)
	require.Equal(t, conv.ID, got.ID)

	// 同租户的另一用户不可读。
	_, err = repo.GetConversation(ctx, conv.ID, tenantA, userA2)
	require.Error(t, err, "同租户内他人会话不得可读")
}

// TestGetMessagesRejectsOtherTenant 锁跨租户不可读消息。
//
// message 表自身没有 tenant_id 字段，租户过滤必须经 conversation 边做。
func TestGetMessagesRejectsOtherTenant(t *testing.T) {
	ctx := context.Background()
	repo, tenantA, userA1, _ := newConversationGuardRepo(t)

	conv, err := repo.CreateConversation(ctx, &Conversation{
		Title: "tenant-a 的对话", UserID: userA1, TenantID: tenantA,
	})
	require.NoError(t, err)
	_, err = repo.CreateMessage(ctx, &Message{
		ConversationID: conv.ID, Role: "user", Content: "tenant-a 的内容",
	})
	require.NoError(t, err)

	// 本租户可读。
	msgs, err := repo.GetMessages(ctx, conv.ID, tenantA)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	// 别的租户即便知道 conversation_id 也读不到。
	msgs, err = repo.GetMessages(ctx, conv.ID, tenantA+999)
	require.NoError(t, err)
	require.Empty(t, msgs, "跨租户不得读到任何消息")
}

// TestDeleteConversationRejectsOtherUser 锁删除同样是所有权操作。
func TestDeleteConversationRejectsOtherUser(t *testing.T) {
	ctx := context.Background()
	repo, tenantA, userA1, userA2 := newConversationGuardRepo(t)

	conv, err := repo.CreateConversation(ctx, &Conversation{
		Title: "a1 的对话", UserID: userA1, TenantID: tenantA,
	})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteConversation(ctx, conv.ID, tenantA, userA2),
		"同租户他人不得删除会话")

	// 属主删除成功。
	require.NoError(t, repo.DeleteConversation(ctx, conv.ID, tenantA, userA1))
	_, err = repo.GetConversation(ctx, conv.ID, tenantA, userA1)
	require.Error(t, err, "属主删除后不得再读到")
}
