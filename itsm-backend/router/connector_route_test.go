package router

import (
	"net/http"
	"testing"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	connectorHandler "itsm-backend/handlers/connector"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 连接器密钥轮换必须真的挂在路由上。
//
// 背景：Handler.RotateSecret 实现完整（含 audit_log 写入），但没有任何路由注册，
// 属于"零引用的死代码"——契约里「租户密钥轮换」因此长期挂在待验收项。
// 这个用例锁定接线本身：删掉路由行会立刻转红，而不是等到某天有人试调 404。
func TestConnectorRotateSecretRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := connectorHandler.NewHandler(
		connector.NewManager(connector.Default(), zap.NewNop().Sugar()),
		connector.Default(),
		marketplace.New(),
		zap.NewNop().Sugar(),
	)

	engine := gin.New()
	SetupConnectorRoutes(engine.Group("/api/v1"), h)

	const wantPath = "/api/v1/connectors/configs/:name/rotate-secret"
	found := false
	for _, r := range engine.Routes() {
		if r.Method == http.MethodPost && r.Path == wantPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("POST %s is not registered; credential rotation is unreachable", wantPath)
	}
}

// 轮换与停用共用 connector:write，但路径必须各自独立（避免 /configs/:name 通配吞掉子路径）。
func TestConnectorRoutes_PermissionPrecheckResolves(t *testing.T) {
	path := "/api/v1/connectors/configs/*/rotate-secret"
	methodMap, ok := middleware.ResourceActionMap[http.MethodPost]
	if !ok {
		t.Fatalf("no POST entries in ResourceActionMap")
	}
	perm, ok := methodMap[path]
	if !ok {
		t.Fatalf("precheck map missing %s — 非 super_admin 会在此路径被恒 403", path)
	}
	if perm.Resource != "connector" || perm.Action != "write" {
		t.Fatalf("unexpected permission for %s: %+v", path, perm)
	}
}
