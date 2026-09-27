package bpmn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type legacyApprovalCaller struct{ calls int }

func (s *legacyApprovalCaller) SubmitApproval(context.Context, int, int, string, string, *int, int) error {
	s.calls++
	return nil
}

func TestApprovalHandler_RetiredRegisteredTaskNeverSucceeds(t *testing.T) {
	registry := NewCallbackRegistry(nil, zaptest.NewLogger(t).Sugar())
	legacy := &legacyApprovalCaller{}
	registry.SetApprovalService(legacy)
	for _, key := range []string{"approval_task", "approval_handler"} {
		handler := registry.GetHandler(key)
		require.NotNil(t, handler, "retain the registered handler to explicitly fail queued legacy work")
		for _, action := range []string{"approve", "reject", "delegate", "escalate", "unknown", ""} {
			t.Run(key+"/"+action, func(t *testing.T) {
				// Persisted legacy BPMN variables use snake_case; do not add new API fields.
				variables := map[string]interface{}{
					"tenant_id": 7, "user_id": 11, "approval_id": 23,
					"delegate_to_user_id": 13, "action": action, "comment": "private comment",
				}
				ctx := context.WithValue(context.Background(), BPMNTenantIDContextKey, 7)
				for attempt := 0; attempt < 2; attempt++ {
					result, err := handler.Execute(ctx, nil, variables)
					require.ErrorIs(t, err, ErrLegacyApprovalTaskRetired)
					assert.Nil(t, result, "must not mark approval or task completion successful")
					assert.NotContains(t, err.Error(), "private comment")
				}
			})
		}
	}
	assert.Zero(t, legacy.calls, "even an injected legacy writer must never be invoked")
}

func TestApprovalHandler_RetiredWithoutIdentityOrConfiguration(t *testing.T) {
	handler := NewApprovalHandler(nil, zaptest.NewLogger(t).Sugar())
	result, err := handler.Execute(context.Background(), nil, nil)
	require.ErrorIs(t, err, ErrLegacyApprovalTaskRetired)
	assert.Nil(t, result)
	require.ErrorIs(t, handler.Validate(context.Background(), nil), ErrLegacyApprovalTaskRetired)
}
