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

	"itsm-backend/connector"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/middleware"
	marketplaceService "itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
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

	return &handlerFixture{handler: NewHandler(svc), itemID: item.ID}
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
