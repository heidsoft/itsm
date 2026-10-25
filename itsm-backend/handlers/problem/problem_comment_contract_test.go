package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"itsm-backend/middleware"
)

// 问题评论契约（2026-10-21 GA 阻断项落地，取代此前的 503 unready 桩）：
// 写入→读回→分页→隔离四类事实。不存在/跨租户的问题统一 404，
// 不用空成功伪装"暂无评论"，也不用 5xx 毛迹暴露可枚举信号。
func TestProblemCommentsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, svc, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "comments")
	otherTenant := createProblemHandlerTenant(t, ctx, client, "comments-other")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "comments")
	otherUser := createProblemHandlerUser(t, ctx, client, otherTenant.ID, "comments-other")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Set("tenant_id", tenant.ID)
		c.Set("user_id", user.ID)
	})
	r.GET("/api/v1/problems/:id/comments", h.GetProblemComments)
	r.POST("/api/v1/problems/:id/comments", h.AddProblemComment)

	// 空列表：真实 200 + items:[]，不是 503、不是 null
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var empty struct {
		Data struct {
			Items      []*ProblemComment `json:"items"`
			Total      int               `json:"total"`
			Page       int               `json:"page"`
			PageSize   int               `json:"pageSize"`
			TotalPages int               `json:"totalPages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &empty))
	assert.Empty(t, empty.Data.Items)
	assert.Equal(t, 0, empty.Data.Total)
	assert.Equal(t, 1, empty.Data.Page)
	assert.Equal(t, 20, empty.Data.PageSize)
	assert.Equal(t, 0, empty.Data.TotalPages)

	// 写入：内容 Trim、内部备注标志、@提及与附件引用原样回显
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments",
		strings.NewReader(`{"content":"  定位到交换机端口 flap  ","isInternal":true,"mentions":[7,8],"attachments":[3]}`)))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created struct {
		Data ProblemComment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "定位到交换机端口 flap", created.Data.Content)
	assert.True(t, created.Data.IsInternal)
	assert.Equal(t, []int{7, 8}, created.Data.Mentions)
	assert.Equal(t, []int{3}, created.Data.Attachments)
	assert.Equal(t, user.ID, created.Data.UserID)
	// 评论人显示名由服务端投影（测试用户名为 problem-comments，name 同值）
	assert.NotEmpty(t, created.Data.UserName)

	// 读回：列表含刚写的评论，时间正序
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listed struct {
		Data struct {
			Items []*ProblemComment `json:"items"`
			Total int               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
	require.Len(t, listed.Data.Items, 1)
	assert.Equal(t, 1, listed.Data.Total)
	assert.Equal(t, created.Data.ID, listed.Data.Items[0].ID)

	// 空内容：400，不落库
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments",
		strings.NewReader(`{"content":"   "}`)))
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// 不存在的问题：404（不因评论能力已落地而改变）
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/99999/comments", nil))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 跨租户的问题：404——不许拿别家租户的 problem_id 写评论或探测存在性
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+strconv.Itoa(p.ID)+"/comments", nil))
	require.Equal(t, http.StatusOK, w.Code)

	otherProblem, err := svc.Create(ctx, otherTenant.ID, &Problem{
		Title: "Other tenant problem", Priority: "low", CreatedBy: otherUser.ID,
	})
	require.NoError(t, err)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/problems/"+strconv.Itoa(otherProblem.ID)+"/comments", nil))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/problems/"+strconv.Itoa(otherProblem.ID)+"/comments",
		strings.NewReader(`{"content":"cross tenant write"}`)))
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// 跨租户写入未落库：对方租户视角仍是空列表
	items, total, err := svc.ListComments(ctx, otherTenant.ID, otherProblem.ID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, items)
}
