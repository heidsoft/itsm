package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"time"

	"go.uber.org/zap"
)

type TicketWorkflowService struct {
	client *ent.Client
	rawDB  *sql.DB
	logger *zap.SugaredLogger
}

func NewTicketWorkflowService(client *ent.Client, rawDB *sql.DB, logger *zap.SugaredLogger) *TicketWorkflowService {
	return &TicketWorkflowService{
		client: client,
		rawDB:  rawDB,
		logger: logger,
	}
}

// AcceptTicket 接单
func (s *TicketWorkflowService) AcceptTicket(ctx context.Context, req *dto.AcceptTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Accepting ticket", "ticket_id", req.TicketID, "user_id", userID)

	// 检查工单是否存在且未分配或分配给当前用户
	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if ticket.Status != "new" && ticket.Status != "open" {
		return fmt.Errorf("工单当前状态不允许接单: %s", ticket.Status)
	}

	// 更新工单状态和分配人
	updateQuery := `
		UPDATE tickets 
		SET assignee_id = $1, status = 'in_progress', updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, userID, time.Now(), req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to accept ticket: %w", err)
	}

	// 记录流转记录
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionAccept,
		FromStatus: &ticket.Status,
		ToStatus:   ptrString("in_progress"),
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}, tenantID)

	return err
}

// RejectTicket 驳回工单
func (s *TicketWorkflowService) RejectTicket(ctx context.Context, req *dto.RejectTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Rejecting ticket", "ticket_id", req.TicketID, "user_id", userID)

	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 更新工单状态
	returnToStatus := "rejected"
	if req.ReturnToStatus != nil {
		returnToStatus = *req.ReturnToStatus
	}

	updateQuery := `
		UPDATE tickets 
		SET status = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, returnToStatus, time.Now(), req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to reject ticket: %w", err)
	}

	// 记录流转记录
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionReject,
		FromStatus: &ticket.Status,
		ToStatus:   &returnToStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Reason:     req.Reason,
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
	}, tenantID)

	return err
}

// WithdrawTicket 撤回工单
func (s *TicketWorkflowService) WithdrawTicket(ctx context.Context, req *dto.WithdrawTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Withdrawing ticket", "ticket_id", req.TicketID, "user_id", userID)

	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 检查是否是工单创建者
	if ticket.RequesterID != userID {
		return fmt.Errorf("只有工单创建者可以撤回工单")
	}

	if ticket.Status == "closed" || ticket.Status == "cancelled" {
		return fmt.Errorf("工单已关闭或取消，无法撤回")
	}

	// 更新工单状态
	updateQuery := `
		UPDATE tickets 
		SET status = 'cancelled', updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, time.Now(), req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to withdraw ticket: %w", err)
	}

	// 记录流转记录
	newStatus := "cancelled"
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionWithdraw,
		FromStatus: &ticket.Status,
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
		updateQuery := `
			UPDATE tickets 
			SET assignee_id = $1, updated_at = $2
			WHERE id = $3 AND tenant_id = $4
		`
		_, err = s.rawDB.ExecContext(ctx, updateQuery, req.ToUserID, time.Now(), req.TicketID, tenantID)
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
	_, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 添加抄送人
	for _, ccUserID := range req.CCUsers {
		insertQuery := `
			INSERT INTO ticket_cc (ticket_id, user_id, added_by, tenant_id, added_at, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (ticket_id, user_id, tenant_id) DO NOTHING
		`
		_, err = s.rawDB.ExecContext(ctx, insertQuery, req.TicketID, ccUserID, userID, tenantID, time.Now())
		if err != nil {
			s.logger.Warnw("Failed to add CC user", "error", err, "user_id", ccUserID)
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

// ApproveTicket 审批工单
func (s *TicketWorkflowService) ApproveTicket(ctx context.Context, req *dto.ApproveTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Approving ticket", "ticket_id", req.TicketID, "action", req.Action, "user_id", userID)

	// 检查工单是否存在
	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 检查审批记录是否存在
	var approvalLevel int
	var approvalStatus string
	approvalQuery := "SELECT level, status FROM ticket_approvals WHERE id = $1 AND ticket_id = $2 AND tenant_id = $3"
	err = s.rawDB.QueryRowContext(ctx, approvalQuery, req.ApprovalID, req.TicketID, tenantID).Scan(&approvalLevel, &approvalStatus)
	if err != nil {
		return fmt.Errorf("审批记录不存在")
	}

	if approvalStatus != string(dto.ApprovalStatusPending) {
		return fmt.Errorf("审批已处理，当前状态: %s", approvalStatus)
	}

	// 更新审批记录
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
		newApprovalStatus = string(dto.ApprovalStatusPending)
	default:
		return fmt.Errorf("无效的审批操作: %s", req.Action)
	}

	updateApprovalQuery := `
		UPDATE ticket_approvals 
		SET status = $1, action = $2, comment = $3, processed_at = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7
	`
	_, err = s.rawDB.ExecContext(ctx, updateApprovalQuery, 
		newApprovalStatus, req.Action, req.Comment, time.Now(), time.Now(), 
		req.ApprovalID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// 如果是委派，创建新的审批记录
	if req.Action == "delegate" && req.DelegateToUserID != nil {
		insertApprovalQuery := `
			INSERT INTO ticket_approvals (ticket_id, level, level_name, approver_id, status, tenant_id, created_at, updated_at)
			SELECT ticket_id, level, level_name, $1, $2, tenant_id, $3, $4
			FROM ticket_approvals
			WHERE id = $5
		`
		_, err = s.rawDB.ExecContext(ctx, insertApprovalQuery,
			*req.DelegateToUserID, dto.ApprovalStatusPending, time.Now(), time.Now(),
			req.ApprovalID)
		if err != nil {
			s.logger.Warnw("Failed to create delegated approval", "error", err)
		}
	}

	// 检查是否所有审批都已完成
	if req.Action == "approve" {
		// 检查是否还有下一级审批
		nextLevelQuery := `
			SELECT id, level, level_name, approver_id FROM ticket_approvals
			WHERE ticket_id = $1 AND tenant_id = $2 AND level > $3 AND status = $4
			ORDER BY level ASC LIMIT 1
		`
		var nextApprovalID int
		var nextLevel int
		var nextLevelName string
		var nextApproverID int
		err = s.rawDB.QueryRowContext(ctx, nextLevelQuery, req.TicketID, tenantID, approvalLevel, dto.ApprovalStatusPending).Scan(&nextApprovalID, &nextLevel, &nextLevelName, &nextApproverID)

		if err == sql.ErrNoRows {
			// 没有下一级审批，所有审批都已完成
			// 更新工单状态为已审批
			updateTicketQuery := `
				UPDATE tickets
				SET status = 'approved', updated_at = $1
				WHERE id = $2 AND tenant_id = $3
			`
			_, err = s.rawDB.ExecContext(ctx, updateTicketQuery, time.Now(), req.TicketID, tenantID)
			if err != nil {
				s.logger.Warnw("Failed to update ticket status to approved", "error", err)
			}
		} else if err == nil {
			// 有下一级审批，激活下一级审批
			activateQuery := `
				UPDATE ticket_approvals
				SET status = $1, updated_at = $2
				WHERE id = $3 AND tenant_id = $4
			`
			_, err = s.rawDB.ExecContext(ctx, activateQuery, dto.ApprovalStatusPending, time.Now(), nextApprovalID, tenantID)
			if err != nil {
				s.logger.Warnw("Failed to activate next level approval", "error", err)
			}
			s.logger.Infow("Activated next level approval", "ticket_id", req.TicketID, "next_level", nextLevel)
		} else {
			s.logger.Warnw("Failed to check next level approval", "error", err)
		}
	} else if req.Action == "reject" {
		// 审批拒绝，更新工单状态
		updateTicketQuery := `
			UPDATE tickets
			SET status = 'rejected', updated_at = $1
			WHERE id = $2 AND tenant_id = $3
		`
		_, err = s.rawDB.ExecContext(ctx, updateTicketQuery, time.Now(), req.TicketID, tenantID)
		if err != nil {
			s.logger.Warnw("Failed to update ticket status", "error", err)
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

	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     action,
		FromStatus: &ticket.Status,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Comment,
		CreatedAt:  time.Now(),
		Metadata:   metadata,
	}, tenantID)

	return err
}

// ResolveTicket 解决工单
func (s *TicketWorkflowService) ResolveTicket(ctx context.Context, req *dto.ResolveTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Resolving ticket", "ticket_id", req.TicketID, "user_id", userID)

	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	// 更新工单状态
	updateQuery := `
		UPDATE tickets 
		SET status = 'resolved', resolution = $1, resolution_category = $2, resolved_at = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, 
		req.Resolution, req.ResolutionCategory, time.Now(), time.Now(), 
		req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to resolve ticket: %w", err)
	}

	// 记录流转记录
	newStatus := "resolved"
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionResolve,
		FromStatus: &ticket.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.Resolution,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"resolution_category": req.ResolutionCategory,
			"work_notes":          req.WorkNotes,
		},
	}, tenantID)

	return err
}

// CloseTicket 关闭工单
func (s *TicketWorkflowService) CloseTicket(ctx context.Context, req *dto.CloseTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Closing ticket", "ticket_id", req.TicketID, "user_id", userID)

	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if ticket.Status != "resolved" {
		return fmt.Errorf("只有已解决的工单才能关闭")
	}

	// 更新工单状态
	updateQuery := `
		UPDATE tickets 
		SET status = 'closed', closed_at = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, time.Now(), time.Now(), req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to close ticket: %w", err)
	}

	// 记录流转记录
	newStatus := "closed"
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionClose,
		FromStatus: &ticket.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Comment:    req.CloseNotes,
		CreatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"close_reason": req.CloseReason,
		},
	}, tenantID)

	return err
}

// ReopenTicket 重开工单
func (s *TicketWorkflowService) ReopenTicket(ctx context.Context, req *dto.ReopenTicketRequest, userID, tenantID int) error {
	s.logger.Infow("Reopening ticket", "ticket_id", req.TicketID, "user_id", userID)

	ticket, err := s.getTicket(ctx, req.TicketID, tenantID)
	if err != nil {
		return err
	}

	if ticket.Status != "closed" && ticket.Status != "resolved" {
		return fmt.Errorf("只有已关闭或已解决的工单才能重开")
	}

	// 更新工单状态
	updateQuery := `
		UPDATE tickets 
		SET status = 'open', updated_at = $1
		WHERE id = $2 AND tenant_id = $3
	`
	_, err = s.rawDB.ExecContext(ctx, updateQuery, time.Now(), req.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to reopen ticket: %w", err)
	}

	// 记录流转记录
	newStatus := "open"
	err = s.createWorkflowRecord(ctx, &dto.TicketWorkflowRecord{
		TicketID:   req.TicketID,
		Action:     dto.WorkflowActionReopen,
		FromStatus: &ticket.Status,
		ToStatus:   &newStatus,
		Operator:   dto.WorkflowUserInfo{ID: userID},
		Reason:     req.Reason,
		CreatedAt:  time.Now(),
	}, tenantID)

	return err
}

// GetTicketWorkflowState 获取工单流转状态
func (s *TicketWorkflowService) GetTicketWorkflowState(ctx context.Context, ticketID, userID, tenantID int) (*dto.TicketWorkflowState, error) {
	// 查询工单信息
	ticket, err := s.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 查询审批信息
	approvalQuery := `
		SELECT id, level, level_name, status, approver_id FROM ticket_approvals
		WHERE ticket_id = $1 AND tenant_id = $2
		ORDER BY level ASC
	`
	rows, err := s.rawDB.QueryContext(ctx, approvalQuery, ticketID, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to query approval status", "error", err)
	}
	defer rows.Close()

	var approvals []struct {
		ID          int
		Level       int
		LevelName   string
		Status      string
		ApproverID  int
	}
	for rows.Next() {
		var a struct {
			ID          int
			Level       int
			LevelName   string
			Status      string
			ApproverID  int
		}
		if err := rows.Scan(&a.ID, &a.Level, &a.LevelName, &a.Status, &a.ApproverID); err != nil {
			s.logger.Warnw("Failed to scan approval row", "error", err)
			continue
		}
		approvals = append(approvals, a)
	}

	var approvalStatus *dto.ApprovalStatus
	var currentLevel, totalLevels *int
	if len(approvals) > 0 {
		totalLevelsVal := len(approvals)
		totalLevels = &totalLevelsVal

		// 找到当前待审批的级别
		for _, a := range approvals {
			if a.Status == string(dto.ApprovalStatusPending) {
				currentLevel = &a.Level
				approvalStatus = ptrToApprovalStatus(a.Status)
				break
			}
			// 如果有已拒绝的，状态就是已拒绝
			if a.Status == string(dto.ApprovalStatusRejected) {
				approvalStatus = ptrToApprovalStatus(a.Status)
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
			approvalStatus = ptrToApprovalStatus(string(dto.ApprovalStatusApproved))
		}
	}

	// 构建工单流转状态
	state := &dto.TicketWorkflowState{
		TicketID:             ticketID,
		CurrentStatus:        ticket.Status,
		ApprovalStatus:       approvalStatus,
		CurrentApprovalLevel: currentLevel,
		TotalApprovalLevels:  totalLevels,
		AvailableActions:     []dto.TicketWorkflowAction{},
	}

	// 根据当前状态和用户权限判断可执行的操作
	switch ticket.Status {
	case "new", "open":
		state.CanAccept = true
		state.CanForward = true
		state.CanCC = true
		state.AvailableActions = append(state.AvailableActions,
			dto.WorkflowActionAccept,
			dto.WorkflowActionForward,
			dto.WorkflowActionCC)

		if ticket.RequesterID == userID {
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

func (s *TicketWorkflowService) getTicket(ctx context.Context, ticketID, tenantID int) (*struct {
	ID          int
	Status      string
	RequesterID int
	AssigneeID  *int
}, error) {
	query := "SELECT id, status, requester_id, assignee_id FROM tickets WHERE id = $1 AND tenant_id = $2"
	
	var ticket struct {
		ID          int
		Status      string
		RequesterID int
		AssigneeID  *int
	}
	
	err := s.rawDB.QueryRowContext(ctx, query, ticketID, tenantID).Scan(
		&ticket.ID, &ticket.Status, &ticket.RequesterID, &ticket.AssigneeID)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("工单不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	
	return &ticket, nil
}

func (s *TicketWorkflowService) createWorkflowRecord(ctx context.Context, record *dto.TicketWorkflowRecord, tenantID int) error {
	metadataJSON := SafeMarshal(record.Metadata)
	
	query := `
		INSERT INTO ticket_workflow_records (
			ticket_id, action, from_status, to_status, operator_id, from_user_id, to_user_id,
			comment, reason, metadata, tenant_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	var fromUserID, toUserID *int
	if record.FromUser != nil {
		fromUserID = &record.FromUser.ID
	}
	if record.ToUser != nil {
		toUserID = &record.ToUser.ID
	}
	
	_, err := s.rawDB.ExecContext(ctx, query,
		record.TicketID, record.Action, record.FromStatus, record.ToStatus,
		record.Operator.ID, fromUserID, toUserID,
		record.Comment, record.Reason, metadataJSON,
		tenantID, record.CreatedAt,
	)
	
	return err
}

func ptrString(s string) *string {
	return &s
}
