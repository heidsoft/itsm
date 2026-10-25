package incident

// 回归：事件详情只有 resolutionSteps（步骤数组），没有工单那样的平字段
// resolution。agent/CLI 读事件时拿不到「怎么解决的」这一句，必须自己解析
// 步骤数组。resolve 动作本来就把说明写成最后一条 completed 步骤，所以详情
// 派生下发平字段，两者始终同源。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"
	"itsm-backend/service/sla"
)

type resolutionFixture struct {
	router   *gin.Engine
	client   *ent.Client
	tenantID int
	userID   int
}

func newResolutionFixture(t *testing.T) resolutionFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3", "file:incident_resolution_field?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("Resolution").SetCode("resolution").SetDomain("resolution.test").SaveX(ctx)
	user := client.User.Create().
		SetUsername("resolver").SetName("resolver").
		SetEmail("resolver@example.com").SetPasswordHash("hash").
		SetTenantID(tenant.ID).SaveX(ctx)

	production := service.NewIncidentService(client, zap.NewNop().Sugar(), nil)
	repo := NewEntRepository(client)
	slaMonitor := service.NewSLAMonitorService(client, zap.NewNop().Sugar())
	slaMonitor.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
	h := NewHandler(NewService(repo, nil, production, nil, nil, nil, slaMonitor, zap.NewNop().Sugar()))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Get 走 handlerctx.ResolveTenantID（需要 tenant 上下文对象），
		// Resolve 读的是 tenant_id/user_id 标量；两者都置上以覆盖真实链路。
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", user.ID)
	})
	r.GET("/api/v1/incidents/:id", h.Get)
	r.POST("/api/v1/incidents/:id/resolve", h.Resolve)

	return resolutionFixture{router: r, client: client, tenantID: tenant.ID, userID: user.ID}
}

func (f resolutionFixture) seedIncident(t *testing.T, ctx context.Context, status string) int {
	t.Helper()
	inc := f.client.Incident.Create().
		SetTitle("服务不可用").
		SetIncidentNumber("INC-resolution-" + status).
		SetStatus(status).
		SetReporterID(f.userID).
		SetTenantID(f.tenantID).
		SaveX(ctx)
	return inc.ID
}

func (f resolutionFixture) detail(t *testing.T, id int) map[string]interface{} {
	t.Helper()

	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/"+strconv.Itoa(id), nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var envelope struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Equal(t, 0, envelope.Code, w.Body.String())
	return envelope.Data
}

func TestIncidentDetail_ExposesFlatResolutionField(t *testing.T) {
	f := newResolutionFixture(t)
	ctx := context.Background()
	id := f.seedIncident(t, ctx, "in_progress")

	w := httptest.NewRecorder()
	body := `{"resolution":"重启应用节点并回滚到上一版本","rootCause":"新版本内存泄漏"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/"+strconv.Itoa(id)+"/resolve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data := f.detail(t, id)
	assert.Equal(t, "重启应用节点并回滚到上一版本", data["resolution"],
		"事件详情必须下发平字段 resolution，调用方不应被迫解析 resolutionSteps")
	assert.Equal(t, "resolved", data["status"])

	steps, ok := data["resolutionSteps"].([]interface{})
	require.True(t, ok, "resolutionSteps 必须仍然返回全量步骤: %v", data["resolutionSteps"])
	require.NotEmpty(t, steps)
	last, ok := steps[len(steps)-1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "completed", last["status"])
	assert.Equal(t, data["resolution"], last["description"], "平字段必须来自最后一条已完成步骤")
}

func TestIncidentDetail_NoCompletedStepOmitsResolution(t *testing.T) {
	f := newResolutionFixture(t)
	ctx := context.Background()
	id := f.seedIncident(t, ctx, "in_progress")

	// 只有进行中的步骤时不得伪造解决说明。
	f.client.Incident.UpdateOneID(id).
		SetResolutionSteps([]map[string]interface{}{
			{"step": 1, "description": "排查日志中", "status": "in_progress"},
		}).
		ExecX(ctx)

	data := f.detail(t, id)
	_, exists := data["resolution"]
	assert.False(t, exists, "没有已完成步骤时不得下发空 resolution，必须整键省略: %v", data["resolution"])
}
