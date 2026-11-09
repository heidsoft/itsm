package release

import (
	"context"

	"itsm-backend/dto"
)

// ReleaseService 是 release 域对发布管理服务的窄契约，方法集等于本域所有
// 生产文件对 releaseService 字段的真实调用面（8 个方法，含状态流转与审批）。
//
// 请求/响应结构全部位于 dto 包；ErrInvalidReleaseTransition sentinel 也已下沉到
// dto 并由共享层以变量别名暴露，errors.Is 仍命中同一实例，本文件无需引用共享层。
type ReleaseService interface {
	ApplyReleaseApproval(ctx context.Context, id, tenantID, actorID int, action, comment string) (*dto.ReleaseResponse, error)
	CreateRelease(ctx context.Context, req *dto.CreateReleaseRequest, createdBy, tenantID int) (*dto.ReleaseResponse, error)
	DeleteRelease(ctx context.Context, id, tenantID int, actorID int, actorRole string) error
	GetReleaseByID(ctx context.Context, id, tenantID int) (*dto.ReleaseResponse, error)
	GetReleaseStats(ctx context.Context, tenantID int) (*dto.ReleaseStatsResponse, error)
	ListReleases(ctx context.Context, tenantID int, page, pageSize int, status, releaseType string, currentUserID int, currentRole string) (*dto.ReleaseListResponse, error)
	UpdateRelease(ctx context.Context, id, tenantID int, req *dto.UpdateReleaseRequest, actorID int, actorRole string) (*dto.ReleaseResponse, error)
	UpdateReleaseStatus(ctx context.Context, id, tenantID int, status string, actorID int, actorRole string) (*dto.ReleaseResponse, error)
}
