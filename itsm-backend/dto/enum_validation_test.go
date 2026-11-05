package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 回归：change / release / CMDB / problem 的枚举字段长期只有注释词表，
// 非法值（"urgent"、"bogus"）会 200 落库，或落到 service 变成 500。
// 这里用与 gin 相同的 binding tag 名做真实校验，防止 tag 写了却不生效。
func newBindingValidator() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}

func requireValid(t *testing.T, v *validator.Validate, s interface{}) {
	t.Helper()
	require.NoError(t, v.Struct(s), "合法请求不得被枚举校验拒绝")
}

func requireInvalid(t *testing.T, v *validator.Validate, s interface{}, why string) {
	t.Helper()
	assert.Error(t, v.Struct(s), "%s：非法枚举应被 binding 拒绝", why)
}

func TestEnumValidation_Change(t *testing.T) {
	v := newBindingValidator()
	requireValid(t, v, &CreateChangeRequest{Title: "变更"})
	requireValid(t, v, &CreateChangeRequest{
		Title: "变更", Type: "normal", Priority: "high", ImpactScope: "medium", RiskLevel: "low",
	})
	requireInvalid(t, v, &CreateChangeRequest{Title: "变更", Type: "not_a_type"}, "change.type")
	requireInvalid(t, v, &CreateChangeRequest{Title: "变更", Priority: "urgent"}, "change.priority")
	requireInvalid(t, v, &CreateChangeRequest{Title: "变更", ImpactScope: "urgent"}, "change.impactScope")
	requireInvalid(t, v, &CreateChangeRequest{Title: "变更", RiskLevel: "bogus"}, "change.riskLevel")

	badType := ChangeType("bogus")
	badPriority := ChangePriority("urgent")
	requireInvalid(t, v, &UpdateChangeRequest{Type: &badType}, "change update.type 指针")
	requireInvalid(t, v, &UpdateChangeRequest{Priority: &badPriority}, "change update.priority 指针")
	goodType := ChangeTypeEmergency
	requireValid(t, v, &UpdateChangeRequest{Type: &goodType})
	requireValid(t, v, &UpdateChangeRequest{})
}

func TestEnumValidation_Release(t *testing.T) {
	v := newBindingValidator()
	requireValid(t, v, &CreateReleaseRequest{ReleaseNumber: "REL-1", Title: "发布"})
	requireValid(t, v, &CreateReleaseRequest{
		ReleaseNumber: "REL-1", Title: "发布",
		Type: "hotfix", Environment: "production", Severity: "critical",
	})
	requireInvalid(t, v, &CreateReleaseRequest{ReleaseNumber: "R", Title: "T", Type: "bogus"}, "release.type")
	requireInvalid(t, v, &CreateReleaseRequest{ReleaseNumber: "R", Title: "T", Environment: "bogus"}, "release.environment")
	requireInvalid(t, v, &CreateReleaseRequest{ReleaseNumber: "R", Title: "T", Severity: "urgent"}, "release.severity")

	bad := "bogus"
	requireInvalid(t, v, &UpdateReleaseRequest{Type: &bad}, "release update.type 指针")
	good := "patch"
	requireValid(t, v, &UpdateReleaseRequest{Type: &good})
}

func TestEnumValidation_CI(t *testing.T) {
	v := newBindingValidator()
	requireValid(t, v, &CreateCIRequest{Name: "CI", Status: "active"})
	// 前端 constants/cmdb.ts 仍使用 decommissioned，后端必须兼容存量词表。
	requireValid(t, v, &CreateCIRequest{Name: "CI", Status: "decommissioned", Criticality: "critical"})
	requireInvalid(t, v, &CreateCIRequest{Name: "CI", Status: "bogus"}, "ci.status")
	requireInvalid(t, v, &CreateCIRequest{Name: "CI", Status: "active", Criticality: "bogus"}, "ci.criticality")
	requireInvalid(t, v, &UpdateCIRequest{Status: "bogus"}, "ci update.status")
	requireInvalid(t, v, &UpdateCIRequest{Criticality: "bogus"}, "ci update.criticality")
	requireValid(t, v, &UpdateCIRequest{})
}

func TestEnumValidation_Problem(t *testing.T) {
	v := newBindingValidator()
	base := CreateProblemRequest{Title: "问题", Description: "这是一段足够长的问题描述内容用于通过最小长度校验", Priority: "critical"}
	requireValid(t, v, &base)
	urgent := base
	urgent.Priority = "urgent"
	requireInvalid(t, v, &urgent, "problem.priority（urgent 属于工单词表）")

	badPriority := "urgent"
	requireInvalid(t, v, &UpdateProblemRequest{Priority: &badPriority}, "problem update.priority 指针")
	goodPriority := "high"
	requireValid(t, v, &UpdateProblemRequest{Priority: &goodPriority})
}
