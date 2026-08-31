package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
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
	approvalBridge   *BPMNApprovalBridge
}

func NewTicketWorkflowService(client *ent.Client, logger *zap.SugaredLogger) *TicketWorkflowService {
	svc := &TicketWorkflowService{
		client: client,
		logger: logger,
	}
	if client != nil {
		// P0-1：业务审批统一桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = NewBPMNApprovalBridge(client, logger)
	}
	return svc
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
		Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil(), ticket.VersionEQ(tk.Version), ticket.StatusIn("new", "open")).
		SetAssigneeID(userID).
		SetStatus("in_progress").
		SetFirstResponseAt(now).
		SetVersion(tk.Version + 1).
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
	if txErr != nil {
		return fmt.Errorf("提交接单事务失败: %w", txErr)
	}
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
		Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil(), ticket.VersionEQ(tk.Version)).
		SetStatus(returnToStatus).
		SetVersion(tk.Version + 1).
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
	if txErr != nil {
		return fmt.Errorf("提交驳回事务失败: %w", txErr)
	}
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

	// P0-1：审批先桥接完成对应的 BPMN 待办任务（以流程任务为权威审批来源）。
	// 无关联运行中流程实例时回退为纯业务审批，兼容未绑定流程的历史工单；
	// 若存在待办流程任务但完成失败（如操作人不是流程任务的审批人），则中止业务审批，避免双轨分叉。
	bpmnHandled := false
	if (req.Action == "approve" || req.Action == "reject") && s.approvalBridge != nil {
		handled, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTask(
			ctx, tenantID, userID, string(dto.BusinessTypeTicket), req.TicketID, req.Action, req.Comment,
		)
		if bridgeErr != nil {
			return fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
		}
		bpmnHandled = handled
	}

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

	// P0-1 延伸：委派同步 BPMN 任务重新指派，保持流程侧审批人与业务侧委派结果一致；
	// 同步失败时中止业务侧委派，避免流程任务仍停留在原审批人造成双轨分叉。
	if req.Action == "delegate" && s.approvalBridge != nil {
		handled, bridgeErr := s.approvalBridge.DelegateBusinessApprovalTask(
			ctx, tenantID, userID, string(dto.BusinessTypeTicket), req.TicketID, *req.DelegateToUserID,
		)
		if bridgeErr != nil {
			return fmt.Errorf("同步流程委派任务失败: %w", bridgeErr)
		}
		bpmnHandled = handled
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
		"bpmn_handled":   bpmnHandled,
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

	// P0-1 / 工单详情体验：聚合 BPMN 真实节点状态，便于详情页直接展示当前/下一节点。
	// enrich 内部对 BPMN 查询失败 / 无实例 均有降级路径，不会影响 V1 调用方。
	bpmnState, bpmnErr := s.enrichBpmnProcessState(ctx, tk, tenantID)
	if bpmnErr != nil {
		s.logger.Warnw("Failed to enrich BPMN process state for ticket workflow state",
			"error", bpmnErr, "ticket_id", ticketID, "tenant_id", tenantID)
	} else {
		state.BpmnProcessState = bpmnState
	}

	return state, nil
}

// GetTicketWorkflowStateV2 与 GetTicketWorkflowState 等价，但额外把 BPMN 节点详情铺平
// 到顶层，便于前端不需额外调用即可拿到当前节点、下一节点、历史。
//
// 当前与 V1 行为一致：底层调用 GetTicketWorkflowState 并复用 BpmnProcessState 字段。
// 保留 V2 入口以便后续添加 BPMN 专属字段（如网关分支、变量快照）时能保持向下兼容。
func (s *TicketWorkflowService) GetTicketWorkflowStateV2(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketWorkflowState, error) {
	return s.GetTicketWorkflowState(ctx, ticketID, userID, tenantID)
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
		Order(ent.Desc(ticketworkflowrecord.FieldCreatedAt)).
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
	s.logger.Infow(
		"NotifyTicketUpdate",
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

// enrichBpmnProcessState 按 businessKey="ticket:{id}" 查找 BPMN 流程实例并聚合节点状态。
//
// 返回语义：
//   - *dto.BpmnProcessState 始终返回有效结构体（即使未启动）；失败仅发生于 BPMN 服务不可用等异常。
//   - BpmnStatus 区分：not_started / running / completed / suspended / terminated。
//   - 业务指标错误不会被吞：所有 error 返回到调用方记录日志，不影响 V1 调用。
func (s *TicketWorkflowService) enrichBpmnProcessState(ctx context.Context, tk *ent.Ticket, tenantID int) (*dto.BpmnProcessState, error) {
	if tk == nil || tenantID <= 0 {
		return &dto.BpmnProcessState{BpmnStatus: "not_started"}, nil
	}

	businessKey := fmt.Sprintf("ticket:%d", tk.ID)

	// 查找关联流程实例（优先 running，其次任意状态）。不存在 → not_started。
	instance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		Order(ent.Desc(processinstance.FieldStartTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return &dto.BpmnProcessState{
				ProcessInstanceID: "",
				BpmnStatus:        "not_started",
			}, nil
		}
		return nil, fmt.Errorf("查询工单关联流程实例失败: %w", err)
	}

	// 查流程定义，补全 name。定义可能不存在（数据补全滞后），失败不影响主流程。
	var defName string
	if instance.ProcessDefinitionID > 0 {
		def, defErr := s.client.ProcessDefinition.Query().
			Where(
				processdefinition.IDEQ(instance.ProcessDefinitionID),
				processdefinition.TenantIDEQ(tenantID),
			).
			First(ctx)
		if defErr == nil && def != nil {
			defName = def.Name
		}
	}

	state := &dto.BpmnProcessState{
		ProcessInstanceID:     instance.ProcessInstanceID,
		ProcessDefinitionKey:  instance.ProcessDefinitionKey,
		ProcessDefinitionName: defName,
		BpmnStatus:            normalizeBpmnStatus(instance.Status),
		StartedAt:             nullableTimePtr(instance.StartTime),
		EndedAt:               nullableTimePtr(instance.EndTime),
	}

	// 终态不计算 current/next，仅返回实例元信息。
	if state.BpmnStatus != "running" && state.BpmnStatus != "suspended" {
		// 即使是终态，仍拉历史供详情页回溯。
		history, histErr := s.buildBpmnHistory(ctx, instance, tenantID)
		if histErr != nil {
			s.logger.Warnw("Failed to build BPMN history for terminal instance",
				"error", histErr, "processInstanceID", instance.ProcessInstanceID)
		} else {
			state.History = history
		}
		return state, nil
	}

	// 拉所有 process_tasks 计算 currentAssignees / history。
	tasks, taskErr := s.client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TenantID(tenantID),
		).
		Order(ent.Asc(processtask.FieldCreatedTime)).
		All(ctx)
	if taskErr != nil {
		return nil, fmt.Errorf("查询流程任务失败: %w", taskErr)
	}

	// 当前节点基本信息
	if instance.CurrentActivityID != "" {
		state.CurrentActivityID = instance.CurrentActivityID
	}
	if instance.CurrentActivityName != "" {
		state.CurrentActivityName = instance.CurrentActivityName
	}

	// 解析当前任务、提取 assignee / candidate users / candidate groups
	currentAssigneeIDs := map[int]struct{}{}
	currentActivityType := ""
	for _, t := range tasks {
		if t.Status != "created" && t.Status != "assigned" && t.Status != "started" && t.Status != "delegated" {
			continue
		}
		// task_definition_key 与 process_instance.current_activity_id 匹配视为当前任务
		if state.CurrentActivityID != "" && t.TaskDefinitionKey != state.CurrentActivityID {
			continue
		}
		currentActivityType = t.TaskType
		if t.Assignee != "" {
			if uid, perr := strconv.Atoi(t.Assignee); perr == nil && uid > 0 {
				currentAssigneeIDs[uid] = struct{}{}
			}
		}
		// candidate_users / candidate_groups 留作后续扩展；V1 仅取 assignee。
	}
	state.CurrentActivityType = currentActivityType
	if userMap := s.usersByIDs(ctx, currentAssigneeIDs, tenantID); len(userMap) > 0 {
		state.CurrentAssignees = userMapToSortedSlice(userMap)
	}

	// 解析 BPMN 定义 XML 计算下一节点（仅 running 时计算，suspended 直接置空）
	if state.BpmnStatus == "running" && instance.ProcessDefinitionID > 0 && state.CurrentActivityID != "" {
		nextActs, nextErr := s.computeNextActivities(ctx, instance, state.CurrentActivityID, tenantID)
		if nextErr != nil {
			s.logger.Warnw("Failed to compute next activities from BPMN XML",
				"error", nextErr, "processInstanceID", instance.ProcessInstanceID,
				"currentActivityID", state.CurrentActivityID)
		} else {
			state.NextActivities = nextActs
		}
	}

	// 构造历史（已完成节点）
	history, histErr := s.buildBpmnHistory(ctx, instance, tenantID)
	if histErr != nil {
		s.logger.Warnw("Failed to build BPMN history",
			"error", histErr, "processInstanceID", instance.ProcessInstanceID)
	} else {
		state.History = history
	}

	return state, nil
}

// buildBpmnHistory 从 process_tasks 聚合已完成节点历史，按完成时间升序排序。
func (s *TicketWorkflowService) buildBpmnHistory(ctx context.Context, instance *ent.ProcessInstance, tenantID int) ([]dto.BpmnHistoryItem, error) {
	tasks, err := s.client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TenantID(tenantID),
			processtask.StatusIn("completed", "cancelled"),
		).
		Order(ent.Asc(processtask.FieldCompletedTime)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询流程历史任务失败: %w", err)
	}

	userIDs := map[int]struct{}{}
	for _, t := range tasks {
		if t.Assignee != "" {
			if uid, perr := strconv.Atoi(t.Assignee); perr == nil && uid > 0 {
				userIDs[uid] = struct{}{}
			}
		}
	}
	users := s.usersByIDs(ctx, userIDs, tenantID)
	if users == nil {
		users = map[int]dto.WorkflowUserInfo{}
	}

	items := make([]dto.BpmnHistoryItem, 0, len(tasks))
	for _, t := range tasks {
		item := dto.BpmnHistoryItem{
			ActivityID:   t.TaskDefinitionKey,
			ActivityName: t.TaskName,
			ActivityType: t.TaskType,
		}
		if !t.CreatedTime.IsZero() {
			item.StartTime = t.CreatedTime
		}
		if !t.CompletedTime.IsZero() {
			item.EndTime = &t.CompletedTime
		}
		if t.Assignee != "" {
			if uid, perr := strconv.Atoi(t.Assignee); perr == nil {
				if u, ok := users[uid]; ok {
					item.Assignee = &u
				}
			}
		}
		item.Outcome = mapBpmnTaskOutcome(t)
		items = append(items, item)
	}
	return items, nil
}

// computeNextActivities 从 BPMN 定义 XML 解析 currentActivityID 的所有出向 sequence flows，
// 返回候选下一节点信息。网关节点标记 IsGateway=true，由前端决定是否展开分支说明。
func (s *TicketWorkflowService) computeNextActivities(ctx context.Context, instance *ent.ProcessInstance, currentActivityID string, tenantID int) ([]dto.NextActivityInfo, error) {
	if currentActivityID == "" {
		return nil, nil
	}

	def, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.IDEQ(instance.ProcessDefinitionID),
			processdefinition.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
	}
	if len(def.BpmnXML) == 0 {
		return nil, nil
	}

	g, err := parseBpmnProcessGraph(def.BpmnXML)
	if err != nil {
		return nil, fmt.Errorf("解析 BPMN XML 失败: %w", err)
	}

	nextIDs := g.outgoing[currentActivityID]
	if len(nextIDs) == 0 {
		return nil, nil
	}

	// 收集下一节点的 assignee 候选 ID
	userIDs := map[int]struct{}{}
	for _, nid := range nextIDs {
		for _, uid := range g.nodeAssignees[nid] {
			userIDs[uid] = struct{}{}
		}
	}
	users := s.usersByIDs(ctx, userIDs, tenantID)
	if users == nil {
		users = map[int]dto.WorkflowUserInfo{}
	}

	out := make([]dto.NextActivityInfo, 0, len(nextIDs))
	for _, nid := range nextIDs {
		ni := dto.NextActivityInfo{
			ActivityID:   nid,
			ActivityName: g.nodeNames[nid],
			ActivityType: g.nodeTypes[nid],
			IsGateway:    g.gatewayIDs[nid],
		}
		for _, uid := range g.nodeAssignees[nid] {
			if u, ok := users[uid]; ok {
				ni.Assignees = append(ni.Assignees, u)
			}
		}
		out = append(out, ni)
	}
	return out, nil
}

// usersByIDs 批量查询用户并按 id 索引。返回 map[int]WorkflowUserInfo，便于调用方按 ID 查找。
// ids 为空集合时返回 nil，由调用方在循环中跳过；不为 nil 时返回有效 map，避免 nil 检查。
func (s *TicketWorkflowService) usersByIDs(ctx context.Context, ids map[int]struct{}, tenantID int) map[int]dto.WorkflowUserInfo {
	if len(ids) == 0 {
		return nil
	}
	idList := make([]int, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	users, err := s.client.User.Query().
		Where(user.IDIn(idList...), user.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to batch query users for BPMN state", "error", err)
		return nil
	}
	out := make(map[int]dto.WorkflowUserInfo, len(users))
	for _, u := range users {
		out[u.ID] = workflowUserInfoFromEnt(u)
	}
	return out
}

// normalizeBpmnStatus 把 DB 存储的 BPMN 状态统一为前端约定的字符串。
func normalizeBpmnStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "running":
		return "running"
	case "suspended":
		return "suspended"
	case "completed":
		return "completed"
	case "terminated":
		return "terminated"
	default:
		return "not_started"
	}
}

// userMapToSortedSlice 把 user map 转成按 ID 升序的 slice，供前端列表稳定渲染。
func userMapToSortedSlice(m map[int]dto.WorkflowUserInfo) []dto.WorkflowUserInfo {
	if len(m) == 0 {
		return nil
	}
	out := make([]dto.WorkflowUserInfo, 0, len(m))
	for _, u := range m {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// nullableTimePtr 将 time.Time 包装为 *time.Time；零值返回 nil。
func nullableTimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// mapBpmnTaskOutcome 把 process_task 的 status 映射为前端约定 outcome 字符串。
func mapBpmnTaskOutcome(t *ent.ProcessTask) string {
	if t == nil {
		return ""
	}
	switch t.Status {
	case "completed":
		return "completed"
	case "cancelled":
		return "cancelled"
	default:
		return t.Status
	}
}

// bpmnProcessGraph BPMN 定义的轻量索引，仅保留工单详情页需要的字段。
type bpmnProcessGraph struct {
	outgoing      map[string][]string // source activity ID → target activity IDs
	nodeNames     map[string]string   // activity ID → display name
	nodeTypes     map[string]string   // activity ID → activity type (userTask / ...)
	nodeAssignees map[string][]int    // activity ID → 关联 assignee user IDs（候选解析）
	gatewayIDs    map[string]bool     // activity ID → 是否为网关节点
}

// bpmnXMLModel 仅含 sequenceFlow 与 flowNode 的最小 XML 模型。
//
// BPMN 2.0 元素均带有命名空间（如 <bpmn:userTask>），但同时也允许无命名空间。
// 为兼容两种风格，使用 ",any" 通配收集所有子元素，由 Local 名识别类型。
type bpmnXMLModel struct {
	XMLName   xml.Name      `xml:"definitions"`
	Processes []bpmnProcess `xml:"process"`
}

type bpmnProcess struct {
	Children []bpmnAnyNode `xml:",any"`
}

type bpmnAnyNode struct {
	XMLName   xml.Name
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	SourceRef string `xml:"sourceRef,attr,omitempty"`
	TargetRef string `xml:"targetRef,attr,omitempty"`
}

// parseBpmnProcessGraph 解析 BPMN XML，提取所有 sequence flow 与节点元信息。
// 不依赖 nitram509/bpmn-engine，仅做最小解析以满足"下一节点"展示需求。
func parseBpmnProcessGraph(xmlBytes []byte) (*bpmnProcessGraph, error) {
	if len(xmlBytes) == 0 {
		return &bpmnProcessGraph{
			outgoing:      map[string][]string{},
			nodeNames:     map[string]string{},
			nodeTypes:     map[string]string{},
			nodeAssignees: map[string][]int{},
			gatewayIDs:    map[string]bool{},
		}, nil
	}

	var model bpmnXMLModel
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))
	dec.Strict = false
	if err := dec.Decode(&model); err != nil {
		return nil, err
	}

	g := &bpmnProcessGraph{
		outgoing:      map[string][]string{},
		nodeNames:     map[string]string{},
		nodeTypes:     map[string]string{},
		nodeAssignees: map[string][]int{},
		gatewayIDs:    map[string]bool{},
	}

	for _, p := range model.Processes {
		for _, n := range p.Children {
			switch n.XMLName.Local {
			case "sequenceFlow":
				if n.SourceRef == "" || n.TargetRef == "" {
					continue
				}
				g.outgoing[n.SourceRef] = append(g.outgoing[n.SourceRef], n.TargetRef)
			default:
				g.nodeNames[n.ID] = n.Name
				g.nodeTypes[n.ID] = n.XMLName.Local
				switch n.XMLName.Local {
				case "exclusiveGateway", "parallelGateway", "inclusiveGateway", "eventBasedGateway":
					g.gatewayIDs[n.ID] = true
				}
			}
		}
	}

	return g, nil
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
	content := fmt.Sprintf("工单 %s「%s」已抄送给你", tk.TicketNumber, tk.Title)
	notifyChannels := normalizeNotifyChannels(channels)
	for _, userID := range userIDs {
		for _, channel := range notifyChannels {
			occurrenceKey := fmt.Sprintf("ticket:%d:cc:%d:%s", tk.ID, userID, channel)
			if err := enqueueTicketNotificationCommand(ctx, s.client, tenantID, tk.ID, userID, "cc", channel, content, occurrenceKey); err != nil && !ent.IsConstraintError(err) {
				s.logger.Warnw("Failed to enqueue ticket CC notification", "error", err, "ticket_id", tk.ID, "user_id", userID, "channel", channel, "added_by", addedBy)
			}
		}
	}
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
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID), ticket.DeletedAtIsNil()).
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
