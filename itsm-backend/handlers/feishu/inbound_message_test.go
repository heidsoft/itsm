package feishu

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/connector"
	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newInboundFixture(t *testing.T, name string) (*ent.Client, *FeishuSyncService) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:feishu_inbound_%s_%d?mode=memory&cache=shared&_fk=1", name, time.Now().UnixNano()))
	return client, NewFeishuSyncService(client, zap.NewNop().Sugar())
}

func seedInboundTenant(t *testing.T, client *ent.Client, seed int) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("inbound-tenant-%d", seed)).
		SetCode(fmt.Sprintf("inbound-%d", seed)).
		SetDomain(fmt.Sprintf("inbound-%d.example.com", seed)).
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func seedInboundUser(t *testing.T, client *ent.Client, tenantID int, openID string) int {
	t.Helper()
	builder := client.User.Create().
		SetTenantID(tenantID).
		SetUsername(fmt.Sprintf("inbound-user-%d-%s", tenantID, openID)).
		SetEmail(fmt.Sprintf("inbound-%d-%s@example.com", tenantID, openID)).
		SetName("Inbound User").
		SetPasswordHash("hash").
		SetActive(true)
	if openID != "" {
		builder.SetFeishuOpenID(openID)
	}
	u, err := builder.Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

func seedInboundConfig(t *testing.T, client *ent.Client, tenantID int, key, value string) {
	t.Helper()
	_, err := client.SystemConfig.Create().
		SetTenantID(tenantID).
		SetKey(key).
		SetValue(value).
		Save(context.Background())
	require.NoError(t, err)
}

func inboundMessage(messageID, openID, text string) *connector.InboundMessage {
	return &connector.InboundMessage{
		ConnectorName: "feishu",
		ConnectorType: connector.TypeIM,
		MessageID:     messageID,
		UserID:        openID,
		Content:       text,
		Type:          "im.message.receive_v1",
		Raw:           []byte(fmt.Sprintf(`{"uuid":%q}`, messageID)),
	}
}

func TestInboundTicket_AutoCreateConsistentTenantAndDurableOutbox(t *testing.T) {
	client, svc := newInboundFixture(t, "auto")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 11)
	requester := seedInboundUser(t, client, tenantID, "ou-auto")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-auto-1", "ou-auto", "仓库温湿度告警\n请处理"))
	require.NoError(t, err)
	require.Equal(t, "ticket_created", resp.Action)
	require.Equal(t, "im.message.receive_v1", resp.EventType)
	require.Positive(t, resp.TicketID)
	require.NotEmpty(t, resp.TicketNumber)

	ticket, err := client.Ticket.Query().Where(entTicket.ID(resp.TicketID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, ticket.TenantID)
	require.Equal(t, requester, ticket.RequesterID)
	require.Equal(t, "仓库温湿度告警", ticket.Title)
	require.Contains(t, ticket.Description, "请处理")

	audit, err := client.AuditLog.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, audit.TenantID)
	require.Equal(t, "feishu_inbound", audit.Resource)
	require.Equal(t, "ticket.create", audit.Action)
	require.NotNil(t, audit.RequestBody)
	require.NotContains(t, *audit.RequestBody, "请处理", "消息正文不得进入审计")

	cmd, err := client.OperationalCommand.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, cmd.TenantID)
	require.Equal(t, commandbus.CommandDeliverNotification, cmd.CommandType)
	require.Contains(t, cmd.IdempotencyKey, "om-auto-1")
	require.Equal(t, ticket.ID, cmd.AggregateID)
}

func TestInboundTicket_DuplicateMessageDoesNotCreateSecondTicket(t *testing.T) {
	client, svc := newInboundFixture(t, "dup")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 12)
	seedInboundUser(t, client, tenantID, "ou-dup")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	msg := inboundMessage("om-dup-1", "ou-dup", "重复的故障消息")
	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, msg)
	require.NoError(t, err)

	_, err = svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, msg)
	require.True(t, errors.Is(err, errInboundDuplicate), "重复消息必须返回稳定冲突错误，得到 %v", err)

	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	commands, err := client.OperationalCommand.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
}

func TestInboundTicket_ObserveOnlyCreatesNothing(t *testing.T) {
	client, svc := newInboundFixture(t, "observe")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 13)
	seedInboundUser(t, client, tenantID, "ou-observe")

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-obs-1", "ou-observe", "只看不建"))
	require.NoError(t, err)
	require.Equal(t, "observed", resp.Action)
	require.Zero(t, resp.TicketID)

	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Empty(t, tickets)
}

func TestInboundTicket_UnknownSenderIsForbidden(t *testing.T) {
	client, svc := newInboundFixture(t, "forbidden")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 14)
	seedInboundUser(t, client, tenantID, "ou-known")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-fb-1", "ou-unknown", "未映射发送者"))
	require.True(t, errors.Is(err, errInboundRequesterForbidden), "未映射发送者必须 forbidden，得到 %v", err)
	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Empty(t, tickets)
}

func TestInboundTicket_UnconfiguredReporterFailsClosed(t *testing.T) {
	client, svc := newInboundFixture(t, "unconfigured")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 15)
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-uc-1", "", "无申请人"))
	require.True(t, errors.Is(err, errInboundUnconfigured), "无申请人且未配置自动化账号必须 unavailable，得到 %v", err)
}

func TestInboundTicket_AutomationReporterFallback(t *testing.T) {
	client, svc := newInboundFixture(t, "reporter")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 16)
	reporter := seedInboundUser(t, client, tenantID, "")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)
	seedInboundConfig(t, client, tenantID, inboundReporterConfigKey, fmt.Sprintf("%d", reporter))

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-rp-1", "", "自动化账号建单"))
	require.NoError(t, err)
	require.Equal(t, "ticket_created", resp.Action)
	ticket, err := client.Ticket.Query().Where(entTicket.ID(resp.TicketID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, reporter, ticket.RequesterID)
}

func TestInboundTicket_TenantIsolation(t *testing.T) {
	client, svc := newInboundFixture(t, "tenant")
	ctx := context.Background()
	tenantA := seedInboundTenant(t, client, 17)
	tenantB := seedInboundTenant(t, client, 18)
	seedInboundUser(t, client, tenantA, "ou-a")
	seedInboundUser(t, client, tenantB, "ou-b")
	seedInboundConfig(t, client, tenantA, inboundModeConfigKey, inboundModeAutoCreate)
	seedInboundConfig(t, client, tenantB, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantA, &feishuConn.Feishu{}, inboundMessage("om-t-1", "ou-a", "A 租户消息"))
	require.NoError(t, err)

	// 用 B 租户调用同一 open_id 必须拒绝：身份映射是租户本地的，不能跨租户借用。
	_, err = svc.CreateTicketFromInboundMessage(ctx, tenantB, &feishuConn.Feishu{}, inboundMessage("om-t-2", "ou-a", "跨租户注入"))
	require.True(t, errors.Is(err, errInboundRequesterForbidden), "跨租户借用发送者必须拒绝，得到 %v", err)

	owned, err := client.Ticket.Query().Where(entTicket.TenantIDEQ(tenantB)).All(ctx)
	require.NoError(t, err)
	require.Empty(t, owned)
}

func TestInboundTicket_DedupRowKeepsStableErrorClass(t *testing.T) {
	client, svc := newInboundFixture(t, "dedup")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 19)
	seedInboundUser(t, client, tenantID, "ou-dd")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-dd-1", "ou-dd", "去重状态检查"))
	require.NoError(t, err)
	rows, err := client.ConnectorInboundDedup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "ok", rows[0].ResponseStatus)
	require.Equal(t, inboundMessageDedupConnector, rows[0].ConnectorName)
}

// 用不匹配的 tenantctx 让 commandbus 的跨租户守卫在业务 INSERT 之后报错：
// 这会真实触发 enqueue 失败路径，验证建单/审计/命令同生同死，且去重行被释放。
func TestInboundTicket_EnqueueFailureRollsBackAllWrites(t *testing.T) {
	client, svc := newInboundFixture(t, "enqueue-rollback")
	tenantID := seedInboundTenant(t, client, 20)
	seedInboundUser(t, client, tenantID, "ou-rb")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	mismatchedCtx := tenantctx.WithTenantID(context.Background(), tenantID+1)
	msg := inboundMessage("om-rb-1", "ou-rb", "入箱失败必须整体回滚")
	_, err := svc.CreateTicketFromInboundMessage(mismatchedCtx, tenantID, &feishuConn.Feishu{}, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "enqueue inbound ticket notification")

	// 业务写入、审计、命令都不应残留。
	tickets, err := client.Ticket.Query().All(context.Background())
	require.NoError(t, err)
	require.Empty(t, tickets, "enqueue 失败后工单必须回滚")
	audits, err := client.AuditLog.Query().All(context.Background())
	require.NoError(t, err)
	require.Empty(t, audits, "enqueue 失败后审计必须回滚")
	commands, err := client.OperationalCommand.Query().All(context.Background())
	require.NoError(t, err)
	require.Empty(t, commands, "enqueue 失败后不得残留命令")

	// 去重行被释放，平台重投可以重新接管，不会永久丢事件。
	rows, err := client.ConnectorInboundDedup.Query().All(context.Background())
	require.NoError(t, err)
	require.Empty(t, rows)

	// 用正确租户上下文重投必须成功（反向证明回滚没有毒化状态）。
	resp, err := svc.CreateTicketFromInboundMessage(context.Background(), tenantID, &feishuConn.Feishu{}, msg)
	require.NoError(t, err)
	require.Equal(t, "ticket_created", resp.Action)
}

type inboundTestAdvisor struct {
	calls  int
	result dto.TriageResult
}

func (a *inboundTestAdvisor) SuggestForTenant(_ context.Context, _, _ string, _ int) dto.TriageResult {
	a.calls++
	return a.result
}

type inboundDecisionRecord struct {
	tenantID, userID     int
	scenario, inputRef   string
	promptVersion, model string
	confidence           float64
	suggestion           map[string]interface{}
}

type inboundTestAuditor struct {
	records []inboundDecisionRecord
	err     error
}

func (a *inboundTestAuditor) RecordDecisionAudit(_ context.Context, tenantID, userID int, _, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error) {
	if a.err != nil {
		return 0, a.err
	}
	a.records = append(a.records, inboundDecisionRecord{
		tenantID: tenantID, userID: userID, scenario: scenario, inputRef: inputRef,
		promptVersion: promptVersion, model: model, confidence: confidence, suggestion: suggestion,
	})
	return int64(len(a.records)), nil
}

func TestInboundTicket_TriageSuggestionIsLedgerOnly(t *testing.T) {
	client, svc := newInboundFixture(t, "triage")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 21)
	requester := seedInboundUser(t, client, tenantID, "ou-tri")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	advisor := &inboundTestAdvisor{result: dto.TriageResult{
		Category: "network", Priority: "high", Confidence: 0.91,
		Explanation: "关键词命中网络故障", Model: "keyword", PromptVersion: "triage-v1",
	}}
	auditor := &inboundTestAuditor{}
	svc.SetInboundAdvisors(advisor, auditor)

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-tri-1", "ou-tri", "网络中断"))
	require.NoError(t, err)
	require.NotNil(t, resp.TriageSuggestion)
	require.Equal(t, "high", resp.TriageSuggestion.Priority)
	require.EqualValues(t, 1, resp.TriageAuditID)
	require.Equal(t, 1, advisor.calls)
	require.Len(t, auditor.records, 1)

	rec := auditor.records[0]
	require.Equal(t, tenantID, rec.tenantID)
	require.Equal(t, requester, rec.userID)
	require.Equal(t, "feishu_inbound_triage", rec.scenario)
	require.Equal(t, fmt.Sprintf("ticket:%d", resp.TicketID), rec.inputRef)
	require.Equal(t, "triage-v1", rec.promptVersion)
	require.Equal(t, 0.91, rec.confidence)
	require.Equal(t, "high", rec.suggestion["priority"])

	// 建议不得自动应用：工单 priority 保持 schema 默认（建议为 high）。
	ticket, err := client.Ticket.Query().Where(entTicket.ID(resp.TicketID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "medium", ticket.Priority)
	require.Equal(t, "ticket", ticket.Type)
}

func TestInboundTicket_TriageLedgerFailureDoesNotBreakTicket(t *testing.T) {
	client, svc := newInboundFixture(t, "triage-ledger-fail")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 22)
	seedInboundUser(t, client, tenantID, "ou-tlf")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	svc.SetInboundAdvisors(
		&inboundTestAdvisor{result: dto.TriageResult{Category: "db", Priority: "high"}},
		&inboundTestAuditor{err: errors.New("ledger unavailable")},
	)
	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-tlf-1", "ou-tlf", "数据库慢查询"))
	require.NoError(t, err, "建议落账失败不应让已建工单失败")
	require.Equal(t, "ticket_created", resp.Action)
	require.NotNil(t, resp.TriageSuggestion)
	require.Zero(t, resp.TriageAuditID)
}
