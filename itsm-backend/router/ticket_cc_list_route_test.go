package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	ticketWorkflowHandler "itsm-backend/handlers/ticket_workflow"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-6i）：
//   - GET /api/v1/tickets/cc/my  返回 {records,total}
//   - GET /api/v1/tickets/:id/cc 返回 {records,total}
//
// 两个抄送端点实测都**不分页**（service 用 Where(tenant_id)+All(ctx) 整表取回、
// Total=len(records)，handler 直接透传），所以本批只把集合键收敛为平台规定的 items，
// 并保留诚实的两键形状：**不补 page/pageSize/totalPages**，否则等于用响应声明伪造
// 一个不存在的分页协议（AGENTS「能力状态与失败语义」）。
//
// 同批修掉一处实测漂移：前端类型和页面读 `user.name`，而后端 dto.WorkflowUserInfo
// 从未发出该键（真实键是 fullName），抄送人/添加人两列因此一直落到 username。
// 用例 4 用「精确键集合」锁定它，而不是靠注释声明。
//
// 测试打在**生产注册**上（router/ticket_routes.go:35 与 :226，含
// RequirePermission("workflow","read")）。解码目标是本地 wire 结构而不引用 dto，
// 因此修复前也能编译，证明的是运行时行为。

const (
	ticketCCListSecret = "ticket-cc-list-secret"
	// ccListTicketNumberA 是租户 A 的工单号，同时充当跨租户泄漏的探针标记。
	ccListTicketNumberA = "TKT-CC-LIST-CC-A"
)

// 抄送列表断言用的时间基准；并列排序用例复用同一个 time.Time 值。
var ccListBaseTime = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

type ccListActor struct {
	tenantID int
	userID   int
	username string
	// name 是 Ent user.name，对应 DTO 的 fullName；刻意与 username 不同。
	name string
}

type ccListFixture struct {
	tenantID   int
	admin      ccListActor
	recipients []ccListActor
	// ticketID 承载 ccCount 条抄送，收件人逐个排在 recipients 上。
	ticketID int
}

// ccUserWire 只声明本文件真正断言的字段；item/user 的完整键集合由用例 4 精确锁定。
type ccUserWire struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

type ccRecordWire struct {
	ID           int        `json:"id"`
	TicketID     int        `json:"ticketId"`
	TicketNumber string     `json:"ticketNumber"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	User         ccUserWire `json:"user"`
	AddedBy      ccUserWire `json:"addedBy"`
	AddedAt      time.Time  `json:"addedAt"`
	IsActive     bool       `json:"isActive"`
}

type ccListWire struct {
	Items []ccRecordWire `json:"items"`
	Total int            `json:"total"`
}

func TestTicketCCListRoutesEnvelope(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_ticket_cc_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupTicketCCListRouter(t, client, logger)

	// 租户 A：主工单 3 条抄送，added_at 依次递增（第 3 条最新）。
	tenantA := seedCCListTenant(ctx, t, client, "cc-a", 3)
	// 租户 B：1 条抄送，用于收敛与跨租户探针。
	tenantB := seedCCListTenant(ctx, t, client, "cc-b", 1)
	// 空态租户：admin 没有任何抄送记录。
	emptyTenant := seedCCListTenant(ctx, t, client, "cc-empty", 0)

	doCCAs := func(t *testing.T, actor ccListActor, path string) (*httptest.ResponseRecorder, []byte) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(actor.userID, actor.username, "super_admin", actor.tenantID, ticketCCListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w, w.Body.Bytes()
	}

	// envelopeData 取出信封 data 并确认业务码为成功。
	envelopeData := func(t *testing.T, body []byte) json.RawMessage {
		t.Helper()
		var envelope rawEnvelope
		require.NoError(t, json.Unmarshal(body, &envelope), "body=%s", body)
		require.Equal(t, 0, envelope.Code, "body=%s", body)
		return envelope.Data
	}

	getCCList := func(t *testing.T, actor ccListActor, path string) ccListWire {
		t.Helper()
		w, body := doCCAs(t, actor, path)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
		var data ccListWire
		require.NoError(t, json.Unmarshal(envelopeData(t, body), &data), "body=%s", body)
		return data
	}

	// 1. 两个端点都只返回诚实的两键信封，旧集合键 records 永不存在，且不出现分页键。
	for _, tc := range []struct {
		name string
		path string
	}{
		{"我的抄送", "/api/v1/tickets/cc/my"},
		{"工单抄送", fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID)},
	} {
		t.Run(tc.name+"只返回 items+total 两键", func(t *testing.T) {
			w, body := doCCAs(t, tenantA.admin, tc.path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)

			data := envelopeData(t, body)
			assert.Equal(t, []string{"items", "total"}, envelopeKeys(t, data), "data=%s", data)
			assert.NotContains(t, string(data), `"records":`)
			// 不分页的端点不得出现分页键。
			assert.NotContains(t, string(data), `"page"`)
			assert.NotContains(t, string(data), `"pageSize"`)
			assert.NotContains(t, string(data), `"totalPages"`)
		})
	}

	// 2. total 等于真实条数，且 cc/my 只返回给本人的那一条（不是整张工单的全部抄送）。
	t.Run("total 等于 items 长度且 cc/my 按收件人收敛", func(t *testing.T) {
		byTicket := getCCList(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID))
		assert.Equal(t, 3, byTicket.Total)
		assert.Len(t, byTicket.Items, 3)

		for i, recipient := range tenantA.recipients {
			byUser := getCCList(t, recipient, "/api/v1/tickets/cc/my")
			assert.Equal(t, 1, byUser.Total, "recipient %d", i)
			require.Len(t, byUser.Items, 1, "recipient %d", i)
			assert.Equal(t, tenantA.ticketID, byUser.Items[0].TicketID)
			assert.Equal(t, recipient.userID, byUser.Items[0].User.ID, "cc/my 必须只返回给本人")
			assert.Equal(t, tenantA.admin.userID, byUser.Items[0].AddedBy.ID)
		}
	})

	// 3. 空结果序列化为 []，不是 null（前端按数组渲染）。
	t.Run("空抄送序列化为 []", func(t *testing.T) {
		w, body := doCCAs(t, emptyTenant.admin, "/api/v1/tickets/cc/my")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
		assert.Contains(t, string(body), `"items":[]`)
		assert.NotContains(t, string(body), `"items":null`)

		var data ccListWire
		require.NoError(t, json.Unmarshal(envelopeData(t, body), &data), "body=%s", body)
		assert.Equal(t, 0, data.Total)
		assert.Empty(t, data.Items)
	})

	// 4. item 与 user 的精确键集合：证明后端发出的是 fullName，而前端读过的 name 键永不存在。
	t.Run("item 与 user 键集合精确匹配后端 DTO", func(t *testing.T) {
		w, body := doCCAs(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", body)

		data := envelopeData(t, body)
		var rawItems []json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(jsonOf(t, data, "items")), &rawItems), "data=%s", data)
		require.NotEmpty(t, rawItems)

		assert.Equal(t, []string{
			"addedAt", "addedBy", "id", "isActive", "priority", "status", "ticketId", "ticketNumber", "title", "user",
		}, envelopeKeys(t, rawItems[0]))

		userRaw := []byte(jsonOf(t, rawItems[0], "user"))
		// avatar/department 是 omitempty，此处 fixture 用户没填部门，所以只剩这五个键。
		assert.Equal(t, []string{"email", "fullName", "id", "role", "username"}, envelopeKeys(t, userRaw), "user=%s", userRaw)

		var decoded ccUserWire
		require.NoError(t, json.Unmarshal(userRaw, &decoded), "user=%s", userRaw)
		assert.NotEqual(t, decoded.Username, decoded.FullName, "fixture 必须能区分 fullName 与 username")
	})

	// 5. fullName 由 service 用 user.Name 回填：抄送人列是收件人姓名，添加人列是发起人姓名。
	t.Run("user.fullName 与 addedBy.fullName 分别来自双方姓名", func(t *testing.T) {
		data := getCCList(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID))
		require.Len(t, data.Items, 3)

		wantNames := make(map[string]bool, len(tenantA.recipients))
		for _, recipient := range tenantA.recipients {
			wantNames[recipient.name] = true
		}

		for _, item := range data.Items {
			assert.True(t, wantNames[item.User.FullName], "抄送人姓名 %q 不属于租户 A 的收件人", item.User.FullName)
			assert.Equal(t, tenantA.admin.name, item.AddedBy.FullName, "添加人姓名")
			assert.Equal(t, tenantA.admin.username, item.AddedBy.Username)
			assert.True(t, item.IsActive)
			assert.Equal(t, ccListTicketNumberA, item.TicketNumber)
			assert.Equal(t, "抄送信封回归", item.Title)
			assert.Equal(t, "open", item.Status)
			assert.Equal(t, "medium", item.Priority)
		}
	})

	// 6. 相同 added_at 必须按 id 升序稳定排列（本次给两条 DESC 排序补的并列键）。
	//
	// 说明：修复前 `Order(ent.Desc(addedAt))` 在并列时间下由 SQLite 随意决定顺序，
	// 实测它恰好返回插入顺序，而插入顺序与自增 id 同向，所以本用例在修复前也会通过。
	// 它是**契约锁**（防止后续改动把顺序变成不确定），不是「修复前转红」的证据；
	// 本片的红转绿证据是用例 1/3/8/9 的 items 集合键与租户收敛。
	t.Run("相同抄送时间按 id 升序稳定排列", func(t *testing.T) {
		tieTicket, err := client.Ticket.Create().
			SetTicketNumber("TKT-CC-TIE").
			SetTitle("抄送并列键回归").
			SetDescription("同一 added_at 的多条抄送必须有确定顺序").
			SetType("incident").
			SetPriority("medium").
			SetStatus("open").
			SetRequesterID(tenantA.admin.userID).
			SetTenantID(tenantA.tenantID).
			Save(ctx)
		require.NoError(t, err)

		// 收件人乱序写入，保证返回顺序来自排序而非 recipients 数组顺序。
		var createdIDs []int
		for _, idx := range []int{2, 0, 1} {
			row, err := client.TicketCC.Create().
				SetTicketID(tieTicket.ID).
				SetUserID(tenantA.recipients[idx].userID).
				SetAddedBy(tenantA.admin.userID).
				SetTenantID(tenantA.tenantID).
				SetAddedAt(ccListBaseTime).
				SetIsActive(true).
				Save(ctx)
			require.NoError(t, err)
			createdIDs = append(createdIDs, row.ID)
		}
		wantIDs := append([]int(nil), createdIDs...)
		sort.Ints(wantIDs)

		data := getCCList(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tieTicket.ID))
		require.Len(t, data.Items, 3)

		gotIDs := make([]int, 0, len(data.Items))
		for _, item := range data.Items {
			gotIDs = append(gotIDs, item.ID)
			// 并列键用例的前提：三条时间戳完全相同。
			assert.True(t, item.AddedAt.Equal(ccListBaseTime), "addedAt=%s", item.AddedAt)
		}
		assert.Equal(t, wantIDs, gotIDs, "相同 added_at 必须按 id 升序")

		// 重复请求顺序一致，证明调用方拿到的是确定性列表。
		repeat := getCCList(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tieTicket.ID))
		require.Len(t, repeat.Items, 3)
		for i, item := range repeat.Items {
			assert.Equal(t, gotIDs[i], item.ID, "重复请求顺序漂移")
		}
	})

	// 7. 主工单按 added_at DESC：最新抄送排在首位（与用例 6 的并列键分工不同）。
	t.Run("抄送按 added_at 倒序", func(t *testing.T) {
		data := getCCList(t, tenantA.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID))
		require.Len(t, data.Items, 3)
		// 最新一条（base+3h）必须在首位，而不是只断言「非递增」。
		assert.True(t, data.Items[0].AddedAt.Equal(ccListBaseTime.Add(3*time.Hour)), "首位=%s", data.Items[0].AddedAt)
		for i := 1; i < len(data.Items); i++ {
			assert.False(t, data.Items[i-1].AddedAt.Before(data.Items[i].AddedAt),
				"items[%d].addedAt=%s 早于 items[%d].addedAt=%s", i-1, data.Items[i-1].AddedAt, i, data.Items[i].AddedAt)
		}
	})

	// 8. 租户收敛：B 的 cc/my 与 B 自己工单的抄送里都没有 A 的数据。
	t.Run("租户 B 只看到自己的抄送", func(t *testing.T) {
		require.Len(t, tenantB.recipients, 1)
		cases := []struct {
			actor ccListActor
			path  string
		}{
			{tenantB.recipients[0], "/api/v1/tickets/cc/my"},
			{tenantB.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantB.ticketID)},
		}
		for _, tc := range cases {
			w, body := doCCAs(t, tc.actor, tc.path)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", body)
			assert.NotContains(t, string(body), ccListTicketNumberA, tc.path)
			assert.NotContains(t, string(body), "cc-a-admin", tc.path)

			var data ccListWire
			require.NoError(t, json.Unmarshal(envelopeData(t, body), &data), "body=%s", body)
			assert.Equal(t, []string{"items", "total"}, envelopeKeys(t, envelopeData(t, body)), tc.path)
			assert.Equal(t, 1, data.Total, tc.path)
			require.Len(t, data.Items, 1, tc.path)
			assert.Equal(t, tenantB.ticketID, data.Items[0].TicketID, tc.path)
		}
	})

	// 9. 跨租户探测必须 fail closed 且不泄漏对方抄送。
	//
	// 不锁定具体 status/code：service 返回的是 common.NewBusinessError(NotFoundCode, ...)，
	// 但 handler 用 common.FailWithErr 透传，实测固定映射成 HTTP 500 + code 5001，
	// 把「工单不存在」伪装成内部错误。这属于 E4-29 登记的错误语义债（与全站 286 处
	// err.Error()/7 处 AppError 的分类收敛一起在 E4-5 拍板），本片不顺手改语义，
	// 因此只断言「非 2xx + 零泄漏」，等错误分类收敛后再补精确 status。
	t.Run("跨租户工单抄送不泄漏", func(t *testing.T) {
		w, body := doCCAs(t, tenantB.admin, fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID))
		assert.GreaterOrEqual(t, w.Code, http.StatusBadRequest, "body=%s", body)
		assert.NotContains(t, string(body), ccListTicketNumberA, "跨租户不得返回对方工单号")
		assert.NotContains(t, string(body), "cc-a-admin", "跨租户不得返回对方抄送人")
		assert.NotContains(t, string(body), `"items"`, "跨租户不得返回抄送列表载荷")
	})

	// 10. 未认证不得返回列表载荷。
	t.Run("未认证不得返回列表载荷", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/tickets/cc/my",
			fmt.Sprintf("/api/v1/tickets/%d/cc", tenantA.ticketID),
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, path)
			assert.NotContains(t, w.Body.String(), `"items"`, path)
		}
	})
}

func setupTicketCCListRouter(t *testing.T, client *ent.Client, logger *zap.SugaredLogger) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:             ticketCCListSecret,
		Logger:                logger,
		Client:                client,
		TicketWorkflowHandler: ticketWorkflowHandler.NewHandler(service.NewTicketWorkflowService(client, logger), nil, logger),
	})
	return r
}

// seedCCListTenant 建租户 + 一名发起抄送的 super_admin + ccCount 名收件人，
// 并建一张归属该租户的工单，把工单逐个抄送给收件人（added_at 依次递增 1 小时）。
// 用户名与姓名刻意不同（姓名-<username>），用于断言前端读的是 fullName。
func seedCCListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, ccCount int) ccListFixture {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("CC list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	fixture := ccListFixture{tenantID: tenant.ID}
	fixture.admin = createCCListActor(ctx, t, client, tenant.ID, code+"-admin")

	ticket, err := client.Ticket.Create().
		SetTicketNumber(fmt.Sprintf("TKT-CC-LIST-%s", strings.ToUpper(code))).
		SetTitle("抄送信封回归").
		SetDescription("验证工单抄送两个不分页信封的集合键与并列排序键").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(fixture.admin.userID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	fixture.ticketID = ticket.ID

	if code == "cc-a" {
		require.Equal(t, ccListTicketNumberA, ticket.TicketNumber, "探针标记必须与常量一致")
	}

	for i := 0; i < ccCount; i++ {
		recipient := createCCListActor(ctx, t, client, tenant.ID, fmt.Sprintf("%s-cc%d", code, i))
		fixture.recipients = append(fixture.recipients, recipient)

		_, err := client.TicketCC.Create().
			SetTicketID(ticket.ID).
			SetUserID(recipient.userID).
			SetAddedBy(fixture.admin.userID).
			SetTenantID(tenant.ID).
			SetAddedAt(ccListBaseTime.Add(time.Duration(i+1) * time.Hour)).
			SetIsActive(true).
			Save(ctx)
		require.NoError(t, err)
	}

	return fixture
}

func createCCListActor(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username string) ccListActor {
	t.Helper()

	created, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName("姓名-" + username).
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	return ccListActor{
		tenantID: tenantID,
		userID:   created.ID,
		username: username,
		name:     created.Name,
	}
}
