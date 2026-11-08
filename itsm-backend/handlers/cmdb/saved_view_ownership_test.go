package cmdb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 编译期归属断言：保存视图实现与构造器必须属于本域包。
// 旧结构下它们都在共享 service 包，本文件在旧实现下无法编译。
var _ = NewCMDBSavedViewService(nil, nil)

func TestCMDBDomainOwnsSavedViewImplementation(t *testing.T) {
	require.NotNil(t, NewCMDBSavedViewService(nil, nil), "构造器必须返回本包实现")
}

// 源码守卫：迁入的实现不得反向 import 共享 service 包。
// 本域其它生产文件（production_service.go 等）仍依赖共享层，属批次 2 的
// ci_/cmdb_ 中型域后续范围，因此这里只锁本次迁入的文件本身。
func TestSavedViewImplementationHasNoSharedServiceImport(t *testing.T) {
	body, err := os.ReadFile("saved_view_service.go")
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"itsm-backend/service"`,
		"saved_view_service.go 不得 import 共享 service 包")
}
