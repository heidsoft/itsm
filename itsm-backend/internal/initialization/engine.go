package initialization

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"entgo.io/ent/dialect"
)

var ErrLeaseHeld = errors.New("initialization component lease is held by another executor")

type Scope struct {
	Type string
	ID   int64
}

type Request struct {
	Scope          Scope
	TargetVersion  string
	ReleaseVersion string
	RequestedBy    string
	ExecutorID     string
}

type Plan struct {
	Component      string
	FromVersion    string
	TargetVersion  string
	SourceChecksum string
	Actions        []Action
}

type Action struct {
	Type      string
	SourceKey string
	Summary   string
}

type Result struct {
	Summary          map[string]any
	RollbackMetadata map[string]any
}

type Initializer interface {
	Name() string
	Dependencies() []string
	Plan(context.Context, Scope) (Plan, error)
	Apply(context.Context, Scope, Plan, dialect.Driver) (Result, error)
	Verify(context.Context, Scope, Plan) error
}

type ComponentTransaction interface {
	Driver() dialect.Driver
	Complete(context.Context, Result) error
	Commit() error
	Rollback() error
}

type Lease struct {
	FencingToken int64
}

// ValidateScope keeps scope type and id consistent: the platform baseline is a
// single unnamed scope, while a tenant scope must name an existing tenant.
func ValidateScope(scope Scope) error {
	switch scope.Type {
	case "platform":
		if scope.ID != 0 {
			return fmt.Errorf("platform scope must have id 0, got %d", scope.ID)
		}
	case "tenant":
		if scope.ID <= 0 {
			return fmt.Errorf("tenant scope requires a positive id, got %d", scope.ID)
		}
	default:
		return fmt.Errorf("unsupported scope type %q", scope.Type)
	}
	return nil
}

type Store interface {
	BeginRun(context.Context, Request) (int64, error)
	FinishRun(context.Context, int64, string, map[string]any, error) error
	AcquireLease(context.Context, Scope, string, string, time.Duration) (Lease, error)
	Heartbeat(context.Context, Scope, string, string, int64, time.Duration) error
	ReleaseLease(context.Context, Scope, string, string, int64) error
	StartAttempt(context.Context, int64, Scope, Plan, int64) (int64, error)
	BeginComponent(context.Context, int64, int64, Scope, Plan, string, int64) (ComponentTransaction, error)
	FailComponent(context.Context, int64, int64, Scope, Plan, string, int64, error) error
}

type Engine struct {
	store      Store
	components map[string]Initializer
	leaseTTL   time.Duration
}

func NewEngine(store Store, components []Initializer, leaseTTL time.Duration) (*Engine, error) {
	if store == nil {
		return nil, fmt.Errorf("initialization store is required")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("lease TTL must be positive")
	}
	index := make(map[string]Initializer, len(components))
	for _, component := range components {
		if component == nil || component.Name() == "" {
			return nil, fmt.Errorf("initializer name is required")
		}
		if _, exists := index[component.Name()]; exists {
			return nil, fmt.Errorf("duplicate initializer %q", component.Name())
		}
		index[component.Name()] = component
	}
	if _, err := orderComponents(index); err != nil {
		return nil, err
	}
	return &Engine{store: store, components: index, leaseTTL: leaseTTL}, nil
}

func (e *Engine) Plan(ctx context.Context, scope Scope) ([]Plan, error) {
	ordered, err := orderComponents(e.components)
	if err != nil {
		return nil, err
	}
	plans := make([]Plan, 0, len(ordered))
	for _, component := range ordered {
		plan, err := component.Plan(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("plan component %s: %w", component.Name(), err)
		}
		plan.Component = component.Name()
		plans = append(plans, plan)
	}
	return plans, nil
}

func (e *Engine) Apply(ctx context.Context, request Request) (runID int64, err error) {
	if err := ValidateScope(request.Scope); err != nil {
		return 0, err
	}
	if request.ExecutorID == "" || request.RequestedBy == "" {
		return 0, fmt.Errorf("executor and requester are required")
	}
	runID, err = e.store.BeginRun(ctx, request)
	if err != nil {
		return 0, err
	}
	runStatus := "succeeded"
	runSummary := map[string]any{"components": 0}
	defer func() {
		if err != nil {
			runStatus = "failed"
		}
		finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err = errors.Join(err, e.store.FinishRun(finishCtx, runID, runStatus, runSummary, err))
	}()

	plans, err := e.Plan(ctx, request.Scope)
	if err != nil {
		return runID, err
	}
	for _, plan := range plans {
		component := e.components[plan.Component]
		lease, acquireErr := e.store.AcquireLease(
			ctx, request.Scope, plan.Component, request.ExecutorID, e.leaseTTL,
		)
		if acquireErr != nil {
			return runID, fmt.Errorf("acquire %s lease: %w", plan.Component, acquireErr)
		}
		attemptID, startErr := e.store.StartAttempt(ctx, runID, request.Scope, plan, lease.FencingToken)
		if startErr != nil {
			_ = e.store.ReleaseLease(ctx, request.Scope, plan.Component, request.ExecutorID, lease.FencingToken)
			return runID, startErr
		}
		applyCtx, cancelApply := context.WithCancel(ctx)
		heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
		heartbeatErr := make(chan error, 1)
		go e.maintainLease(
			heartbeatCtx,
			request.Scope,
			plan.Component,
			request.ExecutorID,
			lease.FencingToken,
			cancelApply,
			heartbeatErr,
		)
		tx, applyErr := e.store.BeginComponent(
			applyCtx, attemptID, runID, request.Scope, plan, request.ExecutorID, lease.FencingToken,
		)
		var result Result
		if applyErr == nil {
			result, applyErr = component.Apply(applyCtx, request.Scope, plan, tx.Driver())
		}
		stopHeartbeat()
		applyErr = errors.Join(applyErr, <-heartbeatErr)
		if applyErr == nil {
			applyErr = tx.Complete(applyCtx, result)
		}
		if applyErr == nil {
			applyErr = tx.Commit()
		}
		if tx != nil {
			_ = tx.Rollback()
		}
		cancelApply()
		if applyErr != nil {
			failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			completeErr := e.store.FailComponent(
				failureCtx, attemptID, runID, request.Scope, plan, request.ExecutorID,
				lease.FencingToken, applyErr,
			)
			cancel()
			return runID, fmt.Errorf("apply component %s: %w", plan.Component, errors.Join(applyErr, completeErr))
		}
		runSummary["components"] = runSummary["components"].(int) + 1
	}
	return runID, nil
}

func (e *Engine) maintainLease(
	ctx context.Context,
	scope Scope,
	component, executorID string,
	token int64,
	cancelApply context.CancelFunc,
	result chan<- error,
) {
	interval := e.leaseTTL / 3
	if interval <= 0 {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			result <- nil
			return
		case <-ticker.C:
			if err := e.store.Heartbeat(
				ctx, scope, component, executorID, token, e.leaseTTL,
			); err != nil {
				if ctx.Err() != nil {
					result <- nil
					return
				}
				cancelApply()
				result <- fmt.Errorf("heartbeat component %s: %w", component, err)
				return
			}
		}
	}
}

func orderComponents(index map[string]Initializer) ([]Initializer, error) {
	visiting := make(map[string]bool, len(index))
	visited := make(map[string]bool, len(index))
	ordered := make([]Initializer, 0, len(index))
	names := make([]string, 0, len(index))
	for name := range index {
		names = append(names, name)
	}
	sort.Strings(names)
	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("initializer dependency cycle at %q", name)
		}
		if visited[name] {
			return nil
		}
		component, exists := index[name]
		if !exists {
			return fmt.Errorf("unknown initializer dependency %q", name)
		}
		visiting[name] = true
		dependencies := append([]string(nil), component.Dependencies()...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		ordered = append(ordered, component)
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
