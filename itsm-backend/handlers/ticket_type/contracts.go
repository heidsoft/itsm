package ticket_type

// 域内窄契约：工单类型定义、预设与状态管理，方法签名与生产实现逐一对应。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
)

type TicketTypeService interface {
	CreateTicketType(ctx context.Context, req *dto.CreateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error)
	UpdateTicketType(ctx context.Context, id int, req *dto.UpdateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error)
	DeleteTicketType(ctx context.Context, id, tenantID, userID int) error
	GetTicketType(ctx context.Context, id, tenantID int) (*dto.TicketTypeDefinition, error)
	ListTicketTypes(ctx context.Context, req *dto.ListTicketTypesRequest, tenantID int) (*dto.TicketTypeListResponse, error)
	ListPresets() []dto.TicketTypePreset
	InstallPreset(ctx context.Context, presetID string, req *dto.InstallTicketTypePresetRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error)
	CloneTicketType(ctx context.Context, id, tenantID, userID int, code, name string) (*dto.TicketTypeDefinition, error)
	SetStatus(ctx context.Context, id, tenantID, userID int, status dto.TicketTypeStatus) (*dto.TicketTypeDefinition, error)
	RestoreTicketType(ctx context.Context, id, tenantID, userID int) (*dto.TicketTypeDefinition, error)
}
