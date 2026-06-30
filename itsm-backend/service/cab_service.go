package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cabmember"
	"itsm-backend/ent/change"

	"go.uber.org/zap"
)

// CABService 变更咨询委员会服务
type CABService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	changeApprovalServ  *ChangeApprovalService
}

// NewCABService 创建CAB服务
func NewCABService(client *ent.Client, logger *zap.SugaredLogger, changeApprovalServ *ChangeApprovalService) *CABService {
	return &CABService{
		client:             client,
		logger:             logger,
		changeApprovalServ: changeApprovalServ,
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

	// 创建成员
	member, err := s.client.CABMember.Create().
		SetUserID(req.UserID).
		SetType(req.Type).
		SetRole(req.Role).
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

// ListCABMembers 获取CAB成员列表
func (s *CABService) ListCABMembers(ctx context.Context, boardType string, tenantID int) ([]*dto.CABMemberResponse, error) {
	// 查询成员
	members, err := s.client.CABMember.Query().
		Where(
			cabmember.Type(boardType),
			cabmember.TenantID(tenantID),
			cabmember.IsActive(true),
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

// CreateCABApprovalWorkflow 为变更创建CAB审批工作流
func (s *CABService) CreateCABApprovalWorkflow(ctx context.Context, changeID int, tenantID int) error {
	// 获取变更信息
	ch, err := s.client.Change.Query().
		Where(change.ID(changeID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("change not found: %w", err)
	}

	// 根据变更类型确定审批委员会
	var boardType string
	switch ch.Type {
	case string(dto.ChangeTypeEmergency):
		boardType = "ECAB" // 紧急变更走ECAB审批
	default:
		boardType = "CAB" // 普通变更走CAB审批
	}

	// 获取CAB成员
	members, err := s.client.CABMember.Query().
		Where(
			cabmember.Type(boardType),
			cabmember.TenantID(tenantID),
			cabmember.IsActive(true),
		).
		Order(ent.Asc(cabmember.FieldRole)). // Chair先审批，然后是其他成员
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get CAB members: %w", err)
	}

	if len(members) == 0 {
		s.logger.Warnw("No CAB members found for change", "change_id", changeID, "type", boardType)
		return fmt.Errorf("no %s members available for approval", boardType)
	}

	// 构建审批链
	var approvalChain []dto.ChangeApprovalChainItem
	for i, member := range members {
		approvalChain = append(approvalChain, dto.ChangeApprovalChainItem{
			Level:      i + 1,
			ApproverID: member.UserID,
			Role:       member.Role,
			IsRequired: true, // CAB审批都是必需的
		})
	}

	// 创建审批工作流
	err = s.changeApprovalServ.CreateChangeApprovalWorkflow(ctx, &dto.ChangeApprovalWorkflowRequest{
		ChangeID:      changeID,
		ApprovalChain: approvalChain,
	}, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create CAB approval workflow", "error", err, "change_id", changeID)
		return fmt.Errorf("failed to create approval workflow: %w", err)
	}

	s.logger.Infow("%s approval workflow created for change", boardType, "change_id", changeID, "approvers", len(approvalChain))
	return nil
}

// GetCABApprovalSummary 获取CAB审批摘要
func (s *CABService) GetCABApprovalSummary(ctx context.Context, changeID int, tenantID int) (*dto.ChangeApprovalSummary, error) {
	return s.changeApprovalServ.GetChangeApprovalSummary(ctx, changeID, tenantID)
}

// ApproveCABChange CAB审批变更
func (s *CABService) ApproveCABChange(ctx context.Context, req *dto.CABApprovalRequest, tenantID int) (*dto.ChangeApprovalResponse, error) {
	// 验证审批人是否是CAB成员
	ch, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 确定需要的委员会类型
	var requiredBoardType string
	switch ch.Type {
	case string(dto.ChangeTypeEmergency):
		requiredBoardType = "ECAB"
	default:
		requiredBoardType = "CAB"
	}

	// 验证用户是否是对应委员会的成员
	isMember, err := s.client.CABMember.Query().
		Where(
			cabmember.UserID(req.ApproverID),
			cabmember.Type(requiredBoardType),
			cabmember.TenantID(tenantID),
			cabmember.IsActive(true),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify approver membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of %s", requiredBoardType)
	}

	// 更新审批状态
	return s.changeApprovalServ.UpdateChangeApproval(ctx, req.ApprovalID, &dto.UpdateChangeApprovalRequest{
		Status:  req.Status,
		Comment: &req.Comment,
	}, tenantID)
}
