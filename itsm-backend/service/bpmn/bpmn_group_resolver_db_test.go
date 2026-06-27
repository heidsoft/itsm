package bpmn

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
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
