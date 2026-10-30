package router

import (
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	cmdbHandler "itsm-backend/handlers/cmdb"
	service_catalog "itsm-backend/handlers/service_catalog"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件锁「报表清单不含死链」。
//
// 背景：GET /reports 的列表自报 7 张报表，其中 cmdb-quality 与 catalog-usage
// 从来没有对应的路由——清单是内联在列表接口里的字面量，与路由注册是两份
// 互不相干的文本，谁也不会因为对方缺失而失败。结果就是界面上摆着两个点了
// 就 404 的入口，且没有任何一道关口会发现。
//
// 两处修复：清单提取成 reportCatalog 由接口与测试共用；两条遗漏的报表接上
// 既有数据源（CMDB 治理看板、服务目录统计）。
//
// 这里直接打 SetupReportsRoutes 而不是完整 SetupRoutes：完整装配需要
// dashboard/ticket/incident 三个服务，与「清单与路由是否对齐」无关，
// 反而会把守卫测试拖成集成测试。

// reportsCatalogRoutes 装配一个只含报表路由的 engine 并返回注册路径集合。
func reportsCatalogRoutes(t *testing.T, withCMDB, withCatalog bool) map[string]bool {
	t.Helper()
	dsn := fmt.Sprintf("file:reports_catalog_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var cmdb *cmdbHandler.Handler
	if withCMDB {
		cmdb = cmdbHandler.NewHandler(cmdbHandler.NewService(cmdbHandler.NewEntRepository(client), nil, logger))
	}
	var catalog *service_catalog.Handler
	if withCatalog {
		catalog = service_catalog.NewHandler(service_catalog.NewService(service_catalog.NewEntRepository(client), logger))
	}

	// h 传 nil：本测试只关心路由是否注册，不触发任何 handler 逻辑。
	SetupReportsRoutes(r.Group("/api/v1/t"), nil, nil, nil, cmdb, catalog)

	registered := map[string]bool{}
	for _, ri := range r.Routes() {
		registered[ri.Method+" "+ri.Path] = true
	}
	return registered
}

func TestReportsCatalogHasNoDeadLinks(t *testing.T) {
	registered := reportsCatalogRoutes(t, true, true)

	require.NotEmpty(t, reportCatalog, "报表清单不能为空，否则这条守卫会零命中地通过")
	for _, item := range reportCatalog {
		require.NotEmpty(t, item.Path, "报表 %q 缺 path", item.ID)
		require.True(t, registered["GET /api/v1/t"+item.Path],
			"报表清单里的 %q 承诺了 path=%s，但没有注册任何对应路由（点了就是 404）", item.ID, item.Path)
	}
}

// TestReportsCatalogHidesUnwiredDomains 清单必须与「实际注册的路由」一致，
// 而不是与「静态清单」一致。
//
// 清单是静态的，路由随域装配动态变化：CMDB / 服务目录域未装配时对应路由不
// 注册，清单若照旧宣传这两张报表，前端报表中心就会渲染出两张点了 404 的卡片
// ——正是本次修复要消灭的那类死链，只是触发条件从「忘了注册」变成「域未装配」。
func TestReportsCatalogHidesUnwiredDomains(t *testing.T) {
	ids := func(items []gin.H) map[string]bool {
		out := make(map[string]bool, len(items))
		for _, item := range items {
			id, _ := item["id"].(string)
			out[id] = true
		}
		return out
	}

	full := ids(availableReports(true, true))
	for _, item := range reportCatalog {
		require.True(t, full[item.ID], "域已装配时清单应包含 %q", item.ID)
	}

	partial := ids(availableReports(false, false))
	require.False(t, partial["cmdb-quality"], "CMDB 域未装配时清单不得宣传该报表")
	require.False(t, partial["catalog-usage"], "服务目录域未装配时清单不得宣传该报表")
	require.True(t, partial["tickets"], "不依赖域装配的报表仍应在清单里")

	// 对外契约字段只有 id/name/path：内部依赖标记不得泄漏进响应。
	for _, item := range availableReports(true, true) {
		require.Len(t, item, 3, "报表清单项只允许 id/name/path 三个字段")
	}
}

// TestReportsCatalogRoutesAbsentWhenDomainNotWired 域未装配时对应报表不注册——
// 与其注册一条必然 500 的路由，不如让它从路由表里消失。反过来说，只要它出现
// 在路由表里，背后就一定是真实数据源。
func TestReportsCatalogRoutesAbsentWhenDomainNotWired(t *testing.T) {
	registered := reportsCatalogRoutes(t, false, false)
	require.False(t, registered["GET /api/v1/t/reports/cmdb-quality"], "CMDB 域未装配时不该注册该报表")
	require.False(t, registered["GET /api/v1/t/reports/catalog-usage"], "目录域未装配时不该注册该报表")
	require.True(t, registered["GET /api/v1/t/reports/tickets"], "不依赖域装配的报表仍应注册")
}
