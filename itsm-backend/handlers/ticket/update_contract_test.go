package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeHeaders() map[string]string {
	return map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"}
}

// seedTicket 直接经 mock 仓储建单，避免 create 链路掩盖 update 的契约问题。
func seedTicket(t *testing.T, repo *mockRepository, mutate func(*Ticket)) int {
	t.Helper()
	tk, err := repo.Create(context.Background(), &CreateParams{
		Title:       "Original title",
		Description: "Original description",
		Priority:    "medium",
		Type:        "incident",
		RequesterID: 7,
	}, 1)
	require.NoError(t, err)
	if mutate != nil {
		repo.mu.Lock()
		mutate(tk)
		repo.mu.Unlock()
	}
	return tk.ID
}

func ticketPath(id int) string {
	return "/api/v1/tickets/" + strconv.Itoa(id)
}

// 更新词表必须与创建词表一致：urgent 曾只允许创建、禁止更新，
// 导致前端编辑紧急工单时整个表单提交被 400 拒绝。
func TestUpdate_PriorityVocabularyMatchesCreate(t *testing.T) {
	for _, priority := range []string{"low", "medium", "high", "critical", "urgent"} {
		t.Run(priority, func(t *testing.T) {
			r, repo := newTestHarness(t)
			id := seedTicket(t, repo, nil)

			w := doJSON(t, r, http.MethodPatch, ticketPath(id),
				map[string]string{"priority": priority}, writeHeaders())
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, priority, repo.tickets[id].Priority)
		})
	}

	t.Run("unknown priority rejected", func(t *testing.T) {
		r, repo := newTestHarness(t)
		id := seedTicket(t, repo, nil)

		w := doJSON(t, r, http.MethodPatch, ticketPath(id),
			map[string]string{"priority": "whenever"}, writeHeaders())
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

// 更新与创建的 description 长度口径必须一致：创建接受短描述，
// 更新曾额外要求 min=10，同一段文本创建成功、再保存却失败。
func TestUpdate_DescriptionLengthMatchesCreate(t *testing.T) {
	t.Run("short description accepted", func(t *testing.T) {
		r, repo := newTestHarness(t)
		id := seedTicket(t, repo, nil)

		w := doJSON(t, r, http.MethodPatch, ticketPath(id),
			map[string]string{"description": "tiny"}, writeHeaders())
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, "tiny", repo.tickets[id].Description)
	})

	t.Run("create accepts the same short description", func(t *testing.T) {
		r, _ := newTestHarness(t)
		w := doJSON(t, r, http.MethodPost, "/api/v1/tickets",
			dto.CreateTicketRequest{Title: "Short desc", Description: "tiny", Priority: "low"},
			writeHeaders())
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("over limit rejected on both paths", func(t *testing.T) {
		r, repo := newTestHarness(t)
		id := seedTicket(t, repo, nil)
		tooLong := strings.Repeat("x", 5001)

		w := doJSON(t, r, http.MethodPatch, ticketPath(id),
			map[string]string{"description": tooLong}, writeHeaders())
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

		w = doJSON(t, r, http.MethodPost, "/api/v1/tickets",
			dto.CreateTicketRequest{Title: "Too long", Description: tooLong, Priority: "low"},
			writeHeaders())
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

// type 曾在 DTO 里声明、校验通过，但 handler 从未下传，PATCH 静默无效。
func TestUpdate_TypeIsApplied(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)

	w := doJSON(t, r, http.MethodPatch, ticketPath(id),
		map[string]string{"type": "problem"}, writeHeaders())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var env struct {
		Data dto.TicketResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "problem", env.Data.Type)
	assert.Equal(t, "problem", repo.tickets[id].Type)
}

// 前端编辑表单会回传整个表单（含未变化的 status）。未变化字段必须从落库参数
// 中摘掉；否则仓储的 SetStatus 会把 resolved_at/closed_at 重置为此刻。
func TestUpdate_OnlyChangedFieldsReachRepository(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)

	w := doJSON(t, r, http.MethodPatch, ticketPath(id), map[string]interface{}{
		"title":    "New title",
		"status":   "new",
		"priority": "medium",
		"type":     "incident",
		"version":  1,
	}, writeHeaders())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	p := repo.lastUpdateParams
	require.NotNil(t, p)
	assert.NotNil(t, p.Title)
	assert.Nil(t, p.Status, "回传的当前状态不得落库")
	assert.Nil(t, p.Priority)
	assert.Nil(t, p.Type)
}

func TestUpdate_UnchangedPayloadDoesNotWrite(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)
	original := *repo.tickets[id]

	w := doJSON(t, r, http.MethodPatch, ticketPath(id), map[string]interface{}{
		"title":       original.Title,
		"description": original.Description,
		"status":      original.Status,
		"priority":    original.Priority,
		"type":        original.Type,
		"version":     original.Version,
	}, writeHeaders())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, 0, repo.updateCalls, "无变化更新不得触达仓储")
	assert.Equal(t, original.Version, repo.tickets[id].Version)
	assert.Equal(t, original.UpdatedAt, repo.tickets[id].UpdatedAt)

	var env struct {
		Data dto.TicketResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, original.Version, env.Data.Version)
}

// 空更新也必须遵守乐观锁：过期版本不能借「没有变化」绕过 409。
func TestUpdate_StaleVersionNoOpStillConflicts(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)
	original := *repo.tickets[id]

	w := doJSON(t, r, http.MethodPatch, ticketPath(id), map[string]interface{}{
		"title":   original.Title,
		"version": original.Version + 99,
	}, writeHeaders())
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	assert.Equal(t, 0, repo.updateCalls)
}

// 本端点没有标签名解析能力；静默丢弃会让调用方以为标签已写入。
func TestUpdate_TagsAreRejectedNotSilentlyDropped(t *testing.T) {
	r, repo := newTestHarness(t)
	id := seedTicket(t, repo, nil)

	w := doJSON(t, r, http.MethodPatch, ticketPath(id),
		map[string]interface{}{"tags": []string{"vpn"}}, writeHeaders())
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "relations")
	assert.Equal(t, 0, repo.updateCalls)
}

// 列表过滤参数曾完全不校验：status=bogus 返回 200 + 空列表，
// 调用方无法区分「没有此类工单」和「参数写错了」。
func TestList_RejectsUnknownQueryVocabulary(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"unknown status", "status=bogus", http.StatusBadRequest},
		{"unknown priority", "priority=bogus", http.StatusBadRequest},
		{"urgent priority is valid", "priority=urgent", http.StatusOK},
		{"unknown type", "type=bogus", http.StatusBadRequest},
		{"unknown sortBy", "sortBy=bogus", http.StatusBadRequest},
		{"unknown sortOrder", "sortOrder=bogus", http.StatusBadRequest},
		{"repository sort whitelist accepted", "sortBy=ticket_number&sortOrder=asc", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newTestHarness(t)
			w := doJSON(t, r, http.MethodGet, "/api/v1/tickets?"+tc.query, nil, writeHeaders())
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}
