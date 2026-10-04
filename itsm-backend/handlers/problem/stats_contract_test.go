package problem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
)

type problemNameCount struct {
	Name  string
	Count int
}

// problemStatsShape 是 GET /api/v1/problems/stats 响应 data 的完整形状。
// 逐字段断言，禁止用「包含某字段」的宽松校验掩盖分布缺失。
type problemStatsShape struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	HighPriority int `json:"highPriority"`
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

func (s problemStatsShape) statuses() []problemNameCount {
	out := make([]problemNameCount, 0, len(s.ByStatus))
	for _, item := range s.ByStatus {
		out = append(out, problemNameCount{Name: item.Status, Count: item.Count})
	}
	return out
}

func (s problemStatsShape) priorities() []problemNameCount {
	out := make([]problemNameCount, 0, len(s.ByPriority))
	for _, item := range s.ByPriority {
		out = append(out, problemNameCount{Name: item.Priority, Count: item.Count})
	}
	return out
}

func sumProblemNameCounts(in []problemNameCount) int {
	total := 0
	for _, item := range in {
		total += item.Count
	}
	return total
}

// TestProblemStatsDistributionsAreAuthoritative 锁住问题报表分布的权威来源。
// 修复前 /reports/problem-efficiency 用 listProblems({page:1,pageSize:100}) 在浏览器里
// 自己数：第 100 条之后的问题被静默截断，页面却把结果当全量读数展示；identified 从来
// 不在任何单值桶里，因此只有分布能证明「总数对得上」。
func TestProblemStatsDistributionsAreAuthoritative(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:problem_stats_distribution?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	newTenant := func(code string) (int, int) {
		tenant := client.Tenant.Create().
			SetName(code).
			SetCode(code).
			SetDomain(code + ".test").
			SetStatus("active").
			SaveX(ctx)
		user := client.User.Create().
			SetUsername(code).
			SetEmail(code + "@example.com").
			SetName("test").
			SetPasswordHash("hash").
			SetTenantID(tenant.ID).
			SaveX(ctx)
		return tenant.ID, user.ID
	}
	createProblem := func(tenantID, userID int, status, priority, number string) *ent.Problem {
		return client.Problem.Create().
			SetTitle(number).
			SetProblemNumber(number).
			SetStatus(status).
			SetPriority(priority).
			SetCreatedBy(userID).
			SetTenantID(tenantID).
			SaveX(ctx)
	}

	tenantA, userA := newTenant("problem-stats-a")
	createProblem(tenantA, userA, "open", "low", "PRB-A-1")
	createProblem(tenantA, userA, "open", "medium", "PRB-A-2")
	createProblem(tenantA, userA, "investigating", "high", "PRB-A-3")
	createProblem(tenantA, userA, "in_progress", "high", "PRB-A-4")
	createProblem(tenantA, userA, "identified", "critical", "PRB-A-5")
	createProblem(tenantA, userA, "resolved", "low", "PRB-A-6")
	createProblem(tenantA, userA, "closed", "medium", "PRB-A-7")
	// 词表外的历史取值必须原样追加，既不丢计数也不静默并进别的桶。
	createProblem(tenantA, userA, "awaiting_vendor", "legacy_blocker", "PRB-A-8")
	// 软删除的问题不参与统计，也不得进入分布。
	softDeleted := createProblem(tenantA, userA, "resolved", "high", "PRB-A-9")
	require.NoError(t, client.Problem.UpdateOne(softDeleted).SetDeletedAt(time.Now()).Exec(ctx))

	tenantB, userB := newTenant("problem-stats-b")
	createProblem(tenantB, userB, "open", "high", "PRB-B-1")
	createProblem(tenantB, userB, "resolved", "high", "PRB-B-2")

	handler := NewHandler(NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar()))

	get := func(tb *testing.T, tenantID int) problemStatsShape {
		tb.Helper()
		r := gin.New()
		r.GET("/api/v1/problems/stats", func(c *gin.Context) {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
			handler.GetStats(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/stats", nil))
		require.Equal(tb, 200, w.Code, w.Body.String())

		var envelope struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		require.NoError(tb, json.Unmarshal(w.Body.Bytes(), &envelope))
		require.Zero(tb, envelope.Code)

		var out problemStatsShape
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

	a := get(t, tenantA)
	require.Equal(t, 8, a.Total, "软删除的问题不得进入统计")
	require.Equal(t, 2, a.Open)
	require.Equal(t, 2, a.InProgress)
	require.Equal(t, 1, a.Resolved)
	require.Equal(t, 1, a.Closed)
	require.Equal(t, 3, a.HighPriority, "high+critical 折叠口径保持与修复前一致")

	require.Equal(t,
		[]problemNameCount{
			{"open", 2},
			{"investigating", 1},
			{"identified", 1},
			{"in_progress", 1},
			{"resolved", 1},
			{"closed", 1},
			{"awaiting_vendor", 1},
		},
		a.statuses(),
	)
	require.Equal(t,
		[]problemNameCount{
			{"low", 2},
			{"medium", 2},
			{"high", 2},
			{"critical", 1},
			{"legacy_blocker", 1},
		},
		a.priorities(),
	)
	require.Equal(t, a.Total, sumProblemNameCounts(a.statuses()), "状态分布之和必须等于总数")
	require.Equal(t, a.Total, sumProblemNameCounts(a.priorities()), "优先级分布之和必须等于总数")

	require.Equal(t,
		[]string{"byPriority", "byStatus", "closed", "highPriority", "inProgress", "open", "resolved", "total"},
		a.Keys,
	)

	// 租户 B 只看见自己的问题：分布不能跨租户泄漏。
	b := get(t, tenantB)
	require.Equal(t, 2, b.Total)
	require.Equal(t, []problemNameCount{{"open", 1}, {"resolved", 1}}, b.statuses())
	require.Equal(t, []problemNameCount{{"high", 2}}, b.priorities())
}

// TestProblemStatsFailsClosedWithoutTenantContext 锁住缺失租户上下文时的语义：
// 必须是认证失败（2001），不得回落到默认租户或返回空成功。
func TestProblemStatsFailsClosedWithoutTenantContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:problem_stats_failclosed?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	handler := NewHandler(NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar()))
	r := gin.New()
	r.GET("/api/v1/problems/stats", func(c *gin.Context) { handler.GetStats(c) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/stats", nil))

	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 2001, envelope.Code, w.Body.String())
}

func TestOrderedVocabularyKeepsOrderAndAppendsUnknown(t *testing.T) {
	counts := map[string]int{"critical": 1, "open": 0, "zzz_legacy": 2, "high": 3, "medium": 4}
	require.Equal(t, []string{"medium", "high", "critical", "zzz_legacy"}, orderedVocabulary(counts, problemPriorityOrder))
	require.NotNil(t, orderedVocabulary(map[string]int{}, problemStatusOrder))
	require.Empty(t, orderedVocabulary(map[string]int{"open": 0}, problemStatusOrder))
}
