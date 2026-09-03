package cloud

import (
	"context"
	"fmt"

	"itsm-backend/ent"
)

// DiscoveryWorker is the durable-command entry point for tenant cloud discovery.
type DiscoveryWorker struct {
	runner *Runner
}

func NewDiscoveryWorker(runner *Runner) *DiscoveryWorker { return &DiscoveryWorker{runner: runner} }

func (w *DiscoveryWorker) Handle(ctx context.Context, command *ent.OperationalCommand) error {
	if w == nil || w.runner == nil {
		return fmt.Errorf("cloud discovery runner is not configured")
	}
	if command == nil || command.TenantID <= 0 {
		return fmt.Errorf("tenant-scoped cloud discovery command is required")
	}
	return w.runner.RunAll(ctx, command.TenantID)
}
