package human

import "testing"

// approvedFromPRView 回归锁：审批判定的三种批准来源（merged / APPROVED
// review / auto-merge clean）与默认不批准语义。人工审批是这个 agent 的
// 核心安全闸门——判定逻辑的变化必须有测试锁住。
func TestApprovedFromPRView(t *testing.T) {
	tests := []struct {
		name             string
		state            string
		mergeStateStatus string
		reviews          []PRReviewState
		want             bool
	}{
		{name: "merged via mergeStateStatus", state: "open", mergeStateStatus: "merged", want: true},
		{name: "merged via state", state: "merged", mergeStateStatus: "behind", want: true},
		{name: "approved review", state: "open", mergeStateStatus: "blocked",
			reviews: []PRReviewState{{State: "APPROVED"}}, want: true},
		{name: "changes requested is not approved", state: "open", mergeStateStatus: "blocked",
			reviews: []PRReviewState{{State: "CHANGES_REQUESTED"}}, want: false},
		{name: "auto-merge clean", state: "open", mergeStateStatus: "clean", want: true},
		{name: "clean with rejected review still approved", state: "open", mergeStateStatus: "clean",
			reviews: []PRReviewState{{State: "REJECTED"}}, want: true,
		},
		{name: "pending without review", state: "open", mergeStateStatus: "pending", want: false},
		{name: "empty gh output", state: "", mergeStateStatus: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := approvedFromPRView(tt.state, tt.mergeStateStatus, tt.reviews); got != tt.want {
				t.Fatalf("approvedFromPRView(%q, %q, %v) = %v, want %v",
					tt.state, tt.mergeStateStatus, tt.reviews, got, tt.want)
			}
		})
	}
}
