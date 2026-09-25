package service_request

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entuser "itsm-backend/ent/user"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/service"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupApprovalChainTest 组装带审批链求值引擎的服务请求 Service（测试库隔离）。
func setupApprovalChainTest(t *testing.T) (*Service, *ent.Client, int, int) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_chain_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("ChainTenant").SetCode("CH" + srUID()).SetDomain("ch.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)
	cat, err := createServiceCatalogForTest(ctx, scSvc, "ChainCatalog-"+srUID(), "software", "for test", 0, tenant.ID, "enabled", 0, 0)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	cmdbRepo := cmdb.NewEntRepository(client)
	approvalChainSvc := service.NewApprovalChainService(client, logger)
	svc := NewService(repo, scRepo, cmdbRepo, client, logger, approvalChainSvc)
	return svc, client, tenant.ID, cat.ID
}

func mkChainUser(t *testing.T, client *ent.Client, tenantID int, role, dept string) int {
	t.Helper()
	u, err := client.User.Create().
		SetUsername("chain-" + role + "-" + srUID()).
		SetEmail(role + "-" + srUID() + "@example.com").
		SetName("Chain " + role).
		SetPasswordHash("h").
		SetRole(entuser.Role(role)).
		SetDepartment(dept).
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

func chainReqData(title string) *ServiceRequest {
	return &ServiceRequest{
		Title:              title,
		ComplianceAck:      true,
		DataClassification: "internal",
		ExpireAt:           timePtr(time.Now().Add(30 * 24 * time.Hour)),
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// Serial approval configuration alone cannot authorize a write. Once a real
// BPMN graph is present, its two tasks drive the corresponding business levels.
func TestServiceRequest_ApprovalChain_SerialProgression(t *testing.T) {
	svc, client, tenantID, catalogID := setupApprovalChainTest(t)
	ctx := context.Background()

	chainReq := &dto.ApprovalChainRequest{
		Name:       "SR Chain",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "Mgr", IsRequired: true, ApprovalType: "serial"},
			{Level: 2, Role: "agent", Name: "IT", IsRequired: true, ApprovalType: "serial"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	mgr := mkChainUser(t, client, tenantID, "manager", "IT")
	it := mkChainUser(t, client, tenantID, "agent", "IT")
	requester := mkChainUser(t, client, tenantID, "end_user", "IT")

	created, err := svc.Create(ctx, tenantID, requester, catalogID, chainReqData("Chain SR"))
	require.NoError(t, err)
	require.Equal(t, 2, created.TotalLevels, "审批链 2 级应映射到 TotalLevels=2")

	// 审批记录应携带 node（审批链求值结果）
	_, approvals, err := svc.Get(ctx, created.ID, tenantID)
	require.NoError(t, err)
	require.Len(t, approvals, 2)
	require.NotNil(t, approvals[0].Node, "首级审批应写入 chain node")
	require.NotEmpty(t, approvals[0].Node["approver_ids"], "首级 node 应含解析出的审批人")

	before := approvalWriteSnapshot(t, client)
	for attempt := 0; attempt < 2; attempt++ {
		_, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, "approve", "ok", "manager", "IT")
		var businessErr *common.BusinessError
		require.ErrorAs(t, err, &businessErr)
		require.Equal(t, common.ConflictCode, businessErr.Code)
		require.Equal(t, before, approvalWriteSnapshot(t, client))
	}
	instanceID := srCreateSerialBPMNFixture(t, client, tenantID, created.ID, mgr, it)

	// L1 manager 审批 -> manager_approved, currentLevel=2
	req, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, "approve", "ok", "manager", "IT")
	require.NoError(t, err)
	require.Equal(t, SRStatusManagerApproved, req.Status)
	require.Equal(t, 2, req.CurrentLevel)

	// L2 agent 审批（末级）-> security_approved（履约门禁），currentLevel=3
	req, _, err = svc.ApplyApproval(ctx, created.ID, tenantID, it, "approve", "ok", "agent", "IT")
	require.NoError(t, err)
	require.Equal(t, SRStatusSecurityApproved, req.Status, "末级审批应落到 security_approved（履约门禁）")
	require.Equal(t, 2, req.CurrentLevel, "末级后 CurrentLevel 应等于 TotalLevels(2)")
	instance, err := client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	require.Equal(t, "completed", instance.Status)
	decisions, err := client.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, decisions)
}

// Retirement contract: the old parallel quorum evaluator must not collect votes
// or approve a request without BPMN tasks, even after all configured users retry.
// This is not a test of BPMN countersign voting (a separate execution path).
func TestServiceRequest_ApprovalChain_ParallelQuorumRequiresBPMN(t *testing.T) {
	svc, client, tenantID, catalogID := setupApprovalChainTest(t)
	ctx := context.Background()

	// 单级会签：role=manager 解析出 2 名 manager，阈值默认=2
	chainReq := &dto.ApprovalChainRequest{
		Name:       "Quorum Chain",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "Mgr", IsRequired: true, ApprovalType: "parallel"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	mgrA := mkChainUser(t, client, tenantID, "manager", "IT")
	mgrB := mkChainUser(t, client, tenantID, "manager", "IT")
	requester := mkChainUser(t, client, tenantID, "end_user", "IT")

	created, err := svc.Create(ctx, tenantID, requester, catalogID, chainReqData("Quorum SR"))
	require.NoError(t, err)
	require.Equal(t, 1, created.TotalLevels)

	before := approvalWriteSnapshot(t, client)
	for _, actorID := range []int{mgrA, mgrB, mgrA, mgrB} {
		r := gin.New()
		r.Use(_srAuthRole(tenantID, actorID, "manager", "IT"))
		r.POST("/api/v1/service-requests/:id/approval", NewHandler(svc).ApplyApproval)
		status, resp := srDoHTTPReq(t, r, http.MethodPost, fmt.Sprintf("/api/v1/service-requests/%d/approval", created.ID),
			dto.ServiceRequestApprovalActionRequest{Action: "approve", Comment: "must not collect a legacy vote"})
		require.Equal(t, http.StatusConflict, status, "body=%s", srStr(resp))
		require.Equal(t, common.ConflictCode, resp.Code)
		require.Nil(t, resp.Data)
		require.Equal(t, before, approvalWriteSnapshot(t, client))
	}
}

// L1 审批成功后再次尝试 L1 必须被拒绝，且不得产生任何写入。
func TestServiceRequest_ApprovalChain_DuplicateApprovalRejected(t *testing.T) {
	svc, client, tenantID, catalogID := setupApprovalChainTest(t)
	ctx := context.Background()

	chainReq := &dto.ApprovalChainRequest{
		Name:       "SR Dup Chain",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "Mgr", IsRequired: true, ApprovalType: "serial"},
			{Level: 2, Role: "agent", Name: "IT", IsRequired: true, ApprovalType: "serial"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	mgr := mkChainUser(t, client, tenantID, "manager", "IT")
	it := mkChainUser(t, client, tenantID, "agent", "IT")
	requester := mkChainUser(t, client, tenantID, "end_user", "IT")

	created, err := svc.Create(ctx, tenantID, requester, catalogID, chainReqData("Dup SR"))
	require.NoError(t, err)
	require.Equal(t, 2, created.TotalLevels)

	srCreateSerialBPMNFixture(t, client, tenantID, created.ID, mgr, it)

	req, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, "approve", "ok", "manager", "IT")
	require.NoError(t, err)
	require.Equal(t, SRStatusManagerApproved, req.Status)
	require.Equal(t, 2, req.CurrentLevel)

	snapAfterL1 := approvalWriteSnapshot(t, client)

	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			_, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, action, "重复", "manager", "IT")
			require.Error(t, err, "L1 已审批不得再次审批")
			require.Equal(t, snapAfterL1, approvalWriteSnapshot(t, client))
		})
	}
}

// 业务更新失败时 BPMN 任务必须回滚，防止"流程已批、业务未批"的不一致。
func TestServiceRequest_ApprovalChain_AtomicCommit_RollbackOnBusinessFailure(t *testing.T) {
	svc, client, tenantID, catalogID := setupApprovalChainTest(t)
	ctx := context.Background()

	chainReq := &dto.ApprovalChainRequest{
		Name:       "SR Rollback Chain",
		EntityType: "service_request",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "Mgr", IsRequired: true, ApprovalType: "serial"},
		},
	}
	acs := service.NewApprovalChainService(client, zaptest.NewLogger(t).Sugar())
	_, err := acs.CreateApprovalChain(ctx, chainReq, tenantID)
	require.NoError(t, err)

	mgr := mkChainUser(t, client, tenantID, "manager", "IT")
	requester := mkChainUser(t, client, tenantID, "end_user", "IT")

	created, err := svc.Create(ctx, tenantID, requester, catalogID, chainReqData("Rollback SR"))
	require.NoError(t, err)
	require.Equal(t, 1, created.TotalLevels)

	_ = srCreateSerialBPMNFixture(t, client, tenantID, created.ID, mgr, mgr)

	taskBefore, err := client.ProcessTask.Query().First(ctx)
	require.NoError(t, err)
	require.Equal(t, "assigned", taskBefore.Status)

	failServiceRequestUpdates := true
	client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			if _, ok := mutation.(*ent.ServiceRequestMutation); ok && failServiceRequestUpdates {
				return nil, fmt.Errorf("simulated service request update failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})

	req, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, "approve", "ok", "manager", "IT")
	require.Error(t, err, "业务更新失败时审批必须失败")
	require.Nil(t, req)

	sr, _, gerr := svc.Get(ctx, created.ID, tenantID)
	require.NoError(t, gerr)
	require.NotNil(t, sr)
	require.NotEqual(t, SRStatusManagerApproved, sr.Status, "服务请求状态不得变更")

	taskAfter, terr := client.ProcessTask.Get(ctx, taskBefore.ID)
	require.NoError(t, terr)
	require.Equal(t, "assigned", taskAfter.Status, "BPMN 任务必须回滚到原始状态")

	failServiceRequestUpdates = false
	req2, _, err2 := svc.ApplyApproval(ctx, created.ID, tenantID, mgr, "approve", "重试", "manager", "IT")
	require.NoError(t, err2)
	require.NotNil(t, req2)

	taskRetry, terr2 := client.ProcessTask.Get(ctx, taskBefore.ID)
	require.NoError(t, terr2)
	require.Equal(t, "completed", taskRetry.Status, "重试后 BPMN 任务必须完成")
}
