package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"itsm-backend/dto"
	"itsm-backend/handlers/mcp"
	"itsm-backend/service/marketplace"
)

// Dependencies 是 MCP 服务端装配所需的领域能力。
//
// 用窄接口而非具体类型声明，便于端到端测试注入替身，同时让「MCP 依赖什么」
// 这件事在类型层面可见 —— 少一个依赖就少一处装配遗漏的可能。
type Dependencies struct {
	// ToolRegistry 提供工具面（列举）。实现方是 internal/services.ToolRegistry。
	ToolRegistry interface {
		ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition
	}
	// AIService 提供工具执行（含 Gate 2 / Gate 2.5 / 审批 / 审计）。
	AIService interface {
		ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error)
	}
	// Marketplace 提供交付门禁（目录上架 + 租户安装才可调）。
	Marketplace interface {
		SkillInvocationEntitlement(ctx context.Context, tenantID int, code string) (marketplace.SkillEntitlement, error)
	}
	// Logger 是装配期与运行期日志出口。
	Logger *zap.SugaredLogger
}

// BuildMCPHandler 构造 MCP 服务端 handler。
//
// 生产装配与端到端测试**共用这一个入口** —— 测试自己拼一套接线的话，
// 测的就不是生产真正跑的那条路径，测试通过也说明不了问题。
//
// 装配失败返回错误而不 panic：调用方（生产 bootstrap）据此 fail-fast，
// 测试据此断言「装配缺件时必须报错」而不是静默降级成空壳。
func BuildMCPHandler(deps Dependencies) (*mcp.Handler, error) {
	if deps.ToolRegistry == nil {
		return nil, errors.New("mcp 装配缺 ToolRegistry：工具面无来源")
	}
	if deps.AIService == nil {
		return nil, errors.New("mcp 装配缺 AIService：执行链无来源")
	}
	if deps.Marketplace == nil {
		return nil, errors.New("mcp 装配缺 Marketplace：交付门禁无来源")
	}
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	svc, err := mcp.NewService(mcp.Options{
		Catalog: &mcp.ToolCatalogAdapter{
			Lister: deps.ToolRegistry,
		},
		Executor: &mcp.ToolExecutorAdapter{
			Executor: deps.AIService,
		},
		Entitlement: &mcp.EntitlementAdapter{
			Checker: deps.Marketplace,
		},
		Logger:  logger,
		Version: mcp.Version(),
	})
	if err != nil {
		return nil, fmt.Errorf("mcp service 构造失败: %w", err)
	}
	return mcp.NewHandler(svc, logger), nil
}
