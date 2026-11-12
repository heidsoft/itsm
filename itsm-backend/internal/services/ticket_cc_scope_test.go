package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/ent"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createCCScopeTestUser 显式指定主角色，用于验证抄送可见性的角色边界。
func createCCScopeTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string, role user.Role) *ent.User {
	return client.User.Create().
		SetUsername("ccscope" + suffix).
		SetEmail("ccscope" + suffix + "@test.com").
		SetName("CC Scope User " + suffix).
		SetPasswordHash("hashedpassword").
		SetRole(role).
		SetActive(true).
		SetTenantID(tenantID).
		SaveX(ctx)
}

// TestListTicketCCRecords_RoleScopeUsesAuthoritativeAdminSet 锁定 P1 放宽缺陷：
// 修复前裸写 "super_admin","admin","manager","technician"，technician 作为普通
// Tier-1 主角色却能看到他人工单的抄送名单（含收件人身份），而 sysadmin/audit_admin
// 反而被漏配。可见性判定必须回到 domain/role.IsAdminLike 这一单一事实源。
func TestListTicketCCRecords_RoleScopeUsesAuthoritativeAdminSet(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	tenant, err := createTicketWorkflowTestTenant(ctx, client, "ccscope")
	require.NoError(t, err)

	// 请求人不能是任何被测角色，否则会先命中 requester 分支绕过角色判定。
	requester := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope-requester", domainrole.Agent)
	ticket, err := createTicketWorkflowTestTicket(ctx, client, tenant.ID, requester.ID, "new")
	require.NoError(t, err)

	for _, tc := range []struct {
		role      user.Role
		wantAllow bool
		why       string
	}{
		{domainrole.SuperAdmin, true, "管理面角色"},
		{domainrole.Admin, true, "管理面角色"},
		{domainrole.Manager, true, "管理面角色"},
		{domainrole.SysAdmin, true, "系统管理员，持全权限码"},
		{domainrole.AuditAdmin, true, "审计员按等保分立只读全租户"},
		{domainrole.Technician, false, "technician 不是管理角色"},
		{domainrole.Agent, false, "agent 不是管理角色"},
		{domainrole.ITAdmin, false, "it_admin 不在 IsAdminLike 词表内"},
		{domainrole.EndUser, false, "end_user 不是管理角色"},
	} {
		t.Run(string(tc.role), func(t *testing.T) {
			viewer := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope-"+string(tc.role), tc.role)
			require.Equal(t, tc.wantAllow, domainrole.IsAdminLike(string(tc.role)), "用例前提：与权威词表一致")

			records, err := service.ListTicketCCRecords(ctx, ticket.ID, viewer.ID, tenant.ID)
			if tc.wantAllow {
				require.NoError(t, err, tc.why)
				assert.Empty(t, records.Items, "无抄送记录时管理角色应拿到空列表而非报错")
				return
			}
			require.Error(t, err, "角色 %s 不应看到他人工单的抄送名单", tc.role)
			var bizErr *common.BusinessError
			require.ErrorAs(t, err, &bizErr)
			assert.Equal(t, common.ForbiddenCode, bizErr.Code)
		})
	}
}

// TestListTicketCCRecords_ParticipantBypassesRoleCheck 证明限制只作用于「无关第三方」：
// 请求人、处理人、被抄送人仍按参与关系放行，不因角色收紧而误伤。
func TestListTicketCCRecords_ParticipantBypassesRoleCheck(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	tenant, err := createTicketWorkflowTestTenant(ctx, client, "ccscope2")
	require.NoError(t, err)

	requester := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope2-requester", domainrole.EndUser)
	assignee := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope2-assignee", domainrole.EndUser)
	ccUser := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope2-cc", domainrole.EndUser)
	outsider := createCCScopeTestUser(ctx, client, tenant.ID, "ccscope2-outsider", domainrole.EndUser)

	ticket := client.Ticket.Create().
		SetTitle("CC Scope Ticket").
		SetStatus("new").
		SetPriority("medium").
		SetTicketNumber("CC-SCOPE-1").
		SetRequesterID(requester.ID).
		SetAssigneeID(assignee.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)

	client.TicketCC.Create().
		SetTicketID(ticket.ID).
		SetUserID(ccUser.ID).
		SetAddedBy(requester.ID).
		SetTenantID(tenant.ID).
		SaveX(ctx)

	for _, participantID := range []int{requester.ID, assignee.ID, ccUser.ID} {
		_, err := service.ListTicketCCRecords(ctx, ticket.ID, participantID, tenant.ID)
		assert.NoError(t, err, "参与人 %d 应可读取抄送名单", participantID)
	}

	_, err = service.ListTicketCCRecords(ctx, ticket.ID, outsider.ID, tenant.ID)
	require.Error(t, err, "与工单无关的 end_user 必须被拒绝")
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.ForbiddenCode, bizErr.Code)
}
