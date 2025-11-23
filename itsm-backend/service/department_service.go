package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/department"
)

type DepartmentService struct {
	client *ent.Client
}

func NewDepartmentService(client *ent.Client) *DepartmentService {
	return &DepartmentService{client: client}
}

// Department Methods

func (s *DepartmentService) CreateDepartment(ctx context.Context, name, code, description string, managerID, parentID, tenantID int) (*ent.Department, error) {
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
		Where(department.TenantID(tenantID)).
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
	// 检查部门是否存在且属于当前租户
	_, err := s.client.Department.Query().
		Where(
			department.IDEQ(id),
			department.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Department.UpdateOneID(id)
	if name != "" {
		update = update.SetName(name)
	}
	if code != "" {
		update = update.SetCode(code)
	}
	if description != "" {
		update = update.SetDescription(description)
	}
	if managerID > 0 {
		update = update.SetManagerID(managerID)
	}
	if parentID >= 0 { // 允许设置为0（根节点）
		update = update.SetParentID(parentID)
	}

	return update.Save(ctx)
}

// DeleteDepartment 删除部门
func (s *DepartmentService) DeleteDepartment(ctx context.Context, id int, tenantID int) error {
	// 检查部门是否存在且属于当前租户
	exists, err := s.client.Department.Query().
		Where(
			department.IDEQ(id),
			department.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("部门不存在: id=%d", id)
	}

	// 检查是否有子部门
	hasChildren, err := s.client.Department.Query().
		Where(department.ParentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasChildren {
		return fmt.Errorf("无法删除部门：该部门下存在子部门")
	}

	// 删除部门
	return s.client.Department.DeleteOneID(id).Exec(ctx)
}
