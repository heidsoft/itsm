package problem

import (
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProblemStatusMachineMatchesVocabulary 锁定问题状态机与数据库 CHECK 词表一致。
//
// 2026-10-31 实测缺陷：common 里曾有 ProblemStatusNew = "new"，而问题域
// 创建/状态机/报表全用 "open"。状态机是第二处可能漂移的地方——新增一个
// 状态但没进 CHECK 词表时，mock 单测全绿、生产写入撞 23514。
func TestProblemStatusMachineMatchesVocabulary(t *testing.T) {
	require.NotEmpty(t, problemStatusTransitions)

	for from, targets := range problemStatusTransitions {
		assert.True(t, common.StatusValueAllowed("problems", "status", from),
			"问题状态机以 %q 为源状态，但该值不在 CHECK 词表内", from)
		for to := range targets {
			assert.True(t, common.StatusValueAllowed("problems", "status", to),
				"问题状态机允许迁移到 %q，但该值不在 CHECK 词表内（生产会 23514）", to)
		}
	}
}

// TestProblemDefaultStatusIsCanonical 锁定创建时的默认状态。
// ent schema 的默认值是 "open"，若有人把创建路径改成 "new"，
// 数据库 CHECK 会拒绝——这里先一步给出可读的失败信息。
func TestProblemDefaultStatusIsCanonical(t *testing.T) {
	assert.True(t, common.StatusValueAllowed("problems", "status", "open"))
	assert.False(t, common.StatusValueAllowed("problems", "status", "new"),
		`"new" 不是问题域的合法状态：权威词表用 "open"（历史幽灵常量 ProblemStatusNew 已删除）`)
}
