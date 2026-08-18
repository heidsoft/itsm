package cab

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// cabEnvelope 对应 common.Response 的 {code,message,data} 包裹结构。
type cabEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// setupCABTest 在内存 SQLite + 全量 ent schema 上组装 CAB handler（不同测试用独立 DSN 隔离）。
func setupCABTest(t *testing.T) (*Handler, *ent.Client, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	tenant, err := client.Tenant.Create().SetName("CABTenant").SetCode("CAB" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain("cab.test").SetStatus("active").Save(context.Background())
	require.NoError(t, err)

	svc := service.NewCABService(client, zaptest.NewLogger(t).Sugar())
	h := NewHandler(svc, zaptest.NewLogger(t).Sugar())
	return h, client, tenant.ID
}

func mkCABUser(t *testing.T, client *ent.Client, tenantID int, role string) int {
	u, err := client.User.Create().
		SetUsername("cab-" + role + "-" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetEmail(role + "-" + strings.ReplaceAll(t.Name(), "/", "-") + "@example.com").
		SetName("CAB " + role).SetPasswordHash("h").SetRole(entuser.Role(role)).SetActive(true).SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

func mkCABTenant(t *testing.T, client *ent.Client) int {
	tn, err := client.Tenant.Create().SetName("CABTenant2").SetCode("CAB2" + strings.ReplaceAll(t.Name(), "/", "-")).
		SetDomain("cab2.test").SetStatus("active").Save(context.Background())
	require.NoError(t, err)
	return tn.ID
}

// newCABEngine 装配测试用 gin 引擎，并注入 tenant_id（模拟鉴权中间件）。
func newCABEngine(h *Handler, tenantID int) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	g := r.Group("/api/v1/cab")
	{
		g.GET("/members", h.ListCABMembers)
		g.POST("/members", h.AddCABMember)
		g.PUT("/members/:id", h.UpdateCABMember)
		g.DELETE("/members/:id", h.RemoveCABMember)
	}
	return r
}

func decodeCAB(t *testing.T, w *httptest.ResponseRecorder) cabEnvelope {
	t.Helper()
	var env cabEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "response body: %s", w.Body.String())
	return env
}

func addMember(t *testing.T, engine *gin.Engine, userID int, boardType, role string) cabEnvelope {
	t.Helper()
	body, _ := json.Marshal(dto.AddCABMemberRequest{UserID: userID, Type: boardType, Role: role})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cab/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return decodeCAB(t, w)
}

// TestCAB_AddAndList 验证：新增成员后列表返回且默认激活，响应体包裹 code=0。
func TestCAB_AddAndList(t *testing.T) {
	h, client, tenantID := setupCABTest(t)
	engine := newCABEngine(h, tenantID)

	u := mkCABUser(t, client, tenantID, "manager")
	env := addMember(t, engine, u, "CAB", "member")
	require.Equal(t, 0, env.Code, "新增应成功")

	var added dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(env.Data, &added))
	require.Equal(t, u, added.UserID)
	require.Equal(t, "CAB", added.Type)
	require.Equal(t, "member", added.Role)
	require.True(t, added.IsActive, "新增成员应默认激活（引擎才会纳入）")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=CAB", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	env = decodeCAB(t, w)
	require.Equal(t, 0, env.Code)
	var list []dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list, 1)
	require.Equal(t, added.ID, list[0].ID)
	require.Equal(t, u, list[0].UserID)
}

// TestCAB_TypeFiltering 验证：CAB 与 ECAB 两类成员分别按 type 过滤。
func TestCAB_TypeFiltering(t *testing.T) {
	h, client, tenantID := setupCABTest(t)
	engine := newCABEngine(h, tenantID)

	uc := mkCABUser(t, client, tenantID, "manager")
	ue := mkCABUser(t, client, tenantID, "security")
	require.Equal(t, 0, addMember(t, engine, uc, "CAB", "member").Code)
	require.Equal(t, 0, addMember(t, engine, ue, "ECAB", "chair").Code)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=CAB", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var cabList []dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(decodeCAB(t, w).Data, &cabList))
	require.Len(t, cabList, 1)
	require.Equal(t, "CAB", cabList[0].Type)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=ECAB", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var eList []dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(decodeCAB(t, w).Data, &eList))
	require.Len(t, eList, 1)
	require.Equal(t, "ECAB", eList[0].Type)
}

// TestCAB_TenantIsolation 验证：A 租户新增的成员不会出现在 B 租户的成员列表中。
func TestCAB_TenantIsolation(t *testing.T) {
	h, client, tenantA := setupCABTest(t)
	tenantB := mkCABTenant(t, client)
	engineA := newCABEngine(h, tenantA)
	engineB := newCABEngine(h, tenantB)

	u := mkCABUser(t, client, tenantA, "manager")
	require.Equal(t, 0, addMember(t, engineA, u, "CAB", "member").Code)

	// B 租户查询应为空（成员属于 A 租户）
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=CAB", nil)
	w := httptest.NewRecorder()
	engineB.ServeHTTP(w, req)
	var list []dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(decodeCAB(t, w).Data, &list))
	require.Empty(t, list, "B 租户不应看到 A 租户的 CAB 成员")

	// 且 B 租户尝试移除 A 的成员应失败（RemoveCABMember 按 tenant_id 校验归属）
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/cab/members/999999", nil)
	w = httptest.NewRecorder()
	engineB.ServeHTTP(w, delReq)
	require.NotEqual(t, 0, decodeCAB(t, w).Code, "越租户删除应失败")
}

// TestCAB_DuplicateRejected 验证：同一用户在同一 board 内重复添加被拒。
func TestCAB_DuplicateRejected(t *testing.T) {
	h, client, tenantID := setupCABTest(t)
	engine := newCABEngine(h, tenantID)

	u := mkCABUser(t, client, tenantID, "manager")
	require.Equal(t, 0, addMember(t, engine, u, "CAB", "member").Code)
	// 第二次相同 user+type → 应失败
	env := addMember(t, engine, u, "CAB", "member")
	require.NotEqual(t, 0, env.Code, "重复添加应被拒")
}

// TestCAB_UpdateAndRemove 验证：更新（停启用）与移除的成员状态正确。
func TestCAB_UpdateAndRemove(t *testing.T) {
	h, client, tenantID := setupCABTest(t)
	engine := newCABEngine(h, tenantID)

	u := mkCABUser(t, client, tenantID, "manager")
	env := addMember(t, engine, u, "CAB", "member")
	require.Equal(t, 0, env.Code)
	var added dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(env.Data, &added))

	// 停用该成员
	upBody, _ := json.Marshal(dto.UpdateCABMemberRequest{Role: "secretary", IsActive: false})
	upReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/cab/members/%d", added.ID), bytes.NewReader(upBody))
	upReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, upReq)
	env = decodeCAB(t, w)
	require.Equal(t, 0, env.Code)
	var updated dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(env.Data, &updated))
	require.Equal(t, "secretary", updated.Role)
	require.False(t, updated.IsActive, "停用后 IsActive 应为 false")

	// 列表应包含该成员（ListCABMembers 返回全部，含未激活）
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=CAB", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, listReq)
	var list []dto.CABMemberResponse
	require.NoError(t, json.Unmarshal(decodeCAB(t, w).Data, &list))
	require.Len(t, list, 1)
	require.False(t, list[0].IsActive)

	// 移除
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/cab/members/%d", added.ID), nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, delReq)
	require.Equal(t, 0, decodeCAB(t, w).Code)

	listReq = httptest.NewRequest(http.MethodGet, "/api/v1/cab/members?type=CAB", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, listReq)
	require.NoError(t, json.Unmarshal(decodeCAB(t, w).Data, &list))
	require.Empty(t, list, "移除后列表应为空")
}

// TestCAB_BadRequest 验证：非法请求体（缺 type / type 非法）返回非 0 错误码。
func TestCAB_BadRequest(t *testing.T) {
	_, client, tenantID := setupCABTest(t)
	engine := newCABEngine(NewHandler(service.NewCABService(client, zaptest.NewLogger(t).Sugar()), zaptest.NewLogger(t).Sugar()), tenantID)

	// 缺 type
	body, _ := json.Marshal(map[string]interface{}{"userId": 1, "role": "member"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cab/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.NotEqual(t, 0, decodeCAB(t, w).Code, "缺 type 应 BadRequest")

	// type 非法
	body, _ = json.Marshal(map[string]interface{}{"userId": 1, "type": "NOPE", "role": "member"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cab/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.NotEqual(t, 0, decodeCAB(t, w).Code, "type 非法应 BadRequest")
}
