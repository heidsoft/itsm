// Package asset 的跨租户 HTTP 回归测试（B8.1：asset + license 双面）。
//
// 权限码：/assets* 用 asset:{read,write,delete}；/licenses* 用
// license:{read,write,delete}；fixture 用户统一 super_admin（RBAC 直通，与
// B2/B4 基线一致），本文件只测租户隔离与错误形态，不测 RBAC 授权面。
//
// 契约（B8.2 修复后语义，扫描方以此为基线）：
//   - 资产与许可证的 Get / Update / UpdateStatus / Delete / Assign / Retire /
//     LicenseAssign 跨租户或不存在一律 404/4004 掩码（service sentinel
//     ErrAssetNotFound / ErrLicenseNotFound + 条件删除验 affected rows）。
//   - [L3 已修复] PUT /licenses/:id/assign 的 userIDs 只保留本租户真实存在的
//     用户 id；跨租户或无效 id 静默跳过（与站内通知收件人语义一致），不写
//     license.Users、不占席位。
//   - [L4 已修复] 席位不足 / 并发扣减冲突映射 409/4090（业务冲突，不再 500）。
//   - [L5 已修复] PUT /licenses/:id 拒绝直接写 usedQuantity（4000）；替换 users 时按
//     去重后的租户内用户数重算已用席位，超出总数为 4090 且整单拒绝、不部分写入。
//   - 同租分配去重 + 幂等：重复提交同一批用户不得再扣席位。
package asset_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/asset"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/enttest"
	assetHandler "itsm-backend/handlers/asset"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine   *gin.Engine
	client   *ent.Client
	secret   string
	tenantA  *ent.Tenant
	tenantB  *ent.Tenant
	userA    *ent.User
	stranger *ent.User
	peerB    *ent.User
	assetA1  *ent.Asset
	assetA2  *ent.Asset
	assetB1  *ent.Asset
	licA1    *ent.AssetLicense
	licB1    *ent.AssetLicense
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:asset_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Asset A").SetCode("asset-a").SetDomain("asset-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Asset B").SetCode("asset-b").SetDomain("asset-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("asset-a-owner").SetEmail("asset-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("asset-b-stranger").SetEmail("asset-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	peerB := client.User.Create().SetUsername("asset-b-peer").SetEmail("asset-b-peer@example.com").SetName("Peer B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	mkAsset := func(number, name string, tenant *ent.Tenant, status string) *ent.Asset {
		return client.Asset.Create().
			SetAssetNumber(number).
			SetName(name).
			SetStatus(status).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}
	assetA1 := mkAsset("ASSET-A-001", "租户A资产一", tenantA, "available")
	assetA2 := mkAsset("ASSET-A-002", "租户A资产二", tenantA, "in-use")
	assetB1 := mkAsset("ASSET-B-001", "租户B资产", tenantB, "available")

	licA1 := client.AssetLicense.Create().
		SetName("租户A许可证").
		SetTotalQuantity(5).
		SetStatus("active").
		SetTenantID(tenantA.ID).
		SaveX(ctx)
	licB1 := client.AssetLicense.Create().
		SetName("租户B许可证").
		SetTotalQuantity(1).
		SetStatus("active").
		SetTenantID(tenantB.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "asset-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:    secret,
		Logger:       logger,
		Client:       client,
		AssetHandler: assetHandler.NewHandler(service.NewAssetService(client, logger), service.NewAssetLicenseService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, stranger: stranger, peerB: peerB,
		assetA1: assetA1, assetA2: assetA2, assetB1: assetB1,
		licA1: licA1, licB1: licB1,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, payload ...string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var body *bytes.Reader
	body = bytes.NewReader(nil)
	if len(payload) > 0 {
		body = bytes.NewReader([]byte(payload[0]))
	}
	req := httptest.NewRequest(method, path, body)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

// TestAssetTenantIsolation 覆盖列表精度 + 读面 404 + 写面钉住缺陷。
func TestAssetTenantIsolation(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/assets"

	t.Run("同租列表正控制total精确", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		var listing struct {
			Items []struct {
				ID          int    `json:"id"`
				AssetNumber string `json:"assetNumber"`
				Name        string `json:"name"`
				Status      string `json:"status"`
			} `json:"items"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &listing), "data=%s", string(env.Data))
		require.Equal(t, 2, listing.Total, "total 必须精确等于本租户资产数")
		for _, item := range listing.Items {
			require.False(t, strings.Contains(item.AssetNumber, "ASSET-B"), "不得混入他租户资产")
		}
	})

	t.Run("同租Get是正控制且字段camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, fx.assetA1.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"assetNumber":"ASSET-A-001"`)
		require.Contains(t, w.Body.String(), `"tenantId"`, "字段必须是 camelCase")
		require.NotContains(t, w.Body.String(), `"asset_id"`, "不得出现 snake_case 字段")
	})

	t.Run("跨租户Get与Update与StatusUpdate均404掩码", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, fx.assetB1.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.assetB1.ID), fx.userA, `{"name":"越权改名"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/status", base, fx.assetB1.ID), fx.userA, `{"status":"retired"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		// 对端资产状态不得被触碰
		a, err := fx.client.Asset.Get(context.Background(), fx.assetB1.ID)
		require.NoError(t, err)
		require.Equal(t, "available", a.Status)
	})

	t.Run("跨租户Delete是404掩码且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, fx.assetB1.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		// 对端行必须还在
		n, err := fx.client.Asset.Query().Where(asset.TenantIDEQ(fx.tenantB.ID)).Count(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, n, "跨租户删除不得注销对端行")
	})

	t.Run("跨租户Assign是404掩码且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.assetB1.ID), fx.userA, `{"assignedTo":`+fmt.Sprint(fx.userA.ID)+`}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		a, err := fx.client.Asset.Get(context.Background(), fx.assetB1.ID)
		require.NoError(t, err)
		require.Equal(t, "available", a.Status, "跨租户 assign 不得改变对端状态")
	})

	t.Run("跨租户Retire是404掩码且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/retire", base, fx.assetB1.ID), fx.userA, `{"retireDate":"2026-10-18","retireReason":"越权退役"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		a, err := fx.client.Asset.Get(context.Background(), fx.assetB1.ID)
		require.NoError(t, err)
		require.Equal(t, "available", a.Status)
	})
}

func TestAssetStatsAndLifecycle(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/assets"

	t.Run("stats正控制精确计数不含他租", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/stats", base), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		var stats struct {
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &stats), "data=%s", string(env.Data))
		require.Equal(t, 2, stats.Total, "stats.total 必须精确等于本租户资产数")
	})

	t.Run("同租生命周期正控制创建分配退役", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA,
			`{"assetNumber":"ASSET-A-NEW","name":"新建资产","type":"hardware","category":"it"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		var created struct {
			ID          int    `json:"id"`
			AssetNumber string `json:"assetNumber"`
			Status      string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created))
		require.Equal(t, "ASSET-A-NEW", created.AssetNumber)
		require.Equal(t, "available", created.Status, "新资产默认 available")

		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, created.ID), fx.userA,
			`{"assignedTo":`+fmt.Sprint(fx.userA.ID)+`}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var assigned struct {
			Status     string `json:"status"`
			AssignedTo *int   `json:"assignedTo"`
		}
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, w).Data, &assigned))
		require.Equal(t, "in-use", assigned.Status, "assign 后进入 in-use")
		require.NotNil(t, assigned.AssignedTo)
		require.Equal(t, fx.userA.ID, *assigned.AssignedTo)

		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/retire", base, created.ID), fx.userA, `{"retireDate":"2026-10-18","retireReason":"折旧"}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"retired"`)
	})

	t.Run("同租重复删除第二次是404不再假装成功", func(t *testing.T) {
		asset := fx.client.Asset.Create().SetAssetNumber("ASSET-A-DEL").SetName("待删资产").SetStatus("available").SetTenantID(fx.tenantA.ID).SaveX(context.Background())
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, asset.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, asset.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})
}

func TestLicenseTenantIsolation(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/licenses"
	ctx := context.Background()

	t.Run("同租列表正控制且跨租户Get404", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		var listing struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
			Total int `json:"total"`
		}
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		require.NoError(t, json.Unmarshal(env.Data, &listing), "data=%s", string(env.Data))
		require.Equal(t, 1, listing.Total)
		require.Equal(t, fx.licA1.ID, listing.Items[0].ID)

		w = fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, fx.licB1.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("跨租户Update404", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.licB1.ID), fx.userA, `{"name":"越权改"}`)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("直接改已用数量是400且零副作用", func(t *testing.T) {
		// used_quantity 只能由分配/替换用户推导，不允许经通用更新直接写入。
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.licA1.ID), fx.userA, `{"usedQuantity":3}`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4000, decodeEnvelope(t, w).Code)
		lic, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		require.Equal(t, 0, lic.UsedQuantity, "被拒绝的更新不得改已用数")
	})

	t.Run("替换用户含跨租户id整单400", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.licA1.ID), fx.userA,
			`{"users":[`+fmt.Sprint(fx.userA.ID)+`,`+fmt.Sprint(fx.stranger.ID)+`]}`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4000, decodeEnvelope(t, w).Code)
		lic, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		require.Empty(t, lic.Users, "整单拒绝不得部分写入")
		require.Equal(t, 0, lic.UsedQuantity)
	})

	t.Run("缩容到已分配以下409", func(t *testing.T) {
		lic := fx.client.AssetLicense.Create().SetName("待缩容许可证").SetTotalQuantity(5).
			SetUsers([]int{fx.userA.ID}).SetUsedQuantity(1).
			SetStatus("active").SetTenantID(fx.tenantA.ID).SaveX(ctx)
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, lic.ID), fx.userA, `{"totalQuantity":0}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)
		after, err := fx.client.AssetLicense.Get(ctx, lic.ID)
		require.NoError(t, err)
		require.Equal(t, 5, after.TotalQuantity)
	})

	t.Run("跨租户Delete是404掩码且零副作用", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, fx.licB1.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		n, err := fx.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(fx.tenantB.ID)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, n, "跨租户删除不得注销对端行")
	})

	t.Run("跨租户license分配是404掩码", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.licB1.ID), fx.userA, `{"userIds":[1]}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("跨租户userId被静默跳过不占席位", func(t *testing.T) {
		// [L3 已修复] userIDs 校验租户归属：跨租户用户 id 静默跳过（与
		// 站内通知收件人语义一致），不写 license.Users、不扣席位。
		before, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.licA1.ID), fx.userA,
			`{"userIds":[`+fmt.Sprint(fx.stranger.ID)+`]}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		after, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		require.NotContains(t, after.Users, fx.stranger.ID, "跨租户用户不得进入 license.Users")
		require.Equal(t, before.UsedQuantity, after.UsedQuantity, "无效/跨租户 id 不得扣席位")
	})

	t.Run("席位不足是409业务冲突", func(t *testing.T) {
		// [L4 已修复] 需求席位（tenantB 两个有效用户 =2）> totalQuantity(1) → 409。
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.licB1.ID), fx.stranger,
			`{"userIds":[`+fmt.Sprint(fx.stranger.ID)+`,`+fmt.Sprint(fx.peerB.ID)+`]}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)
		// 冲突不得扣席位
		lic, err := fx.client.AssetLicense.Get(ctx, fx.licB1.ID)
		require.NoError(t, err)
		require.Equal(t, 0, lic.UsedQuantity)
		require.Empty(t, lic.Users)
	})

	t.Run("同租分配正控制去重跳过跨租户id幂等", func(t *testing.T) {
		// [userA, userA, stranger] → 去重 [userA, stranger] → 租户过滤 [userA]，只扣 1 席
		ids := fmt.Sprintf(`{"userIds":[%d,%d,%d]}`, fx.userA.ID, fx.userA.ID, fx.stranger.ID)
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.licA1.ID), fx.userA, ids)
		require.Equal(t, http.StatusOK, w.Code)
		lic, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []int{fx.userA.ID}, lic.Users, "重复 id 去重、跨租户 id 被过滤")
		require.Equal(t, 1, lic.UsedQuantity)
		// 重复分配同一批用户 → needed=0 no-op，不再扣减
		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d/assign", base, fx.licA1.ID), fx.userA, ids)
		require.Equal(t, http.StatusOK, w.Code)
		lic2, err := fx.client.AssetLicense.Get(ctx, fx.licA1.ID)
		require.NoError(t, err)
		require.Equal(t, lic.UsedQuantity, lic2.UsedQuantity, "幂等：已分配用户重复提交不得再扣席位")
	})

	t.Run("同租重复删除第二次是404不再假装成功", func(t *testing.T) {
		lic := fx.client.AssetLicense.Create().SetName("待删许可证").SetTotalQuantity(1).SetStatus("active").SetTenantID(fx.tenantA.ID).SaveX(ctx)
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, lic.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, lic.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})
}
