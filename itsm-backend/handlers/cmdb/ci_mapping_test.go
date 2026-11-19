package cmdb

import (
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toCIDomain 的认证字段映射锁。
//
// 🔴 这条测试存在的理由是它抓到过一个真实的断裂：上一批把 owner_id/
// department_id/last_confirmed_at/confirmed_by 加到了 ent schema 与 dto，
// 却漏了 toCIDomain —— 领域实体拿不到这些值，于是治理看板「未纳入确认新鲜度」
// 不是「没做展示」，而是数据根本没流到那一层，且外面完全看不出异常
// （字段是零值，看板照常渲染，只是永远显示 0%）。
//
// 单测直接打映射函数而不建库：这一层的契约就是「ent 的字段必须原样落到领域模型」，
// 用 DB 反而把断裂点藏在 SQL 后面。反向验证时把映射行删掉（保留变量消费以通过
// 编译），本测试立即红在断言上。
func TestToCIDomain_MapsCertificationAndScopeFields(t *testing.T) {
	confirmedAt := time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC)
	syncTime := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)

	e := &ent.ConfigurationItem{
		ID:              42,
		TenantID:        1,
		Name:            "prod-db-01",
		OwnerID:         7,
		DepartmentID:    99,
		LastConfirmedAt: confirmedAt,
		ConfirmedBy:     11,
		CloudSyncTime:   syncTime,
	}

	ci := toCIDomain(e)
	require.NotNil(t, ci)

	assert.Equal(t, 7, ci.OwnerID, "行级权限载体必须映射，否则读面过滤拿不到 owner")
	assert.Equal(t, 99, ci.DepartmentID, "部门归属必须映射")
	assert.Equal(t, 11, ci.ConfirmedBy, "确认人必须映射")
	require.NotNil(t, ci.LastConfirmedAt, "确认时间必须映射成指针")
	assert.True(t, ci.LastConfirmedAt.Equal(confirmedAt), "确认时间值必须原样保留")
}

// TestToCIDomain_ZeroConfirmTimeBecomesNil 锁零值语义。
// ent 的 Optional()+Time 生成值类型，未确认时读出 time.Time 零值。若直接取地址
// 下发，调用方会把「从未确认」读成「确认于公元 1 年」——治理看板会把它算成
// 严重过期而不是从未确认，两种状态的运维含义完全不同。
func TestToCIDomain_ZeroConfirmTimeBecomesNil(t *testing.T) {
	e := &ent.ConfigurationItem{ID: 1, TenantID: 1, Name: "never-confirmed"}
	ci := toCIDomain(e)
	require.NotNil(t, ci)
	assert.Nil(t, ci.LastConfirmedAt, "零值必须转成 nil 指针，不能下发 0001-01-01")
	assert.Equal(t, 0, ci.ConfirmedBy)
	assert.Equal(t, 0, ci.OwnerID)
}
