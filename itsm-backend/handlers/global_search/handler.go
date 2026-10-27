package global_search

import (
	"strings"

	"itsm-backend/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Handler 全局搜索HTTP处理器
type Handler struct {
	service *Service
}

// NewHandler creates a new global search handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// SearchResult 搜索结果
type SearchResult struct {
	ID           int    `json:"id"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Status       string `json:"status,omitempty"`
	TicketNumber string `json:"ticketNumber,omitempty"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Total   int             `json:"total"`
}

// Search 全局搜索
// @Summary 全局搜索
// @Description 在工单、事件、问题、变更、知识库中搜索
// @Tags 全局搜索
// @Produce json
// @Param q query string true "搜索关键词"
// @Success 200 {object} common.Response
// @Router /api/v1/global-search [get]
func (h *Handler) Search(ctx *gin.Context) {
	keyword := strings.TrimSpace(ctx.Query("q"))
	if keyword == "" {
		common.Success(ctx, &SearchResponse{Results: []*SearchResult{}, Total: 0})
		return
	}

	tenantID, ok := middleware.TenantIDOrUnauthorized(ctx)
	if !ok {
		return
	}
	if tenantID == 0 {
		common.Fail(ctx, common.AuthFailedCode, "租户信息缺失")
		return
	}

	response, err := h.service.Search(ctx.Request.Context(), tenantID, keyword, middleware.GetContextRoles(ctx))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "搜索失败")
		return
	}

	common.Success(ctx, response)
}

// RegisterRoutes 注册全局搜索路由。
//
// 显式 RequirePermission：全局搜索是跨域聚合面，读权限码取 dashboard:read
// （与工作台同族的「聚合读面」语义）。此前这条路由没有任何权限声明，
// RBAC 中间件只能按路径推断出 global-search:read —— 该码不在 authz 词表、
// 无人持有，DBOnly 下即除 super_admin 外全员 403，功能实际不可用。
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	search := r.Group("/global-search")
	{
		search.GET("", middleware.RequirePermission("dashboard", "read"), h.Search)
	}
}
