package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/tickettype"

	"go.uber.org/zap"
)

type TicketTypeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
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
	if len(req.Code) > 64 || len(req.Name) > 128 {
		return nil, fmt.Errorf("工单类型编码或名称过长")
	}
	if req.ApprovalEnabled && len(req.ApprovalChain) == 0 {
		return nil, fmt.Errorf("启用审批时必须配置审批链")
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
	assignmentRulesMap := toAssignmentRulesMap(req.AssignmentRules)
	approvalChainSlice := toInterfaceSlice(req.ApprovalChain)
	notificationConfigMap := structToMap(req.NotificationConfig)
	permissionConfigMap := structToMap(req.PermissionConfig)

	// Build create mutation via ent client（schema 中 created_at/updated_at 为必填字段，必须显式设置）
	now := time.Now()
	obj, err := s.client.TicketType.Create().
		SetCode(req.Code).
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetColor(req.Color).
		SetStatus(string(dto.TicketTypeStatusActive)).
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
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket type", "error", err)
		return nil, fmt.Errorf("failed to create ticket type: %w", err)
	}

	return s.toDefinition(obj), nil
}

// UpdateTicketType updates an existing ticket type using ent
func (s *TicketTypeService) UpdateTicketType(ctx context.Context, id int, req *dto.UpdateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
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
		update.SetStatus(string(*req.Status))
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
		Order(ent.Desc(tickettype.FieldUpdatedAt))

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

// DeleteTicketType deletes a ticket type
func (s *TicketTypeService) DeleteTicketType(ctx context.Context, id, tenantID int) error {
	existing, err := s.client.TicketType.Get(ctx, int(id))
	if err != nil {
		return fmt.Errorf("ticket type not found: %w", err)
	}
	if int(existing.TenantID) != tenantID {
		return fmt.Errorf("ticket type not found")
	}

	if err := s.client.TicketType.DeleteOneID(int(id)).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete ticket type: %w", err)
	}
	return nil
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

// toDefinition converts an ent TicketType to DTO
func (s *TicketTypeService) toDefinition(t *ent.TicketType) *dto.TicketTypeDefinition {
	return &dto.TicketTypeDefinition{
		ID:                 int(t.ID),
		Code:               t.Code,
		Name:               t.Name,
		Description:        t.Description,
		Icon:               t.Icon,
		Color:              t.Color,
		Status:             dto.TicketTypeStatus(t.Status),
		ApprovalEnabled:    t.ApprovalEnabled,
		ApprovalWorkflowID: strPtrFromInt64(t.ApprovalWorkflowID),
		ApprovalChain:      convertApprovalChain(t.ApprovalChain),
		SLAEnabled:         t.SLAEnabled,
		DefaultSLAID:       intPtrFromInt64(t.DefaultSLAID),
		AutoAssignEnabled:  t.AutoAssignEnabled,
		AssignmentRules:    convertAssignmentRules(t.AssignmentRules),
		NotificationConfig: ptrNotificationConfig(t.NotificationConfig),
		PermissionConfig:   ptrPermissionConfig(t.PermissionConfig),
		TenantID:           int(t.TenantID),
		CreatedBy:          int(t.CreatedBy),
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		UpdatedBy:          intPtr(t.UpdatedBy),
		UsageCount:         t.UsageCount,
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
