package marketplace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 回归：回滚与升级共用 UpgradeInstallation，成功文案必须由方向决定，
// 不能一律说「已从 X 升级到 Y」。
func TestCompareVersionParts(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.0", "9.8.7", -1},
		{"9.8.7", "1.4.0", 1},
		{"1.4.0", "1.4.0", 0},
		{"v1.4.0", "1.4.0", 0},
		{"1.10.0", "1.9.9", 1},
		{"1.4", "1.4.0", 0},
		{"2.0.0", "10.0.0", -1},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, compareSemverStrings(c.a, c.b), "%s vs %s", c.a, c.b)
	}
}
