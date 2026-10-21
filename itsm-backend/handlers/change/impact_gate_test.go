package change

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/incident"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 影响摘要消费门禁（validateImpactGate）的正向回归锁（2026-10-21）。
//
// 契约缺口收口前，GetCMDBImpactSummary 产出的 RequiresReview（关键 CI 命中 /
// 关联开放事件）只进 WorkflowHints 提示文案，从不被 SubmitChange 消费——
// 高风险特征变更以 low 风险等级照样提交成功，风险分析与提交流程断链。
//
// 门禁规则：RequiresReview && RiskLevel != "high" ⇒ 422（失败关闭，可操作文案）；
// 无高风险特征（RequiresReview=false）或 entClient 缺失（legacy 装配）⇒ 放行。

func newImpactGateTestService(t *testing.T) (*Service, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:change_impact_gate_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("变更门禁租户").SetCode("impact-gate").SetDomain("impact-gate.example.com").SetStatus("active").SaveX(ctx)

	repo := NewEntRepository(client, nil)
	svc := NewService(repo, client, zaptest.NewLogger(t).Sugar(), nil)
	return svc, ctx, tenant.ID
}

func TestValidateImpactGate(t *testing.T) {
	svc, ctx, tenantID := newImpactGateTestService(t)
	client := svc.entClient

	user := client.User.Create().SetUsername("impact-gate-u").SetEmail("impact-gate@example.com").SetName("U").
		SetPasswordHash("hash").SetRole("it_admin").SetActive(true).SetTenantID(tenantID).SaveX(ctx)

	// 关键 CI + 未关闭事件 ⇒ RequiresReview=true。
	ciType := client.CIType.Create().SetName("网络设备").SetTenantID(tenantID).SaveX(ctx)
	ci := client.ConfigurationItem.Create().SetName("核心交换机").SetCiType("network_device").
		SetCiTypeID(ciType.ID).
		SetCriticality("critical").SetStatus("in_service").SetEnvironment("production").
		SetTenantID(tenantID).SaveX(ctx)
	client.Incident.Create().SetIncidentNumber("INC-IG-001").SetTitle("核心交换机告警").SetDescription("d").SetStatus("open").
		SetPriority("high").SetConfigurationItemID(ci.ID).SetTenantID(tenantID).SetReporterID(user.ID).SaveX(ctx)

	change := client.Change.Create().SetTitle("更换板卡").SetDescription("d").SetStatus("draft").
		SetChangeNumber("C-IG-001").SetRiskLevel("low").SetCreatedBy(user.ID).
		SetTenantID(tenantID).SaveX(ctx)
	change.Update().AddAffectedCis(ci).SaveX(ctx)

	t.Run("关键CI加未关闭事件且风险等级为low时阻断提交", func(t *testing.T) {
		c, err := svc.repo.Get(ctx, change.ID, tenantID)
		require.NoError(t, err)

		gateErr := svc.validateImpactGate(ctx, change.ID, tenantID, c)
		require.Error(t, gateErr, "RequiresReview 且 riskLevel=low 必须阻断")
		assert.Contains(t, gateErr.Error(), "high", "错误文案必须给可操作指引（要求显式 high）")
	})

	t.Run("显式high风险等级时放行", func(t *testing.T) {
		change.Update().SetRiskLevel("high").SaveX(ctx)
		c, err := svc.repo.Get(ctx, change.ID, tenantID)
		require.NoError(t, err)
		assert.NoError(t, svc.validateImpactGate(ctx, change.ID, tenantID, c))
	})

	t.Run("无高风险特征时放行", func(t *testing.T) {
		// 事件关闭 + CI 降级为非关键 ⇒ RequiresReview=false。
		client.Incident.Update().Where(incident.TenantIDEQ(tenantID)).SetStatus("resolved").SaveX(ctx)
		ci.Update().SetCriticality("low").SaveX(ctx)
		change.Update().SetRiskLevel("low").SaveX(ctx)
		c, err := svc.repo.Get(ctx, change.ID, tenantID)
		require.NoError(t, err)
		assert.NoError(t, svc.validateImpactGate(ctx, change.ID, tenantID, c))
	})

	t.Run("entClient缺失时降级放行", func(t *testing.T) {
		legacySvc := NewService(svc.repo, nil, zaptest.NewLogger(t).Sugar(), nil)
		c, err := svc.repo.Get(ctx, change.ID, tenantID)
		require.NoError(t, err)
		assert.NoError(t, legacySvc.validateImpactGate(ctx, change.ID, tenantID, c))
	})
}
