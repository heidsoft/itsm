package service

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"

	_ "github.com/mattn/go-sqlite3"
)

// ---------------------------------------------------------------------------
// P2-10 守卫：UpdateBinding/CreateBinding 必须 fail-closed 且校验关联引用归属。
//
// 旧实现两个缺口：
//   - `binding.TenantID > 0 && entity.TenantID != binding.TenantID` 让缺省
//     调用（<=0）整段跳过归属比较；
//   - department/team/approval-chain/SLA 外键原样写入，不验证存在性与租户归属。
//
// 关联外键被路由与审计直接消费，指向不存在或他租户资源的绑定是静默错配。
// ---------------------------------------------------------------------------

type bindingGuardFixture struct {
	client  *ent.Client
	ctx     context.Context
	tenantA int
	tenantB int

	targetID int // tenantA 内待更新的绑定

	deptA, deptB   int
	teamA, teamB   int
	chainA, chainB int
	slaA, slaB     int
}

func newBindingGuardFixture(t *testing.T) *bindingGuardFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:bpmn_binding_guard?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx, tenantA := bizTenantCtx(t, client, "bindA")
	tenantB := createTimerTenant(t, client, "bindB")

	f := &bindingGuardFixture{client: client, ctx: ctx, tenantA: tenantA, tenantB: tenantB}

	mkDept := func(tenantID int, code string) int {
		d, err := client.Department.Create().SetName(code).SetCode(code).SetTenantID(tenantID).Save(ctx)
		require.NoError(t, err)
		return d.ID
	}
	f.deptA = mkDept(tenantA, "bind-dept-a")
	f.deptB = mkDept(tenantB, "bind-dept-b")

	mkTeam := func(tenantID int, code string) int {
		tm, err := client.Team.Create().SetName(code).SetCode(code).SetTenantID(tenantID).Save(ctx)
		require.NoError(t, err)
		return tm.ID
	}
	f.teamA = mkTeam(tenantA, "bind-team-a")
	f.teamB = mkTeam(tenantB, "bind-team-b")

	mkChain := func(tenantID int, name string) int {
		c, err := client.ApprovalChain.Create().
			SetName(name).SetEntityType("ticket").SetStatus("active").
			SetChain([]schema.ApprovalChainStep{}).SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return c.ID
	}
	f.chainA = mkChain(tenantA, "bind-chain-a")
	f.chainB = mkChain(tenantB, "bind-chain-b")

	mkSLA := func(tenantID int, name string) int {
		s, err := client.SLAPolicy.Create().
			SetName(name).SetResponseTimeMinutes(60).SetResolutionTimeMinutes(240).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		return s.ID
	}
	f.slaA = mkSLA(tenantA, "bind-sla-a")
	f.slaB = mkSLA(tenantB, "bind-sla-b")

	// UpdateBinding 目标：tenantA 内一条普通绑定。
	target, err := client.ProcessBinding.Create().
		SetBusinessType("ticket").
		SetProcessDefinitionKey("ticket_general_flow").
		SetTenantID(tenantA).
		Save(ctx)
	require.NoError(t, err)
	f.targetID = target.ID

	return f
}

func (f *bindingGuardFixture) update(binding *dto.ProcessBinding) (*dto.ProcessBinding, error) {
	svc := &ProcessBindingService{client: f.client}
	return svc.UpdateBinding(f.ctx, f.targetID, binding)
}

// baseUpdate 返回只带合法租户的最小更新请求（其余字段全零=不引用关联）。
func (f *bindingGuardFixture) baseUpdate() *dto.ProcessBinding {
	return &dto.ProcessBinding{TenantID: f.tenantA}
}

func requireAppErrorStatus(t *testing.T, err error, wantStatus int, msgContains string) {
	t.Helper()
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, wantStatus, appErr.HTTPStatus)
	assert.Contains(t, appErr.Message, msgContains)
}

func requireAuthFailedBusinessError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.AuthFailedCode, bizErr.Code)
	assert.Contains(t, bizErr.Message, "缺少有效租户上下文")
}

func TestUpdateBindingMissingTenantContextFailsClosed(t *testing.T) {
	f := newBindingGuardFixture(t)

	// 缺省租户（0）旧实现整段跳过归属比较——必须拒绝而不是放行。
	req := f.baseUpdate()
	req.TenantID = 0
	_, err := f.update(req)
	requireAuthFailedBusinessError(t, err)
}

func TestUpdateBindingCrossTenantRejected(t *testing.T) {
	f := newBindingGuardFixture(t)

	req := f.baseUpdate()
	req.TenantID = f.tenantB
	_, err := f.update(req)
	requireAppErrorStatus(t, err, http.StatusForbidden, "无权操作此流程绑定")
}

func TestUpdateBindingRejectsForeignAndMissingReferences(t *testing.T) {
	f := newBindingGuardFixture(t)

	cases := []struct {
		name        string
		mutate      func(*dto.ProcessBinding)
		msgContains string
	}{
		{"不存在的部门", func(b *dto.ProcessBinding) { b.DepartmentID = 999999 }, "部门"},
		{"他租户的部门", func(b *dto.ProcessBinding) { b.DepartmentID = f.deptB }, "部门"},
		{"他租户的团队", func(b *dto.ProcessBinding) { b.TeamID = f.teamB }, "团队"},
		{"非数字审批链引用", func(b *dto.ProcessBinding) { b.ApprovalChainID = "chain-abc" }, "审批链"},
		{"他租户的审批链", func(b *dto.ProcessBinding) { b.ApprovalChainID = strconv.Itoa(f.chainB) }, "审批链"},
		{"他租户的SLA策略", func(b *dto.ProcessBinding) { b.SLAPolicyID = strconv.Itoa(f.slaB) }, "SLA 策略"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := f.baseUpdate()
			tc.mutate(req)
			_, err := f.update(req)
			requireAppErrorStatus(t, err, http.StatusBadRequest, tc.msgContains)
		})
	}
}

func TestUpdateBindingAcceptsSameTenantReferences(t *testing.T) {
	f := newBindingGuardFixture(t)

	req := f.baseUpdate()
	req.DepartmentID = f.deptA
	req.TeamID = f.teamA
	req.ApprovalChainID = strconv.Itoa(f.chainA)
	req.SLAPolicyID = strconv.Itoa(f.slaA)

	got, err := f.update(req)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, f.deptA, got.DepartmentID)
	assert.Equal(t, f.teamA, got.TeamID)
	assert.Equal(t, strconv.Itoa(f.chainA), got.ApprovalChainID)
	assert.Equal(t, strconv.Itoa(f.slaA), got.SLAPolicyID)
}

func TestCreateBindingMissingTenantContextFailsClosed(t *testing.T) {
	f := newBindingGuardFixture(t)
	svc := &ProcessBindingService{client: f.client}

	_, err := svc.CreateBinding(f.ctx, &dto.ProcessBinding{
		BusinessType:         "ticket",
		ProcessDefinitionKey: "ticket_general_flow",
		TenantID:             0,
	})
	requireAuthFailedBusinessError(t, err)
}

func TestCreateBindingRejectsForeignDepartment(t *testing.T) {
	f := newBindingGuardFixture(t)
	svc := &ProcessBindingService{client: f.client}

	_, err := svc.CreateBinding(f.ctx, &dto.ProcessBinding{
		BusinessType:         "ticket",
		ProcessDefinitionKey: "ticket_general_flow",
		TenantID:             f.tenantA,
		DepartmentID:         f.deptB,
	})
	requireAppErrorStatus(t, err, http.StatusBadRequest, "部门")
}

func TestCreateBindingAcceptsSameTenantDepartment(t *testing.T) {
	f := newBindingGuardFixture(t)
	svc := &ProcessBindingService{client: f.client}

	// CreateBinding 要求目标流程定义存在且 active+latest（与 StartProcess 同口径）。
	deployment, err := f.client.ProcessDeployment.Create().
		SetDeploymentID("DEP-BIND-GUARD").SetDeploymentName("bind").
		SetIsActive(true).SetTenantID(f.tenantA).
		Save(f.ctx)
	require.NoError(t, err)
	_, err = f.client.ProcessDefinition.Create().
		SetKey("ticket_general_flow").SetName("Ticket General").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(f.tenantA).
		Save(f.ctx)
	require.NoError(t, err)

	got, err := svc.CreateBinding(f.ctx, &dto.ProcessBinding{
		BusinessType:         "ticket",
		ProcessDefinitionKey: "ticket_general_flow",
		TenantID:             f.tenantA,
		DepartmentID:         f.deptA,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, f.deptA, got.DepartmentID)
	assert.Equal(t, f.tenantA, got.TenantID)
}
