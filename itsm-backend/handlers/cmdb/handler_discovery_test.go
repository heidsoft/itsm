package cmdb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 云发现任务面已从 fail-closed（503 拒绝创建）转正为真实契约：
// 创建入队 / 列表 / 详情 / 取消 / 重试（路由见 router/cmdb_routes.go）。
//
// 原过渡期契约 TestCreateDiscoveryJobFailsClosedUntilProductionReady 锁定的
// 「禁止创建永远停在 queued 的假任务」由两条不变量兑现：
//  1. 创建与命令入队同一事务（repo.CreateDiscoveryJobWithCommand），不存在无命令的任务；
//  2. 命令执行器按 discovery_jobs 状态机推进，终态幂等（见 discovery_job_executor_test.go）。

func TestDiscoveryJobsHandler_ServiceNotWiredIs503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(nil)
	router.POST("/api/v1/cmdb/discovery/jobs", handler.CreateDiscoveryJob)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/cmdb/discovery/jobs",
		bytes.NewBufferString(`{"sourceId":"source-1"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	// 服务未装配仍是显式 503（部署残缺可见降级），但不再挡住正常装配路径。
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "5003")
}
