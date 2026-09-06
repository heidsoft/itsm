package dto

import (
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/schema"
)

// TestToCIRelationshipResponse_FillsTypeName P1-5（2026-09-06 UAT 修复）：
// CIRelationshipResponse.RelationshipTypeName 必须从 schema.CIRelationshipTypeVocabulary
// 填充中文展示名（"depends_on" → "依赖"）。之前永远空字符串，导致前端
// CMDB 关系列表"关系类型"列渲染空白。
func TestToCIRelationshipResponse_FillsTypeName(t *testing.T) {
	for _, meta := range schema.CIRelationshipTypeVocabulary {
		rel := &ent.CIRelationship{
			ID:              1,
			RelationshipType: string(meta.Type),
		}
		res := ToCIRelationshipResponse(rel)
		if res == nil {
			t.Fatalf("ToCIRelationshipResponse returned nil for type=%s", meta.Type)
		}
		if res.RelationshipTypeName != meta.Name {
			t.Errorf("type=%s expected name=%q got %q", meta.Type, meta.Name, res.RelationshipTypeName)
		}
	}
}

// TestToCIRelationshipResponse_UnknownTypeGraceful：未知关系类型不填充但 nil 安全。
func TestToCIRelationshipResponse_UnknownTypeGraceful(t *testing.T) {
	rel := &ent.CIRelationship{
		ID:              99,
		RelationshipType: "deprecated_legacy_type",
	}
	res := ToCIRelationshipResponse(rel)
	if res == nil {
		t.Fatal("expected non-nil response even for unknown type")
	}
	if res.RelationshipTypeName != "" {
		t.Errorf("unknown type should leave name empty, got %q", res.RelationshipTypeName)
	}
}