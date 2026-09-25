package dto

import (
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
)

// MarketplaceItemResponse 是面向 API 的 marketplace item DTO。
// 字段统一 camelCase：snake_case 只允许出现在 ent 列名、SQL 与数据库迁移中。
// 严禁前端或第三方回退到 long_description / install_count / icon_url 等同名字段。
type MarketplaceItemResponse struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"`
	Title              string    `json:"title"`
	Provider           string    `json:"provider"`
	Description        string    `json:"description,omitempty"`
	LongDescription    string    `json:"longDescription,omitempty"`
	IconURL            string    `json:"iconUrl,omitempty"`
	Tags               []string  `json:"tags"`
	Rating             float64   `json:"rating"`
	InstallCount       int       `json:"installCount"`
	LatestVersion      string    `json:"latestVersion"`
	MinSystemVersion   string    `json:"minSystemVersion,omitempty"`
	Status             string    `json:"status"`
	IsOfficial         bool      `json:"isOfficial"`
	IsFree             bool      `json:"isFree"`
	Price              float64   `json:"price"`
	Category           string    `json:"category,omitempty"`
	Capabilities       []string  `json:"capabilities,omitempty"`
	RequiredPerms      []string  `json:"requiredPermissions,omitempty"`
	AuthorID           string    `json:"authorId,omitempty"`
	AuthorName         string    `json:"authorName,omitempty"`
	Homepage           string    `json:"homepage,omitempty"`
	Repository         string    `json:"repository,omitempty"`
	License            string    `json:"license,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
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