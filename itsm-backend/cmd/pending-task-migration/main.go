// 存量 pending 待办迁移运维命令：把只读盘点升级为「计划 + 执行」两段。
// 默认 dry-run（只输出动作清单），显式传 -apply 才在单个事务里写入。
//
// 只自动处理两类安全动作：业务已终态但流程未结束的孤儿实例（取消遗留待办
// + 终结实例），以及运维显式指定目标用户的未指派待办；陈旧待办只列 review。
// 每个动作带幂等键并做状态复核，重复运行会得到全 skipped。
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
	"itsm-backend/service"

	"go.uber.org/zap"
)

func main() {
	tenantID := flag.Int("tenant-id", 0, "只处理指定租户（必填，禁止跨租户迁移）")
	actorID := flag.Int("actor-id", 0, "写入审计的操作人 ID（apply 时必填）")
	assigneeID := flag.Int("assignee-id", 0, "未指派待办的改派目标用户 ID；0 表示只列 review")
	reason := flag.String("reason", "", "迁移原因（apply 时必填，写入审计）")
	apply := flag.Bool("apply", false, "实际执行迁移；缺省只做 dry-run")
	maxItems := flag.Int("max-items", 500, "动作清单上限（最大 2000）")
	staleHours := flag.Int("stale-threshold-hours", 24, "陈旧待办阈值（小时，最大 720）")
	flag.Parse()

	if *tenantID <= 0 {
		fmt.Fprintln(os.Stderr, "-tenant-id is required and must be positive")
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

	ctx := tenantctx.WithTenantID(context.Background(), *tenantID)
	opts := service.PendingMigrationOptions{
		TenantID: *tenantID, ActorUserID: *actorID, Reason: *reason,
		AssigneeID: *assigneeID, StaleThresholdHours: *staleHours,
		MaxItems: *maxItems, DryRun: !*apply,
	}
	plan, err := service.BuildPendingMigrationPlan(ctx, client, opts)
	if err != nil {
		exitf("build migration plan: %v", err)
	}
	engine := service.NewCustomProcessEngine(client, sugar).(*service.CustomProcessEngine)
	result, err := engine.ApplyPendingMigration(ctx, plan, opts)
	if err != nil {
		writeJSON(map[string]interface{}{"error": err.Error(), "plan": plan})
		os.Exit(1)
	}
	writeJSON(map[string]interface{}{"plan": plan, "result": result})
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
