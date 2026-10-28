package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
	"itsm-backend/middleware"
	marketplaceService "itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// handlerStubConnector 只有带 token 才能 Init 成功，用来制造
// 「安装记录已建但连接器没起来」的真实故障形态。
type handlerStubConnector struct {
	name string
}

func (s *handlerStubConnector) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                s.name,
		Version:             "1.0.0",
		Title:               s.name,
		Type:                connector.TypeIM,
		RequiredPermissions: []string{"connector:write"},
	}
}

func (s *handlerStubConnector) Init(_ context.Context, cfg connector.Config) error {
	if cfg.Credentials["token"] == "" {
		return errors.New("credentials.token is required")
	}
	return nil
}

func (s *handlerStubConnector) Send(context.Context, *connector.Message) error { return nil }

func (s *handlerStubConnector) HealthCheck(context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true, CheckedAt: time.Now()}
}

func (s *handlerStubConnector) Close() error { return nil }

type handlerFixture struct {
	handler *Handler
	itemID  int
	client  *ent.Client
}

func newHandlerFixture(t *testing.T, name string) *handlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:marketplace_handler_"+name+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	registry := connector.NewRegistry()
	registry.Register(func() connector.Connector { return &handlerStubConnector{name: name} })
	manager := connector.NewManager(registry, zaptest.NewLogger(t).Sugar())

	svc := marketplaceService.NewService(client, zaptest.NewLogger(t).Sugar())
	svc.SetConnectorManager(manager)

	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeConnector).
		SetTitle(name).
		SetProvider("Open Source").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	return &handlerFixture{handler: NewHandler(svc), itemID: item.ID, client: client}
}

// newContext 构造带租户/用户上下文的 gin 上下文（放行 middleware 的 fail-closed 校验）。
func newContext(method, target, body, itemID string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: itemID}}
	c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
	c.Set("user_id", 1001)
	c.Set("tenant_id", 1)
	return c, rec
}

// TestInstallItemHandler_Returns200WithFailedStatus 锁住 handler 的诚实语义：
// 连接器没起来时不能折成 500「安装组件失败」，而要返回记录本体
// （status=failed + errorMessage），让前端能显示「要补配置」而不是「压根没装上」。
func TestInstallItemHandler_Returns200WithFailedStatus(t *testing.T) {
	f := newHandlerFixture(t, "handlerfail")

	c, rec := newContext(http.MethodPost, "/marketplace/items/1/install", "", strconv.Itoa(f.itemID))
	f.handler.InstallItem(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with failed installation payload, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status       string `json:"status"`
			ErrorMessage string `json:"errorMessage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Status != "failed" {
		t.Fatalf("expected status failed in payload, got %q (%s)", payload.Data.Status, rec.Body.String())
	}
	if !strings.Contains(payload.Data.ErrorMessage, "credentials.token") {
		t.Fatalf("errorMessage must carry the real cause, got %q", payload.Data.ErrorMessage)
	}
	if !strings.Contains(payload.Message, "未就绪") {
		t.Fatalf("message must tell the operator the connector is not ready, got %q", payload.Message)
	}
}

// TestUpdateConfigHandler_ConflictsWhenProvisionFails 配置没生效必须 409，
// 不能静默 200——否则用户以为新凭据已生效，实际仍在用旧的发消息。
func TestUpdateConfigHandler_ConflictsWhenProvisionFails(t *testing.T) {
	f := newHandlerFixture(t, "handlercfg")

	// 先安装（无凭据 → failed）
	c, _ := newContext(http.MethodPost, "/marketplace/items/1/install", "", strconv.Itoa(f.itemID))
	f.handler.InstallItem(c)

	// 再提交一份仍然缺少凭据的配置
	c2, rec2 := newContext(http.MethodPut, "/marketplace/installations/1/config",
		`{"credentials":{"token":""}}`, strconv.Itoa(f.itemID))
	f.handler.UpdateInstallationConfig(c2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 when new config cannot be provisioned, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "已保留原配置") {
		t.Fatalf("conflict message must state that the previous config is kept, got %s", rec2.Body.String())
	}
}

// TestInstallItemHandler_DependencyUnsatisfiedReturns409 依赖缺口是业务冲突：
// 商品本身可装，缺的是前置链条，handler 必须映射成 409 并把逐项明细交给前端，
// 而不是折成 500「安装组件失败」让管理员无从下手。
func TestInstallItemHandler_DependencyUnsatisfiedReturns409(t *testing.T) {
	f := newHandlerFixture(t, "handlerdep")

	dependent, err := f.client.MarketplaceItem.Create().
		SetName("handlerdep-dependent").
		SetType(marketplaceitem.TypeConnector).
		SetTitle("Dependent").
		SetProvider("Open Source").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create dependent item: %v", err)
	}
	if _, err := f.client.ItemVersion.Create().
		SetItemID(dependent.ID).
		SetVersion("1.0.0").
		SetDependencies(map[string]string{"handlerdep": ">=1.0.0"}).
		Save(context.Background()); err != nil {
		t.Fatalf("create dependent version: %v", err)
	}

	c, rec := newContext(http.MethodPost, "/marketplace/items/1/install", "", strconv.Itoa(dependent.ID))
	f.handler.InstallItem(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for unsatisfied dependency, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未安装") || !strings.Contains(rec.Body.String(), "handlerdep") {
		t.Fatalf("conflict message must list the missing prerequisite, got %s", rec.Body.String())
	}
}

// TestUninstallItemHandler_DependentsBlockReturns409 被依赖的组件不许卸：
// 卸掉之后依赖方仍标 active，实际调用全部失败。
func TestUninstallItemHandler_DependentsBlockReturns409(t *testing.T) {
	f := newHandlerFixture(t, "handlerundep")
	ctx := context.Background()

	dependent, err := f.client.MarketplaceItem.Create().
		SetName("handlerundep-dependent").
		SetType(marketplaceitem.TypeConnector).
		SetTitle("Dependent").
		SetProvider("Open Source").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	if err != nil {
		t.Fatalf("create dependent item: %v", err)
	}
	if _, err := f.client.ItemVersion.Create().
		SetItemID(dependent.ID).
		SetVersion("1.0.0").
		SetDependencies(map[string]string{"handlerundep": "1.0.0"}).
		Save(ctx); err != nil {
		t.Fatalf("create dependent version: %v", err)
	}
	for _, itemID := range []int{f.itemID, dependent.ID} {
		if _, err := f.client.TenantInstallation.Create().
			SetTenantID(1).
			SetItemID(itemID).
			SetInstalledVersion("1.0.0").
			SetStatus(tenantinstallation.StatusActive).
			SetInstalledBy("1001").
			Save(ctx); err != nil {
			t.Fatalf("create installation for item %d: %v", itemID, err)
		}
	}

	c, rec := newContext(http.MethodPost, "/marketplace/items/1/uninstall", "", strconv.Itoa(f.itemID))
	f.handler.UninstallItem(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when dependents are installed, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Dependent") {
		t.Fatalf("conflict message must list the dependents, got %s", rec.Body.String())
	}
}

func marketplaceRouteRequest(f *handlerFixture, tenantID int, role, method, path, body string) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("client", f.client)
		c.Set("role", role)
		c.Set("user_id", 1001)
		if tenantID > 0 {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	f.handler.RegisterRoutes(r.Group("/api/v1"))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMarketplaceRoutes_UnsupportedDelivery(t *testing.T) {
	for _, itemType := range []marketplaceitem.Type{marketplaceitem.TypeSkill, marketplaceitem.TypePlugin} {
		for _, tc := range []struct {
			name               string
			method             string
			tenantID           int
			role               string
			installationStatus tenantinstallation.Status
			wantStatus         int
			wantCode           int
		}{
			{"install", http.MethodPost, 1, "super_admin", "", http.StatusNotImplemented, common.NotImplementedCode},
			{"config", http.MethodPut, 1, "super_admin", tenantinstallation.StatusActive, http.StatusNotImplemented, common.NotImplementedCode},
			{"missing_config", http.MethodPut, 1, "super_admin", "", http.StatusNotFound, common.NotFoundCode},
			{"uninstalled_config", http.MethodPut, 1, "super_admin", tenantinstallation.StatusUninstalled, http.StatusNotFound, common.NotFoundCode},
			{"cross_tenant_config", http.MethodPut, 2, "super_admin", tenantinstallation.StatusActive, http.StatusNotFound, common.NotFoundCode},
			{"install_forbidden", http.MethodPost, 1, "marketplace_no_permissions", "", http.StatusForbidden, common.ForbiddenCode},
			{"config_forbidden", http.MethodPut, 1, "marketplace_no_permissions", tenantinstallation.StatusActive, http.StatusForbidden, common.ForbiddenCode},
			{"install_no_tenant", http.MethodPost, 0, "super_admin", "", http.StatusUnauthorized, common.AuthFailedCode},
			{"config_no_tenant", http.MethodPut, 0, "super_admin", tenantinstallation.StatusActive, http.StatusUnauthorized, common.AuthFailedCode},
		} {
			t.Run(string(itemType)+"/"+tc.name, func(t *testing.T) {
				f := newHandlerFixture(t, "unsupported_"+string(itemType)+"_"+tc.name)
				ctx := context.Background()
				_, err := f.client.MarketplaceItem.UpdateOneID(f.itemID).SetType(itemType).Save(ctx)
				require.NoError(t, err)
				itemID := strconv.Itoa(f.itemID)
				path := "/api/v1/marketplace/items/" + itemID + "/install"
				if tc.method == http.MethodPut {
					path = "/api/v1/marketplace/installations/" + itemID + "/config"
				}
				var existing *ent.TenantInstallation
				if tc.installationStatus != "" {
					existing, err = f.client.TenantInstallation.Create().
						SetTenantID(1).SetItemID(f.itemID).SetInstalledVersion("1.0.0").
						SetStatus(tc.installationStatus).SetInstalledBy("1001").
						SetConfig(map[string]interface{}{"token": "stored-test-token"}).Save(ctx)
					require.NoError(t, err)
					existing, err = f.client.TenantInstallation.Get(ctx, existing.ID)
					require.NoError(t, err)
				}

				rec := marketplaceRouteRequest(f, tc.tenantID, tc.role, tc.method, path, `{"token":"submitted-test-token"}`)
				require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
				var payload common.Response
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
				require.Equal(t, tc.wantCode, payload.Code)
				require.Nil(t, payload.Data)
				require.NotEmpty(t, payload.Message)
				require.NotContains(t, rec.Body.String(), "test-token")
				count, err := f.client.TenantInstallation.Query().Count(ctx)
				require.NoError(t, err)
				if existing == nil {
					require.Zero(t, count)
				} else {
					require.Equal(t, 1, count)
					current, err := f.client.TenantInstallation.Get(ctx, existing.ID)
					require.NoError(t, err)
					before, err := json.Marshal(existing)
					require.NoError(t, err)
					after, err := json.Marshal(current)
					require.NoError(t, err)
					require.JSONEq(t, string(before), string(after))
				}
			})
		}
	}
}

func TestMarketplaceRoutes_UnsupportedHistoryCanBeReadAndUninstalled(t *testing.T) {
	for _, itemType := range []marketplaceitem.Type{marketplaceitem.TypeSkill, marketplaceitem.TypePlugin} {
		t.Run(string(itemType), func(t *testing.T) {
			f := newHandlerFixture(t, "history_"+string(itemType))
			ctx := context.Background()
			_, err := f.client.MarketplaceItem.UpdateOneID(f.itemID).SetType(itemType).SetInstallCount(1).Save(ctx)
			require.NoError(t, err)
			installation, err := f.client.TenantInstallation.Create().
				SetTenantID(1).SetItemID(f.itemID).SetInstalledVersion("1.0.0").
				SetStatus(tenantinstallation.StatusActive).SetInstalledBy("1001").
				SetConfig(map[string]interface{}{"token": "historical-test-token"}).Save(ctx)
			require.NoError(t, err)
			itemID := strconv.Itoa(f.itemID)
			rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/installations/"+itemID, "")
			require.Equal(t, http.StatusOK, rec.Code)
			var payload struct {
				Code int                  `json:"code"`
				Data InstallationResponse `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
			require.Zero(t, payload.Code)
			require.Equal(t, 1, payload.Data.TenantID)
			require.Equal(t, f.itemID, payload.Data.ItemID)
			require.Equal(t, "***", payload.Data.Config["token"])
			require.NotContains(t, rec.Body.String(), "historical-test-token")
			require.NotContains(t, rec.Body.String(), "tenant_id")

			rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+itemID+"/uninstall", "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			current, err := f.client.TenantInstallation.Get(ctx, installation.ID)
			require.NoError(t, err)
			require.Equal(t, tenantinstallation.StatusUninstalled, current.Status)
		})
	}
}

func TestMarketplaceRoutes_ConnectorInstallAndConfigure(t *testing.T) {
	f := newHandlerFixture(t, "route_connector")
	itemID := strconv.Itoa(f.itemID)
	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+itemID+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var payload struct {
		Code int                  `json:"code"`
		Data InstallationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Zero(t, payload.Code)
	require.Equal(t, "failed", payload.Data.Status)

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPut, "/api/v1/marketplace/installations/"+itemID+"/config", `{"credentials":{"token":"connector-test-token"}}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Zero(t, payload.Code)
	require.Equal(t, "active", payload.Data.Status)
	require.Equal(t, 1, payload.Data.TenantID)
	require.Equal(t, f.itemID, payload.Data.ItemID)
	require.NotContains(t, rec.Body.String(), "connector-test-token")
}
