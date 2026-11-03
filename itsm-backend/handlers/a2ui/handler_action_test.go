package a2ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/handlers/ticket"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTicketCreator 记录 A2UI submit 传给工单领域的参数，
// 用于断言 tenant/user 只来自认证上下文、表单字段已完成归一。
type fakeTicketCreator struct {
	calls    int
	tenantID int
	params   *ticket.CreateParams
	result   *ticket.Ticket
	err      error
}

func (f *fakeTicketCreator) Create(_ context.Context, tenantID int, params *ticket.CreateParams) (*ticket.Ticket, error) {
	f.calls++
	f.tenantID = tenantID
	f.params = params
	return f.result, f.err
}

type actionEnvelope struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
	Success  bool     `json:"success"`
}

// newActionRouter 挂载真实 HandleAction，并通过中间件模拟认证上下文。
// 租户/用户只从 header 注入到 gin context，请求体永远无法影响身份。
func newActionRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-Tenant"); raw != "" {
			if tid, err := strconv.Atoi(raw); err == nil {
				c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tid})
				c.Set("tenant_id", tid)
			}
		}
		if raw := c.GetHeader("X-Test-User"); raw != "" {
			if uid, err := strconv.Atoi(raw); err == nil {
				c.Set("user_id", uid)
			}
		}
		c.Next()
	})
	r.POST("/api/v1/a2ui/ticket/action", h.HandleAction)
	return r
}

func doAction(t *testing.T, r *gin.Engine, tenant, user string, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/a2ui/ticket/action", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func submitBody() map[string]interface{} {
	return map[string]interface{}{
		"action":    "submit",
		"surfaceId": "ticket-create",
		"context": map[string]interface{}{
			"data": map[string]interface{}{
				"title":       " 打印机无法打印 ",
				"type":        "hardware",
				"priority":    "high",
				"description": "办公室打印机离线",
			},
		},
	}
}

func TestHandleActionSubmitCreatesTicket(t *testing.T) {
	creator := &fakeTicketCreator{
		result: &ticket.Ticket{ID: 1001, TicketNumber: "TKT-202610-001001"},
	}
	h := NewHandler(nil, creator, nil)
	r := newActionRouter(h)

	w := doAction(t, r, "7", "42", submitBody())
	require.Equal(t, http.StatusOK, w.Code)

	var resp actionEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.True(t, resp.Success)
	require.Len(t, resp.Messages, 2)

	// 身份只来自认证上下文，不接受请求体伪造。
	require.Equal(t, 1, creator.calls)
	assert.Equal(t, 7, creator.tenantID)
	assert.Equal(t, 42, creator.params.RequesterID)
	assert.Equal(t, "打印机无法打印", creator.params.Title)
	assert.Equal(t, "service_request", creator.params.Type)
	assert.Equal(t, "high", creator.params.Priority)
	assert.Equal(t, "hardware", creator.params.FormFields["aiTicketType"])

	var dataMsg service.A2UIDataModelUpdate
	require.NoError(t, json.Unmarshal([]byte(resp.Messages[0]), &dataMsg))
	assert.Equal(t, "ticket-create", dataMsg.DataModelUpdate.SurfaceID)
	assert.Equal(t, "created", contentValue(dataMsg, "status"))
	assert.Equal(t, "TKT-202610-001001", contentValue(dataMsg, "ticketNumber"))
	require.NotNil(t, contentNumber(dataMsg, "ticketId"))
	assert.Equal(t, 1001, *contentNumber(dataMsg, "ticketId"))

	var deleteMsg service.A2UIDeleteSurface
	require.NoError(t, json.Unmarshal([]byte(resp.Messages[1]), &deleteMsg))
	assert.Equal(t, "ticket-create", deleteMsg.DeleteSurface.SurfaceID)
}

func contentValue(msg service.A2UIDataModelUpdate, key string) string {
	for _, c := range msg.DataModelUpdate.Contents {
		if c.Key == key {
			return c.ValueString
		}
	}
	return ""
}

func contentNumber(msg service.A2UIDataModelUpdate, key string) *int {
	for _, c := range msg.DataModelUpdate.Contents {
		if c.Key == key {
			return c.ValueNumber
		}
	}
	return nil
}

func TestHandleActionSubmitFailsClosedWithoutTenant(t *testing.T) {
	creator := &fakeTicketCreator{result: &ticket.Ticket{ID: 1}}
	h := NewHandler(nil, creator, nil)
	r := newActionRouter(h)

	w := doAction(t, r, "", "42", submitBody())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, 0, creator.calls, "缺失租户上下文不得调用工单创建")
}

func TestHandleActionSubmitRejectsInvalidForm(t *testing.T) {
	cases := []struct {
		name        string
		body        map[string]interface{}
		wantMessage string
	}{
		{
			name: "空标题",
			body: map[string]interface{}{
				"action": "submit", "surfaceId": "s1",
				"context": map[string]interface{}{"data": map[string]interface{}{"title": "   "}},
			},
			wantMessage: "工单标题不能为空",
		},
		{
			name: "非法优先级",
			body: map[string]interface{}{
				"action": "submit", "surfaceId": "s1",
				"context": map[string]interface{}{"data": map[string]interface{}{"title": "标题", "priority": "p0"}},
			},
			wantMessage: "无效的工单优先级",
		},
		{
			name:        "缺少表单数据",
			body:        map[string]interface{}{"action": "submit", "surfaceId": "s1"},
			wantMessage: "缺少工单表单数据",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			creator := &fakeTicketCreator{result: &ticket.Ticket{ID: 1}}
			h := NewHandler(nil, creator, nil)
			r := newActionRouter(h)

			w := doAction(t, r, "7", "42", tc.body)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, common.ParamErrorCode, decodeCode(t, w.Body.Bytes()))
			assert.Equal(t, tc.wantMessage, decodeMessage(t, w.Body.Bytes()), "只回声明过的表单文案，不透出内部错误串")
			assert.Equal(t, 0, creator.calls)
		})
	}
}

// 未登记的表单错误必须落到固定文案：响应里不得出现原始错误串（棘轮锁的是调用形态，
// 这里锁行为）。
func TestA2UIPublicFormErrorHidesUnlistedErrors(t *testing.T) {
	assert.Equal(t, "工单标题不能为空", a2uiPublicFormError(errA2UITitleRequired))
	assert.Equal(t, "工单表单数据不完整", a2uiPublicFormError(errors.New("sql: driver closed connection")))
}

func TestHandleActionRejectsUnknownAction(t *testing.T) {
	creator := &fakeTicketCreator{result: &ticket.Ticket{ID: 1}}
	h := NewHandler(nil, creator, nil)
	r := newActionRouter(h)

	w := doAction(t, r, "7", "42", map[string]interface{}{"action": "cancel", "surfaceId": "s1"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, decodeCode(t, w.Body.Bytes()))
	assert.Equal(t, 0, creator.calls)
}

func TestHandleActionSubmitMapsDomainError(t *testing.T) {
	creator := &fakeTicketCreator{
		err: common.NewBadRequestError("title不能为空", nil),
	}
	h := NewHandler(nil, creator, nil)
	r := newActionRouter(h)

	w := doAction(t, r, "7", "42", submitBody())
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 4000, decodeCode(t, w.Body.Bytes()))
}

func TestHandleActionSubmitHidesUnclassifiedError(t *testing.T) {
	creator := &fakeTicketCreator{
		err: errors.New("driver detail: connection to secret internal host failed"),
	}
	h := NewHandler(nil, creator, nil)
	r := newActionRouter(h)

	w := doAction(t, r, "7", "42", submitBody())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "secret internal host", "未分类错误原文不得透出")
}

func decodeCode(t *testing.T, body []byte) int {
	t.Helper()
	var resp actionEnvelope
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp.Code
}

func decodeMessage(t *testing.T, body []byte) string {
	t.Helper()
	var resp actionEnvelope
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp.Message
}
