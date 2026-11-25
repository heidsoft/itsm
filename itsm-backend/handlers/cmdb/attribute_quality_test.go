package cmdb

import (
	"testing"
	"time"

	"itsm-backend/common"
)

// 本文件锁「属性级完整率」与「CI 间重复聚类」两个新口径。
//
// 反向验证要点（每条断言都必须能区分新旧实现）：
//   - 若把 Assesable 判据改回「所有 CI 都算分母」，无属性定义的租户会得到 100%
//     完整率 → TestComputeAttributeQuality_NoSpecsIsNotPerfect 必红；
//   - 若把 Incomplete 判据改回 data_governance.go 的 owner/来源口径，
//     TestComputeAttributeQuality_UsesRequiredAttributesNotOwner 必红。

func makeAttrCI(id, ciTypeID int, lifecycle string, attrs map[string]interface{}) *ConfigurationItem {
	return &ConfigurationItem{
		ID:              id,
		CINumber:        "CI-" + itoa(id),
		Name:            "host-" + itoa(id),
		CITypeID:        ciTypeID,
		LifecycleStatus: lifecycle,
		Attributes:      attrs,
	}
}

// 旧口径下这个 CI 会被判 complete（有 discovery source + owner），
// 但它的必填属性一个都没填。新口径必须报 incomplete。
func TestComputeAttributeQuality_UsesRequiredAttributesNotOwner(t *testing.T) {
	ci := makeAttrCI(1, 10, common.CILifecycleStatusOnline, map[string]interface{}{})
	ci.DiscoverySource = "aliyun"
	ci.Source = "aliyun"
	ci.OwnedBy = "alice"

	specs := map[int][]string{10: {"serial_number", "vendor"}}
	m := ComputeAttributeQuality([]*ConfigurationItem{ci}, specs, 7*24*time.Hour, time.Now())

	if m.Assessable != 1 {
		t.Fatalf("Assessable: got %d want 1", m.Assessable)
	}
	if m.Incomplete != 1 {
		t.Fatalf("有 owner 但必填属性全空，必须计 incomplete；got %d want 1", m.Incomplete)
	}
	if m.CompletenessPct != 0.0 {
		t.Fatalf("CompletenessPct: got %v want 0", m.CompletenessPct)
	}
}

// 正例：必填属性齐 → complete，且 3 个 CI 中 2 个齐 → 66.67%。
func TestComputeAttributeQuality_CountsFilledRequiredAttributes(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusOnline, map[string]interface{}{
			"serial_number": "SN-1", "vendor": "Dell",
		}),
		makeAttrCI(2, 10, common.CILifecycleStatusOnline, map[string]interface{}{
			"serial_number": "SN-2", "vendor": "HP",
		}),
		// 缺 vendor
		makeAttrCI(3, 10, common.CILifecycleStatusOnline, map[string]interface{}{
			"serial_number": "SN-3",
		}),
	}
	specs := map[int][]string{10: {"serial_number", "vendor"}}
	m := ComputeAttributeQuality(cis, specs, 7*24*time.Hour, time.Now())

	if m.Assessable != 3 {
		t.Fatalf("Assessable: got %d want 3", m.Assessable)
	}
	if m.Incomplete != 1 {
		t.Fatalf("Incomplete: got %d want 1", m.Incomplete)
	}
	if m.CompletenessPct != 66.67 {
		t.Fatalf("CompletenessPct: got %v want 66.67", m.CompletenessPct)
	}
}

// 🔴 核心回归锁：租户没有配任何必填属性定义时，绝不能报 100%。
// 旧实现（data_governance.go:66）用固定字段清单，这个场景根本不存在——
// 因此断言必须能区分「无可评估」与「全部合格」。
func TestComputeAttributeQuality_NoSpecsIsNotPerfect(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusOnline, nil),
		makeAttrCI(2, 10, common.CILifecycleStatusOnline, nil),
	}
	// 空 map：一个类型都没有必填定义
	m := ComputeAttributeQuality(cis, map[int][]string{}, 7*24*time.Hour, time.Now())

	if m.Assessable != 0 {
		t.Fatalf("无属性定义时 Assessable 必须为 0；got %d", m.Assessable)
	}
	if m.Incomplete != 0 {
		t.Fatalf("无属性定义时不得凭空报 incomplete；got %d", m.Incomplete)
	}
	if m.CompletenessPct != 0 {
		t.Fatalf("无属性定义时完整率必须为 0（不可评估），不得为 100；got %v", m.CompletenessPct)
	}
	if m.AssessedTypes != 0 {
		t.Fatalf("AssessedTypes: got %d want 0", m.AssessedTypes)
	}
	if m.Total != 2 || m.Active != 2 {
		t.Fatalf("总数/活跃数仍应正确统计：%+v", m)
	}
}

// 类型 A 有必填定义、类型 B 没有：完整率分母只含 A，B 不参与。
func TestComputeAttributeQuality_MixedTypesOnlyAssessableCount(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusOnline, map[string]interface{}{"serial_number": "SN-1"}),
		// 类型 20 无定义：即使属性全空也不算 incomplete，不进分母
		makeAttrCI(2, 20, common.CILifecycleStatusOnline, nil),
		makeAttrCI(3, 20, common.CILifecycleStatusOnline, nil),
	}
	specs := map[int][]string{10: {"serial_number"}}
	m := ComputeAttributeQuality(cis, specs, 7*24*time.Hour, time.Now())

	if m.Assessable != 1 {
		t.Fatalf("Assessable: got %d want 1（只有类型 10 可评估）", m.Assessable)
	}
	if m.Incomplete != 0 {
		t.Fatalf("Incomplete: got %d want 0", m.Incomplete)
	}
	if m.CompletenessPct != 100 {
		t.Fatalf("CompletenessPct: got %v want 100", m.CompletenessPct)
	}
	if m.AssessedTypes != 1 {
		t.Fatalf("AssessedTypes: got %d want 1", m.AssessedTypes)
	}
	if m.Total != 3 {
		t.Fatalf("Total: got %d want 3", m.Total)
	}
}

// 空白串 / nil / 空数组 都必须算「未填」。
// 旧实现只看 `""`，清空过的数组字段会被算成已填。
func TestComputeAttributeQuality_BlankValuesCountAsMissing(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusOnline, map[string]interface{}{
			"serial_number": "   ", // 纯空白
			"vendor":        nil,   // 显式 nil
			"tags":          []interface{}{},
			"spec":          map[string]interface{}{},
		}),
	}
	specs := map[int][]string{10: {"serial_number", "vendor", "tags", "spec"}}
	m := ComputeAttributeQuality(cis, specs, 7*24*time.Hour, time.Now())

	if m.Incomplete != 1 {
		t.Fatalf("全部空白/nil/空集合都必须判为缺失；got Incomplete=%d want 1", m.Incomplete)
	}
}

// 退役 CI 进 Total/Retired，但也参与完整率评估（它仍在账里，
// 只是不再在用）。这条锁的是「退役 ≠ 免检」。
func TestComputeAttributeQuality_RetiredStillAssessed(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusScrapped, nil),
	}
	specs := map[int][]string{10: {"serial_number"}}
	m := ComputeAttributeQuality(cis, specs, 7*24*time.Hour, time.Now())

	if m.Retired != 1 {
		t.Fatalf("Retired: got %d want 1", m.Retired)
	}
	// 退役 CI 计入 Assessable：完整性是账目属性，不因退役而消失。
	if m.Assessable != 1 {
		t.Fatalf("Assessable: got %d want 1（退役 CI 仍应可评估）", m.Assessable)
	}
	if m.Incomplete != 1 {
		t.Fatalf("Incomplete: got %d want 1", m.Incomplete)
	}
}

// 缺失属性统计：用于告诉运维「最该补哪个字段」。
func TestComputeAttributeQuality_MissingStatsOrderedByMissingCount(t *testing.T) {
	cis := []*ConfigurationItem{
		makeAttrCI(1, 10, common.CILifecycleStatusOnline, map[string]interface{}{"serial_number": "SN-1"}),
		makeAttrCI(2, 10, common.CILifecycleStatusOnline, map[string]interface{}{}),
		makeAttrCI(3, 10, common.CILifecycleStatusOnline, map[string]interface{}{}),
	}
	specs := map[int][]string{10: {"serial_number", "vendor"}}
	m := ComputeAttributeQuality(cis, specs, 7*24*time.Hour, time.Now())

	if len(m.MissingByAttribute) != 2 {
		t.Fatalf("应统计两个缺失属性；got %d", len(m.MissingByAttribute))
	}
	// vendor 3 条全缺、serial_number 缺 2 条 → vendor 排第一
	if m.MissingByAttribute[0].Attribute != "vendor" {
		t.Fatalf("缺失最多的属性应排第一；got %s", m.MissingByAttribute[0].Attribute)
	}
	if m.MissingByAttribute[0].Missing != 3 {
		t.Fatalf("vendor 缺失数：got %d want 3", m.MissingByAttribute[0].Missing)
	}
	if m.MissingByAttribute[1].Attribute != "serial_number" || m.MissingByAttribute[1].Missing != 2 {
		t.Fatalf("serial_number 缺失数应为 2；got %+v", m.MissingByAttribute[1])
	}
	// Applicable 是该类型 CI 总数
	if m.MissingByAttribute[0].Applicable != 3 {
		t.Fatalf("Applicable: got %d want 3", m.MissingByAttribute[0].Applicable)
	}
}

func TestComputeAttributeQuality_EmptySet(t *testing.T) {
	m := ComputeAttributeQuality(nil, map[int][]string{10: {"a"}}, time.Hour, time.Now())
	if m.Total != 0 || m.Assessable != 0 || m.CompletenessPct != 0 {
		t.Fatalf("空集合必须全 0；got %+v", m)
	}
}

// ----- 重复聚类 -----

func TestDetectDuplicateGroups_CloudIdentity(t *testing.T) {
	cis := []*ConfigurationItem{
		{
			ID: 1, CINumber: "CI-1", LifecycleStatus: common.CILifecycleStatusOnline,
			CloudProvider: "aliyun", CloudResourceID: "i-abc",
		},
		{
			ID: 2, CINumber: "CI-2", LifecycleStatus: common.CILifecycleStatusOnline,
			CloudProvider: "ALIYUN", CloudResourceID: " I-ABC ",
		},
		{
			ID: 3, CINumber: "CI-3", LifecycleStatus: common.CILifecycleStatusOnline,
			CloudProvider: "aliyun", CloudResourceID: "i-other",
		},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 1 {
		t.Fatalf("预期 1 组重复；got %d", len(groups))
	}
	g := groups[0]
	if g.Reason != DuplicateReasonCloudIdentity {
		t.Fatalf("Reason: got %s want %s", g.Reason, DuplicateReasonCloudIdentity)
	}
	if len(g.CIIDs) != 2 || g.CIIDs[0] != 1 || g.CIIDs[1] != 2 {
		t.Fatalf("组内 ID 应升序且为 [1 2]；got %v", g.CIIDs)
	}
}

func TestDetectDuplicateGroups_AssetTag(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 5, CINumber: "CI-5", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "A-100"},
		{ID: 6, CINumber: "CI-6", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "a-100"},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 1 || groups[0].Reason != DuplicateReasonAssetTag {
		t.Fatalf("资产标签重复未识别：%+v", groups)
	}
	if groups[0].Key != "tag:a-100" {
		t.Fatalf("判重键应归一化为小写：got %s", groups[0].Key)
	}
}

func TestDetectDuplicateGroups_SerialPlusName(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 7, CINumber: "CI-7", LifecycleStatus: common.CILifecycleStatusOnline, Name: "web-01", SerialNumber: "SN-X"},
		{ID: 8, CINumber: "CI-8", LifecycleStatus: common.CILifecycleStatusOnline, Name: "WEB-01 ", SerialNumber: " sn-x "},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 1 || groups[0].Reason != DuplicateReasonSerialName {
		t.Fatalf("序列号+名称重复未识别：%+v", groups)
	}
}

// 判重键优先级：有云身份就走云身份，不再按资产标签重复报一遍。
func TestDetectDuplicateGroups_PriorityAvoidsDoubleCounting(t *testing.T) {
	cis := []*ConfigurationItem{
		{
			ID: 1, CINumber: "CI-1", LifecycleStatus: common.CILifecycleStatusOnline,
			CloudProvider: "aliyun", CloudResourceID: "i-1", AssetTag: "T-9",
		},
		{
			ID: 2, CINumber: "CI-2", LifecycleStatus: common.CILifecycleStatusOnline,
			CloudProvider: "aliyun", CloudResourceID: "i-1", AssetTag: "T-9",
		},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 1 {
		t.Fatalf("同一对 CI 只能出现在一组；got %d 组: %+v", len(groups), groups)
	}
	if groups[0].Reason != DuplicateReasonCloudIdentity {
		t.Fatalf("应优先按云身份判重；got %s", groups[0].Reason)
	}
}

// 退役 CI 不参与判重：退役动作本身不该变成数据质量告警。
func TestDetectDuplicateGroups_ScrappedExcluded(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 1, CINumber: "CI-1", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "T-1"},
		{ID: 2, CINumber: "CI-2", LifecycleStatus: common.CILifecycleStatusScrapped, AssetTag: "T-1"},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 0 {
		t.Fatalf("退役 CI 不得参与判重；got %+v", groups)
	}
}

// 无任何身份信息的 CI 不参与判重（聚类全靠猜等于制造告警）。
func TestDetectDuplicateGroups_NoIdentityNoGroup(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 1, CINumber: "CI-1", LifecycleStatus: common.CILifecycleStatusOnline},
		{ID: 2, CINumber: "CI-2", LifecycleStatus: common.CILifecycleStatusOnline},
	}
	if groups := DetectDuplicateGroups(cis); len(groups) != 0 {
		t.Fatalf("无身份字段时不得判重；got %+v", groups)
	}
}

// 输出稳定性：组间按 key 排序，组内按 ID 排序。
func TestDetectDuplicateGroups_StableOrder(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 9, CINumber: "CI-9", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "Z-1"},
		{ID: 3, CINumber: "CI-3", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "Z-1"},
		{ID: 4, CINumber: "CI-4", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "A-1"},
		{ID: 5, CINumber: "CI-5", LifecycleStatus: common.CILifecycleStatusOnline, AssetTag: "A-1"},
	}
	groups := DetectDuplicateGroups(cis)
	if len(groups) != 2 {
		t.Fatalf("预期 2 组；got %d", len(groups))
	}
	if groups[0].Key != "tag:a-1" {
		t.Fatalf("组应按 key 升序；got %s", groups[0].Key)
	}
	if groups[0].CIIDs[0] != 4 || groups[0].CIIDs[1] != 5 {
		t.Fatalf("组内应按 ID 升序；got %v", groups[0].CIIDs)
	}
	if groups[1].CIIDs[0] != 3 || groups[1].CIIDs[1] != 9 {
		t.Fatalf("组内应按 ID 升序；got %v", groups[1].CIIDs)
	}
}

func TestCountDuplicateCIs(t *testing.T) {
	groups := []DuplicateGroup{
		{CIIDs: []int{1, 2}},
		{CIIDs: []int{3, 4, 5}},
	}
	if n := CountDuplicateCIs(groups); n != 5 {
		t.Fatalf("got %d want 5", n)
	}
	if n := CountDuplicateCIs(nil); n != 0 {
		t.Fatalf("空组应为 0；got %d", n)
	}
}

func TestDuplicateRatePct(t *testing.T) {
	cases := []struct {
		dup, total int
		want       float64
	}{
		{0, 100, 0},
		{1, 100, 1},
		{2, 100, 2},
		{1, 3, 33.33},
		{5, 0, 0}, // 除零保护
		{-1, 10, 0},
	}
	for _, tc := range cases {
		if got := DuplicateRatePct(tc.dup, tc.total); got != tc.want {
			t.Fatalf("DuplicateRatePct(%d,%d)=%v want %v", tc.dup, tc.total, got, tc.want)
		}
	}
}
