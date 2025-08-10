package service

import (
	"context"
	"errors"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettag"
	"time"
)

// TicketTagService 工单标签服务
type TicketTagService struct {
	client *ent.Client
}

// NewTicketTagService 创建工单标签服务实例
func NewTicketTagService(client *ent.Client) *TicketTagService {
	return &TicketTagService{client: client}
}

// CreateTag 创建工单标签
func (s *TicketTagService) CreateTag(ctx context.Context, req *CreateTagRequest) (*ent.TicketTag, error) {
	// 验证标签名称唯一性
	exists, err := s.client.TicketTag.Query().
		Where(tickettag.NameEQ(req.Name)).
		Where(tickettag.TenantID(req.TenantID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("标签名称已存在")
	}

	tag, err := s.client.TicketTag.Create().
		SetName(req.Name).
		SetColor(req.Color).
		SetDescription(req.Description).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return tag, nil
}

// GetTag 获取工单标签
func (s *TicketTagService) GetTag(ctx context.Context, id int) (*ent.TicketTag, error) {
	return s.client.TicketTag.Get(ctx, id)
}

// ListTags 获取工单标签列表
func (s *TicketTagService) ListTags(ctx context.Context, req *ListTagsRequest) ([]*ent.TicketTag, int, error) {
	query := s.client.TicketTag.Query()

	// 应用过滤条件
	if req.IsActive != nil {
		query = query.Where(tickettag.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(tickettag.TenantID(req.TenantID))
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

	// 默认按名称排序
	query = query.Order(ent.Asc(tickettag.FieldName))

	tags, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return tags, total, nil
}

// UpdateTag 更新工单标签
func (s *TicketTagService) UpdateTag(ctx context.Context, id int, req *UpdateTagRequest, tenantID int) (*ent.TicketTag, error) {
	update := s.client.TicketTag.UpdateOneID(id)

	if req.Name != "" {
		// 验证标签名称唯一性
		exists, err := s.client.TicketTag.Query().
			Where(tickettag.NameEQ(req.Name)).
			Where(tickettag.TenantIDEQ(tenantID)).
			Where(tickettag.IDNEQ(id)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("标签名称已存在")
		}
		update.SetName(req.Name)
	}
	if req.Color != "" {
		update.SetColor(req.Color)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	update.SetUpdatedAt(time.Now())

	return update.Save(ctx)
}

// DeleteTag 删除工单标签
func (s *TicketTagService) DeleteTag(ctx context.Context, id int) error {
	// 检查是否有工单使用此标签
	ticketsCount, err := s.client.Ticket.Query().
		Where(ticket.HasTagsWith(tickettag.ID(id))).
		Count(ctx)
	if err != nil {
		return err
	}
	if ticketsCount > 0 {
		return errors.New("无法删除正在使用的标签")
	}

	return s.client.TicketTag.DeleteOneID(id).Exec(ctx)
}

// AssignTagsToTicket 为工单分配标签
func (s *TicketTagService) AssignTagsToTicket(ctx context.Context, ticketID int, tagIDs []int) error {
	// 获取工单
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	// 获取标签
	tags, err := s.client.TicketTag.Query().
		Where(tickettag.IDIn(tagIDs...)).
		All(ctx)
	if err != nil {
		return err
	}

	// 更新工单标签关联
	_, err = ticket.Update().
		AddTags(tags...).
		Save(ctx)

	return err
}

// RemoveTagsFromTicket 从工单移除标签
func (s *TicketTagService) RemoveTagsFromTicket(ctx context.Context, ticketID int, tagIDs []int) error {
	// 获取工单
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	// 获取标签
	tags, err := s.client.TicketTag.Query().
		Where(tickettag.IDIn(tagIDs...)).
		All(ctx)
	if err != nil {
		return err
	}

	// 更新工单标签关联
	_, err = ticket.Update().
		RemoveTags(tags...).
		Save(ctx)

	return err
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	TenantID    int    `json:"tenant_id" binding:"required"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

// ListTagsRequest 获取标签列表请求
type ListTagsRequest struct {
	Page     int   `json:"page" form:"page"`
	PageSize int   `json:"page_size" form:"page_size"`
	IsActive *bool `json:"is_active" form:"is_active"`
	TenantID int   `json:"tenant_id" form:"tenant_id"`
}
