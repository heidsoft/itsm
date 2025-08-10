package service

import (
	"context"
	"errors"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/ticket"
	"time"
)

// TicketCategoryService 工单分类服务
type TicketCategoryService struct {
	client *ent.Client
}

// NewTicketCategoryService 创建工单分类服务实例
func NewTicketCategoryService(client *ent.Client) *TicketCategoryService {
	return &TicketCategoryService{client: client}
}

// CreateCategory 创建工单分类
func (s *TicketCategoryService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*ent.TicketCategory, error) {
	// 验证分类代码唯一性
	exists, err := s.client.TicketCategory.Query().
		Where(ticketcategory.CodeEQ(req.Code)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("分类代码已存在")
	}

	// 验证父分类
	var level int = 1
	if req.ParentID > 0 {
		parent, err := s.client.TicketCategory.Get(ctx, req.ParentID)
		if err != nil {
			return nil, errors.New("父分类不存在")
		}
		level = parent.Level + 1
	}

	category, err := s.client.TicketCategory.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCode(req.Code).
		SetParentID(req.ParentID).
		SetLevel(level).
		SetSortOrder(req.SortOrder).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return category, nil
}

// GetCategory 获取工单分类
func (s *TicketCategoryService) GetCategory(ctx context.Context, id int) (*ent.TicketCategory, error) {
	return s.client.TicketCategory.Get(ctx, id)
}

// ListCategories 获取工单分类列表
func (s *TicketCategoryService) ListCategories(ctx context.Context, req *ListCategoriesRequest) ([]*ent.TicketCategory, int, error) {
	query := s.client.TicketCategory.Query()

	// 应用过滤条件
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			// 获取顶级分类
			query = query.Where(ticketcategory.ParentIDIsNil())
		} else {
			query = query.Where(ticketcategory.ParentIDEQ(*req.ParentID))
		}
	}
	if req.Level > 0 {
		query = query.Where(ticketcategory.LevelEQ(req.Level))
	}
	if req.IsActive != nil {
		query = query.Where(ticketcategory.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(ticketcategory.TenantID(req.TenantID))
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

	// 默认按排序顺序和名称排序
	query = query.Order(ent.Asc(ticketcategory.FieldSortOrder), ent.Asc(ticketcategory.FieldName))

	categories, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// GetCategoryTree 获取分类树结构
func (s *TicketCategoryService) GetCategoryTree(ctx context.Context, tenantID int) ([]*CategoryTreeItem, error) {
	// 获取所有分类
	categories, err := s.client.TicketCategory.Query().
		Where(ticketcategory.TenantID(tenantID)).
		Where(ticketcategory.IsActive(true)).
		Order(ent.Asc(ticketcategory.FieldSortOrder), ent.Asc(ticketcategory.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 构建分类树
	categoryMap := make(map[int]*CategoryTreeItem)
	var rootCategories []*CategoryTreeItem

	for _, category := range categories {
		item := &CategoryTreeItem{
			ID:          category.ID,
			Name:        category.Name,
			Description: category.Description,
			Code:        category.Code,
			Level:       category.Level,
			SortOrder:   category.SortOrder,
			IsActive:    category.IsActive,
			Children:    []*CategoryTreeItem{},
		}
		categoryMap[category.ID] = item

		if category.ParentID == 0 {
			rootCategories = append(rootCategories, item)
		} else {
			if parent, exists := categoryMap[category.ParentID]; exists {
				parent.Children = append(parent.Children, item)
			}
		}
	}

	return rootCategories, nil
}

// UpdateCategory 更新工单分类
func (s *TicketCategoryService) UpdateCategory(ctx context.Context, id int, req *UpdateCategoryRequest) (*ent.TicketCategory, error) {
	update := s.client.TicketCategory.UpdateOneID(id)

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Code != "" {
		// 验证分类代码唯一性
		exists, err := s.client.TicketCategory.Query().
			Where(ticketcategory.CodeEQ(req.Code)).
			Where(ticketcategory.IDNEQ(id)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("分类代码已存在")
		}
		update.SetCode(req.Code)
	}
	if req.ParentID != nil {
		if *req.ParentID > 0 {
			// 验证父分类
			parent, err := s.client.TicketCategory.Get(ctx, *req.ParentID)
			if err != nil {
				return nil, errors.New("父分类不存在")
			}
			update.SetParentID(*req.ParentID)
			update.SetLevel(parent.Level + 1)
		} else {
			update.SetParentID(0)
			update.SetLevel(1)
		}
	}
	if req.SortOrder != nil {
		update.SetSortOrder(*req.SortOrder)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	update.SetUpdatedAt(time.Now())

	return update.Save(ctx)
}

// DeleteCategory 删除工单分类
func (s *TicketCategoryService) DeleteCategory(ctx context.Context, id int) error {
	// 检查是否有子分类
	childrenCount, err := s.client.TicketCategory.Query().
		Where(ticketcategory.ParentIDEQ(id)).
		Count(ctx)
	if err != nil {
		return err
	}
	if childrenCount > 0 {
		return errors.New("无法删除有子分类的分类")
	}

	// 检查是否有工单使用此分类
	ticketsCount, err := s.client.Ticket.Query().
		Where(ticket.CategoryIDEQ(id)).
		Count(ctx)
	if err != nil {
		return err
	}
	if ticketsCount > 0 {
		return errors.New("无法删除正在使用的分类")
	}

	return s.client.TicketCategory.DeleteOneID(id).Exec(ctx)
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Code        string `json:"code" binding:"required"`
	ParentID    int    `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
	TenantID    int    `json:"tenant_id" binding:"required"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	ParentID    *int   `json:"parent_id"`
	SortOrder   *int   `json:"sort_order"`
	IsActive    *bool  `json:"is_active"`
}

// ListCategoriesRequest 获取分类列表请求
type ListCategoriesRequest struct {
	Page      int  `json:"page" form:"page"`
	PageSize  int  `json:"page_size" form:"page_size"`
	ParentID  *int `json:"parent_id" form:"parent_id"`
	Level     int  `json:"level" form:"level"`
	IsActive  *bool `json:"is_active" form:"is_active"`
	TenantID  int  `json:"tenant_id" form:"tenant_id"`
}

// CategoryTreeItem 分类树项目
type CategoryTreeItem struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Code        string             `json:"code"`
	Level       int                `json:"level"`
	SortOrder   int                `json:"sort_order"`
	IsActive    bool               `json:"is_active"`
	Children    []*CategoryTreeItem `json:"children"`
}
