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
	"itsm-backend/service"
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

// statsWindowShape/statsTrendShape/statsSLAShape 是带窗口请求才会出现的契约段。
type statsWindowShape struct {
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
	Days     int    `json:"days"`
}

type statsTrendShape struct {
	Date     string `json:"date"`
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

type statsSLAShape struct {
	ComplianceRate     float64 `json:"complianceRate"`
	SampledTickets     int     `json:"sampledTickets"`
	ViolatedTickets    int     `json:"violatedTickets"`
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	AvgResolutionHours float64 `json:"avgResolutionHours"`
	WindowDays         int     `json:"windowDays"`
	Scope              string  `json:"scope"`
}

// statsShape 是 /api/v1/tickets/stats 响应 data 的完整形状。
// 逐字段断言，禁止用“包含某字段”这种宽松校验掩盖分布缺失。
type statsShape struct {
	Total        int               `json:"total"`
	Open         int               `json:"open"`
	InProgress   int               `json:"inProgress"`
	Resolved     int               `json:"resolved"`
	Closed       int               `json:"closed"`
	Pending      int               `json:"pending"`
	HighPriority int               `json:"highPriority"`
	Overdue      int               `json:"overdue"`
	Scope        string            `json:"scope"`
	Window       *statsWindowShape `json:"window"`
	DailyTrend   []statsTrendShape `json:"dailyTrend"`
	SLA          *statsSLAShape    `json:"sla"`
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
		[]string{"byPriority", "byStatus", "closed", "highPriority", "inProgress", "open", "overdue", "pending", "resolved", "scope", "total"},
		a.Keys,
	)
	// 未带日期时不返回 window/dailyTrend/sla：口径必须能被前端区分，
	// 空数组和 0 会被读成「窗口内确实没有工单」。
	require.Nil(t, a.Window, "未传日期不得回显窗口")
	require.Nil(t, a.DailyTrend, "未传日期不得返回趋势数组")
	require.Nil(t, a.SLA, "未传日期不得返回 SLA 读数")
	require.Equal(t, "all", a.Scope, "管理角色统计口径必须是租户全量")

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

func timePtr(v time.Time) *time.Time { return &v }

// statsEnvelope 是一次 /tickets/stats 请求的完整结果：HTTP 状态、业务码与 data。
// 契约测试必须三者都断言，只看 data 会把 400 的空 body 读成「统计为 0」。
type statsEnvelope struct {
	httpStatus int
	code       int
	message    string
	data       statsShape
}

// TestTicketStatsWindowedContract 锁住带窗口的 /tickets/stats 契约。
//
// 合并工单仪表盘后，这个端点是工单统计页的唯一数据源：窗口、趋势、SLA 三段
// 必须由它一起给出。任何一段回落到第二个数据源，同一屏就会重新出现两种口径
// （卡片是本人可见 12 张、趋势图是租户全量 300 张）。
func TestTicketStatsWindowedContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_stats_window?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("win-a").SetCode("win-a").SetDomain("win-a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("win-b").SetCode("win-b").SetDomain("win-b.test").SaveX(ctx)
	alice := client.User.Create().SetUsername("win-alice").SetEmail("alice@win.test").SetName("alice").SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)
	bob := client.User.Create().SetUsername("win-bob").SetEmail("bob@win.test").SetName("bob").SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)
	carol := client.User.Create().SetUsername("win-carol").SetEmail("carol@win.test").SetName("carol").SetPasswordHash("hash").SetTenantID(tenantB.ID).SaveX(ctx)
	slaDef := client.SLADefinition.Create().SetName("win-sla").SetResponseTime(30).SetResolutionTime(240).SetTenantID(tenantA.ID).SaveX(ctx)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	label := today.Format("2006-01-02")

	mk := func(number string, tenantID, requesterID int, createdAt time.Time, resolvedAt *time.Time) int {
		create := client.Ticket.Create().
			SetTitle(number).
			SetTicketNumber(number).
			SetStatus("open").
			SetPriority("medium").
			SetRequesterID(requesterID).
			SetTenantID(tenantID).
			SetCreatedAt(createdAt)
		if resolvedAt != nil {
			create = create.SetStatus("resolved").SetResolvedAt(*resolvedAt)
		}
		return create.SaveX(ctx).ID
	}

	// 今天 2 张（alice 已解决 + bob 未解决），8 天前 1 张落在窗口外，
	// 另 1 张属于租户 B：任何一条都不该被窗口的 2 张覆盖或泄漏。
	resolvedToday := mk("WIN-1", tenantA.ID, alice.ID, today.Add(time.Hour), timePtr(today.Add(2*time.Hour)))
	mk("WIN-2", tenantA.ID, bob.ID, today.Add(3*time.Hour), nil)
	mk("WIN-3", tenantA.ID, alice.ID, today.AddDate(0, 0, -8), nil)
	mk("WIN-B", tenantB.ID, carol.ID, today.Add(time.Hour), nil)

	client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(resolvedToday).
		SetViolationType("resolution").
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	h := NewHandler(NewService(
		NewEntRepository(ticketrepo.NewEntRepository(client, zap.NewNop().Sugar()), client),
		nil,
		service.NewSLAMonitorService(client, zap.NewNop().Sugar()),
		zap.NewNop().Sugar(),
	))

	get := func(tb *testing.T, query string, tenantID, userID int, role string) statsEnvelope {
		tb.Helper()
		r := gin.New()
		r.GET("/api/v1/tickets/stats", func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
			c.Set("user_id", userID)
			c.Set("role", role)
			h.GetTicketStats(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/tickets/stats?"+query, nil))

		var envelope struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		require.NoError(tb, json.Unmarshal(w.Body.Bytes(), &envelope))
		out := statsEnvelope{httpStatus: w.Code, code: envelope.Code, message: envelope.Message}
		if w.Code == http.StatusOK {
			require.NoError(tb, json.Unmarshal(envelope.Data, &out.data))
		}
		return out
	}

	t.Run("窗口内计数器/分布/趋势共用同一口径", func(t *testing.T) {
		got := get(t, "dateFrom="+label+"&dateTo="+label, tenantA.ID, alice.ID, "admin")
		require.Equal(t, http.StatusOK, got.httpStatus, got.message)
		require.Zero(t, got.code)
		require.Equal(t, 2, got.data.Total, "窗口外的历史工单不得计入")
		require.Equal(t, "all", got.data.Scope)
		require.NotNil(t, got.data.Window)
		require.Equal(t, statsWindowShape{DateFrom: label, DateTo: label, Days: 1}, *got.data.Window)

		// 分布之和 == 窗口总数：两侧谓词不一致时这里必然对不上。
		require.Equal(t, got.data.Total, sumNameCounts(got.data.statuses()))
		require.Equal(t, got.data.Total, sumNameCounts(got.data.priorities()))

		require.Len(t, got.data.DailyTrend, 1)
		require.Equal(t, statsTrendShape{Date: label, Created: 2, Resolved: 1}, got.data.DailyTrend[0])
	})

	t.Run("SLA 读数随窗口返回并标注租户口径", func(t *testing.T) {
		got := get(t, "dateFrom="+label+"&dateTo="+label, tenantA.ID, alice.ID, "admin")
		require.NotNil(t, got.data.SLA, "带窗口的请求必须给出 SLA 读数，否则页面只能继续硬编码 0")
		require.Equal(t, 2, got.data.SLA.SampledTickets, "SLA 抽样只数窗口内新建的工单，且不跨租户")
		require.Equal(t, 1, got.data.SLA.ViolatedTickets)
		require.InDelta(t, 50.0, got.data.SLA.ComplianceRate, 0.01)
		require.Equal(t, "tenant", got.data.SLA.Scope, "SLA 是租户口径，页面必须把它和统计范围一起显示")
		require.Equal(t, 1, got.data.SLA.WindowDays)
	})

	t.Run("不传日期保持全量口径且不返回窗口段", func(t *testing.T) {
		got := get(t, "", tenantA.ID, alice.ID, "admin")
		require.Equal(t, http.StatusOK, got.httpStatus, got.message)
		require.Equal(t, 3, got.data.Total)
		require.Nil(t, got.data.Window)
		require.Nil(t, got.data.DailyTrend)
		require.Nil(t, got.data.SLA)
	})

	t.Run("窗口跟随行级数据范围", func(t *testing.T) {
		got := get(t, "dateFrom="+label+"&dateTo="+label, tenantA.ID, alice.ID, "end_user")
		require.Equal(t, http.StatusOK, got.httpStatus, got.message)
		require.Equal(t, "owner", got.data.Scope)
		require.Equal(t, 1, got.data.Total, "bob 今天建的工单不能出现在 alice 的统计里")
		require.Len(t, got.data.DailyTrend, 1)
		require.Equal(t, 1, got.data.DailyTrend[0].Created)
	})

	t.Run("窗口统计不跨租户", func(t *testing.T) {
		got := get(t, "dateFrom="+label+"&dateTo="+label, tenantB.ID, carol.ID, "admin")
		require.Equal(t, http.StatusOK, got.httpStatus, got.message)
		// 租户 A 的 3 张（含窗口内 2 张）都不得进租户 B 的读数，只剩 B 自己的 WIN-B。
		require.Equal(t, 1, got.data.Total)
		require.Len(t, got.data.DailyTrend, 1)
		require.Equal(t, 1, got.data.DailyTrend[0].Created)
		require.Zero(t, got.data.DailyTrend[0].Resolved)
		require.NotNil(t, got.data.SLA)
		require.Equal(t, 1, got.data.SLA.SampledTickets, "SLA 抽样同样按租户收窄")
		require.Zero(t, got.data.SLA.ViolatedTickets)
	})

	t.Run("窗口参数非法返回 400 而不是静默回落", func(t *testing.T) {
		yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
		tooWide := today.AddDate(-2, 0, 0).Format("2006-01-02")
		cases := []struct {
			name    string
			query   string
			message string
		}{
			{"只传 dateFrom", "dateFrom=" + label, "dateFrom 与 dateTo 必须同时提供"},
			{"只传 dateTo", "dateTo=" + label, "dateFrom 与 dateTo 必须同时提供"},
			{"区间倒置", "dateFrom=" + label + "&dateTo=" + yesterday, "dateFrom 不能晚于 dateTo"},
			{"格式非法", "dateFrom=2026%2F01%2F01&dateTo=" + label, "dateFrom 格式错误"},
			{"跨度过大", "dateFrom=" + tooWide + "&dateTo=" + label, "统计窗口最长 366 天"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := get(t, tc.query, tenantA.ID, alice.ID, "admin")
				require.Equal(t, http.StatusBadRequest, got.httpStatus, tc.query)
				require.Equal(t, 1001, got.code)
				require.Contains(t, got.message, tc.message)
			})
		}
	})
}

// TestResolveStatsWindowBoundaries 锁住窗口边界的两处易错点：
// 上界必须覆盖当日最后一刻（List 用 LTE 比较 created_at），
// 都不传日期必须返回 nil 而不是默认 30 天——否则工作台卡片会在无人改筛选时变小。
func TestResolveStatsWindowBoundaries(t *testing.T) {
	now := time.Date(2026, 11, 1, 15, 4, 5, 0, time.Local)

	missing, err := ResolveStatsWindow("", "", now)
	require.NoError(t, err)
	require.Nil(t, missing, "未传日期不得补默认窗口")

	window, err := ResolveStatsWindow("2026-11-01", "2026-11-01", now)
	require.NoError(t, err)
	require.NotNil(t, window)
	require.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local), window.From)
	require.Equal(t, "2026-11-01 23:59:59", window.To.Format("2006-01-02 15:04:05"),
		"上界必须含当日整天，否则今天 00:00 之后创建的工单会被 LTE 排除")

	echo := StatsWindowEcho(window)
	require.Equal(t, &TicketStatsWindow{DateFrom: "2026-11-01", DateTo: "2026-11-01", Days: 1}, echo)

	// RFC3339 带时区也要归一到本地整日，与 YYYY-MM-DD 同口径。
	rfc, err := ResolveStatsWindow("2026-10-31T00:00:00Z", "2026-11-01T12:00:00Z", now)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local), rfc.From)
	require.Equal(t, "2026-11-01", StatsWindowEcho(rfc).DateTo)
	require.Equal(t, 2, StatsWindowEcho(rfc).Days)
}
