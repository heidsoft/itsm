package service_request

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

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

// TestServiceRequest_ApprovalChain_SerialProgression 验证：租户级激活审批链被
// ResolveApprovalPlan 解析并驱动分级审批（manager -> agent），层级推进与遗留三级一致。
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
}

// TestServiceRequest_ApprovalChain_ParallelQuorum 验证：会签层级（阈值=审批人数）
// 需全员批准才推进；单人批准仅记录进度，不推进请求状态。
func TestServiceRequest_ApprovalChain_ParallelQuorum(t *testing.T) {
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

	// mgrA 批准：quorum 未满足 -> 仍为 submitted，不推进
	req, _, err := svc.ApplyApproval(ctx, created.ID, tenantID, mgrA, "approve", "a", "manager", "IT")
	require.NoError(t, err)
	require.Equal(t, SRStatusSubmitted, req.Status)
	require.Equal(t, 1, req.CurrentLevel)

	// mgrB 批准：quorum 满足（末级）-> security_approved
	req, _, err = svc.ApplyApproval(ctx, created.ID, tenantID, mgrB, "approve", "b", "manager", "IT")
	require.NoError(t, err)
	require.Equal(t, SRStatusSecurityApproved, req.Status)
	require.Equal(t, 1, req.CurrentLevel, "末级后 CurrentLevel 应等于 TotalLevels(1)")
}
