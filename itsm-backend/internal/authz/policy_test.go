package authz

import (
	"context"
	"strings"
	"testing"
)

// AI 默认零权限是整条架构的安全底座：任何派生/兼容解析路径都不能
// 把 agent 悄悄放进 AllowedPrincipals。
func TestDerivePolicy_AIHasZeroPermissionsByDefault(t *testing.T) {
	for _, action := range []string{"read", "write", "delete", "approve", "provision", "unknown-action"} {
		p := DerivePolicy("ticket", action)
		if p.Allows(PrincipalAgent) {
			t.Errorf("action=%q: 派生策略竟然允许 agent", action)
		}
		if p.Allows(PrincipalService) {
			t.Errorf("action=%q: 派生策略竟然允许 service", action)
		}
		if !p.Allows(PrincipalHuman) {
			t.Errorf("action=%q: 派生策略必须允许 human", action)
		}
		if !p.IsActive || !p.IsSystem {
			t.Errorf("action=%q: 派生策略应为 active+system", action)
		}
		if err := ValidatePolicy(p); err != nil {
			t.Errorf("action=%q: 派生策略必须自洽: %v", action, err)
		}
	}
}

func TestRiskForAction_IsRuleBasedAndFailClosed(t *testing.T) {
	cases := map[string]string{
		"read": RiskLow, "view": RiskLow, "list": RiskLow,
		"write": RiskMedium, "create": RiskMedium, "some_new_action": RiskMedium,
		"approve": RiskHigh, "rollback": RiskHigh,
		"delete": RiskCritical, "provision": RiskCritical, "override": RiskCritical,
	}
	for action, want := range cases {
		if got := RiskForAction(action); got != want {
			t.Errorf("RiskForAction(%q) = %q, want %q", action, got, want)
		}
	}
	if MinApprovalsFor(RiskLow) != 0 || MinApprovalsFor(RiskMedium) != 1 ||
		MinApprovalsFor(RiskHigh) != 2 || MinApprovalsFor(RiskCritical) != 2 {
		t.Fatal("风险分级必须映射到 0/1/2 人审批")
	}
}

// 安全不变量：给 AI 开了权限就不能同时关掉审批或降低双人要求。
func TestValidatePolicy_RejectsUnsafeAgentGrants(t *testing.T) {
	base := DerivePolicy("ticket", "delete")
	base.AllowedPrincipals = []string{PrincipalHuman, PrincipalAgent}
	base.RiskLevel = RiskCritical
	base.RequiresApproval = true
	base.MinApprovals = 2
	if err := ValidatePolicy(base); err != nil {
		t.Fatalf("合法的高危 AI 授权被拒绝: %v", err)
	}

	noApproval := base
	noApproval.RequiresApproval = false
	if err := ValidatePolicy(noApproval); err == nil {
		t.Error("允许 AI 执行 critical 操作却关闭审批，必须被拒绝")
	}

	singleApprover := base
	singleApprover.MinApprovals = 1
	if err := ValidatePolicy(singleApprover); err == nil {
		t.Error("允许 AI 执行 critical 操作却只配单人审批，必须被拒绝")
	}

	badPrincipal := base
	badPrincipal.AllowedPrincipals = []string{"root"}
	if err := ValidatePolicy(badPrincipal); err == nil {
		t.Error("未知主体类型必须被拒绝")
	}

	dup := base
	dup.AllowedPrincipals = []string{"human", "human"}
	if err := ValidatePolicy(dup); err == nil {
		t.Error("重复主体必须被拒绝")
	}

	lowWithApproval := base
	lowWithApproval.RiskLevel = RiskLow
	lowWithApproval.RequiresApproval = true
	if err := ValidatePolicy(lowWithApproval); err == nil {
		t.Error("低风险强制审批应被拒绝")
	}
}

func TestEnvelopeRoundTrip(t *testing.T) {
	p := DerivePolicy("service_request", "provision")
	p.AllowedPrincipals = []string{PrincipalHuman, PrincipalAgent}
	p.DisplayName = "执行服务交付"
	raw, err := EncodeDefinition("触发真实资源交付", p)
	if err != nil {
		t.Fatal(err)
	}
	if !IsDefinitionEnvelope(raw) {
		t.Fatal("编码结果必须被识别为 v1 信封")
	}
	desc, got, err := DecodeDefinition(raw, "service_request", "provision")
	if err != nil {
		t.Fatal(err)
	}
	if desc != "触发真实资源交付" || got.DisplayName != "执行服务交付" ||
		!got.Allows(PrincipalAgent) || got.MinApprovals != 2 {
		t.Fatalf("信封往返丢失数据: desc=%q policy=%+v", desc, got)
	}
}

// 存量库里的纯文本描述必须平滑升级成「描述 + 派生默认」，
// 且升级后 AI 仍然是零权限（不能因为兼容而放宽）。
func TestDecodeDefinition_LegacyPlainTextStaysAIDenied(t *testing.T) {
	for _, raw := range []string{"", "允许对 ticket 执行 write 操作", "  plain text  "} {
		desc, p, err := DecodeDefinition(raw, "ticket", "write")
		if err != nil {
			t.Fatalf("raw=%q: %v", raw, err)
		}
		if desc != strings.TrimSpace(raw) {
			t.Errorf("raw=%q: desc=%q", raw, desc)
		}
		if p.Allows(PrincipalAgent) {
			t.Errorf("raw=%q: 兼容解析让 agent 拿到了权限", raw)
		}
		if IsDefinitionEnvelope(raw) {
			t.Errorf("raw=%q: 纯文本被误判为信封", raw)
		}
	}
}

func TestDecodeDefinition_CorruptEnvelopeFailsClosed(t *testing.T) {
	for _, raw := range []string{
		`{"v":999,"policy":{}}`,
		`{"v":1,"policy":{"riskLevel":"yolo"}}`,
		`{"v":1, broken`,
	} {
		if _, _, err := DecodeDefinition(raw, "ticket", "write"); err == nil {
			t.Errorf("raw=%s: 损坏信封必须报错", raw)
		}
		if IsDefinitionEnvelope(raw) {
			t.Errorf("raw=%s: 损坏信封不得被当成已配置", raw)
		}
	}
}

func TestPrincipalFromContext(t *testing.T) {
	if got := PrincipalFromContext(context.Background()); got != PrincipalHuman {
		t.Fatalf("缺省主体应为 human，得到 %q", got)
	}
	if got := PrincipalFromContext(nil); got != PrincipalHuman {
		t.Fatalf("nil ctx 应为 human，得到 %q", got)
	}
	ctx := WithPrincipal(context.Background(), PrincipalAgent)
	if got := PrincipalFromContext(ctx); got != PrincipalAgent {
		t.Fatalf("WithPrincipal 未生效，得到 %q", got)
	}
}
