package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterEngineVariables(t *testing.T) {
	t.Run("nil 输入返回 nil，而不是空 map", func(t *testing.T) {
		assert.Nil(t, FilterEngineVariables(nil))
	})

	t.Run("空 map 返回空 map", func(t *testing.T) {
		got := FilterEngineVariables(map[string]interface{}{})
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("只剔除下划线前缀的引擎控制键", func(t *testing.T) {
		got := FilterEngineVariables(map[string]interface{}{
			"_done_":             []string{"startEvent"},
			"_commandId":         "cmd-1",
			"_idempotencyKey":    "idem-1",
			"_compensationOrder": 1,
			"ticketId":           "42",
			"assignee_id":        "7",
			"priority":           "",
			"count":              0,
		})
		assert.Equal(t, map[string]interface{}{
			"ticketId":    "42",
			"assignee_id": "7",
			"priority":    "",
			"count":       0,
		}, got, "业务变量（含外部连接器写入的 snake_case 键）必须原样保留")
	})

	t.Run("返回副本，不得改写输入", func(t *testing.T) {
		input := map[string]interface{}{"_done_": true, "ticketId": "42"}
		got := FilterEngineVariables(input)
		assert.Contains(t, input, "_done_", "过滤不得破坏引擎读取的原始变量")
		assert.Len(t, got, 1)
	})
}
