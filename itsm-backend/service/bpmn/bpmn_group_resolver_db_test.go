package bpmn

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"
)

func setupGroupResolverDB(t *testing.T) (*ent.Client, *GroupResolver, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:resolver_test?mode=memory&cache=shared&_fk=1")
	return client, NewGroupResolver(client), context.Background()
}

func createTestTenant(t *testing.T, client *ent.Client, code string) *ent.Tenant {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetCode(code).
		SetName("测试租户-" + code).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant
}

func createTestUser(t *testing.T, client *ent.Client, tenantID int, username, email string) *ent.User {
	t.Helper()
	ctx := context.Background()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(email).
		SetName(username).
		SetPasswordHash("test").
		SetTenantID(tenantID).
		SetActive(true).
		Save(ctx)
	require.NoError(t, err)
	return u
}

func createTestGroup(t *testing.T, client *ent.Client, tenantID int, name string, memberIDs ...int) *ent.Group {
	t.Helper()
	ctx := context.Background()
	g, err := client.Group.Create().
		SetName(name).
		SetTenantID(tenantID).
		AddMemberIDs(memberIDs...).
		Save(ctx)
	require.NoError(t, err)
	return g
}

func TestGroupResolver_ExpandGroupsToUsers_NoGroups(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()
	createTestTenant(t, client, "T1")

	ids, names, err := resolver.ExpandGroupsToUsers(ctx, 1, "")
	require.NoError(t, err)
	assert.Nil(t, ids)
	assert.Nil(t, names)

	ids, names, err = resolver.ExpandGroupsToUsers(ctx, 1, "  , , ")
	require.NoError(t, err)
	assert.Nil(t, ids)
	assert.Nil(t, names)
}

func TestGroupResolver_ExpandGroupsToUsers_NonexistentGroupIsTolerated(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()
	createTestTenant(t, client, "T1")

	// 不存在的组名应该被容忍，不返回错误
	ids, names, err := resolver.ExpandGroupsToUsers(ctx, 1, "ghost-group")
	require.NoError(t, err)
	assert.Empty(t, ids)
	assert.Empty(t, names)
}

func TestGroupResolver_ExpandGroupsToUsers_HappyPath(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()
	tenant := createTestTenant(t, client, "T1")

	alice := createTestUser(t, client, tenant.ID, "alice", "alice@example.com")
	bob := createTestUser(t, client, tenant.ID, "bob", "bob@example.com")
	carol := createTestUser(t, client, tenant.ID, "carol", "carol@example.com")

	// 三个组：managers / engineers / ops（ops 不在 candidateGroups 中）
	// 每个用户只属于一个组（group_members 是唯一边）
	createTestGroup(t, client, tenant.ID, "managers", alice.ID)
	createTestGroup(t, client, tenant.ID, "engineers", bob.ID)
	createTestGroup(t, client, tenant.ID, "ops", carol.ID)

	ids, names, err := resolver.ExpandGroupsToUsers(ctx, tenant.ID, "managers,engineers")
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{alice.ID, bob.ID}, ids)

	// 三个用户都有 username 和 email，所以 usernames 都是 username
	assert.ElementsMatch(t, []string{"alice", "bob"}, names)
}

func TestGroupResolver_ExpandGroupsToUsers_TenantIsolation(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()

	t1 := createTestTenant(t, client, "T1")
	t2 := createTestTenant(t, client, "T2")

	// alice 属于 T1 的 managers 组
	alice := createTestUser(t, client, t1.ID, "alice", "alice@t1.com")
	createTestGroup(t, client, t1.ID, "managers", alice.ID)

	// 在 T2 上查询 managers，应该拿不到 alice（租户隔离）
	ids, _, err := resolver.ExpandGroupsToUsers(ctx, t2.ID, "managers")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestGroupResolver_GetUserGroupNames_NoUser(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()

	got, err := resolver.GetUserGroupNames(ctx, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestGroupResolver_GetUserGroupNames_HappyPath(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()
	tenant := createTestTenant(t, client, "T1")
	alice := createTestUser(t, client, tenant.ID, "alice", "alice@example.com")
	_ = alice

	// user 只能属于一个 group，所以测试一个用户一个组的场景
	// 多组场景需要多用户
	bob := createTestUser(t, client, tenant.ID, "bob", "bob@example.com")
	carol := createTestUser(t, client, tenant.ID, "carol", "carol@example.com")

	createTestGroup(t, client, tenant.ID, "managers", alice.ID)
	createTestGroup(t, client, tenant.ID, "engineers", bob.ID)
	createTestGroup(t, client, tenant.ID, "ops", carol.ID)

	// 验证每个用户只属于一个组
	gotAlice, err := resolver.GetUserGroupNames(ctx, tenant.ID, alice.ID)
	require.NoError(t, err)
	assert.Equal(t, "managers", gotAlice)

	gotBob, err := resolver.GetUserGroupNames(ctx, tenant.ID, bob.ID)
	require.NoError(t, err)
	assert.Equal(t, "engineers", gotBob)
}

func TestGroupResolver_GetUserGroupNames_TenantIsolation(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()

	t1 := createTestTenant(t, client, "T1")
	t2 := createTestTenant(t, client, "T2")

	alice := createTestUser(t, client, t1.ID, "alice", "alice@t1.com")
	createTestGroup(t, client, t1.ID, "managers", alice.ID)

	// 在 T2 上查询 alice 的组应该返回空（租户隔离）
	got, err := resolver.GetUserGroupNames(ctx, t2.ID, alice.ID)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

// TestGroupResolver_ExpandGroupsToUsers_RoleFallback 验证角色回退：组名不存在时
// 若与 roles 表角色 code 同名，按角色解析用户（M2M ∪ 主角色枚举）。
// 背景：groups 表无种子时设计器选角色（code 同名）此前产出空候选集。
func TestGroupResolver_ExpandGroupsToUsers_RoleFallback(t *testing.T) {
	client, resolver, ctx := setupGroupResolverDB(t)
	defer client.Close()
	tenant := createTestTenant(t, client, "T1")
	otherTenant := createTestTenant(t, client, "T2")

	itAdmin, err := client.Role.Create().SetCode("it_admin").SetName("IT管理").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.Role.Create().SetCode("it_admin").SetName("IT管理B").SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)

	newUser := func(tenantID int, username, enumRole string, roleIDs ...int) *ent.User {
		u := createTestUser(t, client, tenantID, username, username+"@example.com")
		u, err = u.Update().SetRole(user.Role(enumRole)).Save(ctx)
		require.NoError(t, err)
		if len(roleIDs) > 0 {
			_, err = client.User.UpdateOne(u).AddRoleIDs(roleIDs...).Save(ctx)
			require.NoError(t, err)
		}
		return u
	}

	_ = newUser(tenant.ID, "enum-admin", "it_admin")                   // 仅主角色枚举命中
	m2mUser := newUser(tenant.ID, "m2m-admin", "end_user", itAdmin.ID) // 仅 M2M 命中
	_ = m2mUser
	_ = newUser(tenant.ID, "plain", "end_user")              // 无关角色
	_ = newUser(otherTenant.ID, "foreign-admin", "it_admin") // 他租户排除
	inactive := newUser(tenant.ID, "inactive-admin", "it_admin")
	_, err = inactive.Update().SetActive(false).Save(ctx)
	require.NoError(t, err)

	ids, names, err := resolver.ExpandGroupsToUsers(ctx, tenant.ID, "it_admin")
	require.NoError(t, err)
	assert.NotEmpty(t, ids, "角色回退应命中本租户 it_admin 用户")
	assert.Len(t, ids, 2, "枚举+M2M 两名活跃用户")
	assert.NotContains(t, ids, inactive.ID)
	assert.ElementsMatch(t, []string{"enum-admin", "m2m-admin"}, names)

	// 组与角色同名时组优先：显式建的组是对角色的窄化/覆盖，不再叠加角色回退
	groupUser := createTestUser(t, client, tenant.ID, "group-member", "gm@example.com")
	_ = createTestGroup(t, client, tenant.ID, "it_admin", groupUser.ID)
	ids, names, err = resolver.ExpandGroupsToUsers(ctx, tenant.ID, "it_admin")
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{groupUser.ID}, ids, "同名组存在时按组解析，角色回退不叠加")

	// 纯幽灵名仍容忍，组名 + 角色名混排各自解析
	ids, names, err = resolver.ExpandGroupsToUsers(ctx, tenant.ID, "ghost-group,managers-group")
	require.NoError(t, err)
	assert.Empty(t, ids, "幽灵名被容忍不报错")
	ids, _, err = resolver.ExpandGroupsToUsers(ctx, tenant.ID, "ghost-group,it_admin")
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{groupUser.ID}, ids, "幽灵名不影响同段内其他名字解析")
}
