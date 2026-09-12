package incident

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件覆盖 GET /api/v1/incidents 的筛选契约，走真实 handler + production
// service + Ent 仓储。
//
// 修复前 handler 只解析 status/priority/keyword，其余条件被静默丢弃，但页面
// 已经在用（事件列表页的「来源」、NOC 大屏的 isMajorIncident），且 scope=me
// 写入 filters["assignee_id"] 后仓储层从不读取，因此这些筛选全部返回全量：
//   - source / type / category / assigneeId / isMajorIncident / dateFrom / dateTo
//     必须真正下推到查询。
//   - isMajorIncident=false 是合法筛选，不得因为假值而被当成「不过滤」。
//   - scope=me 只能使用认证上下文里的 user_id，不接受查询参数注入。

type listContractFixture struct {
	client  *ent.Client
	handler *IncidentHandler
	tenantA int
	tenantB int
	agentA  int
	agentA2 int
	agentB  int
}

func newListContractFixture(t *testing.T) *listContractFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:incident_list_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().
		SetName("ListA").SetCode("list-a").SetDomain("a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().
		SetName("ListB").SetCode("list-b").SetDomain("b.test").SaveX(ctx)

	mkUser := func(name string, tenantID int) int {
		return client.User.Create().
			SetUsername(name).SetName(name).SetEmail(name + "@example.com").
			SetPasswordHash("hash").SetTenantID(tenantID).SaveX(ctx).ID
	}

	return &listContractFixture{
		client: client,
		handler: NewHandler(NewService(NewEntRepository(client),
			service.NewIncidentService(client, zap.NewNop().Sugar(), nil),
			nil, nil, nil, zap.NewNop().Sugar())),
		tenantA: tenantA.ID,
		tenantB: tenantB.ID,
		agentA:  mkUser("agent-a", tenantA.ID),
		agentA2: mkUser("agent-a2", tenantA.ID),
		agentB:  mkUser("agent-b", tenantB.ID),
	}
}

type incidentSeed struct {
	number       string
	source       string
	category     string
	incidentType string
	assignee     int
	major        bool
	createdAt    time.Time
}

func (f *listContractFixture) seed(t *testing.T, tenantID int, in incidentSeed) {
	t.Helper()
	ctx := context.Background()
	create := f.client.Incident.Create().
		SetTitle("列表契约 " + in.number).
		SetDescription("d").
		SetStatus("open").
		SetPriority("medium").
		SetSeverity("medium").
		SetIncidentNumber(in.number).
		SetReporterID(f.agentA).
		SetTenantID(tenantID)
	if in.source != "" {
		create.SetSource(in.source)
	}
	if in.category != "" {
		create.SetCategory(in.category)
	}
	if in.incidentType != "" {
		create.SetType(in.incidentType)
	}
	if in.assignee != 0 {
		create.SetAssigneeID(in.assignee)
	}
	if in.major {
		create.SetIsMajorIncident(true)
	}
	if !in.createdAt.IsZero() {
		create.SetCreatedAt(in.createdAt)
	}
	create.SaveX(ctx)
}

// doList 以 admin 角色发起列表查询，返回 HTTP status 与已解析的响应体，
// 保证断言能看到业务 code 而不是只看到「响应非空」。
func (f *listContractFixture) doList(t *testing.T, query string, tenantID, actorID int) (int, map[string]interface{}) {
	t.Helper()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", actorID)
		c.Set("role", "admin")
	})
	r.GET("/api/v1/incidents", f.handler.Lists)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/incidents?"+query, nil))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body
}

func (f *listContractFixture) listData(t *testing.T, query string, tenantID, actorID int) map[string]interface{} {
	t.Helper()
	status, body := f.doList(t, query, tenantID, actorID)
	require.Equal(t, http.StatusOK, status, body["message"])
	require.Equal(t, float64(0), body["code"], body["message"])
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是列表信封，实际 %v", body)
	return data
}

func (f *listContractFixture) listTotal(t *testing.T, query string) int {
	t.Helper()
	total, ok := f.listData(t, query, f.tenantA, f.agentA)["total"].(float64)
	require.True(t, ok, "data.total 缺失")
	return int(total)
}

func (f *listContractFixture) listNumbersAs(t *testing.T, query string, actorID int) []string {
	t.Helper()
	items, ok := f.listData(t, query, f.tenantA, actorID)["items"].([]interface{})
	require.True(t, ok, "data.items 缺失")

	numbers := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]interface{})
		require.True(t, ok)
		number, _ := item["incidentNumber"].(string)
		numbers = append(numbers, number)
	}
	return numbers
}

// baseSeed 建 5 条覆盖各维度的租户 A 事件。
func (f *listContractFixture) baseSeed(t *testing.T) {
	t.Helper()
	f.seed(t, f.tenantA, incidentSeed{
		number: "INC-MON-MAJOR", source: "monitoring", category: "hardware",
		incidentType: "infrastructure", assignee: f.agentA, major: true,
		createdAt: time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC),
	})
	f.seed(t, f.tenantA, incidentSeed{
		number: "INC-MON-PLAIN", source: "monitoring", category: "software",
		incidentType: "application", assignee: f.agentA,
		createdAt: time.Date(2024, 8, 20, 0, 0, 0, 0, time.UTC),
	})
	f.seed(t, f.tenantA, incidentSeed{
		number: "INC-MAN-PLAIN", source: "manual", category: "hardware",
		incidentType: "infrastructure", assignee: f.agentA2,
		createdAt: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
	})
	f.seed(t, f.tenantA, incidentSeed{
		number: "INC-EMAIL-MAJOR", source: "email", category: "network",
		incidentType: "infrastructure", assignee: f.agentA2, major: true,
		createdAt: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	f.seed(t, f.tenantA, incidentSeed{
		number: "INC-NOSRC", assignee: f.agentA,
		createdAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	})
}

func TestList_HonorsCamelCaseSourceFilter(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 修复前 source 被丢弃，这里会拿到全部 5 条。
	assert.ElementsMatch(t, []string{"INC-MON-MAJOR", "INC-MON-PLAIN"}, f.listNumbersAs(t, "source=monitoring", f.agentA))
	assert.Equal(t, 1, f.listTotal(t, "source=email"))
	assert.Equal(t, 5, f.listTotal(t, ""), "未传筛选时必须返回全量")
}

func TestList_HonorsMajorIncidentFilter(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// NOC 大屏依赖该过滤：修复前所谓「重大事件」列表其实是所有事件。
	assert.Equal(t, 2, f.listTotal(t, "isMajorIncident=true"))
	// false 是合法筛选值，不得被当成「未传」。
	assert.Equal(t, 3, f.listTotal(t, "isMajorIncident=false"))

	status, body := f.doList(t, "isMajorIncident=abc", f.tenantA, f.agentA)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, float64(1001), body["code"])
}

func TestList_HonorsAssigneeAndScopeMe(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.ElementsMatch(t, []string{"INC-MON-MAJOR", "INC-MON-PLAIN", "INC-NOSRC"},
		f.listNumbersAs(t, fmt.Sprintf("assigneeId=%d", f.agentA), f.agentA))

	// scope=me 必须取认证上下文里的 user_id：同一条 query 换 actor 结果不同。
	assert.ElementsMatch(t, []string{"INC-MON-MAJOR", "INC-MON-PLAIN", "INC-NOSRC"},
		f.listNumbersAs(t, "scope=me", f.agentA))
	assert.ElementsMatch(t, []string{"INC-MAN-PLAIN", "INC-EMAIL-MAJOR"},
		f.listNumbersAs(t, "scope=me", f.agentA2))

	// 其他租户的 user id 不得放宽可见范围。
	assert.Equal(t, 0, f.listTotal(t, fmt.Sprintf("assigneeId=%d", f.agentB)))
}

func TestList_HonorsCategoryAndType(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.Equal(t, 2, f.listTotal(t, "category=hardware"))
	assert.Equal(t, 1, f.listTotal(t, "category=network&type=infrastructure"))
	assert.Equal(t, 3, f.listTotal(t, "type=infrastructure"))
}

func TestList_HonorsCreatedAtRange(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	assert.ElementsMatch(t, []string{"INC-MON-MAJOR", "INC-MON-PLAIN", "INC-NOSRC"},
		f.listNumbersAs(t, "dateFrom=2024-01-01&dateTo=2024-12-31", f.agentA))
	assert.Equal(t, 2, f.listTotal(t, "dateFrom=2024-01-01T00:00:00Z&dateTo=2024-06-30T23:59:59Z"))

	status, body := f.doList(t, "dateFrom=not-a-date", f.tenantA, f.agentA)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, float64(1001), body["code"])
}

func TestList_FiltersStayWithinTenant(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)
	// 租户 B 也有一条 monitoring 事件。
	f.seed(t, f.tenantB, incidentSeed{number: "INC-B-MON", source: "monitoring"})

	assert.ElementsMatch(t, []string{"INC-MON-MAJOR", "INC-MON-PLAIN"},
		f.listNumbersAs(t, "source=monitoring", f.agentA))

	// 从租户 B 发起同样查询，只能看到自己的那条。
	data := f.listData(t, "source=monitoring", f.tenantB, f.agentB)
	assert.Equal(t, float64(1), data["total"])
}
