package skill

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
)

// 技能注册表租户隔离回归锁。
//
// 缺陷（2026-11-05 白盒审查）：SkillRegistry 是全进程共享的 map，只有 code 一个
// 维度。任一租户持 marketplace:read 就能列出别人注册的自定义技能，持
// marketplace:write 就能凭 code 直接 PUT/DELETE 覆盖或删掉别的租户的技能，
// 持 ai:read 就能调用别人的技能——注册表成了跨租户横向移动的通道。
//
// 现在内置技能 owner=0（全局），自定义技能记录创建租户；读/改/删/调用一律
// 带租户视角，非本租户的技能按「不存在」处理。

// scopedTestSkill 是最小可注册技能：manifest 满足 name/version/permissions 三项必填。
type scopedTestSkill struct {
	code  string
	tags  []string
	calls int
}

func (s *scopedTestSkill) Code() string   { return s.code }
func (s *scopedTestSkill) Name() string   { return s.code }
func (s *scopedTestSkill) Tags() []string { return s.tags }
func (s *scopedTestSkill) Manifest() dto.SkillManifest {
	return dto.SkillManifest{
		Name:                s.code,
		Version:             "v1",
		Category:            "pilot",
		RequiredPermissions: []string{"ai:read"},
	}
}
func (s *scopedTestSkill) GetMetrics() dto.SkillMetrics { return dto.SkillMetrics{} }
func (s *scopedTestSkill) Validate(_ interface{}) error {
	return nil
}
func (s *scopedTestSkill) Execute(_ context.Context, _ interface{}) (interface{}, error) {
	s.calls++
	return nil, nil
}

func TestSkillRegistry_ListIsTenantScoped(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.Register(&scopedTestSkill{code: "builtin.echo"})) // 内置，owner=0
	require.NoError(t, reg.RegisterScoped(1, &scopedTestSkill{code: "tenant1.skill"}))
	require.NoError(t, reg.RegisterScoped(2, &scopedTestSkill{code: "tenant2.skill"}))

	t1 := codesOfEntries(reg.ListEntriesScoped(1))
	require.Contains(t, t1, "builtin.echo", "内置技能对所有租户可见")
	require.Contains(t, t1, "tenant1.skill")
	require.NotContains(t, t1, "tenant2.skill", "租户 1 不得看到租户 2 的技能")

	t2 := codesOfEntries(reg.ListEntriesScoped(2))
	require.Contains(t, t2, "builtin.echo")
	require.Contains(t, t2, "tenant2.skill")
	require.NotContains(t, t2, "tenant1.skill")
}

func TestSkillRegistry_GetIsTenantScoped(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.Register(&scopedTestSkill{code: "builtin.echo"}))
	require.NoError(t, reg.RegisterScoped(1, &scopedTestSkill{code: "tenant1.skill"}))

	_, err := reg.GetScoped(1, "tenant1.skill")
	require.NoError(t, err)

	_, err = reg.GetScoped(2, "tenant1.skill")
	require.True(t, errors.Is(err, ErrSkillNotFound), "跨租户取用必须表现为不存在")

	_, err = reg.GetScoped(2, "builtin.echo")
	require.NoError(t, err, "内置技能所有租户可读")

	_, err = reg.GetEntryScoped(2, "tenant1.skill")
	require.True(t, errors.Is(err, ErrSkillNotFound))
}

func TestSkillRegistry_UnregisterCannotTouchOtherTenant(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.RegisterScoped(1, &scopedTestSkill{code: "tenant1.skill"}))

	err := reg.UnregisterScoped(2, "tenant1.skill")
	require.True(t, errors.Is(err, ErrSkillNotFound), "不能删除其它租户的技能")

	_, err = reg.GetScoped(1, "tenant1.skill")
	require.NoError(t, err, "原租户技能必须仍然存在")

	// 内置技能对租户面同样不可删除（owner=0 != tenantID）。
	require.NoError(t, reg.Register(&scopedTestSkill{code: "builtin.echo"}))
	err = reg.UnregisterScoped(1, "builtin.echo")
	require.True(t, errors.Is(err, ErrSkillNotFound), "租户不得删除平台内置技能")
	require.True(t, reg.Has("builtin.echo"))
}

func TestSkillRegistry_CodeIsGloballyUnique(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.RegisterScoped(1, &scopedTestSkill{code: "shared.code"}))

	err := reg.RegisterScoped(2, &scopedTestSkill{code: "shared.code"})
	require.True(t, errors.Is(err, ErrSkillAlreadyRegistered), "code 全局唯一，不能被别的租户抢占")
}

func TestSkillRegistry_ScopedInvokeRejectsForeignSkill(t *testing.T) {
	reg := NewSkillRegistry()
	foreign := &scopedTestSkill{code: "tenant1.skill"}
	require.NoError(t, reg.RegisterScoped(1, foreign))

	// 调用入口先 GetEntryScoped 再 InvokeWithMetrics；跨租户在第一步就被挡下，
	// 技能实现根本不会被执行（calls 保持 0）。
	if _, err := reg.GetEntryScoped(2, "tenant1.skill"); err == nil {
		if _, err := reg.Invoke(context.Background(), "tenant1.skill", nil); err != nil {
			t.Fatalf("invoke: %v", err)
		}
	}
	require.Equal(t, 0, foreign.calls, "跨租户调用不得执行目标技能")
}

func TestSkillRegistry_EntryCarriesOwner(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.Register(&scopedTestSkill{code: "builtin.echo"}))
	require.NoError(t, reg.RegisterScoped(7, &scopedTestSkill{code: "tenant7.skill"}))

	builtin, err := reg.GetEntryScoped(1, "builtin.echo")
	require.NoError(t, err)
	require.Equal(t, 0, builtin.TenantID)
	require.True(t, builtin.IsBuiltin)

	custom, err := reg.GetEntryScoped(7, "tenant7.skill")
	require.NoError(t, err)
	require.Equal(t, 7, custom.TenantID)
	require.False(t, custom.IsBuiltin, "自定义技能不应被标成内置")
}

func codesOfEntries(entries []dto.SkillEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Code)
	}
	return out
}
