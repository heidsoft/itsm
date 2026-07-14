package ticket

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/repository/base"

	"go.uber.org/zap"
)

// SequenceProvider 工单号生成接口（避免循环依赖）
type SequenceProvider interface {
	GetNextSequenceWithExpiry(ctx context.Context, key string, expiredAt time.Time) (int64, error)
}

// sequenceServiceAdapter SequenceService 适配器
type sequenceServiceAdapter struct {
	logger *zap.SugaredLogger
	client *ent.Client
}

// NewSequenceServiceAdapter 创建适配器
func NewSequenceServiceAdapter(logger *zap.SugaredLogger, client *ent.Client) *sequenceServiceAdapter {
	return &sequenceServiceAdapter{logger: logger, client: client}
}

// EntRepository Ent 实现的工单仓储
type EntRepository struct {
	*base.EntRepository
	logger          *zap.SugaredLogger
	sequenceService SequenceProvider
	rawDB           *sql.DB // for transactional SELECT FOR UPDATE
}

// NewEntRepository 创建 Ent 工单仓储
func NewEntRepository(client *ent.Client, logger *zap.SugaredLogger) *EntRepository {
	return &EntRepository{
		EntRepository: base.NewEntRepository(client),
		logger:        logger,
	}
}

// SetSequenceService 设置序列服务（用于 Redis 工单号生成）
func (r *EntRepository) SetSequenceService(seqSvc SequenceProvider) {
	// A typed nil pointer stored in an interface is not equal to nil. Redis
	// initialization returns (*SequenceService)(nil) when unavailable; without
	// this guard the repository attempts to call it and panics instead of using
	// the database sequence fallback.
	if seqSvc == nil || (reflect.ValueOf(seqSvc).Kind() == reflect.Ptr && reflect.ValueOf(seqSvc).IsNil()) {
		r.sequenceService = nil
		return
	}
	r.sequenceService = seqSvc
}

// SetRawDB 设置原生数据库连接（用于事务性编号生成）
func (r *EntRepository) SetRawDB(db *sql.DB) {
	r.rawDB = db
}

// Create 创建工单
func (r *EntRepository) Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error) {
	for attempt := 0; attempt < 3; attempt++ {
		ticketNumber, err := r.GenerateTicketNumber(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("generate ticket number: %w", err)
		}

		builder := r.Client().Ticket.Create().
			SetTitle(params.Title).
			SetDescription(params.Description).
			SetType(string(params.Type)).
			SetPriority(string(params.Priority)).
			SetTicketNumber(ticketNumber).
			SetRequesterID(params.RequesterID).
			SetTenantID(tenantID).
			SetStatus(string(StatusNew))

		if params.AssigneeID != nil {
			builder.SetAssigneeID(*params.AssigneeID)
		}
		if params.CategoryID != nil {
			builder.SetCategoryID(*params.CategoryID)
		}

		entity, err := builder.Save(ctx)
		if err == nil {
			return toDomainModel(entity), nil
		}

		if ent.IsConstraintError(err) || strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			r.logger.Warnw("ticket number collision detected during create, retrying",
				"ticket_number", ticketNumber,
				"tenant_id", tenantID,
				"attempt", attempt+1,
				"error", err)
			continue
		}

		return nil, fmt.Errorf("create ticket: %w", err)
	}

	return nil, fmt.Errorf("create ticket: ticket number collision persisted after retries")
}

// GetByID 根据 ID 获取工单
func (r *EntRepository) GetByID(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	entity, err := r.Client().Ticket.Query().
		Where(
			ticket.ID(id),
			ticket.TenantID(tenantID),
			ticket.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found: %w", err)
		}
		return nil, fmt.Errorf("get ticket: %w", err)
	}

	return toDomainModel(entity), nil
}

// GetByNumber 根据工单编号获取工单
func (r *EntRepository) GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error) {
	entity, err := r.Client().Ticket.Query().
		Where(
			ticket.TicketNumber(ticketNumber),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found: %w", err)
		}
		return nil, fmt.Errorf("get ticket by number: %w", err)
	}

	return toDomainModel(entity), nil
}

// Update 更新工单
func (r *EntRepository) Update(ctx context.Context, id int, params *UpdateParams, tenantID int) (*Ticket, error) {
	// 先获取当前工单（包含版本号）
	current, err := r.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 乐观锁检查
	if current.Version != params.Version {
		return nil, fmt.Errorf("version conflict: expected %d, got %d", current.Version, params.Version)
	}

	builder := r.Client().Ticket.UpdateOneID(id).
		SetVersion(current.Version + 1) // 版本号递增

	if params.Title != nil {
		builder.SetTitle(*params.Title)
	}
	if params.Description != nil {
		builder.SetDescription(*params.Description)
	}
	if params.Status != nil {
		builder.SetStatus(string(*params.Status))
	}
	if params.Priority != nil {
		builder.SetPriority(string(*params.Priority))
	}
	if params.AssigneeID != nil {
		builder.SetAssigneeID(*params.AssigneeID)
	}
	if params.Resolution != nil {
		builder.SetResolution(*params.Resolution)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update ticket: %w", err)
	}

	return toDomainModel(entity), nil
}

// Delete 删除工单
func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	_, err := r.Client().Ticket.Delete().
		Where(
			ticket.ID(id),
			ticket.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete ticket: %w", err)
	}
	return nil
}

// List 列表查询工单
func (r *EntRepository) List(ctx context.Context, tenantID int, filters *FilterParams, pagination *base.QueryParams) (*base.ListResult[Ticket], error) {
	query := r.Client().Ticket.Query().
		Where(ticket.TenantID(tenantID))

	// 应用过滤条件
	if filters != nil {
		if filters.Status != nil {
			query = query.Where(ticket.StatusEQ(string(*filters.Status)))
		}
		if filters.Priority != nil {
			query = query.Where(ticket.PriorityEQ(string(*filters.Priority)))
		}
		if filters.Type != nil {
			query = query.Where(ticket.TypeEQ(string(*filters.Type)))
		}
		if filters.RequesterID != nil {
			query = query.Where(ticket.RequesterID(*filters.RequesterID))
		}
		if filters.AssigneeID != nil {
			query = query.Where(ticket.AssigneeID(*filters.AssigneeID))
		}
		if filters.CategoryID != nil {
			query = query.Where(ticket.CategoryID(*filters.CategoryID))
		}
		if filters.Keyword != "" {
			query = query.Where(ticket.Or(
				ticket.TitleContains(filters.Keyword),
				ticket.DescriptionContains(filters.Keyword),
			))
		}
		if filters.DateFrom != nil {
			query = query.Where(ticket.CreatedAtGTE(*filters.DateFrom))
		}
		if filters.DateTo != nil {
			query = query.Where(ticket.CreatedAtLTE(*filters.DateTo))
		}
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count tickets: %w", err)
	}

	// 排序
	if pagination != nil && pagination.OrderBy != "" {
		orderField := toEntField(pagination.OrderBy)
		if pagination.OrderDir == "asc" {
			query = query.Order(ent.Asc(orderField))
		} else {
			query = query.Order(ent.Desc(orderField))
		}
	} else {
		query = query.Order(ent.Desc(ticket.FieldCreatedAt))
	}

	// 分页
	if pagination != nil {
		query = query.Offset(pagination.CalculateOffset()).Limit(pagination.GetLimit())
	}

	entities, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	result := &base.ListResult[Ticket]{
		Data:  make([]*Ticket, len(entities)),
		Total: total,
	}

	for i, e := range entities {
		result.Data[i] = toDomainModel(e)
	}

	return result, nil
}

// BatchDelete 批量删除工单
func (r *EntRepository) BatchDelete(ctx context.Context, ids []int, tenantID int) error {
	_, err := r.Client().Ticket.Delete().
		Where(
			ticket.IDIn(ids...),
			ticket.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("batch delete tickets: %w", err)
	}
	return nil
}

// Exists 检查工单是否存在
func (r *EntRepository) Exists(ctx context.Context, id int, tenantID int) (bool, error) {
	return r.Client().Ticket.Query().
		Where(
			ticket.ID(id),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
}

// FindByAssignee 根据处理人查询工单
func (r *EntRepository) FindByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error) {
	entities, err := r.Client().Ticket.Query().
		Where(
			ticket.AssigneeID(assigneeID),
			ticket.TenantID(tenantID),
			ticket.StatusNotIn(string(StatusClosed), string(StatusCancelled)),
		).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find tickets by assignee: %w", err)
	}

	return toDomainModels(entities), nil
}

// FindByRequester 根据申请人查询工单
func (r *EntRepository) FindByRequester(ctx context.Context, requesterID int, tenantID int) ([]*Ticket, error) {
	entities, err := r.Client().Ticket.Query().
		Where(
			ticket.RequesterID(requesterID),
			ticket.TenantID(tenantID),
		).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find tickets by requester: %w", err)
	}

	return toDomainModels(entities), nil
}

// FindOverdue 查询逾期工单
func (r *EntRepository) FindOverdue(ctx context.Context, tenantID int) ([]*Ticket, error) {
	now := time.Now()
	entities, err := r.Client().Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusNotIn(string(StatusClosed), string(StatusCancelled), string(StatusResolved)),
			ticket.SLAResolutionDeadlineLT(now),
		).
		Order(ent.Asc(ticket.FieldSLAResolutionDeadline)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("find overdue tickets: %w", err)
	}

	return toDomainModels(entities), nil
}

// CountByStatus 按状态统计工单数量
func (r *EntRepository) CountByStatus(ctx context.Context, tenantID int) (map[Status]int, error) {
	// 使用 Ent 的聚合功能
	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	var results []statusCount
	err := r.Client().Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		GroupBy(ticket.FieldStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &results)
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}

	counts := make(map[Status]int)
	for _, r := range results {
		counts[Status(r.Status)] = r.Count
	}

	return counts, nil
}

// CountByPriority 按优先级统计工单数量
func (r *EntRepository) CountByPriority(ctx context.Context, tenantID int) (map[Priority]int, error) {
	type priorityCount struct {
		Priority string `json:"priority"`
		Count    int    `json:"count"`
	}

	var results []priorityCount
	err := r.Client().Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		GroupBy(ticket.FieldPriority).
		Aggregate(ent.Count()).
		Scan(ctx, &results)
	if err != nil {
		return nil, fmt.Errorf("count by priority: %w", err)
	}

	counts := make(map[Priority]int)
	for _, r := range results {
		counts[Priority(r.Priority)] = r.Count
	}

	return counts, nil
}

// GenerateTicketNumber 生成工单编号
// 格式: TKT-YYYYMM-XXXXXX
// 优先使用 Redis 序列服务（原子递增，避免并发重复）；否则使用数据库回退
func (r *EntRepository) GenerateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 计算本月最后一天作为过期时间
	expiredAt := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)

	// 优先使用 Redis 序列服务
	if r.sequenceService != nil {
		return r.generateTicketNumberWithRedis(ctx, tenantID, year, month, expiredAt)
	}

	// 备用方案：数据库查询
	return r.generateTicketNumberWithDB(ctx, tenantID, year, month)
}

// generateTicketNumberWithRedis 使用 Redis INCR 生成工单编号
func (r *EntRepository) generateTicketNumberWithRedis(ctx context.Context, tenantID, year, month int, expiredAt time.Time) (string, error) {
	key := fmt.Sprintf("sequence:ticket:%d:%d%02d", tenantID, year, month)

	// 获取序列号（带过期时间）
	seq, err := r.sequenceService.GetNextSequenceWithExpiry(ctx, key, expiredAt)
	if err != nil {
		r.logger.Warnw("Redis sequence failed for ticket, fallback to DB", "error", err)
		return r.generateTicketNumberWithDB(ctx, tenantID, year, month)
	}

	return fmt.Sprintf("TKT-%04d%02d-%06d", year, month, seq), nil
}

// generateTicketNumberWithDB 使用数据库事务+SELECT FOR UPDATE NOWAIT 生成工单编号（备用方案）
// 重试机制（最多3次）解决并发竞态：当编号已存在时重新查询并生成
func (r *EntRepository) generateTicketNumberWithDB(ctx context.Context, tenantID int, year, month int) (string, error) {
	prefix := fmt.Sprintf("TKT-%04d%02d-", year, month)

	for attempt := 0; attempt < 3; attempt++ {
		var candidate string

		// 路径1：有 rawDB，优先使用事务 + NOWAIT
		if r.rawDB != nil {
			tx, err := r.rawDB.BeginTx(ctx, nil)
			if err != nil {
				r.logger.Warnw("BeginTx failed, trying Ent fallback", "error", err, "attempt", attempt+1)
				// fall through to Ent fallback below
			} else {
				// FOR UPDATE NOWAIT：立即失败而非跳过锁（快速感知冲突）
				query := `SELECT ticket_number FROM tickets WHERE tenant_id = $1 AND ticket_number LIKE $2 AND ticket_number IS NOT NULL AND ticket_number != '' ORDER BY ticket_number DESC LIMIT 1 FOR UPDATE NOWAIT`
				var maxTicketNum string
				err = tx.QueryRowContext(ctx, query, tenantID, prefix+"%").Scan(&maxTicketNum)

				var seq int = 0
				if err == nil && maxTicketNum != "" {
					if idx := strings.LastIndex(maxTicketNum, "-"); idx >= 0 {
						fmt.Sscanf(maxTicketNum[idx+1:], "%d", &seq)
					}
				} else if err == sql.ErrNoRows {
					r.logger.Infow("No existing tickets this month, starting from seed", "tenant", tenantID, "year", year, "month", month)
				} else if err != nil {
					tx.Rollback()
					r.logger.Warnw("NOWAIT query failed, trying Ent fallback", "error", err, "attempt", attempt+1)
					// fall through to Ent fallback
				}

				if seq > 0 {
					candidate = fmt.Sprintf("TKT-%04d%02d-%06d", year, month, seq+1)
				} else {
					candidate = fmt.Sprintf("TKT-%04d%02d-%06d", year, month, 1)
				}

				r.logger.Infow("DB transaction generated ticket number",
					"number", candidate, "tenant", tenantID, "attempt", attempt+1)

				// 提交事务释放锁
				if err := tx.Commit(); err != nil {
					r.logger.Warnw("tx commit failed, retrying", "error", err, "attempt", attempt+1)
					continue
				}

				// 双重保险：验证编号是否真的唯一
				checkQuery := `SELECT COUNT(*) FROM tickets WHERE ticket_number = $1 AND tenant_id = $2`
				var count int
				if checkErr := r.rawDB.QueryRowContext(ctx, checkQuery, candidate, tenantID).Scan(&count); checkErr == nil && count > 0 {
					r.logger.Warnw("Ticket number collision detected, retrying", "number", candidate, "attempt", attempt+1)
					continue
				}

				return candidate, nil
			}
		}

		// 路径2：Ent ORM fallback（没有 rawDB 或 rawDB 路径失败）
		tickets, err := r.Client().Ticket.Query().
			Where(
				ticket.TenantID(tenantID),
				ticket.TicketNumberContains(prefix[:len(prefix)-1]),
			).
			Order(ent.Desc(ticket.FieldTicketNumber)).
			Limit(1).
			All(ctx)

		var seq int
		if err != nil || len(tickets) == 0 {
			seq = 1
		} else {
			maxNum := tickets[0].TicketNumber
			if idx := strings.LastIndex(maxNum, "-"); idx >= 0 {
				fmt.Sscanf(maxNum[idx+1:], "%d", &seq)
				seq++
			} else {
				seq = 1
			}
		}

		candidate = fmt.Sprintf("TKT-%04d%02d-%06d", year, month, seq)
		r.logger.Infow("Ent fallback generated ticket number",
			"number", candidate, "tenant", tenantID, "attempt", attempt+1)

		// 如果有 rawDB，再验证一次（双重保险）
		if r.rawDB != nil {
			checkQuery := `SELECT COUNT(*) FROM tickets WHERE ticket_number = $1 AND tenant_id = $2`
			var count int
			if checkErr := r.rawDB.QueryRowContext(ctx, checkQuery, candidate, tenantID).Scan(&count); checkErr == nil && count > 0 {
				r.logger.Warnw("Ent fallback ticket number collision, retrying", "number", candidate, "attempt", attempt+1)
				continue
			}
		}

		return candidate, nil
	}

	return "", fmt.Errorf("failed to generate unique ticket number after 3 attempts")
}

// uniqueFallbackSuffix 生成唯一后缀（用于当月第一条记录的回退）
func uniqueFallbackSuffix() string {
	// 使用时间戳+随机数生成唯一后缀
	n := time.Now().UnixNano()
	return fmt.Sprintf("%010d", n)[2:]
}

// UpdateStatus 更新工单状态
func (r *EntRepository) UpdateStatus(ctx context.Context, id int, status Status, tenantID int) (*Ticket, error) {
	entity, err := r.Client().Ticket.UpdateOneID(id).
		Where(ticket.TenantID(tenantID)).
		SetStatus(string(status)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update ticket status: %w", err)
	}

	return toDomainModel(entity), nil
}

// AssignTicket 分配工单
func (r *EntRepository) AssignTicket(ctx context.Context, id int, assigneeID int, tenantID int) (*Ticket, error) {
	entity, err := r.Client().Ticket.UpdateOneID(id).
		Where(ticket.TenantID(tenantID)).
		SetAssigneeID(assigneeID).
		SetStatus(string(StatusOpen)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("assign ticket: %w", err)
	}

	return toDomainModel(entity), nil
}

// UpdateSLADeadlines 更新 SLA 截止时间
func (r *EntRepository) UpdateSLADeadlines(ctx context.Context, id int, responseDeadline, resolutionDeadline *time.Time, slaDefinitionID *int, tenantID int) error {
	builder := r.Client().Ticket.UpdateOneID(id).
		Where(ticket.TenantID(tenantID))

	if responseDeadline != nil {
		builder.SetSLAResponseDeadline(*responseDeadline)
	}
	if resolutionDeadline != nil {
		builder.SetSLAResolutionDeadline(*resolutionDeadline)
	}
	if slaDefinitionID != nil {
		builder.SetSLADefinitionID(*slaDefinitionID)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("update sla deadlines: %w", err)
	}

	return nil
}

// MarkFirstResponse 标记首次响应
func (r *EntRepository) MarkFirstResponse(ctx context.Context, id int, tenantID int) error {
	now := time.Now()
	_, err := r.Client().Ticket.UpdateOneID(id).
		Where(ticket.TenantID(tenantID)).
		SetFirstResponseAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark first response: %w", err)
	}
	return nil
}

// GetVersion 获取工单版本号
func (r *EntRepository) GetVersion(ctx context.Context, id int, tenantID int) (int, error) {
	entity, err := r.Client().Ticket.Query().
		Where(
			ticket.ID(id),
			ticket.TenantID(tenantID),
		).
		Select(ticket.FieldVersion).
		Only(ctx)
	if err != nil {
		return 0, fmt.Errorf("get ticket version: %w", err)
	}

	return entity.Version, nil
}

// ==================== 辅助函数 ====================

// toDomainModel 将 Ent 实体转换为领域模型
func toDomainModel(e *ent.Ticket) *Ticket {
	if e == nil {
		return nil
	}

	t := &Ticket{
		ID:             e.ID,
		TicketNumber:   e.TicketNumber,
		Title:          e.Title,
		Description:    e.Description,
		Status:         Status(e.Status),
		Type:           Type(e.Type),
		Priority:       Priority(e.Priority),
		RequesterID:    e.RequesterID,
		TenantID:       e.TenantID,
		Version:        e.Version,
		IsManagedByMSP: e.IsManagedByMsp,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}

	// 可选字段
	if e.AssigneeID != 0 {
		t.AssigneeID = &e.AssigneeID
	}
	if e.TemplateID != 0 {
		t.TemplateID = &e.TemplateID
	}
	if e.CategoryID != 0 {
		t.CategoryID = &e.CategoryID
	}
	if e.DepartmentID != 0 {
		t.DepartmentID = &e.DepartmentID
	}
	if e.ParentTicketID != 0 {
		t.ParentTicketID = &e.ParentTicketID
	}
	if e.SLADefinitionID != 0 {
		t.SLADefinitionID = &e.SLADefinitionID
	}

	// 时间字段
	if !e.SLAResponseDeadline.IsZero() {
		t.SLAResponseDeadline = &e.SLAResponseDeadline
	}
	if !e.SLAResolutionDeadline.IsZero() {
		t.SLAResolutionDeadline = &e.SLAResolutionDeadline
	}
	if !e.FirstResponseAt.IsZero() {
		t.FirstResponseAt = &e.FirstResponseAt
	}
	if !e.ResolvedAt.IsZero() {
		t.ResolvedAt = &e.ResolvedAt
	}
	if e.DeletedAt != nil {
		t.DeletedAt = e.DeletedAt
	}

	// MSP 相关
	if e.MspProviderID != 0 {
		t.MSPProviderID = &e.MspProviderID
	}
	if e.ManagedByUserID != 0 {
		t.ManagedByUserID = &e.ManagedByUserID
	}
	if e.MspTicketID != "" {
		t.MSPTicketID = &e.MspTicketID
	}

	return t
}

// toDomainModels 批量转换
func toDomainModels(entities []*ent.Ticket) []*Ticket {
	result := make([]*Ticket, len(entities))
	for i, e := range entities {
		result[i] = toDomainModel(e)
	}
	return result
}

// toEntField 将字段名转换为 Ent 字段
func toEntField(field string) string {
	fieldMap := map[string]string{
		"created_at": ticket.FieldCreatedAt,
		"updated_at": ticket.FieldUpdatedAt,
		"title":      ticket.FieldTitle,
		"status":     ticket.FieldStatus,
		"priority":   ticket.FieldPriority,
	}

	if entField, ok := fieldMap[field]; ok {
		return entField
	}
	return ticket.FieldCreatedAt
}
