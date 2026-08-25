package email_intake

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"
)

func TestResolver_VerifiesActiveContractAndBranch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-intake-resolver?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	customer := client.ServiceCustomer.Create().
		SetTenantID(1).SetName("上海ABC有限公司").SetNormalizedName("上海abc有限公司").
		SetAliases([]string{"ABC"}).SaveX(ctx)
	branch := client.CustomerBranch.Create().
		SetTenantID(1).SetCustomerID(customer.ID).SetName("苏州分公司").SetNormalizedName("苏州分公司").
		SaveX(ctx)
	contract := client.SupportContract.Create().
		SetTenantID(1).SetCustomerID(customer.ID).SetBranchID(branch.ID).
		SetContractNumber("SUP-2026-008").SetNormalizedContractNumber("sup2026008").SetStatus("active").
		SaveX(ctx)

	result, err := NewResolver(client).Resolve(ctx, 1, IntakeFields{
		CustomerName: "ABC", BranchName: "苏州分公司", ReportedContractNumber: "SUP-2026-008",
	})
	require.NoError(t, err)
	require.Equal(t, ResolutionVerified, result.Status)
	require.Equal(t, customer.ID, result.CustomerID)
	require.Equal(t, branch.ID, result.BranchID)
	require.Equal(t, contract.ID, result.SupportContractID)
}

func TestOrchestrator_AutoCreatesOneIncidentAndDurableReply(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-intake-e2e?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("tenant").SetCode("email-e2e").SaveX(ctx)
	reporter := client.User.Create().SetTenantID(tenant.ID).SetUsername("automation").SetEmail("automation@example.com").SetPasswordHash("x").SetName("自动化").SetActive(true).SaveX(ctx)
	engineer := client.User.Create().SetTenantID(tenant.ID).SetUsername("engineer-e2e").SetEmail("engineer-e2e@example.com").SetPasswordHash("x").SetName("工程师").SetActive(true).SaveX(ctx)
	group := client.Group.Create().SetTenantID(tenant.ID).SetName("Network Team").AddMemberIDs(engineer.ID).SaveX(ctx)
	schedule := client.OnCallSchedule.Create().SetTenantID(tenant.ID).SetGroupID(group.ID).SetName("default").SaveX(ctx)
	now := time.Now()
	client.OnCallShift.Create().SetTenantID(tenant.ID).SetScheduleID(schedule.ID).SetUserID(engineer.ID).SetStartAt(now.Add(-time.Hour)).SetEndAt(now.Add(time.Hour)).SaveX(ctx)
	customer := client.ServiceCustomer.Create().SetTenantID(tenant.ID).SetName("上海ABC有限公司").SetNormalizedName("上海abc有限公司").SaveX(ctx)
	client.SupportContract.Create().SetTenantID(tenant.ID).SetCustomerID(customer.ID).SetContractNumber("SUP-1").SetNormalizedContractNumber("sup1").SetStatus("active").SaveX(ctx)

	incidentService := service.NewIncidentService(client, zaptest.NewLogger(t).Sugar())
	extractor := NewEmailIntakeExtractor(fakeEmailLLM{output: `{"intent":"report_incident","sourceOrganizationName":"","customerName":"上海ABC有限公司","branchName":"","reportedContractNumber":"SUP-1","title":"MPLS线路中断","description":"链路中断","occurredAt":"","impact":"high","urgency":"high","missingFields":[],"confidence":0.96}`}, "test-model")
	orchestrator := NewEmailIntakeOrchestrator(client, extractor, incidentService, OrchestratorConfig{Mode: ModeAutoCreate, AutomationReporterUserID: reporter.ID, DefaultAssignmentGroupID: &group.ID})
	email := ReceivedEmail{SenderAuthenticated: true, MailboxInstanceKey: "mailbox-1", UIDValidity: 10, UID: 20, ExternalMessageID: "<message@example.com>", FromAddress: "customer@example.com", ToAddresses: []string{"noc@example.com"}, Subject: "报障", PlainText: "线路中断", RawMIME: []byte("raw"), ReceivedAt: now}

	conversation, err := orchestrator.Ingest(ctx, tenant.ID, email)
	require.NoError(t, err)
	require.Equal(t, "INCIDENT_CREATED", conversation.Status)
	incidents, err := client.Incident.Query().Where(incident.TenantIDEQ(tenant.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	require.True(t, incidents[0].IsAutomated)
	require.Equal(t, engineer.ID, incidents[0].AssigneeID)
	require.NotNil(t, incidents[0].AssignmentGroupID)

	duplicate, err := orchestrator.Ingest(ctx, tenant.ID, email)
	require.NoError(t, err)
	require.Equal(t, conversation.ID, duplicate.ID)
	require.Equal(t, 1, client.Incident.Query().Where(incident.TenantIDEQ(tenant.ID)).CountX(ctx))
	require.Equal(t, 1, client.OperationalCommand.Query().Where(operationalcommand.CommandTypeEQ(commandbus.CommandSendIntakeEmail)).CountX(ctx))
}

func TestOrchestrator_MergesSupplementIntoExistingConversation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-intake-supplement?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	customer := client.ServiceCustomer.Create().SetTenantID(1).SetName("客户A").SetNormalizedName("客户a").SaveX(ctx)
	client.SupportContract.Create().SetTenantID(1).SetCustomerID(customer.ID).SetContractNumber("SUP-2").SetNormalizedContractNumber("sup2").SetStatus("active").SaveX(ctx)
	gateway := &sequenceEmailLLM{outputs: []string{
		`{"intent":"report_incident","sourceOrganizationName":"","customerName":"客户A","branchName":"","reportedContractNumber":"","title":"线路中断","description":"down","occurredAt":"","impact":"high","urgency":"high","missingFields":["reportedContractNumber"],"confidence":0.95}`,
		`{"intent":"report_incident","sourceOrganizationName":"","customerName":"","branchName":"","reportedContractNumber":"SUP-2","title":"","description":"","occurredAt":"","impact":"","urgency":"","missingFields":[],"confidence":0.94}`,
	}}
	orchestrator := NewEmailIntakeOrchestrator(client, NewEmailIntakeExtractor(gateway, "test"), nil, OrchestratorConfig{Mode: ModeManualConfirm})
	first, err := orchestrator.Ingest(ctx, 1, ReceivedEmail{SenderAuthenticated: true, MailboxInstanceKey: "box", UIDValidity: 1, UID: 1, ExternalMessageID: "<first>", FromAddress: "customer@example.com", Subject: "报障", PlainText: "客户A线路中断", RawMIME: []byte("first")})
	require.NoError(t, err)
	require.Equal(t, ResolutionNeedInfo, first.Status)
	second, err := orchestrator.Ingest(ctx, 1, ReceivedEmail{SenderAuthenticated: true, MailboxInstanceKey: "box", UIDValidity: 1, UID: 2, ExternalMessageID: "<second>", InReplyTo: "<first>", FromAddress: "customer@example.com", Subject: "Re: 报障", PlainText: "合同号 SUP-2", RawMIME: []byte("second")})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, ResolutionVerified, second.Status)
	detail := client.EmailConversation.Query().WithMessages().OnlyX(ctx)
	require.Len(t, detail.Edges.Messages, 2)
	require.Equal(t, "客户A", detail.CanonicalData["customerName"])
	require.Equal(t, "SUP-2", detail.CanonicalData["reportedContractNumber"])
}

func TestResolver_RejectsTerminatedAndCrossTenantContracts(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-intake-reject?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	customer := client.ServiceCustomer.Create().SetTenantID(1).SetName("客户A").SetNormalizedName("客户a").SaveX(ctx)
	client.SupportContract.Create().SetTenantID(1).SetCustomerID(customer.ID).
		SetContractNumber("STOP-1").SetNormalizedContractNumber("stop1").SetStatus("terminated").SaveX(ctx)

	result, err := NewResolver(client).Resolve(ctx, 1, IntakeFields{CustomerName: "客户A", ReportedContractNumber: "STOP-1"})
	require.NoError(t, err)
	require.Equal(t, ResolutionRejected, result.Status)

	other := client.ServiceCustomer.Create().SetTenantID(2).SetName("客户B").SetNormalizedName("客户b").SaveX(ctx)
	client.SupportContract.Create().SetTenantID(2).SetCustomerID(other.ID).
		SetContractNumber("OTHER-1").SetNormalizedContractNumber("other1").SetStatus("active").SaveX(ctx)
	result, err = NewResolver(client).Resolve(ctx, 1, IntakeFields{CustomerName: "客户A", ReportedContractNumber: "OTHER-1"})
	require.NoError(t, err)
	require.Equal(t, ResolutionManualReview, result.Status)
}

func TestOnCallService_RejectsOverlapAndFindsCurrentResolver(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-intake-oncall?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("tenant").SetCode("email-intake").SaveX(ctx)
	user := client.User.Create().SetTenantID(tenant.ID).SetUsername("engineer").SetEmail("engineer@example.com").
		SetPasswordHash("x").SetName("工程师").SetActive(true).SaveX(ctx)
	group := client.Group.Create().SetTenantID(tenant.ID).SetName("Network Team").AddMemberIDs(user.ID).SaveX(ctx)
	schedule := client.OnCallSchedule.Create().SetTenantID(tenant.ID).SetGroupID(group.ID).SetName("default").SaveX(ctx)
	now := time.Now().UTC()

	svc := NewOnCallService(client)
	_, err := svc.CreateShift(ctx, tenant.ID, schedule.ID, user.ID, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	_, err = svc.CreateShift(ctx, tenant.ID, schedule.ID, user.ID, now, now.Add(2*time.Hour))
	require.ErrorIs(t, err, ErrOverlappingShift)

	current, err := svc.CurrentResolver(ctx, tenant.ID, group.ID, now)
	require.NoError(t, err)
	require.NotNil(t, current)
	require.Equal(t, user.ID, current.UserID)
}
