package approval

// 域内窄契约：审批工作流管理与旧流程迁移，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
)

type ApprovalService interface {
	CreateWorkflow(ctx context.Context, req *dto.CreateApprovalWorkflowRequest, tenantID int) (*dto.ApprovalWorkflowResponse, error)
	UpdateWorkflow(ctx context.Context, id int, req *dto.UpdateApprovalWorkflowRequest, tenantID int) (*dto.ApprovalWorkflowResponse, error)
	DeleteWorkflow(ctx context.Context, id int, tenantID int) error
	GetWorkflow(ctx context.Context, id int, tenantID int) (*dto.ApprovalWorkflowResponse, error)
	ListWorkflows(ctx context.Context, filter *dto.WorkflowListFilter, tenantID int, page, pageSize int) ([]*dto.ApprovalWorkflowResponse, int, error)
	GetApprovalRecords(ctx context.Context, req *dto.GetApprovalRecordsRequest, tenantID int) ([]*dto.ApprovalRecordResponse, int, error)
	MigrateWorkflowToBPMN(ctx context.Context, workflowID, tenantID int, dryRun bool) (*dto.LegacyApprovalMigrationResult, error)
}
