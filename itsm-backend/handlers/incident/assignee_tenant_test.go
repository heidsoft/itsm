package incident

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
	"itsm-backend/service/sla"

	_ "github.com/mattn/go-sqlite3"
)

// Update 部分更新 AssigneeID 是跨租户指派入口，必须经
// productionService.ValidateAssignee 拒绝他租户用户。
func TestUpdate_RejectsCrossTenantAssignee(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident-assign-tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zap.NewNop().Sugar()

	tenant1, err := client.Tenant.Create().SetName("T1").SetCode("inc-t1").SetDomain("t1.test").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().SetName("T2").SetCode("inc-t2").SetDomain("t2.test").Save(ctx)
	require.NoError(t, err)
	reporter, err := client.User.Create().SetUsername("rep1").SetEmail("rep1@example.com").SetName("Rep").SetPasswordHash("hash").SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)
	assigneeT2, err := client.User.Create().SetUsername("a2").SetEmail("a2inc@example.com").SetName("A2").SetPasswordHash("hash").SetActive(true).SetTenantID(tenant2.ID).Save(ctx)
	require.NoError(t, err)

	inc := client.Incident.Create().
		SetTitle("跨租户指派回归").SetIncidentNumber("INC-ASSIGN-1").SetStatus("active").
		SetReporterID(reporter.ID).SetTenantID(tenant1.ID).SaveX(ctx)

	production := service.NewIncidentService(client, logger, nil)
	repo := NewEntRepository(client)
	slaMonitor := service.NewSLAMonitorService(client, logger)
	slaMonitor.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
	svc := NewService(repo, nil, production, nil, nil, nil, slaMonitor, logger)

	assigneeID := assigneeT2.ID
	_, err = svc.Update(ctx, tenant1.ID, inc.ID, &Incident{AssigneeID: &assigneeID}, reporter.ID, "super_admin")
	require.ErrorContains(t, err, "处理人不存在或不可用")

	reloaded, err := client.Incident.Get(ctx, inc.ID)
	require.NoError(t, err)
	require.Zero(t, reloaded.AssigneeID)
}
