package orchestrator

import "testing"

// in-flight 集合并发回归锁：mark/isProcessed 与 mark/isInFlight 的配对语义
// （审批 goroutine 收尾时 markProcessed 必须同时清掉 in-flight，防止
// issue 永远留在后台集合里无法重新分派）。用 -race 跑。
func TestInFlightLifecycle(t *testing.T) {
	a := &Agent{
		processedIssues: make(map[int]bool),
		inFlight:        make(map[int]bool),
	}

	a.markInFlight(7)
	if !a.isInFlight(7) {
		t.Fatal("issue 7 must be in-flight after markInFlight")
	}
	if a.isProcessed(7) {
		t.Fatal("issue 7 must not be processed while awaiting review")
	}

	// 审批收尾：markProcessed 同时清 in-flight。
	a.markProcessed(7)
	if a.isInFlight(7) {
		t.Fatal("markProcessed must clear the in-flight flag")
	}
	if !a.isProcessed(7) {
		t.Fatal("issue 7 must be processed after markProcessed")
	}
}
