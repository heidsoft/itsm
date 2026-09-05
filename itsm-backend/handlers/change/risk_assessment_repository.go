package change

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// changeRiskAssessmentRepository 负责 change_risk_assessments 表的 CRUD。
// 该表目前没有 Ent schema，因此按用户偏好封装成独立 repository：
//   - SQL 参数化、强制 tenant_id 谓词、NotFound 行为与领域契约一致
//   - 由 EntRepository 聚合持有，对外保持原 Repository 接口不变
type changeRiskAssessmentRepository struct {
	db *sql.DB
}

func newChangeRiskAssessmentRepository(db *sql.DB) *changeRiskAssessmentRepository {
	if db == nil {
		return nil
	}
	return &changeRiskAssessmentRepository{db: db}
}

// Create 插入一条风险评估记录并回填 ID/createdAt/updatedAt。
func (r *changeRiskAssessmentRepository) Create(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	const query = `
		INSERT INTO change_risk_assessments (
			change_id, tenant_id, risk_level, risk_description, impact_analysis,
			mitigation_measures, contingency_plan, risk_owner, risk_review_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	now := time.Now()
	if err := r.db.QueryRowContext(ctx, query,
		ra.ChangeID, ra.TenantID, ra.RiskLevel, ra.RiskDescription, ra.ImpactAnalysis,
		ra.MitigationMeasures, ra.ContingencyPlan, ra.RiskOwner, ra.RiskReviewDate,
		now, now,
	).Scan(&ra.ID, &ra.CreatedAt); err != nil {
		return nil, err
	}
	ra.UpdatedAt = now
	return ra, nil
}

// GetByChange 返回指定变更的风险评估记录。
// 找不到记录时返回 (nil, nil)：领域语义是"未做过风险评估"而非错误，
// 上层 service 据此决定是否创建。
func (r *changeRiskAssessmentRepository) GetByChange(ctx context.Context, changeID, tenantID int) (*RiskAssessment, error) {
	const query = `
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
			return nil, nil
		}
		return nil, fmt.Errorf("query risk assessment: %w", err)
	}
	ra.ChangeID = changeID
	if riskReviewDate.Valid {
		ra.RiskReviewDate = &riskReviewDate.Time
	}
	return &ra, nil
}

// Update 按 change+tenant 维度更新一条已有风险评估。
// 注意：WHERE 子句限定 change_id + tenant_id，避免跨租户改写。
func (r *changeRiskAssessmentRepository) Update(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	const query = `
		UPDATE change_risk_assessments
		SET risk_level = $1, risk_description = $2, impact_analysis = $3,
		    mitigation_measures = $4, contingency_plan = $5, risk_owner = $6,
		    risk_review_date = $7, updated_at = $8
		WHERE change_id = $9 AND tenant_id = $10
		RETURNING id, created_at, updated_at
	`
	if err := r.db.QueryRowContext(ctx, query,
		ra.RiskLevel, ra.RiskDescription, ra.ImpactAnalysis,
		ra.MitigationMeasures, ra.ContingencyPlan, ra.RiskOwner,
		ra.RiskReviewDate, time.Now(), ra.ChangeID, ra.TenantID,
	).Scan(&ra.ID, &ra.CreatedAt, &ra.UpdatedAt); err != nil {
		return nil, err
	}
	return ra, nil
}
