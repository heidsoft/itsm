// BPMN 补偿管理面命令：对指定租户的流程实例显式触发补偿。
// 默认 dry-run（只列出将补偿的活动），显式传 -apply 才写入。
//
// 补偿范围来自 ServiceTask 完成时登记的 _compensationRegistered；流程定义里
// 挂在活动上的补偿边界事件（boundaryEvent + compensateEventDefinition）会
// 被用来解析 activityRef。重复触发由 _compensationExecuted 幂等。
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
	tenantID := flag.Int("tenant-id", 0, "流程实例所属租户 ID（必填，禁止跨租户触发）")
	instanceID := flag.Int("instance-id", 0, "流程实例 ID（必填）")
	actorID := flag.Int("actor-id", 0, "操作人用户 ID（必填，写入审计）")
	activityRef := flag.String("activity-ref", "", "只补偿指定活动；留空则按边界事件声明或全部已登记活动")
	reason := flag.String("reason", "", "补偿原因（必填，写入审计与留痕）")
	apply := flag.Bool("apply", false, "实际执行补偿；缺省只做 dry-run")
	flag.Parse()

	if *tenantID <= 0 || *instanceID <= 0 || *actorID <= 0 {
		fmt.Fprintln(os.Stderr, "-tenant-id / -instance-id / -actor-id are required and must be positive")
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
	engine := service.NewCustomProcessEngine(client, sugar).(*service.CustomProcessEngine)
	result, err := engine.TriggerCompensation(ctx, service.CompensationTriggerRequest{
		TenantID:    *tenantID,
		ActorUserID: *actorID,
		InstanceID:  *instanceID,
		ActivityRef: *activityRef,
		Reason:      *reason,
		DryRun:      !*apply,
	})
	if err != nil {
		writeJSON(map[string]interface{}{
			"dryRun": !*apply,
			"error":  err.Error(),
		})
		os.Exit(1)
	}
	writeJSON(result)
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
