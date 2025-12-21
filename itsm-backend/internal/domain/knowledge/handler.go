package knowledge

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// toArticleDTO maps domain Article to DTO
func toArticleDTO(a *Article) *dto.KnowledgeArticleResponse {
	if a == nil {
		return nil
	}
	return &dto.KnowledgeArticleResponse{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		Category:  a.Category,
		Tags:      a.Tags,
		TenantID:  a.TenantID,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// CreateArticle handles POST /api/v1/knowledge-articles
func (h *Handler) CreateArticle(c *gin.Context) {
	var req dto.CreateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	userIDVal, _ := c.Get("user_id")

	article := &Article{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		AuthorID: userIDVal.(int),
		TenantID: tenantIDVal.(int),
	}

	res, err := h.svc.CreateArticle(c.Request.Context(), article)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toArticleDTO(res))
}

// GetArticle handles GET /api/v1/knowledge-articles/:id
func (h *Handler) GetArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	res, err := h.svc.GetArticle(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Article not found")
		return
	}

	common.Success(c, toArticleDTO(res))
}

// ListArticles handles GET /api/v1/knowledge-articles
func (h *Handler) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	category := c.Query("category")
	search := c.Query("search")
	status := c.Query("status")
	tenantIDVal, _ := c.Get("tenant_id")

	list, total, err := h.svc.ListArticles(c.Request.Context(), tenantIDVal.(int), page, pageSize, category, search, status)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var dtos []dto.KnowledgeArticleResponse
	for _, item := range list {
		dtos = append(dtos, *toArticleDTO(item))
	}

	common.Success(c, dto.KnowledgeArticleListResponse{
		Articles: dtos,
		Total:    total,
		Page:     page,
		Size:     pageSize,
	})
}

// UpdateArticle handles PUT /api/v1/knowledge-articles/:id
func (h *Handler) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	var req dto.UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := h.svc.GetArticle(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Article not found")
		return
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Content != nil {
		existing.Content = *req.Content
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}

	res, err := h.svc.UpdateArticle(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toArticleDTO(res))
}

// DeleteArticle handles DELETE /api/v1/knowledge-articles/:id
func (h *Handler) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	if err := h.svc.DeleteArticle(c.Request.Context(), id, tenantIDVal.(int)); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetCategories handles GET /api/v1/knowledge-articles/categories
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")

	list, err := h.svc.GetCategories(c.Request.Context(), tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, list)
}
