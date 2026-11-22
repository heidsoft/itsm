package skill_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/dto"
	"itsm-backend/handlers/skill"
	"itsm-backend/middleware"
)

// InputSchema 执行前校验回归锁。
//
// 字段即声明：manifest.InputSchema 此前只声明/持久化/发现面暴露，执行入口
// 从不消费——技能作者写下的 required/type/enum 全是装饰。这里直接打真实路由，
// 并断言「技能有没有被执行」，而不是只看 HTTP 状态码。

// schemaProbeSkill 带自定义 InputSchema 并记录真实执行次数。
type schemaProbeSkill struct {
	*dto.BaseSkill
	schema map[string]interface{}
	calls  int
}

func newSchemaProbeSkill(code string, schema map[string]interface{}) *schemaProbeSkill {
	return &schemaProbeSkill{
		BaseSkill: dto.NewBaseSkill(code, code, "v1", "pilot", nil, []string{"ai:read"}, nil),
		schema:    schema,
	}
}

func (s *schemaProbeSkill) Manifest() dto.SkillManifest {
	m := s.BuildManifest()
	m.InputSchema = s.schema
	return m
}
func (s *schemaProbeSkill) Validate(interface{}) error { return nil }
func (s *schemaProbeSkill) Execute(context.Context, interface{}) (interface{}, error) {
	s.calls++
	return "executed", nil
}

type schemaFixture struct {
	router *gin.Engine
	probe  *schemaProbeSkill
}

func newSchemaFixture(t *testing.T, schema map[string]interface{}) *schemaFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly

	reg := skill.NewSkillRegistry()
	probe := newSchemaProbeSkill("schema.probe", schema)
	require.NoError(t, reg.Register(probe))
	h := skill.NewHandler(reg, zap.NewNop().Sugar(), nil)

	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("tenant_id", 1)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	v1.POST("/admin/skills/:code/invoke", h.Invoke)
	return &schemaFixture{router: r, probe: probe}
}

func (f *schemaFixture) invoke(t *testing.T, input map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{"input": input})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skills/schema.probe/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// 缺少 required 字段 → 400 且技能零执行。
func TestInvoke_InputSchemaRequiredRejects(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"query"},
	})

	w := f.invoke(t, map[string]any{"other": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, 0, f.probe.calls, "required 缺失时不得执行技能")
}

// 类型不符 → 400 且零执行。
func TestInvoke_InputSchemaTypeRejects(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{"type": "integer"},
		},
	})

	w := f.invoke(t, map[string]any{"limit": "not-a-number"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, 0, f.probe.calls, "类型不符时不得执行技能")
}

// enum 不符 → 400 且零执行。
func TestInvoke_InputSchemaEnumRejects(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"level": map[string]interface{}{"type": "string", "enum": []interface{}{"low", "high"}},
		},
	})

	w := f.invoke(t, map[string]any{"level": "medium"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, 0, f.probe.calls, "enum 不符时不得执行技能")
}

// 合法入参仍执行一次（防止改成一律拒绝）。
func TestInvoke_InputSchemaValidInputExecutes(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"query"},
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string"},
		},
	})

	w := f.invoke(t, map[string]any{"query": "what is cmdb"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, f.probe.calls)
}

// 空 schema 向后兼容：多数内置技能未声明 InputSchema，仍可执行。
func TestInvoke_InputSchemaEmptyAllows(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{})

	w := f.invoke(t, map[string]any{"anything": 1})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, f.probe.calls)
}

// 顺序锁：校验必须发生在身份字段注入之前。
// schema 声明 required=["tenantId"]，调用方不传。若 handler 先注入 tenantId
// 再校验，这个字段会被填进而假通过；正确顺序下必须 400、技能零执行。
func TestInvoke_InputSchemaValidatedBeforeIdentityInjection(t *testing.T) {
	f := newSchemaFixture(t, map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"tenantId"},
	})

	w := f.invoke(t, map[string]any{"query": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, 0, f.probe.calls, "身份字段注入不得掩盖调用方缺失的必填字段")
}
