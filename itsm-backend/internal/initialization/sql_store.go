package initialization

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type SQLStore struct {
	db *sql.DB
}

type InstallationStatus struct {
	ScopeType        string         `json:"scopeType"`
	ScopeID          int64          `json:"scopeId"`
	Component        string         `json:"component"`
	InstalledVersion string         `json:"installedVersion"`
	SourceChecksum   string         `json:"sourceChecksum"`
	Status           string         `json:"status"`
	FencingToken     int64          `json:"fencingToken"`
	LeaseOwner       string         `json:"leaseOwner,omitempty"`
	LeaseExpiresAt   *time.Time     `json:"leaseExpiresAt,omitempty"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	ResultSummary    map[string]any `json:"resultSummary,omitempty"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Status(ctx context.Context, scope Scope) ([]InstallationStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT scope_type, scope_id, component, installed_version, source_checksum,
		       status, fencing_token, lease_owner, lease_expires_at,
		       error_code, error_message, result_summary, updated_at
		FROM initialization_installations
		WHERE scope_type = $1 AND scope_id = $2
		ORDER BY component
	`, scope.Type, scope.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	statuses := make([]InstallationStatus, 0)
	for rows.Next() {
		var status InstallationStatus
		var lease sql.NullTime
		var summary []byte
		if err := rows.Scan(
			&status.ScopeType, &status.ScopeID, &status.Component,
			&status.InstalledVersion, &status.SourceChecksum, &status.Status,
			&status.FencingToken, &status.LeaseOwner, &lease,
			&status.ErrorCode, &status.ErrorMessage, &summary, &status.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lease.Valid {
			status.LeaseExpiresAt = &lease.Time
		}
		if len(summary) > 0 {
			if err := json.Unmarshal(summary, &status.ResultSummary); err != nil {
				return nil, fmt.Errorf("decode initialization status summary: %w", err)
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, rows.Err()
}

func (s *SQLStore) BeginRun(ctx context.Context, request Request) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO initialization_runs
			(scope_type, scope_id, target_version, release_version, requested_by, executor_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'running')
		RETURNING id
	`, request.Scope.Type, request.Scope.ID, request.TargetVersion, request.ReleaseVersion,
		request.RequestedBy, request.ExecutorID).Scan(&id)
	return id, err
}

func (s *SQLStore) FinishRun(
	ctx context.Context,
	runID int64,
	status string,
	summary map[string]any,
	runErr error,
) error {
	payload, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE initialization_runs
		SET status = $1, completed_at = NOW(), result_summary = $2, error_message = $3
		WHERE id = $4 AND status = 'running'
	`, status, payload, errorMessage(runErr), runID)
	return err
}

func (s *SQLStore) AcquireLease(
	ctx context.Context,
	scope Scope,
	component, executorID string,
	ttl time.Duration,
) (Lease, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Lease{}, err
	}
	defer tx.Rollback()
	var token int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO initialization_installations
			(scope_type, scope_id, component, status, fencing_token, lease_owner, heartbeat_at, lease_expires_at)
		VALUES ($1, $2, $3, 'running', 1, $4, NOW(), NOW() + ($5 * INTERVAL '1 millisecond'))
		ON CONFLICT (scope_type, scope_id, component) DO UPDATE
		SET status = 'running',
		    fencing_token = initialization_installations.fencing_token + 1,
		    lease_owner = EXCLUDED.lease_owner,
		    heartbeat_at = NOW(),
		    lease_expires_at = EXCLUDED.lease_expires_at,
		    updated_at = NOW()
		WHERE initialization_installations.lease_expires_at IS NULL
		   OR initialization_installations.lease_expires_at < NOW()
		RETURNING fencing_token
	`, scope.Type, scope.ID, component, executorID, ttl.Milliseconds()).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		return Lease{}, ErrLeaseHeld
	}
	if err != nil {
		return Lease{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		WITH abandoned AS (
			UPDATE initialization_component_attempts
			SET status = 'failed', completed_at = NOW(), error_message = 'initialization lease superseded'
			WHERE scope_type = $1 AND scope_id = $2 AND component = $3
			  AND status = 'running' AND fencing_token < $4
			RETURNING run_id
		)
		UPDATE initialization_runs SET status = 'failed', completed_at = NOW(),
		    error_message = 'initialization lease superseded'
		WHERE id IN (SELECT run_id FROM abandoned) AND status = 'running'
	`, scope.Type, scope.ID, component, token); err != nil {
		return Lease{}, err
	}
	return Lease{FencingToken: token}, tx.Commit()
}

func (s *SQLStore) Heartbeat(
	ctx context.Context,
	scope Scope,
	component, executorID string,
	token int64,
	ttl time.Duration,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE initialization_installations
		SET heartbeat_at = NOW(),
		    lease_expires_at = NOW() + ($1 * INTERVAL '1 millisecond'),
		    updated_at = NOW()
		WHERE scope_type = $2 AND scope_id = $3 AND component = $4
		  AND lease_owner = $5 AND fencing_token = $6 AND status = 'running'
		  AND lease_expires_at > clock_timestamp()
	`, ttl.Milliseconds(), scope.Type, scope.ID, component, executorID, token)
	return requireOneFencedRow(result, err)
}

func (s *SQLStore) ReleaseLease(
	ctx context.Context,
	scope Scope,
	component, executorID string,
	token int64,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE initialization_installations
		SET lease_owner = '', lease_expires_at = NULL, heartbeat_at = NOW(), updated_at = NOW()
		WHERE scope_type = $1 AND scope_id = $2 AND component = $3
		  AND lease_owner = $4 AND fencing_token = $5
	`, scope.Type, scope.ID, component, executorID, token)
	return requireOneFencedRow(result, err)
}

func (s *SQLStore) StartAttempt(
	ctx context.Context,
	runID int64,
	scope Scope,
	plan Plan,
	token int64,
) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO initialization_component_attempts
			(run_id, scope_type, scope_id, component, attempt, from_version,
			 target_version, source_checksum, fencing_token, status)
		SELECT $1, $2::text, $3, $4::text,
		       COALESCE((SELECT MAX(attempt) + 1 FROM initialization_component_attempts
		                 WHERE scope_type = $2::text AND scope_id = $3 AND component = $4::text), 1),
		       $5, $6, $7, $8, 'running'
		FROM initialization_installations
		WHERE scope_type = $2::text AND scope_id = $3 AND component = $4::text
		  AND fencing_token = $8 AND status = 'running' AND lease_expires_at > clock_timestamp()
		FOR UPDATE
		RETURNING id
	`, runID, scope.Type, scope.ID, plan.Component, plan.FromVersion,
		plan.TargetVersion, plan.SourceChecksum, token).Scan(&id)
	return id, err
}

type componentTransaction struct {
	*sql.Tx
	attemptID, runID int64
	scope            Scope
	plan             Plan
	executorID       string
	token            int64
}

func (s *SQLStore) BeginComponent(ctx context.Context, attemptID, runID int64, scope Scope, plan Plan, executorID string, token int64) (ComponentTransaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &componentTransaction{Tx: tx, attemptID: attemptID, runID: runID, scope: scope, plan: plan, executorID: executorID, token: token}, nil
}

func (t *componentTransaction) Driver() dialect.Driver {
	return NewTransactionDriver(&entsql.Tx{Conn: entsql.Conn{ExecQuerier: t.Tx}, Tx: t.Tx}, dialect.Postgres)
}

// NewTransactionDriver keeps nested Ent mutations inside the caller-owned transaction.
func NewTransactionDriver(tx dialect.Tx, name string) dialect.Driver {
	return &transactionDriver{ExecQuerier: tx, name: name}
}

type transactionDriver struct {
	dialect.ExecQuerier
	name string
}

func (d *transactionDriver) Dialect() string                        { return d.name }
func (d *transactionDriver) Close() error                           { return nil }
func (d *transactionDriver) Tx(context.Context) (dialect.Tx, error) { return dialect.NopTx(d), nil }

func (t *componentTransaction) Complete(ctx context.Context, result Result) error {
	summary, err := json.Marshal(result.Summary)
	if err != nil {
		return err
	}
	rollback, err := json.Marshal(result.RollbackMetadata)
	if err != nil {
		return err
	}
	// The fenced update locks the lease through the business commit, excluding takeover.
	installationResult, err := t.ExecContext(ctx, `
		UPDATE initialization_installations
		SET installed_version = $1, source_checksum = $2, status = 'succeeded',
		    last_run_id = $3, result_summary = $4, error_message = '', error_code = '',
		    lease_owner = '', lease_expires_at = NULL, heartbeat_at = clock_timestamp(), updated_at = clock_timestamp()
		WHERE scope_type = $5 AND scope_id = $6 AND component = $7
		  AND lease_owner = $8 AND fencing_token = $9 AND status = 'running'
		  AND lease_expires_at > clock_timestamp()
	`, t.plan.TargetVersion, t.plan.SourceChecksum, t.runID, summary,
		t.scope.Type, t.scope.ID, t.plan.Component, t.executorID, t.token)
	if err := requireOneFencedRow(installationResult, err); err != nil {
		return err
	}
	attemptResult, err := t.ExecContext(ctx, `
		UPDATE initialization_component_attempts
		SET status = 'succeeded', completed_at = clock_timestamp(), result_summary = $1,
		    rollback_metadata = $2, error_message = ''
		WHERE id = $3 AND run_id = $4 AND fencing_token = $5 AND status = 'running'
	`, summary, rollback, t.attemptID, t.runID, t.token)
	return requireOneFencedRow(attemptResult, err)
}

func (s *SQLStore) FailComponent(ctx context.Context, attemptID, runID int64, scope Scope, plan Plan, executorID string, token int64, applyErr error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// A stale executor may close its own attempt, but must not change its successor's lease.
	if _, err := tx.ExecContext(ctx, `
		UPDATE initialization_installations
		SET status = 'failed', last_run_id = $1, error_message = $2,
		    lease_owner = '', lease_expires_at = NULL, updated_at = clock_timestamp()
		WHERE scope_type = $3 AND scope_id = $4 AND component = $5
		  AND lease_owner = $6 AND fencing_token = $7 AND status = 'running'
	`, runID, errorMessage(applyErr), scope.Type, scope.ID, plan.Component, executorID, token); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE initialization_component_attempts
		SET status = 'failed', completed_at = clock_timestamp(), error_message = $1
		WHERE id = $2 AND run_id = $3 AND fencing_token = $4 AND status = 'running'
	`, errorMessage(applyErr), attemptID, runID, token); err != nil {
		return err
	}
	return tx.Commit()
}

func requireOneFencedRow(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("initialization lease lost or fencing token rejected")
	}
	return nil
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
