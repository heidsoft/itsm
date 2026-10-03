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
	"itsm-backend/ent/servicecatalog"
	serviceCatalogHandler "itsm-backend/handlers/service_catalog"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-6f）：服务目录列表信封原先回
// {catalogs,total,page,size} —— 领域名集合键 + size 别名，是棘轮里最后一条 size 违规。
// 页长同时有 4 个所有者：handler 不夹紧、Service.List 缺省 10、Service.Search 缺省 20
// 且把 Page/Size 写死成 1/20（/search 因此永远只能拿到第一页），
// dto.GetServiceCatalogsRequest 上的 binding:"max=1000" 是第四套真相。
//
// 后果是数据错误而不是样式问题：repository 收到零值分页会算出 Limit(0)，而 Ent 的
// Limit(0) 等于不加 LIMIT，一次调用就把该租户整表读出来；缺 ID 并列键时同秒数据的
// 页边界归属不确定，翻页会重或漏。
//
// 本测试打在**生产注册**上（SetupRoutes → SetupServiceCatalogRoutes →
// RequirePermission("service_catalog","read")），解码目标声明为本地 wire 结构而不引用
// dto，因此修复前也能编译，证明的是运行时行为。

const serviceCatalogListSecret = "service-catalog-list-secret"

// serviceCatalogListCreatedAt 让全部种子同秒：排序列 created_at 非唯一，缺 ID 并列键时
// 翻页归属由数据库返回顺序决定。
var serviceCatalogListCreatedAt = time.Date(2026, 9, 21, 4, 0, 0, 0, time.UTC)

type serviceCatalogListTenant struct {
	tenantID  int
	adminID   int
	adminName string
}

type serviceCatalogListWire struct {
	Items []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Status   string `json:"status"`
	} `json:"items"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PageSize   int  `json:"pageSize"`
	TotalPages int  `json:"totalPages"`
	Size       *int `json:"size"`
}

type serviceCatalogListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func serviceCatalogListItemIDs(res serviceCatalogListWire) []int {
	ids := make([]int, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestServiceCatalogListRoutesEnvelopeAndPagination(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_service_catalog_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupServiceCatalogListRouter(t, client, logger)

	// A：25 条 enabled（前 10 条 compute，其余 network），用于分页与过滤断言。
	tenantA := seedServiceCatalogListTenant(ctx, t, client, "sc-a", 25, 0)
	// B：2 enabled + 3 disabled，用于租户收敛与「search 只见启用项」断言。
	tenantB := seedServiceCatalogListTenant(ctx, t, client, "sc-b", 2, 3)

	allA := seededServiceCatalogIDs(ctx, t, client, tenantA.tenantID)
	require.Len(t, allA, 25)
	allB := seededServiceCatalogIDs(ctx, t, client, tenantB.tenantID)
	require.Len(t, allB, 5)

	get := func(t *testing.T, tenant serviceCatalogListTenant, path string) (int, string, serviceCatalogListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenant.adminID, tenant.adminName, "super_admin", tenant.tenantID, serviceCatalogListSecret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env serviceCatalogListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}
	decode := func(t *testing.T, body string, env serviceCatalogListEnvelope) serviceCatalogListWire {
		t.Helper()
		var res serviceCatalogListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		return res
	}
	// list 只打 A 租户的列表端点，便于按查询串断言。
	list := func(t *testing.T, query string) (int, string, serviceCatalogListEnvelope) {
		path := "/api/v1/service-catalogs"
		if query != "" {
			path += "?" + query
		}
		return get(t, tenantA, path)
	}

	t.Run("四个列表端点都只返回标准五键", func(t *testing.T) {
		paths := []struct{ name, path string }{
			{"service-catalog", "/api/v1/service-catalog"},
			{"service-catalogs", "/api/v1/service-catalogs"},
			{"service-catalog-services", "/api/v1/service-catalog-services"},
			{"service-catalogs/search", "/api/v1/service-catalogs/search"},
		}
		for _, tc := range paths {
			t.Run(tc.name, func(t *testing.T) {
				status, body, env := get(t, tenantA, tc.path)
				require.Equal(t, http.StatusOK, status, body)
				require.Equal(t, 0, env.Code, body)
				assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data),
					"%s 必须收敛为平台信封五键: body=%s", tc.name, body)
			})
		}
	})

	t.Run("默认页长 20 且 totalPages 如实算出", func(t *testing.T) {
		status, body, env := list(t, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize, "缺省页长统一到平台 20，不再是 Service.List 私有的 10")
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Len(t, res.Items, 20)
		assert.Equal(t, allA[:20], serviceCatalogListItemIDs(res), "默认第一页必须是全序的前 20 条")
		assert.Nil(t, res.Size, "响应不得再回显 size")
	})

	t.Run("三页拼接不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			_, body, env := list(t, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, 0, env.Code, body)
			res := decode(t, body, env)
			assert.Equal(t, page, res.Page, body)
			assert.Equal(t, 10, res.PageSize, body)
			assert.Equal(t, 3, res.TotalPages, body)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want, body)
			paged = append(paged, serviceCatalogListItemIDs(res)...)
		}
		assert.Equal(t, allA, paged, "created_at 同秒时必须由 ID 并列键兜住页边界")

		// 页码越过总页数后必须是空集，而不是回绕或重复最后一页。
		_, body, env := list(t, "page=4&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Empty(t, res.Items, "越界页码必须返回空: body=%s", body)
	})

	t.Run("页长只有一个所有者且 size 别名无效", func(t *testing.T) {
		// size=1000/1001 与 pageSize=0/300 都必须回落到平台缺省 20。
		// 修复前实测（120 条种子）：?size=1000 通过 binding:"max=1000" 后被 handler
		// 静默夹成 100 条，?size=1001 却被 binding 拒绝返回业务 code 1001 ——
		// binding 上限 1000 与 handler 上限 100 是两套真相，相邻输入行为不一致。
		for _, query := range []string{"size=1000", "size=1001", "pageSize=0", "pageSize=300", "page=-1&pageSize=abc"} {
			t.Run(query, func(t *testing.T) {
				status, body, env := list(t, query)
				require.Equal(t, http.StatusOK, status, body)
				require.Equal(t, 0, env.Code, body)
				res := decode(t, body, env)
				assert.Equal(t, 20, res.PageSize, "非法/越界页长必须回落到 20，而不是夹紧或整表: body=%s", body)
				assert.Equal(t, 1, res.Page, "page<=0 必须回落到 1，不得算出负 OFFSET: body=%s", body)
				assert.Equal(t, 2, res.TotalPages, body)
				assert.Len(t, res.Items, 20, body)
			})
		}

		// 100 是平台上限内的合法页长，必须如实采纳（证明规则不是「一律压成 20」）。
		_, body, env := list(t, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 100, res.PageSize, body)
		assert.Equal(t, 25, res.Total, body)
		assert.Equal(t, 1, res.TotalPages, body)
		assert.Len(t, res.Items, 25, body)
	})

	t.Run("过滤后 total 与 totalPages 按过滤集算", func(t *testing.T) {
		_, body, env := list(t, "category=compute&pageSize=5")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 10, res.Total, "A 租户 compute 只有 10 条: body=%s", body)
		assert.Equal(t, 2, res.TotalPages, body)
		require.Len(t, res.Items, 5, body)
		for _, item := range res.Items {
			assert.Equal(t, "compute", item.Category, body)
		}

		_, body, env = list(t, "status=enabled&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res = decode(t, body, env)
		assert.Equal(t, 25, res.Total, "enabled 过滤后仍是 A 的全量: body=%s", body)
	})

	t.Run("search 分页真正生效", func(t *testing.T) {
		// 原实现把 Page/Size 写死 1/20，第二页永远拿不到数据。
		search := func(t *testing.T, page int) serviceCatalogListWire {
			t.Helper()
			_, body, env := get(t, tenantA, fmt.Sprintf("/api/v1/service-catalogs/search?q=Svc&pageSize=10&page=%d", page))
			require.Equal(t, 0, env.Code, body)
			res := decode(t, body, env)
			assert.Equal(t, 25, res.Total, "q=Svc 命中 A 租户全部 25 条: body=%s", body)
			assert.Equal(t, 3, res.TotalPages, body)
			assert.Equal(t, page, res.Page, body)
			require.Len(t, res.Items, 10, body)
			for _, item := range res.Items {
				assert.Contains(t, item.Name, "Svc", body)
			}
			return res
		}
		p1 := search(t, 1)
		p2 := search(t, 2)
		assert.NotEqual(t, serviceCatalogListItemIDs(p1), serviceCatalogListItemIDs(p2),
			"/search 第二页必须返回不同条目")
		assert.Equal(t, allA[10:20], serviceCatalogListItemIDs(p2), "/search 必须与 List 用同一套全序")
	})

	t.Run("空结果必须是数组而不是 null", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/service-catalogs?category=no-such-category",
			"/api/v1/service-catalogs/search?q=no-such-keyword",
		} {
			_, body, env := get(t, tenantA, path)
			require.Equal(t, 0, env.Code, body)
			assert.Equal(t, "[]", jsonOf(t, env.Data, "items"), "空集合必须序列化为 []: body=%s", body)
			res := decode(t, body, env)
			assert.Equal(t, 0, res.Total, body)
			assert.Equal(t, 0, res.TotalPages, body)
		}
	})

	t.Run("租户 B 只见自己的数据且 search 只含启用项", func(t *testing.T) {
		status, body, env := get(t, tenantB, "/api/v1/service-catalogs?pageSize=100")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 5, res.Total, "不得跨租户读到 A 的 25 条: body=%s", body)
		assert.Equal(t, allB, serviceCatalogListItemIDs(res), body)

		_, body, env = get(t, tenantB, "/api/v1/service-catalogs/search?q=Svc&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res = decode(t, body, env)
		assert.Equal(t, 2, res.Total, "search 按租户收敛且只返回启用项: body=%s", body)
		for _, item := range res.Items {
			assert.Equal(t, "enabled", item.Status, body)
		}

		_, body, env = get(t, tenantB, "/api/v1/service-catalogs?status=disabled")
		require.Equal(t, 0, env.Code, body)
		res = decode(t, body, env)
		assert.Equal(t, 3, res.Total, "B 租户 disabled 有 3 条: body=%s", body)
	})

	t.Run("缺少租户上下文必须 fail closed", func(t *testing.T) {
		// tenantID=0 不得退化成「无租户过滤 = 全表」。
		token, err := middleware.GenerateAccessToken(1, "no-tenant-sc", "super_admin", 0, serviceCatalogListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/service-catalogs", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env serviceCatalogListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		assert.NotEqual(t, 0, env.Code, "缺少租户上下文不得返回成功: body=%s", w.Body.String())
		assert.NotContains(t, w.Body.String(), `"total"`, "不得返回任何集合载荷: body=%s", w.Body.String())
	})
}

func setupServiceCatalogListRouter(t *testing.T, client *ent.Client, logger *zap.SugaredLogger) *gin.Engine {
	t.Helper()

	svc := serviceCatalogHandler.NewService(serviceCatalogHandler.NewEntRepository(client), logger)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:             serviceCatalogListSecret,
		Logger:                logger,
		Client:                client,
		ServiceCatalogHandler: serviceCatalogHandler.NewHandler(svc),
	})
	return r
}

// seedServiceCatalogListTenant 建租户 + 一名 super_admin，并种 enabledCount 条 enabled
// 与 disabledCount 条 disabled 服务目录。全部行写同一 created_at/updated_at，用来暴露
// 排序缺并列键时的页边界问题。名称形如 "Svc sc-a 01"，q=Svc 可命中整租户。
func seedServiceCatalogListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, enabledCount, disabledCount int) serviceCatalogListTenant {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Service catalog list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantID := tenant.ID

	username := code + "-admin"
	admin, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	create := func(i int, status string) {
		category := "network"
		if i <= 10 {
			category = "compute"
		}
		_, err := client.ServiceCatalog.Create().
			SetName(fmt.Sprintf("Svc %s %02d", code, i)).
			SetCategory(category).
			SetDescription("seeded for envelope regression").
			SetStatus(status).
			SetIsActive(status == "enabled").
			SetTenantID(tenantID).
			SetCreatedAt(serviceCatalogListCreatedAt).
			SetUpdatedAt(serviceCatalogListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}
	for i := 1; i <= enabledCount; i++ {
		create(i, "enabled")
	}
	for i := enabledCount + 1; i <= enabledCount+disabledCount; i++ {
		create(i, "disabled")
	}

	return serviceCatalogListTenant{tenantID: tenantID, adminID: admin.ID, adminName: username}
}

func seededServiceCatalogIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.ServiceCatalog.Query().
		Where(servicecatalog.TenantIDEQ(tenantID)).
		IDs(ctx)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
