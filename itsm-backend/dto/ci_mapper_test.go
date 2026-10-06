package dto

import (
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
