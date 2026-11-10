package service

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("Failed to create group: %w", err)
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
		return nil, fmt.Errorf("Group does not exist or does not have access: %w", err)
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
		return nil, 0, fmt.Errorf("Failed to query total number: %w", err)
	}

	groups, err := query.
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Order(ent.Asc(group.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to query group list: %w", err)
	}

	return groups, total, nil
}

// UpdateGroup 更新组
func (s *GroupService) UpdateGroup(ctx context.Context, id int, req *dto.UpdateGroupRequest, tenantID int) (*ent.Group, error) {
	// 先检查组是否存在且属于当前租户
	exists, err := s.client.Group.Query().
		Where(
			group.IDEQ(id),
			group.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("Query group failed: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("Group does not exist or does not have access")
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
		return nil, fmt.Errorf("Failed to update group: %w", err)
	}
	return g, nil
}

// DeleteGroup 删除组
func (s *GroupService) DeleteGroup(ctx context.Context, id, tenantID int) error {
	// 检查组是否存在且属于当前租户
	exists, err := s.client.Group.Query().
		Where(
			group.IDEQ(id),
			group.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("Query group failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("Group does not exist or does not have access")
	}

	// 删除组（会自动清除关联）
	err = s.client.Group.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete group: %w", err)
	}
	return nil
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
		return fmt.Errorf("Query group failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("Group does not exist or does not have access")
	}

	// 验证用户是否存在且属于当前租户
	u, err := s.client.User.Query().
		Where(
			user.IDEQ(userID),
			user.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("User does not exist or does not have access rights: %w", err)
	}

	// 添加用户到组
	err = s.client.Group.UpdateOneID(groupID).
		AddMembers(u).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user to group: %w", err)
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
		return fmt.Errorf("Query group failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("Group does not exist or does not have access")
	}

	// 移除用户（使用ID直接移除）
	err = s.client.Group.UpdateOneID(groupID).
		RemoveMemberIDs(userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("Failed to remove user from group: %w", err)
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
		return nil, 0, fmt.Errorf("Failed to query the total number of members: %w", err)
	}

	users, err := query.
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Order(ent.Asc(user.FieldUsername)).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to query group members: %w", err)
	}

	return users, total, nil
}
