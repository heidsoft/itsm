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
	"itsm-backend/ent/changepir"
	"itsm-backend/ent/enttest"
	changeHandler "itsm-backend/handlers/change"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-6b）：GET /api/v1/changes/pirs 是**真分页**的列表
// （service/pir_service.go 用 Count + Offset/Limit），但修复前只返回 {items,total}：
// 调用方拿到 20 条却无从知道「第几页、共几页」，只能把已到手的当前页当成整个结果集。
//
// 同时实测到 page/pageSize 完全没有夹紧：handler 用裸 strconv.Atoi 把查询参数原样交给
// service，而
//   - pageSize=0 在 Ent 的 sqlgraph 里是 `if q.Limit != 0` —— 0 表示**不加 LIMIT**，
//     整表返回（25 条数据实测返回 25 条，而不是任何一页）；
//   - page=0/负数会算出负 Offset，Ent 里 `if q.Offset != 0` 仍会把负数写进 SQL。
//
// 因此这条回归同时锁住「五键齐备」与「夹紧后的页语义」，并且打在**生产注册**
// （SetupRoutes → SetupChangeRoutes → middleware.RequirePermission）上。

const (
	pirChangeSecret  = "change-pir-secret"
	pirReviewDateKey = "2026-09-01T00:00:00Z"
)

func TestChangePIRListRoutePaginationEnvelope(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_change_pir_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	// db 只在变更审批链/outbox 的原生 SQL 路径用到，PIR 列表全程走 Ent，测试传 nil
	// 与 handlers/change/repository_outbox_test.go:17 一致。
	logger := zaptest.NewLogger(t).Sugar()
	repo := changeHandler.NewEntRepository(client, nil)
	handler := changeHandler.NewHandler(changeHandler.NewService(repo, client, logger, nil))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     pirChangeSecret,
		Logger:        logger,
		Client:        client,
		ChangeHandler: handler,
	})

	// 租户 A：25 条 PIR，review_date 全部相同（制造并列键，逼出 ID 兜底排序）。
	// 其中 20 条 successful、5 条 failed，用于验证 result 过滤把 total 收敛。
	tenantA := seedPIRTenant(ctx, t, client, "pir-a", 20, 5)
	// 租户 B：3 条全部 failed，用于验证 result=successful 时的空页形状与租户收敛。
	tenantB := seedPIRTenant(ctx, t, client, "pir-b", 0, 3)

	// 完整升序 ID 序列（= 并列 review_date 下期望的全序）。
	allA := pirIDs(ctx, t, client, tenantA)
	require.Len(t, allA, 25)

	do := func(t *testing.T, tenantID int, query string) (int, string, pirListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(1, "pir-admin", "super_admin", tenantID, pirChangeSecret, time.Hour)
		require.NoError(t, err)

		path := "/api/v1/changes/pirs"
		if query != "" {
			path += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env pirListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}

	t.Run("默认请求返回标准五键，且按 page/pageSize 截断而不是整表返回", func(t *testing.T) {
		status, body, env := do(t, tenantA, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)

		keys := envelopeKeys(t, env.Data)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, keys,
			"PIR 列表必须只返回标准信封五键: body=%s", body)

		var page pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &page))
		assert.Equal(t, 1, page.Page)
		assert.Equal(t, 20, page.PageSize)
		assert.Equal(t, 25, page.Total, "total 必须是过滤后的全量条数，不是当前页长度")
		assert.Equal(t, 2, page.TotalPages)
		assert.Len(t, page.Items, 20)
		assert.Equal(t, allA[:20], itemIDs(page.Items), "默认第一页必须是全序的前 20 条")
	})

	t.Run("翻页不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			status, body, env := do(t, tenantA, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, http.StatusOK, status, body)
			var res pirListWire
			require.NoError(t, json.Unmarshal(env.Data, &res), body)
			assert.Equal(t, page, res.Page)
			assert.Equal(t, 10, res.PageSize)
			assert.Equal(t, 25, res.Total)
			assert.Equal(t, 3, res.TotalPages)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want)
			paged = append(paged, itemIDs(res.Items)...)
		}
		assert.Equal(t, allA, paged, "三页拼接必须等于完整升序序列：并列 review_date 也要有确定归属")
	})

	t.Run("pageSize=0 夹紧为默认页长，不再整表返回", func(t *testing.T) {
		_, body, env := do(t, tenantA, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		var res pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize, "pageSize=0 在 Ent 里等于不加 LIMIT，必须夹紧后再查")
		assert.Len(t, res.Items, 20, "修复前这条返回全部 25 条")
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("page 为 0 或负数时回到第一页", func(t *testing.T) {
		_, body, env := do(t, tenantA, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		var res pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page, "负 Offset 不得出现在查询里，页码必须回落为 1")
		assert.Equal(t, allA[:10], itemIDs(res.Items))
	})

	t.Run("result 过滤把 total 收敛到匹配条数", func(t *testing.T) {
		_, body, env := do(t, tenantA, "result=failed")
		require.Equal(t, 0, env.Code, body)
		var res pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 5, res.Total, "total 必须是过滤后的数量")
		assert.Len(t, res.Items, 5)
		assert.Equal(t, 1, res.TotalPages)
		for _, item := range res.Items {
			assert.Equal(t, "failed", item.OverallResult)
		}
	})

	t.Run("空结果序列化为 [] 而不是 null", func(t *testing.T) {
		_, body, env := do(t, tenantB, "result=successful")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		var res pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 0, res.Total)
		assert.Equal(t, 0, res.TotalPages)
	})

	t.Run("租户 B 只看到自己的 PIR", func(t *testing.T) {
		_, body, env := do(t, tenantB, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res pirListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 3, res.Total)
		require.Len(t, res.Items, 3)
		assert.Equal(t, pirIDs(ctx, t, client, tenantB), itemIDs(res.Items))
		for _, item := range res.Items {
			assert.Equal(t, tenantB, item.TenantID)
			assert.NotContains(t, allA, item.ID, "跨租户 ID 不得出现在响应里")
		}
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/changes/pirs", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})
}

// pirListWire 按**线上契约**声明解析目标，不引用 dto 结构体：回归锁的是响应形状本身，
// 实现侧换 DTO、复用别的结构体都绕不过断言。
type pirListWire struct {
	Items      []pirItemWire `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

type pirItemWire struct {
	ID            int    `json:"id"`
	TenantID      int    `json:"tenantId"`
	OverallResult string `json:"overallResult"`
}

// pirListEnvelope 是 { code, message, data } 的解析目标，data 保留原始 JSON 以便核对键集合。
type pirListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// seedPIRTenant 建一个租户及其 PIR：先 successful 条数，再 failed 条数，ID 递增。
func seedPIRTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, successful, failed int) int {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("PIR " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "@example.com").
		SetName("PIR Admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	reviewDate, err := time.Parse(time.RFC3339, pirReviewDateKey)
	require.NoError(t, err)

	for i, result := range pirSeedPlan(successful, failed) {
		changeEntity, err := client.Change.Create().
			SetTitle(fmt.Sprintf("change-%s-%d", code, i)).
			SetDescription("验证 PIR 分页信封").
			SetStatus("completed").
			SetCreatedBy(creator.ID).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.ChangePIR.Create().
			SetReviewerID(creator.ID).
			SetOverallResult(result).
			SetTenantID(tenant.ID).
			SetReviewDate(reviewDate).
			SetChangeID(changeEntity.ID).
			Save(ctx)
		require.NoError(t, err)
	}
	return tenant.ID
}

// pirSeedPlan 生成 successful 条 failed 条的 overall_result 序列，顺序即 ID 顺序。
func pirSeedPlan(successful, failed int) []string {
	plan := make([]string, 0, successful+failed)
	for i := 0; i < successful; i++ {
		plan = append(plan, "successful")
	}
	for i := 0; i < failed; i++ {
		plan = append(plan, "failed")
	}
	return plan
}

// pirIDs 返回某租户 PIR 的完整升序 ID 序列。
func pirIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.ChangePIR.Query().Where(changepir.TenantID(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Ints(ids)
	return ids
}

func itemIDs(items []pirItemWire) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// envelopeKeys 返回 data 对象的键名升序集合，用于断言精确键集合而非「包含某键」。
func envelopeKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj), "data=%s", string(raw))
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// jsonOf 取出 data 里某个键的原始 JSON。
func jsonOf(t *testing.T, raw json.RawMessage, key string) string {
	t.Helper()
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj), "data=%s", string(raw))
	value, ok := obj[key]
	require.True(t, ok, "缺少键 %s", key)
	return strings.TrimSpace(string(value))
}
