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

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/asset"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/enttest"
	assetHandler "itsm-backend/handlers/asset"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-6g）：GET /api/v1/assets 与 GET /api/v1/licenses
// 的集合键原先是领域名（assets / licenses），许可列表连 page/pageSize/totalPages 都没有。
//
// 更严重的是页长有三套真相，其中一套会把整表读出来：
//   - handler 用 strconv 自己校验，允许 pageSize 到 200 并对非法值返回 400；
//   - 服务层再把 >100 的值换成 20，于是 101..200 这一段「handler 说可以、服务层不认」；
//   - 许可侧 handler 用 page, _ := strconv.Atoi(...) 忽略解析错误，服务层再写
//     `if page > 0 && pageSize > 0` 才加 Offset/Limit —— 漏写或写错分页参数时条件为假，
//     查询不带 LIMIT，一次返回该租户全部许可证。
//
// 两个列表都只按 created_at 排序且没有 ID 并列键，翻页归属不确定。
//
// 本测试打在**生产注册**上（SetupRoutes → assetHandler.SetupRoutes →
// RequirePermission("asset"|"license","read")），解码目标声明为本地 wire 结构而不引用
// dto，因此修复前也能编译，证明的是运行时行为。

const assetListSecret = "asset-list-secret"

// assetListCreatedAt 让全部种子同秒：排序列 created_at 非唯一，缺 ID 并列键时
// 翻页归属由数据库返回顺序决定。
var assetListCreatedAt = time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC)

type assetListTenant struct {
	tenantID int
	userID   int
	username string
}

type assetListWire struct {
	Items []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Status   string `json:"status"`
		Category string `json:"category"`
	} `json:"items"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PageSize   int  `json:"pageSize"`
	TotalPages int  `json:"totalPages"`
	Size       *int `json:"size"`
}

type assetListEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func assetListItemIDs(res assetListWire) []int {
	ids := make([]int, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestAssetAndLicenseListRoutesEnvelopeAndPagination(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_asset_list_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupAssetListRouter(t, client, logger)

	tenantA := seedAssetListTenant(ctx, t, client, "asset-a", 25)
	tenantB := seedAssetListTenant(ctx, t, client, "asset-b", 4)

	assetsA := seededAssetIDs(ctx, t, client, tenantA.tenantID)
	require.Len(t, assetsA, 25)
	licensesA := seededLicenseIDs(ctx, t, client, tenantA.tenantID)
	require.Len(t, licensesA, 25)

	// list 打 A 租户的某个列表端点，返回状态码、原始报文与 data 段。
	list := func(t *testing.T, path string) (int, string, string, assetListWire) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenantA.userID, tenantA.username, "super_admin", tenantA.tenantID, assetListSecret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env assetListEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		var res assetListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), string(env.Data), res
	}

	t.Run("两个列表端点都只返回标准五键", func(t *testing.T) {
		for _, path := range []string{"/api/v1/assets", "/api/v1/licenses"} {
			status, body, data, res := list(t, path)
			assert.Equal(t, http.StatusOK, status, body)
			assert.Len(t, res.Items, 20, "默认页长 20: %s", body)

			var keys map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(data), &keys), body)
			names := make([]string, 0, len(keys))
			for k := range keys {
				names = append(names, k)
			}
			sort.Strings(names)
			assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, names, "%s 键集合漂移: %s", path, body)

			assert.Equal(t, 25, res.Total, path)
			assert.Equal(t, 1, res.Page, path)
			assert.Equal(t, 20, res.PageSize, path)
			assert.Equal(t, 2, res.TotalPages, path)
			assert.NotContains(t, body, `"assets"`, "不得再返回领域名集合键: %s", body)
			assert.NotContains(t, body, `"licenses"`, "不得再返回领域名集合键: %s", body)
			assert.Nil(t, res.Size, "不得返回 size 别名: %s", body)
		}
	})

	t.Run("三页拼接不重不漏且等于完整升序集合", func(t *testing.T) {
		seen := make([]int, 0, 25)
		for page := 1; page <= 3; page++ {
			_, body, _, res := list(t, fmt.Sprintf("/api/v1/assets?page=%d&pageSize=10", page))
			require.Equal(t, 10, res.PageSize, body)
			require.Equal(t, 3, res.TotalPages, body)
			if page == 3 {
				assert.Len(t, res.Items, 5, "最后一页应是剩余 5 条: %s", body)
			} else {
				assert.Len(t, res.Items, 10, body)
			}
			seen = append(seen, assetListItemIDs(res)...)
		}
		sort.Ints(seen)
		assert.Equal(t, assetsA, seen, "翻页结果必须等于该租户全部资产")
	})

	t.Run("非法与越界页长一律回落默认页长而不是整表", func(t *testing.T) {
		cases := []struct{ name, query string }{
			{"页长缺省", ""},
			{"页长为 0", "pageSize=0"},
			{"页长负数", "pageSize=-5"},
			{"页长非数字", "pageSize=abc"},
			{"页长越界", "pageSize=5000"},
			// 原 handler 允许 101..200 并报「必须在 1 到 200 之间」，服务层却换成 20。
			{"原 handler 自称允许的 150", "pageSize=150"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				for _, base := range []string{"/api/v1/assets", "/api/v1/licenses"} {
					path := base
					if tc.query != "" {
						path += "?" + tc.query
					}
					status, body, _, res := list(t, path)
					assert.Equal(t, http.StatusOK, status, "%s: %s", path, body)
					assert.Equal(t, 20, res.PageSize, "%s: %s", path, body)
					assert.Len(t, res.Items, 20, "%s: %s", path, body)
					assert.Equal(t, 2, res.TotalPages, "%s: %s", path, body)
				}
			})
		}
	})

	t.Run("页码非法回落第一页而不是负偏移", func(t *testing.T) {
		// 负 OFFSET 在 Postgres 上是 ERROR: OFFSET must not be negative（生产库 500）。
		for _, query := range []string{"page=0", "page=-1", "page=abc"} {
			for _, base := range []string{"/api/v1/assets", "/api/v1/licenses"} {
				status, body, _, res := list(t, base+"?"+query+"&pageSize=5")
				assert.Equal(t, http.StatusOK, status, "%s?%s: %s", base, query, body)
				assert.Equal(t, 1, res.Page, "%s?%s: %s", base, query, body)
				assert.Len(t, res.Items, 5, "%s?%s: %s", base, query, body)
			}
		}
	})

	t.Run("size 与 offset 别名不生效", func(t *testing.T) {
		for _, base := range []string{"/api/v1/assets", "/api/v1/licenses"} {
			_, body, _, res := list(t, base+"?size=5&offset=10&page_size=5")
			assert.Equal(t, 20, res.PageSize, body)
			assert.Equal(t, 1, res.Page, "offset 不得参与页码计算: %s", body)
			assert.Len(t, res.Items, 20, body)
		}
	})

	t.Run("过滤条件按后端真实参数生效", func(t *testing.T) {
		_, body, _, res := list(t, "/api/v1/assets?type=hardware")
		assert.Equal(t, 25, res.Total, "种子全是 hardware: %s", body)

		_, body, _, res = list(t, "/api/v1/assets?type=software")
		assert.Equal(t, 0, res.Total, body)
		assert.Contains(t, body, `"items":[]`, "空结果必须序列化为 []: "+body)

		_, body, _, res = list(t, "/api/v1/assets?category=compute")
		assert.Equal(t, 10, res.Total, "前 10 条是 compute: %s", body)

		_, body, _, res = list(t, "/api/v1/assets?status=available")
		assert.Equal(t, 25, res.Total, body)

		_, body, _, res = list(t, "/api/v1/licenses?type=perpetual")
		assert.Equal(t, 25, res.Total, "种子全是 perpetual: %s", body)

		_, body, _, res = list(t, "/api/v1/licenses?status=expired")
		assert.Equal(t, 0, res.Total, body)

		// license_type 的查询名是 type；snake_case 别名不得被读取。
		_, body, _, res = list(t, "/api/v1/licenses?license_type=perpetual")
		assert.Equal(t, 25, res.Total, "snake_case 参数必须被忽略（不过滤=仍返回全部）: %s", body)
	})

	t.Run("租户 B 只看到自己的资产与许可证", func(t *testing.T) {
		token, err := middleware.GenerateAccessToken(tenantB.userID, tenantB.username, "super_admin", tenantB.tenantID, assetListSecret, time.Hour)
		require.NoError(t, err)

		for _, path := range []string{"/api/v1/assets", "/api/v1/licenses"} {
			req := httptest.NewRequest(http.MethodGet, path+"?pageSize=100", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			var env assetListEnvelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
			var res assetListWire
			require.NoError(t, json.Unmarshal(env.Data, &res), "body=%s", w.Body.String())

			assert.Equal(t, 4, res.Total, path)
			assert.Len(t, res.Items, 4, path)
			for _, item := range res.Items {
				assert.Contains(t, item.Name, "asset-b", "%s 出现跨租户行: %+v", path, item)
			}
		}
	})

	t.Run("未认证不得返回列表载荷", func(t *testing.T) {
		for _, path := range []string{"/api/v1/assets", "/api/v1/licenses"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())
		}
	})
}

// TestAssetListDTOKeys 在 DTO 层锁死集合键：本批把 assets/licenses 改成 items，
// 任何把 json tag 改回领域名的改动都会在这里红。
func TestAssetListDTOKeys(t *testing.T) {
	envelopeKeys := func(t *testing.T, payload any) []string {
		t.Helper()
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		var data map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &data))
		names := make([]string, 0, len(data))
		for k := range data {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	}

	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"},
		envelopeKeys(t, dto.AssetListResponse{Items: []dto.AssetResponse{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}))
	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"},
		envelopeKeys(t, dto.LicenseListResponse{Items: []dto.LicenseResponse{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}))
}

func setupAssetListRouter(t *testing.T, client *ent.Client, logger *zap.SugaredLogger) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:    assetListSecret,
		Logger:       logger,
		Client:       client,
		AssetHandler: assetHandler.NewHandler(service.NewAssetService(client, logger), service.NewAssetLicenseService(client, logger), logger),
	})
	return r
}

// seedAssetListTenant 建租户 + 一名 super_admin，并种 25 条资产与 25 条许可证
// （B 租户只有 4 条）。前 10 条资产分类为 compute，其余 network；全部行写同一
// created_at/updated_at，用来暴露排序缺并列键时的页边界问题。
func seedAssetListTenant(ctx context.Context, t *testing.T, client *ent.Client, code string, count int) assetListTenant {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Asset list " + code).
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

	for i := 1; i <= count; i++ {
		category := "network"
		if i <= 10 {
			category = "compute"
		}
		_, err := client.Asset.Create().
			SetAssetNumber(fmt.Sprintf("AST-%s-%02d", code, i)).
			SetName(fmt.Sprintf("Asset %s %02d", code, i)).
			SetType("hardware").
			SetStatus("available").
			SetCategory(category).
			SetTenantID(tenantID).
			SetCreatedAt(assetListCreatedAt).
			SetUpdatedAt(assetListCreatedAt).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.AssetLicense.Create().
			SetName(fmt.Sprintf("License %s %02d", code, i)).
			SetLicenseType("perpetual").
			SetStatus("active").
			SetTotalQuantity(10).
			SetTenantID(tenantID).
			SetCreatedAt(assetListCreatedAt).
			SetUpdatedAt(assetListCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}

	return assetListTenant{tenantID: tenantID, userID: admin.ID, username: username}
}

func seededAssetIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.Asset.Query().Where(asset.TenantIDEQ(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func seededLicenseIDs(ctx context.Context, t *testing.T, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids, err := client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID)).IDs(ctx)
	require.NoError(t, err)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
