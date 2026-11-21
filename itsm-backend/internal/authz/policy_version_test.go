package authz_test

// 策略版本（审计回放）的契约测试（2026-11-21）。
//
// 这组测试锁的不是「有个字段能存字符串」，而是三件事：
//  1. 同一套放行判据恒得同一指纹（可重算 = 可回放）；
//  2. 影响鉴权的任一字段变化必然改变指纹（否则版本号形同虚设）；
//  3. 只影响展示的字段不改变指纹（否则管理员改个颜色就让所有审计记录
//     「看起来策略变了」，版本失去区分力）。
//
// 反向验证纪律：把 PolicyVersion 改成 return "" 或去掉某个参与字段，
// 对应用例必须红在断言上（不是编译错误）。

import (
	"context"
	"testing"

	"itsm-backend/internal/authz"
)

func TestPolicyVersionIsStableForIdenticalDecidingInputs(t *testing.T) {
	p := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalHuman, authz.PrincipalAgent},
		RequiresApproval:  true,
		MinApprovals:      1,
		Category:          authz.CategoryServiceOps,
		IsActive:          true,
		IsSystem:          true,
	}
	first := p.PolicyVersion()
	second := p.PolicyVersion()
	if first == "" {
		t.Fatal("策略指纹为空：审计无法回答「依据哪版策略放行」")
	}
	if first != second {
		t.Fatalf("同一策略两次取指纹不一致: %q != %q", first, second)
	}
	if len(first) != 32 {
		t.Fatalf("指纹应为 32 位十六进制（sha256 前 16 字节），实际 %d 位: %q", len(first), first)
	}
}

func TestPolicyVersionIgnoresPrincipalOrder(t *testing.T) {
	// 语义相同的策略不得因数组顺序不同而被算成两个版本：
	// AllowedPrincipals 是集合语义，DB 里的顺序不构成放行判据差异。
	a := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalHuman, authz.PrincipalAgent},
		RequiresApproval:  true, MinApprovals: 1,
		Category: authz.CategoryServiceOps, IsActive: true,
	}
	b := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalAgent, authz.PrincipalHuman},
		RequiresApproval:  true, MinApprovals: 1,
		Category: authz.CategoryServiceOps, IsActive: true,
	}
	if a.PolicyVersion() != b.PolicyVersion() {
		t.Fatal("仅主体数组顺序不同却得到不同指纹：版本会把「等价策略」误报为变更")
	}
}

func TestPolicyVersionChangesOnEveryDecidingField(t *testing.T) {
	base := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalHuman},
		RequiresApproval:  true,
		MinApprovals:      1,
		Category:          authz.CategoryServiceOps,
		IsActive:          true,
	}
	baseline := base.PolicyVersion()

	cases := []struct {
		name   string
		mutate func(p *authz.PermissionPolicy)
	}{
		{"风险等级变化", func(p *authz.PermissionPolicy) { p.RiskLevel = authz.RiskHigh }},
		{"允许主体增加", func(p *authz.PermissionPolicy) {
			p.AllowedPrincipals = []string{authz.PrincipalHuman, authz.PrincipalAgent}
		}},
		{"审批开关关闭", func(p *authz.PermissionPolicy) { p.RequiresApproval = false }},
		{"审批人数变化", func(p *authz.PermissionPolicy) { p.MinApprovals = 2 }},
		{"停用策略", func(p *authz.PermissionPolicy) { p.IsActive = false }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := base
			p.AllowedPrincipals = append([]string(nil), base.AllowedPrincipals...)
			c.mutate(&p)
			if got := p.PolicyVersion(); got == baseline {
				t.Fatalf("%s 后指纹未变（%q）：该字段影响鉴权结果，必须让版本号变",
					c.name, got)
			}
		})
	}
}

func TestPolicyVersionIgnoresPresentationFields(t *testing.T) {
	// 展示字段不属于「放行判据」。若它们参与指纹，管理员改个图标或排序
	// 就会让此后所有审计记录的版本号都对不上，回放失去意义。
	base := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalHuman},
		RequiresApproval:  true,
		MinApprovals:      1,
		Category:          authz.CategoryServiceOps,
		IsActive:          true,
	}
	baseline := base.PolicyVersion()

	cosmetic := base
	cosmetic.DisplayName = "改个显示名"
	cosmetic.Icon = "new-icon"
	cosmetic.Color = "#ff0000"
	cosmetic.I18nKey = "permission.ticket.write"
	cosmetic.SortOrder = 99
	if got := cosmetic.PolicyVersion(); got != baseline {
		t.Fatalf("仅改展示字段却改变了策略指纹（%q != %q）：政策版本会被展示调整污染",
			got, baseline)
	}
}

func TestDefinitionPolicyVersionNeverEmpty(t *testing.T) {
	// 读面契约：Definition.PolicyVersion() 永远不返回空串。
	// 空串会让审计把「有策略但因故没取到」误读成「无策略参与判定」
	// （human 主体的正常语义），两种情况的处置完全不同。
	if got := (authz.Definition{}).PolicyVersion(); got != authz.PolicyVersionUnknown {
		t.Fatalf("空定义应返回 %q，实际 %q", authz.PolicyVersionUnknown, got)
	}

	def := authz.Definition{
		Code:     "ticket:write",
		Resource: "ticket",
		Action:   "write",
		Policy: authz.PermissionPolicy{
			RiskLevel:         authz.RiskMedium,
			AllowedPrincipals: []string{authz.PrincipalHuman, authz.PrincipalAgent},
			RequiresApproval:  true,
			MinApprovals:      1,
			Category:          authz.CategoryServiceOps,
			IsActive:          true,
		},
	}
	if got := def.PolicyVersion(); got == "" || got == authz.PolicyVersionUnknown {
		t.Fatalf("有效定义应返回真实指纹，实际 %q", got)
	}
	if def.PolicyVersion() != def.Policy.PolicyVersion() {
		t.Fatal("Definition.PolicyVersion 与 Policy.PolicyVersion 必须同源，否则两处读面口径分叉")
	}
}

// TestPolicyVersionDetectsReplayDrift 是「回放」这个承诺的最小验证：
// 用当时记录的指纹与当前 DB 策略重算的指纹比对，相等即证明判据未变，
// 不等即证明策略在此期间被改过。这是能回答「依据哪版策略放行」的前提。
func TestPolicyVersionDetectsReplayDrift(t *testing.T) {
	atDecisionTime := authz.PermissionPolicy{
		RiskLevel:         authz.RiskMedium,
		AllowedPrincipals: []string{authz.PrincipalHuman, authz.PrincipalAgent},
		RequiresApproval:  true,
		MinApprovals:      1,
		Category:          authz.CategoryServiceOps,
		IsActive:          true,
	}
	recorded := atDecisionTime.PolicyVersion()

	// 场景 1：策略未变 → 重算必须相等（回放成立）。
	if recomputed := atDecisionTime.PolicyVersion(); recomputed != recorded {
		t.Fatal("策略未变却重算出不同指纹：回放不可用")
	}

	// 场景 2：管理员事后把审批人数从 1 提到 2 → 必须不等（漂移可见）。
	afterChange := atDecisionTime
	afterChange.MinApprovals = 2
	if recomputed := afterChange.PolicyVersion(); recomputed == recorded {
		t.Fatal("审批强度已变却得同一指纹：策略漂移无法被发现")
	}
}

func TestEnvelopeRoundTripPreservesPolicyVersion(t *testing.T) {
	// 指纹必须能经「编码进 DB → 解码回来」这一趟后保持稳定。
	// 若编解码会丢字段或改顺序，记录的版本与实际放行依据就对不上。
	original := authz.PermissionPolicy{
		RiskLevel:         authz.RiskHigh,
		AllowedPrincipals: []string{authz.PrincipalHuman, authz.PrincipalAgent},
		RequiresApproval:  true,
		MinApprovals:      2,
		Category:          authz.CategoryAssets,
		IsActive:          true,
	}
	encoded, err := authz.EncodeDefinition("变更审批", original)
	if err != nil {
		t.Fatalf("编码策略失败: %v", err)
	}
	_, decoded, err := authz.DecodeDefinition(encoded, "change", "approve")
	if err != nil {
		t.Fatalf("解码策略失败: %v", err)
	}
	if got, want := decoded.PolicyVersion(), original.PolicyVersion(); got != want {
		t.Fatalf("编解码往返后指纹变化: %q != %q", got, want)
	}
}

// 编译期断言：Catalog 接口在加方法时会牵连实现方；本批刻意不改该接口
// （版本号由 Definition 携带，而不是 AuthorizePrincipal 多一个返回值——
// 后者会让所有实现与调用点连锁改动，且四条执行通道共用一个三元组契约）。
var _ authz.Catalog = authz.NewRegistry(nil)

func TestAuthorizePrincipalStillDeniesWithoutCatalog(t *testing.T) {
	// 未装配 client 的注册表必须 fail-closed，且不因本批改动而放宽。
	r := authz.NewRegistry(nil)
	if _, _, err := r.AuthorizePrincipal(context.Background(), 1, "ticket:write", authz.PrincipalAgent); err == nil {
		t.Fatal("未装配目录的注册表不应放行任何主体")
	}
}
