package service_request

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/role"
	"itsm-backend/ent/servicerequestapproval"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 单管理员租户的真实故障形态：发起人就是唯一的「审批人」（审批角色下只有他），
// BPMN 按自审批禁令不给发起人指派待办，业务层继续走完成路径只会拿到引擎原始
// 错误并被包成 500「操作失败」。必须在动流程前返回可行动的业务错误。
func TestApplyApproval_SelfOnlyApproversReturnActionableConflict(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_self_approval.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("SelfApprovalTenant").
		SetCode("SA" + srUID()).
		SetDomain("sa.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("sa-requester-" + srUID()).
		SetEmail("sa-" + srUID() + "@example.com").
		SetName("单管理员").
		SetPasswordHash("hash").
		SetRole("admin").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	catalog, err := createServiceCatalogForTest(ctx, scSvc, "SA-Catalog-"+srUID(), "software", "self approval", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	sr, err := client.ServiceRequest.Create().
		SetTenantID(tenant.ID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requester.ID).
		SetTitle("自审批请求").
		SetStatus(SRStatusSubmitted).
		SetCurrentLevel(1).
		SetTotalLevels(1).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ServiceRequestApproval.Create().
		SetTenantID(tenant.ID).
		SetServiceRequestID(sr.ID).
		SetLevel(1).
		SetStatus(ApprovalStatusPending).
		SetNode(map[string]interface{}{"step_label": "manager", "approver_ids": []interface{}{float64(requester.ID)}, "quorum_threshold": float64(1)}).
		Save(ctx)
	require.NoError(t, err)

	svc := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)

	_, _, err = svc.ApplyApproval(ctx, sr.ID, tenant.ID, requester.ID, "approve", "自批", "admin", "IT")
	require.Error(t, err)

	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "必须是可映射 HTTP 的业务错误，不能是裸 error（会变 500）")
	require.Equal(t, common.ConflictCode, bizErr.Code)
	require.Contains(t, bizErr.Message, "禁止自审批")

	// 业务状态不得被改动：审批被拦下时请求仍停在待审状态。
	reloaded, err := client.ServiceRequest.Get(ctx, sr.ID)
	require.NoError(t, err)
	require.Equal(t, SRStatusSubmitted, reloaded.Status)
	require.Equal(t, 1, reloaded.CurrentLevel)

	approvals, err := client.ServiceRequestApproval.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, approvals, 1)
	require.Equal(t, ApprovalStatusPending, approvals[0].Status)
}

// 默认三级审批每级是「任一审批人批准即通过」：quorum_threshold 必须显式落 1。
// 不写时审批执行侧按候选人数兜底成本级会签，而发起人又被流程层禁止审批，
// 这一级永远凑不齐票，请求卡死在「已批 1/2 人」。
func TestCreate_DefaultApprovalLevelsCarrySingleApproverThreshold(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_default_quorum.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("QuorumTenant").
		SetCode("QT" + srUID()).
		SetDomain("qt.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("qt-requester-" + srUID()).
		SetEmail("qt-" + srUID() + "@example.com").
		SetName("申请人").
		SetPasswordHash("hash").
		SetRole("agent").
		SetDepartment("IT").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 三级审批每一级都要求「申请人以外」的可审批人，否则创建阶段 fail-closed。
	srSeedApprover(t, ctx, client, tenant.ID, "IT")

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	catalog, err := createServiceCatalogForTest(ctx, scSvc, "QT-Catalog-"+srUID(), "software", "quorum", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	svc := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	req := &ServiceRequest{
		TenantID:           tenant.ID,
		CatalogID:          catalog.ID,
		RequesterID:        requester.ID,
		Title:              "默认三级审批",
		Status:             SRStatusSubmitted,
		ComplianceAck:      true,
		DataClassification: "internal",
		ExpireAt:           expireInDays(30),
	}
	created, err := svc.Create(ctx, tenant.ID, requester.ID, catalog.ID, req)
	require.NoError(t, err)

	approvals, err := client.ServiceRequestApproval.Query().
		Where(servicerequestapproval.ServiceRequestIDEQ(created.ID)).
		Order(ent.Asc(servicerequestapproval.FieldLevel)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, approvals, 3)

	for _, approval := range approvals {
		// JSONB 往返后数字是 float64；int 形态只在内存态出现，两种都接受。
		var threshold int
		switch typed := approval.Node["quorum_threshold"].(type) {
		case float64:
			threshold = int(typed)
		case int:
			threshold = typed
		default:
			require.Failf(t, "缺少 quorum_threshold", "level %d 必须显式写阈值（缺省会被当成全员会签）", approval.Level)
		}
		require.Equal(t, 1, threshold, "默认三级审批每级一人通过即可")
	}
}

// expireInDays 返回 n 天后的时间指针，供要求资源过期时间的目录项使用。
func expireInDays(n int) *time.Time {
	at := time.Now().AddDate(0, 0, n)
	return &at
}

// srApprovalRoleCodes 是默认三级审批要求的角色码。
var srApprovalRoleCodes = []string{"manager", "it_admin", "security_admin"}

// srEnsureApprovalRoles 保证租户下三个审批角色存在，返回角色 ID。
// 角色按租户隔离（FindActiveUsersByRole 的 user_roles 分支要求 role.tenant_id 匹配），
// 因此不能复用其他租户的角色行。
func srEnsureApprovalRoles(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) []int {
	t.Helper()
	ids := make([]int, 0, len(srApprovalRoleCodes))
	for _, code := range srApprovalRoleCodes {
		existing, err := client.Role.Query().
			Where(role.CodeEQ(code), role.TenantIDEQ(tenantID)).
			Only(ctx)
		if err == nil {
			ids = append(ids, existing.ID)
			continue
		}
		created, err := client.Role.Create().
			SetName(code).
			SetCode(code).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}
	return ids
}

// srSeedApprover 建一名「除申请人外还能审批」的用户，绑定三个审批角色。
// 夹具里只有申请人自己时，创建会被 fail-closed 拦下——那正是真实单管理员
// 租户的形态（提交后请求卡死在审批环节），所以夹具必须反映真实租户：
// 至少有另一名具备审批角色的在职用户。
func srSeedApprover(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, dept string) int {
	t.Helper()
	roleIDs := srEnsureApprovalRoles(t, ctx, client, tenantID)
	approver, err := client.User.Create().
		SetUsername("sr-approver-" + srUID()).
		SetEmail("sr-approver-" + srUID() + "@example.com").
		SetName("SR Approver").
		SetPasswordHash("hash").
		SetRole("agent").
		SetDepartment(dept).
		SetActive(true).
		SetTenantID(tenantID).
		AddRoleIDs(roleIDs...).
		Save(ctx)
	require.NoError(t, err)
	return approver.ID
}

// normalizeApprovalPlan 必须把审批名单收敛成「引擎真正会指派的人」：
// 剔除发起人、把会签阈值钳到实际可批人数、无人可批时打显式标记。
// 三条逐处断言，不取并集——任一处漏改都会让请求重新卡死。
func TestNormalizeApprovalPlan_StripsRequesterAndClampsQuorum(t *testing.T) {
	svc := &Service{}

	staffed := []*ServiceRequestApproval{{
		Level:  1,
		Step:   ApprovalStepManager,
		Status: ApprovalStatusPending,
		Node: map[string]interface{}{
			"approver_ids":     []interface{}{float64(7), float64(8), float64(9)},
			"quorum_threshold": float64(3),
		},
	}}
	svc.normalizeApprovalPlan(staffed, 7)
	require.Equal(t, []int{8, 9}, nodeApproverIDs(staffed[0].Node), "发起人必须从候选里剔除（与引擎自审批禁令同口径）")
	require.Equal(t, 2, staffed[0].Node["quorum_threshold"], "三人会签在只剩两人可批时必须钳到 2")
	require.NotContains(t, staffed[0].Node, "approval_unstaffed", "有人可批的级别不得打标记")

	unstaffed := []*ServiceRequestApproval{{
		Level:  2,
		Step:   ApprovalStepIT,
		Status: ApprovalStatusPending,
		Node: map[string]interface{}{
			"approver_ids":     []interface{}{float64(7)},
			"quorum_threshold": float64(3),
		},
	}}
	svc.normalizeApprovalPlan(unstaffed, 7)
	require.Empty(t, nodeApproverIDs(unstaffed[0].Node))
	require.Equal(t, true, unstaffed[0].Node["approval_unstaffed"], "无人可批必须显式标记，不能靠『名单为空』推断")
	require.Equal(t, 1, unstaffed[0].Node["quorum_threshold"], "阈值至少为 1")

	// 反向验证：去掉标记后 ensureActionableApprovalPlan 不应再拦，否则守卫是假绿。
	delete(unstaffed[0].Node, "approval_unstaffed")
	require.NoError(t, svc.ensureActionableApprovalPlan(unstaffed))
}

// prod 形态（workflow outbox 启用 = 会启动真实 BPMN 流程）下，单管理员租户
// 提交服务请求必须在创建阶段被拒绝，并给出「缺哪个角色」的可行动文案；
// 同步路径（未启用 outbox）不受影响，补一名审批人后也必须放行。
func TestCreate_OutboxFailClosedWhenApprovalLevelUnstaffed(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_unstaffed.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("UnstaffedTenant").SetCode("US" + srUID()).SetDomain("us.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().
		SetUsername("us-requester-" + srUID()).
		SetEmail("us-" + srUID() + "@example.com").
		SetName("单管理员").SetPasswordHash("hash").SetRole("admin").
		SetDepartment("IT").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	catalog, err := createServiceCatalogForTest(ctx, scSvc, "US-Catalog-"+srUID(), "software", "unstaffed", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	newRequest := func() *ServiceRequest {
		return &ServiceRequest{
			Title: "无人可批的提交", ComplianceAck: true, DataClassification: "internal", ExpireAt: expireInDays(30),
		}
	}

	// ① 同步路径：不会启动真实 BPMN 流程，不存在空指派死路，允许创建。
	svcSync := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	created, err := svcSync.Create(ctx, tenant.ID, requester.ID, catalog.ID, newRequest())
	require.NoError(t, err, "同步审批路径不应被 fail-closed 拦下")
	require.NotNil(t, created)

	// ② prod 路径：启用 outbox 后，同一租户提交必须被拦在创建阶段。
	svcOutbox := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	svcOutbox.EnableWorkflowOutbox()
	_, err = svcOutbox.Create(ctx, tenant.ID, requester.ID, catalog.ID, newRequest())
	require.Error(t, err, "会启动真实 BPMN 流程时，无人可批的请求不能落库")
	require.Contains(t, err.Error(), "没有可审批的人")
	require.Contains(t, err.Error(), "role=manager", "文案必须点明缺哪个角色，否则用户无法行动")

	// ③ 反向验证：补一名除申请人外的审批人后，同一 prod 路径必须放行。
	approverID := srSeedApprover(t, ctx, client, tenant.ID, "IT")
	require.NotEqual(t, requester.ID, approverID)
	created, err = svcOutbox.Create(ctx, tenant.ID, requester.ID, catalog.ID, newRequest())
	require.NoError(t, err, "有可审批人后不应再拒绝创建")

	approvals, err := client.ServiceRequestApproval.Query().
		Where(servicerequestapproval.ServiceRequestIDEQ(created.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, approvals, 3)
	for _, a := range approvals {
		require.NotEmpty(t, nodeApproverIDs(a.Node), "第 %d 级必须解析出可审批人", a.Level)
		require.NotContains(t, a.Node, "approval_unstaffed")
	}
}

// 修复前落库的历史请求（node 带 approval_unstaffed 标记）点审批时，
// 必须返回可行动的 409，而不是把引擎的「无待办」原始错误包成 500。
func TestApplyApproval_UnstaffedHistoricApprovalIsActionableConflict(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_unstaffed_historic.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("HistoricTenant").SetCode("HT" + srUID()).SetDomain("ht.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().
		SetUsername("ht-requester-" + srUID()).
		SetEmail("ht-" + srUID() + "@example.com").
		SetName("申请人").SetPasswordHash("hash").SetRole("agent").
		SetDepartment("IT").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	catalog, err := createServiceCatalogForTest(ctx, scSvc, "HT-Catalog-"+srUID(), "software", "historic", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	sr, err := client.ServiceRequest.Create().SetTenantID(tenant.ID).SetCatalogID(catalog.ID).
		SetRequesterID(requester.ID).SetTitle("历史死路请求").SetStatus(SRStatusSubmitted).
		SetCurrentLevel(1).SetTotalLevels(1).Save(ctx)
	require.NoError(t, err)
	_, err = client.ServiceRequestApproval.Create().SetTenantID(tenant.ID).SetServiceRequestID(sr.ID).
		SetLevel(1).SetStep(ApprovalStepManager).SetStatus(ApprovalStatusPending).
		SetNode(map[string]interface{}{
			"step_label": "manager", "approver_ids": []interface{}{},
			"quorum_threshold": float64(1), "approval_unstaffed": true,
		}).Save(ctx)
	require.NoError(t, err)

	svc := NewService(NewEntRepository(client), scRepo, cmdb.NewEntRepository(client), client, logger, nil)
	svc.EnableWorkflowOutbox()
	_, _, err = svc.ApplyApproval(ctx, sr.ID, tenant.ID, requester.ID, "approve", "", "admin", "IT")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "必须是可映射 HTTP 的业务错误，不能是裸 error（会变 500）")
	require.Equal(t, common.ConflictCode, bizErr.Code)
	require.Contains(t, bizErr.Message, "没有可审批的人")
	require.Contains(t, bizErr.Message, "role=manager")
}

// 审批人名单有内存态 int 与 JSONB 往返态 float64 两种形态。
// checkEligibility 只认其中一种时，刚创建（未落库往返）的请求会把合法审批人
// 判成 403，而落库后又能通过——按请求生命周期出现两种结论，属于隐蔽的类型耦合。
func TestCheckEligibility_AcceptsBothIntAndJSONApproverIDShapes(t *testing.T) {
	svc := &Service{}
	for _, tc := range []struct {
		name string
		ids  []interface{}
	}{
		{"内存态 int（创建路径写入）", []interface{}{42}},
		{"JSONB 往返态 float64", []interface{}{float64(42)}},
		{"混合形态", []interface{}{float64(9), 42}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			approval := &ServiceRequestApproval{
				Step: ApprovalStepManager,
				Node: map[string]interface{}{"approver_ids": tc.ids},
			}
			require.NoError(t, svc.checkEligibility(42, "agent", "IT", "IT", approval),
				"名单里的合法审批人不得被误判为无权限")

			// 反向验证：名单外的用户仍必须被拒，避免守卫写成无条件放行。
			require.Error(t, svc.checkEligibility(999, "agent", "IT", "IT", approval))
		})
	}
}
