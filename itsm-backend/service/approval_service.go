package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/approvalrecord"

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
			"reject_action": node.RejectAction,
		}
		if node.MinimumApprovals != nil {
			nodeMap["minimum_approvals"] = *node.MinimumApprovals
		}
		if node.TimeoutHours != nil {
			nodeMap["timeout_hours"] = *node.TimeoutHours
		}
		if node.Conditions != nil && len(node.Conditions) > 0 {
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

	return s.toWorkflowResponse(workflow), nil
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
				"reject_action": node.RejectAction,
			}
			if node.MinimumApprovals != nil {
				nodeMap["minimum_approvals"] = *node.MinimumApprovals
			}
			if node.TimeoutHours != nil {
				nodeMap["timeout_hours"] = *node.TimeoutHours
			}
			if node.Conditions != nil && len(node.Conditions) > 0 {
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

	return s.toWorkflowResponse(workflow), nil
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
		responses[i] = s.toWorkflowResponse(workflow)
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

	return s.toWorkflowResponse(workflow), nil
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
func (s *ApprovalService) SubmitApproval(ctx context.Context, recordID int, action string, comment string, delegateToUserID *int, tenantID int) error {
	s.logger.Infow("Submitting approval", "record_id", recordID, "action", action, "tenant_id", tenantID)

	// 获取审批记录
	_, err := s.client.ApprovalRecord.Query().
		Where(
			approvalrecord.IDEQ(recordID),
			approvalrecord.TenantIDEQ(tenantID),
			approvalrecord.StatusEQ("pending"),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("approval record not found or already processed: %w", err)
	}

	// 更新审批记录
	update := s.client.ApprovalRecord.UpdateOneID(recordID).
		SetStatus(action).
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

	// TODO: 处理审批后的逻辑
	// - 如果批准，检查是否需要进入下一级
	// - 如果拒绝，根据reject_action处理
	// - 如果委托，创建新的审批记录

	return nil
}

// 辅助方法：转换为响应DTO
func (s *ApprovalService) toWorkflowResponse(workflow *ent.ApprovalWorkflow) *dto.ApprovalWorkflowResponse {
	nodes := make([]dto.ApprovalNodeResponse, 0)
	if workflow.Nodes != nil {
		for i, nodeMap := range workflow.Nodes {
			node := dto.ApprovalNodeResponse{
				ID:            fmt.Sprintf("node%d", i+1),
				Level:         int(nodeMap["level"].(float64)),
				Name:          nodeMap["name"].(string),
				ApproverType:  nodeMap["approver_type"].(string),
				ApprovalMode:  nodeMap["approval_mode"].(string),
				AllowReject:   nodeMap["allow_reject"].(bool),
				AllowDelegate: nodeMap["allow_delegate"].(bool),
				RejectAction:  nodeMap["reject_action"].(string),
			}

			if approverIDs, ok := nodeMap["approver_ids"].([]interface{}); ok {
				node.ApproverIDs = make([]int, 0, len(approverIDs))
				for _, id := range approverIDs {
					if idVal, ok := id.(float64); ok {
						node.ApproverIDs = append(node.ApproverIDs, int(idVal))
					} else if idVal, ok := id.(int); ok {
						node.ApproverIDs = append(node.ApproverIDs, idVal)
					}
				}
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

			// TODO: 获取审批人姓名
			node.ApproverNames = make([]string, len(node.ApproverIDs))
			for i, id := range node.ApproverIDs {
				node.ApproverNames[i] = fmt.Sprintf("用户%d", id) // 临时，需要查询用户表
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
		ID:            record.ID,
		TicketID:      record.TicketID,
		TicketNumber:  record.TicketNumber,
		TicketTitle:   record.TicketTitle,
		WorkflowID:    record.WorkflowID,
		WorkflowName:  record.WorkflowName,
		CurrentLevel:  record.CurrentLevel,
		TotalLevels:   record.TotalLevels,
		ApproverID:    record.ApproverID,
		ApproverName:  record.ApproverName,
		Status:        record.Status,
		CreatedAt:     record.CreatedAt,
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

