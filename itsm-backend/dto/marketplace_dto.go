package dto

import (
	"sort"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
)

// MarketplaceItemResponse 是面向 API 的 marketplace item DTO。
// 字段统一 camelCase：snake_case 只允许出现在 ent 列名、SQL 与数据库迁移中。
// 严禁前端或第三方回退到 long_description / install_count / icon_url 等同名字段。
type MarketplaceItemResponse struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Provider         string   `json:"provider"`
	Description      string   `json:"description,omitempty"`
	LongDescription  string   `json:"longDescription,omitempty"`
	IconURL          string   `json:"iconUrl,omitempty"`
	Screenshots      []string `json:"screenshots,omitempty"`
	Tags             []string `json:"tags"`
	Rating           float64  `json:"rating"`
	InstallCount     int      `json:"installCount"`
	LatestVersion    string   `json:"latestVersion"`
	MinSystemVersion string   `json:"minSystemVersion,omitempty"`
	Status           string   `json:"status"`
	IsOfficial       bool     `json:"isOfficial"`
	IsFree           bool     `json:"isFree"`
	Price            float64  `json:"price"`
	Category         string   `json:"category,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	RequiredPerms    []string `json:"requiredPermissions,omitempty"`
	AuthorID         string   `json:"authorId,omitempty"`
	AuthorName       string   `json:"authorName,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	Repository       string   `json:"repository,omitempty"`
	License          string   `json:"license,omitempty"`
	// Deliverable 平台是否能真正交付该商品（连接器/技能）。
	// 由 service/marketplace 的交付词表计算，前端据此决定「安装」入口是否可用——
	// 界面不得再自己写一份 type 白名单，否则目录放开新类型时前端会静默拦。
	Deliverable bool `json:"deliverable"`
	// Versions 是租户可见的版本历史，只在详情按已发布版本填充。
	// 列表不加载版本边，因此留空（omitempty），界面不得把它读成「没有历史版本」。
	// 这里不复用 MarketplaceItemVersionResponse：目录侧那份带 manifestPath、
	// downloadUrl 与 configSchema，属于平台上架细节，不该随租户详情外泄。
	Versions  []MarketplaceItemVersionSummary `json:"versions,omitempty"`
	CreatedAt time.Time                       `json:"createdAt"`
	UpdatedAt time.Time                       `json:"updatedAt"`
}

// MarketplaceItemVersionSummary 是租户可见的版本条目。
type MarketplaceItemVersionSummary struct {
	Version    string    `json:"version"`
	Changelog  string    `json:"changelog,omitempty"`
	ReleasedAt time.Time `json:"releasedAt"`
}

// ToMarketplaceItemVersionSummaries 过滤出已发布版本并按发布时间倒序。
//
// 草稿版本不进租户视野：它代表「平台还没决定要发」，显示出来等于把
// 目录内部排期当成商品信息。
func ToMarketplaceItemVersionSummaries(versions []*ent.ItemVersion) []MarketplaceItemVersionSummary {
	out := make([]MarketplaceItemVersionSummary, 0, len(versions))
	for _, v := range versions {
		if v == nil || v.Status != itemversion.StatusPublished {
			continue
		}
		out = append(out, MarketplaceItemVersionSummary{
			Version:    v.Version,
			Changelog:  v.Changelog,
			ReleasedAt: v.ReleasedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReleasedAt.After(out[j].ReleasedAt) })
	return out
}

// ToMarketplaceItemResponse 将 ent.MarketplaceItem 转换为 camelCase DTO。
// 严禁 Service/Handler 直接序列化 ent 模型：
// - ent JSON tag 仍为 snake_case（schema legacy 字段），会污染 API 契约；
// - 所有 status 枚举值稳定为小写字符串，与 ListItems 强制的 StatusEQ(StatusPublished) 对齐；
// - 即使 ent 安装 hook 后续把 status 改为大写，也必须先经此 DTO 才进入响应。
func ToMarketplaceItemResponse(item *ent.MarketplaceItem) *MarketplaceItemResponse {
	if item == nil {
		return nil
	}
	return &MarketplaceItemResponse{
		ID:               item.ID,
		Name:             item.Name,
		Type:             string(item.Type),
		Title:            item.Title,
		Provider:         item.Provider,
		Description:      item.Description,
		LongDescription:  item.LongDescription,
		IconURL:          item.IconURL,
		Screenshots:      item.Screenshots,
		Tags:             item.Tags,
		Rating:           item.Rating,
		InstallCount:     item.InstallCount,
		LatestVersion:    item.LatestVersion,
		MinSystemVersion: item.MinSystemVersion,
		Status:           string(item.Status),
		IsOfficial:       item.IsOfficial,
		IsFree:           item.IsFree,
		Price:            item.Price,
		Category:         item.Category,
		Capabilities:     item.Capabilities,
		RequiredPerms:    item.RequiredPermissions,
		AuthorID:         item.AuthorID,
		AuthorName:       item.AuthorName,
		Homepage:         item.Homepage,
		Repository:       item.Repository,
		License:          item.License,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

// ToMarketplaceItemResponseList 批量转换。
func ToMarketplaceItemResponseList(items []*ent.MarketplaceItem) []*MarketplaceItemResponse {
	if items == nil {
		return nil
	}
	out := make([]*MarketplaceItemResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, ToMarketplaceItemResponse(item))
	}
	return out
}

// _ 确保引用 marketplaceitem 包，避免 ent 端 import 删除后该 DTO 文件被静默遗漏。
var _ = marketplaceitem.StatusPublished

// MarketplaceItemVersionResponse 是商品版本的出站 DTO。
//
// 版本是目录管理的主要操作对象：上架新版本才会推进 item.latest_version，
// 而 latest_version 又决定新安装记录的 installed_version。因此版本列表
// 必须包含 status 与 releasedAt——「哪个版本现在是线上」只能由这两个字段
// 判断，缺一个就只能靠猜。
type MarketplaceItemVersionResponse struct {
	Version          string                 `json:"version"`
	Changelog        string                 `json:"changelog,omitempty"`
	DownloadURL      string                 `json:"downloadUrl,omitempty"`
	ManifestPath     string                 `json:"manifestPath,omitempty"`
	MinSystemVersion string                 `json:"minSystemVersion,omitempty"`
	Dependencies     map[string]string      `json:"dependencies,omitempty"`
	Status           string                 `json:"status"`
	DownloadCount    int                    `json:"downloadCount"`
	ReleasedAt       time.Time              `json:"releasedAt"`
	ConfigSchema     map[string]interface{} `json:"configSchema,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// ToMarketplaceItemVersionResponse 转换单个版本。
func ToMarketplaceItemVersionResponse(v *ent.ItemVersion) *MarketplaceItemVersionResponse {
	if v == nil {
		return nil
	}
	return &MarketplaceItemVersionResponse{
		Version:          v.Version,
		Changelog:        v.Changelog,
		DownloadURL:      v.DownloadURL,
		ManifestPath:     v.ManifestPath,
		MinSystemVersion: v.MinSystemVersion,
		Dependencies:     v.Dependencies,
		Status:           string(v.Status),
		DownloadCount:    v.DownloadCount,
		ReleasedAt:       v.ReleasedAt,
		ConfigSchema:     v.ConfigSchema,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

// ToMarketplaceItemVersionResponseList 批量转换。
func ToMarketplaceItemVersionResponseList(items []*ent.ItemVersion) []*MarketplaceItemVersionResponse {
	out := make([]*MarketplaceItemVersionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, ToMarketplaceItemVersionResponse(item))
	}
	return out
}

// MarketplaceDependencyResponse 是单条已安装组件的依赖对齐结果。
//
// status 三态：ok / unsatisfied / declaration_missing。第三态不能并进「没问题」——
// 目录里缺版本行的商品没有依赖声明可查，界面若显示健康就把最需要先修的存量数据藏掉了。
type MarketplaceDependencyResponse struct {
	ItemID             int               `json:"itemId"`
	ItemName           string            `json:"itemName"`
	ItemTitle          string            `json:"itemTitle"`
	ItemType           string            `json:"itemType"`
	Status             string            `json:"status"`
	InstalledVersion   string            `json:"installedVersion"`
	InstallationStatus string            `json:"installationStatus"`
	Dependencies       map[string]string `json:"dependencies,omitempty"`
	Problems           []string          `json:"problems,omitempty"`
}

// MarketplaceDependencyReportResponse 是租户存量依赖对齐报告。
type MarketplaceDependencyReportResponse struct {
	TenantID           int                              `json:"tenantId"`
	CheckedAt          time.Time                        `json:"checkedAt"`
	Total              int                              `json:"total"`
	Satisfied          int                              `json:"satisfied"`
	Unsatisfied        int                              `json:"unsatisfied"`
	DeclarationMissing int                              `json:"declarationMissing"`
	Findings           []*MarketplaceDependencyResponse `json:"findings"`
}
