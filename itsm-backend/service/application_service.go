package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/application"
	"itsm-backend/ent/microservice"
)

type ApplicationService struct {
	client *ent.Client
}

func NewApplicationService(client *ent.Client) *ApplicationService {
	return &ApplicationService{client: client}
}

// Application Methods

func (s *ApplicationService) CreateApplication(ctx context.Context, name, code, appType string, projectID, tenantID int) (*ent.Application, error) {
	query := s.client.Application.Create().
		SetName(name).
		SetCode(code).
		SetType(appType).
		SetTenantID(tenantID)

	if projectID > 0 {
		query.SetProjectID(projectID)
	}
	return query.Save(ctx)
}

func (s *ApplicationService) ListApplications(ctx context.Context, tenantID int) ([]*ent.Application, error) {
	return s.client.Application.Query().
		Where(application.TenantID(tenantID)).
		WithMicroservices().
		All(ctx)
}

// Microservice Methods

func (s *ApplicationService) CreateMicroservice(ctx context.Context, name, code, language, framework string, appID, tenantID int) (*ent.Microservice, error) {
	return s.client.Microservice.Create().
		SetName(name).
		SetCode(code).
		SetLanguage(language).
		SetFramework(framework).
		SetApplicationID(appID).
		SetTenantID(tenantID).
		Save(ctx)
}

// UpdateApplication 更新应用
func (s *ApplicationService) UpdateApplication(ctx context.Context, id int, name, code, appType *string, projectID *int, tenantID int) (*ent.Application, error) {
	// 检查应用是否存在且属于当前租户
	_, err := s.client.Application.Query().
		Where(
			application.IDEQ(id),
			application.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Application.UpdateOneID(id)
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		update = update.SetCode(*code)
	}
	if appType != nil {
		update = update.SetType(*appType)
	}
	if projectID != nil {
		update = update.SetProjectID(*projectID)
	}

	return update.Save(ctx)
}

// DeleteApplication 删除应用
func (s *ApplicationService) DeleteApplication(ctx context.Context, id int, tenantID int) error {
	// 检查应用是否存在且属于当前租户
	exists, err := s.client.Application.Query().
		Where(
			application.IDEQ(id),
			application.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("应用不存在: id=%d", id)
	}

	// 删除应用
	return s.client.Application.DeleteOneID(id).Exec(ctx)
}

// ListMicroservices 获取微服务列表
func (s *ApplicationService) ListMicroservices(ctx context.Context, tenantID int) ([]*ent.Microservice, error) {
	return s.client.Microservice.Query().
		Where(microservice.TenantIDEQ(tenantID)).
		All(ctx)
}

// UpdateMicroservice 更新微服务
func (s *ApplicationService) UpdateMicroservice(ctx context.Context, id int, name, code, language, framework *string, appID *int, tenantID int) (*ent.Microservice, error) {
	// 检查微服务是否存在且属于当前租户
	_, err := s.client.Microservice.Query().
		Where(
			microservice.IDEQ(id),
			microservice.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Microservice.UpdateOneID(id)
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		update = update.SetCode(*code)
	}
	if language != nil {
		update = update.SetLanguage(*language)
	}
	if framework != nil {
		update = update.SetFramework(*framework)
	}
	if appID != nil {
		update = update.SetApplicationID(*appID)
	}

	return update.Save(ctx)
}

// DeleteMicroservice 删除微服务
func (s *ApplicationService) DeleteMicroservice(ctx context.Context, id int, tenantID int) error {
	// 检查微服务是否存在且属于当前租户
	exists, err := s.client.Microservice.Query().
		Where(
			microservice.IDEQ(id),
			microservice.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("微服务不存在: id=%d", id)
	}

	// 删除微服务
	return s.client.Microservice.DeleteOneID(id).Exec(ctx)
}
