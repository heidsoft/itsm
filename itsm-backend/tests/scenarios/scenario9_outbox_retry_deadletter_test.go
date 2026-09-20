package scenarios

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/handlers/operations"
	"itsm-backend/internal/commandbus"
)

// Scenario 9: Outbox retry / dead-letter / replay 行为测试
//
// 覆盖：
//   - Enqueue 校验（TenantID/AggregateID/CommandType/IdempotencyKey 必填）
//   - Worker.RunOnce 成功路径：pending → processing → succeeded
//   - Worker.RunOnce 失败重试：pending → processing → pending (exponential backoff)
//   - Worker.RunOnce 达到 maxAttempts → dead_letter
//   - Replay 只允许从 dead_letter / cancelled 恢复
//   - Cancel 只允许从 pending / processing 取消
//   - 跨租户隔离：tenant A 不能操作 tenant B 的 command

func TestScenario9_OutboxRetryDeadLetter(t *testing.T) {
	ctx := context.Background()
	dsn := scenarioDSN("outbox9")
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	registry := commandbus.NewRegistry()
	opsSvc := operations.NewService(client)
	actor := operations.Actor{UserID: 1, RequestID: "req-1", IP: "127.0.0.1", Path: "/ops", Method: "POST"}

	t.Run("enqueue validation rejects invalid requests", func(t *testing.T) {
		_, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 0, CommandType: "test", AggregateType: "ticket", AggregateID: 1, IdempotencyKey: "k1",
		})
		require.Error(t, err, "TenantID=0 应被拒绝")

		_, err = commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "", AggregateType: "ticket", AggregateID: 1, IdempotencyKey: "k2",
		})
		require.Error(t, err, "空 CommandType 应被拒绝")

		_, err = commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test", AggregateType: "ticket", AggregateID: 0, IdempotencyKey: "k3",
		})
		require.Error(t, err, "AggregateID=0 应被拒绝")

		_, err = commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test", AggregateType: "ticket", AggregateID: 1, IdempotencyKey: "",
		})
		require.Error(t, err, "空 IdempotencyKey 应被拒绝")
	})

	t.Run("enqueue valid request succeeds", func(t *testing.T) {
		cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.success", AggregateType: "ticket",
			AggregateID: 100, IdempotencyKey: "idem-001",
			Payload: map[string]interface{}{"action": "notify"},
		})
		require.NoError(t, err)
		require.Equal(t, commandbus.StatusPending, cmd.Status)
		require.Equal(t, 8, cmd.MaxAttempts, "默认 maxAttempts 应为 8")
	})

	t.Run("worker RunOnce processes command successfully", func(t *testing.T) {
		handlerCalled := false
		require.NoError(t, registry.Register("test.success", func(ctx context.Context, cmd *ent.OperationalCommand) error {
			handlerCalled = true
			return nil
		}))

		worker := commandbus.NewWorker(client, registry, logger, "test-worker")
		processed, err := worker.RunOnce(ctx)
		require.NoError(t, err)
		require.True(t, processed, "应处理至少一条命令")
		require.True(t, handlerCalled, "handler 应被调用")

		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-001")).
			All(ctx)
		require.Len(t, cmds, 1)
		require.Equal(t, commandbus.StatusSucceeded, cmds[0].Status)
	})

	t.Run("worker retries on handler failure with exponential backoff", func(t *testing.T) {
		require.NoError(t, registry.Register("test.retry", func(ctx context.Context, cmd *ent.OperationalCommand) error {
			return fmt.Errorf("transient error")
		}))

		_, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.retry", AggregateType: "ticket",
			AggregateID: 200, IdempotencyKey: "idem-retry", MaxAttempts: 3,
		})
		require.NoError(t, err)

		worker := commandbus.NewWorker(client, registry, logger, "test-worker-retry")

		processed, err := worker.RunOnce(ctx)
		require.NoError(t, err)
		require.True(t, processed)

		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-retry")).
			All(ctx)
		require.Len(t, cmds, 1)
		require.Equal(t, commandbus.StatusPending, cmds[0].Status, "首次失败应回退到 pending")
		require.Equal(t, 1, cmds[0].Attempt)
		require.Contains(t, cmds[0].LastError, "transient error")
	})

	t.Run("worker dead-letters after max attempts", func(t *testing.T) {
		require.NoError(t, registry.Register("test.deadletter", func(ctx context.Context, cmd *ent.OperationalCommand) error {
			return fmt.Errorf("permanent failure")
		}))

		_, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.deadletter", AggregateType: "ticket",
			AggregateID: 300, IdempotencyKey: "idem-dead", MaxAttempts: 1,
		})
		require.NoError(t, err)

		worker := commandbus.NewWorker(client, registry, logger, "test-worker-dl")
		processed, err := worker.RunOnce(ctx)
		require.NoError(t, err)
		require.True(t, processed)

		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-dead")).
			All(ctx)
		require.Len(t, cmds, 1)
		require.Equal(t, commandbus.StatusDeadLetter, cmds[0].Status, "达到 maxAttempts 应进入 dead_letter")
	})

	t.Run("replay from dead_letter restores to pending", func(t *testing.T) {
		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-dead")).
			All(ctx)
		require.Len(t, cmds, 1)
		cmdID := cmds[0].ID

		dto, err := opsSvc.Replay(ctx, 1, cmdID, actor)
		require.NoError(t, err)
		require.Equal(t, commandbus.StatusPending, dto.Status, "replay 应恢复到 pending")
		require.Empty(t, dto.LastError, "replay 应清除 lastError")
	})

	t.Run("replay rejected from non-dead-letter status", func(t *testing.T) {
		cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.noreplay", AggregateType: "ticket",
			AggregateID: 400, IdempotencyKey: "idem-noreplay",
		})
		require.NoError(t, err)

		_, err = opsSvc.Replay(ctx, 1, cmd.ID, actor)
		require.ErrorIs(t, err, operations.ErrInvalidState, "pending 状态不允许 replay")
	})

	t.Run("cancel from pending sets cancelled", func(t *testing.T) {
		cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.cancel", AggregateType: "ticket",
			AggregateID: 500, IdempotencyKey: "idem-cancel",
		})
		require.NoError(t, err)

		dto, err := opsSvc.Cancel(ctx, 1, cmd.ID, actor)
		require.NoError(t, err)
		require.Equal(t, operations.StatusCancelled, dto.Status)
	})

	t.Run("cancel rejected from succeeded status", func(t *testing.T) {
		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-001")).
			All(ctx)
		require.Len(t, cmds, 1)

		_, err := opsSvc.Cancel(ctx, 1, cmds[0].ID, actor)
		require.ErrorIs(t, err, operations.ErrInvalidState, "succeeded 状态不允许 cancel")
	})

	t.Run("cross-tenant isolation on replay", func(t *testing.T) {
		cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
			TenantID: 1, CommandType: "test.isolation", AggregateType: "ticket",
			AggregateID: 600, IdempotencyKey: "idem-iso", MaxAttempts: 1,
		})
		require.NoError(t, err)

		// Force to dead_letter for replay test
		_, err = opsSvc.Replay(ctx, 999, cmd.ID, actor)
		require.ErrorIs(t, err, operations.ErrCommandNotFound, "跨租户 replay 应返回 not found")
	})

	t.Run("replay from cancelled also works", func(t *testing.T) {
		cmds, _ := client.OperationalCommand.Query().
			Where(operationalcommand.IdempotencyKeyEQ("idem-cancel")).
			All(ctx)
		require.Len(t, cmds, 1)

		dto, err := opsSvc.Replay(ctx, 1, cmds[0].ID, actor)
		require.NoError(t, err)
		require.Equal(t, commandbus.StatusPending, dto.Status, "从 cancelled replay 应恢复到 pending")
	})
}
