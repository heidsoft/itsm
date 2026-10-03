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
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"
	changeHandler "itsm-backend/handlers/change"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-10）：GET /api/v1/changes 的分页参数此前是裸
// strconv.Atoi 且忽略错误，直接交给 repository_impl.go 的 Offset/Limit，而响应的分页
// 元数据在 SuccessWithPagination 里另夹一次——SQL 侧与声明侧对同一个非法入参给两个答案：
//   - pageSize=0 在 Ent 的 sqlgraph 里等于**不加 LIMIT**，实测 25 条种子一次返回 25 条，
//     同时响应写着 pageSize:10；
//   - pageSize=5000 让 SQL 真去取 5000 条（只被数据量兜住），响应却声明 pageSize:100；
//   - page<=0 会算出负 OFFSET，仍被写进 SQL（SQLite 当 0 吞掉，Postgres 拒绝负 OFFSET）。
//
// 同一用例还并存两套风险等级参数名（先读 risk_level、空则读 riskLevel），属 AGENTS
// 「请求/响应多字段兼容零新增」明令禁止的写法；实测前端只发 riskLevel，snake_case 一支
// 从未被使用，因此按「只保留 camelCase」收口，而不是反过来补一个 snake_case 契约。
//
// 测试打在**生产注册**上（SetupRoutes → SetupChangeRoutes → RequirePermission("change","read")），
// 并额外锁住三条既有正确行为：行级 DataScope（agent 只见本人创建/受理）、租户收敛、未认证 401。

const changeListSecret = "change-list-secret"

// changeListFixedCreatedAt 让全部种子的 created_at 相同：排序只有 created_at 时，
// 同一秒创建的变更在页边界的归属不确定，翻页会重/漏。
var changeListFixedCreatedAt = time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

func TestChangeListRoutePaginationAndRiskFilter(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_change_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	repo := changeHandler.NewEntRepository(client, nil)
	handler := changeHandler.NewHandler(changeHandler.NewService(repo, client, logger, nil))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     changeListSecret,
		Logger:        logger,
		Client:        client,
		ChangeHandler: handler,
	})

	// 租户 A：25 条变更（10 条 high / 15 条 medium；20 条 draft / 5 条 scheduled），
	// 其中最后 2 条由 agent 本人创建、另有 1 条受理给 agent，用于行级数据权限断言。
	tenantA, adminA, agentA := seedChangeListTenant(ctx, t, client, "chg-a")
	tenantB, adminB, _ := seedChangeListTenant(ctx, t, client, "chg-b")

	allA := changeIDs(ctx, t, client, tenantA)
	require.Len(t, allA, 25)

	do := func(t *testing.T, userID int, username, role string, tenantID int, query string) (int, string, changeListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, username, role, tenantID, changeListSecret, time.Hour)
		require.NoError(t, err)

		path := "/api/v1/changes"
		if query != "" {
			path += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env changeListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}

	asAdminA := func(t *testing.T, query string) (int, string, changeListEnvelope) {
		return do(t, adminA, "chg-a-admin", "super_admin", tenantA, query)
	}

	t.Run("默认请求返回标准五键并按默认页长截断", func(t *testing.T) {
		status, body, env := asAdminA(t, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data),
			"变更列表必须只返回标准信封五键: body=%s", body)

		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize)
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Len(t, res.Items, 20)
		assert.Equal(t, allA[:20], changeItemIDs(res.Items), "默认第一页必须是全序的前 20 条")
	})

	t.Run("翻页不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			status, body, env := asAdminA(t, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, http.StatusOK, status, body)
			var res changeListWire
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
			paged = append(paged, changeItemIDs(res.Items)...)
		}
		assert.Equal(t, allA, paged, "三页拼接必须等于完整升序序列：created_at 并列也要有确定归属")
	})

	t.Run("pageSize=0 夹紧为默认页长而不是整表返回", func(t *testing.T) {
		_, body, env := asAdminA(t, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, "pageSize=0 在 Ent 里等于不加 LIMIT，必须夹紧后再查")
		assert.Len(t, res.Items, 20, "修复前这条返回全部 25 条")
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("page 为负数时回到第一页", func(t *testing.T) {
		_, body, env := asAdminA(t, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page, "负 Offset 不得出现在查询里，页码必须回落为 1")
		assert.Equal(t, allA[:10], changeItemIDs(res.Items))
	})

	t.Run("越界页长回落默认页长而不是原样下传", func(t *testing.T) {
		_, body, env := asAdminA(t, "pageSize=5000")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, "修复前 5000 被原样接受，一次拉走全租户数据")
		assert.Len(t, res.Items, 20)
	})

	t.Run("riskLevel 过滤把 total 收敛到匹配条数", func(t *testing.T) {
		_, body, env := asAdminA(t, "riskLevel=high&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 10, res.Total, "total 必须是过滤后的数量")
		require.Len(t, res.Items, 10)
		for _, item := range res.Items {
			assert.Equal(t, "high", item.RiskLevel, body)
		}
	})

	t.Run("snake_case 风险参数名不再是第二套契约", func(t *testing.T) {
		// 修复前：risk_level=high 被当成过滤器（total 收敛到 10）。现在只认 riskLevel，
		// 未识别的参数一律忽略，绝不再有两套名字「谁先来算谁」的行为。
		_, body, env := asAdminA(t, "risk_level=high&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 25, res.Total, "risk_level 不得再生效: body=%s", body)
	})

	t.Run("status 过滤与空结果序列化为 []", func(t *testing.T) {
		_, body, env := asAdminA(t, "status=scheduled&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 5, res.Total, body)
		require.Len(t, res.Items, 5, body)
		for _, item := range res.Items {
			assert.Equal(t, "scheduled", item.Status, body)
		}

		_, body, env = asAdminA(t, "status=failed&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		assert.Equal(t, 0, res.Total)
		assert.Equal(t, 0, res.TotalPages)
	})

	t.Run("行级数据权限：agent 只看到本人创建或受理的变更", func(t *testing.T) {
		status, body, env := do(t, agentA, "chg-a-agent", "agent", tenantA, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 3, res.Total, "本人创建 2 条 + 受理 1 条，其余 22 条不得出现: body=%s", body)
		for _, item := range res.Items {
			assert.True(t, item.CreatedBy == agentA || (item.AssigneeID != nil && *item.AssigneeID == agentA),
				"越界行: id=%d createdBy=%d assigneeId=%v body=%s", item.ID, item.CreatedBy, derefInt(item.AssigneeID), body)
		}
	})

	t.Run("租户 B 只看到自己的变更", func(t *testing.T) {
		status, body, env := do(t, adminB, "chg-b-admin", "super_admin", tenantB, "pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		var res changeListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 25, res.Total, "每个租户各 25 条，跨租户不得合并")
		ownB := changeIDs(ctx, t, client, tenantB)
		assert.Equal(t, ownB, changeItemIDs(res.Items))
		for _, item := range res.Items {
			assert.Equal(t, tenantB, item.TenantID)
		}
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/changes", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})
}

// changeListWire 按线上契约声明解析目标，不引用 dto：锁的是响应形状本身。
type changeListWire struct {
	Items      []changeItemWire `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

type changeItemWire struct {
	ID         int    `json:"id"`
	TenantID   int    `json:"tenantId"`
	Status     string `json:"status"`
	RiskLevel  string `json:"riskLevel"`
	CreatedBy  int    `json:"createdBy"`
	AssigneeID *int   `json:"assigneeId"`
}

type changeListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func changeItemIDs(items []changeItemWire) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// seedChangeListTenant 建租户 + super_admin/agent 两个用户 + 25 条变更，布局固定：
// i<10 → riskLevel=high（其余 medium）、i>=20 → status=scheduled（其余 draft）；
// i=23、24 由 agent 本人创建，i=10 受理给 agent ⇒ agent 应见 3 条。
// 全部 created_at 相同，用于逼出排序的 ID 兜底。
func seedChangeListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) (tenantID, adminID, agentID int) {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Change list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	admin, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName("Change Admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	agent, err := client.User.Create().
		SetUsername(code + "-agent").
		SetEmail(code + "-agent@example.com").
		SetName("Change Agent").
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
		status := change.StatusDraft
		if i >= 20 {
			status = change.StatusScheduled
		}

		builder := client.Change.Create().
			SetTitle(fmt.Sprintf("CHG-%s-%02d", code, i)).
			SetDescription("验证变更列表分页夹紧").
			SetType("normal").
			SetStatus(status).
			SetPriority("medium").
			SetRiskLevel(riskLevelForIndex(i)).
			SetCreatedBy(creator).
			SetTenantID(tenant.ID).
			SetCreatedAt(changeListFixedCreatedAt).
			SetUpdatedAt(changeListFixedCreatedAt)
		if i == 10 {
			builder = builder.SetAssigneeID(agent.ID)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}
	return tenant.ID, admin.ID, agent.ID
}

func riskLevelForIndex(i int) string {
	if i < 10 {
		return "high"
	}
	return "medium"
}

// changeIDs 返回某租户变更的完整升序 ID 序列。
func changeIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.Change.Query().Where(change.TenantID(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Ints(ids)
	return ids
}
