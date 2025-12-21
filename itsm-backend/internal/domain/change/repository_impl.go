package change

import (
	"context"
	"database/sql"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
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
	return toDomain(ec), nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Change, error) {
	ec, err := r.client.Change.Query().
		Where(change.ID(id), change.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(ec), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, status, search string) ([]*Change, int, error) {
	q := r.client.Change.Query().Where(change.TenantID(tenantID))

	if status != "" && status != "全部" {
		q = q.Where(change.Status(status))
	}
	if search != "" {
		q = q.Where(change.Or(
			change.TitleContains(search),
			change.DescriptionContains(search),
		))
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
	return results, total, nil
}

func (r *EntRepository) Update(ctx context.Context, c *Change) (*Change, error) {
	update := r.client.Change.UpdateOneID(c.ID).
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
	return toDomain(ec), nil
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
	total, _ := r.client.Change.Query().Where(change.TenantID(tenantID)).Count(ctx)
	stats.Total = total

	// By Status
	statuses := []string{"pending", "approved", "in_progress", "completed", "rolled_back", "rejected", "cancelled"}
	for _, s := range statuses {
		count, _ := r.client.Change.Query().Where(change.TenantID(tenantID), change.Status(s)).Count(ctx)
		switch s {
		case "pending":
			stats.Pending = count
		case "approved":
			stats.Approved = count
		case "in_progress":
			stats.InProgress = count
		case "completed":
			stats.Completed = count
		case "rolled_back":
			stats.RolledBack = count
		case "rejected":
			stats.Rejected = count
		case "cancelled":
			stats.Cancelled = count
		}
	}

	return stats, nil
}

// Approval Records (Raw SQL)
func (r *EntRepository) CreateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	query := `
		INSERT INTO change_approvals (change_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, rec.ChangeID, rec.ApproverID, rec.Status, rec.Comment, time.Now(), time.Now()).
		Scan(&rec.ID, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *EntRepository) UpdateApprovalRecord(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	query := `
		UPDATE change_approvals 
		SET status = $1, comment = $2, approved_at = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, change_id, approver_id, status, comment, approved_at, created_at
	`
	var approvedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, rec.Status, rec.Comment, time.Now(), time.Now(), rec.ID).
		Scan(&rec.ID, &rec.ChangeID, &rec.ApproverID, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	if approvedAt.Valid {
		rec.ApprovedAt = &approvedAt.Time
	}
	return rec, nil
}

func (r *EntRepository) GetApprovalHistory(ctx context.Context, changeID int) ([]*ApprovalRecord, error) {
	query := `
		SELECT a.id, a.approver_id, u.name as approver_name, a.status, a.comment, a.approved_at, a.created_at
		FROM change_approvals a
		LEFT JOIN users u ON a.approver_id = u.id
		WHERE a.change_id = $1
		ORDER BY a.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, changeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*ApprovalRecord
	for rows.Next() {
		var rec ApprovalRecord
		var approvedAt sql.NullTime
		err := rows.Scan(&rec.ID, &rec.ApproverID, &rec.ApproverName, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			rec.ApprovedAt = &approvedAt.Time
		}
		records = append(records, &rec)
	}
	return records, nil
}

// Approval Chain (Raw SQL)
func (r *EntRepository) CreateApprovalChain(ctx context.Context, chain []*ApprovalChain) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range chain {
		query := `
			INSERT INTO change_approval_chains (change_id, level, approver_id, role, status, is_required, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.ExecContext(ctx, query, item.ChangeID, item.Level, item.ApproverID, item.Role, item.Status, item.IsRequired, time.Now())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *EntRepository) GetApprovalChain(ctx context.Context, changeID int) ([]*ApprovalChain, error) {
	query := `
		SELECT c.id, c.level, c.approver_id, u.name as approver_name, c.role, c.status, c.is_required, c.created_at
		FROM change_approval_chains c
		LEFT JOIN users u ON c.approver_id = u.id
		WHERE c.change_id = $1
		ORDER BY c.level ASC
	`
	rows, err := r.db.QueryContext(ctx, query, changeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chain []*ApprovalChain
	for rows.Next() {
		var item ApprovalChain
		err := rows.Scan(&item.ID, &item.Level, &item.ApproverID, &item.ApproverName, &item.Role, &item.Status, &item.IsRequired, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		item.ChangeID = changeID
		chain = append(chain, &item)
	}
	return chain, nil
}

func (r *EntRepository) DeleteApprovalChain(ctx context.Context, changeID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1", changeID)
	return err
}

// Risk Assessment (Raw SQL)
func (r *EntRepository) CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	query := `
		INSERT INTO change_risk_assessments (
			change_id, risk_level, risk_description, impact_analysis, 
			mitigation_measures, contingency_plan, risk_owner, risk_review_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query,
		ra.ChangeID, ra.RiskLevel, ra.RiskDescription, ra.ImpactAnalysis,
		ra.MitigationMeasures, ra.ContingencyPlan, ra.RiskOwner, ra.RiskReviewDate,
		now, now).
		Scan(&ra.ID, &ra.CreatedAt)
	if err != nil {
		return nil, err
	}
	ra.UpdatedAt = now
	return ra, nil
}

func (r *EntRepository) GetRiskAssessment(ctx context.Context, changeID int) (*RiskAssessment, error) {
	query := `
		SELECT id, risk_level, risk_description, impact_analysis,
		       mitigation_measures, contingency_plan, risk_owner, risk_review_date,
		       created_at, updated_at
		FROM change_risk_assessments 
		WHERE change_id = $1
	`
	var ra RiskAssessment
	var riskReviewDate sql.NullTime
	err := r.db.QueryRowContext(ctx, query, changeID).Scan(
		&ra.ID, &ra.RiskLevel, &ra.RiskDescription, &ra.ImpactAnalysis,
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
