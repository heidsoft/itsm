package cmdb

import (
	"sort"
	"strings"
	"time"

	"itsm-backend/common"
)

// CMDB 属性级数据质量：对齐 ServiceNow CMDB Health 的 completeness / duplicate
// 两个维度，替换 data_governance.go 里那套「owner 覆盖率」口径。
//
// 为什么必须换：旧口径 Incomplete 判据是 `DiscoverySource=="" && Source=="" &&
// OwnedBy==""`——它算的是「有没有登记来源或负责人」，与属性填了多少毫无关系。
// 一个 owner 齐全但 serial/vendor/型号全空的 CI 会被判为 complete；而任何必填
// 属性定义（企业实际用来管数据质量的东西）都不参与计算。前端 reports/cmdb-quality
// 又另行硬编码 9 个字段在前端自算，两套口径互不相通，同一个租户在两处看到两个
// 数字。
//
// 本文件只做纯计算：CI 集合 × 必填属性定义 → 指标。必填定义由调用方按 ciTypeID
// 查好传入（事实来源仍是 CIAttributeValidator.RequiredAttributes），这里不查库、
// 不依赖 ent，可独立单测。

// RequiredAttributeSpec 是某 CI 类型当前生效的必填属性名集合。
// 之所以按 ciTypeID 分桶传入而不是逐 CI 查一次：类型数量远小于 CI 数量，
// 一次解析整租户的类型集，避免 N+1 查询。
type RequiredAttributeSpec struct {
	// CITypeID 对应的 CI 类型 ID；0 表示「不属于任何已知类型」。
	CITypeID int
	// Names 是该类型（含继承链）全部必填属性的字段名。
	Names []string
}

// AttributeQualityMetrics 是属性级完整率结果。
type AttributeQualityMetrics struct {
	Total   int
	Active  int
	Retired int
	Stale   int
	Orphan  int
	// Incomplete 是必填属性未齐的 CI 数。
	Incomplete int
	// CompletenessPct 只在 Assessable 内有意义。
	CompletenessPct float64
	// Assessable 是「其 CI 类型存在必填属性定义」的 CI 数（完整率的分母）。
	Assessable int
	// AssessedTypes 是评估中用到必填定义的 CI 类型数。
	AssessedTypes int
	// MissingByAttribute 汇总「哪些必填属性最缺」，按缺失次数降序。
	// 这是完整率唯一可行动的产出——只说「完整率 62%」不能告诉运维该补什么。
	MissingByAttribute []AttributeMissingStat
}

// AttributeMissingStat 是单个必填属性在整租户内的缺失统计。
type AttributeMissingStat struct {
	CITypeID   int    `json:"ciTypeId"`
	Attribute  string `json:"attribute"`
	Missing    int    `json:"missing"`
	Applicable int    `json:"applicable"`
}

// attributeIsBlank 判定属性值是否可视为「未填」。
//
// 与 internal/services 的 isBlankAttributeString 保持同一语义（空白串算缺），
// 但额外覆盖 nil、空切片/空 map：JSON 属性列里 `""`、`null`、`[]`、`{}` 都
// 表示「没填」，只判 `""` 会让「清空过的数组字段」被算成已填。
func attributeIsBlank(value interface{}, exists bool) bool {
	if !exists || value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	switch typed := value.(type) {
	case []interface{}:
		return len(typed) == 0
	case map[string]interface{}:
		return len(typed) == 0
	}
	return false
}

// attributeValuePresent 取出属性值并判定非空。返回值便于单测直接断言。
func attributeValuePresent(attributes map[string]interface{}, name string) bool {
	if attributes == nil {
		return false
	}
	value, exists := attributes[name]
	return !attributeIsBlank(value, exists)
}

// ComputeAttributeQuality 按「该 CI 类型的必填属性定义」计算完整率与其它治理指标。
//
// specs 的键是 ciTypeID，值是该类型的必填属性名列表。未在 specs 里出现的类型
// （或 specs 为空）一律不计入 Assessable：无法评估就不该产生百分比。这一点是
// 本函数与旧口径最关键的差别——旧实现把「没有定义必需的字段」隐式算成已填，
// 于是全新租户（属性定义表为空）直接呈现 100% 完整率。
//
// threshold / now 只作用于 IsStaleCI；IsOrphanCI 的语义不变（无 owner 且无来源），
// 它与完整率是两个不同维度：orphan 是「没人负责」，incomplete 是「字段不齐」。
func ComputeAttributeQuality(
	cis []*ConfigurationItem,
	specs map[int][]string,
	threshold time.Duration,
	now time.Time,
) AttributeQualityMetrics {
	m := AttributeQualityMetrics{Total: len(cis)}
	if len(cis) == 0 {
		return m
	}

	missingCounts := map[int]map[string]int{}
	assessedTypes := map[int]struct{}{}

	for _, ci := range cis {
		if ci == nil {
			continue
		}
		if ci.LifecycleStatus == common.CILifecycleStatusScrapped {
			m.Retired++
		} else {
			m.Active++
		}
		if IsStaleCI(ci, threshold, now) {
			m.Stale++
		}
		if IsOrphanCI(ci) {
			m.Orphan++
		}

		requiredNames, hasSpec := specs[ci.CITypeID]
		if !hasSpec || len(requiredNames) == 0 {
			continue
		}

		m.Assessable++
		assessedTypes[ci.CITypeID] = struct{}{}
		if missingCounts[ci.CITypeID] == nil {
			missingCounts[ci.CITypeID] = map[string]int{}
		}
		typeMissing := 0
		for _, name := range requiredNames {
			if attributeValuePresent(ci.Attributes, name) {
				continue
			}
			typeMissing++
			missingCounts[ci.CITypeID][name]++
		}
		if typeMissing > 0 {
			m.Incomplete++
		}
	}

	m.AssessedTypes = len(assessedTypes)
	if m.Assessable > 0 {
		complete := m.Assessable - m.Incomplete
		ratio := float64(complete) / float64(m.Assessable) * 100
		m.CompletenessPct = roundToTwoDecimals(ratio)
	}

	for ciTypeID, byAttr := range missingCounts {
		names := make([]string, 0, len(byAttr))
		for name := range byAttr {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			m.MissingByAttribute = append(m.MissingByAttribute, AttributeMissingStat{
				CITypeID:   ciTypeID,
				Attribute:  name,
				Missing:    byAttr[name],
				Applicable: countCIsOfType(cis, ciTypeID),
			})
		}
	}
	sort.Slice(m.MissingByAttribute, func(i, j int) bool {
		a, b := m.MissingByAttribute[i], m.MissingByAttribute[j]
		if a.Missing != b.Missing {
			return a.Missing > b.Missing
		}
		if a.CITypeID != b.CITypeID {
			return a.CITypeID < b.CITypeID
		}
		return a.Attribute < b.Attribute
	})
	return m
}

// countCIsOfType 数该类型下的 CI 条数（含退役），作为该属性缺失率的分母。
func countCIsOfType(cis []*ConfigurationItem, ciTypeID int) int {
	n := 0
	for _, ci := range cis {
		if ci != nil && ci.CITypeID == ciTypeID {
			n++
		}
	}
	return n
}

// roundToTwoDecimals 与旧引擎同一取整口径（四舍五入到 0.01）。
func roundToTwoDecimals(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// DuplicateReason 是判重键的语义来源。
type DuplicateReason string

const (
	// DuplicateReasonCloudIdentity 云上身份相同（同 provider 同资源 ID）。
	DuplicateReasonCloudIdentity DuplicateReason = "cloud_identity"
	// DuplicateReasonAssetTag 资产标签相同（asset_tag 无唯一约束，可重复写入）。
	DuplicateReasonAssetTag DuplicateReason = "asset_tag"
	// DuplicateReasonSerialName 序列号 + 归一化名称相同。serial_number 本身有
	// (tenant_id, serial_number) 唯一索引，单独不会重复；但序列号缺失时多人
	// 各建一条同主机名的 CI 是真实场景，因此键里必须带名称。
	DuplicateReasonSerialName DuplicateReason = "serial_name"
)

// DuplicateGroup 一组判为同实体的 CI。
type DuplicateGroup struct {
	Key       string
	Reason    DuplicateReason
	CIIDs     []int
	CINumbers []string
}

// duplicateIdentity 描述一条 CI 参与判重所需的身份字段。
// 单独抽出是为了让单测能只测归一化、不构造完整 ConfigurationItem。
type duplicateIdentity struct {
	ID              int
	CINumber        string
	Name            string
	AssetTag        string
	SerialNumber    string
	CloudProvider   string
	CloudResourceID string
	Model           string
	LifecycleStatus string
}

func normalizeKeyPart(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// identityOfCI 抽出参与判重的字段。退役 CI 不参与判重：它已经从在用账里退出，
// 把它算成「重复」会让退役动作本身变成数据质量告警。
func identityOfCI(ci *ConfigurationItem) (duplicateIdentity, bool) {
	if ci == nil {
		return duplicateIdentity{}, false
	}
	if ci.LifecycleStatus == common.CILifecycleStatusScrapped {
		return duplicateIdentity{}, false
	}
	return duplicateIdentity{
		ID:              ci.ID,
		CINumber:        ci.CINumber,
		Name:            normalizeKeyPart(ci.Name),
		AssetTag:        normalizeKeyPart(ci.AssetTag),
		SerialNumber:    normalizeKeyPart(ci.SerialNumber),
		CloudProvider:   normalizeKeyPart(ci.CloudProvider),
		CloudResourceID: normalizeKeyPart(ci.CloudResourceID),
		Model:           normalizeKeyPart(ci.Model),
		LifecycleStatus: ci.LifecycleStatus,
	}, true
}

// DetectDuplicateGroups 对 CI 集合做身份键聚类，返回重复组（组大小 ≥2）。
//
// 判重键按优先级取第一个可用的，避免同一对 CI 被三条规则各报一次：
//  1. cloud_identity：provider + 云上资源 ID 都非空——这是最强证据（同一条云资源）；
//  2. asset_tag：资产标签非空——物理资产的粘贴标签，同一标签不可能贴两台设备；
//  3. serial_name：序列号 + 归一化名称都非空——兜住「序列号相同但名称写法不同」
//     与「序列号缺失时同名重复建账」两类；
//
// 三者都不满足的 CI 不参与判重（无身份信息，任何聚类都是猜）。
//
// 组内按 CI ID 升序，组间按 key 字典序，保证同一输入产出稳定结果——
// 否则前端每次刷新看到的重复组顺序都在变。
func DetectDuplicateGroups(cis []*ConfigurationItem) []DuplicateGroup {
	byKey := map[string][]duplicateIdentity{}
	reasonByKey := map[string]DuplicateReason{}
	order := make([]string, 0)

	register := func(key string, reason DuplicateReason, ident duplicateIdentity) {
		if key == "" {
			return
		}
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
			reasonByKey[key] = reason
		}
		byKey[key] = append(byKey[key], ident)
	}

	for _, ci := range cis {
		ident, ok := identityOfCI(ci)
		if !ok {
			continue
		}
		switch {
		case ident.CloudProvider != "" && ident.CloudResourceID != "":
			register("cloud:"+ident.CloudProvider+":"+ident.CloudResourceID,
				DuplicateReasonCloudIdentity, ident)
		case ident.AssetTag != "":
			register("tag:"+ident.AssetTag, DuplicateReasonAssetTag, ident)
		case ident.SerialNumber != "" && ident.Name != "":
			register("sn:"+ident.SerialNumber+":"+ident.Name,
				DuplicateReasonSerialName, ident)
		}
	}

	groups := make([]DuplicateGroup, 0, len(order))
	for _, key := range order {
		idents := byKey[key]
		if len(idents) < 2 {
			continue
		}
		sort.Slice(idents, func(i, j int) bool { return idents[i].ID < idents[j].ID })
		group := DuplicateGroup{Key: key, Reason: reasonByKey[key]}
		for _, ident := range idents {
			group.CIIDs = append(group.CIIDs, ident.ID)
			group.CINumbers = append(group.CINumbers, ident.CINumber)
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })
	return groups
}

// CountDuplicateCIs 统计参与重复组的 CI 条数（组内每条都算）。
func CountDuplicateCIs(groups []DuplicateGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.CIIDs)
	}
	return n
}

// DuplicateRatePct 计算重复率：参与重复的 CI / 在用 CI 总数 ×100。
//
// 分母用「在用 CI」（排除 scrapped）而不是 Assessable：重复问题是资产账问题，
// 与「有没有配必填属性定义」无关——手工建账但没配属性定义的租户同样会产生重复。
// total<=0 返回 0，避免除零。
func DuplicateRatePct(duplicateCIs, inUseCIs int) float64 {
	if inUseCIs <= 0 || duplicateCIs <= 0 {
		return 0
	}
	return roundToTwoDecimals(float64(duplicateCIs) / float64(inUseCIs) * 100)
}
