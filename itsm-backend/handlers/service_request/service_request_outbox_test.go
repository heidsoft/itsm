package service_request

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/internal/commandbus"
)

func srOutboxSetup(t *testing.T) (*EntRepository, *ent.Client, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_outbox_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("SROutboxTenant").
		SetCode(fmt.Sprintf("SROB-%d", time.Now().UnixNano())).
		SetDomain("sr-outbox.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("sr-outbox-user").
		SetEmail(fmt.Sprintf("sr-outbox-%d@example.com", time.Now().UnixNano())).
		SetName("Outbox User").
		SetPasswordHash("hash").
		SetRole("manager").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	return repo, client, tenant.ID
}

func TestCreateWithWorkflowCommand_EnqueuesOutbox(t *testing.T) {
	repo, client, tenantID := srOutboxSetup(t)
	ctx := context.Background()

	expireAt := time.Now().Add(72 * time.Hour)
	req := &ServiceRequest{
		TenantID:           tenantID,
		CatalogID:          1,
		RequesterID:        1,
		Status:             SRStatusSubmitted,
		Title:              "Outbox Test Request",
		Reason:             "testing outbox enqueue",
		DataClassification: "internal",
		ComplianceAck:      true,
		ExpireAt:           &expireAt,
		CurrentLevel:       1,
		TotalLevels:        3,
	}
	approvals := []*ServiceRequestApproval{
		{TenantID: tenantID, Level: 1, Step: ApprovalStepManager, Status: ApprovalStatusPending, TimeoutHours: 24},
	}

	created, err := repo.CreateWithWorkflowCommand(ctx, req, approvals)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Greater(t, created.ID, 0)

	cmds, err := client.OperationalCommand.Query().
		Where(
			operationalcommand.AggregateTypeEQ("service_request"),
			operationalcommand.AggregateIDEQ(created.ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, cmds, 1, "expected exactly one outbox command for service_request")

	cmd := cmds[0]
	require.Equal(t, commandbus.CommandStartBPMN, cmd.CommandType)
	require.Equal(t, fmt.Sprintf("service_request:%d:workflow:start", created.ID), cmd.IdempotencyKey)
	require.Equal(t, tenantID, cmd.TenantID)

	srCount, err := client.ServiceRequest.Query().
		Where(servicerequest.IDEQ(created.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, srCount)
}

func TestCreateWithWorkflowCommand_MultipleCreatesProduceDistinctCommands(t *testing.T) {
	repo, client, tenantID := srOutboxSetup(t)
	ctx := context.Background()

	expireAt := time.Now().Add(72 * time.Hour)
	approvals := []*ServiceRequestApproval{
		{TenantID: tenantID, Level: 1, Step: ApprovalStepManager, Status: ApprovalStatusPending, TimeoutHours: 24},
	}

	req1 := &ServiceRequest{
		TenantID: tenantID, CatalogID: 1, RequesterID: 1,
		Status: SRStatusSubmitted, Title: "First Request",
		DataClassification: "internal", ComplianceAck: true,
		ExpireAt: &expireAt, CurrentLevel: 1, TotalLevels: 3,
	}
	first, err := repo.CreateWithWorkflowCommand(ctx, req1, approvals)
	require.NoError(t, err)

	req2 := &ServiceRequest{
		TenantID: tenantID, CatalogID: 2, RequesterID: 1,
		Status: SRStatusSubmitted, Title: "Second Request",
		DataClassification: "internal", ComplianceAck: true,
		ExpireAt: &expireAt, CurrentLevel: 1, TotalLevels: 3,
	}
	second, err := repo.CreateWithWorkflowCommand(ctx, req2, approvals)
	require.NoError(t, err)

	cmds, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("service_request")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, cmds, 2, "each create should produce its own outbox command")

	require.Equal(t,
		fmt.Sprintf("service_request:%d:workflow:start", first.ID),
		cmds[0].IdempotencyKey)
	require.Equal(t,
		fmt.Sprintf("service_request:%d:workflow:start", second.ID),
		cmds[1].IdempotencyKey)
	require.NotEqual(t, cmds[0].IdempotencyKey, cmds[1].IdempotencyKey,
		"idempotency keys must be distinct per aggregate")
}

func TestService_EnableWorkflowOutbox_UsesOutboxPath(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_svc_outbox_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("SvcOutboxTenant").
		SetCode(fmt.Sprintf("SVC-OB-%d", time.Now().UnixNano())).
		SetDomain("svc-outbox.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("svc-outbox-user").
		SetEmail(fmt.Sprintf("svc-outbox-%d@example.com", time.Now().UnixNano())).
		SetName("Svc Outbox User").
		SetPasswordHash("hash").
		SetRole("manager").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := scSvc.Create(ctx, &service_catalog.ServiceCatalog{
		Name: "OutboxCatalog", Category: "software", Description: "for outbox test",
		DeliveryTime: 0, TenantID: tenant.ID, Status: "enabled",
	})
	require.NoError(t, err)

	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	svc := NewService(repo, scRepo, cmdbRepo, client, logger, nil)
	svc.EnableWorkflowOutbox()

	expireAt := time.Now().Add(72 * time.Hour)
	created, err := svc.Create(ctx, tenant.ID, user.ID, cat.ID, &ServiceRequest{
		Title:              "Service Outbox Test",
		Reason:             "testing service-level outbox",
		DataClassification: "internal",
		ComplianceAck:      true,
		NeedsPublicIP:      false,
		ExpireAt:           &expireAt,
	})
	require.NoError(t, err)
	require.NotNil(t, created)

	cmds, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("service_request")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, cmds, 1, "service-level outbox should enqueue exactly one command")
	require.Equal(t, commandbus.CommandStartBPMN, cmds[0].CommandType)
}
