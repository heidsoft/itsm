package knowledgeaccess

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupGuardFixture 建一个租户 + 两位用户（财务专员 end_user、管理员 admin）。
func setupGuardFixture(t *testing.T) (*ent.Client, *Guard, context.Context, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:knowledgeaccess?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("acme").SetCode("acme").SetDomain("acme.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("intern").SetName("intern").
		SetEmail("intern@acme.test").SetPasswordHash("x").
		SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	return client, NewGuard(client, zaptest.NewLogger(t).Sugar()), ctx, tenant.ID, user.ID
}

// restrictCategory 为某分类创建一条 permission 记录，即完成「纳管」。
func restrictCategory(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, category string) {
	t.Helper()
	_, err := client.Permission.Create().
		SetCode("knowledge_category_" + category).
		SetName("知识分类-" + category).
		SetResource(CategoryResource).
		SetAction(ActionForCategory(category)).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
}

func TestActionForCategory(t *testing.T) {
	require.Equal(t, "read:财务制度", ActionForCategory("财务制度"))
}

func TestViewerContextRoundTrip(t *testing.T) {
	ctx := WithViewer(context.Background(), Viewer{UserID: 7, Role: "agent"})
	v, ok := ViewerFrom(ctx)
	require.True(t, ok)
	require.Equal(t, 7, v.UserID)
	require.Equal(t, "agent", v.Role)

	_, ok = ViewerFrom(context.Background())
	require.False(t, ok, "未注入时应返回 false，供调用方按匿名处理")
}

func TestRestrictedCategories_EmptyWhenNotManaged(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restricted, err := g.RestrictedCategories(ctx, tenantID)
	require.NoError(t, err)
	require.Empty(t, restricted, "未纳管任何分类时，受限集合应为空")
}

func TestRestrictedCategories_DetectsManagedCategory(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	restricted, err := g.RestrictedCategories(ctx, tenantID)
	require.NoError(t, err)
	require.True(t, restricted["财务制度"])
	require.False(t, restricted["故障处理"], "未纳管的分类不应被标记受限")
}

func TestRestrictedCategories_IsolatesTenant(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	other, err := client.Tenant.Create().
		SetName("other").SetCode("other").SetDomain("other.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	restricted, err := g.RestrictedCategories(ctx, other.ID)
	require.NoError(t, err)
	require.Empty(t, restricted, "A 租户的纳管配置不得泄漏到 B 租户")
}

func TestCanReadCategory_UnmanagedCategoryIsOpen(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// 未纳管分类：任何角色都可读（向后兼容）
	require.True(t, g.CanReadCategory(ctx, tenantID, Viewer{UserID: 1, Role: "end_user"}, "故障处理", 0))
}

func TestCanReadCategory_ManagedCategoryDeniesWithoutPermission(t *testing.T) {
	client, g, ctx, tenantID, userID := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// 已纳管分类：end_user 无对应权限 → 拒绝（fail-closed）
	require.False(t, g.CanReadCategory(ctx, tenantID, Viewer{UserID: userID, Role: "end_user"}, "财务制度", 0))
}

func TestCanReadCategory_ManagedCategoryDeniesAnonymous(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// 未注入身份（空 Viewer）→ 拒绝
	require.False(t, g.CanReadCategory(ctx, tenantID, Viewer{}, "财务制度", 0))
}

func TestCanReadCategory_AuthorExempt(t *testing.T) {
	client, g, ctx, tenantID, userID := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// 作者本人可读自己写的文章，即便落在受限分类
	require.True(t, g.CanReadCategory(ctx, tenantID, Viewer{UserID: userID, Role: "end_user"}, "财务制度", userID))
}

func TestCanReadCategory_SuperAdminBypasses(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// super_admin 由 HasResourcePermission 内部直通
	require.True(t, g.CanReadCategory(ctx, tenantID, Viewer{UserID: 1, Role: "super_admin"}, "财务制度", 0))
}

func TestCanReadCategory_EmptyCategoryIsOpen(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")
	require.True(t, g.CanReadCategory(ctx, tenantID, Viewer{}, "", 0), "无分类文章不参与分类管控")
}

func TestFilterArticles_PreservesOrderAndDropsDenied(t *testing.T) {
	client, g, ctx, tenantID, userID := setupGuardFixture(t)
	defer client.Close()

	restrictCategory(t, ctx, client, tenantID, "财务制度")

	// 受限文章的作者刻意设为他人，避免触发作者豁免而掩盖拦截逻辑
	otherAuthor, err := client.User.Create().
		SetUsername("cfo").SetName("cfo").
		SetEmail("cfo@acme.test").SetPasswordHash("x").
		SetRole("end_user").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	open1 := mustArticle(t, ctx, client, tenantID, userID, "VPN 配置", "故障处理")
	secret := mustArticle(t, ctx, client, tenantID, otherAuthor.ID, "薪酬结构", "财务制度")
	open2 := mustArticle(t, ctx, client, tenantID, userID, "打印机卡纸", "故障处理")

	kept, dropped := g.FilterArticles(ctx, tenantID, Viewer{UserID: userID, Role: "end_user"},
		[]*ent.KnowledgeArticle{open1, secret, open2})

	require.Equal(t, 1, dropped)
	require.Len(t, kept, 2)
	require.Equal(t, open1.ID, kept[0].ID, "应保持原顺序")
	require.Equal(t, open2.ID, kept[1].ID)
}

func TestFilterArticles_EmptyInput(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	kept, dropped := g.FilterArticles(ctx, tenantID, Viewer{}, nil)
	require.Equal(t, 0, dropped)
	require.Empty(t, kept)
}

func TestInvalidate_ClearsCache(t *testing.T) {
	client, g, ctx, tenantID, _ := setupGuardFixture(t)
	defer client.Close()

	_, err := g.RestrictedCategories(ctx, tenantID)
	require.NoError(t, err)
	require.Empty(t, g.RestrictedCategoriesMust(t, ctx, tenantID))

	restrictCategory(t, ctx, client, tenantID, "财务制度")
	g.Invalidate(tenantID)

	require.True(t, g.RestrictedCategoriesMust(t, ctx, tenantID)["财务制度"],
		"Invalidate 后应能读到新的纳管配置")
}

// RestrictedCategoriesMust 测试辅助：忽略错误版本。
func (g *Guard) RestrictedCategoriesMust(t *testing.T, ctx context.Context, tenantID int) map[string]bool {
	t.Helper()
	r, err := g.RestrictedCategories(ctx, tenantID)
	require.NoError(t, err)
	return r
}

func mustArticle(t *testing.T, ctx context.Context, client *ent.Client, tenantID, authorID int, title, category string) *ent.KnowledgeArticle {
	t.Helper()
	a, err := client.KnowledgeArticle.Create().
		SetTitle(title).SetContent("内容：" + title).
		SetCategory(category).SetAuthorID(authorID).
		SetTenantID(tenantID).SetIsPublished(true).Save(ctx)
	require.NoError(t, err)
	return a
}
