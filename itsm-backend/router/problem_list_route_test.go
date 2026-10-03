package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/problem"
	problemHandler "itsm-backend/handlers/problem"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-6c）：GET /api/v1/problems 的分页有**三个所有者**，
// 每个对同一个非法入参给不同答案——
//   - handler 把 ShouldBindQuery 的原值直接交给 service/repo；
//   - repository_impl.go 自己按 page<1→1、size<1→10、size>200→200 夹紧后才写 Offset/Limit；
//   - 响应的分页元数据在 handler 末尾又按 page<1→1、pageSize<1→10 手夹一次，
//     上限完全不管，totalPages 用未夹紧的原值手算。
//
// 实测后果不是「返回多了」而是**翻页丢数据**：205 条数据配 pageSize=250 时 SQL 只按 200 取，
// 响应却用未夹紧的原值声明 pageSize:250、totalPages=1——按声明契约翻页的客户端翻完第 1 页就停，
// 第 201~205 条在任何一页里都拿不到。因此这里锁的是「SQL 侧与声明侧同一个真相」。
//
// 同一用例还有两套集合键：响应写 problems，前端却有 3 份类型声明（problem-api 的
// problems+items?、problem-service 的 problems、types/biz 的 problems?+items?）和
// ProblemList.tsx 的 resp.problems || resp.items || [] fallback。按 docs/api-reference.md
// 收口为 data.items 单键，并在本批删掉那四个从未被后端使用的排序/时间参数
// （sortBy/sortOrder/dateFrom/dateTo 在 handler 与 repo 里都不参与查询，属假契约）。
//
// 测试打在**生产注册**上（SetupRoutes → SetupProblemRoutes → RequirePermission("problem","read")），
// 并锁住三条既有正确行为：行级 DataScope、租户收敛、未认证 401。

const problemListSecret = "problem-list-secret"

// problemListFixedCreatedAt 让全部种子的 created_at 相同：排序只有 created_at 时，
// 同一秒创建的问题在页边界的归属不确定，翻页会重/漏。
var problemListFixedCreatedAt = time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

func TestProblemListRouteEnvelopeAndPagination(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_problem_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	handler := problemHandler.NewHandler(problemHandler.NewService(problemHandler.NewEntRepository(client), logger))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:      problemListSecret,
		Logger:         logger,
		Client:         client,
		ProblemHandler: handler,
	})

	// 租户 A：25 条问题（priority: i<10 high / 10<=i<20 medium / i>=20 low；
	// status: i<15 open / i>=15 resolved；category: i<5 network / 其余 hardware；
	// i<3 的标题带 disk-array 关键词）。i=23、24 由 agent 本人创建、i=10 受理给 agent
	// ⇒ agent 应见 3 条。
	tenantA, adminA, agentA := seedProblemListTenant(ctx, t, client, "prb-a")
	tenantB, adminB, _ := seedProblemListTenant(ctx, t, client, "prb-b")
	// 租户 C 只为了证明「静默截断」：205 条数据下，修复前有一批行永远翻不到。
	tenantC, adminC := seedProblemListOverflowTenant(ctx, t, client, "prb-c", 205)

	allA := problemIDs(ctx, t, client, tenantA)
	require.Len(t, allA, 25)

	do := func(t *testing.T, userID int, username, role string, tenantID int, query string) (int, string, problemListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, username, role, tenantID, problemListSecret, time.Hour)
		require.NoError(t, err)

		path := "/api/v1/problems"
		if query != "" {
			path += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env problemListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}

	asAdminA := func(t *testing.T, query string) (int, string, problemListEnvelope) {
		return do(t, adminA, "prb-a-admin", "super_admin", tenantA, query)
	}

	t.Run("集合键只有 items，五键齐全且默认页长为 20", func(t *testing.T) {
		status, body, env := asAdminA(t, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data),
			"问题列表必须只返回标准信封五键（修复前是 problems 键）: body=%s", body)

		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize, "缺省页长统一到 common 的 20，不再是 repo 私有的 10")
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Len(t, res.Items, 20)
		assert.Equal(t, allA[:20], problemItemIDs(res.Items), "默认第一页必须是全序的前 20 条")
	})

	t.Run("三页拼接不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			status, body, env := asAdminA(t, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, http.StatusOK, status, body)
			var res problemListWire
			require.NoError(t, json.Unmarshal(env.Data, &res), body)
			assert.Equal(t, page, res.Page, body)
			assert.Equal(t, 10, res.PageSize, body)
			assert.Equal(t, 25, res.Total, body)
			assert.Equal(t, 3, res.TotalPages, body)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want, body)
			paged = append(paged, problemItemIDs(res.Items)...)
		}
		assert.Equal(t, allA, paged, "created_at 并列时必须有 ID 兜底，否则页边界归属不确定")
	})

	t.Run("pageSize=0 回到默认页长", func(t *testing.T) {
		_, body, env := asAdminA(t, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, body)
		assert.Len(t, res.Items, 20, body)
		assert.Equal(t, 2, res.TotalPages, body)
	})

	t.Run("越界页长回落默认页长，SQL 与声明同一个真相", func(t *testing.T) {
		// 修复前：repo 按 200 截断、响应声明 300、totalPages 用 300 手算成 1。
		_, body, env := asAdminA(t, "pageSize=300")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, "越界页长不得原样回显: body=%s", body)
		assert.Len(t, res.Items, 20, body)
		assert.Equal(t, 2, res.TotalPages, body)
	})

	t.Run("越界页长下的翻页不丢数据", func(t *testing.T) {
		// 修复前这一页是空的：Offset 按夹紧后的 200 算，客户端以为上一页已经拿满 300 条。
		_, body, env := asAdminA(t, "page=2&pageSize=300")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 2, res.Page, body)
		assert.Equal(t, allA[20:], problemItemIDs(res.Items), "第 2 页必须是第 21~25 条: body=%s", body)
	})

	t.Run("page 为负数时回到第一页", func(t *testing.T) {
		_, body, env := asAdminA(t, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page, "负 Offset 不得出现在查询里")
		assert.Equal(t, allA[:10], problemItemIDs(res.Items), body)
	})

	t.Run("status/priority/category/keyword 过滤把 total 收敛到匹配条数", func(t *testing.T) {
		cases := []struct {
			name  string
			query string
			total int
		}{
			{"status", "status=resolved&pageSize=100", 10},
			{"priority", "priority=high&pageSize=100", 10},
			{"category", "category=network&pageSize=100", 5},
			{"keyword", "keyword=disk-array&pageSize=100", 3},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, body, env := asAdminA(t, tc.query)
				require.Equal(t, 0, env.Code, body)
				var res problemListWire
				require.NoError(t, json.Unmarshal(env.Data, &res), body)
				assert.Equal(t, tc.total, res.Total, "%s 过滤后 total 必须收敛: body=%s", tc.name, body)
				require.Len(t, res.Items, tc.total, body)
				assert.Equal(t, 1, res.TotalPages, body)
			})
		}
	})

	t.Run("过滤组合与空结果序列化为 []", func(t *testing.T) {
		_, body, env := asAdminA(t, "status=resolved&priority=high&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 0, res.Total, "resolved 只有 i>=15，high 只有 i<10，交集为空: body=%s", body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		assert.Equal(t, 0, res.TotalPages)
	})

	t.Run("未识别参数不再被当成过滤器", func(t *testing.T) {
		// sortBy/dateFrom/sortOrder 从未参与查询（handler 的 filters 里只有四个键），
		// 本批把请求 DTO 里这四个假契约字段删掉；这里锁「传了也不改变结果」。
		_, body, env := asAdminA(t, "pageSize=100&sortBy=title&sortOrder=asc&dateFrom=2020-01-01T00:00:00Z")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 25, res.Total, body)
		assert.Equal(t, allA, problemItemIDs(res.Items), "排序参数不得悄悄改变返回顺序: body=%s", body)
	})

	t.Run("行级数据权限：agent 只看到本人创建或受理的问题", func(t *testing.T) {
		status, body, env := do(t, agentA, "prb-a-agent", "agent", tenantA, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 3, res.Total, "本人创建 2 条 + 受理 1 条，其余 22 条不得出现: body=%s", body)
		for _, item := range res.Items {
			assert.True(t, item.CreatedBy == agentA || (item.AssigneeID != nil && *item.AssigneeID == agentA),
				"越界行: id=%d createdBy=%d assigneeId=%v body=%s", item.ID, item.CreatedBy, derefInt(item.AssigneeID), body)
		}
	})

	t.Run("租户 B 只看到自己的问题", func(t *testing.T) {
		status, body, env := do(t, adminB, "prb-b-admin", "super_admin", tenantB, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 25, res.Total, "每个租户各 25 条，跨租户不得合并")
		assert.Equal(t, problemIDs(ctx, t, client, tenantB), problemItemIDs(res.Items))
		for _, item := range res.Items {
			assert.Equal(t, tenantB, item.TenantID)
		}
	})

	t.Run("越界页长不再让数据翻不到", func(t *testing.T) {
		// 205 条种子 + pageSize=250：修复前 repository 把页长夹到 200，第 1 页只给 200 条，
		// 但响应用未夹紧的原值声明 pageSize:250、totalPages:1——客户端按声明的契约翻完第 1 页
		// 就认为拿全了，第 201~205 条永远不会出现在任何一页里。
		allC := problemIDs(ctx, t, client, tenantC)
		require.Len(t, allC, 205)

		_, body, env := do(t, adminC, "prb-c-admin", "super_admin", tenantC, "page=1&pageSize=250")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, "越界页长回落默认页长，不能回显客户端传来的 250: body=%s", body)
		assert.Equal(t, 205, res.Total, body)
		assert.Equal(t, 11, res.TotalPages, "totalPages 必须按真实页长算: body=%s", body)

		var paged []int
		for page := 1; page <= res.TotalPages; page++ {
			_, pageBody, pageEnv := do(t, adminC, "prb-c-admin", "super_admin", tenantC, fmt.Sprintf("page=%d&pageSize=20", page))
			var pageRes problemListWire
			require.NoError(t, json.Unmarshal(pageEnv.Data, &pageRes), pageBody)
			paged = append(paged, problemItemIDs(pageRes.Items)...)
		}
		assert.Equal(t, allC, paged, "逐页拼接必须覆盖全部 205 条，一条都不能翻不到")
	})

	t.Run("已软删除的问题不出现在列表里", func(t *testing.T) {
		target := allA[0]
		deletedAt := time.Now()
		_, err := client.Problem.UpdateOneID(target).SetDeletedAt(deletedAt).Save(ctx)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := client.Problem.UpdateOneID(target).ClearDeletedAt().Save(ctx)
			require.NoError(t, err)
		})

		_, body, env := asAdminA(t, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res problemListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 24, res.Total, body)
		assert.NotContains(t, problemItemIDs(res.Items), target, body)
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/problems", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})
}

// problemListWire 按线上契约声明解析目标，不引用 dto：锁的是响应形状本身。
type problemListWire struct {
	Items      []problemItemWire `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

type problemItemWire struct {
	ID         int    `json:"id"`
	TenantID   int    `json:"tenantId"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	Category   string `json:"category"`
	CreatedBy  int    `json:"createdBy"`
	AssigneeID *int   `json:"assigneeId"`
}

type problemListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func problemItemIDs(items []problemItemWire) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// seedProblemListTenant 建租户 + super_admin/agent 两个用户 + 25 条问题，布局固定：
// priority 按 i<10 / i<20 / 其余分 high/medium/low，status 在 i>=15 变 resolved，
// category 在 i<5 为 network，i<3 的标题带 disk-array；i=23、24 由 agent 创建、i=10 受理给 agent。
// 全部 created_at 相同，用于逼出排序的 ID 兜底。
func seedProblemListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) (tenantID, adminID, agentID int) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Problem list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName("Problem Admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	agent, err := client.User.Create().
		SetUsername(code + "-agent").
		SetEmail(code + "-agent@example.com").
		SetName("Problem Agent").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < 25; i++ {
		creator := admin.ID
		if i >= 23 {
			creator = agent.ID
		}

		builder := client.Problem.Create().
			SetTitle(fmt.Sprintf("PRB-%s-%02d%s", code, i, diskArrayMarker(i))).
			SetDescription("验证问题列表分页夹紧").
			SetStatus(problemStatusForIndex(i)).
			SetPriority(problemPriorityForIndex(i)).
			SetCategory(problemCategoryForIndex(i)).
			SetCreatedBy(creator).
			SetTenantID(tenant.ID).
			SetCreatedAt(problemListFixedCreatedAt).
			SetUpdatedAt(problemListFixedCreatedAt)
		if i == 10 {
			builder = builder.SetAssigneeID(agent.ID)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}
	return tenant.ID, admin.ID, agent.ID
}

func diskArrayMarker(i int) string {
	if i < 3 {
		return " disk-array"
	}
	return ""
}

func problemStatusForIndex(i int) string {
	if i >= 15 {
		return "resolved"
	}
	return "open"
}

func problemPriorityForIndex(i int) string {
	switch {
	case i < 10:
		return "high"
	case i < 20:
		return "medium"
	default:
		return "low"
	}
}

func problemCategoryForIndex(i int) string {
	if i < 5 {
		return "network"
	}
	return "hardware"
}

// seedProblemListOverflowTenant 建一个大页长越界场景专用的租户：1 个 super_admin + count 条问题。
// 全部同 created_at、同过滤维度，只为逐页拼接的完整性断言服务。
func seedProblemListOverflowTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) (tenantID, adminID int) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Problem list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName("Problem Overflow Admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		_, err := client.Problem.Create().
			SetTitle(fmt.Sprintf("PRB-%s-%03d", code, i)).
			SetDescription("验证问题列表页长越界时的静默截断").
			SetStatus("open").
			SetPriority("medium").
			SetCategory("hardware").
			SetCreatedBy(admin.ID).
			SetTenantID(tenant.ID).
			SetCreatedAt(problemListFixedCreatedAt).
			SetUpdatedAt(problemListFixedCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}
	return tenant.ID, admin.ID
}

// problemIDs 返回某租户问题的完整升序 ID 序列。
func problemIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.Problem.Query().Where(problem.TenantID(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Ints(ids)
	return ids
}
