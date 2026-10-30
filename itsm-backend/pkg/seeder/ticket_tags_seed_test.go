package seeder

import (
	"testing"

	"itsm-backend/ent/tickettag"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedTicketTags 的账实分离守卫：种子必须落进工单标签域真正读取的表。
//
// 此前种子写的是 tags 表（组织标签）——那张表没有任何管理面与读取方
// （BindTagToEntity 无调用方，五张 join 表在 prod 实测全 0），而工单标签
// 域（service.TicketTagService 的 CRUD 与 ResolveTagIDsByNames、前端
// /admin/tags 页）读写的是 ticket_tags 表。结果是每个新环境都把 15 个
// 工单标签种进一张死表，工单标签页打开是空的（prod 实证：tags 15 行、
// ticket_tags 0 行）。2026-10-30 修正，本测试防止回退。
func TestSeedTicketTagsWritesTicketTagTable(t *testing.T) {
	seeder, ctx := newTestSeeder(t, "private")
	seeder.seedDefaultTenant(ctx)

	tenant, err := seeder.baselineTenant(ctx)
	require.NoError(t, err)

	require.NoError(t, seeder.seedTicketTags(ctx))

	// 1) 目标表必须拿到全部基线标签（数量对齐清单，不取并集）。
	expected := seeder.expectedTicketTags()
	rows, err := seeder.client.TicketTag.Query().
		Where(tickettag.TenantIDEQ(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	names := make(map[string]bool, len(rows))
	for _, r := range rows {
		names[r.Name] = true
	}
	for _, e := range expected {
		assert.True(t, names[e.Name], "ticket_tags 缺少基线标签 %q", e.Name)
	}
	assert.Len(t, rows, len(expected), "ticket_tags 行数应与基线清单一致（不重复种）")

	// 2) 组织标签表（tags）不得再被工单标签种子污染：它是死表面，
	// 任何新写入都意味着种子又写错了表。
	orgTags, err := seeder.client.Tag.Query().All(ctx)
	require.NoError(t, err)
	assert.Empty(t, orgTags, "工单标签种子不得写入组织标签表（tags）——该表无读取方，写入即账实分离")

	// 3) 幂等：重跑不重复插入。
	require.NoError(t, seeder.seedTicketTags(ctx))
	again, err := seeder.client.TicketTag.Query().
		Where(tickettag.TenantIDEQ(tenant.ID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(expected), again, "seedTicketTags 必须幂等")
}
