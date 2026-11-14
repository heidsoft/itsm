package ai

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/middleware"
)

// TestLedgerRequestID_NoTimeBasedIdentity 验证底账 request ID 不携带时间语义：
// AGENTS.md 幂等规则禁止用当前时间拼身份，旧实现 chat_%d_%d(tr UnixNano)
// 会在重试/重放时漂移，同一纳秒内还可能碰撞。
func TestLedgerRequestID_NoTimeBasedIdentity(t *testing.T) {
	rid := ledgerRequestID(context.Background(), "chat")

	if !strings.HasPrefix(rid, "chat_") {
		t.Fatalf("scenario prefix missing: %q", rid)
	}
	suffix := strings.TrimPrefix(rid, "chat_")
	if suffix == "" {
		t.Fatal("random suffix must not be empty")
	}
	if strings.ContainsAny(suffix, "0123456789") && isUnixNanoLike(suffix) {
		t.Fatalf("suffix must not be a timestamp-like integer: %q", suffix)
	}

	// 同参数两次生成必须不同（随机性，而非时间戳确定性）。
	again := ledgerRequestID(context.Background(), "chat")
	if again == rid {
		t.Fatalf("two calls must produce distinct IDs, got %q twice", rid)
	}
}

// TestLedgerRequestID_ReusesRequestContext 验证链路里已有 X-Request-Id 时
// 底账 request_id 与日志/metric 归因同源，而不是另造一套。
func TestLedgerRequestID_ReusesRequestContext(t *testing.T) {
	const wired = "req-abc-123"
	ctx := middleware.WithRequestID(context.Background(), wired)

	rid := ledgerRequestID(ctx, "triage")
	if rid != "triage_"+wired {
		t.Fatalf("want triage_%s, got %q", wired, rid)
	}
}

// isUnixNanoLike 判断字符串是否是纯十进制整数（时间戳形态）。
func isUnixNanoLike(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
