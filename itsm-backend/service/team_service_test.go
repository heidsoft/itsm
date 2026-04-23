package service

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/team"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 测试设置辅助函数 ====================

func setupTeamTest(t *testing.T) (*ent.Client, *TeamService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	service := NewTeamService(client)
	ctx := context.Background()
	return client, service, ctx
}

func createTeamTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createTeamTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser" + suffix).
		SetEmail("test" + suffix + "@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== 创建团队测试 ====================

func TestTeamService_CreateTeam_Success(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "create")
	require.NoError(t, err)

	testUser, err := createTeamTestUser(ctx, client, testTenant.ID, "manager")
	require.NoError(t, err)

	team, err := service.CreateTeam(ctx, "运维团队", "OPS-001", "负责系统运维", testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "运维团队", team.Name)
	assert.Equal(t, "OPS-001", team.Code)
	assert.Equal(t, "负责系统运维", team.Description)
	assert.Equal(t, testUser.ID, team.ManagerID)
}

func TestTeamService_CreateTeam_WithoutManager(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "nomgr")
	require.NoError(t, err)

	team, err := service.CreateTeam(ctx, "支持团队", "SUPPORT-001", "技术支持团队", 0, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, team)
	assert.Equal(t, "支持团队", team.Name)
}

// ==================== 添加成员测试 ====================

func TestTeamService_AddMember_Success(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "member")
	require.NoError(t, err)

	testUser, err := createTeamTestUser(ctx, client, testTenant.ID, "member")
	require.NoError(t, err)

	// 创建团队
	teamEntity, err := service.CreateTeam(ctx, "开发团队", "DEV-001", "开发团队", 0, testTenant.ID)
	require.NoError(t, err)

	// 添加成员
	err = service.AddMember(ctx, teamEntity.ID, testUser.ID)
	require.NoError(t, err)

	// 验证成员已添加
	updated, err := client.Team.Query().WithUsers().Where(team.IDEQ(teamEntity.ID)).First(ctx)
	require.NoError(t, err)
	assert.Len(t, updated.Edges.Users, 1)
}

// ==================== 列出团队测试 ====================

func TestTeamService_ListTeams_Success(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "list")
	require.NoError(t, err)

	// 创建多个团队
	for i := 0; i < 5; i++ {
		_, err := service.CreateTeam(ctx, fmt.Sprintf("团队-%d", i), fmt.Sprintf("TEAM-%03d", i), "", 0, testTenant.ID)
		require.NoError(t, err)
	}

	teams, err := service.ListTeams(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 5)
}

func TestTeamService_ListTeams_Empty(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	teams, err := service.ListTeams(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.Empty(t, teams)
}

// ==================== 更新团队测试 ====================

func TestTeamService_UpdateTeam_Success(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "update")
	require.NoError(t, err)

	testUser, err := createTeamTestUser(ctx, client, testTenant.ID, "newmgr")
	require.NoError(t, err)

	// 创建团队
	team, err := service.CreateTeam(ctx, "原团队名", "ORIG-001", "原描述", 0, testTenant.ID)
	require.NoError(t, err)

	// 更新团队
	newName := "新团队名"
	newCode := "NEW-001"
	newDesc := "新描述"

	updated, err := service.UpdateTeam(ctx, team.ID, &newName, &newCode, &newDesc, &testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, newCode, updated.Code)
	assert.Equal(t, newDesc, updated.Description)
	assert.Equal(t, testUser.ID, updated.ManagerID)
}

func TestTeamService_UpdateTeam_PartialUpdate(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "partial")
	require.NoError(t, err)

	// 创建团队
	team, err := service.CreateTeam(ctx, "原团队名", "ORIG-002", "原描述", 0, testTenant.ID)
	require.NoError(t, err)

	// 只更新名称
	newName := "仅更新名称"
	updated, err := service.UpdateTeam(ctx, team.ID, &newName, nil, nil, nil, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, "ORIG-002", updated.Code) // 代码保持不变
}

func TestTeamService_UpdateTeam_TenantMismatch(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant1, err := createTeamTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createTeamTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	// 在租户1创建团队
	team, err := service.CreateTeam(ctx, "团队1", "TEAM-001", "", 0, testTenant1.ID)
	require.NoError(t, err)

	// 尝试用租户2的ID更新，应该失败
	newName := "尝试跨租户更新"
	_, err = service.UpdateTeam(ctx, team.ID, &newName, nil, nil, nil, testTenant2.ID)
	require.Error(t, err)
}

// ==================== 删除团队测试 ====================

func TestTeamService_DeleteTeam_Success(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "delete")
	require.NoError(t, err)

	// 创建团队
	team, err := service.CreateTeam(ctx, "待删除团队", "DEL-001", "", 0, testTenant.ID)
	require.NoError(t, err)

	// 删除团队
	err = service.DeleteTeam(ctx, team.ID, testTenant.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = client.Team.Get(ctx, team.ID)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

func TestTeamService_DeleteTeam_NotFound(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant, err := createTeamTestTenant(ctx, client, "delnotfound")
	require.NoError(t, err)

	err = service.DeleteTeam(ctx, 99999, testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "团队不存在")
}

func TestTeamService_DeleteTeam_TenantMismatch(t *testing.T) {
	client, service, ctx := setupTeamTest(t)
	defer client.Close()

	testTenant1, err := createTeamTestTenant(ctx, client, "deltenant1")
	require.NoError(t, err)

	testTenant2, err := createTeamTestTenant(ctx, client, "deltenant2")
	require.NoError(t, err)

	// 在租户1创建团队
	team, err := service.CreateTeam(ctx, "团队", "TEAM-DEL", "", 0, testTenant1.ID)
	require.NoError(t, err)

	// 尝试用租户2的ID删除，应该失败
	err = service.DeleteTeam(ctx, team.ID, testTenant2.ID)
	require.Error(t, err)
}
