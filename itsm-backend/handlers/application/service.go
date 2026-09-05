package application

import (
	"context"

	"itsm-backend/ent"
)

// Service 定义应用管理域业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 service.ApplicationService 提供。
// 原 handler 构造函数直接接收 *ent.Client 并内联 new service（依赖注入下沉、
// 无法 mock），现改为依赖域内接口，由组合根负责装配。
type Service interface {
	CreateApplication(ctx context.Context, name, code, appType string, projectID, tenantID int) (*ent.Application, error)
	ListApplications(ctx context.Context, tenantID int) ([]*ent.Application, error)
	UpdateApplication(ctx context.Context, id int, name, code, appType *string, projectID *int, tenantID int) (*ent.Application, error)
	DeleteApplication(ctx context.Context, id int, tenantID int) error
	CreateMicroservice(ctx context.Context, name, code, language, framework string, appID, tenantID int) (*ent.Microservice, error)
	ListMicroservices(ctx context.Context, tenantID int) ([]*ent.Microservice, error)
	UpdateMicroservice(ctx context.Context, id int, name, code, language, framework *string, appID *int, tenantID int) (*ent.Microservice, error)
	DeleteMicroservice(ctx context.Context, id int, tenantID int) error
}