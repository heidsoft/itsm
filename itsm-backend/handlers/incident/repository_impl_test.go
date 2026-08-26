package incident

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestService_GetStats_TableDriven 锁住 P0-2 (Bug 4) 契约：
//   - 注入 N 条已解决事件后，avgResolutionTime 必须 > 0（之前硬编码 0）
//   - service.GetStats → repo.GetStats 走仓储，handler 不再直接访问 ent.Client（Bug 3）
//   - 响应字段为 camelCase，与 dto.IncidentMetrics 对齐
func TestService_GetStats_TableDriven(t *testing.T) {
	cases := []struct {
		name           string
		seed           func(repo *mockRepository)
		tenantID       int
		wantTotal      int
		wantOpen       int
		wantCritical   int
		wantMajor      int
		wantResolved   int
		wantAvgMinGt0  bool
		wantAvgEqExact int
	}{
		{
			name:     "空集合：所有计数为 0，avg=0",
			tenantID: 1,
		},
		{
			name: "3 条已解决事件各 30 分钟：avg=30",
			seed: func(repo *mockRepository) {
				repo.nextID = 0
				for i := 0; i < 3; i++ {
					now := time.Now()
					id := i + 1
					repo.incidents[id] = &Incident{
						ID:         id,
						TenantID:   1,
						Status:     "resolved",
						Priority:   "low",
						CreatedAt:  now.Add(-30 * time.Minute),
						ResolvedAt: &now,
					}
				}
			},
			tenantID:       1,
			wantTotal:      3,
			wantResolved:   3,
			wantAvgMinGt0:  true,
			wantAvgEqExact: 30,
		},
		{
			name: "open + critical + high 混合统计",
			seed: func(repo *mockRepository) {
				repo.incidents[1] = &Incident{ID: 1, TenantID: 1, Status: "open", Priority: "critical"}
				repo.incidents[2] = &Incident{ID: 2, TenantID: 1, Status: "in_progress", Priority: "high"}
				repo.incidents[3] = &Incident{ID: 3, TenantID: 1, Status: "resolved", Priority: "medium"}
			},
			tenantID:     1,
			wantTotal:    3,
			wantOpen:     2,
			wantCritical: 1,
			wantMajor:    1,
			wantResolved: 1,
		},
		{
			name: "跨租户隔离：tenant=2 的事件不应计入 tenant=1 的统计",
			seed: func(repo *mockRepository) {
				repo.incidents[1] = &Incident{ID: 1, TenantID: 2, Status: "resolved", Priority: "critical"}
			},
			tenantID:  1,
			wantTotal: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			if tc.seed != nil {
				tc.seed(repo)
			}
			svc := NewService(repo, nil)

			stats, err := svc.GetStats(t.Context(), tc.tenantID)
			assert.NoError(t, err)
			if !assert.NotNil(t, stats) {
				return
			}
			assert.Equal(t, tc.wantTotal, stats.TotalIncidents, "TotalIncidents")
			assert.Equal(t, tc.wantOpen, stats.OpenIncidents, "OpenIncidents")
			assert.Equal(t, tc.wantCritical, stats.CriticalIncidents, "CriticalIncidents")
			assert.Equal(t, tc.wantMajor, stats.MajorIncidents, "MajorIncidents")
			assert.Equal(t, tc.wantResolved, stats.ResolvedIncidents, "ResolvedIncidents")

			if tc.wantAvgMinGt0 {
				assert.Greater(t, stats.AvgResolutionTime, 0,
					"P0-2 验收：注入 resolved 事件后 avgResolutionTime 必须 > 0")
			}
			if tc.wantAvgEqExact > 0 {
				assert.Equal(t, tc.wantAvgEqExact, stats.AvgResolutionTime,
					"P0-2 验收：avgResolutionTime 应等于 (resolved_at - created_at) 的分钟均值")
			}

			// camelCase 字段必须出现在响应里（Bug 4 验收：snake_case 已切换）
			js, mErr := json.Marshal(stats)
			assert.NoError(t, mErr)
			body := string(js)
			assert.Contains(t, body, `"totalIncidents"`, "P0-3 验收：snake_case → camelCase")
			assert.Contains(t, body, `"openIncidents"`)
			assert.Contains(t, body, `"criticalIncidents"`)
			assert.Contains(t, body, `"majorIncidents"`)
			assert.Contains(t, body, `"resolvedIncidents"`)
			assert.Contains(t, body, `"avgResolutionTime"`)
			assert.NotContains(t, body, `"total_incidents"`, "残留 snake_case 字段")
			assert.NotContains(t, body, `"avg_resolution_time"`, "残留 snake_case 字段")
		})
	}
}

// TestService_GetStats_NoDirectEntAccess 验证 service.GetStats 必须委托 repo.GetStats。
// Bug 3 验收：handler/service 不再直接拿 ent.Client。
func TestService_GetStats_NoDirectEntAccess(t *testing.T) {
	var calls int
	repo := newMockRepository()
	repo.statsCallCount = &calls
	svc := NewService(repo, nil)

	_, err := svc.GetStats(t.Context(), 42)
	assert.NoError(t, err)
	assert.Equal(t, 1, calls, "service.GetStats 必须委托 repo.GetStats 一次")
}

// TestAutoPriorityByKeyword_DuplicateCritical 锁住 Bug 1 修复。
// 修复前：critical 关键字在数组中出现两次，containsAny 仍返回 true，函数语义
// 不变，但数组拷贝维护期容易产生误判。修复后数组中 critical 仅出现一次。
// 通过 ensure 函数 + reflect 读取源码字符串非常脆弱，这里直接走行为验证：
// 输入包含 "critical" 时返回 "critical"，且不被重复计数影响。
func TestAutoPriorityByKeyword_DuplicateCritical(t *testing.T) {
	cases := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{"仅 critical 关键词", "critical incident", "", "critical"},
		{"critical + production 都命中", "Production outage", "critical database", "critical"},
		{"down 优先级不高于 critical", "down", "outage", "critical"},
		{"宕机中文命中", "宕机了", "", "critical"},
		{"未命中则 high", "service is slow", "", "medium"},
		{"未命中且无关键词则返回 medium", "issue in module", "", "medium"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := autoPriorityByKeyword(tc.title, tc.description)
			assert.Equal(t, tc.want, got,
				"Bug 1 验收：autoPriorityByKeyword 必须正确分流，与 critical 重复出现无关")
		})
	}
}