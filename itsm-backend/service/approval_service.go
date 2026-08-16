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
	"itsm-backend/service/bpmn"

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

func (s *ApprovalService) MigrateWorkflowToBPMN(ctx context.Context, workflowID, tenantID int, dryRun bool) (*LegacyApprovalMigrationResult, error) {
	workflow, err := s.client.ApprovalWorkflow.Query().
		Where(approvalworkflow.ID(workflowID), approvalworkflow.TenantID(tenantID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("legacy approval workflow not found: %w", err)
	}
	return NewLegacyApprovalMigrationService(s.client).Migrate(ctx, workflow, dryRun)
}

// CreateWorkflow 创建审批工作流
func (s *ApprovalService) CreateWorkflow(ctx context.Context, req *dto.CreateApprovalWorkflowRequest, tenantID int) (*dto.ApprovalWorkflowResponse, error) {
	s.logger.Infow("Creating approval workflow", "name", req.Name, "tenant_id", tenantID)

	// 强类型转换：ApprovalNodeRequest -> ApprovalNodeConfig -> map (Ent存储)
	configs := dto.NodesToConfigs(req.Nodes)
	nodesMap, err := nodesToMaps(configs)
	if err != nil {
		s.logger.Errorw("Failed to convert nodes to maps", "error", err)
		return nil, fmt.Errorf("failed to convert approval nodes: %w", err)
	}

	create := s.client.ApprovalWorkflow.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetNodes(nodesMap).
		SetIsActive(req.IsActive).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if req.TicketType != nil {
		create = create.SetTicketType(*req.TicketType)
	}
	if req.Priority != nil {
		create = create.SetPriority(*req.Priority)
	}

	workflow, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create approval workflow", "error", err)
		return nil, fmt.Errorf("failed to create approval workflow: %w", err)
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
		// 强类型转换：ApprovalNodeRequest -> ApprovalNodeConfig -> map (Ent存储)
		configs := dto.NodesToConfigs(*req.Nodes)
		nodesMap, err := nodesToMaps(configs)
		if err != nil {
			s.logger.Errorw("Failed to convert nodes to maps", "error", err)
			return nil, fmt.Errorf("failed to convert approval nodes: %w", err)
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
func (s *ApprovalService) ListWorkflows(ctx context.Context, filter *dto.WorkflowListFilter, tenantID int, page, pageSize int) ([]*dto.ApprovalWorkflowResponse, int, error) {
	s.logger.Infow("Listing approval workflows", "tenant_id", tenantID)

	query := s.client.ApprovalWorkflow.Query().
		Where(approvalworkflow.TenantIDEQ(tenantID))

	if filter != nil {
		if filter.TicketType != "" {
			query = query.Where(approvalworkflow.TicketTypeEQ(filter.TicketType))
		}
		if filter.Priority != "" {
			query = query.Where(approvalworkflow.PriorityEQ(filter.Priority))
		}
		if filter.IsActive != nil {
			query = query.Where(approvalworkflow.IsActiveEQ(*filter.IsActive))
		}
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

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start approval transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.submitApproval(ctx, tx.Client(), recordID, userID, action, comment, delegateToUserID, tenantID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit approval transaction: %w", err)
	}

	// ③ 同步驱动 BPMN 审批任务：若工单已启动审批流程实例，则经桥接完成对应 BPMN 任务，
	// 保证流程实例与审批记录状态一致。桥接失败（如当前用户非该任务候选）仅告警，不回滚记录。
	if action == "approve" || action == "reject" {
		if rec, rerr := s.client.ApprovalRecord.Get(ctx, recordID); rerr == nil && rec != nil {
			bridge := NewBPMNApprovalBridge(s.client, s.logger)
			if _, berr := bridge.CompleteBusinessApprovalTask(ctx, tenantID, userID, "approval", rec.TicketID, action, comment); berr != nil {
				s.logger.Warnw("同步BPMN审批任务失败（审批记录已更新）", "error", berr, "record_id", recordID, "ticket_id", rec.TicketID)
			}
		}
	}

	return nil
}

func (s *ApprovalService) submitApproval(ctx context.Context, client *ent.Client, recordID int, userID int, action string, comment string, delegateToUserID *int, tenantID int) error {
	// 先校验 action 合法性，确保"invalid action"错误能在 workflow 检查之前准确返回
	switch action {
	case "approve", "reject", "delegate":
		// 合法
	default:
		return fmt.Errorf("invalid action: %s", action)
	}

	// 获取审批记录
	approvalRecord, err := client.ApprovalRecord.Query().
		Where(
			approvalrecord.IDEQ(recordID),
			approvalrecord.TenantIDEQ(tenantID),
			approvalrecord.StatusEQ("pending"),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("approval record not found or already processed: %w", err)
	}

	// 顺序审批校验（H1 修复）：仅允许审批当前最低待审批级别，防止越级审批
	minLevel, err := s.minPendingLevel(ctx, client, approvalRecord.WorkflowID, approvalRecord.TicketID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to determine approval order: %w", err)
	}
	if approvalRecord.CurrentLevel > minLevel {
		s.logger.Warnw("Approval out of order", "record_level", approvalRecord.CurrentLevel, "min_pending_level", minLevel)
		return fmt.Errorf("请先完成前序级别（第 %d 级）的审批", minLevel)
	}

	// 权限检查：验证用户是否是该审批记录的指定审批人
	if approvalRecord.ApproverID != userID {
		s.logger.Warnw("User is not the assigned approver", "user_id", userID, "approver_id", approvalRecord.ApproverID, "record_id", recordID)
		return fmt.Errorf("user is not authorized to approve this record")
	}

	// 获取审批工作流以检查操作权限
	workflow, err := client.ApprovalWorkflow.Query().
		Where(
			approvalworkflow.IDEQ(approvalRecord.WorkflowID),
			approvalworkflow.TenantIDEQ(tenantID),
		).
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

	// T-2 修复：加乐观锁。仅当记录仍处于 pending 才允许更新，
	// 使并发提交的第二条事务因命中 0 行而失败，杜绝重复审批/重复委派。
	update := client.ApprovalRecord.UpdateOneID(recordID).
		Where(approvalrecord.StatusEQ("pending")).
		SetStatus(newStatus).
		SetAction(action).
		SetProcessedAt(time.Now())

	if comment != "" {
		update = update.SetComment(comment)
	}

	_, err = update.Save(ctx)
	if err != nil {
		// 并发场景下另一条事务已抢先处理（状态不再是 pending），
		// Ent 的 UpdateOne 命中 0 行会返回 not-found，这里转为明确业务错误。
		if ent.IsNotFound(err) {
			s.logger.Warnw("Approval record already processed by another request (race lost)", "record_id", recordID)
			return fmt.Errorf("审批记录已被其他请求处理，请刷新后重试")
		}
		s.logger.Errorw("Failed to update approval record", "error", err)
		return fmt.Errorf("failed to update approval record: %w", err)
	}

	// 处理审批后的逻辑
	switch action {
	case "approve":
		if err := s.handleApprovalApproved(ctx, client, approvalRecord); err != nil {
			s.logger.Errorw("Failed to handle approved action", "error", err)
			return fmt.Errorf("failed to handle approved action: %w", err)
		}
	case "reject":
		rejectAction := "terminate" // 默认拒绝动作，可以从参数中获取
		if err := s.handleApprovalRejected(ctx, client, approvalRecord, rejectAction); err != nil {
			s.logger.Errorw("Failed to handle rejected action", "error", err)
			return fmt.Errorf("failed to handle rejected action: %w", err)
		}
	case "delegate":
		if delegateToUserID != nil {
			if err := s.handleApprovalDelegated(ctx, client, approvalRecord, *delegateToUserID); err != nil {
				s.logger.Errorw("Failed to handle delegated action", "error", err)
				return fmt.Errorf("failed to handle delegated action: %w", err)
			}
		} else {
			return fmt.Errorf("delegate_to_user_id is required for delegation")
		}
	}

	return nil
}

// minPendingLevel 返回该工单在当前工作流下的最小待审批级别
func (s *ApprovalService) minPendingLevel(ctx context.Context, client *ent.Client, workflowID, ticketID, tenantID int) (int, error) {
	records, err := client.ApprovalRecord.Query().
		Where(
			approvalrecord.WorkflowIDEQ(workflowID),
			approvalrecord.TicketIDEQ(ticketID),
			approvalrecord.TenantIDEQ(tenantID),
			approvalrecord.StatusEQ("pending"),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	min := records[0].CurrentLevel
	for _, r := range records {
		if r.CurrentLevel < min {
			min = r.CurrentLevel
		}
	}
	return min, nil
}

// handleApprovalApproved 处理审批通过
func (s *ApprovalService) handleApprovalApproved(ctx context.Context, client *ent.Client, record *ent.ApprovalRecord) error {
	// 检查该工单在该工作流下是否还有待审批项
	remainingApprovals, err := client.ApprovalRecord.Query().
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
		_, err := client.Ticket.UpdateOneID(record.TicketID).
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
func (s *ApprovalService) handleApprovalRejected(ctx context.Context, client *ent.Client, record *ent.ApprovalRecord, rejectAction string) error {
	// 根据拒绝动作处理
	switch rejectAction {
	case "terminate":
		// 取消同一工单、同一工作流中的其他待审批项
		_, err := client.ApprovalRecord.Update().
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

		_, err = client.Ticket.UpdateOneID(record.TicketID).
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
		return s.handleApprovalRejected(ctx, client, record, "terminate")
	}

	return nil
}

// canPerformAction 检查在指定审批级别是否允许执行该操作
func (s *ApprovalService) canPerformAction(workflow *ent.ApprovalWorkflow, level int, action string) bool {
	configs := mapsToNodesUnsafe(workflow.Nodes)
	if len(configs) == 0 {
		// 如果没有节点配置，默认拒绝所有操作（H2 修复：缺配置不再默认放行）
		return false
	}

	// 查找当前级别的节点配置（缺失 level 时回退索引+1，与 parseWorkflowNodes 一致）
	for i, node := range configs {
		nodeLevel := node.Level
		if nodeLevel < 1 {
			nodeLevel = i + 1
		}
		if nodeLevel == level {
			switch dto.ApprovalAction(action) {
			case dto.ApprovalActionApprove:
				return true // 审批通过总是允许
			case dto.ApprovalActionReject:
				return node.AllowReject
			case dto.ApprovalActionDelegate:
				return node.AllowDelegate
			}
		}
	}

	return false // 未找到对应级别节点时默认拒绝（H2 修复）
}

// handleApprovalDelegated 处理审批委托
func (s *ApprovalService) handleApprovalDelegated(ctx context.Context, client *ent.Client, record *ent.ApprovalRecord, delegateTo int) error {
	delegateUser, err := client.User.Query().
		Where(
			user.IDEQ(delegateTo),
			user.TenantIDEQ(record.TenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get delegate user: %w", err)
	}

	// 创建新的审批记录给被委托人
	_, err = client.ApprovalRecord.Create().
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
	// 强类型反序列化：map -> ApprovalNodeConfig
	configs, err := mapsToNodes(workflow.Nodes)
	if err != nil {
		s.logger.Errorw("Failed to parse workflow nodes", "error", err, "workflow_id", workflow.ID)
		configs = []dto.ApprovalNodeConfig{}
	}

	// 使用强类型转换生成响应节点
	nodes := dto.ConfigsToResponses(configs)

	// 获取审批人姓名（批量查询优化）
	for i := range nodes {
		node := &nodes[i]
		if len(node.ApproverIDs) == 0 {
			node.ApproverNames = []string{}
			continue
		}
		node.ApproverNames = make([]string, len(node.ApproverIDs))
		for j, id := range node.ApproverIDs {
			userEntity, err := s.client.User.Query().
				Where(
					user.IDEQ(id),
					user.TenantIDEQ(workflow.TenantID),
				).
				Only(ctx)
			if err != nil {
				node.ApproverNames[j] = fmt.Sprintf("用户%d", id)
				continue
			}
			node.ApproverNames[j] = userEntity.Name
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

	// ②/③ 解析审批的工单类型对应的流程：BPMN ProcessBinding 优先，未迁移类型回退旧 ApprovalWorkflow。
	workflow, defKey, err := s.resolveApprovalWorkflow(ctx, req.TicketType, req.Priority, req.TenantID)
	if err != nil {
		s.logger.Warnw("Error resolving approval workflow", "error", err)
		return nil, nil
	}

	if workflow == nil {
		s.logger.Info("No active approval workflow found, skipping approval")
		return nil, nil
	}

	// ③ 解析到 BPMN 审批流程：启动流程实例，由引擎真实求值（候选组→审批人）并生成待办任务。
	// 失败仅告警，不阻塞工单创建。
	if defKey != "" {
		s.startApprovalProcess(ctx, req, defKey)
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

// resolveApprovalWorkflow 解析工单类型对应的审批工作流。
// ② 改为优先通过 BPMN ProcessBinding(business_type="approval", subType=工单类型) 解析迁移后的
// legacy_approval_* 流程定义；未迁移的工单类型回退旧 ApprovalWorkflow 查询以兼容历史数据。
// 返回值：(用于生成审批记录的原始 ApprovalWorkflow, BPMN 流程定义 Key 或空串)。
func (s *ApprovalService) resolveApprovalWorkflow(ctx context.Context, ticketType, priority string, tenantID int) (*ent.ApprovalWorkflow, string, error) {
	bindingSvc := NewProcessBindingService(s.client)
	binding, err := bindingSvc.FindBestBinding(ctx, dto.BusinessType("approval"), ticketType, tenantID)
	if err == nil && binding != nil {
		if wid := parseLegacyApprovalWorkflowID(binding.ProcessDefinitionKey); wid > 0 {
			if wf, werr := s.client.ApprovalWorkflow.Get(ctx, wid); werr == nil {
				return wf, binding.ProcessDefinitionKey, nil
			}
		}
	}
	// 兼容回退：未迁移类型仍按旧 ApprovalWorkflow 查询。
	wf, ferr := s.findMatchingWorkflow(ctx, ticketType, priority, tenantID)
	if ferr != nil {
		return nil, "", ferr
	}
	return wf, "", nil
}

// startApprovalProcess 启动审批 BPMN 流程实例（业务键 approval:<ticketID>），
// 由 BPMN 引擎真实求值候选组并生成待办任务。失败仅告警，不阻塞工单创建。
func (s *ApprovalService) startApprovalProcess(ctx context.Context, req *ApprovalTriggerRequest, defKey string) {
	triggerCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, req.TenantID)
	engine := NewCustomProcessEngine(s.client, s.logger)
	businessKey := fmt.Sprintf("approval:%d", req.TicketID)
	variables := map[string]interface{}{
		"ticketId":     req.TicketID,
		"ticketNumber": req.TicketNumber,
		"ticketType":   req.TicketType,
		"priority":     req.Priority,
	}
	if _, err := engine.StartProcess(triggerCtx, defKey, businessKey, variables); err != nil {
		s.logger.Warnw("启动审批BPMN流程实例失败", "error", err, "ticket_id", req.TicketID, "def_key", defKey)
	}
}

// parseLegacyApprovalWorkflowID 从迁移流程定义 Key（legacy_approval_<id>）提取原始工作流 ID。
func parseLegacyApprovalWorkflowID(key string) int {
	const prefix = "legacy_approval_"
	if !strings.HasPrefix(key, prefix) {
		return 0
	}
	id, err := strconv.Atoi(strings.TrimPrefix(key, prefix))
	if err != nil {
		return 0
	}
	return id
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

// parseWorkflowNodes 解析工作流节点（强类型版本）
func (s *ApprovalService) parseWorkflowNodes(nodesJSON interface{}) ([]workflowNode, error) {
	if nodesJSON == nil {
		return nil, nil
	}

	// 将 interface{} 转为 []map[string]interface{}
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

	// 使用强类型转换器解析
	configs, err := mapsToNodes(nodesArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow nodes: %w", err)
	}

	nodes := make([]workflowNode, 0, len(configs))
	for i, config := range configs {
		node := workflowNode{
			Level:         config.Level,
			Name:          config.Name,
			ApproverIDs:   config.ApproverIDs,
			ApprovalMode:  string(config.ApprovalMode),
			AssigneeType:  config.AssigneeType,
			AssigneeValue: config.AssigneeValue,
		}
		if node.Level < 1 {
			node.Level = i + 1
		}
		if node.Name == "" {
			node.Name = fmt.Sprintf("Step %d", i+1)
		}
		if node.ApprovalMode == "" {
			node.ApprovalMode = "any"
		}
		if config.TimeoutHours != nil {
			node.TimeoutHours = *config.TimeoutHours
		}

		// 如果 AssigneeType 仍为空：尝试从原始节点 map 提取审批人规格，
		// 兼容 legacy snake_case 形状（approver_type / assignee_type / role 等），
		// 与迁移脚本 resolveLegacyApprovalAssignee 的 candidateGroups 语义保持一致。
		// 动态角色（如 manager）一律按候选组（group）解析，使 ApprovalRecord 镜像的
		// 审批人与 BPMN 引擎任务的候选用户一致；同时兼容 camelCase ApproverType 直接为 manager。
		if node.AssigneeType == "" {
			if at, av := extractApproverSpec(nodesArray[i]); at != "" {
				node.AssigneeType, node.AssigneeValue = at, av
			} else {
				switch config.ApproverType {
				case "manager":
					node.AssigneeType = "group"
					node.AssigneeValue = "manager"
				case dto.ApprovalNodeTypeDeptManager,
					dto.ApprovalNodeTypeTeamLeader,
					dto.ApprovalNodeTypeProjectManager,
					dto.ApprovalNodeTypeTempTeamLeader,
					dto.ApprovalNodeTypeAmountBased:
					node.AssigneeType = string(config.ApproverType)
				}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// extractApproverSpec 从原始节点 map 提取审批人规格（assigneeType/assigneeValue）。
// 兼容三种 legacy 形状：camelCase(assigneeType/assigneeValue)、snake_case(assignee_type/assignee_value)、
// 以及 seeder 形状(approver_type + role/user_id/group)。动态角色（如 manager/dept_manager）按候选组解析，
// 与 BPMN 引擎 resolveLegacyApprovalAssignee 的 candidateGroups 语义对齐，保证镜像审批人与引擎候选用户一致。
func extractApproverSpec(raw map[string]interface{}) (string, string) {
	if v, ok := raw["assigneeType"].(string); ok && v != "" {
		return v, fmt.Sprint(raw["assigneeValue"])
	}
	if v, ok := raw["assignee_type"].(string); ok && v != "" {
		return v, fmt.Sprint(raw["assignee_value"])
	}
	if at, ok := raw["approver_type"].(string); ok && at != "" {
		switch at {
		case "role":
			return "role", fmt.Sprint(raw["role"])
		case "user":
			return "user", fmt.Sprint(raw["user_id"])
		case "group":
			return "group", fmt.Sprint(raw["group"])
		default:
			// 动态角色（manager 等）按候选组名解析
			return "group", at
		}
	}
	return "", ""
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
	case "group":
		// 候选组解析，与 BPMN 引擎的 GroupResolver 一致，保证镜像审批人与引擎候选用户同源。
		// ApprovalRecord 单记录模型与 legacy role 解析一致：取组内首位候选人为该节点审批人。
		resolver := bpmn.NewGroupResolver(s.client)
		ids, _, err := resolver.ExpandGroupsToUsers(ctx, tenantID, assigneeValue)
		if err != nil {
			return 0, "", err
		}
		if len(ids) == 0 {
			return 0, "", fmt.Errorf("审批组 '%s' 未解析到成员", assigneeValue)
		}
		return ids[0], "", nil
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
