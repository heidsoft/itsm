package marketplace

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 技能交付 + 升级 + 自动升级 + 依赖对齐的 HTTP 面契约。
//
// 这一组用例是「市场能投入生产」的判定依据：它走的是真实注册的路由
// （Handler.RegisterRoutes + RequirePermission），而不是未接线的 helper，
// 所以任何一条挂掉都意味着界面点下去会失败。

// createRouteSkillItem 建一个目录已上架、带已发布版本行与定位符的技能商品。
func createRouteSkillItem(t *testing.T, ctx context.Context, client *ent.Client, name string, deps ...map[string]string) *ent.MarketplaceItem {
	t.Helper()

	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeSkill).
		SetTitle(name + " 技能").
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)

	create := client.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion("1.0.0").
		SetStatus(itemversion.StatusPublished).
		SetManifestPath("skill://ai.triage").
		SetReleasedAt(time.Now())
	if len(deps) > 0 {
		create = create.SetDependencies(deps[0])
	}
	_, err = create.Save(ctx)
	require.NoError(t, err)
	return item
}

// addRouteSkillVersion 追加一个版本行（默认已发布），用于升级/回滚目标。
func addRouteSkillVersion(t *testing.T, ctx context.Context, client *ent.Client, item *ent.MarketplaceItem, version, locator string, status itemversion.Status) {
	t.Helper()
	_, err := client.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion(version).
		SetStatus(status).
		SetManifestPath(locator).
		Save(ctx)
	require.NoError(t, err)
}

func decodeInstallation(t *testing.T, body string) InstallationResponse {
	t.Helper()
	var payload struct {
		Code int                  `json:"code"`
		Data InstallationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &payload))
	require.Zero(t, payload.Code, body)
	return payload.Data
}

// TestMarketplaceRoutes_SkillInstallActivates 技能安装走真实交付：
// 定位符命中运行时码 ⇒ 200 + active，且出站 DTO 带上商品边（deliverable/itemName）。
func TestMarketplaceRoutes_SkillInstallActivates(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_install")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "triage-skill")

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	data := decodeInstallation(t, rec.Body.String())
	assert.Equal(t, "active", data.Status)
	assert.Equal(t, "1.0.0", data.InstalledVersion)
	assert.Equal(t, "triage-skill", data.ItemName)
	assert.Equal(t, "skill", data.ItemType)
	assert.True(t, data.Deliverable, "技能必须报告为可交付，否则市场页会隐藏安装入口")
	assert.Empty(t, data.ErrorMessage)
	assert.NotContains(t, rec.Body.String(), "tenant_id")
}

// TestMarketplaceRoutes_InstallationListCarriesItemDisplayFields 「我的应用」页的商品展示字段
// 必须由安装列表一次查询带出：此前前端读的是 installation.edges.item，而 DTO 里从来没有
// edges，于是每张卡片只剩「应用 #id」、类型过滤失效、连接器配置按钮再也点不到。
// 让界面再拉一次目录来补名称会造出第二份真相，因此这里锁住扁平字段契约。
func TestMarketplaceRoutes_InstallationListCarriesItemDisplayFields(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_installation_display")
	ctx := context.Background()
	item, err := f.client.MarketplaceItem.Create().
		SetName("display-skill").
		SetType(marketplaceitem.TypeSkill).
		SetTitle("工单分诊技能").
		SetDescription("按标题与附件自动分诊").
		SetIconURL("https://cdn.invalid/display-skill.png").
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	addRouteSkillVersion(t, ctx, f.client, item, "1.0.0", "skill://ai.triage", itemversion.StatusPublished)

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/installations", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Items []InstallationResponse `json:"items"`
			Total int                    `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Zero(t, payload.Code, rec.Body.String())
	require.Len(t, payload.Data.Items, 1)
	require.Equal(t, 1, payload.Data.Total)

	installed := payload.Data.Items[0]
	assert.Equal(t, "display-skill", installed.ItemName)
	assert.Equal(t, "工单分诊技能", installed.ItemTitle)
	assert.Equal(t, "按标题与附件自动分诊", installed.ItemDescription)
	assert.Equal(t, "https://cdn.invalid/display-skill.png", installed.ItemIconURL)
	assert.Equal(t, "skill", installed.ItemType)
	assert.True(t, installed.Deliverable)
	assert.Equal(t, "1.0.0", installed.LatestVersion)
	assert.False(t, installed.UpgradeAvailable, "刚装的就是 latest，不应报可升级")

	body := rec.Body.String()
	// 界面只能拿到扁平 DTO：既不能出现 ent 的 edges 形态，也不能拿到版本行定位符。
	assert.NotContains(t, body, "\"edges\"")
	assert.NotContains(t, body, "manifestPath")
	assert.NotContains(t, body, "skill://")
	assert.NotContains(t, body, "tenant_id")
}

// TestMarketplaceRoutes_SkillInstallWithoutLocatorFailsHonest 版本行缺定位符时
// 不能返回 501（那是「平台不交付」），也不能假装装好：记录留 failed + 原因。
func TestMarketplaceRoutes_SkillInstallWithoutLocatorFailsHonest(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_no_locator")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "bare-skill")
	_, err := f.client.ItemVersion.Update().
		Where(itemversion.ItemID(item.ID)).
		ClearManifestPath().
		Save(ctx)
	require.NoError(t, err)

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decodeInstallation(t, rec.Body.String())
	assert.Equal(t, "failed", data.Status)
	assert.Contains(t, data.ErrorMessage, "locator")
	assert.True(t, data.Deliverable, "类型可交付与本次是否装成是两件事，不能混为一谈")

	// 目录里可用的技能码写进了原因：管理员据此判断是补定位符还是先发技能。
	_, err = f.client.ItemVersion.Update().
		Where(itemversion.ItemID(item.ID)).
		SetManifestPath("skill://ai.not-registered").
		Save(ctx)
	require.NoError(t, err)
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data = decodeInstallation(t, rec.Body.String())
	assert.Equal(t, "failed", data.Status)
	assert.Contains(t, data.ErrorMessage, "ai.triage", "报错要列出运行时到底有哪些码")
}

// TestMarketplaceRoutes_UnsupportedTypeStillReturns501 插件仍是 501：
// 「平台不交付」与「交付物没能起来」是两种语义，不能被后者吞掉。
func TestMarketplaceRoutes_UnsupportedTypeStillReturns501(t *testing.T) {
	f := newHandlerFixture(t, "plugin_route_501")
	ctx := context.Background()
	item, err := f.client.MarketplaceItem.Create().
		SetName("legacy-plugin").
		SetType(marketplaceitem.TypePlugin).
		SetTitle("遗留插件").
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusNotImplemented, rec.Code, rec.Body.String())
	var payload common.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, common.NotImplementedCode, payload.Code)
	assert.Nil(t, payload.Data)

	count, err := f.client.TenantInstallation.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "不交付的类型不能留下安装记录")
}

// TestMarketplaceRoutes_UpgradeSkill 升级：目标必须是已发布版本行，成功后换版本号并留审计。
func TestMarketplaceRoutes_UpgradeSkill(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_upgrade")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "upgrade-skill")
	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	addRouteSkillVersion(t, ctx, f.client, item, "1.1.0", "skill://ai.summarize", itemversion.StatusPublished)
	_, err := f.client.MarketplaceItem.UpdateOneID(item.ID).SetLatestVersion("1.1.0").Save(ctx)
	require.NoError(t, err)

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost,
		"/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/upgrade", `{"targetVersion":"1.1.0"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decodeInstallation(t, rec.Body.String())
	assert.Equal(t, "1.1.0", data.InstalledVersion)
	assert.Equal(t, "active", data.Status)
	assert.Empty(t, data.ErrorMessage)

	// 幂等：已在目标版本时是成功而不是错误，界面显示「无需升级」。
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost,
		"/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/upgrade", `{"targetVersion":"1.1.0"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var idempotent struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &idempotent))
	assert.Zero(t, idempotent.Code)
	assert.Contains(t, idempotent.Message, "无需升级")

	// 回滚：退到更旧的已发布版本仍走同一入口。
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost,
		"/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/upgrade", `{"targetVersion":"v1.0.0"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "1.0.0", decodeInstallation(t, rec.Body.String()).InstalledVersion)

	// 审计：升级必须由服务端记录，不能只有版本号变了却查不到是谁升的。
	auditCount, err := f.client.AuditLog.Query().
		Where(auditlog.Resource("marketplace_item:" + strconv.Itoa(item.ID))).
		Count(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, auditCount, 2, "两次成功升级（含回滚）各留一条审计")
}

// TestMarketplaceRoutes_UpgradeRejections 升级的三类拒绝必须可区分：
// 目标不是已发布版本（400）/ 运行态起不来（409，版本不变）/ 依赖没满足（409）/ 未安装（404）/ 越权（404）。
func TestMarketplaceRoutes_UpgradeRejections(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_upgrade_reject")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "reject-skill")
	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	path := "/api/v1/marketplace/items/" + strconv.Itoa(item.ID) + "/upgrade"

	// 草稿版本不是发布物：点名它属于参数错误。
	addRouteSkillVersion(t, ctx, f.client, item, "2.0.0", "skill://ai.triage", itemversion.StatusDraft)
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, path, `{"targetVersion":"2.0.0"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	// 已发布但定位符指向运行时不存在的码：升级被挡，版本保持不变。
	addRouteSkillVersion(t, ctx, f.client, item, "2.1.0", "skill://ai.ghost", itemversion.StatusPublished)
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, path, `{"targetVersion":"2.1.0"}`)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "仍在使用当前版本")
	current := requireInstallation(t, ctx, f.client, 1, item.ID)
	assert.Equal(t, "1.0.0", current.InstalledVersion, "被挡下的升级不得改动版本号")
	assert.Contains(t, current.ErrorMessage, "升级到 2.1.0 失败")

	// 新版本新增前置依赖：升级重新过闸门，不能把悬空依赖装进租户。
	dep := createRouteSkillItem(t, ctx, f.client, "dep-skill")
	addRouteSkillVersion(t, ctx, f.client, item, "2.2.0", "skill://ai.triage", itemversion.StatusPublished)
	_, err := f.client.ItemVersion.Update().
		Where(itemversion.ItemID(item.ID), itemversion.Version("2.2.0")).
		SetDependencies(map[string]string{dep.Name: ">=1.0.0"}).
		Save(ctx)
	require.NoError(t, err)
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, path, `{"targetVersion":"2.2.0"}`)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "未安装")

	// targetVersion 传空串是调用方错误，不能被当成「升到最新」。
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, path, `{"targetVersion":""}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	// 未安装的租户没有可升级的对象。
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, path, "")
	// 空 body 合法（升到 latest），此条只确认 latest 已发布时可 200。
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 2, "super_admin", http.MethodPost, path, `{"targetVersion":"1.0.0"}`)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 1, "marketplace_no_permissions", http.MethodPost, path, `{"targetVersion":"1.0.0"}`)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// 无租户上下文必须失败关闭，而不是落到某个默认租户。
	rec = marketplaceRouteRequest(f, 0, "super_admin", http.MethodPost, path, `{"targetVersion":"1.0.0"}`)
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
}

// TestMarketplaceRoutes_SetAutoUpgrade 自动升级意愿是 PATCH 且必须显式传布尔：
// 缺字段不能被当成 false，否则一次误发就永久关掉自动升级。
func TestMarketplaceRoutes_SetAutoUpgrade(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_auto_upgrade")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "auto-skill")
	path := "/api/v1/marketplace/items/" + strconv.Itoa(item.ID) + "/auto-upgrade"

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodPost, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID)+"/install", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, decodeInstallation(t, rec.Body.String()).AutoUpgrade, "默认开启自动升级")

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPatch, path, `{"autoUpgrade":false}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.False(t, decodeInstallation(t, rec.Body.String()).AutoUpgrade)

	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodPatch, path, `{}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 2, "super_admin", http.MethodPatch, path, `{"autoUpgrade":true}`)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 1, "marketplace_no_permissions", http.MethodPatch, path, `{"autoUpgrade":true}`)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// 关闭后必须留审计：谁同意了这个租户被静默升级（或不再被升级）是故障回溯的必需项。
	auditCount, err := f.client.AuditLog.Query().
		Where(auditlog.Resource("marketplace_item:" + strconv.Itoa(item.ID))).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount, "只在真正改变时写审计；未传字段的 400 不该留痕")
}

// TestMarketplaceRoutes_DependencyStatus 存量依赖对齐：三态可读、按租户隔离。
func TestMarketplaceRoutes_DependencyStatus(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_dependency")
	ctx := context.Background()

	// A：已装组件声明了前置，而前置在本租户从未安装 → unsatisfied
	base := createRouteSkillItem(t, ctx, f.client, "base-skill")
	dependent, err := f.client.MarketplaceItem.Create().
		SetName("dependent-skill").
		SetType(marketplaceitem.TypeSkill).
		SetTitle("依赖方").
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	_, err = f.client.ItemVersion.Create().
		SetItemID(dependent.ID).
		SetVersion("1.0.0").
		SetStatus(itemversion.StatusPublished).
		SetManifestPath("skill://ai.triage").
		SetDependencies(map[string]string{base.Name: ">=1.0.0"}).
		SetReleasedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	// 安装记录用直插而不是 svc.InstallItem：现在安装会被依赖闸门拦住，
	// 而这份读面要呈现的正是闸门上线前已经存在的「装了依赖方、没装前置」的存量数据。
	_, err = f.client.TenantInstallation.Create().
		SetTenantID(1).SetItemID(dependent.ID).SetInstalledVersion("1.0.0").
		SetStatus(tenantinstallation.StatusActive).SetInstalledBy("1001").Save(ctx)
	require.NoError(t, err)

	// C：安装记录指向目录里不存在的版本 → declaration_missing
	orphan, err := f.client.MarketplaceItem.Create().
		SetName("orphan-skill").
		SetType(marketplaceitem.TypeSkill).
		SetTitle("无版本行技能").
		SetProvider("internal").
		SetLatestVersion("").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	_, err = f.client.TenantInstallation.Create().
		SetTenantID(1).SetItemID(orphan.ID).SetInstalledVersion("9.9.9").
		SetStatus(tenantinstallation.StatusActive).SetInstalledBy("1001").Save(ctx)
	require.NoError(t, err)

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/dependency-status", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var payload struct {
		Code int `json:"code"`
		Data struct {
			TenantID           int `json:"tenantId"`
			Total              int `json:"total"`
			Satisfied          int `json:"satisfied"`
			Unsatisfied        int `json:"unsatisfied"`
			DeclarationMissing int `json:"declarationMissing"`
			Findings           []struct {
				ItemName         string   `json:"itemName"`
				Status           string   `json:"status"`
				InstalledVersion string   `json:"installedVersion"`
				Problems         []string `json:"problems"`
			} `json:"findings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Zero(t, payload.Code)
	assert.Equal(t, 1, payload.Data.TenantID)
	assert.Equal(t, 2, payload.Data.Total)
	assert.Equal(t, 1, payload.Data.Unsatisfied)
	assert.Equal(t, 1, payload.Data.DeclarationMissing)
	require.Len(t, payload.Data.Findings, 2)
	byName := map[string]string{}
	for _, finding := range payload.Data.Findings {
		byName[finding.ItemName] = finding.Status
		assert.NotEmpty(t, finding.Problems, "每条缺口都要给出可操作的明细")
	}
	assert.Equal(t, "unsatisfied", byName["dependent-skill"])
	assert.Equal(t, "declaration_missing", byName["orphan-skill"])
	// 契约：只有 camelCase，不能把数据库列名带出去。
	assert.NotContains(t, rec.Body.String(), "installed_version")
	assert.NotContains(t, rec.Body.String(), "tenant_id")

	// 另一个租户看不到租户 1 的安装记录。
	rec = marketplaceRouteRequest(f, 2, "super_admin", http.MethodGet, "/api/v1/marketplace/dependency-status", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Zero(t, payload.Data.Total)
	assert.Equal(t, 2, payload.Data.TenantID)

	rec = marketplaceRouteRequest(f, 0, "super_admin", http.MethodGet, "/api/v1/marketplace/dependency-status", "")
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())

	rec = marketplaceRouteRequest(f, 1, "marketplace_no_permissions", http.MethodGet, "/api/v1/marketplace/dependency-status", "")
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

// TestMarketplaceRoutes_ItemDetailCarriesTenantSafeVersionHistory 详情页的版本历史：
// 只给已发布版本、按发布日期倒序，且不带目录侧的交付细节（manifestPath / downloadUrl /
// configSchema）。界面「版本历史」页签长期读的是从未返回过的 edges.versions，
// 这条契约把它换成真实数据，同时锁住泄漏面。
func TestMarketplaceRoutes_ItemDetailCarriesTenantSafeVersionHistory(t *testing.T) {
	f := newHandlerFixture(t, "skill_route_version_history")
	ctx := context.Background()
	item := createRouteSkillItem(t, ctx, f.client, "history-skill")
	baseReleased := time.Now()
	// 首版建在「此刻」，会把倒序断言变成建表顺序断言：压回最早，
	// 让 versions[0] 只可能由 releasedAt 决定。
	_, err := f.client.ItemVersion.Update().
		Where(itemversion.ItemID(item.ID), itemversion.Version("1.0.0")).
		SetReleasedAt(baseReleased.Add(-3 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 1.0.0 由 createRouteSkillItem 建好（已发布、无 changelog），补两条更晚的已发布版本
	// 和一条草稿：草稿不得进入租户可见历史，否则界面会引导租户去装还没上架的版本。
	for _, v := range []struct {
		version   string
		changelog string
		status    itemversion.Status
		ago       time.Duration
	}{
		{"1.1.0", "修正分诊置信度阈值", itemversion.StatusPublished, 2 * time.Hour},
		{"1.2.0", "支持多语言关键词", itemversion.StatusPublished, time.Hour},
		{"2.0.0", "尚未发布", itemversion.StatusDraft, 30 * time.Minute},
	} {
		_, err := f.client.ItemVersion.Create().
			SetItemID(item.ID).
			SetVersion(v.version).
			SetChangelog(v.changelog).
			SetStatus(v.status).
			SetManifestPath("skill://ai.triage").
			SetDownloadURL("https://release.invalid/history-skill-" + v.version + ".tgz").
			SetConfigSchema(map[string]interface{}{"token": map[string]interface{}{"type": "string"}}).
			SetReleasedAt(baseReleased.Add(-v.ago)).
			Save(ctx)
		require.NoError(t, err)
	}

	rec := marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/items/"+strconv.Itoa(item.ID), "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var payload struct {
		Code int `json:"code"`
		Data struct {
			ID       int  `json:"id"`
			Deliver  bool `json:"deliverable"`
			Versions []struct {
				Version    string    `json:"version"`
				Changelog  string    `json:"changelog"`
				ReleasedAt time.Time `json:"releasedAt"`
			} `json:"versions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Zero(t, payload.Code, rec.Body.String())
	require.Len(t, payload.Data.Versions, 3, "草稿不进历史；已发布的 1.0.0/1.1.0/1.2.0 都要在")
	assert.Equal(t, []string{"1.2.0", "1.1.0", "1.0.0"},
		[]string{payload.Data.Versions[0].Version, payload.Data.Versions[1].Version, payload.Data.Versions[2].Version})
	assert.Equal(t, "支持多语言关键词", payload.Data.Versions[0].Changelog)
	assert.False(t, payload.Data.Versions[2].ReleasedAt.IsZero(), "releasedAt 是界面排序依据，不能为空")
	assert.True(t, payload.Data.Deliver)

	body := rec.Body.String()
	for _, leaked := range []string{"manifest_path", "manifestPath", "download_url", "downloadUrl", "configSchema", "config_schema", "skill://", "release.invalid"} {
		assert.NotContains(t, body, leaked, "目录侧交付细节不得随租户详情外泄")
	}

	// 列表不加载版本边：versions 必须整个缺字段，而不是返回空数组被读成「没有历史」。
	rec = marketplaceRouteRequest(f, 1, "super_admin", http.MethodGet, "/api/v1/marketplace/items?page=1&pageSize=50", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "\"versions\"", "列表不带版本历史；界面缺失即「未加载」")
}

// requireInstallation 读取某租户对某条目的安装记录本体（绕过 DTO，核对真实落库）。
func requireInstallation(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, itemID int) *ent.TenantInstallation {
	t.Helper()
	installation, err := client.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(tenantID), tenantinstallation.ItemID(itemID)).
		Only(ctx)
	require.NoError(t, err)
	return installation
}
