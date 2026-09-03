package systemconfig

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

func TestConfigPaginationContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:config_page_contract?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	for i, key := range []string{"first", "second", "third", "foreign"} {
		tenant, category := 1, "mail"
		if i == 2 {
			category = "general"
		}
		if i == 3 {
			tenant = 2
		}
		client.SystemConfig.Create().SetKey(key).SetValue(key).SetValueType("string").SetCategory(category).SetTenantID(tenant).SaveX(context.Background())
	}
	h := NewHandler(service.NewSystemConfigService(client, zap.NewNop().Sugar()), zap.NewNop().Sugar())
	r := gin.New()
	r.GET("/api/v1/system-configs", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		h.ListConfigs(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/system-configs?category=mail&page=2&pageSize=1", nil))
	require.Equal(t, 200, w.Code, w.Body.String())
	var response struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.Len(t, response.Data, 5)
	for key, want := range map[string]string{"total": "2", "page": "2", "pageSize": "1", "totalPages": "2"} {
		require.JSONEq(t, want, string(response.Data[key]))
	}
	var items []map[string]interface{}
	require.NoError(t, json.Unmarshal(response.Data["items"], &items))
	require.Len(t, items, 1)
	require.Equal(t, float64(1), items[0]["tenantId"])
	require.Equal(t, "mail", items[0]["category"])
}
