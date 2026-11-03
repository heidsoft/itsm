package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/permission"
	"itsm-backend/internal/knowledgeaccess"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 知识分类可见性回归夹具：一篇公开、一篇受限、一篇无分类，
// 受限分类经真实纳管入口（SetCategoryRestriction）写入 RBAC permission 表。
type visibilityFixture struct {
	client       *ent.Client
	svc          *Service
	ctx          context.Context
	tenantID     int
	authorID     int
	readerID     int
	openID       int
	secretID     int
	unfiledID    int
	restrictName string
}

func newVisibilityFixture(t *testing.T) *visibilityFixture {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("kb-vis").SetCode("kb-vis-" + t.Name()).SetDomain("kb.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewService(NewEntRepository(client), logger)
	svc.SetEntClient(client)
	svc.SetKnowledgeGuard(knowledgeaccess.NewGuard(client, logger))

	const (
		authorID = 1001
		readerID = 1002
	)
	open, err := svc.CreateArticle(ctx, &Article{
		Title: "VPN 连接配置指南", Content: "导入证书即可",
		Category: "故障处理", AuthorID: authorID, TenantID: tenant.ID, IsPublished: true,
	})
	require.NoError(t, err)
	secret, err := svc.CreateArticle(ctx, &Article{
		Title: "2026 年薪酬结构调整方案", Content: "薪酬带宽属机密",
		Category: "财务制度", AuthorID: authorID, TenantID: tenant.ID, IsPublished: true,
	})
	require.NoError(t, err)
	unfiled, err := svc.CreateArticle(ctx, &Article{
		Title: "放假通知", Content: "国庆放假安排",
		AuthorID: authorID, TenantID: tenant.ID, IsPublished: true,
	})
	require.NoError(t, err)

	require.NoError(t, svc.SetCategoryRestriction(ctx, tenant.ID, "财务制度", true))

	return &visibilityFixture{
		client: client, svc: svc, ctx: ctx, tenantID: tenant.ID,
		authorID: authorID, readerID: readerID,
		openID: open.ID, secretID: secret.ID, unfiledID: unfiled.ID,
		restrictName: "财务制度",
	}
}

func (f *visibilityFixture) readerCtx() context.Context {
	return knowledgeaccess.WithViewer(f.ctx, knowledgeaccess.Viewer{UserID: f.readerID, Role: "end_user"})
}

func (f *visibilityFixture) titles(t *testing.T, ctx context.Context) []string {
	t.Helper()
	articles, _, err := f.svc.ListArticles(ctx, f.tenantID, 1, 20, "", "", "")
	require.NoError(t, err)
	titles := make([]string, 0, len(articles))
	for _, a := range articles {
		titles = append(titles, a.Title)
	}
	return titles
}

// 列表先按可见性收窄再分页：total 必须等于访问者真正能看到的条数，
// 不能先算全量再在内存里剔除。
func TestKnowledgeVisibility_ListHidesRestrictedCategory(t *testing.T) {
	f := newVisibilityFixture(t)

	articles, total, err := f.svc.ListArticles(f.readerCtx(), f.tenantID, 1, 1, "", "", "")
	require.NoError(t, err)
	require.Equal(t, 2, total, "受限分类不得计入 total")
	require.Len(t, articles, 1)

	_, filteredTotal, err := f.svc.ListArticles(f.readerCtx(), f.tenantID, 1, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, 2, filteredTotal)

	assert.NotContains(t, f.titles(t, f.readerCtx()), "2026 年薪酬结构调整方案")

	// 显式按受限分类过滤也不能绕过
	_, categoryTotal, err := f.svc.ListArticles(f.readerCtx(), f.tenantID, 1, 20, f.restrictName, "", "")
	require.NoError(t, err)
	assert.Zero(t, categoryTotal)
}

func TestKnowledgeVisibility_AuthorReadsOwnRestrictedArticle(t *testing.T) {
	f := newVisibilityFixture(t)

	authorCtx := knowledgeaccess.WithViewer(f.ctx, knowledgeaccess.Viewer{UserID: f.authorID, Role: "end_user"})
	_, total, err := f.svc.ListArticles(authorCtx, f.tenantID, 1, 20, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, 3, total, "作者豁免必须让本人看到自己写的受限文章")

	got, err := f.svc.GetArticle(authorCtx, f.secretID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, f.secretID, got.ID)
}

func TestKnowledgeVisibility_RoleGrantReadsRestrictedArticle(t *testing.T) {
	f := newVisibilityFixture(t)

	role, err := f.client.Role.Create().
		SetName("知识读者").SetCode("kb_reader").SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)
	perm, err := f.client.Permission.Query().
		Where(
			permission.TenantIDEQ(f.tenantID),
			permission.ResourceEQ(knowledgeaccess.CategoryResource),
			permission.ActionEQ(knowledgeaccess.ActionForCategory(f.restrictName)),
		).
		Only(f.ctx)
	require.NoError(t, err)
	_, err = f.client.RolePermission.Create().
		SetRoleID(role.ID).SetPermissionID(perm.ID).SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)

	granted := knowledgeaccess.WithViewer(f.ctx, knowledgeaccess.Viewer{UserID: f.readerID, Role: "kb_reader"})
	got, err := f.svc.GetArticle(granted, f.secretID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, f.secretID, got.ID)
}

func TestKnowledgeVisibility_DetailAndManagePathsDenyRestrictedArticle(t *testing.T) {
	f := newVisibilityFixture(t)
	ctx := f.readerCtx()

	_, err := f.svc.GetArticle(ctx, f.secretID, f.tenantID)
	require.ErrorIs(t, err, ErrArticleNotFound)

	cases := map[string]func() error{
		"update": func() error {
			_, err := f.svc.UpdateArticle(ctx, &Article{
				ID: f.secretID, TenantID: f.tenantID, Title: "被篡改的标题", Content: "x",
			})
			return err
		},
		"publish": func() error {
			_, err := f.svc.SetArticlePublished(ctx, f.secretID, f.tenantID, false)
			return err
		},
		"review-submit": func() error {
			_, err := f.svc.SubmitReview(ctx, f.secretID, f.tenantID, f.readerID, "")
			return err
		},
		"review-history": func() error {
			_, _, err := f.svc.ListReviews(ctx, f.secretID, f.tenantID, 1, 20)
			return err
		},
		"versions": func() error {
			_, _, err := f.svc.ListVersions(ctx, f.secretID, f.tenantID, 1, 20)
			return err
		},
		"version-get": func() error {
			_, err := f.svc.GetVersion(ctx, f.secretID, 1, f.tenantID)
			return err
		},
		"version-restore": func() error {
			_, err := f.svc.RestoreVersion(ctx, f.secretID, 1, f.tenantID, f.readerID)
			return err
		},
		"mark-reviewed": func() error {
			_, err := f.svc.MarkArticleReviewed(ctx, f.secretID, f.tenantID)
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, call(), ErrArticleNotFound,
				"受限分类下的写/读附属资源必须与不可见一致")
		})
	}

	// 公开文章不受影响，确认拒绝来自分类归属而不是路径整体失效。
	require.NotEqual(t, f.secretID, f.openID)
	_, err = f.svc.GetArticle(ctx, f.openID, f.tenantID)
	require.NoError(t, err)
}

func TestKnowledgeVisibility_CategoriesAndStatsExcludeRestrictedCategory(t *testing.T) {
	f := newVisibilityFixture(t)

	categories, err := f.svc.GetCategories(f.readerCtx(), f.tenantID)
	require.NoError(t, err)
	assert.NotContains(t, categories, f.restrictName)

	stats, err := f.svc.GetStats(f.readerCtx(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Published)
	for _, c := range stats.Categories {
		assert.NotEqual(t, f.restrictName, c.Name)
	}

	adminCtx := knowledgeaccess.WithViewer(f.ctx, knowledgeaccess.Viewer{UserID: 1, Role: "super_admin"})
	adminStats, err := f.svc.GetStats(adminCtx, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 3, adminStats.Total, "管理员统计应包含受限分类")
}

// 未装配 RAG 时 SearchArticles 退化为列表检索，可见性必须在同一路径上生效。
func TestKnowledgeVisibility_SearchFallbackHidesRestrictedCategory(t *testing.T) {
	f := newVisibilityFixture(t)

	results, err := f.svc.SearchArticles(f.readerCtx(), f.tenantID, "薪酬", "", 20)
	require.NoError(t, err)
	assert.Empty(t, results)

	openResults, err := f.svc.SearchArticles(f.readerCtx(), f.tenantID, "VPN", "", 20)
	require.NoError(t, err)
	require.Len(t, openResults, 1)
	assert.Equal(t, f.openID, openResults[0].ID)
}

// 权限状态不可得时必须失败关闭：API 读面报错，详情按不存在处理。
func TestKnowledgeVisibility_FailsClosedWhenGuardCannotQueryPermissions(t *testing.T) {
	f := newVisibilityFixture(t)
	f.svc.SetKnowledgeGuard(knowledgeaccess.NewGuard(nil, zaptest.NewLogger(t).Sugar()))

	_, _, err := f.svc.ListArticles(f.readerCtx(), f.tenantID, 1, 20, "", "", "")
	require.Error(t, err)

	_, err = f.svc.GetCategories(f.readerCtx(), f.tenantID)
	require.Error(t, err)

	_, err = f.svc.GetStats(f.readerCtx(), f.tenantID)
	require.Error(t, err)

	_, err = f.svc.SearchArticles(f.readerCtx(), f.tenantID, "VPN", "", 20)
	require.Error(t, err)

	_, err = f.svc.GetArticle(f.readerCtx(), f.openID, f.tenantID)
	require.True(t, errors.Is(err, ErrArticleNotFound), "权限未知时不得放行有分类文章")
}

// 仓储层哨兵：权限状态未知时仅放行无分类文章。
func TestKnowledgeVisibility_DenyAllSentinelOnlyAllowsUncategorized(t *testing.T) {
	f := newVisibilityFixture(t)

	repo := NewEntRepository(f.client)
	articles, total, err := repo.List(f.ctx, f.tenantID, 1, 20, "", "", "", Visibility{
		DeniedCategories: []string{knowledgeaccess.DenyAllSentinel},
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, articles, 1)
	assert.Equal(t, f.unfiledID, articles[0].ID)
}

// 生产入口契约：Router 经过 Handler 时，user_id/role 必须进入 Viewer，
// 否则分类守卫会把所有请求当匿名处理（全放行或全拒绝都不可接受）。
func TestHandler_ListArticles_HidesRestrictedCategoryForReader(t *testing.T) {
	f := newVisibilityFixture(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(f.svc)
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", f.tenantID)
		c.Set("user_id", f.readerID)
		c.Set("role", "end_user")
	})
	r.GET("/api/v1/knowledge/articles", h.ListArticles)
	r.GET("/api/v1/knowledge/articles/:id", h.GetArticle)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/articles?page=1&pageSize=20", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID       int    `json:"id"`
				Title    string `json:"title"`
				Category string `json:"category"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, 0, body.Code)
	assert.Equal(t, 2, body.Data.Total)
	for _, item := range body.Data.Items {
		assert.NotEqual(t, f.restrictName, item.Category)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/knowledge/articles/%d", f.secretID), nil))
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
