package bpmn

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type mockApprovalChainResolver struct {
	chainID      int
	levelsJSON   []byte
	passed       bool
	pendingLevel int
	blocked      bool
	err          error
	calls        int
	lastArgs     resolveCallArgs
}

type resolveCallArgs struct {
	tenantID   int
	entityType string
	requester  int
	priority   string
	amount     float64
	approvals  map[int][]int
}

func (m *mockApprovalChainResolver) ResolveApprovalPlanRaw(
	ctx context.Context,
	tenantID int,
	entityType string,
	requesterID int,
	priority string,
	amount float64,
	approvals map[int][]int,
) (int, []byte, bool, int, bool, error) {
	m.calls++
	m.lastArgs = resolveCallArgs{
		tenantID: tenantID, entityType: entityType, requester: requesterID,
		priority: priority, amount: amount, approvals: approvals,
	}
	return m.chainID, m.levelsJSON, m.passed, m.pendingLevel, m.blocked, m.err
}

func TestApprovalChainHandler_Registration(t *testing.T) {
	registry := NewCallbackRegistry(nil, zaptest.NewLogger(t).Sugar())
	for _, key := range []string{"approval_chain_handler", "approval_chain_task"} {
		h := registry.GetHandler(key)
		require.NotNil(t, h, "handler must be reachable by %s", key)
		assert.Equal(t, "approval_chain_task", h.GetTaskType())
		assert.Equal(t, "approval_chain_handler", h.GetHandlerID())
	}
}

func TestApprovalChainHandler_NoResolverReturnsError(t *testing.T) {
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	vars := map[string]interface{}{
		"tenant_id":    1,
		"entity_type":  "ticket",
		"requester_id": 10,
		"action":       "resolve_plan",
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 1)
	result, err := h.Execute(ctx, nil, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批链求值器未注入")
	assert.Nil(t, result)
}

func TestApprovalChainHandler_UnknownAction(t *testing.T) {
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	h.SetApprovalChainResolver(&mockApprovalChainResolver{})
	vars := map[string]interface{}{
		"tenant_id":    1,
		"entity_type":  "ticket",
		"requester_id": 10,
		"action":       "do_something_else",
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 1)
	result, err := h.Execute(ctx, nil, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知动作")
	assert.Nil(t, result)
}

func TestApprovalChainHandler_MissingEntityType(t *testing.T) {
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	h.SetApprovalChainResolver(&mockApprovalChainResolver{})
	vars := map[string]interface{}{
		"tenant_id":    1,
		"requester_id": 10,
		"action":       "resolve_plan",
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 1)
	_, err := h.Execute(ctx, nil, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entity_type")
}

func TestApprovalChainHandler_MissingRequesterID(t *testing.T) {
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	h.SetApprovalChainResolver(&mockApprovalChainResolver{})
	vars := map[string]interface{}{
		"tenant_id":   1,
		"entity_type": "ticket",
		"action":      "resolve_plan",
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 1)
	_, err := h.Execute(ctx, nil, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requester_id")
}

func TestApprovalChainHandler_ResolvePlanSuccess(t *testing.T) {
	levels := []map[string]interface{}{{"level": 1, "approvers": []string{"manager"}}}
	levelsRaw, _ := json.Marshal(levels)

	mock := &mockApprovalChainResolver{
		chainID:      42,
		levelsJSON:   levelsRaw,
		passed:       false,
		pendingLevel: 1,
		blocked:      false,
	}
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	h.SetApprovalChainResolver(mock)

	vars := map[string]interface{}{
		"tenant_id":    5,
		"entity_type":  "change",
		"requester_id": 20,
		"priority":     "high",
		"amount":       15000.0,
		"action":       "resolve_plan",
		"approvals": map[string]interface{}{
			"1": []interface{}{float64(101), float64(102)},
		},
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 5)
	result, err := h.Execute(ctx, nil, vars)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "审批链求值完成", result.Message)

	assert.Equal(t, 42, result.OutputVars["chain_id"])
	assert.Equal(t, string(levelsRaw), result.OutputVars["levels"])
	assert.Equal(t, false, result.OutputVars["passed"])
	assert.Equal(t, 1, result.OutputVars["pending_level"])
	assert.Equal(t, false, result.OutputVars["blocked"])

	assert.Equal(t, 1, mock.calls)
	assert.Equal(t, 5, mock.lastArgs.tenantID)
	assert.Equal(t, "change", mock.lastArgs.entityType)
	assert.Equal(t, 20, mock.lastArgs.requester)
	assert.Equal(t, "high", mock.lastArgs.priority)
	assert.Equal(t, 15000.0, mock.lastArgs.amount)
	assert.Equal(t, map[int][]int{1: {101, 102}}, mock.lastArgs.approvals)
}

func TestApprovalChainHandler_ResolverErrorBlocksProcess(t *testing.T) {
	mock := &mockApprovalChainResolver{
		err: assert.AnError,
	}
	h := NewApprovalChainServiceTaskHandler(nil, zaptest.NewLogger(t).Sugar())
	h.SetApprovalChainResolver(mock)

	vars := map[string]interface{}{
		"tenant_id":    1,
		"entity_type":  "ticket",
		"requester_id": 10,
		"action":       "resolve_plan",
	}
	ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 1)
	result, err := h.Execute(ctx, nil, vars)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "流程阻塞")
	assert.Nil(t, result)
	assert.Equal(t, 1, mock.calls)
}

func TestDecodeApprovals(t *testing.T) {
	raw := map[string]interface{}{
		"1": []interface{}{float64(10), float64(20)},
		"2": []interface{}{float64(30)},
	}
	var out map[int][]int
	err := decodeApprovals(raw, &out)
	require.NoError(t, err)
	assert.Equal(t, map[int][]int{1: {10, 20}, 2: {30}}, out)
}

func TestDecodeApprovals_NonMapIsNoop(t *testing.T) {
	var out map[int][]int
	err := decodeApprovals("not-a-map", &out)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestApprovalChainHandler_SetViaRegistry(t *testing.T) {
	registry := NewCallbackRegistry(nil, zaptest.NewLogger(t).Sugar())
	mock := &mockApprovalChainResolver{chainID: 99, levelsJSON: []byte("[]"), passed: true}
	registry.SetApprovalChainResolver(mock)

	h := registry.GetHandler("approval_chain_handler")
	require.NotNil(t, h)
	ah := h.(*ApprovalChainServiceTaskHandler)
	assert.NotNil(t, ah.resolver, "resolver must be injected via registry setter")
}
