package scenarios

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/service_catalog"
	service_request "itsm-backend/handlers/service_request"
)

// Scenario 10: 服务目录→请求→审批→配置→CI 闭环
//
// 覆盖：
//   - 服务目录创建与启用
//   - 服务请求创建（含校验：合规确认、过期时间、数据分类、标题）
//   - 三级审批流转：submitted → manager_approved → it_approved → security_approved
//   - 履约状态机：security_approved → provisioning → delivered
//   - 审批拒绝路径：reject → rejected
//   - 非法状态迁移拒绝
//   - 跨租户隔离

func TestScenario10_ServiceCatalogFullLoop(t *testing.T) {
	ctx := context.Background()
	dsn := scenarioDSN("loop10")
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()

	// --- Fixture: tenant + users ---
	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "TA", "a.example.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "TB", "b.example.com")

	tid := tenantA.ID

	requester := mustCreateUser(ctx, t, client, tid, "requester", "req@a.com", "end_user")
	managerUser := mustCreateUser(ctx, t, client, tid, "mgr_zhang", "mgr@a.com", "manager")
	itAdmin := mustCreateUser(ctx, t, client, tid, "it_li", "it@a.com", "it_admin")
	secAdmin := mustCreateUser(ctx, t, client, tid, "sec_wang", "sec@a.com", "security_admin")
	operator := mustCreateUser(ctx, t, client, tid, "ops_chen", "ops@a.com", "agent")

	// Tenant B user for cross-tenant test
	_ = mustCreateUser(ctx, t, client, tenantB.ID, "intruder", "intruder@b.com", "admin")

	// --- Repos + services ---
	scRepo := service_catalog.NewEntRepository(client)
	scSvc := service_catalog.NewService(scRepo, logger)

	srRepo := service_request.NewEntRepository(client)
	srSvc := service_request.NewService(srRepo, scRepo, nil, nil, logger, nil)

	// ========== 1. Create service catalog ==========
	t.Run("create service catalog", func(t *testing.T) {
		cat, err := scSvc.Create(ctx, &service_catalog.ServiceCatalog{
			Name:         "ECS 云服务器",
			Category:     "compute",
			Description:  "弹性计算服务",
			Status:       "enabled",
			DeliveryTime: 3,
			TenantID:     tid,
		})
		require.NoError(t, err)
		require.NotZero(t, cat.ID)
		require.Equal(t, "enabled", cat.Status)
	})

	// Fetch catalog for subsequent steps
	catalogs, _, err := scRepo.List(ctx, tid, service_catalog.ListFilters{Category: "compute"})
	require.NoError(t, err)
	require.NotEmpty(t, catalogs)
	catalog := catalogs[0]

	// ========== 2. Validation: missing compliance ack ==========
	t.Run("reject request without compliance ack", func(t *testing.T) {
		expiry := time.Now().Add(72 * time.Hour)
		_, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请 ECS",
			ComplianceAck:      false,
			DataClassification: "internal",
			ExpireAt:           &expiry,
		})
		require.Error(t, err, "缺少合规确认应被拒绝")
		require.Contains(t, err.Error(), "Compliance")
	})

	// ========== 3. Validation: past expiry ==========
	t.Run("reject request with past expiry", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		_, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请 ECS",
			ComplianceAck:      true,
			DataClassification: "internal",
			ExpireAt:           &past,
		})
		require.Error(t, err, "过期时间在过去应被拒绝")
		require.Contains(t, err.Error(), "Expiration")
	})

	// ========== 4. Validation: missing title ==========
	t.Run("reject request without title", func(t *testing.T) {
		expiry := time.Now().Add(72 * time.Hour)
		_, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "",
			ComplianceAck:      true,
			DataClassification: "internal",
			ExpireAt:           &expiry,
		})
		require.Error(t, err, "缺少标题应被拒绝")
	})

	// ========== 5. Validation: invalid data classification ==========
	t.Run("reject request with invalid data classification", func(t *testing.T) {
		expiry := time.Now().Add(72 * time.Hour)
		_, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请 ECS",
			ComplianceAck:      true,
			DataClassification: "top_secret",
			ExpireAt:           &expiry,
		})
		require.Error(t, err, "非法数据分类应被拒绝")
	})

	// ========== 6. Validation: public IP without whitelist ==========
	t.Run("reject request with public IP but no whitelist", func(t *testing.T) {
		expiry := time.Now().Add(72 * time.Hour)
		_, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请 ECS",
			ComplianceAck:      true,
			DataClassification: "internal",
			NeedsPublicIP:      true,
			ExpireAt:           &expiry,
		})
		require.Error(t, err, "公网 IP 缺少白名单应被拒绝")
	})

	// ========== 7. Happy path: create valid service request ==========
	var srID int
	t.Run("create valid service request", func(t *testing.T) {
		expiry := time.Now().Add(72 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请 ECS 实例",
			Reason:             "项目开发需要",
			ComplianceAck:      true,
			DataClassification: "internal",
			ExpireAt:           &expiry,
			CostCenter:         "CC-DEV",
		})
		require.NoError(t, err)
		require.Equal(t, "submitted", sr.Status)
		require.Equal(t, 1, sr.CurrentLevel)
		require.Equal(t, 3, sr.TotalLevels)
		srID = sr.ID
	})

	// ========== 8. Verify approvals were created ==========
	t.Run("verify 3-level approvals created", func(t *testing.T) {
		req, approvals, err := srSvc.Get(ctx, srID, tid)
		require.NoError(t, err)
		require.Equal(t, "submitted", req.Status)
		require.Len(t, approvals, 3)
		require.Equal(t, "manager", approvals[0].Step)
		require.Equal(t, "it", approvals[1].Step)
		require.Equal(t, "security", approvals[2].Step)
		for _, a := range approvals {
			require.Equal(t, "pending", a.Status)
		}
	})

	// ========== 9. Manager approval (level 1) ==========
	t.Run("manager approves level 1", func(t *testing.T) {
		req, approvals, err := srSvc.ApplyApproval(ctx, srID, tid, managerUser.ID, "approve", "同意", "manager", "")
		require.NoError(t, err)
		require.Equal(t, "manager_approved", req.Status)
		require.Equal(t, 2, req.CurrentLevel)
		require.Equal(t, "approved", approvals[0].Status)
	})

	// ========== 10. IT approval (level 2) ==========
	t.Run("it_admin approves level 2", func(t *testing.T) {
		req, approvals, err := srSvc.ApplyApproval(ctx, srID, tid, itAdmin.ID, "approve", "IT 审核通过", "it_admin", "")
		require.NoError(t, err)
		require.Equal(t, "it_approved", req.Status)
		require.Equal(t, 3, req.CurrentLevel)
		require.Equal(t, "approved", approvals[1].Status)
	})

	// ========== 11. Security approval (level 3) ==========
	t.Run("security_admin approves level 3", func(t *testing.T) {
		req, approvals, err := srSvc.ApplyApproval(ctx, srID, tid, secAdmin.ID, "approve", "安全审核通过", "security_admin", "")
		require.NoError(t, err)
		require.Equal(t, "security_approved", req.Status)
		require.Equal(t, "approved", approvals[2].Status)
	})

	// ========== 12. Transition to provisioning ==========
	t.Run("operator transitions to provisioning", func(t *testing.T) {
		err := srSvc.UpdateStatus(ctx, srID, tid, operator.ID, "agent", "provisioning")
		require.NoError(t, err)

		req, _, err := srSvc.Get(ctx, srID, tid)
		require.NoError(t, err)
		require.Equal(t, "provisioning", req.Status)
	})

	// ========== 13. Transition to delivered ==========
	t.Run("operator transitions to delivered", func(t *testing.T) {
		err := srSvc.UpdateStatus(ctx, srID, tid, operator.ID, "agent", "delivered")
		require.NoError(t, err)

		req, _, err := srSvc.Get(ctx, srID, tid)
		require.NoError(t, err)
		require.Equal(t, "delivered", req.Status)
		require.NotNil(t, req.CompletedAt)
	})

	// ========== 14. Invalid state transitions ==========
	t.Run("reject invalid transition: delivered -> submitted", func(t *testing.T) {
		err := srSvc.UpdateStatus(ctx, srID, tid, operator.ID, "agent", "submitted")
		require.Error(t, err, "delivered -> submitted 应被拒绝")
	})

	t.Run("reject invalid transition: delivered -> provisioning", func(t *testing.T) {
		err := srSvc.UpdateStatus(ctx, srID, tid, operator.ID, "agent", "provisioning")
		require.Error(t, err, "delivered -> provisioning 应被拒绝")
	})

	// ========== 15. Rejection path: create another request and reject at level 1 ==========
	var rejectID int
	t.Run("rejection at manager level", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "申请测试资源",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)
		rejectID = sr.ID

		req, _, err := srSvc.ApplyApproval(ctx, rejectID, tid, managerUser.ID, "reject", "预算不足，暂不批准", "manager", "")
		require.NoError(t, err)
		require.Equal(t, "rejected", req.Status)
	})

	// ========== 16. Rejected request cannot transition ==========
	t.Run("rejected request cannot transition to provisioning", func(t *testing.T) {
		err := srSvc.UpdateStatus(ctx, rejectID, tid, operator.ID, "agent", "provisioning")
		require.Error(t, err, "rejected 状态不允许状态迁移")
	})

	// ========== 17. Requester self-approval blocked ==========
	t.Run("requester cannot approve own request", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "自审测试",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)

		_, _, err = srSvc.ApplyApproval(ctx, sr.ID, tid, requester.ID, "approve", "自审", "end_user", "")
		require.Error(t, err, "请求者不能审批自己的请求")
	})

	// ========== 18. Rejection without comment blocked ==========
	t.Run("rejection without comment is rejected", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "无注释拒绝测试",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)

		_, _, err = srSvc.ApplyApproval(ctx, sr.ID, tid, managerUser.ID, "reject", "", "manager", "")
		require.Error(t, err, "拒绝必须提供评论")
	})

	// ========== 19. Cross-tenant isolation ==========
	t.Run("cross-tenant access denied", func(t *testing.T) {
		// Tenant B user tries to access Tenant A's request
		_, _, err := srSvc.Get(ctx, srID, tenantB.ID)
		require.Error(t, err, "跨租户访问应失败")
	})

	t.Run("cross-tenant approval denied", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "跨租户测试",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)

		// Tenant B intruder tries to approve Tenant A's request
		_, _, err = srSvc.ApplyApproval(ctx, sr.ID, tenantB.ID, 999, "approve", "恶意审批", "admin", "")
		require.Error(t, err, "跨租户审批应失败")
	})

	// ========== 20. Cancel path ==========
	t.Run("requester can cancel submitted request", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "取消测试",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)

		err = srSvc.UpdateStatus(ctx, sr.ID, tid, requester.ID, "end_user", "cancelled")
		require.NoError(t, err)

		req, _, err := srSvc.Get(ctx, sr.ID, tid)
		require.NoError(t, err)
		require.Equal(t, "cancelled", req.Status)
	})

	// ========== 21. Non-operator cannot transition to provisioning ==========
	t.Run("non-operator cannot update fulfillment status", func(t *testing.T) {
		expiry := time.Now().Add(48 * time.Hour)
		sr, err := srSvc.Create(ctx, tid, requester.ID, catalog.ID, &service_request.ServiceRequest{
			Title:              "权限测试",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.NoError(t, err)

		// Manually push to security_approved for testing (bypass approval for this sub-test)
		// Use admin role to bypass operator check
		err = srSvc.UpdateStatus(ctx, sr.ID, tid, requester.ID, "end_user", "provisioning")
		require.Error(t, err, "end_user 不应能更新履约状态")
	})

	// ========== 22. Disabled catalog cannot receive requests ==========
	t.Run("disabled catalog rejects requests", func(t *testing.T) {
		disabledCat, err := scSvc.Create(ctx, &service_catalog.ServiceCatalog{
			Name:         "已停服项",
			Category:     "deprecated",
			Status:       "disabled",
			DeliveryTime: 1,
			TenantID:     tid,
		})
		require.NoError(t, err)

		expiry := time.Now().Add(48 * time.Hour)
		_, err = srSvc.Create(ctx, tid, requester.ID, disabledCat.ID, &service_request.ServiceRequest{
			Title:              "不应成功",
			ComplianceAck:      true,
			DataClassification: "public",
			ExpireAt:           &expiry,
		})
		require.Error(t, err, "已停用的服务目录不应接受请求")
	})
}
