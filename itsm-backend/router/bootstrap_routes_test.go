package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	bootstrapauth "itsm-backend/pkg/bootstrap"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBootstrapRoutesExposeStatusAndConsumeTokenOnce(t *testing.T) {
	r, client := setupTestEngine(t)
	defer client.Close()
	ctx := context.Background()
	rootTenant, err := client.Tenant.Create().
		SetName("Default").SetCode("default").SetDomain("default.local").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	manager := bootstrapauth.NewBootstrapTokenManager(client, zaptest.NewLogger(t).Sugar())
	rawToken, err := manager.GenerateToken(ctx, rootTenant.ID)
	require.NoError(t, err)

	status := httptest.NewRecorder()
	r.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap/status", nil))
	require.Equal(t, http.StatusOK, status.Code)
	require.Contains(t, status.Body.String(), `"state":"ready"`)
	require.NotContains(t, status.Body.String(), rawToken)

	body := []byte(`{"token":"` + rawToken + `","password":"A-secure-admin-password-2026!"}`)
	created := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bootstrap/create-admin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(created, req)
	require.Equal(t, http.StatusOK, created.Code)
	require.Contains(t, created.Body.String(), `"state":"completed"`)

	replayed := httptest.NewRecorder()
	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/bootstrap/create-admin", bytes.NewReader(body))
	replayReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(replayed, replayReq)
	require.NotEqual(t, http.StatusOK, replayed.Code)
}
