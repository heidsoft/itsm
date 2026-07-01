package marketplace

import (
	"net/http"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/middleware"
	"itsm-backend/service/marketplace"

	"github.com/gin-gonic/gin"
)

// Controller 市场控制器
type Controller struct {
	service          *marketplace.Service
	connectorManager *connector.Manager
}

// NewController 创建市场控制器
func NewController(service *marketplace.Service, managers ...*connector.Manager) *Controller {
	var connectorManager *connector.Manager
	if len(managers) > 0 {
		connectorManager = managers[0]
	}
	return &Controller{
		service:          service,
		connectorManager: connectorManager,
	}
}

func getTenantID(ctx *gin.Context) (int, bool) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, err.Error())
		return 0, false
	}
	return tenantID, true
}

func getUserID(ctx *gin.Context) (int, bool) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.AuthFailedCode, err.Error())
		return 0, false
	}
	return userID, true
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r *gin.RouterGroup) {
	marketplaceGroup := r.Group("/marketplace")
	{
		// 公开接口
		marketplaceGroup.GET("/items", c.ListItems)
		marketplaceGroup.GET("/items/:id", c.GetItem)

		// 需要登录的接口
		{
			marketplaceGroup.POST("/items/:id/install", c.InstallItem)
			marketplaceGroup.POST("/items/:id/uninstall", c.UninstallItem)
			marketplaceGroup.GET("/installations", c.ListInstallations)
			marketplaceGroup.GET("/installations/:id", c.GetInstallation)
			marketplaceGroup.PUT("/installations/:id/config", c.UpdateInstallationConfig)
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
// @Param is_official query boolean false "是否是官方组件"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} common.Response{data=object{items=[]ent.MarketplaceItem, total=int, page=int, page_size=int}}
// @Router /api/v1/marketplace/items [get]
func (c *Controller) ListItems(ctx *gin.Context) {
	itemType := ctx.Query("type")
	category := ctx.Query("category")
	search := ctx.Query("search")
	isOfficialStr := ctx.Query("is_official")
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var isOfficial *bool
	if isOfficialStr != "" {
		boo, _ := strconv.ParseBool(isOfficialStr)
		isOfficial = &boo
	}

	items, total, err := c.service.ListItems(ctx, itemType, category, search, isOfficial, page, pageSize)
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
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
func (c *Controller) GetItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	item, err := c.service.GetItem(ctx, itemID)
	if err != nil {
		common.Fail(ctx, http.StatusNotFound, err.Error())
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
func (c *Controller) InstallItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := getTenantID(ctx)
	if !ok {
		return
	}
	userID, ok := getUserID(ctx)
	if !ok {
		return
	}

	installation, err := c.service.InstallItem(ctx, tenantID, itemID, strconv.Itoa(userID))
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(ctx, installation)
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
func (c *Controller) UninstallItem(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := getTenantID(ctx)
	if !ok {
		return
	}

	err = c.service.UninstallItem(ctx, tenantID, itemID)
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
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
func (c *Controller) ListInstallations(ctx *gin.Context) {
	status := ctx.Query("status")
	tenantID, ok := getTenantID(ctx)
	if !ok {
		return
	}

	installations, err := c.service.ListInstallations(ctx, tenantID, status)
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(ctx, installations)
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
func (c *Controller) GetInstallation(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	tenantID, ok := getTenantID(ctx)
	if !ok {
		return
	}

	installation, err := c.service.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		common.Fail(ctx, http.StatusNotFound, err.Error())
		return
	}

	common.Success(ctx, installation)
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
func (c *Controller) UpdateInstallationConfig(ctx *gin.Context) {
	itemIDStr := ctx.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid item id")
		return
	}

	var config map[string]interface{}
	if err := ctx.ShouldBindJSON(&config); err != nil {
		common.Fail(ctx, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}

	tenantID, ok := getTenantID(ctx)
	if !ok {
		return
	}

	installation, err := c.service.UpdateInstallationConfig(ctx, tenantID, itemID, config)
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	installation, err = c.service.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if err := c.provisionConnectorInstallation(ctx, installation); err != nil {
		common.Fail(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(ctx, installation)
}

func (c *Controller) provisionConnectorInstallation(ctx *gin.Context, installation *ent.TenantInstallation) error {
	if c.connectorManager == nil || installation == nil || installation.Edges.Item == nil {
		return nil
	}
	item := installation.Edges.Item
	if item.Type != marketplaceitem.TypeConnector {
		return nil
	}
	connectorName := strings.TrimSuffix(item.Name, "-connector")
	cfg := connector.Config{
		TenantID:    installation.TenantID,
		Name:        connectorName,
		Provider:    connectorName,
		Enabled:     installation.Status == "active",
		Credentials: stringMapFromConfig(installation.Config["credentials"]),
		Settings:    interfaceMapFromConfig(installation.Config["settings"]),
		Labels: map[string]string{
			"marketplace_item_id": strconv.Itoa(item.ID),
			"marketplace_name":    item.Name,
			"marketplace_title":   item.Title,
		},
		CreatedAt: installation.InstalledAt,
		UpdatedAt: installation.UpdatedAt,
	}
	if cfg.Provider == "" {
		cfg.Provider = item.Name
	}
	return c.connectorManager.Provision(ctx.Request.Context(), cfg)
}

func stringMapFromConfig(value interface{}) map[string]string {
	out := map[string]string{}
	if typed, ok := value.(map[string]string); ok {
		for k, v := range typed {
			out[k] = v
		}
		return out
	}
	if typed, ok := value.(map[string]interface{}); ok {
		for k, v := range typed {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func interfaceMapFromConfig(value interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	if typed, ok := value.(map[string]interface{}); ok {
		for k, v := range typed {
			out[k] = v
		}
		return out
	}
	if typed, ok := value.(map[string]string); ok {
		for k, v := range typed {
			out[k] = v
		}
	}
	return out
}
