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
	"itsm-backend/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-20）：云账号/云服务/云资源曾经有两套平行的已注册表面
// —— /api/v1/cloud/*（15 条路由，带真分页与 fail-closed 租户）与 /api/v1/cmdb/cloud-*
// （UI 唯一在用的那套，裸数组、无排序无上限整表读取、service_id 别名、err.Error() 外露）。
// 产品裁决是「留 cmdb，删 /api/v1/cloud/*」，于是被删那一侧仅有的能力必须搬进活表面，
// 否则收敛就是能力退化。本测试把裁决后的契约钉在**生产注册的路由**上：
//
//  1. 云资源列表 = 标准五键信封 + 真实 Count + updated_at DESC/id ASC 确定性排序；
//  2. 云账号/云服务列表作为选择器数据源整份返回，只给诚实的 {items,total}，不得补假分页键；
//  3. service_id 别名不再被静默接受，写错的 serviceId 是 400/1001 而不是解析失败当 0；
//  4. provider/cloudAccountId/region/status/search 过滤与租户收敛同时成立；
//  5. credential_ref 只以 hasCredential 外露（取代 dto 层已随死映射一起删除的那条守卫）；
//  6. 旧 /api/v1/cloud/* 在真实 Router 上确实是 404。

// cloudSurfaceTime 让全部种子同秒：排序列 updated_at 非唯一，缺 ID 并列键时页边界归属不确定。
var cloudSurfaceTime = time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC)

const (
	cloudSurfaceSecretID    = "cloud-surface-secret-id"
	cloudSurfaceSecretValue = "cloud-surface-secret-value"
)

type cloudSurfaceTenant struct {
	tenantID    int
	adminID     int
	adminName   string
	accountID   int
	serviceIDs  []int
	resourceIDs []int
}

type cloudResourceListWire struct {
	Items []struct {
		ID             int    `json:"id"`
		ResourceID     string `json:"resourceId"`
		CloudAccountID int    `json:"cloudAccountId"`
		ServiceID      int    `json:"serviceId"`
		Region         string `json:"region"`
		Status         string `json:"status"`
	} `json:"items"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PageSize   int  `json:"pageSize"`
	TotalPages int  `json:"totalPages"`
	Size       *int `json:"size"`
}

func cloudResourceItemIDs(res cloudResourceListWire) []int {
	ids := make([]int, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

// seedCloudSurfaceTenant 建租户 + super_admin，再种 1 个带凭据引用的云账号、2 个云服务、
// count 个同秒云资源。资源分配刻意可被过滤条件切分：
// service[0] 承载前 15 条、service[1] 承载其余；region 前 20 条 cn-hangzhou、其余 cn-shanghai；
// status 最后 1 条 stopped；resourceId 为 i-00..i-(count-1)。
func seedCloudSurfaceTenant(ctx context.Context, t *testing.T, client *ent.Client, code, provider string, count int) cloudSurfaceTenant {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Cloud surface " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantID := tenant.ID

	user, err := client.User.Create().
		SetUsername(code + "-admin").
		SetEmail(code + "-admin@example.com").
		SetName(code + "-admin").
		SetPasswordHash("hash").
		SetRole("super_admin").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	account, err := client.CloudAccount.Create().
		SetProvider(provider).
		SetAccountID(code + "-acct").
		SetAccountName(code + " production").
		SetCredentialRef(fmt.Sprintf(`{"access_key_id":%q,"access_key_secret":%q}`, cloudSurfaceSecretID, cloudSurfaceSecretValue)).
		SetTenantID(tenantID).
		SetCreatedAt(cloudSurfaceTime).
		SetUpdatedAt(cloudSurfaceTime).
		Save(ctx)
	require.NoError(t, err)

	var serviceIDs []int
	for _, svc := range []struct{ code, name, typeCode, typeName string }{
		{"ecs", "ECS", "instance", "Instance"},
		{"rds", "RDS", "db_instance", "DB Instance"},
	} {
		e, err := client.CloudService.Create().
			SetProvider(provider).
			SetServiceCode(svc.code).
			SetServiceName(svc.name).
			SetResourceTypeCode(svc.typeCode).
			SetResourceTypeName(svc.typeName).
			SetTenantID(tenantID).
			SetCreatedAt(cloudSurfaceTime).
			SetUpdatedAt(cloudSurfaceTime).
			Save(ctx)
		require.NoError(t, err)
		serviceIDs = append(serviceIDs, e.ID)
	}

	fixture := cloudSurfaceTenant{
		tenantID:   tenantID,
		adminID:    user.ID,
		adminName:  user.Username,
		accountID:  account.ID,
		serviceIDs: serviceIDs,
	}

	for i := 0; i < count; i++ {
		serviceID := serviceIDs[1]
		if i < 15 {
			serviceID = serviceIDs[0]
		}
		region := "cn-shanghai"
		if i < 20 {
			region = "cn-hangzhou"
		}
		status := "running"
		if i == count-1 {
			status = "stopped"
		}
		e, err := client.CloudResource.Create().
			SetCloudAccountID(account.ID).
			SetServiceID(serviceID).
			SetResourceID(fmt.Sprintf("i-%02d", i)).
			SetRegion(region).
			SetStatus(status).
			SetTenantID(tenantID).
			SetCreatedAt(cloudSurfaceTime).
			SetUpdatedAt(cloudSurfaceTime).
			Save(ctx)
		require.NoError(t, err)
		fixture.resourceIDs = append(fixture.resourceIDs, e.ID)
	}
	sort.Ints(fixture.resourceIDs)
	return fixture
}

func TestCMDBCloudListSurfaceRouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_cmdb_cloud_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	r := setupCMDBListRouter(t, client, logger)

	a := seedCloudSurfaceTenant(ctx, t, client, "cloud-a", "aliyun", 25)
	b := seedCloudSurfaceTenant(ctx, t, client, "cloud-b", "aws", 5)

	// 同秒种子的有效全序就是 ID 升序。
	allA := a.resourceIDs
	allB := b.resourceIDs
	require.Len(t, allA, 25)
	require.Len(t, allB, 5)

	get := func(t *testing.T, tenant cloudSurfaceTenant, path string) (int, string, cmdbListEnvelope) {
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
	// rawGet 不解析信封，用于断言非 JSON 的传输层结果（如已注销路由的 404）。
	rawGet := func(t *testing.T, tenant cloudSurfaceTenant, path string) (int, string) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(tenant.adminID, tenant.adminName, "super_admin", tenant.tenantID, cmdbListSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
	resources := func(t *testing.T, tenant cloudSurfaceTenant, query string) (int, string, cloudResourceListWire) {
		t.Helper()
		path := "/api/v1/cmdb/cloud-resources"
		if query != "" {
			path += "?" + query
		}
		status, body, env := get(t, tenant, path)
		require.Equal(t, http.StatusOK, status, "body=%s", body)
		require.Equal(t, 0, env.Code, body)
		var res cloudResourceListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		return status, body, res
	}

	t.Run("云资源列表只返回标准五键", func(t *testing.T) {
		_, body, env := get(t, a, "/api/v1/cmdb/cloud-resources")
		require.Equal(t, 0, env.Code, body)
		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, envelopeKeys(t, env.Data), body)
	})

	t.Run("云账号与云服务列表只给诚实的 items/total", func(t *testing.T) {
		for _, path := range []string{"/api/v1/cmdb/cloud-accounts", "/api/v1/cmdb/cloud-services"} {
			_, body, env := get(t, a, path)
			require.Equal(t, 0, env.Code, body)
			// 这两条是 CI 表单的选择器数据源，整份返回；刻意不分页就不得伪造 page 键。
			assert.Equal(t, []string{"items", "total"}, envelopeKeys(t, env.Data),
				"%s 必须只含 items/total: body=%s", path, body)
		}
	})

	t.Run("默认页长 1/20 且 totalPages 如实算出", func(t *testing.T) {
		_, body, res := resources(t, a, "")
		assert.Equal(t, 1, res.Page, body)
		assert.Equal(t, 20, res.PageSize, body)
		assert.Equal(t, 25, res.Total, "total 必须是全量条数，不是当前页长度: body=%s", body)
		assert.Equal(t, 2, res.TotalPages, body)
		assert.Len(t, res.Items, 20, body)
		assert.Equal(t, allA[:20], cloudResourceItemIDs(res), body)
		assert.Nil(t, res.Size, "响应不得回显 size: body=%s", body)
	})

	t.Run("三页拼接不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			_, body, res := resources(t, a, fmt.Sprintf("page=%d&pageSize=10", page))
			assert.Equal(t, page, res.Page, body)
			assert.Equal(t, 3, res.TotalPages, body)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want, body)
			paged = append(paged, cloudResourceItemIDs(res)...)
		}
		assert.Equal(t, allA, paged, "updated_at 同秒时必须由 ID 并列键兜住页边界")

		_, body, res := resources(t, a, "page=4&pageSize=10")
		assert.Empty(t, res.Items, "越界页码必须返回空: body=%s", body)
	})

	t.Run("越界与非法页长回落默认页长", func(t *testing.T) {
		for _, tc := range []struct {
			name                          string
			query                         string
			wantPage, wantSize, wantItems int
		}{
			{"pageSize 超上限", "pageSize=300", 1, 20, 20},
			{"pageSize=0", "page=1&pageSize=0", 1, 20, 20},
			{"page 为负", "page=-1&pageSize=10", 1, 10, 10},
			{"page/pageSize 非数字", "page=abc&pageSize=xyz", 1, 20, 20},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, body, res := resources(t, a, tc.query)
				assert.Equal(t, tc.wantPage, res.Page, body)
				assert.Equal(t, tc.wantSize, res.PageSize, "非法页长必须回落到生效值: body=%s", body)
				assert.Len(t, res.Items, tc.wantItems, "非法页长不得退化成不加 LIMIT: body=%s", body)
			})
		}
	})

	t.Run("service_id 别名不再被接受", func(t *testing.T) {
		// 修复前：handler 先读 serviceId 再回退 service_id，snake_case 别名继续生效。
		_, body, res := resources(t, a, fmt.Sprintf("service_id=%d&pageSize=100", a.serviceIDs[0]))
		assert.Equal(t, 25, res.Total, "snake_case 别名不得参与过滤: body=%s", body)
		assert.Len(t, res.Items, 25, body)
	})

	t.Run("serviceId 非法值返回 400/1001 而不是静默当 0", func(t *testing.T) {
		status, body, env := get(t, a, "/api/v1/cmdb/cloud-resources?serviceId=abc")
		assert.Equal(t, http.StatusBadRequest, status, body)
		assert.Equal(t, 1001, env.Code, body)
		assert.Equal(t, "请求参数错误", env.Message, "响应必须给稳定公开文案: body=%s", body)
		assert.NotContains(t, body, "strconv", body)
		assert.NotContains(t, body, "invalid syntax", body)
	})

	t.Run("过滤条件按契约生效", func(t *testing.T) {
		cases := []struct {
			name  string
			query string
			total int
		}{
			{"serviceId", fmt.Sprintf("serviceId=%d&pageSize=100", a.serviceIDs[0]), 15},
			{"cloudAccountId", fmt.Sprintf("cloudAccountId=%d&pageSize=100", a.accountID), 25},
			{"region", "region=cn-shanghai&pageSize=100", 5},
			{"status", "status=stopped&pageSize=100", 1},
			{"search 前缀", "search=i-2&pageSize=100", 5},
			{"provider 命中", "provider=aliyun&pageSize=100", 25},
			{"provider 不命中", "provider=aws&pageSize=100", 0},
			{"组合过滤", fmt.Sprintf("serviceId=%d&region=cn-hangzhou&pageSize=100", a.serviceIDs[0]), 15},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, body, res := resources(t, a, tc.query)
				assert.Equal(t, tc.total, res.Total, body)
				assert.Len(t, res.Items, tc.total, body)
			})
		}
	})

	t.Run("租户 B 只见自己的资源", func(t *testing.T) {
		_, body, res := resources(t, b, "pageSize=100")
		assert.Equal(t, 5, res.Total, body)
		assert.Equal(t, allB, cloudResourceItemIDs(res), body)
		for _, id := range cloudResourceItemIDs(res) {
			assert.NotContains(t, allA, id, "租户 B 的响应里不得混入租户 A 的云资源")
		}

		// provider 过滤走账号边：B 的账号是 aws，用 aliyun 过滤必须为空而不是跨租户命中 A。
		_, body, env := get(t, b, "/api/v1/cmdb/cloud-resources?provider=aliyun&pageSize=100")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		var filtered cloudResourceListWire
		require.NoError(t, json.Unmarshal(env.Data, &filtered), body)
		assert.Zero(t, filtered.Total, "跨租户 provider 过滤不得命中租户 A 的资源: body="+body)
	})

	t.Run("空结果序列化为 []", func(t *testing.T) {
		_, body, env := get(t, a, "/api/v1/cmdb/cloud-resources?search=no-such-resource")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		var res cloudResourceListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 0, res.Total, body)
		assert.Equal(t, 0, res.TotalPages, body)
	})

	t.Run("云账号不外泄凭据引用", func(t *testing.T) {
		status, body, env := get(t, a, "/api/v1/cmdb/cloud-accounts")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)
		assert.Contains(t, body, `"hasCredential":true`, "有凭据必须可见为布尔标记: body="+body)
		assert.NotContains(t, body, "credentialRef", "凭据引用本身不得出接口边界: body="+body)
		assert.NotContains(t, body, cloudSurfaceSecretID, body)
		assert.NotContains(t, body, cloudSurfaceSecretValue, body)

		var data struct {
			Items []map[string]interface{} `json:"items"`
			Total int                      `json:"total"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &data), body)
		require.Len(t, data.Items, 1, body)
		assert.Equal(t, 1, data.Total, body)
		assert.Equal(t, "aliyun", data.Items[0]["provider"], body)
	})

	t.Run("云服务列表整份返回", func(t *testing.T) {
		_, body, env := get(t, a, "/api/v1/cmdb/cloud-services")
		require.Equal(t, 0, env.Code, body)
		var data struct {
			Items []map[string]interface{} `json:"items"`
			Total int                      `json:"total"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &data), body)
		assert.Equal(t, 2, data.Total, body)
		assert.Len(t, data.Items, 2, body)
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/cmdb/cloud-resources",
			"/api/v1/cmdb/cloud-accounts",
			"/api/v1/cmdb/cloud-services",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, path+" body="+w.Body.String())
		}
	})

	t.Run("旧 /api/v1/cloud 表面已注销", func(t *testing.T) {
		// 被删的是同一用例的第二套业务规则所有者；真实 Router 上必须彻底不存在。
		for _, path := range []string{
			"/api/v1/cloud/accounts", "/api/v1/cloud/services", "/api/v1/cloud/resources",
		} {
			status, body := rawGet(t, a, path)
			assert.Equal(t, http.StatusNotFound, status, "旧表面路由必须已注销: path=%s body=%s", path, body)
		}
	})
}
