// Package provisioning 的启动交付状态守卫文案回归测试。
//
// 钉住的契约：状态不符（未经安全审批就启动交付）必须 409 且**带领域文案**
// ——「当前状态不允许启动交付（需要 security_approved）」，而不是被兜底文案
// 「创建交付任务失败」整段吞掉。运维拿到 409 时需要知道该去补审批链还是改
// 状态；只有「创建交付任务失败」时他只能翻后端日志。
//
// 历史：StartProvisioning 曾用 common.FailWithErr，它会把 AppError/BusinessError
// 的安全文案替换成 handler 兜底文案（writeClassified 收 publicMsg）。同类
// 「文案被吞」问题在仓库里是已知地雷，透传统一走 RespondError。
package provisioning_test

import (
	"context"
	"net/http"
	"testing"

	"itsm-backend/common"
	domainServiceRequest "itsm-backend/domain/servicerequest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 未达 security_approved 的请求启动交付：409 + 可操作文案，且不含兜底文案。
func TestStartProvisioningSurfacesStateGuardReasonNotFallback(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()

	catalog := fx.client.ServiceCatalog.Create().
		SetName("catalog for submitted sr").
		SetStatus("active").
		SetTenantID(fx.tenantA.ID).
		SaveX(ctx)
	sr := fx.client.ServiceRequest.Create().
		SetTitle("sr-submitted-guard").
		SetStatus(domainServiceRequest.StatusSubmitted).
		SetCatalogID(catalog.ID).
		SetRequesterID(fx.userA.ID).
		SetTenantID(fx.tenantA.ID).
		SaveX(ctx)

	w := fx.do(t, http.MethodPost, "/api/v1/service-requests/"+itoa(sr.ID)+"/provision", fx.userA, nil)
	require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Equal(t, float64(common.ConflictCode), env["code"])
	msg, _ := env["message"].(string)
	assert.Contains(t, msg, "security_approved", "409 必须说明缺哪一级审批: %s", msg)
	assert.NotContains(t, msg, "创建交付任务失败", "兜底文案不得覆盖领域文案: %s", msg)

	// 状态守卫拒绝时不得留下任何交付任务与状态改写。
	tasks := fx.client.ProvisioningTask.Query().CountX(ctx)
	require.Equal(t, 0, tasks, "状态不符不得创建交付任务")
}

// 反向钉住：非状态类失败（租户内不存在）不得把内部错误细节透出，
// 仍走 404 + 稳定文案。RespondError 的「未分类回落兜底」不能被改坏。
func TestStartProvisioningUnknownRequestStaysStable404(t *testing.T) {
	fx := newFixture(t)

	w := fx.do(t, http.MethodPost, "/api/v1/service-requests/999999/provision", fx.userA, nil)
	require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())

	env := decodeEnvelope(t, w.Body.Bytes())
	assert.Equal(t, float64(common.NotFoundCode), env["code"])
	msg, _ := env["message"].(string)
	assert.NotEmpty(t, msg)
}
