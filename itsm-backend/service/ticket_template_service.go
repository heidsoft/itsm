package service

import (
	"context"
	"encoding/json"
	"errors"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/tickettemplate"
	"time"
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
	// 验证分类是否存在
	if req.Category != "" {
		_, err := s.client.TicketCategory.Query().
			Where(ticketcategory.NameEQ(req.Category)).
			First(ctx)
		if err != nil {
			return nil, errors.New("指定的分类不存在")
		}
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
func (s *TicketTemplateService) GetTemplate(ctx context.Context, id int) (*ent.TicketTemplate, error) {
	return s.client.TicketTemplate.Get(ctx, id)
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
func (s *TicketTemplateService) UpdateTemplate(ctx context.Context, id int, req *UpdateTemplateRequest) (*ent.TicketTemplate, error) {
	update := s.client.TicketTemplate.UpdateOneID(id)

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

// DeleteTemplate 删除工单模板
func (s *TicketTemplateService) DeleteTemplate(ctx context.Context, id int) error {
	// 检查是否有工单使用此模板
	count, err := s.client.Ticket.Query().
		Where(ticket.TemplateIDEQ(id)).
		Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("无法删除正在使用的模板")
	}

	return s.client.TicketTemplate.DeleteOneID(id).Exec(ctx)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category" binding:"required"`
	Priority      string                   `json:"priority"`
	FormFields    map[string]interface{}   `json:"form_fields"`
	WorkflowSteps []map[string]interface{} `json:"workflow_steps"`
	IsActive      bool                     `json:"is_active"`
	TenantID      int                      `json:"tenant_id" binding:"required"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category"`
	Priority      string                   `json:"priority"`
	FormFields    map[string]interface{}   `json:"form_fields"`
	WorkflowSteps []map[string]interface{} `json:"workflow_steps"`
	IsActive      *bool                    `json:"is_active"`
}

// ListTemplatesRequest 获取模板列表请求
type ListTemplatesRequest struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
	Category  string `json:"category" form:"category"`
	IsActive  *bool  `json:"is_active" form:"is_active"`
	TenantID  int    `json:"tenant_id" form:"tenant_id"`
	SortBy    string `json:"sort_by" form:"sort_by"`
	SortOrder string `json:"sort_order" form:"sort_order"`
}
