package cloudsignal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"itsm-backend/connector/alert"
	"itsm-backend/database"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type upsertResult struct {
	ID      int64
	Created bool
}

type findingPage struct {
	Items []*SecurityFinding
	Total int
}

type alertPage struct {
	Items []*AlertRecord
	Total int
}

// ---------------------------------------------------------------------------
// 同步状态
// ---------------------------------------------------------------------------

type syncStateRepository struct {
	db *sql.DB
}

const syncStateColumns = `
	s.id, s.tenant_id, s.cloud_account_id, COALESCE(a.account_name, ''), COALESCE(a.provider, ''),
	COALESCE(a.is_active, FALSE), s.signal_kind, s.state, s.last_attempt_at, s.last_success_at,
	s.watermark, s.last_error_code, s.last_error_message, s.items_synced,
	s.consecutive_failures, s.created_at, s.updated_at`

func (r *syncStateRepository) List(ctx context.Context, tenantID int, kind SignalKind) ([]*SyncState, error) {
	if !kind.Valid() {
		return nil, fmt.Errorf("invalid signal kind %q", kind)
	}
	query := `
		SELECT ` + syncStateColumns + `
		FROM cloud_signal_sync_states s
		LEFT JOIN cloud_accounts a ON a.id = s.cloud_account_id AND a.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1 AND s.signal_kind = $2
		ORDER BY s.cloud_account_id ASC`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) ([]*SyncState, error) {
		rows, err := q.QueryContext(ctx, query, tenantID, string(kind))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		states := make([]*SyncState, 0)
		for rows.Next() {
			state, err := scanSyncState(rows)
			if err != nil {
				return nil, err
			}
			states = append(states, state)
		}
		return states, rows.Err()
	})
}

func (r *syncStateRepository) Get(ctx context.Context, tenantID int, accountID int64, kind SignalKind) (*SyncState, error) {
	if !kind.Valid() || accountID <= 0 {
		return nil, fmt.Errorf("invalid sync state lookup")
	}
	query := `
		SELECT ` + syncStateColumns + `
		FROM cloud_signal_sync_states s
		LEFT JOIN cloud_accounts a ON a.id = s.cloud_account_id AND a.tenant_id = s.tenant_id
		WHERE s.tenant_id = $1 AND s.cloud_account_id = $2 AND s.signal_kind = $3`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (*SyncState, error) {
		state, err := scanSyncState(q.QueryRowContext(ctx, query, tenantID, accountID, string(kind)))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return state, err
	})
}

func (r *syncStateRepository) MarkSuccess(ctx context.Context, tenantID int, accountID int64, kind SignalKind, watermark, now time.Time, items int64) error {
	if !kind.Valid() || accountID <= 0 {
		return fmt.Errorf("invalid sync success input")
	}
	const query = `
		INSERT INTO cloud_signal_sync_states
			(tenant_id, cloud_account_id, signal_kind, state, last_attempt_at, last_success_at,
			 watermark, items_synced, consecutive_failures, last_error_code, last_error_message,
			 created_at, updated_at)
		VALUES ($1, $2, $3, 'ready', $4, $4, $5, $6, 0, '', '', $4, $4)
		ON CONFLICT (tenant_id, cloud_account_id, signal_kind) DO UPDATE SET
			state = 'ready',
			last_attempt_at = EXCLUDED.last_attempt_at,
			last_success_at = EXCLUDED.last_success_at,
			watermark = EXCLUDED.watermark,
			items_synced = EXCLUDED.items_synced,
			consecutive_failures = 0,
			last_error_code = '',
			last_error_message = '',
			updated_at = EXCLUDED.updated_at`
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx, query, tenantID, accountID, string(kind), now, watermark, items)
		return struct{}{}, execErr
	})
	return err
}

func (r *syncStateRepository) MarkFailure(ctx context.Context, tenantID int, accountID int64, kind SignalKind, state, code, message string, now time.Time) error {
	if !kind.Valid() || accountID <= 0 {
		return fmt.Errorf("invalid sync failure input")
	}
	if state == "" {
		state = SyncStateUnavailable
	}
	const query = `
		INSERT INTO cloud_signal_sync_states
			(tenant_id, cloud_account_id, signal_kind, state, last_attempt_at, consecutive_failures,
			 last_error_code, last_error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 1, $6, $7, $5, $5)
		ON CONFLICT (tenant_id, cloud_account_id, signal_kind) DO UPDATE SET
			state = EXCLUDED.state,
			last_attempt_at = EXCLUDED.last_attempt_at,
			consecutive_failures = cloud_signal_sync_states.consecutive_failures + 1,
			last_error_code = EXCLUDED.last_error_code,
			last_error_message = EXCLUDED.last_error_message,
			updated_at = EXCLUDED.updated_at`
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx, query, tenantID, accountID, string(kind), state, now, code, message)
		return struct{}{}, execErr
	})
	return err
}

func (r *syncStateRepository) MarkUnconfigured(ctx context.Context, tenantID int, accountID int64, kind SignalKind, now time.Time) error {
	if !kind.Valid() || accountID <= 0 {
		return fmt.Errorf("invalid sync state input")
	}
	const query = `
		INSERT INTO cloud_signal_sync_states
			(tenant_id, cloud_account_id, signal_kind, state, last_attempt_at,
			 last_error_code, last_error_message, created_at, updated_at)
		VALUES ($1, $2, $3, 'unconfigured', $4, 'credential_unconfigured', '', $4, $4)
		ON CONFLICT (tenant_id, cloud_account_id, signal_kind) DO UPDATE SET
			state = 'unconfigured',
			last_attempt_at = EXCLUDED.last_attempt_at,
			last_error_code = EXCLUDED.last_error_code,
			last_error_message = '',
			updated_at = EXCLUDED.updated_at`
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx, query, tenantID, accountID, string(kind), now)
		return struct{}{}, execErr
	})
	return err
}

func scanSyncState(row rowScanner) (*SyncState, error) {
	var (
		state     SyncState
		kind      string
		attempted sql.NullTime
		succeeded sql.NullTime
		watermark sql.NullTime
	)
	if err := row.Scan(&state.ID, &state.TenantID, &state.CloudAccountID, &state.AccountName, &state.Provider,
		&state.AccountActive, &kind, &state.State, &attempted, &succeeded,
		&watermark, &state.LastErrorCode, &state.LastErrorMessage, &state.ItemsSynced,
		&state.ConsecutiveFailures, &state.CreatedAt, &state.UpdatedAt); err != nil {
		return nil, err
	}
	state.SignalKind = SignalKind(kind)
	state.LastAttemptAt = nullTimePtr(attempted)
	state.LastSuccessAt = nullTimePtr(succeeded)
	state.Watermark = nullTimePtr(watermark)
	return &state, nil
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

// ---------------------------------------------------------------------------
// 漏洞
// ---------------------------------------------------------------------------

type findingRepository struct {
	db *sql.DB
}

const findingSelectColumns = `
	f.id, f.tenant_id, f.cloud_account_id, f.provider, f.vuln_type, f.finding_key,
	f.name, f.alias_name, f.severity, f.status, f.necessity,
	f.instance_id, f.instance_name, f.uuid, f.internet_ip, f.intranet_ip, f.region,
	f.ci_id, COALESCE(ci.ci_number, ''), COALESCE(ci.name, ''),
	f.cvss_score, f.raw_payload, f.first_seen_at, f.last_seen_at, f.last_synced_at,
	f.resolved_at, f.created_at, f.updated_at`

const findingFromClause = `
	FROM security_findings f
	LEFT JOIN configuration_items ci ON ci.id = f.ci_id AND ci.tenant_id = f.tenant_id`

func (r *findingRepository) Upsert(ctx context.Context, tenantID int, finding *SecurityFinding) (int64, bool, error) {
	if tenantID <= 0 || finding == nil || finding.CloudAccountID <= 0 || finding.FindingKey == "" {
		return 0, false, fmt.Errorf("invalid security finding input")
	}
	const query = `
		INSERT INTO security_findings
			(tenant_id, cloud_account_id, provider, vuln_type, finding_key, name, alias_name,
			 severity, status, necessity, instance_id, instance_name, uuid, internet_ip, intranet_ip,
			 region, cvss_score, raw_payload, first_seen_at, last_seen_at, last_synced_at,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18::jsonb, COALESCE($19::timestamptz, $21), COALESCE($20::timestamptz, $21), $21, $21, $21)
		ON CONFLICT (tenant_id, cloud_account_id, vuln_type, finding_key) DO UPDATE SET
			name = EXCLUDED.name,
			alias_name = EXCLUDED.alias_name,
			severity = EXCLUDED.severity,
			status = CASE
				WHEN security_findings.status = 'ignored' AND EXCLUDED.status IN ('open', 'fixing') THEN 'ignored'
				ELSE EXCLUDED.status
			END,
			necessity = EXCLUDED.necessity,
			instance_id = EXCLUDED.instance_id,
			instance_name = EXCLUDED.instance_name,
			uuid = EXCLUDED.uuid,
			internet_ip = EXCLUDED.internet_ip,
			intranet_ip = EXCLUDED.intranet_ip,
			region = EXCLUDED.region,
			cvss_score = EXCLUDED.cvss_score,
			raw_payload = EXCLUDED.raw_payload,
			first_seen_at = COALESCE(security_findings.first_seen_at, EXCLUDED.first_seen_at),
			last_seen_at = GREATEST(security_findings.last_seen_at, EXCLUDED.last_seen_at),
			last_synced_at = EXCLUDED.last_synced_at,
			resolved_at = CASE
				WHEN EXCLUDED.status IN ('open', 'fixing') AND security_findings.status IS DISTINCT FROM 'ignored' THEN NULL
				ELSE COALESCE(security_findings.resolved_at, EXCLUDED.last_synced_at)
			END,
			updated_at = EXCLUDED.updated_at
		RETURNING id, (xmax = 0) AS inserted`
	result, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (upsertResult, error) {
		var out upsertResult
		err := q.QueryRowContext(ctx, query,
			tenantID, finding.CloudAccountID, finding.Provider, finding.VulnType, finding.FindingKey,
			finding.Name, finding.AliasName, finding.Severity, finding.Status, finding.Necessity,
			finding.InstanceID, finding.InstanceName, finding.UUID, finding.InternetIP, finding.IntranetIP,
			finding.Region, finding.CVSSScore, rawPayloadArg(finding.RawPayload),
			finding.FirstSeenAt, finding.LastSeenAt, finding.LastSyncedAt,
		).Scan(&out.ID, &out.Created)
		return out, err
	})
	return result.ID, result.Created, err
}

func rawPayloadArg(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func (r *findingRepository) Get(ctx context.Context, tenantID int, id int64) (*SecurityFinding, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid finding id")
	}
	query := `SELECT ` + findingSelectColumns + findingFromClause + `
		WHERE f.tenant_id = $1 AND f.id = $2`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (*SecurityFinding, error) {
		finding, err := scanFinding(q.QueryRowContext(ctx, query, tenantID, id))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return finding, err
	})
}

func (r *findingRepository) List(ctx context.Context, tenantID int, filter FindingFilter) ([]*SecurityFinding, int, error) {
	where, args := buildFindingWhere(tenantID, filter)
	_, pageSize, offset := normalizePagination(filter.Page, filter.PageSize)

	countQuery := `SELECT COUNT(*)` + findingFromClause + ` WHERE ` + where
	listQuery := `SELECT ` + findingSelectColumns + findingFromClause + ` WHERE ` + where + `
		ORDER BY CASE f.severity
			WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END,
			f.last_seen_at DESC NULLS LAST, f.id DESC
		LIMIT $` + fmt.Sprint(len(args)+1) + ` OFFSET $` + fmt.Sprint(len(args)+2)

	listArgs := append(append([]any{}, args...), pageSize, offset)

	page, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (findingPage, error) {
		var result findingPage
		if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&result.Total); err != nil {
			return result, err
		}
		rows, err := q.QueryContext(ctx, listQuery, listArgs...)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		result.Items = make([]*SecurityFinding, 0, pageSize)
		for rows.Next() {
			finding, err := scanFinding(rows)
			if err != nil {
				return result, err
			}
			result.Items = append(result.Items, finding)
		}
		return result, rows.Err()
	})
	return page.Items, page.Total, err
}

func (r *findingRepository) Summary(ctx context.Context, tenantID int) (*FindingSummary, error) {
	const query = `
		SELECT severity, status, COUNT(*)::int
		FROM security_findings
		WHERE tenant_id = $1 AND status IN ('open', 'fixing')
		GROUP BY severity, status`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (*FindingSummary, error) {
		rows, err := q.QueryContext(ctx, query, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		summary := &FindingSummary{BySeverity: map[string]int{}, ByStatus: map[string]int{}}
		for rows.Next() {
			var (
				severity, status string
				count            int
			)
			if err := rows.Scan(&severity, &status, &count); err != nil {
				return nil, err
			}
			summary.Total += count
			summary.BySeverity[severity] += count
			summary.ByStatus[status] += count
		}
		return summary, rows.Err()
	})
}

func (r *findingRepository) SetStatus(ctx context.Context, tenantID int, id int64, status string, now time.Time) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("invalid finding id")
	}
	const query = `
		UPDATE security_findings
		SET status = $3::text,
		    resolved_at = CASE WHEN $3::text IN ('fixed', 'ignored') THEN $4 ELSE NULL END,
		    updated_at = $4
		WHERE tenant_id = $1 AND id = $2`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (bool, error) {
		res, err := q.ExecContext(ctx, query, tenantID, id, status, now)
		if err != nil {
			return false, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("read rows affected: %w", err)
		}
		return affected > 0, nil
	})
}

func (r *findingRepository) MarkUnseenResolved(ctx context.Context, tenantID int, accountID int64, vulnType string, syncedBefore, now time.Time) (int64, error) {
	if accountID <= 0 || vulnType == "" {
		return 0, fmt.Errorf("invalid finding sweep input")
	}
	// 只有本轮完整拉取覆盖到的类型才能清扫：last_synced_at 早于本轮起点，
	// 说明这次云上已经不再返回它。
	const query = `
		UPDATE security_findings
		SET status = 'fixed', resolved_at = $5, updated_at = $5
		WHERE tenant_id = $1 AND cloud_account_id = $2 AND vuln_type = $3
		  AND status IN ('open', 'fixing')
		  AND last_synced_at < $4`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (int64, error) {
		res, err := q.ExecContext(ctx, query, tenantID, accountID, vulnType, syncedBefore, now)
		if err != nil {
			return 0, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("read rows affected: %w", err)
		}
		return affected, nil
	})
}

func (r *findingRepository) LinkCIs(ctx context.Context, tenantID int, accountID int64) (int64, error) {
	if accountID <= 0 {
		return 0, fmt.Errorf("invalid finding link input")
	}
	// 一个实例号可能被人工建过多个 CI，取最早的一条作为稳定归属。
	const query = `
		UPDATE security_findings f
		SET ci_id = ci.id, updated_at = NOW()
		FROM (
			SELECT DISTINCT ON (cloud_resource_id) id, cloud_provider AS provider, cloud_resource_id
			FROM configuration_items
			WHERE tenant_id = $1 AND cloud_resource_id <> '' AND cloud_resource_id IS NOT NULL
			ORDER BY cloud_resource_id, id ASC
		) ci
		WHERE f.tenant_id = $1 AND f.cloud_account_id = $2
		  AND f.instance_id <> ''
		  AND f.provider = ci.provider
		  AND f.instance_id = ci.cloud_resource_id
		  AND f.ci_id IS DISTINCT FROM ci.id`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (int64, error) {
		res, err := q.ExecContext(ctx, query, tenantID, accountID)
		if err != nil {
			return 0, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("read rows affected: %w", err)
		}
		return affected, nil
	})
}

func buildFindingWhere(tenantID int, filter FindingFilter) (string, []any) {
	clauses := []string{"f.tenant_id = $1"}
	args := []any{tenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if len(filter.Statuses) > 0 {
		add("f.status = ANY($%d)", pq.Array(filter.Statuses))
	}
	if len(filter.Severities) > 0 {
		add("f.severity = ANY($%d)", pq.Array(filter.Severities))
	}
	if filter.VulnType != "" {
		add("f.vuln_type = $%d", filter.VulnType)
	}
	if filter.InstanceID != "" {
		add("f.instance_id = $%d", filter.InstanceID)
	}
	if filter.CloudAccountID > 0 {
		add("f.cloud_account_id = $%d", filter.CloudAccountID)
	}
	if term := strings.TrimSpace(filter.Search); term != "" {
		args = append(args, "%"+escapeLike(term)+"%")
		n := len(args)
		clauses = append(clauses, fmt.Sprintf(
			"(f.name ILIKE $%d ESCAPE '\\' OR f.alias_name ILIKE $%d ESCAPE '\\' "+
				"OR f.instance_name ILIKE $%d ESCAPE '\\' OR f.instance_id ILIKE $%d ESCAPE '\\' "+
				"OR f.finding_key ILIKE $%d ESCAPE '\\')", n, n, n, n, n))
	}
	return strings.Join(clauses, " AND "), args
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func scanFinding(row rowScanner) (*SecurityFinding, error) {
	var (
		finding    SecurityFinding
		ciID       sql.NullInt64
		cvss       sql.NullFloat64
		raw        []byte
		firstSeen  sql.NullTime
		lastSeen   sql.NullTime
		resolvedAt sql.NullTime
	)
	if err := row.Scan(&finding.ID, &finding.TenantID, &finding.CloudAccountID, &finding.Provider,
		&finding.VulnType, &finding.FindingKey, &finding.Name, &finding.AliasName, &finding.Severity,
		&finding.Status, &finding.Necessity, &finding.InstanceID, &finding.InstanceName, &finding.UUID,
		&finding.InternetIP, &finding.IntranetIP, &finding.Region, &ciID, &finding.CINumber, &finding.CIName,
		&cvss, &raw, &firstSeen, &lastSeen, &finding.LastSyncedAt, &resolvedAt,
		&finding.CreatedAt, &finding.UpdatedAt); err != nil {
		return nil, err
	}
	if ciID.Valid {
		id := ciID.Int64
		finding.CIID = &id
	}
	if cvss.Valid {
		score := cvss.Float64
		finding.CVSSScore = &score
	}
	if len(raw) > 0 {
		finding.RawPayload = json.RawMessage(raw)
	}
	finding.FirstSeenAt = nullTimePtr(firstSeen)
	finding.LastSeenAt = nullTimePtr(lastSeen)
	finding.ResolvedAt = nullTimePtr(resolvedAt)
	return &finding, nil
}

// ---------------------------------------------------------------------------
// 告警（复用 alerts 表）
// ---------------------------------------------------------------------------

type alertRepository struct {
	db *sql.DB
}

const alertSelectColumns = `
	id, tenant_id, external_alert_id, name, description, severity, status,
	labels, annotations, source_ip, service, fired_at, acknowledged_at, resolved_at,
	created_at, updated_at`

func (r *alertRepository) Upsert(ctx context.Context, tenantID int, a *alert.StandardAlert, now time.Time) (int64, bool, error) {
	if tenantID <= 0 || a == nil || a.AlertID == "" {
		return 0, false, fmt.Errorf("invalid alert input")
	}
	labels, err := marshalStringMap(a.Labels)
	if err != nil {
		return 0, false, err
	}
	annotations, err := marshalStringMap(a.Annotations)
	if err != nil {
		return 0, false, err
	}
	tags, err := json.Marshal(nonNilStrings(a.Tags))
	if err != nil {
		return 0, false, err
	}
	rawPayload := "{}"
	if len(a.RawPayload) > 0 {
		encoded, err := json.Marshal(a.RawPayload)
		if err != nil {
			return 0, false, err
		}
		rawPayload = string(encoded)
	}
	source := a.Source
	if source == "" {
		source = AlertSourceCloudMonitor
	}

	// 重复告警只刷新展示字段，保留本地确认/解决状态与建单桥的记录：
	// 已经关联过工单的告警被下一轮轮询改回 firing 会让操作员误以为工单丢了。
	const query = `
		INSERT INTO alerts
			(tenant_id, source, external_alert_id, source_raw, name, description, severity, status,
			 labels, annotations, source_ip, service, tags, fired_at, raw_payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11, $12, $13::jsonb, $14, $15::jsonb, $16, $16)
		ON CONFLICT (tenant_id, source, external_alert_id) DO UPDATE SET
			source_raw = EXCLUDED.source_raw,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			severity = EXCLUDED.severity,
			labels = EXCLUDED.labels,
			annotations = COALESCE(alerts.annotations, '{}'::jsonb) || COALESCE(EXCLUDED.annotations, '{}'::jsonb),
			source_ip = EXCLUDED.source_ip,
			service = EXCLUDED.service,
			tags = EXCLUDED.tags,
			fired_at = LEAST(alerts.fired_at, EXCLUDED.fired_at),
			raw_payload = EXCLUDED.raw_payload,
			updated_at = EXCLUDED.updated_at
		RETURNING id, (xmax = 0) AS inserted`
	result, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (upsertResult, error) {
		var out upsertResult
		err := q.QueryRowContext(ctx, query,
			tenantID, source, a.AlertID, a.SourceRaw, a.Name, a.Description, string(a.Severity), a.Status,
			labels, annotations, a.SourceIP, a.Service, string(tags), a.FiredAt, rawPayload, now,
		).Scan(&out.ID, &out.Created)
		return out, err
	})
	return result.ID, result.Created, err
}

func (r *alertRepository) List(ctx context.Context, tenantID int, filter AlertFilter) ([]*AlertRecord, int, error) {
	where := "tenant_id = $1 AND source = $2"
	args := []any{tenantID, AlertSourceCloudMonitor}
	add := func(clause string, value any) {
		args = append(args, value)
		where += " AND " + fmt.Sprintf(clause, len(args))
	}
	if len(filter.Statuses) > 0 {
		add("status = ANY($%d)", pq.Array(filter.Statuses))
	}
	if len(filter.Severities) > 0 {
		add("severity = ANY($%d)", pq.Array(filter.Severities))
	}
	if filter.CloudAccountID > 0 {
		add("labels->>'cloudAccountId' = $%d", strconv.FormatInt(filter.CloudAccountID, 10))
	}
	if term := strings.TrimSpace(filter.Search); term != "" {
		args = append(args, "%"+escapeLike(term)+"%")
		n := len(args)
		where += fmt.Sprintf(" AND (name ILIKE $%d ESCAPE '\\' OR external_alert_id ILIKE $%d ESCAPE '\\' "+
			"OR description ILIKE $%d ESCAPE '\\')", n, n, n)
	}
	_, pageSize, offset := normalizePagination(filter.Page, filter.PageSize)
	args = append(args, pageSize, offset)
	listQuery := `SELECT ` + alertSelectColumns + ` FROM alerts WHERE ` + where + `
		ORDER BY fired_at DESC, id DESC LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	countQuery := `SELECT COUNT(*) FROM alerts WHERE ` + where

	page, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (alertPage, error) {
		var result alertPage
		if err := q.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&result.Total); err != nil {
			return result, err
		}
		rows, err := q.QueryContext(ctx, listQuery, args...)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		result.Items = make([]*AlertRecord, 0, pageSize)
		for rows.Next() {
			record, err := scanAlert(rows)
			if err != nil {
				return result, err
			}
			result.Items = append(result.Items, record)
		}
		return result, rows.Err()
	})
	return page.Items, page.Total, err
}

func (r *alertRepository) Get(ctx context.Context, tenantID int, id int64) (*AlertRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid alert id")
	}
	query := `SELECT ` + alertSelectColumns + ` FROM alerts
		WHERE tenant_id = $1 AND source = $2 AND id = $3`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (*AlertRecord, error) {
		record, err := scanAlert(q.QueryRowContext(ctx, query, tenantID, AlertSourceCloudMonitor, id))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return record, err
	})
}

func (r *alertRepository) Acknowledge(ctx context.Context, tenantID int, id int64, now time.Time) (bool, error) {
	const query = `
		UPDATE alerts
		SET status = 'acknowledged', acknowledged_at = COALESCE(acknowledged_at, $4), updated_at = $4
		WHERE tenant_id = $1 AND source = $2 AND id = $3 AND status = 'firing'`
	return r.transition(ctx, tenantID, query, tenantID, AlertSourceCloudMonitor, id, now)
}

func (r *alertRepository) Resolve(ctx context.Context, tenantID int, id int64, now time.Time) (bool, error) {
	const query = `
		UPDATE alerts
		SET status = 'resolved', resolved_at = COALESCE(resolved_at, $4), updated_at = $4
		WHERE tenant_id = $1 AND source = $2 AND id = $3 AND status IN ('firing', 'acknowledged')`
	return r.transition(ctx, tenantID, query, tenantID, AlertSourceCloudMonitor, id, now)
}

func (r *alertRepository) transition(ctx context.Context, tenantID int, query string, args ...any) (bool, error) {
	if tenantID <= 0 {
		return false, fmt.Errorf("invalid alert transition input")
	}
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (bool, error) {
		res, err := q.ExecContext(ctx, query, args...)
		if err != nil {
			return false, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("read rows affected: %w", err)
		}
		return affected > 0, nil
	})
}

func (r *alertRepository) ClaimBridgeCandidates(ctx context.Context, tenantID int, severities []string, limit, maxAttempts int, staleBefore, now time.Time) ([]*BridgeCandidate, error) {
	if tenantID <= 0 || limit <= 0 || len(severities) == 0 {
		return nil, fmt.Errorf("invalid bridge claim input")
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	// 认领条件与子查询条件重复一遍：并发 worker 在锁释放后会重新评估外层
	// WHERE，只有这一份才是最终裁决，子查询只负责挑选与加锁。
	const query = `
		UPDATE alerts a
		SET annotations = COALESCE(a.annotations, '{}'::jsonb) || jsonb_build_object(
				'bridgeStatus', 'claimed',
				'bridgeAttempts', (COALESCE(a.annotations->>'bridgeAttempts', '0')::int + 1)::text,
				'bridgeClaimedAt', to_char($5::timestamptz AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			),
			updated_at = $5
		WHERE a.id IN (
				SELECT id FROM alerts
				WHERE tenant_id = $1 AND source = $2 AND status = 'firing'
				  AND severity = ANY($3)
				  AND (
						COALESCE(annotations->>'bridgeStatus', '') IN ('', 'failed')
						OR (annotations->>'bridgeStatus' = 'claimed'
							AND COALESCE((annotations->>'bridgeClaimedAt')::timestamptz, to_timestamp(0)) < $4)
					)
				  AND COALESCE((annotations->>'bridgeAttempts')::int, 0) < $6
				ORDER BY fired_at ASC, id ASC
				LIMIT $7
				FOR UPDATE SKIP LOCKED
			)
		  AND a.tenant_id = $1 AND a.source = $2 AND a.status = 'firing'
		  AND a.severity = ANY($3)
		  AND (
				COALESCE(a.annotations->>'bridgeStatus', '') IN ('', 'failed')
				OR (a.annotations->>'bridgeStatus' = 'claimed'
					AND COALESCE((a.annotations->>'bridgeClaimedAt')::timestamptz, to_timestamp(0)) < $4)
			)
		  AND COALESCE((a.annotations->>'bridgeAttempts')::int, 0) < $6
		RETURNING ` + alertSelectColumns

	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) ([]*BridgeCandidate, error) {
		rows, err := q.QueryContext(ctx, query, tenantID, AlertSourceCloudMonitor, pq.Array(severities),
			staleBefore, now, maxAttempts, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		candidates := make([]*BridgeCandidate, 0, limit)
		for rows.Next() {
			record, err := scanAlert(rows)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, &BridgeCandidate{AlertRecord: *record})
		}
		return candidates, rows.Err()
	})
}

func (r *alertRepository) MarkBridged(ctx context.Context, tenantID int, id, incidentID int64, now time.Time) error {
	const query = `
		UPDATE alerts
		SET annotations = COALESCE(annotations, '{}'::jsonb) || jsonb_build_object(
				'bridgeStatus', 'bridged',
				'incidentId', $3::text,
				'bridgedAt', to_char($4::timestamptz AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			),
			updated_at = $4
		WHERE tenant_id = $1 AND id = $2`
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx, query, tenantID, id, incidentID, now)
		return struct{}{}, execErr
	})
	return err
}

func (r *alertRepository) MarkBridgeFailed(ctx context.Context, tenantID int, id int64, reason string, now time.Time) error {
	reason = truncateRunes(reason, 200)
	const query = `
		UPDATE alerts
		SET annotations = COALESCE(annotations, '{}'::jsonb) || jsonb_build_object(
				'bridgeStatus', 'failed',
				'bridgeError', $3::text
			),
			updated_at = $4
		WHERE tenant_id = $1 AND id = $2`
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx, query, tenantID, id, reason, now)
		return struct{}{}, execErr
	})
	return err
}

func scanAlert(row rowScanner) (*AlertRecord, error) {
	var (
		record       AlertRecord
		labels       []byte
		annotations  []byte
		acknowledged sql.NullTime
		resolvedAt   sql.NullTime
	)
	if err := row.Scan(&record.ID, &record.TenantID, &record.AlertID, &record.Name, &record.Description,
		&record.Severity, &record.Status, &labels, &annotations, &record.SourceIP, &record.Service,
		&record.FiredAt, &acknowledged, &resolvedAt, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return nil, err
	}
	record.Labels = decodeStringMap(labels)
	record.Annotations = decodeStringMap(annotations)
	record.AcknowledgedAt = nullTimePtr(acknowledged)
	record.ResolvedAt = nullTimePtr(resolvedAt)
	return &record, nil
}

func marshalStringMap(value map[string]string) (string, error) {
	if value == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// decodeStringMap 容忍非字符串值（例如外部写入的数字），统一转成字符串，
// 避免一条格式不规范的告警让整页列表解析失败。
func decodeStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	if err := json.Unmarshal(raw, &out); err == nil {
		return out
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return map[string]string{}
	}
	for key, value := range generic {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok {
			out[key] = text
			continue
		}
		out[key] = fmt.Sprint(value)
	}
	return out
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func normalizePagination(page, pageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}
