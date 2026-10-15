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

		ctx := c.Request.Context()
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
func (h *Handler) SubmitReview(c *gin.Context) { h.reviewHandler(ReviewActionSubmit)(c) }

// ApproveReview POST /knowledge/articles/:id/review/approve
func (h *Handler) ApproveReview(c *gin.Context) { h.reviewHandler(ReviewActionApprove)(c) }

// RejectReview POST /knowledge/articles/:id/review/reject
func (h *Handler) RejectReview(c *gin.Context) { h.reviewHandler(ReviewActionReject)(c) }

// WithdrawReview POST /knowledge/articles/:id/review/withdraw
func (h *Handler) WithdrawReview(c *gin.Context) { h.reviewHandler(ReviewActionWithdraw)(c) }

// RepublishReview POST /knowledge/articles/:id/review/republish
func (h *Handler) RepublishReview(c *gin.Context) { h.reviewHandler(ReviewActionRepubish)(c) }

// ListArticleReviews GET /knowledge/articles/:id/reviews
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
	records, total, err := h.svc.ListReviews(c.Request.Context(), id, tenantID, pg.Page, pg.PageSize)
	if err != nil {
		common.NotFound(c, "Article not found")
		return
	}

	common.SuccessWithPagination(c, records, pg.Page, pg.PageSize, int64(total))
}
