package change

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// getFaultRepo 只改写 Get 注入精准 repo 故障；其余方法原样继承 Repository。
// GetChange / UpdateChange 前置 / AssignChange 前置都走 Service.GetChange → repo.Get，
// 一个注入点覆盖 audit 标记的 handler.go:161/448/777 三处误分类。
type getFaultRepo struct {
	Repository
	getErr error
}

func (f *getFaultRepo) Get(ctx context.Context, id, tenantID int) (*Change, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return nil, errors.New("unexpected fallback to unconfigured base repo")
}

func buildChangeErrorRouter(t *testing.T, repo Repository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo, nil, zap.NewNop().Sugar(), nil))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Set("role", "super_admin")
		c.Next()
	})
	r.GET("/changes/:id", h.GetChange)
	r.PUT("/changes/:id", h.UpdateChange)
	r.POST("/changes/:id/assignee", h.AssignChange)
	return r
}

// TestChangeHandler_DBFaultIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁。
// 旧实现把 GetChange 的**所有** error 无条件 common.NotFound，DB 故障被伪装成
// 「变更不存在」(404/4004)，driver 错误被丢弃。repo.Get 的两条错误来源：
//   - First(ctx)：ent.NotFoundError → 真不存在；其余 → DB 故障；
//   - hydrateUsers(ctx)：用户表读失败 → DB 故障（变更本身可能存在）。
//
// 正确语义：ent.NotFoundError → 404；其余 → 500/5001。
func TestChangeHandler_DBFaultIsNotReportedAsNotFound(t *testing.T) {
	dbFault := errors.New("injected DB connection failure")

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"GetChange", http.MethodGet, "/changes/1", nil},
		{"UpdateChange", http.MethodPut, "/changes/1", dto.UpdateChangeRequest{}},
		{"AssignChange", http.MethodPost, "/changes/1/assignee", struct {
			AssigneeID int `json:"assigneeId"`
		}{AssigneeID: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := buildChangeErrorRouter(t, &getFaultRepo{getErr: dbFault})
			body := []byte(nil)
			if tc.body != nil {
				var err error
				body, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.InternalErrorCode, resp.Code, "DB 故障业务码必须是 5001")
			assert.NotEqual(t, common.NotFoundCode, resp.Code, "不得误报 404 not found")
			assert.NotContains(t, w.Body.String(), dbFault.Error(), "原始驱动错误不得透出")
		})
	}
}

// TestChangeHandler_GenuineNotFoundStill404 锁住「真不存在 → 404」语义，
// 防止修 DB 故障分类时把 ent.NotFoundError 误升级成 500。
func TestChangeHandler_GenuineNotFoundStill404(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"GetChange", http.MethodGet, "/changes/1", nil},
		{"UpdateChange", http.MethodPut, "/changes/1", dto.UpdateChangeRequest{}},
		{"AssignChange", http.MethodPost, "/changes/1/assignee", struct {
			AssigneeID int `json:"assigneeId"`
		}{AssigneeID: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := buildChangeErrorRouter(t, &getFaultRepo{getErr: &ent.NotFoundError{}})
			body := []byte(nil)
			if tc.body != nil {
				var err error
				body, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.NotFoundCode, resp.Code)
		})
	}
}
