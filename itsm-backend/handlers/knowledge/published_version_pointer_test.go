package knowledge

import (
	"context"
	"testing"

	"itsm-backend/ent/knowledgearticle"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// 本文件锁定「线上是哪个版本」的唯一事实源。
//
// 背景：发布此前只是把 is_published 翻 true，版本表存的又是「更新前快照」——
// 发布时刻的正文在版本表里没有对应行，于是「线上内容属于哪个版本」这个问题
// 没有任何地方能回答。知识被 RAG 引用、被工单引用后，「我当时引的是哪一版」
// 就是审计问题，不能靠猜。
//
// 指针语义：published_version 指向发布那一刻落的快照行；nil = 从未发布或
// 已撤回。发布路径有三条（评审通过 /publish / PUT 的 status），三条都必须
// 维护指针——只维护其中一条，另外两条就会让指针与事实脱节，比没有指针更
// 危险（因为指针看起来是权威的）。

// TestPublishSetsPublishedVersionPointer 发布必须在版本表里留下「发布时刻的
// 正文」并把指针指过去。
func TestPublishSetsPublishedVersionPointer(t *testing.T) {
	ctx := context.Background()
	svc, _ := newPublishTestService(t, "file:pubptr_publish?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "数据库连接池耗尽", Content: "连接池未设上限导致雪崩", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	require.Nil(t, created.PublishedVersion, "新建未发布文章的指针必须为空")

	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	approved, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)

	require.True(t, approved.IsPublished)
	require.NotNil(t, approved.PublishedVersion, "评审通过发布后必须有线上版本指针")
	require.Equal(t, 1, *approved.PublishedVersion)

	// 指针指向的快照必须**存在**，且内容就是发布时刻的正文。
	snap, err := svc.GetVersion(ctx, created.ID, *approved.PublishedVersion, publishTestTenant)
	require.NoError(t, err, "指针必须指向真实存在的版本快照")
	require.Equal(t, "连接池未设上限导致雪崩", snap.Content, "快照必须是发布时刻的正文，不是更新前的旧正文")
	require.Equal(t, "数据库连接池耗尽", snap.Title)
}

// TestUnpublishClearsPointer 撤回必须清指针：撤下后线上已无内容，
// 保留指针会让「线上是哪个版本」指向一个已不可见的版本号。
func TestUnpublishClearsPointer(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t, "file:pubptr_unpublish?mode=memory&cache=shared&_fk=1")
	_ = client

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "交换机固件升级", Content: "先在备机验证再灰度", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	published, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.NotNil(t, published.PublishedVersion)

	withdrawn, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, false)
	require.NoError(t, err)
	require.False(t, withdrawn.IsPublished)
	require.Nil(t, withdrawn.PublishedVersion, "撤回后指针必须清空")

	// 重新发布应得到**新的**版本号，而不是复用旧指针。
	// 此时 review_status 仍是 approved，直接走 /publish 即可（approved→submit 不合法）。
	republished, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, true)
	require.NoError(t, err)
	require.NotNil(t, republished.PublishedVersion)
	require.Greater(t, *republished.PublishedVersion, *published.PublishedVersion,
		"重新发布应指向更新的版本快照")
}

// TestRejectReviewClearsPointer 驳回走的是 TransitionReview 这条主路径，
// 与 /publish 撤回是两条不同的代码路径——两条都必须清指针。
func TestRejectReviewClearsPointer(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t, "file:pubptr_reject?mode=memory&cache=shared&_fk=1")
	_ = client

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "临时方案", Content: "先重启顶住", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	published, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.NotNil(t, published.PublishedVersion)

	// 驳回只能从 under_review 出发（approved→reject 不是合法流转），
	// 这里先把已发布的文章落回 under_review，模拟「发布后又进入复审被驳回」。
	_, err = client.KnowledgeArticle.UpdateOneID(created.ID).
		SetReviewStatus(knowledgearticle.ReviewStatus(ReviewStatusUnderReview)).
		Save(ctx)
	require.NoError(t, err)

	rejected, err := svc.RejectReview(ctx, created.ID, publishTestTenant, 2, "结论缺证据，驳回")
	require.NoError(t, err)
	require.False(t, rejected.IsPublished)
	require.Nil(t, rejected.PublishedVersion, "驳回撤下可见性时同样必须清指针")
}

// TestPutPublishPathMaintainsPointer PUT 的 status=published 与 /publish 是
// 同一可见性投影，若只有 /publish 维护指针，PUT 就会成为让指针撒谎的后门。
func TestPutPublishPathMaintainsPointer(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t, "file:pubptr_put?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "磁盘扩容", Content: "先扩只读副本", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)

	// 先让评审通过但不走发布路径：直接落 approved + is_published=false。
	_, err = client.KnowledgeArticle.UpdateOneID(created.ID).
		SetReviewStatus(knowledgearticle.ReviewStatus(ReviewStatusApproved)).
		SetIsPublished(false).
		Save(ctx)
	require.NoError(t, err)

	updated, err := svc.UpdateArticle(ctx, &Article{
		ID: created.ID, TenantID: publishTestTenant, AuthorID: 1,
		Title: "磁盘扩容", Content: "先扩只读副本再切主", Category: "运维",
		IsPublished: true, ReviewStatus: ReviewStatusApproved,
	})
	require.NoError(t, err)
	require.True(t, updated.IsPublished)
	require.NotNil(t, updated.PublishedVersion, "PUT 发布入口也必须建立指针")

	snap, err := svc.GetVersion(ctx, created.ID, *updated.PublishedVersion, publishTestTenant)
	require.NoError(t, err)
	require.Equal(t, "先扩只读副本再切主", snap.Content,
		"PUT 路径的快照必须是**即将上线**的正文，不是被覆写前的旧正文")
}

// TestPutCannotForgePointer 指针是系统事实：客户端自带的 publishedVersion
// 一律忽略，否则任何一次 PUT 都能改写「线上是哪个版本」的答案。
func TestPutCannotForgePointer(t *testing.T) {
	ctx := context.Background()
	svc, _ := newPublishTestService(t, "file:pubptr_forge?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "基线检查", Content: "每晚 02:00 跑基线", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	published, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.NotNil(t, published.PublishedVersion)

	forged := 999
	updated, err := svc.UpdateArticle(ctx, &Article{
		ID: created.ID, TenantID: publishTestTenant, AuthorID: 1,
		Title: "基线检查", Content: "每晚 02:00 跑基线", Category: "运维",
		IsPublished:      true,
		ReviewStatus:     ReviewStatusApproved,
		PublishedVersion: &forged,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.PublishedVersion)
	require.Equal(t, *published.PublishedVersion, *updated.PublishedVersion,
		"客户端传入的 publishedVersion 必须被忽略（已发布且未翻转时指针不应改变）")
}

// TestRestoreVersionClearsPointer 恢复旧版本会把文章打回 draft 并撤下可见性，
// 指针必须一起清——否则会指向一个已被覆写掉的版本。
func TestRestoreVersionClearsPointer(t *testing.T) {
	ctx := context.Background()
	svc, client := newPublishTestService(t, "file:pubptr_restore?mode=memory&cache=shared&_fk=1")
	_ = client

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "v1 正文", Content: "第一版内容", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	published, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.NotNil(t, published.PublishedVersion)

	restored, err := svc.RestoreVersion(ctx, created.ID, *published.PublishedVersion, publishTestTenant, 1)
	require.NoError(t, err)
	require.False(t, restored.IsPublished, "恢复必须打回未发布，由作者重新走评审")
	require.Nil(t, restored.PublishedVersion, "恢复撤下可见性时必须清指针")
}

// TestPublishPointerSnapshotGrows 每次发布都新增一个快照，指针跟着前移——
// 保证「线上内容」在版本链上始终有对应行，且历史不被覆盖。
func TestPublishPointerSnapshotGrows(t *testing.T) {
	ctx := context.Background()
	svc, _ := newPublishTestService(t, "file:pubptr_grow?mode=memory&cache=shared&_fk=1")

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "变更窗口", Content: "第一版", Category: "运维",
		AuthorID: 1, TenantID: publishTestTenant,
	})
	require.NoError(t, err)
	_, err = svc.SubmitReview(ctx, created.ID, publishTestTenant, 1, "")
	require.NoError(t, err)
	first, err := svc.ApproveReview(ctx, created.ID, publishTestTenant, 2, "")
	require.NoError(t, err)
	require.NotNil(t, first.PublishedVersion)

	// 撤回 → 改正文 → 重新评审通过
	_, err = svc.SetArticlePublished(ctx, created.ID, publishTestTenant, false)
	require.NoError(t, err)
	_, err = svc.UpdateArticle(ctx, &Article{
		ID: created.ID, TenantID: publishTestTenant, AuthorID: 1,
		Title: "变更窗口", Content: "第二版", Category: "运维",
		IsPublished: false, ReviewStatus: ReviewStatusDraft,
	})
	require.NoError(t, err)
	second, err := svc.SetArticlePublished(ctx, created.ID, publishTestTenant, true)
	require.NoError(t, err)

	require.NotNil(t, second.PublishedVersion)
	require.Greater(t, *second.PublishedVersion, *first.PublishedVersion)

	_, total, err := svc.ListVersions(ctx, created.ID, publishTestTenant, 1, 100)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, *second.PublishedVersion, "指针不得超过已有版本号")

	snap, err := svc.GetVersion(ctx, created.ID, *second.PublishedVersion, publishTestTenant)
	require.NoError(t, err)
	require.Equal(t, "第二版", snap.Content, "指针必须指向第二次发布时刻的正文")
}
