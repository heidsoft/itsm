package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/pkg/seeder"

	"go.uber.org/zap"
)

func main() {
	tenantID := flag.Int("tenant-id", 0, "existing tenant ID to provision")
	templateVersion := flag.String(
		"template-version",
		seeder.CurrentTenantTemplateVersion,
		"product tenant template version",
	)
	flag.Parse()
	if *tenantID <= 0 {
		fmt.Fprintln(os.Stderr, "-tenant-id must be positive")
		os.Exit(2)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	client, err := database.InitDatabaseWithRLS(&cfg.Database, &cfg.RLS, sugar)
	if err != nil {
		sugar.Fatalw("connect database", "error", err)
	}
	defer client.Close()

	ctx := tenantctx.SystemContext(
		context.Background(),
		"bootstrap:provision_tenant",
		fmt.Sprintf("install product template %s for tenant %d", *templateVersion, *tenantID),
	)
	provisioner := seeder.NewSeeder(client, sugar, cfg)
	if err := provisioner.ProvisionTenant(ctx, *tenantID, *templateVersion); err != nil {
		sugar.Fatalw("tenant provisioning failed", "tenant_id", *tenantID, "error", err)
	}
	sugar.Infow("tenant provisioning completed", "tenant_id", *tenantID, "template_version", *templateVersion)
}
