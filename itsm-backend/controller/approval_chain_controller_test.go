package controller

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupApprovalChainController(t *testing.T) (*gin.Engine, *ent.Client, int, int) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "approval_chain_test.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	tenantID, userID := seedTenantUser(t, client)

	svc := service.NewApprovalChainService(client, logger)
	ctrl := NewApprovalChainController(svc, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(withTestAuth(tenantID, userID))
	r.POST("/api/v1/approval-chains", ctrl.CreateChain)
	r.GET("/api/v1/approval-chains", ctrl.ListChains)
	r.GET("/api/v1/approval-chains/stats", ctrl.GetStats)
	r.GET("/api/v1/approval-chains/:id", ctrl.GetChain)
	r.PUT("/api/v1/approval-chains/:id", ctrl.UpdateChain)
	r.DELETE("/api/v1/approval-chains/:id", ctrl.DeleteChain)
	return r, client, tenantID, userID
}

func TestApprovalChainController_CreateChain(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	t.Run("成功创建审批链", func(t *testing.T) {
		body := dto.ApprovalChainRequest{
			Name:       "变更审批链",
			EntityType: "change",
			Chain: []dto.ApprovalChainStepDTO{
				{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true},
				{Level: 2, Role: "director", Name: "总监审批", IsRequired: true},
			},
		}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "变更审批链", data["name"])
		assert.Equal(t, float64(2), float64(len(data["chain"].([]interface{}))))
	})

	t.Run("缺少名称应返回参数错误", func(t *testing.T) {
		body := dto.ApprovalChainRequest{EntityType: "change", Chain: []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}}}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("缺少租户上下文应返回未授权", func(t *testing.T) {
		body := dto.ApprovalChainRequest{Name: "x", EntityType: "change", Chain: []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}}}
		resp := doReq(t, r, "POST", "/api/v1/approval-chains", body, true)
		assert.Equal(t, common.UnauthorizedCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestApprovalChainController_ListAndStats(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	t.Run("列表返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("统计返回成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/stats", nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})
}

func TestApprovalChainController_GetChain(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	created := doReq(t, r, "POST", "/api/v1/approval-chains", dto.ApprovalChainRequest{
		Name:       "查询测试链",
		EntityType: "change",
		Chain:      []dto.ApprovalChainStepDTO{{Level: 1, Role: "x", Name: "n"}},
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	t.Run("按ID获取成功", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("非法ID应返回参数错误", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/abc", nil, false)
		assert.Equal(t, common.ParamErrorCode, resp.Code, "body=%s", mustString(resp))
	})

	t.Run("不存在的ID应返回未找到", func(t *testing.T) {
		resp := doReq(t, r, "GET", "/api/v1/approval-chains/999999", nil, false)
		assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
	})
}

// TestApprovalChainController_MultiLevelChain 3 级链式锁链：
// 验证创建时序（经理 → 总监 → VP）能正确持久化为 3 个 step，且
// level/role/IsRequired 在 readback 后保持一致。
func TestApprovalChainController_MultiLevelChain(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	body := dto.ApprovalChainRequest{
		Name:       "3级变更审批链",
		EntityType: "change",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true, ApproverID: 1001},
			{Level: 2, Role: "director", Name: "总监审批", IsRequired: true, ApproverID: 1002},
			{Level: 3, Role: "vp", Name: "VP审批", IsRequired: true, ApproverID: 1003},
		},
	}
	created := doReq(t, r, "POST", "/api/v1/approval-chains", body, false)
	require.Equal(t, common.SuccessCode, created.Code, "body=%s", mustString(created))
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	// Readback 必须返回 3 个 step 且 level 与 role 严格一致。
	got := doReq(t, r, "GET", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, got.Code, "body=%s", mustString(got))
	data := got.Data.(map[string]interface{})
	chain := data["chain"].([]interface{})
	require.Len(t, chain, 3, "3 级链应返回 3 个 step")

	wantLevels := []float64{1, 2, 3}
	wantRoles := []string{"manager", "director", "vp"}
	for i, step := range chain {
		m := step.(map[string]interface{})
		assert.Equal(t, wantLevels[i], m["level"], "step %d level 应为 %v", i, wantLevels[i])
		assert.Equal(t, wantRoles[i], m["role"], "step %d role 应为 %s", i, wantRoles[i])
		assert.Equal(t, true, m["isRequired"], "step %d IsRequired 应为 true", i)
	}
}

// TestApprovalChainController_Countersign_ReassignApprover 加签：
// 通过 PUT 重新指定某 step 的 ApproverID，模拟加签行为。
// readback 后被加签人（approverId）必须持久化可见。
func TestApprovalChainController_Countersign_ReassignApprover(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	created := doReq(t, r, "POST", "/api/v1/approval-chains", dto.ApprovalChainRequest{
		Name:       "加签测试链",
		EntityType: "change",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true, ApproverID: 1001},
			{Level: 2, Role: "director", Name: "总监审批", IsRequired: true, ApproverID: 1002},
		},
	}, false)
	require.Equal(t, common.SuccessCode, created.Code, "body=%s", mustString(created))
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	// 把 level=1 的 ApproverID 改成 9999（加签给其他人）。
	updated := doReq(t, r, "PUT", "/api/v1/approval-chains/"+strconv.Itoa(id), dto.ApprovalChainRequest{
		Name:        "加签测试链",
		Description: "已加签给 L2",
		EntityType:  "change",
		Status:      "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true, ApproverID: 9999},
			{Level: 2, Role: "director", Name: "总监审批", IsRequired: true, ApproverID: 1002},
		},
	}, false)
	require.Equal(t, common.SuccessCode, updated.Code, "body=%s", mustString(updated))

	got := doReq(t, r, "GET", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, got.Code)
	chain := got.Data.(map[string]interface{})["chain"].([]interface{})
	step0 := chain[0].(map[string]interface{})
	assert.Equal(t, float64(9999), step0["approverId"],
		"加签后 level=1 ApproverID 必须为 9999，原 1001 被覆盖")
}

// TestApprovalChainController_Withdraw_SetInactive 撤回：
// 把 chain 标记为 inactive（draft 或 withdrawn 之外的兼容态）。
// 这里直接使用 update 把 Status 改为 inactive，模拟"发起人 24h 窗口撤回"。
// 验证：readback 后 status 必须为 inactive。
func TestApprovalChainController_Withdraw_SetInactive(t *testing.T) {
	r, _, _, _ := setupApprovalChainController(t)

	created := doReq(t, r, "POST", "/api/v1/approval-chains", dto.ApprovalChainRequest{
		Name:       "撤回测试链",
		EntityType: "change",
		Status:     "active",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true},
		},
	}, false)
	require.Equal(t, common.SuccessCode, created.Code)
	id := int(created.Data.(map[string]interface{})["id"].(float64))

	withdrawn := doReq(t, r, "PUT", "/api/v1/approval-chains/"+strconv.Itoa(id), dto.ApprovalChainRequest{
		Name:       "撤回测试链",
		EntityType: "change",
		Status:     "inactive",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true},
		},
	}, false)
	require.Equal(t, common.SuccessCode, withdrawn.Code, "body=%s", mustString(withdrawn))

	got := doReq(t, r, "GET", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
	require.Equal(t, common.SuccessCode, got.Code)
	assert.Equal(t, "inactive", got.Data.(map[string]interface{})["status"],
		"撤回后 status 必须为 inactive")

	// 同时验证 Delete 在 inactive 状态下仍可正常进行
	deleted := doReq(t, r, "DELETE", "/api/v1/approval-chains/"+strconv.Itoa(id), nil, false)
	assert.Equal(t, common.SuccessCode, deleted.Code, "body=%s", mustString(deleted))
}

// TestApprovalChainController_CrossTenantIsolation 越权：
// 跨租户不能 Get/Update/Delete chain 配置 → 必须返回 NotFound/InternalError。
//
// 创建一个 tenant A 的链，然后用 tenant B 的 auth context 访问，
// 不应看到任何 tenant A 的资源。
func TestApprovalChainController_CrossTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + filepath.Join(t.TempDir(), "approval_chain_iso.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	tenantAID, userAID := seedTenantUser(t, client)

	// 第二个 tenant 共享同一 client；seedTenantUser 用 uniqueTestID，
	// 直接复用会冲突，所以手工创一个 tenantB。
	uidB := uniqueTestID()
	tenantB, err := client.Tenant.Create().
		SetName("TenantB").
		SetCode("TB" + uidB).
		SetDomain("tenant-b.com").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	userB, err := client.User.Create().
		SetUsername("tenant-b-user" + uidB).
		SetEmail("tenant-b" + uidB + "@example.com").
		SetPasswordHash("hashed").
		SetName("Tenant B User").
		SetActive(true).
		SetRole("admin").
		SetTenantID(tenantB.ID).
		Save(context.Background())
	require.NoError(t, err)

	svc := service.NewApprovalChainService(client, logger)
	ctrl := NewApprovalChainController(svc, logger)

	rA := gin.New()
	rA.Use(gin.Recovery())
	rA.Use(withTestAuth(tenantAID, userAID))
	rA.POST("/api/v1/approval-chains", ctrl.CreateChain)
	rA.GET("/api/v1/approval-chains/:id", ctrl.GetChain)
	rA.PUT("/api/v1/approval-chains/:id", ctrl.UpdateChain)
	rA.DELETE("/api/v1/approval-chains/:id", ctrl.DeleteChain)

	rB := gin.New()
	rB.Use(gin.Recovery())
	rB.Use(withTestAuth(tenantB.ID, userB.ID))
	rB.POST("/api/v1/approval-chains", ctrl.CreateChain)
	rB.GET("/api/v1/approval-chains", ctrl.ListChains)
	rB.GET("/api/v1/approval-chains/:id", ctrl.GetChain)
	rB.PUT("/api/v1/approval-chains/:id", ctrl.UpdateChain)
	rB.DELETE("/api/v1/approval-chains/:id", ctrl.DeleteChain)

	// 1. tenant A 创建一条链
	created := doReq(t, rA, "POST", "/api/v1/approval-chains", dto.ApprovalChainRequest{
		Name:       "tenant A 私有链",
		EntityType: "change",
		Chain: []dto.ApprovalChainStepDTO{
			{Level: 1, Role: "manager", Name: "经理审批", IsRequired: true},
		},
	}, false)
	require.Equal(t, common.SuccessCode, created.Code, "body=%s", mustString(created))
	idA := int(created.Data.(map[string]interface{})["id"].(float64))

	// 2. tenant B GET 该 id 必须返回 NotFoundCode（不能 leak 存在性）
	t.Run("跨租户GET返回404", func(t *testing.T) {
		resp := doReq(t, rB, "GET", "/api/v1/approval-chains/"+strconv.Itoa(idA), nil, false)
		assert.Equal(t, common.NotFoundCode, resp.Code, "body=%s", mustString(resp))
	})

	// 3. tenant B UPDATE 该 id 必须失败（非 SuccessCode）
	t.Run("跨租户UPDATE失败", func(t *testing.T) {
		resp := doReq(t, rB, "PUT", "/api/v1/approval-chains/"+strconv.Itoa(idA), dto.ApprovalChainRequest{
			Name:       "越权篡改",
			EntityType: "change",
			Chain: []dto.ApprovalChainStepDTO{
				{Level: 1, Role: "vp", Name: "VP审批", IsRequired: true},
			},
		}, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code,
			"跨租户 UPDATE 必须不返回 SuccessCode；body=%s", mustString(resp))
		// 业务上更期望 4xx（404/403），但当前 controller 在 service 错误上
		// 返回 5001（InternalErrorCode）。这里只断言非 Success 即可。
	})

	// 4. tenant B DELETE 该 id 必须失败
	t.Run("跨租户DELETE失败", func(t *testing.T) {
		resp := doReq(t, rB, "DELETE", "/api/v1/approval-chains/"+strconv.Itoa(idA), nil, false)
		assert.NotEqual(t, common.SuccessCode, resp.Code,
			"跨租户 DELETE 必须不返回 SuccessCode；body=%s", mustString(resp))
	})

	// 5. tenant B LIST 不能看到 tenant A 的链
	t.Run("跨租户LIST不可见", func(t *testing.T) {
		resp := doReq(t, rB, "GET", "/api/v1/approval-chains", nil, false)
		require.Equal(t, common.SuccessCode, resp.Code)
		data := resp.Data.(map[string]interface{})
		list := data["items"].([]interface{})
		for _, item := range list {
			m := item.(map[string]interface{})
			if id, ok := m["id"].(float64); ok && int(id) == idA {
				t.Fatalf("tenant B 不应看到 tenant A 的链 id=%d", idA)
			}
		}
	})

	// 6. tenant A 自己仍能正常读到自己刚创的链（确认 tenant A 没被影响）
	t.Run("本租户读自己仍然成功", func(t *testing.T) {
		resp := doReq(t, rA, "GET", "/api/v1/approval-chains/"+strconv.Itoa(idA), nil, false)
		assert.Equal(t, common.SuccessCode, resp.Code, "body=%s", mustString(resp))
	})
}
