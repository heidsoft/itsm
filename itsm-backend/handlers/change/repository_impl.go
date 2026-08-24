package change

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	entuser "itsm-backend/ent/user"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/internal/commandbus"
)

type EntRepository struct {
	client *ent.Client
	db     *sql.DB
}

func NewEntRepository(client *ent.Client, db *sql.DB) *EntRepository {
	return &EntRepository{
		client: client,
		db:     db,
	}
}

// Map ent entity to domain entity
func toDomain(ec *ent.Change) *Change {
	if ec == nil {
		return nil
	}
	return &Change{
		ID:                 ec.ID,
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
	stats := &Stats{}

	// Total
	total, err := r.client.Change.Query().Where(change.TenantID(tenantID)).Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Single GROUP BY query instead of 11 sequential COUNT queries
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM changes
		WHERE tenant_id = $1
		GROUP BY status
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		switch status {
		case "draft":
			stats.Draft = count
		case "pending":
			stats.Pending += count
		case "pending_review":
			// pending_review is a seed-data alias for pending (changes awaiting approval)
			stats.Pending += count
		case "approved":
			stats.Approved = count
		case "scheduled":
			stats.Scheduled = count
		case "in_progress":
			stats.InProgress = count
		case "completed":
			stats.Completed = count
		case "failed":
			stats.Failed = count
		case "rolled_back":
			stats.RolledBack = count
		case "rejected":
			stats.Rejected = count
		case "cancelled":
			stats.Cancelled = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// InProgress reflects changes actively being implemented (status='in_progress').
	// Scheduled is reported separately so the frontend can distinguish "已排期" from "实施中".
	// (The previous implementation summed Scheduled + Implementing, but Implementing was
	// never written anywhere — see canonical statuses in dto.ChangeStatus and the
	// canonical change status definitions.)

	return stats, nil
}

// Approval Records (Raw SQL)
func (r *EntRepository) CreateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	query := `
		INSERT INTO change_approvals (change_id, tenant_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, rec.ChangeID, rec.TenantID, rec.ApproverID, rec.Status, rec.Comment, now, now).
		Scan(&rec.ID, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *EntRepository) SubmitForApproval(
	ctx context.Context,
	changeID, tenantID int,
	plan []ApprovalLevelPlan,
	comment string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE changes SET status = 'pending', updated_at = $1
		 WHERE id = $2 AND tenant_id = $3 AND status = 'draft'`,
		time.Now(), changeID, tenantID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("change is not an editable draft")
	}

	now := time.Now()
	for _, lvl := range plan {
		approvalType := lvl.ApprovalType
		if approvalType == "" {
			approvalType = "serial"
		}
		seen := make(map[int]struct{}, len(lvl.ApproverIDs))
		for _, approverID := range lvl.ApproverIDs {
			if _, ok := seen[approverID]; ok {
				continue
			}
			seen[approverID] = struct{}{}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO change_approvals
					(change_id, tenant_id, approver_id, status, comment, created_at, updated_at)
				VALUES ($1, $2, $3, 'pending', $4, $5, $5)
			`, changeID, tenantID, approverID, comment, now); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO change_approval_chains
					(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
				VALUES ($1, $2, $3, $4, 'approver', 'pending', $5, $6, $7, $8)
			`, changeID, tenantID, lvl.Level, approverID, lvl.Required, approvalType, lvl.Threshold, now); err != nil {
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

func (r *EntRepository) UpdateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	// C-5 修复：必须加 AND status = 'pending' 条件，防止已驳回/已批准的审批被重复修改
	// 校验 RowsAffected == 1，否则返回 409 冲突，避免幂等问题
	query := `
		UPDATE change_approvals 
		SET status = $1, comment = $2, approved_at = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6 AND status = 'pending'
		RETURNING id, change_id, tenant_id, approver_id, status, comment, approved_at, created_at
	`
	var approvedAt sql.NullTime
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, rec.Status, rec.Comment, now, now, rec.ID, rec.TenantID).
		Scan(&rec.ID, &rec.ChangeID, &rec.TenantID, &rec.ApproverID, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// 没有匹配的 pending 记录：要么记录不存在，要么已被处理过（已批准/已驳回）
			// 先读一下当前记录状态，返回更精确的错误
			var curStatus string
			_ = r.db.QueryRowContext(ctx, `SELECT status FROM change_approvals WHERE id = $1 AND tenant_id = $2`, rec.ID, rec.TenantID).Scan(&curStatus)
			if curStatus != "" {
				return nil, fmt.Errorf("审批记录已处理（当前状态=%s），不可重复审批", curStatus)
			}
			return nil, fmt.Errorf("审批记录不存在或跨租户")
		}
		return nil, err
	}
	if approvedAt.Valid {
		rec.ApprovedAt = &approvedAt.Time
	}
	return rec, nil
}

func (r *EntRepository) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	// P1 修复：同时派生该审批人在审批链中所属层级（levels，逗号分隔），供 service 层按
	// (approverID, level) 双重匹配，避免跨层互相串。level 不存在于 change_approvals，
	// 由 change_approval_chains 子查询派生（不产生行扇出，保持历史记录条数不变）。
	//
	// SQL 跨方言兼容性：PostgreSQL `string_agg(int_value, sep)` 接受 int 入参并自动隐式转 text，
	// 因此 `::text` 显式 cast 在 SQLite 上不被识别（`unrecognized token: ":"`）。
	// 去除 `::text` 后双方言均可运行（PG 仍走 string_agg；SQLite 把它识别为自定义聚合，
	// SQLite ≥ 3.39 起对 `string_agg` 提供兼容别名）。
	query := `
		SELECT a.id, a.approver_id, u.name as approver_name, a.status, a.comment, a.approved_at, a.created_at,
		       COALESCE((
				   SELECT string_agg(c.level, ',' ORDER BY c.level)
				   FROM change_approval_chains c
				   WHERE c.change_id = a.change_id AND c.tenant_id = a.tenant_id AND c.approver_id = a.approver_id
			   ), '') AS levels
		FROM change_approvals a
		LEFT JOIN users u ON a.approver_id = u.id
		LEFT JOIN changes c ON a.change_id = c.id
		WHERE a.change_id = $1 AND a.tenant_id = $2 AND c.tenant_id = $2
		ORDER BY a.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 返回空切片而非 nil，避免 JSON 序列化为 null 导致前端崩溃
	records := make([]*ApprovalRecord, 0)
	for rows.Next() {
		var rec ApprovalRecord
		var approvedAt sql.NullTime
		var levelsCSV string
		err := rows.Scan(&rec.ID, &rec.ApproverID, &rec.ApproverName, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt, &levelsCSV)
		if err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			rec.ApprovedAt = &approvedAt.Time
		}
		rec.Levels = parseLevelCSV(levelsCSV)
		rec.ChangeID = changeID
		rec.TenantID = tenantID
		records = append(records, &rec)
	}
	return records, nil
}

// parseLevelCSV 将 "1,2,3" 解析为 []int{1,2,3}；空串返回 nil。
func parseLevelCSV(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, err := strconv.Atoi(p); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// Approval Chain (Raw SQL)
func (r *EntRepository) GetApprovalChain(ctx context.Context, changeID int, tenantID int) ([]*ApprovalChain, error) {
	query := `
		SELECT c.id, c.level, c.approver_id, u.name as approver_name, c.role, c.status, c.is_required, c.approval_type, c.threshold, c.created_at
		FROM change_approval_chains c
		LEFT JOIN users u ON c.approver_id = u.id
		WHERE c.change_id = $1 AND c.tenant_id = $2
		ORDER BY c.level ASC
	`
	rows, err := r.db.QueryContext(ctx, query, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 返回空切片而非 nil，避免 JSON 序列化为 null 导致前端崩溃
	chain := make([]*ApprovalChain, 0)
	for rows.Next() {
		var item ApprovalChain
		err := rows.Scan(&item.ID, &item.Level, &item.ApproverID, &item.ApproverName, &item.Role, &item.Status, &item.IsRequired, &item.ApprovalType, &item.Threshold, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		item.ChangeID = changeID
		item.TenantID = tenantID
		chain = append(chain, &item)
	}
	return chain, nil
}

func (r *EntRepository) DeleteApprovalChain(ctx context.Context, changeID int, tenantID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2", changeID, tenantID)
	return err
}

func (r *EntRepository) ReplaceApprovalChain(
	ctx context.Context,
	changeID, tenantID int,
	chain []*ApprovalChain,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2",
		changeID, tenantID); err != nil {
		return err
	}
	for _, item := range chain {
		// 保留 Quorum 元数据（与 SubmitForApproval 一致），否则重解析审批链会丢失
		// approval_type/threshold，导致推进逻辑退化为纯串行、会签/或签失效。
		approvalType := item.ApprovalType
		if approvalType == "" {
			approvalType = "serial"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO change_approval_chains
				(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, changeID, tenantID, item.Level, item.ApproverID, item.Role, item.Status, item.IsRequired, approvalType, item.Threshold, time.Now()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Risk Assessment (Raw SQL)
func (r *EntRepository) CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	query := `
		INSERT INTO change_risk_assessments (
			change_id, tenant_id, risk_level, risk_description, impact_analysis,
			mitigation_measures, contingency_plan, risk_owner, risk_review_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query,
		ra.ChangeID, ra.TenantID, ra.RiskLevel, ra.RiskDescription, ra.ImpactAnalysis,
		ra.MitigationMeasures, ra.ContingencyPlan, ra.RiskOwner, ra.RiskReviewDate,
		now, now).
		Scan(&ra.ID, &ra.CreatedAt)
	if err != nil {
		return nil, err
	}
	ra.UpdatedAt = now
	return ra, nil
}

func (r *EntRepository) GetRiskAssessment(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error) {
	query := `
		SELECT id, tenant_id, risk_level, risk_description, impact_analysis,
		       mitigation_measures, contingency_plan, risk_owner, risk_review_date,
		       created_at, updated_at
		FROM change_risk_assessments 
		WHERE change_id = $1 AND tenant_id = $2
	`
	var ra RiskAssessment
	var riskReviewDate sql.NullTime
	err := r.db.QueryRowContext(ctx, query, changeID, tenantID).Scan(
		&ra.ID, &ra.TenantID, &ra.RiskLevel, &ra.RiskDescription, &ra.ImpactAnalysis,
		&ra.MitigationMeasures, &ra.ContingencyPlan, &ra.RiskOwner, &riskReviewDate,
		&ra.CreatedAt, &ra.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error here
		}
		return nil, err
	}
	ra.ChangeID = changeID
	if riskReviewDate.Valid {
		ra.RiskReviewDate = &riskReviewDate.Time
	}
	return &ra, nil
}

func (r *EntRepository) UpdateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	query := `
		UPDATE change_risk_assessments
		SET risk_level = $1, risk_description = $2, impact_analysis = $3,
		    mitigation_measures = $4, contingency_plan = $5, risk_owner = $6,
		    risk_review_date = $7, updated_at = $8
		WHERE change_id = $9 AND tenant_id = $10
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(
		ctx, query,
		ra.RiskLevel, ra.RiskDescription, ra.ImpactAnalysis,
		ra.MitigationMeasures, ra.ContingencyPlan, ra.RiskOwner,
		ra.RiskReviewDate, time.Now(), ra.ChangeID, ra.TenantID,
	).Scan(&ra.ID, &ra.CreatedAt, &ra.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return ra, nil
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
		Where(change.TenantID(tenantID))

	if status != "" {
		query = query.Where(change.Status(status))
	}

	// Filter by planned date range in memory
	changes, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Change, 0)
	for _, c := range changes {
		if !c.PlannedStartDate.IsZero() && !c.PlannedEndDate.IsZero() {
			// Check if date ranges overlap
			if (c.PlannedStartDate.Before(end) || c.PlannedStartDate.Equal(end)) &&
				(c.PlannedEndDate.After(start) || c.PlannedEndDate.Equal(start)) {
				result = append(result, toDomain(c))
			}
		}
	}

	return result, nil
}
