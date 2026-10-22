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
