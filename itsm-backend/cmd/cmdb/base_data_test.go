package main

// 回归：基础播种清单必须包含业务服务类型（cmdb_ci_business_service）。
//
// 业务服务是 CMDB 从技术资产跨到业务线的锚点：影响分析靠 CIRelationship 的
// impacts/depends_on/uses 边做 BFS，业务服务作为普通 CI 节点被遍历到，无需
// 按类型分支。若播种清单漏掉它，全新库（cmd/cmdb 播种路径）里就没有该类型，
// Agent 与 UI 都无法创建业务服务 CI，影响分析也就跨不到业务线。
//
// 同时锁住幂等性：重复播种不得产生重复类型（唯一索引 tenant_id+name）。

import (
	"context"
	"testing"

	"itsm-backend/ent/citype"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/enttest"
)

func TestInitializeBaseData_SeedsBusinessService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:cmdb_base_data?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	require.NoError(t, initializeBaseData(ctx, client))

	bs, err := client.CIType.Query().
		Where(citype.Name("cmdb_ci_business_service"), citype.TenantID(1)).
		Only(ctx)
	require.NoError(t, err, "播种清单必须含业务服务类型")
	assert.Equal(t, "业务服务", bs.Description)
	assert.NotEmpty(t, bs.Icon, "类型需带图标供 UI 渲染")

	// 幂等：再跑一次不得产生重复
	require.NoError(t, initializeBaseData(ctx, client))
	count, err := client.CIType.Query().
		Where(citype.Name("cmdb_ci_business_service"), citype.TenantID(1)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "重复播种不得产生重复业务服务类型")
}
