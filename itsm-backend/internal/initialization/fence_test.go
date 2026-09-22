package initialization

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestInitializationTestDatabaseGuard(t *testing.T) {
	for _, name := range []string{"itsm", "postgres", "", "itsm_init_test_", "customer_itsm_init_test_1", "ITSM_INIT_TEST_1"} {
		t.Run("reject_"+name, func(t *testing.T) {
			require.Error(t, validateInitializationTestDatabase(name))
		})
	}
	require.NoError(t, validateInitializationTestDatabase("itsm_init_test_regressions"))
}

func TestPostgresSQLStoreRejectsLostAndExpiredHeartbeats(t *testing.T) {
	for _, scenario := range []string{"wrong_owner", "wrong_token", "expired", "taken_over", "released"} {
		t.Run(scenario, func(t *testing.T) {
			f := newInitializationPostgresFixture(t)
			request := initializationRequest("executor-old")
			const component = "rbac"
			lease, err := f.store.AcquireLease(f.ctx, request.Scope, component, request.ExecutorID, time.Hour)
			require.NoError(t, err)
			require.NoError(t, f.store.Heartbeat(f.ctx, request.Scope, component, request.ExecutorID, lease.FencingToken, 2*time.Hour))
			owner, token := request.ExecutorID, lease.FencingToken
			switch scenario {
			case "wrong_owner":
				owner = "other-executor"
			case "wrong_token":
				token++
			case "expired":
				f.expireLease(t, request.Scope, component)
			case "taken_over":
				f.expireLease(t, request.Scope, component)
				fresh, err := f.second.AcquireLease(f.ctx, request.Scope, component, "executor-new", time.Hour)
				require.NoError(t, err)
				require.Greater(t, fresh.FencingToken, lease.FencingToken)
			case "released":
				require.NoError(t, f.store.ReleaseLease(f.ctx, request.Scope, component, owner, token))
			}
			before := f.installation(t, request.Scope)
			err = f.store.Heartbeat(f.ctx, request.Scope, component, owner, token, time.Hour)
			require.ErrorContains(t, err, "lease lost or fencing token rejected")
			require.Equal(t, before, f.installation(t, request.Scope), "rejected heartbeat must not renew or mutate the lease")
		})
	}
}

func TestPostgresEngineTakeoverRollsBackStaleBusinessWrite(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-old")
	written := make(chan struct{})
	resume := make(chan struct{})
	var resumeOnce sync.Once
	release := func() { resumeOnce.Do(func() { close(resume) }) }
	runCtx, cancelRun := context.WithCancel(f.ctx)
	defer cancelRun()
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		result, err := writeAndVerifyInitializationFixture(ctx, scope, plan, driver)
		if err != nil {
			return result, err
		}
		close(written)
		select {
		case <-resume:
			return result, nil
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}}
	engine, err := NewEngine(f.store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	type outcome struct {
		runID int64
		err   error
	}
	done := make(chan outcome, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		runID, err := engine.Apply(runCtx, request)
		done <- outcome{runID, err}
	}()
	t.Cleanup(func() {
		cancelRun()
		release()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		select {
		case <-finished:
		case <-cleanupCtx.Done():
			t.Error("initializer did not stop before database cleanup")
		}
	})
	select {
	case <-written:
	case result := <-done:
		t.Fatalf("initializer exited before its business write: %v", result.err)
	case <-f.ctx.Done():
		t.Fatal("initializer did not reach its business write")
	}
	require.Zero(t, f.businessCount(t), "even nested transaction Commit must not publish the business write")
	oldLease := f.installation(t, request.Scope)
	var oldRunID int64
	require.NoError(t, f.secondDB.QueryRowContext(f.ctx, "SELECT run_id FROM initialization_component_attempts").Scan(&oldRunID))
	f.assertRunAndAttempt(t, oldRunID, "running")

	// A real second SQLStore takes over while the first transaction is still open.
	f.expireLease(t, request.Scope, component.Name())
	fresh, err := f.second.AcquireLease(f.ctx, request.Scope, component.Name(), "executor-new", time.Hour)
	require.NoError(t, err)
	require.Greater(t, fresh.FencingToken, oldLease.FencingToken)
	f.assertRunAndAttempt(t, oldRunID, "failed")
	newRequest := initializationRequest("executor-new")
	newRunID, err := f.second.BeginRun(f.ctx, newRequest)
	require.NoError(t, err)
	plan, err := component.Plan(f.ctx, request.Scope)
	require.NoError(t, err)
	plan.Component = component.Name()
	newAttemptID, err := f.second.StartAttempt(f.ctx, newRunID, request.Scope, plan, fresh.FencingToken)
	require.NoError(t, err)
	before := f.installationSnapshot(t, request.Scope)

	release()
	select {
	case result := <-done:
		require.Equal(t, oldRunID, result.runID)
		require.ErrorContains(t, result.err, "lease lost or fencing token rejected")
	case <-f.ctx.Done():
		t.Fatal("stale initializer did not finish")
	}
	require.Zero(t, f.businessCount(t))
	f.assertRunAndAttempt(t, oldRunID, "failed")
	f.assertRunAndAttempt(t, newRunID, "running")
	require.Equal(t, before, f.installationSnapshot(t, request.Scope), "old failure writeback must not pollute the successor lease")
	var abandonedReason string
	require.NoError(t, f.secondDB.QueryRowContext(f.ctx, "SELECT error_message FROM initialization_runs WHERE id = $1", oldRunID).Scan(&abandonedReason))
	require.Equal(t, "initialization lease superseded", abandonedReason, "late FinishRun must not overwrite takeover evidence")
	require.ErrorContains(t, f.store.ReleaseLease(f.ctx, request.Scope, component.Name(), request.ExecutorID, oldLease.FencingToken), "lease lost")
	require.Equal(t, before, f.installationSnapshot(t, request.Scope))

	// The successor still owns a usable lease and can commit normally.
	require.NoError(t, f.second.Heartbeat(f.ctx, request.Scope, component.Name(), newRequest.ExecutorID, fresh.FencingToken, time.Hour))
	tx, err := f.second.BeginComponent(f.ctx, newAttemptID, newRunID, request.Scope, plan, newRequest.ExecutorID, fresh.FencingToken)
	require.NoError(t, err)
	defer tx.Rollback()
	result, err := writeAndVerifyInitializationFixture(f.ctx, request.Scope, plan, tx.Driver())
	require.NoError(t, err)
	require.NoError(t, tx.Complete(f.ctx, result))
	require.NoError(t, tx.Commit())
	require.NoError(t, f.second.FinishRun(f.ctx, newRunID, "succeeded", result.Summary, nil))
	f.assertRunAndAttempt(t, newRunID, "succeeded")
	require.Equal(t, 1, f.businessCount(t))
}

func TestPostgresEngineRejectsExpiredFenceAfterBusinessWrite(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-old")
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		result, err := writeAndVerifyInitializationFixture(ctx, scope, plan, driver)
		if err != nil {
			return result, err
		}
		f.expireLease(t, scope, plan.Component)
		return result, nil
	}}
	engine, err := NewEngine(f.store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	runID, err := engine.Apply(f.ctx, request)
	require.ErrorContains(t, err, "lease lost or fencing token rejected")
	require.Zero(t, f.businessCount(t))
	f.assertRunAndAttempt(t, runID, "failed")
	status := f.installation(t, request.Scope)
	require.Equal(t, "failed", status.Status)
	require.Empty(t, status.InstalledVersion)
	require.Empty(t, status.SourceChecksum)
}

func TestPostgresEngineFailureRollsBackBusinessAndSuccessLedger(t *testing.T) {
	verificationErr := errors.New("in-transaction verification rejected initialized data")
	for _, scenario := range []struct {
		name, ddl, constraint string
	}{
		{name: "verification"},
		{
			name:       "installation_ledger",
			ddl:        `ALTER TABLE initialization_installations ADD CONSTRAINT reject_installation_success CHECK (status <> 'succeeded')`,
			constraint: "reject_installation_success",
		},
		{
			name:       "attempt_ledger",
			ddl:        `ALTER TABLE initialization_component_attempts ADD CONSTRAINT reject_attempt_success CHECK (status <> 'succeeded')`,
			constraint: "reject_attempt_success",
		},
		{
			name: "commit",
			// A deferred constraint trigger raises only at the real PostgreSQL COMMIT,
			// after both fenced success updates and all business writes have succeeded.
			ddl: `CREATE FUNCTION reject_initialization_commit() RETURNS trigger LANGUAGE plpgsql AS $$
				BEGIN
					RAISE EXCEPTION 'injected deferred commit failure' USING ERRCODE = '23514', CONSTRAINT = 'reject_deferred_success';
				END;
			$$;
			CREATE CONSTRAINT TRIGGER reject_deferred_success AFTER UPDATE ON initialization_component_attempts
			DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.status = 'succeeded')
			EXECUTE FUNCTION reject_initialization_commit()`,
			constraint: "reject_deferred_success",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newInitializationPostgresFixture(t)
			if scenario.ddl != "" {
				_, err := f.db.ExecContext(f.ctx, scenario.ddl)
				require.NoError(t, err)
			}
			verified := false
			component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
				result, err := writeAndVerifyInitializationFixture(ctx, scope, plan, driver)
				if err != nil {
					return result, err
				}
				verified = true
				require.Zero(t, f.businessCount(t), "business data must remain invisible until the Engine commits")
				if scenario.name == "verification" {
					return result, verificationErr
				}
				return result, nil
			}}
			engine, err := NewEngine(f.store, []Initializer{component}, time.Hour)
			require.NoError(t, err)
			request := initializationRequest("executor-1")
			runID, err := engine.Apply(f.ctx, request)
			require.Error(t, err)
			require.True(t, verified, "failure must occur after transaction-local business write and verification")
			if scenario.name == "verification" {
				require.ErrorIs(t, err, verificationErr)
			} else {
				var pgErr *pq.Error
				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, pq.ErrorCode("23514"), pgErr.Code)
				require.Equal(t, scenario.constraint, pgErr.Constraint)
			}
			require.Zero(t, f.businessCount(t), "business write must roll back with success ledger/commit failure")
			f.assertRunAndAttempt(t, runID, "failed")
			status := f.installation(t, request.Scope)
			require.Equal(t, "failed", status.Status)
			require.Empty(t, status.InstalledVersion)
			require.Empty(t, status.SourceChecksum)
			require.Empty(t, status.ResultSummary)
			require.Empty(t, status.LeaseOwner)
			require.Nil(t, status.LeaseExpiresAt)
			var successMetadata bool
			require.NoError(t, f.secondDB.QueryRowContext(f.ctx, `SELECT result_summary <> '{}'::jsonb OR rollback_metadata <> '{}'::jsonb
				FROM initialization_component_attempts WHERE run_id = $1`, runID).Scan(&successMetadata))
			require.False(t, successMetadata, "rolled-back success metadata must not survive in the failed attempt")
		})
	}
}

func TestPostgresEngineCancellationClosesFailedRun(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		result, err := writeAndVerifyInitializationFixture(ctx, scope, plan, driver)
		if err != nil {
			return result, err
		}
		cancel()
		return result, ctx.Err()
	}}
	engine, err := NewEngine(f.store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	request := initializationRequest("executor-1")
	runID, err := engine.Apply(ctx, request)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, f.businessCount(t))
	f.assertRunAndAttempt(t, runID, "failed")
	status := f.installation(t, request.Scope)
	require.Equal(t, "failed", status.Status)
	require.Empty(t, status.LeaseOwner)
	require.Nil(t, status.LeaseExpiresAt)
}

func TestPostgresEngineRepeatedSuccessIsIdempotentAndScopeIsolated(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	component := &testInitializer{name: "rbac", apply: writeAndVerifyInitializationFixture}
	engine, err := NewEngine(f.store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	request := initializationRequest("executor-1")
	var priorToken int64
	for range 2 {
		runID, err := engine.Apply(f.ctx, request)
		require.NoError(t, err)
		f.assertRunAndAttempt(t, runID, "succeeded")
		require.Equal(t, 1, f.businessCount(t), "retry must not duplicate initialized business data")
		status := f.installation(t, request.Scope)
		require.Equal(t, "succeeded", status.Status)
		require.Equal(t, "1", status.InstalledVersion)
		require.Equal(t, "rbac-checksum", status.SourceChecksum)
		require.Equal(t, map[string]any{"verified": true}, status.ResultSummary)
		require.Greater(t, status.FencingToken, priorToken)
		require.Empty(t, status.LeaseOwner)
		require.Nil(t, status.LeaseExpiresAt)
		priorToken = status.FencingToken
	}
	before := f.installation(t, request.Scope)
	otherTenant := request
	otherTenant.Scope.ID = 43
	otherRunID, err := engine.Apply(f.ctx, otherTenant)
	require.NoError(t, err)
	f.assertRunAndAttempt(t, otherRunID, "succeeded")
	require.Equal(t, 2, f.businessCount(t))
	require.Equal(t, before, f.installation(t, request.Scope), "tenant B must not modify tenant A's installation")
	require.Equal(t, int64(1), f.installation(t, otherTenant.Scope).FencingToken)
	var running, attempts int
	require.NoError(t, f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM initialization_component_attempts WHERE status = 'running'").Scan(&running))
	require.Zero(t, running)
	require.NoError(t, f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM initialization_component_attempts").Scan(&attempts))
	require.Equal(t, 3, attempts, "each invocation remains auditable without duplicate product data")
}
