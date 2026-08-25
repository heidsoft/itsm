package email_intake

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

func TestOutboundHandlerDoesNotReclaimActiveFirstAttempt(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:email-outbound-cas?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	conversation := client.EmailConversation.Create().SetTenantID(1).SetConversationToken("token-cas").SetLastMessageAt(time.Now()).SaveX(ctx)
	outbound := client.EmailOutboundMessage.Create().SetTenantID(1).SetConversationID(conversation.ID).SetMailboxInstanceKey("mailbox").SetReplyType("missing_information").SetRevision(1).SetToAddress("customer@example.com").SetSubject("subject").SetBodyText("body").SetStatus("SENDING").SaveX(ctx)
	manager := connector.NewManager(connector.NewRegistry(), zaptest.NewLogger(t).Sugar())
	defer manager.CloseAll()
	handler := NewOutboundCommandHandler(client, manager)

	err := handler.Handle(ctx, &ent.OperationalCommand{TenantID: 1, AggregateType: "email_outbound_message", AggregateID: outbound.ID, Attempt: 1})
	require.NoError(t, err)
	reloaded := client.EmailOutboundMessage.GetX(ctx, outbound.ID)
	require.Equal(t, "SENDING", reloaded.Status)
	require.Zero(t, reloaded.Attempts)
}
