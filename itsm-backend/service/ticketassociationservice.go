package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
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

// UpdateTicketAssociations 更新工单关联关系。
//
// tenantID 必须是认证上下文里的调用者租户：锚点工单与后续每一处关联写入都按它收敛，
// 缺租户上下文（<=0）时 getTicketForAssociation 直接 fail-closed 成 404。
func (s *TicketAssociationService) UpdateTicketAssociations(ctx context.Context, ticketID, tenantID int, req *UpdateAssociationsRequest) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
	if err != nil {
		return err
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

// GetRelatedTickets 获取关联工单。
//
// 锚点与关联行都按 tenantID 过滤：related_tickets 是自关联 M2M 边，历史上只按锚点行
// 自己的租户过滤，等于把「谁在读」交给数据行决定，跨租户读取因此毫无阻力。
func (s *TicketAssociationService) GetRelatedTickets(ctx context.Context, ticketID, tenantID int) ([]*TicketResponse, error) {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	related, err := ticketEntity.QueryRelatedTickets().
		Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
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

// GetTicketDependencies 获取工单依赖关系。
//
// 先按调用者租户确认锚点工单，再遍历三条关联路径：跨租户与「工单不存在」在这里就返回
// 同一个 404，调用者无法用响应差异探测对方租户是否存在某个工单 ID。
func (s *TicketAssociationService) GetTicketDependencies(ctx context.Context, ticketID, tenantID int) (*TicketDependencies, error) {
	if _, err := s.getTicketForAssociation(ctx, ticketID, tenantID); err != nil {
		return nil, err
	}

	// 获取父工单链
	parentChain, err := s.getParentChain(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 获取子工单树
	childrenTree, err := s.getChildrenTree(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 获取关联工单
	relatedTickets, err := s.GetRelatedTickets(ctx, ticketID, tenantID)
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

// getParentChain 沿 parent_ticket_id 向上收集祖先工单，全部限定在 tenantID 内。
//
// 每一跳都按租户过滤：parent_ticket_id 只是普通 int 字段，没有外键保证它指向同租户的
// 工单，缺少谓词时这条链会把别的租户的祖先工单一并读出来。visited 与深度上限保证
// 脏数据里的父子环不会让请求挂死。
func (s *TicketAssociationService) getParentChain(ctx context.Context, ticketID, tenantID int) ([]*TicketResponse, error) {
	const maxDepth = 64

	chain := []*TicketResponse{}
	visited := map[int]struct{}{}
	currentID := ticketID

	for currentID != 0 && len(visited) < maxDepth {
		if _, seen := visited[currentID]; seen {
			break
		}
		visited[currentID] = struct{}{}

		ticketEntity, err := s.client.Ticket.Query().
			Where(ticket.ID(currentID), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			break
		}
		if ticketEntity.ParentTicketID == 0 {
			break
		}

		parent, err := s.client.Ticket.Query().
			Where(ticket.ID(ticketEntity.ParentTicketID), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			break
		}

		chain = append([]*TicketResponse{s.buildTicketResponse(parent)}, chain...)
		currentID = parent.ID
	}

	return chain, nil
}

// getChildrenTree 收集直接子工单。
//
// tenantID 谓词不可省：这里按 parent_ticket_id 匹配，修复前是**全库**扫描，所以只要
// 别的租户有一行的 parent_ticket_id 恰好等于本次的工单 ID，它就会被当作本租户的子工单
// 返回。关联表因此成为绕过租户隔离的第二条读取通道。
func (s *TicketAssociationService) getChildrenTree(ctx context.Context, ticketID, tenantID int) ([]*TicketResponse, error) {
	children, err := s.client.Ticket.Query().
		Where(
			ticket.ParentTicketID(ticketID),
			ticket.TenantIDEQ(tenantID),
			ticket.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	tree := []*TicketResponse{}
	for _, child := range children {
		// 只返回直接子工单并压平：更深的后代在它们各自的 childrenTree 里，
		// 在此递归会让同一工单在响应中重复出现。
		tree = append(tree, s.buildTicketResponse(child))
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
func (s *TicketAssociationService) AddConfigurationItem(ctx context.Context, ticketID, ciID, tenantID int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
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
func (s *TicketAssociationService) RemoveConfigurationItem(ctx context.Context, ticketID, ciID, tenantID int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
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
func (s *TicketAssociationService) GetConfigurationItems(ctx context.Context, ticketID, tenantID int) ([]*ConfigurationItemResponse, error) {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
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
func (s *TicketAssociationService) SetConfigurationItems(ctx context.Context, ticketID int, ciIDs []int, tenantID int) error {
	ticketEntity, err := s.getTicketForAssociation(ctx, ticketID, tenantID)
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
		if err := s.AddConfigurationItem(ctx, ticketID, ciID, tenantID); err != nil {
			return err
		}
	}

	return nil
}

// getTicketForAssociation 是本服务定位工单的唯一入口：任何一次读取或写入都必须带上
// 调用者租户，租户谓词缺失（tenantID<=0）时结果集自然为空并按「不存在」处理，
// 因此缺少租户上下文的调用是 fail-closed，而不是退化成全库可见。
//
// 跨租户与真实不存在必须返回同一个 404/4004 与同一句文案；此前这里把 ent 的错误包进
// fmt.Errorf("工单不存在") 再由 router 原样写进响应，既泄漏底层错误串，又让不存在的
// 工单返回 500/5001、与真实存在但无权的工单可被区分。
func (s *TicketAssociationService) getTicketForAssociation(ctx context.Context, ticketID, tenantID int) (*ent.Ticket, error) {
	ticketEntity, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewBusinessError(common.NotFoundCode, "工单不存在", "")
		}
		return nil, fmt.Errorf("获取工单失败: %w", err)
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
