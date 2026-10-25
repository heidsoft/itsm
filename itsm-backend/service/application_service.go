package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/application"
	"itsm-backend/ent/microservice"
	"itsm-backend/ent/project"
)

type ApplicationService struct {
	client *ent.Client
}

func NewApplicationService(client *ent.Client) *ApplicationService {
	return &ApplicationService{client: client}
}

// validateProjectTenant 校验项目引用属于当前租户，避免写入跨租户外键。
func (s *ApplicationService) validateProjectTenant(ctx context.Context, projectID, tenantID int) error {
	if projectID <= 0 {
		return nil
	}
	exists, err := s.client.Project.Query().
		Where(project.IDEQ(projectID), project.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查所属项目失败: %w", err)
	}
	if !exists {
		return common.NewBusinessError(common.ParamErrorCode, "所属项目不存在或不属于当前租户", "")
	}
	return nil
}

// validateApplicationTenant 校验应用引用属于当前租户（且未被软删除）。
func (s *ApplicationService) validateApplicationTenant(ctx context.Context, appID, tenantID int) error {
	if appID <= 0 {
		return nil
	}
	exists, err := s.client.Application.Query().
		Where(application.IDEQ(appID), application.TenantIDEQ(tenantID), application.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查所属应用失败: %w", err)
	}
	if !exists {
		return common.NewBusinessError(common.ParamErrorCode, "所属应用不存在或不属于当前租户", "")
	}
	return nil
}

// Application Methods

func (s *ApplicationService) CreateApplication(ctx context.Context, name, code, appType string, projectID, tenantID int) (*ent.Application, error) {
	if err := s.validateProjectTenant(ctx, projectID, tenantID); err != nil {
		return nil, err
	}
	codeExists, err := s.client.Application.Query().
		Where(application.CodeEQ(code), application.TenantIDEQ(tenantID), application.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查应用代码失败: %w", err)
	}
	if codeExists {
		return nil, fmt.Errorf("应用代码已存在: %s", code)
	}
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
		Where(application.TenantID(tenantID), application.DeletedAtIsNil()).
		WithMicroservices().
		All(ctx)
}

// Microservice Methods

func (s *ApplicationService) CreateMicroservice(ctx context.Context, name, code, language, framework string, appID, tenantID int) (*ent.Microservice, error) {
	if err := s.validateApplicationTenant(ctx, appID, tenantID); err != nil {
		return nil, err
	}
	create := s.client.Microservice.Create().
		SetName(name).
		SetCode(code).
		SetLanguage(language).
		SetFramework(framework).
		SetTenantID(tenantID)
	if appID > 0 {
		create = create.SetApplicationID(appID)
	}
	return create.Save(ctx)
}

// UpdateApplication 更新应用
func (s *ApplicationService) UpdateApplication(ctx context.Context, id int, name, code, appType *string, projectID *int, tenantID int) (*ent.Application, error) {
	// 检查应用是否存在且属于当前租户
	_, err := s.client.Application.Query().
		Where(
			application.IDEQ(id),
			application.TenantIDEQ(tenantID),
			application.DeletedAtIsNil(),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	if projectID != nil {
		if err := s.validateProjectTenant(ctx, *projectID, tenantID); err != nil {
			return nil, err
		}
	}

	update := s.client.Application.UpdateOneID(id).
		Where(application.DeletedAtIsNil(), application.TenantIDEQ(tenantID))
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		codeExists, err := s.client.Application.Query().
			Where(application.CodeEQ(*code), application.TenantIDEQ(tenantID), application.DeletedAtIsNil(), application.IDNEQ(id)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查应用代码失败: %w", err)
		}
		if codeExists {
			return nil, fmt.Errorf("应用代码已存在: %s", *code)
		}
		update = update.SetCode(*code)
	}
	if appType != nil {
		update = update.SetType(*appType)
	}
	if projectID != nil {
		if *projectID == 0 {
			update = update.ClearProjectID()
		} else {
			update = update.SetProjectID(*projectID)
		}
	}

	updated, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewBusinessError(common.NotFoundCode, "应用不存在", "")
		}
		return nil, fmt.Errorf("更新应用失败: %w", err)
	}
	return updated, nil
}

// DeleteApplication 删除应用（软删除，租户条件更新）
func (s *ApplicationService) DeleteApplication(ctx context.Context, id int, tenantID int) error {
	_, err := s.client.Application.UpdateOneID(id).
		Where(application.TenantIDEQ(tenantID), application.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return common.NewBusinessError(common.NotFoundCode, "应用不存在", "")
		}
		return fmt.Errorf("删除应用失败: %w", err)
	}
	return nil
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

	if appID != nil {
		if err := s.validateApplicationTenant(ctx, *appID, tenantID); err != nil {
			return nil, err
		}
	}

	update := s.client.Microservice.UpdateOneID(id).Where(microservice.TenantIDEQ(tenantID))
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
		if *appID == 0 {
			update = update.ClearApplicationID()
		} else {
			update = update.SetApplicationID(*appID)
		}
	}

	updated, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewBusinessError(common.NotFoundCode, "微服务不存在", "")
		}
		return nil, fmt.Errorf("更新微服务失败: %w", err)
	}
	return updated, nil
}

// DeleteMicroservice 删除微服务
func (s *ApplicationService) DeleteMicroservice(ctx context.Context, id int, tenantID int) error {
	deleted, err := s.client.Microservice.Delete().
		Where(microservice.IDEQ(id), microservice.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除微服务失败: %w", err)
	}
	if deleted == 0 {
		return common.NewBusinessError(common.NotFoundCode, "微服务不存在", "")
	}
	return nil
}
