package service

import (
	"context"
	"encoding/json"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newSelfProfileTestUser(t *testing.T, client *ent.Client, tenantID int, username, email string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(email).
		SetPasswordHash("hashedpassword").
		SetName("Old Name").
		SetDepartment("Old Dept").
		SetPhone("13800000000").
		SetRole(user.RoleEndUser).
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

// TestUpdateOwnProfile_UpdatesWhitelistedFields 锁定自助面只改姓名/邮箱/电话/部门。
func TestUpdateOwnProfile_UpdatesWhitelistedFields(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP1").SetDomain("sp1.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u := newSelfProfileTestUser(t, client, tenant.ID, "selfuser", "self@sp1.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	updated, err := svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{
		Name:       "新名字",
		Email:      "new@sp1.com",
		Phone:      ptr("13900000000"),
		Department: ptr("新部门"),
	})
	require.NoError(t, err)
	assert.Equal(t, "新名字", updated.Name)
	assert.Equal(t, "new@sp1.com", updated.Email)
	assert.Equal(t, "13900000000", updated.Phone)
	assert.Equal(t, "新部门", updated.Department)
}

// TestUpdateOwnProfile_DistinguishesMissingFromClear 锁定「未传」保留原值、传空串才清空。
//
// 非指针字段下空串会被当成未传，用户点了清空却保存回旧值；反过来把未传当清空
// 会让只改姓名的表单静默抹掉电话。
func TestUpdateOwnProfile_DistinguishesMissingFromClear(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP2").SetDomain("sp2.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u := newSelfProfileTestUser(t, client, tenant.ID, "selfuser2", "self2@sp2.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	updated, err := svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{Name: "只改名字", Email: "self2@sp2.com"})
	require.NoError(t, err)
	assert.Equal(t, "只改名字", updated.Name)
	assert.Equal(t, "13800000000", updated.Phone, "未传 phone 必须保留原值")
	assert.Equal(t, "Old Dept", updated.Department, "未传 department 必须保留原值")

	cleared, err := svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{
		Name:       "只改名字",
		Email:      "self2@sp2.com",
		Phone:      ptr(""),
		Department: ptr(""),
	})
	require.NoError(t, err)
	assert.Empty(t, cleared.Phone, "传空串必须清空")
	assert.Empty(t, cleared.Department, "传空串必须清空")
}

// TestUpdateOwnProfile_RejectsTakenEmail 锁定邮箱占用校验，否则自助面可夺走他人登录邮箱。
func TestUpdateOwnProfile_RejectsTakenEmail(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP3").SetDomain("sp3.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	newSelfProfileTestUser(t, client, tenant.ID, "other", "taken@sp3.com")
	u := newSelfProfileTestUser(t, client, tenant.ID, "selfuser3", "self3@sp3.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	_, err = svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{Name: "me", Email: "taken@sp3.com"})
	require.Error(t, err)

	var conflict *common.BusinessError
	require.ErrorAs(t, err, &conflict)
	assert.Equal(t, common.ConflictCode, conflict.Code, "邮箱冲突必须是 conflict，不得伪装成内部错误")
	assert.Contains(t, conflict.Message, "邮箱")
}

// TestUpdateOwnProfile_RejectsEmptyName 锁定姓名必填，避免自助面把姓名清空成不可识别账号。
func TestUpdateOwnProfile_RejectsEmptyName(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP4").SetDomain("sp4.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u := newSelfProfileTestUser(t, client, tenant.ID, "selfuser4", "self4@sp4.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	_, err = svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{Name: "   ", Email: "self4@sp4.com"})
	require.Error(t, err)

	// 邮箱是登录身份，清空必须被拒绝而不是静默跳过。
	_, err = svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &dto.UpdateOwnProfileRequest{Name: "me", Email: "  "})
	require.Error(t, err)
}

// TestUpdateOwnProfile_DoesNotEscalateRole 锁定自助面无法通过请求体改角色/用户名/启用状态。
//
// 该端点只挂认证不挂 user:write（与自助改密同一权限面），因此字段白名单是唯一的
// 提权屏障：结构体里根本没有 role/username/active 字段，JSON 里的同名字段被忽略。
func TestUpdateOwnProfile_DoesNotEscalateRole(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP5").SetDomain("sp5.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u := newSelfProfileTestUser(t, client, tenant.ID, "selfuser5", "self5@sp5.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	// 直接反序列化带提权字段的请求体，模拟恶意客户端。
	var req dto.UpdateOwnProfileRequest
	require.NoError(t, json.Unmarshal([]byte(`{"name":"me","email":"self5@sp5.com","role":"admin","username":"pwn","active":false,"tenantId":99}`), &req))

	updated, err := svc.UpdateOwnProfile(ctx, u.ID, tenant.ID, &req)
	require.NoError(t, err)
	assert.Equal(t, user.RoleEndUser, updated.Role, "角色不得被自助面修改")
	assert.Equal(t, "selfuser5", updated.Username, "用户名不得被自助面修改")
	assert.True(t, updated.Active, "启用状态不得被自助面修改")
	assert.Equal(t, tenant.ID, updated.TenantID, "租户归属不得被自助面修改")
}

// TestUpdateOwnProfile_CrossTenantFailsClosed 锁定目标用户不属于调用者租户时直接失败。
func TestUpdateOwnProfile_CrossTenantFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenantA, err := client.Tenant.Create().SetName("A").SetCode("SPA").SetDomain("spa.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("B").SetCode("SPB").SetDomain("spb.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u := newSelfProfileTestUser(t, client, tenantA.ID, "selfuser6", "self6@spa.com")
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	// email 必须带上合法值：留空会在参数校验分支提前返回，租户谓词根本不会被执行，
	// 这条用例就成了「因为另一个原因失败」的假绿。
	_, err = svc.UpdateOwnProfile(ctx, u.ID, tenantB.ID, &dto.UpdateOwnProfileRequest{Name: "Hacked", Email: "self6@spa.com"})
	require.Error(t, err)

	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.NotFoundCode, bizErr.Code, "跨租户必须按 not-found 掩码，不得泄漏租户归属细节")

	// 未被篡改：跨租户调用必须一个字段都写不进去。
	fresh, err := client.User.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Old Name", fresh.Name)
	assert.Equal(t, "self6@spa.com", fresh.Email)
}

// TestUpdateOwnProfile_RequiresAuthContext 锁定 userID 缺失时 fail closed，
// 不允许退化成「更新任意用户」。
func TestUpdateOwnProfile_RequiresAuthContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("SP7").SetDomain("sp7.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	newSelfProfileTestUser(t, client, tenant.ID, "selfuser7", "self7@sp7.com")

	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())
	_, err = svc.UpdateOwnProfile(ctx, 0, tenant.ID, &dto.UpdateOwnProfileRequest{Name: "me", Email: "self7@sp7.com"})
	require.Error(t, err)

	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.UnauthorizedCode, bizErr.Code)
}
