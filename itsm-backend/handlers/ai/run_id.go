package ai

// Agent 运行轮次（run）标识的生成与传播（2026-11-21）。
//
// 为什么需要 run_id：
//
// 战略文档 C 段要求「不可抵赖审计 + 执行证据、策略版本与回放」。现状是
// tool_invocations / ai_feedbacks(ai_audit) / auditlogs 三处审计各自可查，
// 但**一次 Agent 会话里的多次工具调用无法作为整体取出**：
//   - 聊天路径的一轮对话可能触发 3~5 次工具调用，各自带不同的 X-Request-Id；
//   - 调查编排（investigation.go）每步一次 ExecuteTool，步骤间无共同标识；
//   - MCP 会话由客户端驱动，服务端本无「这是一次连续会话」的概念。
//
// 于是「这个 Agent 这一轮做了什么」这个问题在数据层无解。缺的不是表，
// 而是一个跨请求贯穿的字段——本文件只负责生成与传播这个值。
//
// 与 request_id 的分工（两者不得互相冒充）：
//   - request_id（middleware 注入）= **单次 HTTP 请求**，与 auditlogs 对齐，
//     回答「这次请求打了哪些审计点」；
//   - run_id（本文件）= **一次逻辑运行**，可跨多个 HTTP 请求，回答
//     「这个 Agent 这一轮做了哪些动作、每步依据哪版策略放行」。
//
// 刻意不放进 JWT：run 是执行期的临时上下文，一次登录可以发起任意多个 run；
// 也不进 ToolInvocation 的构造参数（那会让四条通道都改签名）。

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// runIDContextKey 是运行轮次标识的 context 键（未导出类型，避免跨包冲突）。
type runIDContextKey struct{}

// WithRunID 把运行轮次标识注入 context。
//
// 调用方在「一轮逻辑运行的起点」生成一次并传播：
//   - ChatStream：一轮对话开始时；
//   - StartInvestigation：一次调查编排开始时；
//   - MCP initialize：一次会话建立时（服务端生成的会话轮次）。
//
// 已有的 run_id 不会被覆盖：嵌套调用（如 skill 内部再调工具）必须落在
// 同一个 run 里，否则证据链再次断裂。
func WithRunID(ctx context.Context, runID string) context.Context {
	if runID == "" {
		return ctx
	}
	if existing, ok := RunIDFromContext(ctx); ok && existing != "" {
		return ctx
	}
	return context.WithValue(ctx, runIDContextKey{}, runID)
}

// RunIDFromContext 取运行轮次标识；未注入时 ok=false。
func RunIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(runIDContextKey{}).(string)
	return v, ok && v != ""
}

// NewRunID 生成一个运行轮次标识。
//
// 禁止用当前时间拼身份（与 ledgerRequestID 同一条纪律）：重试/重放场景下
// 时间戳不稳定、同一纳秒可能碰撞。crypto/rand 失败属极罕见，退化为零字节
// hex 仍是合法标识（唯一性下降但语义正确，不伪造时间语义）。
//
// 前缀 "run_" 让日志与 DB 里一眼可辨这是轮次而非请求 ID。
func NewRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 与 middleware.generateRequestID 同口径：退化为零字节 hex。
	}
	return "run_" + hex.EncodeToString(b[:])
}

// EnsureRunID 保证 context 里有一个 run_id：已有则复用，没有则生成并注入。
// 返回 (可能已增强的 ctx, 生效的 run_id)，调用方不需要再判断。
//
// 这是执行链的单一入口：ExecuteTool 与 MCP 都调它，避免两处各写一份
// 「有就用、没有就造」的逻辑（那种写法必然有一处忘记回写 ctx）。
func EnsureRunID(ctx context.Context) (context.Context, string) {
	if id, ok := RunIDFromContext(ctx); ok {
		return ctx, id
	}
	id := NewRunID()
	return WithRunID(ctx, id), id
}
