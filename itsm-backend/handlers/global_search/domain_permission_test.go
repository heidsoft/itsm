package global_search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGlobalSearchDomainPermissionFilter 验证全局搜索按域读权限裁剪：
// 搜索结果带标题与描述正文，不是「有就无」的元数据——没有某域读权限的角色
// 不该搜到该域内容。
//
// 反向证明：把 allowedDomains 退化成恒真，本用例必须转红。
func TestGlobalSearchDomainPermissionFilter(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:gs_domain_filter?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	tenantID, userID := createSearchTenantAndUser(t, client, "domain")
	// 每个域一条可搜到的记录
	createSearchTicket(t, client, tenantID, userID, "关键词工单", "描述")
	createSearchIncident(t, client, tenantID, userID, "关键词事件", "描述")
	createSearchKnowledge(t, client, tenantID, userID, "关键词知识", "描述", true)
	createSearchKnowledge(t, client, tenantID, userID, "关键词草稿", "描述", false)

	ticketReader := seedSearchRole(t, client, tenantID, "gs-ticket-reader", []string{"ticket:read"})

	t.Run("只持 ticket:read 只搜到工单域", func(t *testing.T) {
		resp, err := NewService(client).Search(ctx, tenantID, "关键词", []string{ticketReader})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Results, "有 ticket:read 至少应搜到工单")
		for _, r := range resp.Results {
			assert.Equal(t, "ticket", r.Type, "无权域不得出现在结果里（type=%s）", r.Type)
		}
	})

	t.Run("无任何域权限返回空结果", func(t *testing.T) {
		noPerm := seedSearchRole(t, client, tenantID, "gs-noperm", nil)
		resp, err := NewService(client).Search(ctx, tenantID, "关键词", []string{noPerm})
		require.NoError(t, err)
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("草稿知识不可被搜出（未发布内容不外泄）", func(t *testing.T) {
		knowledgeReader := seedSearchRole(t, client, tenantID, "gs-knowledge-reader", []string{"knowledge:read"})
		resp, err := NewService(client).Search(ctx, tenantID, "关键词", []string{knowledgeReader})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Results, "已发布知识应可搜到")
		for _, r := range resp.Results {
			assert.Equal(t, "knowledge", r.Type)
			assert.NotEqual(t, "关键词草稿", r.Title, "未发布（草稿）文章不得出现在搜索结果")
			assert.Equal(t, "published", r.Status)
		}
	})

	t.Run("多角色并集可见两域", func(t *testing.T) {
		incidentReader := seedSearchRole(t, client, tenantID, "gs-incident-reader", []string{"incident:read"})
		resp, err := NewService(client).Search(ctx, tenantID, "关键词", []string{ticketReader, incidentReader})
		require.NoError(t, err)
		types := map[string]bool{}
		for _, r := range resp.Results {
			types[r.Type] = true
		}
		assert.True(t, types["ticket"])
		assert.True(t, types["incident"])
		assert.False(t, types["knowledge"], "两角色都不持 knowledge:read")
	})

	t.Run("super_admin 全域可见且草稿仍不外泄", func(t *testing.T) {
		resp, err := NewService(client).Search(ctx, tenantID, "关键词", []string{"super_admin"})
		require.NoError(t, err)
		types := map[string]bool{}
		for _, r := range resp.Results {
			types[r.Type] = true
			assert.NotEqual(t, "关键词草稿", r.Title, "草稿不对任何角色外泄")
		}
		assert.True(t, types["ticket"])
		assert.True(t, types["incident"])
		assert.True(t, types["knowledge"])
	})
}

// TestGlobalSearchRouteRequiresPermission 锁路由层的显式权限声明：
// 全局搜索此前无任何 RequirePermission，RBAC 中间件只能按路径推断出
// global-search:read——该码不在 authz 词表、无人持有，DBOnly 下即除
// super_admin 外全员 403，功能实际不可用。
func TestGlobalSearchRouteRequiresPermission(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:gs_route_perm?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	tenantID, _ := createSearchTenantAndUser(t, client, "routeperm")

	// 一个不持有 dashboard:read 的角色
	reader := seedSearchRole(t, client, tenantID, "gs-route-reader", []string{"ticket:read"})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("role", reader)
		c.Set("user_id", 1)
		c.Set("tenant_id", tenantID)
		c.Set("client", client)
		c.Next()
	})
	NewHandler(NewService(client)).RegisterRoutes(router.Group("/api/v1"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/global-search?q=x", nil))
	assert.Equal(t, http.StatusForbidden, rec.Code, "无 dashboard:read 应 403: body=%s", rec.Body.String())
}

// createSearchIncident 建一条可搜到的事件。
func createSearchIncident(t *testing.T, client *ent.Client, tenantID, reporterID int, title, description string) *ent.Incident {
	t.Helper()
	item, err := client.Incident.Create().
		SetTitle(title).
		SetDescription(description).
		SetIncidentNumber(fmt.Sprintf("IN-%d-%s", tenantID, title)).
		SetReporterID(reporterID).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return item
}

// createSearchKnowledge 建一条知识文章（published 决定是否对外可见）。
func createSearchKnowledge(t *testing.T, client *ent.Client, tenantID, authorID int, title, content string, published bool) *ent.KnowledgeArticle {
	t.Helper()
	item, err := client.KnowledgeArticle.Create().
		SetTitle(title).
		SetContent(content).
		SetAuthorID(authorID).
		SetTenantID(tenantID).
		SetIsPublished(published).
		Save(context.Background())
	require.NoError(t, err)
	return item
}

// seedSearchRole 建角色并授予权限码，返回角色 code。
func seedSearchRole(t *testing.T, client *ent.Client, tenantID int, roleCode string, permCodes []string) string {
	t.Helper()
	ctx := context.Background()
	role, err := client.Role.Create().
		SetName(roleCode).
		SetCode(roleCode).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	for _, code := range permCodes {
		resource, action := splitPermCode(t, code)
		perm, err := client.Permission.Create().
			SetCode(code).
			SetName(code).
			SetResource(resource).
			SetAction(action).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.RolePermission.Create().
			SetRoleID(role.ID).
			SetPermissionID(perm.ID).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}
	return roleCode
}

func splitPermCode(t *testing.T, code string) (string, string) {
	t.Helper()
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			return code[:i], code[i+1:]
		}
	}
	t.Fatalf("权限码格式应为 resource:action，实际 %q", code)
	return "", ""
}
