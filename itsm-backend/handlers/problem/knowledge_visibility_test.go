package problem

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/internal/knowledgeaccess"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubKnowledgeSink 只实现反查所需行为；Create 在反查用例中不会被调用。
type stubKnowledgeSink struct {
	articles []*knowledge.Article
}

func (s *stubKnowledgeSink) Create(ctx context.Context, a *knowledge.Article) (*knowledge.Article, error) {
	return a, nil
}

func (s *stubKnowledgeSink) ListBySourceProblem(ctx context.Context, tenantID, problemID int) ([]*knowledge.Article, error) {
	return s.articles, nil
}

func restrictProblemKnowledgeCategory(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, category string) {
	t.Helper()
	_, err := client.Permission.Create().
		SetCode("knowledge_category_problem_" + category).
		SetName("知识分类-" + category).
		SetResource(knowledgeaccess.CategoryResource).
		SetAction(knowledgeaccess.ActionForCategory(category)).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
}

// 问题页反查沉淀知识是知识读面的旁路：受限分类必须同样不可见。
func TestListProblemKnowledge_FiltersRestrictedCategory(t *testing.T) {
	client, svc, ctx := setupProblemHandlerTest(t)
	defer client.Close()

	tenant := createProblemHandlerTenant(t, ctx, client, "kb-vis")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "kb-vis")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	restrictProblemKnowledgeCategory(t, ctx, client, tenant.ID, "财务制度")
	svc.SetKnowledgeGuard(knowledgeaccess.NewGuard(client, svc.logger))
	svc.SetKnowledgeRepository(&stubKnowledgeSink{articles: []*knowledge.Article{
		{ID: 1, Title: "VPN 指南", Category: "故障处理", AuthorID: user.ID, TenantID: tenant.ID},
		{ID: 2, Title: "薪酬方案", Category: "财务制度", AuthorID: user.ID + 1000, TenantID: tenant.ID},
	}})

	readerCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: user.ID, Role: "agent"})
	articles, err := svc.ListProblemKnowledge(readerCtx, tenant.ID, p.ID)
	require.NoError(t, err)
	require.Len(t, articles, 1)
	assert.Equal(t, "VPN 指南", articles[0].Title)

	// 作者本人仍可读自己沉淀的受限文章。
	authorCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: user.ID + 1000, Role: "agent"})
	articles, err = svc.ListProblemKnowledge(authorCtx, tenant.ID, p.ID)
	require.NoError(t, err)
	assert.Len(t, articles, 2)

	adminCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: 1, Role: "super_admin"})
	articles, err = svc.ListProblemKnowledge(adminCtx, tenant.ID, p.ID)
	require.NoError(t, err)
	assert.Len(t, articles, 2)
}
