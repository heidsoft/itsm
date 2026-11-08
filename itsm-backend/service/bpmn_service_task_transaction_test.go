package service

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ServiceTask 事务收敛回归锁（2026-11-08）：
//
// 旧实现的三段事务（执行意图 / 输出变量 / 完成事务）让「副作用完成凭据」
// 落在副作用之前，且输出变量有一个独立落库窗口。现把输出、完成凭据、
// receipt、审计、补偿登记、节点完成与流程推进收进同一个 ent.Tx：
//   - 完成事务回滚时，凭据/输出/流程推进必须一起消失（原子性）；
//   - 提交后再投递同一命令，凭据必须阻止副作用重复执行（幂等性）；
//   - 崩溃恢复按凭据缺失重放 handler（见 convergence 测试）。

func findConvergenceCommand(t *testing.T, ctx context.Context, client *ent.Client, tenantID, instanceID int) *ent.OperationalCommand {
	t.Helper()
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instanceID),
	).Only(ctx)
	require.NoError(t, err)
	return cmd
}

// TestServiceTaskCompletionTransactionRollsBackReceiptAndAdvance 完成事务里
// 任意一次写入失败都必须让「副作用完成凭据 + 输出变量 + 流程推进」一起回滚。
// 这里用 ent hook 精确炸掉带 _serviceTaskCompletedOccurrences 的实例变量写入。
func TestServiceTaskCompletionTransactionRollsBackReceiptAndAdvance(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handler, tenantID, workflowCtx := setupConvergenceEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "convergenceService", "change:900", map[string]interface{}{})
	require.NoError(t, err)
	cmd := findConvergenceCommand(t, ctx, client, tenantID, instance.ID)

	failCommit := true
	client.ProcessInstance.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if failCommit {
				if vars, ok := m.Field("variables"); ok {
					if varsMap, isMap := vars.(map[string]interface{}); isMap {
						if _, done := varsMap["_serviceTaskCompletedOccurrences"]; done {
							return nil, assert.AnError
						}
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	claimed := claimServiceTaskLease(t, ctx, client, cmd, "worker-atomic", 1)
	err = engine.HandleBPMNServiceTaskCommand(workflowCtx, claimed)
	require.Error(t, err, "完成事务必须因注入失败整体回滚")
	require.Equal(t, 1, handler.calls, "副作用已在完成事务外执行过一次")

	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", inst.Status, "完成事务回滚不得推进流程")
	assert.NotContains(t, inst.Variables, "serviceResult", "输出变量不得再走独立小事务提前落库")
	assert.Empty(t, intMapVariable(inst.Variables, "_serviceTaskExecutedOccurrences"),
		"副作用完成凭据必须与流程完成同事务，回滚后不可见")
	assert.NotContains(t, inst.Variables, "_serviceTaskReceipts", "receipt 不得先于完成事务落库")

	// 副作用前的「执行意图」仍独立提交（诊断用），但它不再是完成凭据；
	// 恢复路径正是靠「有意图、无完成凭据」判定需要重放。
	assert.Equal(t, 1, intMapVariable(inst.Variables, "_serviceTaskAttempts")["ServiceTask_1"])
}

// TestServiceTaskCompletedReceiptPreventsReplay 完成事务提交后，用更高的
// fencing token 重放同一条命令：handler 不得再次执行，流程状态保持不变。
func TestServiceTaskCompletedReceiptPreventsReplay(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handler, tenantID, workflowCtx := setupConvergenceEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "convergenceService", "change:901", map[string]interface{}{})
	require.NoError(t, err)
	cmd := findConvergenceCommand(t, ctx, client, tenantID, instance.ID)

	first := claimServiceTaskLease(t, ctx, client, cmd, "worker-first", 1)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(workflowCtx, first))
	require.Equal(t, 1, handler.calls)

	replay := claimServiceTaskLease(t, ctx, client, cmd, "worker-replay", 2)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(workflowCtx, replay))
	require.Equal(t, 1, handler.calls, "完成凭据必须阻止重复副作用")

	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", inst.Status)
	assert.Equal(t, 1, intMapVariable(inst.Variables, "_serviceTaskExecutedOccurrences")["ServiceTask_1"])
	assert.Equal(t, 1, intMapVariable(inst.Variables, "_serviceTaskAttempts")["ServiceTask_1"],
		"重放不得虚增执行次数")
}
