package standard_change

import (
	"context"

	"itsm-backend/ent"
	entstandardchange "itsm-backend/ent/standardchange"
)

type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service {
	return &Service{client: client}
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
	return s.client.StandardChange.Create().
		SetTenantID(input.TenantID).
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetImplementationPlan(input.ImplementationPlan).
		SetRollbackPlan(input.RollbackPlan).
		SetJustification(input.Justification).
		SetCategory(input.Category).
		SetRiskLevel(input.RiskLevel).
		SetImpactScope(input.ImpactScope).
		SetExpectedDuration(input.ExpectedDuration).
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
	_, err := s.client.StandardChange.Delete().
		Where(entstandardchange.ID(id), entstandardchange.TenantID(tenantID)).
		Exec(ctx)
	return err
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
func (s *Service) Instantiate(ctx context.Context, tenantID, templateID int, createdBy int) (*ent.Change, error) {
	template, err := s.client.StandardChange.Query().
		Where(entstandardchange.ID(templateID), entstandardchange.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.Change.Create().
		SetTenantID(tenantID).
		SetTitle(template.Title).
		SetDescription(template.Description).
		SetType("standard").
		SetRiskLevel(template.RiskLevel).
		SetImpactScope(template.ImpactScope).
		SetStatus("pending").
		SetCreatedBy(createdBy).
		Save(ctx)
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
