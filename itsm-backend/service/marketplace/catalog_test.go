package marketplace

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"

	// 注册一个真实连接器 provider：目录管理要求连接器商品的 name 必须对应
	// 已注册 provider，测试需要至少一个真实 manifest 才能验证「补齐元数据」
	// 与「版本必须与运行时一致」两条不变量。webhook 是最小且无外部依赖的一个。
	_ "itsm-backend/connector/builtin/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newCatalogService(t *testing.T, dsn string) (*Service, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+dsn+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	return NewService(client, zaptest.NewLogger(t).Sugar()), client
}

func strPtr(v string) *string { return &v }

// 连接器商品不传标题时，从运行时 manifest 补齐：目录描述与代码能力一致，
// 避免「目录写一套、实际能力另一套」的经典漂移。
func TestCreateCatalogItem_ConnectorPrefillsFromManifest(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_create_connector")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.NotNil(t, detail.Item)

	assert.Equal(t, "webhook-connector", detail.Item.Name)
	// 以下三项都来自 connector/builtin/webhook 的 Manifest()。
	assert.Equal(t, "通用 Webhook 出站", detail.Item.Title)
	assert.Equal(t, "generic", detail.Item.Provider)
	assert.Equal(t, "1.0.0", detail.Item.LatestVersion)
	// 条目仍是草稿：能创建 ≠ 已上架。
	assert.Equal(t, string(marketplaceitem.StatusDraft), detail.Item.Status)

	require.Len(t, detail.Versions, 1)
	assert.Equal(t, "1.0.0", detail.Versions[0].Version)
	// 首个版本即 published：它描述的是已存在的运行时代码。
	assert.Equal(t, string(itemversion.StatusPublished), detail.Versions[0].Status)
}

// 连接器商品的 name 必须对应已注册 provider，否则商品能上架、能被安装，
// 但 provision 永远失败，且报错只出现在租户点安装的那一刻。
func TestCreateCatalogItem_RejectsUnregisteredConnector(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_unregistered")
	ctx := context.Background()

	_, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name:  strPtr("ghost-connector"),
		Type:  strPtr("connector"),
		Title: strPtr("幽灵连接器"),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogConnectorUnregistered), "got %v", err)
}

func TestCreateCatalogItem_RejectsInvalidName(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_bad_name")
	ctx := context.Background()

	for _, name := range []string{"Feishu Connector", "-leading", "trailing-", "中文名"} {
		_, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
			Name:    strPtr(name),
			Type:    strPtr("skill"),
			Title:   strPtr("x"),
			Version: strPtr("1.0.0"),
		})
		require.Error(t, err, "name=%q 应被拒绝", name)
		assert.True(t, errors.Is(err, ErrCatalogNameInvalid), "name=%q got %v", name, err)
	}
}

// 非连接器商品没有 manifest 兜底，必须显式给初始版本——否则 latest_version
// 为空，安装记录会写下空版本号。
func TestCreateCatalogItem_NonConnectorRequiresVersion(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_skill_version")
	ctx := context.Background()

	_, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name:     strPtr("excel-export-skill"),
		Type:     strPtr("skill"),
		Title:    strPtr("Excel 导出"),
		Provider: strPtr("internal"),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogVersionInvalid), "got %v", err)

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name:     strPtr("excel-export-skill"),
		Type:     strPtr("skill"),
		Title:    strPtr("Excel 导出"),
		Provider: strPtr("internal"),
		Version:  strPtr("v1.2.3"),
	})
	require.NoError(t, err)
	// v 前缀被归一：v1.2.3 与 1.2.3 在 UNIQUE(item_id,version) 眼里是两行。
	assert.Equal(t, "1.2.3", detail.Item.LatestVersion)
}

// 上架必须有已发布版本，并把 latest_version 对齐到最近发布的那个。
func TestPublishCatalogItem_RequiresPublishedVersion(t *testing.T) {
	svc, client := newCatalogService(t, "catalog_publish_no_version")
	ctx := context.Background()

	item, err := client.MarketplaceItem.Create().
		SetName("no-version-skill").
		SetType(marketplaceitem.TypeSkill).
		SetTitle("无版本技能").
		SetProvider("internal").
		SetLatestVersion("").
		SetStatus(marketplaceitem.StatusDraft).
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.PublishCatalogItem(ctx, item.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogNoPublishedVersion), "got %v", err)
}

// 上架/下架的可见性边界：租户列表只看得见 published，下架不影响已安装租户
// 的卸载路径（条目仍可读），但会挡住新安装。
func TestCatalogPublishUnpublish_Visibility(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_visibility")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)
	itemID := detail.Item.ID

	// 草稿：平台侧可见，租户侧不可见。
	adminItems, _, err := svc.AdminListItems(ctx, CatalogListFilter{})
	require.NoError(t, err)
	assert.Len(t, adminItems, 1)
	tenantItems, _, err := svc.ListItems(ctx, "", "", "", nil, 1, 20)
	require.NoError(t, err)
	assert.Empty(t, tenantItems)

	_, err = svc.PublishCatalogItem(ctx, itemID)
	require.NoError(t, err)
	tenantItems, _, err = svc.ListItems(ctx, "", "", "", nil, 1, 20)
	require.NoError(t, err)
	require.Len(t, tenantItems, 1)
	assert.Equal(t, "1.0.0", tenantItems[0].LatestVersion)

	_, err = svc.UnpublishCatalogItem(ctx, itemID)
	require.NoError(t, err)
	tenantItems, _, err = svc.ListItems(ctx, "", "", "", nil, 1, 20)
	require.NoError(t, err)
	assert.Empty(t, tenantItems, "下架后不应出现在租户市场")

	// 条目本身仍可读：已安装租户的卸载/配置路径依赖它。
	still, err := svc.GetCatalogItem(ctx, itemID)
	require.NoError(t, err)
	assert.Equal(t, string(marketplaceitem.StatusDeprecated), still.Item.Status)

	// 新安装被拒。
	_, err = svc.InstallItem(ctx, 1, itemID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceItemUnavailable), "got %v", err)
}

// name/type 是身份，不可改：name 决定连接器 provider。
func TestUpdateCatalogItem_RejectsIdentityChange(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_immutable")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)

	_, err = svc.UpdateCatalogItem(ctx, detail.Item.ID, CatalogItemPatch{Name: strPtr("other-connector")})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogItemIdentityImmutable), "got %v", err)

	_, err = svc.UpdateCatalogItem(ctx, detail.Item.ID, CatalogItemPatch{Type: strPtr("plugin")})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogItemIdentityImmutable), "got %v", err)

	// 元数据可以改。
	updated, err := svc.UpdateCatalogItem(ctx, detail.Item.ID, CatalogItemPatch{
		Title: strPtr("Webhook 出站（已改名）"),
		Tags:  &[]string{"webhook", "outbound"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Webhook 出站（已改名）", updated.Item.Title)
	assert.Equal(t, []string{"webhook", "outbound"}, updated.Item.Tags)
	// 未提交的字段不被清空。
	assert.Equal(t, "1.0.0", updated.Item.LatestVersion)
}

// 删除必须挡住「已上架」与「已被安装」两种状态，否则会删掉已安装租户的
// 卸载路径。
func TestDeleteCatalogItem_Guards(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_delete_guard")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)
	itemID := detail.Item.ID

	_, err = svc.PublishCatalogItem(ctx, itemID)
	require.NoError(t, err)
	err = svc.DeleteCatalogItem(ctx, itemID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogItemPublished), "got %v", err)

	// 安装一台后下架，再删仍应被拒。
	_, err = svc.InstallItem(ctx, 1, itemID, "1001")
	require.NoError(t, err)
	_, err = svc.UnpublishCatalogItem(ctx, itemID)
	require.NoError(t, err)
	err = svc.DeleteCatalogItem(ctx, itemID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogItemInstalled), "got %v", err)
}

func TestPublishItemVersion_AdvanceAndRollback(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_version_advance")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name:     strPtr("excel-export-skill"),
		Type:     strPtr("skill"),
		Title:    strPtr("Excel 导出"),
		Provider: strPtr("internal"),
		Version:  strPtr("1.0.0"),
	})
	require.NoError(t, err)
	itemID := detail.Item.ID
	_, err = svc.PublishCatalogItem(ctx, itemID)
	require.NoError(t, err)

	// 新版本默认 draft，不影响 latest_version。
	_, err = svc.CreateItemVersion(ctx, itemID, ItemVersionInput{Version: "1.1.0", Changelog: "支持多 sheet"})
	require.NoError(t, err)
	afterDraft, err := svc.GetCatalogItem(ctx, itemID)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", afterDraft.Item.LatestVersion)

	// 发布后推进。
	afterPublish, err := svc.PublishItemVersion(ctx, itemID, "1.1.0")
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", afterPublish.Item.LatestVersion)

	installation, err := svc.InstallItem(ctx, 7, itemID, "1001")
	require.ErrorIs(t, err, ErrMarketplaceDeliveryUnsupported)
	require.Nil(t, installation)

	// 回滚：重新发布 1.0.0。
	rolledBack, err := svc.PublishItemVersion(ctx, itemID, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", rolledBack.Item.LatestVersion)
}

// 连接器版本必须与运行时 manifest 一致：目录声明 9.9.9、实际跑 1.0.0，
// 会让「已安装版本」变成无法核对的说法。
func TestCreateItemVersion_ConnectorMustMatchManifest(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_version_manifest")
	ctx := context.Background()

	detail, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)

	_, err = svc.CreateItemVersion(ctx, detail.Item.ID, ItemVersionInput{Version: "9.9.9"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogVersionInvalid), "got %v", err)

	_, err = svc.CreateItemVersion(ctx, detail.Item.ID, ItemVersionInput{Version: "1.0.0"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCatalogVersionExists), "got %v", err)
}

// 同名商品不能重复创建：name 是唯一标识，也是安装时的 provider 映射键。
func TestCreateCatalogItem_RejectsDuplicateName(t *testing.T) {
	svc, _ := newCatalogService(t, "catalog_duplicate")
	ctx := context.Background()

	_, err := svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.NoError(t, err)
	_, err = svc.CreateCatalogItem(ctx, CatalogItemPatch{
		Name: strPtr("webhook-connector"),
		Type: strPtr("connector"),
	})
	require.Error(t, err)
}
