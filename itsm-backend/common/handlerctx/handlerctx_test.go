package handlerctx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"itsm-backend/ent"
	"itsm-backend/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

// newCtx returns a gin.Context backed by a recorded ResponseWriter so
// we can inspect what the helper emitted to the client.
func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestResolveTenantID_Success(t *testing.T) {
	c, w := newCtx()
	// Simulate the tenant context being set by middleware.
	c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 42})

	id, ok := ResolveTenantID(c)
	if !ok {
		t.Fatalf("ResolveTenantID should succeed when context is set")
	}
	if id != 42 {
		t.Fatalf("expected tenant id 42, got %d", id)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected no body, got %s", w.Body.String())
	}
}

func TestResolveTenantID_MissingContext(t *testing.T) {
	c, w := newCtx()

	_, ok := ResolveTenantID(c)
	if ok {
		t.Fatalf("ResolveTenantID should fail when tenant context is absent")
	}
	// middleware.AbortIfTenantError maps ErrNoTenantContext to 401 + code 2001.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	// code 2001 is the existing middleware contract; do not change it.
	if int(body["code"].(float64)) != 2001 {
		t.Fatalf("expected code 2001 (middleware contract), got %v", body["code"])
	}
}

func TestGetEntClient_Success(t *testing.T) {
	c, _ := newCtx()
	want := &ent.Client{}
	c.Set("client", want)

	got, ok := GetEntClient(c)
	if !ok {
		t.Fatalf("GetEntClient should succeed when client is present")
	}
	if got != want {
		t.Fatalf("returned client pointer differs from what was set")
	}
}

func TestGetEntClient_Missing(t *testing.T) {
	c, w := newCtx()

	_, ok := GetEntClient(c)
	if ok {
		t.Fatalf("GetEntClient should fail when client is absent")
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	if int(body["code"].(float64)) != 5001 {
		t.Fatalf("expected code 5001, got %v", body["code"])
	}
}

func TestGetEntClient_WrongType(t *testing.T) {
	c, w := newCtx()
	c.Set("client", "not-an-ent-client")

	_, ok := GetEntClient(c)
	if ok {
		t.Fatalf("GetEntClient should fail when value is not *ent.Client")
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}