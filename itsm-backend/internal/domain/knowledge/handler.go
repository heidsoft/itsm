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
	status := "draft"
	if a.IsPublished {
		status = "published"
	}
	return &dto.KnowledgeArticleResponse{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		Category:  a.Category,
		Tags:      a.Tags,
		Status:    status,
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

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	userIDVal, ok := c.Get("user_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "User ID not found")
		return
	}

	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	article := &Article{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		AuthorID: userID,
		TenantID: tenantID,
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid article ID")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	res, err := h.svc.GetArticle(c.Request.Context(), id, tenantID)
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

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	list, total, err := h.svc.ListArticles(c.Request.Context(), tenantID, page, pageSize, category, search, status)
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid article ID")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req dto.UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := h.svc.GetArticle(c.Request.Context(), id, tenantID)
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
	if req.Status != nil {
		existing.IsPublished = *req.Status == "published"
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid article ID")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	if err := h.svc.DeleteArticle(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetArticleComments handles GET /api/v1/knowledge/articles/:id/comments
func (h *Handler) GetArticleComments(c *gin.Context) {
	// Stub implementation
	common.Success(c, gin.H{
		"comments": []interface{}{},
		"total":    0,
	})
}

// AddArticleComment handles POST /api/v1/knowledge/articles/:id/comments
func (h *Handler) AddArticleComment(c *gin.Context) {
	// Stub implementation
	common.Success(c, gin.H{
		"id":        "stub_comment_id",
		"content":   "This is a stub comment",
		"createdAt": "2024-01-01T00:00:00Z",
	})
}

// SearchArticles handles POST /api/v1/knowledge/search
func (h *Handler) SearchArticles(c *gin.Context) {
	// Stub implementation
	common.Success(c, gin.H{
		"items": []interface{}{},
		"total": 0,
	})
}

// GetRecommendations handles GET /api/v1/knowledge/recommendations
func (h *Handler) GetRecommendations(c *gin.Context) {
	// Stub implementation
	common.Success(c, []interface{}{})
}

// GetRecentArticles handles GET /api/v1/knowledge/recent
func (h *Handler) GetRecentArticles(c *gin.Context) {
	// Stub implementation
	common.Success(c, []interface{}{})
}

// GetCategories handles GET /api/v1/knowledge-articles/categories
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	list, err := h.svc.GetCategories(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, list)
}
