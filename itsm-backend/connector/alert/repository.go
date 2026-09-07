package alert

import (
	"context"
	"database/sql"
	"errors"

	"itsm-backend/ent"
	"itsm-backend/ent/alert"
)

type alertRepository interface {
	Store(context.Context, int, *StandardAlert) (int64, bool, error)
}

type entAlertRepository struct {
	client *ent.Client
}

func newEntAlertRepository(client *ent.Client) alertRepository {
	if client == nil {
		return nil
	}
	return &entAlertRepository{client: client}
}

// Store atomically creates an alert or returns the existing row for a duplicate
// tenant/source/external-alert identity. The DB-level UNIQUE INDEX on
// (tenant_id, source, external_alert_id) is the authoritative deduplication
// boundary; ON CONFLICT DO NOTHING keeps the path idempotent for retried
// webhook deliveries without surfacing unique-constraint errors as failures.
func (r *entAlertRepository) Store(ctx context.Context, tenantID int, a *StandardAlert) (int64, bool, error) {
	if tenantID <= 0 || a == nil {
		return 0, false, errors.New("invalid alert persistence input")
	}

	build := r.client.Alert.Create().
		SetTenantID(tenantID).
		SetSource(a.Source).
		SetExternalAlertID(a.AlertID).
		SetSourceRaw(a.SourceRaw).
		SetName(a.Name).
		SetDescription(a.Description).
		SetSeverity(string(a.Severity)).
		SetStatus(a.Status).
		SetLabels(a.Labels).
		SetAnnotations(a.Annotations).
		SetSourceIP(a.SourceIP).
		SetService(a.Service).
		SetTags(a.Tags).
		SetFiredAt(a.FiredAt).
		SetRawPayload(a.RawPayload)

	// 设置可选的时间字段
	if a.AcknowledgedAt != nil {
		build.SetAcknowledgedAt(*a.AcknowledgedAt)
	}
	if a.ResolvedAt != nil {
		build.SetResolvedAt(*a.ResolvedAt)
	}

	// upsert via separate query (OnConflict not available in current ent version)
	created, err := build.Save(ctx)
	switch {
	case err == nil && created != nil:
		return int64(created.ID), true, nil
	case err != nil && !isEntNotFound(err):
		return 0, false, err
	}

	existing, qerr := r.client.Alert.Query().
		Where(
			alert.TenantIDEQ(tenantID),
			alert.SourceEQ(a.Source),
			alert.ExternalAlertIDEQ(a.AlertID),
		).
		Only(ctx)
	if qerr != nil {
		return 0, false, qerr
	}
	return int64(existing.ID), false, nil
}

func isEntNotFound(err error) bool {
	return ent.IsNotFound(err) || errors.Is(err, sql.ErrNoRows)
}
