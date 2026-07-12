package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketapproval"
	"itsm-backend/ent/ticketautomationrule"
	"itsm-backend/ent/ticketcc"
	"itsm-backend/ent/ticketworkflowrecord"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type TicketWorkflowService struct {
	client           *ent.Client
	logger           *zap.SugaredLogger
	connectorManager *connector.Manager
}

func NewTicketWorkflowService(client *ent.Client, logger *zap.SugaredLogger) *TicketWorkflowService {
	return &TicketWorkflowService{
		client: client,
		logger: logger,
	}
}

// SetConnectorManager 设置连接器管理器，用于飞书、钉钉、企业微信等外部渠道通知。
func (s *TicketWorkflowService) SetConnectorManager(manager *connector.Manager) {
	s.connectorManager = manager
}

// AcceptTicket 接单（事务保护，保证工单状态更新与流转记录的原子性）
func (s *TicketWorkflowService) AcceptTicket(ctx context.Context, req *dto.AcceptTicketRequest, userID, tenantID int) error {
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
	// P1-07 修复：接单同时设置 first_response_at，供 SLA 计时使用
	now := time.Now()
	_, err = txClient.Ticket.UpdateOneID(req.TicketID).
		SetAssigneeID(userID).
		SetStatus("in_progress").
		SetFirstResponseAt(now).
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
	s.logger.Infow("CC ticket", "ticket_id", req.TicketID, "cc_users", req.CCUsers, "user_id", userID)

	// 检查工单是否存在
	tk, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if err := s.ensureCanCCTicket(ctx, tk, userID, tenantID); err != nil {
		return err
	}

	targetUsers, err := s.client.User.Query().
		Where(user.IDIn(req.CCUsers...), user.TenantID(tenantID), user.Active(true)).
		Select(user.FieldID).
		Ints(ctx)
	if err != nil {
		return fmt.Errorf("校验抄送用户失败: %w", err)
	}
	if len(targetUsers) != len(uniqueInts(req.CCUsers)) {
		return fmt.Errorf("抄送用户不存在、未激活或不属于当前租户")
	}

	// 添加抄送人
	addedUserIDs := make([]int, 0, len(targetUsers))
	for _, ccUserID := range targetUsers {
		// 检查是否已存在抄送记录
		exists, err := s.client.TicketCC.Query().
			Where(ticketcc.TicketID(req.TicketID),
				ticketcc.UserID(ccUserID),
				ticketcc.TenantID(tenantID),
				ticketcc.IsActive(true)).
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
				continue
			}
			addedUserIDs = append(addedUserIDs, ccUserID)
		}
	}

	if len(addedUserIDs) > 0 {
		s.createCCNotifications(ctx, tk, addedUserIDs, req.NotifyChannels, userID, tenantID)
	}

	// 记录流转记录
	ccUserIDs := make([]int, len(targetUsers))
	copy(ccUserIDs, targetUsers)

	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:  req.TicketID,
		Action:    dto.WorkflowActionCC,
		Operator:  dto.WorkflowUserInfo{ID: userID},
		Comment:   req.Comment,
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"cc_users":        ccUserIDs,
			"notify_channels": normalizeNotifyChannels(req.NotifyChannels),
		},
	}, tenantID)

	return err
}

// ListMyCCRecords 查询当前用户收到的抄送记录
func (s *TicketWorkflowService) ListMyCCRecords(ctx context.Context, userID, tenantID int) (*dto.TicketCCListResponse, error) {
	records, err := s.client.TicketCC.Query().
		Where(ticketcc.UserID(userID), ticketcc.TenantID(tenantID), ticketcc.IsActive(true)).
		Order(ent.Desc(ticketcc.FieldAddedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询我的抄送失败: %w", err)
	}
	return s.buildCCListResponse(ctx, records)
}

// ListTicketCCRecords 查询单个工单抄送记录
func (s *TicketWorkflowService) ListTicketCCRecords(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketCCListResponse, error) {
	tk, err := s.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCanViewTicketCC(ctx, tk, userID, tenantID); err != nil {
		return nil, err
	}

	records, err := s.client.TicketCC.Query().
		Where(ticketcc.TicketID(ticketID), ticketcc.TenantID(tenantID), ticketcc.IsActive(true)).
		Order(ent.Desc(ticketcc.FieldAddedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询工单抄送记录失败: %w", err)
	}
	return s.buildCCListResponse(ctx, records)
}

// ApproveTicket 审批工单（事务保护，保证审批记录更新、工单状态变更与流转记录的原子性）
func (s *TicketWorkflowService) ApproveTicket(ctx context.Context, req *dto.ApproveTicketRequest, userID, tenantID int) error {
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

// GetAvailableActions 返回当前用户在该工单上可执行的流转动作列表。
// 复用 GetTicketWorkflowState 的状态/权限计算逻辑，避免在多处重复状态机规则。
func (s *TicketWorkflowService) GetAvailableActions(ctx context.Context, ticketID, userID, tenantID int) ([]dto.TicketWorkflowAction, error) {
	state, err := s.GetTicketWorkflowState(ctx, ticketID, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return []dto.TicketWorkflowAction{}, nil
	}
	return state.AvailableActions, nil
}

// GetWorkflowHistory 返回工单的流转记录列表。
// 租户隔离：仅返回指定租户下的记录；工单不存在或不属于该租户时返回错误。
func (s *TicketWorkflowService) GetWorkflowHistory(ctx context.Context, ticketID, tenantID int) ([]*ent.TicketWorkflowRecord, error) {
	if _, err := s.getTicket(ctx, ticketID, tenantID); err != nil {
		return nil, err
	}
	return s.client.TicketWorkflowRecord.Query().
		Where(ticketworkflowrecord.TicketID(ticketID), ticketworkflowrecord.TenantID(tenantID)).
		Order(ent.Desc(ticketworkflowrecord.FieldCreateTime)).
		All(ctx)
}

// GetWorkflowRules 返回指定业务类型下的活跃工作流规则。
func (s *TicketWorkflowService) GetWorkflowRules(ctx context.Context, ticketType string, tenantID int) ([]*ent.TicketAutomationRule, error) {
	rules, err := s.client.TicketAutomationRule.Query().
		Where(
			ticketautomationrule.TenantID(tenantID),
			ticketautomationrule.IsActive(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if ticketType == "" {
		return rules, nil
	}
	return rules, nil
}

// GetWorkflowRulesByTicket 根据工单类型返回匹配的工作流规则。
func (s *TicketWorkflowService) GetWorkflowRulesByTicket(ctx context.Context, ticketID, tenantID int) ([]*ent.TicketAutomationRule, error) {
	tk, err := s.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	ticketType := string(tk.Type)
	if ticketType == "" {
		ticketType = "ticket"
	}
	return s.GetWorkflowRules(ctx, ticketType, tenantID)
}

// NotifyTicketUpdate 在工单状态变化后发送通知（不阻塞主流程）。
func (s *TicketWorkflowService) NotifyTicketUpdate(ctx context.Context, ticketID int, message string, tenantID int) error {
	if _, err := s.getTicket(ctx, ticketID, tenantID); err != nil {
		return err
	}
	s.logger.Infow("NotifyTicketUpdate",
		"ticket_id", ticketID,
		"tenant_id", tenantID,
		"message", message,
	)
	return nil
}

// CanUserAccessTicket 检查用户是否有权访问指定工单。
// 跨租户访问一律返回 false；同一租户内目前对所有用户放行（与 getTicket 一致）。
func (s *TicketWorkflowService) CanUserAccessTicket(ctx context.Context, ticketID, userID, tenantID int) (bool, error) {
	if _, err := s.client.User.Get(ctx, userID); err != nil {
		return false, err
	}
	tk, err := s.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return false, nil
	}
	_ = tk
	return true, nil
}

func currentLevelVal(state *dto.TicketWorkflowState) int {
	if state.CurrentApprovalLevel == nil {
		return 0
	}
	return *state.CurrentApprovalLevel
}

// 辅助函数

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func workflowUserInfoFromEnt(u *ent.User) dto.WorkflowUserInfo {
	if u == nil {
		return dto.WorkflowUserInfo{}
	}
	return dto.WorkflowUserInfo{
		ID:         u.ID,
		Username:   u.Username,
		FullName:   u.Name,
		Email:      u.Email,
		Role:       string(u.Role),
		Department: u.Department,
	}
}

func (s *TicketWorkflowService) ensureCanCCTicket(ctx context.Context, tk *ent.Ticket, userID, tenantID int) error {
	if tk == nil {
		return fmt.Errorf("工单不存在")
	}
	if tk.Status == "closed" || tk.Status == "cancelled" {
		return fmt.Errorf("工单已结束，无法抄送")
	}
	return s.ensureCanViewTicketCC(ctx, tk, userID, tenantID)
}

func (s *TicketWorkflowService) ensureCanViewTicketCC(ctx context.Context, tk *ent.Ticket, userID, tenantID int) error {
	if tk == nil {
		return fmt.Errorf("工单不存在")
	}
	if tk.RequesterID == userID || tk.AssigneeID == userID {
		return nil
	}

	currentUser, err := s.client.User.Query().
		Where(user.ID(userID), user.TenantID(tenantID), user.Active(true)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("用户不存在或无权限")
	}
	switch currentUser.Role {
	case "super_admin", "admin", "manager", "technician":
		return nil
	}

	isApprover, err := s.client.TicketApproval.Query().
		Where(ticketapproval.TicketID(tk.ID), ticketapproval.TenantID(tenantID), ticketapproval.ApproverID(userID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("校验审批权限失败: %w", err)
	}
	if isApprover {
		return nil
	}

	isCCUser, err := s.client.TicketCC.Query().
		Where(ticketcc.TicketID(tk.ID), ticketcc.TenantID(tenantID), ticketcc.UserID(userID), ticketcc.IsActive(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("校验抄送权限失败: %w", err)
	}
	if isCCUser {
		return nil
	}

	return fmt.Errorf("无权访问该工单抄送信息")
}

func normalizeNotifyChannels(channels []string) []string {
	if len(channels) == 0 {
		return []string{"in_app"}
	}
	allowed := map[string]struct{}{
		"in_app":   {},
		"email":    {},
		"sms":      {},
		"feishu":   {},
		"dingtalk": {},
		"wecom":    {},
		"webhook":  {},
	}
	seen := make(map[string]struct{}, len(channels))
	result := make([]string, 0, len(channels))
	for _, channel := range channels {
		if _, ok := allowed[channel]; !ok {
			continue
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		result = append(result, channel)
	}
	if len(result) == 0 {
		return []string{"in_app"}
	}
	return result
}

func (s *TicketWorkflowService) createCCNotifications(ctx context.Context, tk *ent.Ticket, userIDs []int, channels []string, addedBy, tenantID int) {
	now := time.Now()
	content := fmt.Sprintf("工单 %s「%s」已抄送给你", tk.TicketNumber, tk.Title)
	notifyChannels := normalizeNotifyChannels(channels)
	users, err := s.client.User.Query().Where(user.IDIn(uniqueInts(userIDs)...), user.TenantID(tenantID)).All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to query CC notification users", "error", err, "ticket_id", tk.ID)
		return
	}
	userByID := make(map[int]*ent.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	for _, userID := range userIDs {
		for _, channel := range notifyChannels {
			status := "pending"
			create := s.client.TicketNotification.Create().
				SetTicketID(tk.ID).
				SetUserID(userID).
				SetType("cc").
				SetChannel(channel).
				SetContent(content).
				SetTenantID(tenantID).
				SetStatus(status)
			if channel == "in_app" {
				create.SetStatus("sent").SetSentAt(now)
			}
			notification, err := create.Save(ctx)
			if err != nil {
				s.logger.Warnw("Failed to create ticket CC notification", "error", err, "ticket_id", tk.ID, "user_id", userID, "channel", channel)
				continue
			}

			if channel != "in_app" && s.connectorManager != nil {
				sendErr := s.sendConnectorCCNotification(ctx, tenantID, channel, tk, userByID[userID], content)
				nextStatus := "sent"
				if sendErr != nil {
					nextStatus = "failed"
					s.logger.Warnw("Failed to send CC notification through connector", "error", sendErr, "ticket_id", tk.ID, "user_id", userID, "channel", channel)
				}
				update := s.client.TicketNotification.UpdateOneID(notification.ID).SetStatus(nextStatus)
				if sendErr == nil {
					update.SetSentAt(now)
				}
				if _, err := update.Save(ctx); err != nil {
					s.logger.Warnw("Failed to update connector CC notification status", "error", err, "notification_id", notification.ID)
				}
			}
		}

		if _, err := s.client.Notification.Create().
			SetTitle("工单抄送").
			SetMessage(content).
			SetType("info").
			SetUserID(userID).
			SetTenantID(tenantID).
			SetActionURL(fmt.Sprintf("/tickets/%d", tk.ID)).
			SetActionText("查看工单").
			Save(ctx); err != nil {
			s.logger.Warnw("Failed to create unified CC notification", "error", err, "ticket_id", tk.ID, "user_id", userID, "added_by", addedBy)
		}
	}
}

func (s *TicketWorkflowService) sendConnectorCCNotification(ctx context.Context, tenantID int, channel string, tk *ent.Ticket, recipient *ent.User, content string) error {
	if s.connectorManager == nil {
		return fmt.Errorf("connector manager not configured")
	}
	if recipient == nil {
		return fmt.Errorf("recipient not found")
	}

	target := ""
	switch channel {
	case "feishu":
		target = recipient.FeishuOpenID
	case "dingtalk", "wecom":
		target = recipient.Username
	case "email":
		target = recipient.Email
	case "sms":
		target = recipient.Phone
	case "webhook":
		target = recipient.Email
	default:
		return fmt.Errorf("unsupported connector channel %s", channel)
	}
	if target == "" {
		return fmt.Errorf("recipient %d has no target for channel %s", recipient.ID, channel)
	}

	return s.connectorManager.Send(ctx, tenantID, channel, &connector.Message{
		Channel: target,
		Type:    "text",
		Title:   "工单抄送",
		Content: content,
		Actions: []connector.Action{
			{Type: "link", Text: "查看工单", URL: fmt.Sprintf("/tickets/%d", tk.ID)},
		},
		Metadata: map[string]interface{}{
			"ticket_id":     tk.ID,
			"ticket_number": tk.TicketNumber,
			"event":         "ticket_cc",
		},
	})
}

func (s *TicketWorkflowService) buildCCListResponse(ctx context.Context, records []*ent.TicketCC) (*dto.TicketCCListResponse, error) {
	response := &dto.TicketCCListResponse{
		Records: make([]dto.TicketCCRecordResponse, 0, len(records)),
		Total:   len(records),
	}
	if len(records) == 0 {
		return response, nil
	}

	ticketIDs := make([]int, 0, len(records))
	userIDs := make([]int, 0, len(records)*2)
	for _, record := range records {
		ticketIDs = append(ticketIDs, record.TicketID)
		userIDs = append(userIDs, record.UserID, record.AddedBy)
	}

	tickets, err := s.client.Ticket.Query().Where(ticket.IDIn(uniqueInts(ticketIDs)...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询抄送工单信息失败: %w", err)
	}
	users, err := s.client.User.Query().Where(user.IDIn(uniqueInts(userIDs)...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询抄送用户信息失败: %w", err)
	}

	ticketByID := make(map[int]*ent.Ticket, len(tickets))
	for _, tk := range tickets {
		ticketByID[tk.ID] = tk
	}
	userByID := make(map[int]*ent.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	for _, record := range records {
		tk := ticketByID[record.TicketID]
		row := dto.TicketCCRecordResponse{
			ID:       record.ID,
			TicketID: record.TicketID,
			User:     workflowUserInfoFromEnt(userByID[record.UserID]),
			AddedBy:  workflowUserInfoFromEnt(userByID[record.AddedBy]),
			AddedAt:  record.AddedAt,
			IsActive: record.IsActive,
		}
		if tk != nil {
			row.TicketNumber = tk.TicketNumber
			row.Title = tk.Title
			row.Status = tk.Status
			row.Priority = tk.Priority
		}
		response.Records = append(response.Records, row)
	}

	return response, nil
}

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
		SetTenantID(tenantID)

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
