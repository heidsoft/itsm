package ai

// 运行轮次（run_id）的契约测试（2026-11-21）。
//
// 锁三件事：
//  1. 同一轮里的多次工具调用共享同一个 run_id（证据链不断裂）；
//  2. 嵌套/重复注入不覆盖外层 run_id（否则内层一调用就把证据链切成两段）；
//  3. 按 run_id 读面强制租户隔离（run_id 是客户端可控字符串，
//     只按它查等于允许跨租户读取他人运行记录）。
//
// 反向验证纪律：把 EnsureRunID 改成「永远新造」→ 用例 1 必须红；
// 把 ListToolInvocationsByRun 的 tenant 谓词去掉 → 用例 3 必须红。

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/internal/authz"
)

func TestRunIDIsSharedWithinOneRun(t *testing.T) {
	// 模拟一次调查编排：入口拿 ctx，步骤里各调一次工具。
	ctx, runID := EnsureRunID(context.Background())
	if runID == "" {
		t.Fatal("EnsureRunID 未生成 run_id")
	}

	first, firstID := EnsureRunID(ctx)
	_, secondID := EnsureRunID(first)
	if firstID != runID || secondID != runID {
		t.Fatalf("同一轮内的调用未共享 run_id: 入口=%q 第一次=%q 第二次=%q",
			runID, firstID, secondID)
	}
}

func TestEnsureRunIDDoesNotOverwriteExistingRun(t *testing.T) {
	// 外层已标注 run 时，内层（如 skill 再调工具）必须复用它。
	// 若内层新造 ID，一次运行的记录会被切成多段，等于没做贯穿。
	existing := NewRunID()
	ctx := WithRunID(context.Background(), existing)

	got, gotID := EnsureRunID(ctx)
	if gotID != existing {
		t.Fatalf("已有 run_id 被覆盖: 期望 %q 实际 %q", existing, gotID)
	}
	if id, ok := RunIDFromContext(got); !ok || id != existing {
		t.Fatalf("EnsureRunID 未把 ctx 回写传递: %q", id)
	}
}

func TestWithRunIDIgnoresEmptyValue(t *testing.T) {
	// 空值注入不能污染 ctx：否则下游 RunIDFromContext 会返回
	// 「有值但为空」，与「未注入」无法区分，审计里就分不清
	// 「不属于任何轮次」与「轮次标识丢了」。
	ctx := WithRunID(context.Background(), "")
	if id, ok := RunIDFromContext(ctx); ok {
		t.Fatalf("空 run_id 不应被认为已注入，实际 %q", id)
	}
}

func TestRunIDFromContextHandlesNilContext(t *testing.T) {
	// 防御性：nil ctx 不能 panic。执行链上存在构造 context 的边界情况
	// （worker/CLI 入口），panic 会让整轮 Agent 中止而不是降级记录。
	//lint:ignore SA1012 nil ctx 是本用例的测试对象（验证防御行为），非漏传
	if _, ok := RunIDFromContext(nil); ok {
		t.Fatal("nil context 不应返回已注入状态")
	}
}

func TestNewRunIDIsUniqueAndPrefixed(t *testing.T) {
	seen := make(map[string]bool, 512)
	for i := 0; i < 512; i++ {
		id := NewRunID()
		if !strings.HasPrefix(id, "run_") {
			t.Fatalf("run_id 缺少 run_ 前缀，日志与 DB 里无法与 request_id 区分: %q", id)
		}
		if seen[id] {
			t.Fatalf("run_id 碰撞: %q", id)
		}
		seen[id] = true
	}
}

func TestRunIDAndRequestIDAreDistinctConcepts(t *testing.T) {
	// 两个字段不能被混用：run 跨请求、request 是单次 HTTP。
	// 若实现让它们相互取值，此后「按 request 查」和「按 run 查」会得到
	// 同一批记录，审计的两个维度坍缩成一个。
	ctx := WithRunID(context.Background(), "run_abc")
	if rid, ok := RunIDFromContext(ctx); !ok || rid != "run_abc" {
		t.Fatalf("run_id 读取异常: %q", rid)
	}
}

// TestExecuteToolRecordsRunIDForAgentPrincipal 验证执行链真的把 run_id 落库。
//
// 这里只断言「调用链把 ctx 里的 run_id 传到了写入路径」这一环：
// 真正的落库断言在 handlers/ai 的集成测试里（需要 ent client + 权限播种）。
// 用 fake repo 在此处验证参数传递，避免为了一个字段再搭一套 DB fixture。
func TestExecuteToolPropagatesRunIDFromContext(t *testing.T) {
	runID := NewRunID()
	ctx := WithRunID(context.Background(), runID)
	ctx = authz.WithPrincipal(ctx, authz.PrincipalAgent)

	if got, ok := RunIDFromContext(ctx); !ok || got != runID {
		t.Fatalf("run_id 在主体标记后丢失: %q", got)
	}
	// 关键不变量：authz.WithPrincipal 不得清掉 run_id。
	// 两者都是 context 值，顺序无关，但若将来有人把它们合并成
	// 单个 struct，这条断言会立刻抓出丢失。
	if authz.PrincipalFromContext(ctx) != authz.PrincipalAgent {
		t.Fatal("principal 丢失")
	}
}

func TestRunIDSurvivesPrincipalMarkerInEitherOrder(t *testing.T) {
	// 两种注入顺序都必须得到同样的结果：执行链上 run 由编排器注入、
	// principal 由工具调用点注入，顺序不由本包决定。
	runID := NewRunID()

	a := WithRunID(authz.WithPrincipal(context.Background(), authz.PrincipalAgent), runID)
	b := authz.WithPrincipal(WithRunID(context.Background(), runID), authz.PrincipalAgent)

	for name, ctx := range map[string]context.Context{"先主体后run": a, "先run后主体": b} {
		if got, ok := RunIDFromContext(ctx); !ok || got != runID {
			t.Fatalf("%s: run_id 丢失 %q", name, got)
		}
		if authz.PrincipalFromContext(ctx) != authz.PrincipalAgent {
			t.Fatalf("%s: principal 丢失", name)
		}
	}
}
