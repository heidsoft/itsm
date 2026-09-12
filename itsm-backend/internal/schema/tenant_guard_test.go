package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantExemptTables_NoDuplicates 单一真相源内部一致性：表名不能重复。
func TestTenantExemptTables_NoDuplicates(t *testing.T) {
	seen := map[string]int{}
	for _, e := range TenantExemptTables {
		seen[e.TableName]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("duplicate exempt entry: %s appears %d times", name, count)
		}
	}
}

// TestTenantExemptTables_RequiredFields 每条豁免必须写齐 reason/owner/scope + 复核日期。
func TestTenantExemptTables_RequiredFields(t *testing.T) {
	for _, e := range TenantExemptTables {
		assert.NotEmpty(t, e.TableName, "TableName required")
		assert.NotEmpty(t, e.Reason, "%s: reason required (governance)", e.TableName)
		assert.NotEmpty(t, e.Owner, "%s: owner required", e.TableName)
		assert.Contains(t, []string{"platform", "global", "derived"}, e.Scope,
			"%s: scope must be one of platform/global/derived", e.TableName)
		assert.False(t, e.ReviewedAt.IsZero(), "%s: reviewed_at required", e.TableName)
		assert.True(t, e.ReviewedAt.Before(time.Now().UTC().AddDate(0, 0, 1)),
			"%s: reviewed_at must not be in the future", e.TableName)
	}
}

// TestBuildExemptSet 与 TenantExemptTables 一一对应。
func TestBuildExemptSet(t *testing.T) {
	set := buildExemptSet()
	assert.Len(t, set, len(TenantExemptTables))
	for _, e := range TenantExemptTables {
		assert.True(t, set[e.TableName], "missing %s in exempt set", e.TableName)
	}
}

// TestResolvePolicy 验证策略解析：env 默认 + 显式覆盖。
func TestResolvePolicy(t *testing.T) {
	cases := []struct {
		env, policy string
		want        Policy
	}{
		{"production", "", PolicyFatal},
		{"private", "", PolicyFatal},
		{"saas", "", PolicyFatal},
		{"saas_msp", "", PolicyFatal},
		{"dev", "", PolicyWarn},
		{"", "", PolicyWarn}, // unset → dev default
		{"production", "warn", PolicyWarn},
		{"production", "silent", PolicySilent},
		{"production", "fatal", PolicyFatal},
		{"production", "bogus", PolicyFatal}, // invalid → fallback to env-based
	}
	for _, c := range cases {
		t.Run(c.env+"_"+c.policy, func(t *testing.T) {
			t.Setenv("ENV", c.env)
			t.Setenv("ITSM_TENANT_GUARD_POLICY", c.policy)
			assert.Equal(t, c.want, ResolvePolicy())
		})
	}
}

// TestApplyGuard_Silent 验证 silent 策略不读 DB（用于测试隔离）。
func TestApplyGuard_Silent(t *testing.T) {
	violations, err := ApplyGuard(nil, nil, nil, PolicySilent)
	require.NoError(t, err)
	assert.Nil(t, violations)
}
