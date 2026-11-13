package service

import (
	"context"
	"path/filepath"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件是 prod O4 的回归锁。
//
// 实测现象：给用户附加「MSP管理员」角色并重新登录后，/msp 仍显示
// 「您当前不是 MSP 员工」，/msp/management 显示「您没有权限访问此页面」，
// 登录响应 mspRole 恒为空串。
//
// 根因：MSP 工作台门控（middleware.IsMSPStaff）只读 users.msp_role，
// 而编辑用户的「附加角色」写的是 user_roles 边——两条互不相通的链路。
// 更关键的是：msp_role **只有创建用户时能写**，编辑用户改不了、也清不掉，
// UI 没有任何入口，于是「把某人设为 MSP 员工」在产品上无法完成。
//
// 修复：UpdateUser 支持写 msp_role（指针语义：nil=不动，''=清空）。

func TestUserService_UpdateUserSetsMSPRole(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "user_msp_role.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	tenant, err := client.Tenant.Create().
		SetName("MSP").SetCode("MSP-ROLE").SetDomain("msp.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("msp-user").
		SetEmail("msp@test.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetRole("technician").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	require.Equal(t, user.MspRole(""), u.MspRole, "前置：新建用户默认没有 MSP 身份")

	// ① 写入 provider_admin：编辑用户必须能设置 MSP 员工身份。
	providerAdmin := string(user.MspRoleProviderAdmin)
	updated, err := svc.UpdateUser(ctx, u.ID, &dto.UpdateUserRequest{
		MSPRole: &providerAdmin,
	}, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, user.MspRoleProviderAdmin, updated.MspRole,
		"编辑用户必须能写入 MSP 员工身份，否则 /msp 对任何人都不可达")

	reloaded, err := client.User.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, user.MspRoleProviderAdmin, reloaded.MspRole, "MSP 身份必须已持久化")

	// ② 清空：'' 语义是显式解除，不是「未传」。
	empty := ""
	updated, err = svc.UpdateUser(ctx, u.ID, &dto.UpdateUserRequest{
		MSPRole: &empty,
	}, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, user.MspRole(""), updated.MspRole, "空串必须能显式清空 MSP 身份")

	// ③ 未传（nil）：保持原值不变。
	updated, err = svc.UpdateUser(ctx, u.ID, &dto.UpdateUserRequest{
		MSPRole: &providerAdmin,
	}, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, user.MspRoleProviderAdmin, updated.MspRole)

	updated, err = svc.UpdateUser(ctx, u.ID, &dto.UpdateUserRequest{
		Name: "改名不改身份",
	}, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, user.MspRoleProviderAdmin, updated.MspRole, "未传 mspRole 时不得清空既有身份")
}

// TestUserService_UpdateUserRejectsInvalidMSPRole 钉住非法值必须失败关闭：
// 写入未知 MSP 身份会让 IsMSPStaff 的判定语义变得不可预期。
func TestUserService_UpdateUserRejectsInvalidMSPRole(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "user_msp_role_invalid.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	tenant, err := client.Tenant.Create().
		SetName("MSP2").SetCode("MSP-ROLE-INV").SetDomain("msp2.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("msp-user-invalid").
		SetEmail("msp-invalid@test.com").
		SetName("MSP Invalid").
		SetPasswordHash("hash").
		SetRole("technician").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	invalid := "root_admin"
	_, err = svc.UpdateUser(ctx, u.ID, &dto.UpdateUserRequest{
		MSPRole: &invalid,
	}, tenant.ID)
	require.Error(t, err, "未知 MSP 身份必须被拒绝")
	assert.Contains(t, err.Error(), "无效的 MSP 角色")

	reloaded, err := client.User.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, user.MspRole(""), reloaded.MspRole, "被拒绝时不得写入任何身份")
}
