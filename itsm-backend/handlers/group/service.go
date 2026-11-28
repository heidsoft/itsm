package group

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/group"
	"itsm-backend/ent/user"
)

type GroupService struct {
	client *ent.Client
}

func NewGroupService(client *ent.Client) *GroupService {
	return &GroupService{client: client}
}

// CreateGroup 创建组
func (s *GroupService) CreateGroup(ctx context.Context, req *dto.CreateGroupRequest) (*ent.Group, error) {
	g, err := s.client.Group.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建组失败: %w", err)
	}
	return g, nil
}

// GetGroup 获取组详情
func (s *GroupService) GetGroup(ctx context.Context, id, tenantID int) (*ent.Group, error) {
	g, err := s.client.Group.Query().
		Where(
			group.IDEQ(id),
			group.TenantIDEQ(tenantID),
		).
		WithMembers().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("组不存在或无权访问: %w", err)
	}
	return g, nil
}

// ListGroups 获取组列表（分页）
func (s *GroupService) ListGroups(ctx context.Context, req *dto.ListGroupsRequest) ([]*ent.Group, int, error) {
	query := s.client.Group.Query().
		Where(group.TenantIDEQ(req.TenantID))

	// 搜索
	if req.Search != "" {
		query = query.Where(
			group.NameContainsFold(req.Search),
		)
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	groups, err := query.
		// 只取成员 ID，供列表响应填充 memberCount，避免把整份用户档案拉回内存。
		WithMembers(func(q *ent.UserQuery) { q.Select(user.FieldID) }).
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Order(ent.Asc(group.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询组列表失败: %w", err)
	}

	return groups, total, nil
}

// UpdateGroup 更新组
func (s *GroupService) UpdateGroup(ctx context.Context, id int, req *dto.UpdateGroupRequest, tenantID int) (*ent.Group, error) {
	// 先检查组是否存在且属于当前租户
	existing, err := s.client.Group.Query().
		Where(
			group.IDEQ(id),
			group.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("组不存在或无权访问")
	}

	// 组名是审批人解析键（BPMN candidateGroups / 审批链 group:<name>）。
	// 改名会让被引用组的候选审批人静默变空，属于审批绕过，必须先阻断。
	if req.Name != "" && req.Name != existing.Name {
		if err := s.assertNotReferenced(ctx, tenantID, existing.Name, "改名"); err != nil {
			return nil, err
		}
	}

	update := s.client.Group.UpdateOneID(id)
	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}

	g, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新组失败: %w", err)
	}
	return g, nil
}

// DeleteGroup 删除组
func (s *GroupService) DeleteGroup(ctx context.Context, id, tenantID int) error {
	// 检查组是否存在且属于当前租户
	existing, err := s.client.Group.Query().
		Where(
			group.IDEQ(id),
			group.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("组不存在或无权访问")
	}

	// 组名是审批人解析键，删除被引用组会让候选审批人静默变空，必须先阻断。
	if err := s.assertNotReferenced(ctx, tenantID, existing.Name, "删除"); err != nil {
		return err
	}

	// 删除组（会自动清除关联）
	err = s.client.Group.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除组失败: %w", err)
	}
	return nil
}

// assertNotReferenced 断言组名未被本租户的审批配置引用；被引用则以 409 阻断，
// 并在用户可见的 message 中列出引用位置，便于管理员先调整流程/审批链再变更组。
func (s *GroupService) assertNotReferenced(ctx context.Context, tenantID int, groupName, action string) error {
	refs, err := findGroupReferences(ctx, s.client, tenantID, groupName)
	if err != nil {
		// 无法确认引用面时失败关闭：拒绝变更，避免坏 XML 绕过守卫。
		return common.NewStateConflictError(
			fmt.Sprintf("无法确认审批组 %q 的引用情况，已阻止%s以保证审批链安全", groupName, action),
			err.Error(),
		)
	}
	if len(refs) == 0 {
		return nil
	}
	var parts []string
	for _, r := range refs {
		parts = append(parts, fmt.Sprintf("%s:%s(节点 %s)", r.Kind, r.RefKey, r.NodeID))
	}
	return common.NewStateConflictError(
		fmt.Sprintf("审批组 %q 已被 %d 处审批配置引用，不能%s；请先修改引用后再操作：%s",
			groupName, len(refs), action, strings.Join(parts, "、")),
		"",
	)
}

// AddUserToGroup 添加用户到组
func (s *GroupService) AddUserToGroup(ctx context.Context, groupID, userID, tenantID int) error {
	// 验证组是否存在且属于当前租户
	exists, err := s.client.Group.Query().
		Where(
			group.IDEQ(groupID),
			group.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询组失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("组不存在或无权访问")
	}

	// 验证用户是否存在且属于当前租户
	u, err := s.client.User.Query().
		Where(
			user.IDEQ(userID),
			user.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("用户不存在或无权访问: %w", err)
	}

	// 添加用户到组
	err = s.client.Group.UpdateOneID(groupID).
		AddMembers(u).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("添加用户到组失败: %w", err)
	}
	return nil
}

// RemoveUserFromGroup 从组移除用户
func (s *GroupService) RemoveUserFromGroup(ctx context.Context, groupID, userID, tenantID int) error {
	// 验证组是否存在且属于当前租户
	exists, err := s.client.Group.Query().
		Where(
			group.IDEQ(groupID),
			group.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询组失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("组不存在或无权访问")
	}

	// 移除用户（使用ID直接移除）
	err = s.client.Group.UpdateOneID(groupID).
		RemoveMemberIDs(userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("从组移除用户失败: %w", err)
	}
	return nil
}

// GetGroupMembers 获取组成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, groupID, tenantID int, page, pageSize int) ([]*ent.User, int, error) {
	query := s.client.Group.Query().
		Where(
			group.IDEQ(groupID),
			group.TenantIDEQ(tenantID),
		).
		QueryMembers()

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询成员总数失败: %w", err)
	}

	users, err := query.
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Order(ent.Asc(user.FieldUsername)).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询组成员失败: %w", err)
	}

	return users, total, nil
}
