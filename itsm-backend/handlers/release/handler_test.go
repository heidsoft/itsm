package release

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	entrelease "itsm-backend/ent/release"
	"itsm-backend/middleware"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestHandler_UpdateReleaseStatus_IllegalTransition_409 回归：状态机白名单拒绝
// 曾被映射为 400（ParamError），客户端会把"被流程防护拦截"误判为输入错误而
// 修参重试；按全局错误语义规范，非法 ITIL 状态迁移属于 conflict。
// 修复后必须返回 HTTP 409 + 业务码 4090。
func TestHandler_UpdateReleaseStatus_IllegalTransition_409(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:release-handler-409?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("rel-t").
		SetCode("rel-t").
		SetDomain("rel-t.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("rel-owner").
		SetEmail("rel-owner@example.com").
		SetName("rel-owner").
		SetPasswordHash("hash").
		SetRole("admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	releaseService := service.NewReleaseService(client, zaptest.NewLogger(t).Sugar())
	rel, err := releaseService.CreateRelease(ctx, &dto.CreateReleaseRequest{
		ReleaseNumber: "REL-409-1",
		Title:         "状态机 409 回归",
		Type:          "minor",
	}, user.ID, tenant.ID)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	h := NewHandler(zap.NewNop().Sugar(), releaseService)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("user_id", user.ID)
		c.Next()
	}
	api := r.Group("/api/v1", auth)
	api.PUT("/releases/:id/status", h.UpdateReleaseStatus)

	doReleaseJSON := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// 非法迁移 draft -> completed（draft 只能取消）。
	w := doReleaseJSON(http.MethodPut, "/api/v1/releases/"+strconv.Itoa(rel.ID)+"/status",
		dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCompleted})
	assert.Equal(t, http.StatusConflict, w.Code, "非法状态迁移必须 409: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "4090", "业务错误码必须是 4090")

	// 状态未被篡改，仍是 draft。
	got, qErr := client.Release.Get(ctx, rel.ID)
	require.NoError(t, qErr)
	assert.Equal(t, entrelease.StatusDraft, got.Status, "被拒绝的迁移不得落库")
}
