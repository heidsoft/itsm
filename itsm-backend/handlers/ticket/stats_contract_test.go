package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"
)

func TestTicketStatsCountsOverdueWithinTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_stats_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	for _, tid := range []int{1, 2} {
		suffix := strconv.Itoa(tid)
		tenant := client.Tenant.Create().SetName(suffix).SetCode(suffix).SetDomain(suffix + ".test").SaveX(ctx)
		user := client.User.Create().SetUsername(suffix).SetEmail(suffix + "@example.com").SetName("test").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
		for _, status := range []string{"open", "resolved", "closed", "cancelled"} {
			tk := client.Ticket.Create().SetTitle(status).SetTicketNumber(suffix + status).SetStatus(status).SetRequesterID(user.ID).SetTenantID(tenant.ID).SetSLAResolutionDeadline(time.Now().Add(-time.Hour)).SaveX(ctx)
			// overdue 统计现在以 sla_states 为权威源
			if status == "open" {
				_, err := client.SLAState.Create().
					SetTenantID(tenant.ID).
					SetAggregateType("ticket").
					SetAggregateID(tk.ID).
					SetResolutionDeadline(time.Now().Add(-time.Hour)).
					SetStatus("active").
					Save(ctx)
				require.NoError(t, err)
			}
		}
	}
	h := NewHandler(NewService(NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()), client), nil, nil, zap.NewNop().Sugar()))
	r := gin.New()
	r.GET("/api/v1/tickets/stats", func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		// 本用例断言的是租户全量分布，必须按管理角色调用：stats 与列表共用
		// 行级数据范围，非管理角色只统计本人工单（见 TestTicketStatsScopesToViewerDataRange）。
		c.Set("user_id", 1)
		c.Set("role", "admin")
		h.GetTicketStats(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/stats", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var response struct {
		Code int `json:"code"`
		Data struct {
			Overdue int `json:"overdue"`
			Total   int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Equal(t, 1, response.Data.Overdue)
	require.Equal(t, 4, response.Data.Total)
}

type nameCount struct {
	Name  string
	Count int
}

// statsShape 是 /api/v1/tickets/stats 响应 data 的完整形状。
// 逐字段断言，禁止用“包含某字段”这种宽松校验掩盖分布缺失。
type statsShape struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	Pending      int `json:"pending"`
	HighPriority int `json:"highPriority"`
	Overdue      int `json:"overdue"`
	ByStatus     []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	} `json:"byStatus"`
	ByPriority []struct {
		Priority string `json:"priority"`
		Count    int    `json:"count"`
	} `json:"byPriority"`
	Keys []string
}

func (s statsShape) statuses() []nameCount {
	out := make([]nameCount, 0, len(s.ByStatus))
	for _, item := range s.ByStatus {
		out = append(out, nameCount{Name: item.Status, Count: item.Count})
	}
	return out
}

func (s statsShape) priorities() []nameCount {
	out := make([]nameCount, 0, len(s.ByPriority))
	for _, item := range s.ByPriority {
		out = append(out, nameCount{Name: item.Priority, Count: item.Count})
	}
	return out
}

func sumNameCounts(in []nameCount) int {
	total := 0
	for _, item := range in {
		total += item.Count
	}
	return total
}

// TestTicketStatsDistributionsAreAuthoritative 锁住工单报表分布的权威来源。
//
// 报表不能用 listTickets({pageSize:200}) 在浏览器里自己数：总数会被页长截断，
// 而超时卡片过滤的 status === 'overdue' 在工单状态机里并不存在，读数恒为 0。
// 租户全量的分组计数必须由 stats 输出 CountByStatus/CountByPriority 原 map，
// 只折叠成几个桶就把明细丢掉，等于算完即丢。
func TestTicketStatsDistributionsAreAuthoritative(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_stats_distribution?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	type tenantFixture struct {
		tenantID int
		userID   int
	}
	newTenant := func(code string) tenantFixture {
		tenant := client.Tenant.Create().SetName(code).SetCode(code).SetDomain(code + ".test").SaveX(ctx)
		user := client.User.Create().
			SetUsername(code).
			SetEmail(code + "@example.com").
			SetName("test").
			SetPasswordHash("hash").
			SetTenantID(tenant.ID).
			SaveX(ctx)
		return tenantFixture{tenantID: tenant.ID, userID: user.ID}
	}
	createTicket := func(f tenantFixture, status, priority, number string) {
		client.Ticket.Create().
			SetTitle(number).
			SetTicketNumber(number).
			SetStatus(status).
			SetPriority(priority).
			SetRequesterID(f.userID).
			SetTenantID(f.tenantID).
			SaveX(ctx)
	}

	tenantA := newTenant("stats-a")
	createTicket(tenantA, "open", "low", "A-1")
	createTicket(tenantA, "open", "medium", "A-2")
	createTicket(tenantA, "in_progress", "high", "A-3")
	createTicket(tenantA, "resolved", "critical", "A-4")
	createTicket(tenantA, "closed", "urgent", "A-5")
	createTicket(tenantA, "cancelled", "low", "A-6")
	// 词表外的历史取值：分布必须原样追加，既不丢计数也不静默并进别的桶。
	createTicket(tenantA, "awaiting_vendor", "legacy_blocker", "A-7")

	tenantB := newTenant("stats-b")
	createTicket(tenantB, "open", "high", "B-1")
	createTicket(tenantB, "resolved", "high", "B-2")

	h := NewHandler(NewService(NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()), client), nil, nil, zap.NewNop().Sugar()))

	get := func(tb *testing.T, tenantID int) statsShape {
		tb.Helper()
		r := gin.New()
		r.GET("/api/v1/tickets/stats", func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
			// 同上：本用例锁的是租户全量分布的权威性，按管理角色调用。
			c.Set("user_id", 1)
			c.Set("role", "admin")
			h.GetTicketStats(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/stats", nil))
		require.Equal(tb, 200, w.Code, w.Body.String())

		var envelope struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		require.NoError(tb, json.Unmarshal(w.Body.Bytes(), &envelope))
		require.Zero(tb, envelope.Code)

		var out statsShape
		require.NoError(tb, json.Unmarshal(envelope.Data, &out))

		var keyed map[string]json.RawMessage
		require.NoError(tb, json.Unmarshal(envelope.Data, &keyed))
		names := make([]string, 0, len(keyed))
		for name := range keyed {
			names = append(names, name)
		}
		sort.Strings(names)
		out.Keys = names
		return out
	}

	a := get(t, tenantA.tenantID)
	require.Equal(t, 7, a.Total)
	require.Equal(t, 2, a.Open)
	require.Equal(t, 1, a.InProgress)
	require.Equal(t, 1, a.Resolved)
	require.Equal(t, 1, a.Closed)
	require.Zero(t, a.Pending)
	require.Zero(t, a.Overdue)
	require.Equal(t, 1, a.HighPriority)

	require.Equal(t,
		[]nameCount{{"open", 2}, {"in_progress", 1}, {"resolved", 1}, {"closed", 1}, {"cancelled", 1}, {"awaiting_vendor", 1}},
		a.statuses(),
	)
	require.Equal(t,
		[]nameCount{{"low", 2}, {"medium", 1}, {"high", 1}, {"urgent", 1}, {"critical", 1}, {"legacy_blocker", 1}},
		a.priorities(),
	)
	require.Equal(t, a.Total, sumNameCounts(a.statuses()), "状态分布之和必须等于总数，否则有工单被漏计")
	require.Equal(t, a.Total, sumNameCounts(a.priorities()), "优先级分布之和必须等于总数")

	require.Equal(t,
		[]string{"byPriority", "byStatus", "closed", "highPriority", "inProgress", "open", "overdue", "pending", "resolved", "total"},
		a.Keys,
	)

	// 租户 B 只看见自己的工单：分布不能跨租户泄漏。
	b := get(t, tenantB.tenantID)
	require.Equal(t, 2, b.Total)
	require.Equal(t, []nameCount{{"open", 1}, {"resolved", 1}}, b.statuses())
	require.Equal(t, []nameCount{{"high", 2}}, b.priorities())
}

// TestTicketStatsScopesToViewerDataRange 锁定 stats 与列表使用同一行级数据范围。
//
// 实测缺陷：end_user 的 GET /tickets 只返回本人工单（total 0/1），但同一页面的
// GET /tickets/stats 返回租户全量（total 11），卡片读数把别人的工单也算进自己的
// 工作台，/profile 的「提交工单」同样虚高。统计口径必须由调用者数据范围决定，
// 且缺身份时 fail closed —— 不能因为拿不到 user_id 就退回全量。
func TestTicketStatsScopesToViewerDataRange(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_stats_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("scope").SetCode("scope").SetDomain("scope.test").SaveX(ctx)
	other := client.Tenant.Create().SetName("scope-b").SetCode("scope-b").SetDomain("scope-b.test").SaveX(ctx)
	owner := client.User.Create().SetUsername("owner").SetEmail("owner@scope.test").SetName("owner").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	assignee := client.User.Create().SetUsername("assignee").SetEmail("assignee@scope.test").SetName("assignee").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("stranger").SetEmail("stranger@scope.test").SetName("stranger").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)

	mk := func(number string, requesterID, assigneeID int, tenantID int) {
		create := client.Ticket.Create().
			SetTitle(number).
			SetTicketNumber(number).
			SetStatus("open").
			SetPriority("medium").
			SetRequesterID(requesterID).
			SetTenantID(tenantID)
		if assigneeID > 0 {
			create = create.SetAssigneeID(assigneeID)
		}
		create.SaveX(ctx)
	}
	// 属于 owner：本人创建 1 张 + 受理给 owner 之外的另一人 1 张（不可见）
	mk("S-1", owner.ID, 0, tenant.ID)
	mk("S-2", stranger.ID, assignee.ID, tenant.ID)
	mk("S-3", stranger.ID, 0, tenant.ID)
	// 另一租户 2 张，任何情况下都不该出现在上面的统计里
	mk("X-1", owner.ID, 0, other.ID)
	mk("X-2", stranger.ID, 0, other.ID)

	h := NewHandler(NewService(NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()), client), nil, nil, zap.NewNop().Sugar()))

	get := func(t *testing.T, userID int, role string) int {
		t.Helper()
		r := gin.New()
		r.GET("/api/v1/tickets/stats", func(c *gin.Context) {
			c.Set("tenant_id", tenant.ID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
			c.Set("user_id", userID)
			c.Set("role", role)
			h.GetTicketStats(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/stats", nil))
		require.Equal(t, 200, w.Code, w.Body.String())
		var envelope struct {
			Code int `json:"code"`
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
		require.Zero(t, envelope.Code)
		return envelope.Data.Total
	}

	require.Equal(t, 1, get(t, owner.ID, "end_user"), "端用户只应统计本人创建或受理的工单")
	require.Equal(t, 1, get(t, assignee.ID, "agent"), "受理人同样只看自己的数据范围")
	require.Equal(t, 3, get(t, owner.ID, "admin"), "管理角色可见租户全量（跨租户仍被隔离）")
	require.Zero(t, get(t, 0, ""), "缺少身份上下文必须 fail closed，不得退回全量")
}

func TestOrderedNamesKeepsVocabularyOrderAndAppendsUnknown(t *testing.T) {
	counts := map[string]int{"critical": 1, "open": 0, "zzz_legacy": 2, "high": 3, "medium": 4}
	require.Equal(t, []string{"medium", "high", "critical", "zzz_legacy"}, orderedNames(counts, ticketPriorityOrder))
	require.NotNil(t, orderedNames(map[string]int{}, ticketStatusOrder))
	require.Empty(t, orderedNames(map[string]int{"open": 0}, ticketStatusOrder))
}
