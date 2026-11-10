package dashboard

import (
	"testing"

	"itsm-backend/dto"
)

// 窗口上限常量下沉到 dto 后，parseQualityWindow 的行为必须逐字不变：
// 默认 30、非法/非正值回落 30、超上限钳制到 dto.MaxQualityWindowDays。
// 只锁「源码里引用了 dto 常量」不够——常量值与解析逻辑一起改坏时源码守卫
// 仍会绿，故这里用固定输入做行为等价锁。
func TestParseQualityWindowUsesDTOUpperBound(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 30},
		{"7", 7},
		{"0", 30},
		{"-3", 30},
		{"abc", 30},
		{"999", 90},
	}
	for _, c := range cases {
		if got := parseQualityWindow(c.raw); got != c.want {
			t.Errorf("parseQualityWindow(%q) = %d，期望 %d", c.raw, got, c.want)
		}
	}
	if dto.MaxQualityWindowDays != 90 {
		t.Errorf("窗口上限常量值漂移：%d，期望 90", dto.MaxQualityWindowDays)
	}
}
