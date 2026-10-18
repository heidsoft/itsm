package service

import (
	"context"
	"fmt"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/changereviewmember"
	entuser "itsm-backend/ent/user"

	"go.uber.org/zap"
)

// ChangeReviewService 变更评审组服务（仅负责 REVIEW/EREVIEW 成员名册管理）。
// 注：评审审批流转已统一由审批链引擎（review:REVIEW / review:EREVIEW 解析器）驱动，
// 不再通过独立的 ChangeApprovalService 自建 raw-SQL 审批链（消除双路径）。
type ChangeReviewService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewChangeReviewService 创建变更评审组服务
func NewChangeReviewService(client *ent.Client, logger *zap.SugaredLogger) *ChangeReviewService {
	return &ChangeReviewService{
		client: client,
		logger: logger,
	}
}

// AddMember 添加评审组成员
func (s *ChangeReviewService) AddMember(ctx context.Context, req *dto.AddChangeReviewMemberRequest, tenantID int) (*dto.ChangeReviewMemberResponse, error) {
	user, err := s.client.User.Query().
		Where(entuser.ID(req.UserID), entuser.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 引用型资源缺失（不存在或跨租户）：按业务参数错误映射 400/1001，
			// 与工单评论「提及用户无效」同口径；不再裸 error 兜底 500。
			return nil, common.NewBusinessError(common.ParamErrorCode, "评审组成员用户不存在或不属于当前租户", "")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	exists, err := s.client.ChangeReviewMember.Query().
		Where(
			changereviewmember.UserID(req.UserID),
			changereviewmember.Type(req.Type),
			changereviewmember.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check member existence: %w", err)
	}
	if exists {
		// 业务重复：映射 409/4090，不再裸 error 兜底 500。
		return nil, common.NewBusinessError(common.ConflictCode, "该用户已加入该评审组", "")
	}

	member, err := s.client.ChangeReviewMember.Create().
		SetUserID(req.UserID).
		SetType(req.Type).
		SetRole(req.Role).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to add change review member", "error", err, "user_id", req.UserID, "type", req.Type)
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	if user == nil {
		s.logger.Warnw("User is nil after tenant check", "user_id", req.UserID)
		return nil, fmt.Errorf("user not found")
	}

	s.logger.Infow("Change review member added", "member_id", member.ID, "user_id", req.UserID, "type", req.Type)
	return &dto.ChangeReviewMemberResponse{
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

// RemoveMember 移除评审组成员
func (s *ChangeReviewService) RemoveMember(ctx context.Context, memberID int, tenantID int) error {
	member, err := s.client.ChangeReviewMember.Query().
		Where(
			changereviewmember.ID(memberID),
			changereviewmember.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 不存在或跨租户同形 404：不再裸 error 兜底 500（存在性可探测面）。
			return common.NewBusinessError(common.NotFoundCode, "评审组成员不存在", "")
		}
		return fmt.Errorf("failed to get member: %w", err)
	}

	err = s.client.ChangeReviewMember.DeleteOneID(memberID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to remove change review member", "error", err, "member_id", memberID)
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.logger.Infow("Change review member removed", "member_id", memberID, "user_id", member.UserID, "type", member.Type)
	return nil
}

// ListMembers 获取评审组成员列表（含未激活，便于管理端展示与启停）。
// 审批链引擎 review: 解析器另行按 is_active=true 过滤，二者互不干扰。
func (s *ChangeReviewService) ListMembers(ctx context.Context, boardType string, tenantID int) ([]*dto.ChangeReviewMemberResponse, error) {
	members, err := s.client.ChangeReviewMember.Query().
		Where(
			changereviewmember.Type(boardType),
			changereviewmember.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list change review members", "error", err, "type", boardType)
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	response := make([]*dto.ChangeReviewMemberResponse, 0, len(members))
	for _, member := range members {
		// 纵深防御：成员行虽经租户谓词筛出，用户详情读取仍带租户谓词，
		// 让历史脏数据（成员行指向他租用户）在列表里静默跳过而不是回显他租身份。
		user, err := s.client.User.Query().
			Where(
				entuser.ID(member.UserID),
				entuser.TenantID(tenantID),
			).
			Only(ctx)
		if err != nil {
			s.logger.Warnw("Failed to get user info", "error", err, "user_id", member.UserID)
			continue
		}

		response = append(response, &dto.ChangeReviewMemberResponse{
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

// UpdateMember 更新评审组成员（角色 / 激活状态）
func (s *ChangeReviewService) UpdateMember(ctx context.Context, memberID, tenantID int, role string, isActive bool) (*dto.ChangeReviewMemberResponse, error) {
	_, err := s.client.ChangeReviewMember.Query().
		Where(
			changereviewmember.ID(memberID),
			changereviewmember.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 不存在或跨租户同形 404：不再裸 error 兜底 500。
			return nil, common.NewBusinessError(common.NotFoundCode, "评审组成员不存在", "")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	member, err := s.client.ChangeReviewMember.UpdateOneID(memberID).
		SetRole(role).
		SetIsActive(isActive).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update change review member", "error", err, "member_id", memberID)
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	user, err := s.client.User.Get(ctx, member.UserID)
	if err != nil {
		s.logger.Warnw("Failed to get user info", "error", err, "user_id", member.UserID)
	}

	s.logger.Infow("Change review member updated", "member_id", memberID, "role", role, "is_active", isActive)
	return &dto.ChangeReviewMemberResponse{
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
