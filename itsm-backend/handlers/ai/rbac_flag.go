package ai

import (
	"os"
	"strings"
	"sync"
)

// P2-6 AI 工具 RBAC 校验 Feature Flag
//
// 默认 off（兼容历史行为）。
// 启用方式：
//   AI_TOOL_RBAC_ENABLED=true     // 校验失败时记录 denied 审计日志，但不拦截请求（影子模式）
//   AI_TOOL_RBAC_ENFORCE=true     // 校验失败时实际拦截请求（执行模式）
//
// 影子模式先行 3-7 天，确认无误杀后再开启 ENFORCE 模式。
// 两个开关均为 off 时，Gate 2 跳过校验，permission_check 写入 "skipped"。

var (
	rbacFlagOnce     sync.Once
	rbacFlagEnabled  bool
	rbacFlagEnforce  bool
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
