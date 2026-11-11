package skill_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/handlers/skill"
	"itsm-backend/middleware"
	marketplaceService "itsm-backend/service/marketplace"
)

// 技能调用入口的市场交付门禁测试。
//
// 这条链把「租户装了什么」变成运行时约束：目录里已上架的技能商品声明了某个技能码，
// 租户必须先装上才能调，卸载即撤权。测试走 handler 的真实路由而不是直接调 service，
// 因为门禁挂在 Invoke 上——只测 service 判定证明不了调用真的被挡住。

type entitlementFixture struct {
	router  *gin.Engine
	handler *skill.Handler
	client  *ent.Client
	svc     *marketplaceService.Service
	tenant  int
}

func newEntitlementFixture(t *testing.T, dsn string) *entitlementFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// RequiredPermissions 的判定切到 HardcodeOnly（与 handler_test.go 一致），
	// 让本文件只反映交付门禁这一层，权限声明由 handler_test.go 单独锁定。
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly

	client := enttest.Open(t, "sqlite3", "file:"+dsn+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	registry := skill.NewSkillRegistry()
	svc := marketplaceService.NewService(client, zaptest.NewLogger(t).Sugar())
	// 运行时代码集合直接用真实注册表：注册了哪些码，目录就只能声明哪些定位符。
	svc.SetSkillRuntime(registry)

	h := skill.NewHandler(registry, zap.NewNop().Sugar(), client)
	h.SetSkillEntitlements(svc)

	f := &entitlementFixture{handler: h, client: client, svc: svc, tenant: 1}

	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		// tenant 取自 fixture，用于切换调用方身份；身份仍只来自认证上下文。
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: f.tenant})
		c.Set("tenant_id", f.tenant)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	v1.POST("/admin/skills", h.Create)
	v1.POST("/admin/skills/:code/invoke", h.Invoke)
	f.router = r
	return f
}

// registerRuntimeSkill 用真实注册路由把技能码放进运行时注册表。
// 走路由而不是手搭 manifest，避免测试自建一份与生产不同的注册路径。
func (f *entitlementFixture) registerRuntimeSkill(t *testing.T, code string) {
	t.Helper()
	w := entitlementRequest(t, f.router, http.MethodPost, "/api/v1/admin/skills", map[string]any{
		"code":                code,
		"version":             "v1",
		"title":               code,
		"requiredPermissions": []string{"marketplace:read"},
		"executor":            map[string]any{"target": "echo"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// publishSkillItem 建一个已上架的技能商品，其 1.0.0 版本行交付指定技能码。
func (f *entitlementFixture) publishSkillItem(t *testing.T, name, code string) int {
	t.Helper()
	ctx := context.Background()
	item, err := f.client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeSkill).
		SetTitle(name).
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	_, err = f.client.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion("1.0.0").
		SetStatus(itemversion.StatusPublished).
		SetManifestPath("skill://" + code).
		SetReleasedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return item.ID
}

func (f *entitlementFixture) invoke(t *testing.T, code string) *httptest.ResponseRecorder {
	t.Helper()
	return entitlementRequest(t, f.router, http.MethodPost,
		"/api/v1/admin/skills/"+code+"/invoke", map[string]any{"input": map[string]any{"q": "1"}})
}

func entitlementRequest(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewBuffer(raw)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload), "body=%s", w.Body.String())
	return payload
}

// TestInvoke_DeliveredSkillRequiresInstallation 目录交付的技能码：未装 403、装后 200、卸载再 403。
func TestInvoke_DeliveredSkillRequiresInstallation(t *testing.T) {
	f := newEntitlementFixture(t, "skill_gate_lifecycle")
	ctx := context.Background()

	f.registerRuntimeSkill(t, "ai.triage")
	itemID := f.publishSkillItem(t, "triage-skill", "ai.triage")

	// 未安装：技能确实在运行时里，但目录把它变成了商品，租户没装就没有调用权。
	w := f.invoke(t, "ai.triage")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	msg, _ := decodeEnvelope(t, w)["message"].(string)
	assert.Contains(t, msg, "triage-skill", "拒绝文案要指出该装哪个商品")

	// 装上即开通。
	_, err := f.svc.InstallItem(ctx, 1, itemID, "1001")
	require.NoError(t, err)
	w = f.invoke(t, "ai.triage")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data, ok := decodeEnvelope(t, w)["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ai.triage", data["code"])

	// 卸载即撤权：调用权跟着安装记录走，不是首次放行后永久有效。
	require.NoError(t, f.svc.UninstallItem(ctx, 1, itemID))
	w = f.invoke(t, "ai.triage")
	assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

// TestInvoke_CrossTenantInstallationDoesNotGrant B 租户的安装记录不能给 A 租户开调用权。
func TestInvoke_CrossTenantInstallationDoesNotGrant(t *testing.T) {
	f := newEntitlementFixture(t, "skill_gate_cross_tenant")
	ctx := context.Background()

	f.registerRuntimeSkill(t, "ai.triage")
	itemID := f.publishSkillItem(t, "triage-skill", "ai.triage")

	_, err := f.svc.InstallItem(ctx, 2, itemID, "2001")
	require.NoError(t, err)

	f.tenant = 1
	w := f.invoke(t, "ai.triage")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	f.tenant = 2
	w = f.invoke(t, "ai.triage")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestInvoke_SkillNotDeliveredByCatalogIsNotGated 目录里没有商品交付的技能码不受市场约束：
// 内置能力不能因为「没上架」而被突然下线。
func TestInvoke_SkillNotDeliveredByCatalogIsNotGated(t *testing.T) {
	f := newEntitlementFixture(t, "skill_gate_not_delivered")

	f.registerRuntimeSkill(t, "user.echo")
	w := f.invoke(t, "user.echo")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// entitlementCheckerStub 用于验证「判定失败即拒绝」，不依赖市场服务的内部错误路径。
type entitlementCheckerStub struct {
	allowed bool
	err     error
	calls   int
}

func (s *entitlementCheckerStub) SkillInvocationEntitlement(ctx context.Context, tenantID int, code string) (marketplaceService.SkillEntitlement, error) {
	s.calls++
	return marketplaceService.SkillEntitlement{Allowed: s.allowed, Reason: "stub"}, s.err
}

// TestInvoke_EntitlementCheckFailureFailsClosed 判定本身报错时必须拒绝执行。
//
// 「查不动就放行」会让安装记录变成装饰，也让卸载撤权失效：DB 抖动的一分钟内，
// 已卸载的租户照样能调用付费交付的技能。
func TestInvoke_EntitlementCheckFailureFailsClosed(t *testing.T) {
	f := newEntitlementFixture(t, "skill_gate_checker_error")
	f.registerRuntimeSkill(t, "ai.triage")

	stub := &entitlementCheckerStub{err: errors.New("installation store unavailable")}
	f.handler.SetSkillEntitlements(stub)

	w := f.invoke(t, "ai.triage")
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.calls, "门禁必须先于执行发生")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Nil(t, payload["data"], "校验失败不得执行技能，也就没有 output")
	// 内部错误原因只进日志，不随响应外泄。
	assert.NotContains(t, w.Body.String(), "installation store unavailable")
}

// TestNewHandlerKeepsEntClient 锁定构造函数保留 client。
//
// client 一旦在构造时被丢弃，DBOnly 模式会判定成「DB 不可用」并 fail-closed，
// 所有声明了 RequiredPermissions 的 skill 一律 403；编译和既有单测都不会报错
// （它们跑在 HardcodeOnly 或 nil client 下），所以这里用「有 client=放行」这一侧
// 把装配钉住：client 为 nil 时同一条请求会变成 403。
func TestNewHandlerKeepsEntClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:skill_new_handler_client?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	// DBOnly：角色在库里没有任何授权行 → 「未配置」分支 → 走 authz 默认权限兜底。
	// 这条分支必须有 client 才能走到，因此它就是本测试的观测点。
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeDBOnly
	t.Cleanup(func() { middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly })

	h := skill.NewHandler(skill.NewSkillRegistry(), zap.NewNop().Sugar(), client)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 9})
		c.Set("tenant_id", 9)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	v1.POST("/admin/skills", h.Create)
	v1.POST("/admin/skills/:code/invoke", h.Invoke)

	require.Equal(t, http.StatusOK, entitlementRequest(t, r, http.MethodPost, "/api/v1/admin/skills", map[string]any{
		"code":                "client.probe",
		"version":             "v1",
		"requiredPermissions": []string{"marketplace:read"},
		"executor":            map[string]any{"target": "echo"},
	}).Code)

	w := entitlementRequest(t, r, http.MethodPost, "/api/v1/admin/skills/client.probe/invoke", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 对照组：同一个 handler 构造时不传 client，同一条请求必须被 fail-closed 拒绝。
	broken := skill.NewHandler(skill.NewSkillRegistry(), zap.NewNop().Sugar(), nil)
	r2 := gin.New()
	v2 := r2.Group("/api/v1")
	v2.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 9})
		c.Set("tenant_id", 9)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	v2.POST("/admin/skills", broken.Create)
	v2.POST("/admin/skills/:code/invoke", broken.Invoke)
	require.Equal(t, http.StatusOK, entitlementRequest(t, r2, http.MethodPost, "/api/v1/admin/skills", map[string]any{
		"code":                "client.probe",
		"version":             "v1",
		"requiredPermissions": []string{"marketplace:read"},
		"executor":            map[string]any{"target": "echo"},
	}).Code)
	assert.Equal(t, http.StatusForbidden,
		entitlementRequest(t, r2, http.MethodPost, "/api/v1/admin/skills/client.probe/invoke", nil).Code,
		"client 丢失时这里才会 403；两组不一致才证明构造函数真的保留了 client")
}
