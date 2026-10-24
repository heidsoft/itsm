package ticket

import (
	"encoding/json"
	"net/http"
	"testing"

	"itsm-backend/domain/role"
	"itsm-backend/internal/datascope"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandler_SearchAndOverdue_ShareListDataScope 锁定 /tickets/search 与
// /tickets/overdue 必须沿用与 /tickets 相同的行级数据范围。
//
// 这两个端点是工单列表的「过滤视图」而非独立读取通道，但此前 Search/FindOverdue
// 都不接 currentUserID/currentRole：DataScope 零值恰好是 DataScopeAll（无过滤），
// 于是任何非管理角色都能用 /tickets/search?q=x 或 /tickets/overdue 读到全租户
// 工单，与 /tickets 列表把普通角色收窄为「本人创建或受理」的口径直接矛盾。
// 本用例对每个内置角色逐一断言：管理角色拿到全量档，其余角色拿到收窄档且确实
// 看不到他人工单。mock 若重新忽略 ds/currentUserID，这里会立即失败。
func TestHandler_SearchAndOverdue_ShareListDataScope(t *testing.T) {
	for _, code := range role.All {
		code := code
		t.Run(code, func(t *testing.T) {
			// 每个子用例独立 harness：seed 出来的 3 张工单由用户 7 创建，
			// 与下面用 9001 身份访问的 actor 无关。
			r, repo := newTestHarness(t)
			seedTickets(t, r, 3)

			repo.mu.Lock()
			repo.overdueIDs = []int{1, 2, 3}
			repo.mu.Unlock()

			headers := map[string]string{
				"X-Test-TenantID": "1",
				"X-Test-UserID":   "9001",
				"X-Test-Role":     code,
			}

			// ---- /tickets/search ----
			w := doJSON(t, r, http.MethodGet, "/api/v1/tickets/search?q=seed&pageSize=100", nil, headers)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			var searchEnv listEnvelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &searchEnv))

			repo.mu.Lock()
			searchScope := repo.lastSearchDataScope
			searchUser := repo.lastSearchUserID
			repo.mu.Unlock()

			require.Equal(t, 9001, searchUser, "handler 必须把当前用户 ID 下传到仓储")

			// ---- /tickets/overdue ----
			w2 := doJSON(t, r, http.MethodGet, "/api/v1/tickets/overdue?pageSize=100", nil, headers)
			require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

			var overdueEnv listEnvelope
			require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &overdueEnv))

			repo.mu.Lock()
			overdueScope := repo.lastOverdueDataScope
			overdueUser := repo.lastOverdueUserID
			repo.mu.Unlock()

			require.Equal(t, 9001, overdueUser, "handler 必须把当前用户 ID 下传到仓储")

			if role.IsAdminLike(code) {
				assert.Equal(t, datascope.DataScopeAll, searchScope, "%s 搜索应下传全量档", code)
				assert.Equal(t, datascope.DataScopeAll, overdueScope, "%s 逾期列表应下传全量档", code)
				assert.Equal(t, 3, searchEnv.Data.Total, "%s 应搜到租户全部 3 张工单", code)
				assert.Equal(t, 3, overdueEnv.Data.Total, "%s 应看到租户全部 3 张逾期工单", code)
				return
			}

			assert.Equal(t, datascope.DataScopeOwnedOrAssigned, searchScope,
				"%s 不是管理角色，搜索不得获得全租户视野", code)
			assert.Equal(t, datascope.DataScopeOwnedOrAssigned, overdueScope,
				"%s 不是管理角色，逾期列表不得获得全租户视野", code)
			assert.Equal(t, 0, searchEnv.Data.Total, "%s 不应搜到他人工单", code)
			assert.Equal(t, 0, overdueEnv.Data.Total, "%s 不应看到他人逾期工单", code)
		})
	}
}

// TestHandler_SearchAndOverdue_FailClosedWithoutIdentity 身份缺失时的兜底：
// 非全量档 + 无 user_id 必须收窄为空集，而不是退回全租户可读。
// datascope 的不变量是「身份缺失 fail-closed」，搜索/逾期两处不得例外。
func TestHandler_SearchAndOverdue_FailClosedWithoutIdentity(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 3)

	repo.mu.Lock()
	repo.overdueIDs = []int{1, 2, 3}
	repo.mu.Unlock()

	// 不带 X-Test-UserID：user_id 解析为 0。
	headers := map[string]string{"X-Test-TenantID": "1"}

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets/search?q=seed&pageSize=100", nil, headers)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var searchEnv listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &searchEnv))
	assert.Equal(t, 0, searchEnv.Data.Total, "身份缺失时搜索必须返回空，不得退回全量")

	w2 := doJSON(t, r, http.MethodGet, "/api/v1/tickets/overdue?pageSize=100", nil, headers)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	var overdueEnv listEnvelope
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &overdueEnv))
	assert.Equal(t, 0, overdueEnv.Data.Total, "身份缺失时逾期列表必须返回空，不得退回全量")
}
