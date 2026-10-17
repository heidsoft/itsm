package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectSLABusinessHours(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]interface{}
		want *SLABusinessHours
	}{
		{
			name: "空配置返回 nil",
			raw:  nil,
			want: nil,
		},
		{
			name: "数据库解码后的形态",
			raw: map[string]interface{}{
				"time_zone":  "Asia/Shanghai",
				"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
				"start_time": "09:00",
				"end_time":   "18:00",
			},
			want: &SLABusinessHours{
				TimeZone:  "Asia/Shanghai",
				WorkDays:  []int{1, 2, 3, 4, 5},
				StartTime: "09:00",
				EndTime:   "18:00",
			},
		},
		{
			name: "模板装配时直接写入的 Go 值",
			raw: map[string]interface{}{
				"time_zone": "Asia/Shanghai",
				"is_24_7":   true,
			},
			want: &SLABusinessHours{TimeZone: "Asia/Shanghai", AlwaysOn: true},
		},
		{
			name: "旧键名仍然可读",
			raw: map[string]interface{}{
				"timezone":        "Europe/London",
				"workdays":        []int{1, 5},
				"work_hour_start": "08:30",
				"work_hour_end":   "17:30",
			},
			want: &SLABusinessHours{
				TimeZone:  "Europe/London",
				WorkDays:  []int{1, 5},
				StartTime: "08:30",
				EndTime:   "17:30",
			},
		},
		{
			name: "is_24_7 字符串写法",
			raw:  map[string]interface{}{"is_24_7": "true"},
			want: &SLABusinessHours{AlwaysOn: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ProjectSLABusinessHours(tc.raw))
		})
	}
}

func TestProjectSLAEscalationLevels(t *testing.T) {
	t.Run("解码与内存两种来源都投影成同一种结构", func(t *testing.T) {
		decoded := map[string]interface{}{
			"levels": []interface{}{
				map[string]interface{}{
					"level":             float64(1),
					"threshold_minutes": float64(15),
					"notify_roles":      []interface{}{"manager", "it_admin"},
				},
			},
		}
		inMemory := map[string]interface{}{
			"levels": []map[string]interface{}{
				{"level": 1, "threshold_minutes": 15, "notify_roles": []string{"manager", "it_admin"}},
			},
		}
		want := []SLAEscalationLevel{{Level: 1, ThresholdMinutes: 15, NotifyRoles: []string{"manager", "it_admin"}}}

		assert.Equal(t, want, ProjectSLAEscalationLevels(decoded))
		assert.Equal(t, want, ProjectSLAEscalationLevels(inMemory))
	})

	t.Run("缺失或非层级条目被跳过", func(t *testing.T) {
		raw := map[string]interface{}{
			"levels": []interface{}{
				"broken",
				map[string]interface{}{"level": float64(2), "threshold_minutes": float64(60)},
			},
		}
		assert.Equal(t, []SLAEscalationLevel{{Level: 2, ThresholdMinutes: 60}}, ProjectSLAEscalationLevels(raw))
	})

	t.Run("未配置升级为 nil", func(t *testing.T) {
		assert.Nil(t, ProjectSLAEscalationLevels(map[string]interface{}{"threshold": 10}))
	})
}

// 存储层是 snake_case JSON，API 边界必须只暴露 camelCase，前端不得再读原始键。
func TestSLADefinitionResponseKeysAreCamelCase(t *testing.T) {
	payload := SLADefinitionResponse{
		ID: 1,
		BusinessHours: ProjectSLABusinessHours(map[string]interface{}{
			"time_zone":  "Asia/Shanghai",
			"start_time": "09:00",
			"end_time":   "18:00",
		}),
		EscalationLevels: ProjectSLAEscalationLevels(map[string]interface{}{
			"levels": []interface{}{
				map[string]interface{}{"level": float64(1), "threshold_minutes": float64(15)},
			},
		}),
	}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	assert.Equal(t, "Asia/Shanghai", decoded["businessHours"].(map[string]interface{})["timeZone"])
	assert.Equal(
		t,
		float64(15),
		decoded["escalationLevels"].([]interface{})[0].(map[string]interface{})["thresholdMinutes"],
	)
	assert.NotContains(t, string(encoded), "threshold_minutes")
	assert.NotContains(t, string(encoded), "time_zone")
	assert.NotContains(t, string(encoded), "escalationRules")
}
