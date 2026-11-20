package knowledge

import (
	"context"
	"errors"
	"strings"

	"itsm-backend/internal/knowledgeaccess"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
	// freshness 时效性判定器，仅用于响应里给出时效状态描述（展示用途）。
	// 真正的准入判定发生在检索路径上，这里必须与 Service 用同一套策略，
	// 否则界面会显示「可引用」而检索实际把它挡掉（或反之）。
	freshness *knowledgeaccess.FreshnessJudger
}

// requestContext 在请求 context 上注入访问者身份，供分类级可见性判定。
// 读写文章、版本、评审的入口都必须使用它：缺少 Viewer 会被按匿名处理，
// 已纳管分类对合法用户也会变成 not-found。
func (h *Handler) requestContext(c *gin.Context) context.Context {
	return knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{
		UserID: c.GetInt("user_id"),
		Role:   c.GetString("role"),
	})
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
		ID:          a.ID,
		Title:       a.Title,
		Content:     a.Content,
		Category:    a.Category,
		Author:      a.AuthorName,
		Tags:        a.Tags,
		Status:      status,
		IsPublished: a.IsPublished,
		TenantID:    a.TenantID,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,

		// 评审状态与发布状态各自独立透出：status 只回答「是否对外可见」，
		// 发布门禁与前端评审按钮看的是 reviewStatus。
		ReviewStatus: NormalizeReviewStatus(a.ReviewStatus),

		ValidFrom:          a.ValidFrom,
		ValidUntil:         a.ValidUntil,
		LastReviewedAt:     a.LastReviewedAt,
		ReviewIntervalDays: a.ReviewIntervalDays,
		AuthorityLevel:     a.AuthorityLevel,
		Freshness:          h.describeFreshness(a),

		SourceProblemID: a.SourceProblemID,
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
// @Summary 创建知识文章
// @Description 新建知识文章（草稿）。分类可见性在读取路径按访问者身份过滤；需要 knowledge:write 权限。兼容旧路径 POST /api/v1/knowledge-articles。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param body body dto.CreateKnowledgeArticleRequest true "文章创建请求"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles [post]
func (h *Handler) CreateArticle(c *gin.Context) {
	var req dto.CreateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	userIDVal, ok := c.Get("user_id")
	if !ok {
		common.ParamError(c, "User ID not found")
		return
	}

	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
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
// @Summary 获取知识文章
// @Description 按 ID 查询文章；不存在与分类不可读统一返回 404（防枚举，调用方无法探测受限文章是否存在），DB 故障返回 500。兼容旧路径 GET /api/v1/knowledge-articles/{id}。
// @Tags 知识管理
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "无效ID"
// @Failure 401 "上下文缺失"
// @Failure 404 "不存在或无权读取"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id} [get]
func (h *Handler) GetArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	res, err := h.svc.GetArticle(h.requestContext(c), id, tenantID)
	if err != nil {
		// 不存在/分类不可读 → 404（防枚举）；DB 故障 → 500，不得伪装成「文章不存在」。
		knowledgeArticleError(c, err)
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// ListArticles handles GET /api/v1/knowledge-articles
// @Summary 查询知识文章列表
// @Description 分页列出文章，按分类可见性与时效策略过滤后计数（先过滤再分页，total 为当前访问者真正可见条数）；空列表返回 [] 而非 null。兼容旧路径 GET /api/v1/knowledge-articles。
// @Tags 知识管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Param category query string false "分类筛选"
// @Param search query string false "关键字"
// @Param status query string false "状态筛选(draft|published)"
// @Success 200 {object} common.Response "data 含 items/total/page/pageSize/totalPages"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles [get]
func (h *Handler) ListArticles(c *gin.Context) {
	pg := common.GetPaginationFromQuery(c)
	category := c.Query("category")
	search := c.Query("search")
	status := c.Query("status")

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	list, total, err := h.svc.ListArticles(h.requestContext(c), tenantID, pg.Page, pg.PageSize, category, search, status)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 空列表必须是 []，不能是 null：前端列表页直接 map 渲染
	dtos := make([]dto.KnowledgeArticleResponse, 0, len(list))
	for _, item := range list {
		dtos = append(dtos, *h.toArticleDTO(item))
	}

	// 标准信封：data.items + total/page/pageSize/totalPages
	common.SuccessWithPagination(c, dtos, pg.Page, pg.PageSize, int64(total))
}

// UpdateArticle handles PUT /api/v1/knowledge-articles/:id
// @Summary 更新知识文章
// @Description 局部更新文章字段（仅请求体中非 nil 字段生效）；前置读取与 GetArticle 同口径：不可读维持 404 防枚举，DB 故障落 500。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body dto.UpdateKnowledgeArticleRequest true "文章更新请求"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 404 "不存在或无权访问"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id} [put]
func (h *Handler) UpdateArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	var req dto.UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "Invalid request body")
		return
	}

	existing, err := h.svc.GetArticle(h.requestContext(c), id, tenantID)
	if err != nil {
		// 与 GetArticle 同口径：防枚举的 404 保留，DB 故障不再伪装成 not found。
		knowledgeArticleError(c, err)
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

	res, err := h.svc.UpdateArticle(h.requestContext(c), existing)
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
// @Summary 记录内容复核
// @Description 确认文章内容仍然适用，更新 last_reviewed_at 以解除「逾期未复核」的检索过滤；只改复核时间不触碰正文。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "无效ID"
// @Failure 404 "不存在或无权访问"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review [post]
func (h *Handler) MarkArticleReviewed(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	res, err := h.svc.MarkArticleReviewed(h.requestContext(c), id, tenantID)
	if err != nil {
		// 不存在/分类不可读 → 404（防枚举）；DB 故障 → 500。
		knowledgeArticleError(c, err)
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// PublishArticle handles POST /api/v1/knowledge/articles/:id/publish
// @Summary 发布知识文章
// @Description 发布前校验 review_status=approved，未通过返回 409；通过后同事务落版本快照并把线上版本指针指向它。需要 knowledge:write 权限。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "无效ID"
// @Failure 404 "不存在或无权访问"
// @Failure 409 "评审未通过"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/publish [post]
func (h *Handler) PublishArticle(c *gin.Context) {
	h.setArticlePublished(c, true)
}

// UnpublishArticle handles POST /api/v1/knowledge/articles/:id/unpublish
// @Summary 撤回知识文章
// @Description 把文章从可见面撤下并清除线上版本指针；撤回不设评审门槛（撤下永远是安全方向）。需要 knowledge:write 权限。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "无效ID"
// @Failure 404 "不存在或无权访问"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/unpublish [post]
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
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	// 发布门禁在服务层：review_status != approved 时返回 409/4090，
	// 不让 HTTP 入口成为绕过评审的旁路（见 Service.SetArticlePublished）。
	res, err := h.svc.SetArticlePublished(h.requestContext(c), id, tenantID, published)
	if err != nil {
		// ent.NotFoundError → 4004/404；评审未通过 → 4090/409。
		common.RespondError(c, err, "操作失败")
		return
	}

	common.Success(c, h.toArticleDTO(res))
}

// DeleteArticle handles DELETE /api/v1/knowledge-articles/:id
// @Summary 删除知识文章
// @Description 软删除文章并同步移除向量索引；删除后不再出现在检索结果。需要 knowledge:delete 权限。
// @Tags 知识管理
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response
// @Failure 400 "无效ID"
// @Failure 404 "不存在或无权访问"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id} [delete]
func (h *Handler) DeleteArticle(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	if err := h.svc.DeleteArticle(c.Request.Context(), id, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// GetArticleComments handles GET /api/v1/knowledge/articles/:id/comments
// 评论尚未落地（无存储模型）：显式返回 unready，而不是空成功伪装成"暂无评论"。
// @Summary 查询文章评论
// @Description 文章评论能力当前未落地（无存储模型），端点显式返回 503/5003(unready)，不会用空列表伪装「暂无评论」。
// @Tags 知识管理
// @Produce json
// @Param id path int true "文章ID"
// @Success 503 {object} common.Response "5003 能力未实现"
// @Failure 503 "能力未实现(5003)"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/comments [get]
func (h *Handler) GetArticleComments(c *gin.Context) {
	common.Fail(c, common.ServiceUnavailableCode, "知识文章评论功能尚未实现")
}

// AddArticleComment handles POST /api/v1/knowledge/articles/:id/comments
// @Summary 添加文章评论
// @Description 文章评论能力当前未落地（无存储模型），端点显式返回 503/5003(unready)，不会返回假评论对象。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 503 {object} common.Response "5003 能力未实现"
// @Failure 503 "能力未实现(5003)"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/comments [post]
func (h *Handler) AddArticleComment(c *gin.Context) {
	common.Fail(c, common.ServiceUnavailableCode, "知识文章评论功能尚未实现")
}

// ListRestrictedCategories handles GET /api/v1/knowledge/categories/restricted
// 返回当前租户已被纳管（仅指定角色可读）的知识分类清单。
// @Summary 查询受限分类
// @Description 列出当前租户被纳管为「仅授权角色可读」的知识分类（L0 权限边界）。需要 knowledge:write 权限。
// @Tags 知识管理
// @Produce json
// @Success 200 {object} common.Response "分类数组"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/categories/restricted [get]
func (h *Handler) ListRestrictedCategories(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.ParamError(c, "租户信息缺失")
		return
	}

	restricted, err := h.svc.ListRestrictedCategories(c.Request.Context(), tenantID)
	if err != nil {
		h.svc.logger.Warnw("获取受限知识分类失败", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "获取受限分类失败")
		return
	}

	common.Success(c, gin.H{"items": restricted, "total": len(restricted)})
}

// SetCategoryRestriction handles POST /api/v1/knowledge/categories/restricted
// 纳管或解除纳管某个知识分类。body: { "category": "财务制度", "restricted": true }
// @Summary 设置分类可见性
// @Description 将指定分类标记为受限/非受限（restricted 省略时默认 true）；纳管后该分类对未授权访问者按不可读处理。需要 knowledge:write 权限。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param body body object true "{category: string, restricted?: bool}"
// @Success 200 {object} common.Response "{category, restricted}"
// @Failure 400 "参数错误"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/categories/restricted [post]
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
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	if req.Restricted == nil {
		enabled := true
		req.Restricted = &enabled
	}

	if err := h.svc.SetCategoryRestriction(c.Request.Context(), tenantID, req.Category, *req.Restricted); err != nil {
		h.svc.logger.Warnw("设置知识分类可见性失败",
			"error", err, "tenant_id", tenantID, "category", req.Category)
		common.FailWithErr(c, err, "设置失败")
		return
	}

	common.Success(c, gin.H{"category": req.Category, "restricted": *req.Restricted})
}

// SearchArticles handles POST /api/v1/knowledge/search
// @Summary 检索知识文章
// @Description 按关键字检索并做分类级可见性过滤（L0）；pageSize 夹紧到 (0,50]，缺省 20。请求须注入访问者身份，否则受限分类一律不可见（fail-closed）。
// @Tags 知识管理
// @Accept json
// @Produce json
// @Param body body object true "{query: string(必填), category?: string, pageSize?: int}"
// @Success 200 {object} common.Response{data=dto.KnowledgeSearchResponse}
// @Failure 400 "参数错误"
// @Failure 401 "上下文缺失"
// @Failure 500 "检索失败"
// @Security BearerAuth
// @Router /api/v1/knowledge/search [post]
func (h *Handler) SearchArticles(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		Category string `json:"category"`
		PageSize int    `json:"pageSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantIDInt, ok := tenantIDVal.(int)
	if !ok || tenantIDInt == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := req.PageSize
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 注入访问者身份：知识检索据此做分类级可见性过滤（L0 权限边界）。
	// 不注入则按匿名处理，已纳管的受限分类一律不可见（fail-closed）。
	searchCtx := h.requestContext(c)

	articles, err := h.svc.SearchArticles(searchCtx, tenantIDInt, req.Query, req.Category, limit)
	if err != nil {
		common.FailWithErr(c, err, "搜索失败")
		return
	}

	items := make([]dto.KnowledgeSearchItem, 0, len(articles))
	for _, a := range articles {
		items = append(items, dto.KnowledgeSearchItem{
			ID:             a.ID,
			Title:          a.Title,
			Category:       a.Category,
			Snippet:        snippet(a.Content, 200),
			Tags:           a.Tags,
			IsPublished:    a.IsPublished,
			RelevanceScore: a.RelevanceScore,
			CreatedAt:      a.CreatedAt,
			UpdatedAt:      a.UpdatedAt,
		})
	}

	// 检索是 top-N，不是分页遍历：total 只代表本次返回条数，
	// 是否还有更多由 truncated 明示（items 打满 limit 即视为被截断）。
	common.Success(c, dto.KnowledgeSearchResponse{
		Items:     items,
		Total:     len(items),
		Limit:     limit,
		Truncated: len(items) >= limit,
	})
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetRecommendations handles GET /api/v1/knowledge/recommendations
// @Summary 推荐知识文章
// @Description 基于时效策略与访问者可见性返回推荐文章（Citable 的未过期内容），已纳管的受限分类不可见。需要 knowledge:read 权限。
// @Tags 知识管理
// @Produce json
// @Success 200 {object} common.Response "文章数组"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/recommendations [get]
func (h *Handler) GetRecommendations(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok || tenantID == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := 5
	articles, _, err := h.svc.ListArticles(h.requestContext(c), tenantID, 1, limit, "", "", "published")
	if err != nil {
		common.FailWithErr(c, err, "获取推荐文章失败")
		return
	}

	items := make([]interface{}, 0, len(articles))
	for _, a := range articles {
		items = append(items, h.toArticleDTO(a))
	}
	common.Success(c, items)
}

// GetRecentArticles handles GET /api/v1/knowledge/recent
// @Summary 最近知识文章
// @Description 列出最近更新的文章，同样经分类可见性过滤。需要 knowledge:read 权限。
// @Tags 知识管理
// @Produce json
// @Success 200 {object} common.Response "文章数组"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/recent [get]
func (h *Handler) GetRecentArticles(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok || tenantID == 0 {
		common.ParamError(c, "Invalid tenant ID")
		return
	}

	limit := 10
	articles, _, err := h.svc.ListArticles(h.requestContext(c), tenantID, 1, limit, "", "", "published")
	if err != nil {
		common.FailWithErr(c, err, "获取最近文章失败")
		return
	}

	items := make([]interface{}, 0, len(articles))
	for _, a := range articles {
		items = append(items, h.toArticleDTO(a))
	}
	common.Success(c, items)
}

// GetCategories handles GET /api/v1/knowledge-articles/categories
// @Summary 知识文章分类
// @Description 列出当前租户可见的文章分类（受限分类按权限剔除）。兼容旧路径 GET /api/v1/knowledge-articles/categories。
// @Tags 知识管理
// @Produce json
// @Success 200 {object} common.Response "分类数组"
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/categories [get]
func (h *Handler) GetCategories(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	list, err := h.svc.GetCategories(h.requestContext(c), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, list)
}

// GetStats handles GET /api/v1/knowledge/stats
// @Summary 知识库统计
// @Description 统计当前可见范围内文章总数、已发布/草稿数、浏览与点赞数及分类分布；统计读取失败 fail-closed 返回 500。
// @Tags 知识管理
// @Produce json
// @Success 200 {object} common.Response{data=dto.KnowledgeStatsResponse}
// @Failure 401 "上下文缺失"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		// 租户上下文类型错误本质是认证/上下文问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return
	}

	stats, err := h.svc.GetStats(h.requestContext(c), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}

// ListArticleVersions handles GET /api/v1/knowledge/articles/:id/versions.
// 版本历史列表（版本号倒序）。2026-10-14 前该端点未注册，前端版本 Tab 打开即报错。
// @Summary 查询版本历史
// @Description 按版本号倒序列出文章快照；版本表无 tenant_id，租户隔离经文章归属校验实现。
// @Tags 知识版本
// @Produce json
// @Param id path int true "文章ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response "分页信封"
// @Failure 400 "无效ID"
// @Failure 404 "文章不存在"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/versions [get]
func (h *Handler) ListArticleVersions(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}

	pg := common.GetPaginationFromQuery(c)

	versions, _, err := h.svc.ListVersions(h.requestContext(c), id, tenantID, pg.Page, pg.PageSize)
	if err != nil {
		knowledgeVersionError(c, err)
		return
	}
	common.Success(c, versions)
}

// RestoreArticleVersion handles POST /api/v1/knowledge/articles/:id/versions/:version/restore.
// @Summary 恢复历史版本
// @Description 先用当前正文建新快照，再把正文覆写为指定版本内容；恢复动作同样留下新版本，历史链不断。需要 knowledge:write 权限。
// @Tags 知识版本
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param version path int true "版本号"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 404 "文章或版本不存在"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/versions/{version}/restore [post]
func (h *Handler) RestoreArticleVersion(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	version, ok := common.ParsePositiveID(c, "version")
	if !ok {
		return
	}
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")

	article, err := h.svc.RestoreVersion(h.requestContext(c), id, version, tenantID, userID)
	if err != nil {
		knowledgeVersionError(c, err)
		return
	}
	common.Success(c, h.toArticleDTO(article))
}

// CompareArticleVersions handles GET /api/v1/knowledge/articles/:id/versions/compare?from&to.
// @Summary 对比版本差异
// @Description 按行级 LCS 对比 from/to 两个版本的正文，返回渲染后的 diff 与变更行列表。
// @Tags 知识版本
// @Produce json
// @Param id path int true "文章ID"
// @Param from query int true "起始版本号"
// @Param to query int true "目标版本号"
// @Success 200 {object} common.Response "{diff, changes}"
// @Failure 400 "参数错误"
// @Failure 404 "文章或版本不存在"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/versions/compare [get]
func (h *Handler) CompareArticleVersions(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	// from/to 在 query 上，不是 path 参数（路由只有 :id）。
	// 曾误用 ParsePositiveID（读 c.Param），导致本端点永久 400 from is required。
	from, ok := common.ParseRequiredPositiveIDFromQuery(c, "from")
	if !ok {
		return
	}
	to, ok := common.ParseRequiredPositiveIDFromQuery(c, "to")
	if !ok {
		return
	}

	fromVersion, err := h.svc.GetVersion(h.requestContext(c), id, from, tenantID)
	if err != nil {
		knowledgeVersionError(c, err)
		return
	}
	toVersion, err := h.svc.GetVersion(h.requestContext(c), id, to, tenantID)
	if err != nil {
		knowledgeVersionError(c, err)
		return
	}

	changes := diffLines(fromVersion.Content, toVersion.Content)
	common.Success(c, gin.H{
		"diff":    renderLineDiff(changes),
		"changes": changes,
	})
}

// tenantIDFromContext 从 gin 上下文取租户 ID，缺失/类型错误统一 401。
func tenantIDFromContext(c *gin.Context) (int, bool) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		// 租户上下文缺失属认证问题，统一为 401。
		common.Fail(c, common.AuthFailedCode, "Tenant ID not found")
		return 0, false
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok {
		common.Fail(c, common.AuthFailedCode, "Invalid tenant ID")
		return 0, false
	}
	return tenantID, true
}

// respondKnowledgeError 是知识域「读/写文章」错误的统一分流，单一事实源：
//   - ErrArticleNotFound（分类不可读的防枚举伪装）与 ent.NotFoundError → 404，
//     两者同码同文案，调用方无法据此探测受限文章是否存在；
//   - 其余（含 DB 故障、上下文错误）→ common.RespondError 归 500/5001，
//     原始 driver 错误只进日志、不进响应。
//
// 注意：绝不能只写 common.RespondError——ErrArticleNotFound 是裸 errors.New，
// 未被 classifyError 识别，会被兜成 500，从而泄露「文章存在但不可读」。
func respondKnowledgeError(c *gin.Context, err error, notFoundMsg, fallbackMsg string) {
	if errors.Is(err, ErrArticleNotFound) || ent.IsNotFound(err) {
		common.NotFound(c, notFoundMsg)
		return
	}
	common.RespondError(c, err, fallbackMsg)
}

// knowledgeVersionError 版本操作错误映射：不存在/非本租户 → 404，其余按错误分类映射，
// 未分类的落 500。原始错误串只进日志，不进响应。
func knowledgeVersionError(c *gin.Context, err error) {
	respondKnowledgeError(c, err, "Article or version not found", "版本操作失败")
}

// knowledgeArticleError 文章读/前置校验错误映射，与版本路径共用同一分流：
// 不可读/不存在维持 404（防枚举），DB 故障落 500，不再被误报为「文章不存在」。
func knowledgeArticleError(c *gin.Context, err error) {
	respondKnowledgeError(c, err, "Article not found", "操作失败")
}

// diffChange 单行差异：type 为 added / removed。
type diffChange struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// diffLines 以行为单位对比两段文本（LCS），返回增删序列。
// 知识文章版本对比只需行级可读性，不引入三方 diff 依赖。
func diffLines(from, to string) []diffChange {
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")

	// LCS 长度表
	m, n := len(fromLines), len(toLines)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if fromLines[i] == toLines[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	changes := make([]diffChange, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case fromLines[i] == toLines[j]:
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			changes = append(changes, diffChange{Type: "removed", Content: fromLines[i]})
			i++
		default:
			changes = append(changes, diffChange{Type: "added", Content: toLines[j]})
			j++
		}
	}
	for ; i < m; i++ {
		changes = append(changes, diffChange{Type: "removed", Content: fromLines[i]})
	}
	for ; j < n; j++ {
		changes = append(changes, diffChange{Type: "added", Content: toLines[j]})
	}
	return changes
}

// renderLineDiff 把增删序列渲染为 unified diff 风格文本。
func renderLineDiff(changes []diffChange) string {
	var sb strings.Builder
	for _, ch := range changes {
		switch ch.Type {
		case "added":
			sb.WriteString("+ " + ch.Content + "\n")
		case "removed":
			sb.WriteString("- " + ch.Content + "\n")
		}
	}
	return sb.String()
}
