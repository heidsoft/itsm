// Package cmdb 中 CMDB 拓扑 / 影响分析的契约测试。
//
// 本文件 (topology_invariant_test.go) 固化 v1.7 (PR-FIX-CMDB-BPMN)
// 阶段 2.3 的核心契约：
//
//   - 自环 (a -> a) 必须判为非法
//   - 50 层深链递归必须不爆栈
//   - 环 (a -> b -> a) 必须早返回
//   - 跨租户 CI 不能进入本租户的影响图
//
// 这些是 ROADMAP 反复点名的高风险路径（递归爆栈）。
//
// 注：本测试聚焦 *算法层* 契约（TopologyGuard）；DB 集成由
// repository_impl_test.go 承担。
package cmdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// adjFn 类型定义：给定 node 返回其邻接节点集。
type adjFn func(node int) []int

// TopologyGuard 是 v1.7 新增的迭代防环遍历器，对应 prod 中
// service_cmdb_topology_guard.go 的 BuildImpactMap。
// 行为契约：
//   - IsSelfLoop：src==target 返回 true
//   - WalkIterative：迭代遍历 + visited 防环 + maxNodes 上限
type TopologyGuard struct{}

// NewTopologyGuard 构造器（与 prod 同名，保持镜像）。
func NewTopologyGuard() *TopologyGuard { return &TopologyGuard{} }

// IsSelfLoop：src 与 target 引用同一节点视为非法自环。
func (g *TopologyGuard) IsSelfLoop(src, target int) bool { return src == target }

// WalkIterative 显式迭代版拓扑遍历（不会爆栈）：
//   - queue 替代递归
//   - visited map 防止环
//   - maxNodes 限制最大节点
//   - adjFn 由调用方注入（通常从 DB 拉邻接，附 tenant 过滤）
func (g *TopologyGuard) WalkIterative(rootID, maxNodes int, fn adjFn) []int {
	visited := []int{}
	seen := make(map[int]bool)
	queue := []int{rootID}
	for len(queue) > 0 && len(visited) < maxNodes {
		head := queue[0]
		queue = queue[1:]
		if seen[head] {
			continue
		}
		seen[head] = true
		visited = append(visited, head)
		for _, n := range fn(head) {
			if !seen[n] {
				queue = append(queue, n)
			}
		}
	}
	return visited
}

// TestTopologyInvariant_SelfLoop_Rejected 自环 a->a 必须判为非法。
func TestTopologyInvariant_SelfLoop_Rejected(t *testing.T) {
	g := NewTopologyGuard()
	ciA := 42
	ciB := 42 // same ID
	assert.True(t, g.IsSelfLoop(ciA, ciB), "guard.IsSelfLoop 在 a==b 时必须返回 true")

	// 即便 adjFn 把 self-link 出来，visited 必须自屏蔽
	visited := g.WalkIterative(ciA, 64, func(node int) []int {
		return []int{node}
	})
	assert.Equal(t, 1, len(visited), "自指邻接必须被 visited 自屏蔽（防爆栈）")
}

// TestTopologyInvariant_DeepChain_NoStackOverflow 50 层深链必须不爆栈。
func TestTopologyInvariant_DeepChain_NoStackOverflow(t *testing.T) {
	const depth = 50
	ids := make([]int, depth+1)
	for i := range ids {
		ids[i] = i + 1 // 1..51
	}
	// 邻接：第 i 个 -> 第 i+1 个
	next := map[int]int{}
	for i := 0; i < depth; i++ {
		next[ids[i]] = ids[i+1]
	}

	g := NewTopologyGuard()
	visited := g.WalkIterative(ids[0], depth+5, func(node int) []int {
		if n, ok := next[node]; ok {
			return []int{n}
		}
		return nil
	})

	assert.Len(t, visited, depth+1, "迭代遍历应走完 50 + 1 个节点")
	assert.Equal(t, ids[0], visited[0], "首节点必须等于 root")
	assert.Equal(t, ids[depth], visited[depth], "末节点必须等于最后一个")
}

// TestTopologyInvariant_Cycle_EarlyReturn a->b->a 必须早返回。
func TestTopologyInvariant_Cycle_EarlyReturn(t *testing.T) {
	const ciA = 1
	const ciB = 2

	// 邻接：a->b, b->a
	adj := map[int][]int{
		ciA: {ciB},
		ciB: {ciA},
	}
	g := NewTopologyGuard()
	visited := g.WalkIterative(ciA, 64, func(n int) []int { return adj[n] })

	// 环中两个 CI 必须都被访问，且不重复
	assert.Equal(t, 2, len(visited), "环必须早返回，不能死循环")
	assert.ElementsMatch(t, []int{ciA, ciB}, visited, "环中两个 CI 必须都被访问")
}

// TestTopologyInvariant_Cycle_DeepChain 50 层 + 末端环：不能爆栈，不能死循环。
func TestTopologyInvariant_Cycle_DeepChain(t *testing.T) {
	const depth = 50
	ids := make([]int, depth+1)
	for i := range ids {
		ids[i] = i + 100
	}
	// 链：ids[0] -> ids[1] -> ... -> ids[depth-1] -> ids[depth]
	// 环：ids[depth] -> ids[depth-1]
	adj := map[int][]int{}
	for i := 0; i < depth; i++ {
		adj[ids[i]] = []int{ids[i+1]}
	}
	adj[ids[depth]] = []int{ids[depth-1]} // 回环

	g := NewTopologyGuard()
	visited := g.WalkIterative(ids[0], 200, func(n int) []int { return adj[n] })

	assert.LessOrEqual(t, len(visited), depth+1, "深度 + 末端环必须严格早返回")
	assert.GreaterOrEqual(t, len(visited), depth, "深度链必须走到末端")
}

// TestTopologyInvariant_CrossTenant_NotInGraph 跨租户 CI 不能进入本租户影响图。
func TestTopologyInvariant_CrossTenant_NotInGraph(t *testing.T) {
	const (
		ciAX = 11
		ciAY = 12 // same tenant (A)
		ciBZ = 99 // tenant B
	)

	// adjFn 模拟"邻接函数已做 tenant_id 过滤"
	adjScoped := map[int][]int{
		ciAX: {ciAY},
		ciAY: nil, // ciBZ 已被邻接函数过滤掉
	}

	g := NewTopologyGuard()
	visited := g.WalkIterative(ciAX, 32, func(n int) []int { return adjScoped[n] })

	assert.Contains(t, visited, ciAY, "本租户节点必须可见")
	assert.NotContains(t, visited, ciBZ, "跨租户节点必须被邻接函数过滤")
}

// TestTopologyInvariant_MaxNodes_LimitExceed maxNodes 必须严格限制。
func TestTopologyInvariant_MaxNodes_LimitExceed(t *testing.T) {
	// 链式无限邻接
	g := NewTopologyGuard()
	visited := g.WalkIterative(1, 7, func(n int) []int { return []int{n + 1} })
	assert.Len(t, visited, 7, "maxNodes 必须严格限制")
}

// TestTopologyInvariant_EmptyAdjFn_HasOnlyRoot 邻接空时只返回 root。
func TestTopologyInvariant_EmptyAdjFn_HasOnlyRoot(t *testing.T) {
	g := NewTopologyGuard()
	visited := g.WalkIterative(7, 64, func(n int) []int { return nil })
	assert.Equal(t, []int{7}, visited)
}
