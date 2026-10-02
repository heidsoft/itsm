package email_intake

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
)

// tiedTime 故意让所有会话共用同一个 last_message_at：并列排序键正是页边界不确定性的来源。
var tiedTime = time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)

// newEmailIntakeClient 每个用例独占一个内存库名，避免 cache=shared 时互相串数据。
func newEmailIntakeClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:email-intake-list-%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

// seedConversations 在租户 1 建 20 条 MANUAL_REVIEW + 5 条 VERIFIED，在租户 2 建 3 条，
// 全部共用 tiedTime。返回租户 1 的 ID（按创建顺序，即 ID 升序）。
func seedConversations(t *testing.T, client *ent.Client) []int {
	t.Helper()
	ctx := context.Background()

	tenantOneIDs := make([]int, 0, 25)
	for i := 1; i <= 25; i++ {
		status := "MANUAL_REVIEW"
		if i > 20 {
			status = "VERIFIED"
		}
		tenantOneIDs = append(tenantOneIDs, client.EmailConversation.Create().
			SetTenantID(1).SetConversationToken(fmt.Sprintf("t1-thread-%02d", i)).
			SetStatus(status).SetLastMessageAt(tiedTime).SaveX(ctx).ID)
	}
	for i := 1; i <= 3; i++ {
		client.EmailConversation.Create().
			SetTenantID(2).SetConversationToken(fmt.Sprintf("t2-thread-%02d", i)).
			SetStatus("MANUAL_REVIEW").SetLastMessageAt(tiedTime).SaveX(ctx)
	}
	return tenantOneIDs
}

// 回归（2026-10-02 边缘功能收口 E3-4）：邮件处理队列页此前只发 {status}，落到后端默认
// pageSize=20，第 21 条起永不可见且没有任何提示。契约必须能把「总量与页」如实交出去。
func TestListConversations_PaginatesAndReportsFullTotal(t *testing.T) {
	client := newEmailIntakeClient(t)
	svc := NewService(client)
	ctx := context.Background()
	all := seedConversations(t, client)

	first, total, err := svc.ListConversations(ctx, 1, "", 1, 20)
	require.NoError(t, err)
	require.Len(t, first, 20, "第一页必须正好 20 条")
	require.Equal(t, 25, total, "total 必须是全量计数，不能是当前页长度")

	second, totalAgain, err := svc.ListConversations(ctx, 1, "", 2, 20)
	require.NoError(t, err)
	require.Len(t, second, 5, "第 21~25 条必须能从第二页拿到")
	require.Equal(t, 25, totalAgain)

	third, _, err := svc.ListConversations(ctx, 1, "", 3, 20)
	require.NoError(t, err)
	require.Empty(t, third, "越界页返回空集合而不是报错，也不得回落到第一页")

	require.ElementsMatch(t, all, append(pageIDs(first), pageIDs(second)...), "两页并起来必须恰好覆盖全集")
}

func TestListConversations_StatusFilterScopesTotal(t *testing.T) {
	client := newEmailIntakeClient(t)
	svc := NewService(client)
	ctx := context.Background()
	seedConversations(t, client)

	items, total, err := svc.ListConversations(ctx, 1, "VERIFIED", 1, 20)
	require.NoError(t, err)
	require.Equal(t, 5, total, "total 要跟着 status 过滤走，否则队列页会把别的状态算进总数")
	require.Len(t, items, 5)
	for _, item := range items {
		require.Equal(t, "VERIFIED", item.Status)
	}
}

func TestListConversations_ExcludesOtherTenantRows(t *testing.T) {
	client := newEmailIntakeClient(t)
	svc := NewService(client)
	ctx := context.Background()
	tenantOne := seedConversations(t, client)

	items, total, err := svc.ListConversations(ctx, 2, "", 1, 20)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, items, 3)
	require.Empty(t, intersect(pageIDs(items), tenantOne), "租户 2 的队列里不得出现租户 1 的会话")
}

// 排序契约固化：并列 last_message_at 时必须按 ID 升序给出确定顺序，逐页拼接等于全集。
// 诚实口径：sqlite 在只按 tied 时间戳排序时恰好按 rowid 返回，本用例构造不出「补并列键前必红」；
// 它锁的是页边界确定性契约，缺并列键时的真实乱序风险由数据库实现决定。
func TestListConversations_DeterministicOrderAcrossPages(t *testing.T) {
	client := newEmailIntakeClient(t)
	svc := NewService(client)
	ctx := context.Background()
	all := seedConversations(t, client)

	var paged []int
	for page := 1; page <= 7; page++ {
		items, _, err := svc.ListConversations(ctx, 1, "", page, 4)
		require.NoError(t, err)
		paged = append(paged, pageIDs(items)...)
	}
	require.Equal(t, all, paged, "每页顺序必须可复现：全部并列时间戳下按 ID 升序翻页")
	require.Len(t, paged, 25)
}

func pageIDs(items []*ent.EmailConversation) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func intersect(a, b []int) []int {
	set := make(map[int]bool, len(b))
	for _, v := range b {
		set[v] = true
	}
	var out []int
	for _, v := range a {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}

// ─── HTTP 契约：真实 RegisterRoutes 链上的信封、租户边界与 fail-closed ──────────

func newEmailIntakeRouter(t *testing.T, client *ent.Client, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// RequirePermission 的上游注入：super_admin 在 AuthorizeResource 里短路放行。
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		c.Set("role", "super_admin")
		c.Set("client", client)
		if tenantID > 0 {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	h := NewHandler(NewService(client))
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func doGet(t *testing.T, r *gin.Engine, path string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "path=%s body=%s", path, w.Body.String())
	return w.Code, envelope
}

func listData(t *testing.T, envelope map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是对象，邮件队列不得再嵌套第二层 data")
	return data
}

func TestListConversationsHTTP_StandardEnvelopeAndServerSidePaging(t *testing.T) {
	client := newEmailIntakeClient(t)
	seedConversations(t, client)
	r := newEmailIntakeRouter(t, client, 1)

	code, envelope := doGet(t, r, "/api/v1/email-intake/conversations?page=2&pageSize=10&status=MANUAL_REVIEW")
	require.Equal(t, http.StatusOK, code)
	require.EqualValues(t, 0, envelope["code"])

	data := listData(t, envelope)
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	require.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, keys,
		"列表信封只允许这 5 个键，此前缺 totalPages")
	require.EqualValues(t, 2, data["page"])
	require.EqualValues(t, 10, data["pageSize"])
	require.EqualValues(t, 20, data["total"], "status 过滤后的全量计数")
	require.EqualValues(t, 2, data["totalPages"])
	require.Len(t, data["items"], 10)

	// 不带分页参数时后端默认第 1 页 20 条：total 仍须如实报 25，
	// 前端才能显式告诉运维「还有未加载的部分」，而不是把第一页当整个队列。
	_, defaults := doGet(t, r, "/api/v1/email-intake/conversations")
	defaultData := listData(t, defaults)
	require.EqualValues(t, 25, defaultData["total"])
	require.EqualValues(t, 2, defaultData["totalPages"])
	require.Len(t, defaultData["items"], 20)
}

func TestListConversationsHTTP_TenantScopedAndRequiresTenantContext(t *testing.T) {
	client := newEmailIntakeClient(t)
	tenantOne := seedConversations(t, client)

	_, envelope := doGet(t, newEmailIntakeRouter(t, client, 2), "/api/v1/email-intake/conversations")
	data := listData(t, envelope)
	require.EqualValues(t, 3, data["total"])
	for _, raw := range data["items"].([]interface{}) {
		item := raw.(map[string]interface{})
		require.NotContains(t, tenantOne, int(item["id"].(float64)), "租户 2 不得读到租户 1 的会话")
	}

	code, envelope := doGet(t, newEmailIntakeRouter(t, client, 0), "/api/v1/email-intake/conversations")
	require.Equal(t, http.StatusUnauthorized, code, "缺少租户上下文必须 fail closed")
	require.NotEqualValues(t, 0, envelope["code"])
}
