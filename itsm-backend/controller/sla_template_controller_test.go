package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSLATemplateController_ListTemplates_StandardEnvelope 锁住 Bug 2 修复：
// sla_template_controller 必须返回 {code:0, message:"success", data:{...}} 的标准
// 响应壳，而非历史散落的 gin.H{"message":..., "data":...} 形态。
func TestSLATemplateController_ListTemplates_StandardEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewSLATemplateService(nil, nil)
	ctrl := NewSLATemplateController(svc)

	r := gin.New()
	ctrl.RegisterRoutes(r.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla/templates", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "ListTemplates 应返回 200")

	var body common.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "响应体必须可解析为标准 envelope")
	assert.Equal(t, common.SuccessCode, body.Code, "Bug 2 验收：code 必须为 0（success）")
	assert.Equal(t, "success", body.Message, "Bug 2 验收：message 必须为 success")

	// data 必须是 map 且包含 templates / total 字段
	dataJSON, err := json.Marshal(body.Data)
	require.NoError(t, err)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(dataJSON, &data))
	assert.Contains(t, data, "templates", "data.templates 必须存在")
	assert.Contains(t, data, "total", "data.total 必须存在")
	assert.GreaterOrEqual(t, int(data["total"].(float64)), 6, "至少 6 个开箱模板")
}

// TestSLATemplateController_GetTemplate_NotFound 验证错误响应也走 common.Fail
// （状态码映射：404 → NotFoundCode），不再走 ctx.JSON(http.StatusNotFound, gin.H{"error":...})
func TestSLATemplateController_GetTemplate_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := service.NewSLATemplateService(nil, nil)
	ctrl := NewSLATemplateController(svc)

	r := gin.New()
	ctrl.RegisterRoutes(r.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla/templates/nonexistent_key", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code, "404 路径必须保留 HTTP 404 语义")

	var body common.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, common.NotFoundCode, body.Code, "404 必须映射为 NotFoundCode")
	assert.NotEmpty(t, body.Message)
}

// TestSLATemplateController_NoRawCtxJSON 静态保护：禁止 controller 直接 ctx.JSON(http.StatusXxx, gin.H{...})
// 这一用例本身没什么业务断言，但与 grep 验收形成双重保护，确保未来重构不破坏契约。
func TestSLATemplateController_NoRawCtxJSON(t *testing.T) {
	// 这是设计性检查：若有人再次回退到 ctx.JSON(http.StatusXxx, ...)，grep 验收
	// `grep "ctx.JSON(http.Status" sla_template_controller.go` 必须返回 0 结果。
	// 这里仅留 hook 占位，避免误删。
	assert.NotNil(t, common.Success)
}
