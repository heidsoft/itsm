// 服务请求流程升级运维命令：统一终止绑定旧版内置流程的在途实例，
// 把请求置为 cancelled 并通知发起人重新提交。默认 dry-run，显式传 -apply 才写入。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	domaincmdb "itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	servicerequest "itsm-backend/handlers/service_request"
	"itsm-backend/service"

	"go.uber.org/zap"
)

type tenantSummary struct {
	TenantID   int                                  `json:"tenantId"`
	Error      string                               `json:"error,omitempty"`
	Cancelled  int                                  `json:"cancelled"`
	Terminated int                                  `json:"terminatedWorkflows"`
	Skipped    int                                  `json:"skipped"`
	Items      []servicerequest.StaleInstanceResult `json:"items"`
}

func main() {
	tenantID := flag.Int("tenant-id", 0, "只处理指定租户；0 表示全部 active 租户")
	actorID := flag.Int("actor-id", 0, "写入审计的操作人 ID（必填）")
	apply := flag.Bool("apply", false, "实际执行终止；缺省只做 dry-run")
	reason := flag.String("reason", "", "终止原因（写入审计与发起人通知）")
	flag.Parse()

	if *actorID <= 0 {
		fmt.Fprintln(os.Stderr, "-actor-id is required")
		os.Exit(2)
	}

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

	systemCtx := tenantctx.SystemContext(context.Background(), "service_request_maintenance", "operator bulk-terminate stale process instances")
	targets, err := resolveTenants(systemCtx, client, *tenantID)
	if err != nil {
		exitf("resolve tenants: %v", err)
	}

	svc := servicerequest.NewService(
		servicerequest.NewEntRepository(client),
		service_catalog.NewEntRepository(client),
		domaincmdb.NewEntRepository(client),
		client,
		sugar,
		service.NewApprovalChainService(client, sugar),
	)
	svc.SetSystemNotifier(service.NewNotificationService(client))

	summaries := make([]tenantSummary, 0, len(targets))
	exitCode := 0
	for _, t := range targets {
		ctx := tenantctx.WithTenantID(context.Background(), t.ID)
		items, err := svc.TerminateStaleServiceRequestInstances(ctx, t.ID, *actorID, *reason, !*apply)
		summary := tenantSummary{TenantID: t.ID, Items: items}
		if err != nil {
			summary.Error = err.Error()
			exitCode = 1
		}
		for _, item := range items {
			switch item.Action {
			case "cancel":
				summary.Cancelled++
			case "terminate_workflow":
				summary.Terminated++
			default:
				summary.Skipped++
			}
		}
		summaries = append(summaries, summary)
	}

	writeJSON(map[string]interface{}{
		"dryRun":  !*apply,
		"tenants": summaries,
	})
	os.Exit(exitCode)
}

func resolveTenants(ctx context.Context, client *ent.Client, tenantID int) ([]*ent.Tenant, error) {
	query := client.Tenant.Query()
	if tenantID > 0 {
		query = query.Where(tenant.ID(tenantID))
	} else {
		query = query.Where(tenant.StatusEQ("active"))
	}
	tenants, err := query.Order(ent.Asc(tenant.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(tenants) == 0 {
		return nil, fmt.Errorf("no matching tenant")
	}
	return tenants, nil
}

func writeJSON(value interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		exitf("encode result: %v", err)
	}
}

func exitf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
