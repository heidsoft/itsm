package initialization

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testInitializer struct {
	name         string
	dependencies []string
	apply        func()
}

func (i *testInitializer) Name() string           { return i.name }
func (i *testInitializer) Dependencies() []string { return i.dependencies }
func (i *testInitializer) Plan(context.Context, Scope) (Plan, error) {
	return Plan{TargetVersion: "1", SourceChecksum: i.name + "-checksum"}, nil
}

func (i *testInitializer) Apply(context.Context, Scope, Plan, int64) (Result, error) {
	if i.apply != nil {
		i.apply()
	}
	return Result{Summary: map[string]any{"component": i.name}}, nil
}
func (i *testInitializer) Verify(context.Context, Scope, Plan) error { return nil }

type memoryLease struct {
	owner   string
	token   int64
	expires time.Time
}

type memoryStore struct {
	mu           sync.Mutex
	nextID       int64
	leases       map[string]memoryLease
	runStatuses  map[int64]string
	attemptCount int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{leases: map[string]memoryLease{}, runStatuses: map[int64]string{}}
}

func (s *memoryStore) BeginRun(context.Context, Request) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.runStatuses[s.nextID] = "running"
	return s.nextID, nil
}

func (s *memoryStore) FinishRun(_ context.Context, id int64, status string, _ map[string]any, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	current, exists := s.leases[key]
	if exists && time.Now().Before(current.expires) {
		return Lease{}, ErrLeaseHeld
	}
	current.token++
	current.owner = owner
	current.expires = time.Now().Add(ttl)
	s.leases[key] = current
	return Lease{FencingToken: current.token}, nil
}

func (s *memoryStore) Heartbeat(_ context.Context, scope Scope, component, owner string, token int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.leases[leaseKey(scope, component)]
	if current.owner != owner || current.token != token {
		return errors.New("lease lost")
	}
	current.expires = time.Now().Add(ttl)
	s.leases[leaseKey(scope, component)] = current
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
	delete(s.leases, key)
	return nil
}

func (s *memoryStore) StartAttempt(context.Context, int64, Scope, Plan, int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.attemptCount++
	return s.nextID, nil
}

func (s *memoryStore) CompleteComponent(
	_ context.Context,
	_, _ int64,
	scope Scope,
	plan Plan,
	owner string,
	token int64,
	_ Result,
	_ error,
) error {
	return s.ReleaseLease(context.Background(), scope, plan.Component, owner, token)
}

func TestEngineOrdersDependenciesAndCompletesRun(t *testing.T) {
	store := newMemoryStore()
	var order []string
	engine, err := NewEngine(store, []Initializer{
		&testInitializer{name: "menu", dependencies: []string{"rbac"}, apply: func() { order = append(order, "menu") }},
		&testInitializer{name: "rbac", apply: func() { order = append(order, "rbac") }},
	}, time.Second)
	require.NoError(t, err)

	runID, err := engine.Apply(context.Background(), Request{
		Scope: Scope{Type: "tenant", ID: 42}, TargetVersion: "1",
		ReleaseVersion: "v1", RequestedBy: "test", ExecutorID: "executor-1",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"rbac", "menu"}, order)
	require.Equal(t, "succeeded", store.runStatuses[runID])
	require.Equal(t, 2, store.attemptCount)
}

func TestEngineRejectsConcurrentRunWithSameExecutorIdentity(t *testing.T) {
	store := newMemoryStore()
	started := make(chan struct{})
	release := make(chan struct{})
	component := &testInitializer{name: "rbac", apply: func() {
		close(started)
		<-release
	}}
	engine, err := NewEngine(store, []Initializer{component}, 100*time.Millisecond)
	require.NoError(t, err)
	request := Request{
		Scope: Scope{Type: "tenant", ID: 42}, TargetVersion: "1",
		ReleaseVersion: "v1", RequestedBy: "test", ExecutorID: "executor-1",
	}
	firstDone := make(chan error, 1)
	go func() {
		_, err := engine.Apply(context.Background(), request)
		firstDone <- err
	}()
	<-started

	_, err = engine.Apply(context.Background(), request)
	require.ErrorIs(t, err, ErrLeaseHeld)
	close(release)
	require.NoError(t, <-firstDone)
}

func TestEngineRejectsDependencyCycle(t *testing.T) {
	_, err := NewEngine(newMemoryStore(), []Initializer{
		&testInitializer{name: "a", dependencies: []string{"b"}},
		&testInitializer{name: "b", dependencies: []string{"a"}},
	}, time.Second)
	require.ErrorContains(t, err, "dependency cycle")
}
