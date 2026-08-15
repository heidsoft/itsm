package commandbus

import (
	"context"
	"errors"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newCommandTestClient(t *testing.T) *ent.Client {
	t.Helper()
	return enttest.Open(t, dialect.SQLite, "file:commandbus?mode=memory&cache=shared&_fk=1")
}

func TestWorkerProcessesCommandExactlyOnceAfterSuccess(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	registry := NewRegistry()
	calls := 0
	require.NoError(t, registry.Register(CommandStartBPMN, func(_ context.Context, cmd *ent.OperationalCommand) error {
		calls++
		require.Equal(t, 42, cmd.TenantID)
		return nil
	}))
	_, err := Enqueue(ctx, client, EnqueueRequest{
		TenantID: 42, CommandType: CommandStartBPMN,
		AggregateType: "incident", AggregateID: 7, IdempotencyKey: "incident:7:workflow:start",
	})
	require.NoError(t, err)

	worker := NewWorker(client, registry, zap.NewNop().Sugar(), "worker-a")
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	processed, err = worker.RunOnce(ctx)
	require.NoError(t, err)
	require.False(t, processed)
	require.Equal(t, 1, calls)
	stored, err := client.OperationalCommand.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, StatusSucceeded, stored.Status)
	require.Equal(t, 1, stored.Attempt)
}

func TestWorkerRetriesAndDeadLetters(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	registry := NewRegistry()
	require.NoError(t, registry.Register("always.fail", func(context.Context, *ent.OperationalCommand) error {
		return errors.New("provider secret must not appear here")
	}))
	cmd, err := Enqueue(ctx, client, EnqueueRequest{
		TenantID: 9, CommandType: "always.fail",
		AggregateType: "notification", AggregateID: 3, IdempotencyKey: "notification:3", MaxAttempts: 2,
	})
	require.NoError(t, err)

	worker := NewWorker(client, registry, zap.NewNop().Sugar(), "worker-a")
	now := time.Now()
	worker.now = func() time.Time { return now }
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	stored, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, stored.Status)

	now = stored.AvailableAt.Add(time.Second)
	processed, err = worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	stored, err = client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, StatusDeadLetter, stored.Status)
	require.Equal(t, 2, stored.Attempt)
}

func TestStaleFencingTokenCannotCompleteCommand(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	now := time.Now()
	cmd, err := client.OperationalCommand.Create().
		SetTenantID(1).SetCommandType("test").SetAggregateType("incident").SetAggregateID(1).
		SetIdempotencyKey("test:1").SetStatus(StatusProcessing).SetLeaseOwner("worker-b").
		SetLeaseExpiresAt(now.Add(time.Minute)).SetFencingToken(2).Save(ctx)
	require.NoError(t, err)

	_, err = client.OperationalCommand.UpdateOneID(cmd.ID).
		Where(operationalcommand.LeaseOwnerEQ("worker-a"), operationalcommand.FencingTokenEQ(1)).
		SetStatus(StatusSucceeded).Save(ctx)
	require.True(t, ent.IsNotFound(err))
	stored, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, StatusProcessing, stored.Status)
}

func TestEnqueueRejectsDuplicateIdempotencyKey(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	req := EnqueueRequest{TenantID: 1, CommandType: CommandStartBPMN, AggregateType: "change", AggregateID: 1, IdempotencyKey: "change:1"}
	_, err := Enqueue(ctx, client, req)
	require.NoError(t, err)
	_, err = Enqueue(ctx, client, req)
	require.True(t, ent.IsConstraintError(err))
}
