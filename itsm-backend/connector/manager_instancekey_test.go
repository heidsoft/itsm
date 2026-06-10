package connector

import "testing"

// TestInstanceKeyUniqueness verifies Bug #4 fix: same name with different
// providers must produce different keys (was previously producing same key).
func TestInstanceKeyUniqueness(t *testing.T) {
	k1 := instanceKey(Config{TenantID: 1, Name: "feishu", Provider: "feishu"})
	k2 := instanceKey(Config{TenantID: 1, Name: "feishu", Provider: "dingtalk"})

	if k1 == k2 {
		t.Errorf("Bug #4 not fixed: same name with different providers collided. k1=%q k2=%q", k1, k2)
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
