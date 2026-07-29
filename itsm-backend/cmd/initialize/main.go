package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/seeder"

	"go.uber.org/zap"
)

func main() {
	action := flag.String("action", "status", "plan|apply|status|verify|retry")
	releaseVersion := flag.String("release-version", os.Getenv("ITSM_RELEASE_VERSION"), "release version")
	requestedBy := flag.String("requested-by", "operator", "audited requester identity")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		exitf("load config: %v", err)
	}
	logger, err := zap.NewProduction()
	if err != nil {
		exitf("initialize logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	client, err := database.InitDatabaseWithRLS(&cfg.Database, &cfg.RLS, sugar)
	if err != nil {
		exitf("connect database: %v", err)
	}
	defer client.Close()

	store, err := initialization.NewSQLStore(database.GetRawDB())
	if err != nil {
		exitf("create initialization store: %v", err)
	}
	productSeeder := seeder.NewSeeder(client, sugar, cfg)
	components, err := seeder.ProductionInitializers(productSeeder)
	if err != nil {
		exitf("create initializer: %v", err)
	}
	engine, err := initialization.NewEngine(
		store, components, 30*time.Second,
	)
	if err != nil {
		exitf("create engine: %v", err)
	}
	scope := initialization.Scope{Type: "platform", ID: 0}
	ctx := tenantctx.SystemContext(
		context.Background(),
		"initialization:cli",
		"operator requested "+*action,
	)

	switch strings.ToLower(strings.TrimSpace(*action)) {
	case "plan":
		plans, err := engine.Plan(ctx, scope)
		if err != nil {
			exitf("plan initialization: %v", err)
		}
		writeJSON(plans)
	case "apply", "retry":
		if strings.TrimSpace(*releaseVersion) == "" {
			exitf("-release-version or ITSM_RELEASE_VERSION is required")
		}
		executorID, _ := os.Hostname()
		executorID, err = initialization.NewExecutorID(executorID)
		if err != nil {
			exitf("create executor id: %v", err)
		}
		runID, err := engine.Apply(ctx, initialization.Request{
			Scope:          scope,
			TargetVersion:  seeder.CurrentTenantTemplateVersion,
			ReleaseVersion: *releaseVersion,
			RequestedBy:    *requestedBy,
			ExecutorID:     executorID,
		})
		if err != nil {
			exitf("initialization run %d failed: %v", runID, err)
		}
		writeJSON(map[string]any{"runId": runID, "status": "succeeded"})
	case "status":
		status, err := store.Status(ctx, scope)
		if err != nil {
			exitf("read initialization status: %v", err)
		}
		writeJSON(status)
	case "verify":
		plans, err := engine.Plan(ctx, scope)
		if err != nil {
			exitf("plan verification: %v", err)
		}
		componentByName := make(map[string]initialization.Initializer, len(components))
		for _, component := range components {
			componentByName[component.Name()] = component
		}
		for _, plan := range plans {
			if err := componentByName[plan.Component].Verify(ctx, scope, plan); err != nil {
				exitf("verify %s: %v", plan.Component, err)
			}
		}
		writeJSON(map[string]any{"status": "verified", "components": len(plans)})
	default:
		exitf("unsupported action %q", *action)
	}
}

func writeJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		exitf("encode output: %v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
