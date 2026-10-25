package commandbus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
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
	// 每个测试独立内存库：共享库会让跨测试残留的 pending 命令
	// 被 worker.RunOnce 抢先领取，造成顺序依赖的假失败。
	return enttest.Open(t, dialect.SQLite, fmt.Sprintf("file:commandbus-%s?mode=memory&cache=shared&_fk=1", t.Name()))
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

// TestWorkerCancelsGoneAggregateImmediately 资源不存在（聚合已删除）属于永久失败，
// worker 必须立即 cancelled（而非 dead_letter），不得无谓重试占用调度轮次，也不得污染死信队列。
func TestWorkerCancelsGoneAggregateImmediately(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	registry := NewRegistry()
	calls := 0
	require.NoError(t, registry.Register(CommandStartBPMN, func(_ context.Context, _ *ent.OperationalCommand) error {
		calls++
		return &ent.NotFoundError{}
	}))
	cmd, err := Enqueue(ctx, client, EnqueueRequest{
		TenantID: 7, CommandType: CommandStartBPMN,
		AggregateType: "change", AggregateID: 99, IdempotencyKey: "change:99:workflow:start",
	})
	require.NoError(t, err)

	worker := NewWorker(client, registry, zap.NewNop().Sugar(), "worker-a")
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)

	stored, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, stored.Status, "聚合已删除应一次失败即 cancelled")
	require.Equal(t, 1, calls, "不得重试")
}

// TestWorkerCancelsCommonAppErrorNotFoundImmediately 领域层 common.NewNotFoundError
// 返回的 *common.AppError 也必须被识别为「聚合已删除」，直接 cancelled。
// 这是 prod 死信 208 条孤儿命令的真实错误类型（"ticket not found" 等），
// 修复前因未被 errors.As(*ent.NotFoundError) 识别而走满 8 次重试才进死信。
func TestWorkerCancelsCommonAppErrorNotFoundImmediately(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	registry := NewRegistry()
	calls := 0
	require.NoError(t, registry.Register(CommandDeliverNotification, func(_ context.Context, _ *ent.OperationalCommand) error {
		calls++
		return common.NewNotFoundError("ticket")
	}))
	cmd, err := Enqueue(ctx, client, EnqueueRequest{
		TenantID: 1, CommandType: CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: 404, IdempotencyKey: "ticket:404:notify",
	})
	require.NoError(t, err)

	worker := NewWorker(client, registry, zap.NewNop().Sugar(), "worker-a")
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)

	stored, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, stored.Status, "*common.AppError(NotFound) 应直接 cancelled")
	require.Equal(t, 1, calls, "不得重试")

	// 对照：普通瞬时错误仍应重试（pending），而非 cancelled / dead_letter。
	registry2 := NewRegistry()
	require.NoError(t, registry2.Register("transient.fail", func(context.Context, *ent.OperationalCommand) error {
		return errors.New("downstream timeout")
	}))
	cmd2, err := Enqueue(ctx, client, EnqueueRequest{
		TenantID: 2, CommandType: "transient.fail",
		AggregateType: "ticket", AggregateID: 5, IdempotencyKey: "ticket:5:transient", MaxAttempts: 3,
	})
	require.NoError(t, err)
	worker2 := NewWorker(client, registry2, zap.NewNop().Sugar(), "worker-b")
	_, err = worker2.RunOnce(ctx)
	require.NoError(t, err)
	stored2, err := client.OperationalCommand.Get(ctx, cmd2.ID)
	require.NoError(t, err)
	require.Equal(t, StatusPending, stored2.Status, "瞬时错误应继续重试（pending）")
}

func TestEnqueueTxIdempotentTreatsDuplicateKeyAsSkipped(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := context.Background()
	req := EnqueueRequest{
		TenantID: 7, CommandType: CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: 11, IdempotencyKey: "notification:dup",
		Payload: map[string]interface{}{"recipientId": 3},
	}

	first, err := client.Tx(ctx)
	require.NoError(t, err)
	cmd, skipped, err := EnqueueTxIdempotent(ctx, first, req)
	require.NoError(t, err)
	require.False(t, skipped)
	require.NotNil(t, cmd)
	require.Equal(t, req.IdempotencyKey, cmd.IdempotencyKey)
	require.NoError(t, first.Commit())

	second, err := client.Tx(ctx)
	require.NoError(t, err)
	cmd, skipped, err = EnqueueTxIdempotent(ctx, second, req)
	require.NoError(t, err)
	require.True(t, skipped)
	require.Nil(t, cmd)

	// 重复键必须只让该语句无操作，事务仍然可用；先查后插+撞约束会在这里把事务弄脏。
	other := req
	other.IdempotencyKey = "notification:other"
	_, skipped, err = EnqueueTxIdempotent(ctx, second, other)
	require.NoError(t, err)
	require.False(t, skipped)
	require.NoError(t, second.Commit())

	count, err := client.OperationalCommand.Query().
		Where(operationalcommand.IdempotencyKeyIn(req.IdempotencyKey, other.IdempotencyKey)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestEnqueueTxIdempotentRejectsCrossTenantContext(t *testing.T) {
	client := newCommandTestClient(t)
	ctx := tenantctx.WithTenantID(context.Background(), 1)
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	_, _, err = EnqueueTxIdempotent(ctx, tx, EnqueueRequest{
		TenantID: 2, CommandType: CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: 9, IdempotencyKey: "notification:cross-tenant",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cross-tenant")
}

func TestEnqueueSQLTxIgnoresDuplicateKey(t *testing.T) {
	dsn := fmt.Sprintf("file:commandbus-sql-%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, dialect.SQLite, dsn)
	defer client.Close()
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	req := EnqueueRequest{
		TenantID: 3, CommandType: CommandDeliverNotification,
		AggregateType: "change", AggregateID: 5, IdempotencyKey: "notification:change-5",
		Payload: map[string]interface{}{"recipientId": 8},
	}
	exec := func() error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if err := EnqueueSQLTx(ctx, tx, req); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}

	require.NoError(t, exec())
	require.NoError(t, exec(), "duplicate idempotency key must not abort the business transaction")

	count, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
