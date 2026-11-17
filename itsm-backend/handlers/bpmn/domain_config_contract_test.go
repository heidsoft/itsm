package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GET /api/v1/domain-configs/effective 契约回归：
//   - 作用域参数是 camelCase departmentId / teamId。历史实现读取 snake_case，
//     导致部门/团队作用域静默退化为租户级。
//   - 全链路未命中配置是合法空结果，必须返回 code=0 + 空 data，
//     而不是 500/5001。
//   - 租户上下文 fail-closed。

func newDomainConfigTestRouter(t *testing.T, client *ent.Client, tenantID int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()
	configSvc := service.NewConfigInheritanceService(client, logger)
	handler := NewProcessTriggerHandler(nil, nil, configSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		// 与生产 TenantMiddleware 一致：TenantIDOrUnauthorized 读取的是
		// middleware.TenantContext，而不是裸 tenant_id 整数。
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("tenant_id", tenantID)
		c.Set("user_id", 1)
		// 路由级 RequirePermission 需要 role + client 两个上下文键。
		// 本测试库未跑 seeder（角色表为空 = DBOnly unconfigured），
		// 判定回落到 middleware.RolePermissions["admin"]，即真实鉴权链路而非绕过。
		c.Set("role", "admin")
		c.Set("client", client)
		c.Next()
	})
	group := r.Group("/api/v1")
	handler.RegisterRoutes(group)
	return r
}

func seedDomainConfig(t *testing.T, client *ent.Client, tenantID, departmentID, teamID int, value map[string]interface{}, mode string) {
	t.Helper()
	err := client.DomainConfig.Create().
		SetConfigType("sla_rule").
		SetConfigKey("response_time").
		SetConfigValue(value).
		SetInheritMode(mode).
		SetTenantID(tenantID).
		SetDepartmentID(departmentID).
		SetTeamID(teamID).
		SetIsActive(true).
		Exec(context.Background())
	require.NoError(t, err)
}

// data 使用指针：响应在「未命中」时省略 data 字段（Response.Data 带 omitempty），
// 指针形态能同时表达 absent 与 null 两种空结果。
type effectiveEnvelope struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Data    *service.ResolvedConfig `json:"data"`
}

func getEffective(t *testing.T, r *gin.Engine, query string) (int, effectiveEnvelope) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/domain-configs/effective?"+query, nil))

	var env effectiveEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return w.Code, env
}

func postDomainConfig(t *testing.T, r *gin.Engine, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/domain-configs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var env map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return w.Code, env
}

func TestGetEffectiveDomainConfig_HonorsCamelCaseDepartmentScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_effective_dept?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 租户级基线 + 部门级覆盖：只有 departmentId 真正参与解析，
	// 结果才会命中 department 层。
	seedDomainConfig(t, client, 1, 0, 0, map[string]interface{}{"targetMinutes": float64(60)}, "override")
	seedDomainConfig(t, client, 1, 5, 0, map[string]interface{}{"targetMinutes": float64(15)}, "override")

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=response_time&departmentId=5")

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, env.Code)
	require.NotNil(t, env.Data)
	assert.Equal(t, "department:5", env.Data.Source)
	assert.Equal(t, float64(15), env.Data.Value["targetMinutes"])
}

func TestGetEffectiveDomainConfig_HonorsCamelCaseTeamScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_effective_team?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedDomainConfig(t, client, 1, 0, 0, map[string]interface{}{"targetMinutes": float64(60)}, "override")
	seedDomainConfig(t, client, 1, 7, 3, map[string]interface{}{"targetMinutes": float64(5)}, "override")

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=response_time&departmentId=7&teamId=3")

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, env.Code)
	require.NotNil(t, env.Data)
	assert.Equal(t, "team:3", env.Data.Source)
	assert.Equal(t, float64(5), env.Data.Value["targetMinutes"])
}

func TestGetEffectiveDomainConfig_MissingConfigReturnsEmptySuccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_effective_empty?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=absent")

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, env.Code)
	assert.Nil(t, env.Data)
}

func TestGetEffectiveDomainConfig_RejectsNonNumericScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_effective_badscope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=response_time&departmentId=abc")

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, 1001, env.Code)
}

func TestGetEffectiveDomainConfig_CrossTenantScopeFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_effective_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// 仅租户 2 存在部门 5 的配置；租户 1 使用相同 departmentId 不得读到。
	seedDomainConfig(t, client, 2, 5, 0, map[string]interface{}{"targetMinutes": float64(15)}, "override")

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=response_time&departmentId=5")

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, 0, env.Code)
	assert.Nil(t, env.Data)
}

// 未显式指定 inheritMode 的新增配置必须能独立生效。历史默认 inherit 会让该
// 层级在解析时被整条跳过，用户看到「配置已保存」但有效配置始终未命中。
func TestSetDomainConfig_DefaultsToOverrideAndResolves(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_default_override?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newDomainConfigTestRouter(t, client, 1)
	status, _ := postDomainConfig(t, r,
		`{"configType":"sla_rule","configKey":"response_time","configValue":{"targetMinutes":30}}`)
	require.Equal(t, http.StatusOK, status)

	code, env := getEffective(t, r, "configType=sla_rule&configKey=response_time")
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, env.Data)
	assert.Equal(t, dto.InheritModeOverride, env.Data.InheritMode)
	assert.Equal(t, float64(30), env.Data.Value["targetMinutes"])
}

func TestSetDomainConfig_RejectsUnknownInheritMode(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_invalid_mode?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := postDomainConfig(t, r,
		`{"configType":"sla_rule","configKey":"response_time","configValue":{"x":1},"inheritMode":"merge"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, float64(1001), env["code"])
}

// extend 必须在嵌套对象上递归合并：子级只改一个字段不应抹掉父级的兄弟字段。
func TestGetEffectiveDomainConfig_ExtendDeepMergesNestedObjects(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:dc_extend_nested?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	seedDomainConfig(t, client, 1, 0, 0,
		map[string]interface{}{"sla": map[string]interface{}{"firstResponse": float64(30), "resolution": float64(240)}},
		dto.InheritModeOverride)
	seedDomainConfig(t, client, 1, 5, 0,
		map[string]interface{}{"sla": map[string]interface{}{"resolution": float64(60)}},
		dto.InheritModeExtend)

	r := newDomainConfigTestRouter(t, client, 1)
	status, env := getEffective(t, r, "configType=sla_rule&configKey=response_time&departmentId=5")

	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, env.Data)
	merged, ok := env.Data.Value["sla"].(map[string]interface{})
	require.True(t, ok, "nested sla object must survive the merge: %#v", env.Data.Value)
	assert.Equal(t, float64(30), merged["firstResponse"])
	assert.Equal(t, float64(60), merged["resolution"])
}
