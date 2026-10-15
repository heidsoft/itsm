package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"itsm-backend/domain/role"
	"itsm-backend/internal/datascope"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestHandler_ListTickets_DataScopeMatchesAdminLikeVocabulary 锁定生产路由
// GET /api/v1/tickets 下发的行级数据范围档位与 domain/role.IsAdminLike 完全一致。
//
// handlers/ticket 曾自行维护 {admin, super_admin, manager}，比单一事实源少了
// sysadmin 与 audit_admin：这两个角色的工单列表被收窄到本人创建/受理，
// 而删除走 datascope 的完整档位，出现「列表读不到却能删」，
// 违反 datascope.CanWriteResource 声明的「写权限 ⊆ 读权限」不变量。
func TestHandler_ListTickets_DataScopeMatchesAdminLikeVocabulary(t *testing.T) {
	r, repo := newTestHarness(t)
	seedTickets(t, r, 3) // 全部由用户 7 创建，与下面每个 actor 无关

	for _, code := range role.All {
		code := code
		t.Run(code, func(t *testing.T) {
			w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?pageSize=100", nil,
				map[string]string{
					"X-Test-TenantID": "1",
					"X-Test-UserID":   "9001",
					"X-Test-Role":     code,
				})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			var got listEnvelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

			repo.mu.Lock()
			scope := repo.lastListDataScope
			repo.mu.Unlock()

			if role.IsAdminLike(code) {
				assert.Equal(t, datascope.DataScopeAll, scope,
					"%s 应下传全量档，否则管理面读视野被收窄", code)
				assert.Equal(t, 3, got.Data.Total, "%s 应看到租户全部 3 张工单", code)
				return
			}

			assert.Equal(t, datascope.DataScopeOwnedOrAssigned, scope,
				"%s 不是管理角色，不得获得全租户工单视野", code)
			assert.Equal(t, 0, got.Data.Total, "%s 不应看到他人工单", code)
		})
	}
}

// TestService_ListAndBatchDeleteShareOneRoleVocabulary 用同一批 actor 身份分别驱动
// 读路径（Service.List）与写路径（Service.BatchDelete），断言两者对「谁是管理角色」
// 的判定逐角色一致，且都以 domain/role.IsAdminLike 为准。
//
// 只要任一入口自己另写一份枚举，这个用例就会失败；这是防止词表再次分叉的结构锁。
func TestService_ListAndBatchDeleteShareOneRoleVocabulary(t *testing.T) {
	ctx := context.Background()

	for _, code := range role.All {
		code := code
		t.Run(code, func(t *testing.T) {
			repo := newMockRepository()
			foreign, err := repo.Create(ctx, &CreateParams{Title: "foreign", Priority: "low", RequesterID: 7}, 1)
			require.NoError(t, err)

			svc := NewService(repo, nil, nil, zap.NewNop().Sugar())

			_, _, err = svc.List(ctx, 1, 1, 10, nil, 9001, code)
			require.NoError(t, err)
			repo.mu.Lock()
			readScope := repo.lastListDataScope
			repo.mu.Unlock()

			deleteErr := svc.BatchDelete(ctx, []int{foreign.ID}, 1, 9001, code)

			if role.IsAdminLike(code) {
				assert.Equal(t, datascope.DataScopeAll, readScope, "%s 读视野应为全量", code)
				assert.NoError(t, deleteErr, "%s 为管理角色，删除不应被行级归属拦下", code)
				return
			}

			assert.Equal(t, datascope.DataScopeOwnedOrAssigned, readScope,
				"%s 非管理角色却拿到全租户读视野", code)
			assert.Error(t, deleteErr, "%s 删除他人工单必须被拒", code)
		})
	}
}
