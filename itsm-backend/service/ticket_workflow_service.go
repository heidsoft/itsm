package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketapproval"
	"itsm-backend/ent/ticketcc"

	"go.uber.org/zap"
)

type TicketWorkflowService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketWorkflowService(client *ent.Client, logger *zap.SugaredLogger) *TicketWorkflowService {
	return &TicketWorkflowService{
		client: client,
		logger: logger,
	}
}

// AcceptTicket 接单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) AcceptTicket(ctx context.Context, req *dto.AcceptTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Accepting ticket", "ticket_id", req.TicketID, "user_id", userID)

	// 检查工单是否存在且状态允许接单（读操作，事务外执行）
	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if tk.Status != "new" && tk.Status != "open" {
		return fmt.Errorf("工单当前状态不允许接单: %s", tk.Status)
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	// 更新工单状态和分配人
	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetAssigneeID(userID).
		SetStatus("in_progress").
		Save(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to accept ticket: %w", err)
		return txErr
	}

	// 记录流转记录
	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionAccept,
		FromStatus: &tk.Status,
		ToStatus:   ptrString("in_progress"),
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// RejectTicket 驳回工单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) RejectTicket(ctx context.Context, req *dto.RejectTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Rejecting ticket", "ticket_id", req.TicketID, "user_id", userID)

	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 更新工单状态
	returnToStatus := "rejected"
	if req.ReturnToStatus != nil {
		returnToStatus = *req.ReturnToStatus
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetStatus(returnToStatus).
		Save(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to reject ticket: %w", err)
		return txErr
	}

	// 记录流转记录
	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionReject,
		FromStatus: &tk.Status,
		ToStatus:   &returnToStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Reason:     req.Reason,
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// WithdrawTicket 撤回工单
func (s *TicketWorkflowService) WithdrawTicket(ctx context.Context, req *dto.WithdrawTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Withdrawing ticket", "ticket_id", req.TicketID, "user_id", userID)

	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 检查是否是工单创建者
	if tk.RequesterID != userID {
		return fmt.Errorf("只有工单创建者可以撤回工单")
	}

	if tk.Status == "closed" || tk.Status == "cancelled" {
		return fmt.Errorf("工单已关闭或取消，无法撤回")
	}

	// 更新工单状态
	_, err = s.client.Ticket.UpdateOneID(req.TicketID).
		SetStatus("cancelled").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to withdraw ticket: %w", err)
	}

	// 记录流转记录
	newStatus := "cancelled"
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionWithdraw,
		FromStatus: &tk.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Reason:     req.Reason,
		CreatedAt:  time.Now(),
	}, tenantID)

	return err
}

// ForwardTicket 转发工单
func (s *TicketWorkflowService) ForwardTicket(ctx context.Context, req *dto.ForwardTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Forwarding ticket", "ticket_id", req.TicketID, "to_user_id", req.ToUserID, "user_id", userID)

	_, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 如果转移所有权，更新assignee
	if req.TransferOwnership {
		_, err = s.client.Ticket.UpdateOneID(req.TicketID).
			SetAssigneeID(req.ToUserID).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to forward ticket: %w", err)
		}
	}

	// 记录流转记录
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:  req.TicketID,
		Action:    dto.WorkflowActionForward,
		Operator:  dto.WorkflowUserInfo{ID: userID},
		FromUser:  &dto.WorkflowUserInfo{ID: userID},
		ToUser:    &dto.WorkflowUserInfo{ID: req.ToUserID},
		Comment:   req.Comment,
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"transfer_ownership": req.TransferOwnership,
		},
	}, tenantID)

	return err
}

// CCTicket 抄送工单
func (s *TicketWorkflowService) CCTicket(ctx context.Context, req *dto.CCTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("CC ticket", "ticket_id", req.TicketID, "cc_users", req.CCUsers, "user_id", userID)

	// 检查工单是否存在
	_, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 添加抄送人
	for _, ccUserID := range req.CCUsers {
		// 检查是否已存在抄送记录
		exists, err := s.client.TicketCC.Query().
			Where(ticketcc.TicketID(req.TicketID),
				ticketcc.UserID(ccUserID),
				ticketcc.TenantID(tenantID)).
			Exist(ctx)
		if err != nil {
			s.logger.Warnw("Failed to check CC existence", "error", err, "user_id", ccUserID)
			continue
		}
		if !exists {
			err = s.client.TicketCC.Create().
				SetTicketID(req.TicketID).
				SetUserID(ccUserID).
				SetAddedBy(userID).
				SetTenantID(tenantID).
				SetAddedAt(time.Now()).
				SetIsActive(true).
				Exec(ctx)
			if err != nil {
				s.logger.Warnw("Failed to add CC user", "error", err, "user_id", ccUserID)
			}
		}
	}

	// 记录流转记录
	ccUserIDs := make([]int, len(req.CCUsers))
	for i, id := range req.CCUsers {
		ccUserIDs[i] = id
	}

	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:  req.TicketID,
		Action:    dto.WorkflowActionCC,
		Operator:  dto.WorkflowUserInfo{ID: userID},
		Comment:   req.Comment,
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"cc_users": ccUserIDs,
		},
	}, tenantID)

	return err
}

// ApproveTicket 审批工单（事务保护，保证审批记录更新、工单状态变更与流转记录的原子性）
func (s *TicketWorkflowService) ApproveTicket(ctx context.Context, req *dto.ApproveTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Approving ticket", "ticket_id", req.TicketID, "action", req.Action, "user_id", userID)

	// 检查工单是否存在（读操作，事务外执行）
	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 检查审批记录是否存在
	approval, err := s.client.TicketApproval.Query().
		Where(ticketapproval.ID(req.ApprovalID), ticketapproval.TicketID(req.TicketID), ticketapproval.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("审批记录不存在")
	}

	if approval.Status != string(dto.ApprovalStatusPending) {
		return fmt.Errorf("审批已处理，当前状态: %s", approval.Status)
	}

	if approval.ApproverID != userID {
		return fmt.Errorf("无权限审批该记录")
	}

	approvalLevel := approval.Level

	// 确定审批结果状态
	var newApprovalStatus string
	switch req.Action {
	case "approve":
		newApprovalStatus = string(dto.ApprovalStatusApproved)
	case "reject":
		newApprovalStatus = string(dto.ApprovalStatusRejected)
	case "delegate":
		if req.DelegateToUserID == nil {
			return fmt.Errorf("委派时必须指定委派人")
		}
		newApprovalStatus = string(dto.ApprovalStatusCancelled)
	default:
		return fmt.Errorf("无效的审批操作: %s", req.Action)
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	// 更新审批记录
	updateBuilder := txClient.TicketApproval.UpdateOneID(req.ApprovalID).
		SetStatus(newApprovalStatus).
		SetAction(req.Action).
		SetComment(req.Comment).
		SetProcessedAt(time.Now())

	if req.DelegateToUserID != nil {
		updateBuilder.SetDelegateToUserID(*req.DelegateToUserID)
	}

	err = updateBuilder.Exec(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to update approval: %w", err)
		return txErr
	}

	// 如果是委派，创建新的审批记录
	if req.Action == "delegate" && req.DelegateToUserID != nil {
		_, err = txClient.TicketApproval.Create().
			SetTicketID(req.TicketID).
			SetLevel(approval.Level).
			SetLevelName(approval.LevelName).
			SetApproverID(*req.DelegateToUserID).
			SetStatus(string(dto.ApprovalStatusPending)).
			SetTenantID(tenantID).
			Save(ctx)
		if err != nil {
			txErr = fmt.Errorf("创建委派审批记录失败: %w", err)
			return txErr
		}
	}

	if req.Action == "approve" {
		// 检查是否还有待审批的记录
		pendingCount, err := txClient.TicketApproval.Query().
			Where(ticketapproval.TicketID(req.TicketID),
				ticketapproval.TenantID(tenantID),
				ticketapproval.Status(string(dto.ApprovalStatusPending))).
			Count(ctx)
		if err != nil {
			txErr = fmt.Errorf("查询待审批记录失败: %w", err)
			return txErr
		}
		if pendingCount == 0 {
			_, err = txClient.Ticket.UpdateOneID(req.TicketID).
				SetStatus("approved").
				Save(ctx)
			if err != nil {
				txErr = fmt.Errorf("更新工单状态为已审批失败: %w", err)
				return txErr
			}
		}
	} else if req.Action == "reject" {
		// 审批拒绝，更新工单状态
		_, err = txClient.Ticket.UpdateOneID(req.TicketID).
			SetStatus("rejected").
			Save(ctx)
		if err != nil {
			txErr = fmt.Errorf("更新工单状态为已拒绝失败: %w", err)
			return txErr
		}

		// 取消其他待审批记录
		_, err = txClient.TicketApproval.Update().
			Where(ticketapproval.TicketID(req.TicketID),
				ticketapproval.TenantID(tenantID),
				ticketapproval.Status(string(dto.ApprovalStatusPending)),
				ticketapproval.IDNEQ(req.ApprovalID)).
			SetStatus(string(dto.ApprovalStatusCancelled)).
			Save(ctx)
		if err != nil {
			txErr = fmt.Errorf("取消其他待审批记录失败: %w", err)
			return txErr
		}
	}

	// 记录流转记录
	action := dto.WorkflowActionApprove
	if req.Action == "reject" {
		action = dto.WorkflowActionApproveReject
	} else if req.Action == "delegate" {
		action = dto.WorkflowActionDelegate
	}

	metadata := map[string]interface{}{
		"approval_id":    req.ApprovalID,
		"approval_level": approvalLevel,
	}
	if req.DelegateToUserID != nil {
		metadata["delegate_to_user_id"] = *req.DelegateToUserID
	}

	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     action,
		FromStatus: &tk.Status,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
		Metadata:   metadata,
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// ResolveTicket 解决工单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) ResolveTicket(ctx context.Context, req *dto.ResolveTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Resolving ticket", "ticket_id", req.TicketID, "user_id", userID)

	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	// 更新工单状态
	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetStatus("resolved").
		SetResolution(req.Resolution).
		SetResolutionCategory(req.ResolutionCategory).
		SetResolvedAt(time.Now()).
		Save(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to resolve ticket: %w", err)
		return txErr
	}

	// 记录流转记录
	newStatus := "resolved"
	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionResolve,
		FromStatus: &tk.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Resolution,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"resolution_category": req.ResolutionCategory,
			"work_notes":          req.WorkNotes,
		},
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// CloseTicket 关闭工单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) CloseTicket(ctx context.Context, req *dto.CloseTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Closing ticket", "ticket_id", req.TicketID, "user_id", userID)

	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if tk.Status != "resolved" {
		return fmt.Errorf("只有已解决的工单才能关闭")
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	// 更新工单状态
	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetStatus("closed").
		SetClosedAt(time.Now()).
		Save(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to close ticket: %w", err)
		return txErr
	}

	// 记录流转记录
	newStatus := "closed"
	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionClose,
		FromStatus: &tk.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.CloseNotes,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"close_reason": req.CloseReason,
		},
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// ReopenTicket 重开工单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) ReopenTicket(ctx context.Context, req *dto.ReopenTicketRequest, userID, tenantID int) error {
	// 兼容 ticket_id 和 ticketId 两种字段名
	if req.TicketID == 0 && req.TicketIDAlt != 0 {
		req.TicketID = req.TicketIDAlt
	}
	s.logger.Infow("Reopening ticket", "ticket_id", req.TicketID, "user_id", userID)

	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if tk.Status != "closed" && tk.Status != "resolved" {
		return fmt.Errorf("只有已关闭或已解决的工单才能重开")
	}

	// 开启事务，保证原子性
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	txClient := tx.Client()

	// 更新工单状态
	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetStatus("open").
		Save(ctx)
	if err != nil {
		txErr = fmt.Errorf("failed to reopen ticket: %w", err)
		return txErr
	}

	// 记录流转记录
	newStatus := "open"
	err = s.createWorkflowRecordWithClient(ctx, txClient, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionReopen,
		FromStatus: &tk.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Reason:     req.Reason,
		CreatedAt:  time.Now(),
	}, tenantID)
	if err != nil {
		txErr = fmt.Errorf("记录流转记录失败: %w", err)
		return txErr
	}

	txErr = tx.Commit()
	return txErr
}

// GetTicketWorkflowState 获取工单流转状态
func (s *TicketWorkflowService) GetTicketWorkflowState(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketWorkflowState, error) {
	// 查询工单信息
	tk, err := s.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 查询审批信息
	approvals, err := s.client.TicketApproval.Query().
		Where(ticketapproval.TicketID(ticketID), ticketapproval.TenantID(tenantID)).
		Order(ent.Asc(ticketapproval.FieldLevel)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to query approval status", "error", err)
	}

	var approvalStatus *dto.ApprovalStatus
	var currentLevel, totalLevels *int
	if len(approvals) > 0 {
		totalLevelsVal := len(approvals)
		totalLevels = &totalLevelsVal

		// 找到当前待审批的级别
		for _, a := range approvals {
			if a.Status == string(dto.ApprovalStatusPending) {
				lv := a.Level
				currentLevel = &lv
				st := dto.ApprovalStatus(a.Status)
				approvalStatus = &st
				break
			}
			// 如果有已拒绝的，状态就是已拒绝
			if a.Status == string(dto.ApprovalStatusRejected) {
				st := dto.ApprovalStatus(a.Status)
				approvalStatus = &st
			}
		}

		// 如果所有审批都通过
		allApproved := true
		for _, a := range approvals {
			if a.Status != string(dto.ApprovalStatusApproved) {
				allApproved = false
				break
			}
		}
		if allApproved {
			st := dto.ApprovalStatus(string(dto.ApprovalStatusApproved))
			approvalStatus = &st
		}
	}

	// 构建工单流转状态
	state := &dto.TicketWorkflowState{
		TicketID:             ticketID,
		CurrentStatus:        tk.Status,
		ApprovalStatus:       approvalStatus,
		CurrentApprovalLevel: currentLevel,
		TotalApprovalLevels:  totalLevels,
		AvailableActions:     []dto.TicketWorkflowAction{},
	}

	// 根据当前状态和用户权限判断可执行的操作
	switch tk.Status {
	case "new", "open":
		state.CanAccept = true
		state.CanForward = true
		state.CanCC = true
		state.AvailableActions = append(state.AvailableActions,
			dto.WorkflowActionAccept,
			dto.WorkflowActionForward,
			dto.WorkflowActionCC)

		if tk.RequesterID == userID {
			state.CanWithdraw = true
			state.AvailableActions = append(state.AvailableActions, dto.WorkflowActionWithdraw)
		}
	case "in_progress":
		state.CanResolve = true
		state.CanForward = true
		state.CanCC = true
		state.AvailableActions = append(state.AvailableActions,
			dto.WorkflowActionResolve,
			dto.WorkflowActionForward,
			dto.WorkflowActionCC)
	case "resolved":
		state.CanClose = true
		state.AvailableActions = append(state.AvailableActions,
			dto.WorkflowActionClose,
			dto.WorkflowActionReopen)
	case "closed":
		state.AvailableActions = append(state.AvailableActions, dto.WorkflowActionReopen)
	}

	// 检查审批权限
	if approvalStatus != nil && *approvalStatus == dto.ApprovalStatusPending {
		// 检查当前用户是否是当前审批级别的审批人
		for _, a := range approvals {
			if a.Level == currentLevelVal(state) && a.ApproverID == userID {
				state.CanApprove = true
				state.AvailableActions = append(state.AvailableActions,
					dto.WorkflowActionApprove,
					dto.WorkflowActionApproveReject)
				break
			}
		}
	}

	return state, nil
}

func currentLevelVal(state *dto.TicketWorkflowState) int {
	if state.CurrentApprovalLevel == nil {
		return 0
	}
	return *state.CurrentApprovalLevel
}

func ptrToApprovalStatus(s string) *dto.ApprovalStatus {
	status := dto.ApprovalStatus(s)
	return &status
}

// 辅助函数

func (s *TicketWorkflowService) getTicket(ctx context.Context, ticketID, tenantID int) (*ent.Ticket, error) {
	tk, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("工单不存在")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	return tk, nil
}

func (s *TicketWorkflowService) createWorkflowRecord(ctx context.Context, record *dto.TicketWorkflowRecord, tenantID int) error {
	return s.createWorkflowRecordWithClient(ctx, s.client, record, tenantID)
}

// createWorkflowRecordWithClient 使用指定的 Ent 客户端创建流转记录（支持事务内复用）
func (s *TicketWorkflowService) createWorkflowRecordWithClient(ctx context.Context, client *ent.Client, record *dto.TicketWorkflowRecord, tenantID int) error {
	create := client.TicketWorkflowRecord.Create().
		SetTicketID(record.TicketID).
		SetAction(string(record.Action)).
		SetOperatorID(record.Operator.ID).
		SetTenantID(tenantID).
		SetCreatedAt(record.CreatedAt)

	if record.FromStatus != nil {
		create.SetFromStatus(*record.FromStatus)
	}
	if record.ToStatus != nil {
		create.SetToStatus(*record.ToStatus)
	}
	if record.FromUser != nil {
		create.SetFromUserID(record.FromUser.ID)
	}
	if record.ToUser != nil {
		create.SetToUserID(record.ToUser.ID)
	}
	if record.Comment != "" {
		create.SetComment(record.Comment)
	}
	if record.Reason != "" {
		create.SetReason(record.Reason)
	}
	if record.Metadata != nil {
		create.SetMetadata(record.Metadata)
	}

	_, err := create.Save(ctx)
	return err
}

func ptrString(s string) *string {
	return &s
}
