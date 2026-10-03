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
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	cmdbHandler "itsm-backend/handlers/cmdb"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-6e）：CMDB 的 5 个列表信封原先只回
// {items,total,page,size}——既没有 pageSize/totalPages，又把请求别名 size 当页长；
// 页长同时有 3 个所有者：handler 用裸 strconv.Atoi(ctx.DefaultQuery("size")) 读值且
// 不夹紧，dto.ListCIRequest 上写着 binding:"min=1,max=200"（与平台 MaxPageSize=100 冲突），
// service 再把原值直接写进 Offset/Limit。
//
// 后果是可观测的数据错误而不是样式问题：Limit(0) 在 Ent 的 sqlgraph 里等于**不加 LIMIT**
// 直接返回整表，page<=0 算出负 OFFSET，没有排序时翻页由数据库返回顺序决定会重/漏。
//
// 本测试打在**生产注册**上（SetupRoutes → SetupCMDBRoutes → RequirePermission("cmdb","read")），
// 解码目标声明为本地 wire 结构而不引用 dto，因此修复前也能编译，证明的是运行时行为。

const cmdbListSecret = "cmdb-list-secret"

// cmdbListCreatedAt 让全部种子同秒：排序列 created_at 非唯一，缺 ID 并列键时页边界的
// 归属不确定，翻页会重复或漏条目。
var cmdbListCreatedAt = time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)

type cmdbListWire struct {
	Items []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"items"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PageSize   int  `json:"pageSize"`
	TotalPages int  `json:"totalPages"`
	Size       *int `json:"size"`
}

type cmdbListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// cmdbListTenantFixture 只带断言需要的身份：列表与视图可见性都按认证上下文收敛。
type cmdbListTenantFixture struct {
	tenantID  int
	adminID   int
	adminName string
}

func cmdbListItemIDs(res cmdbListWire) []int {
	ids := make([]int, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestCMDBListRoutesEnvelopeAndPagination(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupCMDBListRouter(t, client, logger)

	tenantA := seedCMDBListTenant(ctx, t, client, "cmdb-a", 25)
	tenantB := seedCMDBListTenant(ctx, t, client, "cmdb-b", 5)

	allA := seededCIIDs(ctx, t, client, tenantA.tenantID)
	require.Len(t, allA, 25)
	allB := seededCIIDs(ctx, t, client, tenantB.tenantID)
	ciForHistory := allA[0]

	// get 走生产路由，返回平台信封的原始 JSON。
	get := func(t *testing.T, tenant cmdbListTenantFixture, path string) (int, string, cmdbListEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenant.adminID, tenant.adminName, "super_admin", tenant.tenantID, cmdbListSecret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env cmdbListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}
	decode := func(t *testing.T, body string, env cmdbListEnvelope) cmdbListWire {
		t.Helper()
		var res cmdbListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		return res
	}
	// list 只打 CI 列表端点，便于按查询串断言。
	list := func(t *testing.T, query string) (int, string, cmdbListEnvelope) {
		path := "/api/v1/cmdb/cis"
		if query != "" {
			path += "?" + query
		}
		return get(t, tenantA, path)
	}

	t.Run("五个信封端点都只返回标准五键", func(t *testing.T) {
		paths := []struct {
			name string
			path string
		}{
			{"CI 列表", "/api/v1/cmdb/cis"},
			{"CI 类型列表", "/api/v1/cmdb/ci-types"},
			{"CI 标签列表", "/api/v1/cmdb/tags"},
			{"CI 变更历史", fmt.Sprintf("/api/v1/cmdb/cis/%d/history", ciForHistory)},
			{"保存视图列表", "/api/v1/cmdb/views"},
			{"导入任务列表", "/api/v1/cmdb/import"},
			{"导出任务列表", "/api/v1/cmdb/export"},
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

	t.Run("CI 列表默认页长 20 且 totalPages 如实算出", func(t *testing.T) {
		status, body, env := list(t, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize, "缺省页长统一到平台 20，不再是 handler 私有的值")
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Len(t, res.Items, 20)
		assert.Equal(t, allA[:20], cmdbListItemIDs(res), "默认第一页必须是全序的前 20 条")
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
			paged = append(paged, cmdbListItemIDs(res)...)
		}
		assert.Equal(t, allA, paged, "created_at 同秒时必须由 ID 并列键兜住页边界")

		// 页码越过总页数后必须是空集，而不是回绕或重复最后一页。
		_, body, env := list(t, "page=4&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Empty(t, res.Items, "越界页码必须返回空: body=%s", body)
	})

	t.Run("标签列表同秒种子逐页不重不漏", func(t *testing.T) {
		seen := map[int]bool{}
		for page := 1; page <= 4; page++ {
			_, body, env := get(t, tenantA, fmt.Sprintf("/api/v1/cmdb/tags?page=%d&pageSize=5", page))
			require.Equal(t, 0, env.Code, body)
			res := decode(t, body, env)
			assert.Equal(t, 12, res.Total, body)
			assert.Equal(t, 3, res.TotalPages, body)
			for _, id := range cmdbListItemIDs(res) {
				require.False(t, seen[id], "第 %d 页返回了重复标签: body=%s", page, body)
				seen[id] = true
			}
			if len(res.Items) < res.PageSize {
				break
			}
		}
		assert.Len(t, seen, 12, "12 个标签必须恰好各出现一次")
	})

	t.Run("变更历史按版本倒序且分页如实", func(t *testing.T) {
		_, body, env := get(t, tenantA, fmt.Sprintf("/api/v1/cmdb/cis/%d/history?page=2&pageSize=4", ciForHistory))
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 6, res.Total, "该 CI 有 6 条历史: body=%s", body)
		assert.Equal(t, 2, res.TotalPages, body)
		assert.Len(t, res.Items, 2, body)
	})

	t.Run("size 不再是 pageSize 的别名", func(t *testing.T) {
		// 修复前：handler 读的就是 size，这个请求返回 5 条。
		_, body, env := list(t, "size=5")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 20, res.PageSize, "size 不得作为页长别名生效: body=%s", body)
		assert.Len(t, res.Items, 20, body)
	})

	t.Run("offset/limit 别名不参与分页决策", func(t *testing.T) {
		_, body, env := list(t, "pageSize=10&offset=100&limit=25")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 10, res.PageSize, "limit 不得替代 pageSize: body=%s", body)
		assert.Equal(t, allA[:10], cmdbListItemIDs(res), "offset 不得替代 page 计算偏移: body=%s", body)
	})

	t.Run("越界页长回落默认页长而不是整表", func(t *testing.T) {
		// 修复前：binding:"max=200" 把 pageSize=5000 拦成业务码 1001 参数错误，
		// 而 size=5000 会原样下传给 Ent。
		status, body, env := list(t, "pageSize=5000")
		require.Equal(t, http.StatusOK, status, "越界页长应回落默认页长: body=%s", body)
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 20, res.PageSize, "越界页长不得原样回显: body=%s", body)
		assert.Len(t, res.Items, 20, body)
		assert.Equal(t, 2, res.TotalPages, body)
	})

	t.Run("pageSize=0 回到默认页长", func(t *testing.T) {
		_, body, env := list(t, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 20, res.PageSize, body)
		assert.Len(t, res.Items, 20, "页长 0 不得退化成不加 LIMIT: body=%s", body)
	})

	t.Run("page 为负数时回到第一页", func(t *testing.T) {
		_, body, env := list(t, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Page, "负 Offset 不得出现在查询里")
		assert.Equal(t, allA[:10], cmdbListItemIDs(res), body)
	})

	t.Run("CI 类型列表按租户收敛", func(t *testing.T) {
		_, body, env := get(t, tenantB, "/api/v1/cmdb/ci-types?pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 2, res.Total, "每个租户各 2 个 CI 类型，跨租户不得合并: body=%s", body)
		assert.Equal(t, 1, res.TotalPages, body)
		assert.Len(t, res.Items, 2)
	})

	t.Run("租户 B 的 CI 列表只含自己的条目", func(t *testing.T) {
		_, body, env := get(t, tenantB, "/api/v1/cmdb/cis?pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 5, res.Total, body)
		assert.Equal(t, allB, cmdbListItemIDs(res), body)
		for _, id := range cmdbListItemIDs(res) {
			assert.NotContains(t, allA, id, "租户 B 的响应里不得混入租户 A 的 CI")
		}
	})

	t.Run("保存视图只看本人创建的和公开的", func(t *testing.T) {
		// 种子：本人 1 条私有、他人 1 条公开、他人 1 条私有。
		_, body, env := get(t, tenantA, "/api/v1/cmdb/views?pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 2, res.Total, "本人 1 条 + 他人公开 1 条，他人私有不可见: body=%s", body)
	})

	t.Run("includePublic=false 只看本人视图", func(t *testing.T) {
		_, body, env := get(t, tenantA, "/api/v1/cmdb/views?pageSize=100&includePublic=false")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Total, "关闭公开视图后只剩本人 1 条: body=%s", body)
	})

	t.Run("跨租户读取 CI 历史 fail closed", func(t *testing.T) {
		// 租户 A 的 CI id 在租户 B 的上下文里不存在，不得返回租户 A 的历史行。
		status, body, env := get(t, tenantB, fmt.Sprintf("/api/v1/cmdb/cis/%d/history?pageSize=100", ciForHistory))
		assert.NotEqual(t, 0, env.Code, "跨租户读取必须失败而不是返回数据: body=%s", body)
		if env.Code == 0 {
			res := decode(t, body, env)
			assert.Zero(t, res.Total, body)
		}
		t.Logf("跨租户历史读取: http=%d code=%d message=%s", status, env.Code, env.Message)
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		for _, path := range []string{"/api/v1/cmdb/cis", "/api/v1/cmdb/ci-types", "/api/v1/cmdb/tags", "/api/v1/cmdb/views"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, path+" body="+w.Body.String())
		}
	})

	t.Run("search 过滤把 total 收敛到匹配条数", func(t *testing.T) {
		_, body, env := list(t, "search=ci-07&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Total, "只有 ci-07 命中: body=%s", body)
		require.Len(t, res.Items, 1, body)
		assert.Equal(t, "ci-07", res.Items[0].Name, body)
	})

	t.Run("空结果序列化为 []", func(t *testing.T) {
		_, body, env := list(t, "search=no-such-ci")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		res := decode(t, body, env)
		assert.Equal(t, 0, res.Total, body)
		assert.Equal(t, 0, res.TotalPages, body)
	})

	t.Run("兼容别名路径返回同一份信封", func(t *testing.T) {
		// /api/v1/configuration-items 是同一 handler 的旧前缀，契约必须一致。
		_, body, env := get(t, tenantA, "/api/v1/configuration-items?pageSize=10")
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data), body)
		res := decode(t, body, env)
		assert.Equal(t, allA[:10], cmdbListItemIDs(res), body)
	})

	t.Run("CI 类型搜索按名称过滤", func(t *testing.T) {
		_, body, env := get(t, tenantA, "/api/v1/cmdb/ci-types?search=Server&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		res := decode(t, body, env)
		assert.Equal(t, 1, res.Total, "只命中 Server-cmdb-a: body=%s", body)
	})
}

// setupCMDBListRouter 用生产构造函数装配 CMDB handler 并注册进 SetupRoutes。
func setupCMDBListRouter(t *testing.T, client *ent.Client, logger *zap.SugaredLogger) *gin.Engine {
	t.Helper()

	ciTypeSvc := service.NewCITypeService(client, logger)
	ciAttrSvc := service.NewCIAttributeDefinitionService(client, logger)
	historySvc := service.NewCIHistoryService(client, logger)
	tagSvc := service.NewCITagService(client, logger)
	ciSvc := service.NewConfigurationItemService(client, logger, historySvc, tagSvc)
	relSvc := service.NewCIRelationshipService(client, logger)
	importExportSvc := service.NewCMDBImportExportService(client, logger, ciSvc, tagSvc)
	savedViewSvc := service.NewCMDBSavedViewService(client, logger)

	production := cmdbHandler.NewProductionService(logger, ciTypeSvc, ciAttrSvc, ciSvc, relSvc, historySvc, tagSvc, importExportSvc, savedViewSvc)
	svc := cmdbHandler.NewService(cmdbHandler.NewEntRepository(client), production, logger)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:   cmdbListSecret,
		Logger:      logger,
		Client:      client,
		CMDBHandler: cmdbHandler.NewHandler(svc),
	})
	return r
}

// seedCMDBListTenant 建租户 + 两名 super_admin，并种 count 个同秒 CI、2 个同秒 CI 类型、
// 12 个同秒标签、给最小 ID 的 CI 种 6 条历史、3 条保存视图（本人私有/他人公开/他人私有）。
func seedCMDBListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) cmdbListTenantFixture {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("CMDB list " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantID := tenant.ID

	admin := seedCMDBListUser(ctx, t, client, tenantID, code+"-admin")
	other := seedCMDBListUser(ctx, t, client, tenantID, code+"-other")

	serverType, err := client.CIType.Create().
		SetName("Server-" + code).SetTenantID(tenantID).
		SetCreatedAt(cmdbListCreatedAt).SetUpdatedAt(cmdbListCreatedAt).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.CIType.Create().
		SetName("Database-" + code).SetTenantID(tenantID).
		SetCreatedAt(cmdbListCreatedAt).SetUpdatedAt(cmdbListCreatedAt).
		Save(ctx)
	require.NoError(t, err)

	for i := 0; i < 12; i++ {
		_, err := client.CITag.Create().
			SetKey(fmt.Sprintf("k%02d", i)).
			SetValue(fmt.Sprintf("v%02d", i)).
			SetTenantID(tenantID).
			SetCreatedAt(cmdbListCreatedAt).
			SetUpdatedAt(cmdbListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}

	for i := 0; i < count; i++ {
		_, err := client.ConfigurationItem.Create().
			SetName(fmt.Sprintf("ci-%02d", i)).
			SetCiTypeID(serverType.ID).
			SetTenantID(tenantID).
			SetStatus("active").
			SetEnvironment("dev").
			SetCriticality("low").
			SetAssetTag(fmt.Sprintf("tag-%s-%02d", code, i)).
			SetSerialNumber(fmt.Sprintf("%s-%02d-sn", code, i)).
			SetCreatedAt(cmdbListCreatedAt).
			SetUpdatedAt(cmdbListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}

	firstCI, err := client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		Order(ent.Asc(configurationitem.FieldID)).
		First(ctx)
	require.NoError(t, err)
	for version := 1; version <= 6; version++ {
		_, err := client.ConfigurationItemHistory.Create().
			SetCiID(firstCI.ID).
			SetVersion(version).
			SetOperation("update").
			SetBefore(map[string]interface{}{"name": fmt.Sprintf("v%d", version-1)}).
			SetAfter(map[string]interface{}{"name": fmt.Sprintf("v%d", version)}).
			SetChangedFields([]string{"name"}).
			SetOperatorID(other.ID).
			SetOperatorName(code + "-other").
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	createView := func(name string, creatorID int, isPublic bool) {
		_, err := client.CMDBSavedView.Create().
			SetName(name).
			SetTenantID(tenantID).
			SetCreatorID(creatorID).
			SetIsPublic(isPublic).
			SetFilters(map[string]interface{}{"status": "active"}).
			SetCreatedAt(cmdbListCreatedAt).
			SetUpdatedAt(cmdbListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}
	createView("mine", admin.ID, false)
	createView("shared", other.ID, true)
	createView("hidden", other.ID, false)

	return cmdbListTenantFixture{tenantID: tenantID, adminID: admin.ID, adminName: admin.Username}
}

func seedCMDBListUser(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func seededCIIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		IDs(ctx)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
