package global_search

import (
	"net/http"
	"strings"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/ticket"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Handler 全局搜索HTTP处理器
type Handler struct {
	client *ent.Client
}

// NewHandler creates a new global search handler
func NewHandler(client *ent.Client) *Handler {
	return &Handler{client: client}
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
	Total   int            `json:"total"`
}

// Search 全局搜索
// @Summary 全局搜索
// @Description 在工单、事件、问题、变更、知识库中搜索
// @Tags 全局搜索
// @Produce json
// @Param keyword query string true "搜索关键词"
// @Success 200 {object} common.Response
// @Router /api/v1/global-search [get]
func (h *Handler) Search(ctx *gin.Context) {
	keyword := strings.TrimSpace(ctx.Query("keyword"))
	if keyword == "" {
		common.Success(ctx, &SearchResponse{Results: []*SearchResult{}, Total: 0})
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil || tenantID == 0 {
		common.Fail(ctx, http.StatusBadRequest, "租户上下文缺失")
		return
	}

	results := make([]*SearchResult, 0)

	// 搜索工单
	tickets, err := h.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.Or(
				ticket.TitleContainsFold(keyword),
				ticket.DescriptionContainsFold(keyword),
				ticket.TicketNumberContainsFold(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, t := range tickets {
			results = append(results, &SearchResult{
				ID:           t.ID,
				Type:         "ticket",
				Title:        t.Title,
				Description:  t.Description,
				Status:       t.Status,
				TicketNumber: t.TicketNumber,
			})
		}
	}

	// 搜索事件
	incidents, err := h.client.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.Or(
				incident.TitleContainsFold(keyword),
				incident.DescriptionContainsFold(keyword),
				incident.IncidentNumberContainsFold(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, i := range incidents {
			results = append(results, &SearchResult{
				ID:           i.ID,
				Type:         "incident",
				Title:        i.Title,
				Description:  i.Description,
				Status:       i.Status,
				TicketNumber: i.IncidentNumber,
			})
		}
	}

	// 搜索问题
	problems, err := h.client.Problem.Query().
		Where(
			problem.TenantID(tenantID),
			problem.Or(
				problem.TitleContainsFold(keyword),
				problem.DescriptionContainsFold(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, p := range problems {
			results = append(results, &SearchResult{
				ID:          p.ID,
				Type:        "problem",
				Title:       p.Title,
				Description: p.Description,
				Status:      p.Status,
			})
		}
	}

	// 搜索变更
	changes, err := h.client.Change.Query().
		Where(
			change.TenantID(tenantID),
			change.Or(
				change.TitleContainsFold(keyword),
				change.DescriptionContainsFold(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, ch := range changes {
			results = append(results, &SearchResult{
				ID:          ch.ID,
				Type:        "change",
				Title:       ch.Title,
				Description: ch.Description,
				Status:      ch.Status,
			})
		}
	}

	// 搜索知识库文章
	articles, err := h.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
			knowledgearticle.Or(
				knowledgearticle.TitleContainsFold(keyword),
				knowledgearticle.ContentContainsFold(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, a := range articles {
			status := "draft"
			if a.IsPublished {
				status = "published"
			}
			results = append(results, &SearchResult{
				ID:          a.ID,
				Type:        "knowledge",
				Title:       a.Title,
				Description: a.Content,
				Status:      status,
			})
		}
	}

	common.Success(ctx, &SearchResponse{
		Results: results,
		Total:   len(results),
	})
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	search := r.Group("/global-search")
	{
		search.GET("", h.Search)
	}
}
