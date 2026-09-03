package standard_change

import (
	"context"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entstandardchange "itsm-backend/ent/standardchange"
	changedomain "itsm-backend/handlers/change"
)

type ChangeCreator interface {
	CreateChange(context.Context, *changedomain.Change) (*changedomain.Change, error)
}

type Service struct {
	client  *ent.Client
	changes ChangeCreator
}

func NewService(client *ent.Client, changes ChangeCreator) *Service {
	return &Service{client: client, changes: changes}
}

func (s *Service) ListStandardChanges(ctx context.Context, tenantID int, page, pageSize int, category, search string, activeOnly bool) ([]*ent.StandardChange, int, error) {
	query := s.client.StandardChange.Query().Where(entstandardchange.TenantID(tenantID))
	if activeOnly {
		query = query.Where(entstandardchange.IsActive(true))
	}
	if category != "" {
		query = query.Where(entstandardchange.Category(category))
	}
	if search != "" {
		query = query.Where(
			entstandardchange.Or(
				entstandardchange.TitleContains(search),
				entstandardchange.DescriptionContains(search),
			),
		)
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	items, err := query.
		Order(ent.Desc(entstandardchange.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetStandardChange(ctx context.Context, tenantID, id int) (*ent.StandardChange, error) {
	return s.client.StandardChange.Query().
		Where(entstandardchange.ID(id), entstandardchange.TenantID(tenantID)).
		Only(ctx)
}

func (s *Service) CreateStandardChange(ctx context.Context, input *SCCreateInput) (*ent.StandardChange, error) {
	// 默认值属于业务规则，必须在 service 层落地：handler 只负责 HTTP 编排。
	// 这些默认值在 controller → handlers 的 service 化重构中一度丢失（req 空值被
	// 直接写入），导致 category/risk_level/impact_scope 落库为空串。
	riskLevel := input.RiskLevel
	if riskLevel == "" {
		riskLevel = "low"
	}
	impactScope := input.ImpactScope
	if impactScope == "" {
		impactScope = "low"
	}
	category := input.Category
	if category == "" {
		category = "general"
	}
	// expected_duration：省略时 JSON 零值 0 会被显式写入并覆盖 schema 默认值 30，
	// 因此任何非正值都回落到 30，保证模板有合理的预计工期。
	expectedDuration := input.ExpectedDuration
	if expectedDuration <= 0 {
		expectedDuration = 30
	}

	return s.client.StandardChange.Create().
		SetTenantID(input.TenantID).
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetImplementationPlan(input.ImplementationPlan).
		SetRollbackPlan(input.RollbackPlan).
		SetJustification(input.Justification).
		SetCategory(category).
		SetRiskLevel(riskLevel).
		SetImpactScope(impactScope).
		SetExpectedDuration(expectedDuration).
		SetApprovalRequired(input.ApprovalRequired).
		SetAffectedCis(input.AFFECTEDCIs).
		SetPrerequisites(input.Prerequisites).
		SetRemarks(input.Remarks).
		SetCreatedBy(input.CreatedBy).
		SetIsActive(true).
		Save(ctx)
}

func (s *Service) UpdateStandardChange(ctx context.Context, tenantID, id int, input *SCUpdateInput) (*ent.StandardChange, error) {
	update := s.client.StandardChange.UpdateOneID(id).
		Where(entstandardchange.TenantID(tenantID))
	if input.Title != nil {
		update.SetTitle(*input.Title)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.ImplementationPlan != nil {
		update.SetImplementationPlan(*input.ImplementationPlan)
	}
	if input.RollbackPlan != nil {
		update.SetRollbackPlan(*input.RollbackPlan)
	}
	if input.Justification != nil {
		update.SetJustification(*input.Justification)
	}
	if input.Category != nil {
		update.SetCategory(*input.Category)
	}
	if input.RiskLevel != nil {
		update.SetRiskLevel(*input.RiskLevel)
	}
	if input.ImpactScope != nil {
		update.SetImpactScope(*input.ImpactScope)
	}
	if input.ExpectedDuration != nil {
		update.SetExpectedDuration(*input.ExpectedDuration)
	}
	if input.ApprovalRequired != nil {
		update.SetApprovalRequired(*input.ApprovalRequired)
	}
	if input.AFFECTEDCIs != nil {
		update.SetAffectedCis(*input.AFFECTEDCIs)
	}
	if input.Prerequisites != nil {
		update.SetPrerequisites(*input.Prerequisites)
	}
	if input.Remarks != nil {
		update.SetRemarks(*input.Remarks)
	}
	if input.IsActive != nil {
		update.SetIsActive(*input.IsActive)
	}
	return update.Save(ctx)
}

func (s *Service) DeleteStandardChange(ctx context.Context, tenantID, id int) error {
	// Retain template history and make deletion idempotent for this tenant.
	return s.client.StandardChange.UpdateOneID(id).
		Where(entstandardchange.TenantID(tenantID)).
		SetIsActive(false).
		Exec(ctx)
}

func (s *Service) GetCategories(ctx context.Context, tenantID int) ([]string, error) {
	items, err := s.client.StandardChange.Query().
		Where(entstandardchange.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	catSet := make(map[string]bool)
	cats := []string{}
	for _, sc := range items {
		if sc.Category != "" && !catSet[sc.Category] {
			catSet[sc.Category] = true
			cats = append(cats, sc.Category)
		}
	}
	return cats, nil
}

// Instantiate creates a Change from a StandardChange template
func (s *Service) Instantiate(ctx context.Context, tenantID, templateID int, createdBy int, req *dto.InstantiateStandardChangeRequest) (*changedomain.Change, error) {
	template, err := s.client.StandardChange.Query().
		Where(entstandardchange.ID(templateID), entstandardchange.TenantID(tenantID), entstandardchange.IsActive(true)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if s.changes == nil {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, "变更创建服务未就绪", "")
	}
	title := template.Title
	if req.Title != "" {
		title = req.Title
	}
	affected := template.AffectedCis
	if req.AffectedCis != nil {
		affected = req.AffectedCis
	}
	return s.changes.CreateChange(ctx, &changedomain.Change{
		TenantID: tenantID, CreatedBy: createdBy, Title: title,
		Description: template.Description, Justification: template.Justification,
		Type: "standard", Status: "draft", Priority: "medium",
		RiskLevel: template.RiskLevel, ImpactScope: template.ImpactScope,
		ImplementationPlan: template.ImplementationPlan, RollbackPlan: template.RollbackPlan,
		AffectedCIs: affected, PlannedStartDate: req.PlannedStartDate, PlannedEndDate: req.PlannedEndDate,
	})
}

// Input DTOs
type SCCreateInput struct {
	TenantID           int
	Title              string
	Description        string
	ImplementationPlan string
	RollbackPlan       string
	Justification      string
	Category           string
	RiskLevel          string
	ImpactScope        string
	ExpectedDuration   int
	ApprovalRequired   bool
	AFFECTEDCIs        []string
	Prerequisites      []string
	Remarks            string
	CreatedBy          int
}

type SCUpdateInput struct {
	Title              *string
	Description        *string
	ImplementationPlan *string
	RollbackPlan       *string
	Justification      *string
	Category           *string
	RiskLevel          *string
	ImpactScope        *string
	ExpectedDuration   *int
	ApprovalRequired   *bool
	AFFECTEDCIs        *[]string
	Prerequisites      *[]string
	Remarks            *string
	IsActive           *bool
}
