package known_error

import (
	"context"

	"itsm-backend/ent"
	entknownerror "itsm-backend/ent/knownerror"
)

type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

func (s *Service) ListKnownErrors(ctx context.Context, tenantID int, page, pageSize int) ([]*ent.KnownError, int, error) {
	query := s.client.KnownError.Query().Where(entknownerror.TenantID(tenantID))
	offset := (page - 1) * pageSize
	items, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetKnownError(ctx context.Context, tenantID, id int) (*ent.KnownError, error) {
	return s.client.KnownError.Query().
		Where(entknownerror.ID(id), entknownerror.TenantID(tenantID)).
		Only(ctx)
}

func (s *Service) CreateKnownError(ctx context.Context, input *KnownErrorCreateInput) (*ent.KnownError, error) {
	return s.client.KnownError.Create().
		SetTenantID(input.TenantID).
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetSymptoms(input.Symptoms).
		SetRootCause(input.RootCause).
		SetWorkaround(input.Workaround).
		SetResolution(input.Resolution).
		SetStatus(input.Status).
		SetCategory(input.Category).
		SetSeverity(input.Severity).
		SetAffectedProducts(input.AffectedProducts).
		SetAffectedCis(input.AffectedCIs).
		SetKeywords(input.Keywords).
		SetCreatedBy(input.CreatedBy).
		Save(ctx)
}

func (s *Service) UpdateKnownError(ctx context.Context, tenantID, id int, input *KnownErrorUpdateInput) (*ent.KnownError, error) {
	update := s.client.KnownError.UpdateOneID(id).
		Where(entknownerror.TenantID(tenantID))
	if input.Title != nil {
		update.SetTitle(*input.Title)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.Symptoms != nil {
		update.SetSymptoms(*input.Symptoms)
	}
	if input.RootCause != nil {
		update.SetRootCause(*input.RootCause)
	}
	if input.Workaround != nil {
		update.SetWorkaround(*input.Workaround)
	}
	if input.Resolution != nil {
		update.SetResolution(*input.Resolution)
	}
	if input.Status != nil {
		update.SetStatus(*input.Status)
	}
	if input.Category != nil {
		update.SetCategory(*input.Category)
	}
	if input.Severity != nil {
		update.SetSeverity(*input.Severity)
	}
	if input.AffectedProducts != nil {
		update.SetAffectedProducts(*input.AffectedProducts)
	}
	if input.AffectedCIs != nil {
		update.SetAffectedCis(*input.AffectedCIs)
	}
	if input.Keywords != nil {
		update.SetKeywords(*input.Keywords)
	}
	return update.Save(ctx)
}

func (s *Service) DeleteKnownError(ctx context.Context, tenantID, id int) error {
	_, err := s.client.KnownError.Delete().
		Where(entknownerror.ID(id), entknownerror.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (s *Service) SearchKnownErrors(ctx context.Context, tenantID int, keyword string, page, pageSize int) ([]*ent.KnownError, int, error) {
	baseQuery := s.client.KnownError.Query().Where(entknownerror.TenantID(tenantID))
	if keyword != "" {
		baseQuery = baseQuery.Where(entknownerror.Or(
			entknownerror.TitleContainsFold(keyword),
			entknownerror.DescriptionContainsFold(keyword),
		))
	}
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	items, err := baseQuery.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type Stats struct {
	Total      int
	Active     int
	Resolved   int
	Deprecated int
	Critical   int
	High       int
	Medium     int
	Low        int
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	baseQuery := s.client.KnownError.Query().Where(entknownerror.TenantID(tenantID))
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}
	active, err := baseQuery.Where(entknownerror.Status("active")).Count(ctx)
	if err != nil {
		return nil, err
	}
	resolved, err := baseQuery.Where(entknownerror.Status("resolved")).Count(ctx)
	if err != nil {
		return nil, err
	}
	deprecated, err := baseQuery.Where(entknownerror.Status("deprecated")).Count(ctx)
	if err != nil {
		return nil, err
	}
	critical, err := baseQuery.Where(entknownerror.Severity("critical")).Count(ctx)
	if err != nil {
		return nil, err
	}
	high, err := baseQuery.Where(entknownerror.Severity("high")).Count(ctx)
	if err != nil {
		return nil, err
	}
	medium, err := baseQuery.Where(entknownerror.Severity("medium")).Count(ctx)
	if err != nil {
		return nil, err
	}
	low, err := baseQuery.Where(entknownerror.Severity("low")).Count(ctx)
	if err != nil {
		return nil, err
	}
	return &Stats{
		Total:      total,
		Active:     active,
		Resolved:   resolved,
		Deprecated: deprecated,
		Critical:   critical,
		High:       high,
		Medium:     medium,
		Low:        low,
	}, nil
}

// KnownErrorCreateInput DTO for create
type KnownErrorCreateInput struct {
	TenantID         int
	Title            string
	Description      string
	Symptoms         string
	RootCause        string
	Workaround       string
	Resolution       string
	Status           string
	Category         string
	Severity         string
	AffectedProducts []string
	AffectedCIs      []string
	Keywords         []string
	CreatedBy        int
}

// KnownErrorUpdateInput DTO for update
type KnownErrorUpdateInput struct {
	Title            *string
	Description      *string
	Symptoms         *string
	RootCause        *string
	Workaround       *string
	Resolution       *string
	Status           *string
	Category         *string
	Severity         *string
	AffectedProducts *[]string
	AffectedCIs      *[]string
	Keywords         *[]string
}
