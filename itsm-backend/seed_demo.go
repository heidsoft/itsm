//go:build seed_demo
// +build seed_demo

// 演示数据播种 CLI：make dev-seed-demo 底层入口。
//
// 用法（在 itsm-backend/ 目录下，连接参数经 config.yaml 的 ${ENV:default} 插值）：
//
//	DB_HOST=localhost DB_PORT=55432 DB_USER=itsm_user DB_PASSWORD=dev123 \
//	ITSM_SEED_CONFIG=config/seed/demo.json go run -tags seed_demo .
//
// 行为：执行完整 SeedAll（租户/角色/菜单/SLA/服务目录等功能模板 + 演示业务记录），
// 全程幂等，可在已初始化的环境上重复执行。生产配置（默认 default.json）不含
// 业务记录数组，因此本入口不会给生产部署引入虚构数据。

package main

import (
	"context"
	"fmt"
	"os"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/pkg/seeder"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	client, err := database.InitDatabaseWithRLS(&cfg.Database, &cfg.RLS, sugar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx := tenantctx.SystemContext(
		context.Background(),
		"seed-demo:cli",
		"seed demo data via make dev-seed-demo",
	)

	s := seeder.NewSeeder(client, sugar, cfg)
	s.SeedAll(ctx)

	fmt.Println("=== demo seed finished ===")
	fmt.Println("功能模板（租户/角色/菜单/SLA/服务目录/BPMN）与演示业务记录已就绪。")
	fmt.Println("默认管理员: admin（密码取 ADMIN_PASSWORD，dev 环境为 admin123）")
}
