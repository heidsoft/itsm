package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

func TestGetDefaultAssignee_FromVariables(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     client,
		logger:     logger,
		exprEngine: NewExpressionEngine(),
	}

	tests := []struct {
		name      string
		variables map[string]interface{}
		want      string
	}{
		{
			name:      "string assignee",
			variables: map[string]interface{}{"assignee": "42"},
			want:      "42",
		},
		{
			name:      "int assignee",
			variables: map[string]interface{}{"assignee": 7},
			want:      "7",
		},
		{
			name:      "float64 assignee",
			variables: map[string]interface{}{"assignee": float64(3)},
			want:      "3",
		},
		{
			name:      "empty string ignored",
			variables: map[string]interface{}{"assignee": ""},
			want:      "",
		},
		{
			name:      "zero string ignored",
			variables: map[string]interface{}{"assignee": "0"},
			want:      "",
		},
		{
			name:      "zero int ignored",
			variables: map[string]interface{}{"assignee": 0},
			want:      "",
		},
		{
			name:      "no assignee key",
			variables: map[string]interface{}{"other": "value"},
			want:      "",
		},
		{
			name:      "nil variables",
			variables: nil,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := (&processInstanceStub{variables: tt.variables}).toEnt()
			task := &BPMNUserTask{Name: "审批任务"}

			got := engine.getDefaultAssignee(context.Background(), instance, task)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDefaultAssignee_NoChineseKeywordFallback(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     client,
		logger:     logger,
		exprEngine: NewExpressionEngine(),
	}

	instance := (&processInstanceStub{variables: nil, tenantID: 1}).toEnt()

	task := &BPMNUserTask{Name: "审批任务"}
	got := engine.getDefaultAssignee(context.Background(), instance, task)
	require.Empty(t, got, "中文关键词兜底已移除，应返回空字符串")

	task2 := &BPMNUserTask{Name: "处理工单"}
	got2 := engine.getDefaultAssignee(context.Background(), instance, task2)
	require.Empty(t, got2, "中文关键词兜底已移除，应返回空字符串")
}

type processInstanceStub struct {
	variables map[string]interface{}
	tenantID  int
}

func (s *processInstanceStub) toEnt() *ent.ProcessInstance {
	inst := &ent.ProcessInstance{
		TenantID: s.tenantID,
	}
	if s.variables != nil {
		inst.Variables = s.variables
	}
	return inst
}
