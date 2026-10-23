package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
)

// 目录（商品）管理面：平台运营维护全局商品目录。
//
// 与安装面（InstallItem 等）的边界：安装是租户动作（marketplace:write），
// 目录是平台动作（marketplace_catalog:write）。marketplace_items 没有
// tenant_id，所有租户共享同一份目录，所以目录写必须是平台级权限——否则任一
// 租户管理员都能改写别人看到、安装的条目。
//
// 本文件的核心不是 CRUD，而是几条让「目录内容」与「运行态」不脱钩的不变量：
//  1. name 是身份：创建后不可改。安装时 name 决定连接器 provider（去掉
//     -connector 后缀），改名等于让已安装的租户指向另一个 provider。
//  2. 连接器类商品的 name 必须对应一个已注册 provider。否则商品能上架、
//     能被安装，但 provision 永远失败——这类错误只有在租户点安装时才暴露，
//     且文案是「connector not registered」，与目录编辑毫无关联。
//  3. latest_version 只能由「发布某个版本」派生，不能手填。手填会出现
//     latest_version=2.0.0 而版本表里只有 1.0.0 的情况：安装记录写上
//     2.0.0，后续升级/回滚逻辑全部对不上。
//  4. 下架用 deprecated 而不是删除：已安装租户的卸载路径仍需要读该条目。

var (
	ErrCatalogNameInvalid           = errors.New("marketplace catalog name is invalid")
	ErrCatalogItemIdentityImmutable = errors.New("marketplace catalog item name/type is immutable")
	ErrCatalogConnectorUnregistered = errors.New("marketplace catalog connector is not registered")
	ErrCatalogVersionInvalid        = errors.New("marketplace catalog version is invalid")
	ErrCatalogVersionExists         = errors.New("marketplace catalog version already exists")
	ErrCatalogVersionNotFound       = errors.New("marketplace catalog version not found")
	ErrCatalogNoPublishedVersion    = errors.New("marketplace catalog item has no published version")
	ErrCatalogItemPublished         = errors.New("marketplace catalog item is published")
	ErrCatalogItemInstalled         = errors.New("marketplace catalog item has tenant installations")
	ErrCatalogFieldRequired         = errors.New("marketplace catalog field is required")
)

// catalogNamePattern 商品标识：小写字母数字与连字符，不能以连字符开头/结尾。
// 该值会作为连接器 provider 使用，必须与 connector 注册名同形。
var catalogNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// catalogVersionPattern 语义化版本（允许 -rc.1 这类预发布后缀）。
// 允许 v 前缀但入库前统一去掉：v1.0.0 与 1.0.0 在 UNIQUE(item_id, version)
// 眼里是两个版本，会让「同一版本」在目录里出现两行。
var catalogVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// CatalogItemPatch 目录项创建/更新入参。
//
// 全部字段可空：更新是 PATCH 语义，nil 表示「不动这个字段」。创建时由
// CreateCatalogItem 单独校验必填项（name/type/title/provider）。
type CatalogItemPatch struct {
	Name             *string
	Type             *string
	Title            *string
	Provider         *string
	Description      *string
	LongDescription  *string
	IconURL          *string
	Screenshots      *[]string
	Tags             *[]string
	Category         *string
	Capabilities     *[]string
	RequiredPerms    *[]string
	ConfigSchema     *map[string]interface{}
	IsOfficial       *bool
	IsFree           *bool
	Price            *float64
	AuthorName       *string
	Homepage         *string
	Repository       *string
	License          *string
	MinSystemVersion *string
	// Version 仅创建时使用：初始版本号。连接器商品可省略（取 manifest 版本）。
	Version *string
}

// CatalogListFilter 目录列表筛选（与租户侧 ListItems 不同，这里包含非 published）。
type CatalogListFilter struct {
	Status   string
	Type     string
	Search   string
	Page     int
	PageSize int
}

// CatalogItemDetail 目录项详情：条目 + 版本列表。
// 版本是目录管理的主要操作对象（发布新版本才会推进 latest_version），
// 所以详情一次带出，避免页面先拿条目再补一次请求。
type CatalogItemDetail struct {
	Item     *dto.MarketplaceItemResponse
	Versions []*dto.MarketplaceItemVersionResponse
}

// AdminListItems 平台侧目录列表：包含 draft/reviewing/deprecated。
//
// 租户侧 ListItems 强制 StatusEQ(published)，草稿因此对租户不可见——这也
// 意味着目录管理必须走独立方法，而不是给 ListItems 加一个 includeDrafts 开关：
// 那个开关一旦被 HTTP 层漏传/误传，草稿商品就直接出现在租户市场里。
func (s *Service) AdminListItems(ctx context.Context, filter CatalogListFilter) ([]*dto.MarketplaceItemResponse, int, error) {
	query := s.db.MarketplaceItem.Query()

	if filter.Status != "" {
		query = query.Where(marketplaceitem.StatusEQ(marketplaceitem.Status(filter.Status)))
	}
	if filter.Type != "" {
		query = query.Where(marketplaceitem.TypeEQ(marketplaceitem.Type(filter.Type)))
	}
	if filter.Search != "" {
		query = query.Where(marketplaceitem.Or(
			marketplaceitem.NameContains(filter.Search),
			marketplaceitem.TitleContains(filter.Search),
			marketplaceitem.ProviderContains(filter.Search),
		))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count catalog items: %w", err)
	}

	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	items, err := query.
		Order(ent.Desc(marketplaceitem.FieldUpdatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list catalog items: %w", err)
	}
	return dto.ToMarketplaceItemResponseList(items), total, nil
}

// GetCatalogItem 目录项详情（含草稿），附带全部版本。
func (s *Service) GetCatalogItem(ctx context.Context, itemID int) (*CatalogItemDetail, error) {
	item, err := s.db.MarketplaceItem.Query().
		Where(marketplaceitem.ID(itemID)).
		WithVersions().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	versions := item.Edges.Versions
	sortVersionsByRelease(versions)
	return &CatalogItemDetail{
		Item:     dto.ToMarketplaceItemResponse(item),
		Versions: dto.ToMarketplaceItemVersionResponseList(versions),
	}, nil
}

// CreateCatalogItem 创建目录项（初始状态 draft，不对外可见）。
//
// 连接器类商品会从已注册的 provider manifest 补齐元数据：标题、描述、图标、
// 能力、配置 Schema、最低系统版本都以运行时代码为准。这样「上架一个已存在的
// 连接器」只需 name + type，不会出现目录描述与实际能力不符的经典漂移。
func (s *Service) CreateCatalogItem(ctx context.Context, patch CatalogItemPatch) (*CatalogItemDetail, error) {
	name := strings.TrimSpace(derefString(patch.Name))
	if !catalogNamePattern.MatchString(name) {
		return nil, fmt.Errorf("%w: %q 必须是小写字母/数字/连字符，且不以连字符开头或结尾", ErrCatalogNameInvalid, name)
	}
	itemType := marketplaceitem.Type(strings.TrimSpace(derefString(patch.Type)))
	if !validCatalogType(itemType) {
		return nil, fmt.Errorf("%w: type 必须是 connector/skill/plugin", ErrCatalogFieldRequired)
	}
	// 先校验 provider 是否已注册：未注册时后面的 manifest 补齐会静默跳过，
	// 报错会退化成「title 必填」这种与真实原因无关的提示。
	if err := validateConnectorProvider(name, itemType); err != nil {
		return nil, err
	}
	manifest, hasManifest := catalogConnectorManifest(name, itemType)

	title := strings.TrimSpace(derefString(patch.Title))
	provider := strings.TrimSpace(derefString(patch.Provider))
	if hasManifest {
		if title == "" {
			title = manifest.Title
		}
		if provider == "" {
			provider = manifest.Provider
		}
	}
	if title == "" {
		return nil, fmt.Errorf("%w: title", ErrCatalogFieldRequired)
	}
	if provider == "" {
		return nil, fmt.Errorf("%w: provider", ErrCatalogFieldRequired)
	}
	version, err := initialCatalogVersion(patch, hasManifest, manifest)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	create := tx.MarketplaceItem.Create().
		SetName(name).
		SetType(itemType).
		SetTitle(title).
		SetProvider(provider).
		SetLatestVersion(version).
		SetStatus(marketplaceitem.StatusDraft)
	if hasManifest {
		// manifest 是运行时代码的自我描述，优先级低于显式入参、高于空值。
		create = applyManifestDefaults(create, manifest)
	}
	create = applyCreatePatch(create, patch)

	// 事务：条目与首个版本必须同时存在。否则条目 latest_version 指向一个
	// 不存在的版本，PublishCatalogItem 的「必须有已发布版本」永远不成立。
	item, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: name=%q 已存在", ErrCatalogNameInvalid, name)
		}
		return nil, fmt.Errorf("failed to create catalog item: %w", err)
	}
	// 首个版本直接标记 published：它描述的是已经存在的运行时代码/发布物，
	// 不是待审内容。条目本身仍是 draft，上架是另一个显式动作。
	if _, err := tx.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion(version).
		SetStatus(itemversion.StatusPublished).
		SetReleasedAt(time.Now()).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: version=%q", ErrCatalogVersionExists, version)
		}
		return nil, fmt.Errorf("failed to create initial version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}
	return s.GetCatalogItem(ctx, item.ID)
}

// UpdateCatalogItem 更新目录项元数据。
//
// name/type 不可变：name 决定连接器 provider，改名会让已安装租户指向另一个
// provider；type 决定安装时的副作用路径（connector 才会 provision）。
// latest_version 也不接受直接写入——它只能由发布版本派生。
func (s *Service) UpdateCatalogItem(ctx context.Context, itemID int, patch CatalogItemPatch) (*CatalogItemDetail, error) {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	if patch.Name != nil && strings.TrimSpace(*patch.Name) != item.Name {
		return nil, fmt.Errorf("%w: name 不可修改（安装时决定连接器 provider）", ErrCatalogItemIdentityImmutable)
	}
	if patch.Type != nil && marketplaceitem.Type(*patch.Type) != item.Type {
		return nil, fmt.Errorf("%w: type 不可修改（决定安装副作用路径）", ErrCatalogItemIdentityImmutable)
	}
	if patch.Version != nil {
		return nil, fmt.Errorf("%w: 版本只能通过版本接口维护", ErrCatalogVersionInvalid)
	}

	update := applyUpdatePatch(s.db.MarketplaceItem.UpdateOneID(itemID), patch)
	if _, err := update.Save(ctx); err != nil {
		return nil, fmt.Errorf("failed to update catalog item: %w", err)
	}
	return s.GetCatalogItem(ctx, itemID)
}

// PublishCatalogItem 上架：draft/rejected/deprecated → published。
//
// 前提是至少有一个已发布版本，并把 latest_version 对齐到「最近发布的那个」。
// 没有这个前提，条目会带着指向不存在版本的 latest_version 上架，安装记录
// 里写下一个查不到的版本号。
func (s *Service) PublishCatalogItem(ctx context.Context, itemID int) (*CatalogItemDetail, error) {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	if item.Status == marketplaceitem.StatusPublished {
		return s.GetCatalogItem(ctx, itemID)
	}

	latest, err := s.latestPublishedVersion(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if latest == "" {
		return nil, ErrCatalogNoPublishedVersion
	}

	if err := s.db.MarketplaceItem.UpdateOneID(itemID).
		SetStatus(marketplaceitem.StatusPublished).
		SetLatestVersion(latest).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to publish catalog item: %w", err)
	}
	s.logger.Infow("marketplace catalog item published", "item_id", itemID, "name", item.Name, "latest_version", latest)
	return s.GetCatalogItem(ctx, itemID)
}

// UnpublishCatalogItem 下架：published → deprecated。
//
// 不删除：已安装租户的卸载/配置路径仍要读该条目，删了会让卸载 404。
// 下架只影响「新安装」——InstallItem 要求 status=published。
func (s *Service) UnpublishCatalogItem(ctx context.Context, itemID int) (*CatalogItemDetail, error) {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	if item.Status != marketplaceitem.StatusPublished {
		return s.GetCatalogItem(ctx, itemID)
	}
	if err := s.db.MarketplaceItem.UpdateOneID(itemID).
		SetStatus(marketplaceitem.StatusDeprecated).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to unpublish catalog item: %w", err)
	}
	s.logger.Infow("marketplace catalog item unpublished", "item_id", itemID, "name", item.Name)
	return s.GetCatalogItem(ctx, itemID)
}

// DeleteCatalogItem 删除目录项（仅限从未被安装、且未上架的条目）。
//
// 已上架必须先下架：让「删除」无法绕过下架这一步，避免有人用删除来达成下架，
// 结果把已安装租户的卸载路径一起删掉。
func (s *Service) DeleteCatalogItem(ctx context.Context, itemID int) error {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrMarketplaceItemNotFound
		}
		return fmt.Errorf("failed to get catalog item: %w", err)
	}
	if item.Status == marketplaceitem.StatusPublished {
		return ErrCatalogItemPublished
	}
	installed, err := s.db.TenantInstallation.Query().
		Where(tenantinstallation.ItemID(itemID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count installations: %w", err)
	}
	if installed > 0 {
		return fmt.Errorf("%w: %d", ErrCatalogItemInstalled, installed)
	}

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	if _, err := tx.ItemVersion.Delete().Where(itemversion.ItemID(itemID)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete versions: %w", err)
	}
	if err := tx.MarketplaceItem.DeleteOneID(itemID).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete catalog item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	s.logger.Infow("marketplace catalog item deleted", "item_id", itemID, "name", item.Name)
	return nil
}

// ListItemVersions 列出版本（按发布时间倒序）。
func (s *Service) ListItemVersions(ctx context.Context, itemID int) ([]*dto.MarketplaceItemVersionResponse, error) {
	if _, err := s.db.MarketplaceItem.Get(ctx, itemID); err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	versions, err := s.db.ItemVersion.Query().
		Where(itemversion.ItemID(itemID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	sortVersionsByRelease(versions)
	return dto.ToMarketplaceItemVersionResponseList(versions), nil
}

// ItemVersionInput 新版本入参。
type ItemVersionInput struct {
	Version          string
	Changelog        string
	DownloadURL      string
	ManifestPath     string
	MinSystemVersion string
	Dependencies     map[string]string
	ConfigSchema     map[string]interface{}
	// Publish 为 true 时同时发布该版本（并把 latest_version 推进到它）。
	Publish bool
}

// CreateItemVersion 新增版本。
//
// 默认 draft：新版本不该在「发布」动作之前影响任何租户。Publish=true 是
// 便捷路径（上架即生效），内部走与 PublishItemVersion 相同的推进逻辑。
func (s *Service) CreateItemVersion(ctx context.Context, itemID int, in ItemVersionInput) (*dto.MarketplaceItemVersionResponse, error) {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	version, err := normalizeCatalogVersion(in.Version)
	if err != nil {
		return nil, err
	}
	if item.Type == marketplaceitem.TypeConnector {
		// 连接器版本号必须与代码 manifest 一致：目录写 9.9.9、实际跑的是 1.2.0，
		// 会让「已安装版本」变成无法核对的说法。
		if manifest, ok := catalogConnectorManifest(item.Name, item.Type); ok {
			manifestVersion, _ := normalizeCatalogVersion(manifest.Version)
			if manifestVersion != version {
				return nil, fmt.Errorf("%w: 连接器 %s 的运行时版本是 %s，目录不能声明 %s",
					ErrCatalogVersionInvalid, item.Name, manifestVersion, version)
			}
		}
	}

	status := itemversion.StatusDraft
	if in.Publish {
		status = itemversion.StatusPublished
	}
	create := s.db.ItemVersion.Create().
		SetItemID(itemID).
		SetVersion(version).
		SetStatus(status).
		SetChangelog(strings.TrimSpace(in.Changelog)).
		SetDownloadURL(strings.TrimSpace(in.DownloadURL)).
		SetManifestPath(strings.TrimSpace(in.ManifestPath)).
		SetMinSystemVersion(strings.TrimSpace(in.MinSystemVersion))
	if len(in.Dependencies) > 0 {
		create = create.SetDependencies(in.Dependencies)
	}
	if len(in.ConfigSchema) > 0 {
		create = create.SetConfigSchema(in.ConfigSchema)
	}
	if in.Publish {
		create = create.SetReleasedAt(time.Now())
	}
	created, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: version=%q", ErrCatalogVersionExists, version)
		}
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	if in.Publish {
		if err := s.advanceLatestVersion(ctx, itemID, version); err != nil {
			return nil, err
		}
	}
	return dto.ToMarketplaceItemVersionResponse(created), nil
}

// PublishItemVersion 发布指定版本，并把 latest_version 推进到它。
//
// 允许发布「比当前更旧」的版本：那就是回滚。latest_version 以发布动作为准，
// 而不是按版本号比大小——按大小推导会让回滚变成不可能。
func (s *Service) PublishItemVersion(ctx context.Context, itemID int, rawVersion string) (*CatalogItemDetail, error) {
	if _, err := s.db.MarketplaceItem.Get(ctx, itemID); err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	version, err := normalizeCatalogVersion(rawVersion)
	if err != nil {
		return nil, err
	}
	target, err := s.db.ItemVersion.Query().
		Where(itemversion.ItemID(itemID), itemversion.Version(version)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: version=%q", ErrCatalogVersionNotFound, version)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	// 已发布版本允许再次「发布」：那就是回滚。把 latest_version 拉回旧版本
	// 是运维最需要的动作之一，若在这里报错，回滚就只能改数据库。
	if target.Status != itemversion.StatusPublished {
		if _, err := s.db.ItemVersion.UpdateOneID(target.ID).
			SetStatus(itemversion.StatusPublished).
			SetReleasedAt(time.Now()).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("failed to publish version: %w", err)
		}
	}
	if err := s.advanceLatestVersion(ctx, itemID, version); err != nil {
		return nil, err
	}
	s.logger.Infow("marketplace catalog version published", "item_id", itemID, "version", version)
	return s.GetCatalogItem(ctx, itemID)
}

// advanceLatestVersion 把条目的 latest_version 推进到指定版本。
//
// 只对已上架条目生效：draft 条目的 latest_version 在 PublishCatalogItem 时
// 统一对齐，这里提前改会让「未上架却指向某个版本」的状态出现两种解释。
func (s *Service) advanceLatestVersion(ctx context.Context, itemID int, version string) error {
	if err := s.db.MarketplaceItem.UpdateOneID(itemID).
		Where(marketplaceitem.StatusEQ(marketplaceitem.StatusPublished)).
		SetLatestVersion(version).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to advance latest version: %w", err)
	}
	return nil
}

// latestPublishedVersion 返回最近发布的版本号；无则空串。
func (s *Service) latestPublishedVersion(ctx context.Context, itemID int) (string, error) {
	latest, err := s.db.ItemVersion.Query().
		Where(itemversion.ItemID(itemID), itemversion.StatusEQ(itemversion.StatusPublished)).
		Order(ent.Desc(itemversion.FieldReleasedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to query latest version: %w", err)
	}
	return latest.Version, nil
}

// initialCatalogVersion 决定创建时的首个版本号。
func initialCatalogVersion(patch CatalogItemPatch, hasManifest bool, manifest connector.Manifest) (string, error) {
	if explicit := strings.TrimSpace(derefString(patch.Version)); explicit != "" {
		return normalizeCatalogVersion(explicit)
	}
	if hasManifest {
		if v, err := normalizeCatalogVersion(manifest.Version); err == nil {
			return v, nil
		}
	}
	return "", fmt.Errorf("%w: 非连接器商品必须提供初始版本号", ErrCatalogVersionInvalid)
}

// catalogConnectorManifest 解析连接器商品对应的运行时 manifest。
//
// 命名口径与 provisionConnector 一致：条目名可带 -connector 后缀
// （seeder 的 feishu-connector），安装时按去掉后缀的名字找 provider。
// 这里用同一规则校验，保证「能上架的商品」= 「能 provision 的 provider」。
func catalogConnectorManifest(name string, itemType marketplaceitem.Type) (connector.Manifest, bool) {
	if itemType != marketplaceitem.TypeConnector {
		return connector.Manifest{}, false
	}
	provider := strings.TrimSuffix(name, "-connector")
	manifest, ok := connector.Default().GetManifest(provider)
	return manifest, ok
}

// validateConnectorProvider 校验连接器商品的 provider 是否存在。
//
// 报错里带上已注册清单：调用方看到「未注册」时下一步动作是「改名字」还是
// 「先注册 provider」，取决于能看到哪些名字可用。
func validateConnectorProvider(name string, itemType marketplaceitem.Type) error {
	if itemType != marketplaceitem.TypeConnector {
		return nil
	}
	provider := strings.TrimSuffix(name, "-connector")
	if _, ok := connector.Default().GetManifest(provider); !ok {
		return fmt.Errorf("%w: %q（已注册：%s）", ErrCatalogConnectorUnregistered, provider,
			strings.Join(connector.Default().Names(), ", "))
	}
	return nil
}

func validCatalogType(t marketplaceitem.Type) bool {
	switch t {
	case marketplaceitem.TypeConnector, marketplaceitem.TypeSkill, marketplaceitem.TypePlugin:
		return true
	}
	return false
}

// applyCreatePatch 把 patch 应用到新建条目。
//
// ent 的 Create/UpdateOne 没有共享的 builder 接口（SetNillableX 返回各自类型），
// 所以这里保留两份几乎相同的函数。合并成反射/泛型会让「哪些字段可写」变得
// 不可 grep，代价高于这点重复。
func applyCreatePatch(create *ent.MarketplaceItemCreate, patch CatalogItemPatch) *ent.MarketplaceItemCreate {
	create = create.
		SetNillableDescription(patch.Description).
		SetNillableLongDescription(patch.LongDescription).
		SetNillableIconURL(patch.IconURL).
		SetNillableCategory(patch.Category).
		SetNillableIsOfficial(patch.IsOfficial).
		SetNillableIsFree(patch.IsFree).
		SetNillablePrice(patch.Price).
		SetNillableAuthorName(patch.AuthorName).
		SetNillableHomepage(patch.Homepage).
		SetNillableRepository(patch.Repository).
		SetNillableLicense(patch.License).
		SetNillableMinSystemVersion(patch.MinSystemVersion)
	// JSON 字段（screenshots/tags/capabilities/required_permissions/config_schema）
	// ent 不生成 SetNillable*，必须显式判空，否则「不传」会被写成空数组。
	if patch.Screenshots != nil {
		create = create.SetScreenshots(*patch.Screenshots)
	}
	if patch.Tags != nil {
		create = create.SetTags(*patch.Tags)
	}
	if patch.Capabilities != nil {
		create = create.SetCapabilities(*patch.Capabilities)
	}
	if patch.RequiredPerms != nil {
		create = create.SetRequiredPermissions(*patch.RequiredPerms)
	}
	if patch.ConfigSchema != nil {
		create = create.SetConfigSchema(*patch.ConfigSchema)
	}
	return create
}

// applyUpdatePatch 把 patch 应用到已有条目（与 applyCreatePatch 字段集一致）。
func applyUpdatePatch(update *ent.MarketplaceItemUpdateOne, patch CatalogItemPatch) *ent.MarketplaceItemUpdateOne {
	update = update.
		SetNillableTitle(patch.Title).
		SetNillableProvider(patch.Provider).
		SetNillableDescription(patch.Description).
		SetNillableLongDescription(patch.LongDescription).
		SetNillableIconURL(patch.IconURL).
		SetNillableCategory(patch.Category).
		SetNillableIsOfficial(patch.IsOfficial).
		SetNillableIsFree(patch.IsFree).
		SetNillablePrice(patch.Price).
		SetNillableAuthorName(patch.AuthorName).
		SetNillableHomepage(patch.Homepage).
		SetNillableRepository(patch.Repository).
		SetNillableLicense(patch.License).
		SetNillableMinSystemVersion(patch.MinSystemVersion)
	if patch.Screenshots != nil {
		update = update.SetScreenshots(*patch.Screenshots)
	}
	if patch.Tags != nil {
		update = update.SetTags(*patch.Tags)
	}
	if patch.Capabilities != nil {
		update = update.SetCapabilities(*patch.Capabilities)
	}
	if patch.RequiredPerms != nil {
		update = update.SetRequiredPermissions(*patch.RequiredPerms)
	}
	if patch.ConfigSchema != nil {
		update = update.SetConfigSchema(*patch.ConfigSchema)
	}
	return update
}

// applyManifestDefaults 用 manifest 填充未显式提供的元数据。
func applyManifestDefaults(create *ent.MarketplaceItemCreate, manifest connector.Manifest) *ent.MarketplaceItemCreate {
	if manifest.Description != "" {
		create = create.SetDescription(manifest.Description)
	}
	if manifest.IconURL != "" {
		create = create.SetIconURL(manifest.IconURL)
	}
	if len(manifest.Tags) > 0 {
		create = create.SetTags(manifest.Tags)
	}
	if manifest.Category != "" {
		create = create.SetCategory(manifest.Category)
	}
	if manifest.MinITSMVer != "" {
		create = create.SetMinSystemVersion(manifest.MinITSMVer)
	}
	if len(manifest.Capabilities) > 0 {
		caps := make([]string, 0, len(manifest.Capabilities))
		for _, c := range manifest.Capabilities {
			caps = append(caps, string(c))
		}
		create = create.SetCapabilities(caps)
	}
	if len(manifest.RequiredPermissions) > 0 {
		create = create.SetRequiredPermissions(manifest.RequiredPermissions)
	}
	if manifest.ConfigSchema != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(manifest.ConfigSchema), &schema); err == nil {
			create = create.SetConfigSchema(schema)
		}
	}
	return create
}

func normalizeCatalogVersion(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if !catalogVersionPattern.MatchString(v) {
		return "", fmt.Errorf("%w: %q 必须是 x.y.z（可带 -预发布后缀）", ErrCatalogVersionInvalid, raw)
	}
	return v, nil
}

func sortVersionsByRelease(versions []*ent.ItemVersion) {
	// 最近发布的在前；未发布的（零值 ReleasedAt）排在后面。
	for i := 1; i < len(versions); i++ {
		for j := i; j > 0 && versions[j].ReleasedAt.After(versions[j-1].ReleasedAt); j-- {
			versions[j], versions[j-1] = versions[j-1], versions[j]
		}
	}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
