package alert

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type alertRepository interface {
	Store(context.Context, int, *StandardAlert) (int64, bool, error)
}

type sqlAlertRepository struct {
	db *sql.DB
}

func newSQLAlertRepository(db *sql.DB) alertRepository {
	if db == nil {
		return nil
	}
	return &sqlAlertRepository{db: db}
}

// Store atomically creates an alert or returns the existing row for a duplicate
// tenant/source/external-alert identity.
func (r *sqlAlertRepository) Store(ctx context.Context, tenantID int, alert *StandardAlert) (int64, bool, error) {
	if tenantID <= 0 || alert == nil {
		return 0, false, errors.New("invalid alert persistence input")
	}
	labels, err := json.Marshal(alert.Labels)
	if err != nil {
		return 0, false, fmt.Errorf("marshal alert labels: %w", err)
	}
	annotations, err := json.Marshal(alert.Annotations)
	if err != nil {
		return 0, false, fmt.Errorf("marshal alert annotations: %w", err)
	}
	tags, err := json.Marshal(alert.Tags)
	if err != nil {
		return 0, false, fmt.Errorf("marshal alert tags: %w", err)
	}
	rawPayload, err := json.Marshal(alert.RawPayload)
	if err != nil {
		return 0, false, fmt.Errorf("marshal alert payload: %w", err)
	}

	const insert = `
INSERT INTO alerts (
    tenant_id, source, external_alert_id, source_raw, name, description,
    severity, status, labels, annotations, source_ip, service, tags,
    fired_at, acknowledged_at, resolved_at, raw_payload
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
ON CONFLICT (tenant_id, source, external_alert_id) DO NOTHING
RETURNING id`
	var id int64
	err = r.db.QueryRowContext(ctx, insert,
		tenantID, alert.Source, alert.AlertID, alert.SourceRaw, alert.Name, alert.Description,
		string(alert.Severity), alert.Status, labels, annotations, alert.SourceIP, alert.Service, tags,
		alert.FiredAt, alert.AcknowledgedAt, alert.ResolvedAt, rawPayload,
	).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, fmt.Errorf("insert alert: %w", err)
	}

	const existing = `
SELECT id FROM alerts
WHERE tenant_id = $1 AND source = $2 AND external_alert_id = $3`
	if err := r.db.QueryRowContext(ctx, existing, tenantID, alert.Source, alert.AlertID).Scan(&id); err != nil {
		return 0, false, fmt.Errorf("load duplicate alert: %w", err)
	}
	return id, false, nil
}
