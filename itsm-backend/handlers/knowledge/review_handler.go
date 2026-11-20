package knowledge

import (
	"strings"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"

	"github.com/gin-gonic/gin"
)

// reviewRequest 是评审动作的请求体。
type reviewRequest struct {
	Comment string `json:"comment"`
}

// reviewResponder 抽出的公共执行逻辑。
//
// 五个动作（submit/approve/reject/withdraw/republish）的 HTTP 层完全同构：
// 取文章 ID → 取操作人 → 调 service。差异只在 service 内的状态机，
// 因此这里用闭包参数复用，避免五份几乎一样的 handler。
func (h *Handler) reviewHandler(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := common.ParsePositiveID(c, "id")
		if !ok {
			return
		}

		tenantID, ok := handlerctx.ResolveTenantID(c)
		if !ok {
			return
		}

		// 操作人取鉴权上下文，不信任请求体 —— 否则可以冒充他人评审。
		operatorID := 0
		if v, exists := c.Get("user_id"); exists {
			operatorID, _ = v.(int)
		}
		if operatorID <= 0 {
			common.Fail(c, common.UnauthorizedCode, "未授权访问")
			return
		}

		var req reviewRequest
		// 允许空 body：submit/withdraw 无需意见。只有解析失败才算参数错误。
		if err := c.ShouldBindJSON(&req); err != nil && !strings.Contains(err.Error(), "EOF") {
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}

		ctx := h.requestContext(c)
		var (
			article *Article
			err     error
		)
		switch action {
		case ReviewActionSubmit:
			article, err = h.svc.SubmitReview(ctx, id, tenantID, operatorID, req.Comment)
		case ReviewActionApprove:
			article, err = h.svc.ApproveReview(ctx, id, tenantID, operatorID, req.Comment)
		case ReviewActionReject:
			article, err = h.svc.RejectReview(ctx, id, tenantID, operatorID, req.Comment)
		case ReviewActionWithdraw:
			article, err = h.svc.WithdrawReview(ctx, id, tenantID, operatorID, req.Comment)
		case ReviewActionRepubish:
			article, err = h.svc.RepublishReview(ctx, id, tenantID, operatorID, req.Comment)
		default:
			common.Fail(c, common.NotImplementedCode, "未实现的评审动作: "+action)
			return
		}

		if err != nil {
			// 状态机拒绝（当前状态不允许）由 service 返回 409 型错误，
			// RespondError 负责映射，不要在此吞成 500。
			common.RespondError(c, err, "评审操作失败")
			return
		}

		common.Success(c, article)
	}
}

// SubmitReview POST /knowledge/articles/:id/review/submit
// @Summary 提交评审
// @Description 作者把文章提交送审；当前状态机不允许该动作时返回 409。需要 knowledge:write 权限。
// @Tags 知识评审
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body object false "{comment?: string}"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 404 "文章不存在"
// @Failure 409 "状态机拒绝"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review/submit [post]
func (h *Handler) SubmitReview(c *gin.Context) { h.reviewHandler(ReviewActionSubmit)(c) }

// ApproveReview POST /knowledge/articles/:id/review/approve
// @Summary 通过评审
// @Description 评审人通过文章评审；操作人取鉴权上下文，不信任请求体（防冒充）。需要 knowledge:write 权限。
// @Tags 知识评审
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body object false "{comment?: string}"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 403 "非评审人"
// @Failure 409 "状态机拒绝"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review/approve [post]
func (h *Handler) ApproveReview(c *gin.Context) { h.reviewHandler(ReviewActionApprove)(c) }

// RejectReview POST /knowledge/articles/:id/review/reject
// @Summary 驳回评审
// @Description 评审人驳回文章，必须给出理由（review comment 为空返回 400）。需要 knowledge:write 权限。
// @Tags 知识评审
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body object true "{comment: string(必填)}"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "缺少驳回理由"
// @Failure 403 "非评审人"
// @Failure 409 "状态机拒绝"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review/reject [post]
func (h *Handler) RejectReview(c *gin.Context) { h.reviewHandler(ReviewActionReject)(c) }

// WithdrawReview POST /knowledge/articles/:id/review/withdraw
// @Summary 撤回送审
// @Description 作者撤回已提交但未完成的评审。需要 knowledge:write 权限。
// @Tags 知识评审
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body object false "{comment?: string}"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 404 "文章不存在"
// @Failure 409 "状态机拒绝"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review/withdraw [post]
func (h *Handler) WithdrawReview(c *gin.Context) { h.reviewHandler(ReviewActionWithdraw)(c) }

// RepublishReview POST /knowledge/articles/:id/review/republish
// @Summary 重新送审
// @Description 已通过评审的内容被修改后需重新送审；状态机拒绝非法重提。需要 knowledge:write 权限。
// @Tags 知识评审
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body object false "{comment?: string}"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 "参数错误"
// @Failure 404 "文章不存在"
// @Failure 409 "状态机拒绝"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/review/republish [post]
func (h *Handler) RepublishReview(c *gin.Context) { h.reviewHandler(ReviewActionRepubish)(c) }

// ListArticleReviews GET /knowledge/articles/:id/reviews
// @Summary 查询评审历史
// @Description 时间倒序返回文章的评审流水（提交/通过/驳回/撤回及操作人与意见）；不可读维持 404 防枚举，DB 故障落 500。
// @Tags 知识评审
// @Produce json
// @Param id path int true "文章ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response{data=dto.KnowledgeReviewResponse} "分页信封"
// @Failure 400 "无效ID"
// @Failure 404 "不存在或无权访问"
// @Failure 500 "后端故障"
// @Security BearerAuth
// @Router /api/v1/knowledge/articles/{id}/reviews [get]
func (h *Handler) ListArticleReviews(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	pg := common.GetPaginationFromQuery(c)
	records, total, err := h.svc.ListReviews(h.requestContext(c), id, tenantID, pg.Page, pg.PageSize)
	if err != nil {
		// 不存在/分类不可读 → 404（防枚举）；DB 故障 → 500，不伪装成 not found。
		knowledgeArticleError(c, err)
		return
	}

	common.SuccessWithPagination(c, records, pg.Page, pg.PageSize, int64(total))
}
