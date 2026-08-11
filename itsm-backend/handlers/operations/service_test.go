package operations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func newOperationsTestClient(t *testing.T) *ent.Client {
	t.Helper()
	databaseName := strings.ReplaceAll(t.Name(), "/", "_")
	return enttest.Open(t, dialect.SQLite, fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", databaseName))
}

func createCommand(t *testing.T, client *ent.Client, tenantID int, status string) *ent.OperationalCommand {
	t.Helper()
	command, err := client.OperationalCommand.Create().
		SetTenantID(tenantID).SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("incident").SetAggregateID(10).
		SetIdempotencyKey("incident:10:workflow:start:" + status).
		SetPayload(map[string]interface{}{"accessToken": "must-not-leak", "incidentId": 10}).
		SetStatus(status).Save(context.Background())
	require.NoError(t, err)
	return command
}

func TestServiceIsTenantScopedAndSanitizesPayload(t *testing.T) {
	client := newOperationsTestClient(t)
	command := createCommand(t, client, 1, commandbus.StatusPending)
	service := NewService(client)

	_, err := service.Get(context.Background(), 2, command.ID)
	require.ErrorIs(t, err, ErrCommandNotFound)
	got, err := service.Get(context.Background(), 1, command.ID)
	require.NoError(t, err)
	require.Equal(t, "******", got.Payload["accessToken"])
	require.EqualValues(t, 10, got.Payload["incidentId"])
}

func TestReplayPreservesIdentityAndAttemptAndWritesAudit(t *testing.T) {
	client := newOperationsTestClient(t)
	command := createCommand(t, client, 1, commandbus.StatusDeadLetter)
	_, err := client.OperationalCommand.UpdateOneID(command.ID).
		SetAttempt(4).SetLastError("temporary failure").SetCompletedAt(time.Now()).Save(context.Background())
	require.NoError(t, err)
	service := NewService(client)
	now := time.Now().Add(time.Minute)
	service.now = func() time.Time { return now }

	got, err := service.Replay(context.Background(), 1, command.ID, Actor{
		UserID: 9, RequestID: "req-1", IP: "127.0.0.1",
		Path: "/api/v1/admin/operations/commands/1/replay", Method: "POST",
	})
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusPending, got.Status)
	require.Equal(t, 4, got.Attempt)
	require.Equal(t, command.IdempotencyKey, got.IdempotencyKey)
	require.Equal(t, command.FencingToken+1, got.FencingToken)

	audit, err := client.AuditLog.Query().Where(
		auditlog.TenantIDEQ(1), auditlog.ActionEQ("replay"),
	).Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, 9, audit.UserID)
	require.Contains(t, *audit.RequestBody, command.IdempotencyKey)
}

func TestCancelFencesProcessingOwnerAndRejectsInvalidReplay(t *testing.T) {
	client := newOperationsTestClient(t)
	command := createCommand(t, client, 1, commandbus.StatusProcessing)
	lease := time.Now().Add(time.Minute)
	command, err := client.OperationalCommand.UpdateOneID(command.ID).
		SetLeaseOwner("worker-a").SetLeaseExpiresAt(lease).SetFencingToken(7).Save(context.Background())
	require.NoError(t, err)
	service := NewService(client)

	cancelled, err := service.Cancel(context.Background(), 1, command.ID, Actor{UserID: 1, Path: "/cancel", Method: "POST"})
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, cancelled.Status)
	require.EqualValues(t, 8, cancelled.FencingToken)
	require.Empty(t, cancelled.LeaseOwner)
	require.Nil(t, cancelled.LeaseExpiresAt)

	_, err = service.Replay(context.Background(), 1, createCommand(t, client, 1, commandbus.StatusSucceeded).ID, Actor{})
	require.True(t, errors.Is(err, ErrInvalidState))
}

func TestListReturnsOperationalSummaryForTenant(t *testing.T) {
	client := newOperationsTestClient(t)
	createCommand(t, client, 1, commandbus.StatusPending)
	createCommand(t, client, 1, commandbus.StatusDeadLetter)
	createCommand(t, client, 2, commandbus.StatusPending)
	service := NewService(client)

	page, err := service.List(context.Background(), ListRequest{TenantID: 1, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Equal(t, 1, page.Summary.Pending)
	require.Equal(t, 1, page.Summary.DeadLetter)
	require.NotNil(t, page.Summary.OldestWaiting)
	for _, item := range page.Items {
		require.Equal(t, 1, item.TenantID)
		require.Nil(t, item.Payload)
	}

	count, err := client.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(2)).Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
