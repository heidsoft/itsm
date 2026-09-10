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
	return createCommandAt(t, client, tenantID, status, 10)
}

func createCommandAt(t *testing.T, client *ent.Client, tenantID int, status string, aggregateID int) *ent.OperationalCommand {
	t.Helper()
	command, err := client.OperationalCommand.Create().
		SetTenantID(tenantID).SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("incident").SetAggregateID(aggregateID).
		SetIdempotencyKey(fmt.Sprintf("incident:%d:workflow:start:%s", aggregateID, status)).
		SetPayload(map[string]interface{}{"accessToken": "must-not-leak", "incidentId": aggregateID}).
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

func TestListFiltersByCommandTypeAndAggregateType(t *testing.T) {
	client := newOperationsTestClient(t)
	a := createCommand(t, client, 1, commandbus.StatusPending)
	other, err := client.OperationalCommand.Create().
		SetTenantID(1).SetCommandType(commandbus.CommandDeliverNotification).
		SetAggregateType("ticket").SetAggregateID(99).
		SetIdempotencyKey("ticket:99:notify").
		SetStatus(commandbus.StatusPending).Save(context.Background())
	require.NoError(t, err)

	service := NewService(client)
	page, err := service.List(context.Background(), ListRequest{
		TenantID: 1, CommandType: commandbus.CommandStartBPMN, Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, a.ID, page.Items[0].ID)

	page, err = service.List(context.Background(), ListRequest{
		TenantID: 1, AggregateType: "ticket", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, other.ID, page.Items[0].ID)

	_, err = service.List(context.Background(), ListRequest{TenantID: 1, Status: "garbage"})
	require.Error(t, err)
}

func TestBulkReplayRestoresCommandsAndKeepsIdempotencyKey(t *testing.T) {
	client := newOperationsTestClient(t)
	a := createCommandAt(t, client, 1, commandbus.StatusDeadLetter, 11)
	b := createCommandAt(t, client, 1, commandbus.StatusDeadLetter, 12)
	_, err := client.OperationalCommand.UpdateOneID(a.ID).
		SetLastError("boom").SetAttempt(3).SetCompletedAt(time.Now()).Save(context.Background())
	require.NoError(t, err)
	service := NewService(client)

	result, err := service.BulkReplay(context.Background(), BulkFilter{
		TenantID: 1, Limit: 10,
	}, Actor{UserID: 7, Path: "/bulk-replay", Method: "POST"})
	require.NoError(t, err)
	require.Equal(t, 2, result.Updated)
	require.ElementsMatch(t, []int{a.ID, b.ID}, result.MatchedIDs)

	for _, id := range []int{a.ID, b.ID} {
		got, err := client.OperationalCommand.Get(context.Background(), id)
		require.NoError(t, err)
		require.Equal(t, commandbus.StatusPending, got.Status)
		require.Empty(t, got.LastError)
		require.NotNil(t, got.CreatedAt)
	}

	// audit 必须每条都写，不能只写一条
	audits, err := client.AuditLog.Query().
		Where(auditlog.ActionEQ("bulk_replay")).All(context.Background())
	require.NoError(t, err)
	require.Len(t, audits, 2)
}

func TestBulkReplayRejectsWrongStatusAndEmptyFilter(t *testing.T) {
	client := newOperationsTestClient(t)
	createCommand(t, client, 1, commandbus.StatusPending)
	service := NewService(client)

	_, err := service.BulkReplay(context.Background(), BulkFilter{TenantID: 1, Status: commandbus.StatusPending}, Actor{})
	require.Error(t, err)

	_, err = service.BulkReplay(context.Background(), BulkFilter{TenantID: 1}, Actor{})
	require.ErrorIs(t, err, ErrBulkEmpty)
}

func TestBulkCancelOnlyTouchesStuckLeasesWhenAsked(t *testing.T) {
	client := newOperationsTestClient(t)
	stuck := createCommandAt(t, client, 1, commandbus.StatusProcessing, 21)
	active := createCommandAt(t, client, 1, commandbus.StatusProcessing, 22)
	_, err := client.OperationalCommand.UpdateOneID(stuck.ID).
		SetLeaseOwner("worker-dead").
		SetLeaseExpiresAt(time.Now().Add(-5 * time.Minute)).
		SetFencingToken(2).Save(context.Background())
	require.NoError(t, err)
	_, err = client.OperationalCommand.UpdateOneID(active.ID).
		SetLeaseOwner("worker-live").
		SetLeaseExpiresAt(time.Now().Add(5 * time.Minute)).
		SetFencingToken(2).Save(context.Background())
	require.NoError(t, err)
	service := NewService(client)
	service.now = func() time.Time { return time.Now() }

	// 仅撤离 lease 过期：active 必须保持 processing
	result, err := service.BulkCancel(context.Background(), BulkFilter{
		TenantID: 1, LeaseExpired: true,
	}, Actor{UserID: 1, Path: "/bulk-cancel", Method: "POST"})
	require.NoError(t, err)
	require.Equal(t, []int{stuck.ID}, result.MatchedIDs)

	gotStuck, err := client.OperationalCommand.Get(context.Background(), stuck.ID)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, gotStuck.Status)
	require.Empty(t, gotStuck.LeaseOwner)

	gotActive, err := client.OperationalCommand.Get(context.Background(), active.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusProcessing, gotActive.Status)
}

func TestSummaryExposesStuckLeaseAndFailureRate(t *testing.T) {
	client := newOperationsTestClient(t)
	pending := createCommand(t, client, 1, commandbus.StatusPending)
	processing := createCommand(t, client, 1, commandbus.StatusProcessing)
	dead := createCommand(t, client, 1, commandbus.StatusDeadLetter)
	succeeded := createCommand(t, client, 1, commandbus.StatusSucceeded)

	// 把 processing 命令设为 lease 已过期，验证 stuckLeases
	_, err := client.OperationalCommand.UpdateOneID(processing.ID).
		SetLeaseExpiresAt(time.Now().Add(-time.Hour)).Save(context.Background())
	require.NoError(t, err)

	// 给 failed/succeeded 命令补 completed_at
	now := time.Now()
	_, err = client.OperationalCommand.UpdateOneID(dead.ID).
		SetCompletedAt(now).Save(context.Background())
	require.NoError(t, err)
	_, err = client.OperationalCommand.UpdateOneID(succeeded.ID).
		SetCompletedAt(now).Save(context.Background())
	require.NoError(t, err)
	_, err = client.OperationalCommand.UpdateOneID(pending.ID).
		SetCompletedAt(now).Save(context.Background())
	require.NoError(t, err)

	page, err := NewService(client).List(context.Background(), ListRequest{TenantID: 1, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 1, page.Summary.StuckLeases)
	require.Equal(t, 1, page.Summary.Pending)
	require.Equal(t, 1, page.Summary.DeadLetter)
	require.Equal(t, 1, page.Summary.Succeeded)

	var bpmnRow *CommandTypeStat
	for i := range page.ByTypeRows {
		if page.ByTypeRows[i].CommandType == commandbus.CommandStartBPMN {
			bpmnRow = &page.ByTypeRows[i]
		}
	}
	require.NotNil(t, bpmnRow)
	require.Equal(t, 1, bpmnRow.DeadLetter)
	require.Equal(t, 1, bpmnRow.SucceededRecent)
	require.Equal(t, 1, bpmnRow.FailedRecent)
	require.InDelta(t, 0.5, bpmnRow.FailureRate, 0.001)
}
