package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/project"
	entuser "itsm-backend/ent/user"
)

// ErrProjectNotFound 表示按租户谓词查不到项目（不存在或跨租户）。
// 条件删除 0 行不再静默假装成功（B10 批，Z1 同型收敛）。
var ErrProjectNotFound = fmt.Errorf("project not found in current tenant")

type ProjectService struct {
	client *ent.Client
}

func NewProjectService(client *ent.Client) *ProjectService {
	return &ProjectService{client: client}
}

// validateRefTenant 校验管理器 / 部门引用属于当前租户。
// 引用型资源缺失（不存在或跨租户）按业务参数错误 400/1001 处理，
// 与工单评论「提及用户无效」同口径——不再静默落库跨租户引用。
func (s *ProjectService) validateRefTenant(ctx context.Context, tenantID, managerID, deptID int) error {
	if managerID != 0 {
		exists, err := s.client.User.Query().
			Where(entuser.ID(managerID), entuser.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check project manager: %w", err)
		}
		if !exists {
			return common.NewBusinessError(common.ParamErrorCode, "项目经理不存在或不属于当前租户", "")
		}
	}
	if deptID != 0 {
		exists, err := s.client.Department.Query().
			Where(department.ID(deptID), department.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check project department: %w", err)
		}
		if !exists {
			return common.NewBusinessError(common.ParamErrorCode, "所属部门不存在或不属于当前租户", "")
		}
	}
	return nil
}

// CreateProject 创建项目
func (s *ProjectService) CreateProject(ctx context.Context, name, code string, deptID, managerID, tenantID int) (*ent.Project, error) {
	if err := s.validateRefTenant(ctx, tenantID, managerID, deptID); err != nil {
		return nil, err
	}
	create := s.client.Project.Create().
		SetName(name).
		SetCode(code).
		SetTenantID(tenantID).
		SetStartDate(time.Now())
	// 0 = 未指定：manager_id/department_id 是 Optional 字段，显式 Set 0 会
	// 写出指向不存在部门的引用值（FK 严格模式下直接 500）。
	if managerID != 0 {
		create = create.SetManagerID(managerID)
	}
	if deptID != 0 {
		create = create.SetDepartmentID(deptID)
	}
	return create.Save(ctx)
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
	// 条件删除并以受影响行数为准：0 行=不存在或跨租户，映射 404/4004，
	// 不再静默假装成功（此前裸「项目不存在: id=..」还把资源 id 带进错误文本兜底 500）。
	deleted, err := s.client.Project.Delete().
		Where(project.IDEQ(id), project.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete project %d: %w", id, err)
	}
	if deleted == 0 {
		return common.NewBusinessError(common.NotFoundCode, "项目不存在", "")
	}

	return nil
}

// GetProject 获取单个项目
func (s *ProjectService) GetProject(ctx context.Context, id int, tenantID int) (*ent.Project, error) {
	return s.client.Project.Query().
		Where(
			project.IDEQ(id),
			project.TenantIDEQ(tenantID),
		).
		First(ctx)
}
