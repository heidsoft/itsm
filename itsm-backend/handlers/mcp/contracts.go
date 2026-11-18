package mcp

import (
	"context"

	"itsm-backend/dto"
)

// ToolCatalog 是工具面的窄契约。
//
// 方法集 = 本包全部生产文件对工具注册表的调用并集：
//   - 列举（tools/list）：ListToolsForTenant
//   - 执行（tools/call）：走 ToolExecutor（不是直接拿注册表 Execute，
//     因为执行必须经过 ai.Service.ExecuteTool 的完整判定链）
//
// 为什么要窄契约而不是直接持有 *ToolRegistry：
// 本包是协议适配层，不该把 AI 域的执行语义整个吞进来。窄契约让
// 「列举」与「执行」两条路径各自只依赖最小的面，也便于契约测试替身注入。
type ToolCatalog interface {
	// ListToolsForTenant 返回该租户可见的工具定义。
	ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition
}

// ToolExecutor 是工具执行的窄契约。
//
// 实现方是 handlers/ai.Service.ExecuteTool 的适配器 —— 它内部串好
// Gate 2 角色 RBAC、Gate 2.5 主体零信任、写工具审批流与审计落库。
// MCP 层不允许绕过它自己执行工具。
type ToolExecutor interface {
	// ExecuteTool 执行工具。返回结果与 invocation ID（只读工具恒为 0，
	// 审计异步落库），错误由调用方映射为 MCP 工具执行错误。
	ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error)
}

// EntitlementChecker 是交付权益判定的窄契约。
//
// 实现方是 service/marketplace.Service.SkillInvocationEntitlement ——
// 「目录上架 + 租户安装才可用」的既有门禁。MCP 暴露的工具面必须受同一门禁约束，
// 否则 MCP 通道会成为绕过市场交付的旁路。
//
// 注意：方法签名返回 marketplace.SkillEntitlement。本包不 import 该包，
// 而由装配层提供适配器把结果投影成本包的 Entitlement 视图，
// 避免协议层与市场实现耦合。
type EntitlementChecker interface {
	// Check 返回某工具码在指定租户下的交付权益。
	Check(ctx context.Context, tenantID int, code string) (Entitlement, error)
}

// Entitlement 是交付权益的投影视图。
type Entitlement struct {
	// Allowed 为真表示该租户已安装并可调用该工具码。
	Allowed bool
	// Reason 在拒绝时说明原因（未上架 / 未安装 / 未授权），用于审计与客户端提示。
	Reason string
}
