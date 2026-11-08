package dashboard

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 编译期归属断言：仪表盘配置 store 的实现与构造器必须属于本域包。
// 旧结构下它们在共享 service 包，本文件在旧实现下无法编译。
var _ = NewDashboardConfigStore(nil, nil)

func TestDashboardDomainOwnsConfigStore(t *testing.T) {
	require.NotNil(t, NewDashboardConfigStore(nil, nil), "构造器必须返回本包实现")
}

// 源码守卫：迁入的实现不得反向 import 共享 service 包。
// 本域 handler.go 仍依赖共享层 DTO/服务（属 dashboard 中型域后续范围），
// 因此这里只锁本次迁入的文件本身。
func TestConfigStoreHasNoSharedServiceImport(t *testing.T) {
	body, err := os.ReadFile("config_store.go")
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"itsm-backend/service"`,
		"config_store.go 不得 import 共享 service 包")
}
