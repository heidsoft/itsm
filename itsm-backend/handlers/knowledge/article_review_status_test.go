package knowledge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestToArticleDTOKeepsReviewStatus 锁定「评审状态必须随文章一起返回」。
//
// 背景（由 MCP 实测发现）：库里存在 review_status = under_review 的文章，
// 而 KnowledgeArticleResponse 只有由 is_published 派生的 status（draft /
// published）。两条轴缺一条，API 消费者就只能看到 status=draft，于是：
//   · 分不清「卡在评审中」和「只是没发布」；
//   · 无法预判 POST /knowledge/articles/:id/publish 会不会被 409 拒绝
//     （发布门禁要求 review_status == approved）；
//   · 前端的「提交评审 / 审批」按钮只能靠额外请求试出来。
//
// 与工单的 buildTicketResponse 漏映射是同一类缺陷：写进库了，读不出来。
func TestToArticleDTOKeepsReviewStatus(t *testing.T) {
	h := &Handler{}

	cases := []struct {
		name         string
		reviewStatus string
		isPublished  bool
		wantReview   string
		wantStatus   string
	}{
		{name: "待评审未发布", reviewStatus: ReviewStatusUnderReview, isPublished: false, wantReview: ReviewStatusUnderReview, wantStatus: "draft"},
		{name: "已通过未发布", reviewStatus: ReviewStatusApproved, isPublished: false, wantReview: ReviewStatusApproved, wantStatus: "draft"},
		{name: "已通过且已发布", reviewStatus: ReviewStatusApproved, isPublished: true, wantReview: ReviewStatusApproved, wantStatus: "published"},
		{name: "已驳回", reviewStatus: ReviewStatusRejected, isPublished: false, wantReview: ReviewStatusRejected, wantStatus: "draft"},
		// 历史数据可能从未回填该列；归一到 draft 而不是吐空串，
		// 否则消费方要把 "" 也当作一种状态来处理。
		{name: "空值归一到 draft", reviewStatus: "", isPublished: false, wantReview: ReviewStatusDraft, wantStatus: "draft"},
		{name: "未知值归一到 draft", reviewStatus: "garbage", isPublished: true, wantReview: ReviewStatusDraft, wantStatus: "published"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.toArticleDTO(&Article{
				ID:           7,
				Title:        "打印机驱动兼容处理指南",
				ReviewStatus: tc.reviewStatus,
				IsPublished:  tc.isPublished,
			})
			require.NotNil(t, got)
			require.Equal(t, tc.wantReview, got.ReviewStatus)
			require.Equal(t, tc.wantStatus, got.Status)
		})
	}
}

// TestToArticleDTOReviewStatusIsIndependentFromPublishStatus 锁住「两条轴不可互相推导」。
//
// 反例：如果只保留 status，那么「已发布但未过评审」（历史数据）和
// 「草稿待评审」看起来完全一样，发布门禁也就无从预判。
func TestToArticleDTOReviewStatusIsIndependentFromPublishStatus(t *testing.T) {
	h := &Handler{}

	unpublishedApproved := h.toArticleDTO(&Article{ReviewStatus: ReviewStatusApproved, IsPublished: false})
	publishedDraft := h.toArticleDTO(&Article{ReviewStatus: ReviewStatusDraft, IsPublished: true})

	require.Equal(t, "draft", unpublishedApproved.Status)
	require.Equal(t, ReviewStatusApproved, unpublishedApproved.ReviewStatus)
	require.Equal(t, "published", publishedDraft.Status)
	require.Equal(t, ReviewStatusDraft, publishedDraft.ReviewStatus)
	require.NotEqual(t, unpublishedApproved.Status, publishedDraft.Status)
}
