package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func mustCreateTenant(t *testing.T, ctx context.Context, client *ent.Client, code string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(code).
		SetCode(code).
		SetDomain(code + ".local").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID
}

// decodeTicketItems 按标准信封解出 data.items，保留原始键集合，
// 这样「姓名缺失」与「带了不该出现的 PII 字段」都能被断言。
func decodeTicketItems(t *testing.T, body []byte) []map[string]interface{} {
	t.Helper()
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]interface{} `json:"items"`
			Total int                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Zero(t, envelope.Code)
	return envelope.Data.Items
}

// assertNoUserObjects 固定住这次收口的契约：工单响应只投影显示名，
// 不再返回 requester/assignee 对象，也不带出 username/email/phone/role。
func assertNoUserObjects(t *testing.T, item map[string]interface{}) {
	t.Helper()
	for _, key := range []string{"requester", "assignee", "username", "email", "phone", "role"} {
		require.NotContains(t, item, key, "工单响应不应带出用户对象或 PII 字段")
	}
}

// TestTicketNameProjectionIsTenantScoped 走真实仓储与路由：
// 同租户 ID 投影姓名，跨租户 ID 只回 ID，未派工既无 ID 也无姓名。
func TestTicketNameProjectionIsTenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_name_projection?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenantA := mustCreateTenant(t, ctx, client, "np-a")
	tenantB := mustCreateTenant(t, ctx, client, "np-b")
	alice := mustCreateUser(t, ctx, client, tenantA, "np-alice")
	bob := mustCreateUser(t, ctx, client, tenantA, "np-bob")
	outsider := mustCreateUser(t, ctx, client, tenantB, "np-outsider")

	seedScopeTicket(t, ctx, client, tenantA, "NP-1", alice.ID, &bob.ID)
	seedScopeTicket(t, ctx, client, tenantA, "NP-2", alice.ID, &outsider.ID)
	seedScopeTicket(t, ctx, client, tenantA, "NP-3", alice.ID, nil)

	h := NewHandler(NewService(
		NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()), client),
		nil, nil, zap.NewNop().Sugar()))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantA})
		c.Set("user_id", alice.ID)
		c.Set("role", "admin")
		c.Next()
	})
	api.GET("/tickets", h.ListTickets)
	api.GET("/tickets/:id", h.GetTicket)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?page=1&pageSize=50", nil, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	items := decodeTicketItems(t, w.Body.Bytes())
	require.Len(t, items, 3)

	byNumber := make(map[string]map[string]interface{}, len(items))
	for _, item := range items {
		number, _ := item["ticketNumber"].(string)
		byNumber[number] = item
		assertNoUserObjects(t, item)
	}

	assigned := byNumber["NP-1"]
	require.Equal(t, float64(bob.ID), assigned["assigneeId"])
	require.Equal(t, "np-bob", assigned["assigneeName"])
	require.Equal(t, "np-alice", assigned["requesterName"])

	// 处理人 ID 属于另一个租户：ID 照原样返回，姓名必须投影不到。
	crossTenant := byNumber["NP-2"]
	require.Equal(t, float64(outsider.ID), crossTenant["assigneeId"])
	require.NotContains(t, crossTenant, "assigneeName", "跨租户用户 ID 不得投影出姓名")

	unassigned := byNumber["NP-3"]
	require.NotContains(t, unassigned, "assigneeId")
	require.NotContains(t, unassigned, "assigneeName")

	detailID := strconv.FormatFloat(assigned["id"].(float64), 'f', 0, 64)
	detail := doJSON(t, r, http.MethodGet, "/api/v1/tickets/"+detailID, nil, nil)
	require.Equal(t, http.StatusOK, detail.Code, detail.Body.String())
	var detailEnvelope struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &detailEnvelope))
	require.Zero(t, detailEnvelope.Code)
	assertNoUserObjects(t, detailEnvelope.Data)
	require.Equal(t, "np-bob", detailEnvelope.Data["assigneeName"])
	require.Equal(t, "np-alice", detailEnvelope.Data["requesterName"])
}

// TestTicketListUsesOneDedupedNameQuery 钉住投影的调用形状：一页工单只查一次，
// 请求 ID 去重；未登记在租户目录里的 ID 不回姓名，由前端回退显示用户 ID。
func TestTicketListUsesOneDedupedNameQuery(t *testing.T) {
	r, repo := newTestHarness(t)
	repo.userNames[1] = map[int]string{7: "爱丽丝", 8: "鲍勃"}
	repo.userNames[2] = map[int]string{7: "跨租户爱丽丝"}

	ctx := context.Background()
	assigneeBob := 8
	assigneeUnknown := 9
	for _, params := range []*CreateParams{
		{Title: "A", Priority: "low", Type: "incident", RequesterID: 7, AssigneeID: &assigneeBob},
		{Title: "B", Priority: "low", Type: "incident", RequesterID: 7, AssigneeID: &assigneeUnknown},
		{Title: "C", Priority: "low", Type: "incident", RequesterID: 7},
	} {
		p := params
		_, err := repo.Create(ctx, p, 1)
		require.NoError(t, err)
	}

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?page=1&pageSize=50", nil, map[string]string{
		"X-Test-TenantID": "1",
		"X-Test-UserID":   "1",
		"X-Test-Role":     "admin",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, repo.namesCalls, "一页工单只能触发一次姓名查询")
	require.Equal(t, 1, repo.lastNamesTenant, "姓名查询必须按当前租户")
	got := append([]int(nil), repo.lastNamesIDs...)
	sort.Ints(got)
	require.Equal(t, []int{7, 8, 9}, got, "报障人/处理人 ID 必须去重后一次查询")

	items := decodeTicketItems(t, w.Body.Bytes())
	require.Len(t, items, 3)
	names := map[float64]string{}
	for _, item := range items {
		assertNoUserObjects(t, item)
		require.Equal(t, "爱丽丝", item["requesterName"])
		if assigneeID, ok := item["assigneeId"].(float64); ok {
			name, _ := item["assigneeName"].(string)
			names[assigneeID] = name
		}
	}
	require.Equal(t, "鲍勃", names[8])
	require.Empty(t, names[9], "租户目录里没有的 ID 不得伪造姓名")
}
