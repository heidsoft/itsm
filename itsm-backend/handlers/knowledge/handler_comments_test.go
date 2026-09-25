package knowledge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 回归（2026-09-25 假成功收口）：评论未落地（无存储模型）时必须显式 unready（5003），
// 不得返回假的评论对象，也不得用空成功伪装"暂无评论"。
func TestArticleCommentsReturnExplicitUnready(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil)
	r := gin.New()
	r.GET("/api/v1/knowledge/articles/:id/comments", h.GetArticleComments)
	r.POST("/api/v1/knowledge/articles/:id/comments", h.AddArticleComment)

	for _, tc := range []struct{ name, method string }{
		{"list", http.MethodGet},
		{"add", http.MethodPost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1/knowledge/articles/1/comments", strings.NewReader(`{"content":"hi"}`)))
			require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
			var response struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, 5003, response.Code)
			require.NotEmpty(t, response.Message)
			require.NotContains(t, w.Body.String(), "stub_comment_id", "禁止返回假评论对象")
			require.NotContains(t, w.Body.String(), `"comments"`, "禁止用空成功伪装暂无评论")
		})
	}
}
