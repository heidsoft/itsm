package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/team"
	"itsm-backend/ent/user"
)

type TeamService struct {
	client *ent.Client
}

func NewTeamService(client *ent.Client) *TeamService {
	return &TeamService{client: client}
}

// CreateTeam 创建团队
func (s *TeamService) CreateTeam(ctx context.Context, name, code, description string, managerID, tenantID int) (*ent.Team, error) {
	if managerID > 0 {
		if err := s.validateTenantUser(ctx, managerID, tenantID); err != nil {
			return nil, fmt.Errorf("团队负责人无效: %w", err)
		}
	}

	query := s.client.Team.Create().
		SetName(name).
		SetCode(code).
		SetDescription(description).
		SetTenantID(tenantID)

	if managerID > 0 {
		query.SetManagerID(managerID)
	}

	return query.Save(ctx)
}

// AddMember 添加成员
func (s *TeamService) AddMember(ctx context.Context, teamID, userID, tenantID int) error {
	exists, err := s.client.Team.Query().
		Where(team.IDEQ(teamID), team.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查团队失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("团队不存在或不属于当前租户: id=%d", teamID)
	}
	if err := s.validateTenantUser(ctx, userID, tenantID); err != nil {
		return fmt.Errorf("团队成员无效: %w", err)
	}

	return s.client.Team.UpdateOneID(teamID).
		AddUserIDs(userID).
		Exec(ctx)
}

func (s *TeamService) validateTenantUser(ctx context.Context, userID, tenantID int) error {
	exists, err := s.client.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("用户不存在、已停用或不属于当前租户: id=%d", userID)
	}
	return nil
}

// ListTeams 获取团队列表
func (s *TeamService) ListTeams(ctx context.Context, tenantID int) ([]*ent.Team, error) {
	return s.client.Team.Query().
		Where(team.TenantID(tenantID)).
		WithUsers().
		All(ctx)
}

// UpdateTeam 更新团队
func (s *TeamService) UpdateTeam(ctx context.Context, id int, name, code, description *string, managerID *int, tenantID int) (*ent.Team, error) {
	// 检查团队是否存在且属于当前租户
	_, err := s.client.Team.Query().
		Where(
			team.IDEQ(id),
			team.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}

	update := s.client.Team.UpdateOneID(id)
	if name != nil {
		update = update.SetName(*name)
	}
	if code != nil {
		update = update.SetCode(*code)
	}
	if description != nil {
		update = update.SetDescription(*description)
	}
	if managerID != nil {
		if *managerID > 0 {
			if err := s.validateTenantUser(ctx, *managerID, tenantID); err != nil {
				return nil, fmt.Errorf("团队负责人无效: %w", err)
			}
		}
		update = update.SetManagerID(*managerID)
	}

	return update.Save(ctx)
}

// DeleteTeam 删除团队
func (s *TeamService) DeleteTeam(ctx context.Context, id int, tenantID int) error {
	// 检查团队是否存在且属于当前租户
	exists, err := s.client.Team.Query().
		Where(
			team.IDEQ(id),
			team.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("团队不存在: id=%d", id)
	}

	// 删除团队
	return s.client.Team.DeleteOneID(id).Exec(ctx)
}
