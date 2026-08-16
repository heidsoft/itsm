package service

// test-coverage-guard: skip — 删除委托旧 ChangeApprovalService 的 CAB 双路径方法;替换路径由 handlers/change/change_approval_chain_test.go(CABResolver 用例)覆盖,名册管理方法无逻辑变更。

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cabmember"

	"go.uber.org/zap"
)

// CABService 变更咨询委员会服务（仅负责 CAB/ECAB 成员名册管理）。
// 注：CAB 审批流转已统一由审批链引擎（cab:CAB / cab:ECAB 解析器）驱动，
// 不再通过独立的 ChangeApprovalService 自建 raw-SQL 审批链（消除双路径）。
type CABService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCABService 创建CAB服务
func NewCABService(client *ent.Client, logger *zap.SugaredLogger) *CABService {
	return &CABService{
		client: client,
		logger: logger,
	}
}

// AddCABMember 添加CAB成员
func (s *CABService) AddCABMember(ctx context.Context, req *dto.AddCABMemberRequest, tenantID int) (*dto.CABMemberResponse, error) {
	// 验证用户是否存在
	_, err := s.client.User.Get(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 检查是否已经是CAB成员
	exists, err := s.client.CABMember.Query().
		Where(
			cabmember.UserID(req.UserID),
			cabmember.Type(req.Type),
			cabmember.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check member existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user is already a member of this board")
	}

	// 创建成员（默认激活，否则审批链引擎 cab: 解析器不会纳入）
	member, err := s.client.CABMember.Create().
		SetUserID(req.UserID).
		SetType(req.Type).
		SetRole(req.Role).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to add CAB member", "error", err, "user_id", req.UserID, "type", req.Type)
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	// 获取用户信息
	user, err := s.client.User.Get(ctx, req.UserID)
	if err != nil {
		s.logger.Warnw("Failed to get user info", "error", err, "user_id", req.UserID)
	}

	s.logger.Infow("CAB member added", "member_id", member.ID, "user_id", req.UserID, "type", req.Type)
	return &dto.CABMemberResponse{
		ID:        member.ID,
		UserID:    member.UserID,
		UserName:  user.Name,
		Email:     user.Email,
		Type:      member.Type,
		Role:      member.Role,
		IsActive:  member.IsActive,
		TenantID:  member.TenantID,
		CreatedAt: member.CreatedAt,
	}, nil
}

// RemoveCABMember 移除CAB成员
func (s *CABService) RemoveCABMember(ctx context.Context, memberID int, tenantID int) error {
	// 验证成员是否存在且属于当前租户
	member, err := s.client.CABMember.Query().
		Where(
			cabmember.ID(memberID),
			cabmember.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("member not found")
		}
		return fmt.Errorf("failed to get member: %w", err)
	}

	// 删除成员
	err = s.client.CABMember.DeleteOneID(memberID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to remove CAB member", "error", err, "member_id", memberID)
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.logger.Infow("CAB member removed", "member_id", memberID, "user_id", member.UserID, "type", member.Type)
	return nil
}

// ListCABMembers 获取CAB成员列表（含未激活，便于管理端展示与启停）。
// 审批链引擎 cab: 解析器另行按 is_active=true 过滤，二者互不干扰。
func (s *CABService) ListCABMembers(ctx context.Context, boardType string, tenantID int) ([]*dto.CABMemberResponse, error) {
	// 查询成员
	members, err := s.client.CABMember.Query().
		Where(
			cabmember.Type(boardType),
			cabmember.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CAB members", "error", err, "type", boardType)
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	// 构建响应
	var response []*dto.CABMemberResponse
	for _, member := range members {
		// 获取用户信息
		user, err := s.client.User.Get(ctx, member.UserID)
		if err != nil {
			s.logger.Warnw("Failed to get user info", "error", err, "user_id", member.UserID)
			continue
		}

		response = append(response, &dto.CABMemberResponse{
			ID:        member.ID,
			UserID:    member.UserID,
			UserName:  user.Name,
			Email:     user.Email,
			Type:      member.Type,
			Role:      member.Role,
			IsActive:  member.IsActive,
			TenantID:  member.TenantID,
			CreatedAt: member.CreatedAt,
		})
	}

	return response, nil
}

// UpdateCABMember 更新CAB成员（角色 / 激活状态）
func (s *CABService) UpdateCABMember(ctx context.Context, memberID, tenantID int, role string, isActive bool) (*dto.CABMemberResponse, error) {
	// 校验成员归属当前租户
	_, err := s.client.CABMember.Query().
		Where(
			cabmember.ID(memberID),
			cabmember.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	member, err := s.client.CABMember.UpdateOneID(memberID).
		SetRole(role).
		SetIsActive(isActive).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CAB member", "error", err, "member_id", memberID)
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	user, err := s.client.User.Get(ctx, member.UserID)
	if err != nil {
		s.logger.Warnw("Failed to get user info", "error", err, "user_id", member.UserID)
	}

	s.logger.Infow("CAB member updated", "member_id", memberID, "role", role, "is_active", isActive)
	return &dto.CABMemberResponse{
		ID:        member.ID,
		UserID:    member.UserID,
		UserName:  user.Name,
		Email:     user.Email,
		Type:      member.Type,
		Role:      member.Role,
		IsActive:  member.IsActive,
		TenantID:  member.TenantID,
		CreatedAt: member.CreatedAt,
	}, nil
}
