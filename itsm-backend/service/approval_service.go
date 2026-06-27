package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/approvalrecord"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/service/approver"

	"go.uber.org/zap"
)

type ApprovalService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewApprovalService(client *ent.Client, logger *zap.SugaredLogger) *ApprovalService {
	return &ApprovalService{
		client: client,
		logger: logger,
	}
}

// CreateWorkflow 创建审批工作流
func (s *ApprovalService) CreateWorkflow(ctx context.Context, req *dto.CreateApprovalWorkflowRequest, tenantID int) (*dto.ApprovalWorkflowResponse, error) {
	s.logger.Infow("Creating approval workflow", "name", req.Name, "tenant_id", tenantID)

	// 转换节点配置为map格式
	nodesMap := make([]map[string]interface{}, len(req.Nodes))
	for i, node := range req.Nodes {
		nodeMap := map[string]interface{}{
			"level":          node.Level,
			"name":           node.Name,
			"approver_type":  node.ApproverType,
			"approver_ids":   node.ApproverIDs,
			"approval_mode":  node.ApprovalMode,
			"allow_reject":   node.AllowReject,
			"allow_delegate": node.AllowDelegate,
			"reject_action":  node.RejectAction,
		}
		if node.AssigneeType != "" {
			nodeMap["assignee_type"] = node.AssigneeType
		}
		if node.AssigneeValue != "" {
			nodeMap["assignee_value"] = node.AssigneeValue
		}
		if node.MinimumApprovals != nil {
			nodeMap["minimum_approvals"] = *node.MinimumApprovals
		}
		if node.TimeoutHours != nil {
			nodeMap["timeout_hours"] = *node.TimeoutHours
		}
		if len(node.Conditions) > 0 {
			conditionsMap := make([]map[string]interface{}, len(node.Conditions))
			for j, cond := range node.Conditions {
				conditionsMap[j] = map[string]interface{}{
					"field":    cond.Field,
					"operator": cond.Operator,
					"value":    cond.Value,
				}
			}
			nodeMap["conditions"] = conditionsMap
		}
		if node.ReturnToLevel != nil {
			nodeMap["return_to_level"] = *node.ReturnToLevel
		}
		nodesMap[i] = nodeMap
	}

	workflow, err := s.client.ApprovalWorkflow.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetNodes(nodesMap).
		SetIsActive(req.IsActive).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create approval workflow", "error", err)
		return nil, fmt.Errorf("failed to create approval workflow: %w", err)
	}

	if req.TicketType != nil {
		workflow, err = workflow.Update().SetTicketType(*req.TicketType).Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update ticket type", "error", err)
		}
	}
	if req.Priority != nil {
		workflow, err = workflow.Update().SetPriority(*req.Priority).Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update priority", "error", err)
		}
	}

	return s.toWorkflowResponse(ctx, workflow), nil
}

// UpdateWorkflow 更新审批工作流
func (s *ApprovalService) UpdateWorkflow(ctx context.Context, id int, req *dto.UpdateApprovalWorkflowRequest, tenantID int) (*dto.ApprovalWorkflowResponse, error) {
	s.logger.Infow("Updating approval workflow", "id", id, "tenant_id", tenantID)

	update := s.client.ApprovalWorkflow.Update().
		Where(
			approvalworkflow.IDEQ(id),
			approvalworkflow.TenantIDEQ(tenantID),
		).
		SetUpdatedAt(time.Now())

	if req.Name != nil {
		update = update.SetName(*req.Name)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.TicketType != nil {
		update = update.SetTicketType(*req.TicketType)
	}
	if req.Priority != nil {
		update = update.SetPriority(*req.Priority)
	}
	if req.Nodes != nil {
		// 转换节点配置为map格式
		nodesMap := make([]map[string]interface{}, len(*req.Nodes))
		for i, node := range *req.Nodes {
			nodeMap := map[string]interface{}{
				"level":          node.Level,
				"name":           node.Name,
				"approver_type":  node.ApproverType,
				"approver_ids":   node.ApproverIDs,
				"approval_mode":  node.ApprovalMode,
				"allow_reject":   node.AllowReject,
				"allow_delegate": node.AllowDelegate,
				"reject_action":  node.RejectAction,
			}
			if node.AssigneeType != "" {
				nodeMap["assignee_type"] = node.AssigneeType
			}
			if node.AssigneeValue != "" {
				nodeMap["assignee_value"] = node.AssigneeValue
			}
			if node.MinimumApprovals != nil {
				nodeMap["minimum_approvals"] = *node.MinimumApprovals
			}
			if node.TimeoutHours != nil {
				nodeMap["timeout_hours"] = *node.TimeoutHours
			}
			if len(node.Conditions) > 0 {
				conditionsMap := make([]map[string]interface{}, len(node.Conditions))
				for j, cond := range node.Conditions {
					conditionsMap[j] = map[string]interface{}{
						"field":    cond.Field,
						"operator": cond.Operator,
						"value":    cond.Value,
					}
				}
				nodeMap["conditions"] = conditionsMap
			}
			if node.ReturnToLevel != nil {
				nodeMap["return_to_level"] = *node.ReturnToLevel
			}
			nodesMap[i] = nodeMap
		}
		update = update.SetNodes(nodesMap)
	}
	if req.IsActive != nil {
		update = update.SetIsActive(*req.IsActive)
	}

	_, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update approval workflow", "error", err)
		return nil, fmt.Errorf("failed to update approval workflow: %w", err)
	}

	// 重新获取更新后的工作流
	workflow, err := s.client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.IDEQ(id),
			approvalworkflow.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated workflow: %w", err)
	}

	return s.toWorkflowResponse(ctx, workflow), nil
}

// DeleteWorkflow 删除审批工作流
func (s *ApprovalService) DeleteWorkflow(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting approval workflow", "id", id, "tenant_id", tenantID)

	// 先检查是否存在
	_, err := s.client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.IDEQ(id),
			approvalworkflow.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("workflow not found")
		}
		return fmt.Errorf("failed to query workflow: %w", err)
	}

	// 删除
	err = s.client.ApprovalWorkflow.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete approval workflow", "error", err)
		return fmt.Errorf("failed to delete approval workflow: %w", err)
	}

	return nil
}

// ListWorkflows 获取审批工作流列表
func (s *ApprovalService) ListWorkflows(ctx context.Context, filters map[string]interface{}, tenantID int, page, pageSize int) ([]*dto.ApprovalWorkflowResponse, int, error) {
	s.logger.Infow("Listing approval workflows", "tenant_id", tenantID)

	query := s.client.ApprovalWorkflow.Query().
		Where(approvalworkflow.TenantIDEQ(tenantID))

	if ticketType, ok := filters["ticket_type"].(string); ok && ticketType != "" {
		query = query.Where(approvalworkflow.TicketTypeEQ(ticketType))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(approvalworkflow.PriorityEQ(priority))
	}
	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where(approvalworkflow.IsActiveEQ(isActive))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count approval workflows", "error", err)
		return nil, 0, fmt.Errorf("failed to count approval workflows: %w", err)
	}

	// 分页查询
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	workflows, err := query.
		Order(ent.Desc(approvalworkflow.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list approval workflows", "error", err)
		return nil, 0, fmt.Errorf("failed to list approval workflows: %w", err)
	}

	responses := make([]*dto.ApprovalWorkflowResponse, len(workflows))
	for i, workflow := range workflows {
		responses[i] = s.toWorkflowResponse(ctx, workflow)
	}

	return responses, total, nil
}

// GetWorkflow 获取审批工作流详情
func (s *ApprovalService) GetWorkflow(ctx context.Context, id int, tenantID int) (*dto.ApprovalWorkflowResponse, error) {
	s.logger.Infow("Getting approval workflow", "id", id, "tenant_id", tenantID)

	workflow, err := s.client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.IDEQ(id),
			approvalworkflow.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get approval workflow", "error", err)
		return nil, fmt.Errorf("failed to get approval workflow: %w", err)
	}

	return s.toWorkflowResponse(ctx, workflow), nil
}

// GetApprovalRecords 获取审批记录
func (s *ApprovalService) GetApprovalRecords(ctx context.Context, req *dto.GetApprovalRecordsRequest, tenantID int) ([]*dto.ApprovalRecordResponse, int, error) {
	s.logger.Infow("Getting approval records", "tenant_id", tenantID)

	query := s.client.ApprovalRecord.Query().
		Where(approvalrecord.TenantIDEQ(tenantID))

	if req.TicketID != nil {
		query = query.Where(approvalrecord.TicketIDEQ(*req.TicketID))
	}
	if req.WorkflowID != nil {
		query = query.Where(approvalrecord.WorkflowIDEQ(*req.WorkflowID))
	}
	if req.Status != nil && *req.Status != "" {
		query = query.Where(approvalrecord.StatusEQ(*req.Status))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count approval records", "error", err)
		return nil, 0, fmt.Errorf("failed to count approval records: %w", err)
	}

	// 分页查询
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	records, err := query.
		Order(ent.Desc(approvalrecord.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get approval records", "error", err)
		return nil, 0, fmt.Errorf("failed to get approval records: %w", err)
	}

	responses := make([]*dto.ApprovalRecordResponse, len(records))
	for i, record := range records {
		responses[i] = s.toRecordResponse(record)
	}

	return responses, total, nil
}

// SubmitApproval 提交审批
func (s *ApprovalService) SubmitApproval(ctx context.Context, recordID int, userID int, action string, comment string, delegateToUserID *int, tenantID int) error {
	s.logger.Infow("Submitting approval", "record_id", recordID, "user_id", userID, "action", action, "tenant_id", tenantID)

	// 获取审批记录
	approvalRecord, err := s.client.ApprovalRecord.Query().
		Where(
			approvalrecord.IDEQ(recordID),
			approvalrecord.TenantIDEQ(tenantID),
			approvalrecord.StatusEQ("pending"),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("approval record not found or already processed: %w", err)
	}

	// 权限检查：验证用户是否是该审批记录的指定审批人
	if approvalRecord.ApproverID != userID {
		s.logger.Warnw("User is not the assigned approver", "user_id", userID, "approver_id", approvalRecord.ApproverID, "record_id", recordID)
		return fmt.Errorf("user is not authorized to approve this record")
	}

	// 获取审批工作流以检查操作权限
	workflow, err := s.client.ApprovalWorkflow.Query().
		Where(approvalworkflow.IDEQ(approvalRecord.WorkflowID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get approval workflow: %w", err)
	}

	// 验证当前节点是否允许该操作
	if !s.canPerformAction(workflow, approvalRecord.CurrentLevel, action) {
		s.logger.Warnw("Action not allowed at current level", "action", action, "level", approvalRecord.CurrentLevel)
		return fmt.Errorf("action '%s' is not allowed at this approval level", action)
	}

	// 更新审批记录
	var newStatus string
	switch action {
	case "approve":
		newStatus = "approved"
	case "reject":
		newStatus = "rejected"
	case "delegate":
		newStatus = "delegated"
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	update := s.client.ApprovalRecord.UpdateOneID(recordID).
		SetStatus(newStatus).
		SetAction(action).
		SetProcessedAt(time.Now())

	if comment != "" {
		update = update.SetComment(comment)
	}

	_, err = update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update approval record", "error", err)
		return fmt.Errorf("failed to update approval record: %w", err)
	}

	// 处理审批后的逻辑
	switch action {
	case "approve":
		if err := s.handleApprovalApproved(ctx, approvalRecord); err != nil {
			s.logger.Errorw("Failed to handle approved action", "error", err)
			return fmt.Errorf("failed to handle approved action: %w", err)
		}
	case "reject":
		rejectAction := "terminate" // 默认拒绝动作，可以从参数中获取
		if err := s.handleApprovalRejected(ctx, approvalRecord, rejectAction); err != nil {
			s.logger.Errorw("Failed to handle rejected action", "error", err)
			return fmt.Errorf("failed to handle rejected action: %w", err)
		}
	case "delegate":
		if delegateToUserID != nil {
			if err := s.handleApprovalDelegated(ctx, approvalRecord, *delegateToUserID); err != nil {
				s.logger.Errorw("Failed to handle delegated action", "error", err)
				return fmt.Errorf("failed to handle delegated action: %w", err)
			}
		} else {
			return fmt.Errorf("delegate_to_user_id is required for delegation")
		}
	}

	return nil
}

// handleApprovalApproved 处理审批通过
func (s *ApprovalService) handleApprovalApproved(ctx context.Context, record *ent.ApprovalRecord) error {
	// 检查该工单在该工作流下是否还有待审批项
	remainingApprovals, err := s.client.ApprovalRecord.Query().
		Where(
			approvalrecord.WorkflowIDEQ(record.WorkflowID),
			approvalrecord.TicketIDEQ(record.TicketID),
			approvalrecord.TenantIDEQ(record.TenantID),
			approvalrecord.StatusEQ("pending"),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count remaining approvals: %w", err)
	}

	// 如果没有剩余的待审批项，标记工单为已审批
	if remainingApprovals == 0 {
		_, err := s.client.Ticket.UpdateOneID(record.TicketID).
			Where(ticket.TenantIDEQ(record.TenantID)).
			SetStatus("approved").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update ticket status to approved: %w", err)
		}
	}

	return nil
}

// handleApprovalRejected 处理审批拒绝
func (s *ApprovalService) handleApprovalRejected(ctx context.Context, record *ent.ApprovalRecord, rejectAction string) error {
	// 根据拒绝动作处理
	switch rejectAction {
	case "terminate":
		// 取消同一工单、同一工作流中的其他待审批项
		_, err := s.client.ApprovalRecord.Update().
			Where(
				approvalrecord.WorkflowIDEQ(record.WorkflowID),
				approvalrecord.TicketIDEQ(record.TicketID),
				approvalrecord.TenantIDEQ(record.TenantID),
				approvalrecord.StatusEQ("pending"),
			).
			SetStatus("cancelled").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to cancel remaining approvals: %w", err)
		}

		_, err = s.client.Ticket.UpdateOneID(record.TicketID).
			Where(ticket.TenantIDEQ(record.TenantID)).
			SetStatus("rejected").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update ticket status to rejected: %w", err)
		}

	case "return_to_submitter":
		return nil

	default:
		// 默认行为：终止工作流
		return s.handleApprovalRejected(ctx, record, "terminate")
	}

	return nil
}

// canPerformAction 检查在指定审批级别是否允许执行该操作
func (s *ApprovalService) canPerformAction(workflow *ent.ApprovalWorkflow, level int, action string) bool {
	if len(workflow.Nodes) == 0 {
		// 如果没有节点配置，默认允许所有操作
		return true
	}

	// 查找当前级别的节点配置
	for _, nodeMap := range workflow.Nodes {
		if nodeLevel, ok := nodeMap["level"].(float64); ok && int(nodeLevel) == level {
			// 检查各个操作的允许状态
			switch action {
			case "approve":
				return true // 审批通过总是允许的
			case "reject":
				if allowReject, ok := nodeMap["allow_reject"].(bool); ok {
					return allowReject
				}
				return false // 默认不允许拒绝
			case "delegate":
				if allowDelegate, ok := nodeMap["allow_delegate"].(bool); ok {
					return allowDelegate
				}
				return false // 默认不允许委托
			}
		}
	}

	return true // 未找到节点配置时默认允许
}

// handleApprovalDelegated 处理审批委托
func (s *ApprovalService) handleApprovalDelegated(ctx context.Context, record *ent.ApprovalRecord, delegateTo int) error {
	delegateUser, err := s.client.User.Query().
		Where(
			user.IDEQ(delegateTo),
			user.TenantIDEQ(record.TenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get delegate user: %w", err)
	}

	// 创建新的审批记录给被委托人
	_, err = s.client.ApprovalRecord.Create().
		SetWorkflowID(record.WorkflowID).
		SetWorkflowName(record.WorkflowName).
		SetTicketID(record.TicketID).
		SetTicketNumber(record.TicketNumber).
		SetTicketTitle(record.TicketTitle).
		SetCurrentLevel(record.CurrentLevel).
		SetTotalLevels(record.TotalLevels).
		SetApproverID(delegateTo).
		SetApproverName(delegateUser.Name).
		SetStepOrder(record.StepOrder).
		SetStatus("pending").
		SetNillableDueDate(record.DueDate).
		SetTenantID(record.TenantID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create delegated approval record: %w", err)
	}

	// 可以在这里发送通知给被委托人
	s.logger.Infow("Approval delegated", "originalApprover", record.ApproverID, "delegateTo", delegateTo)

	return nil
}

// 辅助方法：转换为响应DTO
func (s *ApprovalService) toWorkflowResponse(ctx context.Context, workflow *ent.ApprovalWorkflow) *dto.ApprovalWorkflowResponse {
	nodes := make([]dto.ApprovalNodeResponse, 0)
	if workflow.Nodes != nil {
		for i, nodeMap := range workflow.Nodes {
			level := 0
			if l, ok := nodeMap["level"].(float64); ok {
				level = int(l)
			}
			node := dto.ApprovalNodeResponse{
				ID:            fmt.Sprintf("node%d", i+1),
				Level:         level,
				Name:          getStringValue(nodeMap["name"]),
				ApproverType:  getStringValue(nodeMap["approver_type"]),
				AssigneeType:  getStringValue(nodeMap["assignee_type"]),
				AssigneeValue: getStringValue(nodeMap["assignee_value"]),
				ApprovalMode:  getStringValue(nodeMap["approval_mode"]),
				AllowReject:   getBoolValue(nodeMap["allow_reject"]),
				AllowDelegate: getBoolValue(nodeMap["allow_delegate"]),
				RejectAction:  getStringValue(nodeMap["reject_action"]),
			}

			switch approverIDs := nodeMap["approver_ids"].(type) {
			case []interface{}:
				node.ApproverIDs = make([]int, 0, len(approverIDs))
				for _, id := range approverIDs {
					if idVal, ok := id.(float64); ok {
						node.ApproverIDs = append(node.ApproverIDs, int(idVal))
					} else if idVal, ok := id.(int); ok {
						node.ApproverIDs = append(node.ApproverIDs, idVal)
					}
				}
			case []int:
				node.ApproverIDs = append(node.ApproverIDs, approverIDs...)
			}

			if minApprovals, ok := nodeMap["minimum_approvals"].(float64); ok {
				min := int(minApprovals)
				node.MinimumApprovals = &min
			}
			if timeoutHours, ok := nodeMap["timeout_hours"].(float64); ok {
				timeout := int(timeoutHours)
				node.TimeoutHours = &timeout
			}
			if returnToLevel, ok := nodeMap["return_to_level"].(float64); ok {
				level := int(returnToLevel)
				node.ReturnToLevel = &level
			}

			if conditions, ok := nodeMap["conditions"].([]interface{}); ok {
				node.Conditions = make([]dto.ApprovalConditionConfig, 0, len(conditions))
				for _, cond := range conditions {
					if condMap, ok := cond.(map[string]interface{}); ok {
						node.Conditions = append(node.Conditions, dto.ApprovalConditionConfig{
							Field:    condMap["field"].(string),
							Operator: condMap["operator"].(string),
							Value:    condMap["value"],
						})
					}
				}
			}

			// 获取审批人姓名
			node.ApproverNames = make([]string, len(node.ApproverIDs))
			for i, id := range node.ApproverIDs {
				userEntity, err := s.client.User.Query().
					Where(
						user.IDEQ(id),
						user.TenantIDEQ(workflow.TenantID),
					).
					Only(ctx)
				if err != nil {
					node.ApproverNames[i] = fmt.Sprintf("用户%d", id)
					continue
				}
				node.ApproverNames[i] = userEntity.Name
			}

			nodes = append(nodes, node)
		}
	}

	response := &dto.ApprovalWorkflowResponse{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Nodes:       nodes,
		IsActive:    workflow.IsActive,
		TenantID:    workflow.TenantID,
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
	}

	if workflow.TicketType != "" {
		response.TicketType = &workflow.TicketType
	}
	if workflow.Priority != "" {
		response.Priority = &workflow.Priority
	}

	return response
}

func (s *ApprovalService) toRecordResponse(record *ent.ApprovalRecord) *dto.ApprovalRecordResponse {
	response := &dto.ApprovalRecordResponse{
		ID:           record.ID,
		TicketID:     record.TicketID,
		TicketNumber: record.TicketNumber,
		TicketTitle:  record.TicketTitle,
		WorkflowID:   record.WorkflowID,
		WorkflowName: record.WorkflowName,
		CurrentLevel: record.CurrentLevel,
		TotalLevels:  record.TotalLevels,
		ApproverID:   record.ApproverID,
		ApproverName: record.ApproverName,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt,
	}

	if record.Action != "" {
		response.Action = &record.Action
	}
	if record.Comment != "" {
		response.Comment = &record.Comment
	}
	if !record.ProcessedAt.IsZero() {
		response.ProcessedAt = &record.ProcessedAt
	}

	return response
}

// ApprovalTriggerRequest 审批触发请求
type ApprovalTriggerRequest struct {
	TicketID     int
	TicketNumber string
	TicketTitle  string
	TicketType   string // incident, change, service_request, ticket
	Priority     string
	RequesterID  int
	Amount       float64
	TenantID     int
}

// TriggerApproval 触发审批流程
func (s *ApprovalService) TriggerApproval(ctx context.Context, req *ApprovalTriggerRequest) ([]*ent.ApprovalRecord, error) {
	s.logger.Infow("Triggering approval", "ticket_number", req.TicketNumber, "ticket_type", req.TicketType, "priority", req.Priority)

	// 查找匹配的审批工作流
	workflow, err := s.findMatchingWorkflow(ctx, req.TicketType, req.Priority, req.TenantID)
	if err != nil {
		s.logger.Warnw("Error finding matching workflow", "error", err)
		return nil, nil
	}

	if workflow == nil {
		s.logger.Info("No active approval workflow found, skipping approval")
		return nil, nil
	}

	existing, err := s.client.ApprovalRecord.Query().
		Where(
			approvalrecord.TicketIDEQ(req.TicketID),
			approvalrecord.WorkflowIDEQ(workflow.ID),
			approvalrecord.TenantIDEQ(req.TenantID),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing approval records: %w", err)
	}
	if existing > 0 {
		return nil, nil
	}

	// 解析工作流节点
	nodes, err := s.parseWorkflowNodes(workflow.Nodes)
	if err != nil {
		s.logger.Errorw("Failed to parse workflow nodes", "error", err)
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	if len(nodes) == 0 {
		s.logger.Warnw("Workflow has no nodes", "workflow_id", workflow.ID)
		return nil, nil
	}

	// 创建审批记录
	records := make([]*ent.ApprovalRecord, 0)
	for i, node := range nodes {
		level := node.Level
		if level < 1 {
			level = i + 1
		}

		// 计算截止时间
		dueDate := time.Now()
		if node.TimeoutHours > 0 {
			dueDate = dueDate.Add(time.Duration(node.TimeoutHours) * time.Hour)
		}

		approverIDs := node.ApproverIDs
		if len(approverIDs) == 0 && node.AssigneeType != "" && node.AssigneeValue != "" {
			approverID, _, err := s.resolveApprover(ctx, node.AssigneeType, node.AssigneeValue, req.TenantID, req.Amount)
			if err != nil {
				s.logger.Warnw("Failed to resolve approver", "error", err, "node", i)
				continue
			}
			approverIDs = []int{approverID}
		}

		if len(approverIDs) == 0 {
			continue
		}

		if node.ApprovalMode != "all" {
			approverIDs = approverIDs[:1]
		}

		for _, approverID := range approverIDs {
			userEntity, err := s.client.User.Query().
				Where(
					user.IDEQ(approverID),
					user.TenantIDEQ(req.TenantID),
				).
				Only(ctx)
			if err != nil {
				continue
			}

			record, err := s.client.ApprovalRecord.Create().
				SetTicketNumber(req.TicketNumber).
				SetTicketTitle(req.TicketTitle).
				SetWorkflowName(workflow.Name).
				SetCurrentLevel(level).
				SetTotalLevels(len(nodes)).
				SetApproverID(approverID).
				SetApproverName(userEntity.Name).
				SetStatus("pending").
				SetWorkflowID(workflow.ID).
				SetTicketID(req.TicketID).
				SetStepOrder(level).
				SetDueDate(dueDate).
				SetTenantID(req.TenantID).
				SetCreatedAt(time.Now()).
				Save(ctx)
			if err != nil {
				s.logger.Errorw("Failed to create approval record", "error", err, "node", i)
				continue
			}

			records = append(records, record)
			s.logger.Infow("Created approval record", "record_id", record.ID, "approver", userEntity.Name, "level", level)
		}
	}

	return records, nil
}

// findMatchingWorkflow 查找匹配的审批工作流
func (s *ApprovalService) findMatchingWorkflow(ctx context.Context, ticketType, priority string, tenantID int) (*ent.ApprovalWorkflow, error) {
	// 先尝试精确匹配（类型+优先级）
	workflow, err := s.client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.TenantIDEQ(tenantID),
			approvalworkflow.IsActiveEQ(true),
			approvalworkflow.TicketTypeEQ(ticketType),
			approvalworkflow.PriorityEQ(priority),
		).
		First(ctx)

	if err == nil && workflow != nil {
		return workflow, nil
	}

	// 尝试匹配类型（不带优先级）
	workflow, err = s.client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.TenantIDEQ(tenantID),
			approvalworkflow.IsActiveEQ(true),
			approvalworkflow.TicketTypeEQ(ticketType),
		).
		First(ctx)

	if err == nil && workflow != nil {
		return workflow, nil
	}

	// 没有找到匹配的审批工作流
	return nil, nil
}

// workflowNode 审批节点
type workflowNode struct {
	Level         int
	Name          string
	ApproverIDs   []int
	ApprovalMode  string
	AssigneeType  string
	AssigneeValue string
	TimeoutHours  int
}

// parseWorkflowNodes 解析工作流节点JSON
func (s *ApprovalService) parseWorkflowNodes(nodesJSON interface{}) ([]workflowNode, error) {
	if nodesJSON == nil {
		return nil, nil
	}

	var nodesArray []map[string]interface{}
	switch v := nodesJSON.(type) {
	case []map[string]interface{}:
		nodesArray = v
	case []interface{}:
		nodesArray = make([]map[string]interface{}, 0, len(v))
		for _, raw := range v {
			if m, ok := raw.(map[string]interface{}); ok {
				nodesArray = append(nodesArray, m)
			}
		}
	default:
		return nil, fmt.Errorf("invalid nodes format")
	}

	nodes := make([]workflowNode, 0, len(nodesArray))
	for i, nodeMap := range nodesArray {
		node := workflowNode{
			Level:        i + 1,
			Name:         fmt.Sprintf("Step %d", i+1),
			ApprovalMode: "any",
		}

		if level, ok := nodeMap["level"].(float64); ok {
			node.Level = int(level)
		} else if level, ok := nodeMap["level"].(int); ok {
			node.Level = level
		}
		if name, ok := nodeMap["name"].(string); ok {
			node.Name = name
		}
		if approvalMode, ok := nodeMap["approval_mode"].(string); ok && approvalMode != "" {
			node.ApprovalMode = approvalMode
		}
		if timeout, ok := nodeMap["timeout_hours"].(float64); ok {
			node.TimeoutHours = int(timeout)
		} else if timeout, ok := nodeMap["timeout_hours"].(int); ok {
			node.TimeoutHours = timeout
		}

		switch ids := nodeMap["approver_ids"].(type) {
		case []interface{}:
			node.ApproverIDs = make([]int, 0, len(ids))
			for _, id := range ids {
				if idVal, ok := id.(float64); ok {
					node.ApproverIDs = append(node.ApproverIDs, int(idVal))
				} else if idVal, ok := id.(int); ok {
					node.ApproverIDs = append(node.ApproverIDs, idVal)
				}
			}
		case []int:
			node.ApproverIDs = append(node.ApproverIDs, ids...)
		}

		if assigneeType, ok := nodeMap["assignee_type"].(string); ok {
			node.AssigneeType = assigneeType
		}
		if assigneeValue, ok := nodeMap["assignee_value"].(string); ok {
			node.AssigneeValue = assigneeValue
		}
		if node.AssigneeType == "" {
			if approverType, ok := nodeMap["approver_type"].(string); ok {
				switch approverType {
				case "dept_manager", "team_leader", "project_manager", "temp_team_leader", "amount_based":
					node.AssigneeType = approverType
				}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// resolveApprover 解析审批人
func (s *ApprovalService) resolveApprover(ctx context.Context, assigneeType, assigneeValue string, tenantID int, amount float64) (int, string, error) {
	switch assigneeType {
	case "role":
		// 根据角色查找用户
		user, err := s.client.User.Query().
			Where(user.RoleEQ(user.Role(assigneeValue)), user.TenantID(tenantID), user.Active(true)).
			First(ctx)
		if err != nil || user == nil {
			// 如果没找到，返回错误而不是回退到用户ID 1
			return 0, "", fmt.Errorf("未找到具有角色 '%s' 的有效用户", assigneeValue)
		}
		return user.ID, user.Name, nil
	case "user":
		// 根据用户ID查找
		userID, err := strconv.Atoi(assigneeValue)
		if err != nil {
			return 0, "", fmt.Errorf("无效的用户ID: %s", assigneeValue)
		}
		user, err := s.client.User.Query().
			Where(user.ID(userID), user.TenantID(tenantID)).
			Only(ctx)
		if err != nil || user == nil {
			return 0, "", fmt.Errorf("未找到用户ID: %d", userID)
		}
		return user.ID, user.Name, nil
	case "dept_manager", "team_leader", "project_manager", "temp_team_leader":
		scopeID, err := strconv.Atoi(assigneeValue)
		if err != nil {
			return 0, "", fmt.Errorf("无效的审批人范围ID: %s", assigneeValue)
		}

		appCtx := &approver.ApproverContext{TenantID: tenantID}
		switch assigneeType {
		case "dept_manager":
			appCtx.DepartmentID = scopeID
		case "team_leader":
			appCtx.TeamID = scopeID
		case "project_manager":
			appCtx.ProjectID = scopeID
		case "temp_team_leader":
			appCtx.TeamID = scopeID
		}

		registry := approver.NewResolverRegistry(s.logger)
		registry.Register(approver.NewDeptManagerResolver())
		registry.Register(approver.NewTeamLeaderResolver())
		registry.Register(approver.NewProjectMgrResolver())
		registry.Register(approver.NewTempTeamResolver())

		approvers, err := registry.Resolve(ctx, s.client, assigneeType, appCtx)
		if err != nil {
			return 0, "", err
		}
		if len(approvers) == 0 {
			return 0, "", fmt.Errorf("未解析到审批人: %s/%s", assigneeType, assigneeValue)
		}
		return approvers[0].UserID, approvers[0].UserName, nil
	case "amount_based":
		thresholds, err := parseAmountThresholds(assigneeValue)
		if err != nil {
			return 0, "", err
		}
		registry := approver.NewResolverRegistry(s.logger)
		registry.Register(approver.NewAmountResolver(thresholds))
		approvers, err := registry.Resolve(ctx, s.client, "amount_based", &approver.ApproverContext{
			TenantID: tenantID,
			Amount:   amount,
		})
		if err != nil {
			return 0, "", fmt.Errorf("amount_based approver resolution failed: %w", err)
		}
		if len(approvers) == 0 {
			return 0, "", fmt.Errorf("amount_based resolved no approvers")
		}
		return approvers[0].UserID, approvers[0].UserName, nil
	default:
		return 0, "", fmt.Errorf("不支持的审批人类型: %s", assigneeType)
	}
}

func parseAmountThresholds(raw string) ([]approver.AmountThreshold, error) {
	if raw == "" {
		return nil, fmt.Errorf("amount_based requires assignee_value thresholds")
	}

	parts := strings.Split(raw, ",")
	thresholds := make([]approver.AmountThreshold, 0, len(parts))
	for _, part := range parts {
		rangeAndRole := strings.Split(strings.TrimSpace(part), ":")
		if len(rangeAndRole) != 2 {
			return nil, fmt.Errorf("invalid amount threshold %q", part)
		}
		bounds := strings.Split(rangeAndRole[0], "-")
		if len(bounds) != 2 {
			return nil, fmt.Errorf("invalid amount range %q", rangeAndRole[0])
		}
		minAmount, err := strconv.ParseFloat(bounds[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min amount %q: %w", bounds[0], err)
		}
		maxAmount, err := strconv.ParseFloat(bounds[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max amount %q: %w", bounds[1], err)
		}
		role := strings.TrimSpace(rangeAndRole[1])
		if role == "" {
			return nil, fmt.Errorf("amount threshold role cannot be empty")
		}
		thresholds = append(thresholds, approver.AmountThreshold{
			MinAmount: minAmount,
			MaxAmount: maxAmount,
			Role:      role,
		})
	}
	return thresholds, nil
}

// 辅助函数：安全获取字符串值
func getStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// 辅助函数：安全获取布尔值
func getBoolValue(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
