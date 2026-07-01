package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettag"
)

// TicketTagService 工单标签服务
type TicketTagService struct {
	client *ent.Client
}

// NewTicketTagService 创建工单标签服务实例
func NewTicketTagService(client *ent.Client) *TicketTagService {
	return &TicketTagService{client: client}
}

// ResolveTagIDsByNames resolves tenant-scoped tag names to IDs. When createMissing
// is true, missing names are created as active tags so existing frontend flows that
// submit tag labels can work without a separate tag dictionary setup step.
func (s *TicketTagService) ResolveTagIDsByNames(ctx context.Context, names []string, tenantID int, createMissing bool) ([]int, error) {
	uniqueNames := make([]string, 0, len(names))
	seenNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := strings.TrimSpace(name)
		if normalized == "" {
			continue
		}
		if _, ok := seenNames[normalized]; ok {
			continue
		}
		seenNames[normalized] = struct{}{}
		uniqueNames = append(uniqueNames, normalized)
	}
	if len(uniqueNames) == 0 {
		return nil, errors.New("标签不能为空")
	}

	existingTags, err := s.client.TicketTag.Query().
		Where(
			tickettag.NameIn(uniqueNames...),
			tickettag.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(uniqueNames))
	found := make(map[string]int, len(existingTags))
	for _, tag := range existingTags {
		found[tag.Name] = tag.ID
		ids = append(ids, tag.ID)
	}

	missing := make([]string, 0)
	for _, name := range uniqueNames {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 && !createMissing {
		return nil, errors.New("部分标签不存在")
	}

	for _, name := range missing {
		tag, err := s.client.TicketTag.Create().
			SetName(name).
			SetColor("#1890ff").
			SetIsActive(true).
			SetTenantID(tenantID).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		ids = append(ids, tag.ID)
	}

	return ids, nil
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
func (s *TicketTagService) GetTag(ctx context.Context, id int, tenantID int) (*ent.TicketTag, error) {
	return s.client.TicketTag.Query().
		Where(
			tickettag.ID(id),
			tickettag.TenantID(tenantID),
		).
		Only(ctx)
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
	update := s.client.TicketTag.UpdateOneID(id).
		Where(tickettag.TenantID(tenantID))

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
func (s *TicketTagService) DeleteTag(ctx context.Context, id int, tenantID int) error {
	// 检查是否有工单使用此标签
	ticketsCount, err := s.client.Ticket.Query().
		Where(
			ticket.HasTagsWith(tickettag.ID(id)),
			ticket.TenantID(tenantID),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if ticketsCount > 0 {
		return errors.New("无法删除正在使用的标签")
	}

	return s.client.TicketTag.DeleteOneID(id).
		Where(tickettag.TenantID(tenantID)).
		Exec(ctx)
}

// AssignTagsToTicket 为工单分配标签
func (s *TicketTagService) AssignTagsToTicket(ctx context.Context, ticketID int, tagIDs []int, tenantID int) error {
	// 获取工单
	ticket, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	// 获取标签
	tags, err := s.client.TicketTag.Query().
		Where(
			tickettag.IDIn(tagIDs...),
			tickettag.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		return err
	}
	if len(tags) != len(tagIDs) {
		return errors.New("部分标签不存在")
	}

	// 更新工单标签关联
	_, err = ticket.Update().
		AddTags(tags...).
		Save(ctx)

	return err
}

// RemoveTagsFromTicket 从工单移除标签
func (s *TicketTagService) RemoveTagsFromTicket(ctx context.Context, ticketID int, tagIDs []int, tenantID int) error {
	// 获取工单
	ticket, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	// 获取标签
	tags, err := s.client.TicketTag.Query().
		Where(
			tickettag.IDIn(tagIDs...),
			tickettag.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		return err
	}
	if len(tags) != len(tagIDs) {
		return errors.New("部分标签不存在")
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
