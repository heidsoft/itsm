package ai

import (
	"os"
	"strings"
	"sync"
)

// Legacy rollout flags are retained for configuration compatibility only.
// Tool execution and discovery always enforce resource permissions; these
// flags cannot disable authorization or permit execution in shadow mode.

var (
	rbacFlagOnce    sync.Once
	rbacFlagEnabled bool
	rbacFlagEnforce bool
)

// loadRBACFlag 从环境变量读取开关，只执行一次
func loadRBACFlag() {
	rbacFlagOnce.Do(func() {
		rbacFlagEnabled = parseBoolEnv("AI_TOOL_RBAC_ENABLED")
		rbacFlagEnforce = parseBoolEnv("AI_TOOL_RBAC_ENFORCE")
		// ENFORCE 模式隐含 ENABLED
		if rbacFlagEnforce {
			rbacFlagEnabled = true
		}
	})
}

// IsToolRBACEnabled 返回是否开启 AI 工具 RBAC 校验
func IsToolRBACEnabled() bool {
	loadRBACFlag()
	return rbacFlagEnabled
}

// IsToolRBACEnforce 返回是否执行拦截模式（false=影子模式只记录日志）
func IsToolRBACEnforce() bool {
	loadRBACFlag()
	return rbacFlagEnforce
}

// ResetRBACFlagForTest 仅用于测试：重置开关，允许重新加载环境变量
func ResetRBACFlagForTest() {
	rbacFlagOnce = sync.Once{}
	rbacFlagEnabled = false
	rbacFlagEnforce = false
}

func parseBoolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
