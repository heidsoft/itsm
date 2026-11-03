package capability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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
			Items             []Capability `json:"items"`
			AcceptanceVersion string       `json:"acceptanceVersion"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.NotEmpty(t, response.Data.Items)
	require.Equal(t, acceptanceVersion, response.Data.AcceptanceVersion)
	for _, item := range response.Data.Items {
		require.Equal(t, acceptanceVersion, item.AcceptanceVersion)
	}

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityPilot, byKey["marketplace"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["marketplace"].AllowedActions)
	require.False(t, byKey["connector.feishu"].DeploymentReady)
	require.NotEmpty(t, byKey["connector.feishu"].DegradedReason)
	// SSO/OIDC 已实现 11 个端点，矩阵不得再声明为 disabled。
	require.Equal(t, MaturityPilot, byKey["identity.oidc"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["identity.oidc"].AllowedActions)
}

// acceptanceVersion 是能力矩阵对外的验收基线，不是内部备注：值改了但商业契约
// 没改（或反之）会让售前拿到一份两端不一致的承诺，因此两边必须同源。
func TestAcceptanceVersionMatchesCommercialContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "product", "itsm-commercial-capability-contract.md"))
	require.NoError(t, err)
	doc := string(raw)
	require.Contains(t, doc, "`acceptanceVersion`")
	require.Contains(t, doc, "`"+acceptanceVersion+"`", "商业能力契约未声明当前验收基线")
}

// 发版即重新确认能力矩阵：acceptanceVersion 早于最新发布版本，说明矩阵已经
// 跨过至少一个版本没有被复核，继续对外宣称旧基线就是在卖上一版的事实。
func TestAcceptanceVersionIsNotBehindLatestRelease(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "CHANGELOG.md"))
	require.NoError(t, err)
	match := regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\] - (\d{4}-\d{2}-\d{2})\s*$`).FindSubmatch(raw)
	require.Len(t, match, 3, "CHANGELOG 缺少带日期的发布版本标题")
	releaseDate, err := time.Parse("2006-01-02", string(match[2]))
	require.NoError(t, err)
	acceptedAt, err := time.Parse("2006.01.02", acceptanceVersion)
	require.NoError(t, err, "acceptanceVersion 必须是 YYYY.MM.DD")
	require.False(t, acceptedAt.Before(releaseDate),
		"能力矩阵验收基线 %s 早于最新发布 %s（%s）：发版时必须复核矩阵并同批更新",
		acceptanceVersion, string(match[2]), string(match[1]))
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

// marketplace 的 manage 反映租户 admin 的 marketplace:write，而不是平台目录管理权：
// 普通成员只能读；商品目录上架/下架仍由 marketplace_catalog:* 独占授予 sysadmin。
func TestMarketplaceManageFollowsTenantAdminPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	actionsFor := func(role string) []string {
		router := gin.New()
		router.GET("/capabilities", func(c *gin.Context) {
			c.Set("tenant_id", 7)
			c.Set("role", role)
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
		for _, item := range response.Data.Items {
			if item.Key == "marketplace" {
				return item.AllowedActions
			}
		}
		require.FailNow(t, "能力矩阵缺少 marketplace 条目")
		return nil
	}

	require.Equal(t, []string{"read", "manage"}, actionsFor("admin"))
	require.Equal(t, []string{"read"}, actionsFor("technician"))
}

// 没有实现证据的能力不得申报 GA。标签维持 Pilot：工单标签完整，组织标签
// （tags 表）有只读查询但没有写入口。workflow 的 Pilot 理由见
// TestWorkflowCapabilityStaysPilotUntilBridgingIsClosed。
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

// TestWorkflowCapabilityStaysPilotUntilBridgingIsClosed 锁定 2026-10-31 的运行时降级。
//
// 矩阵此前把 workflow 申报为 GA，而 README 成熟度表与商业能力契约都写 Pilot。
// 矩阵是前端与控制台的事实源：谎报 GA 会让运营把 BPMN 当成已验收能力交付。
// 当前真实缺口（问题域无 workflow.start 生产者、补偿无内置模板、ServiceTask
// 副作用在完成事务之外、缺跨域实例级 E2E）全部写进 pilotReason；等这些项逐个
// 闭环并拿到证据后，才允许连同本测试一起晋升。
func TestWorkflowCapabilityStaysPilotUntilBridgingIsClosed(t *testing.T) {
	var found *definition
	for i := range registry {
		if registry[i].key == "workflow" {
			found = &registry[i]
			break
		}
	}
	require.NotNil(t, found, "能力矩阵缺少 workflow 条目")
	require.Equal(t, MaturityPilot, found.maturity, "workflow 未补齐跨域闭环前不得申报 GA")
	require.True(t, found.built)
	require.Contains(t, found.pilotReason, "晋升条件")
	require.Contains(t, found.pilotReason, "问题域")
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
//
// 2026-10-31 复核又补三例：组织标签并非「无任何管理面」（有只读 GET）、
// 云发现从未有增量通道、OIDC 已有假 IdP 全链路；同时移除了宣称存在、
// 实际只切换日志标签的 AI「生产验收模式」。
func TestPilotReasonsDoNotCiteClosedBlockers(t *testing.T) {
	closedBlockers := map[string][]string{
		// 2026-10-23：Job 级执行器消费 cmdb.cloud_discovery.run，
		// DiscoveryScheduler 消费 discovery_sources.schedule。
		// 2026-10-31：增量通道从未实现，不得再说成「增量发现未验收」。
		"cmdbDiscovery":      {"队列无消费者", "没有执行者", "仅被单测调用", "worker 消费", "全量与增量发现未验收"},
		"cmdbReconciliation": {"发现链路没有消费者", "只能对空集", "没有确认/驳回的写入面"},
		// 2026-10-23：分诊/RCA 已强制 promptVersion + model + confidence + 底账。
		// 2026-10-31：仿真「生产验收模式」的 ITSM_AI_EVAL_USE_LLM 只会切换日志标签，已移除。
		// 2026-10-31（晚）：代码侧三项验收缺口已闭环（a9dbec0aa）——RAG 期望改按
		// 文章标题引用真实语料（不再是虚构 ID）、预测分数由生产 SLABreachPredictor
		// 产出（不再取自数据集自带 probability）、LLM 档验收命令 cmd/ai-eval 已就绪
		// （不再「没有可用的真实 LLM 候选生成器」）。这些表述不得再出现在理由里。
		"ai": {
			"confidence、接受/拒绝反馈", "evaluator 未接入调用链", "ITSM_AI_EVAL_USE_LLM", "生产验收模式",
			"expected_doc_ids 是虚构 ID", "expected_doc_ids 为虚构 ID", "预测分值仍取自数据集", "没有可用的真实 LLM 候选生成器",
		},
		// 2026-10-24：ServiceTask action 两端口径接通；2026-10-23 工单审批桥接
		// 收进业务事务；发布创建与 workflow.start 同事务入队。这些不再是缺口。
		"workflow": {"ServiceTask action 链路未打通", "工单审批桥接非原子", "release 域自动起流"},
		// 2026-10-22：命令死信/失败投递重放产品面与市场安装/配置/卸载闭环落地。
		"connector.feishu": {"死信重放未实现", "市场安装闭环未验收", "死信人工重放与市场安装闭环未验收"},
		// 2026-10-31：分类可见性已在全部知识读面收口并补齐跨角色回归。
		"knowledge": {"分类可见性的跨角色检索未做端到端验收", "版本化发布", "发布版本语义未收敛"},
		// 2026-10-31：组织标签有只读查询端点，缺的只是写入口；不得再说成「无任何管理面」。
		"tag": {"无任何管理面", "没有管理面", "从未被装配"},
		// 2026-10-31：假 IdP 已覆盖登录→回调→换票→JWT 整链，缺的只是真实 IdP 联调。
		"identity.oidc": {"现有覆盖为服务单测与路由测试", "没有与真实 IdP 的端到端联调证据"},
		// 2026-10-31：交付已接阿里云 ECS RunInstances（租户凭据引用、未配置失败关闭）。
		// 2026-11-02：真实账号验收已执行、凭据支持 env://、失败靠幂等键对账 +
		// 人工判定兜底，以下旧结论不得再写回理由。
		"serviceRequest": {
			"交付 Provider 仍是 StubProvider", "未接真实云 API", "交付 Provider 仍为 Stub",
			"租户云账号 + secret:// 凭据引用", "多步交付中途失败后的资源回滚/补偿未实现",
			"至少一条真实账号交付验收",
		},
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

// validateServiceRequestPilotReason 校验服务请求的 Pilot 理由是否与 2026-11-02
// 之后的交付现实一致。抽成返回 error 的纯函数，是为了在测试里把历史旧文案
// 也喂进来做反向验证：如果守卫只是「检查当前字符串里有没有关键词」，改回旧
// 结论同样会绿，那就是假绿。
func validateServiceRequestPilotReason(reason string) error {
	// 已落地/已澄清的事实，必须逐处在理由里出现。
	for _, phrase := range []string{
		"env://ITSM_ALIYUN", // 凭据双形态；真实账号验收用的正是 env://
		"ClientToken",       // 幂等键 + 失败对账证据
		"人工裁决",              // 判定落审计谱系
		"永久性",               // 永久云错误快速失败，不再重投
		"真实账号",              // 端到端验收已执行
		"自动释放",              // 仍缺的动作，不能再笼统写成「缺补偿」
		"晋升条件",
	} {
		if !strings.Contains(reason, phrase) {
			return fmt.Errorf("pilotReason 缺少已落地/已澄清事实 %q", phrase)
		}
	}
	// 2026-10-31 的旧结论，任何一条重新出现都说明理由没跟上代码。
	for _, phrase := range []string{
		"租户云账号 + secret:// 凭据引用",
		"多步交付中途失败后的资源回滚/补偿未实现",
		"至少一条真实账号交付验收",
		"交付 Provider 仍是 StubProvider",
	} {
		if strings.Contains(reason, phrase) {
			return fmt.Errorf("pilotReason 引用了过期结论 %q", phrase)
		}
	}
	return nil
}

// TestServiceRequestPilotReasonReflectsDeliveryReality 双向锁定服务请求的
// Pilot 理由：当前值必须通过，退回旧文案或删掉任一已落地事实必须被拒。
func TestServiceRequestPilotReasonReflectsDeliveryReality(t *testing.T) {
	var item *definition
	for i := range registry {
		if registry[i].key == "serviceRequest" {
			item = &registry[i]
			break
		}
	}
	require.NotNil(t, item, "能力矩阵必须有 serviceRequest 条目")
	require.Equal(t, MaturityPilot, item.maturity)
	require.NoError(t, validateServiceRequestPilotReason(item.pilotReason))

	// 反向验证一：10-31 的旧文案喂回同一判定函数，必须被拒绝。
	old := "交付已接阿里云 ECS 真实 Provider（租户云账号 + secret:// 凭据引用，未配置即失败关闭），但目前只覆盖 ECS 一种资源；多步交付中途失败后的资源回滚/补偿未实现；目录项到 CI Type、交付模板、BPMN、SLA 的绑定口径仍分散。晋升条件：补偿动作 + 目录项绑定统一 + 至少一条真实账号交付验收。"
	require.Error(t, validateServiceRequestPilotReason(old),
		"旧文案必须被拒，否则守卫只是匹配当前字符串（假绿）")

	// 反向验证二：逐项删掉一个已落地事实也必须被拒，防止断言取并集后
	// 靠其它关键词偶然通过。
	for _, drop := range []string{"ClientToken", "人工裁决", "env://ITSM_ALIYUN"} {
		require.Error(t, validateServiceRequestPilotReason(strings.Replace(item.pilotReason, drop, "", 1)),
			"移除 %q 后守卫仍应拒绝", drop)
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
