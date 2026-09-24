package seeder

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
)

// TenantBaselineAudit is one tenant's read-only product baseline report.
type TenantBaselineAudit struct {
	TenantID   int                     `json:"tenantId"`
	TenantCode string                  `json:"tenantCode"`
	Ready      bool                    `json:"ready"`
	Components []ComponentVerification `json:"components"`
}

// AuditTenantBaselines reports every tenant's product baseline state without
// writing anything. It is the read-only input to a forward-fix plan: run it,
// review the gaps per tenant, then decide what (if anything) to repair. A
// tenant with gaps is a finding, not an error, so the audit reports it instead
// of aborting.
func (s *Seeder) AuditTenantBaselines(ctx context.Context) ([]TenantBaselineAudit, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("baseline audit requires a database client")
	}
	tenants, err := s.client.Tenant.Query().Order(ent.Asc(tenant.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	audits := make([]TenantBaselineAudit, 0, len(tenants))
	for _, target := range tenants {
		verifications, err := s.VerifyTenantBaseline(ctx, target.ID)
		if err != nil {
			return audits, fmt.Errorf("audit tenant %s (%d): %w", target.Code, target.ID, err)
		}
		entry := TenantBaselineAudit{
			TenantID:   target.ID,
			TenantCode: target.Code,
			Ready:      true,
			Components: verifications,
		}
		for _, verification := range verifications {
			if !verification.Verified {
				entry.Ready = false
				break
			}
		}
		audits = append(audits, entry)
	}
	return audits, nil
}
