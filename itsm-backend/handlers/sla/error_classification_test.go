package sla

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
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// faultRepo 在既有 Repository 之上只改写两个 Get 方法，注入精准的 DB 故障。
// 内嵌 Repository 让其余 20 个方法原样继承，避免为测试实现整个接口。
type faultRepo struct {
	Repository
	getDefinitionErr error
	getAlertRuleErr  error
}

func (f *faultRepo) GetDefinition(ctx context.Context, id, tenantID int) (*SLADefinition, error) {
	if f.getDefinitionErr != nil {
		return nil, f.getDefinitionErr
	}
	return f.Repository.GetDefinition(ctx, id, tenantID)
}

func (f *faultRepo) GetAlertRule(ctx context.Context, id, tenantID int) (*SLAAlertRule, error) {
	if f.getAlertRuleErr != nil {
		return nil, f.getAlertRuleErr
	}
	return f.Repository.GetAlertRule(ctx, id, tenantID)
}

// buildFaultRouter 用注入故障的 repo 装配 handler，路由覆盖四个误分类点。
func buildFaultRouter(t *testing.T, repo Repository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo, zap.NewNop().Sugar()))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Next()
	})
	r.GET("/definitions/:id", h.GetSLADefinition)
	r.PUT("/definitions/:id", h.UpdateSLADefinition)
	r.GET("/alert-rules/:id", h.GetAlertRule)
	r.PUT("/alert-rules/:id", h.UpdateAlertRule)
	return r
}

// TestSLAHandler_DBFaultIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁：
// 旧实现把 GetDefinition/GetAlertRule 的**所有** error 无条件映射成 404
// （common.NotFound），DB 故障时客户端看到「SLA 定义/告警规则不存在」，
// 以为是资源被删而非后端故障，driver 错误也被丢弃。
// 正确语义：真正的 ent.NotFoundError → 404；其余（含 DB 故障）→ 500/5001。
func TestSLAHandler_DBFaultIsNotReportedAsNotFound(t *testing.T) {
	dbFault := errors.New("injected DB connection failure")

	cases := []struct {
		name   string
		repo   *faultRepo
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "GetSLADefinition",
			repo:   &faultRepo{getDefinitionErr: dbFault},
			method: http.MethodGet,
			path:   "/definitions/1",
		},
		{
			name:   "UpdateSLADefinition",
			repo:   &faultRepo{getDefinitionErr: dbFault},
			method: http.MethodPut,
			path:   "/definitions/1",
			body:   dto.UpdateSLADefinitionRequest{},
		},
		{
			name:   "GetAlertRule",
			repo:   &faultRepo{getAlertRuleErr: dbFault},
			method: http.MethodGet,
			path:   "/alert-rules/1",
		},
		{
			name:   "UpdateAlertRule",
			repo:   &faultRepo{getAlertRuleErr: dbFault},
			method: http.MethodPut,
			path:   "/alert-rules/1",
			body:   dto.UpdateSLAAlertRuleRequest{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := buildFaultRouter(t, tc.repo)
			var bodyReader *bytes.Reader
			if tc.body != nil {
				payload, err := json.Marshal(tc.body)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(payload)
			} else {
				bodyReader = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(tc.method, tc.path, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.InternalErrorCode, resp.Code, "DB 故障业务码必须是 5001")
			assert.NotEqual(t, common.NotFoundCode, resp.Code, "不得误报 404 not found")
		})
	}
}
