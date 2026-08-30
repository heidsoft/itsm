package middleware

import (
	"context"
	"testing"
	"time"
)

// P1-2 修复验证：用户级 token 最低签发时间（MinIssuedAt）机制。
func TestMemoryStore_MinIssuedAt(t *testing.T) {
	store := newMemoryAccessTokenRevocationStore()
	ctx := context.Background()

	// 初始无约束
	got, err := store.MinIssuedAt(ctx, 42)
	if err != nil || !got.IsZero() {
		t.Fatalf("初始应为零值约束, got %v err %v", got, err)
	}

	// 设置约束
	t1 := time.Now().Add(-time.Minute)
	if err := store.SetMinIssuedAt(ctx, 42, t1); err != nil {
		t.Fatalf("SetMinIssuedAt failed: %v", err)
	}
	got, err = store.MinIssuedAt(ctx, 42)
	if err != nil || !got.Equal(t1) {
		t.Fatalf("应读到 t1, got %v err %v", got, err)
	}

	// 只增不减：回退写入被忽略
	t0 := t1.Add(-time.Hour)
	if err := store.SetMinIssuedAt(ctx, 42, t0); err != nil {
		t.Fatalf("SetMinIssuedAt failed: %v", err)
	}
	got, _ = store.MinIssuedAt(ctx, 42)
	if !got.Equal(t1) {
		t.Fatalf("回退写入应被忽略, got %v want %v", got, t1)
	}

	// 提升生效
	t2 := t1.Add(time.Hour)
	_ = store.SetMinIssuedAt(ctx, 42, t2)
	got, _ = store.MinIssuedAt(ctx, 42)
	if !got.Equal(t2) {
		t.Fatalf("提升应生效, got %v want %v", got, t2)
	}

	// 其它用户不受影响
	other, _ := store.MinIssuedAt(ctx, 43)
	if !other.IsZero() {
		t.Fatalf("其它用户不应受影响, got %v", other)
	}
}

// P1-4 修复验证：hasResourcePermission 仅 super_admin 硬编码直通，sysadmin 必须走权限数据。
func TestHasResourcePermission_SysadminNotHardcoded(t *testing.T) {
	// nil client 且 DB 权限不可得时：super_admin 仍直通；sysadmin 不应直通。
	if !hasResourcePermission(context.Background(), nil, "super_admin", "any", "any", 1) {
		t.Fatal("super_admin 应无条件直通")
	}
	if hasResourcePermission(context.Background(), nil, "sysadmin", "any", "any", 1) {
		t.Fatal("sysadmin 不应硬编码直通（DBOnly 语义下权限收回必须可生效）")
	}
}

// InvalidateUserAccessTokens 对外语义：调用后该用户更早签发的 token 全部失效。
func TestInvalidateUserAccessTokens(t *testing.T) {
	store := newMemoryAccessTokenRevocationStore()
	setAccessTokenRevocationStore(store)
	defer setAccessTokenRevocationStore(newMemoryAccessTokenRevocationStore())

	now := time.Now()
	if err := InvalidateUserAccessTokens(context.Background(), 7, now); err != nil {
		t.Fatalf("InvalidateUserAccessTokens failed: %v", err)
	}
	minIAT, err := currentAccessTokenRevocationStore().MinIssuedAt(context.Background(), 7)
	if err != nil || minIAT.IsZero() {
		t.Fatalf("应已设置最低签发时间, got %v err %v", minIAT, err)
	}
	// 早于 minIAT 的签发时间应被判定为过期（时间比较语义由 AuthMiddleware 执行）。
	if !minIAT.Equal(now.Truncate(0)) && minIAT.Before(now.Add(-time.Second)) {
		t.Fatalf("minIAT 应约为 now, got %v", minIAT)
	}
}
