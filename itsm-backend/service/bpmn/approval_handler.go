package bpmn

import (
	"context"
	"errors"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// ErrLegacyApprovalTaskRetired prevents automatic decisions through legacy service tasks.
var ErrLegacyApprovalTaskRetired = errors.New("legacy approval_task is retired; use BPMN user tasks and task decisions")

// ApprovalServiceInterface preserves the registry's legacy injection contract.
// Deprecated: approval_task no longer calls a legacy approval service.
type ApprovalServiceInterface interface {
	SubmitApproval(ctx context.Context, recordID int, userID int, action string, comment string, delegateToUserID *int, tenantID int) error
}

// ApprovalHandler is a registered tombstone, not an automated approval engine.
// Keeping it registered makes queued legacy work fail visibly instead of being
// silently completed or falling back to another handler.
type ApprovalHandler struct {
	HandlerBase
}

// NewApprovalHandler retains constructor compatibility without keeping a writer.
func NewApprovalHandler(_ *ent.Client, _ *zap.SugaredLogger) *ApprovalHandler {
	return &ApprovalHandler{}
}

// SetApprovalService is retained for old registry wiring; it cannot enable writes.
// Deprecated: use BPMN user tasks and task decisions, not approval_task.
func (h *ApprovalHandler) SetApprovalService(_ ApprovalServiceInterface) {}

func (h *ApprovalHandler) GetTaskType() string {
	return "approval_task"
}

func (h *ApprovalHandler) GetHandlerID() string {
	return "approval_handler"
}

// Execute rejects every action, including unknown actions, without reading
// untrusted variables or returning success. The worker retains its normal
// retry/dead-letter behavior; no legacy record is changed or auto-migrated.
func (h *ApprovalHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	return nil, ErrLegacyApprovalTaskRetired
}

func (h *ApprovalHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return ErrLegacyApprovalTaskRetired
}

var _ ServiceTaskHandlerInterface = (*ApprovalHandler)(nil)
