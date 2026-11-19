package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ciattributedefinition"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// CITypeService CI类型服务
type CITypeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
	// rawDB 用于绕过 Ent 读拦截器统计**含已退役**的 CI 引用。
	// Ent 的 registerSoftDeleteInterceptor 会对每个 ConfigurationItemQuery 附加
	// lifecycle_status != 'scrapped'，而 configuration_items.ci_type_id 是
	// Required 外键、不可置空——退役 CI 仍然物理占用该类型。只走 Ent 预检会
	// 把「有退役引用」误判为「可删」，随后撞库外键 23503，用户只看到 500。
	rawDB *sql.DB
}

// NewCITypeService 创建CI类型服务
func NewCITypeService(client *ent.Client, logger *zap.SugaredLogger) *CITypeService {
	return &CITypeService{
		client: client,
		logger: logger,
	}
}

// SetRawDB 注入原生数据库连接，用于带退役 CI 的类型引用统计。
// 未注入时退化为 Ent 查询（看不到退役行，仍靠 23503 兜底给出 409）。
func (s *CITypeService) SetRawDB(db *sql.DB) {
	s.rawDB = db
}

// CreateCIType 创建CI类型
func (s *CITypeService) CreateCIType(ctx context.Context, req *dto.CreateCITypeRequest, tenantID int) (*dto.CITypeResponse, error) {
	if err := s.validateParent(ctx, 0, req.ParentTypeID, tenantID); err != nil {
		return nil, err
	}

	create := s.client.CIType.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetIcon(req.Icon).
		SetColor(req.Color).
		SetAttributeSchema(req.AttributeSchema).
		SetTenantID(tenantID)
	if req.ParentTypeID != nil {
		create.SetParentTypeID(*req.ParentTypeID)
	}
	ciType, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI type", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("failed to create CI type: %w", err)
	}

	s.logger.Infow("CI type created successfully", "ci_type_id", ciType.ID, "tenant_id", tenantID, "name", ciType.Name)
	return dto.ToCITypeResponse(ciType), nil
}

// GetCITypeByID 根据ID获取CI类型
func (s *CITypeService) GetCITypeByID(ctx context.Context, id, tenantID int) (*dto.CITypeResponse, error) {
	ciType, err := s.client.CIType.Query().
		Where(citype.IDEQ(id), citype.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", id)
		return nil, fmt.Errorf("failed to get CI type: %w", err)
	}

	return dto.ToCITypeResponse(ciType), nil
}

// ListCITypes 获取CI类型列表
func (s *CITypeService) ListCITypes(ctx context.Context, tenantID int, page, pageSize int, search string) (*dto.CITypeListResponse, error) {
	// HTTP 入口的页长夹紧归 common.GetPaginationFromQuery；这一层只兜住绕过 HTTP 的调用方
	// （如 /cmdb/ontology），否则 page<=0 会算出负 OFFSET、pageSize=0 在 Ent 的 sqlgraph 里
	// 等于不加 LIMIT 而返回整表。
	page, pageSize = common.ValidatePagination(page, pageSize)

	query := s.client.CIType.Query().Where(citype.TenantIDEQ(tenantID), citype.IsActiveEQ(true))

	if search != "" {
		query = query.Where(citype.NameContains(search))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI types", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count CI types: %w", err)
	}

	ciTypes, err := query.
		Offset((page-1)*pageSize).
		Limit(pageSize).
		// created_at 非唯一列，补 ID 并列键，否则同秒写入的类型在翻页时可能重复或漏出。
		Order(ent.Desc(citype.FieldCreatedAt), ent.Asc(citype.FieldID)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI types", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list CI types: %w", err)
	}

	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return &dto.CITypeListResponse{
		Items:      dto.ToCITypeResponseList(ciTypes),
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// UpdateCIType 更新CI类型
func (s *CITypeService) UpdateCIType(ctx context.Context, id, tenantID int, req *dto.UpdateCITypeRequest) (*dto.CITypeResponse, error) {
	if err := s.validateParent(ctx, id, req.ParentTypeID, tenantID); err != nil {
		return nil, err
	}

	update := s.client.CIType.UpdateOneID(id).
		Where(citype.TenantIDEQ(tenantID))

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Icon != nil {
		update.SetIcon(*req.Icon)
	}
	if req.Color != nil {
		update.SetColor(*req.Color)
	}
	if req.AttributeSchema != nil {
		update.SetAttributeSchema(*req.AttributeSchema)
	}
	if req.ClearParent {
		update.ClearParentTypeID()
	} else if req.ParentTypeID != nil {
		update.SetParentTypeID(*req.ParentTypeID)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	ciType, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI type", "error", err, "ci_type_id", id)
		return nil, fmt.Errorf("failed to update CI type: %w", err)
	}

	s.logger.Infow("CI type updated successfully", "ci_type_id", ciType.ID, "tenant_id", tenantID)
	return dto.ToCITypeResponse(ciType), nil
}

func (s *CITypeService) validateParent(ctx context.Context, currentID int, parentID *int, tenantID int) error {
	if parentID == nil {
		return nil
	}
	if *parentID == currentID {
		return fmt.Errorf("CI type cannot inherit from itself")
	}

	parent, err := s.client.CIType.Query().
		Where(citype.IDEQ(*parentID), citype.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("parent CI type not found")
		}
		return fmt.Errorf("failed to validate parent CI type: %w", err)
	}

	visited := map[int]struct{}{currentID: {}}
	for parent.ParentTypeID != nil {
		if _, exists := visited[parent.ID]; exists {
			return fmt.Errorf("CI type inheritance cycle detected")
		}
		visited[parent.ID] = struct{}{}
		parent, err = s.client.CIType.Query().
			Where(citype.IDEQ(*parent.ParentTypeID), citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			return fmt.Errorf("failed to validate CI type inheritance chain: %w", err)
		}
	}
	if _, exists := visited[parent.ID]; exists {
		return fmt.Errorf("CI type inheritance cycle detected")
	}
	return nil
}

// DeleteCIType 删除CI类型
//
// 三条阻塞条件都必须给出 409 而不是 500：类型被占用是**可预期的业务结果**，
// 客户端要据此提示用户「先处理哪些引用」，而不是看到「删除CI类型失败」。
func (s *CITypeService) DeleteCIType(ctx context.Context, id, tenantID int) error {
	// 检查是否有关联的CI。必须把已退役（scrapped）的 CI 也算进来：ci_type_id
	// 是 Required 外键不可置空，退役 CI 仍然物理占用该类型，数据库 DELETE
	// 会以 23503 拒绝。若这里只看在线 CI，预检放行后必然在最后一步失败。
	count, err := s.countCIsReferencingType(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to check CI type usage", "error", err, "ci_type_id", id)
		return fmt.Errorf("failed to check CI type usage: %w", err)
	}

	if count > 0 {
		return common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("该CI类型仍被 %d 个配置项引用（含已退役），无法删除；请先迁移或彻底清理这些配置项", count),
			fmt.Sprintf("ci_type_id=%d,ci_count=%d", id, count))
	}
	childCount, err := s.client.CIType.Query().
		Where(citype.ParentTypeIDEQ(id), citype.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check child CI types: %w", err)
	}
	if childCount > 0 {
		return common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("该CI类型仍有 %d 个子类型，无法删除；请先删除子类型", childCount),
			fmt.Sprintf("ci_type_id=%d,child_count=%d", id, childCount))
	}
	attributeCount, err := s.client.CIAttributeDefinition.Query().
		Where(
			ciattributedefinition.CiTypeIDEQ(id),
			ciattributedefinition.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check CI attribute definitions: %w", err)
	}
	if attributeCount > 0 {
		return common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("该CI类型仍有 %d 个属性定义，无法删除；请先清理属性定义", attributeCount),
			fmt.Sprintf("ci_type_id=%d,attribute_count=%d", id, attributeCount))
	}

	err = s.client.CIType.DeleteOneID(id).
		Where(citype.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		// 兜底：并发写入的 CI 可能绕过上面的预检，最终仍由数据库外键拦截。
		// 把它归类为 409 冲突，避免「类型被占用」表现为 5xx 毛刺。
		if isForeignKeyViolation(err) {
			s.logger.Warnw("CI type delete blocked by foreign key", "error", err, "ci_type_id", id)
			return common.NewBusinessError(common.ConflictCode,
				"该CI类型仍被配置项引用，无法删除；请先迁移或彻底清理这些配置项",
				fmt.Sprintf("ci_type_id=%d", id))
		}
		s.logger.Errorw("Failed to delete CI type", "error", err, "ci_type_id", id)
		return fmt.Errorf("failed to delete CI type: %w", err)
	}

	s.logger.Infow("CI type deleted successfully", "ci_type_id", id, "tenant_id", tenantID)
	return nil
}

// countCIsReferencingType 统计引用该类型的配置项数量，**包含已退役**的 CI。
//
// 为什么不能用 Ent：registerSoftDeleteInterceptor（database/security.go）会给
// 每个 ConfigurationItemQuery 附加 lifecycle_status != 'scrapped'。删除类型的
// 判定依据是「物理外键是否被占用」，与读面可见性无关，两套口径不能混用。
func (s *CITypeService) countCIsReferencingType(ctx context.Context, ciTypeID, tenantID int) (int, error) {
	if s.rawDB == nil {
		// 未注入原生连接（CLI/单测）：退化为 Ent 查询。此时看不到退役行，
		// 只可能是「少算」，最终由 DeleteOneID 的 23503 兜底返回 409。
		return s.client.ConfigurationItem.Query().
			Where(configurationitem.CiTypeIDEQ(ciTypeID), configurationitem.TenantIDEQ(tenantID)).
			Count(ctx)
	}
	var count int
	err := s.rawDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM configuration_items WHERE ci_type_id = $1 AND tenant_id = $2`,
		ciTypeID, tenantID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// isForeignKeyViolation 判断驱动层报错是否为外键约束冲突。
//
// 只做字符串匹配、不复用 ent.IsConstraintError：后者按 *ConstraintError 类型判定，
// 会把唯一约束（23505）也算成约束错误，而这里只关心「父行被子行引用」这一类。
func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, pattern := range []string{
		"23503",                           // Postgres 外键违反
		"violates foreign key constraint", // Postgres 文案
		"FOREIGN KEY constraint failed",   // SQLite
		"Error 1451",                      // MySQL 删除父行被引用
	} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
