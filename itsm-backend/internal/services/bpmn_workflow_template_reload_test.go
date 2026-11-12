package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBumpMinorVersion 覆盖 bumpMinorVersion 的语义：major 不变、minor+1、patch 归零，
// 与 (tenant_id,key,version) 唯一索引兼容；不可解析片段回退为 1.0.0，避免撞号。
func TestBumpMinorVersion(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string returns 1.0.0", input: "", want: "1.0.0"},
		{name: "first deployment 1.0.0", input: "1.0.0", want: "1.1.0"},
		{name: "increments minor", input: "1.5.0", want: "1.6.0"},
		{name: "preserves major", input: "2.3.4", want: "2.4.0"},
		{name: "pads minor with double digit", input: "1.10.0", want: "1.11.0"},
		{name: "unparseable returns 1.0.0", input: "abc", want: "1.0.0"},
		{name: "only major parses", input: "3", want: "3.1.0"},
		{name: "negative minor clamps to 0+1", input: "1.-5.0", want: "1.1.0"},
		{name: "whitespace tolerated", input: " 1.7.0 ", want: "1.8.0"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := bumpMinorVersion(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBumpMinorVersion_NeverConflictsAtCurrentSequence 验证 bumpMinorVersion
// 在连续调用下不会产生重复版本号（与唯一索引兼容）。
func TestBumpMinorVersion_NeverConflictsAtCurrentSequence(t *testing.T) {
	current := ""
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		next := bumpMinorVersion(current)
		if seen[next] {
			t.Fatalf("duplicate version emitted: %s at iteration %d", next, i)
		}
		seen[next] = true
		current = next
	}
}
