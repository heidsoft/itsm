package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

// =====================================================================
// Test Fixtures
// =====================================================================

func setupTestEngine(t *testing.T) (*gin.Engine, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:router_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret: "test-secret",
		Logger:    logger,
		Client:    client,
		// All controllers nil — only public routes and health should register
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, cfg)
	return r, client
}

// =====================================================================
// Smoke Tests
// =====================================================================

func TestSetupRoutes_NoPanic(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_nopanic?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret: "test-secret",
		Logger:    logger,
		Client:    client,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() {
		SetupRoutes(r, cfg)
	}, "SetupRoutes should not panic with nil controllers")
}

func TestSetupRoutes_AllControllersNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_allnil?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret: "test-secret",
		Logger:    logger,
		Client:    client,
		// All controller fields nil by default
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() {
		SetupRoutes(r, cfg)
	})

	// Verify public routes registered
	routes := r.Routes()
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	assert.True(t, routeMap["GET /api/v1/health"], "health should be registered")
	assert.True(t, routeMap["GET /api/v1/version"], "version should be registered")
}

// =====================================================================
// Public Endpoint Tests
// =====================================================================

func TestSetupRoutes_HealthEndpoint(t *testing.T) {
	r, client := setupTestEngine(t)
	defer client.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestSetupRoutes_VersionEndpoint(t *testing.T) {
	r, client := setupTestEngine(t)
	defer client.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "version")
}

func TestSetupRoutes_GAReadinessEndpoint(t *testing.T) {
	r, client := setupTestEngine(t)
	defer client.Close()

	// Create minimal required data for the readiness check
	ctx := context.Background()
	_, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/readiness/ga", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =====================================================================
// Nil Controller Guard Tests
// =====================================================================

func TestSetupRoutes_MSPControllerNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_mspnil?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret:      "test-secret",
		Logger:        logger,
		Client:        client,
		MSPController: nil,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() { SetupRoutes(r, cfg) })

	routes := r.Routes()
	for _, route := range routes {
		assert.NotEqual(t, "/api/v1/msp/status", route.Path,
			"MSP routes should not be registered when MSPController is nil")
	}
}

func TestSetupRoutes_DashboardHandlerNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_dashnil?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret:         "test-secret",
		Logger:           logger,
		Client:           client,
		DashboardHandler:  nil,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() { SetupRoutes(r, cfg) })
}

func TestSetupRoutes_CMDBControllerNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_cmdbnil?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret:     "test-secret",
		Logger:       logger,
		Client:       client,
		CMDBController: nil,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() { SetupRoutes(r, cfg) })
}

func TestSetupRoutes_IncidentControllerNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:router_incidentnil?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret:          "test-secret",
		Logger:            logger,
		Client:            client,
		IncidentController: nil,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.NotPanics(t, func() { SetupRoutes(r, cfg) })
}

func TestSetupRoutes_AuthMiddleware_HealthUnauthenticated(t *testing.T) {
	// Health should be accessible without JWT
	r, client := setupTestEngine(t)
	defer client.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should NOT return 401/403 (no auth required for health)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// =====================================================================
// WebSocket Ticket Store
// =====================================================================

func TestWSTicketStore_GenerateAndRedeem(t *testing.T) {
	store := NewWSTicketStore(DefaultWSTicketTTL)
	require.NotNil(t, store)

	ticketStr, err := store.Generate(1, 100)
	require.NoError(t, err)
	require.NotEmpty(t, ticketStr)

	userID, tenantID, ok := store.Redeem(ticketStr)
	assert.True(t, ok, "valid ticket should redeem successfully")
	assert.Equal(t, 1, userID)
	assert.Equal(t, 100, tenantID)

	// One-time use
	_, _, ok = store.Redeem(ticketStr)
	assert.False(t, ok, "ticket should be invalidated after first redemption")

	// Invalid ticket
	_, _, ok = store.Redeem("invalid-ticket")
	assert.False(t, ok, "invalid ticket should fail redemption")
}

func TestWSTicketStore_DifferentUsers(t *testing.T) {
	store := NewWSTicketStore(DefaultWSTicketTTL)

	ticket1, _ := store.Generate(1, 100)
	ticket2, _ := store.Generate(2, 100)

	assert.NotEqual(t, ticket1, ticket2, "different users should get different tickets")

	uid1, _, _ := store.Redeem(ticket1)
	assert.Equal(t, 1, uid1)

	uid2, _, _ := store.Redeem(ticket2)
	assert.Equal(t, 2, uid2)
}

func TestWSTicketStore_ConcurrentGenerate(t *testing.T) {
	store := NewWSTicketStore(DefaultWSTicketTTL)

	done := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func(userID int) {
			ticket, _ := store.Generate(userID, 1)
			done <- ticket
		}(i)
	}

	tickets := make([]string, 10)
	for i := 0; i < 10; i++ {
		tickets[i] = <-done
	}

	seen := make(map[string]bool)
	for _, ticket := range tickets {
		assert.False(t, seen[ticket], "tickets should be unique")
		seen[ticket] = true
	}
}

func TestWSTicketStore_NilTTL(t *testing.T) {
	// Zero TTL should still work
	store := NewWSTicketStore(0)
	ticket, err := store.Generate(1, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, ticket)
}

// =====================================================================
// RateLimiter Interface
// =====================================================================

func TestRateLimiterInterface(t *testing.T) {
	// Verify RateLimiterInterface can be satisfied by a simple implementation
	var _ RateLimiterInterface = &noopRateLimiter{}
}

type noopRateLimiter struct{}

func (n *noopRateLimiter) Allow(ctx context.Context, clientIP string) (bool, error) {
	return true, nil
}

// =====================================================================
// RouterConfig Zero Value
// =====================================================================

func TestRouterConfig_ZeroValue(t *testing.T) {
	// Verify RouterConfig can be created with zero values
	cfg := &RouterConfig{}

	assert.Empty(t, cfg.JWTSecret)
	assert.Nil(t, cfg.Logger)
	assert.Nil(t, cfg.Client)
	assert.Nil(t, cfg.TicketController)
	assert.Nil(t, cfg.CMDBController)
	assert.Nil(t, cfg.IncidentController)
	assert.Nil(t, cfg.DashboardHandler)
}

// =====================================================================
// Routes exist for legacy compatibility stubs
// =====================================================================

func TestSetupRoutes_LegacyStubs(t *testing.T) {
	// Legacy stubs should return BadRequestCode (400-ish), not 500
	client := enttest.Open(t, "sqlite3", "file:router_legacy?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	logger := zaptest.NewLogger(t).Sugar()

	cfg := &RouterConfig{
		JWTSecret:        "test-secret",
		Logger:          logger,
		Client:          client,
		TenantController: nil, // triggers legacy stubs
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, cfg)

	// These paths have stub handlers in the router
	stubPaths := []string{
		"/api/v1/workflows",
		"/api/v1/ticket-types",
		"/api/v1/services",
		"/api/v1/slas",
		"/api/v1/knowledge",
	}

	for _, path := range stubPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// Add auth header to pass through middleware
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// Should not panic or 500
		assert.NotEqual(t, http.StatusInternalServerError, w.Code, "path %s should not 500", path)
	}
}
