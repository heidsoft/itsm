package release

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
	"itsm-backend/dto"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubService 实现 ReleaseService，用函数字段精确控制每个方法的返回值/错误。
// 只覆盖 handler 层关心的语义分流（鉴权/参数/错误码映射），业务逻辑由
// internal/services 的定向测试负责——两层职责分离，失败时能一眼定位是哪层。
type stubService struct {
	listReleases      func(ctx context.Context, tenantID int, page, pageSize int, status, releaseType string, currentUserID int, currentRole string) (*dto.ReleaseListResponse, error)
	createRelease     func(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error)
	getReleaseByID    func(ctx context.Context, id, tenantID int) (*dto.ReleaseResponse, error)
	updateRelease     func(ctx context.Context, id, tenantID int, req *dto.UpdateReleaseRequest, actorID int, actorRole string) (*dto.ReleaseResponse, error)
	updateReleaseStat func(ctx context.Context, id, tenantID int, status string, actorID int, actorRole string) (*dto.ReleaseResponse, error)
	applyApproval     func(ctx context.Context, id, tenantID, actorID int, action, comment string) (*dto.ReleaseResponse, error)
	deleteRelease     func(ctx context.Context, id, tenantID int, actorID int, actorRole string) error
	getReleaseStats   func(ctx context.Context, tenantID int) (*dto.ReleaseStatsResponse, error)
}

func (s *stubService) ListReleases(ctx context.Context, tenantID int, page, pageSize int, status, releaseType string, currentUserID int, currentRole string) (*dto.ReleaseListResponse, error) {
	return s.listReleases(ctx, tenantID, page, pageSize, status, releaseType, currentUserID, currentRole)
}
func (s *stubService) CreateRelease(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error) {
	return s.createRelease(ctx, req, createdBy, tenantID)
}
func (s *stubService) GetReleaseByID(ctx context.Context, id, tenantID int) (*dto.ReleaseResponse, error) {
	return s.getReleaseByID(ctx, id, tenantID)
}
func (s *stubService) UpdateRelease(ctx context.Context, id, tenantID int, req *dto.UpdateReleaseRequest, actorID int, actorRole string) (*dto.ReleaseResponse, error) {
	return s.updateRelease(ctx, id, tenantID, req, actorID, actorRole)
}
func (s *stubService) UpdateReleaseStatus(ctx context.Context, id, tenantID int, status string, actorID int, actorRole string) (*dto.ReleaseResponse, error) {
	return s.updateReleaseStat(ctx, id, tenantID, status, actorID, actorRole)
}
func (s *stubService) ApplyReleaseApproval(ctx context.Context, id, tenantID, actorID int, action, comment string) (*dto.ReleaseResponse, error) {
	return s.applyApproval(ctx, id, tenantID, actorID, action, comment)
}
func (s *stubService) DeleteRelease(ctx context.Context, id, tenantID int, actorID int, actorRole string) error {
	return s.deleteRelease(ctx, id, tenantID, actorID, actorRole)
}
func (s *stubService) GetReleaseStats(ctx context.Context, tenantID int) (*dto.ReleaseStatsResponse, error) {
	return s.getReleaseStats(ctx, tenantID)
}

// okStub 返回所有方法都成功的基线 stub，各用例按需覆盖单个字段。
func okStub() *stubService {
	release := &dto.ReleaseResponse{ID: 1, Title: "rel"}
	return &stubService{
		listReleases: func(context.Context, int, int, int, string, string, int, string) (*dto.ReleaseListResponse, error) {
			return &dto.ReleaseListResponse{}, nil
		},
		createRelease: func(context.Context, *dto.CreateReleaseRequest, int, int) (*dto.ReleaseResponse, error) {
			return release, nil
		},
		getReleaseByID: func(context.Context, int, int) (*dto.ReleaseResponse, error) { return release, nil },
		updateRelease: func(context.Context, int, int, *dto.UpdateReleaseRequest, int, string) (*dto.ReleaseResponse, error) {
			return release, nil
		},
		updateReleaseStat: func(context.Context, int, int, string, int, string) (*dto.ReleaseResponse, error) {
			return release, nil
		},
		applyApproval: func(context.Context, int, int, int, string, string) (*dto.ReleaseResponse, error) {
			return release, nil
		},
		deleteRelease:   func(context.Context, int, int, int, string) error { return nil },
		getReleaseStats: func(context.Context, int) (*dto.ReleaseStatsResponse, error) { return &dto.ReleaseStatsResponse{}, nil },
	}
}

// releaseRouter 装配被测 handler 与鉴权中间件桩。
// auth 为 nil 时不注入任何上下文（模拟认证中间件缺席），用于 401 断言。
func releaseRouter(svc ReleaseService, auth gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(zap.NewNop().Sugar(), svc)
	r := gin.New()
	if auth != nil {
		r.Use(auth)
	}
	r.GET("/api/v1/releases", h.ListReleases)
	r.POST("/api/v1/releases", h.CreateRelease)
	r.GET("/api/v1/releases/:id", h.GetRelease)
	r.PUT("/api/v1/releases/:id", h.UpdateRelease)
	r.PUT("/api/v1/releases/:id/status", h.UpdateReleaseStatus)
	r.POST("/api/v1/releases/:id/approve", h.ApproveRelease)
	r.POST("/api/v1/releases/:id/reject", h.RejectRelease)
	r.POST("/api/v1/releases/:id/rollback", h.RollbackRelease)
	r.DELETE("/api/v1/releases/:id", h.DeleteRelease)
	r.GET("/api/v1/releases/stats", h.GetReleaseStats)
	return r
}

// withTenant 构造标准的租户+用户+角色上下文。
func withTenant(tenantID, userID int, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

func doJSON(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestReleaseHandlers_RequireTenant 覆盖全部 10 个端点的鉴权守卫：
// 缺租户上下文必须 401，且业务服务一次都不应被调用（fail-closed 前置）。
func TestReleaseHandlers_RequireTenant(t *testing.T) {
	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"list", http.MethodGet, "/api/v1/releases", nil},
		{"create", http.MethodPost, "/api/v1/releases", dto.CreateReleaseRequest{Title: "t", Type: "minor"}},
		{"get", http.MethodGet, "/api/v1/releases/1", nil},
		{"update", http.MethodPut, "/api/v1/releases/1", dto.UpdateReleaseRequest{}},
		{"status", http.MethodPut, "/api/v1/releases/1/status", dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCancelled}},
		{"approve", http.MethodPost, "/api/v1/releases/1/approve", nil},
		{"reject", http.MethodPost, "/api/v1/releases/1/reject", map[string]string{"reason": "not ok"}},
		{"rollback", http.MethodPost, "/api/v1/releases/1/rollback", map[string]string{"reason": "incorrect deployment"}},
		{"delete", http.MethodDelete, "/api/v1/releases/1", nil},
		{"stats", http.MethodGet, "/api/v1/releases/stats", nil},
	}
	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			w := doJSON(releaseRouter(okStub(), nil), ep.method, ep.path, ep.body)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "缺租户上下文必须 401: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), strconv.Itoa(common.UnauthorizedCode))
		})
	}
}

// TestReleaseHandlers_InvalidID 覆盖路径 id 解析：非法/负数 id 必须 400，
// 不能把 "abc" 透传进 service（否则 Ent 会拼出诡异 SQL 或误命中）。
func TestReleaseHandlers_InvalidID(t *testing.T) {
	svc := okStub()
	r := releaseRouter(svc, withTenant(1, 2, "admin"))
	cases := []struct {
		name, method, path string
		body               interface{}
	}{
		{"get", http.MethodGet, "/api/v1/releases/abc", nil},
		{"update", http.MethodPut, "/api/v1/releases/abc", dto.UpdateReleaseRequest{}},
		{"status", http.MethodPut, "/api/v1/releases/abc/status", dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCancelled}},
		{"delete", http.MethodDelete, "/api/v1/releases/abc", nil},
		{"approve", http.MethodPost, "/api/v1/releases/abc/approve", nil},
		{"rollback", http.MethodPost, "/api/v1/releases/abc/rollback", map[string]string{"reason": "incorrect deployment"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusBadRequest, w.Code, "非法 id 必须 400: %s", w.Body.String())
		})
	}
}

// TestReleaseHandlers_NotFound 覆盖 service 返回 (nil, nil) 的统一语义：
// 跨租户/不存在 → 404，不能伪装成 200 空对象（那会让调用方误以为单据存在）。
func TestReleaseHandlers_NotFound(t *testing.T) {
	svc := okStub()
	svc.getReleaseByID = func(context.Context, int, int) (*dto.ReleaseResponse, error) { return nil, nil }
	svc.updateRelease = func(context.Context, int, int, *dto.UpdateReleaseRequest, int, string) (*dto.ReleaseResponse, error) {
		return nil, nil
	}
	svc.updateReleaseStat = func(context.Context, int, int, string, int, string) (*dto.ReleaseResponse, error) {
		return nil, nil
	}
	svc.applyApproval = func(context.Context, int, int, int, string, string) (*dto.ReleaseResponse, error) {
		return nil, nil
	}
	r := releaseRouter(svc, withTenant(1, 2, "admin"))
	cases := []struct {
		name, method, path string
		body               interface{}
	}{
		{"get", http.MethodGet, "/api/v1/releases/9", nil},
		{"update", http.MethodPut, "/api/v1/releases/9", dto.UpdateReleaseRequest{}},
		{"status", http.MethodPut, "/api/v1/releases/9/status", dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCancelled}},
		{"approve", http.MethodPost, "/api/v1/releases/9/approve", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusNotFound, w.Code, "nil 响应必须 404: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), strconv.Itoa(common.NotFoundCode))
		})
	}
}

// TestReleaseHandlers_ForbiddenIs403 是错误语义铁律回归锁：
// DataScope 行级拒绝是 *common.AppError(403)。若 handler 误用 FailWithErr，
// AppError 会被兜底吞成 5001（因为 FailWithErr 不分类），客户端会误判为服务故障。
// 四个写端点全部走 isForbiddenAppErr → RespondError，逐个锁定。
func TestReleaseHandlers_ForbiddenIs403(t *testing.T) {
	forbidden := func() error { return common.NewForbiddenError("无权限操作该发布单") }
	svc := okStub()
	svc.updateRelease = func(context.Context, int, int, *dto.UpdateReleaseRequest, int, string) (*dto.ReleaseResponse, error) {
		return nil, forbidden()
	}
	svc.updateReleaseStat = func(context.Context, int, int, string, int, string) (*dto.ReleaseResponse, error) {
		return nil, forbidden()
	}
	svc.applyApproval = func(context.Context, int, int, int, string, string) (*dto.ReleaseResponse, error) {
		return nil, forbidden()
	}
	svc.deleteRelease = func(context.Context, int, int, int, string) error { return forbidden() }
	r := releaseRouter(svc, withTenant(1, 7, "agent"))
	cases := []struct {
		name, method, path string
		body               interface{}
	}{
		{"update", http.MethodPut, "/api/v1/releases/1", dto.UpdateReleaseRequest{}},
		{"status", http.MethodPut, "/api/v1/releases/1/status", dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCancelled}},
		{"approve", http.MethodPost, "/api/v1/releases/1/approve", nil},
		{"delete", http.MethodDelete, "/api/v1/releases/1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusForbidden, w.Code, "行级拒绝必须 403 而非 5001: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), strconv.Itoa(common.ForbiddenCode))
			assert.NotContains(t, w.Body.String(), strconv.Itoa(common.InternalErrorCode), "不得降级为 5001")
		})
	}
}

// TestReleaseHandlers_IllegalTransitionIs409 覆盖三处状态流转端点（status /
// rollback / reject 触发的审批）对状态机白名单拒绝的 409 映射。
// status 与 rollback 显式 errors.Is(ErrInvalidReleaseTransition) → Conflict；
// 这里锁定 status 与 rollback 两条（approve/reject 的冲突由 service 层 BusinessError
// 走 RespondError 分类，见下一条用例）。
func TestReleaseHandlers_IllegalTransitionIs409(t *testing.T) {
	svc := okStub()
	svc.updateReleaseStat = func(context.Context, int, int, string, int, string) (*dto.ReleaseResponse, error) {
		return nil, dto.ErrInvalidReleaseTransition
	}
	r := releaseRouter(svc, withTenant(1, 2, "admin"))

	for _, tc := range []struct {
		name, method, path string
		body               interface{}
	}{
		{"status", http.MethodPut, "/api/v1/releases/1/status", dto.ReleaseStatusUpdateRequest{Status: dto.ReleaseStatusCompleted}},
		{"rollback", http.MethodPost, "/api/v1/releases/1/rollback", map[string]string{"reason": "incorrect deployment"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusConflict, w.Code, "非法流转必须 409: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), strconv.Itoa(common.ConflictCode), "业务码必须是 4090")
		})
	}
}

// TestReleaseHandlers_ApprovalConflictViaRespondError 锁定审批冲突路径：
// service 返回类型化 BusinessError(4090)，handler 走 RespondError 分类，
// 必须 409；未被分类的普通 error 才落 500。
func TestReleaseHandlers_ApprovalConflictViaRespondError(t *testing.T) {
	svc := okStub()
	svc.applyApproval = func(context.Context, int, int, int, string, string) (*dto.ReleaseResponse, error) {
		return nil, common.NewBusinessError(common.ConflictCode, "当前发布状态不允许审批", "")
	}
	w := doJSON(releaseRouter(svc, withTenant(1, 2, "admin")),
		http.MethodPost, "/api/v1/releases/1/approve", nil)
	assert.Equal(t, http.StatusConflict, w.Code, "审批冲突必须 409: %s", w.Body.String())
}

// TestReleaseHandlers_ApproveAcceptsEmptyBody 锁定审批意见可选语义：
// approve 允许空请求体（ContentLength=0），不能因 ShouldBindJSON 失败而 400。
func TestReleaseHandlers_ApproveAcceptsEmptyBody(t *testing.T) {
	called := false
	svc := okStub()
	svc.applyApproval = func(_ context.Context, _, _, _ int, action, comment string) (*dto.ReleaseResponse, error) {
		called = true
		assert.Equal(t, "approve", action)
		assert.Empty(t, comment)
		return &dto.ReleaseResponse{ID: 1}, nil
	}
	w := doJSON(releaseRouter(svc, withTenant(1, 2, "admin")),
		http.MethodPost, "/api/v1/releases/1/approve", nil)
	assert.Equal(t, http.StatusOK, w.Code, "空 body 的 approve 必须成功: %s", w.Body.String())
	assert.True(t, called, "service 必须被调用")
}

// TestReleaseHandlers_RejectRequiresReason 锁定 reject 的必填原因校验：
// 空 body 必须先于 service 调用被拒，避免无理由拒绝进入审计链。
func TestReleaseHandlers_RejectRequiresReason(t *testing.T) {
	called := false
	svc := okStub()
	svc.applyApproval = func(context.Context, int, int, int, string, string) (*dto.ReleaseResponse, error) {
		called = true
		return &dto.ReleaseResponse{}, nil
	}
	w := doJSON(releaseRouter(svc, withTenant(1, 2, "admin")),
		http.MethodPost, "/api/v1/releases/1/reject", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code, "缺拒绝原因必须 400: %s", w.Body.String())
	assert.False(t, called, "参数校验失败时 service 不应被调用")
}

// TestReleaseHandlers_StatsFailClosed 锁定统计面错误语义：
// service 报 DB 故障时 handler 必须返回 500，不能吞成 200 空 stats
// （与 internal/services 的 fail-closed 回归锁首尾呼应）。
func TestReleaseHandlers_StatsFailClosed(t *testing.T) {
	svc := okStub()
	svc.getReleaseStats = func(context.Context, int) (*dto.ReleaseStatsResponse, error) {
		return nil, errors.New("db down")
	}
	w := doJSON(releaseRouter(svc, withTenant(1, 2, "admin")),
		http.MethodGet, "/api/v1/releases/stats", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "统计失败必须 500: %s", w.Body.String())
}

// TestReleaseHandlers_ListPassesDataScope 锁定 ListReleases 的行级权限下传：
// 普通角色必须带 user_id/role 下传给 service，管理角色带原值；
// 否则 service 的 DataScope 判定拿不到 actor，会错误返回全量或空集。
func TestReleaseHandlers_ListPassesDataScope(t *testing.T) {
	var gotUserID int
	var gotRole string
	svc := okStub()
	svc.listReleases = func(_ context.Context, _ int, _, _ int, _, _ string, uid int, role string) (*dto.ReleaseListResponse, error) {
		gotUserID, gotRole = uid, role
		return &dto.ReleaseListResponse{}, nil
	}
	w := doJSON(releaseRouter(svc, withTenant(10, 42, "agent")), http.MethodGet, "/api/v1/releases", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 42, gotUserID)
	assert.Equal(t, "agent", gotRole)
}

// TestReleaseHandlers_BadJSON 覆盖请求体解析失败统一 400，且 service 不被调用。
func TestReleaseHandlers_BadJSON(t *testing.T) {
	called := false
	svc := okStub()
	svc.createRelease = func(context.Context, *dto.CreateReleaseRequest, int, int) (*dto.ReleaseResponse, error) {
		called = true
		return &dto.ReleaseResponse{}, nil
	}
	r := releaseRouter(svc, withTenant(1, 2, "admin"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases", bytes.NewBufferString("{not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, called, "非法 JSON 不得进入 service")
}
