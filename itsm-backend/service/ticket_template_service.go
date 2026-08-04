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
	client *ent.Client
}

// NewTicketTemplateService 创建工单模板服务实例
func NewTicketTemplateService(client *ent.Client) *TicketTemplateService {
	return &TicketTemplateService{client: client}
}

// CreateTemplate 创建工单模板
func (s *TicketTemplateService) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*ent.TicketTemplate, error) {
	if err := validateTicketTemplate(req.Name, req.Category, req.Priority, req.FormFields); err != nil {
		return nil, err
	}
	// 序列化表单字段
	formFieldsBytes, err := json.Marshal(req.FormFields)
	if err != nil {
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
		SetFormFields(formFieldsBytes).
		SetWorkflowSteps(workflowStepsBytes).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
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
	if req.FormFields != nil {
		if err := validateTemplateFields(req.FormFields); err != nil {
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
	if req.FormFields != nil {
		formFieldsBytes, err := json.Marshal(req.FormFields)
		if err != nil {
			return nil, err
		}
		update.SetFormFields(formFieldsBytes)
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

	return update.Save(ctx)
}

// RenderTemplate applies defaults and validates required fields before a template is used to create a ticket.
func (s *TicketTemplateService) RenderTemplate(ctx context.Context, id, tenantID int, values map[string]interface{}) (map[string]interface{}, error) {
	tmpl, err := s.GetTemplate(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("模板不存在")
	}
	if !tmpl.IsActive {
		return nil, fmt.Errorf("模板已下架")
	}
	var schema map[string]interface{}
	if len(tmpl.FormFields) > 0 {
		if err := json.Unmarshal(tmpl.FormFields, &schema); err != nil {
			return nil, fmt.Errorf("模板表单配置损坏: %w", err)
		}
	}
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		result[key] = value
	}
	for key, raw := range schema {
		field, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := result[key]; !exists {
			if defaultValue, hasDefault := field["default"]; hasDefault {
				result[key] = defaultValue
			}
		}
		if required, _ := field["required"].(bool); required {
			value, exists := result[key]
			if !exists || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
				return nil, fmt.Errorf("必填字段 %s 未填写", key)
			}
		}
	}
	return result, nil
}

func validateTicketTemplate(name, category, priority string, fields map[string]interface{}) error {
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

func validateTemplateFields(fields map[string]interface{}) error {
	if len(fields) > 200 {
		return fmt.Errorf("模板字段不能超过 200 个")
	}
	for key, raw := range fields {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("模板字段名不能为空")
		}
		field, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("模板字段 %s 配置格式无效", key)
		}
		if required, exists := field["required"]; exists {
			if _, ok := required.(bool); !ok {
				return fmt.Errorf("模板字段 %s 的 required 必须为布尔值", key)
			}
		}
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

	return s.client.TicketTemplate.DeleteOneID(id).
		Where(tickettemplate.TenantIDEQ(tenantID)).
		Exec(ctx)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category" binding:"required"`
	Priority      string                   `json:"priority"`
	FormFields    map[string]interface{}   `json:"formFields"`
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
	FormFields    map[string]interface{}   `json:"formFields"`
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
