package initialization

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
)

type testInitializer struct {
	name         string
	dependencies []string
	apply        func(context.Context, Scope, Plan, dialect.Driver) (Result, error)
}

func (i *testInitializer) Name() string           { return i.name }
func (i *testInitializer) Dependencies() []string { return i.dependencies }
func (i *testInitializer) Plan(context.Context, Scope) (Plan, error) {
	return Plan{TargetVersion: "1", SourceChecksum: i.name + "-checksum"}, nil
}
func (i *testInitializer) Apply(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
	if i.apply != nil {
		return i.apply(ctx, scope, plan, driver)
	}
	return Result{Summary: map[string]any{"component": i.name}}, nil
}
func (i *testInitializer) Verify(context.Context, Scope, Plan) error {
	return errors.New("Engine.Apply must not call out-of-transaction Verify")
}

type memoryLease struct {
	owner   string
	token   int64
	expires time.Time
}

type memoryStore struct {
	mu              sync.Mutex
	nextID          int64
	leases          map[string]memoryLease
	runStatuses     map[int64]string
	attemptStatuses map[int64]string
	attemptCount    int
	writes          int
	events          []string
	beginErr        error
	completeErr     error
	commitErr       error
	failErr         error
	heartbeat       func(context.Context) error
	failureContext  context.Context
	finishContext   context.Context
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		leases: map[string]memoryLease{}, runStatuses: map[int64]string{},
		attemptStatuses: map[int64]string{},
	}
}

func (s *memoryStore) record(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}
func (s *memoryStore) BeginRun(ctx context.Context, _ Request) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.runStatuses[s.nextID] = "running"
	return s.nextID, nil
}
func (s *memoryStore) FinishRun(ctx context.Context, id int64, status string, _ map[string]any, _ error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "finish:"+status)
	s.finishContext = ctx
	s.runStatuses[id] = status
	return nil
}
func leaseKey(scope Scope, component string) string {
	return scope.Type + ":" + component + ":" + strconv.FormatInt(scope.ID, 10)
}
func (s *memoryStore) AcquireLease(_ context.Context, scope Scope, component, owner string, ttl time.Duration) (Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := leaseKey(scope, component)
	current := s.leases[key]
	if time.Now().Before(current.expires) {
		return Lease{}, ErrLeaseHeld
	}
	current.token++
	current.owner = owner
	current.expires = time.Now().Add(ttl)
	s.leases[key] = current
	return Lease{FencingToken: current.token}, nil
}
func (s *memoryStore) Heartbeat(ctx context.Context, scope Scope, component, owner string, token int64, ttl time.Duration) error {
	if s.heartbeat != nil {
		return s.heartbeat(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := leaseKey(scope, component)
	current := s.leases[key]
	if current.owner != owner || current.token != token || !time.Now().Before(current.expires) {
		return errors.New("lease lost")
	}
	current.expires = time.Now().Add(ttl)
	s.leases[key] = current
	return nil
}
func (s *memoryStore) ReleaseLease(_ context.Context, scope Scope, component, owner string, token int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := leaseKey(scope, component)
	current := s.leases[key]
	if current.owner != owner || current.token != token {
		return errors.New("lease lost")
	}
	current.owner = ""
	current.expires = time.Time{}
	s.leases[key] = current // Preserve the monotonically increasing fencing token.
	return nil
}
func (s *memoryStore) StartAttempt(context.Context, int64, Scope, Plan, int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.attemptCount++
	s.attemptStatuses[s.nextID] = "running"
	return s.nextID, nil
}
func (s *memoryStore) BeginComponent(ctx context.Context, attemptID, runID int64, scope Scope, plan Plan, owner string, token int64) (ComponentTransaction, error) {
	s.record("begin:" + plan.Component)
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	tx := &memoryComponentTransaction{store: s, ctx: ctx, attemptID: attemptID, runID: runID, scope: scope, plan: plan, owner: owner, token: token}
	tx.driver.tx = tx
	return tx, nil
}
func (s *memoryStore) FailComponent(ctx context.Context, attemptID, _ int64, scope Scope, plan Plan, owner string, token int64, _ error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.record("fail:" + plan.Component)
	s.mu.Lock()
	s.failureContext = ctx
	s.attemptStatuses[attemptID] = "failed"
	s.mu.Unlock()
	return errors.Join(s.failErr, s.ReleaseLease(ctx, scope, plan.Component, owner, token))
}

type memoryComponentTransaction struct {
	store            *memoryStore
	ctx              context.Context
	attemptID, runID int64
	scope            Scope
	plan             Plan
	owner            string
	token            int64
	driver           memoryDriver
	pending          int
	completed        bool
	closed           bool
}

func (tx *memoryComponentTransaction) Driver() dialect.Driver { return &tx.driver }
func (tx *memoryComponentTransaction) Complete(ctx context.Context, _ Result) error {
	tx.store.record("complete:" + tx.plan.Component)
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx.store.completeErr != nil {
		return tx.store.completeErr
	}
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()
	lease := tx.store.leases[leaseKey(tx.scope, tx.plan.Component)]
	if lease.owner != tx.owner || lease.token != tx.token || !time.Now().Before(lease.expires) {
		return errors.New("lease lost")
	}
	tx.completed = true
	return nil
}
func (tx *memoryComponentTransaction) Commit() error {
	tx.store.record("commit:" + tx.plan.Component)
	if err := tx.ctx.Err(); err != nil {
		return err
	}
	if tx.store.commitErr != nil {
		return tx.store.commitErr
	}
	if !tx.completed || tx.closed {
		return errors.New("transaction is not ready to commit")
	}
	tx.store.mu.Lock()
	tx.store.writes += tx.pending
	tx.store.attemptStatuses[tx.attemptID] = "succeeded"
	tx.store.mu.Unlock()
	tx.closed = true
	return tx.store.ReleaseLease(tx.ctx, tx.scope, tx.plan.Component, tx.owner, tx.token)
}
func (tx *memoryComponentTransaction) Rollback() error {
	tx.store.record("rollback:" + tx.plan.Component)
	tx.pending = 0
	tx.closed = true
	return nil
}

// This driver only models transaction-local visibility. PostgreSQL tests below
// exercise the actual SQLStore, transaction driver, constraints, and rollback.
type memoryDriver struct{ tx *memoryComponentTransaction }

func (d *memoryDriver) Exec(ctx context.Context, _ string, _ any, _ any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.tx.store.record("write:" + d.tx.plan.Component)
	d.tx.pending++
	return nil
}
func (d *memoryDriver) Query(ctx context.Context, _ string, _ any, dest any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	count, ok := dest.(*int)
	if !ok {
		return fmt.Errorf("unsupported verification destination %T", dest)
	}
	d.tx.store.record("verify:" + d.tx.plan.Component)
	*count = d.tx.pending
	return nil
}
func (d *memoryDriver) Dialect() string                        { return dialect.Postgres }
func (d *memoryDriver) Close() error                           { return nil }
func (d *memoryDriver) Tx(context.Context) (dialect.Tx, error) { return dialect.NopTx(d), nil }

func memoryApply(ctx context.Context, _ Scope, _ Plan, driver dialect.Driver) (Result, error) {
	if err := driver.Exec(ctx, "write", nil, nil); err != nil {
		return Result{}, err
	}
	var count int
	if err := driver.Query(ctx, "verify", nil, &count); err != nil {
		return Result{}, err
	}
	if count != 1 {
		return Result{}, fmt.Errorf("verification did not see transaction-local write: %d", count)
	}
	return Result{Summary: map[string]any{"writes": count}}, nil
}

func initializationRequest(owner string) Request {
	return Request{
		Scope: Scope{Type: "tenant", ID: 42}, TargetVersion: "1",
		ReleaseVersion: "v1", RequestedBy: "test", ExecutorID: owner,
	}
}

func TestEngineOrdersDependenciesAndCompletesRun(t *testing.T) {
	store := newMemoryStore()
	engine, err := NewEngine(store, []Initializer{
		&testInitializer{name: "menu", dependencies: []string{"rbac"}, apply: memoryApply},
		&testInitializer{name: "rbac", apply: memoryApply},
	}, time.Hour)
	require.NoError(t, err)
	runID, err := engine.Apply(context.Background(), initializationRequest("executor-1"))
	require.NoError(t, err)
	require.Equal(t, []string{
		"begin:rbac", "write:rbac", "verify:rbac", "complete:rbac", "commit:rbac", "rollback:rbac",
		"begin:menu", "write:menu", "verify:menu", "complete:menu", "commit:menu", "rollback:menu", "finish:succeeded",
	}, store.events)
	require.Equal(t, "succeeded", store.runStatuses[runID])
	require.Equal(t, 2, store.attemptCount)
	require.Equal(t, 2, store.writes)
	for _, status := range store.attemptStatuses {
		require.Equal(t, "succeeded", status)
	}
}

func TestEngineRollsBackBeforeRecordingFailure(t *testing.T) {
	failure := errors.New("injected failure")
	for _, stage := range []string{"begin", "verification", "ledger", "commit"} {
		t.Run(stage, func(t *testing.T) {
			store := newMemoryStore()
			apply := memoryApply
			wantEvents := []string{"begin:rbac"}
			switch stage {
			case "begin":
				store.beginErr = failure
			case "verification":
				apply = func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
					result, err := memoryApply(ctx, scope, plan, driver)
					return result, errors.Join(err, failure)
				}
			case "ledger":
				store.completeErr = failure
			case "commit":
				store.commitErr = failure
			}
			if stage != "begin" {
				wantEvents = append(wantEvents, "write:rbac", "verify:rbac")
				if stage != "verification" {
					wantEvents = append(wantEvents, "complete:rbac")
				}
				if stage == "commit" {
					wantEvents = append(wantEvents, "commit:rbac")
				}
				wantEvents = append(wantEvents, "rollback:rbac")
			}
			wantEvents = append(wantEvents, "fail:rbac", "finish:failed")
			engine, err := NewEngine(store, []Initializer{
				&testInitializer{name: "rbac", apply: apply},
				&testInitializer{name: "menu", dependencies: []string{"rbac"}, apply: memoryApply},
			}, time.Hour)
			require.NoError(t, err)
			runID, err := engine.Apply(context.Background(), initializationRequest("executor-1"))
			require.ErrorIs(t, err, failure)
			require.Equal(t, wantEvents, store.events)
			require.Zero(t, store.writes)
			require.Equal(t, "failed", store.runStatuses[runID])
			require.Equal(t, 1, store.attemptCount, "dependent component must not run")
			for _, status := range store.attemptStatuses {
				require.Equal(t, "failed", status)
			}
		})
	}
}

func TestEngineStopsHeartbeatBeforeCompletingTransaction(t *testing.T) {
	store := newMemoryStore()
	entered := make(chan struct{})
	store.heartbeat = func(ctx context.Context) error {
		close(entered)
		<-ctx.Done()
		store.record("heartbeat:stopped")
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		select {
		case <-entered:
			return memoryApply(ctx, scope, plan, driver)
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}}
	engine, err := NewEngine(store, []Initializer{component}, time.Second)
	require.NoError(t, err)
	_, err = engine.Apply(ctx, initializationRequest("executor-1"))
	require.NoError(t, err)
	require.Equal(t, []string{
		"begin:rbac", "write:rbac", "verify:rbac", "heartbeat:stopped",
		"complete:rbac", "commit:rbac", "rollback:rbac", "finish:succeeded",
	}, store.events)
}

func TestEngineHeartbeatFailureCancelsApplyAndRollsBack(t *testing.T) {
	store := newMemoryStore()
	written := make(chan struct{})
	leaseLost := errors.New("injected lease loss")
	store.heartbeat = func(ctx context.Context) error {
		select {
		case <-written:
			return leaseLost
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		result, err := memoryApply(ctx, scope, plan, driver)
		close(written)
		if err != nil {
			return result, err
		}
		<-ctx.Done()
		return result, ctx.Err()
	}}
	engine, err := NewEngine(store, []Initializer{component}, time.Second)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runID, err := engine.Apply(ctx, initializationRequest("executor-1"))
	require.ErrorIs(t, err, leaseLost)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, store.writes)
	require.Equal(t, "failed", store.runStatuses[runID])
	require.Equal(t, []string{"begin:rbac", "write:rbac", "verify:rbac", "rollback:rbac", "fail:rbac", "finish:failed"}, store.events)
}

func TestEngineCancellationStillFinishesFailedRun(t *testing.T) {
	type contextKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "request-scope"))
	defer cancel()
	store := newMemoryStore()
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		result, err := memoryApply(ctx, scope, plan, driver)
		cancel()
		return result, errors.Join(err, ctx.Err())
	}}
	engine, err := NewEngine(store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	runID, err := engine.Apply(ctx, initializationRequest("executor-1"))
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, store.writes)
	require.Equal(t, "failed", store.runStatuses[runID])
	require.Equal(t, []string{"begin:rbac", "write:rbac", "verify:rbac", "rollback:rbac", "fail:rbac", "finish:failed"}, store.events)
	for _, cleanupCtx := range []context.Context{store.failureContext, store.finishContext} {
		require.NotNil(t, cleanupCtx)
		require.Equal(t, "request-scope", cleanupCtx.Value(contextKey{}), "cleanup must retain identity context")
		_, bounded := cleanupCtx.Deadline()
		require.True(t, bounded, "cleanup must have a deadline")
	}
}

func TestEnginePreservesApplyAndFailureLedgerErrors(t *testing.T) {
	store := newMemoryStore()
	store.completeErr = errors.New("ledger completion failed")
	store.failErr = errors.New("failure ledger failed")
	engine, err := NewEngine(store, []Initializer{&testInitializer{name: "rbac", apply: memoryApply}}, time.Hour)
	require.NoError(t, err)
	runID, err := engine.Apply(context.Background(), initializationRequest("executor-1"))
	require.ErrorIs(t, err, store.completeErr)
	require.ErrorIs(t, err, store.failErr)
	require.Zero(t, store.writes)
	require.Equal(t, "failed", store.runStatuses[runID])
}

func TestEngineRejectsConcurrentRunWithSameExecutorIdentity(t *testing.T) {
	store := newMemoryStore()
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	component := &testInitializer{name: "rbac", apply: func(ctx context.Context, scope Scope, plan Plan, driver dialect.Driver) (Result, error) {
		close(started)
		select {
		case <-release:
			return memoryApply(ctx, scope, plan, driver)
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}}
	engine, err := NewEngine(store, []Initializer{component}, time.Hour)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request := initializationRequest("executor-1")
	firstDone := make(chan error, 1)
	go func() {
		_, err := engine.Apply(ctx, request)
		firstDone <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("first initializer did not start")
	}
	_, err = engine.Apply(ctx, request)
	require.ErrorIs(t, err, ErrLeaseHeld)
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-firstDone:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("first initializer did not finish")
	}
}

func TestEngineRejectsDependencyCycle(t *testing.T) {
	_, err := NewEngine(newMemoryStore(), []Initializer{
		&testInitializer{name: "a", dependencies: []string{"b"}},
		&testInitializer{name: "b", dependencies: []string{"a"}},
	}, time.Second)
	require.ErrorContains(t, err, "dependency cycle")
}
