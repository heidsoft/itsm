package approval_chain

// 域内窄契约：审批链 CRUD 与统计，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

type ApprovalChainService interface {
	CreateApprovalChain(ctx context.Context, req *dto.ApprovalChainRequest, tenantID int) (*ent.ApprovalChain, error)
	UpdateApprovalChain(ctx context.Context, id int, req *dto.ApprovalChainRequest, tenantID int) (*ent.ApprovalChain, error)
	DeleteApprovalChain(ctx context.Context, id int, tenantID int) error
	GetApprovalChain(ctx context.Context, id int, tenantID int) (*ent.ApprovalChain, error)
	GetApprovalChainStats(ctx context.Context, tenantID int) (*dto.ApprovalChainStats, error)
	ListApprovalChains(ctx context.Context, tenantID int, entityType, status, name string, page, pageSize int) ([]*ent.ApprovalChain, int, error)
}
