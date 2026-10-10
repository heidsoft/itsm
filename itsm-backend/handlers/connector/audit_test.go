package connector

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"itsm-backend/ent/enttest"
	domainCommon "itsm-backend/handlers/common"
)

func TestRecordInboundFailureRedactsSensitiveHeadersAndPersists(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:connector-audit?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	RecordInboundFailure(ctx, domainCommon.NewEntRepository(client), 1,
		"dingtalk.webhook.verify_signature", "/dingtalk/webhook/1",
		"signature mismatch",
		map[string]string{"Signature": "secret-sig", "X-Custom": "plain"},
		zap.NewNop().Sugar())

	audit, err := client.AuditLog.Query().Only(ctx)
	if err != nil {
		t.Fatalf("query audit log: %v", err)
	}
	if audit.TenantID != 1 || audit.Action != "dingtalk.webhook.verify_signature" ||
		audit.Resource != "connector_inbound" || audit.StatusCode != 401 {
		t.Fatalf("unexpected audit row: %+v", audit)
	}
	if audit.RequestBody == nil {
		t.Fatalf("expected audit request body to be persisted")
	}
	if strings.Contains(*audit.RequestBody, "secret-sig") {
		t.Fatalf("sensitive header leaked into audit body: %s", *audit.RequestBody)
	}
	if !strings.Contains(*audit.RequestBody, "***") || !strings.Contains(*audit.RequestBody, "plain") {
		t.Fatalf("expected redacted headers in audit body: %s", *audit.RequestBody)
	}
}

func TestRecordInboundFailureToleratesMissingDependencies(t *testing.T) {
	RecordInboundFailure(context.Background(), nil, 1, "a", "/p", "r", nil, nil)
	RecordInboundFailure(context.Background(), domainCommon.NewEntRepository(nil), 1, "a", "/p", "r", nil, nil)
}
