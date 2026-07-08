package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettag"
)

// TicketAssociationService 工单关联服务
type TicketAssociationService struct {
	client *ent.Client
}

// NewTicketAssociationService 创建工单关联服务实例
func NewTicketAssociationService(client *ent.Client) *TicketAssociationService {
	return &TicketAssociationService{
		client: client,
	}
}

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Priority     string                 `json:"priority"`
	CategoryID   *int                   `json:"categoryId,omitempty"`
	TemplateID   *int                   `json:"templateId,omitempty"`
	ParentID     *int                   `json:"parentId,omitempty"`
	RelatedIDs   []int                  `json:"relatedIds,omitempty"`
	TagIDs       []int                  `json:"tagIds,omitempty"`
	TenantID     int                    `json:"tenantId"`
	AssignedTo   *int                   `json:"assignedTo,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Priority     string                 `json:"priority"`
	Status       string                 `json:"status"`
	CategoryID   *int                   `json:"categoryId,omitempty"`
	TemplateID   *int                   `json:"templateId,omitempty"`
	ParentID     *int                   `json:"parentId,omitempty"`
	RelatedIDs   []int                  `json:"relatedIds,omitempty"`
	TagIDs       []int                  `json:"tagIds,omitempty"`
	TenantID     int                    `json:"tenantId"`
	AssignedTo   *int                   `json:"assignedTo,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// CreateTicket 创建工单
func (s *TicketAssociationService) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*TicketResponse, error) {
	// 验证父工单是否存在
	if req.ParentID != nil {
		parentExists, err := s.client.Ticket.Query().
			Where(ticket.ID(*req.ParentID)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("验证父工单失败: %w", err)
		}
		if !parentExists {
			return nil, fmt.Errorf("父工单不存在")
		}
	}

	// 验证关联工单是否存在
	if len(req.RelatedIDs) > 0 {
		relatedExists, err := s.client.Ticket.Query().
			Where(ticket.IDIn(req.RelatedIDs...)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("验证关联工单失败: %w", err)
		}
		if !relatedExists {
			return nil, fmt.Errorf("关联工单不存在")
		}
	}

	// 验证标签是否存在
	if len(req.TagIDs) > 0 {
		tagsExist, err := s.client.TicketTag.Query().
			Where(tickettag.IDIn(req.TagIDs...)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("验证标签失败: %w", err)
		}
		if !tagsExist {
			return nil, fmt.Errorf("标签不存在")
		}
	}

	// 创建工单
	ticketCreate := s.client.Ticket.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetStatus("open").
		SetTenantID(req.TenantID)

	if req.CategoryID != nil {
		ticketCreate.SetCategoryID(*req.CategoryID)
	}
	if req.TemplateID != nil {
		ticketCreate.SetTemplateID(*req.TemplateID)
	}
	if req.ParentID != nil {
		ticketCreate.SetParentTicketID(*req.ParentID)
	}
	if req.AssignedTo != nil {
		ticketCreate.SetAssigneeID(*req.AssignedTo)
	}

	ticketEntity, err := ticketCreate.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	// 添加标签关联
	if len(req.TagIDs) > 0 {
		_, err = ticketEntity.Update().
			AddTagIDs(req.TagIDs...).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("添加标签失败: %w", err)
		}
	}

	// 添加关联工单（RelatedTickets 边暂未在 ent schema 中定义，跳过）
	// TODO: 在 ent/schema/ticket.go 中添加 related_tickets 自关联边后恢复
	_ = req.RelatedIDs

	return s.buildTicketResponse(ticketEntity), nil
}

// GetTicketWithAssociations 获取工单及其关联信息
func (s *TicketAssociationService) GetTicketWithAssociations(ctx context.Context, ticketID int) (*TicketResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		WithTags().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	return s.buildTicketResponse(ticketEntity), nil
}

// GetTicketHierarchy 获取工单层级结构
func (s *TicketAssociationService) GetTicketHierarchy(ctx context.Context, ticketID int) (*TicketHierarchy, error) {
	ticketEntity, err := s.client.Ticket.Query().
		WithTags().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	// 获取子工单
	children, err := s.client.Ticket.Query().
		WithTags().
		Where(ticket.ParentTicketID(ticketID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取子工单失败: %w", err)
	}

	// 构建层级结构
	hierarchy := &TicketHierarchy{
		Ticket:   s.buildTicketResponse(ticketEntity),
		Children: make([]*TicketResponse, len(children)),
	}

	for i, child := range children {
		hierarchy.Children[i] = s.buildTicketResponse(child)
	}

	return hierarchy, nil
}

// UpdateTicketAssociations 更新工单关联关系
func (s *TicketAssociationService) UpdateTicketAssociations(ctx context.Context, ticketID int, req *UpdateAssociationsRequest) error {
	// 验证工单是否存在
	_, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}

	update := s.client.Ticket.UpdateOneID(ticketID)

	// 更新父工单
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			// 设置为null
			update.SetNillableParentTicketID(nil)
		} else {
			// 验证父工单是否存在
			parentExists, err := s.client.Ticket.Query().
				Where(ticket.ID(*req.ParentID)).
				Exist(ctx)
			if err != nil {
				return fmt.Errorf("验证父工单失败: %w", err)
			}
			if !parentExists {
				return fmt.Errorf("父工单不存在")
			}
			update.SetParentTicketID(*req.ParentID)
		}
	}

	// 更新标签
	if req.TagIDs != nil {
		// 先清除现有标签
		_, err = s.client.Ticket.UpdateOneID(ticketID).
			ClearTags().
			Save(ctx)
		if err != nil {
			return fmt.Errorf("清除标签失败: %w", err)
		}

		// 添加新标签
		if len(req.TagIDs) > 0 {
			_, err = s.client.Ticket.UpdateOneID(ticketID).
				AddTagIDs(req.TagIDs...).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("添加标签失败: %w", err)
			}
		}
	}

	// 更新关联工单（RelatedTickets 边暂未在 ent schema 中定义，跳过）
	// TODO: 在 ent/schema/ticket.go 中添加 related_tickets 自关联边后恢复
	_ = req.RelatedIDs

	return nil
}

// GetRelatedTickets 获取关联工单
// TODO: RelatedTickets 边暂未在 ent schema 中定义，当前返回空列表
func (s *TicketAssociationService) GetRelatedTickets(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	_, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	return []*TicketResponse{}, nil
}

// GetTicketDependencies 获取工单依赖关系
func (s *TicketAssociationService) GetTicketDependencies(ctx context.Context, ticketID int) (*TicketDependencies, error) {
	// 获取父工单链
	parentChain, err := s.getParentChain(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// 获取子工单树
	childrenTree, err := s.getChildrenTree(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// 获取关联工单
	relatedTickets, err := s.GetRelatedTickets(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	return &TicketDependencies{
		ParentChain:    parentChain,
		ChildrenTree:   childrenTree,
		RelatedTickets: relatedTickets,
	}, nil
}

// UpdateAssociationsRequest 更新关联关系请求
type UpdateAssociationsRequest struct {
	ParentID   *int  `json:"parentId,omitempty"`
	RelatedIDs []int `json:"relatedIds,omitempty"`
	TagIDs     []int `json:"tagIds,omitempty"`
}

// TicketHierarchy 工单层级结构
type TicketHierarchy struct {
	Ticket   *TicketResponse   `json:"ticket"`
	Children []*TicketResponse `json:"children"`
}

// TicketDependencies 工单依赖关系
type TicketDependencies struct {
	ParentChain    []*TicketResponse `json:"parentChain"`
	ChildrenTree   []*TicketResponse `json:"childrenTree"`
	RelatedTickets []*TicketResponse `json:"relatedTickets"`
}

// buildTicketResponse 构建工单响应
func (s *TicketAssociationService) buildTicketResponse(ticket *ent.Ticket) *TicketResponse {
	response := &TicketResponse{
		ID:          ticket.ID,
		Title:       ticket.Title,
		Description: ticket.Description,
		Priority:    ticket.Priority,
		Status:      ticket.Status,
		TenantID:    ticket.TenantID,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
	}

	if ticket.CategoryID != 0 {
		response.CategoryID = &ticket.CategoryID
	}
	if ticket.TemplateID != 0 {
		response.TemplateID = &ticket.TemplateID
	}
	if ticket.ParentTicketID != 0 {
		response.ParentID = &ticket.ParentTicketID
	}
	if ticket.AssigneeID != 0 {
		response.AssignedTo = &ticket.AssigneeID
	}

	// 处理标签
	if ticket.Edges.Tags != nil {
		response.TagIDs = make([]int, len(ticket.Edges.Tags))
		for i, tag := range ticket.Edges.Tags {
			response.TagIDs[i] = tag.ID
		}
	}

	// 处理关联工单（RelatedTickets 边暂未在 ent schema 中定义）
	// TODO: 在 ent/schema/ticket.go 中添加 related_tickets 自关联边后恢复

	return response
}

// getParentChain 获取父工单链
func (s *TicketAssociationService) getParentChain(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	var chain []*TicketResponse
	currentID := ticketID

	for {
		ticketEntity, err := s.client.Ticket.Query().
			Where(ticket.ID(currentID)).
			Only(ctx)
		if err != nil {
			break
		}

		if ticketEntity.ParentTicketID == 0 {
			break
		}

		parent, err := s.client.Ticket.Query().
			Where(ticket.ID(ticketEntity.ParentTicketID)).
			Only(ctx)
		if err != nil {
			break
		}

		chain = append([]*TicketResponse{s.buildTicketResponse(parent)}, chain...)
		currentID = parent.ID
	}

	return chain, nil
}

// getChildrenTree 获取子工单树
func (s *TicketAssociationService) getChildrenTree(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	children, err := s.client.Ticket.Query().
		Where(ticket.ParentTicketID(ticketID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var tree []*TicketResponse
	for _, child := range children {
		childResponse := s.buildTicketResponse(child)

		// 递归获取子工单的子工单
		// 这里可以扩展结构来支持树形显示
		// 暂时只返回扁平列表

		tree = append(tree, childResponse)
	}

	return tree, nil
}

// ConfigurationItemResponse 配置项响应
type ConfigurationItemResponse struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	CIType       string `json:"ciType"`
	Status       string `json:"status"`
	SerialNumber string `json:"serialNumber,omitempty"`
}

// AddConfigurationItem 添加配置项关联
func (s *TicketAssociationService) AddConfigurationItem(ctx context.Context, ticketID, ciID int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID)
	if err != nil {
		return err
	}

	ci, err := s.getConfigurationItemForAssociation(ctx, ciID, ticketEntity.TenantID)
	if err != nil {
		return err
	}

	_, err = s.client.ConfigurationItem.UpdateOneID(ci.ID).
		AddTicketIDs(ticketID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("添加配置项关联失败: %w", err)
	}
	return nil
}

// RemoveConfigurationItem 移除配置项关联
func (s *TicketAssociationService) RemoveConfigurationItem(ctx context.Context, ticketID, ciID int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID)
	if err != nil {
		return err
	}

	ci, err := s.getConfigurationItemForAssociation(ctx, ciID, ticketEntity.TenantID)
	if err != nil {
		return err
	}

	_, err = s.client.ConfigurationItem.UpdateOneID(ci.ID).
		RemoveTicketIDs(ticketID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("移除配置项关联失败: %w", err)
	}
	return nil
}

// GetConfigurationItems 获取配置项列表
func (s *TicketAssociationService) GetConfigurationItems(ctx context.Context, ticketID int) ([]*ConfigurationItemResponse, error) {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	items, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(ticketEntity.TenantID),
			configurationitem.HasTicketsWith(ticket.ID(ticketID)),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取配置项列表失败: %w", err)
	}

	responses := make([]*ConfigurationItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, &ConfigurationItemResponse{
			ID:           item.ID,
			Name:         item.Name,
			CIType:       item.CiType,
			Status:       item.Status,
			SerialNumber: item.SerialNumber,
		})
	}

	return responses, nil
}

// SetConfigurationItems 批量设置配置项
func (s *TicketAssociationService) SetConfigurationItems(ctx context.Context, ticketID int, ciIDs []int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID)
	if err != nil {
		return err
	}

	existingItems, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(ticketEntity.TenantID),
			configurationitem.HasTicketsWith(ticket.ID(ticketID)),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询现有关联配置项失败: %w", err)
	}
	for _, item := range existingItems {
		if _, err := s.client.ConfigurationItem.UpdateOneID(item.ID).
			RemoveTicketIDs(ticketID).
			Save(ctx); err != nil {
			return fmt.Errorf("清除现有配置项关联失败: %w", err)
		}
	}

	for _, ciID := range ciIDs {
		if err := s.AddConfigurationItem(ctx, ticketID, ciID); err != nil {
			return err
		}
	}

	return nil
}

func (s *TicketAssociationService) getTicketForAssociation(ctx context.Context, ticketID int) (*ent.Ticket, error) {
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("工单不存在")
	}
	return ticketEntity, nil
}

func (s *TicketAssociationService) getConfigurationItemForAssociation(ctx context.Context, ciID, tenantID int) (*ent.ConfigurationItem, error) {
	ci, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(ciID),
			configurationitem.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("配置项不存在")
	}
	return ci, nil
}
