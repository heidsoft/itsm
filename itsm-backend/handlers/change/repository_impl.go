package change

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	entuser "itsm-backend/ent/user"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type EntRepository struct {
	client          *ent.Client
	db              *sql.DB
	stats           *statsRepository
	approvalRecords *changeApprovalRecordRepository
	approvalChains  *changeApprovalChainRepository
	riskAssessments *changeRiskAssessmentRepository
	statusTx        *changeStatusTxRepository
}

func NewEntRepository(client *ent.Client, db *sql.DB) *EntRepository {
	return &EntRepository{
		client:          client,
		db:              db,
		stats:           newStatsRepository(client),
		approvalRecords: newChangeApprovalRecordRepository(db),
		approvalChains:  newChangeApprovalChainRepository(db),
		riskAssessments: newChangeRiskAssessmentRepository(db),
		statusTx:        newChangeStatusTxRepository(),
	}
}

// Map ent entity to domain entity
func toDomain(ec *ent.Change) *Change {
	if ec == nil {
		return nil
	}
	return &Change{
		ID:                 ec.ID,
		ChangeNumber:       ec.ChangeNumber,
		Title:              ec.Title,
		Description:        ec.Description,
		Justification:      ec.Justification,
		Type:               ec.Type,
		Status:             ec.Status,
		Priority:           ec.Priority,
		ImpactScope:        ec.ImpactScope,
		RiskLevel:          ec.RiskLevel,
		AssigneeID:         &ec.AssigneeID,
		CreatedBy:          ec.CreatedBy,
		TenantID:           ec.TenantID,
		PlannedStartDate:   &ec.PlannedStartDate,
		PlannedEndDate:     &ec.PlannedEndDate,
		ActualStartDate:    &ec.ActualStartDate,
		ActualEndDate:      &ec.ActualEndDate,
		ImplementationPlan: ec.ImplementationPlan,
		RollbackPlan:       ec.RollbackPlan,
		AffectedCIs:        ec.AffectedCis,
		RelatedTickets:     ec.RelatedTickets,
		CreatedAt:          ec.CreatedAt,
		UpdatedAt:          ec.UpdatedAt,
	}
}

// assignChangeNumber best-effort 生成变更编号（CHG-YYYYMMDD-XXXX，租户内日序列）。
// 编号写入失败不阻断创建流程，可由后台任务补偿。
func (r *EntRepository) assignChangeNumber(ctx context.Context, ec *ent.Change) {
	if ec == nil || ec.ChangeNumber != "" {
		return
	}
	number := fmt.Sprintf("CHG-%s-%04d", time.Now().Format("20060102"), ec.ID)
	if _, err := r.client.Change.UpdateOneID(ec.ID).SetChangeNumber(number).Save(ctx); err != nil {
		return
	}
	ec.ChangeNumber = number
}

// hydrateUsers loads all users referenced by the supplied changes in one
// tenant-scoped query. Change currently stores user IDs without Ent edges, so
// this provides the domain associations without introducing N+1 queries.
func (r *EntRepository) hydrateUsers(ctx context.Context, changes []*Change, tenantID int) error {
	userIDs := make(map[int]struct{})
	for _, c := range changes {
		if c == nil {
			continue
		}
		if c.CreatedBy > 0 {
			userIDs[c.CreatedBy] = struct{}{}
		}
		if c.AssigneeID != nil && *c.AssigneeID > 0 {
			userIDs[*c.AssigneeID] = struct{}{}
		}
	}
	if len(userIDs) == 0 {
		return nil
	}

	ids := make([]int, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}
	users, err := r.client.User.Query().
		Where(entuser.IDIn(ids...), entuser.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return err
	}

	usersByID := make(map[int]*User, len(users))
	for _, u := range users {
		usersByID[u.ID] = &User{ID: u.ID, Name: u.Name}
	}
	for _, c := range changes {
		if c == nil {
			continue
		}
		c.CreatedByUser = usersByID[c.CreatedBy]
		if c.AssigneeID != nil {
			c.Assignee = usersByID[*c.AssigneeID]
		}
	}
	return nil
}

func (r *EntRepository) Create(ctx context.Context, c *Change) (*Change, error) {
	ec, err := r.client.Change.Create().
		SetTitle(c.Title).
		SetDescription(c.Description).
		SetJustification(c.Justification).
		SetType(c.Type).
		SetStatus(c.Status).
		SetPriority(c.Priority).
		SetImpactScope(c.ImpactScope).
		SetRiskLevel(c.RiskLevel).
		SetCreatedBy(c.CreatedBy).
		SetTenantID(c.TenantID).
		SetImplementationPlan(c.ImplementationPlan).
		SetRollbackPlan(c.RollbackPlan).
		SetNillablePlannedStartDate(c.PlannedStartDate).
		SetNillablePlannedEndDate(c.PlannedEndDate).
		SetAffectedCis(c.AffectedCIs).
		SetRelatedTickets(c.RelatedTickets).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	r.assignChangeNumber(ctx, ec)
	result := toDomain(ec)
	if err := r.hydrateUsers(ctx, []*Change{result}, c.TenantID); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateWithWorkflowCommand atomically persists the change and its workflow command.
func (r *EntRepository) CreateWithWorkflowCommand(ctx context.Context, c *Change) (*Change, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(cause error) (*Change, error) {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("%w; rollback: %v", cause, rbErr)
		}
		return nil, cause
	}
	ec, err := tx.Change.Create().
		SetTitle(c.Title).SetDescription(c.Description).SetJustification(c.Justification).
		SetType(c.Type).SetStatus(c.Status).SetPriority(c.Priority).SetImpactScope(c.ImpactScope).
		SetRiskLevel(c.RiskLevel).SetCreatedBy(c.CreatedBy).SetTenantID(c.TenantID).
		SetImplementationPlan(c.ImplementationPlan).SetRollbackPlan(c.RollbackPlan).
		SetNillablePlannedStartDate(c.PlannedStartDate).SetNillablePlannedEndDate(c.PlannedEndDate).
		SetAffectedCis(c.AffectedCIs).SetRelatedTickets(c.RelatedTickets).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	_, err = commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID: c.TenantID, CommandType: commandbus.CommandStartBPMN,
		AggregateType: "change", AggregateID: ec.ID,
		IdempotencyKey: fmt.Sprintf("change:%d:workflow:start", ec.ID),
		Payload:        map[string]interface{}{"businessType": "change", "businessId": ec.ID},
	})
	if err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return rollback(err)
	}
	r.assignChangeNumber(ctx, ec)
	result := toDomain(ec)
	if err := r.hydrateUsers(ctx, []*Change{result}, c.TenantID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Change, error) {
	ec, err := r.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	result := toDomain(ec)
	if err := r.hydrateUsers(ctx, []*Change{result}, tenantID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, status, search, riskLevel string, dataScope datascope.DataScope, currentUserID int) ([]*Change, int, error) {
	q := r.client.Change.Query().Where(change.TenantID(tenantID))

	if status != "" && status != "全部" {
		q = q.Where(change.Status(status))
	}
	if riskLevel != "" && riskLevel != "全部" {
		q = q.Where(change.RiskLevel(riskLevel))
	}
	if search != "" {
		q = q.Where(change.Or(
			change.TitleContains(search),
			change.DescriptionContains(search),
		))
	}

	// 行级数据权限（推广自 ticket DataScope 模式）：
	// OwnedOrAssigned 时强制追加 Or(CreatedByEQ(uid), AssigneeIDEQ(uid))，
	// 使普通用户只能看到自己创建或分配给自己的变更单。
	// 安全关键路径：即使上层忘记传归属过滤，这里仍会兜底收窄；
	// CurrentUserID<=0 时 fail-closed，返回空集而非全量。
	if dataScope == datascope.DataScopeOwnedOrAssigned {
		if currentUserID <= 0 {
			q = q.Where(change.IDEQ(-1))
		} else {
			q = q.Where(change.Or(
				change.CreatedByEQ(currentUserID),
				change.AssigneeIDEQ(currentUserID),
			))
		}
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	ecs, err := q.Order(ent.Desc(change.FieldCreatedAt)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var results []*Change
	for _, ec := range ecs {
		results = append(results, toDomain(ec))
	}
	if err := r.hydrateUsers(ctx, results, tenantID); err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

// UpdateStatusCAS 用单条条件 UPDATE 实现状态推进，把「读状态—校验—写状态」
// 的竞态窗口收敛为一次原子比较交换。状态已被其他请求抢先修改时返回 false。
func (r *EntRepository) UpdateStatusCAS(ctx context.Context, id, tenantID int, expectedStatus, targetStatus string) (bool, error) {
	affected, err := r.client.Change.Update().
		Where(change.IDEQ(id), change.TenantIDEQ(tenantID), change.StatusEQ(expectedStatus)).
		SetStatus(targetStatus).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *EntRepository) Update(ctx context.Context, c *Change) (*Change, error) {
	// P1 修复：写路径强制租户隔离，避免越权更新跨租户变更。
	// 注：change schema 无 deleted_at（物理删），故不附加 DeletedAtIsNil 守卫。
	update := r.client.Change.UpdateOneID(c.ID).
		Where(change.TenantIDEQ(c.TenantID)).
		SetTitle(c.Title).
		SetDescription(c.Description).
		SetJustification(c.Justification).
		SetType(c.Type).
		SetStatus(c.Status).
		SetPriority(c.Priority).
		SetImpactScope(c.ImpactScope).
		SetRiskLevel(c.RiskLevel).
		SetImplementationPlan(c.ImplementationPlan).
		SetRollbackPlan(c.RollbackPlan).
		SetAffectedCis(c.AffectedCIs).
		SetRelatedTickets(c.RelatedTickets)

	if c.AssigneeID != nil {
		update.SetAssigneeID(*c.AssigneeID)
	}
	if c.PlannedStartDate != nil {
		update.SetPlannedStartDate(*c.PlannedStartDate)
	}
	if c.PlannedEndDate != nil {
		update.SetPlannedEndDate(*c.PlannedEndDate)
	}
	if c.ActualStartDate != nil {
		update.SetActualStartDate(*c.ActualStartDate)
	}
	if c.ActualEndDate != nil {
		update.SetActualEndDate(*c.ActualEndDate)
	}

	ec, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	result := toDomain(ec)
	if err := r.hydrateUsers(ctx, []*Change{result}, c.TenantID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.Change.Delete().
		Where(change.ID(id), change.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	if r.stats == nil {
		return nil, fmt.Errorf("change stats repository not initialised")
	}
	return r.stats.GetStats(ctx, tenantID)
}

// Approval Records —— 已封装到 changeApprovalRecordRepository
func (r *EntRepository) CreateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	if r.approvalRecords == nil {
		return nil, fmt.Errorf("change approval record repository not initialised")
	}
	return r.approvalRecords.Create(ctx, rec)
}

func (r *EntRepository) SubmitForApproval(
	ctx context.Context,
	changeID, tenantID int,
	plan []ApprovalLevelPlan,
	comment string,
) error {
	return r.SubmitForApprovalWithWorkflow(ctx, changeID, tenantID, plan, comment, nil)
}

// SubmitForApprovalWithWorkflow 在同一底层数据库事务内推进 BPMN 并提交变更审批。
// workflow 收到的 Ent client 绑定到当前 sql.Tx，禁止在回调内自行提交事务。
//
// 拆解：变更状态推进 + 审批记录/审批链插入 + 通知 outbox 都委托到对应 repository，
// 事务边界仍由本函数独占管理（*sql.Tx），repository 只负责"对表做正确的事"。
func (r *EntRepository) SubmitForApprovalWithWorkflow(
	ctx context.Context,
	changeID, tenantID int,
	plan []ApprovalLevelPlan,
	comment string,
	workflow func(*ent.Client) error,
) error {
	if r.db == nil {
		return fmt.Errorf("change approval transaction database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if workflow != nil {
		dialectName, err := sqlDialect(r.db)
		if err != nil {
			return err
		}
		// 绑定已有 *sql.Tx 时必须用 nopTx 语义（对齐 ent/tx.go 的 txDriver）：
		// ent builder 的 sqlgraph.UpdateNode 会调用 drv.Tx()，若直接把 *sql.Tx
		// 塞进 entsql.Conn{ExecQuerier}，Driver.DB() 的 *sql.DB 类型断言会 panic
		// （interface conversion: sql.ExecQuerier is *sql.Tx, not *sql.DB）。
		// 包装后 Tx() 返回 nop 事务：Commit/Rollback 均为空操作，事务生命周期
		// 仍由本函数的 tx/defer tx.Rollback() 独占管理。
		txClient := ent.NewClient(ent.Driver(&nopTxDriver{
			drv: entsql.NewDriver(dialectName, entsql.Conn{ExecQuerier: tx}),
		}))
		if err := workflow(txClient); err != nil {
			return fmt.Errorf("advance BPMN workflow: %w", err)
		}
	}

	// 1) 把变更推进到 pending。条件 UPDATE 保证幂等：
	//    只有 status='draft' 时才能推进，避免覆盖别人已经提交/批准的状态。
	if r.statusTx == nil {
		return fmt.Errorf("change status tx repository not initialised")
	}
	promoted, err := r.statusTx.PromoteDraftToPending(ctx, tx, changeID, tenantID, time.Now())
	if err != nil {
		return err
	}
	if !promoted {
		return fmt.Errorf("change is not an editable draft")
	}

	// 2) 写审批记录与审批链（按 (level, approver) 维度展开），并发送 in_app 通知 outbox。
	now := time.Now()
	for _, lvl := range plan {
		seen := make(map[int]struct{}, len(lvl.ApproverIDs))
		for _, approverID := range lvl.ApproverIDs {
			if _, ok := seen[approverID]; ok {
				continue
			}
			seen[approverID] = struct{}{}

			if r.approvalRecords == nil {
				return fmt.Errorf("change approval record repository not initialised")
			}
			if err := r.approvalRecords.CreateTx(ctx, tx, changeID, tenantID, approverID, comment, now); err != nil {
				return err
			}

			if r.approvalChains == nil {
				return fmt.Errorf("change approval chain repository not initialised")
			}
			if err := r.approvalChains.InsertTx(ctx, tx,
				changeID, tenantID, lvl.Level, approverID, "approver",
				lvl.Required, lvl.ApprovalType, lvl.Threshold, now,
			); err != nil {
				return err
			}

			content := fmt.Sprintf("【变更审批】变更 #%d 等待您的审批（第 %d 级）", changeID, lvl.Level)
			occurrenceKey := fmt.Sprintf("change_approval_required:%d:%d:%d:%d", tenantID, changeID, lvl.Level, approverID)
			digest := sha256.Sum256([]byte(fmt.Sprintf("%d|change|%d|%d|%s|in_app|%s", tenantID, changeID, approverID, "change_approval_required", occurrenceKey)))
			if err := commandbus.EnqueueSQLTx(ctx, tx, commandbus.EnqueueRequest{
				TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
				AggregateType: "change", AggregateID: changeID,
				IdempotencyKey: "notification:" + hex.EncodeToString(digest[:16]),
				Payload: map[string]interface{}{
					"resourceType": "change", "resourceId": changeID, "recipientId": approverID,
					"type": "change_approval_required", "channel": "in_app", "content": content,
				},
			}); err != nil {
				return fmt.Errorf("enqueue change approval notification: %w", err)
			}
		}
	}
	return tx.Commit()
}

func sqlDialect(db *sql.DB) (string, error) {
	driverType := fmt.Sprintf("%T", db.Driver())
	switch {
	case strings.Contains(driverType, "sqlite3"):
		return "sqlite3", nil
	case strings.Contains(driverType, "pq") || strings.Contains(driverType, "pgx"):
		return "postgres", nil
	default:
		return "", fmt.Errorf("unsupported database driver %s", driverType)
	}
}

// nopTxDriver 让绑定到 *sql.Tx 的 ent client 在事务内安全执行：
// Tx() 返回自身的 nop 包装（Commit/Rollback 无操作），与 ent 生成代码
// ent/tx.go 中 txDriver 的语义一致。真实的提交/回滚由外层事务管理者执行。
type nopTxDriver struct {
	drv *entsql.Driver
}

func (d *nopTxDriver) Exec(ctx context.Context, query string, args, v any) error {
	return d.drv.Exec(ctx, query, args, v)
}

func (d *nopTxDriver) Query(ctx context.Context, query string, args, v any) error {
	return d.drv.Query(ctx, query, args, v)
}

func (d *nopTxDriver) Tx(context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}

func (d *nopTxDriver) Close() error { return nil }

func (d *nopTxDriver) Dialect() string { return d.drv.Dialect() }

func (r *EntRepository) UpdateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	if r.approvalRecords == nil {
		return nil, fmt.Errorf("change approval record repository not initialised")
	}
	return r.approvalRecords.Update(ctx, rec)
}

func (r *EntRepository) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	if r.approvalRecords == nil {
		return nil, fmt.Errorf("change approval record repository not initialised")
	}
	if r.approvalChains == nil {
		return nil, fmt.Errorf("change approval chain repository not initialised")
	}
	records, err := r.approvalRecords.ListByChange(ctx, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	levelsByApprover, err := r.approvalChains.LevelsByApprover(ctx, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	for _, rec := range records {
		rec.Levels = levelsByApprover[rec.ApproverID]
	}
	return records, nil
}

// Approval Chain —— 已封装到 changeApprovalChainRepository
func (r *EntRepository) GetApprovalChain(ctx context.Context, changeID int, tenantID int) ([]*ApprovalChain, error) {
	if r.approvalChains == nil {
		return nil, fmt.Errorf("change approval chain repository not initialised")
	}
	return r.approvalChains.ListByChange(ctx, changeID, tenantID)
}

func (r *EntRepository) DeleteApprovalChain(ctx context.Context, changeID int, tenantID int) error {
	if r.approvalChains == nil {
		return fmt.Errorf("change approval chain repository not initialised")
	}
	return r.approvalChains.DeleteByChange(ctx, changeID, tenantID)
}

func (r *EntRepository) ReplaceApprovalChain(
	ctx context.Context,
	changeID, tenantID int,
	chain []*ApprovalChain,
) error {
	if r.approvalChains == nil {
		return fmt.Errorf("change approval chain repository not initialised")
	}
	return r.approvalChains.Replace(ctx, changeID, tenantID, chain)
}

// Risk Assessment —— 已封装到 changeRiskAssessmentRepository
func (r *EntRepository) CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	if r.riskAssessments == nil {
		return nil, fmt.Errorf("change risk assessment repository not initialised")
	}
	return r.riskAssessments.Create(ctx, ra)
}

func (r *EntRepository) GetRiskAssessment(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error) {
	if r.riskAssessments == nil {
		return nil, fmt.Errorf("change risk assessment repository not initialised")
	}
	return r.riskAssessments.GetByChange(ctx, changeID, tenantID)
}

func (r *EntRepository) UpdateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	if r.riskAssessments == nil {
		return nil, fmt.Errorf("change risk assessment repository not initialised")
	}
	return r.riskAssessments.Update(ctx, ra)
}

// ValidateApproverBelongsToTenant validates that an approver belongs to the specified tenant
func (r *EntRepository) ValidateApproverBelongsToTenant(ctx context.Context, approverID, tenantID int) (bool, error) {
	exists, err := r.client.User.Query().
		Where(entuser.ID(approverID), entuser.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ListByDateRange retrieves changes within a date range
func (r *EntRepository) ListByDateRange(ctx context.Context, tenantID int, startDate, endDate, status string) ([]*Change, error) {
	// Parse date range
	start, err1 := time.Parse("2006-01-02", startDate)
	end, err2 := time.Parse("2006-01-02", endDate)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("invalid date format")
	}
	end = end.Add(24*time.Hour - time.Second) // End of day

	query := r.client.Change.Query().
		Where(
			change.TenantIDEQ(tenantID),
			change.PlannedStartDateLTE(end),
			change.PlannedEndDateGTE(start),
		)

	if status != "" {
		query = query.Where(change.StatusEQ(status))
	}

	ecs, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Change, 0, len(ecs))
	for _, ec := range ecs {
		result = append(result, toDomain(ec))
	}
	return result, nil
}

// FindOverlappingScheduled 查询与给定时间窗重叠且处于活跃状态（未终态）的
// 其他变更，用于提交/排期前的窗口冲突检查。
//
// 重叠语义（半开区间相交）：existing.start < windowEnd && existing.end > windowStart。
// 状态过滤：draft（无排期约束）与终态/失败（窗口已释放）不参与冲突。
// 排除自身（excludeChangeID<=0 时不排除）。
func (r *EntRepository) FindOverlappingScheduled(ctx context.Context, tenantID int, excludeChangeID int, windowStart, windowEnd time.Time) ([]*Change, error) {
	query := r.client.Change.Query().
		Where(
			change.TenantIDEQ(tenantID),
			change.PlannedStartDateNotNil(),
			change.PlannedEndDateNotNil(),
			change.PlannedStartDateLT(windowEnd),
			change.PlannedEndDateGT(windowStart),
			change.StatusIn(
				"pending",
				"approved",
				"scheduled",
				"in_progress",
			),
		)
	if excludeChangeID > 0 {
		query = query.Where(change.IDNEQ(excludeChangeID))
	}

	ecs, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Change, 0, len(ecs))
	for _, ec := range ecs {
		result = append(result, toDomain(ec))
	}
	return result, nil
}
