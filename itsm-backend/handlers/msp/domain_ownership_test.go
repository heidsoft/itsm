package msp

import (
	"os"
	"testing"

	"itsm-backend/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 编译期归属断言：分配实现与两个哨兵必须属于本域包，且共享层工单实现
// 仍满足本域窄接口（组合根注入点的类型约束由这里在 go test 阶段守住）。
// 回归锁意义：旧结构下 NewMSPAllocationService / TicketService 都不在本包，
// 本文件在旧实现下无法编译，因此能区分新旧实现，不是「只断言结果」的假绿。
var (
	_               = NewMSPAllocationService(nil, nil)
	_               = ErrMSPAllocationForbidden
	_               = ErrMSPAllocationNotFound
	_ TicketService = (*service.TicketService)(nil)
)

func TestMSPDomainOwnsAllocationImplementation(t *testing.T) {
	require.NotNil(t, NewMSPAllocationService(nil, nil), "域包构造函数必须返回本包实现")
	assert.Error(t, ErrMSPAllocationForbidden, "跨域操作 403 依赖该哨兵")
	assert.Error(t, ErrMSPAllocationNotFound, "解除分配 404 依赖该哨兵")
}

// 源码守卫：域包生产文件不得重新引入共享 service 包。
// 测试文件允许 import service（用于上面的编译期断言），L.3 只统计生产文件。
func TestMSPDomainHasNoSharedServiceImport(t *testing.T) {
	for _, name := range []string{"handler.go", "allocation_service_impl.go", "contracts.go"} {
		body, err := os.ReadFile(name)
		require.NoError(t, err, "读取 %s", name)
		assert.NotContains(t, string(body), `"itsm-backend/internal/services"`,
			"%s 不得 import 共享 service 包", name)
	}
}
