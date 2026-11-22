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

// 技能调用入口的租户归属回归锁（HTTP 级）。
//
// 缺陷链：handler 的 GetEntryScoped 失败时不写响应，requiredPermissions 门禁
// 写成 `entryErr == nil && ...` 被整段跳过；随后 InvokeWithMetrics 用无租户
// 视角的 Get(code) 命中全局注册表，照样执行别人的技能。结果：租户 B 只要知道
// 技能码，就能执行租户 A 的自定义技能，且完全绕过 manifest 声明的权限。
//
// 这里直接打真实路由，并把「技能有没有被执行」作为断言——只看 HTTP 状态码
// 不够，旧实现在跨租户请求上恰好返回 200。

// scopeProbeSkill 记录真实执行次数，使「副作用没有发生」可被断言。
type scopeProbeSkill struct {
	*dto.BaseSkill
	calls int
}

func newScopeProbeSkill(code string, perms []string) *scopeProbeSkill {
	return &scopeProbeSkill{
		BaseSkill: dto.NewBaseSkill(code, code, "v1", "pilot", nil, perms, nil),
	}
}

func (s *scopeProbeSkill) Manifest() dto.SkillManifest { return s.BuildManifest() }
func (s *scopeProbeSkill) Validate(interface{}) error  { return nil }
func (s *scopeProbeSkill) Execute(context.Context, interface{}) (interface{}, error) {
	s.calls++
	return "executed", nil
}

type invokeScopeFixture struct {
	router *gin.Engine
	tenant int
	role   string
	probe  *scopeProbeSkill
}

func newInvokeScopeFixture(t *testing.T, code string, perms []string, owner int) *invokeScopeFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// 权限判定走 authz 默认集；本文件只用「格式非法」与「ai:read」两类声明，
	// 不依赖具体角色的权限表内容。
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly

	reg := skill.NewSkillRegistry()
	probe := newScopeProbeSkill(code, perms)
	if owner == 0 {
		require.NoError(t, reg.Register(probe))
	} else {
		require.NoError(t, reg.RegisterScoped(owner, probe))
	}
	h := skill.NewHandler(reg, zap.NewNop().Sugar(), nil)

	f := &invokeScopeFixture{probe: probe, tenant: 1, role: "admin"}
	r := gin.New()
	r.Use(gin.Recovery())
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: f.tenant})
		c.Set("tenant_id", f.tenant)
		c.Set("user_id", 42)
		c.Set("role", f.role)
		c.Next()
	})
	v1.POST("/admin/skills/:code/invoke", h.Invoke)
	f.router = r
	return f
}

func (f *invokeScopeFixture) invoke(t *testing.T, code string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{"input": map[string]any{"q": "1"}})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skills/"+code+"/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// 跨租户调用必须 404，且技能实现一次都不能跑；本租户调用仍正常执行。
func TestInvoke_ForeignTenantSkillIsNotExecuted(t *testing.T) {
	f := newInvokeScopeFixture(t, "tenant1.private", []string{"ai:read"}, 1)

	f.tenant = 2
	w := f.invoke(t, "tenant1.private")
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	assert.Equal(t, 0, f.probe.calls, "跨租户请求不得执行目标技能")
	assert.NotContains(t, w.Body.String(), "executed")

	f.tenant = 1
	w = f.invoke(t, "tenant1.private")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, f.probe.calls, "本租户调用应正常执行")
}

// 第二道防线：scoped 查找成功后，requiredPermissions 门禁必须无条件执行。
// 格式非法的权限声明按 fail-closed 处理，且同样不得触发技能执行。
func TestInvoke_MalformedRequiredPermissionFailsClosed(t *testing.T) {
	f := newInvokeScopeFixture(t, "tenant1.badperm", []string{"malformed"}, 1)

	w := f.invoke(t, "tenant1.badperm")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "权限声明格式错误")
	assert.Equal(t, 0, f.probe.calls, "权限门禁拒绝时不得执行技能")
}

// spyRegistry 模拟旧缺陷的临界状态：scoped 查找失败，但执行入口「若被调用」
// 仍会成功。用来单独锁住 handler 这道防线——否则注册表收口后，handler 漏掉
// fail-closed 也不会被任何 HTTP 测试发现（执行入口自己就拒了）。
type spyRegistry struct {
	entryErr    error
	invokeCalls int
}

func (s *spyRegistry) ListEntriesScoped(int) []dto.SkillEntry              { return nil }
func (s *spyRegistry) ListEntriesByTagScoped(int, string) []dto.SkillEntry { return nil }
func (s *spyRegistry) GetEntryScoped(int, string) (dto.SkillEntry, error) {
	return dto.SkillEntry{}, s.entryErr
}
func (s *spyRegistry) GetScoped(int, string) (dto.Skill, error) { return nil, s.entryErr }
func (s *spyRegistry) RegisterScoped(int, dto.Skill) error      { return nil }
func (s *spyRegistry) UnregisterScoped(int, string) error       { return nil }
func (s *spyRegistry) InvokeWithMetrics(context.Context, int, string, interface{}) (dto.SkillInvokeResult, error) {
	s.invokeCalls++
	return dto.SkillInvokeResult{Output: "executed"}, nil
}

// scoped 查找失败时，handler 必须自己停住，不能把请求继续交给执行入口。
func TestInvoke_ScopedLookupFailureNeverReachesExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	spy := &spyRegistry{entryErr: dto.ErrSkillNotFound}
	h := skill.NewHandler(spy, zap.NewNop().Sugar(), nil)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 2})
		c.Set("tenant_id", 2)
		c.Set("user_id", 42)
		c.Set("role", "admin")
		c.Next()
	})
	v1.POST("/admin/skills/:code/invoke", h.Invoke)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/skills/foreign.skill/invoke", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	assert.Equal(t, 0, spy.invokeCalls, "scoped 查找失败后不得再调用执行入口")
}

// 租户收口不能误伤内置技能：owner=0 的技能对任意租户可见可执行。
func TestInvoke_BuiltinSkillCallableFromAnyTenant(t *testing.T) {
	f := newInvokeScopeFixture(t, "builtin.echo", []string{"ai:read"}, 0)

	f.tenant = 7
	w := f.invoke(t, "builtin.echo")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, f.probe.calls)
}
