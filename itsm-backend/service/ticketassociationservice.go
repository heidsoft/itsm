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

// GetTicketWithAssociations 获取工单及其关联信息
func (s *TicketAssociationService) GetTicketWithAssociations(ctx context.Context, ticketID int) (*TicketResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		WithTags().
		WithRelatedTickets().
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
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("启动工单关联事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()
	update := txClient.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantIDEQ(ticketEntity.TenantID), ticket.DeletedAtIsNil())

	if req.ParentID != nil {
		if *req.ParentID == 0 {
			update.SetNillableParentTicketID(nil)
		} else {
			if err := validateParentAssignment(ctx, txClient, ticketEntity, *req.ParentID); err != nil {
				return err
			}
			update.SetParentTicketID(*req.ParentID)
		}
	}

	if req.TagIDs != nil {
		tagIDs := uniqueIDs(req.TagIDs)
		if len(tagIDs) > 0 {
			count, err := txClient.TicketTag.Query().
				Where(tickettag.IDIn(tagIDs...), tickettag.TenantIDEQ(ticketEntity.TenantID), tickettag.IsActiveEQ(true)).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("验证标签失败: %w", err)
			}
			if count != len(tagIDs) {
				return fmt.Errorf("标签不存在或不可用")
			}
		}
		update.ClearTags()
		if len(tagIDs) > 0 {
			update.AddTagIDs(tagIDs...)
		}
	}

	if req.RelatedIDs != nil {
		ids := uniqueIDs(req.RelatedIDs)
		for _, relatedID := range ids {
			if relatedID == ticketID {
				return fmt.Errorf("工单不能关联自身")
			}
		}
		count, err := txClient.Ticket.Query().
			Where(ticket.IDIn(ids...), ticket.TenantIDEQ(ticketEntity.TenantID), ticket.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("验证关联工单失败: %w", err)
		}
		if count != len(ids) {
			return fmt.Errorf("关联工单不存在")
		}
		currentRelated, err := txClient.Ticket.Query().
			Where(ticket.ID(ticketID)).
			QueryRelatedTickets().
			Where(ticket.TenantIDEQ(ticketEntity.TenantID)).
			All(ctx)
		if err != nil {
			return fmt.Errorf("查询现有关联工单失败: %w", err)
		}
		for _, related := range currentRelated {
			if err := txClient.Ticket.UpdateOneID(related.ID).
				Where(ticket.TenantIDEQ(ticketEntity.TenantID)).
				RemoveRelatedTicketIDs(ticketID).
				Exec(ctx); err != nil {
				return fmt.Errorf("清除反向工单关联失败: %w", err)
			}
		}

		update.ClearRelatedTickets()
		if len(ids) > 0 {
			update.AddRelatedTicketIDs(ids...)
			for _, relatedID := range ids {
				if err := txClient.Ticket.UpdateOneID(relatedID).
					Where(ticket.TenantIDEQ(ticketEntity.TenantID), ticket.DeletedAtIsNil()).
					AddRelatedTicketIDs(ticketID).
					Exec(ctx); err != nil {
					return fmt.Errorf("写入反向工单关联失败: %w", err)
				}
			}
		}
	}

	if _, err := update.Save(ctx); err != nil {
		return fmt.Errorf("更新工单关联失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交工单关联事务失败: %w", err)
	}
	return nil
}

func validateParentAssignment(ctx context.Context, client *ent.Client, child *ent.Ticket, parentID int) error {
	if parentID == child.ID {
		return fmt.Errorf("工单不能作为自己的父工单")
	}
	visited := map[int]struct{}{child.ID: {}}
	currentID := parentID
	for currentID != 0 {
		if _, exists := visited[currentID]; exists {
			return fmt.Errorf("父工单关系不能形成循环")
		}
		visited[currentID] = struct{}{}
		parent, err := client.Ticket.Query().
			Where(ticket.ID(currentID), ticket.TenantIDEQ(child.TenantID), ticket.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			return fmt.Errorf("父工单不存在")
		}
		currentID = parent.ParentTicketID
	}
	return nil
}

// GetRelatedTickets 获取关联工单
func (s *TicketAssociationService) GetRelatedTickets(ctx context.Context, ticketID int) ([]*TicketResponse, error) {
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工单失败: %w", err)
	}

	related, err := ticketEntity.QueryRelatedTickets().
		Where(ticket.TenantIDEQ(ticketEntity.TenantID), ticket.DeletedAtIsNil()).
		WithTags().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取关联工单失败: %w", err)
	}
	responses := make([]*TicketResponse, len(related))
	for i, item := range related {
		responses[i] = s.buildTicketResponse(item)
	}
	return responses, nil
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

	if ticket.Edges.RelatedTickets != nil {
		response.RelatedIDs = make([]int, len(ticket.Edges.RelatedTickets))
		for i, related := range ticket.Edges.RelatedTickets {
			response.RelatedIDs[i] = related.ID
		}
	}

	return response
}

func uniqueIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
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
