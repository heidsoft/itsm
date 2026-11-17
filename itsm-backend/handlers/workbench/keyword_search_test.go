package workbench

import (
	"context"
	"testing"

	"itsm-backend/internal/datascope"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkbenchKeywordFiltersByTitle 是工作台搜索框接通的回归锁：
// 此前前端渲染了搜索框但 WorkbenchQuery 没有 keyword 通道，输入任何内容
// 列表都不动。此用例锁住「按标题模糊搜索 + 大小写不敏感 + 元字符按字面匹配」。
func TestWorkbenchKeywordFiltersByTitle(t *testing.T) {
	f := setupWorkbenchTest(t)
	repo := NewRepository(workbenchTestDB(t, f))
	ctx := context.Background()

	base := WorkbenchQuery{
		TenantID: f.tenantA, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeAll,
	}

	// 命中：标题含「工单」的活跃记录（已软删的 A-工单-已软删 不应出现）
	res, err := repo.Query(ctx, func() WorkbenchQuery {
		q := base
		q.Keyword = "工单"
		return q
	}())
	require.NoError(t, err)
	titles := make([]string, 0, len(res.Items))
	for _, item := range res.Items {
		titles = append(titles, item.Title)
	}
	assert.Contains(t, titles, "A-工单-待处理")
	assert.NotContains(t, titles, "A-事件-新建")
	assert.NotContains(t, titles, "A-工单-已软删", "软删记录不应被搜索召回")
	assert.NotContains(t, titles, "B-工单-待处理", "跨租户记录不应被搜索召回")

	// 大小写不敏感：fixture 无英文标题，用「a-」小写命中「A-」前缀
	resLower, err := repo.Query(ctx, func() WorkbenchQuery {
		q := base
		q.Keyword = "a-事件"
		return q
	}())
	require.NoError(t, err)
	assert.NotEmpty(t, resLower.Items, "小写关键字必须命中大写前缀标题（LOWER LIKE）")

	// 无命中返回空而非错误
	resNone, err := repo.Query(ctx, func() WorkbenchQuery {
		q := base
		q.Keyword = "绝不存在的关键字xyz"
		return q
	}())
	require.NoError(t, err)
	assert.Empty(t, resNone.Items)
	assert.Equal(t, 0, resNone.Total)

	// LIKE 元字符按字面匹配：搜索「100%」不能被 % 当通配符放大结果集
	resLiteral, err := repo.Query(ctx, func() WorkbenchQuery {
		q := base
		q.Keyword = "100%"
		return q
	}())
	require.NoError(t, err)
	assert.Empty(t, resLiteral.Items, "无字面「100%」标题时，% 不得被当作通配符召回全部")

	// 与优先级过滤可组合（同一 WHERE 层 AND）
	resCombo, err := repo.Query(ctx, func() WorkbenchQuery {
		q := base
		q.Keyword = "工单"
		q.RecordType = []string{"ticket"}
		return q
	}())
	require.NoError(t, err)
	for _, item := range resCombo.Items {
		assert.Equal(t, "ticket", item.RecordType)
		assert.Contains(t, item.Title, "工单")
	}
}
