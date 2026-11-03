package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMarketplaceDeliveryRuntimeIsInjected 锁定「技能交付」这条链在装配期就接上：
// 运行时注册表进市场服务、市场服务进技能 Handler。
//
// 少任何一边都会变成假验收：没注入运行时，安装技能商品一律「运行时未装配」失败；
// 没注入 entitlements，卸载商品后租户照样能调用该技能——「装了就有权、卸了就撤权」
// 只存在于服务层的单元测试里。
func TestMarketplaceDeliveryRuntimeIsInjected(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	assembly := functionSource(t, source, "NewApplication")

	for _, required := range []string{
		"marketplaceSvc.SetSkillRuntime(skillRegistry)",
		"skillHandler.SetSkillEntitlements(marketplaceSvc)",
		"MarketplaceService: marketplaceSvc,",
	} {
		require.Truef(t, strings.Contains(assembly, required),
			"NewApplication 缺少装配项 %q", required)
	}

	// 单一实例：后台 pass、路由与技能门禁必须看到同一份安装记录与同一套判定。
	require.Equal(t, 1, strings.Count(assembly, "marketplaceService.NewService("),
		"市场服务只能有一个实例来源")
}

// TestAutoUpgradePassUsesInjectedMarketplaceService 校验 auto_upgrade 的执行者来源：
// 后台 ticker 只能用 Application 上注入的实例，且必须跑在带原因的 system context 上。
//
// 自建实例会绕过 SetSkillRuntime/SetConnectorManager，升级每轮都失败却仍上报 scanned；
// 用 context.Background() 则让关停信号传不进来，进程退出时升级写到一半。
func TestAutoUpgradePassUsesInjectedMarketplaceService(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	worker := functionSource(t, source, "startBackgroundTasks")

	idx := strings.Index(worker, "RunAutoUpgradePass(")
	require.Greaterf(t, idx, -1, "后台任务未执行 auto_upgrade pass")

	require.NotContains(t, worker, "marketplaceService.NewService(",
		"后台任务不得自建市场服务实例")
	require.Contains(t, worker, "app.MarketplaceService.RunAutoUpgradePass(roundCtx)")

	// 升级跨全部租户，只能走显式 system context；roundCtx 必须来自该 context 并带超时。
	require.Contains(t, worker, `tenantctx.SystemContext(ctx, "bootstrap:marketplace-auto-upgrade"`)
	require.Contains(t, worker, "context.WithTimeout(sysCtx,")
}
