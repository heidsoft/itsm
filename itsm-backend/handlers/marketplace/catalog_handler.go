package marketplace

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
)

// 目录管理 HTTP 面（平台运营）。
//
// 与安装面共用同一个 Handler，但挂在不同前缀与不同权限码下：
//   - /marketplace/items/*        → marketplace:write（租户安装/卸载）
//   - /marketplace/catalog/items* → marketplace_catalog:write（平台上架/下架）
//
// 分开前缀而不是靠权限码区分，是为了让「谁在改全局目录」在访问日志里一眼可见。

// catalogItemRequest 目录项创建/更新入参。
//
// 全部字段用指针：PATCH 语义下 nil 表示「不改」，与「改成空串」必须可区分。
// 否则前端只想改标题时，未提交的字段会被一起清空——这类静默覆盖是管理面
// 最常见的资损来源（图标、标签、配置 Schema 被抹掉）。
type catalogItemRequest struct {
	Name                *string                 `json:"name"`
	Type                *string                 `json:"type"`
	Title               *string                 `json:"title"`
	Provider            *string                 `json:"provider"`
	Description         *string                 `json:"description"`
	LongDescription     *string                 `json:"longDescription"`
	IconURL             *string                 `json:"iconUrl"`
	Screenshots         *[]string               `json:"screenshots"`
	Tags                *[]string               `json:"tags"`
	Category            *string                 `json:"category"`
	Capabilities        *[]string               `json:"capabilities"`
	RequiredPermissions *[]string               `json:"requiredPermissions"`
	ConfigSchema        *map[string]interface{} `json:"configSchema"`
	IsOfficial          *bool                   `json:"isOfficial"`
	IsFree              *bool                   `json:"isFree"`
	Price               *float64                `json:"price"`
	AuthorName          *string                 `json:"authorName"`
	Homepage            *string                 `json:"homepage"`
	Repository          *string                 `json:"repository"`
	License             *string                 `json:"license"`
	MinSystemVersion    *string                 `json:"minSystemVersion"`
	Version             *string                 `json:"version"`
}

func (r catalogItemRequest) toPatch() marketplace.CatalogItemPatch {
	return marketplace.CatalogItemPatch{
		Name:             r.Name,
		Type:             r.Type,
		Title:            r.Title,
		Provider:         r.Provider,
		Description:      r.Description,
		LongDescription:  r.LongDescription,
		IconURL:          r.IconURL,
		Screenshots:      r.Screenshots,
		Tags:             r.Tags,
		Category:         r.Category,
		Capabilities:     r.Capabilities,
		RequiredPerms:    r.RequiredPermissions,
		ConfigSchema:     r.ConfigSchema,
		IsOfficial:       r.IsOfficial,
		IsFree:           r.IsFree,
		Price:            r.Price,
		AuthorName:       r.AuthorName,
		Homepage:         r.Homepage,
		Repository:       r.Repository,
		License:          r.License,
		MinSystemVersion: r.MinSystemVersion,
		Version:          r.Version,
	}
}

// catalogItemDetailResponse 目录项详情：条目 + 版本。
// 版本内联返回是因为「上架」的判断依据就是「有没有已发布版本」，
// 分成两次请求会让页面出现「有上架按钮但点了报错」的中间态。
type catalogItemDetailResponse struct {
	Item     *dto.MarketplaceItemResponse          `json:"item"`
	Versions []*dto.MarketplaceItemVersionResponse `json:"versions"`
}

type itemVersionRequest struct {
	Version          string                 `json:"version"`
	Changelog        string                 `json:"changelog"`
	DownloadURL      string                 `json:"downloadUrl"`
	ManifestPath     string                 `json:"manifestPath"`
	MinSystemVersion string                 `json:"minSystemVersion"`
	Dependencies     map[string]string      `json:"dependencies"`
	ConfigSchema     map[string]interface{} `json:"configSchema"`
	Publish          bool                   `json:"publish"`
}

// RegisterCatalogRoutes 注册目录管理路由。
//
// 与 RegisterRoutes 分开：安装面是租户路由（tenant group），目录面是平台路由，
// 权限码不同；混在一个函数里会让「新增端点默认落到哪个权限码」靠记忆决定。
func (c *Handler) RegisterCatalogRoutes(r *gin.RouterGroup) {
	catalog := r.Group("/marketplace/catalog")
	{
		catalog.GET("/items", middleware.RequirePermission("marketplace_catalog", "read"), c.AdminListItems)
		catalog.POST("/items", middleware.RequirePermission("marketplace_catalog", "write"), c.CreateCatalogItem)
		catalog.GET("/items/:id", middleware.RequirePermission("marketplace_catalog", "read"), c.GetCatalogItem)
		catalog.PATCH("/items/:id", middleware.RequirePermission("marketplace_catalog", "write"), c.UpdateCatalogItem)
		catalog.DELETE("/items/:id", middleware.RequirePermission("marketplace_catalog", "write"), c.DeleteCatalogItem)
		catalog.POST("/items/:id/publish", middleware.RequirePermission("marketplace_catalog", "write"), c.PublishCatalogItem)
		catalog.POST("/items/:id/unpublish", middleware.RequirePermission("marketplace_catalog", "write"), c.UnpublishCatalogItem)
		catalog.GET("/items/:id/versions", middleware.RequirePermission("marketplace_catalog", "read"), c.ListItemVersions)
		catalog.POST("/items/:id/versions", middleware.RequirePermission("marketplace_catalog", "write"), c.CreateItemVersion)
		catalog.POST("/items/:id/versions/:version/publish", middleware.RequirePermission("marketplace_catalog", "write"), c.PublishItemVersion)
	}
}

// AdminListItems 平台侧目录列表（含草稿/已下架）。
// @Summary 目录列表（平台）
// @Tags 扩展市场
// @Param status query string false "draft/published/deprecated"
// @Param type query string false "connector/skill/plugin"
// @Param search query string false "名称/标题/提供商关键字"
// @Router /api/v1/marketplace/catalog/items [get]
func (c *Handler) AdminListItems(ctx *gin.Context) {
	pagination := common.GetPaginationFromQuery(ctx)
	filter := marketplace.CatalogListFilter{
		Status:   ctx.Query("status"),
		Type:     ctx.Query("type"),
		Search:   ctx.Query("search"),
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}
	items, total, err := c.service.AdminListItems(ctx, filter)
	if err != nil {
		common.RespondError(ctx, err, "查询目录失败")
		return
	}
	common.Success(ctx, gin.H{
		"items":    items,
		"total":    total,
		"page":     filter.Page,
		"pageSize": filter.PageSize,
	})
}

// GetCatalogItem 目录项详情（含版本）。
// @Summary 目录项详情（平台）
// @Router /api/v1/marketplace/catalog/items/{id} [get]
func (c *Handler) GetCatalogItem(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	detail, err := c.service.GetCatalogItem(ctx, itemID)
	if err != nil {
		respondCatalogError(ctx, err, "查询目录项失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

// CreateCatalogItem 创建目录项（初始 draft）。
// @Summary 创建目录项（平台）
// @Router /api/v1/marketplace/catalog/items [post]
func (c *Handler) CreateCatalogItem(ctx *gin.Context) {
	var req catalogItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.BadRequestWithErr(ctx, err, "请求参数错误")
		return
	}
	detail, err := c.service.CreateCatalogItem(ctx, req.toPatch())
	if err != nil {
		respondCatalogError(ctx, err, "创建目录项失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

// UpdateCatalogItem 更新目录项元数据（name/type 不可改）。
// @Summary 更新目录项（平台）
// @Router /api/v1/marketplace/catalog/items/{id} [patch]
func (c *Handler) UpdateCatalogItem(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	var req catalogItemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.BadRequestWithErr(ctx, err, "请求参数错误")
		return
	}
	detail, err := c.service.UpdateCatalogItem(ctx, itemID, req.toPatch())
	if err != nil {
		respondCatalogError(ctx, err, "更新目录项失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

// DeleteCatalogItem 删除目录项（未上架且从未被安装）。
// @Summary 删除目录项（平台）
// @Router /api/v1/marketplace/catalog/items/{id} [delete]
func (c *Handler) DeleteCatalogItem(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	if err := c.service.DeleteCatalogItem(ctx, itemID); err != nil {
		respondCatalogError(ctx, err, "删除目录项失败")
		return
	}
	common.Success(ctx, nil)
}

// PublishCatalogItem 上架。
// @Summary 上架目录项（平台）
// @Router /api/v1/marketplace/catalog/items/{id}/publish [post]
func (c *Handler) PublishCatalogItem(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	detail, err := c.service.PublishCatalogItem(ctx, itemID)
	if err != nil {
		respondCatalogError(ctx, err, "上架失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

// UnpublishCatalogItem 下架（已安装租户不受影响）。
// @Summary 下架目录项（平台）
// @Router /api/v1/marketplace/catalog/items/{id}/unpublish [post]
func (c *Handler) UnpublishCatalogItem(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	detail, err := c.service.UnpublishCatalogItem(ctx, itemID)
	if err != nil {
		respondCatalogError(ctx, err, "下架失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

// ListItemVersions 版本列表。
// @Summary 目录项版本列表（平台）
// @Router /api/v1/marketplace/catalog/items/{id}/versions [get]
func (c *Handler) ListItemVersions(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	versions, err := c.service.ListItemVersions(ctx, itemID)
	if err != nil {
		respondCatalogError(ctx, err, "查询版本失败")
		return
	}
	common.Success(ctx, versions)
}

// CreateItemVersion 新增版本（默认 draft）。
// @Summary 新增目录项版本（平台）
// @Router /api/v1/marketplace/catalog/items/{id}/versions [post]
func (c *Handler) CreateItemVersion(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	var req itemVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.BadRequestWithErr(ctx, err, "请求参数错误")
		return
	}
	version, err := c.service.CreateItemVersion(ctx, itemID, marketplace.ItemVersionInput{
		Version:          req.Version,
		Changelog:        req.Changelog,
		DownloadURL:      req.DownloadURL,
		ManifestPath:     req.ManifestPath,
		MinSystemVersion: req.MinSystemVersion,
		Dependencies:     req.Dependencies,
		ConfigSchema:     req.ConfigSchema,
		Publish:          req.Publish,
	})
	if err != nil {
		respondCatalogError(ctx, err, "新增版本失败")
		return
	}
	common.Success(ctx, version)
}

// PublishItemVersion 发布指定版本（推进 latest_version）。
// @Summary 发布目录项版本（平台）
// @Router /api/v1/marketplace/catalog/items/{id}/versions/{version}/publish [post]
func (c *Handler) PublishItemVersion(ctx *gin.Context) {
	itemID, ok := parseCatalogItemID(ctx)
	if !ok {
		return
	}
	detail, err := c.service.PublishItemVersion(ctx, itemID, ctx.Param("version"))
	if err != nil {
		respondCatalogError(ctx, err, "发布版本失败")
		return
	}
	common.Success(ctx, toCatalogDetailResponse(detail))
}

func parseCatalogItemID(ctx *gin.Context) (int, bool) {
	itemID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || itemID <= 0 {
		common.ParamError(ctx, "invalid item id")
		return 0, false
	}
	return itemID, true
}

func toCatalogDetailResponse(detail *marketplace.CatalogItemDetail) *catalogItemDetailResponse {
	if detail == nil {
		return nil
	}
	return &catalogItemDetailResponse{Item: detail.Item, Versions: detail.Versions}
}

// respondCatalogError 把目录域错误翻译成可操作的状态码与文案。
//
// 这里刻意不用 RespondError 兜底全部：目录管理的失败大多有明确处置动作
// （去注册 provider / 先发布版本 / 先下架 / 换版本号），笼统的 500 会让
// 运营去查日志，而答案其实在响应里就能给出。
func respondCatalogError(ctx *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, marketplace.ErrMarketplaceItemNotFound), errors.Is(err, marketplace.ErrCatalogVersionNotFound):
		common.NotFoundWithErr(ctx, err, "目录项或版本不存在")
	case errors.Is(err, marketplace.ErrCatalogNameInvalid),
		errors.Is(err, marketplace.ErrCatalogFieldRequired),
		errors.Is(err, marketplace.ErrCatalogVersionInvalid),
		errors.Is(err, marketplace.ErrCatalogConnectorUnregistered):
		// 参数错误必须带上具体原因：运营看到的应该是「ghost 未注册，已注册有 …」
		// 而不是「参数不合法」——后者无法转化为任何下一步动作。
		common.BadRequestWithErr(ctx, err, "目录参数不合法："+err.Error())
	case errors.Is(err, marketplace.ErrCatalogVersionExists):
		common.Conflict(ctx, "版本号已存在，请换一个版本号："+err.Error(), nil)
	case errors.Is(err, marketplace.ErrCatalogItemIdentityImmutable):
		common.Conflict(ctx, "名称与类型创建后不可修改："+err.Error(), nil)
	case errors.Is(err, marketplace.ErrCatalogNoPublishedVersion):
		common.Conflict(ctx, "该目录项还没有已发布版本，请先新增并发布一个版本再上架", nil)
	case errors.Is(err, marketplace.ErrCatalogItemPublished):
		common.Conflict(ctx, "已上架的目录项不能删除，请先下架", nil)
	case errors.Is(err, marketplace.ErrCatalogItemInstalled):
		common.Conflict(ctx, "已有租户安装该目录项，不能删除（下架即可停止新安装）："+err.Error(), nil)
	default:
		common.RespondError(ctx, err, fallback)
	}
}
