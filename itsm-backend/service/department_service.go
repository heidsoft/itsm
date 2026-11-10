package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/department"
	"itsm-backend/ent/user"
)

type DepartmentService struct {
	client *ent.Client
}

func NewDepartmentService(client *ent.Client) *DepartmentService {
	return &DepartmentService{client: client}
}

// GetDepartmentByID 根据ID获取部门
func (s *DepartmentService) GetDepartmentByID(ctx context.Context, id, tenantID int) (*ent.Department, error) {
	dept, err := s.client.Department.Query().
		Where(
			department.IDEQ(id),
			department.TenantIDEQ(tenantID),
			department.DeletedAtIsNil(),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("Department does not exist: id=%d", id)
		}
		return nil, fmt.Errorf("Failed to get department: %w", err)
	}
	return dept, nil
}

// Department Methods

func (s *DepartmentService) CreateDepartment(ctx context.Context, name, code, description string, managerID, parentID, tenantID int) (*ent.Department, error) {
	name, code = strings.TrimSpace(name), strings.TrimSpace(code)
	if name == "" || code == "" {
		return nil, fmt.Errorf("Department name and code cannot be empty")
	}
	codeExists, err := s.client.Department.Query().
		Where(department.CodeEQ(code), department.TenantIDEQ(tenantID), department.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to check department code: %w", err)
	}
	if codeExists {
		return nil, fmt.Errorf("Department code already exists: %s", code)
	}
	if parentID > 0 {
		if _, err := s.GetDepartmentByID(ctx, parentID, tenantID); err != nil {
			return nil, fmt.Errorf("The parent department does not exist or does not belong to the current tenant")
		}
	}
	if managerID > 0 {
		exists, err := s.client.User.Query().Where(user.ID(managerID), user.TenantID(tenantID), user.Active(true)).Exist(ctx)
		if err != nil || !exists {
			return nil, fmt.Errorf("The department head does not exist, is inactive, or does not belong to the current tenant")
		}
	}
	query := s.client.Department.Create().
		SetName(name).
		SetCode(code).
		SetDescription(description).
		SetTenantID(tenantID)

	if managerID > 0 {
		query.SetManagerID(managerID)
	}
	if parentID > 0 {
		query.SetParentID(parentID)
	}

	return query.Save(ctx)
}

func (s *DepartmentService) GetDepartmentTree(ctx context.Context, tenantID int) ([]*ent.Department, error) {
	// 1. 获取该租户下所有部门
	allDepts, err := s.client.Department.Query().
		Where(department.TenantID(tenantID), department.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 构建Map以便快速查找
	deptMap := make(map[int]*ent.Department)
	for _, d := range allDepts {
		// 初始化 Children 边，防止为 nil
		d.Edges.Children = []*ent.Department{}
		deptMap[d.ID] = d
	}

	// 3. 组装树结构
	var roots []*ent.Department
	for _, d := range allDepts {
		// 检查 ParentID 是否存在且不为0 (假设0表示根节点)
		// 如果 schema 定义为 Optional() 但非 Nillable，则是 int 类型，默认 0
		// 如果是 Optional() + Nillable()，则是 *int
		// 这里假设是 int 类型
		if d.ParentID == 0 {
			roots = append(roots, d)
		} else {
			if parent, ok := deptMap[d.ParentID]; ok {
				parent.Edges.Children = append(parent.Edges.Children, d)
			} else {
				// 如果找不到父节点（可能是数据不一致），暂且作为根节点处理
				roots = append(roots, d)
			}
		}
	}

	return roots, nil
}

// UpdateDepartment 更新部门
func (s *DepartmentService) UpdateDepartment(ctx context.Context, id int, name, code, description string, managerID, parentID, tenantID int) (*ent.Department, error) {
	// 先获取原始部门信息
	oldDept, err := s.client.Department.Query().
		Where(
			department.IDEQ(id),
			department.TenantIDEQ(tenantID),
			department.DeletedAtIsNil(),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	// 如果字段为空，保留原值
	if name == "" {
		name = oldDept.Name
	}
	if code == "" {
		code = oldDept.Code
	}
	codeExists, err := s.client.Department.Query().
		Where(department.CodeEQ(code), department.TenantIDEQ(tenantID), department.DeletedAtIsNil(), department.IDNEQ(id)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to check department code: %w", err)
	}
	if codeExists {
		return nil, fmt.Errorf("Department code already exists: %s", code)
	}
	if description == "" {
		description = oldDept.Description
	}
	if managerID <= 0 {
		managerID = oldDept.ManagerID
	}
	if parentID < 0 {
		parentID = oldDept.ParentID
	}
	if parentID == id {
		return nil, fmt.Errorf("Department cannot set itself as parent department")
	}
	if parentID > 0 {
		if _, err := s.GetDepartmentByID(ctx, parentID, tenantID); err != nil {
			return nil, fmt.Errorf("The parent department does not exist or does not belong to the current tenant")
		}
		seen := map[int]bool{id: true}
		cursor := parentID
		for cursor > 0 {
			if seen[cursor] {
				return nil, fmt.Errorf("Parent department settings will form a circular reference")
			}
			seen[cursor] = true
			parent, err := s.GetDepartmentByID(ctx, cursor, tenantID)
			if err != nil {
				return nil, fmt.Errorf("Incomplete department level data")
			}
			cursor = parent.ParentID
		}
	}
	if managerID > 0 {
		exists, err := s.client.User.Query().Where(user.ID(managerID), user.TenantID(tenantID), user.Active(true)).Exist(ctx)
		if err != nil || !exists {
			return nil, fmt.Errorf("The department head does not exist, is inactive, or does not belong to the current tenant")
		}
	}

	update := s.client.Department.UpdateOneID(id).
		Where(department.TenantIDEQ(tenantID), department.DeletedAtIsNil()).
		SetName(name).
		SetCode(code).
		SetDescription(description).
		SetManagerID(managerID).
		SetParentID(parentID)

	return update.Save(ctx)
}

// DeleteDepartment 删除部门
func (s *DepartmentService) DeleteDepartment(ctx context.Context, id int, tenantID int) error {
	// 检查部门是否存在且属于当前租户
	exists, err := s.client.Department.Query().
		Where(
			department.IDEQ(id),
			department.TenantIDEQ(tenantID),
			department.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Department does not exist: id=%d", id)
	}

	// 检查是否有子部门
	hasChildren, err := s.client.Department.Query().
		Where(department.ParentIDEQ(id), department.TenantIDEQ(tenantID), department.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasChildren {
		return fmt.Errorf("Unable to delete department: sub-departments exist under this department")
	}
	hasUsers, err := s.client.User.Query().Where(user.DepartmentID(id), user.TenantID(tenantID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("Failed to check department users: %w", err)
	}
	if hasUsers {
		return fmt.Errorf("Unable to delete department: please transfer or clear users under this department first")
	}

	// 软删除部门，保留历史关联和审计数据。
	_, err = s.client.Department.UpdateOneID(id).
		Where(department.TenantIDEQ(tenantID), department.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	return err
}
