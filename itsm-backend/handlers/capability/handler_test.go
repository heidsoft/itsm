package capability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandlerReturnsTenantAwareCapabilityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ITSM_FEISHU_READY", "false")

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.NotEmpty(t, response.Data.Items)

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityPilot, byKey["marketplace"].Maturity)
	require.Equal(t, []string{"read"}, byKey["marketplace"].AllowedActions)
	require.False(t, byKey["connector.feishu"].DeploymentReady)
	require.NotEmpty(t, byKey["connector.feishu"].DegradedReason)
	// SSO/OIDC 已实现 11 个端点，矩阵不得再声明为 disabled。
	require.Equal(t, MaturityPilot, byKey["identity.oidc"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["identity.oidc"].AllowedActions)
}

func TestHandlerDoesNotGrantActionsWithoutTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/capabilities", NewHandler(nil))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	for _, item := range response.Data.Items {
		require.False(t, item.TenantReady)
		require.Empty(t, item.AllowedActions)
	}
}

// 没有实现证据的能力不得申报 GA。2026-10-22 盘点后仍维持 Pilot 的是标签：
// 工单标签完整，但组织标签（tags 表）只由 seeder 写入、没有管理面。
// 模板此前被「通用 /api/v1/templates 从未注册」这条幽灵缺陷卡住——前端仓库里
// 既没有 template-api.ts，也没有任何该路径调用，实际表面是
// /api/v1/tickets/templates 的 8 个端点 + /tickets/templates 页，已具备 GA 证据。
func TestUnprovenCapabilitiesAreNotDeclaredGA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ITSM_FEISHU_READY", "false")

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityPilot, byKey["tag"].Maturity)
	require.NotEmpty(t, byKey["tag"].PilotReason)
}

// TestPromotedCapabilitiesAreGA 锁定 2026-10-21 的通知域 GA 晋升：
// 投递审计读面 + 渠道诊断补齐后，「没收到通知」的三类根因都可定位。
//
// 2026-10-22 追加问题域与模板：问题域的 SLA/评论/关闭证据/KEDB 关联四项阻断
// 全部闭环且装配进生产路径（bootstrap 里 SetSLAService / SetKnowledgeRepository
// 都有调用点），模板的 8 个端点与页面齐备。此前两者维持 Pilot 的依据是
// 「/problems/:id/sla 仍 503」与「/api/v1/templates 未注册」，两者在代码里
// 都已不成立——注释过期不等于能力未就绪，也不该继续对外宣称 Pilot。
func TestPromotedCapabilitiesAreGA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityGA, byKey["notification"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["notification"].AllowedActions)

	for _, key := range []string{"problem", "template"} {
		require.Equalf(t, MaturityGA, byKey[key].Maturity, "%s 应已晋升 GA", key)
		require.Emptyf(t, byKey[key].PilotReason, "%s 已是 GA，不得再带 pilot 理由", key)
		require.Containsf(t, byKey[key].AllowedActions, "manage", "%s 的 GA 条目应可管理", key)
	}
}

// TestPilotReasonsDoNotCiteClosedBlockers 锁定「理由不得引用已闭环的阻断项」。
//
// 这是 2026-10-22 那条教训的可执行版本：注释会腐烂，接口字段不会，但接口字段
// 同样会腐烂——只要有人修完阻断项却忘了改理由。此前的实例是 problem 的
// 「SLA 仍 503」与 template 的「/api/v1/templates 未注册」，两者在代码里都
// 已不成立。2026-10-23 又出现两例：云发现执行器与 cron 调度落地后，
// cmdbDiscovery 的理由仍写着「队列无消费者」；AI 三契约落地后，ai 的理由
// 仍写着「confidence/prompt 版本未强制」。
//
// 因此把「已修好的事实」写成禁词表：一旦这些能力重新把旧结论写回理由，
// 测试立即失败，逼作者去核对当前代码。
func TestPilotReasonsDoNotCiteClosedBlockers(t *testing.T) {
	closedBlockers := map[string][]string{
		// 2026-10-23：Job 级执行器消费 cmdb.cloud_discovery.run，
		// DiscoveryScheduler 消费 discovery_sources.schedule。
		"cmdbDiscovery":      {"队列无消费者", "没有执行者", "仅被单测调用", "worker 消费"},
		"cmdbReconciliation": {"发现链路没有消费者", "只能对空集", "没有确认/驳回的写入面"},
		// 2026-10-23：分诊/RCA 已强制 promptVersion + model + confidence + 底账。
		"ai": {"confidence、接受/拒绝反馈", "evaluator 未接入调用链"},
	}
	byKey := make(map[string]definition, len(registry))
	for _, item := range registry {
		byKey[item.key] = item
	}
	for key, phrases := range closedBlockers {
		item, ok := byKey[key]
		require.Truef(t, ok, "能力矩阵缺少条目: %s", key)
		for _, phrase := range phrases {
			require.NotContainsf(t, item.pilotReason, phrase,
				"%s 的理由引用了已闭环的阻断项 %q，请按当前代码重写", key, phrase)
		}
	}
}

// TestReportCapabilityIsTracked 锁定报表域在矩阵中的存在。
//
// 报表此前只在 docs/product 的能力契约表里（报表/运营 = Pilot），矩阵里没有
// 这一行——而矩阵才是前端与控制台的事实源，缺行等于这个域对客户不可见。
func TestReportCapabilityIsTracked(t *testing.T) {
	var found *definition
	for i := range registry {
		if registry[i].key == "report" {
			found = &registry[i]
			break
		}
	}
	require.NotNil(t, found, "报表域必须在能力矩阵中可查")
	require.Equal(t, MaturityPilot, found.maturity)
	require.Contains(t, found.pilotReason, "晋升条件")
}

// TestPilotReasonsAreCurrentAndActionable 把「为什么还是 Pilot」从 Go 注释
// 提升为接口契约的一部分，并强制它可执行。
//
// 背景：2026-10-22 盘点发现，注释里的 Pilot 理由会腐烂——problem 的注释写着
// 「SLA 端点仍 503」，而 SLA 早已落地并装配；template 的注释引用了一个前端
// 根本不存在的 template-api.ts。注释没人会回头改，接口字段会。因此：
//   - pilot 条目必须带理由，且理由必须写出「晋升条件」（否则读者无从推进）；
//   - ga 条目必须没有理由（留着就是过期结论）；
//   - pilot 必须 built（未构建的能力是 disabled，不是 pilot）。
func TestPilotReasonsAreCurrentAndActionable(t *testing.T) {
	for _, item := range registry {
		switch item.maturity {
		case MaturityPilot:
			require.Truef(t, item.built, "pilot 能力必须已构建: %s", item.key)
			reason := strings.TrimSpace(item.pilotReason)
			require.NotEmptyf(t, reason, "pilot 能力必须声明 pilotReason: %s", item.key)
			require.GreaterOrEqualf(t, utf8.RuneCountInString(reason), 30,
				"pilotReason 太短，无法说明阻断项: %s", item.key)
			require.Containsf(t, reason, "晋升条件", "pilotReason 必须给出可验证的晋升条件: %s", item.key)
			for _, placeholder := range []string{"TODO", "待补", "暂不支持", "未知"} {
				require.NotContainsf(t, reason, placeholder,
					"pilotReason 含占位词 %q，读者无法据此推进: %s", placeholder, item.key)
			}
		case MaturityGA:
			require.Emptyf(t, item.pilotReason,
				"ga 能力不得保留 pilotReason（过期理由会与 maturity 自相矛盾）: %s", item.key)
		case MaturityDisabled:
			require.Falsef(t, item.built, "disabled 能力不得声称已构建: %s", item.key)
		}
	}
}
