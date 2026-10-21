package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestTemplateRoutes_PersistRealData 回归：模板 CRUD 曾委托给 repository 层返回
// 伪造数据的 stub——创建/更新"成功"但库里无记录、Get 恒 not found，属于把能力
// 缺失包装成空成功。改接 productionSvc（service.TicketService 真实现）后：
// ① POST 必须真实落库；② 不存在的模板必须 404 而非伪造对象；
// ③ 列表必须按租户隔离；④ DELETE 必须真实删除。
func TestTemplateRoutes_PersistRealData(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket-template-crud?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	gin.SetMode(gin.TestMode)
	// repo 传 nil：模板路径已不依赖 Repository，其余方法本测试不触达。
	svc := NewService(nil, service.NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar()), nil, zap.NewNop().Sugar())
	h := NewHandler(svc)

	r := gin.New()
	auth := func(c *gin.Context) {
		if v := c.GetHeader("X-Test-TenantID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: n})
			}
		}
		if v := c.GetHeader("X-Test-UserID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Set("user_id", n)
			}
		}
		c.Next()
	}
	api := r.Group("/api/v1", auth)
	api.POST("/tickets/templates", h.CreateTicketTemplate)
	api.GET("/tickets/templates", h.GetTicketTemplates)
	api.GET("/tickets/templates/:id", h.GetTicketTemplate)
	api.PUT("/tickets/templates/:id", h.UpdateTicketTemplate)
	api.DELETE("/tickets/templates/:id", h.DeleteTicketTemplate)

	hdrs := map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"}
	ctx := context.Background()

	// ① 创建模板：响应 200 且真实落库（stub 时代返回假成功、库中无记录）。
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets/templates", map[string]interface{}{
		"name":       "测试模板回归",
		"category":   "hardware",
		"priority":   "medium",
		"formFields": map[string]interface{}{"location": map[string]interface{}{"required": true}},
	}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, "创建模板应成功: %s", w.Body.String())
	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	createData := createResp["data"].(map[string]interface{})
	templateID := int(createData["id"].(float64))
	require.Greater(t, templateID, 0)

	total, err := client.TicketTemplate.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "模板必须真实持久化，而不是伪造响应")

	// ② 列表：能看到刚创建的模板，响应只带 items + total（无伪分页）。
	w = doJSON(t, r, http.MethodGet, "/api/v1/tickets/templates", nil, hdrs)
	require.Equal(t, http.StatusOK, w.Code)
	var listResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	items := listResp["data"].(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, 1)
	assert.Equal(t, "测试模板回归", items[0].(map[string]interface{})["name"])

	// ③ 不存在的模板：必须 404（ent.NotFoundError → classifyError 映射）。
	w = doJSON(t, r, http.MethodGet, "/api/v1/tickets/templates/99999", nil, hdrs)
	assert.Equal(t, http.StatusNotFound, w.Code, "get 不存在模板应 404: %s", w.Body.String())

	// ④ 租户隔离：另一租户的模板不得出现在租户 1 的列表中。
	_, err = client.TicketTemplate.Create().
		SetName("别租户模板").
		SetCategory("hardware").
		SetPriority("low").
		SetTenantID(2).
		Save(ctx)
	require.NoError(t, err)
	w = doJSON(t, r, http.MethodGet, "/api/v1/tickets/templates", nil, hdrs)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	items = listResp["data"].(map[string]interface{})["items"].([]interface{})
	assert.Len(t, items, 1, "租户 1 的列表不得泄漏租户 2 的模板")

	// ⑤ 更新：真实改动落库。
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/templates/"+strconv.Itoa(templateID), map[string]interface{}{
		"name":     "测试模板回归-改名",
		"category": "hardware",
		"priority": "high",
	}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	entity, err := client.TicketTemplate.Get(ctx, templateID)
	require.NoError(t, err)
	assert.Equal(t, "测试模板回归-改名", entity.Name)

	// ⑥ 删除：真实删除后只剩另一租户的模板。
	w = doJSON(t, r, http.MethodDelete, "/api/v1/tickets/templates/"+strconv.Itoa(templateID), nil, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	remaining, err := client.TicketTemplate.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, remaining, "只应剩下另一租户的模板")
}

// TestTemplateRoutes_ActivateAndWorkflowSteps 回归：创建/更新请求曾在 handler 里
// 被重新组装成只带 6 个字段的实体，`isActive` 与 `workflowSteps` 直接掉落，
// 服务层又把创建的 isActive 写死 true、停用只能靠 `json:"-"` 的内部标志。
// 后果是管理页的「模板状态」开关是个假控件（提交停用仍得到启用模板），
// 前端类型里声明的 workflowSteps 永远存不进库，复制一个停用模板会得到启用副本。
func TestTemplateRoutes_ActivateAndWorkflowSteps(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket-template-flags?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	gin.SetMode(gin.TestMode)
	svc := NewService(nil, service.NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar()), nil, zap.NewNop().Sugar())
	h := NewHandler(svc)

	r := gin.New()
	auth := func(c *gin.Context) {
		if v := c.GetHeader("X-Test-TenantID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: n})
			}
		}
		c.Next()
	}
	api := r.Group("/api/v1", auth)
	api.POST("/tickets/templates", h.CreateTicketTemplate)
	api.POST("/tickets/templates/:id/copy", h.CopyTicketTemplate)
	api.PUT("/tickets/templates/:id", h.UpdateTicketTemplate)

	hdrs := map[string]string{"X-Test-TenantID": "1", "X-Test-UserID": "7"}
	ctx := context.Background()
	steps := []interface{}{map[string]interface{}{"name": "分派", "order": 1}}

	// ① 显式停用：请求里的 isActive=false 必须落库，不能被默认值覆盖。
	w := doJSON(t, r, http.MethodPost, "/api/v1/tickets/templates", map[string]interface{}{
		"name":          "停用模板",
		"category":      "hardware",
		"priority":      "medium",
		"isActive":      false,
		"workflowSteps": steps,
	}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var createResp struct {
		Data struct {
			ID            int                      `json:"id"`
			IsActive      *bool                    `json:"isActive"`
			WorkflowSteps []map[string]interface{} `json:"workflowSteps"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	disabledID := createResp.Data.ID
	require.NotNil(t, createResp.Data.IsActive)
	assert.False(t, *createResp.Data.IsActive, "响应必须回显停用状态，实际 %v", createResp.Data.IsActive)
	require.Len(t, createResp.Data.WorkflowSteps, 1, "响应必须回显工作流步骤")

	entity, err := client.TicketTemplate.Get(ctx, disabledID)
	require.NoError(t, err)
	assert.False(t, entity.IsActive, "停用必须真实落库")
	assert.NotEmpty(t, entity.WorkflowSteps, "workflowSteps 必须真实落库")

	// ② 不传 isActive：保持历史默认（启用），指针语义要能区分「未传」。
	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/templates", map[string]interface{}{
		"name":     "默认启用模板",
		"category": "hardware",
	}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	require.NotNil(t, createResp.Data.IsActive)
	assert.True(t, *createResp.Data.IsActive, "未提交 isActive 时仍按启用创建")
	enabledID := createResp.Data.ID

	// ③ PUT 停用既有模板：与 PATCH /status 同一结果，不能让状态开关只在创建侧生效。
	w = doJSON(t, r, http.MethodPut, "/api/v1/tickets/templates/"+strconv.Itoa(enabledID), map[string]interface{}{
		"name":     "默认启用模板",
		"category": "hardware",
		"isActive": false,
	}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	entity, err = client.TicketTemplate.Get(ctx, enabledID)
	require.NoError(t, err)
	assert.False(t, entity.IsActive, "PUT 提交的停用必须落库")

	// ④ 复制停用模板：副本继承源模板状态，不静默变成启用。
	w = doJSON(t, r, http.MethodPost, "/api/v1/tickets/templates/"+strconv.Itoa(disabledID)+"/copy", map[string]interface{}{}, hdrs)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	require.NotNil(t, createResp.Data.IsActive)
	assert.False(t, *createResp.Data.IsActive, "停用模板的副本仍应是停用")
}
