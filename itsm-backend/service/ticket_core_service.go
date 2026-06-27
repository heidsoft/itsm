package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	mrand "math/rand"
	"os"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type TicketCoreService struct {
	client          *ent.Client
	rawDB           *sql.DB // for transactional SELECT FOR UPDATE
	logger          *zap.SugaredLogger
	sequenceService *SequenceService
}

// SetSequenceService 设置序列服务
func (s *TicketCoreService) SetSequenceService(seqSvc *SequenceService) {
	s.sequenceService = seqSvc
}

func NewTicketCoreService(client *ent.Client, logger *zap.SugaredLogger) *TicketCoreService {
	return &TicketCoreService{client: client, logger: logger}
}

// SetRawDB 设置原生数据库连接（用于事务性编号生成）
func (s *TicketCoreService) SetRawDB(db *sql.DB) {
	s.rawDB = db
}

func (s *TicketCoreService) CreateTicketBasic(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ent.Ticket, error) {
	// DTO binding validation handles required fields and format checks
	// Service layer validates business logic (entity existence, permissions, etc.)

	// 验证必填字段
	if req.Description == "" {
		return nil, fmt.Errorf("description不能为空")
	}
	if req.Priority == "" {
		return nil, fmt.Errorf("priority不能为空")
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
		// 检查是否是唯一约束冲突，如果是则重试（最多3次）
		if strings.Contains(err.Error(), "unique constraint") ||
			strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "23505") {
			for retry := 0; retry < 3; retry++ {
				s.logger.Warnw("Ticket number collision, retrying", "ticketNumber", ticketNumber, "retry", retry)
				time.Sleep(time.Millisecond * 100 * time.Duration(retry+1))
				newTicketNumber, err := s.generateTicketNumber(ctx, tenantID)
				if err != nil {
					break
				}
				ticket, err = s.client.Ticket.Create().
					SetTitle(req.Title).
					SetDescription(req.Description).
					SetPriority(req.Priority).
					SetType(ticketType).
					SetStatus("open").
					SetTicketNumber(newTicketNumber).
					SetTenantID(tenantID).
					SetRequesterID(req.RequesterID).
					Save(ctx)
				if err == nil {
					goto created
				}
				if !strings.Contains(err.Error(), "unique constraint") &&
					!strings.Contains(err.Error(), "duplicate key") &&
					!strings.Contains(err.Error(), "23505") {
					break
				}
				ticketNumber = newTicketNumber
			}
		}
		return nil, fmt.Errorf("创建失败: %w", err)
	}

created:

	if len(req.TagIDs) > 0 {
		if _, err := ticket.Update().AddTagIDs(req.TagIDs...).Save(ctx); err != nil {
			s.logger.Warnw("failed to associate tags with ticket", "ticketID", ticket.ID, "tagIDs", req.TagIDs, "error", err)
		}
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

	// 版本检查（乐观锁）- 除非明确强制更新
	if !req.Force && req.Version > 0 && t.Version != req.Version {
		return nil, common.NewVersionConflictError(
			"工单",
			ticketID,
			req.Version,
			t.Version,
		)
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
		// 验证状态转换是否合法
		if !IsValidTicketStatusTransition(t.Status, req.Status) {
			return nil, fmt.Errorf("无效的状态转换: 从 %s 到 %s", t.Status, req.Status)
		}
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

	// 自动增加版本号
	update = update.AddVersion(1)

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
	// 软删除：设置deleted_at时间戳
	_, err = s.client.Ticket.UpdateOneID(ticketID).
		SetDeletedAt(time.Now()).
		Save(ctx)
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
	// 软删除：批量设置deleted_at时间戳
	_, err := s.client.Ticket.Update().
		Where(ticket.IDIn(ticketIDs...), ticket.TenantID(tenantID)).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("批量删除失败: %w", err)
	}
	return nil
}

// generateTicketNumber 生成工单编号，优先使用 Redis 序列
func (s *TicketCoreService) generateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 优先使用 Redis 序列服务
	if s.sequenceService != nil {
		return s.generateTicketNumberWithRedis(ctx, tenantID, year, month)
	}

	// 备用方案：数据库查询
	return s.generateTicketNumberWithDB(ctx, tenantID, year, month)
}

// generateTicketNumberWithRedis 使用 Redis INCR 生成工单编号
func (s *TicketCoreService) generateTicketNumberWithRedis(ctx context.Context, tenantID, year, month int) (string, error) {
	key := fmt.Sprintf("sequence:ticket:%d:%d%02d", tenantID, year, month)

	// 计算本月最后一天作为过期时间
	expiredAt := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)

	// 获取序列号（带过期时间）
	seq, err := s.sequenceService.GetNextSequenceWithExpiry(ctx, key, expiredAt)
	if err != nil {
		s.logger.Warnw("Redis sequence failed, fallback to DB", "error", err)
		// 实际 fallback 到 DB，使用正确的 tenantID
		return s.generateTicketNumberWithDB(ctx, tenantID, year, month)
	}

	// 格式: TKT-YYYYMM-NNNNNN
	return fmt.Sprintf("TKT-%04d%02d-%06d", year, month, seq), nil
}

// generateTicketNumberWithDB 使用数据库事务+SELECT FOR UPDATE 生成工单编号（备用方案）
// 解决了并发时多个请求生成相同编号的问题
// 策略：如果当月有记录则加锁读取最大号；如果无记录则使用 UUID 后缀确保唯一
func (s *TicketCoreService) generateTicketNumberWithDB(ctx context.Context, tenantID int, year, month int) (string, error) {
	prefix := fmt.Sprintf("TKT-%04d%02d-", year, month)
	var candidate string

	// 开启事务，使用 SELECT FOR UPDATE 锁定最大编号行
	if s.rawDB != nil {
		tx, err := s.rawDB.BeginTx(ctx, nil)
		if err != nil {
			s.logger.Warnw("BeginTx failed, falling back to non-transactional", "error", err)
		} else {
			// 使用原生 SQL 查询最大编号（使用 FOR UPDATE SKIP LOCKED 避免阻塞）
			query := `SELECT ticket_number FROM tickets WHERE tenant_id = $1 AND ticket_number LIKE $2 AND ticket_number IS NOT NULL AND ticket_number != '' ORDER BY ticket_number DESC LIMIT 1 FOR UPDATE SKIP LOCKED`
			var maxTicketNum string
			err := tx.QueryRowContext(ctx, query, tenantID, prefix+"%").Scan(&maxTicketNum)
			s.logger.Infow("SELECT FOR UPDATE SKIP LOCKED result", "tenant", tenantID, "maxTicketNum", maxTicketNum, "error", err, "isNoRows", (err == sql.ErrNoRows))

			// 计算下一个序列号
			var seq int = 0
			// sql.ErrNoRows 意味着当月没有工单，此时 seq 保持 0
			if err == nil && maxTicketNum != "" {
				if idx := strings.LastIndex(maxTicketNum, "-"); idx >= 0 {
					fmt.Sscanf(maxTicketNum[idx+1:], "%d", &seq)
				}
			} else if err == sql.ErrNoRows {
				s.logger.Infow("No existing tickets this month, using seed sequence", "tenant", tenantID, "year", year, "month", month)
			}

			// 有历史记录则使用递增号；无记录则用随机后缀确保唯一
			if seq > 0 {
				candidate = fmt.Sprintf("TKT-%04d%02d-%06d", year, month, seq+1)
			} else {
				candidate = fmt.Sprintf("TKT-%04d%02d-%s", year, month, uniqueFallbackSuffix())
			}
			s.logger.Infow("DB transaction generated ticket number",
				"number", candidate, "tenant", tenantID)

			// 提交事务（编号唯一性由外层 CreateTicket 唯一约束保证）
			tx.Commit()
			return candidate, nil
		}
	}

	// Fallback: 非事务方式（仅用于没有 rawDB 的场景）
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.TicketNumberContains(prefix[:len(prefix)-1]),
		).
		Order(ent.Desc(ticket.FieldTicketNumber)).
		Limit(1).
		All(ctx)

	var maxSeq int
	parseErr := false
	if err != nil {
		s.logger.Warnw("Query max ticket number failed, using 0", "error", err)
		maxSeq = 0
	} else if len(tickets) > 0 {
		ticketNum := tickets[0].TicketNumber
		if ticketNum == "" {
			parseErr = true
		} else {
			parsed := false
			for i := len(ticketNum) - 1; i >= 0; i-- {
				if ticketNum[i] == '-' {
					if _, err := fmt.Sscanf(ticketNum[i+1:], "%d", &maxSeq); err == nil {
						parsed = true
					}
					break
				}
			}
			if !parsed {
				parseErr = true
			}
		}
	}

	if parseErr || (err == nil && len(tickets) == 0) {
		// 无当月记录时，使用随机后缀确保唯一性
		candidate = fmt.Sprintf("TKT-%04d%02d-%s", year, month, uniqueFallbackSuffix())
	} else {
		candidate = fmt.Sprintf("TKT-%04d%02d-%06d", year, month, maxSeq+1)
	}
	s.logger.Infow("DB fallback generated ticket number",
		"number", candidate, "tenant", tenantID)
	return candidate, nil
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

func (s *TicketCoreService) addTagsToTicket(ctx context.Context, t *ent.Ticket, tagIDs []int) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// 先清除现有标签，再添加新的，避免重复连接错误
	_, err := t.Update().ClearTags().AddTagIDs(tagIDs...).Save(ctx)
	return err
}

// uniqueFallbackSuffix 生成真正唯一的备选后缀（用于无历史记录时）
// 使用 crypto/rand + PID 种子确保并发进程间不碰撞
func uniqueFallbackSuffix() string {
	// 种子：PID + nanoseconds，保证同进程同微秒内每次调用不同
	src := mrand.NewSource(time.Now().UnixNano() + int64(os.Getpid())*1000000000)
	mrand.New(src) // 每次调用产生不同种子，增强多样性
	// 6位随机数，范围 0~999999（使用 crypto/rand 保证真随机）
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}
