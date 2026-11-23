package bpmn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P1-01 回归（codex 工作流引擎审计 2026-11-23）：
// ResolveTenantID 只认命令认证 context 注入的租户，流程变量中的 tenant_id
// 仅允许作为同值副本存在，冲突即拒绝；缺 context 时即使变量带 tenant_id
// 也拒绝——旧实现「变量优先」允许租户 A 的流程变量把 handler 指向租户 B。

func TestResolveTenantID_ContextOnly(t *testing.T) {
	t.Run("缺 context 时变量带 tenant_id 也拒绝", func(t *testing.T) {
		id, err := ResolveTenantID(context.Background(), map[string]interface{}{"tenant_id": 42})
		require.Error(t, err, "变量不得成为租户来源")
		assert.Zero(t, id)
		assert.Contains(t, err.Error(), "认证租户上下文")
	})

	t.Run("context 与变量同值时放行", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 7)
		id, err := ResolveTenantID(ctx, map[string]interface{}{"tenant_id": 7})
		require.NoError(t, err)
		assert.Equal(t, 7, id)
	})

	t.Run("变量租户与 context 冲突时拒绝", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 7)
		id, err := ResolveTenantID(ctx, map[string]interface{}{"tenant_id": 999})
		require.Error(t, err, "跨租户变量必须 fail closed")
		assert.Zero(t, id)
		assert.Contains(t, err.Error(), "冲突")
		assert.Contains(t, err.Error(), "999")
	})

	t.Run("context-only 无变量也放行", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 7)
		id, err := ResolveTenantID(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, 7, id)
	})

	t.Run("context 值非法时拒绝", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 0)
		id, err := ResolveTenantID(ctx, map[string]interface{}{"tenant_id": 1})
		require.Error(t, err)
		assert.Zero(t, id)
	})
}
