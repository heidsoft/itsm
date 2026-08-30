package service

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/common/knowledgeaccess"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupKnowledgeAccessE2E 搭建 RAG 知识可见性端到端夹具：
// 一个租户、两位用户（普通员工 + 财务）、三篇文章（两篇公开 + 一篇财务受限）。
func setupKnowledgeAccessE2E(t *testing.T) (*ent.Client, *RAGService, context.Context, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("acme").SetCode("acme").SetDomain("acme.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	employee, err := client.User.Create().
		SetUsername("employee").SetName("employee").
		SetEmail("employee@acme.test").SetPasswordHash("x").
		SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 财务受限文章的作者：既非访问者本人，也用于验证作者豁免不误伤
	cfo, err := client.User.Create().
		SetUsername("cfo").SetName("cfo").
		SetEmail("cfo@acme.test").SetPasswordHash("x").
		SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 公开知识
	_, err = client.KnowledgeArticle.Create().
		SetTitle("VPN 连接配置指南").SetContent("配置 VPN 客户端需要导入证书").
		SetCategory("故障处理").SetAuthorID(cfo.ID).
		SetTenantID(tenant.ID).SetIsPublished(true).Save(ctx)
	require.NoError(t, err)

	// 受限知识：财务制度被纳管后，普通员工不可见
	salary, err := client.KnowledgeArticle.Create().
		SetTitle("2026 年薪酬结构调整方案").SetContent("薪酬结构调整涉及各职级带宽，属机密").
		SetCategory("财务制度").SetAuthorID(cfo.ID).
		SetTenantID(tenant.ID).SetIsPublished(true).Save(ctx)
	require.NoError(t, err)

	// 纳管「财务制度」分类：写入一条 RBAC permission 记录
	_, err = client.Permission.Create().
		SetCode("knowledge_category_finance").
		SetName("知识分类-财务制度").
		SetResource(knowledgeaccess.CategoryResource).
		SetAction(knowledgeaccess.ActionForCategory("财务制度")).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// keyword-only RAG（sqlite 无 pgvector），useKeyword=true 走 keywordSearch
	rag := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	return client, rag, ctx, tenant.ID, employee.ID, salary.ID
}

func TestE2E_RAG_HidesRestrictedKnowledgeFromUnprivilegedUser(t *testing.T) {
	client, rag, ctx, tenantID, employeeID, salaryID := setupKnowledgeAccessE2E(t)
	defer client.Close()

	// 普通员工检索「薪酬」——内容确实存在，但属于受限分类
	viewerCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: employeeID, Role: "end_user"})
	results, err := rag.Ask(viewerCtx, tenantID, "薪酬", 20)
	require.NoError(t, err)

	for _, r := range results {
		require.NotEqual(t, salaryID, r["id"],
			"受限分类知识不得泄漏给无权限用户：%+v", r["title"])
	}
}

func TestE2E_RAG_ReturnsRestrictedKnowledgeToSuperAdmin(t *testing.T) {
	client, rag, ctx, tenantID, _, salaryID := setupKnowledgeAccessE2E(t)
	defer client.Close()

	adminCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: 1, Role: "super_admin"})
	results, err := rag.Ask(adminCtx, tenantID, "薪酬", 20)
	require.NoError(t, err)

	found := false
	for _, r := range results {
		if r["id"] == salaryID {
			found = true
		}
	}
	require.True(t, found, "super_admin 应能检索到受限分类知识")
}

func TestE2E_RAG_AnonymousViewerSeesNoRestrictedKnowledge(t *testing.T) {
	client, rag, ctx, tenantID, _, salaryID := setupKnowledgeAccessE2E(t)
	defer client.Close()

	// 未注入 Viewer 的调用方：按匿名处理，受限分类一律排除（fail-closed）
	results, err := rag.Ask(ctx, tenantID, "薪酬", 20)
	require.NoError(t, err)

	for _, r := range results {
		require.NotEqual(t, salaryID, r["id"],
			"匿名访问者不得看到受限分类知识：%+v", r["title"])
	}
}

func TestE2E_RAG_PublicKnowledgeStillSearchable(t *testing.T) {
	client, rag, ctx, tenantID, employeeID, _ := setupKnowledgeAccessE2E(t)
	defer client.Close()

	// 关键回归：纳管一个分类后，未纳管的公开知识必须照常可检索
	viewerCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: employeeID, Role: "end_user"})
	results, err := rag.Ask(viewerCtx, tenantID, "VPN", 20)
	require.NoError(t, err)

	require.NotEmpty(t, results, "公开知识不应受分类纳管影响")
	require.Equal(t, "VPN 连接配置指南", results[0]["title"])
}

func TestE2E_RAG_AuthorCanReadOwnRestrictedArticle(t *testing.T) {
	client, rag, ctx, tenantID, _, salaryID := setupKnowledgeAccessE2E(t)
	defer client.Close()

	// 作者本人（CFO）即便没有财务权限，也应能检索到自己写的文章。
	// 作者 ID 直接从文章上取（夹具里作者就是 CFO）。
	articles, err := client.KnowledgeArticle.Get(ctx, salaryID)
	require.NoError(t, err)
	cfoID := articles.AuthorID

	authorCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: cfoID, Role: "end_user"})
	results, err := rag.Ask(authorCtx, tenantID, "薪酬", 20)
	require.NoError(t, err)

	found := false
	for _, r := range results {
		if r["id"] == salaryID {
			found = true
		}
	}
	require.True(t, found, "作者本人应能检索到自己写的受限分类文章")
}

func TestE2E_RAG_CrossTenantIsolation(t *testing.T) {
	client, rag, ctx, tenantID, employeeID, _ := setupKnowledgeAccessE2E(t)
	defer client.Close()

	// 另一个租户的公开知识，本租户用户不应看到
	otherTenant, err := client.Tenant.Create().
		SetName("other").SetCode("other").SetDomain("other.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	_, err = client.KnowledgeArticle.Create().
		SetTitle("VPN 其他租户配置").SetContent("配置 VPN 客户端需要导入证书").
		SetCategory("故障处理").SetAuthorID(employeeID).
		SetTenantID(otherTenant.ID).SetIsPublished(true).Save(ctx)
	require.NoError(t, err)

	viewerCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: employeeID, Role: "end_user"})
	results, err := rag.Ask(viewerCtx, tenantID, "VPN", 20)
	require.NoError(t, err)

	for _, r := range results {
		require.NotEqual(t, "VPN 其他租户配置", r["title"], "跨租户知识不得泄漏")
	}
}
