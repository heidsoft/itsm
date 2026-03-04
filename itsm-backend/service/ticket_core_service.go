package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/user"
)

type TicketCoreService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketCoreService(client *ent.Client, logger *zap.SugaredLogger) *TicketCoreService {
	return &TicketCoreService{client: client, logger: logger}
}

func (s *TicketCoreService) CreateTicketBasic(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ent.Ticket, error) {
	if strings.TrimSpace(req.Description) == "" {
		return nil, fmt.Errorf("描述不能为空")
	}
	if strings.ToLower(strings.TrimSpace(req.Priority)) == "" {
		return nil, fmt.Errorf("优先级不能为空")
	}
	if err := s.validateRequester(ctx, req.RequesterID, tenantID); err != nil {
		return nil, fmt.Errorf("验证创建人失败: %w", err)
	}

	ticketNumber, err := s.generateTicketNumber(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("生成工单编号失败: %w", err)
	}

	ticketType := req.Type
	if ticketType == "" {
		ticketType = "ticket"
	}

	var categoryID *int
	if req.CategoryID != nil && *req.CategoryID > 0 {
		categoryID = req.CategoryID
	} else if req.Category != "" {
		if catID, err := s.findCategoryID(ctx, req.Category, tenantID); err == nil {
			categoryID = &catID
		} else {
			s.logger.Warnw("Category not found", "category", req.Category, "error", err)
		}
	}

	createBuilder := s.client.Ticket.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetType(ticketType).
		SetStatus("open").
		SetTicketNumber(ticketNumber).
		SetTenantID(tenantID).
		SetRequesterID(req.RequesterID)

	if categoryID != nil {
		createBuilder = createBuilder.SetCategoryID(*categoryID)
	}
	if req.AssigneeID > 0 {
		if err := s.validateAssignee(ctx, req.AssigneeID, tenantID); err != nil {
			return nil, fmt.Errorf("验证分配人失败: %w", err)
		}
		createBuilder = createBuilder.SetAssigneeID(req.AssigneeID)
	}
	if req.ParentTicketID != nil && *req.ParentTicketID > 0 {
		createBuilder = createBuilder.SetParentTicketID(*req.ParentTicketID)
	}
	if req.TemplateID != nil && *req.TemplateID > 0 {
		createBuilder = createBuilder.SetTemplateID(*req.TemplateID)
	}

	ticket, err := createBuilder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建失败: %w", err)
	}

	if len(req.TagIDs) > 0 {
		_, _ = ticket.Update().AddTagIDs(req.TagIDs...).Save(ctx)
	}

	return ticket, nil
}

func (s *TicketCoreService) GetTicket(ctx context.Context, ticketID int, tenantID int) (*ent.Ticket, error) {
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}
	return t, nil
}

func (s *TicketCoreService) ListTickets(ctx context.Context, req *dto.ListTicketsRequest, tenantID int) ([]*ent.Ticket, error) {
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))

	if req.Status != "" {
		query = query.Where(ticket.StatusEQ(req.Status))
	}
	if req.Priority != "" {
		query = query.Where(ticket.PriorityEQ(req.Priority))
	}
	if req.Type != "" {
		query = query.Where(ticket.TypeEQ(req.Type))
	}
	if req.AssigneeID != nil {
		query = query.Where(ticket.AssigneeID(*req.AssigneeID))
	}
	if req.RequesterID != nil {
		query = query.Where(ticket.RequesterID(*req.RequesterID))
	}
	if req.Keyword != "" {
		keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
		query = query.Where(ticket.Or(ticket.TitleContainsFold(keyword), ticket.DescriptionContainsFold(keyword)))
	}
	if req.DateFrom != nil {
		query = query.Where(ticket.CreatedAtGTE(*req.DateFrom))
	}
	if req.DateTo != nil {
		query = query.Where(ticket.CreatedAtLTE(*req.DateTo))
	}
	if req.ParentTicketID != nil {
		query = query.Where(ticket.ParentTicketID(*req.ParentTicketID))
	}

	// 分类筛选
	if req.CategoryID != nil && *req.CategoryID > 0 {
		query = query.Where(ticket.CategoryID(*req.CategoryID))
	}
	if req.Category != "" {
		// 根据分类名称查找分类ID
		cat, err := s.client.TicketCategory.Query().
			Where(ticketcategory.NameEQ(req.Category), ticketcategory.TenantID(tenantID)).
			Only(ctx)
		if err == nil {
			query = query.Where(ticket.CategoryID(cat.ID))
		} else {
			// 找不到分类，返回空结果
			query = query.Where(ticket.ID(-1))
		}
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	if req.SortBy == "created_at" || req.SortBy == "" {
		if req.SortOrder == "asc" {
			query = query.Order(ent.Asc(ticket.FieldCreatedAt))
		} else {
			query = query.Order(ent.Desc(ticket.FieldCreatedAt))
		}
	} else if req.SortBy == "priority" {
		if req.SortOrder == "asc" {
			query = query.Order(ent.Asc(ticket.FieldPriority))
		} else {
			query = query.Order(ent.Desc(ticket.FieldPriority))
		}
	}

	tickets, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	return tickets, nil
}

func (s *TicketCoreService) CountTickets(ctx context.Context, tenantID int, filters ...func(*ent.TicketQuery)) (int, error) {
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))
	for _, filter := range filters {
		filter(query)
	}
	count, err := query.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("统计失败: %w", err)
	}
	return count, nil
}

func (s *TicketCoreService) UpdateTicketBasic(ctx context.Context, ticketID int, req *dto.UpdateTicketRequest, tenantID int) (*ent.Ticket, error) {
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	update := s.client.Ticket.UpdateOne(t)

	if req.Title != "" {
		update = update.SetTitle(req.Title)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}
	if req.Priority != "" {
		update = update.SetPriority(req.Priority)
	}
	if req.Status != "" {
		update = update.SetStatus(req.Status)
	}
	if req.RequesterID > 0 {
		if err := s.validateRequester(ctx, req.RequesterID, tenantID); err != nil {
			return nil, fmt.Errorf("验证创建人失败: %w", err)
		}
		update = update.SetRequesterID(req.RequesterID)
	}
	if req.AssigneeID != 0 {
		if req.AssigneeID > 0 {
			if err := s.validateAssignee(ctx, req.AssigneeID, tenantID); err != nil {
				return nil, fmt.Errorf("验证分配人失败: %w", err)
			}
		}
		update = update.SetAssigneeID(req.AssigneeID)
	}

	ticket, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新失败: %w", err)
	}
	return ticket, nil
}

func (s *TicketCoreService) DeleteTicket(ctx context.Context, ticketID int, tenantID int) error {
	_, err := s.GetTicket(ctx, ticketID, tenantID)
	if err != nil {
		return err
	}
	err = s.client.Ticket.DeleteOneID(ticketID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

func (s *TicketCoreService) BatchDeleteTickets(ctx context.Context, ticketIDs []int, tenantID int) error {
	if len(ticketIDs) == 0 {
		return nil
	}
	for _, id := range ticketIDs {
		_, err := s.GetTicket(ctx, id, tenantID)
		if err != nil {
			return fmt.Errorf("工单%d不存在: %w", id, err)
		}
	}
	_, err := s.client.Ticket.Delete().
		Where(ticket.IDIn(ticketIDs...), ticket.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("批量删除失败: %w", err)
	}
	return nil
}

func (s *TicketCoreService) generateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	// 查询当月的工单数量
	count, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.CreatedAtGTE(time.Date(year, month, 1, 0, 0, 0, 0, time.Local)),
			ticket.CreatedAtLT(time.Date(year, month+1, 1, 0, 0, 0, 0, time.Local)),
		).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Count failed", "error", err)
		return "", fmt.Errorf("查询失败: %w", err)
	}

	// 格式: TKT-YYYYMM-NNNNNN
	return fmt.Sprintf("TKT-%04d%02d-%06d", year, month, count+1), nil
}

func (s *TicketCoreService) validateRequester(ctx context.Context, userID, tenantID int) error {
	if userID <= 0 {
		return fmt.Errorf("无效用户ID")
	}
	exists, err := s.client.User.Query().
		Where(user.ID(userID), user.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("用户不存在")
	}
	return nil
}

func (s *TicketCoreService) validateAssignee(ctx context.Context, userID, tenantID int) error {
	if userID <= 0 {
		return fmt.Errorf("无效用户ID")
	}
	exists, err := s.client.User.Query().
		Where(user.ID(userID), user.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("用户不存在")
	}
	return nil
}

func (s *TicketCoreService) findCategoryID(ctx context.Context, name string, tenantID int) (int, error) {
	cat, err := s.client.TicketCategory.Query().
		Where(ticketcategory.NameEQ(name), ticketcategory.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return 0, fmt.Errorf("未找到: %w", err)
	}
	return cat.ID, nil
}

func (s *TicketCoreService) findDefaultCategory(ctx context.Context, tenantID int) (*ent.TicketCategory, error) {
	cat, err := s.client.TicketCategory.Query().
		Where(ticketcategory.IsActive(true), ticketcategory.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("未找到: %w", err)
	}
	return cat, nil
}

func (s *TicketCoreService) addTagsToTicket(ctx context.Context, t *ent.Ticket, tagIDs []int) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// 先清除现有标签，再添加新的，避免重复连接错误
	_, err := t.Update().ClearTags().AddTagIDs(tagIDs...).Save(ctx)
	return err
}
