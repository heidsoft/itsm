package problem

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProblemCreateEnqueuesWorkflowStartCommand 是「问题域创建不落 workflow.start」
// 这条 Pilot 阻断项的回归锁：旧实现里 WorkflowStartCommandHandler 有 problem 消费分支
// 却没有任何生产者，问题单永远不进流程。断言必须落在真实的 operational_commands 行上，
// 只检查源码里出现字符串不足以证明单据创建与入箱同事务。
func TestProblemCreateEnqueuesWorkflowStartCommand(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:problem_wf_start_%s?mode=memory&cache=shared&_fk=1", t.Name()))
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("问题起流 E2E").SetCode("problem-wf-start-e2e").
		SetDomain("problem.e2e").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	creator, err := client.User.Create().SetUsername("problem_wf_creator").
		SetEmail("problem-wf@example.com").SetName("问题创建人").SetPasswordHash("hash").
		SetRole("technician").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	created, err := repo.Create(ctx, &Problem{
		Title: "网络间歇性丢包", Description: "待根因分析",
		Status: "open", Priority: "high", Category: "network",
		CreatedBy: creator.ID, TenantID: tenant.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	assert.NotEmpty(t, created.ProblemNumber, "问题编号仍须在同一事务内写入")

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("problem"),
		operationalcommand.AggregateIDEQ(created.ID),
	).Only(ctx)
	require.NoError(t, err, "问题创建必须投出 workflow.start 命令")
	assert.Equal(t, "pending", cmd.Status)
	assert.Equal(t, fmt.Sprintf("problem:%d:workflow:start", created.ID), cmd.IdempotencyKey)
	assert.Equal(t, created.ID, int(cmd.Payload["businessId"].(float64)),
		"payload 必须带上问题 ID，供 WorkflowStartCommandHandler 的 problem 分支加载单据")
}

// TestProblemCreateIsIdempotentPerProblem 每个问题只有一条起流意图；
// 不同问题各有自己的幂等键，不会被误判成重复而丢失起流。
func TestProblemCreateIsIdempotentPerProblem(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:problem_wf_start_%s?mode=memory&cache=shared&_fk=1", t.Name()))
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("问题起流幂等 E2E").SetCode("problem-wf-idem-e2e").
		SetDomain("problem.e2e").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	repo := NewEntRepository(client)
	for i := 1; i <= 2; i++ {
		_, err := repo.Create(ctx, &Problem{
			Title: fmt.Sprintf("问题 %d", i), Description: "回归",
			Status: "open", Priority: "medium", Category: "other",
			CreatedBy: 1, TenantID: tenant.ID,
		})
		require.NoError(t, err)
	}
	count, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("problem"),
	).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
