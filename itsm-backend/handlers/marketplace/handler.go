package marketplace

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
			// 升级与自动升级意愿同样是平台级变更：换版本会重新装载运行态，
			// 权限码必须与安装一致，否则「能装的人不能升」会让 auto_upgrade 开关失去意义。
			marketplaceGroup.POST("/items/:id/upgrade", middleware.RequirePermission("marketplace", "write"), c.UpgradeItem)
			marketplaceGroup.PATCH("/items/:id/auto-upgrade", middleware.RequirePermission("marketplace", "write"), c.SetAutoUpgrade)
			marketplaceGroup.GET("/installations", middleware.RequirePermission("marketplace", "read"), c.ListInstallations)
			// 这三条端点以商品 ID 定位安装记录（service.GetInstallation 按 tenant_id +
			// item_id 查），而安装列表 DTO 同时有 id（安装行主键）和 itemId（商品 ID）。
			// 参数叫 :id 时调用方会把列表里的 id 填进来，命中的是另一条安装或 404。
			marketplaceGroup.GET("/installations/:itemId", middleware.RequirePermission("marketplace", "read"), c.GetInstallation)
			marketplaceGroup.PUT("/installations/:itemId/config", middleware.RequirePermission("marketplace", "write"), c.UpdateInstallationConfig)
			// 存量安装的依赖对齐是只读诊断面：它不修数据，只如实报告哪些前置没满足、
			// 哪些版本行缺失，供管理员决定下一步是补装前置还是补目录版本。
			marketplaceGroup.GET("/dependency-status", middleware.RequirePermission("marketplace", "read"), c.DependencyStatus)
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
// @Success 200 {object} common.Response{data=object{items=[]dto.MarketplaceItemResponse,total=int,page=int,pageSize=int,totalPages=int}}
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

	common.SuccessWithList(ctx, items, total, page, pageSize)
}

// GetItem 获取商品详情
// @Summary 获取商品详情
// @Description 根据ID获取组件的详细信息
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Success 200 {object} common.Response{data=dto.MarketplaceItemResponse}
// @Router /api/v1/marketplace/items/{id} [get]
func (c *Handler) GetItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.ParamError(ctx, "invalid item id")
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
// @Success 200 {object} common.Response{data=marketplace.InstallationResponse}
// @Router /api/v1/marketplace/items/{id}/install [post]
func (c *Handler) InstallItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.ParamError(ctx, "invalid item id")
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

	installation, err := c.service.InstallItem(ctx.Request.Context(), tenantID, itemID, strconv.Itoa(userID))
	if err != nil {
		switch {
		case errors.Is(err, marketplace.ErrMarketplaceItemNotFound):
			common.NotFoundWithErr(ctx, err, "组件不存在或已被下架")
		case errors.Is(err, marketplace.ErrMarketplaceItemUnavailable), errors.Is(err, marketplace.ErrMarketplaceInstalledByMissing):
			common.BadRequestWithErr(ctx, err, "组件当前不可安装")
		case errors.Is(err, marketplace.ErrMarketplaceDeliveryUnsupported):
			common.Fail(ctx, common.NotImplementedCode, "平台不交付该类型商品：插件需要宿主运行时，历史插件条目仅可查看与卸载")
		case marketplace.IsDeliveryNotReady(err):
			// 运行态没能起来（连接器缺凭据 / 技能交付物在本进程不存在）。
			// 安装记录已落库并标为 failed，这里如实返回那条记录，让前端显示
			// 「已安装但未就绪 + errorMessage」，而不是笼统的「安装失败」让人以为压根没装上。
			if installation != nil {
				common.SuccessWithMessage(ctx,
					"安装记录已创建，但运行态未就绪；补全配置或修正目录版本后重新安装即可",
					ToInstallationResponse(installation))
				return
			}
			common.Conflict(ctx, "组件运行态未就绪："+err.Error(), nil)
		case errors.Is(err, marketplace.ErrMarketplaceDependencyUnsatisfied):
			// 前置组件缺失/版本不符属于业务冲突而非参数错误：商品本身可装，
			// 只是依赖链没满足。明细由 service 逐项拼好，直接展示给管理员。
			common.Conflict(ctx, err.Error(), nil)
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
		common.ParamError(ctx, "invalid item id")
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
		if errors.Is(err, marketplace.ErrMarketplaceItemRequiredByDependents) {
			// 与安装期依赖闸门对称：前置被拆掉会让依赖它的组件悬空，
			// 但记录仍显示 active。卸载被拒时安装记录保持原状。
			common.Conflict(ctx, err.Error(), nil)
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
// @Success 200 {object} common.Response{data=object{items=[]marketplace.InstallationResponse,total=int}}
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

	// 不分页的列表：service 整表取回该租户的安装记录，没有 Offset/Limit，
	// 因此只返回诚实的 {items,total}，不伪造 page/pageSize/totalPages。
	// 见 docs/api-reference.md「不分页的列表」。
	items := ToInstallationResponseList(installations)
	common.Success(ctx, gin.H{
		"items": items,
		"total": len(items),
	})
}

// GetInstallation 获取安装详情
// @Summary 获取安装详情
// @Description 获取指定组件的安装详情和配置
// @Tags 市场
// @Accept json
// @Produce json
// @Param itemId path int true "商品ID（安装记录按 tenant + itemId 定位）"
// @Security Bearer
// @Success 200 {object} common.Response{data=marketplace.InstallationResponse}
// @Router /api/v1/marketplace/installations/{itemId} [get]
func (c *Handler) GetInstallation(ctx *gin.Context) {
	itemIDStr := ctx.Param("itemId")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.ParamError(ctx, "invalid item id")
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
// @Param itemId path int true "商品ID（安装记录按 tenant + itemId 定位）"
// @Param config body object true "配置信息"
// @Security Bearer
// @Success 200 {object} common.Response{data=marketplace.InstallationResponse}
// @Router /api/v1/marketplace/installations/{itemId}/config [put]
func (c *Handler) UpdateInstallationConfig(ctx *gin.Context) {
	itemIDStr := ctx.Param("itemId")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.ParamError(ctx, "invalid item id")
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

	installation, err := c.service.UpdateInstallationConfig(ctx.Request.Context(), tenantID, itemID, config)
	if err != nil {
		if errors.Is(err, marketplace.ErrMarketplaceInstallationAbsent) {
			common.NotFoundWithErr(ctx, err, "安装记录不存在或已被卸载")
			return
		}
		if errors.Is(err, marketplace.ErrMarketplaceDeliveryUnsupported) {
			common.Fail(ctx, common.NotImplementedCode, "平台不交付该类型商品：插件需要宿主运行时，历史插件条目仅可查看与卸载")
			return
		}
		if errors.Is(err, marketplace.ErrMarketplaceConfigUnsupported) {
			// 类型本身可交付，但它没有租户级配置面：这是「不接受该请求」，
			// 不是「功能未实现」——收下并忽略配置只会造出一个假保存。
			common.BadRequestWithErr(ctx, err, "该组件类型没有租户级配置项")
			return
		}
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

// upgradeRequest 升级入参。targetVersion 可省略，省略即升到目录当前 latest。
//
// 用指针而不是空串区分「没传」与「传了空串」：后者是调用方明确要求升到一个
// 空版本号，属于参数错误，不能被静默当成「升最新版」。
type upgradeRequest struct {
	TargetVersion *string `json:"targetVersion"`
}

// UpgradeItem 升级已安装的组件
// @Summary 升级已安装的组件
// @Description 把租户的安装记录升到目录当前最新版本或指定的已发布版本；目标版本比当前旧时即为回滚
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Param body body upgradeRequest false "目标版本；省略即升到目录当前 latest"
// @Security Bearer
// @Success 200 {object} common.Response{data=marketplace.InstallationResponse}
// @Router /api/v1/marketplace/items/{id}/upgrade [post]
func (c *Handler) UpgradeItem(ctx *gin.Context) {
	itemID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.ParamError(ctx, "invalid item id")
		return
	}

	// 空 body 表示「升到最新版」，因此只在真的有请求体时绑定，
	// 避免 ShouldBindJSON 在 Content-Length: 0 上返回 EOF 变成参数错误。
	var req upgradeRequest
	targetVersion := ""
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			common.BadRequestWithErr(ctx, err, "请求参数错误")
			return
		}
		if req.TargetVersion != nil {
			targetVersion = strings.TrimSpace(*req.TargetVersion)
			if targetVersion == "" {
				common.ParamError(ctx, "targetVersion 不能为空字符串；省略该字段即升到最新版本")
				return
			}
		}
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}
	userID, ok := middleware.UserIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	result, err := c.service.UpgradeInstallation(ctx.Request.Context(), tenantID, itemID, targetVersion, strconv.Itoa(userID))
	if err != nil {
		switch {
		case errors.Is(err, marketplace.ErrMarketplaceItemNotFound):
			common.NotFoundWithErr(ctx, err, "组件不存在或已被下架")
		case errors.Is(err, marketplace.ErrMarketplaceInstallationAbsent):
			common.NotFoundWithErr(ctx, err, "安装记录不存在或已被卸载")
		case errors.Is(err, marketplace.ErrMarketplaceDeliveryUnsupported):
			common.Fail(ctx, common.NotImplementedCode, "平台不交付该类型商品：插件需要宿主运行时，历史插件条目仅可查看与卸载")
		case errors.Is(err, marketplace.ErrMarketplaceUpgradeTargetInvalid):
			// 调用方点名了一个不可用的版本 → 参数问题；
			// 没点名（即「升最新」）而目录里没有可用版本 → 数据状态冲突。
			if targetVersion != "" {
				common.BadRequestWithErr(ctx, err, "升级目标不可用")
			} else {
				common.Conflict(ctx, "目录里没有可用于升级的已发布版本："+err.Error(), nil)
			}
		case errors.Is(err, marketplace.ErrMarketplaceDependencyUnsatisfied):
			common.Conflict(ctx, err.Error(), nil)
		case marketplace.IsDeliveryNotReady(err), errors.Is(err, marketplace.ErrMarketplaceUpgradeBlocked):
			// 运行态没能起来：installed_version 没换、状态没动，租户仍在跑旧版本。
			// 必须把「没升成」说清楚，否则界面以为升级已完成。
			common.Conflict(ctx, "升级未完成，仍在使用当前版本："+err.Error(), nil)
		default:
			common.RespondError(ctx, err, "升级组件失败")
		}
		return
	}

	if !result.Upgraded {
		common.SuccessWithMessage(ctx,
			fmt.Sprintf("已是版本 %s，无需升级", result.ToVersion),
			ToInstallationResponse(result.Installation))
		return
	}
	common.SuccessWithMessage(ctx,
		fmt.Sprintf("已从 %s 升级到 %s", result.FromVersion, result.ToVersion),
		ToInstallationResponse(result.Installation))
}

// autoUpgradeRequest 自动升级意愿入参。
//
// 必须是 *bool：false 是合法值（关掉自动升级），用 bool 绑定会让「未传」
// 与「显式关闭」都落到 false，一次误发的 PATCH 就把租户的自动升级永久关掉。
type autoUpgradeRequest struct {
	AutoUpgrade *bool `json:"autoUpgrade"`
}

// SetAutoUpgrade 开关某条安装记录的自动升级
// @Summary 设置组件的自动升级意愿
// @Description 开启后，后台升级 pass 会把该安装记录升到目录当前 latest 已发布版本
// @Tags 市场
// @Accept json
// @Produce json
// @Param id path int true "组件ID"
// @Param body body autoUpgradeRequest true "自动升级意愿（autoUpgrade 必填，false 是合法值）"
// @Security Bearer
// @Success 200 {object} common.Response{data=marketplace.InstallationResponse}
// @Router /api/v1/marketplace/items/{id}/auto-upgrade [patch]
func (c *Handler) SetAutoUpgrade(ctx *gin.Context) {
	itemID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.ParamError(ctx, "invalid item id")
		return
	}

	var req autoUpgradeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.BadRequestWithErr(ctx, err, "请求参数错误")
		return
	}
	if req.AutoUpgrade == nil {
		common.ParamError(ctx, "autoUpgrade 必填（true 开启自动升级 / false 关闭）")
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

	installation, err := c.service.SetAutoUpgrade(ctx.Request.Context(), tenantID, itemID, *req.AutoUpgrade, strconv.Itoa(userID))
	if err != nil {
		if errors.Is(err, marketplace.ErrMarketplaceInstallationAbsent) {
			common.NotFoundWithErr(ctx, err, "安装记录不存在或已被卸载")
			return
		}
		common.RespondError(ctx, err, "更新自动升级设置失败")
		return
	}

	common.Success(ctx, ToInstallationResponse(installation))
}

// DependencyStatus 存量安装的依赖对齐报告
// @Summary 查询存量安装的依赖对齐状态
// @Description 逐条判定租户已安装组件的前置依赖是否仍然满足；只读，不修改安装记录
// @Tags 市场
// @Produce json
// @Security Bearer
// @Success 200 {object} common.Response{data=dto.MarketplaceDependencyReportResponse}
// @Router /api/v1/marketplace/dependency-status [get]
func (c *Handler) DependencyStatus(ctx *gin.Context) {
	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}

	report, err := c.service.DependencyStatus(ctx.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(ctx, err, "查询依赖对齐状态失败")
		return
	}

	common.Success(ctx, ToDependencyReportResponse(report))
}
