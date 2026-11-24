package bpmn

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P2-06 守卫：任务列表 handler 的页长/页码只有
// common.GetPaginationFromQuery 一个所有者（缺省 1/20、只采纳 [1,100]）。
// 手写 <=0 兜底不设上界：pageSize=100000 会原样下传给 service 的 SQL LIMIT，
// 一次写错参数的请求就可能把整租户任务读穿。
// 断言落在 fake 记录的 listRequest 上——那正是下传到 service 的值。
func TestListTasksPaginationHasUpperBound(t *testing.T) {
	cases := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"缺省回落1/20", "", 1, 20},
		{"超上界页长回落缺省", "?pageSize=100000", 1, 20},
		{"零与负页参数回落缺省", "?page=0&pageSize=-5", 1, 20},
		{"界内页参数原样透传", "?page=3&pageSize=50", 3, 50},
		{"恰好上界原样透传", "?pageSize=100", 1, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, taskService := newTaskListHandlerTestRouter(t)
			w := doRequest(t, r, http.MethodGet, "/tasks"+tc.query, nil)
			require.Equal(t, http.StatusOK, w.Code)
			require.NotNil(t, taskService.listRequest)
			assert.Equal(t, tc.wantPage, taskService.listRequest.Page)
			assert.Equal(t, tc.wantPageSize, taskService.listRequest.PageSize)
		})
	}
}

// ListAllTasks（admin 面）同口径：AllTasks 分支不得漏掉归一化。
func TestListAllTasksPaginationHasUpperBound(t *testing.T) {
	r, taskService := newTaskListHandlerTestRouter(t)
	w := doRequest(t, r, http.MethodGet, "/tasks/all?pageSize=100000&page=0", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, taskService.listRequest)
	assert.Equal(t, 1, taskService.listRequest.Page)
	assert.Equal(t, 20, taskService.listRequest.PageSize)
}
