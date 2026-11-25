package capability

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readmeDomainStates(t *testing.T) (map[string]string, []string) {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "..", "README.md"))
	if err != nil {
		t.Fatalf("README 能力与成熟度: %v", err)
	}
	defer f.Close()
	states := make(map[string]string)
	var order []string
	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "## 能力与成熟度" {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 4 {
			continue
		}
		name, state := strings.TrimSpace(cols[1]), strings.TrimSpace(cols[2])
		if state != "可用" && state != "预览" {
			continue
		}
		if _, exists := states[name]; exists {
			t.Fatalf("README 域 %q 重复", name)
		}
		states[name] = state
		order = append(order, name)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("README 能力与成熟度解析: %v", err)
	}
	return states, order
}

// Subdomain keys belong to the independent parent for coverage only; they
// never enter the parent's maturity calculation.
func domainKeyOwners(domains []readmeDomain) map[string]string {
	owners := make(map[string]string)
	for _, d := range domains {
		owner := d.name
		if d.subdomainOf != "" {
			owner = d.subdomainOf
		}
		for _, key := range d.primaryKeys {
			owners[key] = owner
		}
	}
	return owners
}

func registryCoverageError(defs []definition, owners, support map[string]string) error {
	for _, d := range defs {
		if strings.TrimSpace(owners[d.key]) == "" && strings.TrimSpace(support[d.key]) == "" {
			return fmt.Errorf("registry key %q 未被独立域或有理由的内部支撑白名单认领", d.key)
		}
	}
	return nil
}

func TestDomainMappingCoversRegistry(t *testing.T) {
	owners := domainKeyOwners(readmeDomains)
	if len(registry) != 28 {
		t.Errorf("registry keys 数量 = %d，期望 28；请复核域映射", len(registry))
	}
	if err := registryCoverageError(registry, owners, internalSupportKeys); err != nil {
		t.Fatal(err)
	}
	known := make(map[string]bool)
	for _, d := range registry {
		if known[d.key] {
			t.Errorf("registry key %q 重复", d.key)
		}
		known[d.key] = true
	}
	seen := make(map[string]string)
	for _, d := range readmeDomains {
		for _, key := range d.primaryKeys {
			if !known[key] {
				t.Errorf("域 %q 引用未知 registry key %q", d.name, key)
			}
			if prior, ok := seen[key]; ok {
				t.Errorf("key %q 被域 %q 和 %q 重复认领", key, prior, d.name)
			}
			if _, ok := internalSupportKeys[key]; ok {
				t.Errorf("域 %q 的 key %q 同时出现在内部支撑白名单", d.name, key)
			}
			seen[key] = d.name
		}
	}
	for key := range internalSupportKeys {
		if !known[key] {
			t.Errorf("内部支撑 key %q 不存在于 registry", key)
		}
	}
	// Prove the guard rejects deletion of every claim, without mutating globals.
	for _, d := range registry {
		t.Run("missing/"+d.key, func(t *testing.T) {
			claims, support := make(map[string]string), make(map[string]string)
			for key, owner := range owners {
				claims[key] = owner
			}
			for key, reason := range internalSupportKeys {
				support[key] = reason
			}
			delete(claims, d.key)
			delete(support, d.key)
			if err := registryCoverageError(registry, claims, support); err == nil || !strings.Contains(err.Error(), d.key) {
				t.Errorf("删除 key %q 的认领后，覆盖守卫未定位该 key: %v", d.key, err)
			}
		})
	}
	if err := registryCoverageError([]definition{{key: "unknown.capability"}}, owners, internalSupportKeys); err == nil {
		t.Error("未知 key unknown.capability 未触发覆盖失败")
	}
}

func TestDomainMappingCoversReadme(t *testing.T) {
	states, order := readmeDomainStates(t)
	if len(states) != 15 || len(readmeDomains) != 15 {
		t.Errorf("README 能力域数量=%d，映射域数量=%d，均应为15；域=%v", len(states), len(readmeDomains), order)
	}
	mapped := make(map[string]readmeDomain)
	for i, d := range readmeDomains {
		if _, exists := mapped[d.name]; exists {
			t.Errorf("映射域 %q 重复", d.name)
		}
		mapped[d.name] = d
		if _, exists := states[d.name]; !exists {
			t.Errorf("映射域 %q 在 README 中缺行", d.name)
		}
		if i < len(order) && order[i] != d.name {
			t.Errorf("域 %q 顺序不符，README 第 %d 行为 %q", d.name, i+1, order[i])
		}
		if d.noIndependentKey {
			if strings.TrimSpace(d.reason) == "" || len(d.primaryKeys) != 0 {
				t.Errorf("无独立 key 域 %q 必须有理由且无 primaryKeys", d.name)
			}
		} else if len(d.primaryKeys) == 0 {
			t.Errorf("域 %q 无 primaryKeys，必须显式声明 noIndependentKey 与理由", d.name)
		}
	}
	for name := range states {
		if _, exists := mapped[name]; !exists {
			t.Errorf("README 域 %q 缺少映射", name)
		}
	}
	for _, d := range readmeDomains {
		if d.subdomainOf == "" {
			continue
		}
		parent, exists := mapped[d.subdomainOf]
		if !exists || parent.subdomainOf != "" || parent.noIndependentKey || len(parent.primaryKeys) == 0 {
			t.Errorf("子域 %q 的归属说明 %q 必须指向有 key 的独立域", d.name, d.subdomainOf)
		}
	}
}

func TestReadmeStateDerivedFromRegistry(t *testing.T) {
	states, _ := readmeDomainStates(t)
	byKey := make(map[string]definition)
	for _, d := range registry {
		byKey[d.key] = d
	}
	for _, d := range readmeDomains {
		if d.subdomainOf != "" || d.noIndependentKey {
			continue
		}
		t.Run(d.name, func(t *testing.T) {
			maturity, ok := domainMaturity(d.primaryKeys, byKey)
			if !ok {
				t.Fatalf("域 %q 的 registry keys %v 缺失或成熟度非法", d.name, d.primaryKeys)
			}
			if want := expectedReadmeState(maturity); states[d.name] != want {
				t.Errorf("域 %q keys %v registry 推导状态=%q，README 实测=%q", d.name, d.primaryKeys, want, states[d.name])
			}
		})
	}
}

func TestInternalSupportKeysAreJustified(t *testing.T) {
	if len(internalSupportKeys) != 6 {
		t.Errorf("内部支撑 keys 数量=%d，期望6；keys=%v", len(internalSupportKeys), internalSupportKeys)
	}
	for key, reason := range internalSupportKeys {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("内部支撑 key %q 缺少理由", key)
		}
		for _, snapshot := range []string{"未触碰", "未修改", "暂不修改", "not touching", "untouched", "not modified"} {
			if strings.Contains(strings.ToLower(reason), snapshot) {
				t.Errorf("内部支撑 key %q 的理由包含快照式声明 %q", key, snapshot)
			}
		}
	}
}

func TestDomainMaturity(t *testing.T) {
	byKey := map[string]definition{
		"ga": {maturity: MaturityGA}, "pilot": {maturity: MaturityPilot},
		"disabled": {maturity: MaturityDisabled}, "invalid": {maturity: Maturity("invalid")},
	}
	for _, tc := range []struct {
		name string
		keys []string
		want Maturity
		ok   bool
	}{
		{"all ga", []string{"ga"}, MaturityGA, true},
		{"pilot", []string{"ga", "pilot"}, MaturityPilot, true},
		{"disabled", []string{"disabled", "ga"}, MaturityPilot, true},
		{"missing after pilot", []string{"pilot", "missing"}, MaturityDisabled, false},
		{"invalid", []string{"invalid"}, MaturityDisabled, false},
		{"empty", nil, MaturityDisabled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domainMaturity(tc.keys, byKey)
			if got != tc.want || ok != tc.ok {
				t.Errorf("keys %v: got (%q,%v), want (%q,%v)", tc.keys, got, ok, tc.want, tc.ok)
			}
		})
	}
	for _, m := range []Maturity{MaturityGA, MaturityPilot, MaturityDisabled} {
		want := "预览"
		if m == MaturityGA {
			want = "可用"
		}
		if got := expectedReadmeState(m); got != want {
			t.Errorf("maturity %q: README state=%q, want %q", m, got, want)
		}
	}
}
