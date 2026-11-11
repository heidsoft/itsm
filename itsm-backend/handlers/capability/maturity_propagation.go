package capability

import "sort"

// nonGADependencies 返回直接依赖中尚未达到 GA 的能力键。
//
// capability 矩阵的 maturity 只描述能力自身，registry 的 dependencies 此前
// 被原样下发，调用方看到 "ga" 会以为整条链路都验收过：例如 incident /
// problem / change 标 GA，却跑在仍是 Pilot 的 workflow 编排层上。这里把依赖
// 成熟度显式传导出来，只标注不改写本能力自身的 maturity（后者是 README 与
// 商业契约共同引用的口径）。pilot / disabled 与「registry 里查不到的键」一律
// 视为风险——未知依赖不能默认安全，fail-closed；结果去重排序保证响应稳定。
func nonGADependencies(dependencies []string, defs []definition) []string {
	maturityByKey := make(map[string]Maturity, len(defs))
	for _, def := range defs {
		maturityByKey[def.key] = def.maturity
	}

	seen := make(map[string]struct{}, len(dependencies))
	risks := make([]string, 0, len(dependencies))
	for _, key := range dependencies {
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		maturity, ok := maturityByKey[key]
		if !ok || maturity != MaturityGA {
			risks = append(risks, key)
		}
	}
	sort.Strings(risks)
	return risks
}
