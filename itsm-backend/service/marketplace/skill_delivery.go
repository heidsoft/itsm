package marketplace

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
)

// 技能商品的交付语义（2026-11-03）。
//
// 此前安装面对 type != connector 一律返回 501，目录里却照样能上架 skill 条目：
// 商品页有「立即安装」的坑位，点下去永远装不上，且失败原因写在实现层而不是契约层。
// 现在 skill 走真实交付：
//
//   - 版本行用 manifest_path 指向运行时技能码（`skill://ai.triage`），
//     上架时就必须能解析、且该 code 在 SkillRegistry 里真的存在——
//     与连接器「name 必须命中已注册 provider」是同一条不变量（见 catalog.go）。
//   - 安装 = 把该运行时能力对该租户开通；卸载 = 撤权。
//     技能实现编译在平台进程里，多个租户共享同一份实现，所以卸载不动运行时，
//     只撤销调用权；调用侧按租户强制校验（见 SkillInvocationEntitlement）。
//   - 平台不托管第三方代码：plugin 类型不受理（catalog 侧拒绝上架），
//     这是安全默认而不是待补功能。

// SkillRuntime 是市场侧对技能运行时的最小视图，由 handlers/skill.SkillRegistry 实现。
// 只暴露存在性查询：marketplace 不需要、也不应拿到 Skill 接口去调用执行。
type SkillRuntime interface {
	Has(code string) bool
	Codes() []string
}

var (
	// ErrMarketplaceRuntimeUnavailable 技能运行时未装配（部署问题，不是用户参数问题）。
	ErrMarketplaceRuntimeUnavailable = errors.New("skill runtime is not available")

	// ErrMarketplaceSkillArtifactMissing 技能版本没有可解析的运行时代码定位符。
	// 典型成因：条目由老种子建立、只有 latest_version 没有版本行，或版本行漏填 manifest_path。
	ErrMarketplaceSkillArtifactMissing = errors.New("skill version has no runtime locator")

	// ErrMarketplaceSkillRuntimeMissing 定位符指向的技能码不在运行时注册表里。
	// 与 ErrMarketplaceConnectorNotReady 同族：安装记录与运行态必须对齐，
	// 装不出来的东西不能标成 active。
	ErrMarketplaceSkillRuntimeMissing = errors.New("skill is not present in the runtime registry")

	// ErrMarketplaceVersionUnavailable 目标/已安装版本没有对应的已发布版本行。
	// 技能交付、升级目标、依赖声明都按版本行读取，缺行时不能退化成「按最新版本猜」。
	ErrMarketplaceVersionUnavailable = errors.New("marketplace item has no published version row")

	// ErrMarketplaceConfigUnsupported 该商品类型没有租户级配置面。
	//
	// 属于参数层面的不可受理（400），不是「功能没做」（501）：调用方传的 config
	// 没有任何运行时代码会读，收下只会让界面显示「已保存配置」而行为不变。
	ErrMarketplaceConfigUnsupported = errors.New("marketplace item type has no tenant configuration")
)

// skillLocatorPrefix 定位符前缀。允许读裸码（历史/手填），写一律用带前缀的形式，
// 让「技能交付物」与「下载地址」在同一个字段上仍可读性区分开。
const skillLocatorPrefix = "skill://"

// normalizeSkillLocator 去掉前缀与空白，返回裸技能码；空值返回空串。
func normalizeSkillLocator(raw string) string {
	code := strings.TrimSpace(raw)
	code = strings.TrimPrefix(code, skillLocatorPrefix)
	return strings.TrimSpace(code)
}

// skillLocatorForms 返回一个技能码在 manifest_path 里可能出现的全部写法，
// 供读侧按等值匹配查询（避免用前缀模糊匹配把 `skill://ai.triage-x` 也捞进来）。
func skillLocatorForms(code string) []string {
	return []string{skillLocatorPrefix + code, code}
}

// deliverableTypes 返回平台真正交付的商品类型，供目录校验与对外契约共用。
// 这里是单一事实源：plugin 不在其中，任何「能不能装」的判断都不得另立词表。
func deliverableTypes() []marketplaceitem.Type {
	return []marketplaceitem.Type{marketplaceitem.TypeConnector, marketplaceitem.TypeSkill}
}

// IsDeliverableType 判定商品类型是否由平台交付。
func IsDeliverableType(t marketplaceitem.Type) bool {
	for _, d := range deliverableTypes() {
		if t == d {
			return true
		}
	}
	return false
}

// toItemResponse 出站商品 DTO，并补上交付判定。
//
// deliverable 在这里算而不是在 dto 包里算：交付词表属于业务规则，DTO 包只负责
// 字段搬运。四个出站调用点统一走这对函数，新增接口不会漏掉这个字段。
func toItemResponse(item *ent.MarketplaceItem) *dto.MarketplaceItemResponse {
	if item == nil {
		return nil
	}
	resp := dto.ToMarketplaceItemResponse(item)
	resp.Deliverable = IsDeliverableType(item.Type)
	// 版本历史来自已加载的 versions 边：详情查询带 WithVersions，列表不带则为空。
	// 界面据此渲染「版本历史」，缺边时是「未加载」而不是「没有版本」。
	resp.Versions = dto.ToMarketplaceItemVersionSummaries(item.Edges.Versions)
	return resp
}

// toItemResponses 批量出站，口径同 toItemResponse。
func toItemResponses(items []*ent.MarketplaceItem) []*dto.MarketplaceItemResponse {
	out := make([]*dto.MarketplaceItemResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, toItemResponse(item))
	}
	return out
}

// skillRuntimeCodesForError 给报错用的已注册技能码（字典序，稳定可断言）。
func (s *Service) skillRuntimeCodesForError() string {
	if s.skillRuntime == nil {
		return "（技能运行时未装配）"
	}
	codes := append([]string(nil), s.skillRuntime.Codes()...)
	sort.Strings(codes)
	if len(codes) == 0 {
		return "（运行时暂无已注册技能）"
	}
	return strings.Join(codes, ", ")
}

// resolveSkillCode 校验一个定位符能否交付：运行时必须已装配，且码必须命中实现。
// 未命中时把可用码列出来——管理员下一步是「改定位符」还是「先发布技能」，
// 取决于能否看到当前运行时到底有哪些码。
func (s *Service) resolveSkillCode(locator string) (string, error) {
	code := normalizeSkillLocator(locator)
	if code == "" {
		return "", fmt.Errorf("%w: 技能商品必须用 manifestPath 声明运行时的技能码（当前可用：%s）",
			ErrMarketplaceSkillArtifactMissing, s.skillRuntimeCodesForError())
	}
	if s.skillRuntime == nil {
		return "", fmt.Errorf("%w: 无法校验技能 %q", ErrMarketplaceRuntimeUnavailable, code)
	}
	if !s.skillRuntime.Has(code) {
		return "", fmt.Errorf("%w: %q（当前可用：%s）", ErrMarketplaceSkillRuntimeMissing, code, s.skillRuntimeCodesForError())
	}
	return code, nil
}

// publishedVersionRow 读取某个版本对应的已发布版本行。
//
// 依赖校验的口径不同：依赖缺版本行算「未声明依赖」，交付与升级缺版本行算
// 「无法进行」——技能的运行时代码写在这行里，升级的合法性由这行的 status 决定，
// 没有它就只能靠 latest_version 猜，而 latest_version 正是最容易与实际发布物脱钩的字段。
func (s *Service) publishedVersionRow(ctx context.Context, itemID int, version string) (*ent.ItemVersion, error) {
	if strings.TrimSpace(version) == "" {
		return nil, fmt.Errorf("%w: item=%d 缺少版本号", ErrMarketplaceVersionUnavailable, itemID)
	}
	row, err := s.db.ItemVersion.Query().
		Where(
			itemversion.ItemID(itemID),
			itemversion.Version(version),
			itemversion.StatusEQ(itemversion.StatusPublished),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: item=%d version=%s 没有已发布版本行",
				ErrMarketplaceVersionUnavailable, itemID, version)
		}
		return nil, fmt.Errorf("failed to load published version row: %w", err)
	}
	return row, nil
}

// latestSkillLocator 返回该技能商品最近一个已声明定位符的版本行内容。
//
// 找不到任何声明时返回空串，由调用方决定是报错还是留空——新增版本时
// 这意味着「目录里从没有版本声明过交付物」，管理员必须显式补一次。
func (s *Service) latestSkillLocator(ctx context.Context, itemID int) (string, error) {
	row, err := s.db.ItemVersion.Query().
		Where(
			itemversion.ItemID(itemID),
			itemversion.ManifestPathNEQ(""),
		).
		Order(ent.Desc(itemversion.FieldID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to load inherited skill locator for item %d: %w", itemID, err)
	}
	return strings.TrimSpace(row.ManifestPath), nil
}

// provisionSkill 确认交付物在本进程运行时存在。
//
// 运行实现随平台编译、由 bootstrap 注册进 SkillRegistry，所以这里不创建实例，
// 只做「装不出来就别标 active」的判定——与 provisionConnector 对连接器的作用一致。
func (s *Service) provisionSkill(ctx context.Context, item *ent.MarketplaceItem, version string) error {
	row, err := s.publishedVersionRow(ctx, item.ID, version)
	if err != nil {
		return err
	}
	_, err = s.resolveSkillCode(row.ManifestPath)
	return err
}

// provisionRuntime 按商品类型把「安装记录」对齐到运行态。
//
// 返回的 error 决定安装记录最终停在 failed 还是 active，所以每类失败都要带上
// 稳定的哨兵：handler 按 errors.Is 映射 HTTP 语义，界面按原因给出下一步动作。
// 连接器沿用 ErrMarketplaceConnectorNotReady（运行实例没起来），技能用各自的
// 哨兵（缺定位符 / 码未注册 / 运行时未装配 / 缺版本行），二者不可互相吞并——
// 「缺凭据」和「平台里没有这个技能」是两种完全不同的运维动作。
func (s *Service) provisionRuntime(
	ctx context.Context,
	item *ent.MarketplaceItem,
	tenantID int,
	installedBy string,
	version string,
	config map[string]interface{},
) error {
	if item == nil {
		return fmt.Errorf("%w: 商品为空", ErrMarketplaceDeliveryUnsupported)
	}
	switch item.Type {
	case marketplaceitem.TypeConnector:
		if err := s.provisionConnector(ctx, item, tenantID, installedBy, config); err != nil {
			return fmt.Errorf("%w: %v", ErrMarketplaceConnectorNotReady, err)
		}
		return nil
	case marketplaceitem.TypeSkill:
		return s.provisionSkill(ctx, item, version)
	default:
		return fmt.Errorf("%w: type=%s", ErrMarketplaceDeliveryUnsupported, item.Type)
	}
}

// IsDeliveryNotReady 判定「安装记录已落库、但运行态没能起来」这一类失败。
//
// handler 用它把这些失败与真正的内部错误分开：前者有可展示的 failed 记录
// （含 error_message 和重试入口），返回 200 + 记录比 500 更有信息量；
// 后者没有任何现场可依，只能如实报内部错误。
func IsDeliveryNotReady(err error) bool {
	return errors.Is(err, ErrMarketplaceConnectorNotReady) ||
		errors.Is(err, ErrMarketplaceRuntimeUnavailable) ||
		errors.Is(err, ErrMarketplaceSkillArtifactMissing) ||
		errors.Is(err, ErrMarketplaceSkillRuntimeMissing) ||
		errors.Is(err, ErrMarketplaceVersionUnavailable)
}

// SkillEntitlement 一次技能调用权判定结果。
type SkillEntitlement struct {
	// DeliveredByCatalog 该技能码是否被目录里已上架的技能商品交付；
	//  false 时调用侧不受市场约束（内置能力照旧可用，避免开门即断流）。
	DeliveredByCatalog bool
	// Allowed 当前租户能否调用。
	Allowed bool
	// ItemName 交付该技能的条目名（用于报错文案，指明该去装哪个商品）。
	ItemName string
	// InstalledVersion 生效的安装版本；未安装时为空。
	InstalledVersion string
	// Reason 判定说明（人类可读，界面直接展示）。
	Reason string
}

// SkillInvocationEntitlement 按租户判定某技能码能否被调用。
//
// 判定链：目录里已上架(published) + 类型 skill + 某个已发布版本行的定位符命中该码
// → 该条目对本租户存在 active 安装 → 允许。
// 任何一环查不动（DB 出错）都返回 error，由调用侧按失败关闭处理——
// 「查不到就放行」等于把市场开通做成装饰。
func (s *Service) SkillInvocationEntitlement(ctx context.Context, tenantID int, code string) (SkillEntitlement, error) {
	if strings.TrimSpace(code) == "" {
		return SkillEntitlement{}, fmt.Errorf("技能码不能为空")
	}

	versions, err := s.db.ItemVersion.Query().
		Where(
			itemversion.ManifestPathIn(skillLocatorForms(code)...),
			itemversion.StatusEQ(itemversion.StatusPublished),
			itemversion.HasItemWith(
				marketplaceitem.TypeEQ(marketplaceitem.TypeSkill),
				marketplaceitem.StatusEQ(marketplaceitem.StatusPublished),
			),
		).
		WithItem().
		All(ctx)
	if err != nil {
		return SkillEntitlement{}, fmt.Errorf("failed to query catalog delivery for skill %q: %w", code, err)
	}
	// 只有定位符解析后确实等于该码的版本行才算交付声明：
	// 裸码与带前缀两种写法都从 ManifestPathIn 进来，但等值匹配已保证一致，
	// 这里再过滤一次是为了不把 `skillai.triage` 这类脏值算成命中。
	var itemIDs []int
	var itemName string
	for _, v := range versions {
		if normalizeSkillLocator(v.ManifestPath) != code {
			continue
		}
		item, err := v.Edges.ItemOrErr()
		if err != nil {
			return SkillEntitlement{}, fmt.Errorf("item version %d has no item: %w", v.ID, err)
		}
		itemIDs = append(itemIDs, item.ID)
		if itemName == "" {
			itemName = item.Name
		}
	}
	if len(itemIDs) == 0 {
		return SkillEntitlement{
			DeliveredByCatalog: false,
			Allowed:            true,
			Reason:             "该技能未由商品目录交付，无需安装即可调用",
		}, nil
	}

	installation, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemIDIn(itemIDs...),
			tenantinstallation.StatusEQ(tenantinstallation.StatusActive),
		).
		Order(ent.Desc(tenantinstallation.FieldLastUpdatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return SkillEntitlement{
				DeliveredByCatalog: true,
				Allowed:            false,
				ItemName:           itemName,
				Reason:             fmt.Sprintf("租户尚未安装交付该技能的组件 %s", itemName),
			}, nil
		}
		return SkillEntitlement{}, fmt.Errorf("failed to check installation for skill %q: %w", code, err)
	}
	return SkillEntitlement{
		DeliveredByCatalog: true,
		Allowed:            true,
		ItemName:           itemName,
		InstalledVersion:   installation.InstalledVersion,
		Reason:             fmt.Sprintf("组件 %s 已在租户 %d 生效（%s）", itemName, tenantID, installation.InstalledVersion),
	}, nil
}
