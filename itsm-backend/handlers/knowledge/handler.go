package knowledge

import (
	"strconv"

	"itsm-backend/handlers/common/knowledgeaccess"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
	// freshness 时效性判定器，仅用于响应里给出时效状态描述（展示用途）。
	// 真正的准入判定发生在检索路径上，这里必须与 Service 用同一套策略，
	// 否则界面会显示「可引用」而检索实际把它挡掉（或反之）。
	freshness *knowledgeaccess.FreshnessJudger
}

func NewHandler(svc *Service) *Handler {
	h := &Handler{svc: svc}
	if svc != nil {
		h.freshness = svc.freshness
	}
	if h.freshness == nil {
		h.freshness = knowledgeaccess.NewFreshnessJudger(knowledgeaccess.DefaultFreshnessPolicy(), nil)
	}
	return h
}

// toArticleDTO maps domain Article to DTO。
// 做成方法而非包级函数，是为了让时效状态描述能跟随注入的策略，
// 而不是写死一份可能与检索口径不一致的默认值。
func (h *Handler) toArticleDTO(a *Article) *dto.KnowledgeArticleResponse {
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

		ValidFrom:          a.ValidFrom,
		ValidUntil:         a.ValidUntil,
		LastReviewedAt:     a.LastReviewedAt,
		ReviewIntervalDays: a.ReviewIntervalDays,
		AuthorityLevel:     a.AuthorityLevel,
		Freshness:          h.describeFreshness(a),
	}
}

// describeFreshness 计算文章当前时效状态，供前端提示「内容可能已过期」。
// 纯展示用途，不做准入判定——准入由检索路径上的 L1 过滤负责。
func (h *Handler) describeFreshness(a *Article) *dto.KnowledgeFreshness {
	if h.freshness == nil {
		return nil
	}
	f := knowledgeaccess.FreshnessFields{
		ValidFrom:          a.ValidFrom,
		ValidUntil:         a.ValidUntil,
		LastReviewedAt:     a.LastReviewedAt,
		ReviewIntervalDays: a.ReviewIntervalDays,
	}
	out := &dto.KnowledgeFreshness{
		Verdict: h.freshness.Judge(f).String(),
		Citable: h.freshness.Citable(f),
	}
	// 下次复核到期时间仅在声明了复核周期时才有意义；
	// 从未复核的文章没有基准时间，给不出到期日。
	if a.ReviewIntervalDays > 0 && a.LastReviewedAt != nil {
		due := a.LastReviewedAt.AddDate(0, 0, a.ReviewIntervalDays)
		out.ReviewDueAt = &due
	}
	return out
}

// CreateArticle handles POST /api/v1/knowledge-articles
func (h *Handler) CreateArticle(c *gin.Context) {
	var req dto.CreateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	userIDVal, ok := c.Get("user_id")
	if !ok {
		common.ParamError(c, "User ID not found")
		return
	}

	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}
	userID, ok := userIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid user ID")
		return
	}

	// 时效窗口自检：生效时间不早于失效时间属于配置错误，
	// 会让文章从创建起就永远不可被引用，而界面上看不出原因，必须在入口挡住。
	if req.ValidFrom != nil && req.ValidUntil != nil && !req.ValidFrom.Before(*req.ValidUntil) {
		common.ParamError(c, "生效时间必须早于失效时间")
		return
	}

	article := &Article{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
		AuthorID: userID,
		TenantID: tenantID,

		// 时效性与权威性均可选：留空表示「长期有效、不设复核、普通权威度」。
		ValidFrom:  req.ValidFrom,
		ValidUntil: req.ValidUntil,
	}
	if req.ReviewIntervalDays != nil {
		article.ReviewIntervalDays = *req.ReviewIntervalDays
	}
	if req.AuthorityLevel != nil {
		article.AuthorityLevel = *req.AuthorityLevel
	}

	res, err := h.svc.CreateArticle(c.Request.Context(), article)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// GetArticle handles GET /api/v1/knowledge-articles/:id
func (h *Handler) GetArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	res, err := h.svc.GetArticle(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Article not found")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// ListArticles handles GET /api/v1/knowledge-articles
func (h *Handler) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	category := c.Query("category")
	search := c.Query("search")
	status := c.Query("status")

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	list, total, err := h.svc.ListArticles(c.Request.Context(), tenantID, page, pageSize, category, search, status)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	var dtos []dto.KnowledgeArticleResponse
	for _, item := range list {
		dtos = append(dtos, *h.toArticleDTO(item))
	}

	common.Success(c, dto.KnowledgeArticleListResponse{
		Articles: dtos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// UpdateArticle handles PUT /api/v1/knowledge-articles/:id
func (h *Handler) UpdateArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	var req dto.UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	existing, err := h.svc.GetArticle(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Article not found")
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

	// 时效性与权威性：部分更新语义，不传即保持原值。
	if req.ValidFrom != nil {
		existing.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		existing.ValidUntil = req.ValidUntil
	}
	if req.ReviewIntervalDays != nil {
		existing.ReviewIntervalDays = *req.ReviewIntervalDays
	}
	if req.AuthorityLevel != nil {
		existing.AuthorityLevel = *req.AuthorityLevel
	}
	// 合并后再自检一次：分两次请求分别改 valid_from 和 valid_until 时，
	// 单看每次都合法，合起来却可能倒挂，导致文章永远不可引用。
	if existing.ValidFrom != nil && existing.ValidUntil != nil &&
		!existing.ValidFrom.Before(*existing.ValidUntil) {
		common.ParamError(c, "生效时间必须早于失效时间")
		return
	}

	res, err := h.svc.UpdateArticle(c.Request.Context(), existing)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// MarkArticleReviewed handles POST /api/v1/knowledge/articles/:id/review
//
// 记录一次内容复核：把 last_reviewed_at 置为当前时间，用于解除「逾期未复核」状态。
// 这是 L1 时效管控的闭环动作——逾期知识会被 RAG 检索过滤掉，
// 复核是把它恢复回检索结果的唯一途径。
//
// 刻意不提供由客户端写入 last_reviewed_at 的能力：
// 一旦可写，就能直接把时间改成未来值来消除逾期提醒，L1 会形同虚设。
func (h *Handler) MarkArticleReviewed(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	res, err := h.svc.MarkArticleReviewed(c.Request.Context(), id, tenantID)
	if err != nil {
		// 不存在与已软删统一按未找到返回，避免暴露资源存在性
		common.NotFound(c, "Article not found")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// PublishArticle handles POST /api/v1/knowledge/articles/:id/publish
func (h *Handler) PublishArticle(c *gin.Context) {
	h.setArticlePublished(c, true)
}

// UnpublishArticle handles POST /api/v1/knowledge/articles/:id/unpublish
func (h *Handler) UnpublishArticle(c *gin.Context) {
	h.setArticlePublished(c, false)
}

func (h *Handler) setArticlePublished(c *gin.Context, published bool) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	article, err := h.svc.GetArticle(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFound(c, "Article not found")
		return
	}

	article.IsPublished = published
	res, err := h.svc.UpdateArticle(c.Request.Context(), article)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// DeleteArticle handles DELETE /api/v1/knowledge-articles/:id
func (h *Handler) DeleteArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	if err := h.svc.DeleteArticle(c.Request.Context(), id, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
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

// ListRestrictedCategories handles GET /api/v1/knowledge/categories/restricted
// 返回当前租户已被纳管（仅指定角色可读）的知识分类清单。
func (h *Handler) ListRestrictedCategories(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.ParamError(c, "租户信息缺失")
		return
	}

	restricted, err := h.svc.ListRestrictedCategories(c.Request.Context(), tenantID)
	if err != nil {
		h.svc.logger.Warnw("获取受限知识分类失败", "error", err, "tenant_id", tenantID)
		common.InternalError(c, "获取受限分类失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"categories": restricted, "total": len(restricted)})
}

// SetCategoryRestriction handles POST /api/v1/knowledge/categories/restricted
// 纳管或解除纳管某个知识分类。body: { "category": "财务制度", "restricted": true }
func (h *Handler) SetCategoryRestriction(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.ParamError(c, "租户信息缺失")
		return
	}

	var req struct {
		Category   string `json:"category" binding:"required"`
		Restricted *bool  `json:"restricted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	if req.Restricted == nil {
		enabled := true
		req.Restricted = &enabled
	}

	if err := h.svc.SetCategoryRestriction(c.Request.Context(), tenantID, req.Category, *req.Restricted); err != nil {
		h.svc.logger.Warnw("设置知识分类可见性失败",
			"error", err, "tenant_id", tenantID, "category", req.Category)
		common.InternalError(c, "设置失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"category": req.Category, "restricted": *req.Restricted})
}

// SearchArticles handles POST /api/v1/knowledge/search
func (h *Handler) SearchArticles(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantIDInt, ok := tenantIDVal.(int)
	if !ok || tenantIDInt == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 注入访问者身份：知识检索据此做分类级可见性过滤（L0 权限边界）。
	// 不注入则按匿名处理，已纳管的受限分类一律不可见（fail-closed）。
	searchCtx := knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{
		UserID: c.GetInt("user_id"),
		Role:   c.GetString("role"),
	})

	articles, err := h.svc.SearchArticles(searchCtx, tenantIDInt, req.Query, req.Category, limit)
	if err != nil {
		common.InternalError(c, "搜索失败: "+err.Error())
		return
	}

	items := make([]interface{}, 0, len(articles))
	for _, a := range articles {
		items = append(items, map[string]interface{}{
			"id":              a.ID,
			"title":           a.Title,
			"category":        a.Category,
			"snippet":         snippet(a.Content, 200),
			"tags":            a.Tags,
			"is_published":    a.IsPublished,
			"relevance_score": a.RelevanceScore,
			"created_at":      a.CreatedAt,
			"updated_at":      a.UpdatedAt,
		})
	}

	common.Success(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetRecommendations handles GET /api/v1/knowledge/recommendations
func (h *Handler) GetRecommendations(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok || tenantID == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := 5
	articles, _, err := h.svc.ListArticles(c.Request.Context(), tenantID, 1, limit, "", "", "published")
	if err != nil {
		common.InternalError(c, "获取推荐文章失败: "+err.Error())
		return
	}

	items := make([]interface{}, 0, len(articles))
	for _, a := range articles {
		items = append(items, h.toArticleDTO(a))
	}
	common.Success(c, items)
}

// GetRecentArticles handles GET /api/v1/knowledge/recent
func (h *Handler) GetRecentArticles(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok || tenantID == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := 10
	articles, _, err := h.svc.ListArticles(c.Request.Context(), tenantID, 1, limit, "", "", "published")
	if err != nil {
		common.InternalError(c, "获取最近文章失败: "+err.Error())
		return
	}

	items := make([]interface{}, 0, len(articles))
	for _, a := range articles {
		items = append(items, h.toArticleDTO(a))
	}
	common.Success(c, items)
}

// GetCategories handles GET /api/v1/knowledge-articles/categories
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	list, err := h.svc.GetCategories(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, list)
}

// GetStats handles GET /api/v1/knowledge/stats
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.ParamError(c, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	stats, err := h.svc.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}
