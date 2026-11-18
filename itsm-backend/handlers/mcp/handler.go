package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"itsm-backend/internal/authz"
	"itsm-backend/middleware"
)

// Handler 是 MCP 的 HTTP 传输层。
//
// 传输形态：单端点 POST JSON-RPC 2.0（兼容 MCP Streamable HTTP 的
// 请求/响应模式）。客户端（Claude Desktop / Claude Code / 第三方 Agent）
// 把 initialize、tools/list、tools/call 都打到同一个 URL。
//
// 认证：由外层中间件完成（AuthMiddleware 解析 JWT → 注入 tenant/user/role/principal）。
// 本 handler 只从 gin context 取已认证身份，不自行解析令牌 ——
// 认证只有一处实现，避免出现第二套令牌语义。
type Handler struct {
	service *Service
	logger  *zap.SugaredLogger
}

// NewHandler 构造 MCP HTTP handler。
func NewHandler(svc *Service, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{service: svc, logger: logger}
}

// RegisterRoutes 注册 MCP 路由。
//
// 权限口径：路由级挂 mcp:read —— MCP 通道本身是发现入口（工具执行时的
// 动作权限由执行链按工具逐个判定，不在路由层一刀切）。
//
// 为什么不按 mcp:read / mcp:invoke 分两条路由：MCP 是单端点多方法协议，
// 同一个 URL 上既有 tools/list（读）也有 tools/call（写面）。若按 HTTP 路径
// 区分权限，就必须把 initialize/tools/list/tools/call 拆成不同路径，
// 与 MCP 客户端的协议约定不符。因此路由层只校准入场（mcp:read），
// 调用面的 mcp:invoke 由服务层按 principal 逐次判定。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}
	rg.POST("/mcp", middleware.RequirePermission("mcp", "read"), h.Handle)
}

// Handle 处理一次 JSON-RPC 请求。
//
// 内容类型：接受 application/json 与 Streamable HTTP 常见的
// application/json, text/event-stream 组合，这里统一按 JSON 解析。
func (h *Handler) Handle(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBytes))
	if err != nil {
		writeRPC(c, &Response{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: NewError(CodeParseError, "读取请求体失败")})
		return
	}
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPC(c, &Response{JSONRPC: "2.0", ID: json.RawMessage("null"),
			Error: NewError(CodeParseError, "请求体不是合法 JSON: "+err.Error())})
		return
	}
	caller := CallerFromContext(c)
	resp := h.service.Handle(c.Request.Context(), &req, caller)
	if resp == nil {
		// 通知：HTTP 语义下返回 202 Accepted 无正文。
		c.Status(http.StatusAccepted)
		return
	}
	writeRPC(c, resp)
}

// maxRequestBytes 限制请求体大小，防止超大入参打爆内存。
const maxRequestBytes = 1 << 20 // 1 MiB

// writeRPC 写出 JSON-RPC 响应。
//
// HTTP 状态码一律 200：JSON-RPC 用 body 里的 error 表达协议级失败，
// HTTP 4xx/5xx 会让部分客户端误以为连接层出错而不读 body。
func writeRPC(c *gin.Context, resp *Response) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, resp)
}

// CallerFromContext 从请求上下文取已认证的调用方身份。
//
// 身份一律来自认证中间件注入的值；此处不做任何"兜底猜测"，
// 取不到就返回零值（service 侧会因权限判定失败而拒绝）。
//
// principal 的读法遵循 middleware/auth.go 的约定：授权依据是
// authz.WithPrincipal 写入的 **request context**（Gate 2.5 读它），
// 不是 gin context 裸键。因此这里从 c.Request.Context() 取，
// 避免出现与授权链不一致的第二个来源。
func CallerFromContext(c *gin.Context) Caller {
	caller := Caller{}
	if v, ok := c.Get("tenant_id"); ok {
		caller.TenantID = toInt(v)
	}
	if v, ok := c.Get("user_id"); ok {
		caller.UserID = toInt(v)
	}
	if v, ok := c.Get("role"); ok {
		if s, ok := v.(string); ok {
			caller.Role = s
		}
	}
	if p := authz.PrincipalFromContext(c.Request.Context()); p != "" {
		caller.Principal = p
	}
	return caller
}

// toInt 把 context 里的数值安全转成 int。
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		return 0
	default:
		return 0
	}
}

// ParseBatch 解析批量请求（JSON 数组）。
//
// JSON-RPC 2.0 允许服务端接受批量请求；MCP 规范未强制要求，
// 但多数客户端在并发初始化时会用，这里支持以提升兼容性。
func ParseBatch(body []byte) ([]Request, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, io.ErrUnexpectedEOF
	}
	if trimmed[0] != '[' {
		var r Request
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, err
		}
		return []Request{r}, nil
	}
	var rs []Request
	if err := json.Unmarshal(body, &rs); err != nil {
		return nil, err
	}
	return rs, nil
}
