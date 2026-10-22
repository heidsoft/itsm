package marketplace

import (
	"errors"
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
)

// Handler 市场 HTTP 入口。
type Handler struct {
	service *marketplace.Service
}

// NewHandler 创建市场 Handler。
func NewHandler(service *marketplace.Service) *Handler {
	return &Handler{service: service}
}

func (c *Handler) RegisterRoutes(r *gin.RouterGroup) {
	marketplaceGroup := r.Group("/marketplace")
	{
		// 公开接口
		marketplaceGroup.GET("/items", c.ListItems)
		marketplaceGroup.GET("/items/:id", c.GetItem)

		// 扩展安装/卸载/配置 = 平台级变更：按 marketplace:* 授权。
		//
		// 口径说明（2026-10-15 复核修正）：此前的注释称"该码尚未授予任何角色，
		// 因此默认无人可调用"，与实际词表不符——internal/authz/roles.go 已把
		// marketplace:read/write 授予 domainrole.Admin，且预检映射
		// （middleware/rbac_precheck_gen.go）中 install/uninstall/installations:*/config
		// 三个写面端点齐备。因此这组端点对 admin 可正常调用，非全员不可用。
		// 保留说明是为了让后续读者不必重新排查一遍授权链路。
		{
			marketplaceGroup.POST("/items/:id/install", middleware.RequirePermission("marketplace", "write"), c.InstallItem)
			marketplaceGroup.POST("/items/:id/uninstall", middleware.RequirePermission("marketplace", "write"), c.UninstallItem)
			marketplaceGroup.GET("/installations", middleware.RequirePermission("marketplace", "read"), c.ListInstallations)
			marketplaceGroup.GET("/installations/:id", middleware.RequirePermission("marketplace", "read"), c.GetInstallation)
			marketplaceGroup.PUT("/installations/:id/config", middleware.RequirePermission("marketplace", "write"), c.UpdateInstallationConfig)
		}
	}
}

// ListItems 查询商品列表
// @Summary 查询市场商品列表
// @Description 根据条件查询市场上的组件列表
// @Tags 市场
// @Accept json
// @Produce json
// @Param type query string false "组件类型：connector/skill/plugin"
// @Param category query string false "分类"
// @Param search query string false "搜索关键词"
// @Param isOfficial query boolean false "是否是官方组件"
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认20"
// @Success 200 {object} common.Response{data=object{items=[]ent.MarketplaceItem, total=int, page=int, pageSize=int}}
// @Router /api/v1/marketplace/items [get]
func (c *Handler) ListItems(ctx *gin.Context) {
	itemType := ctx.Query("type")
	category := ctx.Query("category")
	search := ctx.Query("search")
	isOfficialStr := ctx.Query("isOfficial")

	pagination := common.GetPaginationFromQuery(ctx)
	page, pageSize := pagination.Page, pagination.PageSize

	var isOfficial *bool
	if isOfficialStr != "" {
		boo, _ := strconv.ParseBool(isOfficialStr)
		isOfficial = &boo
	}

	items, total, err := c.service.ListItems(ctx, itemType, category, search, isOfficial, page, pageSize)
	if err != nil {
		common.FailWithErr(ctx, err, "查询组件列表失败")
		return
	}

	common.Success(ctx, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetItem 获取商品详情
// @Summary 获取商品详情
// @Description 根据ID获取组件的详细信息
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Success 200 {object} common.Response{data=ent.MarketplaceItem}
// @Router /api/v1/marketplace/items/{id} [get]
func (c *Handler) GetItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	item, err := c.service.GetItem(ctx, itemID)
	if err != nil {
		common.NotFoundWithErr(ctx, err, "组件不存在或已被下架")
		return
	}

	common.Success(ctx, item)
}

// InstallItem 安装组件
// @Summary 安装组件
// @Description 租户安装指定的组件
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Security Bearer
// @Success 200 {object} common.Response{data=ent.TenantInstallation}
// @Router /api/v1/marketplace/items/{id}/install [post]
func (c *Handler) InstallItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}
	userID, ok := middleware.UserIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	installation, err := c.service.InstallItem(ctx, tenantID, itemID, strconv.Itoa(userID))
	if err != nil {
		switch {
		case errors.Is(err, marketplace.ErrMarketplaceItemNotFound):
			common.NotFoundWithErr(ctx, err, "组件不存在或已被下架")
		case errors.Is(err, marketplace.ErrMarketplaceItemUnavailable), errors.Is(err, marketplace.ErrMarketplaceInstalledByMissing):
			common.BadRequestWithErr(ctx, err, "组件当前不可安装")
		case errors.Is(err, marketplace.ErrMarketplaceConnectorNotReady):
			// 安装记录已落库并标为 failed（连接器没起来：缺凭据 / provider 未注册 /
			// 端点不可达）。这里如实返回那条记录，让前端显示「已安装但未就绪 +
			// errorMessage」，而不是笼统的「安装失败」让人以为压根没装上。
			if installation != nil {
				common.SuccessWithMessage(ctx,
					"安装记录已创建，但连接器未就绪；补全配置后重新安装即可",
					ToInstallationResponse(installation))
				return
			}
			common.Conflict(ctx, "连接器未就绪："+err.Error(), nil)
		default:
			common.RespondError(ctx, err, "安装组件失败")
		}
		return
	}

	common.Success(ctx, ToInstallationResponse(installation))
}

// UninstallItem 卸载组件
// @Summary 卸载组件
// @Description 租户卸载已安装的组件
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Security Bearer
// @Success 200 {object} common.Response
// @Router /api/v1/marketplace/items/{id}/uninstall [post]
func (c *Handler) UninstallItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	err = c.service.UninstallItem(ctx, tenantID, itemID)
	if err != nil {
		if errors.Is(err, marketplace.ErrMarketplaceInstallationAbsent) {
			common.NotFoundWithErr(ctx, err, "安装记录不存在或已被卸载")
			return
		}
		if errors.Is(err, marketplace.ErrMarketplaceConnectorStillRunning) {
			// 运行态没能注销就绝不能标 uninstalled——否则界面显示已卸载，
			// 连接器还在用活着的凭据收发消息。
			common.Conflict(ctx, "连接器运行实例未能注销，卸载已中止（安装记录保持原状态）："+err.Error(), nil)
			return
		}
		common.RespondError(ctx, err, "卸载组件失败")
		return
	}

	common.Success(ctx, nil)
}

// ListInstallations 列出已安装的组件
// @Summary 列出已安装的组件
// @Description 列出租户所有已安装的组件
// @Tags 市场
// @Accept json
// @Produce json
// @Param status query string false "状态过滤：active/disabled/failed"
// @Security Bearer
// @Success 200 {object} common.Response{data=[]ent.TenantInstallation}
// @Router /api/v1/marketplace/installations [get]
func (c *Handler) ListInstallations(ctx *gin.Context) {
	status := ctx.Query("status")
	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	installations, err := c.service.ListInstallations(ctx, tenantID, status)
	if err != nil {
		common.RespondError(ctx, err, "查询安装列表失败")
		return
	}

	common.Success(ctx, ToInstallationResponseList(installations))
}

// GetInstallation 获取安装详情
// @Summary 获取安装详情
// @Description 获取指定组件的安装详情和配置
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Security Bearer
// @Success 200 {object} common.Response{data=ent.TenantInstallation}
// @Router /api/v1/marketplace/installations/{id} [get]
func (c *Handler) GetInstallation(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	installation, err := c.service.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		if errors.Is(err, marketplace.ErrMarketplaceInstallationAbsent) {
			common.NotFoundWithErr(ctx, err, "安装记录不存在或已被卸载")
			return
		}
		common.RespondError(ctx, err, "查询安装详情失败")
		return
	}

	common.Success(ctx, ToInstallationResponse(installation))
}

// UpdateInstallationConfig 更新组件配置
// @Summary 更新组件配置
// @Description 更新已安装组件的配置信息
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Param config body object true "配置信息"
// @Security Bearer
// @Success 200 {object} common.Response{data=ent.TenantInstallation}
// @Router /api/v1/marketplace/installations/{id}/config [put]
func (c *Handler) UpdateInstallationConfig(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	var config map[string]interface{}
	if err := ctx.ShouldBindJSON(&config); err != nil {
		common.BadRequestWithErr(ctx, err, "请求参数错误")
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	installation, err := c.service.UpdateInstallationConfig(ctx, tenantID, itemID, config)
	if err != nil {
		if errors.Is(err, marketplace.ErrMarketplaceConnectorNotReady) {
			// 新配置没能让运行态起来，DB 保持原配置 —— 必须让人知道「这次没生效」，
			// 否则会以为换了凭据而实际仍在用旧的。
			common.Conflict(ctx, "新配置未生效，已保留原配置："+err.Error(), nil)
			return
		}
		common.RespondError(ctx, err, "更新安装配置失败")
		return
	}

	common.Success(ctx, ToInstallationResponse(installation))
}
