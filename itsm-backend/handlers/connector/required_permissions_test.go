package connector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// required_permissions 执行侧强制的回归锁。
//
// 背景：manifest.RequiredPermissions 曾在注册校验/checksum/DTO 投影三处投入，
// 执行侧却零消费——只持 connector:write 的角色可借飞书连接器绕过
// ticket:write/cmdb:write 对外发消息。这里用「计数器桩连接器」做负证明：
// gate 生效时 manager.Send 一次都不能被调用。
type permStub struct {
	manifest  connector.Manifest
	sendCalls *int
}

func (s *permStub) Manifest() connector.Manifest { return s.manifest }
func (s *permStub) Init(_ context.Context, _ connector.Config) error {
	return nil
}
func (s *permStub) Send(_ context.Context, _ *connector.Message) error {
	*s.sendCalls++
	return nil
}
func (s *permStub) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (s *permStub) Close() error { return nil }

// newPermFixture 装配一个声明给定权限的连接器实例，返回 handler 与 Send 调用计数器。
// client 传 nil（HardcodeOnly 不查库）；V6 单独用 DBOnly+nil 证明 fail-closed。
func newPermFixture(t *testing.T, perms []string) (*Handler, *int) {
	t.Helper()
	reg := connector.NewRegistry()
	calls := new(int)
	reg.Register(func() connector.Connector {
		return &permStub{
			manifest: connector.Manifest{
				Name:                "perm-stub",
				Version:             "1.0.0",
				Title:               "权限桩",
				Provider:            "stub",
				Type:                connector.TypeCustom,
				Capabilities:        []connector.Capability{connector.CapSendMessage},
				RequiredPermissions: perms,
			},
			sendCalls: calls,
		}
	})
	mgr := connector.NewManager(reg, zap.NewNop().Sugar())
	if err := mgr.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "perm-stub", Provider: "stub", Enabled: true,
		Settings: map[string]interface{}{"debug_channel": "test-channel"},
	}); err != nil {
		t.Fatalf("provision perm-stub: %v", err)
	}
	return NewHandler(mgr, reg, marketplace.New(), zap.NewNop().Sugar(), nil), calls
}

// useHardcodePermissions 切到 HardcodeOnly 并在用例结束后恢复，避免污染其他测试。
func useHardcodePermissions(t *testing.T) {
	t.Helper()
	orig := middleware.PermissionConfig.Mode
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	t.Cleanup(func() { middleware.PermissionConfig.Mode = orig })
}

func permRequest(h *Handler, role string, tenantID int, body string, fn func(*gin.Context)) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/perm-stub/send", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "perm-stub"}}
	c.Set("role", role)
	c.Set("tenant_id", tenantID)
	fn(c)
	return w
}

const sendBody = `{"name":"perm-stub","channel":"test-channel","type":"text","content":"hi"}`

// V1：manifest 声明 email_intake:review（内置 admin 角色没有该码）→ Send 403，manager 不被调用。
func TestSend_DeniesManifestPermissionMissing(t *testing.T) {
	useHardcodePermissions(t)
	h, calls := newPermFixture(t, []string{"connector:write", "email_intake:review"})
	w := permRequest(h, domainrole.Admin, 1, sendBody, h.Send)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 when manifest permission missing, got %d: %s", w.Code, w.Body.String())
	}
	if *calls != 0 {
		t.Fatalf("manager.Send must not be called on denial, got %d", *calls)
	}
}

// V2：角色持有 manifest 全部声明 → 放行并调用 manager。
func TestSend_AllowsWhenAllManifestPermissionsHeld(t *testing.T) {
	useHardcodePermissions(t)
	// admin 内置角色持 connector:write / ticket:write / cmdb:write（roles.go:258-281）
	h, calls := newPermFixture(t, []string{"connector:write", "ticket:write", "cmdb:write"})
	w := permRequest(h, domainrole.Admin, 1, sendBody, h.Send)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 when all manifest permissions held, got %d: %s", w.Code, w.Body.String())
	}
	if *calls != 1 {
		t.Fatalf("manager.Send should be called once, got %d", *calls)
	}
}

// V3：权限码格式非法（点号）→ fail-closed 403，manager 不被调用。
func TestSend_MalformedPermissionFailsClosed(t *testing.T) {
	useHardcodePermissions(t)
	h, calls := newPermFixture(t, []string{"connector:write", "ticket.write"})
	w := permRequest(h, domainrole.Admin, 1, sendBody, h.Send)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 for malformed permission code, got %d: %s", w.Code, w.Body.String())
	}
	if *calls != 0 {
		t.Fatalf("manager.Send must not be called on malformed declaration, got %d", *calls)
	}
}

// V4：未知 manifest 放行，交由 manager 报未知连接器（不是 403）。
func TestSend_UnknownManifestPassesThrough(t *testing.T) {
	useHardcodePermissions(t)
	h, _ := newPermFixture(t, []string{"connector:write"})
	w := permRequest(h, domainrole.Admin, 1, sendBody, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "name", Value: "not-registered"}}
		h.Send(c)
	})
	if w.Code == http.StatusForbidden {
		t.Fatalf("unknown manifest is not a permission failure; got 403: %s", w.Body.String())
	}
}

// V5：Test 入口同样受 gate（缺 email_intake:review → 403，manager 不被调用）。
func TestTest_DeniesManifestPermissionMissing(t *testing.T) {
	useHardcodePermissions(t)
	h, calls := newPermFixture(t, []string{"connector:write", "email_intake:review"})
	w := permRequest(h, domainrole.Admin, 1, ``, h.Test)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Test: want 403 when manifest permission missing, got %d: %s", w.Code, w.Body.String())
	}
	if *calls != 0 {
		t.Fatalf("Test: manager.Send must not be called on denial, got %d", *calls)
	}
}

// V6：DBOnly + client 丢失 → fail-closed（构造函数必须保留 client）。
func TestEnforceRequiredPermissions_DBOnlyNilClientFailsClosed(t *testing.T) {
	orig := middleware.PermissionConfig.Mode
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeDBOnly
	t.Cleanup(func() { middleware.PermissionConfig.Mode = orig })

	h, _ := newPermFixture(t, []string{"connector:write", "ticket:write"})
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/perm-stub/send", nil)
	c.Set("role", domainrole.Admin)
	c.Set("tenant_id", 1)
	if h.enforceRequiredPermissions(c, "perm-stub") {
		t.Fatal("DBOnly with nil client must fail closed for a non-super_admin role")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 response written, got %d", w.Code)
	}
}

// 接线锁：bootstrap 注入的 client 不被 handler 丢弃时，DBOnly 下持 DB 权限的角色可放行。
// 用 authz 单一真源的默认权限集（HardcodeOnly）作为「判定链可达」的正向观测；
// client 若在构造时被丢弃，V6 会立即转红（nil + DBOnly 恒拒绝）。
func TestEnforceRequiredPermissions_EmptyAndMissingManifestPass(t *testing.T) {
	useHardcodePermissions(t)
	h, _ := newPermFixture(t, []string{"connector:write"})
	gin.SetMode(gin.TestMode)

	// 未知 manifest → 放行（不是权限失败）
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/unknown/send", nil)
	c.Set("role", domainrole.Admin)
	c.Set("tenant_id", 1)
	if !h.enforceRequiredPermissions(c, "not-registered") {
		t.Fatalf("unknown manifest should pass through, got %d: %s", w.Code, w.Body.String())
	}
}

// 确保拒绝响应体确实点名缺失的权限码（审计/前端可读）。
func TestForbiddenMessageMentionsMissingPermission(t *testing.T) {
	useHardcodePermissions(t)
	h, _ := newPermFixture(t, []string{"connector:write", "email_intake:review"})
	w := permRequest(h, domainrole.Admin, 1, sendBody, h.Send)
	if !strings.Contains(w.Body.String(), "email_intake:review") {
		t.Fatalf("denial should name the missing permission, got: %s", w.Body.String())
	}
}
