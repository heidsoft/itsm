package initialization

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFencingTokenPreventsStaleWriter tests that the fencing token mechanism
// prevents a stale writer from completing a transaction after a fresh writer
// has already acquired the lease.
func TestFencingTokenPreventsStaleWriter(t *testing.T) {
	// This test requires a running PostgreSQL instance.
	// Skip if no database is available.
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=itsm sslmode=disable")
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Cannot ping database, skipping integration test: " + err.Error())
	}

	// Ensure RLS is disabled for this test (use superuser)
	_, err = db.ExecContext(ctx, "SET ROLE postgres")
	require.NoError(t, err)

	// Setup: create test installation record
	setupSQL := `
		CREATE TABLE IF NOT EXISTS test_fence_installation (
			id BIGSERIAL PRIMARY KEY,
			scope_type VARCHAR(16) NOT NULL,
			scope_id BIGINT NOT NULL,
			component VARCHAR(100) NOT NULL,
			status VARCHAR(24) NOT NULL DEFAULT 'running',
			fencing_token BIGINT NOT NULL DEFAULT 1,
			lease_owner VARCHAR(255) NOT NULL DEFAULT '',
			lease_expires_at TIMESTAMPTZ,
			UNIQUE(scope_type, scope_id, component)
		);

		TRUNCATE TABLE test_fence_installation;
	`
	_, err = db.ExecContext(ctx, setupSQL)
	require.NoError(t, err)

	// Helper to acquire lease
	acquireLease := func(executorID string, ttl time.Duration) (int64, error) {
		var token int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO test_fence_installation
				(scope_type, scope_id, component, status, fencing_token, lease_owner, lease_expires_at)
			VALUES ('tenant', 1, 'test-component', 'running', 1, $1, NOW() + ($2 * INTERVAL '1 millisecond'))
			ON CONFLICT (scope_type, scope_id, component) DO UPDATE
			SET status = 'running',
			    fencing_token = test_fence_installation.fencing_token + 1,
			    lease_owner = EXCLUDED.lease_owner,
			    lease_expires_at = EXCLUDED.lease_expires_at
			WHERE test_fence_installation.lease_expires_at IS NULL
			   OR test_fence_installation.lease_expires_at < NOW()
			RETURNING fencing_token
		`, executorID, ttl.Milliseconds()).Scan(&token)
		return token, err
	}

	// Helper to heartbeat with token
	heartbeat := func(executorID string, token int64, ttl time.Duration) error {
		result, err := db.ExecContext(ctx, `
			UPDATE test_fence_installation
			SET lease_expires_at = NOW() + ($1 * INTERVAL '1 millisecond')
			WHERE scope_type = 'tenant' AND scope_id = 1 AND component = 'test-component'
			  AND lease_owner = $2 AND fencing_token = $3 AND status = 'running'
		`, ttl.Milliseconds(), executorID, token)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrLeaseLost
		}
		return nil
	}

	// Helper to complete component (requires correct token)
	completeComponent := func(executorID string, token int64) error {
		result, err := db.ExecContext(ctx, `
			UPDATE test_fence_installation
			SET status = 'succeeded', lease_owner = '', lease_expires_at = NULL
			WHERE scope_type = 'tenant' AND scope_id = 1 AND component = 'test-component'
			  AND lease_owner = $1 AND fencing_token = $2
		`, executorID, token)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrLeaseLost
		}
		return nil
	}

	// Test 1: Fresh writer acquires lease successfully
	t.Run("fresh_writer_acquires_lease", func(t *testing.T) {
		token, err := acquireLease("executor-fresh", 5*time.Second)
		require.NoError(t, err)
		assert.Greater(t, token, int64(0))
	})

	// Reset for next test
	_, _ = db.ExecContext(ctx, "TRUNCATE TABLE test_fence_installation")

	// Test 2: Stale writer (old token) cannot complete
	t.Run("stale_writer_token_rejected", func(t *testing.T) {
		// Executor A acquires lease
		tokenA, err := acquireLease("executor-A", 5*time.Second)
		require.NoError(t, err)

		// Executor B acquires same lease (fencing token incremented)
		tokenB, err := acquireLease("executor-B", 5*time.Second)
		require.NoError(t, err)
		assert.Greater(t, tokenB, tokenA, "token should be incremented")

		// Executor A (stale) tries to complete with old token - should fail
		err = completeComponent("executor-A", tokenA)
		assert.ErrorIs(t, err, ErrLeaseLost, "stale writer should be rejected")

		// Executor B (fresh) completes successfully
		err = completeComponent("executor-B", tokenB)
		assert.NoError(t, err, "fresh writer should succeed")
	})

	// Reset for next test
	_, _ = db.ExecContext(ctx, "TRUNCATE TABLE test_fence_installation")

	// Test 3: Heartbeat extends lease
	t.Run("heartbeat_extends_lease", func(t *testing.T) {
		token, err := acquireLease("executor-heartbeat", 2*time.Second)
		require.NoError(t, err)

		// Wait 1 second, heartbeat should still work
		time.Sleep(1 * time.Second)
		err = heartbeat("executor-heartbeat", token, 5*time.Second)
		assert.NoError(t, err)

		// Wait another 2 seconds (total 3s > original 2s TTL)
		// Stale heartbeat with same token should still work (we extended it)
		err = heartbeat("executor-heartbeat", token, 5*time.Second)
		assert.NoError(t, err)
	})

	// Reset for next test
	_, _ = db.ExecContext(ctx, "TRUNCATE TABLE test_fence_installation")

	// Test 4: Stale heartbeat (wrong token) is rejected
	t.Run("stale_heartbeat_rejected", func(t *testing.T) {
		// Executor A acquires lease with token 1
		tokenA, err := acquireLease("executor-A", 5*time.Second)
		require.NoError(t, err)

		// Executor B acquires lease - token increments to 2
		tokenB, err := acquireLease("executor-B", 5*time.Second)
		require.NoError(t, err)
		assert.Greater(t, tokenB, tokenA)

		// Executor A (stale) tries heartbeat with token 1 - should fail
		err = heartbeat("executor-A", tokenA, 5*time.Second)
		assert.ErrorIs(t, err, ErrLeaseLost)

		// Executor B (fresh) heartbeat works
		err = heartbeat("executor-B", tokenB, 5*time.Second)
		assert.NoError(t, err)
	})

	// Cleanup
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_fence_installation")
}

// TestFencingTokenCrashRecovery tests that after a crash, the fencing token
// prevents the crashed executor from resuming work that was taken over.
func TestFencingTokenCrashRecovery(t *testing.T) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=itsm sslmode=disable")
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Cannot ping database, skipping integration test: " + err.Error())
	}

	_, err = db.ExecContext(ctx, "SET ROLE postgres")
	require.NoError(t, err)

	// Setup
	setupSQL := `
		CREATE TABLE IF NOT EXISTS test_crash_installation (
			id BIGSERIAL PRIMARY KEY,
			scope_type VARCHAR(16) NOT NULL,
			scope_id BIGINT NOT NULL,
			component VARCHAR(100) NOT NULL,
			status VARCHAR(24) NOT NULL DEFAULT 'running',
			fencing_token BIGINT NOT NULL DEFAULT 1,
			lease_owner VARCHAR(255) NOT NULL DEFAULT '',
			lease_expires_at TIMESTAMPTZ,
			UNIQUE(scope_type, scope_id, component)
		);
		TRUNCATE TABLE test_crash_installation;
	`
	_, err = db.ExecContext(ctx, setupSQL)
	require.NoError(t, err)

	// Test scenario: executor crashes mid-transaction
	t.Run("crashed_executor_cannot_resume", func(t *testing.T) {
		// Executor A starts and acquires lease
		var tokenA int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO test_crash_installation
				(scope_type, scope_id, component, status, fencing_token, lease_owner, lease_expires_at)
			VALUES ('tenant', 1, 'crash-test', 'running', 1, 'executor-A', NOW() + '5 seconds')
			ON CONFLICT (scope_type, scope_id, component) DO NOTHING
			RETURNING fencing_token
		`).Scan(&tokenA)
		require.NoError(t, err)

		// Simulate: executor-A starts transaction, writes some data
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Executor writes partial work
		_, err = tx.ExecContext(ctx, `
			UPDATE test_crash_installation
			SET status = 'running'  -- still running
			WHERE scope_type = 'tenant' AND scope_id = 1 AND component = 'crash-test'
		`)
		require.NoError(t, err)

		// CRASH: executor-A dies, connection drops (simulated by rollback)
		tx.Rollback()

		// Meanwhile, executor-B takes over
		var tokenB int64
		err = db.QueryRowContext(ctx, `
			INSERT INTO test_crash_installation
				(scope_type, scope_id, component, status, fencing_token, lease_owner, lease_expires_at)
			VALUES ('tenant', 1, 'crash-test', 'running', 1, 'executor-B', NOW() + '5 seconds')
			ON CONFLICT (scope_type, scope_id, component) DO UPDATE
			SET status = 'running',
			    fencing_token = test_crash_installation.fencing_token + 1,
			    lease_owner = EXCLUDED.lease_owner,
			    lease_expires_at = EXCLUDED.lease_expires_at
			WHERE test_crash_installation.lease_expires_at IS NULL
			   OR test_crash_installation.lease_expires_at < NOW()
			RETURNING fencing_token
		`).Scan(&tokenB)
		require.NoError(t, err)
		assert.Greater(t, tokenB, tokenA, "token should be incremented by new owner")

		// Executor-B completes successfully
		_, err = db.ExecContext(ctx, `
			UPDATE test_crash_installation
			SET status = 'succeeded', lease_owner = '', lease_expires_at = NULL
			WHERE scope_type = 'tenant' AND scope_id = 1 AND component = 'crash-test'
			  AND lease_owner = 'executor-B' AND fencing_token = $1
		`, tokenB)
		require.NoError(t, err)

		// Verify: executor-A (crashed) cannot complete with old token
		result, err := db.ExecContext(ctx, `
			UPDATE test_crash_installation
			SET status = 'succeeded', lease_owner = '', lease_expires_at = NULL
			WHERE scope_type = 'tenant' AND scope_id = 1 AND component = 'crash-test'
			  AND lease_owner = 'executor-A' AND fencing_token = $1
		`, tokenA)
		require.NoError(t, err)
		affected, _ := result.RowsAffected()
		assert.Equal(t, int64(0), affected, "crashed executor's stale token should not affect rows")
	})

	// Cleanup
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_crash_installation")
}

// ErrLeaseLost is returned when the lease is held by another executor
var ErrLeaseLost = &LeaseError{message: "lease lost or fencing token rejected"}

// LeaseError represents a lease-related error
type LeaseError struct {
	message string
}

func (e *LeaseError) Error() string {
	return e.message
}
