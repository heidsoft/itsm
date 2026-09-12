package group

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// groupEnvelope 对应 common.Response 的 {code,message,data} 包裹结构。
type groupEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// setupGroupTest 在内存 SQLite + 全量 ent schema 上组装 group handler，
// 并为 tenantA/tenantB 各建若干组。不同测试用独立 DSN 隔离。
func setupGroupTest(t *testing.T) (*Handler, *ent.Client, int, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()))

	ctx := context.Background()
	const tenantA, tenantB = 1, 2

	for _, name := range []string{"A-运维组", "A-服务台"} {
		_, err := client.Group.Create().SetName(name).SetTenantID(tenantA).Save(ctx)
		require.NoError(t, err)
	}
	for _, name := range []string{"B-财务组", "B-法务组", "B-研发组"} {
		_, err := client.Group.Create().SetName(name).SetTenantID(tenantB).Save(ctx)
		require.NoError(t, err)
	}

	h := NewHandler(service.NewGroupService(client), zaptest.NewLogger(t).Sugar())
	return h, client, tenantA, tenantB
}

// doGroupList 走真实 gin 路由发起 GET /groups，返回 HTTP 状态与响应包裹。
// tenantID<=0 时不注入租户上下文，用于验证 fail-closed 分支。
func doGroupList(t *testing.T, h *Handler, tenantID int, query string) (int, groupEnvelope) {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	router.GET("/groups", h.ListGroups)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/groups"+query, nil))

	var env groupEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return w.Code, env
}

func decodeGroups(t *testing.T, env groupEnvelope) *dto.PagedGroupsResponse {
	t.Helper()
	var resp dto.PagedGroupsResponse
	require.NoError(t, json.Unmarshal(env.Data, &resp))
	return &resp
}

// TestListGroups_IgnoresTenantIDQueryParam 跨租户 IDOR 回归。
//
// 修复前 handler「优先使用查询参数」的 tenantId，而 service 直接以
// req.TenantID 作查询谓词，任意登录用户传 ?tenantId=2 即可读取他租户的组。
// 修复后租户身份只能来自认证上下文，查询参数必须被完全忽略。
func TestListGroups_IgnoresTenantIDQueryParam(t *testing.T) {
	h, _, tenantA, tenantB := setupGroupTest(t)

	// 伪造他租户 tenantId，结果必须与不带参数完全一致
	spoofedStatus, spoofedEnv := doGroupList(t, h, tenantA, fmt.Sprintf("?tenantId=%d", tenantB))
	require.Equal(t, http.StatusOK, spoofedStatus)
	spoofed := decodeGroups(t, spoofedEnv)

	baselineStatus, baselineEnv := doGroupList(t, h, tenantA, "")
	require.Equal(t, http.StatusOK, baselineStatus)
	baseline := decodeGroups(t, baselineEnv)

	assert.Equal(t, 2, spoofed.Pagination.Total, "只能看到本租户的 2 个组")
	assert.Equal(t, baseline.Pagination.Total, spoofed.Pagination.Total)
	assert.Equal(t, len(baseline.Groups), len(spoofed.Groups))

	seen := make(map[string]bool, len(spoofed.Groups))
	for _, g := range spoofed.Groups {
		assert.Equal(t, tenantA, g.TenantID, "响应中不得出现他租户的组")
		assert.True(t, strings.HasPrefix(g.Name, "A-"), "只应返回 tenantA 的组: %s", g.Name)
		seen[g.Name] = true
	}
	assert.True(t, seen["A-运维组"] && seen["A-服务台"])
}

// TestListGroups_MissingTenantContextFailsClosed 缺少租户上下文必须 fail closed，
// 禁止退化为无租户谓词的全表查询或默认租户。
func TestListGroups_MissingTenantContextFailsClosed(t *testing.T) {
	h, client, _, _ := setupGroupTest(t)

	status, env := doGroupList(t, h, 0, "")

	// middleware.TenantIDOrUnauthorized 的契约：HTTP 401 + AuthFailedCode(2001)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, common.AuthFailedCode, env.Code)
	assert.Empty(t, env.Data, "未授权分支不得返回任何组数据")

	total, err := client.Group.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, total, "前置数据应包含两个租户共 5 个组")
}

// TestListGroups_TenantBSeesOnlyOwnGroups 反向验证隔离对称：
// 以 tenantB 身份请求同样无法通过参数读到 tenantA 的组。
func TestListGroups_TenantBSeesOnlyOwnGroups(t *testing.T) {
	h, _, tenantA, tenantB := setupGroupTest(t)

	status, env := doGroupList(t, h, tenantB, fmt.Sprintf("?tenantId=%d", tenantA))
	require.Equal(t, http.StatusOK, status)
	resp := decodeGroups(t, env)

	assert.Equal(t, 3, resp.Pagination.Total)
	for _, g := range resp.Groups {
		assert.Equal(t, tenantB, g.TenantID)
		assert.True(t, strings.HasPrefix(g.Name, "B-"), "只应返回 tenantB 的组: %s", g.Name)
	}
}
