package service

import (
	"strings"
	"testing"
)

// TestValidateWorkflowDefinition_Cycle 覆盖 W1 环检测
func TestValidateWorkflowDefinition_Cycle(t *testing.T) {
	s := &WorkflowService{}

	cases := []struct {
		name       string
		def        map[string]interface{}
		wantErr    bool
		errContain string
	}{
		{
			name: "直线 DAG 通过",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps": []interface{}{
					map[string]interface{}{"id": "s1"},
					map[string]interface{}{"id": "s2"},
					map[string]interface{}{"id": "s3"},
				},
				"transitions": []interface{}{
					map[string]interface{}{"fromStep": "s1", "toStep": "s2"},
					map[string]interface{}{"fromStep": "s2", "toStep": "s3"},
				},
			},
			wantErr: false,
		},
		{
			name: "自环拒绝",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps": []interface{}{
					map[string]interface{}{"id": "s1"},
				},
				"transitions": []interface{}{
					map[string]interface{}{"fromStep": "s1", "toStep": "s1"},
				},
			},
			wantErr:    true,
			errContain: "存在环",
		},
		{
			name: "两节点互相回边拒绝",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps": []interface{}{
					map[string]interface{}{"id": "s1"},
					map[string]interface{}{"id": "s2"},
				},
				"transitions": []interface{}{
					map[string]interface{}{"fromStep": "s1", "toStep": "s2"},
					map[string]interface{}{"fromStep": "s2", "toStep": "s1"},
				},
			},
			wantErr:    true,
			errContain: "存在环",
		},
		{
			name: "三节点环 s1→s2→s3→s1 拒绝",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps": []interface{}{
					map[string]interface{}{"id": "s1"},
					map[string]interface{}{"id": "s2"},
					map[string]interface{}{"id": "s3"},
				},
				"transitions": []interface{}{
					map[string]interface{}{"fromStep": "s1", "toStep": "s2"},
					map[string]interface{}{"fromStep": "s2", "toStep": "s3"},
					map[string]interface{}{"fromStep": "s3", "toStep": "s1"},
				},
			},
			wantErr:    true,
			errContain: "存在环",
		},
		{
			name: "缺 startEvent",
			def: map[string]interface{}{
				"steps":       []interface{}{map[string]interface{}{"id": "s1"}},
				"transitions": []interface{}{},
			},
			wantErr:    true,
			errContain: "startEvent",
		},
		{
			name: "steps 空",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps":      []interface{}{},
			},
			wantErr:    true,
			errContain: "steps",
		},
		{
			name: "startEvent 未在 steps",
			def: map[string]interface{}{
				"startEvent": "s99",
				"steps":      []interface{}{map[string]interface{}{"id": "s1"}},
			},
			wantErr:    true,
			errContain: "未在 steps",
		},
		{
			name: "step id 重复",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps": []interface{}{
					map[string]interface{}{"id": "s1"},
					map[string]interface{}{"id": "s1"},
				},
			},
			wantErr:    true,
			errContain: "重复",
		},
		{
			name: "transition fromStep 未定义",
			def: map[string]interface{}{
				"startEvent": "s1",
				"steps":      []interface{}{map[string]interface{}{"id": "s1"}},
				"transitions": []interface{}{
					map[string]interface{}{"fromStep": "ghost", "toStep": "s1"},
				},
			},
			wantErr:    true,
			errContain: "fromStep",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.validateWorkflowDefinition(tc.def)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expect error, got nil")
				}
				if tc.errContain != "" && !strings.Contains(err.Error(), tc.errContain) {
					t.Fatalf("error mismatch: want contain %q got %q", tc.errContain, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
