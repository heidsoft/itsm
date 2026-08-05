package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettemplate"
)

// TicketTemplateService 工单模板服务
type TicketTemplateService struct {
	client   *ent.Client
	fieldSvc *FieldDefinitionService
}

// NewTicketTemplateService 创建工单模板服务实例
func NewTicketTemplateService(client *ent.Client) *TicketTemplateService {
	return &TicketTemplateService{client: client, fieldSvc: NewFieldDefinitionService(client)}
}

// CreateTemplate 创建工单模板
func (s *TicketTemplateService) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*ent.TicketTemplate, error) {
	if err := validateTicketTemplate(req.Name, req.Category, req.Priority, req.Fields); err != nil {
		return nil, err
	}
	// 序列化工作流步骤
	workflowStepsBytes, err := json.Marshal(req.WorkflowSteps)
	if err != nil {
		return nil, err
	}

	template, err := s.client.TicketTemplate.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetPriority(req.Priority).
		SetWorkflowSteps(workflowStepsBytes).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := s.fieldSvc.ReplaceDefinitions(ctx, req.TenantID, "ticket_template", template.ID, req.Fields); err != nil {
		return nil, err
	}

	return template, nil
}

// GetTemplate 获取工单模板
func (s *TicketTemplateService) GetTemplate(ctx context.Context, id int, tenantID int) (*ent.TicketTemplate, error) {
	return s.client.TicketTemplate.Query().
		Where(
			tickettemplate.IDEQ(id),
			tickettemplate.TenantIDEQ(tenantID),
		).
		Only(ctx)
}

// ListTemplates 获取工单模板列表
func (s *TicketTemplateService) ListTemplates(ctx context.Context, req *ListTemplatesRequest) ([]*ent.TicketTemplate, int, error) {
	query := s.client.TicketTemplate.Query()

	// 应用过滤条件
	if req.Category != "" {
		query = query.Where(tickettemplate.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(tickettemplate.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(tickettemplate.TenantID(req.TenantID))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 应用分页和排序
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	if req.SortBy != "" {
		switch req.SortBy {
		case "name":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(tickettemplate.FieldName))
			} else {
				query = query.Order(ent.Asc(tickettemplate.FieldName))
			}
		case "created_at":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(tickettemplate.FieldCreatedAt))
			} else {
				query = query.Order(ent.Asc(tickettemplate.FieldCreatedAt))
			}
		}
	}

	templates, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// UpdateTemplate 更新工单模板
func (s *TicketTemplateService) UpdateTemplate(ctx context.Context, id int, req *UpdateTemplateRequest, tenantID int) (*ent.TicketTemplate, error) {
	if req.Fields != nil {
		if err := validateTemplateFields(req.Fields); err != nil {
			return nil, err
		}
	}
	update := s.client.TicketTemplate.UpdateOneID(id).
		Where(tickettemplate.TenantIDEQ(tenantID))

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.Priority != "" {
		update.SetPriority(req.Priority)
	}
	if req.WorkflowSteps != nil {
		workflowStepsBytes, err := json.Marshal(req.WorkflowSteps)
		if err != nil {
			return nil, err
		}
		update.SetWorkflowSteps(workflowStepsBytes)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	update.SetUpdatedAt(time.Now())

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	if req.Fields != nil {
		if _, err := s.fieldSvc.ReplaceDefinitions(ctx, tenantID, "ticket_template", id, req.Fields); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// RenderTemplate validates required fields before a template is used to create a ticket.
// Field definitions now live in field_definitions (no more per-template FormFields JSON
// column), so this reads them via fieldSvc instead of decoding tmpl.FormFields. The old
// per-field "default value" application is dropped: FieldDefinitionInput/field_definitions
// has no default-value column, and this function has no callers anywhere in the codebase
// yet, so there's no observed behavior depending on it.
func (s *TicketTemplateService) RenderTemplate(ctx context.Context, id, tenantID int, values map[string]interface{}) (map[string]interface{}, error) {
	tmpl, err := s.GetTemplate(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在")
	}
	if !tmpl.IsActive {
		return nil, fmt.Errorf("模板已下架")
	}
	defs, err := s.fieldSvc.ListDefinitions(ctx, tenantID, "ticket_template", id)
	if err != nil {
		return nil, fmt.Errorf("加载模板字段定义失败: %w", err)
	}
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		result[key] = value
	}
	for _, def := range defs {
		if !def.Required {
			continue
		}
		value, exists := result[def.Name]
		if !exists || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return nil, fmt.Errorf("必填字段 %s 未填写", def.Name)
		}
	}
	return result, nil
}

func validateTicketTemplate(name, category, priority string, fields []FieldDefinitionInput) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	if strings.TrimSpace(category) == "" {
		return fmt.Errorf("模板分类不能为空")
	}
	if priority != "" && priority != "low" && priority != "medium" && priority != "high" && priority != "critical" {
		return fmt.Errorf("无效的模板优先级")
	}
	return validateTemplateFields(fields)
}

func validateTemplateFields(fields []FieldDefinitionInput) error {
	if len(fields) > 200 {
		return fmt.Errorf("模板字段不能超过 200 个")
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.Name) == "" {
			return fmt.Errorf("模板字段名不能为空")
		}
		if _, dup := seen[field.Name]; dup {
			return fmt.Errorf("模板字段 %s 重复", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return nil
}

// DeleteTemplate 删除工单模板
func (s *TicketTemplateService) DeleteTemplate(ctx context.Context, id int, tenantID int) error {
	// 检查是否有工单使用此模板
	count, err := s.client.Ticket.Query().
		Where(
			ticket.TemplateIDEQ(id),
			ticket.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("无法删除正在使用的模板")
	}

	if err := s.client.TicketTemplate.DeleteOneID(id).
		Where(tickettemplate.TenantIDEQ(tenantID)).
		Exec(ctx); err != nil {
		return err
	}

	return s.fieldSvc.DeleteDefinitions(ctx, tenantID, "ticket_template", id)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category" binding:"required"`
	Priority      string                   `json:"priority"`
	Fields        []FieldDefinitionInput   `json:"fields"`
	WorkflowSteps []map[string]interface{} `json:"workflowSteps"`
	IsActive      bool                     `json:"isActive"`
	TenantID      int                      `json:"tenantId" binding:"required"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category"`
	Priority      string                   `json:"priority"`
	Fields        []FieldDefinitionInput   `json:"fields"`
	WorkflowSteps []map[string]interface{} `json:"workflowSteps"`
	IsActive      *bool                    `json:"isActive"`
}

// ListTemplatesRequest 获取模板列表请求
type ListTemplatesRequest struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"page_size"`
	Category  string `json:"category" form:"category"`
	IsActive  *bool  `json:"isActive" form:"is_active"`
	TenantID  int    `json:"tenantId" form:"tenant_id"`
	SortBy    string `json:"sortBy" form:"sort_by"`
	SortOrder string `json:"sortOrder" form:"sort_order"`
}
