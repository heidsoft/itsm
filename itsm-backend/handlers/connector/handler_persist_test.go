package connector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 回归用例：持久化 store 未注入时，Provision/Revoke 必须显性失败并拒绝变更内存状态。
//
// 背景：store 曾被误注入到无路由挂载的 controller.ConnectorController，实际承接路由的
// handler 的 h.store 始终为 nil；旧实现用 `if h.store != nil` 静默跳过落库，
// 导致配置只写进内存 manager，重启即丢失且无任何报错。这里锁住 fail-loudly 行为。
func newStorelessHandler() *Handler {
	return NewHandler(
		connector.NewManager(connector.Default(), zap.NewNop().Sugar()),
		connector.Default(),
		marketplace.New(),
		zap.NewNop().Sugar(),
	)
}

func TestProvision_RejectsWhenPersistentStoreMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newStorelessHandler()
	if h.store != nil {
		t.Fatalf("expected nil store before SetPersistentStore, got %v", h.store)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/connectors/configs",
		strings.NewReader(`{"name":"feishu","provider":"feishu","enabled":true,"credentials":{"appId":"x"},"settings":{}}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", 1)

	h.Provision(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 when persistence is unavailable, got %d (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "持久化未就绪") {
		t.Fatalf("expected explicit persistence error, body=%s", w.Body.String())
	}
	// 关键：不得只写内存而绕过落库
	if got := len(h.manager.ListByTenant(1)); got != 0 {
		t.Fatalf("expected no in-memory config when persistence is unavailable, got %d", got)
	}
}

func TestRevoke_RejectsWhenPersistentStoreMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newStorelessHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/connectors/configs/feishu", nil)
	c.Set("tenant_id", 1)
	c.Params = gin.Params{{Key: "name", Value: "feishu"}}

	h.Revoke(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 when persistence is unavailable, got %d (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "持久化未就绪") {
		t.Fatalf("expected explicit persistence error, body=%s", w.Body.String())
	}
}
