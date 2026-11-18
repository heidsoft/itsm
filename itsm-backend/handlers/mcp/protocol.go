// Package mcp 实现 Model Context Protocol（MCP）服务端骨架。
//
// 定位：MCP 是**对外暴露 ITSM 能力的协议适配层**，不是新的业务域。
//
// 设计约束（为什么不重造轮子）：
//
//  1. 工具面不新建。工具定义直接来自 internal/services 的 ToolRegistry
//     （dto.ToolDefinition 已含 ReadOnly / Resource / Action / ArgsSchema），
//     MCP 层只做「协议形状投影」，不新增业务工具、不新增第二套执行器。
//
//  2. 判定不新建。工具调用复用 handlers/ai.Service.ExecuteTool —— 该入口
//     已串好 Gate 2（角色 RBAC）、Gate 2.5（主体零信任，DB 策略）、
//     写工具审批流与审计落库。MCP 层不得自造权限判定。
//
//  3. 交付不新建。工具可见性复用 marketplace 的技能交付门禁
//     （SkillInvocationEntitlement）：目录上架 + 租户安装才可见/可调。
//
//  4. 零外部依赖。协议编解码为原生实现，不引入第三方 MCP SDK ——
//     供应链风险面最小，且与仓库「无 JSON-RPC 既有实现」的现状自洽。
//
// 协议参考：MCP 规范 2025-06-18（initialize / tools/list / tools/call）。
package mcp

import "encoding/json"

// ProtocolVersion 是本服务端支持的 MCP 协议版本。
//
// 版本协商规则（规范 Lifecycle 章节）：客户端在 initialize 里报自己支持的版本，
// 服务端若不支持则回一个自己支持的版本；客户端若也不支持则断开。
const ProtocolVersion = "2025-06-18"

// JSON-RPC 2.0 错误码（仅用规范定义的标准码）。
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Request 是 JSON-RPC 2.0 请求信封。
//
// 注意 ID 用 json.RawMessage 而非 int：JSON-RPC 2.0 允许 id 为字符串或数字，
// 用 RawMessage 原样回显可避免「收到 1 回 1」的强制转换丢类型。
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification 判断是否为通知（无 id）。通知不产生响应。
func (r *Request) IsNotification() bool {
	return len(r.ID) == 0 || string(r.ID) == "null"
}

// Response 是 JSON-RPC 2.0 响应信封。
//
// 成功时 Result 非空；失败时 Error 非空。两者互斥，二者皆空即为非法响应。
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError 是 JSON-RPC 2.0 错误对象。
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return e.Message }

// NewError 构造一个 JSON-RPC 错误。
func NewError(code int, message string) *RPCError {
	return &RPCError{Code: code, Message: message}
}

// NewErrorWithData 构造带 data 字段的 JSON-RPC 错误。
func NewErrorWithData(code int, message string, data interface{}) *RPCError {
	return &RPCError{Code: code, Message: message, Data: data}
}

// 协议方法名。
const (
	MethodInitialize  = "initialize"
	MethodInitialized = "notifications/initialized"
	MethodPing        = "ping"
	MethodToolsList   = "tools/list"
	MethodToolsCall   = "tools/call"
)

// ServerInfo 描述服务端实现信息。
type ServerInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// ClientInfo 描述客户端实现信息（用于审计留痕）。
type ClientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// InitializeParams 是 initialize 请求的 params。
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	ClientInfo      ClientInfo     `json:"clientInfo"`
}

// InitializeResult 是 initialize 响应的 result。
//
// 只声明本服务端真实实现的能力：tools + logging。
// 不声明 prompts / resources / completions —— 未实现的 capability 一旦声明，
// 客户端会尝试调用并得到「method not found」，属于虚假承诺。
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

// ToolAnnotations 是 MCP 工具的行为标注。
//
// 规范明确「客户端 MUST 把工具标注视为不可信」，因此这里只是把 ITSM 侧
// 既有的事实（是否只读、是否破坏性、是否幂等、是否触达外部世界）如实投影，
// 不作为安全边界 —— 安全边界由服务端自身的权限判定强制。
type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   bool   `json:"openWorldHint,omitempty"`
}

// Tool 是 MCP 工具描述。
//
// InputSchema 必须是合法的 JSON Schema（type=object）。ArgsSchema 若缺失，
// 退化为空对象 schema —— 宁可给客户端一个明确的无参契约，也不给非法 schema。
type Tool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ListToolsResult 是 tools/list 的 result。
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// CallToolParams 是 tools/call 的 params。
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ContentItem 是 MCP 内容项。本服务端当前只产出 text 类型。
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallToolResult 是 tools/call 的 result。
//
// 业务执行失败（如权限拒绝、工具报错）走 isError=true + 文本说明，
// 而不是 JSON-RPC error —— 规范区分「协议错误」与「工具执行错误」，
// 前者说明请求本身不合法，后者说明请求合法但业务失败。
type CallToolResult struct {
	Content           []ContentItem     `json:"content"`
	StructuredContent interface{}       `json:"structuredContent,omitempty"`
	IsError           bool              `json:"isError,omitempty"`
	Meta              map[string]any    `json:"_meta,omitempty"`
}

// Content 构造单条 text 内容项。
func Content(text string) []ContentItem {
	return []ContentItem{{Type: "text", Text: text}}
}
