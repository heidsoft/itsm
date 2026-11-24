package bpmn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// P2-09 守卫（handler 层）：tenantIDFromCtx 必须拒绝 tenant_id=0。
//
// 旧实现只查 exists/类型：中间件写入 tenant_id=0 时放行，下游
// `TenantID > 0` 才加租户谓词的查询随即退化成全租户扫描——监控页展示
// 所有租户的实例。0 不是合法租户，与类型错误同口径按 401 拒绝。
// ---------------------------------------------------------------------------

// TestTenantIDFromCtx_RejectsNonPositive 直接守卫解析函数：0/负数必须拒绝，
// 且响应为 401 + AuthFailedCode(2001)（对齐上游中间件的认证失败口径）。
func TestTenantIDFromCtx_RejectsNonPositive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, value := range []int{0, -1} {
		t.Run(func() string {
			if value == 0 {
				return "zero"
			}
			return "negative"
		}(), func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Set("tenant_id", value)

			id, ok := tenantIDFromCtx(ctx)

			require.False(t, ok, "tenant_id=%d 必须拒绝", value)
			assert.Equal(t, 0, id)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "非正租户必须 401")

			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.AuthFailedCode, resp.Code, "业务码必须是认证失败 2001")
		})
	}
}

// TestTenantIDFromCtx_AcceptsPositive 正常路径不受 fail-closed 误伤。
func TestTenantIDFromCtx_AcceptsPositive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set("tenant_id", 7)

	id, ok := tenantIDFromCtx(ctx)
	require.True(t, ok, "合法租户不得被 fail-closed 拦下")
	assert.Equal(t, 7, id)
	assert.Equal(t, http.StatusOK, w.Code, "成功路径不得写响应体")
}

// TestListInstancesStatus_TenantZeroEndToEnd 端到端：中间件注入 tenant_id=0 时，
// GET /bpmn/monitoring/instances/status 必须 401，不得把全租户实例吐出来。
// 走真实路由 + 真实 service（enttest 内存库），锁住 handler→service 的整条链路。
func TestListInstancesStatus_TenantZeroEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:bpmn_mon_tenant0_e2e?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	audit := service.NewBPMNAuditService(client, logger)
	monitoring := NewMonitoringHandler(service.NewBPMNMonitoringService(client, audit, logger))

	// 造两个租户的实例：即使库里有数据，tenant_id=0 也必须被拦在 handler 层。
	seedE2EInstance(t, client, 1, "PI-E2E-T1")
	seedE2EInstance(t, client, 2, "PI-E2E-T2")

	r := gin.New()
	// 模拟上游中间件写入非法租户 0（认证上下文缺失的退化形态）。
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 0)
		c.Next()
	})
	monitoring.RegisterRoutes(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bpmn/monitoring/instances/status", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"tenant_id=0 必须在 handler 层 401，不得进入查询")

	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.AuthFailedCode, resp.Code)
	assert.Nil(t, resp.Data, "401 响应不得携带任何实例数据")
}

// seedE2EInstance 造一条最小可查的 process_instances。
func seedE2EInstance(t *testing.T, client *ent.Client, tenantID int, key string) {
	t.Helper()
	ctx := t.Context()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).
		SetDeploymentName("e2e seed").
		SetTenantID(tenantID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName(key).SetVersion("1").
		SetBpmnXML([]byte("<bpmn/>")).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantID).
		SetIsActive(true).SetIsLatest(true).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID(key).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
}
