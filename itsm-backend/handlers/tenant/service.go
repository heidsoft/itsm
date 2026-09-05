package tenant

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// Service 定义 tenant 域的业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 service.TenantService 提供；
// 测试可注入 mock 实现，使 handler 逻辑可独立验证。
// 与 user 域同属"58 裸奔域迁移样板"：域内自持接口，解耦遗留 service 包。
type Service interface {
	CreateTenant(ctx context.Context, req *dto.CreateTenantRequest) (*ent.Tenant, error)
	ListTenants(ctx context.Context, req *dto.ListTenantsRequest) ([]*ent.Tenant, int, error)
	GetTenant(ctx context.Context, tenantID int) (*ent.Tenant, error)
	UpdateTenant(ctx context.Context, tenantID int, req *dto.UpdateTenantRequest) (*ent.Tenant, error)
	UpdateTenantStatus(ctx context.Context, tenantID int, status string) error
	DeleteTenant(ctx context.Context, tenantID int) error
}
