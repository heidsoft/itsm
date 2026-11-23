package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// conversationFaultRepo 只覆盖会话读/删两法，其余方法由内嵌接口兜底（nil 不会
// 被这两个 handler 触达）。用于注入 DB 故障与 ent.NotFoundError 两类错误。
type conversationFaultRepo struct {
	Repository
	getErr    error
	deleteErr error
}

func (r *conversationFaultRepo) GetConversation(ctx context.Context, id, tenantID, userID int) (*Conversation, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return &Conversation{ID: id, Title: "owner conv", UserID: userID, TenantID: tenantID}, nil
}

func (r *conversationFaultRepo) GetMessages(ctx context.Context, conversationID, tenantID int) ([]*Message, error) {
	return []*Message{}, nil
}

func (r *conversationFaultRepo) DeleteConversation(ctx context.Context, id, tenantID, userID int) error {
	return r.deleteErr
}

func buildConversationErrorRouter(t *testing.T, repo Repository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := &Service{repo: repo, logger: zaptest.NewLogger(t).Sugar()}
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 42)
		c.Set("role", "end_user")
		c.Next()
	})
	r.GET("/conversations/:id", h.GetConversation)
	r.DELETE("/conversations/:id", h.DeleteConversation)
	return r
}

// TestAIConversation_DBFaultIsNotReportedAsNotFound 是「撤掉守卫即失败」的回归锁：
// 旧实现把 GetConversationMessages/DeleteConversation 的任何错误都映射成 404
// 「会话不存在」，DB 故障（表缺失/连接断开）被伪装成资源不存在，驱动错误被
// 丢弃，观测上表现为 404 而非 5xx。改为：ent.NotFound → 404 同文案（保留防
// 枚举语义），其余 DB 故障 → 500。
func TestAIConversation_DBFaultIsNotReportedAsNotFound(t *testing.T) {
	dbFault := errors.New("sql: database is closed")
	for _, tc := range []struct {
		name   string
		method string
		inject func(*conversationFaultRepo)
	}{
		{"GetConversation", http.MethodGet, func(r *conversationFaultRepo) { r.getErr = dbFault }},
		{"DeleteConversation", http.MethodDelete, func(r *conversationFaultRepo) { r.deleteErr = dbFault }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &conversationFaultRepo{}
			tc.inject(repo)
			r := buildConversationErrorRouter(t, repo)
			req := httptest.NewRequest(tc.method, "/conversations/1", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code, "DB 故障必须 500，body=%s", w.Body.String())
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.InternalErrorCode, resp.Code)
		})
	}
}

// TestAIConversation_NotFoundSemanticsPreserved 锁住防枚举语义：
// ent.NotFoundError（真不存在 + 跨租户/跨用户经租户属主谓词表现为 not found）
// 必须与旧实现同码同文案 404「会话不存在」，不得因改用 RespondError 泄露差异。
func TestAIConversation_NotFoundSemanticsPreserved(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		inject func(*conversationFaultRepo)
	}{
		{"GetConversation", http.MethodGet, func(r *conversationFaultRepo) { r.getErr = &ent.NotFoundError{} }},
		{"DeleteConversation", http.MethodDelete, func(r *conversationFaultRepo) { r.deleteErr = &ent.NotFoundError{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &conversationFaultRepo{}
			tc.inject(repo)
			r := buildConversationErrorRouter(t, repo)
			req := httptest.NewRequest(tc.method, "/conversations/1", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
			var resp common.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, common.NotFoundCode, resp.Code)
			assert.Equal(t, "会话不存在", resp.Message, "防枚举文案必须保持原样")
		})
	}
}
