package connector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"itsm-backend/connector"
	"itsm-backend/connector/marketplace"
	domainCommon "itsm-backend/handlers/common"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
)

// 连接器密钥轮换（POST /connectors/configs/:name/rotate-secret）的回归。
//
// 背景：Handler.RotateSecret 早已实现（含 audit_log 写入），但**从未挂到路由**，
// 契约里「租户密钥轮换」因此一直挂在"待验收"。接线时同时修掉三处实现问题：
//   1. 写了一个 grace_period_at 字段却没有任何代码读它 —— 假的回滚窗口；
//   2. 直接修改 manager 内实例共享的 Settings map（未落库的时间戳进了运行态）；
//   3. 落库失败时 Revoke 掉实例，连接器直接变不可用，而不是恢复旧配置。

// rotateStub 是可控的连接器桩：Init 校验 token，可被要求拒绝某个值。
type rotateStub struct {
	mu       *sync.Mutex
	lastInit *map[string]string // 最近一次 Init 收到的凭据
	reject   string             // 该 token 值触发 Init 失败
	leak     string             // 失败时把该值写进错误文案（验证脱敏）
}

func (s *rotateStub) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                "rotate-stub",
		Version:             "1.0.0",
		Title:               "轮换桩",
		Provider:            "stub",
		Type:                connector.TypeCustom,
		Capabilities:        []connector.Capability{connector.CapHealthCheck},
		RequiredPermissions: []string{"connector.write"},
	}
}

func (s *rotateStub) Init(_ context.Context, cfg connector.Config) error {
	token := cfg.Credentials["token"]
	if token == "" {
		return errors.New("credentials.token 未配置")
	}
	if s.reject != "" && token == s.reject {
		return errors.New("第三方拒绝该 token: " + s.leak)
	}
	// 只在 Init 成功后记录——它代表"运行态当前持有的凭据"，
	// 失败的那次尝试不能写进来，否则测试会把"尝试过"误当成"已生效"。
	s.mu.Lock()
	cp := make(map[string]string, len(cfg.Credentials))
	for k, v := range cfg.Credentials {
		cp[k] = v
	}
	*(s.lastInit) = cp
	s.mu.Unlock()
	return nil
}

func (s *rotateStub) Send(_ context.Context, _ *connector.Message) error { return nil }
func (s *rotateStub) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (s *rotateStub) Close() error { return nil }

type capturingAudit struct {
	logs []*domainCommon.AuditLog
	err  error
}

func (c *capturingAudit) CreateAuditLog(_ context.Context, l *domainCommon.AuditLog) error {
	c.logs = append(c.logs, l)
	return c.err
}

type rotateFixture struct {
	handler  *Handler
	manager  *connector.Manager
	client   interface{ Close() error }
	audit    *capturingAudit
	lastInit *map[string]string
}

func newRotateFixture(t *testing.T) *rotateFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:rotate-secret?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	store, err := connector.NewPersistentConfigStore(client, "test-connector-encryption-key")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	mu := &sync.Mutex{}
	lastInit := map[string]string{}
	reg := connector.NewRegistry()
	reg.Register(func() connector.Connector {
		return &rotateStub{mu: mu, lastInit: &lastInit}
	})

	mgr := connector.NewManager(reg, zap.NewNop().Sugar())
	initial := connector.Config{
		TenantID:    1,
		Name:        "rotate-stub",
		Provider:    "stub",
		Type:        connector.TypeCustom,
		Enabled:     true,
		Credentials: map[string]string{"token": "old-token-1111"},
		Settings:    map[string]interface{}{"region": "cn"},
	}
	if err := mgr.Provision(context.Background(), initial); err != nil {
		t.Fatalf("provision initial: %v", err)
	}
	if err := store.Save(context.Background(), initial); err != nil {
		t.Fatalf("save initial: %v", err)
	}

	h := NewHandler(mgr, reg, marketplace.New(), zap.NewNop().Sugar())
	h.SetPersistentStore(store)
	audit := &capturingAudit{}
	h.SetAuditRecorder(audit)
	return &rotateFixture{handler: h, manager: mgr, client: client, audit: audit, lastInit: &lastInit}
}

func (f *rotateFixture) dbCredentials(t *testing.T) map[string]string {
	t.Helper()
	cfgs, err := f.handler.store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("load all: %v", err)
	}
	for _, c := range cfgs {
		if c.Name == "rotate-stub" {
			return c.Credentials
		}
	}
	t.Fatalf("connector config not found in store")
	return nil
}

func (f *rotateFixture) rotate(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/configs/rotate-stub/rotate-secret", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "rotate-stub"}}
	c.Set("tenant_id", 1)
	c.Set("user_id", 42)
	f.handler.RotateSecret(c)
	return w
}

func TestRotateSecret_UpdatesRuntimeStoreAndAudit(t *testing.T) {
	f := newRotateFixture(t)

	w := f.rotate(t, `{"newCredentials":{"token":"new-token-2222"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "new-token-2222") || strings.Contains(body, "old-token-1111") {
		t.Fatalf("response must not echo credentials: %s", body)
	}

	// 运行态：manager 里的实例已换成新凭据
	if got := (*f.lastInit)["token"]; got != "new-token-2222" {
		t.Fatalf("runtime credentials not rotated, got %q", got)
	}
	// 持久层：库里是新凭据（且加密存储由 store 负责）
	if got := f.dbCredentials(t)["token"]; got != "new-token-2222" {
		t.Fatalf("persisted credentials not rotated, got %q", got)
	}

	// 审计：一条 rotate_secret，只含键名不含值
	if len(f.audit.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(f.audit.logs))
	}
	log := f.audit.logs[0]
	if log.Action != "rotate_secret" || log.Resource != "connector" {
		t.Fatalf("unexpected audit action/resource: %s/%s", log.Resource, log.Action)
	}
	if log.UserID != 42 || log.TenantID != 1 {
		t.Fatalf("unexpected audit actor/tenant: %d/%d", log.UserID, log.TenantID)
	}
	if !strings.Contains(log.RequestBody, "token") {
		t.Fatalf("audit should record rotated key names, got %s", log.RequestBody)
	}
	if strings.Contains(log.RequestBody, "new-token-2222") || strings.Contains(log.RequestBody, "old-token-1111") {
		t.Fatalf("audit body leaked credential values: %s", log.RequestBody)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(log.RequestBody), &payload); err != nil {
		t.Fatalf("audit body is not json: %v", err)
	}
	if _, ok := payload["gracePeriodHours"]; ok {
		t.Fatalf("gracePeriodHours must be gone (it was never enforced): %s", log.RequestBody)
	}
}

func TestRotateSecret_RejectsEmptyOrMissingCredentials(t *testing.T) {
	f := newRotateFixture(t)

	for _, body := range []string{`{"newCredentials":{}}`, `{}`} {
		w := f.rotate(t, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %s: expected 400, got %d (%s)", body, w.Code, w.Body.String())
		}
	}
	if (*f.lastInit)["token"] != "old-token-1111" {
		t.Fatalf("runtime must stay on old credentials, got %q", (*f.lastInit)["token"])
	}
	if f.dbCredentials(t)["token"] != "old-token-1111" {
		t.Fatalf("store must stay on old credentials")
	}
}

func TestRotateSecret_UnknownConnectorReturns404(t *testing.T) {
	f := newRotateFixture(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/connectors/configs/not-exist/rotate-secret", strings.NewReader(`{"newCredentials":{"token":"x"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "not-exist"}}
	c.Set("tenant_id", 1)
	c.Set("user_id", 42)

	f.handler.RotateSecret(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (%s)", w.Code, w.Body.String())
	}
}

// Init 失败必须"保留原凭据"，不能把连接器改成半死状态。
func TestRotateSecret_InitFailureKeepsOldCredentials(t *testing.T) {
	f := newRotateFixture(t)
	// 让桩拒绝 bad-token，并在错误文案里回显它 —— 验证响应脱敏
	stub := &rotateStub{mu: &sync.Mutex{}, lastInit: f.lastInit, reject: "bad-token-9999", leak: "bad-token-9999"}
	reg := connector.NewRegistry()
	reg.Register(func() connector.Connector { return stub })
	mgr := connector.NewManager(reg, zap.NewNop().Sugar())
	if err := mgr.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "rotate-stub", Provider: "stub", Enabled: true,
		Credentials: map[string]string{"token": "old-token-1111"},
		Settings:    map[string]interface{}{"region": "cn"},
	}); err != nil {
		t.Fatalf("provision: %v", err)
	}
	f.handler.manager = mgr

	w := f.rotate(t, `{"newCredentials":{"token":"bad-token-9999"}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "保留原凭据") {
		t.Fatalf("expected actionable message, got %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "bad-token-9999") {
		t.Fatalf("error message must not echo the submitted credential: %s", w.Body.String())
	}
	// 运行态与库都在旧凭据上
	if got := (*f.lastInit)["token"]; got != "old-token-1111" {
		t.Fatalf("runtime should keep old credentials, got %q", got)
	}
	if got := f.dbCredentials(t)["token"]; got != "old-token-1111" {
		t.Fatalf("store should keep old credentials, got %q", got)
	}
	if len(f.audit.logs) != 0 {
		t.Fatalf("failed rotation must not write a success audit, got %d", len(f.audit.logs))
	}
}

// 落库失败时把运行态恢复成旧配置，而不是 Revoke 掉实例。
func TestRotateSecret_PersistFailureRestoresRuntime(t *testing.T) {
	f := newRotateFixture(t)
	// 关掉数据库连接 → store.Save 必然失败
	_ = f.client.Close()

	w := f.rotate(t, `{"newCredentials":{"token":"new-token-3333"}}`)
	if w.Code < 500 {
		t.Fatalf("expected 5xx, got %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "已恢复原配置") {
		t.Fatalf("expected restore hint, got %s", w.Body.String())
	}
	if got := (*f.lastInit)["token"]; got != "old-token-1111" {
		t.Fatalf("runtime must be restored to old credentials, got %q", got)
	}
	// 实例仍然存在（旧实现会 Revoke，直接不可用）
	if got := len(f.manager.ListByTenant(1)); got != 1 {
		t.Fatalf("connector instance must survive a failed rotation, got %d", got)
	}
}

// 未注入审计写入器时轮换仍应成功（不 panic、不阻断）。
func TestRotateSecret_WorksWithoutAuditRecorder(t *testing.T) {
	f := newRotateFixture(t)
	f.handler.SetAuditRecorder(nil)

	w := f.rotate(t, `{"newCredentials":{"token":"new-token-4444"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if got := f.dbCredentials(t)["token"]; got != "new-token-4444" {
		t.Fatalf("expected rotated credentials, got %q", got)
	}
}

// 审计写失败不回滚轮换（密钥已在对方平台换掉），但不能静默。
func TestRotateSecret_AuditFailureDoesNotRollbackRotation(t *testing.T) {
	f := newRotateFixture(t)
	f.audit.err = errors.New("audit store down")

	w := f.rotate(t, `{"newCredentials":{"token":"new-token-5555"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even if audit fails, got %d (%s)", w.Code, w.Body.String())
	}
	if got := f.dbCredentials(t)["token"]; got != "new-token-5555" {
		t.Fatalf("rotation must be applied, got %q", got)
	}
}
