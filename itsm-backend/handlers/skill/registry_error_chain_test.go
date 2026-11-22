package skill

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
)

// deniedToolErrorSkill 模拟 ai.agent_tool：执行期被工具运行期策略拒绝。
// 拒绝以 dto.ErrToolPermissionDenied 哨兵返回。
type deniedToolErrorSkill struct{ code string }

func (s *deniedToolErrorSkill) Code() string   { return s.code }
func (s *deniedToolErrorSkill) Name() string   { return s.code }
func (s *deniedToolErrorSkill) Tags() []string { return nil }
func (s *deniedToolErrorSkill) Manifest() dto.SkillManifest {
	return dto.SkillManifest{Name: s.code, Version: "v1", Category: "pilot",
		RequiredPermissions: []string{"ai:read"}}
}
func (s *deniedToolErrorSkill) GetMetrics() dto.SkillMetrics { return dto.SkillMetrics{} }
func (s *deniedToolErrorSkill) Validate(_ interface{}) error { return nil }
func (s *deniedToolErrorSkill) Execute(_ context.Context, _ interface{}) (interface{}, error) {
	return nil, fmt.Errorf("%w: %s", dto.ErrToolPermissionDenied, "AI 主体 agent 未获授权 cmdb:write")
}

// 回归锁：registry 必须用 %w 包装执行错误，否则 handler 无法用
// errors.Is 识别「策略拒绝」，只能把它当内部错误返回 500/5001。
func TestInvokeWithMetrics_PreservesExecErrorChain(t *testing.T) {
	reg := NewSkillRegistry()
	require.NoError(t, reg.Register(&deniedToolErrorSkill{code: "tool.denied"}))

	_, err := reg.InvokeWithMetrics(context.Background(), 1, "tool.denied", map[string]interface{}{})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrSkillInvoke), "必须保留 ErrSkillInvoke")
	require.True(t, errors.Is(err, dto.ErrToolPermissionDenied),
		"必须保留 ErrToolPermissionDenied 哨兵，实际: %v", err)
}
