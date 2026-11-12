package service

import (
	"strings"
	"testing"
)

func TestBuildLegacyApprovalBPMN(t *testing.T) {
	nodes := []map[string]interface{}{
		{"name": "IT总监审批", "assignee_type": "role", "assignee_value": "it_director", "step_order": 2},
		{"name": "经理审批", "assignee_type": "user", "assignee_value": "manager", "step_order": 1},
	}
	xml, err := buildLegacyApprovalBPMN("legacy_1", "变更审批", nodes)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(xml, "经理审批") > strings.Index(xml, "IT总监审批") {
		t.Fatal("nodes were not ordered by step_order")
	}
	for _, want := range []string{`itsm:taskPurpose="approval"`, `itsm:assignee="manager"`, `itsm:candidateGroups="it_director"`} {
		if !strings.Contains(xml, want) {
			t.Fatalf("missing %s", want)
		}
	}
	parsed, err := NewBPMNParser().ParseXML([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Processes[0].UserTasks[0]; got.TaskPurpose != "approval" || got.Assignee != "manager" {
		t.Fatalf("migration attributes were not parsed: %#v", got)
	}
}
