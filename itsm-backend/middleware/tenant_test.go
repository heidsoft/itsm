package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

func TestTenantMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a test client
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()

	// Seed test data
	ctx := context.Background()
	activeTenant, err := client.Tenant.Create().
		SetCode("active-tenant").
		SetName("Active Tenant").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	_ = activeTenant

	inactiveTenant, err := client.Tenant.Create().
		SetCode("inactive-tenant").
		SetName("Inactive Tenant").
		SetStatus("inactive").
		Save(ctx)
	assert.NoError(t, err)

	_ = inactiveTenant

	expiredTenant, err := client.Tenant.Create().
		SetCode("expired-tenant").
		SetName("Expired Tenant").
		SetStatus("active").
		SetExpiresAt(time.Now().Add(-24 * time.Hour)).
		Save(ctx)
	assert.NoError(t, err)

	_ = expiredTenant

	t.Run("Missing Tenant Information Returns Bad Request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		TenantMiddleware(client)(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})

	t.Run("Non-existent Tenant Returns Not Found", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Request.Header.Set("X-Tenant-Code", "non-existent")

		TenantMiddleware(client)(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "租户不存在")
	})

	t.Run("Inactive Tenant Returns Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Request.Header.Set("X-Tenant-Code", "inactive-tenant")

		TenantMiddleware(client)(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "租户已被暂停或过期")
	})

	t.Run("Expired Tenant Returns Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Request.Header.Set("X-Tenant-Code", "expired-tenant")

		TenantMiddleware(client)(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "租户已过期")
	})

	t.Run("Valid Tenant Succeeds", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Request.Header.Set("X-Tenant-Code", "active-tenant")

		TenantMiddleware(client)(c)

		// Should not abort
		assert.False(t, c.IsAborted())

		// Check tenant context is set
		tenantCtx, exists := GetTenantContext(c)
		assert.True(t, exists)
		assert.Equal(t, activeTenant.ID, tenantCtx.TenantID)
		assert.Equal(t, "active-tenant", tenantCtx.Tenant.Code)
	})

	t.Run("Valid Tenant via X-Tenant-ID Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Request.Header.Set("X-Tenant-ID", "1")

		TenantMiddleware(client)(c)

		// Should not abort
		assert.False(t, c.IsAborted())
	})
}

func TestExtractTenantFromHost(t *testing.T) {
	t.Run("Subdomain Extraction", func(t *testing.T) {
		assert.Equal(t, "tenant1", extractTenantFromHost("tenant1.itsm.example.com"))
		assert.Equal(t, "mycompany", extractTenantFromHost("mycompany.itsm.com"))
	})

	t.Run("No Subdomain Returns Empty", func(t *testing.T) {
		// "example.com" has 2 parts, returns empty
		assert.Equal(t, "", extractTenantFromHost("example.com"))
		assert.Equal(t, "", extractTenantFromHost("localhost"))
	})

	t.Run("Single Part Host Returns Empty", func(t *testing.T) {
		assert.Equal(t, "", extractTenantFromHost("localhost"))
	})
}

func TestGetTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Context Not Set Returns False", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		tenantCtx, exists := GetTenantContext(c)
		assert.False(t, exists)
		assert.Nil(t, tenantCtx)
	})

	t.Run("Context Set Returns True", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		testTenant := &ent.Tenant{
			ID:   1,
			Code: "test-tenant",
		}
		tenantCtx := &TenantContext{
			TenantID: 1,
			Tenant:   testTenant,
		}
		c.Set(TenantContextKey, tenantCtx)

		result, exists := GetTenantContext(c)
		assert.True(t, exists)
		assert.Equal(t, 1, result.TenantID)
		assert.Equal(t, "test-tenant", result.Tenant.Code)
	})
}

func TestGetTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No Context Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		id, err := GetTenantID(c)
		assert.Error(t, err)
		assert.Equal(t, 0, id)
	})

	t.Run("With Context Returns ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		testTenant := &ent.Tenant{
			ID:   42,
			Code: "test-tenant",
		}
		tenantCtx := &TenantContext{
			TenantID: 42,
			Tenant:   testTenant,
		}
		c.Set(TenantContextKey, tenantCtx)

		id, err := GetTenantID(c)
		assert.NoError(t, err)
		assert.Equal(t, 42, id)
	})
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No User ID Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		id, err := GetUserID(c)
		assert.Error(t, err)
		assert.Equal(t, 0, id)
	})

	t.Run("Valid User ID Returns ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Set("user_id", 123)

		id, err := GetUserID(c)
		assert.NoError(t, err)
		assert.Equal(t, 123, id)
	})

	t.Run("Invalid User ID Type Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Set("user_id", "invalid")

		id, err := GetUserID(c)
		assert.Error(t, err)
		assert.Equal(t, 0, id)
	})
}
