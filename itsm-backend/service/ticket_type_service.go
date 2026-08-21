package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/ticketassignmentrule"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/tickettype"

	"go.uber.org/zap"
)

type TicketTypeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

var ticketTypePresets = []dto.TicketTypePreset{
	{ID: "general-incident", Name: "通用事件", Description: "通用故障与中断上报", Category: "运维", Definition: dto.CreateTicketTypeRequest{Code: "general_incident", Name: "通用事件", DefaultPriority: "medium", Icon: "Bug", Color: "#ff4d4f", CustomFields: []dto.CustomFieldDefinition{{ID: "symptom", Name: "symptom", Label: "故障现象", Type: dto.CustomFieldTypeTextarea, Required: true, Order: 0}}}},
	{ID: "pacs-incident", Name: "PACS 故障", Description: "医疗影像系统故障模板", Category: "医疗", Definition: dto.CreateTicketTypeRequest{Code: "pacs_incident", Name: "PACS 故障", DefaultPriority: "high", Icon: "Monitor", Color: "#722ed1", CustomFields: []dto.CustomFieldDefinition{{ID: "node", Name: "pacsNode", Label: "PACS 节点", Type: dto.CustomFieldTypeText, Required: true, Order: 0}, {ID: "department", Name: "affectedDepartment", Label: "影响科室", Type: dto.CustomFieldTypeDepartment, Required: true, Order: 1}, {ID: "patients", Name: "affectedPatients", Label: "影响患者数量", Type: dto.CustomFieldTypeNumber, Order: 2}}}},
	{ID: "database-change", Name: "数据库变更", Description: "数据库 DDL/DML 变更申请", Category: "变更", Definition: dto.CreateTicketTypeRequest{Code: "database_change", Name: "数据库变更", DefaultPriority: "high", Icon: "Database", Color: "#fa8c16", CustomFields: []dto.CustomFieldDefinition{{ID: "sql", Name: "sql", Label: "SQL", Type: dto.CustomFieldTypeTextarea, Required: true, Order: 0}, {ID: "backup", Name: "backupConfirmed", Label: "已确认备份", Type: dto.CustomFieldTypeBoolean, Required: true, Order: 1}, {ID: "rollback", Name: "rollbackPlan", Label: "回滚方案", Type: dto.CustomFieldTypeTextarea, Required: true, Order: 2}}}},
	{ID: "account-request", Name: "账号权限申请", Description: "企业系统账号和权限申请", Category: "服务请求", Definition: dto.CreateTicketTypeRequest{Code: "account_request", Name: "账号权限申请", DefaultPriority: "medium", Icon: "User", Color: "#1677ff", CustomFields: []dto.CustomFieldDefinition{{ID: "target", Name: "targetUser", Label: "申请用户", Type: dto.CustomFieldTypeUser, Required: true, Order: 0}, {ID: "role", Name: "role", Label: "权限角色", Type: dto.CustomFieldTypeText, Required: true, Order: 1}, {ID: "expiry", Name: "expiry", Label: "有效期", Type: dto.CustomFieldTypeDate, Order: 2}}}},
}

func (s *TicketTypeService) ListPresets() []dto.TicketTypePreset { return ticketTypePresets }

func (s *TicketTypeService) InstallPreset(ctx context.Context, presetID string, req *dto.InstallTicketTypePresetRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	for _, preset := range ticketTypePresets {
		if preset.ID != presetID {
			continue
		}
		definition := preset.Definition
		if strings.TrimSpace(req.Code) != "" {
			definition.Code = strings.TrimSpace(req.Code)
		}
		if strings.TrimSpace(req.Name) != "" {
			definition.Name = strings.TrimSpace(req.Name)
		}
		tx, err := s.client.Tx(ctx)
		if err != nil {
			return nil, fmt.Errorf("开启预设安装事务失败: %w", err)
		}
		txService := &TicketTypeService{client: tx.Client(), logger: s.logger}
		created, err := txService.CreateTicketType(ctx, &definition, tenantID, userID)
		if err == nil {
			err = txService.recordTicketTypeAudit(ctx, tenantID, userID, created.ID, "preset.install", map[string]interface{}{"presetId": presetID, "code": created.Code, "name": created.Name})
		}
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("提交预设安装事务失败: %w", err)
		}
		return created, nil
	}
	return nil, fmt.Errorf("工单类型预设不存在")
}

func NewTicketTypeService(client *ent.Client, logger *zap.SugaredLogger) *TicketTypeService {
	return &TicketTypeService{
		client: client,
		logger: logger,
	}
}

// CreateTicketType creates a new ticket type using ent (supports transaction via ctx)
func (s *TicketTypeService) CreateTicketType(ctx context.Context, req *dto.CreateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	s.logger.Infow("Creating ticket type", "code", req.Code, "tenant_id", tenantID)
	req.Code, req.Name = strings.TrimSpace(req.Code), strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("工单类型编码和名称不能为空")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(req.Code) {
		return nil, fmt.Errorf("工单类型编码只能包含小写字母、数字和下划线")
	}
	if len(req.Code) > 50 || len(req.Name) > 100 {
		return nil, fmt.Errorf("工单类型编码或名称过长")
	}
	if !validTicketPriority(defaultPriority(req.DefaultPriority)) {
		return nil, fmt.Errorf("无效的默认优先级")
	}
	if req.SortOrder < 0 {
		return nil, fmt.Errorf("排序值不能为负数")
	}
	if req.ApprovalEnabled && len(req.ApprovalChain) == 0 {
		return nil, fmt.Errorf("启用审批时必须配置审批链")
	}
	if err := validateCustomFields(req.CustomFields); err != nil {
		return nil, err
	}

	// Check code uniqueness within tenant
	exists, err := s.checkCodeExists(ctx, req.Code, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check code existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("工单类型编码已存在: %s", req.Code)
	}

	// Serialize complex JSON fields
	customFieldsMap := toCustomFieldsMap(req.CustomFields)
	assignmentRulesMap := toAssignmentRulesMap(req.AssignmentRules)
	approvalChainSlice := toInterfaceSlice(req.ApprovalChain)
	notificationConfigMap := structToMap(req.NotificationConfig)
	permissionConfigMap := structToMap(req.PermissionConfig)

	// Build create mutation via ent client（schema 中 created_at/updated_at 为必填字段，必须显式设置）
	now := time.Now()
	create := s.client.TicketType.Create().
		SetCode(req.Code).
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetColor(req.Color).
		SetStatus(string(dto.TicketTypeStatusActive)).
		SetDefaultPriority(defaultPriority(req.DefaultPriority)).
		SetSortOrder(req.SortOrder).
		SetCustomFields(customFieldsMap).
		SetApprovalEnabled(req.ApprovalEnabled).
		SetApprovalChain(approvalChainSlice).
		SetSLAEnabled(req.SLAEnabled).
		SetAutoAssignEnabled(req.AutoAssignEnabled).
		SetAssignmentRules(assignmentRulesMap).
		SetNotificationConfig(notificationConfigMap).
		SetPermissionConfig(permissionConfigMap).
		SetTenantID(int64(tenantID)).
		SetCreatedBy(int64(userID)).
		SetCreatedAt(now).
		SetUpdatedAt(now)
	if req.DefaultSLAID != nil {
		create.SetDefaultSLAID(int64(*req.DefaultSLAID))
	}
	if req.CategoryID != nil {
		create.SetCategoryID(*req.CategoryID)
	}
	if req.WorkflowDefinitionKey != "" {
		create.SetWorkflowDefinitionKey(req.WorkflowDefinitionKey)
	}
	if req.AssignmentRuleID != nil {
		create.SetAssignmentRuleID(*req.AssignmentRuleID)
	}
	if err := s.validateBindings(ctx, tenantID, req.CategoryID, req.WorkflowDefinitionKey, req.DefaultSLAID, req.AssignmentRuleID); err != nil {
		return nil, err
	}
	obj, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket type", "error", err)
		return nil, fmt.Errorf("failed to create ticket type: %w", err)
	}

	return s.toDefinition(obj), nil
}

// UpdateTicketType updates an existing ticket type using ent
func (s *TicketTypeService) UpdateTicketType(ctx context.Context, id int, req *dto.UpdateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启工单类型更新事务失败: %w", err)
	}
	txService := &TicketTypeService{client: tx.Client(), logger: s.logger}
	updated, err := txService.updateTicketType(ctx, id, req, tenantID, userID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交工单类型更新事务失败: %w", err)
	}
	return updated, nil
}

func (s *TicketTypeService) updateTicketType(ctx context.Context, id int, req *dto.UpdateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	s.logger.Infow("Updating ticket type", "id", id, "tenant_id", tenantID)

	existing, err := s.client.TicketType.Get(ctx, int(id))
	if err != nil {
		return nil, fmt.Errorf("ticket type not found: %w", err)
	}
	if int(existing.TenantID) != tenantID {
		return nil, fmt.Errorf("ticket type not found")
	}

	// Check code uniqueness if being changed (code is immutable; skip for updates)
	_ = existing.Code // suppress unused warning

	// Build update mutation (code is immutable after creation)
	// Build update mutation using UpdateOne(existing) which supports chaining and returns *TicketType
	update := s.client.TicketType.UpdateOne(existing)
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, fmt.Errorf("工单类型名称不能为空")
		}
		update.SetName(*req.Name)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Icon != nil {
		update.SetIcon(*req.Icon)
	}
	if req.Color != nil {
		update.SetColor(*req.Color)
	}
	if req.Status != nil {
		if *req.Status != dto.TicketTypeStatusActive && *req.Status != dto.TicketTypeStatusInactive && *req.Status != dto.TicketTypeStatusDraft {
			return nil, fmt.Errorf("无效的工单类型状态: %s", *req.Status)
		}
		update.SetStatus(string(*req.Status))
	}
	if req.CategoryID != nil {
		update.SetCategoryID(*req.CategoryID)
	}
	if req.DefaultPriority != nil {
		if !validTicketPriority(*req.DefaultPriority) {
			return nil, fmt.Errorf("无效的默认优先级")
		}
		update.SetDefaultPriority(defaultPriority(*req.DefaultPriority))
	}
	if req.SortOrder != nil {
		if *req.SortOrder < 0 {
			return nil, fmt.Errorf("排序值不能为负数")
		}
		update.SetSortOrder(*req.SortOrder)
	}
	if req.WorkflowDefinitionKey != nil {
		update.SetWorkflowDefinitionKey(*req.WorkflowDefinitionKey)
	}
	if req.AssignmentRuleID != nil {
		update.SetAssignmentRuleID(*req.AssignmentRuleID)
	}
	if req.CustomFields != nil {
		if err := validateCustomFields(*req.CustomFields); err != nil {
			return nil, err
		}
		update.SetCustomFields(toCustomFieldsMap(*req.CustomFields))
	}
	if req.ApprovalEnabled != nil {
		update.SetApprovalEnabled(*req.ApprovalEnabled)
	}
	if req.ApprovalChain != nil {
		update.SetApprovalChain(toInterfaceSlice(*req.ApprovalChain))
	}
	approvalEnabled := existing.ApprovalEnabled
	if req.ApprovalEnabled != nil {
		approvalEnabled = *req.ApprovalEnabled
	}
	approvalChainEmpty := len(existing.ApprovalChain) == 0
	if req.ApprovalChain != nil {
		approvalChainEmpty = len(*req.ApprovalChain) == 0
	}
	if approvalEnabled && approvalChainEmpty {
		return nil, fmt.Errorf("启用审批时必须配置审批链")
	}
	if req.SLAEnabled != nil {
		update.SetSLAEnabled(*req.SLAEnabled)
	}
	if req.DefaultSLAID != nil {
		update.SetDefaultSLAID(int64(*req.DefaultSLAID))
	}
	workflowKey := existing.WorkflowDefinitionKey
	if req.WorkflowDefinitionKey != nil {
		workflowKey = *req.WorkflowDefinitionKey
	}
	categoryID := intPtrFromInt(existing.CategoryID)
	if req.CategoryID != nil {
		categoryID = req.CategoryID
	}
	slaID := intPtrFromInt64(existing.DefaultSLAID)
	if req.DefaultSLAID != nil {
		slaID = req.DefaultSLAID
	}
	assignmentRuleID := intPtrFromInt(existing.AssignmentRuleID)
	if req.AssignmentRuleID != nil {
		assignmentRuleID = req.AssignmentRuleID
	}
	if err := s.validateBindings(ctx, tenantID, categoryID, workflowKey, slaID, assignmentRuleID); err != nil {
		return nil, err
	}
	if req.AutoAssignEnabled != nil {
		update.SetAutoAssignEnabled(*req.AutoAssignEnabled)
	}
	if req.AssignmentRules != nil {
		update.SetAssignmentRules(toAssignmentRulesMap(*req.AssignmentRules))
	}
	if req.NotificationConfig != nil {
		update.SetNotificationConfig(structToMap(req.NotificationConfig))
	}
	if req.PermissionConfig != nil {
		update.SetPermissionConfig(structToMap(req.PermissionConfig))
	}
	update.SetUpdatedBy(int64(userID))
	update.SetUpdatedAt(time.Now())

	updatedObj, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket type", "error", err)
		return nil, fmt.Errorf("failed to update ticket type: %w", err)
	}
	if bindingChanges := ticketTypeBindingChanges(existing, updatedObj); len(bindingChanges) > 0 {
		if err := s.recordTicketTypeAudit(ctx, tenantID, userID, updatedObj.ID, "binding.update", bindingChanges); err != nil {
			return nil, err
		}
	}

	return s.toDefinition(updatedObj), nil
}

// GetTicketType retrieves a single ticket type by ID
func (s *TicketTypeService) GetTicketType(ctx context.Context, id, tenantID int) (*dto.TicketTypeDefinition, error) {
	obj, err := s.client.TicketType.Get(ctx, int(id))
	if err != nil {
		return nil, fmt.Errorf("ticket type not found: %w", err)
	}
	if int(obj.TenantID) != tenantID {
		return nil, fmt.Errorf("ticket type not found")
	}
	return s.toDefinition(obj), nil
}

// ListTicketTypes returns paginated ticket types for a tenant
func (s *TicketTypeService) ListTicketTypes(ctx context.Context, req *dto.ListTicketTypesRequest, tenantID int) (*dto.TicketTypeListResponse, error) {
	predicates := []predicate.TicketType{
		tickettype.TenantID(int64(tenantID)),
		// 默认隐藏已归档项；当显式按状态查询时（如 status=inactive 查全部停用项）不额外过滤，
		// 因为 archived_at 是归档动作的内部标记，停用/草稿等状态仍应可查。
		tickettype.ArchivedAtIsNil(),
	}
	if req.Status != nil && *req.Status != "" {
		predicates = append(predicates, tickettype.Status(string(*req.Status)))
	}
	if req.Keyword != "" {
		predicates = append(predicates, tickettype.Or(
			tickettype.NameContains(req.Keyword),
			tickettype.CodeContains(req.Keyword),
		))
	}

	query := s.client.TicketType.Query().
		Where(predicates...).
		Order(ent.Asc(tickettype.FieldSortOrder), ent.Desc(tickettype.FieldUpdatedAt))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count ticket types: %w", err)
	}

	pageSize := 20
	if req.PageSize > 0 && req.PageSize <= 100 {
		pageSize = req.PageSize
	}
	offset := 0
	if req.Page > 0 {
		offset = (req.Page - 1) * pageSize
	}

	items, err := query.
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list ticket types: %w", err)
	}

	types := make([]dto.TicketTypeDefinition, len(items))
	for i, item := range items {
		types[i] = *s.toDefinition(item)
	}

	return &dto.TicketTypeListResponse{
		Types:      types,
		Total:      int64(total),
		Page:       req.Page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// DeleteTicketType archives a ticket type; physical deletion is intentionally forbidden.
func (s *TicketTypeService) DeleteTicketType(ctx context.Context, id, tenantID, userID int) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启归档事务失败: %w", err)
	}
	txc := tx.Client()
	existing, err := txc.TicketType.Get(ctx, int(id))
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("ticket type not found: %w", err)
	}
	if int(existing.TenantID) != tenantID {
		_ = tx.Rollback()
		return fmt.Errorf("ticket type not found")
	}

	_, err = txc.TicketType.UpdateOne(existing).SetStatus(string(dto.TicketTypeStatusInactive)).SetArchivedAt(time.Now()).SetArchivedBy(int64(userID)).SetUpdatedBy(int64(userID)).SetUpdatedAt(time.Now()).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to archive ticket type: %w", err)
	}
	txService := &TicketTypeService{client: txc, logger: s.logger}
	if err := txService.recordTicketTypeAudit(ctx, tenantID, userID, id, "archive", map[string]interface{}{"code": existing.Code, "previousStatus": existing.Status, "status": dto.TicketTypeStatusInactive}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交归档事务失败: %w", err)
	}
	return nil
}

func (s *TicketTypeService) recordTicketTypeAudit(ctx context.Context, tenantID, userID, ticketTypeID int, action string, details interface{}) error {
	body, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("序列化工单类型审计详情失败: %w", err)
	}
	_, err = s.client.AuditLog.Create().SetTenantID(tenantID).SetUserID(userID).
		SetResource(fmt.Sprintf("ticket_type:%d", ticketTypeID)).SetAction(action).
		SetPath(fmt.Sprintf("service://ticket-types/%d", ticketTypeID)).SetMethod("DOMAIN").SetStatusCode(0).
		SetNillableRequestBody(auditStringPtr(string(body))).Save(ctx)
	if err != nil {
		return fmt.Errorf("记录工单类型审计失败: %w", err)
	}
	return nil
}

func ticketTypeBindingChanges(before, after *ent.TicketType) map[string]interface{} {
	changes := map[string]interface{}{}
	if before.WorkflowDefinitionKey != after.WorkflowDefinitionKey {
		changes["workflowDefinitionKey"] = map[string]interface{}{"before": before.WorkflowDefinitionKey, "after": after.WorkflowDefinitionKey}
	}
	if before.DefaultSLAID != after.DefaultSLAID || before.SLAEnabled != after.SLAEnabled {
		changes["sla"] = map[string]interface{}{"beforeId": before.DefaultSLAID, "afterId": after.DefaultSLAID, "beforeEnabled": before.SLAEnabled, "afterEnabled": after.SLAEnabled}
	}
	if before.AssignmentRuleID != after.AssignmentRuleID || before.AutoAssignEnabled != after.AutoAssignEnabled {
		changes["assignment"] = map[string]interface{}{"beforeId": before.AssignmentRuleID, "afterId": after.AssignmentRuleID, "beforeEnabled": before.AutoAssignEnabled, "afterEnabled": after.AutoAssignEnabled}
	}
	return changes
}

func auditStringPtr(value string) *string { return &value }

func (s *TicketTypeService) SetStatus(ctx context.Context, id, tenantID, userID int, status dto.TicketTypeStatus) (*dto.TicketTypeDefinition, error) {
	return s.UpdateTicketType(ctx, id, &dto.UpdateTicketTypeRequest{Status: &status}, tenantID, userID)
}

func (s *TicketTypeService) CloneTicketType(ctx context.Context, id, tenantID, userID int, code, name string) (*dto.TicketTypeDefinition, error) {
	source, err := s.GetTicketType(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	return s.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{
		Code: code, Name: name, Description: source.Description, Icon: source.Icon, Color: source.Color,
		CategoryID: source.CategoryID, DefaultPriority: source.DefaultPriority, SortOrder: source.SortOrder,
		WorkflowDefinitionKey: source.WorkflowDefinitionKey, AssignmentRuleID: source.AssignmentRuleID,
		CustomFields: source.CustomFields, ApprovalEnabled: source.ApprovalEnabled, ApprovalChain: source.ApprovalChain,
		SLAEnabled: source.SLAEnabled, DefaultSLAID: source.DefaultSLAID,
		AutoAssignEnabled: source.AutoAssignEnabled, AssignmentRules: source.AssignmentRules,
		NotificationConfig: source.NotificationConfig, PermissionConfig: source.PermissionConfig,
	}, tenantID, userID)
}

// checkCodeExists checks if a ticket type code already exists within a tenant
func (s *TicketTypeService) checkCodeExists(ctx context.Context, code string, tenantID int) (bool, error) {
	count, err := s.client.TicketType.Query().
		Where(
			tickettype.Code(code),
			tickettype.TenantID(int64(tenantID)),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *TicketTypeService) validateBindings(ctx context.Context, tenantID int, categoryID *int, workflowKey string, slaID, assignmentRuleID *int) error {
	if categoryID != nil {
		exists, err := s.client.TicketCategory.Query().Where(ticketcategory.IDEQ(*categoryID), ticketcategory.TenantIDEQ(tenantID), ticketcategory.IsActiveEQ(true)).Exist(ctx)
		if err != nil || !exists {
			return fmt.Errorf("工单分类不存在、已停用或不属于当前租户")
		}
	}
	if strings.TrimSpace(workflowKey) != "" {
		exists, err := s.client.ProcessDefinition.Query().Where(processdefinition.KeyEQ(workflowKey), processdefinition.TenantIDEQ(tenantID), processdefinition.IsActiveEQ(true), processdefinition.IsLatestEQ(true)).Exist(ctx)
		if err != nil || !exists {
			return fmt.Errorf("工作流不存在、未激活或不属于当前租户")
		}
	}
	if slaID != nil {
		exists, err := s.client.SLADefinition.Query().Where(sladefinition.IDEQ(*slaID), sladefinition.TenantIDEQ(tenantID), sladefinition.IsActiveEQ(true)).Exist(ctx)
		if err != nil || !exists {
			return fmt.Errorf("SLA不存在、已停用或不属于当前租户")
		}
	}
	if assignmentRuleID != nil {
		exists, err := s.client.TicketAssignmentRule.Query().Where(ticketassignmentrule.IDEQ(*assignmentRuleID), ticketassignmentrule.TenantIDEQ(tenantID), ticketassignmentrule.IsActiveEQ(true)).Exist(ctx)
		if err != nil || !exists {
			return fmt.Errorf("分配规则不存在、已停用或不属于当前租户")
		}
	}
	return nil
}

// toDefinition converts an ent TicketType to DTO
func (s *TicketTypeService) toDefinition(t *ent.TicketType) *dto.TicketTypeDefinition {
	return &dto.TicketTypeDefinition{
		ID:                    int(t.ID),
		Code:                  t.Code,
		Name:                  t.Name,
		Description:           t.Description,
		Icon:                  t.Icon,
		Color:                 t.Color,
		Status:                dto.TicketTypeStatus(t.Status),
		CategoryID:            intPtrFromInt(t.CategoryID),
		DefaultPriority:       t.DefaultPriority,
		SortOrder:             t.SortOrder,
		WorkflowDefinitionKey: t.WorkflowDefinitionKey,
		AssignmentRuleID:      intPtrFromInt(t.AssignmentRuleID),
		CustomFields:          convertCustomFields(t.CustomFields),
		ApprovalEnabled:       t.ApprovalEnabled,
		ApprovalWorkflowID:    strPtrFromInt64(t.ApprovalWorkflowID),
		ApprovalChain:         convertApprovalChain(t.ApprovalChain),
		SLAEnabled:            t.SLAEnabled,
		DefaultSLAID:          intPtrFromInt64(t.DefaultSLAID),
		AutoAssignEnabled:     t.AutoAssignEnabled,
		AssignmentRules:       convertAssignmentRules(t.AssignmentRules),
		NotificationConfig:    ptrNotificationConfig(t.NotificationConfig),
		PermissionConfig:      ptrPermissionConfig(t.PermissionConfig),
		TenantID:              int(t.TenantID),
		CreatedBy:             int(t.CreatedBy),
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
		UpdatedBy:             intPtr(t.UpdatedBy),
		UsageCount:            t.UsageCount,
	}
}

// intPtr converts int64 to *int
func intPtr(v int64) *int {
	if v == 0 {
		return nil
	}
	i := int(v)
	return &i
}

func intPtrFromInt(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func defaultPriority(value string) string {
	if strings.TrimSpace(value) == "" {
		return "medium"
	}
	return strings.ToLower(value)
}

func validTicketPriority(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "low", "medium", "high", "urgent", "critical":
		return true
	default:
		return false
	}
}

// intPtrFromInt64 converts int64 to *int
func intPtrFromInt64(v int64) *int {
	if v == 0 {
		return nil
	}
	i := int(v)
	return &i
}

// strPtrFromInt64 converts int64 to *string
func strPtrFromInt64(v int64) *string {
	if v == 0 {
		return nil
	}
	s := fmt.Sprintf("%d", v)
	return &s
}

// convertApprovalChain converts ent slice to DTO ApprovalChainDefinition slice
func convertApprovalChain(items []interface{}) []dto.ApprovalChainDefinition {
	if items == nil {
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return nil
	}
	var result []dto.ApprovalChainDefinition
	json.Unmarshal(data, &result)
	return result
}

// ptrNotificationConfig converts ent map to DTO pointer
func ptrNotificationConfig(m map[string]interface{}) *dto.NotificationConfig {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var result dto.NotificationConfig
	json.Unmarshal(data, &result)
	return &result
}

// ptrPermissionConfig converts ent map to DTO pointer
func ptrPermissionConfig(m map[string]interface{}) *dto.PermissionConfig {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var result dto.PermissionConfig
	json.Unmarshal(data, &result)
	return &result
}

// toCustomFieldsMap converts DTO slice to ent-compatible map
func toCustomFieldsMap(fields []dto.CustomFieldDefinition) map[string]interface{} {
	m := make(map[string]interface{})
	for _, f := range fields {
		data, _ := json.Marshal(f)
		var fm map[string]interface{}
		json.Unmarshal(data, &fm)
		m[f.Name] = fm
	}
	return m
}

// toAssignmentRulesMap converts DTO slice to ent-compatible slice of maps
func toAssignmentRulesMap(rules []dto.AssignmentRule) []interface{} {
	result := make([]interface{}, len(rules))
	for i, r := range rules {
		result[i] = r
	}
	return result
}

// toInterfaceSlice converts any typed slice to []interface{}
func toInterfaceSlice[T any](items []T) []interface{} {
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

// structToMap converts a struct to map via JSON marshal/unmarshal
func structToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}

// convertCustomFields converts ent map to DTO CustomFieldDefinition slice
func convertCustomFields(m map[string]interface{}) []dto.CustomFieldDefinition {
	if m == nil {
		return nil
	}
	// Compatibility for records written by the previous serializer, which stored
	// one field as a flat object instead of a map keyed by field name.
	if _, legacy := m["name"]; legacy {
		data, err := json.Marshal(m)
		if err != nil {
			return nil
		}
		var field dto.CustomFieldDefinition
		if json.Unmarshal(data, &field) == nil && field.Name != "" {
			return []dto.CustomFieldDefinition{field}
		}
		return nil
	}
	result := make([]dto.CustomFieldDefinition, 0, len(m))
	for _, raw := range m {
		data, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var field dto.CustomFieldDefinition
		if json.Unmarshal(data, &field) == nil {
			result = append(result, field)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return result
}

var customFieldNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

func validateCustomFields(fields []dto.CustomFieldDefinition) error {
	seen := make(map[string]struct{}, len(fields))
	for i, field := range fields {
		field.Name = strings.TrimSpace(field.Name)
		if !customFieldNamePattern.MatchString(field.Name) {
			return fmt.Errorf("第%d个字段的 name 无效", i+1)
		}
		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("字段 %s 的 label 不能为空", field.Name)
		}
		if _, ok := seen[field.Name]; ok {
			return fmt.Errorf("字段 name 重复: %s", field.Name)
		}
		seen[field.Name] = struct{}{}
		switch field.Type {
		case dto.CustomFieldTypeText, dto.CustomFieldTypeTextarea, dto.CustomFieldTypeNumber,
			dto.CustomFieldTypeDate, dto.CustomFieldTypeDatetime, dto.CustomFieldTypeSelect,
			dto.CustomFieldTypeMultiSelect, dto.CustomFieldTypeCheckbox, dto.CustomFieldTypeRadio,
			dto.CustomFieldTypeFile, dto.CustomFieldTypeUserPicker, dto.CustomFieldTypeDepartmentPicker,
			dto.CustomFieldTypeBoolean, dto.CustomFieldTypeUser, dto.CustomFieldTypeDepartment, dto.CustomFieldTypeCI:
		default:
			return fmt.Errorf("字段 %s 的类型不受支持: %s", field.Name, field.Type)
		}
		if (field.Type == dto.CustomFieldTypeSelect || field.Type == dto.CustomFieldTypeMultiSelect || field.Type == dto.CustomFieldTypeRadio) && len(field.Options) == 0 {
			return fmt.Errorf("字段 %s 必须配置选项", field.Name)
		}
		// 只读字段用户无法填值；若同时必填则必须提供默认值，否则工单永远无法创建
		if field.Readonly && field.Required && field.DefaultValue == nil {
			return fmt.Errorf("只读字段 %s 为必填项时必须配置默认值", field.Name)
		}
	}
	return nil
}

// convertAssignmentRules converts ent slice to DTO AssignmentRule slice
func convertAssignmentRules(items []interface{}) []dto.AssignmentRule {
	if items == nil {
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return nil
	}
	var result []dto.AssignmentRule
	json.Unmarshal(data, &result)
	return result
}
