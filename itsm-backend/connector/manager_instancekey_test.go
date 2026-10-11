package connector

import "testing"

// TestInstanceKeyUniqueness 验证 instanceKey 由 tenant + name + provider 共同决定：
// 同名但 provider 不同的配置必须产生不同的 key。
func TestInstanceKeyUniqueness(t *testing.T) {
	k1 := instanceKey(Config{TenantID: 1, Name: "feishu", Provider: "feishu"})
	k2 := instanceKey(Config{TenantID: 1, Name: "feishu", Provider: "dingtalk"})

	if k1 == k2 {
		t.Errorf("same name with different providers must not collide. k1=%q k2=%q", k1, k2)
	}

	// Same tenant + name + provider should be the same
	k3 := instanceKey(Config{TenantID: 1, Name: "feishu", Provider: "feishu"})
	if k1 != k3 {
		t.Errorf("identical configs should produce same key, got %q vs %q", k1, k3)
	}

	// Different tenants should be different
	k4 := instanceKey(Config{TenantID: 2, Name: "feishu", Provider: "feishu"})
	if k1 == k4 {
		t.Errorf("different tenants should produce different keys, got same: %q", k1)
	}
}
