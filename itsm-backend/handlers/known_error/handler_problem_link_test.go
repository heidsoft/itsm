package known_error

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// KEDB 与问题的关联是「问题 → 已知错误」闭环的锚点：
//   - 前端问题详情页的「转为已知错误」会在 payload 里带 problemId；
//   - 后端若不声明该字段，json 解码会静默丢弃，记录与来源问题脱钩；
//   - 读取面若不带出 problemId，KEDB 列表无法跳回事故现场。

func newKnownErrorLinkRouter(t *testing.T, client *ent.Client, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(client), zaptest.NewLogger(t).Sugar())

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		c.Set("role", "super_admin")
		c.Set("client", client)
		c.Next()
	})
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func postKnownError(t *testing.T, r *gin.Engine, body map[string]interface{}) (int, map[string]interface{}) {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/known-errors", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "body=%s", w.Body.String())
	return w.Code, envelope
}

func TestCreateKnownErrorLinksProblem(t *testing.T) {
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ke_link_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	defer client.Close()
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("KEDB Link").SetCode("kedb-link").
		SetDomain("kedb-link.example.com").SetStatus("active").SaveX(ctx)
	p := client.Problem.Create().
		SetTenantID(tenant.ID).SetTitle("Collector 反复重启").SetDescription("d").
		SetPriority("high").SetStatus("identified").SetCreatedBy(1).
		SaveX(ctx)

	r := newKnownErrorLinkRouter(t, client, tenant.ID)
	status, envelope := postKnownError(t, r, map[string]interface{}{
		"title":      "Collector 内存泄漏",
		"workaround": "重启并限流",
		"problemId":  p.ID,
	})

	require.Equal(t, http.StatusOK, status, "body=%v", envelope)
	data, ok := envelope["data"].(map[string]interface{})
	require.True(t, ok)
	require.NotNil(t, data["problemId"], "响应必须回显来源问题")
	assert.Equal(t, float64(p.ID), data["problemId"])

	// 边必须真的落库，而不是只回显一个数字。
	stored, err := client.KnownError.Query().WithProblem().Only(ctx)
	require.NoError(t, err)
	require.Len(t, stored.Edges.Problem, 1)
	assert.Equal(t, p.ID, stored.Edges.Problem[0].ID)

	// 列表读取面同样要带出 problemId，否则 KEDB 列表无法跳回问题。
	status, envelope = doKnownErrorRequest(t, r, "/api/v1/known-errors")
	require.Equal(t, http.StatusOK, status)
	listData := envelope["data"].(map[string]interface{})
	items := listData["items"].([]interface{})
	require.Len(t, items, 1)
	first := items[0].(map[string]interface{})
	require.NotNil(t, first["problemId"])
	assert.Equal(t, float64(p.ID), first["problemId"])
}

func TestCreateKnownErrorRejectsForeignProblem(t *testing.T) {
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ke_link_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("KEDB A").SetCode("kedb-a").
		SetDomain("kedb-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("KEDB B").SetCode("kedb-b").
		SetDomain("kedb-b.example.com").SetStatus("active").SaveX(ctx)
	foreign := client.Problem.Create().
		SetTenantID(tenantB.ID).SetTitle("别家问题").SetDescription("d").
		SetPriority("low").SetStatus("open").SetCreatedBy(1).
		SaveX(ctx)

	r := newKnownErrorLinkRouter(t, client, tenantA.ID)
	status, envelope := postKnownError(t, r, map[string]interface{}{
		"title":     "越权关联",
		"problemId": foreign.ID,
	})

	// 404 而不是 400/500：不能把「别家租户是否存在这个 ID」暴露成可枚举信号。
	require.Equal(t, http.StatusNotFound, status, "body=%v", envelope)
	count, err := client.KnownError.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "校验失败不得留下半条记录")
}
