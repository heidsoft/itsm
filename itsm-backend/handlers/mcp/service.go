package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// Service 是 MCP 协议服务层。
//
// 职责边界：只做「协议语义 → 领域调用」的编排。
// 权限判定、审批、审计、交付门禁全部由注入的领域能力承担，本层不重复实现。
type Service struct {
	catalog     ToolCatalog
	executor    ToolExecutor
	entitlement EntitlementChecker
	logger      *zap.SugaredLogger
	version     string
}

// Options 是 Service 的装配选项。
type Options struct {
	Catalog     ToolCatalog
	Executor    ToolExecutor
	Entitlement EntitlementChecker
	Logger      *zap.SugaredLogger
	Version     string
}

// NewService 构造 MCP 服务。
func NewService(opts Options) (*Service, error) {
	if opts.Catalog == nil {
		return nil, errors.New("mcp: nil ToolCatalog")
	}
	if opts.Executor == nil {
		return nil, errors.New("mcp: nil ToolExecutor")
	}
	if opts.Version == "" {
		opts.Version = "0.0.0"
	}
	if opts.Logger == nil {
		opts.Logger = zap.NewNop().Sugar()
	}
	return &Service{
		catalog:     opts.Catalog,
		executor:    opts.Executor,
		entitlement: opts.Entitlement,
		logger:      opts.Logger,
		version:     opts.Version,
	}, nil
}

// Caller 是 MCP 调用的身份上下文。
//
// 由传输层从认证结果构造。对应 ITSM 的三主体模型：客户端持 service JWT
// 时 principal=service；持用户 JWT 时 principal=human。
type Caller struct {
	// UserID 调用方用户 ID。service 主体恒为 0。
	UserID int
	// TenantID 所属租户。
	TenantID int
	// Role 载体角色名。service 主体固定为 audit_admin。
	Role string
	// Principal 主体类型（human / agent / service）。
	Principal string
}

// Handle 处理一条 JSON-RPC 请求，返回响应。
//
// 通知（无 id）返回 nil —— 规范要求通知不产生响应。
func (s *Service) Handle(ctx context.Context, req *Request, caller Caller) *Response {
	if req == nil {
		return &Response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: NewError(CodeInvalidRequest, "请求体不能为空")}
	}
	if req.Method == MethodInitialized {
		// initialized 是客户端就绪通知：无业务语义，仅确认生命周期完成。
		return nil
	}
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodPing:
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case MethodToolsList:
		return s.handleToolsList(ctx, req, caller)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req, caller)
	default:
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: NewError(CodeMethodNotFound, fmt.Sprintf("不支持的 MCP 方法: %s", req.Method))}
	}
}

// handleInitialize 完成协议版本协商与能力声明。
func (s *Service) handleInitialize(req *Request) *Response {
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: NewError(CodeInvalidParams, "initialize params 非法: "+err.Error())}
		}
	}
	// 版本协商：客户端报的版本不是我们支持的，就回自己支持的版本，
	// 由客户端决定是否断开（规范 Lifecycle 章节要求）。
	negotiated := ProtocolVersion
	if params.ProtocolVersion != "" && params.ProtocolVersion != ProtocolVersion {
		s.logger.Warnw("mcp protocol version mismatch",
			"client_version", params.ProtocolVersion, "server_version", ProtocolVersion)
	}
	result := InitializeResult{
		ProtocolVersion: negotiated,
		Capabilities: map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		ServerInfo: ServerInfo{
			Name:    "cloudmesh-itsm-mcp",
			Title:   "CloudMesh ITSM MCP Server",
			Version: s.version,
		},
		Instructions: "本服务端暴露 CloudMesh ITSM 的只读运维能力（事件统计、CMDB 查询、知识检索等）。" +
			"写操作若存在，需经服务端审批流由人工放行；工具列表按租户已购权益动态呈现。",
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}

// handleToolsList 返回该租户权益内的工具面。
//
// 可见性判定 = 工具自身存在 ∧ 交付权益允许。
// 权益检查失败时不静默放行 —— fail-closed，宁可少列工具不可越权。
func (s *Service) handleToolsList(ctx context.Context, req *Request, caller Caller) *Response {
	defs := s.catalog.ListToolsForTenant(ctx, caller.TenantID)
	tools, projErrs := ProjectTools(defs)
	if len(projErrs) > 0 {
		for _, err := range projErrs {
			s.logger.Warnw("mcp tool projection failed, skipped", "error", err, "tenant_id", caller.TenantID)
		}
	}
	visible := make([]Tool, 0, len(tools))
	for _, t := range tools {
		allowed, reason := s.checkEntitlement(ctx, caller, t.Name)
		if !allowed {
			s.logger.Debugw("mcp tool hidden by entitlement",
				"tool", t.Name, "tenant_id", caller.TenantID, "reason", reason)
			continue
		}
		visible = append(visible, t)
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: ListToolsResult{Tools: visible}}
}

// checkEntitlement 判定工具在当前租户下的交付权益。
//
// 没有装配权益判定器时一律放行 —— 此时 MCP 通道由服务自身的租户隔离兜底，
// 但会失去「按购买权益收敛工具面」的商业化能力。此处选择放行而非拒绝，
// 是为了让 MCP 能力可独立于商业化开关先跑起来；一旦装配门禁即自动收紧。
func (s *Service) checkEntitlement(ctx context.Context, caller Caller, code string) (bool, string) {
	if s.entitlement == nil {
		return true, "entitlement checker not configured"
	}
	ent, err := s.entitlement.Check(ctx, caller.TenantID, code)
	if err != nil {
		// 权益查询出错 → fail-closed 隐藏该工具。
		s.logger.Warnw("mcp entitlement check failed, tool hidden",
			"tool", code, "tenant_id", caller.TenantID, "error", err)
		return false, "entitlement check failed"
	}
	if !ent.Allowed {
		return false, ent.Reason
	}
	return true, ""
}

// handleToolsCall 执行一次工具调用。
//
// 执行链：主体授权 → 领域执行（内含 Gate 2/2.5/审批/审计）→ 结果封装。
// 权限拒绝属业务失败（isError=true），不属协议错误。
func (s *Service) handleToolsCall(ctx context.Context, req *Request, caller Caller) *Response {
	var params CallToolParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: NewError(CodeInvalidParams, "tools/call params 非法: "+err.Error())}
		}
	}
	if params.Name == "" {
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: NewError(CodeInvalidParams, "tools/call 缺少工具名")}
	}
	// 权益：未购/未装的工具即便知道名字也不可调。
	allowed, reason := s.checkEntitlement(ctx, caller, params.Name)
	if !allowed {
		s.logger.Warnw("mcp tool call denied by entitlement",
			"tool", params.Name, "tenant_id", caller.TenantID, "reason", reason)
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: CallToolResult{
			Content: Content("工具 " + params.Name + " 在当前租户的交付权益内不可用：" + reason),
			IsError: true,
		}}
	}
	// 主体授权：不在本层做。
	//
	// 执行链（ai.Service.ExecuteTool）已内置 Gate 2.5 主体零信任判定，且是按
	// **工具自身的 resource:action** 逐个判定 —— 比通道级一刀切更细
	// （service 主体可被放行只读工具、仍被拒写工具）。若本层再按通道码判一次，
	// 会形成两套主体判定：更粗的那套会先放行、把细的那套变成摆设。
	//
	// 因此本层只做协议编排与交付门禁，主体权限一律下沉到执行链。
	args := params.Arguments
	if args == nil {
		args = map[string]interface{}{}
	}
	res, invocationID, err := s.executor.ExecuteTool(ctx, caller.UserID, caller.TenantID, caller.Role, params.Name, args)
	if err != nil {
		s.logger.Warnw("mcp tool execution failed",
			"tool", params.Name, "tenant_id", caller.TenantID, "principal", caller.Principal, "error", err)
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: CallToolResult{
			Content: Content("工具 " + params.Name + " 执行失败：" + err.Error()),
			IsError: true,
		}}
	}
	result := CallToolResult{Content: Content(formatResult(res))}
	if invocationID > 0 {
		// invocation ID 回传便于客户端在 ITSM 审计台账里定位本次调用。
		result.Meta = map[string]any{"invocationId": invocationID}
	}
	// 结构化结果：MCP 规范要求 structuredContent 是 JSON 对象。
	// 工具返回数组时包成 {"items":[...]}，与投影层的 outputSchema
	// （projectOutputSchema 的 items 包装）保持同一形状 —— schema 与内容
	// 不一致会让客户端按 schema 校验失败。
	if structured, ok := wrapStructuredContent(res); ok {
		result.StructuredContent = structured
	}
	return &Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}

// wrapStructuredContent 把工具结果整形成 MCP 允许的 structuredContent 形状。
//
// MCP 规范：structuredContent 是 JSON **对象**。因此：
//   - map 结果    → 原样作为对象
//   - slice 结果  → {"items": [...]}
//   - 标量/其它   → {"value": x}
//
// 形状必须与 toolprojection.projectOutputSchema 的包装规则一致，
// 否则客户端按 outputSchema 校验会失败。
func wrapStructuredContent(res interface{}) (map[string]interface{}, bool) {
	switch v := res.(type) {
	case map[string]interface{}:
		return v, true
	case []interface{}:
		return map[string]interface{}{"items": v}, true
	case nil:
		return nil, false
	default:
		return map[string]interface{}{"value": v}, true
	}
}

// formatResult 把工具结果格式化为 MCP 文本内容。
//
// 结构化结果优先用缩进 JSON（对 LLM 更易解析）；非对象结果退化为
// fmt 的默认格式，不丢信息。
func formatResult(res interface{}) string {
	if res == nil {
		return ""
	}
	if obj, ok := res.(map[string]interface{}); ok {
		b, err := json.MarshalIndent(obj, "", "  ")
		if err == nil {
			return string(b)
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%v", res))
}
