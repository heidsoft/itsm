package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimerType_Constants(t *testing.T) {
	assert.Equal(t, TimerType("start"), TimerTypeStart)
	assert.Equal(t, TimerType("intermediate"), TimerTypeIntermediate)
	assert.Equal(t, TimerType("boundary"), TimerTypeBoundary)
	assert.Equal(t, TimerType("task_due"), TimerTypeTaskDue)
}

func TestTimerStatus_Constants(t *testing.T) {
	assert.Equal(t, TimerStatus("pending"), TimerStatusPending)
	assert.Equal(t, TimerStatus("fired"), TimerStatusFired)
	assert.Equal(t, TimerStatus("cancelled"), TimerStatusCancelled)
	assert.Equal(t, TimerStatus("failed"), TimerStatusFailed)
}

func TestCreateTimerRequest_Fields(t *testing.T) {
	pid := 42
	req := CreateTimerRequest{
		TimerType:            TimerTypeBoundary,
		ProcessDefinitionKey: "test_flow",
		ProcessInstanceID:    &pid,
		ActivityID:           "act1",
		TimerExpression:      "PT10M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now(),
		ContextVariables:     map[string]interface{}{"key": "val"},
		TenantID:             1,
	}
	assert.Equal(t, TimerTypeBoundary, req.TimerType)
	assert.Equal(t, "test_flow", req.ProcessDefinitionKey)
	assert.Equal(t, &pid, req.ProcessInstanceID)
	assert.Equal(t, 1, req.TenantID)
}

func TestTimerListFilter_Fields(t *testing.T) {
	f := TimerListFilter{
		TenantID:             1,
		Status:               "pending",
		TimerType:            "start",
		ProcessDefinitionKey: "flow",
		Page:                 1,
		PageSize:             20,
	}
	assert.Equal(t, 1, f.TenantID)
	assert.Equal(t, "pending", f.Status)
	assert.Equal(t, 20, f.PageSize)
}
