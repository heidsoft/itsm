package ticket

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubtask(t *testing.T, repo *mockRepository, parentID int, mutate func(*Ticket)) int {
	t.Helper()
	child, err := repo.Create(context.Background(), &CreateParams{
		Title:          "Child task",
		Description:    "Child description",
		Type:           "incident",
		Priority:       "low",
		RequesterID:    7,
		ParentTicketID: &parentID,
	}, 1)
	require.NoError(t, err)
	if mutate != nil {
		repo.mu.Lock()
		mutate(child)
		repo.mu.Unlock()
	}
	return child.ID
}

func subtaskPath(parentID, subtaskID int) string {
	return "/api/v1/tickets/" + strconv.Itoa(parentID) + "/subtasks/" + strconv.Itoa(subtaskID)
}

// 子任务更新曾只映射 title/description/priority，status/assignee/category/resolution
// 通过 DTO 校验后静默丢弃，调用方看到 200 却什么都没改。
func TestUpdateSubtask_AppliesAllWritableFields(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	subtaskID := seedSubtask(t, repo, parentID, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, subtaskID), map[string]interface{}{
		"status":     "in_progress",
		"priority":   "high",
		"type":       "ticket",
		"assigneeId": 9,
		"categoryId": 3,
		"resolution": "handled",
		"version":    1,
	}, writeHeaders())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	child := repo.tickets[subtaskID]
	assert.Equal(t, "in_progress", child.Status)
	assert.Equal(t, "high", child.Priority)
	assert.Equal(t, "ticket", child.Type)
	require.NotNil(t, child.AssigneeID)
	assert.Equal(t, 9, *child.AssigneeID)
	require.NotNil(t, child.CategoryID)
	assert.Equal(t, 3, *child.CategoryID)
	require.NotNil(t, child.Resolution)
	assert.Equal(t, "handled", *child.Resolution)
}

// 子任务的乐观锁冲突曾掉进 FailWithErr 兜底变成 500；必须与工单更新同口径返回 409。
func TestUpdateSubtask_VersionConflictReturns409(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	subtaskID := seedSubtask(t, repo, parentID, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, subtaskID), map[string]interface{}{
		"title":   "stale write",
		"version": 99,
	}, writeHeaders())

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// 非法状态迁移属于业务拒绝（409），不是 500。
func TestUpdateSubtask_InvalidTransitionReturns409(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	subtaskID := seedSubtask(t, repo, parentID, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, subtaskID), map[string]interface{}{
		"status": "closed",
	}, writeHeaders())

	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// 标签与工单更新保持同一语义：本端点没有标签解析能力，必须显式拒绝。
func TestUpdateSubtask_RejectsTags(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	subtaskID := seedSubtask(t, repo, parentID, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, subtaskID), map[string]interface{}{
		"tags": []string{"network"},
	}, writeHeaders())

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "relations")
}

func TestUpdateSubtask_RejectsForeignChild(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)
	otherParent := seedTicket(t, repo, nil)
	subtaskID := seedSubtask(t, repo, otherParent, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, subtaskID), map[string]interface{}{
		"title": "hijack",
	}, writeHeaders())

	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestUpdateSubtask_NotFoundReturns404(t *testing.T) {
	r, repo := newTestHarness(t)
	parentID := seedTicket(t, repo, nil)

	w := doJSON(t, r, http.MethodPatch, subtaskPath(parentID, 9999), map[string]interface{}{
		"title": "ghost",
	}, writeHeaders())

	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

// 读取故障不得被压成「工单不存在」的 404：真实数据库故障应返回 500，
// 否则监控只能看到客户端错误。
func TestGetTicket_InternalErrorIsNotMaskedAsNotFound(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)
	repo.mu.Lock()
	repo.getErr = errors.New("connection reset by peer")
	repo.mu.Unlock()

	w := doJSON(t, r, http.MethodGet, ticketPath(id), nil, writeHeaders())

	assert.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), strconv.Itoa(common.InternalErrorCode))
}
