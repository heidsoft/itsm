package capability

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// validatePilotReasonShape 校验预览理由能被控制台的三段式切分可靠消费。
//
// 既有 TestPilotReasonsAreCurrentAndActionable 锁的是「出现『晋升条件』四字」，
// 但那挡不住结构性腐坏：ai 条目曾把整段「剩余阻断 + 晋升条件」重复写了两次、
// 还夹带「此句曾…意外丢失，本次恢复」的过程注记，旧守卫照样绿，而控制台
// 只能把 2300 字整段平铺，读者要读到末尾才知道还差什么。这里把「可切分」
// 提升为契约：标记恰好一次、条件后必须有实质内容、不得夹带过程注记。
//
// 写成返回 error 的纯函数而不是直接断言，是为了能把畸形文案喂进来做反向
// 验证——只检查「当前字符串里有没有关键词」的守卫会假绿。
func validatePilotReasonShape(maturity Maturity, reason string) error {
	trimmed := strings.TrimSpace(reason)
	if maturity != MaturityPilot {
		if trimmed != "" {
			return fmt.Errorf("非 pilot 条目不得携带 pilotReason")
		}
		return nil
	}
	if trimmed == "" {
		return fmt.Errorf("pilot 条目缺少 pilotReason")
	}
	if count := strings.Count(trimmed, "晋升条件："); count != 1 {
		return fmt.Errorf("pilotReason 必须恰好含一次「晋升条件：」，实际 %d 次（重复段会让切分结果不可核对）", count)
	}
	if count := strings.Count(trimmed, "剩余阻断："); count > 1 {
		return fmt.Errorf("pilotReason 含 %d 次「剩余阻断：」，重复段会把旧结论留在展示区", count)
	}
	for _, note := range []string{"本次恢复", "意外丢失", "此句曾"} {
		if strings.Contains(trimmed, note) {
			return fmt.Errorf("pilotReason 不得夹带过程性注记 %q", note)
		}
	}
	criteria := strings.TrimSpace(strings.SplitN(trimmed, "晋升条件：", 2)[1])
	criteria = strings.Trim(criteria, "。；; ")
	if utf8.RuneCountInString(criteria) < 8 {
		return fmt.Errorf("晋升条件后必须有实质内容，实际 %d 字", utf8.RuneCountInString(criteria))
	}
	for _, placeholder := range []string{"TODO", "待补", "未知", "暂不支持"} {
		if strings.Contains(criteria, placeholder) {
			return fmt.Errorf("晋升条件含占位词 %q", placeholder)
		}
	}
	return nil
}

// TestPilotReasonShapeIsConsumableByControlPlane 把「预览态可核对」锁在接口
// 文本上：任何一条 Pilot 理由长得无法切分，都等于对客户重新变得不可核对。
func TestPilotReasonShapeIsConsumableByControlPlane(t *testing.T) {
	for _, item := range registry {
		require.NoErrorf(t, validatePilotReasonShape(item.maturity, item.pilotReason),
			"能力 %s 的 pilotReason 无法被控制台可靠切分", item.key)
	}
}

// TestPilotReasonShapeRejectsMalformedReasons 是反向验证：把各类腐坏文案
// 喂回同一个判定函数必须全部被拒，否则守卫只是匹配当前字符串（假绿）。
func TestPilotReasonShapeRejectsMalformedReasons(t *testing.T) {
	valid := "进展已落地并接入生产路径。剩余阻断：真实账号下的发现未验收。晋升条件：完成真实账号发现验收并归档报告。"
	require.NoError(t, validatePilotReasonShape(MaturityPilot, valid))

	cases := map[string]string{
		"缺晋升条件标记": "进展已落地。剩余阻断：真实账号未验收，没有给出下一步。",
		"晋升条件为空":  "进展已落地。剩余阻断：真实账号未验收。晋升条件：",
		"晋升条件过短":  "进展已落地。晋升条件：待定",
		"重复段":     valid + valid,
		"过程注记":    strings.Replace(valid, "真实账号下的发现未验收", "真实账号下的发现未验收（此句曾意外丢失，本次恢复）", 1),
		"占位词":     "进展已落地。剩余阻断：真实账号未验收。晋升条件：TODO 补齐验收口径。",
	}
	for name, reason := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, validatePilotReasonShape(MaturityPilot, reason))
		})
	}
	require.Error(t, validatePilotReasonShape(MaturityGA, valid), "GA 带理由必须被拒")
	require.Error(t, validatePilotReasonShape(MaturityPilot, ""), "pilot 缺理由必须被拒")
}
