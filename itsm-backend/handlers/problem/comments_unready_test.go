package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/middleware"
)

// 回归（2026-09-25 假成功收口）：问题评论未落地时必须显式 unready（5003），
// 不得用空成功伪装"暂无评论"；不存在的问题仍是 404，不因 unready 而丢失。
func TestProblemCommentsReturnExplicitUnready(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, svc, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "comments")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "comments")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("tenant_id", tenant.ID)
	})
	r.GET("/api/v1/problems/:id/comments", h.GetProblemComments)
	r.POST("/api/v1/problems/:id/comments", h.AddProblemComment)

	for _, tc := range []struct{ name, method string }{
		{"list", http.MethodGet},
		{"add", http.MethodPost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments", strings.NewReader(`{"content":"hi"}`)))
			require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
			var response struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, 5003, response.Code)
		})
	}

	t.Run("missing problem still 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/99999/comments", nil))
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})
}
