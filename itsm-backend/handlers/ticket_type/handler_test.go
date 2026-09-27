package ticket_type

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupTicketTypeRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbName := fmt.Sprintf("file:tt_%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewTicketTypeService(client, logger)
	h := NewHandler(svc, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if v := c.GetHeader("X-Test-Tenant"); v != "" {
			if id, err := strconv.Atoi(v); err == nil {
				tenantID = id
			}
		}
		userID := 1
		if v := c.GetHeader("X-Test-User"); v != "" {
			if id, err := strconv.Atoi(v); err == nil {
				userID = id
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("user_id", userID)
		c.Next()
	})

	r.GET("/api/v1/ticket-types", h.ListTicketTypes)
	r.POST("/api/v1/ticket-types", h.CreateTicketType)
	r.GET("/api/v1/ticket-types/:id", h.GetTicketType)
	r.PUT("/api/v1/ticket-types/:id", h.UpdateTicketType)
	r.DELETE("/api/v1/ticket-types/:id", h.DeleteTicketType)
	r.POST("/api/v1/ticket-types/:id/enable", h.EnableTicketType)
	r.POST("/api/v1/ticket-types/:id/disable", h.DisableTicketType)
	r.POST("/api/v1/ticket-types/:id/clone", h.CloneTicketType)
	r.POST("/api/v1/ticket-types/:id/restore", h.RestoreTicketType)
	r.GET("/api/v1/ticket-type-presets", h.ListPresets)
	r.POST("/api/v1/ticket-type-presets/:presetId/install", h.InstallPreset)

	return r
}

func doRequest(r *gin.Engine, method, path string, body interface{}, tenantID, userID int) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if tenantID > 0 {
		req.Header.Set("X-Test-Tenant", strconv.Itoa(tenantID))
	}
	if userID > 0 {
		req.Header.Set("X-Test-User", strconv.Itoa(userID))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "body=%s", w.Body.String())
	return resp
}

func createType(t *testing.T, r *gin.Engine, code, name string, tenantID int) map[string]interface{} {
	t.Helper()
	w := doRequest(r, http.MethodPost, "/api/v1/ticket-types", map[string]interface{}{
		"code": code, "name": name,
	}, tenantID, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	return data
}

func TestHandler_CreateAndGet(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "test_incident", "测试事件", 1)
	assert.Equal(t, "test_incident", data["code"])
	assert.Equal(t, "测试事件", data["name"])
	assert.Equal(t, "active", data["status"])

	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodGet, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &got))
	assert.Equal(t, "test_incident", got["code"])
}

func TestHandler_CreateDuplicateCodeReturnsConflict(t *testing.T) {
	r := setupTicketTypeRouter(t)

	createType(t, r, "dup_code", "第一次", 1)

	w := doRequest(r, http.MethodPost, "/api/v1/ticket-types", map[string]interface{}{
		"code": "dup_code", "name": "第二次",
	}, 1, 1)
	require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
	resp := parseResponse(t, w)
	assert.Equal(t, 4090, resp.Code)
}

func TestHandler_CreateInvalidCodeReturnsBadRequest(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/ticket-types", map[string]interface{}{
		"code": "Bad-Code", "name": "坏编码",
	}, 1, 1)
	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_CreateCustomFieldValidation(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/ticket-types", map[string]interface{}{
		"code": "bad_field", "name": "坏字段",
		"customFields": []map[string]interface{}{
			{"name": "bad-name", "label": "Bad", "type": "text", "order": 0},
		},
	}, 1, 1)
	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_ListAndFilter(t *testing.T) {
	r := setupTicketTypeRouter(t)

	createType(t, r, "list_a", "类型A", 1)
	createType(t, r, "list_b", "类型B", 1)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-types?page=1&pageSize=10", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	var list dto.TicketTypeListResponse
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	assert.Equal(t, int64(2), list.Total)
	assert.Len(t, list.Items, 2)
}

func TestHandler_ListKeywordFilter(t *testing.T) {
	r := setupTicketTypeRouter(t)

	createType(t, r, "kw_alpha", "Alpha类型", 1)
	createType(t, r, "kw_beta", "Beta类型", 1)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-types?keyword=alpha", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var list dto.TicketTypeListResponse
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	assert.Equal(t, int64(1), list.Total)
	assert.Equal(t, "kw_alpha", list.Items[0].Code)
}

func TestHandler_UpdateName(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "upd_type", "原名", 1)
	id := int(data["id"].(float64))

	newName := "新名"
	w := doRequest(r, http.MethodPut, fmt.Sprintf("/api/v1/ticket-types/%d", id), map[string]interface{}{
		"name": &newName,
	}, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	var updated map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &updated))
	assert.Equal(t, "新名", updated["name"])
	assert.Equal(t, "upd_type", updated["code"], "code is immutable")
}

func TestHandler_ArchiveAndRestore(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "archive_me", "待归档", 1)
	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodDelete, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	w = doRequest(r, http.MethodGet, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &got))
	assert.Equal(t, "inactive", got["status"])

	w = doRequest(r, http.MethodPost, fmt.Sprintf("/api/v1/ticket-types/%d/restore", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp = parseResponse(t, w)
	var restored map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &restored))
	assert.Equal(t, "active", restored["status"])
}

func TestHandler_ArchiveExcludesFromDefaultList(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "hidden_type", "隐藏", 1)
	id := int(data["id"].(float64))

	doRequest(r, http.MethodDelete, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 1, 1)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-types", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var list dto.TicketTypeListResponse
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	assert.Equal(t, int64(0), list.Total, "archived types excluded from default list")

	w = doRequest(r, http.MethodGet, "/api/v1/ticket-types?includeArchived=true", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp = parseResponse(t, w)
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	assert.Equal(t, int64(1), list.Total, "archived types included when requested")
}

func TestHandler_RestoreNonArchivedReturnsError(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "active_type", "活跃", 1)
	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodPost, fmt.Sprintf("/api/v1/ticket-types/%d/restore", id), nil, 1, 1)
	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_EnableDisable(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "toggle_type", "切换", 1)
	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodPost, fmt.Sprintf("/api/v1/ticket-types/%d/disable", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp := parseResponse(t, w)
	var disabled map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &disabled))
	assert.Equal(t, "inactive", disabled["status"])

	w = doRequest(r, http.MethodPost, fmt.Sprintf("/api/v1/ticket-types/%d/enable", id), nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp = parseResponse(t, w)
	var enabled map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &enabled))
	assert.Equal(t, "active", enabled["status"])
}

func TestHandler_Clone(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "clone_src", "克隆源", 1)
	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodPost, fmt.Sprintf("/api/v1/ticket-types/%d/clone", id), map[string]interface{}{
		"code": "clone_dst", "name": "克隆目标",
	}, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp := parseResponse(t, w)
	var cloned map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &cloned))
	assert.Equal(t, "clone_dst", cloned["code"])
	assert.Equal(t, "克隆目标", cloned["name"])
	assert.NotEqual(t, data["id"], cloned["id"])
}

func TestHandler_CrossTenantIsolation(t *testing.T) {
	r := setupTicketTypeRouter(t)

	data := createType(t, r, "tenant_a", "租户A", 1)
	id := int(data["id"].(float64))

	w := doRequest(r, http.MethodGet, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 2, 1)
	require.Equal(t, http.StatusNotFound, w.Code, "cross-tenant GET must 404")

	newName := "恶意修改"
	w = doRequest(r, http.MethodPut, fmt.Sprintf("/api/v1/ticket-types/%d", id), map[string]interface{}{
		"name": &newName,
	}, 2, 1)
	assert.Equal(t, http.StatusNotFound, w.Code, "cross-tenant PUT must 404")

	w = doRequest(r, http.MethodDelete, fmt.Sprintf("/api/v1/ticket-types/%d", id), nil, 2, 1)
	assert.Equal(t, http.StatusNotFound, w.Code, "cross-tenant DELETE must 404")
}

func TestHandler_CrossTenantListIsolation(t *testing.T) {
	r := setupTicketTypeRouter(t)

	createType(t, r, "isolated_a", "租户A类型", 1)
	createType(t, r, "isolated_b", "租户B类型", 2)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-types", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	var list dto.TicketTypeListResponse
	require.NoError(t, json.Unmarshal(resp.Data, &list))
	assert.Equal(t, int64(1), list.Total)
	assert.Equal(t, "isolated_a", list.Items[0].Code)
}

func TestHandler_InvalidIDReturnsBadRequest(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-types/abc", nil, 1, 1)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = doRequest(r, http.MethodPut, "/api/v1/ticket-types/abc", map[string]interface{}{"name": "x"}, 1, 1)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = doRequest(r, http.MethodDelete, "/api/v1/ticket-types/abc", nil, 1, 1)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PresetsListAndInstall(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodGet, "/api/v1/ticket-type-presets", nil, 1, 1)
	require.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	var presets []map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &presets))
	assert.GreaterOrEqual(t, len(presets), 4)

	w = doRequest(r, http.MethodPost, "/api/v1/ticket-type-presets/pacs-incident/install", map[string]interface{}{}, 1, 1)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	resp = parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	var installed map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Data, &installed))
	assert.Equal(t, "pacs_incident", installed["code"])
}

func TestHandler_InstallUnknownPresetReturnsNotFound(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/ticket-type-presets/nonexistent/install", map[string]interface{}{}, 1, 1)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}

func TestHandler_CreateMissingRequiredFields(t *testing.T) {
	r := setupTicketTypeRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/ticket-types", map[string]interface{}{
		"description": "只有描述",
	}, 1, 1)
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing code+name must fail validation")
}
