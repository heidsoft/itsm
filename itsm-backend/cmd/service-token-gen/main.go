// Command service-token-gen 签发 itsm-ai-service 回连 Go 后端用的受限 service JWT。
//
// Phase B（2026-11-17，G3b）：sidecar 回连此前把 ITSM_BACKEND_TOKEN 当普通 Bearer
// 发到 /api/v1/ai/*，但后端 AuthMiddleware 强制 JWT 解析——配随机串会 401 并
// 静默降级。本工具签发的 token 带签名保护的 principal=service 声明，后端据此
// 注入 authz.PrincipalService，进入 Gate 2.5 零信任判定；角色固定为只读载体。
//
// 用法（密钥只从环境变量读，绝不落盘/进仓库）：
//
//	JWT_SECRET=<后端同一密钥> go run ./cmd/service-token-gen \
//	    -tenant-id 1 -ttl 8760h > /tmp/service.jwt
//
// 输出只有 token 一行（stderr 是诊断），便于 shell 直接注入 .env.prod：
//
//	ITSM_BACKEND_TOKEN=$(JWT_SECRET=... go run ./cmd/service-token-gen -tenant-id 1)
//
// 轮换：重新签发并把新值注入两个环境的 ITSM_BACKEND_TOKEN；旧 token 到期即失效。
// 租户 ID 必须与 sidecar 被允许访问的租户一致——禁止按请求参数自报租户。
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"itsm-backend/middleware"
)

func main() {
	tenantID := flag.Int("tenant-id", 0, "fixed tenant ID the service token is scoped to (required, >0)")
	ttl := flag.Duration("ttl", 365*24*time.Hour, "token lifetime (default 8760h = 1 year)")
	username := flag.String("username", "svc-ai-sidecar", "auditable service identity name")
	flag.Parse()

	if *tenantID <= 0 {
		fmt.Fprintln(os.Stderr, "-tenant-id must be positive")
		os.Exit(2)
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "JWT_SECRET environment variable is required (must match itsm-backend)")
		os.Exit(2)
	}

	token, err := middleware.GenerateServiceToken(
		*username, middleware.ServicePrincipalRole, *tenantID, secret, *ttl,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign service token: %v\n", err)
		os.Exit(1)
	}

	// stdout 只有 token，便于命令替换；诊断信息一律走 stderr。
	fmt.Fprintln(os.Stdout, token)
}
