package incident

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
	"itsm-backend/service/sla"

	_ "github.com/mattn/go-sqlite3"
)

// TestCreate_RejectsCrossTenantAssignee 锁定创建路径的受指人同租户校验。
//
// 客户端请求体里的 assignee_id 可指向任意租户的用户：Update 路径早有
// productionService.ValidateAssignee 把关，Create 却完全没有——创建事件单时
// 能把单据指派给别家租户的员工，单据详情随后就把该用户暴露出来，形成跨租户
// 信息暴露面。本用例断言：跨租户受指人必须在开事务之前被拒绝（不落库）。
func TestCreate_RejectsCrossTenantAssignee(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident-create-assign-tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zap.NewNop().Sugar()

	tenant1, err := client.Tenant.Create().SetName("T1").SetCode("inc-ct1").SetDomain("ct1.test").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().SetName("T2").SetCode("inc-ct2").SetDomain("ct2.test").Save(ctx)
	require.NoError(t, err)

	reporter, err := client.User.Create().SetUsername("crep1").SetEmail("crep1@example.com").
		SetName("CRep").SetPasswordHash("hash").SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)
	// 他租户的在职用户：ID 合法、状态可用，唯一不合法之处是不属于 tenant1。
	assigneeT2, err := client.User.Create().SetUsername("ca2").SetEmail("ca2@example.com").
		SetName("CA2").SetPasswordHash("hash").SetActive(true).SetTenantID(tenant2.ID).Save(ctx)
	require.NoError(t, err)
	// 同租户在职用户：对照，必须放行。
	assigneeT1, err := client.User.Create().SetUsername("ca1").SetEmail("ca1@example.com").
		SetName("CA1").SetPasswordHash("hash").SetActive(true).SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)

	production := service.NewIncidentService(client, logger, nil)
	repo := NewEntRepository(client)
	slaMonitor := service.NewSLAMonitorService(client, logger)
	slaMonitor.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
	// 注：client 传 nil 时 Create 的 outbox 分支会跳过；本用例只断言受指人校验。
	svc := NewService(repo, nil, production, nil, nil, nil, slaMonitor, logger)

	// 跨租户受指人：拒绝，且不落库。
	foreign := assigneeT2.ID
	_, err = svc.Create(ctx, tenant1.ID, &Incident{
		Title:      "跨租户指派创建回归",
		ReporterID: reporter.ID,
		AssigneeID: &foreign,
	})
	require.ErrorContains(t, err, "处理人不存在或不可用")

	count, err := client.Incident.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "校验失败不得留下半条记录")

	// 同租户在职受指人：不得被误拒。
	//
	// 说明：本用例跑在 sqlite 上，Create 后续会卡在编号生成
	// （incidentNumberRepository 依赖 PostgreSQL 序列，测试环境无 raw DB），
	// 所以这里只断言「失败原因不是受指人校验」——误拒才是我要拦的回归，
	// 编号生成属于无关的前置条件。
	same := assigneeT1.ID
	_, err = svc.Create(ctx, tenant1.ID, &Incident{
		Title:      "同租户指派创建回归",
		ReporterID: reporter.ID,
		AssigneeID: &same,
	})
	if err != nil {
		require.NotContains(t, err.Error(), "处理人不存在或不可用",
			"同租户在职受指人不得被误拒")
	}
}
