package ticket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 回归：GetTicketsByAssignee / GetSubtasks 曾把切片直接 common.Success，
// data 是裸数组 —— 调用方无法判断结果是否被截断，CLI/MCP 的列表整形也只能
// 降级为 shapingSkipped。列表契约要求一律 {items,total,page,pageSize,totalPages}。
//
// 信封字段用指针解码：键缺失（nil）与零值可区分，「裸数组」则根本无法解进
// struct，测试在修复前必然失败。

type listEnvelopeBody struct {
	Code int `json:"code"`
	Data struct {
		Items      []json.RawMessage `json:"items"`
		Total      *int64            `json:"total"`
		Page       *int              `json:"page"`
		PageSize   *int              `json:"pageSize"`
		TotalPages *int              `json:"totalPages"`
	} `json:"data"`
}

func decodeListEnvelope(t *testing.T, w *httptest.ResponseRecorder) listEnvelopeBody {
	t.Helper()
	var body listEnvelopeBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body),
		"data 必须是列表信封对象而不是裸数组: %s", w.Body.String())
	require.Equal(t, 0, body.Code)
	require.NotNil(t, body.Data.Total, "信封缺少 total")
	require.NotNil(t, body.Data.Page, "信封缺少 page")
	require.NotNil(t, body.Data.PageSize, "信封缺少 pageSize")
	require.NotNil(t, body.Data.TotalPages, "信封缺少 totalPages")
	return body
}

func TestGetSubtasks_ReturnsStandardListEnvelope(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	seedSubtask(t, repo, parentID, nil)
	seedSubtask(t, repo, parentID, nil)

	w := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/subtasks", parentID), nil, writeHeaders())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decodeListEnvelope(t, w)
	assert.Equal(t, *body.Data.Total, int64(len(body.Data.Items)), "total 必须与 items 数量一致")
	assert.Equal(t, 1, *body.Data.Page)
}

func TestGetSubtasks_EmptyResultIsNormalizedEnvelope(t *testing.T) {
	r, _ := newTestHarness(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets/999/subtasks", nil, writeHeaders())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decodeListEnvelope(t, w)
	assert.Empty(t, body.Data.Items)
	assert.Equal(t, int64(0), *body.Data.Total)
	// 空结果必须归一化：page=1、pageSize→默认值、totalPages=0，
	// 而不是 pageSize=0 与 totalPages=1 并存的自相矛盾信封。
	assert.Equal(t, 1, *body.Data.Page)
	assert.Equal(t, 20, *body.Data.PageSize)
	assert.Equal(t, 0, *body.Data.TotalPages)
}

func TestGetTicketsByAssignee_ReturnsStandardListEnvelope(t *testing.T) {
	r, repo := newTestHarness(t)
	assigneeID := 9
	seedTicket(t, repo, func(tk *Ticket) { tk.AssigneeID = &assigneeID })

	w := doJSON(t, r, http.MethodGet, "/api/v1/tickets/assignee/9", nil, writeHeaders())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decodeListEnvelope(t, w)
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, int64(1), *body.Data.Total)
	assert.Equal(t, 1, *body.Data.Page)
	assert.Equal(t, 1, *body.Data.PageSize)
	assert.Equal(t, 1, *body.Data.TotalPages)
}
