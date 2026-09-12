package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/domain/provisioning"
	domainSR "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	entsr "itsm-backend/ent/servicerequest"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newProvisioningTestClient(t *testing.T) *ent.Client {
	t.Helper()
	return enttest.Open(t, dialect.SQLite, "file:provisioning-outbox?mode=memory&cache=shared&_fk=1")
}

func seedProvisioningPrerequisites(t *testing.T, ctx context.Context, client *ent.Client) (tenantID, requesterID int) {
	t.Helper()
	tenant, err := client.Tenant.Create().SetName("prov-outbox").SetCode("prov-outbox").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().
		SetTenantID(tenant.ID).
		SetUsername("requester").
		SetEmail("requester@example.com").
		SetName("requester").
		SetPasswordHash("x").
		SetActive(true).
		SetRole(user.RoleAdmin).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, requester.ID
}

func TestCreateTaskFromServiceRequestEnqueuesExecuteCommandAtomically(t *testing.T) {
	client := newProvisioningTestClient(t)
	defer client.Close()
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("provisioning").
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)

	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need ECS").
		SetStatus(string(domainSR.StatusSecurityApproved)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())
	task, err := svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.NoError(t, err)
	require.NotZero(t, task.ID)
	require.Equal(t, string(provisioning.TaskPending), task.Status)

	updatedReq, err := client.ServiceRequest.Query().Where(entsr.IDEQ(req.ID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "provisioning", updatedReq.Status, "ServiceRequest 状态必须随事务提交切到 provisioning")

	cmd, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("provisioning_task"), operationalcommand.AggregateIDEQ(task.ID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, commandbus.CommandExecuteProvisioningTask, cmd.CommandType)
	require.Equal(t, commandbus.StatusPending, cmd.Status)
	require.True(t, strings.HasPrefix(cmd.IdempotencyKey, "provisioning_task:"))
}

func TestCreateTaskFromServiceRequestRejectsNonApprovedState(t *testing.T) {
	client := newProvisioningTestClient(t)
	defer client.Close()
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("provisioning").
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)

	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need ECS").
		SetStatus(string(domainSR.StatusSubmitted)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())
	_, err = svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.Error(t, err)

	// 必须没有泄漏的 task / outbox 行
	count, err := client.ProvisioningTask.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "tx 失败时不应留 provisioning_task 行")
	cmdCount, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cmdCount, "tx 失败时不应留 operational_command 行")
}
