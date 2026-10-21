// Package provisioning 的交付失败恢复 HTTP 回归测试。
//
// 与 cross_tenant_http_test.go 同一搭台方式（真实 router 布线 + JWT + RBAC），
// 从生产入口验证 2026-10-21 新增的两条恢复路径：
//
//   - POST /api/v1/provisioning-tasks/:id/retry   —— 失败任务重新排队
//   - POST /api/v1/provisioning-tasks/:id/resolve —— 自动重试走不通时的人工判定
//
// 钉住的契约：
//   - 已完成任务再次 execute 是 409/4090 且带可操作文案，不是 500；
//   - 重试只对 failed 生效，返回体里 status=pending、attemptCount 保留、
//     maxAttempts 可见（前端据此决定还显示不显示「重试」按钮）；
//   - 人工判定缺 note 是 400/1001，且任务状态零变化；
//   - 跨租户 retry/resolve 统一 404/4004，任务状态零变化；
//   - 未认证 401。
package provisioning_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func itoa(v int) string { return strconv.Itoa(v) }

func (fx *fixture) newTaskInStatus(t *testing.T, status string, attemptCount int) (*ent.ServiceRequest, *ent.ProvisioningTask) {
	t.Helper()
	sr := fx.newApprovedSR(t, "sr-"+status)
	// 任务状态与服务请求状态在真实链路里是同步推进的：failed 任务对应 failed 请求。
	// 若这里留成 security_approved，「重试把请求拉回 provisioning」这条断言会失去意义。
	srStatus := "provisioning"
	if status == "failed" {
		srStatus = "failed"
	}
	if status == "succeeded" {
		srStatus = "delivered"
	}
	sr = fx.client.ServiceRequest.UpdateOneID(sr.ID).SetStatus(srStatus).SaveX(context.Background())
	task := fx.client.ProvisioningTask.Create().
		SetTenantID(fx.tenantA.ID).
		SetServiceRequestID(sr.ID).
		SetProvider("alicloud").
		SetResourceType("ecs").
		SetPayload(map[string]any{"resource_type": "ecs"}).
		SetStatus(status).
		SetAttemptCount(attemptCount).
		SaveX(context.Background())
	return sr, task
}

func decodeEnvelope(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out), "body=%s", string(body))
	return out
}

// 已完成任务重复执行必须 409 且文案可操作：运维看到「不能重复执行」会去核对任务，
// 看到「执行交付任务失败」只会重试。
func TestExecuteSucceededTaskReturnsConflictNotServerError(t *testing.T) {
	fx := newFixture(t)
	_, task := fx.newTaskInStatus(t, "succeeded", 1)

	w := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/"+itoa(task.ID)+"/execute", fx.userA, nil)
	require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Equal(t, float64(common.ConflictCode), env["code"])
	msg, _ := env["message"].(string)
	assert.Contains(t, msg, "已完成")

	reloaded, err := fx.client.ProvisioningTask.Get(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", reloaded.Status)
	assert.Equal(t, 1, reloaded.AttemptCount)
}

func TestRetryFailedTaskRequeuesOverHTTP(t *testing.T) {
	fx := newFixture(t)
	sr, task := fx.newTaskInStatus(t, "failed", 1)

	w := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/"+itoa(task.ID)+"/retry", fx.userA,
		[]byte(`{"reason":"凭据已更新"}`))
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	data, ok := env["data"].(map[string]any)
	require.True(t, ok, "body=%s", w.Body.String())
	assert.Equal(t, "pending", data["status"])
	assert.Equal(t, float64(1), data["attemptCount"], "重试本身不执行，尝试次数在领取时 +1")
	assert.Equal(t, float64(5), data["maxAttempts"], "maxAttempts 必须可见，前端据此判断能否继续重试")

	cmd, err := fx.client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("provisioning_task"),
			operationalcommand.AggregateIDEQ(task.ID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "provisioning_task:"+itoa(task.ID)+":execute:2", cmd.IdempotencyKey)

	updatedSR, err := fx.client.ServiceRequest.Get(context.Background(), sr.ID)
	require.NoError(t, err)
	assert.Equal(t, "provisioning", updatedSR.Status, "重试期间服务请求必须回到 provisioning")
}

func TestRetrySucceededTaskReturnsConflict(t *testing.T) {
	fx := newFixture(t)
	_, task := fx.newTaskInStatus(t, "succeeded", 1)

	w := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/"+itoa(task.ID)+"/retry", fx.userA, nil)
	require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Equal(t, float64(common.ConflictCode), env["code"])
}

func TestResolveRequiresNote(t *testing.T) {
	fx := newFixture(t)
	_, task := fx.newTaskInStatus(t, "failed", 1)

	w := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/"+itoa(task.ID)+"/resolve", fx.userA,
		[]byte(`{"outcome":"delivered"}`))
	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

	reloaded, err := fx.client.ProvisioningTask.Get(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", reloaded.Status, "被拒绝的判定不得改变任务状态")
	assert.Zero(t, reloaded.ResolvedByUserID)
}

func TestResolveDeliveredMarksRequestDelivered(t *testing.T) {
	fx := newFixture(t)
	sr, task := fx.newTaskInStatus(t, "failed", 2)

	w := fx.do(t, http.MethodPost, "/api/v1/provisioning-tasks/"+itoa(task.ID)+"/resolve", fx.userA,
		[]byte(`{"outcome":"delivered","note":"资源已在控制台手工创建 i-abc123"}`))
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	data := env["data"].(map[string]any)
	assert.Equal(t, "succeeded", data["status"])
	assert.Equal(t, float64(fx.userA.ID), data["resolvedByUserId"])
	assert.Contains(t, data["resolutionNote"], "i-abc123")

	updatedSR, err := fx.client.ServiceRequest.Get(context.Background(), sr.ID)
	require.NoError(t, err)
	assert.Equal(t, "delivered", updatedSR.Status)
}

func TestRetryAndResolveAreTenantScopedOverHTTP(t *testing.T) {
	fx := newFixture(t)
	_, task := fx.newTaskInStatus(t, "failed", 1)

	for _, path := range []string{
		"/api/v1/provisioning-tasks/" + itoa(task.ID) + "/retry",
		"/api/v1/provisioning-tasks/" + itoa(task.ID) + "/resolve",
	} {
		var body []byte
		if path[len(path)-5:] == "retry" {
			body = nil
		} else {
			body = []byte(`{"outcome":"delivered","note":"跨租户判定"}`)
		}
		w := fx.do(t, http.MethodPost, path, fx.stranger, body)
		require.Equal(t, http.StatusNotFound, w.Code, "path=%s body=%s", path, w.Body.String())
		env := decodeEnvelope(t, w.Body.Bytes())
		assert.Equal(t, float64(common.NotFoundCode), env["code"], "path=%s", path)
	}

	reloaded, err := fx.client.ProvisioningTask.Get(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", reloaded.Status)
	assert.Zero(t, reloaded.ResolvedByUserID)
}

func TestRetryAndResolveRequireAuth(t *testing.T) {
	fx := newFixture(t)
	_, task := fx.newTaskInStatus(t, "failed", 1)

	for _, path := range []string{
		"/api/v1/provisioning-tasks/" + itoa(task.ID) + "/retry",
		"/api/v1/provisioning-tasks/" + itoa(task.ID) + "/resolve",
	} {
		w := fx.doNoAuth(t, http.MethodPost, path)
		require.Equal(t, http.StatusUnauthorized, w.Code, "path=%s", path)
	}
}
