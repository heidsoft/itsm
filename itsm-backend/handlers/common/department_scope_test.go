package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	appcommon "itsm-backend/common"
)

// Exercise the production handlers and service/repository using the existing SQLite fixture.
func TestDepartmentScopeAndCycles(t *testing.T) {
	env := newTeamTestEnv(t)
	ctx := context.Background()
	repo := NewEntRepository(env.client)
	h := NewHandler(NewService(repo, "test-secret", zaptest.NewLogger(t).Sugar(), env.client))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("tenant_id", env.tenantA); c.Next() })
	router.POST("/api/v1/org/departments", h.CreateDepartment)
	router.PUT("/api/v1/org/departments/:id", h.UpdateDepartment)
	router.GET("/api/v1/org/departments/:id", h.GetDepartment)
	seed := func(tenant int, code string) int {
		d, err := env.client.Department.Create().SetName(code).SetCode(code).SetTenantID(tenant).Save(ctx)
		require.NoError(t, err)
		return d.ID
	}
	a, b, foreign := seed(env.tenantA, "A"), seed(env.tenantA, "B"), seed(env.tenantB, "FOREIGN")
	call := func(t *testing.T, method, path, body string, status int, code int) map[string]interface{} {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, status, w.Code, w.Body.String())
		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		require.EqualValues(t, code, result["code"], w.Body.String())
		require.NotContains(t, w.Body.String(), "password")
		return result
	}
	for _, tc := range []struct {
		name, method        string
		id, parent, manager int
	}{
		{"foreign parent create", http.MethodPost, 0, foreign, 0},
		{"foreign parent update", http.MethodPut, a, foreign, 0},
		{"foreign manager create", http.MethodPost, 0, 0, env.userB1},
		{"foreign manager update", http.MethodPut, a, 0, env.userB1},
		{"missing parent", http.MethodPost, 0, 999999, 0},
		{"missing manager", http.MethodPost, 0, 0, 999999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/org/departments"
			if tc.id != 0 {
				path += fmt.Sprintf("/%d", tc.id)
			}
			before, err := env.client.Department.Query().Count(ctx)
			require.NoError(t, err)
			result := call(t, tc.method, path, fmt.Sprintf(`{"name":"new","code":"new","parentId":%d,"managerId":%d}`, tc.parent, tc.manager), 400, appcommon.BadRequestCode)
			require.NotEmpty(t, result["message"])
			after, err := env.client.Department.Query().Count(ctx)
			require.NoError(t, err)
			require.Equal(t, before, after)
			unchanged, err := env.client.Department.Get(ctx, a)
			require.NoError(t, err)
			require.Zero(t, unchanged.ParentID)
			require.Zero(t, unchanged.ManagerID)
		})
	}
	deleted := seed(env.tenantA, "DELETED")
	require.NoError(t, env.client.Department.UpdateOneID(deleted).SetDeletedAt(time.Now()).Exec(ctx))
	call(t, "POST", "/api/v1/org/departments", fmt.Sprintf(`{"name":"deleted","code":"deleted-ref","parentId":%d}`, deleted), 400, appcommon.BadRequestCode)
	call(t, "PUT", fmt.Sprintf("/api/v1/org/departments/%d", a), fmt.Sprintf(`{"parentId":%d,"managerId":%d}`, b, env.userA1), 200, 0)
	for _, pair := range [][2]int{{b, a}, {a, a}} {
		call(t, "PUT", fmt.Sprintf("/api/v1/org/departments/%d", pair[0]), fmt.Sprintf(`{"parentId":%d}`, pair[1]), 400, appcommon.BadRequestCode)
	}
	created := call(t, "POST", "/api/v1/org/departments", fmt.Sprintf(`{"name":"valid","code":"valid","parentId":%d,"managerId":%d}`, b, env.userA1), 200, 0)["data"].(map[string]interface{})
	require.EqualValues(t, b, created["parentId"])
	require.EqualValues(t, env.userA1, created["managerId"])
	// Deliberately bypass the repository to reproduce historical corrupt cross-tenant links.
	require.NoError(t, env.client.Department.UpdateOneID(foreign).SetParentID(b).Exec(ctx))
	detail := call(t, "GET", fmt.Sprintf("/api/v1/org/departments/%d", b), "", 200, 0)["data"].(map[string]interface{})
	children := detail["children"].([]interface{})
	require.Len(t, children, 2)
	for _, child := range children {
		require.NotEqualValues(t, foreign, child.(map[string]interface{})["id"])
	}
	tree, err := repo.GetDepartmentTree(ctx, env.tenantA)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Equal(t, b, tree[0].ID)
	require.Len(t, tree[0].Children, 2)
}
