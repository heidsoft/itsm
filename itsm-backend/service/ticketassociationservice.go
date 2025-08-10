package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettag"
	"time"
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
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Priority      string                 `json:"priority"`
	CategoryID    *int                   `json:"category_id,omitempty"`
	TemplateID    *int                   `json:"template_id,omitempty"`
	ParentID      *int                   `json:"parent_id,omitempty"`
	RelatedIDs    []int                  `json:"related_ids,omitempty"`
	TagIDs        []int                  `json:"tag_ids,omitempty"`
	TenantID      int                    `json:"tenant_id"`
	AssignedTo    *int                   `json:"assigned_to,omitempty"`
	CustomFields  map[string]interface{} `json:"custom_fields,omitempty"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID            int                    `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Priority      string                 `json:"priority"`
	Status        string                 `json:"status"`
	CategoryID    *int                   `json:"category_id,omitempty"`
	TemplateID    *int                   `json:"template_id,omitempty"`
	ParentID      *int                   `json:"parent_id,omitempty"`
	RelatedIDs    []int                  `json:"related_ids,omitempty"`
	TagIDs        []int                  `json:"tag_ids,omitempty"`
	TenantID      int                    `json:"tenant_id"`
	AssignedTo    *int                   `json:"assigned_to,omitempty"`
	CustomFields  map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
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

	// 添加关联工单
	if len(req.RelatedIDs) > 0 {
		_, err = ticketEntity.Update().
			AddRelatedTicketIDs(req.RelatedIDs...).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("添加关联工单失败: %w", err)
		}
	}

	return s.buildTicketResponse(ticketEntity), nil
}

// GetTicketWithAssociations 获取工单及其关联信息
func (s *TicketAssociationService) GetTicketWithAssociations(ctx context.Context, ticketID int) (*TicketResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		WithTags().
		WithRelatedTickets().
		WithParentTicket().
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
		WithRelatedTickets().
		WithParentTicket().
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

	// 更新关联工单
	if req.RelatedIDs != nil {
		// 先清除现有关联
		_, err = s.client.Ticket.UpdateOneID(ticketID).
			ClearRelatedTickets().
			Save(ctx)
		if err != nil {
			return fmt.Errorf("清除关联工单失败: %w", err)
		}

		// 添加新关联
		if len(req.RelatedIDs) > 0 {
			_, err = s.client.Ticket.UpdateOneID(ticketID).
				AddRelatedTicketIDs(req.RelatedIDs...).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("添加关联工单失败: %w", err)
			}
		}
	}

	return nil
}

// GetRelatedTickets 获取关联工单
func (s *TicketAssociationService) GetRelatedTickets(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		WithTags().
		WithRelatedTickets().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	relatedTickets := ticketEntity.Edges.RelatedTickets
	response := make([]*TicketResponse, len(relatedTickets))
	for i, related := range relatedTickets {
		response[i] = s.buildTicketResponse(related)
	}

	return response, nil
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
	ParentID   *int   `json:"parent_id,omitempty"`
	RelatedIDs []int  `json:"related_ids,omitempty"`
	TagIDs     []int  `json:"tag_ids,omitempty"`
}

// TicketHierarchy 工单层级结构
type TicketHierarchy struct {
	Ticket   *TicketResponse   `json:"ticket"`
	Children []*TicketResponse `json:"children"`
}

// TicketDependencies 工单依赖关系
type TicketDependencies struct {
	ParentChain    []*TicketResponse   `json:"parent_chain"`
	ChildrenTree   []*TicketResponse   `json:"children_tree"`
	RelatedTickets []*TicketResponse   `json:"related_tickets"`
}

// buildTicketResponse 构建工单响应
func (s *TicketAssociationService) buildTicketResponse(ticket *ent.Ticket) *TicketResponse {
	response := &TicketResponse{
		ID:           ticket.ID,
		Title:        ticket.Title,
		Description:  ticket.Description,
		Priority:     ticket.Priority,
		Status:       ticket.Status,
		TenantID:     ticket.TenantID,
		CreatedAt:    ticket.CreatedAt,
		UpdatedAt:    ticket.UpdatedAt,
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

	// 处理关联工单
	if ticket.Edges.RelatedTickets != nil {
		response.RelatedIDs = make([]int, len(ticket.Edges.RelatedTickets))
		for i, related := range ticket.Edges.RelatedTickets {
			response.RelatedIDs[i] = related.ID
		}
	}

	return response
}

// getParentChain 获取父工单链
func (s *TicketAssociationService) getParentChain(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	var chain []*TicketResponse
	currentID := ticketID

	for {
		ticketEntity, err := s.client.Ticket.Query().
			WithParentTicket().
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
