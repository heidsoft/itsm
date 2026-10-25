package dto

import (
	"encoding/json"
	"testing"
	"time"

	"itsm-backend/ent"
)

func TestToCIResponse_NilInput(t *testing.T) {
	if got := ToCIResponse(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %+v", got)
	}
}

func TestToCIResponse_LifecycleFieldsMapped(t *testing.T) {
	effective := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	expire := time.Date(2027, 10, 6, 0, 0, 0, 0, time.UTC)

	ci := &ent.ConfigurationItem{
		ID:              1,
		Name:            "test-ci",
		Description:     "测试描述",
		Status:          "active",
		LifecycleStatus: "online",
		EffectiveAt:     effective,
		ExpireAt:        expire,
		Version:         2,
		TenantID:        1,
	}

	res := ToCIResponse(ci)
	if res == nil {
		t.Fatal("expected non-nil response")
	}
	if res.LifecycleStatus != "online" {
		t.Errorf("LifecycleStatus = %q, want %q", res.LifecycleStatus, "online")
	}
	if res.Description != "测试描述" {
		t.Errorf("Description = %q, want %q", res.Description, "测试描述")
	}
	if res.EffectiveAt == nil {
		t.Fatal("EffectiveAt should not be nil")
	}
	if !res.EffectiveAt.Equal(effective) {
		t.Errorf("EffectiveAt = %v, want %v", *res.EffectiveAt, effective)
	}
	if res.ExpireAt == nil {
		t.Fatal("ExpireAt should not be nil")
	}
	if !res.ExpireAt.Equal(expire) {
		t.Errorf("ExpireAt = %v, want %v", *res.ExpireAt, expire)
	}
}

func TestToCIResponse_ZeroTimeLeavesNilPointers(t *testing.T) {
	ci := &ent.ConfigurationItem{
		ID:              2,
		Name:            "no-dates-ci",
		Status:          "active",
		LifecycleStatus: "draft",
	}

	res := ToCIResponse(ci)
	if res == nil {
		t.Fatal("expected non-nil response")
	}
	if res.LifecycleStatus != "draft" {
		t.Errorf("LifecycleStatus = %q, want %q", res.LifecycleStatus, "draft")
	}
	if res.EffectiveAt != nil {
		t.Errorf("EffectiveAt should be nil for zero time, got %v", *res.EffectiveAt)
	}
	if res.ExpireAt != nil {
		t.Errorf("ExpireAt should be nil for zero time, got %v", *res.ExpireAt)
	}
}

// 回归：CIResponse.LastDiscovered 曾是 time.Time + omitempty ——
// encoding/json 不会把结构体零值当作 empty，于是「从未被发现」的 CI 会下发
// 0001-01-01T00:00:00Z，调用方无法区分「没发现过」和「真的在那天发现过」。
func TestToCIResponse_LastDiscoveredOmittedWhenZero(t *testing.T) {
	ci := &ent.ConfigurationItem{ID: 3, Name: "never-discovered", Status: "active"}

	raw, err := json.Marshal(ToCIResponse(ci))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, exists := payload["lastDiscovered"]; exists {
		t.Fatalf("zero lastDiscovered must be omitted, got %v (raw=%s)", payload["lastDiscovered"], raw)
	}
	if res := ToCIResponse(ci); res.LastDiscovered != nil {
		t.Fatalf("LastDiscovered should be nil for zero time, got %v", *res.LastDiscovered)
	}
}

func TestToCIResponse_LastDiscoveredKeptWhenSet(t *testing.T) {
	discovered := time.Date(2026, 10, 20, 8, 30, 0, 0, time.UTC)
	ci := &ent.ConfigurationItem{ID: 4, Name: "discovered", Status: "active", LastDiscovered: discovered}

	res := ToCIResponse(ci)
	if res.LastDiscovered == nil {
		t.Fatal("LastDiscovered should not be nil when set")
	}
	if !res.LastDiscovered.Equal(discovered) {
		t.Fatalf("LastDiscovered = %v, want %v", *res.LastDiscovered, discovered)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got, exists := payload["lastDiscovered"]; !exists || got != discovered.Format(time.RFC3339) {
		t.Fatalf("lastDiscovered = %v (exists=%v), want %s", got, exists, discovered.Format(time.RFC3339))
	}
}
