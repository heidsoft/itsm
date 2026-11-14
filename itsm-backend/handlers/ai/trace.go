package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"itsm-backend/middleware"
)

// ledgerRequestID 为 AI 决策底账生成 request ID。
//
// 禁止用当前时间（Unix/UnixNano）拼身份：AGENTS.md 幂等规则明确「禁止用
// 显示文本、当前时间或 retry count 作为幂等身份」，且重试/重放场景下时间戳
// 不稳定、同一纳秒内可能碰撞。这里统一收敛为：
//  1. 优先复用请求链路注入的 X-Request-Id（middleware.RequestIDFromContext），
//     让底账的 request_id 与结构化日志、metric 归因同源；
//  2. 没有链路 ID 时（skill 调用、后台任务、无 middleware 的测试路由），
//     生成 128-bit 随机 ID 兜底，仍然不依赖时间。
//
// scenario 前缀（chat/summary/forecast/triage/feedback/ai_audit）仅用于人读，
// 与 kind 列语义重复但保持旧前缀的日志可读性。
func ledgerRequestID(ctx context.Context, scenario string) string {
	if rid, ok := middleware.RequestIDFromContext(ctx); ok && rid != "" {
		return scenario + "_" + rid
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand 失败极罕见；与 middleware.generateRequestID 同口径，
		// 退化为零字节的 hex，仍可区分调用场景且不携带时间语义。
	}
	return scenario + "_" + hex.EncodeToString(b[:])
}
