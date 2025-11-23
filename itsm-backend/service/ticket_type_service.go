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

type TicketTypeService struct {
	client *ent.Client
	rawDB  *sql.DB
	logger *zap.SugaredLogger
}

func NewTicketTypeService(client *ent.Client, rawDB *sql.DB, logger *zap.SugaredLogger) *TicketTypeService {
	return &TicketTypeService{
		client: client,
		rawDB:  rawDB,
		logger: logger,
	}
}

// CreateTicketType 创建工单类型
func (s *TicketTypeService) CreateTicketType(ctx context.Context, req *dto.CreateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	s.logger.Infow("Creating ticket type", "code", req.Code, "tenant_id", tenantID)

	// 检查编码是否已存在
	exists, err := s.checkCodeExists(ctx, req.Code, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check code existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("工单类型编码已存在: %s", req.Code)
	}

	// 序列化复杂字段
	customFieldsJSON, err := json.Marshal(req.CustomFields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom fields: %w", err)
	}

	approvalChainJSON, err := json.Marshal(req.ApprovalChain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approval chain: %w", err)
	}

	assignmentRulesJSON, err := json.Marshal(req.AssignmentRules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal assignment rules: %w", err)
	}

	notificationConfigJSON, err := json.Marshal(req.NotificationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification config: %w", err)
	}

	permissionConfigJSON, err := json.Marshal(req.PermissionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal permission config: %w", err)
	}

	// 插入数据
	query := `
		INSERT INTO ticket_types (
			code, name, description, icon, color, status,
			custom_fields, approval_enabled, approval_workflow_id, approval_chain,
			sla_enabled, default_sla_id, auto_assign_enabled, assignment_rules,
			notification_config, permission_config,
			created_by, tenant_id, created_at, updated_at, usage_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16,
			$17, $18, $19, $20, 0
		)
		RETURNING id
	`

	var id int
	err = s.rawDB.QueryRowContext(ctx, query,
		req.Code, req.Name, req.Description, req.Icon, req.Color, dto.TicketTypeStatusActive,
		customFieldsJSON, req.ApprovalEnabled, nil, approvalChainJSON,
		req.SLAEnabled, req.DefaultSLAID, req.AutoAssignEnabled, assignmentRulesJSON,
		notificationConfigJSON, permissionConfigJSON,
		userID, tenantID, time.Now(), time.Now(),
	).Scan(&id)

	if err != nil {
		s.logger.Errorw("Failed to create ticket type", "error", err)
		return nil, fmt.Errorf("failed to create ticket type: %w", err)
	}

	// 查询并返回创建的工单类型
	return s.GetTicketType(ctx, id, tenantID)
}

// UpdateTicketType 更新工单类型
func (s *TicketTypeService) UpdateTicketType(ctx context.Context, id int, req *dto.UpdateTicketTypeRequest, tenantID, userID int) (*dto.TicketTypeDefinition, error) {
	s.logger.Infow("Updating ticket type", "id", id, "tenant_id", tenantID)

	// 检查工单类型是否存在
	existing, err := s.GetTicketType(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 构建更新语句
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.Icon != nil {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIndex))
		args = append(args, *req.Icon)
		argIndex++
	}

	if req.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argIndex))
		args = append(args, *req.Color)
		argIndex++
	}

	if req.Status != nil {
		updates = append(updates, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	if req.CustomFields != nil {
		customFieldsJSON, err := json.Marshal(req.CustomFields)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal custom fields: %w", err)
		}
		updates = append(updates, fmt.Sprintf("custom_fields = $%d", argIndex))
		args = append(args, customFieldsJSON)
		argIndex++
	}

	if req.ApprovalEnabled != nil {
		updates = append(updates, fmt.Sprintf("approval_enabled = $%d", argIndex))
		args = append(args, *req.ApprovalEnabled)
		argIndex++
	}

	if req.ApprovalChain != nil {
		approvalChainJSON, err := json.Marshal(req.ApprovalChain)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal approval chain: %w", err)
		}
		updates = append(updates, fmt.Sprintf("approval_chain = $%d", argIndex))
		args = append(args, approvalChainJSON)
		argIndex++
	}

	if req.SLAEnabled != nil {
		updates = append(updates, fmt.Sprintf("sla_enabled = $%d", argIndex))
		args = append(args, *req.SLAEnabled)
		argIndex++
	}

	if req.DefaultSLAID != nil {
		updates = append(updates, fmt.Sprintf("default_sla_id = $%d", argIndex))
		args = append(args, *req.DefaultSLAID)
		argIndex++
	}

	if req.AutoAssignEnabled != nil {
		updates = append(updates, fmt.Sprintf("auto_assign_enabled = $%d", argIndex))
		args = append(args, *req.AutoAssignEnabled)
		argIndex++
	}

	if req.AssignmentRules != nil {
		assignmentRulesJSON, err := json.Marshal(req.AssignmentRules)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal assignment rules: %w", err)
		}
		updates = append(updates, fmt.Sprintf("assignment_rules = $%d", argIndex))
		args = append(args, assignmentRulesJSON)
		argIndex++
	}

	if req.NotificationConfig != nil {
		notificationConfigJSON, err := json.Marshal(req.NotificationConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal notification config: %w", err)
		}
		updates = append(updates, fmt.Sprintf("notification_config = $%d", argIndex))
		args = append(args, notificationConfigJSON)
		argIndex++
	}

	if req.PermissionConfig != nil {
		permissionConfigJSON, err := json.Marshal(req.PermissionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal permission config: %w", err)
		}
		updates = append(updates, fmt.Sprintf("permission_config = $%d", argIndex))
		args = append(args, permissionConfigJSON)
		argIndex++
	}

	if len(updates) == 0 {
		return existing, nil
	}

	// 添加updated_by和updated_at
	updates = append(updates, fmt.Sprintf("updated_by = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// 添加WHERE条件
	args = append(args, id, tenantID)

	query := fmt.Sprintf(`
		UPDATE ticket_types 
		SET %s
		WHERE id = $%d AND tenant_id = $%d
	`, joinUpdates(updates), argIndex, argIndex+1)

	_, err = s.rawDB.ExecContext(ctx, query, args...)
	if err != nil {
		s.logger.Errorw("Failed to update ticket type", "error", err)
		return nil, fmt.Errorf("failed to update ticket type: %w", err)
	}

	return s.GetTicketType(ctx, id, tenantID)
}

// GetTicketType 获取工单类型详情
func (s *TicketTypeService) GetTicketType(ctx context.Context, id, tenantID int) (*dto.TicketTypeDefinition, error) {
	query := `
		SELECT 
			tt.id, tt.code, tt.name, tt.description, tt.icon, tt.color, tt.status,
			tt.custom_fields, tt.approval_enabled, tt.approval_workflow_id, tt.approval_chain,
			tt.sla_enabled, tt.default_sla_id, tt.auto_assign_enabled, tt.assignment_rules,
			tt.notification_config, tt.permission_config,
			tt.created_by, u1.name as created_by_name,
			tt.updated_by, u2.name as updated_by_name,
			tt.created_at, tt.updated_at, tt.tenant_id, tt.usage_count
		FROM ticket_types tt
		LEFT JOIN users u1 ON tt.created_by = u1.id
		LEFT JOIN users u2 ON tt.updated_by = u2.id
		WHERE tt.id = $1 AND tt.tenant_id = $2
	`

	var ticketType dto.TicketTypeDefinition
	var customFieldsJSON, approvalChainJSON, assignmentRulesJSON, notificationConfigJSON, permissionConfigJSON []byte
	var updatedBy sql.NullInt64
	var updatedByName sql.NullString

	err := s.rawDB.QueryRowContext(ctx, query, id, tenantID).Scan(
		&ticketType.ID, &ticketType.Code, &ticketType.Name, &ticketType.Description,
		&ticketType.Icon, &ticketType.Color, &ticketType.Status,
		&customFieldsJSON, &ticketType.ApprovalEnabled, &ticketType.ApprovalWorkflowID, &approvalChainJSON,
		&ticketType.SLAEnabled, &ticketType.DefaultSLAID, &ticketType.AutoAssignEnabled, &assignmentRulesJSON,
		&notificationConfigJSON, &permissionConfigJSON,
		&ticketType.CreatedBy, &ticketType.CreatedByName,
		&updatedBy, &updatedByName,
		&ticketType.CreatedAt, &ticketType.UpdatedAt, &ticketType.TenantID, &ticketType.UsageCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("工单类型不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ticket type: %w", err)
	}

	// 处理可为空的字段
	if updatedBy.Valid {
		ub := int(updatedBy.Int64)
		ticketType.UpdatedBy = &ub
	}
	if updatedByName.Valid {
		ticketType.UpdatedByName = &updatedByName.String
	}

	// 反序列化JSON字段
	if len(customFieldsJSON) > 0 {
		if err := json.Unmarshal(customFieldsJSON, &ticketType.CustomFields); err != nil {
			s.logger.Warnw("Failed to unmarshal custom fields", "error", err)
		}
	}

	if len(approvalChainJSON) > 0 {
		if err := json.Unmarshal(approvalChainJSON, &ticketType.ApprovalChain); err != nil {
			s.logger.Warnw("Failed to unmarshal approval chain", "error", err)
		}
	}

	if len(assignmentRulesJSON) > 0 {
		if err := json.Unmarshal(assignmentRulesJSON, &ticketType.AssignmentRules); err != nil {
			s.logger.Warnw("Failed to unmarshal assignment rules", "error", err)
		}
	}

	if len(notificationConfigJSON) > 0 {
		if err := json.Unmarshal(notificationConfigJSON, &ticketType.NotificationConfig); err != nil {
			s.logger.Warnw("Failed to unmarshal notification config", "error", err)
		}
	}

	if len(permissionConfigJSON) > 0 {
		if err := json.Unmarshal(permissionConfigJSON, &ticketType.PermissionConfig); err != nil {
			s.logger.Warnw("Failed to unmarshal permission config", "error", err)
		}
	}

	return &ticketType, nil
}

// ListTicketTypes 获取工单类型列表
func (s *TicketTypeService) ListTicketTypes(ctx context.Context, req *dto.ListTicketTypesRequest, tenantID int) (*dto.TicketTypeListResponse, error) {
	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// 构建WHERE条件
	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	argIndex := 2

	if req.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	if req.Keyword != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex, argIndex))
		keyword := "%" + req.Keyword + "%"
		args = append(args, keyword)
		argIndex++
	}

	whereClause := joinWhereClausesAnd(whereClauses)

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ticket_types WHERE %s", whereClause)
	var total int64
	err := s.rawDB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count ticket types: %w", err)
	}

	// 查询列表
	offset := (req.Page - 1) * req.PageSize
	query := fmt.Sprintf(`
		SELECT 
			id, code, name, description, icon, color, status,
			approval_enabled, sla_enabled, auto_assign_enabled,
			created_by, created_at, updated_at, usage_count
		FROM ticket_types
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, req.PageSize, offset)

	rows, err := s.rawDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ticket types: %w", err)
	}
	defer rows.Close()

	types := []dto.TicketTypeDefinition{}
	for rows.Next() {
		var t dto.TicketTypeDefinition
		err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description, &t.Icon, &t.Color, &t.Status,
			&t.ApprovalEnabled, &t.SLAEnabled, &t.AutoAssignEnabled,
			&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.UsageCount,
		)
		if err != nil {
			s.logger.Warnw("Failed to scan ticket type", "error", err)
			continue
		}
		types = append(types, t)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &dto.TicketTypeListResponse{
		Types:      types,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// DeleteTicketType 删除工单类型
func (s *TicketTypeService) DeleteTicketType(ctx context.Context, id, tenantID int) error {
	// 检查是否有工单使用此类型
	var count int
	countQuery := "SELECT COUNT(*) FROM tickets WHERE ticket_type_id = $1 AND tenant_id = $2"
	err := s.rawDB.QueryRowContext(ctx, countQuery, id, tenantID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check ticket usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("无法删除：有 %d 个工单正在使用此类型", count)
	}

	// 删除工单类型
	deleteQuery := "DELETE FROM ticket_types WHERE id = $1 AND tenant_id = $2"
	result, err := s.rawDB.ExecContext(ctx, deleteQuery, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete ticket type: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("工单类型不存在")
	}

	return nil
}

// checkCodeExists 检查编码是否存在
func (s *TicketTypeService) checkCodeExists(ctx context.Context, code string, tenantID int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM ticket_types WHERE code = $1 AND tenant_id = $2"
	err := s.rawDB.QueryRowContext(ctx, query, code, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 辅助函数
func joinUpdates(updates []string) string {
	result := ""
	for i, update := range updates {
		if i > 0 {
			result += ", "
		}
		result += update
	}
	return result
}

func joinWhereClausesAnd(clauses []string) string {
	result := ""
	for i, clause := range clauses {
		if i > 0 {
			result += " AND "
		}
		result += clause
	}
	return result
}

