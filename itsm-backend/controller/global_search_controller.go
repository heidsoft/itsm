package controller

import (
	"net/http"

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

// GlobalSearchController 全局搜索控制器
type GlobalSearchController struct {
	client *ent.Client
}

// NewGlobalSearchController 创建全局搜索控制器
func NewGlobalSearchController(client *ent.Client) *GlobalSearchController {
	return &GlobalSearchController{client: client}
}

// SearchResult 搜索结果
type SearchResult struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Number      string `json:"ticketNumber,omitempty"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Total   int             `json:"total"`
}

// Search 全局搜索
func (c *GlobalSearchController) Search(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
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
	tickets, err := c.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.Or(
				ticket.TitleContains(keyword),
				ticket.DescriptionContains(keyword),
				ticket.TicketNumberContains(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, t := range tickets {
			results = append(results, &SearchResult{
				ID:          t.ID,
				Type:        "ticket",
				Title:       t.Title,
				Description: t.Description,
				Status:      t.Status,
				Number:      t.TicketNumber,
			})
		}
	}

	// 搜索事件
	incidents, err := c.client.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.Or(
				incident.TitleContains(keyword),
				incident.DescriptionContains(keyword),
				incident.IncidentNumberContains(keyword),
			),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, i := range incidents {
			results = append(results, &SearchResult{
				ID:          i.ID,
				Type:        "incident",
				Title:       i.Title,
				Description: i.Description,
				Status:      i.Status,
				Number:      i.IncidentNumber,
			})
		}
	}

	// 搜索问题
	problems, err := c.client.Problem.Query().
		Where(
			problem.TenantID(tenantID),
			problem.Or(
				problem.TitleContains(keyword),
				problem.DescriptionContains(keyword),
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
	changes, err := c.client.Change.Query().
		Where(
			change.TenantID(tenantID),
			change.Or(
				change.TitleContains(keyword),
				change.DescriptionContains(keyword),
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
	articles, err := c.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.Or(
				knowledgearticle.TitleContains(keyword),
				knowledgearticle.ContentContains(keyword),
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
func (c *GlobalSearchController) RegisterRoutes(r *gin.RouterGroup) {
	search := r.Group("/global-search")
	{
		search.GET("", c.Search)
	}
}
