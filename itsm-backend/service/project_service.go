package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/project"
)

type ProjectService struct {
	client *ent.Client
}

func NewProjectService(client *ent.Client) *ProjectService {
	return &ProjectService{client: client}
}

// CreateProject 创建项目
func (s *ProjectService) CreateProject(ctx context.Context, name, code string, deptID, managerID, tenantID int) (*ent.Project, error) {
	return s.client.Project.Create().
		SetName(name).
		SetCode(code).
		SetDepartmentID(deptID).
		SetManagerID(managerID).
		SetTenantID(tenantID).
		SetStartDate(time.Now()).
		Save(ctx)
}

// ListProjects 获取项目列表
func (s *ProjectService) ListProjects(ctx context.Context, tenantID int) ([]*ent.Project, error) {
	return s.client.Project.Query().
		Where(project.TenantID(tenantID)).
		All(ctx)
}

// UpdateProject 更新项目
func (s *ProjectService) UpdateProject(ctx context.Context, id int, name, code *string, deptID, managerID *int, tenantID int) (*ent.Project, error) {
	// 检查项目是否存在且属于当前租户
	_, err := s.client.Project.Query().
		Where(
			project.IDEQ(id),
			project.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Project.UpdateOneID(id)
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		update = update.SetCode(*code)
	}
	if deptID != nil {
		update = update.SetDepartmentID(*deptID)
	}
	if managerID != nil {
		update = update.SetManagerID(*managerID)
	}

	return update.Save(ctx)
}

// DeleteProject 删除项目
func (s *ProjectService) DeleteProject(ctx context.Context, id int, tenantID int) error {
	// 检查项目是否存在且属于当前租户
	exists, err := s.client.Project.Query().
		Where(
			project.IDEQ(id),
			project.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("项目不存在: id=%d", id)
	}

	// 删除项目
	return s.client.Project.DeleteOneID(id).Exec(ctx)
}
