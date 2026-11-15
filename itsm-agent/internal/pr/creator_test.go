package pr

import (
	"fmt"
	"reflect"
	"testing"
)

// mergeArgs 回归锁：默认**不得**附带 --admin（分支保护必须继续生效），
// 只有显式 merge_with_admin: true 才允许绕过。旧实现硬编码 --admin，
// 「人工审批」成了唯一闸门、分支保护被静默架空。
func TestMergeArgs(t *testing.T) {
	tests := []struct {
		name      string
		withAdmin bool
		want      []string
	}{
		{
			name:      "default does not bypass branch protection",
			withAdmin: false,
			want:      []string{"pr", "merge", "--repo", "o/r", "--auto", "42"},
		},
		{
			name:      "explicit admin flag is appended",
			withAdmin: true,
			want:      []string{"pr", "merge", "--repo", "o/r", "--admin", "--auto", "42"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeArgs("gh", "o", "r", 42, tt.withAdmin)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("mergeArgs = %v, want %v", got, tt.want)
			}
		})
	}
}

// 构造参数顺序防呆：PR 编号必须出现在末尾（gh 的位置参数）。
func TestMergeArgs_PRNumberPosition(t *testing.T) {
	args := mergeArgs("gh", "o", "r", 1234, false)
	if args[len(args)-1] != fmt.Sprint(1234) {
		t.Fatalf("PR number must be the trailing positional arg, got %v", args)
	}
}
