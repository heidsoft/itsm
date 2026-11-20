package knowledge

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

// getFaultRepo 只改写 Get，注入精准的 repo 层故障；其余方法原样继承 Repository，
// 避免为一次分类测试实现整套知识库接口。GetArticle/UpdateArticle 的前置校验都走
// Service.ensureArticleReadable → repo.Get，因此一个注入点即可覆盖两个误分类端点。
type getFaultRepo struct {
	Repository
	getErr error
}

func (f *getFaultRepo) Get(ctx context.Context, id, tenantID int) (*Article, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.Repository != nil {
		return f.Repository.Get(ctx, id, tenantID)
	}
	// 无注入且无底层 repo 时，返回一个最小合法文章：
	// 供「前置校验通过、下游 repo 调用点注入故障」的用例走通完整路径。
	return &Article{ID: id, TenantID: tenantID, Category: "general"}, nil
}

// listReviewsFaultRepo 覆盖 ListArticleReviews 的 repo 调用点。
type listReviewsFaultRepo struct {
	getFaultRepo
	listReviewsErr error
}

func (f *listReviewsFaultRepo) ListReviews(ctx context.Context, articleID, tenantID, page, pageSize int) ([]*ReviewRecord, int, error) {
	if f.listReviewsErr != nil {
		return nil, 0, f.listReviewsErr
	}
	return f.Repository.ListReviews(ctx, articleID, tenantID, page, pageSize)
}

// markReviewedFaultRepo 覆盖 MarkArticleReviewed 的 repo 调用点。
type markReviewedFaultRepo struct {
	getFaultRepo
	markReviewedErr error
}

func (f *markReviewedFaultRepo) MarkReviewed(ctx context.Context, id, tenantID int) (*Article, error) {
	if f.markReviewedErr != nil {
		return nil, f.markReviewedErr
	}
	return f.Repository.MarkReviewed(ctx, id, tenantID)
}

func buildKnowledgeErrorRouter(t *testing.T, repo Repository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo, zap.NewNop().Sugar()))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Set("role", "end_user")
		c.Next()
	})
	r.GET("/articles/:id", h.GetArticle)
	r.PUT("/articles/:id", h.UpdateArticle)
	r.POST("/articles/:id/review", h.MarkArticleReviewed)
	r.GET("/articles/:id/reviews", h.ListArticleReviews)
	return r
}

// TestKnowledgeHandler_DBFaultIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁。
// 旧实现对 GetArticle/UpdateArticle 前置校验的**所有** error 无条件 common.NotFound，
// DB 故障被伪装成「文章不存在」(404/4004)，driver 错误也被丢弃。
// 正确语义：
//   - ErrArticleNotFound（权限拒绝/真不存在的防枚举伪装）与 ent.NotFoundError → 404；
//   - 其余（含 DB 故障）→ 500/5001，原始错误只进日志。
func TestKnowledgeHandler_DBFaultIsNotReportedAsNotFound(t *testing.T) {
	dbFault := errors.New("injected DB connection failure")

	for _, tc := range []struct {
		name   string
		repo   Repository
		method string
		path   string
		body   interface{}
	}{
		{"GetArticle", &getFaultRepo{getErr: dbFault}, http.MethodGet, "/articles/1", nil},
		{"UpdateArticle", &getFaultRepo{getErr: dbFault}, http.MethodPut, "/articles/1", dto.UpdateKnowledgeArticleRequest{}},
		{"MarkArticleReviewed", &markReviewedFaultRepo{markReviewedErr: dbFault}, http.MethodPost, "/articles/1/review", nil},
		{"ListArticleReviews", &listReviewsFaultRepo{listReviewsErr: dbFault}, http.MethodGet, "/articles/1/reviews", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := buildKnowledgeErrorRouter(t, tc.repo)
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
			assert.NotContains(t, w.Body.String(), dbFault.Error(), "原始驱动错误不得透出到响应")
		})
	}
}

// TestKnowledgeHandler_NotFoundSemanticsPreserved 锁住防枚举语义：无论真不存在
// (ent.NotFoundError) 还是分类不可读 (ErrArticleNotFound)，都必须维持 404/4004，
// 不能因为修 DB 故障分类而把「文章存在性」泄露成 500。
func TestKnowledgeHandler_NotFoundSemanticsPreserved(t *testing.T) {
	for _, errCase := range []struct {
		name string
		err  error
	}{
		{"category_denied_anti_enumeration", ErrArticleNotFound},
		{"genuine_not_found", &ent.NotFoundError{}},
	} {
		for _, tc := range []struct {
			name   string
			method string
			path   string
			body   interface{}
		}{
			{"GetArticle", http.MethodGet, "/articles/1", nil},
			{"UpdateArticle", http.MethodPut, "/articles/1", dto.UpdateKnowledgeArticleRequest{}},
			{"MarkArticleReviewed", http.MethodPost, "/articles/1/review", nil},
			{"ListArticleReviews", http.MethodGet, "/articles/1/reviews", nil},
		} {
			t.Run(errCase.name+"/"+tc.name, func(t *testing.T) {
				r := buildKnowledgeErrorRouter(t, &getFaultRepo{getErr: errCase.err})
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
}
