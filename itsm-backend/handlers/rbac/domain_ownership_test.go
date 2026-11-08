package rbac

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 编译期归属断言：角色/权限的实现类型、构造函数返回值与哨兵全部属于本域包。
// 回归锁意义：旧结构里这些符号都在共享 service 包，本文件在旧实现下无法编译，
// 因此能区分新旧实现，不是「只断言结果」的假绿。
var (
	_ RoleService       = (*roleServiceImpl)(nil)
	_ PermissionService = (*permissionServiceImpl)(nil)
	_ RoleService       = NewRoleService(nil, nil)
	_ PermissionService = NewPermissionService(nil, nil)
)

func TestRBACDomainOwnsRoleImplementation(t *testing.T) {
	var svc RoleService = NewRoleService(nil, nil)
	require.NotNil(t, svc, "域包构造函数必须返回域内接口实现")

	assert.Error(t, ErrPermissionNotInTenant)
	assert.True(t, errors.Is(ErrPermissionNotInTenant, ErrPermissionNotInTenant),
		"跨租户权限分配 400 语义依赖哨兵的 errors.Is 判定")
	assert.Contains(t, ErrPermissionNotInTenant.Error(), "不属于当前租户")
}

// 源码守卫：域包生产文件不得重新引入共享 service 包。
// L.3 只在门禁侧挡 import；此锁让 go test 也能直接发现回退。
func TestRBACDomainHasNoSharedServiceImport(t *testing.T) {
	for _, name := range []string{"handler.go", "service.go", "role_service_impl.go"} {
		body, err := os.ReadFile(name)
		require.NoError(t, err, "读取 %s", name)
		assert.NotContains(t, string(body), `"itsm-backend/service"`,
			"%s 不得 import 共享 service 包", name)
	}
}
