package common

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	_ "github.com/mattn/go-sqlite3"
)

// loginService builds the same Service the /api/v1/auth/login route uses.
// repo stays nil on purpose: Login must resolve the account through the client,
// and a nil repo makes any accidental repo dependency fail loudly.
func loginService(client *ent.Client) *Service {
	return NewService(nil, "test-jwt-secret-with-32-bytes!!", zap.NewNop().Sugar(), client)
}

func newLoginTenant(t *testing.T, client *ent.Client, ctx context.Context, code string) *ent.Tenant {
	t.Helper()
	created, err := client.Tenant.Create().
		SetName("Tenant " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return created
}

// newLoginUser 用不同口令区分租户，这样断言能证明登录命中的是哪一行。
// email 在 users 上仍是全局唯一键，因此按租户派生。
func newLoginUser(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, username, password string) *ent.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	created, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "-tenant-" + strconv.Itoa(tenantID) + "@example.com").
		SetName(username).
		SetPasswordHash(string(hash)).
		SetRole(user.RoleAdmin).
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return created
}

// openLoginDB 每个用例一个独立内存库：shared-cache 下同名库会在用例间串数据，
// 而「用户名是否跨租户重名」正是本组用例的被测条件。
func openLoginDB(t *testing.T) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		"file:login-tenant-scope-"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestLoginWithoutTenantKeepsWorkingForUniqueUsername 保住现有 Web 登录页：
// 它只提交 username/password（src/app/(auth)/login/page.tsx），
// 收紧成「必须带租户」会让所有单命中账号登不进去。
func TestLoginWithoutTenantKeepsWorkingForUniqueUsername(t *testing.T) {
	client := openLoginDB(t)
	ctx := context.Background()
	tenantA := newLoginTenant(t, client, ctx, "logna")
	adminA := newLoginUser(t, client, ctx, tenantA.ID, "admin", "password-a")

	result, err := loginService(client).Login(ctx, "admin", "password-a", 0, "")
	require.NoError(t, err)
	require.Equal(t, adminA.ID, result.User.ID)
	require.Equal(t, tenantA.ID, result.User.TenantID)
}

// TestLoginWithDuplicateUsernameRequiresTenant 是本次改动的核心安全断言：
// username 唯一键改成 (tenant_id, username) 后，跨租户同名账号真实存在。
// 旧实现用 Only() 查询，多命中会退化成「用户名或密码错误」——用户被永久锁在门外
// 且得不到可执行的提示；现在必须显式要求消歧，而不是猜一个租户。
func TestLoginWithDuplicateUsernameRequiresTenant(t *testing.T) {
	client := openLoginDB(t)
	ctx := context.Background()
	tenantA := newLoginTenant(t, client, ctx, "logndup-a")
	tenantB := newLoginTenant(t, client, ctx, "logndup-b")
	newLoginUser(t, client, ctx, tenantA.ID, "admin", "password-a")
	newLoginUser(t, client, ctx, tenantB.ID, "admin", "password-b")

	_, err := loginService(client).Login(ctx, "admin", "password-a", 0, "")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrLoginTenantRequired),
		"ambiguous username must ask for a tenant, got %v", err)
}

// TestLoginWithTenantSelectsThatTenantsAccount 证明消歧信息足够登录成功，
// 并且拿到的是被指定租户的账号而不是另一个租户的同名账号。
func TestLoginWithTenantSelectsThatTenantsAccount(t *testing.T) {
	client := openLoginDB(t)
	ctx := context.Background()
	tenantA := newLoginTenant(t, client, ctx, "lognpick-a")
	tenantB := newLoginTenant(t, client, ctx, "lognpick-b")
	adminA := newLoginUser(t, client, ctx, tenantA.ID, "admin", "password-a")
	adminB := newLoginUser(t, client, ctx, tenantB.ID, "admin", "password-b")

	svc := loginService(client)

	resultA, err := svc.Login(ctx, "admin", "password-a", tenantA.ID, "")
	require.NoError(t, err)
	require.Equal(t, adminA.ID, resultA.User.ID)

	resultB, err := svc.Login(ctx, "admin", "password-b", tenantB.ID, "")
	require.NoError(t, err)
	require.Equal(t, adminB.ID, resultB.User.ID)

	// 指定错误租户时不能登录成功，也不能借另一个租户的口令蒙对。
	_, err = svc.Login(ctx, "admin", "password-a", tenantB.ID, "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrLoginTenantRequired)
}

// TestLoginWithTenantCodeResolvesTenant 覆盖 tenantCode 分支（MSP/多租户入口）。
func TestLoginWithTenantCodeResolvesTenant(t *testing.T) {
	client := openLoginDB(t)
	ctx := context.Background()
	tenantA := newLoginTenant(t, client, ctx, "logncode-a")
	tenantB := newLoginTenant(t, client, ctx, "logncode-b")
	adminB := newLoginUser(t, client, ctx, tenantB.ID, "admin", "password-b")
	newLoginUser(t, client, ctx, tenantA.ID, "admin", "password-a")

	result, err := loginService(client).Login(ctx, "admin", "password-b", 0, "logncode-b")
	require.NoError(t, err)
	require.Equal(t, adminB.ID, result.User.ID)
}

// TestLoginWithUnknownTenantCodeFailsClosed 修掉旧的 fail-open：
// 原实现是「解析 tenantCode 失败就沿用 tenantID=0」，等价于拼错租户名后
// 悄悄降级成跨租户按用户名匹配。现在必须按凭证错误拒绝。
func TestLoginWithUnknownTenantCodeFailsClosed(t *testing.T) {
	client := openLoginDB(t)
	ctx := context.Background()
	tenantA := newLoginTenant(t, client, ctx, "lognbad-a")
	tenantB := newLoginTenant(t, client, ctx, "lognbad-b")
	newLoginUser(t, client, ctx, tenantA.ID, "admin", "password-a")
	newLoginUser(t, client, ctx, tenantB.ID, "admin", "password-b")

	_, err := loginService(client).Login(ctx, "admin", "password-a", 0, "tenant-that-does-not-exist")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrLoginTenantRequired,
		"a bad tenant code must not fall back to cross-tenant username matching")
	require.EqualError(t, err, "invalid credentials")
}
