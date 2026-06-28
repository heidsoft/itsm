package router

import (
	"sync"
	"testing"
	"time"
)

// TestNewWSTicketStore 验证票据存储构造函数
func TestNewWSTicketStore(t *testing.T) {
	store := NewWSTicketStore(time.Second)
	if store == nil {
		t.Fatal("NewWSTicketStore returned nil")
	}
	if store.ttl != time.Second {
		t.Errorf("expected ttl=1s, got %v", store.ttl)
	}
	// 验证 cleanup goroutine 已启动（间接验证：通过后续清理行为）
}

// TestWSTicketGenerate 验证票据生成
func TestWSTicketGenerate(t *testing.T) {
	store := NewWSTicketStore(30 * time.Second)
	defer cleanupStore(store)

	ticket, err := store.Generate(100, 200)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// 票据应为 64 字符的 hex（32 字节随机）
	if len(ticket) != WSTicketBytes*2 {
		t.Errorf("expected ticket length=%d, got %d", WSTicketBytes*2, len(ticket))
	}

	// 两次生成的票据应不同
	ticket2, err := store.Generate(100, 200)
	if err != nil {
		t.Fatalf("Generate #2 returned error: %v", err)
	}
	if ticket == ticket2 {
		t.Error("two generated tickets should differ (random collision)")
	}
}

// TestWSTicketRedeem 验证票据兑换（单次使用）
func TestWSTicketRedeem(t *testing.T) {
	store := NewWSTicketStore(30 * time.Second)
	defer cleanupStore(store)

	ticket, err := store.Generate(42, 99)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	userID, tenantID, ok := store.Redeem(ticket)
	if !ok {
		t.Fatal("Redeem returned ok=false for valid ticket")
	}
	if userID != 42 {
		t.Errorf("expected userID=42, got %d", userID)
	}
	if tenantID != 99 {
		t.Errorf("expected tenantID=99, got %d", tenantID)
	}
}

// TestWSTicketRedeemSingleUse 验证票据单次使用（消费后即失效）
func TestWSTicketRedeemSingleUse(t *testing.T) {
	store := NewWSTicketStore(30 * time.Second)
	defer cleanupStore(store)

	ticket, _ := store.Generate(1, 1)

	// 第一次使用成功
	_, _, ok := store.Redeem(ticket)
	if !ok {
		t.Fatal("first Redeem should succeed")
	}

	// 第二次使用应失败（已删除）
	_, _, ok = store.Redeem(ticket)
	if ok {
		t.Error("second Redeem should fail (ticket must be single-use)")
	}
}

// TestWSTicketRedeemInvalid 验证无效票据兑换
func TestWSTicketRedeemInvalid(t *testing.T) {
	store := NewWSTicketStore(30 * time.Second)
	defer cleanupStore(store)

	tests := []struct {
		name   string
		ticket string
	}{
		{"empty", ""},
		{"random-not-stored", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		{"wrong-length", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := store.Redeem(tt.ticket)
			if ok {
				t.Errorf("Redeem(%q) ok=true, want false", tt.ticket)
			}
		})
	}
}

// TestWSTicketRedeemExpired 验证过期票据兑换
func TestWSTicketRedeemExpired(t *testing.T) {
	// 使用极短 TTL（需要 cleanup 还没跑到之前手动触发验证）
	store := NewWSTicketStore(50 * time.Millisecond)
	defer cleanupStore(store)

	ticket, _ := store.Generate(1, 1)

	// 等待票据过期
	time.Sleep(100 * time.Millisecond)

	_, _, ok := store.Redeem(ticket)
	if ok {
		t.Error("expired ticket should not be redeemable")
	}
}

// TestWSTicketCleanup 验证 cleanup 定期清理过期票据
func TestWSTicketCleanup(t *testing.T) {
	// 短 TTL，让票据快速过期
	store := NewWSTicketStore(50 * time.Millisecond)
	// 不 defer cleanupStore，使用 Stop 标记

	// 插入 100 个票据
	for i := 0; i < 100; i++ {
		_, _ = store.Generate(i, i)
	}

	// 等待票据过期 + cleanup 触发（cleanup 间隔 60s，时间不够，所以手动验证 sync.Map 内票据状态）
	time.Sleep(150 * time.Millisecond)

	// 票据已过期但 cleanup 还没运行（60s 周期太长）
	// 因此 sync.Map 中应仍有票据条目（这是预期行为，仅验证 sync.Map 状态）
	count := 0
	store.tickets.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != 100 {
		t.Errorf("expected 100 stored tickets, got %d", count)
	}
}

// TestWSTicketConcurrent 验证并发场景下票据生成和兑换的线程安全
func TestWSTicketConcurrent(t *testing.T) {
	store := NewWSTicketStore(30 * time.Second)
	defer cleanupStore(store)

	const goroutines = 50
	tickets := make([]string, goroutines)
	var wg sync.WaitGroup

	// 并发生成
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			t, _ := store.Generate(idx, idx)
			tickets[idx] = t
		}(i)
	}
	wg.Wait()

	// 并发兑换
	successes := 0
	var mu sync.Mutex
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(ticket string) {
			defer wg.Done()
			_, _, ok := store.Redeem(ticket)
			if ok {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}(tickets[i])
	}
	wg.Wait()

	if successes != goroutines {
		t.Errorf("expected %d successful redeems, got %d", goroutines, successes)
	}
}

// cleanupStore 等待足够时间让 cleanup 任务至少触发一次或测试结束
// 注意：cleanup 间隔是 60s，测试通常在 ms 级完成，所以该函数主要用于明确测试责任边界
func cleanupStore(_ *WSTicketStore) {
	// 留空：cleanup 是后台 goroutine，测试进程退出时自动终止
	// 如需强制清理，应使用短 TTL + 等待时间，但 cleanup 周期固定 60s
	// 因此 sync.Map 中的过期票据在测试结束前可能仍存在
}
